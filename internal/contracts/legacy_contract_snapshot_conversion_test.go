package contracts

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tamnguyendinh/avmatrix-go/internal/cli"
	"github.com/tamnguyendinh/avmatrix-go/internal/httpapi"
	"github.com/tamnguyendinh/avmatrix-go/internal/mcp"
	"github.com/tamnguyendinh/avmatrix-go/internal/repo"
)

func TestLegacyPhase1ContractSnapshotSurfacesAreGoOwned(t *testing.T) {
	root := cli.NewRootCommand(cli.Options{Out: io.Discard, Err: io.Discard})
	if root.CommandPath() != "avmatrix" {
		t.Fatalf("root command path = %q, want avmatrix", root.CommandPath())
	}
	analyzeCommand, _, err := root.Find([]string{"analyze"})
	if err != nil || analyzeCommand == nil || analyzeCommand.Use != "analyze [path]" {
		t.Fatalf("analyze command = %#v, err=%v", analyzeCommand, err)
	}
	if analyzeCommand.Flags().Lookup("skip-compatibility-cross-file") == nil {
		t.Fatalf("analyze command missing --skip-compatibility-cross-file")
	}
	mcpCommand, _, err := root.Find([]string{"mcp"})
	if err != nil || mcpCommand == nil || mcpCommand.Use != "mcp" {
		t.Fatalf("mcp command = %#v, err=%v", mcpCommand, err)
	}

	manifest := WebUIContract()
	if manifest.Status != "go_owned_web_contract_generated" {
		t.Fatalf("manifest status = %q", manifest.Status)
	}
	for _, want := range []string{"typescript", "go", "cobol"} {
		if !languageValueExists(manifest.Languages.CodeLanguages, want) {
			t.Fatalf("manifest languages missing %q: %#v", want, manifest.Languages.CodeLanguages)
		}
	}
	for _, want := range []string{"idle", "extracting", "parsing", "processes", "complete", "error"} {
		if !stringSliceContains(manifest.Pipeline.Phases, want) {
			t.Fatalf("manifest pipeline phases missing %q: %#v", want, manifest.Pipeline.Phases)
		}
	}
	for _, want := range []string{"codex", "claude-code"} {
		if !stringSliceContains(manifest.Session.Providers, want) {
			t.Fatalf("manifest session providers missing %q: %#v", want, manifest.Session.Providers)
		}
	}
	if !stringSliceContains(manifest.Graph.GraphRelationshipTypes, "USES") ||
		!stringSliceContains(manifest.Graph.GraphRelationshipTypes, "INHERITS") ||
		!stringSliceContains(manifest.Graph.GraphRelationshipTypes, "DECORATES") {
		t.Fatalf("manifest graph relationships = %#v", manifest.Graph.GraphRelationshipTypes)
	}

	handler := httpapi.NewHandler(httpapi.Config{Store: repo.NewStore(t.TempDir())})
	initRequest := httptest.NewRequest(http.MethodPost, "/api/mcp", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05"}}`))
	initRequest.Header.Set("Mcp-Protocol-Version", "2024-11-05")
	initResponse := httptest.NewRecorder()
	handler.ServeHTTP(initResponse, initRequest)
	if initResponse.Code != http.StatusOK || initResponse.Header().Get("Mcp-Session-Id") == "" {
		t.Fatalf("HTTP MCP initialize status=%d session=%q body=%s", initResponse.Code, initResponse.Header().Get("Mcp-Session-Id"), initResponse.Body.String())
	}

	responses := serveMCPContractFrames(t,
		mcpContractFrame(t, map[string]any{"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": map[string]any{"protocolVersion": "2024-11-05"}}),
		mcpContractFrame(t, map[string]any{"jsonrpc": "2.0", "id": 2, "method": "tools/list"}),
		mcpContractFrame(t, map[string]any{"jsonrpc": "2.0", "id": 3, "method": "resources/templates/list"}),
		mcpContractFrame(t, map[string]any{"jsonrpc": "2.0", "id": 4, "method": "prompts/list"}),
	)
	if len(responses) != 4 {
		t.Fatalf("MCP response count = %d, want 4", len(responses))
	}
	if !mcpNamedEntryExists(responses[1]["result"].(map[string]any)["tools"].([]any), "impact") {
		t.Fatalf("tools/list missing impact: %#v", responses[1]["result"])
	}
	if !mcpURITemplateExists(responses[2]["result"].(map[string]any)["resourceTemplates"].([]any), "avmatrix://repo/{name}/schema") {
		t.Fatalf("resources/templates/list missing repo schema: %#v", responses[2]["result"])
	}
	prompts := responses[3]["result"].(map[string]any)["prompts"].([]any)
	if !mcpNamedEntryExists(prompts, "detect_impact") || !mcpNamedEntryExists(prompts, "generate_map") {
		t.Fatalf("prompts/list missing expected prompts: %#v", responses[3]["result"])
	}
}

func serveMCPContractFrames(t *testing.T, frames ...[]byte) []map[string]any {
	t.Helper()
	var output bytes.Buffer
	if err := mcp.Serve(context.Background(), bytes.NewReader(bytes.Join(frames, nil)), &output, mcp.Config{Store: repo.NewStore(t.TempDir())}); err != nil {
		t.Fatalf("mcp Serve() error = %v", err)
	}
	return readMCPContractFrames(t, output.Bytes())
}

func mcpContractFrame(t *testing.T, message map[string]any) []byte {
	t.Helper()
	raw, err := json.Marshal(message)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	return []byte(fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(raw), raw))
}

func readMCPContractFrames(t *testing.T, raw []byte) []map[string]any {
	t.Helper()
	reader := bufio.NewReader(bytes.NewReader(raw))
	var responses []map[string]any
	for {
		header, err := reader.ReadString('\n')
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("read header: %v", err)
		}
		if strings.TrimSpace(header) == "" {
			continue
		}
		if !strings.HasPrefix(strings.ToLower(header), "content-length:") {
			t.Fatalf("unexpected MCP frame header %q in %q", header, string(raw))
		}
		lengthText := strings.TrimSpace(strings.TrimPrefix(header, "Content-Length:"))
		var length int
		if _, err := fmt.Sscanf(lengthText, "%d", &length); err != nil {
			t.Fatalf("parse content length %q: %v", lengthText, err)
		}
		blank, err := reader.ReadString('\n')
		if err != nil {
			t.Fatalf("read blank line: %v", err)
		}
		if strings.TrimSpace(blank) != "" {
			t.Fatalf("expected blank line, got %q", blank)
		}
		body := make([]byte, length)
		if _, err := io.ReadFull(reader, body); err != nil {
			t.Fatalf("read body: %v", err)
		}
		var response map[string]any
		if err := json.Unmarshal(body, &response); err != nil {
			t.Fatalf("unmarshal response %s: %v", body, err)
		}
		responses = append(responses, response)
	}
	return responses
}

func mcpNamedEntryExists(entries []any, name string) bool {
	for _, entry := range entries {
		raw, ok := entry.(map[string]any)
		if ok && raw["name"] == name {
			return true
		}
	}
	return false
}

func mcpURITemplateExists(entries []any, uriTemplate string) bool {
	for _, entry := range entries {
		raw, ok := entry.(map[string]any)
		if ok && raw["uriTemplate"] == uriTemplate {
			return true
		}
	}
	return false
}

func languageValueExists(languages []LanguageContract, value string) bool {
	for _, language := range languages {
		if language.Value == value {
			return true
		}
	}
	return false
}

func stringSliceContains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
