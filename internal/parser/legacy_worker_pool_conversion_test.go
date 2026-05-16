package parser

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/tamnguyendinh/avmatrix-go/internal/scanner"
)

func TestLegacyWorkerPoolClusterParsesBatchesAndTracksFailures(t *testing.T) {
	pool := NewPool(nil, PoolOptions{MaxParsersPerGrammar: 2, ParseTimeout: time.Second, CountNodes: true})
	defer pool.Close()

	requests := []Request{
		{
			FilePath: "src/validator.ts",
			Language: scanner.TypeScript,
			Source:   []byte("export function validateInput(value: string): boolean { return value.length > 0; }\n"),
		},
		{
			FilePath: "src/handler.ts",
			Language: scanner.TypeScript,
			Source:   []byte("export function handleRequest() { return validateInput('ok'); }\n"),
		},
		{
			FilePath: "src/constants.go",
			Language: scanner.Go,
			Source: []byte(`package main

const (
	Alpha = "alpha"
	Beta = "beta"
)
`),
		},
	}

	for _, request := range requests {
		result, err := pool.Parse(context.Background(), request)
		if err != nil {
			t.Fatalf("Parse(%s) error = %v", request.FilePath, err)
		}
		if result.RootKind == "" || result.NodeCount == 0 || result.Bytes != len(request.Source) {
			t.Fatalf("Parse(%s) returned incomplete result: %#v", request.FilePath, result)
		}
		result.Close()
	}

	_, err := pool.Parse(context.Background(), Request{
		FilePath: "README.md",
		Language: scanner.Markdown,
		Source:   []byte("# docs\n"),
	})
	if !errors.Is(err, ErrUnsupportedLanguage) {
		t.Fatalf("unsupported Parse() error = %v, want ErrUnsupportedLanguage", err)
	}

	metrics := pool.SnapshotMetrics()
	if metrics.Total != 4 || metrics.Succeeded != 3 || metrics.Unsupported != 1 || metrics.Failed != 1 {
		t.Fatalf("unexpected parser pool metrics: %#v", metrics)
	}
	if metrics.CreatedParsers != 2 {
		t.Fatalf("CreatedParsers = %d, want one TypeScript parser and one Go parser", metrics.CreatedParsers)
	}

	pool.Close()
	pool.Close()
}
