package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestParseAction(t *testing.T) {
	tests := []struct {
		args []string
		want string
	}{
		{nil, "start"},
		{[]string{"register"}, "register"},
		{[]string{"avmatrix://reset"}, "reset"},
		{[]string{"stop"}, "stop"},
		{[]string{"unknown"}, "start"},
	}

	for _, tt := range tests {
		if got := parseAction(tt.args); got != tt.want {
			t.Fatalf("parseAction(%v) = %q, want %q", tt.args, got, tt.want)
		}
	}
}

func TestStaticHandlerServesFilesAndFallsBackToIndex(t *testing.T) {
	webDist := t.TempDir()
	if err := os.WriteFile(filepath.Join(webDist, "index.html"), []byte("index"), 0o644); err != nil {
		t.Fatalf("write index: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(webDist, "assets"), 0o755); err != nil {
		t.Fatalf("mkdir assets: %v", err)
	}
	if err := os.WriteFile(filepath.Join(webDist, "assets", "app.js"), []byte("asset"), 0o644); err != nil {
		t.Fatalf("write asset: %v", err)
	}

	server := httptest.NewServer(staticHandler(webDist))
	defer server.Close()

	assertBody(t, server.URL+"/assets/app.js", "asset")
	assertBody(t, server.URL+"/repo/detail", "index")
}

func TestStaticHandlerInjectsLauncherLifecycleAndRecordsHeartbeat(t *testing.T) {
	webDist := t.TempDir()
	if err := os.WriteFile(filepath.Join(webDist, "index.html"), []byte("<html><body>app</body></html>"), 0o644); err != nil {
		t.Fatalf("write index: %v", err)
	}
	lifecycle := newWebLifecycleMonitor(time.Second, 100*time.Millisecond)
	server := httptest.NewServer(staticHandlerWithLifecycle(webDist, lifecycle))
	defer server.Close()

	response, err := http.Get(server.URL + "/")
	if err != nil {
		t.Fatalf("GET index: %v", err)
	}
	raw, err := io.ReadAll(response.Body)
	_ = response.Body.Close()
	if err != nil {
		t.Fatalf("read index: %v", err)
	}
	body := string(raw)
	if !strings.Contains(body, "data-avmatrix-launcher-lifecycle") || !strings.Contains(body, launcherHeartbeatPath) {
		t.Fatalf("index missing launcher lifecycle script:\n%s", body)
	}

	req, err := http.NewRequest(http.MethodPost, server.URL+launcherHeartbeatPath, nil)
	if err != nil {
		t.Fatalf("heartbeat request: %v", err)
	}
	heartbeat, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST heartbeat: %v", err)
	}
	_ = heartbeat.Body.Close()
	if heartbeat.StatusCode != http.StatusNoContent {
		t.Fatalf("heartbeat status = %d, want 204", heartbeat.StatusCode)
	}
	if lifecycle.expired(time.Now().Add(500 * time.Millisecond)) {
		t.Fatalf("fresh heartbeat should keep lifecycle alive")
	}
}

func TestWebLifecycleMonitorExpiresAfterHeartbeatStops(t *testing.T) {
	lifecycle := newWebLifecycleMonitor(120*time.Millisecond, 40*time.Millisecond)
	lifecycle.start()
	defer lifecycle.stop()

	lifecycle.recordHeartbeat(time.Now().Add(-500 * time.Millisecond))
	select {
	case <-lifecycle.Done():
	case <-time.After(time.Second):
		t.Fatalf("lifecycle did not expire after heartbeat timeout")
	}
}

func TestWebLifecycleClosedSignalUsesGraceBeforeShutdown(t *testing.T) {
	lifecycle := newWebLifecycleMonitor(time.Second, 100*time.Millisecond)
	lifecycle.recordHeartbeat(time.Now())
	lifecycle.recordClosed(time.Now())

	if lifecycle.expired(time.Now().Add(50 * time.Millisecond)) {
		t.Fatalf("close signal should keep a short reload grace window")
	}
	if !lifecycle.expired(time.Now().Add(150 * time.Millisecond)) {
		t.Fatalf("close signal should expire after grace window")
	}
}

func TestVerifyWebDist(t *testing.T) {
	webDist := t.TempDir()
	if err := verifyWebDist(webDist); err == nil {
		t.Fatalf("verifyWebDist without index returned nil")
	}
	if err := os.WriteFile(filepath.Join(webDist, "index.html"), []byte("index"), 0o644); err != nil {
		t.Fatalf("write index: %v", err)
	}
	if err := verifyWebDist(webDist); err != nil {
		t.Fatalf("verifyWebDist with index: %v", err)
	}
}

func TestOpenBrowserCanBeSuppressedForSmokeTests(t *testing.T) {
	t.Setenv("AVMATRIX_LAUNCHER_NO_BROWSER", "1")
	if err := openBrowser("http://127.0.0.1:5228"); err != nil {
		t.Fatalf("openBrowser with suppression: %v", err)
	}
}

func TestWriteAndReadState(t *testing.T) {
	paths := launcherPaths{
		rootDir:   t.TempDir(),
		stateFile: filepath.Join(t.TempDir(), "launcher-state.json"),
	}

	writeState(paths, "ready", 1234)
	state, err := readState(paths)
	if err != nil {
		t.Fatalf("readState: %v", err)
	}
	if state.RootDir != paths.rootDir || state.BackendPID != 1234 || state.Status != "ready" {
		t.Fatalf("state = %#v", state)
	}
}

func assertBody(t *testing.T, url string, want string) {
	t.Helper()
	response, err := http.Get(url)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	defer response.Body.Close()
	raw, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read response %s: %v", url, err)
	}
	if string(raw) != want {
		t.Fatalf("GET %s body = %q, want %q", url, string(raw), want)
	}
}
