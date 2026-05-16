package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	analyzer "github.com/tamnguyendinh/avmatrix-go/internal/analyze"
	"github.com/tamnguyendinh/avmatrix-go/internal/embeddings"
	"github.com/tamnguyendinh/avmatrix-go/internal/lbugload"
	"github.com/tamnguyendinh/avmatrix-go/internal/lbugnative"
	"github.com/tamnguyendinh/avmatrix-go/internal/repo"
)

const analyzeJobTimeout = 30 * time.Minute

type AnalyzeRunner interface {
	Run(context.Context, AnalyzeTarget, func(JobProgress)) (AnalyzeRunResult, error)
}

type AnalyzeTarget struct {
	RepoPath   string
	Embeddings bool
}

type AnalyzeRunResult struct {
	RepoPath string
	RepoName string
}

type AnalyzeService struct {
	Store      repo.Store
	RunAnalyze func(context.Context, string, analyzer.Options) (analyzer.Result, error)
}

type analyzeRequestBody struct {
	Path       string
	URL        string
	Embeddings bool
}

type analyzeStartResponse struct {
	JobID  string    `json:"jobId"`
	Status JobStatus `json:"status"`
}

func (s Server) handleAnalyze(w http.ResponseWriter, r *http.Request) {
	if !methodAllowed(w, r, http.MethodPost) {
		return
	}

	body, status, message, err := parseAnalyzeRequest(r.Body)
	if err != nil {
		writeError(w, status, message)
		return
	}
	if body.URL != "" {
		writeError(w, http.StatusBadRequest, `Remote repository URLs are no longer supported; provide "path" only`)
		return
	}
	if strings.TrimSpace(body.Path) == "" {
		writeError(w, http.StatusBadRequest, `Provide "path" (local path)`)
		return
	}

	resolvedPath, err := repo.ResolveAnalyzePath(body.Path)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if job, ok := s.activeAnalyzeJobForRepo(resolvedPath); ok {
		writeJSON(w, http.StatusAccepted, analyzeStartResponse{JobID: job.ID, Status: job.Status})
		return
	}
	if err := ensureAnalyzeLockAvailable(resolvedPath); err != nil {
		if errors.Is(err, repo.ErrLockHeld) {
			writeError(w, http.StatusConflict, "Embedding or analysis already in progress for this repository")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	job, created, err := s.analyzeJobs.Create(resolvedPath, "")
	if err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	if !created {
		writeJSON(w, http.StatusAccepted, analyzeStartResponse{JobID: job.ID, Status: job.Status})
		return
	}
	s.analyzeJobs.UpdateProgress(job.ID, JobProgress{
		Phase:   string(JobAnalyzing),
		Percent: 0,
		Message: "Preparing local analysis...",
	})
	s.runAnalyzeJob(job.ID, AnalyzeTarget{RepoPath: resolvedPath, Embeddings: body.Embeddings})

	writeJSON(w, http.StatusAccepted, analyzeStartResponse{JobID: job.ID, Status: JobAnalyzing})
}

func (s Server) handleAnalyzeJob(w http.ResponseWriter, r *http.Request) {
	jobID, progress, ok := parseAnalyzeJobPath(r.URL.Path)
	if !ok {
		writeError(w, http.StatusNotFound, "Job not found")
		return
	}
	if progress {
		if !methodAllowed(w, r, http.MethodGet) {
			return
		}
		s.handleAnalyzeProgress(w, r, jobID)
		return
	}

	switch r.Method {
	case http.MethodGet:
		job, ok := s.analyzeJobs.Get(jobID)
		if !ok {
			writeError(w, http.StatusNotFound, "Job not found")
			return
		}
		writeJSON(w, http.StatusOK, job)
	case http.MethodDelete:
		job, ok := s.analyzeJobs.Get(jobID)
		if !ok {
			writeError(w, http.StatusNotFound, "Job not found")
			return
		}
		if isTerminalJobStatus(job.Status) {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("Job already %s", job.Status))
			return
		}
		s.analyzeJobs.Cancel(jobID, "Cancelled by user")
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

func ensureAnalyzeLockAvailable(repoPath string) error {
	lock, err := repo.AcquireStorageLock(repo.Paths(repoPath).AnalyzeLockPath)
	if err != nil {
		return err
	}
	return lock.Release()
}

func (s Server) runAnalyzeJob(jobID string, target AnalyzeTarget) {
	ctx, cancel := context.WithTimeout(context.Background(), analyzeJobTimeout)
	s.analyzeJobs.RegisterCancel(jobID, cancel)

	go func() {
		defer s.analyzeJobs.ClearCancel(jobID)
		defer cancel()

		result, err := s.analyzeRunner.Run(ctx, target, func(progress JobProgress) {
			s.analyzeJobs.UpdateProgress(jobID, progress)
		})
		if err != nil {
			if errors.Is(err, context.Canceled) {
				s.analyzeJobs.Fail(jobID, "Cancelled by user")
				return
			}
			if errors.Is(err, context.DeadlineExceeded) {
				s.analyzeJobs.Fail(jobID, "Analysis timed out (30 minute limit)")
				return
			}
			s.analyzeJobs.Fail(jobID, err.Error())
			return
		}
		s.analyzeJobs.CompleteWithResult(jobID, result.RepoPath, result.RepoName)
	}()
}

func (s Server) handleAnalyzeProgress(w http.ResponseWriter, r *http.Request, jobID string) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "Streaming unsupported")
		return
	}
	if _, ok := s.analyzeJobs.Get(jobID); !ok {
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
		job, ok := s.analyzeJobs.Get(jobID)
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

func (s AnalyzeService) Run(ctx context.Context, target AnalyzeTarget, update func(JobProgress)) (AnalyzeRunResult, error) {
	runAnalyze := s.RunAnalyze
	if runAnalyze == nil {
		runAnalyze = analyzer.Run
	}

	options := analyzer.Options{
		Force:                          true,
		Embeddings:                     target.Embeddings,
		WriteGraphSnapshot:             true,
		DBRunnerFactory:                analyzeNativeDBRunnerFactory,
		ReleaseScopeIRsAfterResolution: true,
		OnEvent: func(event analyzer.Event) {
			update(jobProgressFromAnalyzeEvent(event))
		},
	}
	if target.Embeddings {
		options.EmbeddingConfig = embeddings.DefaultConfig()
		dimensions, ok, err := embeddings.HTTPDimensions(nil)
		if err != nil {
			return AnalyzeRunResult{}, err
		}
		if ok {
			options.EmbeddingConfig.Dimensions = dimensions
		}
	}

	update(JobProgress{Phase: string(JobAnalyzing), Percent: 0, Message: "Preparing local analysis..."})
	result, err := runAnalyze(ctx, target.RepoPath, options)
	if err != nil {
		return AnalyzeRunResult{}, err
	}
	name, err := s.recordResult(result)
	if err != nil {
		return AnalyzeRunResult{}, err
	}
	return AnalyzeRunResult{RepoPath: result.RepoPath, RepoName: name}, nil
}

func (s AnalyzeService) recordResult(result analyzer.Result) (string, error) {
	meta := repo.Meta{
		RepoPath:   result.RepoPath,
		LastCommit: repo.CurrentCommit(result.RepoPath),
		IndexedAt:  time.Now().UTC().Format(time.RFC3339),
		Stats:      statsFromAnalyzeResult(result),
	}
	paths := repo.Paths(result.RepoPath)
	if err := repo.SaveMeta(paths.StoragePath, meta); err != nil {
		return "", err
	}
	return s.Store.Register(result.RepoPath, meta, repo.RegisterOptions{})
}

func analyzeNativeDBRunnerFactory(paths repo.StoragePaths) (lbugload.QueryRunner, func() error, error) {
	runner, err := lbugnative.OpenWriteRunner(paths.LbugPath)
	if errors.Is(err, lbugnative.ErrUnavailable) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}
	return runner, runner.Close, nil
}

func statsFromAnalyzeResult(result analyzer.Result) *repo.Stats {
	files := result.Metrics.Files.Scanned
	nodes := 0
	edges := 0
	if result.Graph != nil {
		nodes = len(result.Graph.Nodes)
		edges = len(result.Graph.Relationships)
	}
	communities := result.Metrics.Communities.CommunitiesEmitted
	processes := result.Metrics.Processes.ProcessesEmitted
	stats := repo.Stats{
		Files:       &files,
		Nodes:       &nodes,
		Edges:       &edges,
		Communities: &communities,
		Processes:   &processes,
	}
	if result.Metrics.Embeddings.TotalNodes > 0 {
		embedded := result.Metrics.Embeddings.EmbeddedNodes + result.Metrics.Embeddings.SkippedFreshNodes
		stats.Embeddings = &embedded
	}
	return &stats
}

func jobProgressFromAnalyzeEvent(event analyzer.Event) JobProgress {
	phase := string(event.Phase)
	percent := analyzePhasePercent(event.Phase, event.Kind)
	message := fmt.Sprintf("%s phase", phase)
	if event.Kind == analyzer.EventPhaseDone {
		message = fmt.Sprintf("%s complete", phase)
	}
	if event.Kind == analyzer.EventProgress && event.Total > 0 {
		percent = boundedPercent(percent + int(float64(event.Current)/float64(event.Total)*5))
		message = fmt.Sprintf("%s %d/%d", phase, event.Current, event.Total)
	}
	if event.Phase == analyzer.PhaseDBLoad {
		phase = string(JobLoading)
		message = "Loading graph into LadybugDB..."
	}
	return JobProgress{Phase: phase, Percent: percent, Message: message}
}

func analyzePhasePercent(phase analyzer.PhaseName, kind analyzer.EventKind) int {
	for index, candidate := range analyzePhaseOrder {
		if candidate == phase {
			if kind == analyzer.EventPhaseDone {
				index++
			}
			return boundedPercent(int(float64(index) / float64(len(analyzePhaseOrder)) * 95))
		}
	}
	return 0
}

func boundedPercent(value int) int {
	if value < 0 {
		return 0
	}
	if value > 99 {
		return 99
	}
	return value
}

var analyzePhaseOrder = []analyzer.PhaseName{
	analyzer.PhaseScan,
	analyzer.PhaseStructure,
	analyzer.PhaseDocuments,
	analyzer.PhaseCobol,
	analyzer.PhaseParse,
	analyzer.PhaseRoutes,
	analyzer.PhaseTools,
	analyzer.PhaseORM,
	analyzer.PhaseCrossFile,
	analyzer.PhaseResolution,
	analyzer.PhaseMRO,
	analyzer.PhaseCommunities,
	analyzer.PhaseProcesses,
	analyzer.PhaseDBLoad,
	analyzer.PhaseEmbeddings,
}

func parseAnalyzeRequest(body io.Reader) (analyzeRequestBody, int, string, error) {
	var raw map[string]json.RawMessage
	if err := json.NewDecoder(body).Decode(&raw); err != nil {
		return analyzeRequestBody{}, http.StatusBadRequest, "Invalid JSON request body", err
	}

	repoURL, err := optionalString(raw, "url")
	if err != nil {
		return analyzeRequestBody{}, http.StatusBadRequest, err.Error(), err
	}
	repoPath, err := optionalString(raw, "path")
	if err != nil {
		return analyzeRequestBody{}, http.StatusBadRequest, err.Error(), err
	}
	embeddingFlag, err := optionalBool(raw, "embeddings")
	if err != nil {
		return analyzeRequestBody{}, http.StatusBadRequest, err.Error(), err
	}

	return analyzeRequestBody{Path: repoPath, URL: repoURL, Embeddings: embeddingFlag}, http.StatusOK, "", nil
}

func optionalString(raw map[string]json.RawMessage, key string) (string, error) {
	value, ok := raw[key]
	if !ok || string(value) == "null" {
		return "", nil
	}
	var out string
	if err := json.Unmarshal(value, &out); err != nil {
		return "", fmt.Errorf(`"%s" must be a string`, key)
	}
	return out, nil
}

func optionalBool(raw map[string]json.RawMessage, key string) (bool, error) {
	value, ok := raw[key]
	if !ok || string(value) == "null" {
		return false, nil
	}
	var out bool
	if err := json.Unmarshal(value, &out); err != nil {
		return false, fmt.Errorf(`"%s" must be a boolean`, key)
	}
	return out, nil
}

func parseAnalyzeJobPath(path string) (string, bool, bool) {
	tail := strings.TrimPrefix(path, "/api/analyze/")
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
