package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tamnguyendinh/avmatrix-go/internal/graph"
	"github.com/tamnguyendinh/avmatrix-go/internal/lbugnative"
	"github.com/tamnguyendinh/avmatrix-go/internal/lbugruntime"
	"github.com/tamnguyendinh/avmatrix-go/internal/repo"
	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
	"github.com/tamnguyendinh/avmatrix-go/internal/version"
)

func TestServeHandlesInitializeAndToolsList(t *testing.T) {
	input := bytes.Join([][]byte{
		testFrame(t, map[string]any{
			"jsonrpc": "2.0",
			"id":      1,
			"method":  "initialize",
			"params": map[string]any{
				"protocolVersion": protocolVersion,
				"clientInfo":      map[string]string{"name": "test", "version": "1.0.0"},
			},
		}),
		testFrame(t, map[string]any{
			"jsonrpc": "2.0",
			"method":  "notifications/initialized",
		}),
		testFrame(t, map[string]any{
			"jsonrpc": "2.0",
			"id":      2,
			"method":  "tools/list",
		}),
	}, nil)

	var output bytes.Buffer
	if err := Serve(context.Background(), bytes.NewReader(input), &output, Config{Store: repo.NewStore(t.TempDir())}); err != nil {
		t.Fatalf("Serve() error = %v", err)
	}
	responses := readTestFrames(t, output.Bytes())
	if len(responses) != 2 {
		t.Fatalf("response count = %d, want 2; raw=%s", len(responses), output.String())
	}

	initResult := responses[0]["result"].(map[string]any)
	if initResult["protocolVersion"] != protocolVersion {
		t.Fatalf("protocolVersion = %v, want %s", initResult["protocolVersion"], protocolVersion)
	}
	serverInfo := initResult["serverInfo"].(map[string]any)
	if serverInfo["name"] != "avmatrix" || serverInfo["version"] != version.Version {
		t.Fatalf("serverInfo = %#v", serverInfo)
	}

	toolsResult := responses[1]["result"].(map[string]any)
	tools := toolsResult["tools"].([]any)
	if len(tools) != 16 {
		t.Fatalf("tools count = %d, want 16", len(tools))
	}
	for _, want := range []string{"list_repos", "query", "cypher", "context", "impact", "detect_changes", "rename", "route_map", "tool_map", "shape_check", "api_impact", "group_list", "group_status", "group_sync", "group_contracts", "group_query"} {
		if !toolExists(tools, want) {
			t.Fatalf("tools/list missing %s: %#v", want, tools)
		}
	}
}

func TestServeInitializesAndListsToolsBeforeRepoDiscovery(t *testing.T) {
	home := t.TempDir()
	if err := os.Mkdir(filepath.Join(home, "registry.json"), 0o755); err != nil {
		t.Fatalf("make registry path unreadable: %v", err)
	}
	input := bytes.Join([][]byte{
		testFrame(t, map[string]any{
			"jsonrpc": "2.0",
			"id":      1,
			"method":  "initialize",
			"params": map[string]any{
				"protocolVersion": protocolVersion,
			},
		}),
		testFrame(t, map[string]any{
			"jsonrpc": "2.0",
			"id":      2,
			"method":  "tools/list",
		}),
	}, nil)

	var output bytes.Buffer
	if err := Serve(context.Background(), bytes.NewReader(input), &output, Config{Store: repo.NewStore(home)}); err != nil {
		t.Fatalf("Serve() error = %v", err)
	}
	responses := readTestFrames(t, output.Bytes())
	if len(responses) != 2 {
		t.Fatalf("response count = %d, want 2; raw=%s", len(responses), output.String())
	}
	if _, ok := responses[0]["result"].(map[string]any); !ok {
		t.Fatalf("initialize response missing result: %#v", responses[0])
	}
	toolsResult := responses[1]["result"].(map[string]any)
	if tools := toolsResult["tools"].([]any); !toolExists(tools, "query") || !toolExists(tools, "impact") {
		t.Fatalf("tools/list missing canonical tools: %#v", tools)
	}
}

func TestServeSuppressesScopedStdoutLeaks(t *testing.T) {
	originalStdout := os.Stdout
	leakReader, leakWriter, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() error = %v", err)
	}
	os.Stdout = leakWriter
	defer func() {
		os.Stdout = originalStdout
	}()

	input := testFrame(t, map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]any{
			"protocolVersion": protocolVersion,
		},
	})
	var output bytes.Buffer
	if err := Serve(
		context.Background(),
		bytes.NewReader(input),
		stdoutLeakingWriter{target: &output},
		Config{Store: repo.NewStore(t.TempDir())},
	); err != nil {
		t.Fatalf("Serve() error = %v", err)
	}

	os.Stdout = originalStdout
	if err := leakWriter.Close(); err != nil {
		t.Fatalf("close leak writer: %v", err)
	}
	rawLeak, err := io.ReadAll(leakReader)
	if err != nil {
		t.Fatalf("read leaked stdout: %v", err)
	}
	if strings.Contains(string(rawLeak), "leaked stdout") {
		t.Fatalf("stdout leak was not suppressed: %q", rawLeak)
	}
	if !strings.Contains(output.String(), "Content-Length:") || !strings.Contains(output.String(), `"protocolVersion"`) {
		t.Fatalf("MCP frame missing from output: %s", output.String())
	}
}

func TestServeCallToolDispatchValidationErrors(t *testing.T) {
	input := bytes.Join([][]byte{
		testFrame(t, map[string]any{
			"jsonrpc": "2.0",
			"id":      1,
			"method":  "tools/call",
			"params": map[string]any{
				"name":      "nonexistent_tool",
				"arguments": map[string]any{},
			},
		}),
		testFrame(t, map[string]any{
			"jsonrpc": "2.0",
			"id":      2,
			"method":  "tools/call",
			"params": map[string]any{
				"name":      "query",
				"arguments": map[string]any{},
			},
		}),
		testFrame(t, map[string]any{
			"jsonrpc": "2.0",
			"id":      3,
			"method":  "tools/call",
			"params": map[string]any{
				"name":      "context",
				"arguments": map[string]any{},
			},
		}),
		testFrame(t, map[string]any{
			"jsonrpc": "2.0",
			"id":      4,
			"method":  "tools/call",
			"params": map[string]any{
				"name": "impact",
				"arguments": map[string]any{
					"target":    "main",
					"direction": "sideways",
				},
			},
		}),
	}, nil)

	var output bytes.Buffer
	if err := Serve(context.Background(), bytes.NewReader(input), &output, Config{Store: repo.NewStore(t.TempDir())}); err != nil {
		t.Fatalf("Serve() error = %v", err)
	}
	responses := readTestFrames(t, output.Bytes())
	if len(responses) != 4 {
		t.Fatalf("response count = %d, want 4; raw=%s", len(responses), output.String())
	}
	unknownErr := responses[0]["error"].(map[string]any)
	if unknownErr["code"] != float64(-32602) || !strings.Contains(unknownErr["message"].(string), "Unknown tool: nonexistent_tool") {
		t.Fatalf("unknown tool error = %#v", unknownErr)
	}
	queryErr := responses[1]["error"].(map[string]any)
	if queryErr["code"] != float64(-32602) || !strings.Contains(queryErr["message"].(string), `Missing "query" argument`) {
		t.Fatalf("query validation error = %#v", queryErr)
	}
	if text := toolTextFromResponse(t, responses[2]); !strings.Contains(text, "Either") || !strings.Contains(text, "uid") {
		t.Fatalf("context validation payload = %s", text)
	}
	if text := toolTextFromResponse(t, responses[3]); !strings.Contains(text, "direction must be one of upstream, downstream") {
		t.Fatalf("impact validation payload = %s", text)
	}
}

func TestResolveResourceRepoMatchesCaseInsensitiveNamesAndRejectsAmbiguity(t *testing.T) {
	store := repo.NewStore(t.TempDir())
	firstRepo := t.TempDir()
	secondRepo := t.TempDir()
	firstMeta := repo.Meta{RepoPath: firstRepo, IndexedAt: "2026-05-15T00:00:00Z", LastCommit: "abc123", Stats: &repo.Stats{}}
	secondMeta := repo.Meta{RepoPath: secondRepo, IndexedAt: "2026-05-15T00:00:00Z", LastCommit: "def456", Stats: &repo.Stats{}}
	if err := repo.SaveMeta(repo.StoragePath(firstRepo), firstMeta); err != nil {
		t.Fatalf("save first meta: %v", err)
	}
	if err := repo.SaveMeta(repo.StoragePath(secondRepo), secondMeta); err != nil {
		t.Fatalf("save second meta: %v", err)
	}
	if _, err := store.Register(firstRepo, firstMeta, repo.RegisterOptions{Name: "fixture"}); err != nil {
		t.Fatalf("register first repo: %v", err)
	}

	server := NewServer(Config{Store: store})
	entry, err := server.resolveResourceRepo("FIXTURE")
	if err != nil {
		t.Fatalf("resolve uppercase repo name: %v", err)
	}
	if !repo.SamePath(entry.Path, firstRepo) {
		t.Fatalf("resolved path = %s, want %s", entry.Path, firstRepo)
	}

	if _, err := store.Register(secondRepo, secondMeta, repo.RegisterOptions{Name: "fixture", AllowDuplicateName: true}); err != nil {
		t.Fatalf("register duplicate repo: %v", err)
	}
	_, err = server.resolveResourceRepo("fixture")
	var ambiguous repo.AmbiguousNameError
	if !errors.As(err, &ambiguous) || ambiguous.Name != "fixture" || len(ambiguous.Matches) != 2 {
		t.Fatalf("ambiguous repo error = %#v", err)
	}
}

func TestServeCallToolDetectChanges(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git executable is not available")
	}

	store := repo.NewStore(t.TempDir())
	repoPath := t.TempDir()
	sourceDir := filepath.Join(repoPath, "src")
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		t.Fatalf("mkdir source: %v", err)
	}
	sourcePath := filepath.Join(sourceDir, "app.ts")
	if err := os.WriteFile(sourcePath, []byte("function main() {\n  return helper()\n}\n"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	runMCPTestGit(t, repoPath, "init")
	runMCPTestGit(t, repoPath, "config", "user.email", "test@example.com")
	runMCPTestGit(t, repoPath, "config", "user.name", "Test User")
	runMCPTestGit(t, repoPath, "add", "src/app.ts")
	runMCPTestGit(t, repoPath, "commit", "-m", "initial")

	meta := repo.Meta{
		RepoPath:   repoPath,
		IndexedAt:  "2026-05-12T00:00:00Z",
		LastCommit: "abc123",
		Stats:      &repo.Stats{},
	}
	if err := repo.SaveMeta(repo.StoragePath(repoPath), meta); err != nil {
		t.Fatalf("save meta: %v", err)
	}
	if _, err := store.Register(repoPath, meta, repo.RegisterOptions{Name: "fixture"}); err != nil {
		t.Fatalf("register repo: %v", err)
	}
	writeMCPResourceGraph(t, repoPath)

	if err := os.WriteFile(sourcePath, []byte("function main() {\n  return helper() + 1\n}\n"), 0o644); err != nil {
		t.Fatalf("modify source: %v", err)
	}

	input := testFrame(t, map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "detect_changes",
			"arguments": map[string]any{"repo": "fixture", "scope": "unstaged"},
		},
	})

	var output bytes.Buffer
	if err := Serve(context.Background(), bytes.NewReader(input), &output, Config{Store: store}); err != nil {
		t.Fatalf("Serve() error = %v", err)
	}
	responses := readTestFrames(t, output.Bytes())
	if len(responses) != 1 {
		t.Fatalf("response count = %d, want 1", len(responses))
	}

	var payload struct {
		Summary struct {
			ChangedCount  int    `json:"changed_count"`
			AffectedCount int    `json:"affected_count"`
			ChangedFiles  int    `json:"changed_files"`
			RiskLevel     string `json:"risk_level"`
		} `json:"summary"`
		ChangedSymbols    []map[string]any `json:"changed_symbols"`
		AffectedProcesses []map[string]any `json:"affected_processes"`
	}
	text := toolTextFromResponse(t, responses[0])
	if !strings.Contains(text, "Review affected processes") {
		t.Fatalf("detect_changes tool missing next-step hint: %s", text)
	}
	if err := json.Unmarshal([]byte(toolJSONTextFromResponse(t, responses[0])), &payload); err != nil {
		t.Fatalf("parse detect_changes JSON: %v", err)
	}
	if payload.Summary.ChangedCount != 1 || payload.Summary.AffectedCount != 1 || payload.Summary.ChangedFiles != 1 || payload.Summary.RiskLevel != "medium" {
		t.Fatalf("detect_changes summary = %#v", payload.Summary)
	}
	if len(payload.ChangedSymbols) != 1 || payload.ChangedSymbols[0]["id"] != "Function:main" {
		t.Fatalf("changed symbols = %#v", payload.ChangedSymbols)
	}
	if len(payload.AffectedProcesses) != 1 || payload.AffectedProcesses[0]["name"] != "MainFlow" {
		t.Fatalf("affected processes = %#v", payload.AffectedProcesses)
	}
}

func TestServeCallToolDetectChangesReportsDeletedSymbols(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git executable is not available")
	}

	store := repo.NewStore(t.TempDir())
	repoPath := t.TempDir()
	sourceDir := filepath.Join(repoPath, "src")
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		t.Fatalf("mkdir source: %v", err)
	}
	sourcePath := filepath.Join(sourceDir, "app.ts")
	if err := os.WriteFile(sourcePath, []byte("function main() {\n  return helper()\n}\n"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	runMCPTestGit(t, repoPath, "init")
	runMCPTestGit(t, repoPath, "config", "user.email", "test@example.com")
	runMCPTestGit(t, repoPath, "config", "user.name", "Test User")
	runMCPTestGit(t, repoPath, "add", "src/app.ts")
	runMCPTestGit(t, repoPath, "commit", "-m", "initial")

	meta := repo.Meta{
		RepoPath:   repoPath,
		IndexedAt:  "2026-05-12T00:00:00Z",
		LastCommit: "abc123",
		Stats:      &repo.Stats{},
	}
	if err := repo.SaveMeta(repo.StoragePath(repoPath), meta); err != nil {
		t.Fatalf("save meta: %v", err)
	}
	if _, err := store.Register(repoPath, meta, repo.RegisterOptions{Name: "fixture"}); err != nil {
		t.Fatalf("register repo: %v", err)
	}
	writeMCPResourceGraph(t, repoPath)
	if err := os.Remove(sourcePath); err != nil {
		t.Fatalf("delete source: %v", err)
	}

	input := testFrame(t, map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "detect_changes",
			"arguments": map[string]any{"repo": "fixture", "scope": "unstaged"},
		},
	})

	var output bytes.Buffer
	if err := Serve(context.Background(), bytes.NewReader(input), &output, Config{Store: store}); err != nil {
		t.Fatalf("Serve() error = %v", err)
	}
	responses := readTestFrames(t, output.Bytes())
	if len(responses) != 1 {
		t.Fatalf("response count = %d, want 1", len(responses))
	}
	var payload struct {
		ChangedSymbols []map[string]any `json:"changed_symbols"`
	}
	if err := json.Unmarshal([]byte(toolJSONTextFromResponse(t, responses[0])), &payload); err != nil {
		t.Fatalf("parse detect_changes JSON: %v", err)
	}
	if len(payload.ChangedSymbols) != 1 || payload.ChangedSymbols[0]["id"] != "Function:main" || payload.ChangedSymbols[0]["change_type"] != "deleted" {
		t.Fatalf("deleted changed symbols = %#v", payload.ChangedSymbols)
	}
}

func TestServeCallToolRenameDryRun(t *testing.T) {
	store := repo.NewStore(t.TempDir())
	repoPath := t.TempDir()
	sourceDir := filepath.Join(repoPath, "src")
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		t.Fatalf("mkdir source: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "app.ts"), []byte("function main() {\n  return helper()\n}\n"), 0o644); err != nil {
		t.Fatalf("write app: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "helper.ts"), []byte("function helper() {\n  return 1\n}\n"), 0o644); err != nil {
		t.Fatalf("write helper: %v", err)
	}
	meta := repo.Meta{
		RepoPath:   repoPath,
		IndexedAt:  "2026-05-12T00:00:00Z",
		LastCommit: "abc123",
		Stats:      &repo.Stats{},
	}
	if err := repo.SaveMeta(repo.StoragePath(repoPath), meta); err != nil {
		t.Fatalf("save meta: %v", err)
	}
	if _, err := store.Register(repoPath, meta, repo.RegisterOptions{Name: "fixture"}); err != nil {
		t.Fatalf("register repo: %v", err)
	}
	writeMCPResourceGraph(t, repoPath)

	input := testFrame(t, map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]any{
			"name": "rename",
			"arguments": map[string]any{
				"repo":        "fixture",
				"symbol_name": "helper",
				"new_name":    "helperRenamed",
				"dry_run":     true,
			},
		},
	})

	var output bytes.Buffer
	if err := Serve(context.Background(), bytes.NewReader(input), &output, Config{Store: store}); err != nil {
		t.Fatalf("Serve() error = %v", err)
	}
	responses := readTestFrames(t, output.Bytes())
	if len(responses) != 1 {
		t.Fatalf("response count = %d, want 1", len(responses))
	}
	var payload struct {
		Status        string `json:"status"`
		OldName       string `json:"old_name"`
		NewName       string `json:"new_name"`
		FilesAffected int    `json:"files_affected"`
		TotalEdits    int    `json:"total_edits"`
		Applied       bool   `json:"applied"`
		Changes       []struct {
			FilePath string       `json:"file_path"`
			Edits    []renameEdit `json:"edits"`
		} `json:"changes"`
	}
	text := toolTextFromResponse(t, responses[0])
	if !strings.Contains(text, "Run detect_changes") {
		t.Fatalf("rename tool missing next-step hint: %s", text)
	}
	if err := json.Unmarshal([]byte(toolJSONTextFromResponse(t, responses[0])), &payload); err != nil {
		t.Fatalf("parse rename JSON: %v", err)
	}
	if payload.Status != "success" || payload.OldName != "helper" || payload.NewName != "helperRenamed" || payload.Applied {
		t.Fatalf("rename payload = %#v", payload)
	}
	if payload.FilesAffected != 2 || payload.TotalEdits != 2 {
		t.Fatalf("rename counts = files %d edits %d; changes=%#v", payload.FilesAffected, payload.TotalEdits, payload.Changes)
	}
}

func TestServeCallToolRenameUsesResolvedRepoStoragePath(t *testing.T) {
	store := repo.NewStore(t.TempDir())
	firstRepoPath := t.TempDir()
	secondRepoPath := t.TempDir()
	meta := repo.Meta{
		IndexedAt:  "2026-05-12T00:00:00Z",
		LastCommit: "abc123",
		Stats:      &repo.Stats{},
	}
	for _, repoPath := range []string{firstRepoPath, secondRepoPath} {
		if err := repo.SaveMeta(repo.StoragePath(repoPath), meta); err != nil {
			t.Fatalf("save meta: %v", err)
		}
	}
	if _, err := store.Register(firstRepoPath, meta, repo.RegisterOptions{Name: "fixture"}); err != nil {
		t.Fatalf("register first repo: %v", err)
	}
	if _, err := store.Register(secondRepoPath, meta, repo.RegisterOptions{Name: "fixture", AllowDuplicateName: true}); err != nil {
		t.Fatalf("register second repo: %v", err)
	}
	sourceDir := filepath.Join(secondRepoPath, "src")
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		t.Fatalf("mkdir source: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "app.ts"), []byte("function main() {\n  return helper()\n}\n"), 0o644); err != nil {
		t.Fatalf("write app: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "helper.ts"), []byte("function helper() {\n  return 1\n}\n"), 0o644); err != nil {
		t.Fatalf("write helper: %v", err)
	}
	writeMCPResourceGraph(t, secondRepoPath)

	input := testFrame(t, map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]any{
			"name": "rename",
			"arguments": map[string]any{
				"repo":        secondRepoPath,
				"symbol_name": "helper",
				"new_name":    "helperRenamed",
				"dry_run":     true,
			},
		},
	})

	var output bytes.Buffer
	if err := Serve(context.Background(), bytes.NewReader(input), &output, Config{Store: store}); err != nil {
		t.Fatalf("Serve() error = %v", err)
	}
	responses := readTestFrames(t, output.Bytes())
	if len(responses) != 1 {
		t.Fatalf("response count = %d, want 1", len(responses))
	}
	var payload struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal([]byte(toolJSONTextFromResponse(t, responses[0])), &payload); err != nil {
		t.Fatalf("parse rename JSON: %v", err)
	}
	if payload.Status != "success" {
		t.Fatalf("rename payload = %#v", payload)
	}
}

func TestApplyRenameChangesUsesTargetLine(t *testing.T) {
	repoPath := t.TempDir()
	sourceDir := filepath.Join(repoPath, "src")
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		t.Fatalf("mkdir source: %v", err)
	}
	sourcePath := filepath.Join(sourceDir, "app.ts")
	if err := os.WriteFile(sourcePath, []byte("call(helper)\ncall(helper)\n"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	err := applyRenameChanges(repoPath, []renameChange{{
		FilePath: "src/app.ts",
		Edits: []renameEdit{{
			Line:    2,
			OldText: "call(helper)",
			NewText: "call(helperRenamed)",
			oldLine: "call(helper)",
			newLine: "call(helperRenamed)",
		}},
	}})
	if err != nil {
		t.Fatalf("applyRenameChanges() error = %v", err)
	}
	raw, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatalf("read source: %v", err)
	}
	if got, want := string(raw), "call(helper)\ncall(helperRenamed)\n"; got != want {
		t.Fatalf("source after rename = %q, want %q", got, want)
	}
}

func TestServeCallToolRenameUsesReferenceLineForSameFileCalls(t *testing.T) {
	store := repo.NewStore(t.TempDir())
	repoPath := t.TempDir()
	sourceDir := filepath.Join(repoPath, "src")
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		t.Fatalf("mkdir source: %v", err)
	}
	sourcePath := filepath.Join(sourceDir, "app.ts")
	source := strings.Join([]string{
		"function helper() {",
		"  return 1;",
		"}",
		"",
		"function main() {",
		"  const helperLabel = \"helper\";",
		"  return helper();",
		"}",
		"",
	}, "\n")
	if err := os.WriteFile(sourcePath, []byte(source), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	meta := repo.Meta{
		RepoPath:   repoPath,
		IndexedAt:  "2026-05-12T00:00:00Z",
		LastCommit: "abc123",
		Stats:      &repo.Stats{},
	}
	if err := repo.SaveMeta(repo.StoragePath(repoPath), meta); err != nil {
		t.Fatalf("save meta: %v", err)
	}
	if _, err := store.Register(repoPath, meta, repo.RegisterOptions{Name: "fixture"}); err != nil {
		t.Fatalf("register repo: %v", err)
	}
	writeMCPSameFileRenameGraph(t, repoPath)

	dryRun := callMCPRename(t, store, "fixture", "helper", "renamedHelper", true)
	if dryRun.TotalEdits != 2 || dryRun.GraphEdits != 2 || dryRun.TextSearchEdits != 0 || dryRun.Applied {
		t.Fatalf("dry-run counts = %#v", dryRun)
	}
	if len(dryRun.Changes) != 1 || len(dryRun.Changes[0].Edits) != 2 {
		t.Fatalf("dry-run changes = %#v", dryRun.Changes)
	}
	seenLines := map[int]bool{}
	for _, edit := range dryRun.Changes[0].Edits {
		seenLines[edit.Line] = true
	}
	if !seenLines[1] || !seenLines[7] || seenLines[6] {
		t.Fatalf("dry-run edit lines = %#v", dryRun.Changes[0].Edits)
	}

	applied := callMCPRename(t, store, "fixture", "helper", "renamedHelper", false)
	if !applied.Applied || applied.TotalEdits != 2 {
		t.Fatalf("applied payload = %#v", applied)
	}
	raw, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatalf("read source: %v", err)
	}
	got := string(raw)
	if !strings.Contains(got, "function renamedHelper() {") || !strings.Contains(got, "return renamedHelper();") {
		t.Fatalf("source missing renamed definition/call:\n%s", got)
	}
	if !strings.Contains(got, "const helperLabel = \"helper\";") {
		t.Fatalf("unrelated text should remain unchanged:\n%s", got)
	}
}

func TestServeCallToolListRepos(t *testing.T) {
	store := repo.NewStore(t.TempDir())
	repoPath := t.TempDir()
	meta := repo.Meta{
		RepoPath:   repoPath,
		IndexedAt:  "2026-05-12T00:00:00Z",
		LastCommit: "abc123",
		Stats:      &repo.Stats{},
	}
	if _, err := store.Register(repoPath, meta, repo.RegisterOptions{Name: "fixture"}); err != nil {
		t.Fatalf("register repo: %v", err)
	}
	entries, err := store.ListRegistered(false)
	if err != nil {
		t.Fatalf("read registry: %v", err)
	}
	expectedPath := entries[0].Path

	input := testFrame(t, map[string]any{
		"jsonrpc": "2.0",
		"id":      "call-1",
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "list_repos",
			"arguments": map[string]any{},
		},
	})

	var output bytes.Buffer
	if err := Serve(context.Background(), bytes.NewReader(input), &output, Config{Store: store}); err != nil {
		t.Fatalf("Serve() error = %v", err)
	}
	responses := readTestFrames(t, output.Bytes())
	if len(responses) != 1 {
		t.Fatalf("response count = %d, want 1", len(responses))
	}
	result := responses[0]["result"].(map[string]any)
	content := result["content"].([]any)
	text := content[0].(map[string]any)["text"].(string)
	jsonText, _, _ := strings.Cut(text, "\n\n---")
	var repos []listReposResult
	if err := json.Unmarshal([]byte(jsonText), &repos); err != nil {
		t.Fatalf("parse list_repos JSON: %v; text=%s", err, text)
	}
	if len(repos) != 1 || repos[0].Name != "fixture" || repos[0].Path != expectedPath {
		t.Fatalf("list_repos parsed result = %#v, want fixture at %q", repos, expectedPath)
	}
	if !strings.Contains(text, "Next:") || !strings.Contains(text, "avmatrix://repo/{name}/context") {
		t.Fatalf("list_repos text missing next-step hint: %s", text)
	}
}

func TestServeCallToolsQueryAndCypher(t *testing.T) {
	store := repo.NewStore(t.TempDir())
	repoPath := t.TempDir()
	meta := repo.Meta{
		RepoPath:   repoPath,
		IndexedAt:  "2026-05-12T00:00:00Z",
		LastCommit: "abc123",
		Stats:      &repo.Stats{},
	}
	if err := repo.SaveMeta(repo.StoragePath(repoPath), meta); err != nil {
		t.Fatalf("save meta: %v", err)
	}
	if _, err := store.Register(repoPath, meta, repo.RegisterOptions{Name: "fixture"}); err != nil {
		t.Fatalf("register repo: %v", err)
	}
	writeMCPResourceGraph(t, repoPath)

	input := bytes.Join([][]byte{
		testFrame(t, map[string]any{
			"jsonrpc": "2.0",
			"id":      1,
			"method":  "tools/call",
			"params": map[string]any{
				"name":      "query",
				"arguments": map[string]any{"repo": "fixture", "query": "MainFlow", "limit": 2},
			},
		}),
		testFrame(t, map[string]any{
			"jsonrpc": "2.0",
			"id":      2,
			"method":  "tools/call",
			"params": map[string]any{
				"name": "cypher",
				"arguments": map[string]any{
					"repo":  "fixture",
					"query": "MATCH (n:Function) RETURN n.id AS id, n.name AS name, n.filePath AS path LIMIT 5",
				},
			},
		}),
		testFrame(t, map[string]any{
			"jsonrpc": "2.0",
			"id":      3,
			"method":  "tools/call",
			"params": map[string]any{
				"name":      "context",
				"arguments": map[string]any{"repo": "fixture", "name": "main"},
			},
		}),
		testFrame(t, map[string]any{
			"jsonrpc": "2.0",
			"id":      4,
			"method":  "tools/call",
			"params": map[string]any{
				"name":      "impact",
				"arguments": map[string]any{"repo": "fixture", "target": "helper", "direction": "upstream"},
			},
		}),
		testFrame(t, map[string]any{
			"jsonrpc": "2.0",
			"id":      5,
			"method":  "tools/call",
			"params": map[string]any{
				"name":      "cypher",
				"arguments": map[string]any{"repo": "fixture", "query": "MERGE (n:File {id: 'x'}) RETURN n"},
			},
		}),
	}, nil)

	var output bytes.Buffer
	if err := Serve(context.Background(), bytes.NewReader(input), &output, Config{Store: store}); err != nil {
		t.Fatalf("Serve() error = %v", err)
	}
	responses := readTestFrames(t, output.Bytes())
	if len(responses) != 5 {
		t.Fatalf("response count = %d, want 5", len(responses))
	}

	var queryPayload struct {
		Query          string            `json:"query"`
		Processes      []resourceProcess `json:"processes"`
		ProcessSymbols []map[string]any  `json:"process_symbols"`
	}
	if err := json.Unmarshal([]byte(toolJSONTextFromResponse(t, responses[0])), &queryPayload); err != nil {
		t.Fatalf("parse query tool JSON: %v", err)
	}
	if queryPayload.Query != "MainFlow" || len(queryPayload.Processes) != 1 || queryPayload.Processes[0].Label != "MainFlow" {
		t.Fatalf("query payload = %#v", queryPayload)
	}
	if len(queryPayload.ProcessSymbols) != 2 {
		t.Fatalf("process symbols count = %d, want 2", len(queryPayload.ProcessSymbols))
	}
	if queryPayload.ProcessSymbols[0]["id"] != "Function:main" {
		t.Fatalf("first process symbol = %#v, want Function:main id", queryPayload.ProcessSymbols[0])
	}

	var cypherPayload struct {
		RowCount int              `json:"row_count"`
		Rows     []map[string]any `json:"rows"`
		Markdown string           `json:"markdown"`
	}
	if err := json.Unmarshal([]byte(toolJSONTextFromResponse(t, responses[1])), &cypherPayload); err != nil {
		t.Fatalf("parse cypher tool JSON: %v", err)
	}
	if cypherPayload.RowCount != 2 || !strings.Contains(cypherPayload.Markdown, "Function:main") {
		t.Fatalf("cypher payload = %#v", cypherPayload)
	}

	var contextPayload struct {
		Status    string                      `json:"status"`
		Symbol    map[string]any              `json:"symbol"`
		Outgoing  map[string][]map[string]any `json:"outgoing"`
		Processes []map[string]any            `json:"processes"`
		Incoming  map[string][]map[string]any `json:"incoming"`
	}
	contextText := toolTextFromResponse(t, responses[2])
	if !strings.Contains(contextText, "impact({target: \"main\"") {
		t.Fatalf("context tool missing next-step hint: %s", contextText)
	}
	if err := json.Unmarshal([]byte(toolJSONTextFromResponse(t, responses[2])), &contextPayload); err != nil {
		t.Fatalf("parse context tool JSON: %v", err)
	}
	if contextPayload.Status != "found" || contextPayload.Symbol["uid"] != "Function:main" {
		t.Fatalf("context payload = %#v", contextPayload)
	}
	if len(contextPayload.Outgoing["calls"]) != 1 || contextPayload.Outgoing["calls"][0]["uid"] != "Function:helper" {
		t.Fatalf("context outgoing calls = %#v", contextPayload.Outgoing)
	}
	if len(contextPayload.Processes) != 1 || contextPayload.Processes[0]["name"] != "MainFlow" {
		t.Fatalf("context processes = %#v", contextPayload.Processes)
	}

	var impactPayload struct {
		Direction         string                      `json:"direction"`
		ImpactedCount     int                         `json:"impactedCount"`
		Risk              string                      `json:"risk"`
		ByDepth           map[string][]map[string]any `json:"byDepth"`
		AffectedProcesses []map[string]any            `json:"affected_processes"`
		AffectedModules   []map[string]any            `json:"affected_modules"`
	}
	impactText := toolTextFromResponse(t, responses[3])
	if !strings.Contains(impactText, "Review d=1 items first") {
		t.Fatalf("impact tool missing next-step hint: %s", impactText)
	}
	if err := json.Unmarshal([]byte(toolJSONTextFromResponse(t, responses[3])), &impactPayload); err != nil {
		t.Fatalf("parse impact tool JSON: %v", err)
	}
	if impactPayload.Direction != "upstream" || impactPayload.ImpactedCount != 1 || impactPayload.Risk != "LOW" {
		t.Fatalf("impact payload = %#v", impactPayload)
	}
	if len(impactPayload.ByDepth["1"]) != 1 || impactPayload.ByDepth["1"][0]["id"] != "Function:main" {
		t.Fatalf("impact byDepth = %#v", impactPayload.ByDepth)
	}
	if len(impactPayload.AffectedProcesses) != 1 || impactPayload.AffectedProcesses[0]["name"] != "MainFlow" {
		t.Fatalf("impact affected processes = %#v", impactPayload.AffectedProcesses)
	}
	if len(impactPayload.AffectedModules) != 1 || impactPayload.AffectedModules[0]["name"] != "Api" {
		t.Fatalf("impact affected modules = %#v", impactPayload.AffectedModules)
	}

	errPayload := responses[4]["error"].(map[string]any)
	if errPayload["code"] != float64(-32602) || !strings.Contains(errPayload["message"].(string), "write operations are not allowed") {
		t.Fatalf("write query error = %#v", errPayload)
	}
}

func TestServeCallToolCypherFallbackRejectsUnsupportedRelationshipPredicates(t *testing.T) {
	store := repo.NewStore(t.TempDir())
	repoPath := t.TempDir()
	meta := repo.Meta{
		RepoPath:   repoPath,
		IndexedAt:  "2026-05-12T00:00:00Z",
		LastCommit: "abc123",
		Stats:      &repo.Stats{},
	}
	if err := repo.SaveMeta(repo.StoragePath(repoPath), meta); err != nil {
		t.Fatalf("save meta: %v", err)
	}
	if _, err := store.Register(repoPath, meta, repo.RegisterOptions{Name: "fixture"}); err != nil {
		t.Fatalf("register repo: %v", err)
	}
	writeMCPResourceGraph(t, repoPath)

	input := testFrame(t, map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]any{
			"name": "cypher",
			"arguments": map[string]any{
				"repo":  "fixture",
				"query": `MATCH (a)-[:CodeRelation {type: 'CALLS'}]->(b:Function {name: "doesNotExist"}) RETURN a.name, a.filePath`,
			},
		},
	})

	var output bytes.Buffer
	if err := Serve(context.Background(), bytes.NewReader(input), &output, Config{Store: store}); err != nil {
		t.Fatalf("Serve() error = %v", err)
	}
	responses := readTestFrames(t, output.Bytes())
	if len(responses) != 1 {
		t.Fatalf("response count = %d, want 1", len(responses))
	}
	errPayload := responses[0]["error"].(map[string]any)
	if errPayload["code"] != float64(-32602) || !strings.Contains(errPayload["message"].(string), "unsupported graph query") {
		t.Fatalf("unsupported cypher error = %#v", errPayload)
	}
}

func TestServeCallToolCypherUsesReadRunner(t *testing.T) {
	store := repo.NewStore(t.TempDir())
	repoPath := t.TempDir()
	meta := repo.Meta{
		RepoPath:   repoPath,
		IndexedAt:  "2026-05-12T00:00:00Z",
		LastCommit: "abc123",
		Stats:      &repo.Stats{},
	}
	if err := repo.SaveMeta(repo.StoragePath(repoPath), meta); err != nil {
		t.Fatalf("save meta: %v", err)
	}
	if _, err := store.Register(repoPath, meta, repo.RegisterOptions{Name: "fixture"}); err != nil {
		t.Fatalf("register repo: %v", err)
	}

	fake := &fakeCypherReadRunner{
		rows: []lbugruntime.Row{{"id": "Function:main", "name": "main"}},
	}
	var openedPath string
	config := Config{
		Store: store,
		OpenReadRunner: func(path string) (lbugnative.ReadRunner, error) {
			openedPath = path
			return fake, nil
		},
	}
	input := testFrame(t, map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "cypher",
			"arguments": map[string]any{"repo": "fixture", "query": "MATCH (n) RETURN n.id AS id, n.name AS name"},
		},
	})

	var output bytes.Buffer
	if err := Serve(context.Background(), bytes.NewReader(input), &output, config); err != nil {
		t.Fatalf("Serve() error = %v", err)
	}
	responses := readTestFrames(t, output.Bytes())
	if len(responses) != 1 {
		t.Fatalf("response count = %d, want 1", len(responses))
	}
	var payload struct {
		RowCount int              `json:"row_count"`
		Rows     []map[string]any `json:"rows"`
		Markdown string           `json:"markdown"`
	}
	if err := json.Unmarshal([]byte(toolJSONTextFromResponse(t, responses[0])), &payload); err != nil {
		t.Fatalf("parse cypher JSON: %v", err)
	}
	if openedPath != filepath.Join(repo.StoragePath(repoPath), "lbug") {
		t.Fatalf("open read runner path = %q", openedPath)
	}
	if fake.query != "MATCH (n) RETURN n.id AS id, n.name AS name" || !fake.closed {
		t.Fatalf("fake runner query/closed = %q/%v", fake.query, fake.closed)
	}
	if payload.RowCount != 1 || payload.Rows[0]["id"] != "Function:main" || !strings.Contains(payload.Markdown, "Function:main") {
		t.Fatalf("cypher payload = %#v", payload)
	}
}

func TestServeCallToolContextExpandsClassIncomingRefs(t *testing.T) {
	store := repo.NewStore(t.TempDir())
	repoPath := t.TempDir()
	meta := repo.Meta{
		RepoPath:   repoPath,
		IndexedAt:  "2026-05-12T00:00:00Z",
		LastCommit: "abc123",
		Stats:      &repo.Stats{},
	}
	if err := repo.SaveMeta(repo.StoragePath(repoPath), meta); err != nil {
		t.Fatalf("save meta: %v", err)
	}
	if _, err := store.Register(repoPath, meta, repo.RegisterOptions{Name: "fixture"}); err != nil {
		t.Fatalf("register repo: %v", err)
	}
	writeMCPClassContextGraph(t, repoPath)

	input := testFrame(t, map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "context",
			"arguments": map[string]any{"repo": "fixture", "name": "User", "kind": "Class"},
		},
	})

	var output bytes.Buffer
	if err := Serve(context.Background(), bytes.NewReader(input), &output, Config{Store: store}); err != nil {
		t.Fatalf("Serve() error = %v", err)
	}
	responses := readTestFrames(t, output.Bytes())
	if len(responses) != 1 {
		t.Fatalf("response count = %d, want 1", len(responses))
	}
	var payload struct {
		Incoming map[string][]map[string]any `json:"incoming"`
	}
	if err := json.Unmarshal([]byte(toolJSONTextFromResponse(t, responses[0])), &payload); err != nil {
		t.Fatalf("parse context JSON: %v", err)
	}
	if len(payload.Incoming["calls"]) != 1 || payload.Incoming["calls"][0]["uid"] != "Function:makeUser" {
		t.Fatalf("expanded constructor calls = %#v", payload.Incoming)
	}
	if len(payload.Incoming["imports"]) != 1 || payload.Incoming["imports"][0]["uid"] != "File:src/consumer.ts" {
		t.Fatalf("expanded file imports = %#v", payload.Incoming)
	}
}

func TestServeCallToolRouteAndToolMaps(t *testing.T) {
	store := repo.NewStore(t.TempDir())
	repoPath := t.TempDir()
	meta := repo.Meta{
		RepoPath:   repoPath,
		IndexedAt:  "2026-05-12T00:00:00Z",
		LastCommit: "abc123",
		Stats:      &repo.Stats{},
	}
	if err := repo.SaveMeta(repo.StoragePath(repoPath), meta); err != nil {
		t.Fatalf("save meta: %v", err)
	}
	if _, err := store.Register(repoPath, meta, repo.RegisterOptions{Name: "fixture"}); err != nil {
		t.Fatalf("register repo: %v", err)
	}
	writeMCPResourceGraph(t, repoPath)

	input := bytes.Join([][]byte{
		testFrame(t, map[string]any{
			"jsonrpc": "2.0",
			"id":      1,
			"method":  "tools/call",
			"params": map[string]any{
				"name":      "route_map",
				"arguments": map[string]any{"repo": "fixture", "route": "/api/users"},
			},
		}),
		testFrame(t, map[string]any{
			"jsonrpc": "2.0",
			"id":      2,
			"method":  "tools/call",
			"params": map[string]any{
				"name":      "shape_check",
				"arguments": map[string]any{"repo": "fixture", "route": "/api/users"},
			},
		}),
		testFrame(t, map[string]any{
			"jsonrpc": "2.0",
			"id":      3,
			"method":  "tools/call",
			"params": map[string]any{
				"name":      "api_impact",
				"arguments": map[string]any{"repo": "fixture", "route": "/api/users"},
			},
		}),
		testFrame(t, map[string]any{
			"jsonrpc": "2.0",
			"id":      4,
			"method":  "tools/call",
			"params": map[string]any{
				"name":      "tool_map",
				"arguments": map[string]any{"repo": "fixture", "tool": "search"},
			},
		}),
	}, nil)

	var output bytes.Buffer
	if err := Serve(context.Background(), bytes.NewReader(input), &output, Config{Store: store}); err != nil {
		t.Fatalf("Serve() error = %v", err)
	}
	responses := readTestFrames(t, output.Bytes())
	if len(responses) != 4 {
		t.Fatalf("response count = %d, want 4", len(responses))
	}

	var routePayload struct {
		Routes []mcpRouteMapItem `json:"routes"`
		Total  int               `json:"total"`
	}
	routeText := toolTextFromResponse(t, responses[0])
	if !strings.Contains(routeText, "api_impact") {
		t.Fatalf("route_map tool missing next-step hint: %s", routeText)
	}
	if err := json.Unmarshal([]byte(toolJSONTextFromResponse(t, responses[0])), &routePayload); err != nil {
		t.Fatalf("parse route_map JSON: %v", err)
	}
	if routePayload.Total != 1 || len(routePayload.Routes) != 1 {
		t.Fatalf("route_map payload = %#v", routePayload)
	}
	route := routePayload.Routes[0]
	if route.Route != "/api/users" || route.Handler != "src/server.ts" || len(route.Consumers) != 1 || route.Consumers[0].FilePath != "src/app.ts" {
		t.Fatalf("route_map route = %#v", route)
	}
	if len(route.Middleware) != 1 || route.Middleware[0] != "withAuth" || len(route.Flows) != 1 || route.Flows[0] != "MainFlow" {
		t.Fatalf("route_map middleware/flows = %#v", route)
	}
	if len(route.Consumers[0].AccessedKeys) != 3 || route.Consumers[0].FetchCount != 2 {
		t.Fatalf("route_map consumer keys/count = %#v", route.Consumers[0])
	}

	var shapePayload struct {
		Routes     []mcpShapeRoute `json:"routes"`
		Total      int             `json:"total"`
		Mismatches int             `json:"mismatches"`
	}
	shapeText := toolTextFromResponse(t, responses[1])
	if !strings.Contains(shapeText, "api_impact") {
		t.Fatalf("shape_check tool missing next-step hint: %s", shapeText)
	}
	if err := json.Unmarshal([]byte(toolJSONTextFromResponse(t, responses[1])), &shapePayload); err != nil {
		t.Fatalf("parse shape_check JSON: %v", err)
	}
	if shapePayload.Total != 1 || shapePayload.Mismatches != 1 || len(shapePayload.Routes) != 1 {
		t.Fatalf("shape_check payload = %#v", shapePayload)
	}
	if shapePayload.Routes[0].Status != "MISMATCH" || len(shapePayload.Routes[0].Consumers[0].Mismatched) != 1 || shapePayload.Routes[0].Consumers[0].Mismatched[0] != "missing" {
		t.Fatalf("shape_check route = %#v", shapePayload.Routes[0])
	}

	var apiPayload mcpAPIImpactRoute
	apiText := toolTextFromResponse(t, responses[2])
	if !strings.Contains(apiText, "Review direct consumers") {
		t.Fatalf("api_impact tool missing next-step hint: %s", apiText)
	}
	if err := json.Unmarshal([]byte(toolJSONTextFromResponse(t, responses[2])), &apiPayload); err != nil {
		t.Fatalf("parse api_impact JSON: %v", err)
	}
	if apiPayload.Route != "/api/users" || len(apiPayload.Mismatches) != 1 || apiPayload.Mismatches[0].Field != "missing" {
		t.Fatalf("api_impact payload = %#v", apiPayload)
	}
	if apiPayload.ImpactSummary["riskLevel"] != "MEDIUM" || apiPayload.ImpactSummary["directConsumers"] != float64(1) {
		t.Fatalf("api_impact summary = %#v", apiPayload.ImpactSummary)
	}

	var toolPayload struct {
		Tools []mcpToolMapItem `json:"tools"`
		Total int              `json:"total"`
	}
	toolText := toolTextFromResponse(t, responses[3])
	if !strings.Contains(toolText, "context(") {
		t.Fatalf("tool_map tool missing next-step hint: %s", toolText)
	}
	if err := json.Unmarshal([]byte(toolJSONTextFromResponse(t, responses[3])), &toolPayload); err != nil {
		t.Fatalf("parse tool_map JSON: %v", err)
	}
	if toolPayload.Total != 1 || len(toolPayload.Tools) != 1 {
		t.Fatalf("tool_map payload = %#v", toolPayload)
	}
	tool := toolPayload.Tools[0]
	if tool.Name != "search" || tool.FilePath != "src/tools.ts" || tool.Description != "Search code intelligence" {
		t.Fatalf("tool_map tool = %#v", tool)
	}
	if len(tool.Flows) != 1 || tool.Flows[0] != "MainFlow" {
		t.Fatalf("tool_map flows = %#v", tool)
	}
}

func TestRouteIndexCacheInvalidatesWhenGraphChanges(t *testing.T) {
	store := repo.NewStore(t.TempDir())
	repoPath := t.TempDir()
	meta := repo.Meta{RepoPath: repoPath, IndexedAt: "2026-05-13T00:00:00Z", LastCommit: "abc123", Stats: &repo.Stats{}}
	if err := repo.SaveMeta(repo.StoragePath(repoPath), meta); err != nil {
		t.Fatalf("save meta: %v", err)
	}
	if _, err := store.Register(repoPath, meta, repo.RegisterOptions{Name: "fixture"}); err != nil {
		t.Fatalf("register repo: %v", err)
	}

	writeMCPRouteOnlyGraph(t, repoPath, "/api/users")
	server := NewServer(Config{Store: store})
	first, err := server.routeMapTool(map[string]any{"repo": "fixture", "route": "/api/users"})
	if err != nil {
		t.Fatalf("first route_map: %v", err)
	}
	if routes, ok := first["routes"].([]mcpRouteMapItem); !ok || len(routes) != 1 || routes[0].Route != "/api/users" {
		t.Fatalf("first route_map routes = %#v", first["routes"])
	}

	time.Sleep(10 * time.Millisecond)
	writeMCPRouteOnlyGraph(t, repoPath, "/api/orders")
	stale, err := server.routeMapTool(map[string]any{"repo": "fixture", "route": "/api/users"})
	if err != nil {
		t.Fatalf("stale route_map: %v", err)
	}
	if total, ok := stale["total"].(int); !ok || total != 0 {
		t.Fatalf("stale route_map total = %#v, want 0", stale["total"])
	}
	next, err := server.routeMapTool(map[string]any{"repo": "fixture", "route": "/api/orders"})
	if err != nil {
		t.Fatalf("next route_map: %v", err)
	}
	if routes, ok := next["routes"].([]mcpRouteMapItem); !ok || len(routes) != 1 || routes[0].Route != "/api/orders" {
		t.Fatalf("next route_map routes = %#v", next["routes"])
	}
}

func TestServeCallToolGroupListAndStatus(t *testing.T) {
	store := repo.NewStore(t.TempDir())
	repoPath := t.TempDir()
	meta := repo.Meta{
		RepoPath:   repoPath,
		IndexedAt:  "2026-05-12T00:00:00Z",
		LastCommit: "abc123",
		Stats:      &repo.Stats{},
	}
	if err := repo.SaveMeta(repo.StoragePath(repoPath), meta); err != nil {
		t.Fatalf("save meta: %v", err)
	}
	if _, err := store.Register(repoPath, meta, repo.RegisterOptions{Name: "test-backend"}); err != nil {
		t.Fatalf("register repo: %v", err)
	}
	writeMCPResourceGraph(t, repoPath)
	groupDir := filepath.Join(store.HomeDir, "groups", "test-group")
	if err := os.MkdirAll(groupDir, 0o755); err != nil {
		t.Fatalf("mkdir group: %v", err)
	}
	groupYAML := `version: 1
name: test-group
description: "fixture group"

repos:
  app/backend: test-backend
  app/frontend: test-frontend

links: []
`
	if err := os.WriteFile(filepath.Join(groupDir, "group.yaml"), []byte(groupYAML), 0o644); err != nil {
		t.Fatalf("write group.yaml: %v", err)
	}
	contractsJSON := map[string]any{
		"version":     1,
		"generatedAt": "2026-05-12T01:00:00Z",
		"repoSnapshots": map[string]any{
			"app/backend": map[string]string{"indexedAt": "2026-05-12T00:00:00Z", "lastCommit": "abc123"},
		},
		"missingRepos": []string{"app/frontend"},
		"contracts": []map[string]any{
			{
				"repo":       "app/backend",
				"contractId": "http::GET::/users",
				"type":       "http",
				"role":       "provider",
				"symbolUid":  "Route:/api/users",
				"symbolRef":  map[string]string{"filePath": "src/server.ts", "name": "/api/users"},
				"symbolName": "/api/users",
				"confidence": 1.0,
				"meta":       map[string]any{"source": "fixture"},
			},
			{
				"repo":       "app/frontend",
				"contractId": "http::GET::/users",
				"type":       "http",
				"role":       "consumer",
				"symbolUid":  "File:src/app.ts",
				"symbolRef":  map[string]string{"filePath": "src/app.ts", "name": "fetchUsers"},
				"symbolName": "fetchUsers",
				"confidence": 0.95,
				"meta":       map[string]any{"source": "fixture"},
			},
			{
				"repo":       "app/backend",
				"contractId": "topic::orphan",
				"type":       "topic",
				"role":       "provider",
				"symbolUid":  "Function:helper",
				"symbolRef":  map[string]string{"filePath": "src/helper.ts", "name": "helper"},
				"symbolName": "helper",
				"confidence": 0.75,
				"meta":       map[string]any{"source": "fixture"},
			},
		},
		"crossLinks": []map[string]any{
			{
				"from":       map[string]any{"repo": "app/frontend", "symbolUid": "File:src/app.ts", "symbolRef": map[string]string{"filePath": "src/app.ts", "name": "fetchUsers"}},
				"to":         map[string]any{"repo": "app/backend", "symbolUid": "Route:/api/users", "symbolRef": map[string]string{"filePath": "src/server.ts", "name": "/api/users"}},
				"type":       "http",
				"contractId": "http::GET::/users",
				"matchType":  "exact",
				"confidence": 1.0,
			},
		},
	}
	contractsRaw, err := json.MarshalIndent(contractsJSON, "", "  ")
	if err != nil {
		t.Fatalf("marshal contracts: %v", err)
	}
	if err := os.WriteFile(filepath.Join(groupDir, "contracts.json"), append(contractsRaw, '\n'), 0o644); err != nil {
		t.Fatalf("write contracts.json: %v", err)
	}

	input := bytes.Join([][]byte{
		testFrame(t, map[string]any{
			"jsonrpc": "2.0",
			"id":      1,
			"method":  "tools/call",
			"params": map[string]any{
				"name":      "group_list",
				"arguments": map[string]any{},
			},
		}),
		testFrame(t, map[string]any{
			"jsonrpc": "2.0",
			"id":      2,
			"method":  "tools/call",
			"params": map[string]any{
				"name":      "group_list",
				"arguments": map[string]any{"name": "test-group"},
			},
		}),
		testFrame(t, map[string]any{
			"jsonrpc": "2.0",
			"id":      3,
			"method":  "tools/call",
			"params": map[string]any{
				"name":      "group_status",
				"arguments": map[string]any{"name": "test-group"},
			},
		}),
		testFrame(t, map[string]any{
			"jsonrpc": "2.0",
			"id":      4,
			"method":  "tools/call",
			"params": map[string]any{
				"name":      "group_contracts",
				"arguments": map[string]any{"name": "test-group", "type": "http"},
			},
		}),
		testFrame(t, map[string]any{
			"jsonrpc": "2.0",
			"id":      5,
			"method":  "tools/call",
			"params": map[string]any{
				"name":      "group_contracts",
				"arguments": map[string]any{"name": "test-group", "unmatchedOnly": true},
			},
		}),
		testFrame(t, map[string]any{
			"jsonrpc": "2.0",
			"id":      6,
			"method":  "tools/call",
			"params": map[string]any{
				"name":      "group_query",
				"arguments": map[string]any{"name": "test-group", "query": "MainFlow", "limit": 2},
			},
		}),
	}, nil)

	var output bytes.Buffer
	if err := Serve(context.Background(), bytes.NewReader(input), &output, Config{Store: store}); err != nil {
		t.Fatalf("Serve() error = %v", err)
	}
	responses := readTestFrames(t, output.Bytes())
	if len(responses) != 6 {
		t.Fatalf("response count = %d, want 6", len(responses))
	}

	var listPayload struct {
		Groups []string `json:"groups"`
	}
	listText := toolTextFromResponse(t, responses[0])
	if !strings.Contains(listText, "group_status") {
		t.Fatalf("group_list tool missing next-step hint: %s", listText)
	}
	if err := json.Unmarshal([]byte(toolJSONTextFromResponse(t, responses[0])), &listPayload); err != nil {
		t.Fatalf("parse group_list JSON: %v", err)
	}
	if len(listPayload.Groups) != 1 || listPayload.Groups[0] != "test-group" {
		t.Fatalf("group list payload = %#v", listPayload)
	}

	var detailPayload struct {
		Name        string            `json:"name"`
		Description string            `json:"description"`
		Repos       map[string]string `json:"repos"`
	}
	if err := json.Unmarshal([]byte(toolJSONTextFromResponse(t, responses[1])), &detailPayload); err != nil {
		t.Fatalf("parse group detail JSON: %v", err)
	}
	if detailPayload.Name != "test-group" || detailPayload.Description != "fixture group" || detailPayload.Repos["app/backend"] != "test-backend" {
		t.Fatalf("group detail payload = %#v", detailPayload)
	}

	var statusPayload struct {
		Group string `json:"group"`
		Repos map[string]struct {
			IndexStale     bool `json:"indexStale"`
			ContractsStale bool `json:"contractsStale"`
			Missing        bool `json:"missing"`
		} `json:"repos"`
	}
	statusText := toolTextFromResponse(t, responses[2])
	if !strings.Contains(statusText, "group_sync") {
		t.Fatalf("group_status tool missing next-step hint: %s", statusText)
	}
	if err := json.Unmarshal([]byte(toolJSONTextFromResponse(t, responses[2])), &statusPayload); err != nil {
		t.Fatalf("parse group_status JSON: %v", err)
	}
	if statusPayload.Group != "test-group" || statusPayload.Repos["app/backend"].Missing || statusPayload.Repos["app/backend"].ContractsStale {
		t.Fatalf("group_status backend = %#v", statusPayload)
	}
	if !statusPayload.Repos["app/frontend"].Missing {
		t.Fatalf("group_status frontend should be missing: %#v", statusPayload)
	}

	var contractsPayload struct {
		Contracts []struct {
			Repo       string `json:"repo"`
			ContractID string `json:"contractId"`
			Type       string `json:"type"`
		} `json:"contracts"`
		CrossLinks []struct {
			ContractID string `json:"contractId"`
		} `json:"crossLinks"`
	}
	contractsText := toolTextFromResponse(t, responses[3])
	if !strings.Contains(contractsText, "group_query") {
		t.Fatalf("group_contracts tool missing next-step hint: %s", contractsText)
	}
	if err := json.Unmarshal([]byte(toolJSONTextFromResponse(t, responses[3])), &contractsPayload); err != nil {
		t.Fatalf("parse group_contracts JSON: %v", err)
	}
	if len(contractsPayload.Contracts) != 2 || contractsPayload.Contracts[0].Type != "http" || len(contractsPayload.CrossLinks) != 1 {
		t.Fatalf("group_contracts payload = %#v", contractsPayload)
	}

	var unmatchedPayload struct {
		Contracts []struct {
			ContractID string `json:"contractId"`
			Type       string `json:"type"`
		} `json:"contracts"`
	}
	if err := json.Unmarshal([]byte(toolJSONTextFromResponse(t, responses[4])), &unmatchedPayload); err != nil {
		t.Fatalf("parse unmatched group_contracts JSON: %v", err)
	}
	if len(unmatchedPayload.Contracts) != 1 || unmatchedPayload.Contracts[0].ContractID != "topic::orphan" {
		t.Fatalf("unmatched group_contracts payload = %#v", unmatchedPayload)
	}

	var queryPayload struct {
		Group   string `json:"group"`
		Query   string `json:"query"`
		Results []struct {
			Repo  string `json:"_repo"`
			Label string `json:"label"`
		} `json:"results"`
		PerRepo []struct {
			Repo  string `json:"repo"`
			Count int    `json:"count"`
		} `json:"per_repo"`
	}
	queryText := toolTextFromResponse(t, responses[5])
	if !strings.Contains(queryText, "group_contracts") {
		t.Fatalf("group_query tool missing next-step hint: %s", queryText)
	}
	if err := json.Unmarshal([]byte(toolJSONTextFromResponse(t, responses[5])), &queryPayload); err != nil {
		t.Fatalf("parse group_query JSON: %v", err)
	}
	if queryPayload.Group != "test-group" || queryPayload.Query != "MainFlow" || len(queryPayload.Results) != 1 || queryPayload.Results[0].Repo != "app/backend" || queryPayload.Results[0].Label != "MainFlow" {
		t.Fatalf("group_query payload = %#v", queryPayload)
	}
	if len(queryPayload.PerRepo) != 2 || queryPayload.PerRepo[0].Count != 1 || queryPayload.PerRepo[1].Count != 0 {
		t.Fatalf("group_query per_repo = %#v", queryPayload.PerRepo)
	}
}

func TestServeCallToolGroupSync(t *testing.T) {
	store := repo.NewStore(t.TempDir())
	backendPath := t.TempDir()
	frontendPath := t.TempDir()
	meta := repo.Meta{
		IndexedAt:  "2026-05-12T00:00:00Z",
		LastCommit: "abc123",
		Stats:      &repo.Stats{},
	}
	meta.RepoPath = backendPath
	if err := repo.SaveMeta(repo.StoragePath(backendPath), meta); err != nil {
		t.Fatalf("save backend meta: %v", err)
	}
	if _, err := store.Register(backendPath, meta, repo.RegisterOptions{Name: "test-backend"}); err != nil {
		t.Fatalf("register backend: %v", err)
	}
	meta.RepoPath = frontendPath
	if err := repo.SaveMeta(repo.StoragePath(frontendPath), meta); err != nil {
		t.Fatalf("save frontend meta: %v", err)
	}
	if _, err := store.Register(frontendPath, meta, repo.RegisterOptions{Name: "test-frontend"}); err != nil {
		t.Fatalf("register frontend: %v", err)
	}
	writeMCPGroupSyncProviderGraph(t, backendPath)
	writeMCPGroupSyncConsumerGraph(t, frontendPath)

	groupDir := filepath.Join(store.HomeDir, "groups", "test-group")
	if err := os.MkdirAll(groupDir, 0o755); err != nil {
		t.Fatalf("mkdir group: %v", err)
	}
	groupYAML := `version: 1
name: test-group
description: "fixture group"

repos:
  app/backend: test-backend
  app/frontend: test-frontend

links: []
`
	if err := os.WriteFile(filepath.Join(groupDir, "group.yaml"), []byte(groupYAML), 0o644); err != nil {
		t.Fatalf("write group.yaml: %v", err)
	}

	input := bytes.Join([][]byte{
		testFrame(t, map[string]any{
			"jsonrpc": "2.0",
			"id":      1,
			"method":  "tools/call",
			"params": map[string]any{
				"name":      "group_sync",
				"arguments": map[string]any{"name": "test-group", "exactOnly": true, "skipEmbeddings": true},
			},
		}),
		testFrame(t, map[string]any{
			"jsonrpc": "2.0",
			"id":      2,
			"method":  "tools/call",
			"params": map[string]any{
				"name":      "group_contracts",
				"arguments": map[string]any{"name": "test-group"},
			},
		}),
	}, nil)

	var output bytes.Buffer
	if err := Serve(context.Background(), bytes.NewReader(input), &output, Config{Store: store}); err != nil {
		t.Fatalf("Serve() error = %v", err)
	}
	responses := readTestFrames(t, output.Bytes())
	if len(responses) != 2 {
		t.Fatalf("response count = %d, want 2", len(responses))
	}

	var syncPayload struct {
		Contracts    int      `json:"contracts"`
		CrossLinks   int      `json:"crossLinks"`
		Unmatched    int      `json:"unmatched"`
		MissingRepos []string `json:"missingRepos"`
	}
	syncText := toolTextFromResponse(t, responses[0])
	if !strings.Contains(syncText, "group_contracts") {
		t.Fatalf("group_sync tool missing next-step hint: %s", syncText)
	}
	if err := json.Unmarshal([]byte(toolJSONTextFromResponse(t, responses[0])), &syncPayload); err != nil {
		t.Fatalf("parse group_sync JSON: %v", err)
	}
	if syncPayload.Contracts != 2 || syncPayload.CrossLinks != 1 || syncPayload.Unmatched != 0 || len(syncPayload.MissingRepos) != 0 {
		t.Fatalf("group_sync payload = %#v", syncPayload)
	}

	var contractsPayload struct {
		Contracts  []map[string]any `json:"contracts"`
		CrossLinks []map[string]any `json:"crossLinks"`
	}
	if err := json.Unmarshal([]byte(toolJSONTextFromResponse(t, responses[1])), &contractsPayload); err != nil {
		t.Fatalf("parse group_contracts JSON: %v", err)
	}
	if len(contractsPayload.Contracts) != 2 || len(contractsPayload.CrossLinks) != 1 {
		t.Fatalf("synced group_contracts payload = %#v", contractsPayload)
	}
}

func TestServeReadsRepoContextResource(t *testing.T) {
	store := repo.NewStore(t.TempDir())
	repoPath := t.TempDir()
	files, nodes, processes := 7, 11, 13
	meta := repo.Meta{
		RepoPath:   repoPath,
		IndexedAt:  "2026-05-12T00:00:00Z",
		LastCommit: "abcdef123456",
		Stats: &repo.Stats{
			Files:     &files,
			Nodes:     &nodes,
			Processes: &processes,
		},
	}
	if err := repo.SaveMeta(repo.StoragePath(repoPath), meta); err != nil {
		t.Fatalf("save meta: %v", err)
	}
	if _, err := store.Register(repoPath, meta, repo.RegisterOptions{Name: "fixture repo"}); err != nil {
		t.Fatalf("register repo: %v", err)
	}

	contextURI := "avmatrix://repo/" + url.PathEscape("fixture repo") + "/context"
	input := bytes.Join([][]byte{
		testFrame(t, map[string]any{
			"jsonrpc": "2.0",
			"id":      1,
			"method":  "resources/templates/list",
		}),
		testFrame(t, map[string]any{
			"jsonrpc": "2.0",
			"id":      2,
			"method":  "resources/read",
			"params":  map[string]any{"uri": contextURI},
		}),
	}, nil)

	var output bytes.Buffer
	if err := Serve(context.Background(), bytes.NewReader(input), &output, Config{Store: store}); err != nil {
		t.Fatalf("Serve() error = %v", err)
	}
	responses := readTestFrames(t, output.Bytes())
	if len(responses) != 2 {
		t.Fatalf("response count = %d, want 2", len(responses))
	}

	templatesResult := responses[0]["result"].(map[string]any)
	templates := templatesResult["resourceTemplates"].([]any)
	var foundContext bool
	for _, rawTemplate := range templates {
		template := rawTemplate.(map[string]any)
		foundContext = foundContext || template["uriTemplate"] == "avmatrix://repo/{name}/context"
	}
	if !foundContext {
		t.Fatalf("resource templates = %#v", templatesResult)
	}

	readResult := responses[1]["result"].(map[string]any)
	contents := readResult["contents"].([]any)
	content := contents[0].(map[string]any)
	if content["uri"] != contextURI || content["mimeType"] != "text/yaml" {
		t.Fatalf("context content metadata = %#v", content)
	}
	text := content["text"].(string)
	for _, want := range []string{"project: fixture repo", "files: 7", "symbols: 11", "processes: 13", "commit: abcdef1", "list_repos"} {
		if !strings.Contains(text, want) {
			t.Fatalf("context resource missing %q:\n%s", want, text)
		}
	}
}

func TestServeReadsRepoContextResourceWarnsWhenIndexIsStale(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git executable is not available")
	}

	store := repo.NewStore(t.TempDir())
	repoPath := t.TempDir()
	if err := os.WriteFile(filepath.Join(repoPath, "app.ts"), []byte("export const value = 1\n"), 0o644); err != nil {
		t.Fatalf("write initial source: %v", err)
	}
	runMCPTestGit(t, repoPath, "init")
	runMCPTestGit(t, repoPath, "config", "user.email", "test@example.com")
	runMCPTestGit(t, repoPath, "config", "user.name", "Test User")
	runMCPTestGit(t, repoPath, "add", "app.ts")
	runMCPTestGit(t, repoPath, "commit", "-m", "initial")
	indexedCommit := repo.CurrentCommit(repoPath)
	if indexedCommit == "" {
		t.Fatal("test git repository has no initial commit")
	}

	if err := os.WriteFile(filepath.Join(repoPath, "app.ts"), []byte("export const value = 2\n"), 0o644); err != nil {
		t.Fatalf("write stale source: %v", err)
	}
	runMCPTestGit(t, repoPath, "add", "app.ts")
	runMCPTestGit(t, repoPath, "commit", "-m", "second")

	meta := repo.Meta{
		RepoPath:   repoPath,
		IndexedAt:  "2026-05-12T00:00:00Z",
		LastCommit: indexedCommit,
		Stats:      &repo.Stats{},
	}
	if err := repo.SaveMeta(repo.StoragePath(repoPath), meta); err != nil {
		t.Fatalf("save meta: %v", err)
	}
	if _, err := store.Register(repoPath, meta, repo.RegisterOptions{Name: "fixture"}); err != nil {
		t.Fatalf("register repo: %v", err)
	}

	input := testFrame(t, map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "resources/read",
		"params":  map[string]any{"uri": "avmatrix://repo/fixture/context"},
	})

	var output bytes.Buffer
	if err := Serve(context.Background(), bytes.NewReader(input), &output, Config{Store: store}); err != nil {
		t.Fatalf("Serve() error = %v", err)
	}
	responses := readTestFrames(t, output.Bytes())
	if len(responses) != 1 {
		t.Fatalf("response count = %d, want 1", len(responses))
	}
	text := resourceTextFromResponse(t, responses[0])
	if !strings.Contains(text, `staleness: "⚠️ Index is 1 commit behind HEAD. Run analyze tool to update."`) {
		t.Fatalf("context resource missing stale-index warning:\n%s", text)
	}
}

func TestServeReadsRepoClustersAndProcessesResources(t *testing.T) {
	store := repo.NewStore(t.TempDir())
	repoPath := t.TempDir()
	meta := repo.Meta{
		RepoPath:   repoPath,
		IndexedAt:  "2026-05-12T00:00:00Z",
		LastCommit: "abcdef123456",
		Stats:      &repo.Stats{},
	}
	if err := repo.SaveMeta(repo.StoragePath(repoPath), meta); err != nil {
		t.Fatalf("save meta: %v", err)
	}
	if _, err := store.Register(repoPath, meta, repo.RegisterOptions{Name: "fixture"}); err != nil {
		t.Fatalf("register repo: %v", err)
	}
	writeMCPResourceGraph(t, repoPath)

	input := bytes.Join([][]byte{
		testFrame(t, map[string]any{
			"jsonrpc": "2.0",
			"id":      1,
			"method":  "resources/templates/list",
		}),
		testFrame(t, map[string]any{
			"jsonrpc": "2.0",
			"id":      2,
			"method":  "resources/read",
			"params":  map[string]any{"uri": "avmatrix://repo/fixture/clusters"},
		}),
		testFrame(t, map[string]any{
			"jsonrpc": "2.0",
			"id":      3,
			"method":  "resources/read",
			"params":  map[string]any{"uri": "avmatrix://repo/fixture/processes"},
		}),
	}, nil)

	var output bytes.Buffer
	if err := Serve(context.Background(), bytes.NewReader(input), &output, Config{Store: store}); err != nil {
		t.Fatalf("Serve() error = %v", err)
	}
	responses := readTestFrames(t, output.Bytes())
	if len(responses) != 3 {
		t.Fatalf("response count = %d, want 3", len(responses))
	}

	templatesResult := responses[0]["result"].(map[string]any)
	templates := templatesResult["resourceTemplates"].([]any)
	var foundClusters, foundProcesses bool
	for _, rawTemplate := range templates {
		template := rawTemplate.(map[string]any)
		foundClusters = foundClusters || template["uriTemplate"] == "avmatrix://repo/{name}/clusters"
		foundProcesses = foundProcesses || template["uriTemplate"] == "avmatrix://repo/{name}/processes"
	}
	if !foundClusters || !foundProcesses {
		t.Fatalf("resource templates missing clusters/processes: %#v", templatesResult)
	}

	clustersText := resourceTextFromResponse(t, responses[1])
	for _, want := range []string{"modules:", "Api", "symbols: 5", "cohesion: 80%"} {
		if !strings.Contains(clustersText, want) {
			t.Fatalf("clusters resource missing %q:\n%s", want, clustersText)
		}
	}

	processesText := resourceTextFromResponse(t, responses[2])
	for _, want := range []string{"processes:", "MainFlow", "type: cross_community", "steps: 2"} {
		if !strings.Contains(processesText, want) {
			t.Fatalf("processes resource missing %q:\n%s", want, processesText)
		}
	}
}

func TestServeReadsSchemaDetailResourcesAndPrompts(t *testing.T) {
	store := repo.NewStore(t.TempDir())
	repoPath := t.TempDir()
	meta := repo.Meta{
		RepoPath:   repoPath,
		IndexedAt:  "2026-05-12T00:00:00Z",
		LastCommit: "abcdef123456",
		Stats:      &repo.Stats{},
	}
	if err := repo.SaveMeta(repo.StoragePath(repoPath), meta); err != nil {
		t.Fatalf("save meta: %v", err)
	}
	if _, err := store.Register(repoPath, meta, repo.RegisterOptions{Name: "fixture"}); err != nil {
		t.Fatalf("register repo: %v", err)
	}
	writeMCPResourceGraph(t, repoPath)

	input := bytes.Join([][]byte{
		testFrame(t, map[string]any{"jsonrpc": "2.0", "id": 1, "method": "resources/list"}),
		testFrame(t, map[string]any{"jsonrpc": "2.0", "id": 2, "method": "resources/templates/list"}),
		testFrame(t, map[string]any{"jsonrpc": "2.0", "id": 3, "method": "resources/read", "params": map[string]any{"uri": "avmatrix://repos"}}),
		testFrame(t, map[string]any{"jsonrpc": "2.0", "id": 4, "method": "resources/read", "params": map[string]any{"uri": "avmatrix://setup"}}),
		testFrame(t, map[string]any{"jsonrpc": "2.0", "id": 5, "method": "resources/read", "params": map[string]any{"uri": "avmatrix://repo/fixture/schema"}}),
		testFrame(t, map[string]any{"jsonrpc": "2.0", "id": 6, "method": "resources/read", "params": map[string]any{"uri": "avmatrix://repo/fixture/cluster/Api"}}),
		testFrame(t, map[string]any{"jsonrpc": "2.0", "id": 7, "method": "resources/read", "params": map[string]any{"uri": "avmatrix://repo/fixture/process/MainFlow"}}),
		testFrame(t, map[string]any{"jsonrpc": "2.0", "id": 8, "method": "prompts/list"}),
		testFrame(t, map[string]any{"jsonrpc": "2.0", "id": 9, "method": "prompts/get", "params": map[string]any{"name": "generate_map", "arguments": map[string]any{"repo": "fixture"}}}),
	}, nil)

	var output bytes.Buffer
	if err := Serve(context.Background(), bytes.NewReader(input), &output, Config{Store: store}); err != nil {
		t.Fatalf("Serve() error = %v", err)
	}
	responses := readTestFrames(t, output.Bytes())
	if len(responses) != 9 {
		t.Fatalf("response count = %d, want 9", len(responses))
	}

	resources := responses[0]["result"].(map[string]any)["resources"].([]any)
	if len(resources) != 2 {
		t.Fatalf("resources/list = %#v", responses[0]["result"])
	}
	templates := responses[1]["result"].(map[string]any)["resourceTemplates"].([]any)
	for _, want := range []string{"avmatrix://repo/{name}/schema", "avmatrix://repo/{name}/cluster/{clusterName}", "avmatrix://repo/{name}/process/{processName}"} {
		if !resourceTemplateExists(templates, want) {
			t.Fatalf("resource templates missing %s: %#v", want, templates)
		}
	}

	for index, want := range map[int]string{
		3: "repos:",
		4: "AVmatrix MCP - fixture",
		5: "INHERITS",
		6: "members:",
		7: "trace:",
	} {
		text := resourceTextFromResponse(t, responses[index-1])
		if !strings.Contains(text, want) {
			t.Fatalf("response %d missing %q:\n%s", index, want, text)
		}
	}

	prompts := responses[7]["result"].(map[string]any)["prompts"].([]any)
	if !promptExists(prompts, "detect_impact") || !promptExists(prompts, "generate_map") {
		t.Fatalf("prompts/list = %#v", responses[7]["result"])
	}
	promptMessages := responses[8]["result"].(map[string]any)["messages"].([]any)
	promptText := promptMessages[0].(map[string]any)["content"].(map[string]any)["text"].(string)
	if !strings.Contains(promptText, "avmatrix://repo/fixture/clusters") {
		t.Fatalf("generate_map prompt = %s", promptText)
	}
}

func testFrame(t *testing.T, message map[string]any) []byte {
	t.Helper()
	raw, err := json.Marshal(message)
	if err != nil {
		t.Fatalf("marshal frame: %v", err)
	}
	return []byte(fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(raw), raw))
}

func resourceTemplateExists(templates []any, uriTemplate string) bool {
	for _, rawTemplate := range templates {
		template := rawTemplate.(map[string]any)
		if template["uriTemplate"] == uriTemplate {
			return true
		}
	}
	return false
}

func toolExists(tools []any, name string) bool {
	for _, rawTool := range tools {
		tool := rawTool.(map[string]any)
		if tool["name"] == name {
			return true
		}
	}
	return false
}

func promptExists(prompts []any, name string) bool {
	for _, rawPrompt := range prompts {
		prompt := rawPrompt.(map[string]any)
		if prompt["name"] == name {
			return true
		}
	}
	return false
}

func resourceTextFromResponse(t *testing.T, response map[string]any) string {
	t.Helper()
	result := response["result"].(map[string]any)
	contents := result["contents"].([]any)
	return contents[0].(map[string]any)["text"].(string)
}

func toolTextFromResponse(t *testing.T, response map[string]any) string {
	t.Helper()
	result := response["result"].(map[string]any)
	content := result["content"].([]any)
	return content[0].(map[string]any)["text"].(string)
}

func toolJSONTextFromResponse(t *testing.T, response map[string]any) string {
	t.Helper()
	text := toolTextFromResponse(t, response)
	jsonText, _, _ := strings.Cut(text, "\n\n---")
	return jsonText
}

type mcpRenamePayload struct {
	Status          string `json:"status"`
	OldName         string `json:"old_name"`
	NewName         string `json:"new_name"`
	FilesAffected   int    `json:"files_affected"`
	TotalEdits      int    `json:"total_edits"`
	GraphEdits      int    `json:"graph_edits"`
	TextSearchEdits int    `json:"text_search_edits"`
	Applied         bool   `json:"applied"`
	Changes         []struct {
		FilePath string       `json:"file_path"`
		Edits    []renameEdit `json:"edits"`
	} `json:"changes"`
}

func callMCPRename(t *testing.T, store repo.Store, repoName string, symbolName string, newName string, dryRun bool) mcpRenamePayload {
	t.Helper()
	input := testFrame(t, map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]any{
			"name": "rename",
			"arguments": map[string]any{
				"repo":        repoName,
				"symbol_name": symbolName,
				"new_name":    newName,
				"dry_run":     dryRun,
			},
		},
	})
	var output bytes.Buffer
	if err := Serve(context.Background(), bytes.NewReader(input), &output, Config{Store: store}); err != nil {
		t.Fatalf("Serve() error = %v", err)
	}
	responses := readTestFrames(t, output.Bytes())
	if len(responses) != 1 {
		t.Fatalf("response count = %d, want 1", len(responses))
	}
	if errPayload, ok := responses[0]["error"].(map[string]any); ok {
		t.Fatalf("rename returned JSON-RPC error: %#v", errPayload)
	}
	var payload mcpRenamePayload
	if err := json.Unmarshal([]byte(toolJSONTextFromResponse(t, responses[0])), &payload); err != nil {
		t.Fatalf("parse rename JSON: %v", err)
	}
	if payload.Status != "success" {
		t.Fatalf("rename payload = %#v", payload)
	}
	return payload
}

func writeMCPResourceGraph(t *testing.T, repoPath string) {
	t.Helper()
	g := graph.New()
	g.AddNode(graph.Node{ID: "File:src/app.ts", Label: scopeir.NodeFile, Properties: graph.NodeProperties{
		"name": "app.ts", "filePath": "src/app.ts",
	}})
	g.AddNode(graph.Node{ID: "Function:main", Label: scopeir.NodeFunction, Properties: graph.NodeProperties{
		"name": "main", "filePath": "src/app.ts", "startLine": 1, "endLine": 3,
	}})
	g.AddNode(graph.Node{ID: "Function:helper", Label: scopeir.NodeFunction, Properties: graph.NodeProperties{
		"name": "helper", "filePath": "src/helper.ts", "startLine": 1, "endLine": 3,
	}})
	g.AddNode(graph.Node{ID: "Process:main", Label: scopeir.NodeProcess, Properties: graph.NodeProperties{
		"name": "MainFlow", "label": "MainFlow", "heuristicLabel": "MainFlow", "processType": "cross_community", "stepCount": 2,
	}})
	g.AddNode(graph.Node{ID: "comm_api", Label: scopeir.NodeCommunity, Properties: graph.NodeProperties{
		"name": "Api", "label": "Api", "heuristicLabel": "Api", "cohesion": 0.8, "symbolCount": 5,
	}})
	g.AddNode(graph.Node{ID: "Route:/api/users", Label: scopeir.NodeRoute, Properties: graph.NodeProperties{
		"name": "/api/users", "filePath": "src/server.ts", "responseKeys": []string{"data", "total"}, "errorKeys": []string{"error"}, "middleware": []string{"withAuth"},
	}})
	g.AddNode(graph.Node{ID: "Tool:search", Label: scopeir.NodeTool, Properties: graph.NodeProperties{
		"name": "search", "filePath": "src/tools.ts", "description": "Search code intelligence",
	}})
	mainStep := 1
	helperStep := 2
	g.AddRelationship(graph.Relationship{ID: "rel:main-process", SourceID: "Function:main", TargetID: "Process:main", Type: graph.RelStepInProcess, Step: &mainStep})
	g.AddRelationship(graph.Relationship{ID: "rel:helper-process", SourceID: "Function:helper", TargetID: "Process:main", Type: graph.RelStepInProcess, Step: &helperStep})
	g.AddRelationship(graph.Relationship{ID: "rel:main-helper", SourceID: "Function:main", TargetID: "Function:helper", Type: graph.RelCalls, Confidence: 0.95, Reason: "fixture"})
	g.AddRelationship(graph.Relationship{ID: "rel:main-member", SourceID: "Function:main", TargetID: "comm_api", Type: graph.RelMemberOf})
	g.AddRelationship(graph.Relationship{ID: "rel:helper-member", SourceID: "Function:helper", TargetID: "comm_api", Type: graph.RelMemberOf})
	g.AddRelationship(graph.Relationship{ID: "rel:app-fetch-users", SourceID: "File:src/app.ts", TargetID: "Route:/api/users", Type: graph.RelFetches, Confidence: 0.9, Reason: "fetch-route|keys:data,total,missing|fetches:2"})
	g.AddRelationship(graph.Relationship{ID: "rel:route-main-flow", SourceID: "Route:/api/users", TargetID: "Process:main", Type: graph.RelEntryPointOf, Confidence: 0.8, Reason: "fixture"})
	g.AddRelationship(graph.Relationship{ID: "rel:tool-main-flow", SourceID: "Tool:search", TargetID: "Process:main", Type: graph.RelEntryPointOf, Confidence: 0.8, Reason: "fixture"})
	raw, err := json.MarshalIndent(g, "", "  ")
	if err != nil {
		t.Fatalf("marshal graph: %v", err)
	}
	graphPath := filepath.Join(repo.StoragePath(repoPath), "graph.json")
	if err := os.MkdirAll(filepath.Dir(graphPath), 0o755); err != nil {
		t.Fatalf("mkdir graph: %v", err)
	}
	if err := os.WriteFile(graphPath, append(raw, '\n'), 0o644); err != nil {
		t.Fatalf("write graph: %v", err)
	}
}

func writeMCPClassContextGraph(t *testing.T, repoPath string) {
	t.Helper()
	g := graph.New()
	g.AddNode(graph.Node{ID: "File:src/user.ts", Label: scopeir.NodeFile, Properties: graph.NodeProperties{
		"name": "user.ts", "filePath": "src/user.ts",
	}})
	g.AddNode(graph.Node{ID: "File:src/consumer.ts", Label: scopeir.NodeFile, Properties: graph.NodeProperties{
		"name": "consumer.ts", "filePath": "src/consumer.ts",
	}})
	g.AddNode(graph.Node{ID: "Class:User", Label: scopeir.NodeClass, Properties: graph.NodeProperties{
		"name": "User", "filePath": "src/user.ts", "startLine": 1, "endLine": 3,
	}})
	g.AddNode(graph.Node{ID: "Constructor:User", Label: scopeir.NodeConstructor, Properties: graph.NodeProperties{
		"name": "User", "filePath": "src/user.ts", "startLine": 2, "endLine": 2,
	}})
	g.AddNode(graph.Node{ID: "Function:makeUser", Label: scopeir.NodeFunction, Properties: graph.NodeProperties{
		"name": "makeUser", "filePath": "src/consumer.ts", "startLine": 4, "endLine": 6,
	}})
	g.AddRelationship(graph.Relationship{ID: "rel:file-defines-class", SourceID: "File:src/user.ts", TargetID: "Class:User", Type: graph.RelDefines})
	g.AddRelationship(graph.Relationship{ID: "rel:class-has-ctor", SourceID: "Class:User", TargetID: "Constructor:User", Type: graph.RelHasMethod})
	g.AddRelationship(graph.Relationship{ID: "rel:make-calls-ctor", SourceID: "Function:makeUser", TargetID: "Constructor:User", Type: graph.RelCalls})
	g.AddRelationship(graph.Relationship{ID: "rel:consumer-imports-user-file", SourceID: "File:src/consumer.ts", TargetID: "File:src/user.ts", Type: graph.RelImports})
	writeMCPGraph(t, repoPath, g)
}

func writeMCPRouteOnlyGraph(t *testing.T, repoPath string, route string) {
	t.Helper()
	g := graph.New()
	g.AddNode(graph.Node{ID: "Route:" + route, Label: scopeir.NodeRoute, Properties: graph.NodeProperties{
		"name": route, "filePath": "src/server.ts",
	}})
	writeMCPGraph(t, repoPath, g)
}

func writeMCPSameFileRenameGraph(t *testing.T, repoPath string) {
	t.Helper()
	g := graph.New()
	g.AddNode(graph.Node{ID: "Function:helper", Label: scopeir.NodeFunction, Properties: graph.NodeProperties{
		"name": "helper", "filePath": "src/app.ts", "startLine": 1, "endLine": 3,
	}})
	g.AddNode(graph.Node{ID: "Function:main", Label: scopeir.NodeFunction, Properties: graph.NodeProperties{
		"name": "main", "filePath": "src/app.ts", "startLine": 5, "endLine": 8,
	}})
	g.AddRelationship(graph.Relationship{
		ID:               "rel:CALLS:Function:main->Function:helper:7:9",
		SourceID:         "Function:main",
		TargetID:         "Function:helper",
		Type:             graph.RelCalls,
		Confidence:       0.95,
		Reason:           "scope-resolution: call",
		ResolutionSource: "scope-resolution",
	})
	writeMCPGraph(t, repoPath, g)
}

func writeMCPGroupSyncProviderGraph(t *testing.T, repoPath string) {
	t.Helper()
	g := graph.New()
	g.AddNode(graph.Node{ID: "Route:/api/users", Label: scopeir.NodeRoute, Properties: graph.NodeProperties{
		"name": "/api/users", "filePath": "src/server.ts",
	}})
	writeMCPGraph(t, repoPath, g)
}

func writeMCPGroupSyncConsumerGraph(t *testing.T, repoPath string) {
	t.Helper()
	g := graph.New()
	g.AddNode(graph.Node{ID: "File:src/app.ts", Label: scopeir.NodeFile, Properties: graph.NodeProperties{
		"name": "app.ts", "filePath": "src/app.ts",
	}})
	g.AddNode(graph.Node{ID: "Route:/api/users", Label: scopeir.NodeRoute, Properties: graph.NodeProperties{
		"name": "/api/users",
	}})
	g.AddRelationship(graph.Relationship{ID: "rel:fetch-users", SourceID: "File:src/app.ts", TargetID: "Route:/api/users", Type: graph.RelFetches, Confidence: 0.9, Reason: "fetch-route"})
	writeMCPGraph(t, repoPath, g)
}

func writeMCPGraph(t *testing.T, repoPath string, g *graph.Graph) {
	t.Helper()
	raw, err := json.MarshalIndent(g, "", "  ")
	if err != nil {
		t.Fatalf("marshal graph: %v", err)
	}
	graphPath := filepath.Join(repo.StoragePath(repoPath), "graph.json")
	if err := os.MkdirAll(filepath.Dir(graphPath), 0o755); err != nil {
		t.Fatalf("mkdir graph: %v", err)
	}
	if err := os.WriteFile(graphPath, append(raw, '\n'), 0o644); err != nil {
		t.Fatalf("write graph: %v", err)
	}
}

func runMCPTestGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, output)
	}
}

func readTestFrames(t *testing.T, raw []byte) []map[string]any {
	t.Helper()
	reader := bufio.NewReader(bytes.NewReader(raw))
	var responses []map[string]any
	for {
		frame, _, err := readMessage(reader)
		if err != nil {
			if err == io.EOF {
				return responses
			}
			t.Fatalf("read output frame: %v", err)
		}
		var response map[string]any
		if err := json.Unmarshal(frame, &response); err != nil {
			t.Fatalf("unmarshal output frame: %v; frame=%s", err, frame)
		}
		responses = append(responses, response)
	}
}

type stdoutLeakingWriter struct {
	target io.Writer
}

func (w stdoutLeakingWriter) Write(payload []byte) (int, error) {
	fmt.Fprint(os.Stdout, "leaked stdout")
	return w.target.Write(payload)
}

type fakeCypherReadRunner struct {
	rows   []lbugruntime.Row
	query  string
	closed bool
}

func (r *fakeCypherReadRunner) QueryRows(query string) ([]lbugruntime.Row, error) {
	r.query = query
	return r.rows, nil
}

func (r *fakeCypherReadRunner) Close() error {
	r.closed = true
	return nil
}
