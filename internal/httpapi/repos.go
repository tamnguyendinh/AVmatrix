package httpapi

import (
	"context"
	"errors"
	"net/http"
	"os"
	"time"

	"github.com/tamnguyendinh/avmatrix-go/internal/repo"
)

const repoHoldQueueTimeout = 10 * time.Minute

type repoListItem struct {
	Name       string      `json:"name"`
	Path       string      `json:"path"`
	IndexedAt  string      `json:"indexedAt"`
	LastCommit string      `json:"lastCommit"`
	Stats      *repo.Stats `json:"stats,omitempty"`
}

type repoInfoResponse struct {
	Name      string `json:"name"`
	RepoPath  string `json:"repoPath"`
	IndexedAt string `json:"indexedAt"`
	Stats     any    `json:"stats"`
}

func (s Server) handleRepos(w http.ResponseWriter, r *http.Request) {
	if !methodAllowed(w, r, http.MethodGet) {
		return
	}

	entries, err := s.store.ListRegistered(false)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list repos")
		return
	}

	items := make([]repoListItem, 0, len(entries))
	for _, entry := range entries {
		items = append(items, repoListItem{
			Name:       entry.Name,
			Path:       entry.Path,
			IndexedAt:  entry.IndexedAt,
			LastCommit: entry.LastCommit,
			Stats:      entry.Stats,
		})
	}
	writeJSON(w, http.StatusOK, items)
}

func (s Server) handleRepo(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleRepoInfo(w, r)
	case http.MethodDelete:
		s.handleRepoDelete(w, r)
	default:
		w.Header().Set("Allow", "GET, DELETE")
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

func (s Server) handleRepoInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		return
	}

	awaitAnalysis := r.URL.Query().Get("awaitAnalysis") == "true"
	if awaitAnalysis {
		if status, message, err := s.waitForAnalyzeQuery(r.Context(), requestedRepo(r)); err != nil {
			writeError(w, status, message)
			return
		}
	}

	entry, status, message, err := s.resolveRequestedRepo(r)
	if err != nil {
		s.logger.Debug("resolve repo failed", "error", err)
		writeError(w, status, message)
		return
	}
	if awaitAnalysis {
		if status, message, err := s.waitForAnalyzePath(r.Context(), entry.Path); err != nil {
			writeError(w, status, message)
			return
		}
	}

	meta, err := repo.LoadMeta(storagePathFor(entry))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	indexedAt := entry.IndexedAt
	stats := any(map[string]any{})
	if entry.Stats != nil {
		stats = entry.Stats
	}
	if meta != nil {
		if meta.IndexedAt != "" {
			indexedAt = meta.IndexedAt
		}
		if meta.Stats != nil {
			stats = meta.Stats
		}
	}

	writeJSON(w, http.StatusOK, repoInfoResponse{
		Name:      entry.Name,
		RepoPath:  entry.Path,
		IndexedAt: indexedAt,
		Stats:     stats,
	})
}

func (s Server) handleRepoDelete(w http.ResponseWriter, r *http.Request) {
	query := requestedRepo(r)
	if query == "" {
		writeError(w, http.StatusBadRequest, "Missing repo name")
		return
	}
	entry, status, message, err := s.resolveRequestedRepo(r)
	if err != nil {
		s.logger.Debug("resolve repo for delete failed", "error", err)
		writeError(w, status, message)
		return
	}
	if _, active := s.activeAnalyzeJobForRepo(entry.Path); active {
		writeError(w, http.StatusConflict, "Repository analysis is still in progress")
		return
	}
	storagePath := storagePathFor(entry)
	_ = os.RemoveAll(storagePath)
	if err := s.store.Unregister(entry.Path); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"deleted": entry.Name})
}

func (s Server) waitForAnalyzeQuery(ctx context.Context, query string) (int, string, error) {
	if query == "" {
		return http.StatusOK, "", nil
	}
	resolvedPath, err := repo.ResolveAnalyzePath(query)
	if err != nil {
		return http.StatusOK, "", nil
	}
	return s.waitForAnalyzePath(ctx, resolvedPath)
}

func (s Server) waitForAnalyzePath(ctx context.Context, repoPath string) (int, string, error) {
	waitCtx, cancel := context.WithTimeout(ctx, repoHoldQueueTimeout)
	defer cancel()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		if _, ok := s.activeAnalyzeJobForRepo(repoPath); !ok {
			return http.StatusOK, "", nil
		}

		select {
		case <-waitCtx.Done():
			if errors.Is(waitCtx.Err(), context.DeadlineExceeded) {
				return http.StatusServiceUnavailable, "Repository analysis is still in progress", waitCtx.Err()
			}
			return http.StatusRequestTimeout, "Repository analysis wait was cancelled", waitCtx.Err()
		case <-ticker.C:
		}
	}
}

func (s Server) activeAnalyzeJobForRepo(repoPath string) (Job, bool) {
	for _, job := range s.analyzeJobs.List() {
		if !isTerminalJobStatus(job.Status) && repo.SamePath(job.RepoPath, repoPath) {
			return job, true
		}
	}
	return Job{}, false
}

func (s Server) resolveRequestedRepo(r *http.Request) (repo.RegistryEntry, int, string, error) {
	entries, err := s.store.ListRegistered(false)
	if err != nil {
		return repo.RegistryEntry{}, http.StatusInternalServerError, err.Error(), err
	}

	return resolveRepoQuery(entries, requestedRepo(r))
}

func resolveRepoQuery(entries []repo.RegistryEntry, query string) (repo.RegistryEntry, int, string, error) {
	if query == "" {
		if len(entries) == 1 {
			return entries[0], http.StatusOK, "", nil
		}
		return repo.RegistryEntry{},
			http.StatusNotFound,
			"Repository not found. Run: avmatrix analyze",
			errors.New("repository not found")
	}

	entry, err := repo.ResolveEntry(entries, query)
	if err == nil {
		return entry, http.StatusOK, "", nil
	}

	var ambiguous repo.AmbiguousNameError
	if errors.As(err, &ambiguous) {
		return repo.RegistryEntry{}, http.StatusBadRequest, err.Error(), err
	}
	return repo.RegistryEntry{},
		http.StatusNotFound,
		"Repository not found. Run: avmatrix analyze",
		err
}

func requestedRepo(r *http.Request) string {
	return r.URL.Query().Get("repo")
}

func storagePathFor(entry repo.RegistryEntry) string {
	if entry.StoragePath != "" {
		return entry.StoragePath
	}
	return repo.StoragePath(entry.Path)
}
