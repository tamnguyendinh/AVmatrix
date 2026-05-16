package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/tamnguyendinh/avmatrix-go/internal/repo"
)

func TestAnalyzeRejectsInvalidRequests(t *testing.T) {
	server, _ := newRepoServer(t, nil)
	defer server.Close()

	assertAnalyzeError(t, server.URL, `{}`, http.StatusBadRequest, `Provide "path" (local path)`)
	assertAnalyzeError(t, server.URL, `{"url":"https://example.com/repo.git","path":"`+escapeJSON(t.TempDir())+`"}`, http.StatusBadRequest, "Remote repository URLs are no longer supported")
	assertAnalyzeError(t, server.URL, `{"path":42}`, http.StatusBadRequest, `"path" must be a string`)
	assertAnalyzeError(t, server.URL, `{"path":"`+escapeJSON(t.TempDir())+`","embeddings":"yes"}`, http.StatusBadRequest, `"embeddings" must be a boolean`)
}

func TestAnalyzeStartsJobAndStreamsCompletionPayload(t *testing.T) {
	runner := &immediateAnalyzeRunner{repoName: "alpha"}
	server, _ := newRepoServerWithConfig(t, nil, func(config *Config) {
		config.AnalyzeRunner = runner
	})
	defer server.Close()

	repoPath := t.TempDir()
	expectedPath, err := repo.ResolveAnalyzePath(repoPath)
	if err != nil {
		t.Fatalf("resolve temp repo path: %v", err)
	}
	started := startAnalyzeForTest(t, server.URL, repoPath)
	if started.Status != JobAnalyzing || started.JobID == "" {
		t.Fatalf("unexpected start response: %#v", started)
	}

	job := waitForAnalyzeStatus(t, server.URL, started.JobID, JobComplete)
	if job.RepoPath != expectedPath || job.RepoName != "alpha" {
		t.Fatalf("job completion identity = %#v", job)
	}
	if runner.target.RepoPath != expectedPath {
		t.Fatalf("runner target = %#v", runner.target)
	}

	resp, err := http.Get(server.URL + "/api/analyze/" + url.PathEscape(started.JobID) + "/progress")
	if err != nil {
		t.Fatalf("GET analyze progress: %v", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read analyze progress: %v", err)
	}
	text := string(body)
	if !strings.Contains(text, "event: complete") || !strings.Contains(text, `"repoName":"alpha"`) || !strings.Contains(text, `"repoPath":"`+jsonEscapedFragment(expectedPath)+`"`) {
		t.Fatalf("completion SSE payload = %q", text)
	}
}

func TestRepoInfoAwaitAnalysisHoldsUntilAnalyzeCompletes(t *testing.T) {
	runner := newReleasableAnalyzeRunner("alpha")
	server, fixtures := newRepoServerWithConfig(t, []repoFixture{{name: "alpha"}}, func(config *Config) {
		config.AnalyzeRunner = runner
	})
	defer server.Close()

	started := startAnalyzeForTest(t, server.URL, fixtures[0].path)
	if started.JobID == "" {
		t.Fatal("analyze job id is empty")
	}
	runner.waitStarted(t)

	done := make(chan repoInfoResult, 1)
	go func() {
		resp, err := http.Get(server.URL + "/api/repo?awaitAnalysis=true&repo=" + url.QueryEscape(fixtures[0].path))
		if err != nil {
			done <- repoInfoResult{err: err}
			return
		}
		defer resp.Body.Close()
		raw, readErr := io.ReadAll(resp.Body)
		result := repoInfoResult{status: resp.StatusCode, body: string(raw), err: readErr}
		if readErr == nil && resp.StatusCode == http.StatusOK {
			result.err = json.Unmarshal(raw, &result.payload)
		}
		done <- result
	}()

	select {
	case result := <-done:
		t.Fatalf("/api/repo returned before analyze completed: %#v", result)
	case <-time.After(75 * time.Millisecond):
	}

	close(runner.release)
	waitForAnalyzeStatus(t, server.URL, started.JobID, JobComplete)

	select {
	case result := <-done:
		if result.err != nil || result.status != http.StatusOK {
			t.Fatalf("/api/repo wait result = %#v", result)
		}
		if result.payload["repoPath"] != fixtures[0].path {
			t.Fatalf("/api/repo payload = %#v", result.payload)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("/api/repo did not return after analyze completed")
	}
}

func TestRepoHoldQueueTimeoutContract(t *testing.T) {
	if repoHoldQueueTimeout != 10*time.Minute {
		t.Fatalf("repoHoldQueueTimeout = %s, want 10m", repoHoldQueueTimeout)
	}
}

func TestAnalyzeRejectsHeldRepoLock(t *testing.T) {
	runner := newBlockingAnalyzeRunner()
	server, _ := newRepoServerWithConfig(t, nil, func(config *Config) {
		config.AnalyzeRunner = runner
	})
	defer server.Close()

	repoPath := t.TempDir()
	resolvedPath, err := repo.ResolveAnalyzePath(repoPath)
	if err != nil {
		t.Fatalf("resolve temp repo path: %v", err)
	}
	lock, err := repo.AcquireStorageLock(repo.Paths(resolvedPath).AnalyzeLockPath)
	if err != nil {
		t.Fatalf("AcquireStorageLock() error = %v", err)
	}
	defer lock.Release()

	resp, err := http.Post(server.URL+"/api/analyze", "application/json", bytes.NewBufferString(`{"path":"`+escapeJSON(repoPath)+`"}`))
	if err != nil {
		t.Fatalf("POST analyze: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusConflict {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("analyze status = %d, want 409; body=%s", resp.StatusCode, body)
	}

	select {
	case <-runner.started:
		t.Fatal("runner started even though repo lock was held")
	case <-time.After(50 * time.Millisecond):
	}
}

func TestAnalyzeDeduplicatesActiveSameRepoJob(t *testing.T) {
	runner := newBlockingAnalyzeRunner()
	server, _ := newRepoServerWithConfig(t, nil, func(config *Config) {
		config.AnalyzeRunner = runner
	})
	defer server.Close()

	repoPath := t.TempDir()
	first := startAnalyzeForTest(t, server.URL, repoPath)
	runner.waitStarted(t)

	second := startAnalyzeForTest(t, server.URL, repoPath)
	if second.JobID != first.JobID {
		t.Fatalf("same-repo analyze job id = %s, want existing %s", second.JobID, first.JobID)
	}
	if second.Status != JobAnalyzing {
		t.Fatalf("same-repo analyze status = %s, want %s", second.Status, JobAnalyzing)
	}
	if calls := runner.callCount(); calls != 1 {
		t.Fatalf("analyze runner calls = %d, want 1", calls)
	}

	req, err := http.NewRequest(http.MethodDelete, server.URL+"/api/analyze/"+url.PathEscape(first.JobID), nil)
	if err != nil {
		t.Fatalf("new delete request: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("DELETE analyze: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("delete status = %d", resp.StatusCode)
	}
	waitForAnalyzeStatus(t, server.URL, first.JobID, JobFailed)
}

func TestAnalyzeRejectsConcurrentJobAndCancelsRunningJob(t *testing.T) {
	runner := newBlockingAnalyzeRunner()
	server, _ := newRepoServerWithConfig(t, nil, func(config *Config) {
		config.AnalyzeRunner = runner
	})
	defer server.Close()

	firstPath := t.TempDir()
	started := startAnalyzeForTest(t, server.URL, firstPath)
	runner.waitStarted(t)

	secondResp, err := http.Post(server.URL+"/api/analyze", "application/json", bytes.NewBufferString(`{"path":"`+escapeJSON(t.TempDir())+`"}`))
	if err != nil {
		t.Fatalf("POST second analyze: %v", err)
	}
	defer secondResp.Body.Close()
	if secondResp.StatusCode != http.StatusConflict {
		body, _ := io.ReadAll(secondResp.Body)
		t.Fatalf("second analyze status = %d, want 409; body=%s", secondResp.StatusCode, body)
	}

	req, err := http.NewRequest(http.MethodDelete, server.URL+"/api/analyze/"+url.PathEscape(started.JobID), nil)
	if err != nil {
		t.Fatalf("new delete request: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("DELETE analyze: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("delete status = %d; body=%s", resp.StatusCode, body)
	}

	job := waitForAnalyzeStatus(t, server.URL, started.JobID, JobFailed)
	if !strings.Contains(job.Error, "Cancelled") {
		t.Fatalf("cancelled job = %#v", job)
	}
}

func TestAnalyzeLockBlocksEmbedAndReleasesAfterCancel(t *testing.T) {
	analyzeRunner := newLockingAnalyzeRunner()
	embedRunner := &recordingEmbedRunner{}
	server, fixtures := newRepoServerWithConfig(t, []repoFixture{{name: "alpha"}}, func(config *Config) {
		config.AnalyzeRunner = analyzeRunner
		config.EmbedRunner = embedRunner
	})
	defer server.Close()

	started := startAnalyzeForTest(t, server.URL, fixtures[0].path)
	analyzeRunner.waitStarted(t)

	var conflict map[string]string
	postJSON(t, server.URL+"/api/embed", `{"repo":"`+jsonEscape(fixtures[0].path)+`"}`, http.StatusConflict, &conflict)
	if !strings.Contains(conflict["error"], "already in progress") {
		t.Fatalf("embed conflict = %#v", conflict)
	}
	if embedRunner.called {
		t.Fatal("embed runner should not start while analyze lock is held")
	}

	req, err := http.NewRequest(http.MethodDelete, server.URL+"/api/analyze/"+url.PathEscape(started.JobID), nil)
	if err != nil {
		t.Fatalf("new delete request: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("DELETE analyze: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("delete status = %d", resp.StatusCode)
	}
	waitForAnalyzeStatus(t, server.URL, started.JobID, JobFailed)

	var embedStart embedStartResponse
	postEmbedEventually(t, server.URL, fixtures[0].path, http.StatusAccepted, &embedStart)
	waitForEmbedJob(t, server.URL, embedStart.JobID, JobComplete)
	if !embedRunner.called {
		t.Fatal("embed runner did not start after analyze lock released")
	}
}

type immediateAnalyzeRunner struct {
	target   AnalyzeTarget
	repoName string
}

func (r *immediateAnalyzeRunner) Run(ctx context.Context, target AnalyzeTarget, update func(JobProgress)) (AnalyzeRunResult, error) {
	r.target = target
	update(JobProgress{Phase: string(JobAnalyzing), Percent: 50, Message: "Analyzing..."})
	return AnalyzeRunResult{RepoPath: target.RepoPath, RepoName: r.repoName}, nil
}

type blockingAnalyzeRunner struct {
	started chan struct{}
	once    sync.Once
	mu      sync.Mutex
	calls   int
}

func newBlockingAnalyzeRunner() *blockingAnalyzeRunner {
	return &blockingAnalyzeRunner{started: make(chan struct{})}
}

func (r *blockingAnalyzeRunner) Run(ctx context.Context, target AnalyzeTarget, update func(JobProgress)) (AnalyzeRunResult, error) {
	r.mu.Lock()
	r.calls++
	r.mu.Unlock()
	update(JobProgress{Phase: string(JobAnalyzing), Percent: 5, Message: "Analyzing..."})
	r.once.Do(func() {
		close(r.started)
	})
	<-ctx.Done()
	return AnalyzeRunResult{}, ctx.Err()
}

func (r *blockingAnalyzeRunner) callCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.calls
}

func (r *blockingAnalyzeRunner) waitStarted(t *testing.T) {
	t.Helper()
	select {
	case <-r.started:
	case <-time.After(time.Second):
		t.Fatal("analyze runner did not start")
	}
}

type lockingAnalyzeRunner struct {
	started chan struct{}
	once    sync.Once
}

func newLockingAnalyzeRunner() *lockingAnalyzeRunner {
	return &lockingAnalyzeRunner{started: make(chan struct{})}
}

func (r *lockingAnalyzeRunner) Run(ctx context.Context, target AnalyzeTarget, update func(JobProgress)) (AnalyzeRunResult, error) {
	lock, err := repo.AcquireStorageLock(repo.Paths(target.RepoPath).AnalyzeLockPath)
	if err != nil {
		return AnalyzeRunResult{}, err
	}
	defer lock.Release()

	update(JobProgress{Phase: string(JobAnalyzing), Percent: 5, Message: "Analyzing..."})
	r.once.Do(func() {
		close(r.started)
	})
	<-ctx.Done()
	return AnalyzeRunResult{}, ctx.Err()
}

func (r *lockingAnalyzeRunner) waitStarted(t *testing.T) {
	t.Helper()
	select {
	case <-r.started:
	case <-time.After(time.Second):
		t.Fatal("locking analyze runner did not start")
	}
}

type releasableAnalyzeRunner struct {
	started  chan struct{}
	release  chan struct{}
	once     sync.Once
	repoName string
}

func newReleasableAnalyzeRunner(repoName string) *releasableAnalyzeRunner {
	return &releasableAnalyzeRunner{
		started:  make(chan struct{}),
		release:  make(chan struct{}),
		repoName: repoName,
	}
}

func (r *releasableAnalyzeRunner) Run(ctx context.Context, target AnalyzeTarget, update func(JobProgress)) (AnalyzeRunResult, error) {
	update(JobProgress{Phase: string(JobAnalyzing), Percent: 5, Message: "Analyzing..."})
	r.once.Do(func() {
		close(r.started)
	})
	select {
	case <-r.release:
		return AnalyzeRunResult{RepoPath: target.RepoPath, RepoName: r.repoName}, nil
	case <-ctx.Done():
		return AnalyzeRunResult{}, ctx.Err()
	}
}

func (r *releasableAnalyzeRunner) waitStarted(t *testing.T) {
	t.Helper()
	select {
	case <-r.started:
	case <-time.After(time.Second):
		t.Fatal("analyze runner did not start")
	}
}

type repoInfoResult struct {
	status  int
	payload map[string]any
	body    string
	err     error
}

func startAnalyzeForTest(t *testing.T, baseURL string, repoPath string) analyzeStartResponse {
	t.Helper()
	resp, err := http.Post(baseURL+"/api/analyze", "application/json", bytes.NewBufferString(`{"path":"`+escapeJSON(repoPath)+`"}`))
	if err != nil {
		t.Fatalf("POST analyze: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("POST analyze status = %d, want 202; body=%s", resp.StatusCode, body)
	}
	var started analyzeStartResponse
	if err := json.NewDecoder(resp.Body).Decode(&started); err != nil {
		t.Fatalf("decode analyze start: %v", err)
	}
	return started
}

func waitForAnalyzeStatus(t *testing.T, baseURL string, jobID string, want JobStatus) Job {
	t.Helper()
	for attempt := 0; attempt < 100; attempt++ {
		var job Job
		getJSON(t, baseURL+"/api/analyze/"+url.PathEscape(jobID), http.StatusOK, &job)
		if job.Status == want {
			return job
		}
		time.Sleep(10 * time.Millisecond)
	}
	var job Job
	getJSON(t, baseURL+"/api/analyze/"+url.PathEscape(jobID), http.StatusOK, &job)
	t.Fatalf("job status = %s, want %s; job=%#v", job.Status, want, job)
	return Job{}
}

func assertAnalyzeError(t *testing.T, baseURL string, body string, wantStatus int, wantError string) {
	t.Helper()
	resp, err := http.Post(baseURL+"/api/analyze", "application/json", bytes.NewBufferString(body))
	if err != nil {
		t.Fatalf("POST analyze error case: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != wantStatus {
		raw, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d, want %d; body=%s", resp.StatusCode, wantStatus, raw)
	}
	var payload map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode error payload: %v", err)
	}
	if !strings.Contains(payload["error"], wantError) {
		t.Fatalf("error = %#v, want contains %q", payload, wantError)
	}
}

func postEmbedEventually(t *testing.T, baseURL string, repoPath string, wantStatus int, target any) {
	t.Helper()
	body := `{"repo":"` + jsonEscape(repoPath) + `"}`
	deadline := time.Now().Add(2 * time.Second)
	var lastStatus int
	var lastBody string
	for time.Now().Before(deadline) {
		resp, err := http.Post(baseURL+"/api/embed", "application/json", bytes.NewBufferString(body))
		if err != nil {
			t.Fatalf("POST embed: %v", err)
		}
		raw, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			t.Fatalf("read embed response: %v", readErr)
		}
		lastStatus = resp.StatusCode
		lastBody = string(raw)
		if resp.StatusCode == wantStatus {
			if err := json.Unmarshal(raw, target); err != nil {
				t.Fatalf("decode embed response: %v; body=%s", err, raw)
			}
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("POST embed status = %d, want %d; body=%s", lastStatus, wantStatus, lastBody)
}

func escapeJSON(value string) string {
	raw, _ := json.Marshal(value)
	return strings.Trim(string(raw), `"`)
}

func jsonEscapedFragment(value string) string {
	return escapeJSON(value)
}
