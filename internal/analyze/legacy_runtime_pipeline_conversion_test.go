package analyze

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/tamnguyendinh/avmatrix-go/internal/parser"
	"github.com/tamnguyendinh/avmatrix-go/internal/scanner"
	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

func TestLegacyWorkerParseClusterEmitsGoSemanticFacts(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "src/constants.go", `package main

const (
	Alpha = "alpha"
	Beta = "beta"
)

type OfflineQueueRepo interface {
	Enqueue() error
}

type OfflineQueueService struct {
	repo OfflineQueueRepo
}

type jsonLogger struct{}

func (l *jsonLogger) Close() error {
	return l.write("close")
}

func (l *jsonLogger) write(msg string) error {
	return nil
}
`)
	writeFile(t, dir, "src/handler.ts", `export function validateInput() { return true; }
export function handleRequest() { return validateInput(); }
`)

	var events []Event
	result, err := parseFiles(context.Background(), dir, []scanner.File{
		{Path: "src/constants.go", Hash: "hash-go", Language: scanner.Go},
		{Path: "src/handler.ts", Hash: "hash-ts", Language: scanner.TypeScript},
	}, Options{
		Parser: parser.PoolOptions{MaxParsersPerGrammar: 2, ParseTimeout: time.Second, CountNodes: true},
		OnEvent: func(event Event) {
			events = append(events, event)
		},
	})
	if err != nil {
		t.Fatalf("parseFiles() error = %v", err)
	}
	if result.Metrics.Parsed != 2 || result.Metrics.Unsupported != 0 || result.Metrics.Failed != 0 {
		t.Fatalf("parse metrics = %#v", result.Metrics)
	}
	if result.Metrics.Parser.Succeeded != 2 || result.Metrics.Parser.CreatedParsers != 2 {
		t.Fatalf("parser metrics = %#v", result.Metrics.Parser)
	}
	if len(events) != 2 || events[0].File != "src/constants.go" || events[1].File != "src/handler.ts" {
		t.Fatalf("parse progress events = %#v", events)
	}

	goIR := result.IRs[0]
	for _, want := range []struct {
		name  string
		label scopeir.NodeLabel
	}{
		{name: "Alpha", label: scopeir.NodeConst},
		{name: "Beta", label: scopeir.NodeConst},
		{name: "jsonLogger.Close", label: scopeir.NodeMethod},
		{name: "jsonLogger.write", label: scopeir.NodeMethod},
		{name: "repo", label: scopeir.NodeProperty},
	} {
		if !containsDefinition(goIR, want.name, want.label) {
			t.Fatalf("Go IR missing %s %s: %#v", want.label, want.name, goIR.Definitions)
		}
	}
	if got := declaredTypeForDefinition(goIR, "repo", scopeir.NodeProperty); got != "OfflineQueueRepo" {
		t.Fatalf("repo declaredType = %q, want OfflineQueueRepo", got)
	}
	if !hasMemberCall(goIR, "write", "l") {
		t.Fatalf("Go IR missing write call in jsonLogger.Close: %#v", goIR.Calls)
	}
}

func TestLegacyParseImplFailuresRecordProgressAndMetrics(t *testing.T) {
	dir := t.TempDir()
	var events []Event
	result, err := parseFiles(context.Background(), dir, []scanner.File{
		{Path: "src/missing.ts", Hash: "hash-missing", Language: scanner.TypeScript},
	}, Options{
		Parser: parser.PoolOptions{ParseTimeout: time.Second},
		OnEvent: func(event Event) {
			events = append(events, event)
		},
	})
	if err == nil || !strings.Contains(err.Error(), "missing.ts") {
		t.Fatalf("parseFiles() error = %v, want missing file path", err)
	}
	if result.Metrics.Failed != 1 || result.Metrics.Parsed != 0 || result.Metrics.Unsupported != 0 {
		t.Fatalf("parse metrics after failure = %#v", result.Metrics)
	}
	if len(events) != 1 || events[0].Kind != EventProgress || events[0].Phase != PhaseParse {
		t.Fatalf("progress events after failure = %#v", events)
	}
}

func TestLegacyPipelineRunnerPhaseWrapperContracts(t *testing.T) {
	var metrics Metrics
	var events []Event
	got, err := runPhase(context.Background(), &metrics, func(event Event) {
		events = append(events, event)
	}, PhaseParse, func() (string, error) {
		return "ok", nil
	})
	if err != nil || got != "ok" {
		t.Fatalf("runPhase success = %q, %v", got, err)
	}
	if len(metrics.Phases) != 1 || metrics.Phases[0].Name != PhaseParse || metrics.Phases[0].Duration < 0 {
		t.Fatalf("phase metrics = %#v", metrics.Phases)
	}
	if len(events) != 2 || events[0].Kind != EventPhaseStart || events[1].Kind != EventPhaseDone {
		t.Fatalf("phase events = %#v", events)
	}

	cause := errors.New("boom")
	_, err = runPhase(context.Background(), &metrics, nil, PhaseResolution, func() (string, error) {
		return "", cause
	})
	if err == nil || !strings.Contains(err.Error(), "resolution phase: boom") || !errors.Is(err, cause) {
		t.Fatalf("runPhase error = %v, want wrapped cause", err)
	}
	if len(metrics.Phases) != 2 || metrics.Phases[1].Name != PhaseResolution {
		t.Fatalf("phase metrics after error = %#v", metrics.Phases)
	}
}

func declaredTypeForDefinition(ir scopeir.ScopeIR, name string, label scopeir.NodeLabel) string {
	for _, def := range ir.Definitions {
		if def.Name == name && def.Label == label {
			return def.DeclaredType
		}
	}
	return ""
}

func hasMemberCall(ir scopeir.ScopeIR, name string, receiver string) bool {
	for _, call := range ir.Calls {
		if call.Name == name && call.CallForm == scopeir.CallMember && call.ExplicitReceiver == receiver {
			return true
		}
	}
	return false
}
