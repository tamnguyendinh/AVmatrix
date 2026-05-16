package session

import (
	"context"
	"strings"
	"testing"
)

type fakeRunner struct {
	run func(command string, args []string, cwd string) CommandResult
}

func (r fakeRunner) Run(_ context.Context, command string, args []string, cwd string) (CommandResult, error) {
	return r.run(command, args, cwd), nil
}

func TestCodexAdapterPrefersWSL2OnWindowsWhenUsable(t *testing.T) {
	adapter := NewCodexAdapter(CodexAdapterOptions{
		Platform: "windows",
		Runner: fakeRunner{run: func(command string, args []string, _ string) CommandResult {
			line := command + " " + strings.Join(args, " ")
			switch {
			case strings.Contains(line, "command -v codex"):
				return CommandResult{Stdout: "/home/test/.local/bin/codex\n"}
			case strings.Contains(line, "--version"):
				return CommandResult{Stdout: "codex-cli test-version\n"}
			case strings.Contains(line, "login status"):
				return CommandResult{Stdout: "Logged in using ChatGPT\n"}
			default:
				return CommandResult{Code: 1}
			}
		}},
	})

	status, err := adapter.GetStatus(context.Background())
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if status.RuntimeEnvironment != RuntimeWSL2 || !status.Available || !status.Authenticated {
		t.Fatalf("unexpected status: %#v", status)
	}
}

func TestCodexAdapterFallsBackToNativeWhenWSLSeesWindowsShim(t *testing.T) {
	adapter := NewCodexAdapter(CodexAdapterOptions{
		Platform: "windows",
		Runner: fakeRunner{run: func(command string, args []string, _ string) CommandResult {
			line := command + " " + strings.Join(args, " ")
			switch {
			case command == "wsl.exe" && strings.Contains(line, "command -v codex"):
				return CommandResult{Stdout: "/mnt/c/Users/test/AppData/Roaming/npm/codex\n"}
			case command == "codex.cmd" && strings.Contains(line, "--version"):
				return CommandResult{Stdout: "codex-cli test-version\n"}
			case command == "codex.cmd" && strings.Contains(line, "login status"):
				return CommandResult{Stdout: "Logged in using ChatGPT\n"}
			default:
				return CommandResult{Code: 1, Stderr: "missing"}
			}
		}},
	})

	status, err := adapter.GetStatus(context.Background())
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if status.RuntimeEnvironment != RuntimeNative || !status.Available || !status.Authenticated {
		t.Fatalf("unexpected status: %#v", status)
	}
}

func TestCodexAdapterReportsUnavailableWhenNoRuntimeUsable(t *testing.T) {
	adapter := NewCodexAdapter(CodexAdapterOptions{
		Platform: "windows",
		Runner: fakeRunner{run: func(command string, args []string, _ string) CommandResult {
			if command == "wsl.exe" {
				return CommandResult{Stdout: "/mnt/c/Users/test/AppData/Roaming/npm/codex\n"}
			}
			return CommandResult{Code: 1, Stderr: "codex.cmd: command not found"}
		}},
	})

	status, err := adapter.GetStatus(context.Background())
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if status.Available || !strings.Contains(status.Message, "No usable local Codex runtime was found on Windows") || !strings.Contains(status.Message, "Preferred: install Codex CLI inside WSL2") {
		t.Fatalf("unexpected unavailable status: %#v", status)
	}
}

func TestCodexAdapterMapsEventsAndFinalContent(t *testing.T) {
	execStdout := strings.Join([]string{
		`{"type":"item.started","item":{"id":"cmd-1","type":"command_execution","command":["dir"]}}`,
		`{"type":"item.completed","item":{"id":"cmd-1","type":"command_execution","command":["dir"],"aggregated_output":"file-a\nfile-b"}}`,
		`{"type":"item.completed","item":{"id":"msg-1","type":"agent_message","text":"Thinking step"}}`,
		`{"type":"turn.completed","usage":{"output_tokens":5}}`,
	}, "\n") + "\n"
	adapter := NewCodexAdapter(CodexAdapterOptions{
		Platform: "linux",
		Runner: fakeRunner{run: func(_ string, args []string, _ string) CommandResult {
			line := strings.Join(args, " ")
			switch {
			case strings.Contains(line, "--version"):
				return CommandResult{Stdout: "codex-cli test-version\n"}
			case strings.Contains(line, "login status"):
				return CommandResult{Stdout: "Logged in using ChatGPT\n"}
			case strings.Contains(line, "exec"):
				return CommandResult{Stdout: execStdout}
			default:
				return CommandResult{Code: 1}
			}
		}},
		FinalReader: func(string) (string, error) { return "Final answer", nil },
	})
	job := NewJob(ProviderCodex, "demo", "/repo")
	events, unsubscribe := job.Subscribe(true)
	defer unsubscribe()

	err := adapter.RunChat(context.Background(), job, ChatRequest{Message: "hello"}, ChatContext{
		Repo: ResolvedRepo{RepoName: "demo", RepoPath: "/repo", Indexed: true},
	})
	if err != nil {
		t.Fatalf("run chat: %v", err)
	}

	var got []Event
	for event := range events {
		got = append(got, event)
	}
	assertHasEvent(t, got, func(event Event) bool {
		return event.Type == "reasoning" && event.Reasoning == "Thinking step"
	})
	assertHasEvent(t, got, func(event Event) bool {
		return event.Type == "tool_result" && event.ToolCall != nil && event.ToolCall.Result == "file-a\nfile-b"
	})
	assertHasEvent(t, got, func(event Event) bool {
		return event.Type == "content" && event.Content == "Final answer"
	})
	assertHasEvent(t, got, func(event Event) bool {
		return event.Type == "done" && event.Usage["output_tokens"] == 5
	})
}

func assertHasEvent(t *testing.T, events []Event, predicate func(Event) bool) {
	t.Helper()
	for _, event := range events {
		if predicate(event) {
			return
		}
	}
	t.Fatalf("event not found in %#v", events)
}
