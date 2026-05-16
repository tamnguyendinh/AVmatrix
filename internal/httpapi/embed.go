package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tamnguyendinh/avmatrix-go/internal/embeddings"
	"github.com/tamnguyendinh/avmatrix-go/internal/graph"
	"github.com/tamnguyendinh/avmatrix-go/internal/lbugnative"
	"github.com/tamnguyendinh/avmatrix-go/internal/lbugruntime"
	"github.com/tamnguyendinh/avmatrix-go/internal/repo"
)

const embedJobTimeout = 30 * time.Minute

var ErrEmbedUnavailable = errors.New("embedding runtime unavailable")

type EmbedRunner interface {
	Run(context.Context, EmbedTarget, func(JobProgress)) error
}

type EmbedTarget struct {
	RepoPath        string
	RepoName        string
	StoragePath     string
	LbugPath        string
	GraphPath       string
	AnalyzeLockPath string
}

type EmbedService struct {
	OpenWriteRunner               func(string) (lbugnative.WriteRunner, error)
	OpenWriteRunnerWithDimensions func(string, int) (lbugnative.WriteRunner, error)
	ResolveEmbedder               func() (embeddings.Embedder, embeddings.Config, error)
	LoadGraph                     func(string) (*graph.Graph, error)
}

type embedRequestBody struct {
	Repo string `json:"repo"`
}

type embedStartResponse struct {
	JobID  string    `json:"jobId"`
	Status JobStatus `json:"status"`
}

func (s Server) handleEmbed(w http.ResponseWriter, r *http.Request) {
	if !methodAllowed(w, r, http.MethodPost) {
		return
	}

	var body embedRequestBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON request body")
		return
	}

	entry, status, message, err := s.resolveEmbedRepo(r, body.Repo)
	if err != nil {
		s.logger.Debug("resolve embed repo failed", "error", err)
		writeError(w, status, message)
		return
	}
	target := embedTargetFor(entry)
	lock, err := repo.AcquireStorageLock(target.AnalyzeLockPath)
	if err != nil {
		if errors.Is(err, repo.ErrLockHeld) {
			writeError(w, http.StatusConflict, "Embedding or analysis already in progress for this repository")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	job, created, err := s.embedJobs.Create(target.StoragePath, target.RepoName)
	if err != nil {
		_ = lock.Release()
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	if !created {
		_ = lock.Release()
		writeJSON(w, http.StatusAccepted, embedStartResponse{JobID: job.ID, Status: job.Status})
		return
	}
	s.embedJobs.UpdateProgress(job.ID, JobProgress{Phase: string(JobAnalyzing), Percent: 0, Message: "Starting embedding generation..."})
	s.runEmbedJob(job.ID, target, lock)

	writeJSON(w, http.StatusAccepted, embedStartResponse{JobID: job.ID, Status: JobAnalyzing})
}

func (s Server) handleEmbedJob(w http.ResponseWriter, r *http.Request) {
	jobID, progress, ok := parseEmbedJobPath(r.URL.Path)
	if !ok {
		writeError(w, http.StatusNotFound, "Job not found")
		return
	}
	if progress {
		if !methodAllowed(w, r, http.MethodGet) {
			return
		}
		s.handleEmbedProgress(w, r, jobID)
		return
	}

	switch r.Method {
	case http.MethodGet:
		job, ok := s.embedJobs.Get(jobID)
		if !ok {
			writeError(w, http.StatusNotFound, "Job not found")
			return
		}
		writeJSON(w, http.StatusOK, job)
	case http.MethodDelete:
		job, ok := s.embedJobs.Get(jobID)
		if !ok {
			writeError(w, http.StatusNotFound, "Job not found")
			return
		}
		if isTerminalJobStatus(job.Status) {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("Job already %s", job.Status))
			return
		}
		s.embedJobs.Cancel(jobID, "Cancelled by user")
		writeJSON(w, http.StatusOK, map[string]string{
			"id":     jobID,
			"status": string(JobFailed),
			"error":  "Cancelled by user",
		})
	default:
		w.Header().Set("Allow", strings.Join([]string{http.MethodGet, http.MethodDelete}, ", "))
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

func (s Server) runEmbedJob(jobID string, target EmbedTarget, lock *repo.StorageLock) {
	ctx, cancel := context.WithTimeout(context.Background(), embedJobTimeout)
	s.embedJobs.RegisterCancel(jobID, cancel)

	go func() {
		defer s.embedJobs.ClearCancel(jobID)
		defer cancel()
		defer func() {
			if err := lock.Release(); err != nil {
				s.logger.Warn("release embed lock failed", "jobId", jobID, "error", err)
			}
		}()

		err := s.embedRunner.Run(ctx, target, func(progress JobProgress) {
			s.embedJobs.UpdateProgress(jobID, progress)
		})
		if err != nil {
			if errors.Is(err, context.Canceled) {
				s.embedJobs.Fail(jobID, "Cancelled by user")
				return
			}
			if errors.Is(err, context.DeadlineExceeded) {
				s.embedJobs.Fail(jobID, "Embedding timed out (30 minute limit)")
				return
			}
			s.embedJobs.Fail(jobID, err.Error())
			return
		}
		s.embedJobs.Complete(jobID)
	}()
}

func (s Server) resolveEmbedRepo(r *http.Request, bodyRepo string) (repo.RegistryEntry, int, string, error) {
	entries, err := s.store.ListRegistered(false)
	if err != nil {
		return repo.RegistryEntry{}, http.StatusInternalServerError, err.Error(), err
	}
	repoQuery := strings.TrimSpace(bodyRepo)
	if repoQuery == "" {
		repoQuery = requestedRepo(r)
	}
	return resolveRepoQuery(entries, repoQuery)
}

func (s Server) handleEmbedProgress(w http.ResponseWriter, r *http.Request, jobID string) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "Streaming unsupported")
		return
	}
	if _, ok := s.embedJobs.Get(jobID); !ok {
		writeError(w, http.StatusNotFound, "Job not found")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()

	lastPayload := ""
	for {
		job, ok := s.embedJobs.Get(jobID)
		if !ok {
			writeSSEEvent(w, "failed", map[string]string{"error": "Job not found"})
			flusher.Flush()
			return
		}
		if job.Status == JobComplete {
			writeSSEEvent(w, string(JobComplete), map[string]string{
				"repoName": job.RepoName,
				"repoPath": job.RepoPath,
			})
			flusher.Flush()
			return
		}
		if job.Status == JobFailed {
			writeSSEEvent(w, string(JobFailed), map[string]string{"error": job.Error})
			flusher.Flush()
			return
		}

		payload, err := json.Marshal(job.Progress)
		if err == nil && string(payload) != lastPayload {
			fmt.Fprintf(w, "data: %s\n\n", payload)
			lastPayload = string(payload)
			flusher.Flush()
		}

		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
		}
	}
}

func (s EmbedService) Run(ctx context.Context, target EmbedTarget, update func(JobProgress)) error {
	loadGraph := s.LoadGraph
	if loadGraph == nil {
		loadGraph = loadGraphSnapshot
	}
	update(JobProgress{Phase: string(JobLoading), Percent: 0, Message: "Loading graph snapshot..."})
	g, err := loadGraph(target.GraphPath)
	if err != nil {
		return err
	}

	resolveEmbedder := s.ResolveEmbedder
	if resolveEmbedder == nil {
		resolveEmbedder = defaultEmbedRuntime
	}
	update(JobProgress{Phase: string(JobLoading), Percent: 5, Message: "Loading embedding runtime..."})
	embedder, config, err := resolveEmbedder()
	if err != nil {
		return err
	}

	runner, err := s.openWriteRunner(target.LbugPath, config.Dimensions)
	if err != nil {
		if errors.Is(err, lbugnative.ErrUnavailable) {
			return fmt.Errorf("%w: native LadybugDB write runner is unavailable", ErrEmbedUnavailable)
		}
		return err
	}
	defer runner.Close()

	existingHashes := map[string]string{}
	hashResult, err := lbugruntime.FetchExistingEmbeddingHashes(runner.QueryRows)
	if err != nil {
		return err
	}
	if hashResult.Hashes != nil {
		existingHashes = hashResult.Hashes
	}

	_, err = embeddings.Run(ctx, g, runner, embedder, embeddings.RunOptions{
		Config:         config,
		ExistingHashes: existingHashes,
		RuntimeContext: embeddings.RuntimeContext{RepoName: target.RepoName},
		OnProgress: func(progress embeddings.Progress) {
			update(jobProgressFromEmbedding(progress))
		},
	})
	return err
}

func (s EmbedService) openWriteRunner(path string, embeddingDims int) (lbugnative.WriteRunner, error) {
	if s.OpenWriteRunnerWithDimensions != nil {
		return s.OpenWriteRunnerWithDimensions(path, embeddingDims)
	}
	if s.OpenWriteRunner != nil {
		return s.OpenWriteRunner(path)
	}
	return lbugnative.OpenWriteRunnerWithEmbeddingDims(path, embeddingDims)
}

func defaultEmbedRuntime() (embeddings.Embedder, embeddings.Config, error) {
	return resolveRuntimeEmbedder()
}

func loadGraphSnapshot(path string) (*graph.Graph, error) {
	file, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("graph snapshot not found; run avmatrix analyze --force first")
		}
		return nil, err
	}
	defer file.Close()

	var g graph.Graph
	if err := json.NewDecoder(file).Decode(&g); err != nil {
		return nil, err
	}
	return &g, nil
}

func embedTargetFor(entry repo.RegistryEntry) EmbedTarget {
	storagePath := storagePathFor(entry)
	paths := repo.Paths(entry.Path)
	if entry.StoragePath != "" {
		paths.StoragePath = storagePath
		paths.LbugPath = filepath.Join(storagePath, "lbug")
		paths.GraphPath = filepath.Join(storagePath, "graph.json")
		paths.AnalyzeLockPath = filepath.Join(storagePath, "analyze.lock")
	}
	return EmbedTarget{
		RepoPath:        entry.Path,
		RepoName:        entry.Name,
		StoragePath:     storagePath,
		LbugPath:        paths.LbugPath,
		GraphPath:       paths.GraphPath,
		AnalyzeLockPath: paths.AnalyzeLockPath,
	}
}

func parseEmbedJobPath(path string) (string, bool, bool) {
	tail := strings.TrimPrefix(path, "/api/embed/")
	if tail == "" || tail == path {
		return "", false, false
	}
	progress := false
	if strings.HasSuffix(tail, "/progress") {
		progress = true
		tail = strings.TrimSuffix(tail, "/progress")
	}
	jobID, err := url.PathUnescape(strings.Trim(tail, "/"))
	if err != nil || jobID == "" {
		return "", false, false
	}
	return jobID, progress, true
}

func jobProgressFromEmbedding(progress embeddings.Progress) JobProgress {
	phase := string(progress.Phase)
	message := fmt.Sprintf("%s (%d%%)", phase, progress.Percent)
	switch progress.Phase {
	case embeddings.PhaseEmbedding:
		message = fmt.Sprintf("Embedding nodes (%d%%)...", progress.Percent)
	case embeddings.PhaseIndexing:
		message = "Creating vector index..."
	case embeddings.PhaseReady:
		phase = string(JobComplete)
		message = "Embeddings complete"
	case embeddings.PhaseError:
		phase = string(JobFailed)
		if progress.Error != "" {
			message = progress.Error
		}
	}
	return JobProgress{
		Phase:   phase,
		Percent: progress.Percent,
		Message: message,
	}
}

func writeSSEEvent(w http.ResponseWriter, event string, payload any) {
	raw, err := json.Marshal(payload)
	if err != nil {
		raw = []byte(`{"error":"Failed to encode event"}`)
	}
	fmt.Fprintf(w, "event: %s\n", event)
	fmt.Fprintf(w, "data: %s\n\n", raw)
}
