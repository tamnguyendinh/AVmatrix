package session

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

type CommandResult struct {
	Code   int
	Stdout string
	Stderr string
}

type CommandRunner interface {
	Run(ctx context.Context, command string, args []string, cwd string) (CommandResult, error)
}

type ExecRunner struct{}

func (ExecRunner) Run(ctx context.Context, command string, args []string, cwd string) (CommandResult, error) {
	cmd := exec.CommandContext(ctx, command, args...)
	if cwd != "" {
		cmd.Dir = cwd
	}
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	code := 0
	if err != nil {
		code = 1
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			code = exitErr.ExitCode()
			err = nil
		}
	}
	return CommandResult{Code: code, Stdout: stdout.String(), Stderr: stderr.String()}, err
}

type CodexAdapterOptions struct {
	Runner        CommandRunner
	Platform      string
	ExecutionMode ExecutionMode
	FinalReader   func(path string) (string, error)
	TempDir       string
}

type CodexAdapter struct {
	runner        CommandRunner
	platform      string
	executionMode ExecutionMode
	runtimeEnv    RuntimeEnvironment
	finalReader   func(path string) (string, error)
	tempDir       string
}

type codexProbe struct {
	runtimeEnvironment RuntimeEnvironment
	executablePath     string
	displayPath        string
	available          bool
	authenticated      bool
	version            string
	message            string
}

func NewCodexAdapter(options CodexAdapterOptions) *CodexAdapter {
	runner := options.Runner
	if runner == nil {
		runner = ExecRunner{}
	}
	platform := options.Platform
	if platform == "" {
		platform = runtime.GOOS
	}
	mode := options.ExecutionMode
	if mode == "" {
		if configured := os.Getenv("AVMATRIX_SESSION_EXECUTION_MODE"); configured == "bypass" {
			mode = ExecutionModeBypass
		} else if platform == "windows" {
			mode = ExecutionModeBypass
		} else {
			mode = ExecutionModeSandboxed
		}
	}
	reader := options.FinalReader
	if reader == nil {
		reader = func(path string) (string, error) {
			raw, err := os.ReadFile(path)
			return string(raw), err
		}
	}
	tempDir := options.TempDir
	if tempDir == "" {
		tempDir = os.TempDir()
	}
	return &CodexAdapter{
		runner:        runner,
		platform:      platform,
		executionMode: mode,
		runtimeEnv:    RuntimeNative,
		finalReader:   reader,
		tempDir:       tempDir,
	}
}

func (a *CodexAdapter) Provider() Provider {
	return ProviderCodex
}

func (a *CodexAdapter) ExecutionMode() ExecutionMode {
	return a.executionMode
}

func (a *CodexAdapter) RuntimeEnvironment() RuntimeEnvironment {
	return a.runtimeEnv
}

func (a *CodexAdapter) GetStatus(ctx context.Context) (Status, error) {
	probe := a.resolveLaunchTarget(ctx)
	a.runtimeEnv = probe.runtimeEnvironment
	availability := AvailabilityNotInstalled
	if probe.available {
		if probe.authenticated {
			availability = AvailabilityReady
		} else {
			availability = AvailabilityNotSignedIn
		}
	}

	recommended := RuntimeNative
	if a.platform == "windows" {
		recommended = RuntimeWSL2
	}
	executable := probe.executablePath
	if probe.displayPath != "" {
		executable = probe.displayPath
	}
	if probe.runtimeEnvironment == RuntimeWSL2 && executable == "" {
		executable = "wsl.exe -> codex"
	}

	return Status{
		Provider:               ProviderCodex,
		Availability:           availability,
		Available:              probe.available,
		Authenticated:          probe.authenticated,
		ExecutablePath:         executable,
		Version:                probe.version,
		Message:                probe.message,
		RecommendedEnvironment: recommended,
		RuntimeEnvironment:     probe.runtimeEnvironment,
		ExecutionMode:          a.executionMode,
		SupportsSSE:            true,
		SupportsCancel:         true,
		SupportsMCP:            true,
	}, nil
}

func (a *CodexAdapter) RunChat(ctx context.Context, job *Job, request ChatRequest, chatContext ChatContext) error {
	status, err := a.GetStatus(ctx)
	if err != nil {
		return err
	}
	if !status.Available {
		return NewRuntimeError(
			ErrorSessionRuntimeUnavailable,
			firstNonEmpty(status.Message, "Codex CLI is not available"),
			503,
			nil,
		)
	}
	if !status.Authenticated {
		return NewRuntimeError(
			ErrorSessionNotSignedIn,
			firstNonEmpty(status.Message, "Codex CLI is not signed in"),
			401,
			nil,
		)
	}

	outputFile := filepath.Join(a.tempDir, "avmatrix-codex-"+job.ID+".txt")
	runtimeRepoPath := chatContext.Repo.RepoPath
	runtimeOutputPath := outputFile
	if status.RuntimeEnvironment == RuntimeWSL2 {
		runtimeRepoPath = toWslPath(runtimeRepoPath)
		runtimeOutputPath = toWslPath(runtimeOutputPath)
	}

	args := []string{
		"exec",
		"--json",
		"--skip-git-repo-check",
		"--output-last-message",
		runtimeOutputPath,
		"--cd",
		runtimeRepoPath,
	}
	if a.executionMode == ExecutionModeBypass {
		args = append(args, "--dangerously-bypass-approvals-and-sandbox")
	} else {
		args = append(args, "--full-auto")
	}
	args = append(args, request.Message)

	command := nativeCodexExecutable(a.platform)
	commandArgs := args
	if status.RuntimeEnvironment == RuntimeWSL2 {
		command = "wsl.exe"
		commandArgs = []string{"-e", "bash", "-lc", shellJoin(append([]string{"codex"}, args...))}
	}

	result, err := a.runner.Run(ctx, command, commandArgs, chatContext.Repo.RepoPath)
	if ctx.Err() != nil {
		job.Emit(Event{Type: "cancelled", Reason: cancelMessage(ctx)})
		return nil
	}
	if err != nil {
		return err
	}

	lastReasoning := ""
	var usage map[string]int
	for _, line := range strings.Split(result.Stdout, "\n") {
		event, reasoning, parsedUsage, ok := ParseCodexEventLine(line)
		if !ok {
			continue
		}
		if event != nil {
			job.Emit(*event)
		}
		if reasoning != "" {
			lastReasoning = reasoning
		}
		if parsedUsage != nil {
			usage = parsedUsage
		}
	}

	if result.Code == 0 {
		finalContent := ""
		if content, err := a.finalReader(outputFile); err == nil {
			finalContent = strings.TrimSpace(content)
		}
		if finalContent == "" {
			finalContent = lastReasoning
		}
		if finalContent != "" {
			job.Emit(Event{Type: "content", Content: finalContent})
		}
		job.Emit(Event{Type: "done", Usage: usage})
		return nil
	}

	job.Emit(Event{
		Type:  "error",
		Code:  ErrorSessionStartFailed,
		Error: firstNonEmpty(strings.TrimSpace(result.Stderr), fmt.Sprintf("Codex exited with code %d", result.Code)),
	})
	return nil
}

func ParseCodexEventLine(line string) (*Event, string, map[string]int, bool) {
	line = strings.TrimSpace(line)
	if line == "" {
		return nil, "", nil, false
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(line), &payload); err != nil {
		return nil, "", nil, false
	}
	eventType, _ := payload["type"].(string)
	switch eventType {
	case "item.started", "item.completed":
		item, _ := payload["item"].(map[string]any)
		if item == nil {
			return nil, "", nil, true
		}
		status := "completed"
		outputType := "tool_result"
		if eventType == "item.started" {
			status = "running"
			outputType = "tool_call"
		}
		if tool := coerceToolCall(item, status); tool != nil {
			return &Event{Type: outputType, ToolCall: tool}, "", nil, true
		}
		if itemType, _ := item["type"].(string); itemType == "agent_message" {
			if text, _ := item["text"].(string); text != "" {
				return &Event{Type: "reasoning", Reasoning: text}, text, nil, true
			}
		}
	case "agent_message":
		if text, _ := payload["text"].(string); text != "" {
			return &Event{Type: "reasoning", Reasoning: text}, text, nil, true
		}
	case "turn.completed":
		usage := map[string]int{}
		if rawUsage, _ := payload["usage"].(map[string]any); rawUsage != nil {
			for key, value := range rawUsage {
				switch typed := value.(type) {
				case float64:
					usage[key] = int(typed)
				case int:
					usage[key] = typed
				}
			}
		}
		if len(usage) > 0 {
			return nil, "", usage, true
		}
	}
	return nil, "", nil, true
}

func (a *CodexAdapter) resolveLaunchTarget(ctx context.Context) codexProbe {
	if a.platform == "windows" {
		wsl := a.probeWSL(ctx)
		if wsl.available {
			return wsl
		}
		native := a.probeNative(ctx)
		if native.available {
			return native
		}
		native.message = strings.Join(nonEmpty(
			"No usable local Codex runtime was found on Windows.",
			"Preferred: install Codex CLI inside WSL2.",
			"Fallback: ensure native Codex CLI is installed and signed in on Windows.",
			wsl.message,
			native.message,
		), " ")
		return native
	}
	return a.probeNative(ctx)
}

func (a *CodexAdapter) probeWSL(ctx context.Context) codexProbe {
	which := a.run(ctx, "wsl.exe", []string{"-e", "bash", "-lc", "command -v codex || true"})
	path := lastNonEmptyLine(which.Stdout)
	if path == "" {
		return codexProbe{runtimeEnvironment: RuntimeWSL2, message: "Codex CLI is not installed inside WSL2 or not available on the WSL PATH."}
	}
	if strings.HasPrefix(strings.ToLower(path), "/mnt/") {
		return codexProbe{
			runtimeEnvironment: RuntimeWSL2,
			message:            "WSL2 is resolving Codex to a Windows-mounted shim at \"" + path + "\". Install Codex CLI inside WSL2 or fix the WSL PATH order.",
		}
	}

	version := a.run(ctx, "wsl.exe", []string{"-e", "bash", "-lc", shellQuote(path) + " --version"})
	if version.Code != 0 {
		return codexProbe{runtimeEnvironment: RuntimeWSL2, message: firstNonEmpty(version.Stderr, version.Stdout, "codex --version failed inside WSL2")}
	}
	login := a.run(ctx, "wsl.exe", []string{"-e", "bash", "-lc", shellQuote(path) + " login status"})
	loginOutput := strings.TrimSpace(login.Stdout + "\n" + login.Stderr)
	authenticated := strings.Contains(strings.ToLower(loginOutput), "logged in") && !strings.Contains(strings.ToLower(loginOutput), "not logged in")
	return codexProbe{
		runtimeEnvironment: RuntimeWSL2,
		executablePath:     "wsl.exe -> codex",
		available:          true,
		authenticated:      authenticated,
		version:            strings.TrimSpace(firstNonEmpty(version.Stdout, version.Stderr)),
		message:            ternary(authenticated, "", firstNonEmpty(loginOutput, "Codex CLI is installed in WSL2 but not signed in")),
	}
}

func (a *CodexAdapter) probeNative(ctx context.Context) codexProbe {
	executable := nativeCodexExecutable(a.platform)
	version := a.run(ctx, executable, []string{"--version"})
	if version.Code != 0 {
		return codexProbe{
			runtimeEnvironment: RuntimeNative,
			executablePath:     executable,
			message:            firstNonEmpty(version.Stderr, version.Stdout, "codex --version failed"),
		}
	}
	login := a.run(ctx, executable, []string{"login", "status"})
	loginOutput := strings.TrimSpace(login.Stdout + "\n" + login.Stderr)
	authenticated := strings.Contains(strings.ToLower(loginOutput), "logged in") && !strings.Contains(strings.ToLower(loginOutput), "not logged in")
	return codexProbe{
		runtimeEnvironment: RuntimeNative,
		executablePath:     executable,
		available:          true,
		authenticated:      authenticated,
		version:            strings.TrimSpace(firstNonEmpty(version.Stdout, version.Stderr)),
		message:            ternary(authenticated, "", firstNonEmpty(loginOutput, "Codex CLI is installed but not signed in")),
	}
}

func (a *CodexAdapter) run(ctx context.Context, command string, args []string) CommandResult {
	result, err := a.runner.Run(ctx, command, args, "")
	if err != nil {
		result.Code = 1
		if result.Stderr == "" {
			result.Stderr = err.Error()
		}
	}
	return result
}

func coerceToolCall(item map[string]any, status string) *ToolCall {
	itemType, _ := item["type"].(string)
	if itemType != "command_execution" && itemType != "mcp_tool_call" {
		return nil
	}
	id, _ := item["id"].(string)
	if id == "" {
		id = newID()
	}
	command := ""
	switch raw := item["command"].(type) {
	case string:
		command = raw
	case []any:
		parts := make([]string, 0, len(raw))
		for _, part := range raw {
			parts = append(parts, strings.TrimSpace(toString(part)))
		}
		command = strings.Join(parts, " ")
	}
	name := command
	if itemType == "mcp_tool_call" {
		name = firstNonEmpty(toString(item["name"]), toString(item["tool_name"]), "mcp_tool_call")
	}
	if name == "" {
		name = firstNonEmpty(toString(item["name"]), "command_execution")
	}

	results := nonEmpty(
		toString(item["aggregated_output"]),
		toString(item["stdout"]),
		toString(item["stderr"]),
		toString(item["output"]),
		toString(item["text"]),
	)
	args := map[string]any(nil)
	if command != "" {
		args = map[string]any{"command": command}
	}
	return &ToolCall{
		ID:     id,
		Name:   name,
		Args:   args,
		Result: strings.TrimSpace(strings.Join(results, "\n")),
		Status: status,
	}
}

func nativeCodexExecutable(platform string) string {
	if configured := os.Getenv("AVMATRIX_CODEX_EXECUTABLE"); configured != "" {
		return configured
	}
	if platform == "windows" {
		return "codex.cmd"
	}
	return "codex"
}

func toWslPath(value string) string {
	if len(value) >= 3 && value[1] == ':' && (value[2] == '\\' || value[2] == '/') {
		drive := strings.ToLower(value[:1])
		rest := strings.ReplaceAll(value[2:], "\\", "/")
		return "/mnt/" + drive + rest
	}
	return strings.ReplaceAll(value, "\\", "/")
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

func shellJoin(args []string) string {
	quoted := make([]string, 0, len(args))
	for _, arg := range args {
		quoted = append(quoted, shellQuote(arg))
	}
	return strings.Join(quoted, " ")
}

func cancelMessage(ctx context.Context) string {
	if cause := context.Cause(ctx); cause != nil {
		return cause.Error()
	}
	return "Session cancelled"
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func nonEmpty(values ...string) []string {
	filtered := make([]string, 0, len(values))
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			filtered = append(filtered, strings.TrimSpace(value))
		}
	}
	return filtered
}

func lastNonEmptyLine(value string) string {
	lines := strings.Split(value, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		if trimmed := strings.TrimSpace(lines[i]); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func toString(value any) string {
	if value == nil {
		return ""
	}
	if text, ok := value.(string); ok {
		return text
	}
	return fmt.Sprint(value)
}

func ternary[T any](condition bool, yes T, no T) T {
	if condition {
		return yes
	}
	return no
}
