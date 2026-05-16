package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

type claudeHookInput struct {
	HookEventName string         `json:"hook_event_name"`
	ToolName      string         `json:"tool_name"`
	ToolInput     map[string]any `json:"tool_input"`
	ToolOutput    map[string]any `json:"tool_output"`
	CWD           string         `json:"cwd"`
}

type claudeHookRunner func(args []string, cwd string, timeout time.Duration) (stdout string, stderr string, status int, err error)
type claudeHookHeadFunc func(cwd string) (string, error)

var (
	claudeHookGlobPattern = regexp.MustCompile(`[*\\/]([a-zA-Z][a-zA-Z0-9_-]{2,})`)
	claudeHookGitMutation = regexp.MustCompile(`\bgit\s+(commit|merge|rebase|cherry-pick|pull)(\s|$)`)
)

func newHookCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "hook",
		Short:  "Run editor hook integrations",
		Hidden: true,
	}
	cmd.AddCommand(&cobra.Command{
		Use:    "claude",
		Short:  "Run Claude Code hook integration",
		Hidden: true,
		Args:   cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runClaudeHook(cmd.InOrStdin(), cmd.OutOrStdout(), runClaudeHookCLI, claudeHookCurrentHead)
		},
	})
	return cmd
}

func runClaudeHook(input io.Reader, output io.Writer, runner claudeHookRunner, currentHead claudeHookHeadFunc) error {
	raw, err := io.ReadAll(input)
	if err != nil {
		return nil
	}
	var payload claudeHookInput
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil
	}
	switch payload.HookEventName {
	case "PreToolUse":
		return handleClaudePreToolUse(payload, output, runner)
	case "PostToolUse":
		return handleClaudePostToolUse(payload, output, currentHead)
	default:
		return nil
	}
}

func handleClaudePreToolUse(input claudeHookInput, output io.Writer, runner claudeHookRunner) error {
	cwd := input.CWD
	if cwd == "" {
		if current, err := os.Getwd(); err == nil {
			cwd = current
		}
	}
	if !filepath.IsAbs(cwd) || findClaudeHookAVmatrixDir(cwd) == "" {
		return nil
	}
	if input.ToolName != "Grep" && input.ToolName != "Glob" && input.ToolName != "Bash" {
		return nil
	}
	pattern := claudeHookPattern(input.ToolName, input.ToolInput)
	if len(pattern) < 3 {
		return nil
	}
	stdout, stderr, status, err := runner([]string{"augment", "--", pattern}, cwd, 7*time.Second)
	if err != nil || status != 0 {
		return nil
	}
	message := strings.TrimSpace(stderr)
	if message == "" {
		message = strings.TrimSpace(stdout)
	}
	if message == "" {
		return nil
	}
	return writeClaudeHookResponse(output, "PreToolUse", message)
}

func handleClaudePostToolUse(input claudeHookInput, output io.Writer, currentHead claudeHookHeadFunc) error {
	if input.ToolName != "Bash" {
		return nil
	}
	command := strings.TrimSpace(stringMapValue(input.ToolInput, "command"))
	if !claudeHookGitMutation.MatchString(command) {
		return nil
	}
	if exitCode, ok := numericMapValue(input.ToolOutput, "exit_code"); ok && exitCode != 0 {
		return nil
	}
	cwd := input.CWD
	if cwd == "" {
		if current, err := os.Getwd(); err == nil {
			cwd = current
		}
	}
	if !filepath.IsAbs(cwd) {
		return nil
	}
	avmatrixDir := findClaudeHookAVmatrixDir(cwd)
	if avmatrixDir == "" {
		return nil
	}
	head, err := currentHead(cwd)
	if err != nil || head == "" {
		return nil
	}
	meta := struct {
		LastCommit string `json:"lastCommit"`
		Stats      struct {
			Embeddings int `json:"embeddings"`
		} `json:"stats"`
	}{}
	_ = readJSONFile(filepath.Join(avmatrixDir, "meta.json"), &meta)
	if head == meta.LastCommit {
		return nil
	}
	analyzeCmd := "avmatrix analyze"
	if meta.Stats.Embeddings > 0 {
		analyzeCmd += " --embeddings"
	}
	last := "never"
	if meta.LastCommit != "" {
		last = shortCommit(meta.LastCommit)
	}
	return writeClaudeHookResponse(
		output,
		"PostToolUse",
		fmt.Sprintf("AVmatrix index is stale (last indexed: %s). Run `%s` to update the knowledge graph.", last, analyzeCmd),
	)
}

func claudeHookPattern(toolName string, toolInput map[string]any) string {
	switch toolName {
	case "Grep":
		return strings.TrimSpace(stringMapValue(toolInput, "pattern"))
	case "Glob":
		match := claudeHookGlobPattern.FindStringSubmatch(stringMapValue(toolInput, "pattern"))
		if len(match) == 2 {
			return match[1]
		}
	case "Bash":
		command := stringMapValue(toolInput, "command")
		if !strings.Contains(command, "rg") && !strings.Contains(command, "grep") {
			return ""
		}
		return claudeHookBashPattern(command)
	}
	return ""
}

func claudeHookBashPattern(command string) string {
	tokens := strings.Fields(command)
	foundCommand := false
	skipNext := false
	flagsWithValues := map[string]bool{
		"-e": true, "-f": true, "-m": true, "-A": true, "-B": true, "-C": true,
		"-g": true, "--glob": true, "-t": true, "--type": true, "--include": true, "--exclude": true,
	}
	for _, token := range tokens {
		if skipNext {
			skipNext = false
			continue
		}
		cleanedCommand := strings.Trim(token, `"'`)
		if !foundCommand {
			base := filepath.Base(path.Base(cleanedCommand))
			if base == "rg" || base == "grep" || base == "rg.exe" || base == "grep.exe" {
				foundCommand = true
			}
			continue
		}
		if strings.HasPrefix(token, "-") {
			if flagsWithValues[token] {
				skipNext = true
			}
			continue
		}
		cleaned := strings.Trim(token, `"'`)
		if len(cleaned) >= 3 {
			return cleaned
		}
	}
	return ""
}

func findClaudeHookAVmatrixDir(startDir string) string {
	dir := startDir
	for i := 0; i < 5; i++ {
		candidate := filepath.Join(dir, ".avmatrix")
		if stat, err := os.Stat(candidate); err == nil && stat.IsDir() {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

func runClaudeHookCLI(args []string, cwd string, timeout time.Duration) (string, string, int, error) {
	exe, err := os.Executable()
	if err != nil || exe == "" {
		exe = setupCommandName
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, exe, args...)
	cmd.Dir = cwd
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()
	status := 0
	if err != nil {
		status = 1
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			status = exitErr.ExitCode()
		}
	}
	return stdout.String(), stderr.String(), status, err
}

func claudeHookCurrentHead(cwd string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "HEAD")
	cmd.Dir = cwd
	raw, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(raw)), nil
}

func writeClaudeHookResponse(output io.Writer, hookEventName string, message string) error {
	encoder := json.NewEncoder(output)
	return encoder.Encode(map[string]any{
		"hookSpecificOutput": map[string]any{
			"hookEventName":     hookEventName,
			"additionalContext": message,
		},
	})
}

func numericMapValue(values map[string]any, key string) (int, bool) {
	switch value := values[key].(type) {
	case int:
		return value, true
	case float64:
		return int(value), true
	default:
		return 0, false
	}
}

func readJSONFile(path string, target any) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, target)
}
