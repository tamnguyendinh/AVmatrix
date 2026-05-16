package session

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tamnguyendinh/avmatrix-go/internal/repo"
)

type fakeAdapter struct {
	status Status
	runs   []string
	run    func(ctx context.Context, job *Job, request ChatRequest, chatContext ChatContext) error
}

func newFakeAdapter() *fakeAdapter {
	return &fakeAdapter{
		status: Status{
			Provider:               ProviderCodex,
			Availability:           AvailabilityReady,
			Available:              true,
			Authenticated:          true,
			ExecutablePath:         "codex.cmd",
			Version:                "codex-cli test",
			RecommendedEnvironment: RuntimeWSL2,
			RuntimeEnvironment:     RuntimeNative,
			ExecutionMode:          ExecutionModeBypass,
			SupportsSSE:            true,
			SupportsCancel:         true,
			SupportsMCP:            true,
		},
	}
}

func (a *fakeAdapter) Provider() Provider                        { return ProviderCodex }
func (a *fakeAdapter) ExecutionMode() ExecutionMode              { return ExecutionModeBypass }
func (a *fakeAdapter) RuntimeEnvironment() RuntimeEnvironment    { return RuntimeNative }
func (a *fakeAdapter) GetStatus(context.Context) (Status, error) { return a.status, nil }
func (a *fakeAdapter) RunChat(ctx context.Context, job *Job, request ChatRequest, chatContext ChatContext) error {
	a.runs = append(a.runs, request.Message)
	if a.run != nil {
		return a.run(ctx, job, request, chatContext)
	}
	<-ctx.Done()
	job.Emit(Event{Type: "cancelled", Reason: cancelMessage(ctx)})
	return nil
}

func TestControllerReportsRepoBindingStatus(t *testing.T) {
	store := repo.NewStore(filepath.Join(t.TempDir(), "home"))
	indexedPath := createSessionRepo(t, store, "demo", true)

	controller := NewController(newFakeAdapter(), store)
	status, err := controller.GetStatus(context.Background(), RepoBinding{RepoName: "demo"})
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	canonicalIndexedPath, err := filepath.EvalSymlinks(indexedPath)
	if err != nil {
		t.Fatalf("canonical indexed path: %v", err)
	}
	if status.Repo == nil || status.Repo.State != "indexed" || status.Repo.ResolvedRepoName != "demo" || !samePath(status.Repo.ResolvedRepoPath, canonicalIndexedPath) {
		t.Fatalf("unexpected indexed repo status: %#v", status.Repo)
	}

	missingPath := filepath.Join(t.TempDir(), "missing")
	if err := store.WriteRegistry([]repo.RegistryEntry{{
		Name:        "missing",
		Path:        missingPath,
		StoragePath: repo.StoragePath(missingPath),
		IndexedAt:   "2026-04-20T00:00:00Z",
		LastCommit:  "abc123",
	}}); err != nil {
		t.Fatalf("write missing registry: %v", err)
	}

	status, err = controller.GetStatus(context.Background(), RepoBinding{RepoName: "missing"})
	if err != nil {
		t.Fatalf("missing status: %v", err)
	}
	if status.Repo == nil || status.Repo.State != "not_found" || !strings.Contains(status.Repo.Message, "no longer exists") {
		t.Fatalf("unexpected missing repo status: %#v", status.Repo)
	}

	unindexedPath := filepath.Join(t.TempDir(), "unindexed")
	mkdir(t, unindexedPath)
	status, err = controller.GetStatus(context.Background(), RepoBinding{RepoPath: unindexedPath})
	if err != nil {
		t.Fatalf("unindexed status: %v", err)
	}
	if status.Repo == nil || status.Repo.State != "index_required" || status.Repo.ResolvedRepoName != filepath.Base(unindexedPath) {
		t.Fatalf("unexpected unindexed repo status: %#v", status.Repo)
	}
}

func TestControllerRejectsInvalidChatStarts(t *testing.T) {
	store := repo.NewStore(filepath.Join(t.TempDir(), "home"))
	indexedPath := createSessionRepo(t, store, "demo", true)
	otherPath := createSessionRepo(t, store, "other", true)
	unindexedPath := filepath.Join(t.TempDir(), "unindexed")
	mkdir(t, unindexedPath)

	controller := NewController(newFakeAdapter(), store)
	assertRuntimeError(t, func() error {
		_, _, err := controller.StartChat(context.Background(), ChatRequest{RepoPath: unindexedPath, Message: "hello"})
		return err
	}, ErrorIndexRequired, 409)

	missingPath := filepath.Join(t.TempDir(), "missing")
	assertRuntimeError(t, func() error {
		_, _, err := controller.StartChat(context.Background(), ChatRequest{RepoPath: missingPath, Message: "hello"})
		return err
	}, ErrorRepoNotFound, 404)

	assertRuntimeError(t, func() error {
		_, _, err := controller.StartChat(context.Background(), ChatRequest{RepoName: "demo", RepoPath: otherPath, Message: "hello"})
		return err
	}, ErrorInvalidRepoBinding, 400)

	assertRuntimeError(t, func() error {
		_, _, err := controller.StartChat(context.Background(), ChatRequest{RepoPath: "https://github.com/example/repo.git", Message: "hello"})
		return err
	}, ErrorInvalidRepoPath, 400)

	assertRuntimeError(t, func() error {
		_, _, err := controller.StartChat(context.Background(), ChatRequest{RepoPath: `\\server\share\repo`, Message: "hello"})
		return err
	}, ErrorInvalidRepoPath, 400)

	if indexedPath == "" {
		t.Fatal("indexed path should be set")
	}
}

func TestControllerCancelsPreviousSessionForSameRepo(t *testing.T) {
	store := repo.NewStore(filepath.Join(t.TempDir(), "home"))
	repoPath := createSessionRepo(t, store, "demo", true)
	adapter := newFakeAdapter()
	controller := NewController(adapter, store)
	defer controller.Dispose()

	first, _, err := controller.StartChat(context.Background(), ChatRequest{RepoPath: repoPath, Message: "first"})
	if err != nil {
		t.Fatalf("start first: %v", err)
	}
	second, _, err := controller.StartChat(context.Background(), ChatRequest{RepoPath: repoPath, Message: "second"})
	if err != nil {
		t.Fatalf("start second: %v", err)
	}

	waitFor(t, func() bool { return first.Status == JobCancelled })
	if second.Status != JobRunning {
		t.Fatalf("second status = %s, want running", second.Status)
	}
	if got := strings.Join(adapter.runs, ","); got != "first,second" {
		t.Fatalf("runs = %s", got)
	}
}

func TestJobReplaysEventsAndDisposeCancelsRunningSessions(t *testing.T) {
	store := repo.NewStore(filepath.Join(t.TempDir(), "home"))
	repoPath := createSessionRepo(t, store, "demo", true)
	controller := NewController(newFakeAdapter(), store)

	job, _, err := controller.StartChat(context.Background(), ChatRequest{RepoPath: repoPath, Message: "hello"})
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	events, unsubscribe := job.Subscribe(true)
	defer unsubscribe()
	event := <-events
	if event.Type != "session_started" {
		t.Fatalf("first replayed event = %s", event.Type)
	}

	controller.Dispose()
	waitFor(t, func() bool { return job.Status == JobCancelled })
}

func createSessionRepo(t *testing.T, store repo.Store, name string, indexed bool) string {
	t.Helper()
	repoPath := filepath.Join(t.TempDir(), name)
	mkdir(t, repoPath)
	if indexed {
		if err := repo.SaveMeta(repo.StoragePath(repoPath), repo.Meta{
			RepoPath:   repoPath,
			LastCommit: "abc123",
			IndexedAt:  "2026-04-20T00:00:00Z",
		}); err != nil {
			t.Fatalf("save meta: %v", err)
		}
	}
	entries, err := store.ReadRegistry()
	if err != nil {
		t.Fatalf("read registry: %v", err)
	}
	entries = append(entries, repo.RegistryEntry{
		Name:        name,
		Path:        repoPath,
		StoragePath: repo.StoragePath(repoPath),
		IndexedAt:   "2026-04-20T00:00:00Z",
		LastCommit:  "abc123",
	})
	if err := store.WriteRegistry(entries); err != nil {
		t.Fatalf("write registry: %v", err)
	}
	return repoPath
}

func mkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
}

func assertRuntimeError(t *testing.T, fn func() error, code ErrorCode, status int) {
	t.Helper()
	err := fn()
	var runtimeErr *RuntimeError
	if !errors.As(err, &runtimeErr) {
		t.Fatalf("error = %v, want RuntimeError", err)
	}
	if runtimeErr.Code != code || runtimeErr.Status != status {
		t.Fatalf("runtime error = (%s, %d), want (%s, %d)", runtimeErr.Code, runtimeErr.Status, code, status)
	}
}

func waitFor(t *testing.T, fn func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if fn() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("condition was not met before deadline")
}
