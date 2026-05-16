package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/tamnguyendinh/avmatrix-go/internal/graph"
	"github.com/tamnguyendinh/avmatrix-go/internal/httpapi"
	"github.com/tamnguyendinh/avmatrix-go/internal/repo"
	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
	"github.com/tamnguyendinh/avmatrix-go/internal/version"
)

func executeForTest(t *testing.T, args ...string) (string, string, error) {
	t.Helper()

	var out bytes.Buffer
	var errOut bytes.Buffer
	cmd := NewRootCommand(Options{
		Out:    &out,
		Err:    &errOut,
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	cmd.SetArgs(args)

	err := cmd.ExecuteContext(context.Background())
	return out.String(), errOut.String(), err
}

func TestVersionCommandPrintsVersion(t *testing.T) {
	out, errOut, err := executeForTest(t, "version")
	if err != nil {
		t.Fatalf("version command returned error: %v", err)
	}
	if errOut != "" {
		t.Fatalf("version command wrote stderr: %q", errOut)
	}
	if out != version.Version+"\n" {
		t.Fatalf("unexpected version output: %q", out)
	}
}

func TestVersionFlagPrintsVersion(t *testing.T) {
	out, errOut, err := executeForTest(t, "--version")
	if err != nil {
		t.Fatalf("--version returned error: %v", err)
	}
	if errOut != "" {
		t.Fatalf("--version wrote stderr: %q", errOut)
	}
	if out != version.Version+"\n" {
		t.Fatalf("unexpected --version output: %q", out)
	}
}

func TestHelpCommandPrintsStubHelp(t *testing.T) {
	out, _, err := executeForTest(t, "help")
	if err != nil {
		t.Fatalf("help returned error: %v", err)
	}
	for _, want := range []string{
		"AVmatrix local CLI and MCP server",
		"analyze",
		"augment",
		"benchmark-compare",
		"clean",
		"context",
		"cypher",
		"detect-changes",
		"group",
		"impact",
		"index",
		"list",
		"mcp",
		"query",
		"serve",
		"setup",
		"status",
		"version",
		"wiki",
		"wiki-mode",
		"local HTTP bridge for the web UI",
		"shared session runtime",
		"detect-changes",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("help output missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "eval-server") {
		t.Fatalf("root help still exposes eval-server:\n%s", out)
	}
}

func TestSetupCommandWritesEditorConfigsAndSkills(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("AVMATRIX_HOME", filepath.Join(home, ".avmatrix-global"))
	t.Setenv("PATH", "")

	for _, dir := range []string{
		filepath.Join(home, ".cursor"),
		filepath.Join(home, ".claude"),
		filepath.Join(home, ".codex"),
		filepath.Join(home, ".config", "opencode"),
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	if err := os.WriteFile(filepath.Join(home, ".claude.json"), []byte(`{"existingKey":"keep-me","mcpServers":{"other":{"command":"foo"}}}`), 0o644); err != nil {
		t.Fatalf("seed .claude.json: %v", err)
	}
	if err := os.WriteFile(filepath.Join(home, ".claude", "settings.json"), []byte(`{"hooks":{"PreToolUse":[{"matcher":"Grep","hooks":[{"type":"command","command":"node old/avmatrix-hook.cjs"}]}]}}`), 0o644); err != nil {
		t.Fatalf("seed Claude settings: %v", err)
	}

	out, errOut, err := executeForTest(t, "setup")
	if err != nil {
		t.Fatalf("setup returned error: %v\nstdout:\n%s\nstderr:\n%s", err, out, errOut)
	}
	if errOut != "" {
		t.Fatalf("setup wrote stderr: %q", errOut)
	}
	for _, want := range []string{"AVmatrix Setup", "Cursor", "Claude Code", "OpenCode", "Codex", "Skills installed to"} {
		if !strings.Contains(out, want) {
			t.Fatalf("setup output missing %q:\n%s", want, out)
		}
	}
	if _, err := os.Stat(filepath.Join(home, ".avmatrix-global")); err != nil {
		t.Fatalf("global AVMATRIX_HOME was not created: %v", err)
	}

	cursorConfig := readTestJSONFile(t, filepath.Join(home, ".cursor", "mcp.json"))
	requireTestMCPEntry(t, cursorConfig["mcpServers"], "Cursor")

	claudeConfig := readTestJSONFile(t, filepath.Join(home, ".claude.json"))
	if claudeConfig["existingKey"] != "keep-me" {
		t.Fatalf("setup did not preserve .claude.json existingKey: %#v", claudeConfig)
	}
	requireTestMCPEntry(t, claudeConfig["mcpServers"], "Claude Code")

	opencodeConfig := readTestJSONFile(t, filepath.Join(home, ".config", "opencode", "opencode.json"))
	requireTestMCPEntry(t, opencodeConfig["mcp"], "OpenCode")

	codexConfig, err := os.ReadFile(filepath.Join(home, ".codex", "config.toml"))
	if err != nil {
		t.Fatalf("read Codex fallback config: %v", err)
	}
	if !strings.Contains(string(codexConfig), "[mcp_servers.avmatrix]") ||
		!strings.Contains(string(codexConfig), `command = "avmatrix"`) ||
		!strings.Contains(string(codexConfig), `"mcp"`) {
		t.Fatalf("Codex fallback config missing MCP entry:\n%s", codexConfig)
	}

	for _, path := range []string{
		filepath.Join(home, ".cursor", "skills", "avmatrix-cli", "SKILL.md"),
		filepath.Join(home, ".claude", "skills", "avmatrix-cli", "SKILL.md"),
		filepath.Join(home, ".config", "opencode", "skill", "avmatrix-cli", "SKILL.md"),
		filepath.Join(home, ".agents", "skills", "avmatrix-cli", "SKILL.md"),
	} {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read installed skill %s: %v", path, err)
		}
		if !strings.Contains(string(raw), "AVmatrix CLI Commands") {
			t.Fatalf("installed skill %s did not copy packaged content:\n%s", path, raw)
		}
	}

	claudeSettings := readTestJSONFile(t, filepath.Join(home, ".claude", "settings.json"))
	settingsJSON, err := json.Marshal(claudeSettings)
	if err != nil {
		t.Fatalf("marshal Claude settings: %v", err)
	}
	if strings.Contains(string(settingsJSON), "avmatrix-hook.cjs") {
		t.Fatalf("Claude settings still reference legacy Node hook: %s", settingsJSON)
	}
	if !strings.Contains(string(settingsJSON), "avmatrix hook claude") ||
		!strings.Contains(string(settingsJSON), "PreToolUse") ||
		!strings.Contains(string(settingsJSON), "PostToolUse") {
		t.Fatalf("Claude settings missing hook entries: %s", settingsJSON)
	}

	out, errOut, err = executeForTest(t, "setup")
	if err != nil {
		t.Fatalf("second setup returned error: %v\nstdout:\n%s\nstderr:\n%s", err, out, errOut)
	}
	codexConfig, err = os.ReadFile(filepath.Join(home, ".codex", "config.toml"))
	if err != nil {
		t.Fatalf("read Codex fallback config after second setup: %v", err)
	}
	if count := strings.Count(string(codexConfig), "[mcp_servers.avmatrix]"); count != 1 {
		t.Fatalf("Codex fallback section duplicated %d times:\n%s", count, codexConfig)
	}
}

func TestSetupClaudeCodeSkipsAndRecoversCorruptConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("AVMATRIX_HOME", filepath.Join(home, ".avmatrix-global"))

	result := setupResult{}
	setupClaudeCode(&result)
	if !containsSetupResult(result.Skipped, "Claude Code (not installed)") {
		t.Fatalf("Claude setup did not skip missing install: %#v", result)
	}
	if _, err := os.Stat(filepath.Join(home, ".claude.json")); !os.IsNotExist(err) {
		t.Fatalf("Claude setup wrote config while missing install: %v", err)
	}

	if err := os.MkdirAll(filepath.Join(home, ".claude"), 0o755); err != nil {
		t.Fatalf("mkdir .claude: %v", err)
	}
	if err := os.WriteFile(filepath.Join(home, ".claude.json"), []byte("{ invalid json"), 0o644); err != nil {
		t.Fatalf("write corrupt .claude.json: %v", err)
	}
	if err := os.WriteFile(filepath.Join(home, ".claude", "settings.json"), []byte(`{"hooks":{"PreToolUse":[{"matcher":"Grep","hooks":[{"type":"command","command":"node old/avmatrix-hook.cjs"}]}]}}`), 0o644); err != nil {
		t.Fatalf("write Claude settings: %v", err)
	}

	result = setupResult{}
	setupClaudeCode(&result)
	if len(result.Errors) != 0 {
		t.Fatalf("Claude setup returned errors: %#v", result.Errors)
	}
	config := readTestJSONFile(t, filepath.Join(home, ".claude.json"))
	requireTestMCPEntry(t, config["mcpServers"], "Claude Code")
	settingsRaw, err := os.ReadFile(filepath.Join(home, ".claude", "settings.json"))
	if err != nil {
		t.Fatalf("read Claude settings: %v", err)
	}
	settings := string(settingsRaw)
	if strings.Contains(settings, "avmatrix-hook.cjs") || !strings.Contains(settings, "avmatrix hook claude") {
		t.Fatalf("Claude hooks were not migrated to Go hook command:\n%s", settings)
	}
}

func TestSetupCodexUsesCLIWhenAvailableAndSkipsMissingInstall(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("AVMATRIX_HOME", filepath.Join(home, ".avmatrix-global"))

	result := setupResult{}
	setupCodex(&result)
	if !containsSetupResult(result.Skipped, "Codex (not installed)") {
		t.Fatalf("Codex setup did not skip missing install: %#v", result)
	}

	if err := os.MkdirAll(filepath.Join(home, ".codex"), 0o755); err != nil {
		t.Fatalf("mkdir .codex: %v", err)
	}
	argsPath := filepath.Join(home, "codex-args.txt")
	fakeBin := t.TempDir()
	writeFakeCodex(t, fakeBin, argsPath)
	t.Setenv("PATH", fakeBin+string(os.PathListSeparator)+os.Getenv("PATH"))

	result = setupResult{}
	setupCodex(&result)
	if len(result.Errors) != 0 {
		t.Fatalf("Codex setup returned errors: %#v", result.Errors)
	}
	if !containsSetupResult(result.Configured, "Codex") {
		t.Fatalf("Codex CLI path did not configure Codex: %#v", result)
	}
	argsRaw, err := os.ReadFile(argsPath)
	if err != nil {
		t.Fatalf("fake codex command was not invoked: %v", err)
	}
	for _, want := range []string{"mcp", "add", "avmatrix", "--", "avmatrix", "mcp"} {
		if !strings.Contains(string(argsRaw), want) {
			t.Fatalf("fake codex args missing %q: %s", want, argsRaw)
		}
	}
	if _, err := os.Stat(filepath.Join(home, ".codex", "config.toml")); !os.IsNotExist(err) {
		t.Fatalf("Codex CLI success should not write fallback config: %v", err)
	}
}

func TestSetupInstallsFlatAndDirectorySkillsFromPackageRoot(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	packageRoot := t.TempDir()
	skillsRoot := filepath.Join(packageRoot, "skills")
	if err := os.MkdirAll(filepath.Join(home, ".cursor"), 0o755); err != nil {
		t.Fatalf("mkdir .cursor: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(skillsRoot, "dir-skill", "references"), 0o755); err != nil {
		t.Fatalf("mkdir dir skill: %v", err)
	}
	if err := os.WriteFile(filepath.Join(skillsRoot, "flat-skill.md"), []byte("# Flat Test Skill\n"), 0o644); err != nil {
		t.Fatalf("write flat skill: %v", err)
	}
	if err := os.WriteFile(filepath.Join(skillsRoot, "dir-skill", "SKILL.md"), []byte("# Directory Test Skill\n"), 0o644); err != nil {
		t.Fatalf("write dir skill: %v", err)
	}
	if err := os.WriteFile(filepath.Join(skillsRoot, "dir-skill", "references", "note.md"), []byte("# Directory Nested File\n"), 0o644); err != nil {
		t.Fatalf("write nested skill file: %v", err)
	}

	withWorkingDir(t, packageRoot, func() {
		result := setupResult{}
		setupCursor(&result)
		if len(result.Errors) != 0 {
			t.Fatalf("Cursor setup returned errors: %#v", result.Errors)
		}
	})

	cursorSkillsRoot := filepath.Join(home, ".cursor", "skills")
	for _, rel := range []string{
		filepath.Join("flat-skill", "SKILL.md"),
		filepath.Join("dir-skill", "SKILL.md"),
		filepath.Join("dir-skill", "references", "note.md"),
	} {
		if _, err := os.Stat(filepath.Join(cursorSkillsRoot, rel)); err != nil {
			t.Fatalf("installed skill file missing %s: %v", rel, err)
		}
	}
}

func TestGroupHelpShowsCompatibilitySubcommands(t *testing.T) {
	out, _, err := executeForTest(t, "group", "--help")
	if err != nil {
		t.Fatalf("group help returned error: %v", err)
	}
	for _, want := range []string{"create", "add", "remove", "list", "status", "sync", "query", "contracts"} {
		if !strings.Contains(out, want) {
			t.Fatalf("group help missing %q:\n%s", want, out)
		}
	}
}

func readTestJSONFile(t *testing.T, path string) map[string]any {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read JSON %s: %v", path, err)
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("parse JSON %s: %v\n%s", path, err, raw)
	}
	return out
}

func requireTestMCPEntry(t *testing.T, value any, label string) {
	t.Helper()
	servers, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("%s MCP container has wrong type: %#v", label, value)
	}
	entry, ok := servers["avmatrix"].(map[string]any)
	if !ok {
		t.Fatalf("%s missing avmatrix MCP entry: %#v", label, servers)
	}
	if entry["command"] != "avmatrix" {
		t.Fatalf("%s MCP command = %#v", label, entry["command"])
	}
	args, ok := entry["args"].([]any)
	if !ok || len(args) != 1 || args[0] != "mcp" {
		t.Fatalf("%s MCP args = %#v", label, entry["args"])
	}
}

func containsSetupResult(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func writeFakeCodex(t *testing.T, dir string, argsPath string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		path := filepath.Join(dir, "codex.cmd")
		content := "@echo off\r\necho %* > " + strconv.Quote(argsPath) + "\r\nexit /b 0\r\n"
		if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
			t.Fatalf("write fake codex.cmd: %v", err)
		}
		return
	}
	path := filepath.Join(dir, "codex")
	content := "#!/bin/sh\nprintf '%s\\n' \"$*\" > " + strconv.Quote(argsPath) + "\n"
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatalf("write fake codex: %v", err)
	}
}

func requireTestOutputContainsPath(t *testing.T, output string, path string) {
	t.Helper()
	for _, candidate := range testPathCandidates(path) {
		if strings.Contains(output, candidate) {
			return
		}
	}
	t.Fatalf("output missing path %q or equivalent:\n%s", path, output)
}

func testPathsEqual(left string, right string) bool {
	leftCandidates := testPathCandidates(left)
	rightCandidates := testPathCandidates(right)
	for _, leftCandidate := range leftCandidates {
		for _, rightCandidate := range rightCandidates {
			if strings.EqualFold(filepath.Clean(leftCandidate), filepath.Clean(rightCandidate)) {
				return true
			}
		}
	}
	return false
}

func testPathCandidates(path string) []string {
	candidates := []string{filepath.Clean(path)}
	if absolute, err := filepath.Abs(path); err == nil {
		candidates = append(candidates, filepath.Clean(absolute))
	}
	if evaluated, err := filepath.EvalSymlinks(path); err == nil {
		candidates = append(candidates, filepath.Clean(evaluated))
	}
	unique := candidates[:0]
	seen := map[string]struct{}{}
	for _, candidate := range candidates {
		key := strings.ToLower(candidate)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		unique = append(unique, candidate)
	}
	return unique
}

func TestDirectToolHelpShowsCompatibilityFlags(t *testing.T) {
	cases := []struct {
		args []string
		want []string
	}{
		{[]string{"query", "--help"}, []string{"avmatrix query <search_query>", "--repo", "--context", "--goal", "--limit", "--content"}},
		{[]string{"context", "--help"}, []string{"avmatrix context [name]", "--repo", "--uid", "--file", "--content"}},
		{[]string{"impact", "--help"}, []string{"--direction", "--repo", "--uid", "--depth", "--include-tests"}},
		{[]string{"cypher", "--help"}, []string{"--repo"}},
		{[]string{"detect-changes", "--help"}, []string{"Analyze uncommitted git changes", "--scope", "--base-ref", "--repo"}},
	}

	for _, tc := range cases {
		out, _, err := executeForTest(t, tc.args...)
		if err != nil {
			t.Fatalf("%v returned error: %v", tc.args, err)
		}
		for _, want := range tc.want {
			if !strings.Contains(out, want) {
				t.Fatalf("%v help missing %q:\n%s", tc.args, want, out)
			}
		}
	}
}

func TestDirectToolCommandsUseLocalMCPRuntime(t *testing.T) {
	home := t.TempDir()
	t.Setenv(repo.HomeEnvName, home)
	repoPath := t.TempDir()
	registerDirectToolCommandRepo(t, repo.NewStore(home), repoPath, "fixture")
	writeDirectToolCommandGraph(t, repoPath)

	out, errOut, err := executeForTest(t, "query", "MainFlow", "--repo", "fixture", "--limit", "2", "--content")
	if err != nil {
		t.Fatalf("query command returned error: %v\nstdout:\n%s\nstderr:\n%s", err, out, errOut)
	}
	if errOut != "" {
		t.Fatalf("query command wrote stderr: %q", errOut)
	}
	for _, want := range []string{`"query": "MainFlow"`, `"Label": "MainFlow"`, `"Function:main"`} {
		if !strings.Contains(out, want) {
			t.Fatalf("query output missing %q:\n%s", want, out)
		}
	}

	out, errOut, err = executeForTest(t, "context", "main", "--repo", "fixture", "--content")
	if err != nil {
		t.Fatalf("context command returned error: %v\nstdout:\n%s\nstderr:\n%s", err, out, errOut)
	}
	if errOut != "" {
		t.Fatalf("context command wrote stderr: %q", errOut)
	}
	for _, want := range []string{`"status": "found"`, `"uid": "Function:main"`, `"Function:helper"`} {
		if !strings.Contains(out, want) {
			t.Fatalf("context output missing %q:\n%s", want, out)
		}
	}

	out, errOut, err = executeForTest(t, "impact", "--uid", "Function:helper", "--repo", "fixture", "--direction", "upstream", "--depth", "2", "--include-tests")
	if err != nil {
		t.Fatalf("impact command returned error: %v\nstdout:\n%s\nstderr:\n%s", err, out, errOut)
	}
	if errOut != "" {
		t.Fatalf("impact command wrote stderr: %q", errOut)
	}
	for _, want := range []string{`"impactedCount": 1`, `"id": "Function:main"`, `"direction": "upstream"`} {
		if !strings.Contains(out, want) {
			t.Fatalf("impact output missing %q:\n%s", want, out)
		}
	}

	out, errOut, err = executeForTest(t, "cypher", "MATCH (n:Function) RETURN n.id AS id, n.name AS name LIMIT 5", "--repo", "fixture")
	if err != nil {
		t.Fatalf("cypher command returned error: %v\nstdout:\n%s\nstderr:\n%s", err, out, errOut)
	}
	if errOut != "" {
		t.Fatalf("cypher command wrote stderr: %q", errOut)
	}
	for _, want := range []string{`"row_count": 2`, `"Function:main"`, `"Function:helper"`} {
		if !strings.Contains(out, want) {
			t.Fatalf("cypher output missing %q:\n%s", want, out)
		}
	}
}

func TestDirectDetectChangesCommandUsesLocalMCPRuntime(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git executable is not available")
	}

	home := t.TempDir()
	t.Setenv(repo.HomeEnvName, home)
	repoPath := t.TempDir()
	sourceDir := filepath.Join(repoPath, "src")
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		t.Fatalf("mkdir source: %v", err)
	}
	sourcePath := filepath.Join(sourceDir, "app.ts")
	if err := os.WriteFile(sourcePath, []byte("function main() {\n  return helper()\n}\n"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	runGit(t, repoPath, "init")
	runGit(t, repoPath, "config", "user.email", "test@example.com")
	runGit(t, repoPath, "config", "user.name", "Test User")
	runGit(t, repoPath, "add", "src/app.ts")
	runGit(t, repoPath, "commit", "-m", "initial")

	store := repo.NewStore(home)
	registerDirectToolCommandRepo(t, store, repoPath, "fixture")
	writeDirectToolCommandGraph(t, repoPath)
	if err := os.WriteFile(sourcePath, []byte("function main() {\n  return helper() + 1\n}\n"), 0o644); err != nil {
		t.Fatalf("modify source: %v", err)
	}

	out, errOut, err := executeForTest(t, "detect-changes", "--repo", "fixture", "--scope", "unstaged")
	if err != nil {
		t.Fatalf("detect-changes command returned error: %v\nstdout:\n%s\nstderr:\n%s", err, out, errOut)
	}
	if errOut != "" {
		t.Fatalf("detect-changes command wrote stderr: %q", errOut)
	}
	for _, want := range []string{`"changed_count": 1`, `"changed_files": 1`, `"id": "Function:main"`} {
		if !strings.Contains(out, want) {
			t.Fatalf("detect-changes output missing %q:\n%s", want, out)
		}
	}
}

func TestAugmentIgnoresShortPattern(t *testing.T) {
	out, errOut, err := executeForTest(t, "augment", "go")
	if err != nil {
		t.Fatalf("augment returned error: %v", err)
	}
	if out != "" || errOut != "" {
		t.Fatalf("augment short pattern wrote output: stdout=%q stderr=%q", out, errOut)
	}
}

func TestMCPHelpShowsStdioServer(t *testing.T) {
	out, _, err := executeForTest(t, "mcp", "--help")
	if err != nil {
		t.Fatalf("mcp help returned error: %v", err)
	}
	for _, want := range []string{"Start MCP server over stdio", "Usage:"} {
		if !strings.Contains(out, want) {
			t.Fatalf("mcp help missing %q:\n%s", want, out)
		}
	}
}

func TestServeHelpShowsDefaultHostAndPortFlags(t *testing.T) {
	out, _, err := executeForTest(t, "serve", "--help")
	if err != nil {
		t.Fatalf("serve help returned error: %v", err)
	}
	for _, want := range []string{
		"--host",
		"--port",
		"Start local HTTP bridge",
		"127.0.0.1",
		strconv.Itoa(httpapi.DefaultPort),
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("serve help missing %q:\n%s", want, out)
		}
	}
}

func TestWikiHelpExposesOnlyLocalCapabilityGate(t *testing.T) {
	out, _, err := executeForTest(t, "wiki", "--help")
	if err != nil {
		t.Fatalf("wiki help returned error: %v", err)
	}
	for _, want := range []string{"Show wiki capability status", "avmatrix wiki [path]"} {
		if !strings.Contains(out, want) {
			t.Fatalf("wiki help missing %q:\n%s", want, out)
		}
	}
	for _, forbidden := range []string{"--provider", "--api-key", "--model", "--gist"} {
		if strings.Contains(out, forbidden) {
			t.Fatalf("wiki help exposed remote option %q:\n%s", forbidden, out)
		}
	}
}

func TestAnalyzeHelpShowsEmbeddingsFlag(t *testing.T) {
	out, _, err := executeForTest(t, "analyze", "--help")
	if err != nil {
		t.Fatalf("analyze help returned error: %v", err)
	}
	for _, want := range []string{
		"--embeddings",
		"enable embedding generation",
		"--skills",
		"--skip-agents-md",
		"--no-stats",
		"--skip-git",
		"--skip-compatibility-cross-file",
		"--benchmark-label",
		"--name",
		"--allow-duplicate-name",
		"--verbose",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("analyze help missing %q:\n%s", want, out)
		}
	}
}

func TestUnknownCommandFails(t *testing.T) {
	_, _, err := executeForTest(t, "missing-command")
	if err == nil {
		t.Fatal("unknown command unexpectedly succeeded")
	}
}

func TestListReportsNoIndexedRepositories(t *testing.T) {
	t.Setenv(repo.HomeEnvName, t.TempDir())

	out, errOut, err := executeForTest(t, "list")
	if err != nil {
		t.Fatalf("list returned error: %v", err)
	}
	if errOut != "" {
		t.Fatalf("list wrote stderr: %q", errOut)
	}
	if out != "No indexed repositories.\n" {
		t.Fatalf("unexpected list output: %q", out)
	}
}

func TestListReportsRegisteredRepositories(t *testing.T) {
	home := t.TempDir()
	t.Setenv(repo.HomeEnvName, home)
	dir := initGitRepo(t)
	commit := repo.CurrentCommit(dir)
	meta := repo.Meta{
		RepoPath:   dir,
		LastCommit: commit,
		IndexedAt:  "2026-05-10T01:02:03Z",
	}
	if err := repo.SaveMeta(repo.Paths(dir).StoragePath, meta); err != nil {
		t.Fatalf("save meta: %v", err)
	}
	if _, err := repo.NewStore(home).Register(dir, meta, repo.RegisterOptions{Name: "demo-repo"}); err != nil {
		t.Fatalf("register repo: %v", err)
	}

	out, errOut, err := executeForTest(t, "list")
	if err != nil {
		t.Fatalf("list returned error: %v", err)
	}
	if errOut != "" {
		t.Fatalf("list wrote stderr: %q", errOut)
	}
	for _, want := range []string{
		"Indexed repositories:",
		"- demo-repo",
		"Path: " + dir,
		"Commit: " + commit[:7],
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("list output missing %q:\n%s", want, out)
		}
	}
}

func TestStatusReportsNotGitRepository(t *testing.T) {
	dir := t.TempDir()
	withWorkingDir(t, dir, func() {
		out, errOut, err := executeForTest(t, "status")
		if err != nil {
			t.Fatalf("status returned error: %v", err)
		}
		if errOut != "" {
			t.Fatalf("status wrote stderr: %q", errOut)
		}
		if out != "Not a git repository.\n" {
			t.Fatalf("unexpected status output: %q", out)
		}
	})
}

func TestStatusReportsIndexedRepositoryState(t *testing.T) {
	dir := initGitRepo(t)
	commit := repo.CurrentCommit(dir)
	if commit == "" {
		t.Fatal("test git repository has no current commit")
	}
	if err := repo.SaveMeta(repo.Paths(dir).StoragePath, repo.Meta{
		RepoPath:   dir,
		LastCommit: commit,
		IndexedAt:  "2026-05-10T01:02:03Z",
	}); err != nil {
		t.Fatalf("save meta: %v", err)
	}

	withWorkingDir(t, dir, func() {
		out, errOut, err := executeForTest(t, "status")
		if err != nil {
			t.Fatalf("status returned error: %v", err)
		}
		if errOut != "" {
			t.Fatalf("status wrote stderr: %q", errOut)
		}
		for _, want := range []string{
			"Repository: " + dir,
			"Indexed commit: " + commit[:7],
			"Current commit: " + commit[:7],
			"Status: ✅ up-to-date",
		} {
			if !strings.Contains(out, want) {
				t.Fatalf("status output missing %q:\n%s", want, out)
			}
		}
	})
}

func TestStatusReportsStaleKuzuIndex(t *testing.T) {
	dir := initGitRepo(t)
	if err := os.MkdirAll(filepath.Join(repo.Paths(dir).StoragePath, "kuzu"), 0o755); err != nil {
		t.Fatalf("create legacy kuzu path: %v", err)
	}

	withWorkingDir(t, dir, func() {
		out, errOut, err := executeForTest(t, "status")
		if err != nil {
			t.Fatalf("status returned error: %v", err)
		}
		if errOut != "" {
			t.Fatalf("status wrote stderr: %q", errOut)
		}
		for _, want := range []string{
			"Repository has a stale KuzuDB index from a previous version.",
			"Run: avmatrix analyze   (rebuilds the index with LadybugDB)",
		} {
			if !strings.Contains(out, want) {
				t.Fatalf("status output missing %q:\n%s", want, out)
			}
		}
	})
}

func TestWikiCommandReportsDisabledAndFailsSilently(t *testing.T) {
	t.Setenv(repo.HomeEnvName, t.TempDir())

	out, errOut, err := executeForTest(t, "wiki")
	if err == nil {
		t.Fatal("wiki unexpectedly succeeded")
	}
	if errOut != "" {
		t.Fatalf("wiki wrote stderr: %q", errOut)
	}
	if !strings.Contains(out, "Wiki capability mode: off") ||
		!strings.Contains(out, "Wiki generation is disabled in local-only mode.") {
		t.Fatalf("unexpected wiki output:\n%s", out)
	}
}

func TestWikiModeWritesRuntimeConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv(repo.HomeEnvName, home)

	out, errOut, err := executeForTest(t, "wiki-mode", "local")
	if err != nil {
		t.Fatalf("wiki-mode local returned error: %v", err)
	}
	if errOut != "" {
		t.Fatalf("wiki-mode local wrote stderr: %q", errOut)
	}
	if !strings.Contains(out, "Wiki capability mode: local") {
		t.Fatalf("wiki-mode local output missing local mode:\n%s", out)
	}
	if !strings.Contains(out, "will not fall back to any remote wiki service") {
		t.Fatalf("wiki-mode local output missing fail-safe remote fallback text:\n%s", out)
	}
	raw, err := os.ReadFile(filepath.Join(home, "runtime.json"))
	if err != nil {
		t.Fatalf("runtime config missing: %v", err)
	}
	if !strings.Contains(string(raw), `"wikiMode": "local"`) {
		t.Fatalf("runtime config did not persist local mode:\n%s", raw)
	}

	out, errOut, err = executeForTest(t, "wiki-mode")
	if err != nil {
		t.Fatalf("wiki-mode returned error: %v", err)
	}
	if errOut != "" {
		t.Fatalf("wiki-mode wrote stderr: %q", errOut)
	}
	if !strings.Contains(out, "Wiki capability mode: local") {
		t.Fatalf("wiki-mode output missing persisted local mode:\n%s", out)
	}
}

func TestWikiModeRejectsInvalidMode(t *testing.T) {
	t.Setenv(repo.HomeEnvName, t.TempDir())

	out, errOut, err := executeForTest(t, "wiki-mode", "remote")
	if err == nil {
		t.Fatal("wiki-mode invalid unexpectedly succeeded")
	}
	if out != "" {
		t.Fatalf("wiki-mode invalid wrote stdout: %q", out)
	}
	if !strings.Contains(errOut, "Invalid wiki mode. Use `off` or `local`.") {
		t.Fatalf("wiki-mode invalid stderr missing guidance:\n%s", errOut)
	}
}

func TestAnalyzeCommandRunsGoPipelineAndWritesBenchmark(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(repo.HomeEnvName, t.TempDir())
	writeCLITestFile(t, dir, "src/main.ts", `
export function helper() {
  return 1;
}
export function main() {
  return helper();
}
`)
	benchmarkPath := filepath.Join(dir, ".tmp", "benchmark.json")

	out, errOut, err := executeForTest(t, "analyze", dir, "--force", "--skip-git", "--benchmark-json", benchmarkPath, "--benchmark-label", "fixture")
	if err != nil {
		t.Fatalf("analyze returned error: %v", err)
	}
	if errOut != "" {
		t.Fatalf("analyze wrote stderr: %q", errOut)
	}
	for _, want := range []string{"analyzed ", "files: scanned=1 parsed=1", "graph: nodes="} {
		if !strings.Contains(out, want) {
			t.Fatalf("analyze output missing %q:\n%s", want, out)
		}
	}
	if _, err := os.Stat(benchmarkPath); err != nil {
		t.Fatalf("benchmark JSON missing: %v", err)
	}
	benchmarkRaw, err := os.ReadFile(benchmarkPath)
	if err != nil {
		t.Fatalf("read benchmark JSON: %v", err)
	}
	if !strings.Contains(string(benchmarkRaw), `"label": "fixture"`) {
		t.Fatalf("benchmark JSON missing label:\n%s", benchmarkRaw)
	}
	if _, err := os.Stat(filepath.Join(dir, ".avmatrix", "graph.json")); err != nil {
		t.Fatalf("graph snapshot missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".avmatrix", "meta.json")); err != nil {
		t.Fatalf("meta file missing: %v", err)
	}
}

func TestAnalyzeNameCollisionRequiresExplicitDuplicateBypass(t *testing.T) {
	home := t.TempDir()
	t.Setenv(repo.HomeEnvName, home)
	repoA := writeAnalyzeCollisionFixture(t)
	repoB := writeAnalyzeCollisionFixture(t)
	repoC := writeAnalyzeCollisionFixture(t)

	out, errOut, err := executeForTest(t, "analyze", repoA, "--force", "--skip-git", "--skip-agents-md", "--no-stats", "--name", "shared")
	if err != nil {
		t.Fatalf("first analyze returned error: %v\nstdout:\n%s\nstderr:\n%s", err, out, errOut)
	}
	if errOut != "" {
		t.Fatalf("first analyze wrote stderr: %q", errOut)
	}
	assertRegisteredRepoNames(t, home, []string{"shared"})

	out, errOut, err = executeForTest(t, "analyze", repoB, "--force", "--skip-git", "--skip-agents-md", "--no-stats", "--name", "shared")
	if err == nil {
		t.Fatal("second analyze with colliding --name unexpectedly succeeded")
	}
	if !strings.Contains(err.Error(), `registry name "shared" is already used`) {
		t.Fatalf("collision error missing registry name: %v\nstdout:\n%s\nstderr:\n%s", err, out, errOut)
	}

	out, errOut, err = executeForTest(t, "analyze", repoB, "--force", "--skip-git", "--skip-agents-md", "--no-stats", "--name", "shared", "--allow-duplicate-name")
	if err != nil {
		t.Fatalf("allow-duplicate analyze returned error: %v\nstdout:\n%s\nstderr:\n%s", err, out, errOut)
	}
	assertRegisteredRepoNames(t, home, []string{"shared", "shared"})

	out, errOut, err = executeForTest(t, "analyze", repoC, "--force", "--skip-git", "--skip-agents-md", "--no-stats", "--name", "shared", "--skills")
	if err == nil {
		t.Fatal("analyze --skills unexpectedly bypassed the name collision guard")
	}
	if !strings.Contains(err.Error(), `registry name "shared" is already used`) {
		t.Fatalf("--skills collision error missing registry name: %v\nstdout:\n%s\nstderr:\n%s", err, out, errOut)
	}

	entries, err := repo.NewStore(home).ReadRegistry()
	if err != nil {
		t.Fatalf("ReadRegistry() error = %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("registry length = %d, want 2 entries after bypass only", len(entries))
	}
	for _, entry := range entries {
		if entry.Name != "shared" {
			t.Fatalf("registry entry name = %q, want shared", entry.Name)
		}
	}
	if repo.SamePath(entries[0].Path, entries[1].Path) {
		t.Fatalf("duplicate-name entries point to the same path: %#v", entries)
	}
}

func assertRegisteredRepoNames(t *testing.T, home string, want []string) {
	t.Helper()
	entries, err := repo.NewStore(home).ReadRegistry()
	if err != nil {
		t.Fatalf("ReadRegistry() error = %v", err)
	}
	if len(entries) != len(want) {
		t.Fatalf("registry length = %d, want %d: %#v", len(entries), len(want), entries)
	}
	for index, name := range want {
		if entries[index].Name != name {
			t.Fatalf("registry[%d].Name = %q, want %q", index, entries[index].Name, name)
		}
	}
}

func TestAnalyzeCommandWritesPprofProfiles(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(repo.HomeEnvName, t.TempDir())
	writeCLITestFile(t, dir, "src/main.ts", `
export function profiled() {
  return 1;
}
`)
	cpuProfilePath := filepath.Join(dir, "cpu.pprof")
	memProfilePath := filepath.Join(dir, "mem.pprof")

	out, errOut, err := executeForTest(t, "analyze", dir, "--force", "--skip-git", "--cpuprofile", cpuProfilePath, "--memprofile", memProfilePath)
	if err != nil {
		t.Fatalf("analyze with pprof returned error: %v", err)
	}
	if errOut != "" {
		t.Fatalf("analyze with pprof wrote stderr: %q", errOut)
	}
	if !strings.Contains(out, "analyzed ") {
		t.Fatalf("analyze output missing success:\n%s", out)
	}
	for _, path := range []string{cpuProfilePath, memProfilePath} {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("profile missing %s: %v", path, err)
		}
		if info.Size() == 0 {
			t.Fatalf("profile is empty: %s", path)
		}
	}
}

func TestAnalyzeCommandGeneratesAIContextAndSkills(t *testing.T) {
	dir := initGitRepo(t)
	home := t.TempDir()
	t.Setenv(repo.HomeEnvName, home)
	writeCLITestFile(t, dir, "src/main.ts", `
export function alpha() {
  return beta();
}
export function beta() {
  return gamma();
}
export function gamma() {
  return 1;
}
`)

	out, errOut, err := executeForTest(t, "analyze", dir, "--force", "--skills")
	if err != nil {
		t.Fatalf("analyze --skills returned error: %v", err)
	}
	if errOut != "" {
		t.Fatalf("analyze --skills wrote stderr: %q", errOut)
	}
	for _, want := range []string{"analyzed ", "skills: generated=", "base=6"} {
		if !strings.Contains(out, want) {
			t.Fatalf("analyze --skills output missing %q:\n%s", want, out)
		}
	}
	for _, rel := range []string{
		"AGENTS.md",
		"CLAUDE.md",
		filepath.Join(".claude", "skills", "avmatrix", "avmatrix-cli", "SKILL.md"),
		filepath.Join(".avmatrix", "meta.json"),
	} {
		if _, err := os.Stat(filepath.Join(dir, rel)); err != nil {
			t.Fatalf("%s missing: %v", rel, err)
		}
	}
	agents, err := os.ReadFile(filepath.Join(dir, "AGENTS.md"))
	if err != nil {
		t.Fatalf("read AGENTS.md: %v", err)
	}
	if !strings.Contains(string(agents), "<!-- avmatrix:start -->") ||
		!strings.Contains(string(agents), ".claude/skills/avmatrix/avmatrix-cli/SKILL.md") {
		t.Fatalf("AGENTS.md missing AVmatrix context:\n%s", agents)
	}
	generatedDir := filepath.Join(dir, ".claude", "skills", "generated")
	generatedEntries, err := os.ReadDir(generatedDir)
	if err != nil {
		t.Fatalf("generated skills dir missing: %v", err)
	}
	if len(generatedEntries) == 0 {
		t.Fatalf("expected generated skills in %s", generatedDir)
	}
	if _, err := os.Stat(filepath.Join(home, "registry.json")); err != nil {
		t.Fatalf("registry missing: %v", err)
	}
}

func TestAnalyzeCommandRequiresGitUnlessSkipped(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(repo.HomeEnvName, t.TempDir())
	writeCLITestFile(t, dir, "src/main.ts", "export function main() { return 1 }\n")

	_, _, err := executeForTest(t, "analyze", dir, "--force")
	if err == nil || !strings.Contains(err.Error(), "--skip-git") {
		t.Fatalf("analyze without git error = %v, want --skip-git guidance", err)
	}
}

func TestCleanRequiresForceAndPreservesSettings(t *testing.T) {
	home := t.TempDir()
	t.Setenv(repo.HomeEnvName, home)
	dir := initGitRepo(t)
	paths := repo.Paths(dir)
	if err := os.MkdirAll(paths.StoragePath, 0o755); err != nil {
		t.Fatalf("mkdir storage: %v", err)
	}
	if err := os.WriteFile(filepath.Join(paths.StoragePath, "settings.json"), []byte(`{"maxExecutionFlows":7}`), 0o644); err != nil {
		t.Fatalf("write settings: %v", err)
	}
	meta := repo.Meta{RepoPath: dir, LastCommit: repo.CurrentCommit(dir), IndexedAt: "2026-05-13T00:00:00Z"}
	if err := repo.SaveMeta(paths.StoragePath, meta); err != nil {
		t.Fatalf("save meta: %v", err)
	}
	if _, err := repo.NewStore(home).Register(dir, meta, repo.RegisterOptions{Name: "clean-demo"}); err != nil {
		t.Fatalf("register: %v", err)
	}

	withWorkingDir(t, dir, func() {
		out, _, err := executeForTest(t, "clean")
		if err != nil {
			t.Fatalf("clean prompt returned error: %v", err)
		}
		if !strings.Contains(out, "Run with --force") {
			t.Fatalf("clean prompt missing force guidance:\n%s", out)
		}
		if _, err := os.Stat(paths.MetaPath); err != nil {
			t.Fatalf("clean without force removed meta: %v", err)
		}

		out, _, err = executeForTest(t, "clean", "--force")
		if err != nil {
			t.Fatalf("clean --force returned error: %v", err)
		}
		if !strings.Contains(out, "Deleted:") {
			t.Fatalf("clean --force output missing Deleted:\n%s", out)
		}
		if _, err := os.Stat(paths.MetaPath); !os.IsNotExist(err) {
			t.Fatalf("meta still exists after clean: %v", err)
		}
		settingsRaw, err := os.ReadFile(filepath.Join(paths.StoragePath, "settings.json"))
		if err != nil {
			t.Fatalf("settings not preserved: %v", err)
		}
		if !strings.Contains(string(settingsRaw), "maxExecutionFlows") {
			t.Fatalf("unexpected settings after clean: %s", settingsRaw)
		}
		entries, err := repo.NewStore(home).ReadRegistry()
		if err != nil {
			t.Fatalf("read registry: %v", err)
		}
		if len(entries) != 0 {
			t.Fatalf("registry still contains entries: %#v", entries)
		}
	})
}

func TestIndexRegistersExistingIndex(t *testing.T) {
	home := t.TempDir()
	t.Setenv(repo.HomeEnvName, home)
	dir := initGitRepo(t)
	paths := repo.Paths(dir)
	if err := os.MkdirAll(paths.LbugPath, 0o755); err != nil {
		t.Fatalf("mkdir lbug: %v", err)
	}
	nodes, edges := 3, 4
	meta := repo.Meta{
		RepoPath:   dir,
		LastCommit: repo.CurrentCommit(dir),
		IndexedAt:  "2026-05-13T00:00:00Z",
		Stats:      &repo.Stats{Nodes: &nodes, Edges: &edges},
	}
	if err := repo.SaveMeta(paths.StoragePath, meta); err != nil {
		t.Fatalf("save meta: %v", err)
	}

	out, errOut, err := executeForTest(t, "index", dir)
	if err != nil {
		t.Fatalf("index returned error: %v", err)
	}
	if errOut != "" {
		t.Fatalf("index wrote stderr: %q", errOut)
	}
	for _, want := range []string{"Repository registered:", "3 nodes | 4 edges"} {
		if !strings.Contains(out, want) {
			t.Fatalf("index output missing %q:\n%s", want, out)
		}
	}
	requireTestOutputContainsPath(t, out, dir)
	entries, err := repo.NewStore(home).ReadRegistry()
	if err != nil {
		t.Fatalf("read registry: %v", err)
	}
	if len(entries) != 1 || !testPathsEqual(entries[0].Path, dir) {
		t.Fatalf("registry entries = %#v", entries)
	}
	gitignore, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	if err != nil {
		t.Fatalf("gitignore missing: %v", err)
	}
	if !strings.Contains(string(gitignore), ".avmatrix/") {
		t.Fatalf("gitignore missing .avmatrix entry:\n%s", gitignore)
	}
}

func TestIndexRejectsMissingIndexInputs(t *testing.T) {
	home := t.TempDir()
	t.Setenv(repo.HomeEnvName, home)

	nonGitDir := t.TempDir()
	out, _, err := executeForTest(t, "index", nonGitDir)
	if err == nil {
		t.Fatal("index non-git path unexpectedly succeeded")
	}
	if !strings.Contains(err.Error(), "use --allow-non-git") {
		t.Fatalf("index non-git error missing allow-non-git guidance: %v", err)
	}
	if out != "" {
		t.Fatalf("index non-git wrote stdout: %q", out)
	}

	dir := initGitRepo(t)
	out, errOut, err := executeForTest(t, "index", dir)
	if err == nil {
		t.Fatal("index missing .avmatrix unexpectedly succeeded")
	}
	if errOut != "" {
		t.Fatalf("index missing .avmatrix wrote stderr: %q", errOut)
	}
	if !strings.Contains(out, "No .avmatrix/ folder found at:") ||
		!strings.Contains(out, "Run `avmatrix analyze` to build the index first.") {
		t.Fatalf("index missing .avmatrix output unexpected:\n%s", out)
	}

	paths := repo.Paths(dir)
	if err := os.MkdirAll(paths.StoragePath, 0o755); err != nil {
		t.Fatalf("mkdir storage: %v", err)
	}
	out, errOut, err = executeForTest(t, "index", dir)
	if err == nil {
		t.Fatal("index missing lbug unexpectedly succeeded")
	}
	if errOut != "" {
		t.Fatalf("index missing lbug wrote stderr: %q", errOut)
	}
	if !strings.Contains(out, "contains no LadybugDB index") ||
		!strings.Contains(out, "Run `avmatrix analyze` to build the index.") {
		t.Fatalf("index missing lbug output unexpected:\n%s", out)
	}
}

func TestIndexRegistersMissingMetaWithForceAndRejectsAmbiguousArgs(t *testing.T) {
	home := t.TempDir()
	t.Setenv(repo.HomeEnvName, home)
	dir := initGitRepo(t)
	paths := repo.Paths(dir)
	if err := os.MkdirAll(paths.LbugPath, 0o755); err != nil {
		t.Fatalf("mkdir lbug: %v", err)
	}

	out, errOut, err := executeForTest(t, "index", dir)
	if err == nil {
		t.Fatal("index missing meta without --force unexpectedly succeeded")
	}
	if errOut != "" {
		t.Fatalf("index missing meta wrote stderr: %q", errOut)
	}
	if !strings.Contains(out, ".avmatrix/ exists but meta.json is missing.") ||
		!strings.Contains(out, "Use --force to register anyway") {
		t.Fatalf("index missing meta output unexpected:\n%s", out)
	}

	out, errOut, err = executeForTest(t, "index", dir, "--force")
	if err != nil {
		t.Fatalf("index --force missing meta returned error: %v\nstdout:\n%s\nstderr:\n%s", err, out, errOut)
	}
	if errOut != "" {
		t.Fatalf("index --force missing meta wrote stderr: %q", errOut)
	}
	if !strings.Contains(out, "Repository registered:") {
		t.Fatalf("index --force output missing registration:\n%s", out)
	}
	if _, err := os.Stat(paths.MetaPath); err != nil {
		t.Fatalf("index --force did not write meta.json: %v", err)
	}

	out, _, err = executeForTest(t, "index", dir, "other")
	if err == nil {
		t.Fatal("index with ambiguous path args unexpectedly succeeded")
	}
	if !strings.Contains(err.Error(), "single path only") {
		t.Fatalf("index ambiguous args error = %v", err)
	}
	if out != "" {
		t.Fatalf("index ambiguous args wrote stdout: %q", out)
	}
}

func TestBenchmarkCompareReportsDeltas(t *testing.T) {
	dir := t.TempDir()
	before := filepath.Join(dir, "before.json")
	after := filepath.Join(dir, "after.json")
	if err := os.WriteFile(before, []byte(`{
  "label": "before",
  "totalDuration": 1000000000,
  "phases": [{"name":"scan","duration":100000000}],
  "files": {"scanned": 2},
  "dbLoad": {"nodeRows": 3}
}`), 0o644); err != nil {
		t.Fatalf("write before: %v", err)
	}
	if err := os.WriteFile(after, []byte(`{
  "label": "after",
  "totalDuration": 1500000000,
  "phases": [{"name":"scan","duration":250000000}],
  "files": {"scanned": 5},
  "dbLoad": {"nodeRows": 8}
}`), 0o644); err != nil {
		t.Fatalf("write after: %v", err)
	}

	out, _, err := executeForTest(t, "benchmark-compare", before, after)
	if err != nil {
		t.Fatalf("benchmark-compare returned error: %v", err)
	}
	for _, want := range []string{
		"AVmatrix benchmark comparison",
		"labels: before -> after",
		"wall: 1000 -> 1500 (+500, +50%)",
		"scan: 100 -> 250 (+150, +150%)",
		"files.scanned: 2 -> 5 (+3, +150%)",
		"dbLoad.nodeRows: 3 -> 8 (+5, +166.7%)",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("benchmark-compare output missing %q:\n%s", want, out)
		}
	}
}

func TestGroupCommandsManageConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv(repo.HomeEnvName, home)

	if _, _, err := executeForTest(t, "group", "create", "../../evil"); err == nil || !strings.Contains(err.Error(), "Invalid group name") {
		t.Fatalf("group create invalid name error = %v", err)
	}

	out, _, err := executeForTest(t, "group", "create", "demo")
	if err != nil {
		t.Fatalf("group create returned error: %v", err)
	}
	if !strings.Contains(out, `Created group "demo"`) {
		t.Fatalf("group create output unexpected:\n%s", out)
	}

	out, _, err = executeForTest(t, "group", "add", "demo", "app/backend", "backend")
	if err != nil {
		t.Fatalf("group add returned error: %v", err)
	}
	if !strings.Contains(out, `Added backend as "app/backend"`) {
		t.Fatalf("group add output unexpected:\n%s", out)
	}

	out, _, err = executeForTest(t, "group", "list", "demo")
	if err != nil {
		t.Fatalf("group list demo returned error: %v", err)
	}
	if !strings.Contains(out, "app/backend -> backend") {
		t.Fatalf("group list output missing repo:\n%s", out)
	}

	out, _, err = executeForTest(t, "group", "remove", "demo", "app/backend")
	if err != nil {
		t.Fatalf("group remove returned error: %v", err)
	}
	if !strings.Contains(out, `Removed "app/backend"`) {
		t.Fatalf("group remove output unexpected:\n%s", out)
	}
}

func TestGroupCommandsSyncQueryAndContracts(t *testing.T) {
	home := t.TempDir()
	t.Setenv(repo.HomeEnvName, home)
	store := repo.NewStore(home)
	backend := t.TempDir()
	frontend := t.TempDir()
	registerGroupCommandRepo(t, store, backend, "backend")
	registerGroupCommandRepo(t, store, frontend, "frontend")
	writeGroupCommandProviderGraph(t, backend)
	writeGroupCommandConsumerGraph(t, frontend)

	if _, _, err := executeForTest(t, "group", "create", "fixture"); err != nil {
		t.Fatalf("group create fixture returned error: %v", err)
	}
	if _, _, err := executeForTest(t, "group", "add", "fixture", "app/backend", "backend"); err != nil {
		t.Fatalf("group add backend returned error: %v", err)
	}
	if _, _, err := executeForTest(t, "group", "add", "fixture", "app/frontend", "frontend"); err != nil {
		t.Fatalf("group add frontend returned error: %v", err)
	}

	out, _, err := executeForTest(t, "group", "sync", "fixture", "--json")
	if err != nil {
		t.Fatalf("group sync returned error: %v", err)
	}
	if !strings.Contains(out, `"crossLinks"`) || !strings.Contains(out, `http::*::/api/users`) {
		t.Fatalf("group sync JSON missing cross-link:\n%s", out)
	}

	out, _, err = executeForTest(t, "group", "contracts", "fixture", "--json")
	if err != nil {
		t.Fatalf("group contracts returned error: %v", err)
	}
	if !strings.Contains(out, `http::*::/api/users`) {
		t.Fatalf("group contracts JSON missing contract:\n%s", out)
	}

	out, _, err = executeForTest(t, "group", "query", "fixture", "UserFlow", "--limit", "2")
	if err != nil {
		t.Fatalf("group query returned error: %v", err)
	}
	if !strings.Contains(out, "[app/backend] UserFlow") {
		t.Fatalf("group query output missing process:\n%s", out)
	}
}

func writeCLITestFile(t *testing.T, root string, rel string, contents string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", rel, err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("write %s: %v", rel, err)
	}
}

func writeAnalyzeCollisionFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	writeCLITestFile(t, dir, "src/main.ts", `export function main() { return 1 }`)
	return dir
}

func registerDirectToolCommandRepo(t *testing.T, store repo.Store, repoPath string, name string) {
	t.Helper()
	meta := repo.Meta{
		RepoPath:   repoPath,
		LastCommit: repo.CurrentCommit(repoPath),
		IndexedAt:  "2026-05-15T00:00:00Z",
		Stats:      &repo.Stats{},
	}
	if meta.LastCommit == "" {
		meta.LastCommit = "abc123"
	}
	if err := repo.SaveMeta(repo.StoragePath(repoPath), meta); err != nil {
		t.Fatalf("save meta: %v", err)
	}
	if _, err := store.Register(repoPath, meta, repo.RegisterOptions{Name: name}); err != nil {
		t.Fatalf("register repo: %v", err)
	}
}

func writeDirectToolCommandGraph(t *testing.T, repoPath string) {
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
		"name": "Api", "label": "Api", "heuristicLabel": "Api", "cohesion": 0.8, "symbolCount": 2,
	}})
	mainStep := 1
	helperStep := 2
	g.AddRelationship(graph.Relationship{ID: "rel:main-process", SourceID: "Function:main", TargetID: "Process:main", Type: graph.RelStepInProcess, Step: &mainStep})
	g.AddRelationship(graph.Relationship{ID: "rel:helper-process", SourceID: "Function:helper", TargetID: "Process:main", Type: graph.RelStepInProcess, Step: &helperStep})
	g.AddRelationship(graph.Relationship{ID: "rel:main-helper", SourceID: "Function:main", TargetID: "Function:helper", Type: graph.RelCalls, Confidence: 0.95, Reason: "fixture"})
	g.AddRelationship(graph.Relationship{ID: "rel:main-member", SourceID: "Function:main", TargetID: "comm_api", Type: graph.RelMemberOf})
	g.AddRelationship(graph.Relationship{ID: "rel:helper-member", SourceID: "Function:helper", TargetID: "comm_api", Type: graph.RelMemberOf})
	writeGroupCommandGraph(t, repoPath, g)
}

func registerGroupCommandRepo(t *testing.T, store repo.Store, repoPath string, name string) {
	t.Helper()
	meta := repo.Meta{
		RepoPath:   repoPath,
		LastCommit: "abc123",
		IndexedAt:  "2026-05-13T00:00:00Z",
		Stats:      &repo.Stats{},
	}
	if err := repo.SaveMeta(repo.StoragePath(repoPath), meta); err != nil {
		t.Fatalf("save meta: %v", err)
	}
	if _, err := store.Register(repoPath, meta, repo.RegisterOptions{Name: name}); err != nil {
		t.Fatalf("register repo: %v", err)
	}
}

func writeGroupCommandProviderGraph(t *testing.T, repoPath string) {
	t.Helper()
	g := graph.New()
	g.AddNode(graph.Node{ID: "Route:/api/users", Label: scopeir.NodeRoute, Properties: graph.NodeProperties{
		"name": "/api/users", "filePath": "src/server.ts",
	}})
	g.AddNode(graph.Node{ID: "Process:user", Label: scopeir.NodeProcess, Properties: graph.NodeProperties{
		"heuristicLabel": "UserFlow", "processType": "route", "stepCount": 1,
	}})
	writeGroupCommandGraph(t, repoPath, g)
}

func writeGroupCommandConsumerGraph(t *testing.T, repoPath string) {
	t.Helper()
	g := graph.New()
	g.AddNode(graph.Node{ID: "File:src/app.ts", Label: scopeir.NodeFile, Properties: graph.NodeProperties{
		"name": "app.ts", "filePath": "src/app.ts",
	}})
	g.AddNode(graph.Node{ID: "Route:/api/users", Label: scopeir.NodeRoute, Properties: graph.NodeProperties{
		"name": "/api/users",
	}})
	g.AddRelationship(graph.Relationship{ID: "rel:fetch-users", SourceID: "File:src/app.ts", TargetID: "Route:/api/users", Type: graph.RelFetches, Confidence: 0.9, Reason: "fetch-route"})
	writeGroupCommandGraph(t, repoPath, g)
}

func writeGroupCommandGraph(t *testing.T, repoPath string, g *graph.Graph) {
	t.Helper()
	raw, err := json.MarshalIndent(g, "", "  ")
	if err != nil {
		t.Fatalf("marshal graph: %v", err)
	}
	graphPath := filepath.Join(repo.StoragePath(repoPath), "graph.json")
	if err := os.MkdirAll(filepath.Dir(graphPath), 0o755); err != nil {
		t.Fatalf("mkdir graph dir: %v", err)
	}
	if err := os.WriteFile(graphPath, append(raw, '\n'), 0o644); err != nil {
		t.Fatalf("write graph: %v", err)
	}
}

func withWorkingDir(t *testing.T, dir string, run func()) {
	t.Helper()
	previous, err := os.Getwd()
	if err != nil {
		t.Fatalf("get cwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir %s: %v", dir, err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(previous); err != nil {
			t.Fatalf("restore cwd: %v", err)
		}
	})
	run()
}

func initGitRepo(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git executable is not available")
	}
	dir := t.TempDir()
	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.email", "test@example.com")
	runGit(t, dir, "config", "user.name", "Test User")
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# test\n"), 0o644); err != nil {
		t.Fatalf("write README: %v", err)
	}
	runGit(t, dir, "add", "README.md")
	runGit(t, dir, "commit", "-m", "initial")
	return dir
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, output)
	}
}
