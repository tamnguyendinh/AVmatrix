package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestClaudeHookPreToolUseAugmentsSearchWithGoCLI(t *testing.T) {
	repoDir := t.TempDir()
	if err := os.Mkdir(filepath.Join(repoDir, ".avmatrix"), 0o755); err != nil {
		t.Fatalf("mkdir .avmatrix: %v", err)
	}
	input := `{"hook_event_name":"PreToolUse","tool_name":"Grep","tool_input":{"pattern":"AuthService"},"cwd":` + strconvQuote(repoDir) + `}`
	var output bytes.Buffer
	var capturedArgs []string
	var capturedCWD string
	err := runClaudeHook(strings.NewReader(input), &output, func(args []string, cwd string, timeout time.Duration) (string, string, int, error) {
		capturedArgs = append([]string{}, args...)
		capturedCWD = cwd
		if timeout != 7*time.Second {
			t.Fatalf("timeout = %s", timeout)
		}
		return "", "graph context", 0, nil
	}, func(cwd string) (string, error) {
		return "", nil
	})
	if err != nil {
		t.Fatalf("runClaudeHook: %v", err)
	}
	if strings.Join(capturedArgs, " ") != "augment -- AuthService" {
		t.Fatalf("augment args = %#v", capturedArgs)
	}
	if capturedCWD != repoDir {
		t.Fatalf("cwd = %q, want %q", capturedCWD, repoDir)
	}
	var payload map[string]map[string]string
	if err := json.Unmarshal(output.Bytes(), &payload); err != nil {
		t.Fatalf("parse hook output: %v\n%s", err, output.String())
	}
	hook := payload["hookSpecificOutput"]
	if hook["hookEventName"] != "PreToolUse" || hook["additionalContext"] != "graph context" {
		t.Fatalf("hook output = %#v", hook)
	}
}

func TestClaudeHookPostToolUseReportsStaleIndex(t *testing.T) {
	repoDir := t.TempDir()
	avmatrixDir := filepath.Join(repoDir, ".avmatrix")
	if err := os.Mkdir(avmatrixDir, 0o755); err != nil {
		t.Fatalf("mkdir .avmatrix: %v", err)
	}
	if err := os.WriteFile(filepath.Join(avmatrixDir, "meta.json"), []byte(`{"lastCommit":"abcdef123456","stats":{"embeddings":1}}`), 0o644); err != nil {
		t.Fatalf("write meta: %v", err)
	}
	input := `{"hook_event_name":"PostToolUse","tool_name":"Bash","tool_input":{"command":"git commit -m test"},"tool_output":{"exit_code":0},"cwd":` + strconvQuote(repoDir) + `}`
	var output bytes.Buffer
	err := runClaudeHook(strings.NewReader(input), &output, nil, func(cwd string) (string, error) {
		return "fedcba654321", nil
	})
	if err != nil {
		t.Fatalf("runClaudeHook: %v", err)
	}
	if !strings.Contains(output.String(), "AVmatrix index is stale") || !strings.Contains(output.String(), "avmatrix analyze --embeddings") {
		t.Fatalf("unexpected stale output:\n%s", output.String())
	}
}

func TestClaudeHookBashPatternSkipsFlagsWithValues(t *testing.T) {
	got := claudeHookBashPattern(`rg -g "*.go" "NewRootCommand" internal/cli`)
	if got != "NewRootCommand" {
		t.Fatalf("pattern = %q", got)
	}
}

func TestClaudeHookPatternExtraction(t *testing.T) {
	tests := []struct {
		name      string
		toolName  string
		toolInput map[string]any
		want      string
	}{
		{name: "grep", toolName: "Grep", toolInput: map[string]any{"pattern": "AuthService"}, want: "AuthService"},
		{name: "glob", toolName: "Glob", toolInput: map[string]any{"pattern": "**/AuthService*.ts"}, want: "AuthService"},
		{name: "bash rg", toolName: "Bash", toolInput: map[string]any{"command": `rg -g "*.go" "NewRootCommand" internal/cli`}, want: "NewRootCommand"},
		{name: "bash grep", toolName: "Bash", toolInput: map[string]any{"command": `grep -R validateUser src`}, want: "validateUser"},
		{name: "short", toolName: "Bash", toolInput: map[string]any{"command": `rg ab`}, want: ""},
		{name: "non search bash", toolName: "Bash", toolInput: map[string]any{"command": `git status`}, want: ""},
		{name: "other tool", toolName: "Read", toolInput: map[string]any{"pattern": "AuthService"}, want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := claudeHookPattern(tt.toolName, tt.toolInput); got != tt.want {
				t.Fatalf("claudeHookPattern() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestClaudeHookPreToolUseSilentCases(t *testing.T) {
	repoDir := t.TempDir()
	if err := os.Mkdir(filepath.Join(repoDir, ".avmatrix"), 0o755); err != nil {
		t.Fatalf("mkdir .avmatrix: %v", err)
	}
	noIndexDir := t.TempDir()
	calls := 0
	runner := func(args []string, cwd string, timeout time.Duration) (string, string, int, error) {
		calls++
		return "", "", 1, fmt.Errorf("not available")
	}
	head := func(cwd string) (string, error) { return "", nil }

	tests := []struct {
		name  string
		input string
	}{
		{name: "relative cwd", input: `{"hook_event_name":"PreToolUse","tool_name":"Grep","tool_input":{"pattern":"AuthService"},"cwd":"relative/path"}`},
		{name: "no index", input: `{"hook_event_name":"PreToolUse","tool_name":"Grep","tool_input":{"pattern":"AuthService"},"cwd":` + strconvQuote(noIndexDir) + `}`},
		{name: "short pattern", input: `{"hook_event_name":"PreToolUse","tool_name":"Grep","tool_input":{"pattern":"ab"},"cwd":` + strconvQuote(repoDir) + `}`},
		{name: "non search tool", input: `{"hook_event_name":"PreToolUse","tool_name":"Read","tool_input":{"pattern":"AuthService"},"cwd":` + strconvQuote(repoDir) + `}`},
		{name: "runner failure", input: `{"hook_event_name":"PreToolUse","tool_name":"Grep","tool_input":{"pattern":"AuthService"},"cwd":` + strconvQuote(repoDir) + `}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var output bytes.Buffer
			if err := runClaudeHook(strings.NewReader(tt.input), &output, runner, head); err != nil {
				t.Fatalf("runClaudeHook() error = %v", err)
			}
			if strings.TrimSpace(output.String()) != "" {
				t.Fatalf("output = %q, want empty", output.String())
			}
		})
	}
	if calls != 1 {
		t.Fatalf("runner calls = %d, want 1", calls)
	}
}

func TestClaudeHookPostToolUseSilentCases(t *testing.T) {
	repoDir := t.TempDir()
	avmatrixDir := filepath.Join(repoDir, ".avmatrix")
	if err := os.Mkdir(avmatrixDir, 0o755); err != nil {
		t.Fatalf("mkdir .avmatrix: %v", err)
	}
	writeClaudeHookMeta(t, avmatrixDir, `{"lastCommit":"head123","stats":{"embeddings":0}}`)
	noIndexDir := t.TempDir()

	tests := []struct {
		name  string
		input string
		head  claudeHookHeadFunc
	}{
		{name: "non bash", input: `{"hook_event_name":"PostToolUse","tool_name":"Grep","tool_input":{"command":"git commit -m test"},"cwd":` + strconvQuote(repoDir) + `}`, head: fixedHead("newhead")},
		{name: "non mutation", input: `{"hook_event_name":"PostToolUse","tool_name":"Bash","tool_input":{"command":"git status"},"tool_output":{"exit_code":0},"cwd":` + strconvQuote(repoDir) + `}`, head: fixedHead("newhead")},
		{name: "failed git", input: `{"hook_event_name":"PostToolUse","tool_name":"Bash","tool_input":{"command":"git commit -m test"},"tool_output":{"exit_code":1},"cwd":` + strconvQuote(repoDir) + `}`, head: fixedHead("newhead")},
		{name: "relative cwd", input: `{"hook_event_name":"PostToolUse","tool_name":"Bash","tool_input":{"command":"git commit -m test"},"tool_output":{"exit_code":0},"cwd":"relative/path"}`, head: fixedHead("newhead")},
		{name: "no index", input: `{"hook_event_name":"PostToolUse","tool_name":"Bash","tool_input":{"command":"git commit -m test"},"tool_output":{"exit_code":0},"cwd":` + strconvQuote(noIndexDir) + `}`, head: fixedHead("newhead")},
		{name: "head error", input: `{"hook_event_name":"PostToolUse","tool_name":"Bash","tool_input":{"command":"git commit -m test"},"tool_output":{"exit_code":0},"cwd":` + strconvQuote(repoDir) + `}`, head: func(string) (string, error) { return "", fmt.Errorf("git failed") }},
		{name: "matching head", input: `{"hook_event_name":"PostToolUse","tool_name":"Bash","tool_input":{"command":"git commit -m test"},"tool_output":{"exit_code":0},"cwd":` + strconvQuote(repoDir) + `}`, head: fixedHead("head123")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var output bytes.Buffer
			if err := runClaudeHook(strings.NewReader(tt.input), &output, nil, tt.head); err != nil {
				t.Fatalf("runClaudeHook() error = %v", err)
			}
			if strings.TrimSpace(output.String()) != "" {
				t.Fatalf("output = %q, want empty", output.String())
			}
		})
	}
}

func TestClaudeHookPostToolUseGitMutationVariants(t *testing.T) {
	repoDir := t.TempDir()
	avmatrixDir := filepath.Join(repoDir, ".avmatrix")
	if err := os.Mkdir(avmatrixDir, 0o755); err != nil {
		t.Fatalf("mkdir .avmatrix: %v", err)
	}
	writeClaudeHookMeta(t, avmatrixDir, `{"lastCommit":"oldcommit","stats":{"embeddings":0}}`)
	commands := []string{
		"git commit -m test",
		"git merge feature",
		"git rebase main",
		"git cherry-pick abc123",
		"git pull origin main",
	}
	for _, command := range commands {
		t.Run(command, func(t *testing.T) {
			input := `{"hook_event_name":"PostToolUse","tool_name":"Bash","tool_input":{"command":` + strconvQuote(command) + `},"tool_output":{"exit_code":0},"cwd":` + strconvQuote(repoDir) + `}`
			var output bytes.Buffer
			if err := runClaudeHook(strings.NewReader(input), &output, nil, fixedHead("newhead")); err != nil {
				t.Fatalf("runClaudeHook() error = %v", err)
			}
			if !strings.Contains(output.String(), "AVmatrix index is stale") {
				t.Fatalf("output = %q, want stale notification", output.String())
			}
		})
	}
}

func TestClaudeHookPostToolUseMissingOrCorruptMetaTreatsAsStale(t *testing.T) {
	for _, tt := range []struct {
		name string
		meta *string
	}{
		{name: "missing", meta: nil},
		{name: "corrupt", meta: ptrString("not valid json")},
	} {
		t.Run(tt.name, func(t *testing.T) {
			repoDir := t.TempDir()
			avmatrixDir := filepath.Join(repoDir, ".avmatrix")
			if err := os.Mkdir(avmatrixDir, 0o755); err != nil {
				t.Fatalf("mkdir .avmatrix: %v", err)
			}
			if tt.meta != nil {
				writeClaudeHookMeta(t, avmatrixDir, *tt.meta)
			}
			input := `{"hook_event_name":"PostToolUse","tool_name":"Bash","tool_input":{"command":"git commit -m test"},"tool_output":{"exit_code":0},"cwd":` + strconvQuote(repoDir) + `}`
			var output bytes.Buffer
			if err := runClaudeHook(strings.NewReader(input), &output, nil, fixedHead("newhead")); err != nil {
				t.Fatalf("runClaudeHook() error = %v", err)
			}
			if !strings.Contains(output.String(), "last indexed: never") {
				t.Fatalf("output = %q, want never", output.String())
			}
		})
	}
}

func TestClaudeHookPostToolUseEmbeddingsHint(t *testing.T) {
	repoDir := t.TempDir()
	avmatrixDir := filepath.Join(repoDir, ".avmatrix")
	if err := os.Mkdir(avmatrixDir, 0o755); err != nil {
		t.Fatalf("mkdir .avmatrix: %v", err)
	}
	writeClaudeHookMeta(t, avmatrixDir, `{"lastCommit":"oldcommit","stats":{"embeddings":42}}`)
	input := `{"hook_event_name":"PostToolUse","tool_name":"Bash","tool_input":{"command":"git merge feature"},"tool_output":{"exit_code":0},"cwd":` + strconvQuote(repoDir) + `}`
	var output bytes.Buffer
	if err := runClaudeHook(strings.NewReader(input), &output, nil, fixedHead("newhead")); err != nil {
		t.Fatalf("runClaudeHook() error = %v", err)
	}
	if !strings.Contains(output.String(), "avmatrix analyze --embeddings") {
		t.Fatalf("output = %q, want embeddings hint", output.String())
	}
}

func TestRunClaudeHookIgnoresMalformedInputAndUnknownEvents(t *testing.T) {
	inputs := []string{"", "not json", `{"hook_event_name":"UnknownEvent","tool_name":"Bash"}`, `{}`}
	for _, input := range inputs {
		t.Run(input, func(t *testing.T) {
			var output bytes.Buffer
			if err := runClaudeHook(strings.NewReader(input), &output, nil, fixedHead("head")); err != nil {
				t.Fatalf("runClaudeHook() error = %v", err)
			}
			if strings.TrimSpace(output.String()) != "" {
				t.Fatalf("output = %q, want empty", output.String())
			}
		})
	}
}

func TestFindClaudeHookAVmatrixDirWalksParents(t *testing.T) {
	repoDir := t.TempDir()
	if err := os.Mkdir(filepath.Join(repoDir, ".avmatrix"), 0o755); err != nil {
		t.Fatalf("mkdir .avmatrix: %v", err)
	}
	nested := filepath.Join(repoDir, "a", "b", "c")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("mkdir nested: %v", err)
	}
	if got := findClaudeHookAVmatrixDir(nested); got != filepath.Join(repoDir, ".avmatrix") {
		t.Fatalf("findClaudeHookAVmatrixDir() = %q", got)
	}
}

func TestClaudePluginHookSourceStaticGuards(t *testing.T) {
	source := readRepoFile(t, "avmatrix-claude-plugin", "hooks", "avmatrix-hook.js")
	if !strings.Contains(source, "npx.cmd") || !strings.Contains(source, "avmatrix.cmd") {
		t.Fatalf("plugin hook is missing Windows .cmd command handling")
	}
	for index, line := range strings.Split(source, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "*") {
			continue
		}
		if strings.Contains(line, "shell: true") || strings.Contains(line, "shell: isWin") {
			t.Fatalf("plugin hook line %d contains shell execution risk: %s", index+1, trimmed)
		}
	}
	if strings.Count(source, "path.isAbsolute(cwd)") < 2 {
		t.Fatalf("plugin hook is missing cwd absolute-path guards")
	}
	if !strings.Contains(source, "const handlers = {") ||
		!strings.Contains(source, "PreToolUse: handlePreToolUse") ||
		!strings.Contains(source, "PostToolUse: handlePostToolUse") {
		t.Fatalf("plugin hook is missing dispatch map routing")
	}
	if strings.Count(source, "hookSpecificOutput") != 1 {
		t.Fatalf("plugin hook should centralize hookSpecificOutput JSON in sendHookResponse")
	}
	if !strings.Contains(source, ".slice(0, 200)") {
		t.Fatalf("plugin hook should truncate debug errors")
	}
}

func fixedHead(head string) claudeHookHeadFunc {
	return func(string) (string, error) {
		return head, nil
	}
}

func writeClaudeHookMeta(t *testing.T, avmatrixDir string, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(avmatrixDir, "meta.json"), []byte(content), 0o644); err != nil {
		t.Fatalf("write meta: %v", err)
	}
}

func ptrString(value string) *string {
	return &value
}

func readRepoFile(t *testing.T, parts ...string) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for {
		candidate := filepath.Join(append([]string{dir}, parts...)...)
		raw, err := os.ReadFile(candidate)
		if err == nil {
			return string(raw)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("repo file %v not found", parts)
		}
		dir = parent
	}
}
