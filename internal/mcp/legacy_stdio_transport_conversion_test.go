package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"strconv"
	"strings"
	"testing"

	"github.com/tamnguyendinh/avmatrix-go/internal/repo"
)

func TestLegacyCompatibleStdioParsesContentLengthAndNewlineFrames(t *testing.T) {
	contentLengthBody := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","clientInfo":{"name":"codex","version":"0.1"}}}`
	newlineBody := `{"jsonrpc":"2.0","id":2,"method":"initialize","params":{"protocolVersion":"2024-11-05","clientInfo":{"name":"cursor","version":"0.1"}}}`
	input := strings.Join([]string{
		"Content-Length: " + strconv.Itoa(len(contentLengthBody)) + "\r\n\r\n" + contentLengthBody,
		newlineBody + "\n",
	}, "")

	var output bytes.Buffer
	if err := Serve(context.Background(), strings.NewReader(input), &output, Config{Store: repo.NewStore(t.TempDir())}); err != nil {
		t.Fatalf("Serve() error = %v", err)
	}
	responses := readTestFrames(t, output.Bytes())
	if len(responses) != 2 {
		t.Fatalf("response count = %d, want 2; raw=%s", len(responses), output.String())
	}
	for index, response := range responses {
		if response["id"] != float64(index+1) {
			t.Fatalf("response[%d] id = %#v", index, response["id"])
		}
		result, ok := response["result"].(map[string]any)
		if !ok || result["protocolVersion"] != "2024-11-05" {
			t.Fatalf("response[%d] result = %#v", index, response["result"])
		}
	}
	if !strings.HasPrefix(output.String(), "Content-Length: ") {
		t.Fatalf("Serve() did not write Content-Length framing: %q", output.String())
	}
}

func TestLegacyCompatibleStdioReadMessageRejectsMalformedFrames(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{name: "missing length value", raw: "Content-Length:\r\n\r\n{}", want: "invalid syntax"},
		{name: "missing header", raw: "X-Test: value\r\n\r\n{}", want: "missing Content-Length header"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := readMessage(bufio.NewReader(strings.NewReader(tt.raw)))
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("readMessage() error = %v, want containing %q", err, tt.want)
			}
		})
	}
}

func TestLegacyCompatibleStdioNewlineOutputCanBeReadByContentLengthClient(t *testing.T) {
	input := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05"}}` + "\n"
	var output bytes.Buffer
	if err := Serve(context.Background(), strings.NewReader(input), &output, Config{Store: repo.NewStore(t.TempDir())}); err != nil {
		t.Fatalf("Serve() error = %v", err)
	}
	frame, _, err := readMessage(bufio.NewReader(bytes.NewReader(output.Bytes())))
	if err != nil {
		t.Fatalf("read response frame: %v\nraw=%s", err, output.String())
	}
	var response map[string]any
	if err := json.Unmarshal(frame, &response); err != nil {
		t.Fatalf("unmarshal response: %v; frame=%s", err, frame)
	}
	if response["id"] != float64(1) {
		t.Fatalf("response id = %#v", response["id"])
	}
	if _, _, err := readMessage(bufio.NewReader(bytes.NewReader(nil))); err != io.EOF {
		t.Fatalf("empty reader error = %v, want EOF", err)
	}
}

func TestLegacyCompatibleStdioRejectsOversizedFrames(t *testing.T) {
	_, _, err := readMessage(bufio.NewReader(strings.NewReader("Content-Length: 20971520\r\n\r\n{}")))
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "exceeds maximum") {
		t.Fatalf("oversized content-length error = %v, want maximum-size error", err)
	}

	_, _, err = readMessage(bufio.NewReader(strings.NewReader("{" + strings.Repeat("a", maxStdioMessageBytes) + "}\n")))
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "maximum size") {
		t.Fatalf("oversized newline error = %v, want maximum-size error", err)
	}
}

func TestLegacyCompatibleStdioWritesNewlineAfterNewlineInput(t *testing.T) {
	input := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05"}}` + "\n"
	var output bytes.Buffer
	if err := Serve(context.Background(), strings.NewReader(input), &output, Config{Store: repo.NewStore(t.TempDir())}); err != nil {
		t.Fatalf("Serve() error = %v", err)
	}
	raw := output.String()
	if strings.HasPrefix(raw, "Content-Length:") {
		t.Fatalf("newline input got content-length response: %q", raw)
	}
	if !strings.HasSuffix(raw, "\n") {
		t.Fatalf("newline response missing newline terminator: %q", raw)
	}
	var response map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(output.Bytes()), &response); err != nil {
		t.Fatalf("unmarshal newline response: %v; raw=%s", err, raw)
	}
	if response["id"] != float64(1) {
		t.Fatalf("response id = %#v", response["id"])
	}
}
