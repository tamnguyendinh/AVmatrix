package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tamnguyendinh/avmatrix-go/internal/embeddings"
	"github.com/tamnguyendinh/avmatrix-go/internal/graph"
	"github.com/tamnguyendinh/avmatrix-go/internal/lbugnative"
	"github.com/tamnguyendinh/avmatrix-go/internal/lbugruntime"
	"github.com/tamnguyendinh/avmatrix-go/internal/repo"
	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

func TestEmbedEndpointStartsJobAndCompletes(t *testing.T) {
	runner := &recordingEmbedRunner{}
	server, fixtures := newRepoServerWithConfig(t, []repoFixture{{name: "alpha"}}, func(config *Config) {
		config.EmbedRunner = runner
	})
	defer server.Close()

	var started embedStartResponse
	postJSON(t, server.URL+"/api/embed", `{"repo":"`+jsonEscape(fixtures[0].path)+`"}`, http.StatusAccepted, &started)

	if started.JobID == "" || started.Status != JobAnalyzing {
		t.Fatalf("start response = %#v", started)
	}
	job := waitForEmbedJob(t, server.URL, started.JobID, JobComplete)
	if job.RepoName != "alpha" || job.Progress.Percent != 100 {
		t.Fatalf("completed job = %#v", job)
	}
	if runner.target.RepoPath != fixtures[0].path {
		t.Fatalf("runner repo = %q, want %q", runner.target.RepoPath, fixtures[0].path)
	}
	if runner.target.LbugPath != filepath.Join(fixtures[0].path, ".avmatrix", "lbug") {
		t.Fatalf("runner lbug path = %q", runner.target.LbugPath)
	}
}

func TestEmbedEndpointRejectsHeldRepoLock(t *testing.T) {
	runner := &recordingEmbedRunner{}
	server, fixtures := newRepoServerWithConfig(t, []repoFixture{{name: "alpha"}}, func(config *Config) {
		config.EmbedRunner = runner
	})
	defer server.Close()

	lock, err := repo.AcquireStorageLock(repo.Paths(fixtures[0].path).AnalyzeLockPath)
	if err != nil {
		t.Fatalf("AcquireStorageLock() error = %v", err)
	}
	defer lock.Release()

	var payload map[string]string
	postJSON(t, server.URL+"/api/embed", `{"repo":"`+jsonEscape(fixtures[0].path)+`"}`, http.StatusConflict, &payload)

	if !strings.Contains(payload["error"], "already in progress") {
		t.Fatalf("error = %#v", payload)
	}
	if runner.called {
		t.Fatalf("runner should not be called when lock is held")
	}
}

func TestEmbedEndpointCancelMarksJobFailed(t *testing.T) {
	block := make(chan struct{})
	runner := &recordingEmbedRunner{block: block, started: make(chan struct{})}
	server, fixtures := newRepoServerWithConfig(t, []repoFixture{{name: "alpha"}}, func(config *Config) {
		config.EmbedRunner = runner
	})
	defer server.Close()

	var started embedStartResponse
	postJSON(t, server.URL+"/api/embed", `{"repo":"`+jsonEscape(fixtures[0].path)+`"}`, http.StatusAccepted, &started)
	<-runner.started

	req, err := http.NewRequest(http.MethodDelete, server.URL+"/api/embed/"+url.PathEscape(started.JobID), nil)
	if err != nil {
		t.Fatalf("new delete request: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("DELETE embed job: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("DELETE status = %d", resp.StatusCode)
	}
	close(block)

	job := waitForEmbedJob(t, server.URL, started.JobID, JobFailed)
	if job.Error != "Cancelled by user" {
		t.Fatalf("cancelled job = %#v", job)
	}
}

func TestEmbedProgressEndpointSendsCompleteEvent(t *testing.T) {
	server, fixtures := newRepoServerWithConfig(t, []repoFixture{{name: "alpha"}}, func(config *Config) {
		config.EmbedRunner = &recordingEmbedRunner{}
	})
	defer server.Close()

	var started embedStartResponse
	postJSON(t, server.URL+"/api/embed", `{"repo":"`+jsonEscape(fixtures[0].path)+`"}`, http.StatusAccepted, &started)
	waitForEmbedJob(t, server.URL, started.JobID, JobComplete)

	resp, err := http.Get(server.URL + "/api/embed/" + url.PathEscape(started.JobID) + "/progress")
	if err != nil {
		t.Fatalf("GET progress: %v", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read progress body: %v", err)
	}
	if !strings.Contains(string(body), "event: complete") {
		t.Fatalf("progress body = %q", string(body))
	}
}

func TestEmbedServiceRunsEmbeddingPipelineFromGraphSnapshot(t *testing.T) {
	tempDir := t.TempDir()
	graphPath := filepath.Join(tempDir, "graph.json")
	writeTestGraph(t, graphPath)
	runner := &embedServiceRunner{}
	service := EmbedService{
		OpenWriteRunnerWithDimensions: func(path string, dimensions int) (lbugnative.WriteRunner, error) {
			if path != filepath.Join(tempDir, "lbug") {
				t.Fatalf("lbug path = %q", path)
			}
			if dimensions != 2 {
				t.Fatalf("embedding dimensions = %d, want 2", dimensions)
			}
			return runner, nil
		},
		ResolveEmbedder: func() (embeddings.Embedder, embeddings.Config, error) {
			return staticSearchEmbedder{vector: []float32{0.1, 0.2}}, embeddings.Config{Dimensions: 2, BatchSize: 1}, nil
		},
	}

	var progress []JobProgress
	err := service.Run(context.Background(), EmbedTarget{
		RepoName:  "alpha",
		LbugPath:  filepath.Join(tempDir, "lbug"),
		GraphPath: graphPath,
	}, func(update JobProgress) {
		progress = append(progress, update)
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !runner.closed {
		t.Fatalf("runner was not closed")
	}
	if !containsQuery(runner.queries, "CREATE_VECTOR_INDEX") || !containsQuery(runner.queries, "CREATE (e:CodeEmbedding") {
		t.Fatalf("embedding queries = %#v", runner.queries)
	}
	if len(progress) == 0 || progress[len(progress)-1].Phase != string(JobComplete) {
		t.Fatalf("progress = %#v", progress)
	}
}

func TestEmbedServiceMapsNativeUnavailable(t *testing.T) {
	service := EmbedService{
		LoadGraph: func(string) (*graph.Graph, error) {
			return graph.New(), nil
		},
		OpenWriteRunner: func(string) (lbugnative.WriteRunner, error) {
			return nil, lbugnative.ErrUnavailable
		},
		ResolveEmbedder: func() (embeddings.Embedder, embeddings.Config, error) {
			return staticSearchEmbedder{vector: []float32{0.1, 0.2}}, embeddings.Config{Dimensions: 2}, nil
		},
	}

	err := service.Run(context.Background(), EmbedTarget{LbugPath: "missing", GraphPath: "graph.json"}, func(JobProgress) {})
	if !errors.Is(err, ErrEmbedUnavailable) {
		t.Fatalf("Run() error = %v, want ErrEmbedUnavailable", err)
	}
}

type recordingEmbedRunner struct {
	called  bool
	target  EmbedTarget
	block   chan struct{}
	started chan struct{}
}

func (r *recordingEmbedRunner) Run(ctx context.Context, target EmbedTarget, update func(JobProgress)) error {
	r.called = true
	r.target = target
	update(JobProgress{Phase: string(JobLoading), Percent: 5, Message: "Loading embedding runtime..."})
	if r.started != nil {
		close(r.started)
	} else {
		r.started = make(chan struct{})
		close(r.started)
	}
	if r.block != nil {
		select {
		case <-r.block:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	update(JobProgress{Phase: string(JobComplete), Percent: 100, Message: "Embeddings complete"})
	return nil
}

type embedServiceRunner struct {
	queries []string
	closed  bool
}

func (r *embedServiceRunner) Query(query string) error {
	r.queries = append(r.queries, query)
	return nil
}

func (r *embedServiceRunner) QueryRows(string) ([]lbugruntime.Row, error) {
	return nil, nil
}

func (r *embedServiceRunner) Close() error {
	r.closed = true
	return nil
}

func waitForEmbedJob(t *testing.T, serverURL string, jobID string, status JobStatus) Job {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		var job Job
		getJSON(t, serverURL+"/api/embed/"+url.PathEscape(jobID), http.StatusOK, &job)
		if job.Status == status {
			return job
		}
		time.Sleep(20 * time.Millisecond)
	}
	var job Job
	getJSON(t, serverURL+"/api/embed/"+url.PathEscape(jobID), http.StatusOK, &job)
	t.Fatalf("job status = %s, want %s; job=%#v", job.Status, status, job)
	return Job{}
}

func writeTestGraph(t *testing.T, path string) {
	t.Helper()

	raw, err := json.Marshal(graph.Graph{
		Nodes: []graph.Node{{
			ID:    "Function:alpha",
			Label: scopeir.NodeFunction,
			Properties: graph.NodeProperties{
				"name":      "alpha",
				"filePath":  "src/app.ts",
				"content":   "function alpha() { return 1 }",
				"startLine": 1,
				"endLine":   1,
			},
		}},
	})
	if err != nil {
		t.Fatalf("marshal graph: %v", err)
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatalf("write graph: %v", err)
	}
}

func containsQuery(queries []string, substring string) bool {
	for _, query := range queries {
		if strings.Contains(query, substring) {
			return true
		}
	}
	return false
}
