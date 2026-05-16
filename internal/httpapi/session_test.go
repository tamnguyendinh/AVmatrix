package httpapi

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/tamnguyendinh/avmatrix-go/internal/session"
)

type fakeSessionRuntime struct {
	status       session.Status
	startChat    func(request session.ChatRequest) (*session.Job, session.ResolvedRepo, error)
	cancelResult bool
	cancelCalls  []string
}

func newFakeSessionRuntime() *fakeSessionRuntime {
	return &fakeSessionRuntime{
		status: session.Status{
			Provider:               session.ProviderCodex,
			Availability:           session.AvailabilityReady,
			Available:              true,
			Authenticated:          true,
			ExecutablePath:         "codex.cmd",
			Version:                "codex-cli test",
			RecommendedEnvironment: session.RuntimeWSL2,
			RuntimeEnvironment:     session.RuntimeNative,
			ExecutionMode:          session.ExecutionModeBypass,
			SupportsSSE:            true,
			SupportsCancel:         true,
			SupportsMCP:            true,
		},
		cancelResult: true,
	}
}

func (r *fakeSessionRuntime) GetStatus(_ context.Context, binding session.RepoBinding) (session.Status, error) {
	status := r.status
	if binding.RepoName != "" || binding.RepoPath != "" {
		status.Repo = &session.RepoResolution{
			RepoName:         binding.RepoName,
			RepoPath:         binding.RepoPath,
			State:            "indexed",
			ResolvedRepoName: "demo",
			ResolvedRepoPath: "C:/demo",
		}
	}
	return status, nil
}

func (r *fakeSessionRuntime) StartChat(_ context.Context, request session.ChatRequest) (*session.Job, session.ResolvedRepo, error) {
	if r.startChat != nil {
		return r.startChat(request)
	}
	job := session.NewJob(session.ProviderCodex, "demo", "C:/demo")
	job.Emit(session.Event{
		Type:               "session_started",
		RuntimeEnvironment: session.RuntimeNative,
		ExecutionMode:      session.ExecutionModeBypass,
	})
	go func() {
		time.Sleep(10 * time.Millisecond)
		job.Emit(session.Event{Type: "content", Content: "echo:" + request.Message})
		job.Emit(session.Event{Type: "done"})
	}()
	return job, session.ResolvedRepo{RepoName: "demo", RepoPath: "C:/demo", Indexed: true}, nil
}

func (r *fakeSessionRuntime) CancelSession(sessionID string, reason string) bool {
	r.cancelCalls = append(r.cancelCalls, sessionID+"|"+reason)
	return r.cancelResult
}

func TestSessionStatusEndpointReturnsRepoBindingState(t *testing.T) {
	runtime := newFakeSessionRuntime()
	server := httptest.NewServer(NewHandler(Config{SessionRuntime: runtime}))
	defer server.Close()

	var body session.Status
	getJSON(t, server.URL+"/api/session/status?repoName=demo", http.StatusOK, &body)
	if body.Provider != session.ProviderCodex || body.Repo == nil || body.Repo.State != "indexed" || body.Repo.ResolvedRepoName != "demo" {
		t.Fatalf("unexpected status body: %#v", body)
	}
}

func TestSessionChatRejectsMissingMessage(t *testing.T) {
	server := httptest.NewServer(NewHandler(Config{SessionRuntime: newFakeSessionRuntime()}))
	defer server.Close()

	response, err := http.Post(server.URL+"/api/session/chat", "application/json", strings.NewReader(`{"repoName":"demo"}`))
	if err != nil {
		t.Fatalf("post chat: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d", response.StatusCode)
	}
	var body map[string]any
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if body["code"] != string(session.ErrorBadRequest) {
		t.Fatalf("unexpected error body: %#v", body)
	}
}

func TestSessionChatStreamsEventsOverSSE(t *testing.T) {
	server := httptest.NewServer(NewHandler(Config{SessionRuntime: newFakeSessionRuntime()}))
	defer server.Close()

	response, err := http.Post(server.URL+"/api/session/chat", "application/json", strings.NewReader(`{"repoName":"demo","message":"hello"}`))
	if err != nil {
		t.Fatalf("post chat: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("status = %d body=%s", response.StatusCode, body)
	}
	if contentType := response.Header.Get("Content-Type"); !strings.Contains(contentType, "text/event-stream") {
		t.Fatalf("content type = %q", contentType)
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	text := string(body)
	for _, want := range []string{"event: session_started", "event: content", "event: done", `"content":"echo:hello"`} {
		if !strings.Contains(text, want) {
			t.Fatalf("stream missing %q in %q", want, text)
		}
	}
}

func TestSessionChatSurfacesRuntimeErrorWithoutStream(t *testing.T) {
	runtime := newFakeSessionRuntime()
	runtime.startChat = func(session.ChatRequest) (*session.Job, session.ResolvedRepo, error) {
		return nil, session.ResolvedRepo{}, session.NewRuntimeError(
			session.ErrorIndexRequired,
			"Repository is not indexed yet. Run analyze first.",
			409,
			map[string]any{"repoPath": "C:/demo"},
		)
	}
	server := httptest.NewServer(NewHandler(Config{SessionRuntime: runtime}))
	defer server.Close()

	response, err := http.Post(server.URL+"/api/session/chat", "application/json", strings.NewReader(`{"repoPath":"C:/demo","message":"hello"}`))
	if err != nil {
		t.Fatalf("post chat: %v", err)
	}
	defer response.Body.Close()
	var body map[string]any
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if response.StatusCode != http.StatusConflict || body["code"] != string(session.ErrorIndexRequired) {
		t.Fatalf("unexpected response: status=%d body=%#v", response.StatusCode, body)
	}
}

func TestSessionDeleteCancelsKnownSessionsAnd404sUnknown(t *testing.T) {
	runtime := newFakeSessionRuntime()
	server := httptest.NewServer(NewHandler(Config{SessionRuntime: runtime}))
	defer server.Close()

	request, err := http.NewRequest(http.MethodDelete, server.URL+"/api/session/session-123", nil)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusOK || len(runtime.cancelCalls) != 1 || runtime.cancelCalls[0] != "session-123|Cancelled by user" {
		t.Fatalf("unexpected cancel: status=%d calls=%#v", response.StatusCode, runtime.cancelCalls)
	}

	runtime.cancelResult = false
	request, err = http.NewRequest(http.MethodDelete, server.URL+"/api/session/missing", nil)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	response, err = http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("delete missing: %v", err)
	}
	defer response.Body.Close()
	var body map[string]any
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode missing: %v", err)
	}
	if response.StatusCode != http.StatusNotFound || body["code"] != string(session.ErrorSessionNotFound) {
		t.Fatalf("unexpected missing response: status=%d body=%#v", response.StatusCode, body)
	}
}

func TestSessionStreamCanReplayStartedEvent(t *testing.T) {
	runtime := newFakeSessionRuntime()
	runtime.startChat = func(session.ChatRequest) (*session.Job, session.ResolvedRepo, error) {
		job := session.NewJob(session.ProviderCodex, "demo", "C:/demo")
		job.Emit(session.Event{Type: "session_started"})
		go job.Emit(session.Event{Type: "done"})
		return job, session.ResolvedRepo{RepoName: "demo", RepoPath: "C:/demo", Indexed: true}, nil
	}
	server := httptest.NewServer(NewHandler(Config{SessionRuntime: runtime}))
	defer server.Close()

	response, err := http.Post(server.URL+"/api/session/chat", "application/json", strings.NewReader(`{"repoName":"demo","message":"hello"}`))
	if err != nil {
		t.Fatalf("post chat: %v", err)
	}
	defer response.Body.Close()
	reader := bufio.NewReader(response.Body)
	line, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("read first line: %v", err)
	}
	if line != "event: session_started\n" {
		t.Fatalf("first stream line = %q", line)
	}
}
