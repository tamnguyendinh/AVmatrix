package logging

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
)

func TestNewTextLoggerWritesStructuredRecord(t *testing.T) {
	var out bytes.Buffer
	logger := NewTextLogger(&out, slog.LevelInfo)

	logger.InfoContext(context.Background(), "phase ready", "phase", 2)

	got := out.String()
	for _, want := range []string{"level=INFO", "msg=\"phase ready\"", "phase=2"} {
		if !strings.Contains(got, want) {
			t.Fatalf("log output missing %q: %q", want, got)
		}
	}
}

func TestNewTextLoggerAcceptsNilWriter(t *testing.T) {
	logger := NewTextLogger(nil, slog.LevelInfo)
	if logger == nil {
		t.Fatal("expected logger")
	}
	logger.Info("discarded")
}
