package httpapi

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/tamnguyendinh/avmatrix-go/internal/repo"
)

func TestMCPHTTPInitializesAndListsTools(t *testing.T) {
	server, _ := newRepoServer(t, []repoFixture{{name: "alpha"}})
	defer server.Close()

	initPayload := postMCPJSON(t, server.URL+"/api/mcp", "", `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25"}}`, http.StatusOK)
	sessionID := initPayload.sessionID
	if sessionID == "" {
		t.Fatal("initialize response missing Mcp-Session-Id header")
	}
	if !strings.Contains(initPayload.contentType, "text/event-stream") {
		t.Fatalf("initialize content type = %q, want event stream", initPayload.contentType)
	}
	result := initPayload.body["result"].(map[string]any)
	if result["protocolVersion"] != "2025-11-25" {
		t.Fatalf("protocolVersion = %v, want 2025-11-25", result["protocolVersion"])
	}

	toolsPayload := postMCPJSON(t, server.URL+"/api/mcp", sessionID, `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`, http.StatusOK)
	if toolsPayload.sessionID != sessionID {
		t.Fatalf("tools/list session header = %q, want %q", toolsPayload.sessionID, sessionID)
	}
	if !strings.Contains(toolsPayload.contentType, "text/event-stream") {
		t.Fatalf("tools/list content type = %q, want event stream", toolsPayload.contentType)
	}
	toolsResult := toolsPayload.body["result"].(map[string]any)
	tools := toolsResult["tools"].([]any)
	if len(tools) != 16 {
		t.Fatalf("tools/list count = %d, want 16; payload=%#v", len(tools), toolsResult)
	}
	for _, want := range []string{"list_repos", "query", "cypher", "context", "impact", "detect_changes", "rename", "route_map", "tool_map", "shape_check", "api_impact", "group_list", "group_status", "group_sync", "group_contracts", "group_query"} {
		if !mcpHTTPToolExists(tools, want) {
			t.Fatalf("tools/list missing %s: %#v", want, toolsResult)
		}
	}
}

func TestMCPHTTPUnknownSessionReturnsJSONRPCError(t *testing.T) {
	server, _ := newRepoServer(t, nil)
	defer server.Close()

	payload := postMCPJSON(t, server.URL+"/api/mcp", "missing", `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`, http.StatusNotFound)
	errPayload := payload.body["error"].(map[string]any)
	if errPayload["code"] != float64(-32001) || !strings.Contains(errPayload["message"].(string), "Session not found") {
		t.Fatalf("unexpected MCP error payload: %#v", payload.body)
	}
}

func TestMCPHTTPRequiresInitializeBeforeSession(t *testing.T) {
	server, _ := newRepoServer(t, nil)
	defer server.Close()

	payload := postMCPJSON(t, server.URL+"/api/mcp", "", `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`, http.StatusBadRequest)
	if payload.sessionID != "" {
		t.Fatalf("tools/list before initialize created session %q", payload.sessionID)
	}
	errPayload := payload.body["error"].(map[string]any)
	if errPayload["code"] != float64(-32000) || !strings.Contains(errPayload["message"].(string), "initialize required") {
		t.Fatalf("unexpected MCP error payload: %#v", payload.body)
	}
}

func TestMCPHTTPNotificationReturnsAcceptedAfterInitialize(t *testing.T) {
	server, _ := newRepoServer(t, nil)
	defer server.Close()

	initPayload := postMCPJSON(t, server.URL+"/api/mcp", "", `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25"}}`, http.StatusOK)
	sessionID := initPayload.sessionID
	if sessionID == "" {
		t.Fatal("initialize response missing Mcp-Session-Id header")
	}

	request, err := http.NewRequest(http.MethodPost, server.URL+"/api/mcp", strings.NewReader(`{"jsonrpc":"2.0","method":"notifications/initialized"}`))
	if err != nil {
		t.Fatalf("build notification request: %v", err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json, text/event-stream")
	request.Header.Set("Mcp-Session-Id", sessionID)
	request.Header.Set("Mcp-Protocol-Version", "2025-11-25")
	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("POST /api/mcp notification failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("notification status = %d, want %d; body=%s", resp.StatusCode, http.StatusAccepted, body)
	}
	if got := resp.Header.Get("Mcp-Session-Id"); got != sessionID {
		t.Fatalf("notification session header = %q, want %q", got, sessionID)
	}
}

func TestMCPHTTPGETOpensSingleSSEStream(t *testing.T) {
	server, _ := newRepoServer(t, nil)
	defer server.Close()

	initPayload := postMCPJSON(t, server.URL+"/api/mcp", "", `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25"}}`, http.StatusOK)
	sessionID := initPayload.sessionID
	if sessionID == "" {
		t.Fatal("initialize response missing Mcp-Session-Id header")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL+"/api/mcp", nil)
	if err != nil {
		t.Fatalf("build GET /api/mcp request: %v", err)
	}
	request.Header.Set("Accept", "text/event-stream")
	request.Header.Set("Mcp-Session-Id", sessionID)
	request.Header.Set("Mcp-Protocol-Version", "2025-11-25")

	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("GET /api/mcp failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("GET /api/mcp status = %d, want %d; body=%s", resp.StatusCode, http.StatusOK, body)
	}
	if got := resp.Header.Get("Mcp-Session-Id"); got != sessionID {
		t.Fatalf("GET /api/mcp session header = %q, want %q", got, sessionID)
	}
	if !strings.Contains(resp.Header.Get("Content-Type"), "text/event-stream") {
		t.Fatalf("GET /api/mcp content type = %q, want event stream", resp.Header.Get("Content-Type"))
	}

	duplicate, err := http.NewRequest(http.MethodGet, server.URL+"/api/mcp", nil)
	if err != nil {
		t.Fatalf("build duplicate GET /api/mcp request: %v", err)
	}
	duplicate.Header.Set("Accept", "text/event-stream")
	duplicate.Header.Set("Mcp-Session-Id", sessionID)
	duplicateResp, err := http.DefaultClient.Do(duplicate)
	if err != nil {
		t.Fatalf("duplicate GET /api/mcp failed: %v", err)
	}
	defer duplicateResp.Body.Close()
	if duplicateResp.StatusCode != http.StatusConflict {
		body, _ := io.ReadAll(duplicateResp.Body)
		t.Fatalf("duplicate GET /api/mcp status = %d, want %d; body=%s", duplicateResp.StatusCode, http.StatusConflict, body)
	}
}

func TestMCPHTTPDeleteClosesSession(t *testing.T) {
	server, _ := newRepoServer(t, nil)
	defer server.Close()

	initPayload := postMCPJSON(t, server.URL+"/api/mcp", "", `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25"}}`, http.StatusOK)
	sessionID := initPayload.sessionID
	if sessionID == "" {
		t.Fatal("initialize response missing Mcp-Session-Id header")
	}

	request, err := http.NewRequest(http.MethodDelete, server.URL+"/api/mcp", nil)
	if err != nil {
		t.Fatalf("build DELETE /api/mcp request: %v", err)
	}
	request.Header.Set("Mcp-Session-Id", sessionID)
	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("DELETE /api/mcp failed: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("DELETE /api/mcp status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	payload := postMCPJSON(t, server.URL+"/api/mcp", sessionID, `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`, http.StatusNotFound)
	errPayload := payload.body["error"].(map[string]any)
	if errPayload["code"] != float64(-32001) {
		t.Fatalf("post-delete MCP error payload: %#v", payload.body)
	}
}

func TestMCPHTTPSessionsExpireIdleSessions(t *testing.T) {
	now := time.Date(2026, 5, 12, 10, 0, 0, 0, time.UTC)
	sessions := newMCPHTTPSessions(repo.NewStore(t.TempDir()))
	sessions.now = func() time.Time {
		return now
	}

	sessionID, _, err := sessions.create()
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	if _, ok := sessions.get(sessionID); !ok {
		t.Fatal("new session was not registered")
	}

	now = now.Add(mcpHTTPSessionTTL + time.Nanosecond)
	sessions.deleteExpired()
	if _, ok := sessions.get(sessionID); ok {
		t.Fatal("expired session was still registered")
	}
}

type mcpHTTPPayload struct {
	sessionID   string
	contentType string
	body        map[string]any
}

func mcpHTTPToolExists(tools []any, name string) bool {
	for _, rawTool := range tools {
		tool := rawTool.(map[string]any)
		if tool["name"] == name {
			return true
		}
	}
	return false
}

func postMCPJSON(t *testing.T, targetURL string, sessionID string, body string, wantStatus int) mcpHTTPPayload {
	t.Helper()

	request, err := http.NewRequest(http.MethodPost, targetURL, strings.NewReader(body))
	if err != nil {
		t.Fatalf("build MCP POST request: %v", err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json, text/event-stream")
	request.Header.Set("Mcp-Protocol-Version", "2025-11-25")
	if sessionID != "" {
		request.Header.Set("Mcp-Session-Id", sessionID)
	}

	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("POST %s failed: %v", targetURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != wantStatus {
		raw, _ := io.ReadAll(resp.Body)
		t.Fatalf("POST %s status = %d, want %d; body=%s", targetURL, resp.StatusCode, wantStatus, raw)
	}
	var payload map[string]any
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read %s response: %v", targetURL, err)
	}
	contentType := resp.Header.Get("Content-Type")
	switch {
	case strings.Contains(contentType, "text/event-stream"):
		payload = parseMCPSSEPayload(t, raw)
	case strings.TrimSpace(string(raw)) == "":
		payload = map[string]any{}
	default:
		if err := json.Unmarshal(raw, &payload); err != nil {
			t.Fatalf("decode %s JSON: %v; body=%s", targetURL, err, raw)
		}
	}
	return mcpHTTPPayload{
		sessionID:   resp.Header.Get("Mcp-Session-Id"),
		contentType: contentType,
		body:        payload,
	}
}

func parseMCPSSEPayload(t *testing.T, raw []byte) map[string]any {
	t.Helper()
	normalized := strings.ReplaceAll(string(raw), "\r\n", "\n")
	blocks := strings.Split(normalized, "\n\n")
	for _, block := range blocks {
		lines := strings.Split(block, "\n")
		for _, line := range lines {
			data, ok := strings.CutPrefix(line, "data: ")
			if !ok || strings.TrimSpace(data) == "" {
				continue
			}
			var payload map[string]any
			if err := json.Unmarshal([]byte(data), &payload); err != nil {
				t.Fatalf("decode MCP SSE payload: %v; block=%s", err, block)
			}
			return payload
		}
	}
	t.Fatalf("MCP SSE payload missing data event: %s", raw)
	return nil
}
