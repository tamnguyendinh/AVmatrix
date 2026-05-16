package parser

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/tamnguyendinh/avmatrix-go/internal/scanner"
)

func TestPoolParsesJavaScriptAndTypeScript(t *testing.T) {
	pool := NewPool(nil, PoolOptions{MaxParsersPerGrammar: 1, ParseTimeout: time.Second, CountNodes: true})
	defer pool.Close()

	tests := []Request{
		{
			FilePath: "src/app.js",
			Language: scanner.JavaScript,
			Source:   []byte("function add(a, b) { return a + b; }\n"),
		},
		{
			FilePath: "src/app.ts",
			Language: scanner.TypeScript,
			Source:   []byte("export function add(a: number, b: number): number { return a + b; }\n"),
		},
		{
			FilePath: "src/App.tsx",
			Language: scanner.TypeScript,
			Source:   []byte("export const App = () => <main>{'ok'}</main>;\n"),
		},
	}

	for _, test := range tests {
		result, err := pool.Parse(context.Background(), test)
		if err != nil {
			t.Fatalf("Parse(%s) failed: %v", test.FilePath, err)
		}
		defer result.Close()
		if result.RootKind != "program" {
			t.Fatalf("Parse(%s) root = %q, want program", test.FilePath, result.RootKind)
		}
		if result.HasError {
			t.Fatalf("Parse(%s) unexpectedly had syntax errors", test.FilePath)
		}
		if result.NodeCount == 0 || result.Bytes != len(test.Source) {
			t.Fatalf("Parse(%s) returned incomplete metrics: %#v", test.FilePath, result)
		}
	}

	metrics := pool.SnapshotMetrics()
	if metrics.Total != len(tests) || metrics.Succeeded != len(tests) || metrics.CreatedParsers != 3 {
		t.Fatalf("unexpected metrics: %#v", metrics)
	}
}

func TestPoolReusesParserPerGrammarAndKeepsTSXSeparate(t *testing.T) {
	pool := NewPool(nil, PoolOptions{MaxParsersPerGrammar: 1, ParseTimeout: time.Second})
	defer pool.Close()

	tests := []Request{
		{
			FilePath: "src/first.ts",
			Language: scanner.TypeScript,
			Source:   []byte("export const first: number = 1;\n"),
		},
		{
			FilePath: "src/app.js",
			Language: scanner.JavaScript,
			Source:   []byte("export const value = 1;\n"),
		},
		{
			FilePath: "src/second.ts",
			Language: scanner.TypeScript,
			Source:   []byte("export const second: number = 2;\n"),
		},
		{
			FilePath: "src/App.tsx",
			Language: scanner.TypeScript,
			Source:   []byte("export const App = () => <main>{'ok'}</main>;\n"),
		},
	}

	for _, test := range tests {
		result, err := pool.Parse(context.Background(), test)
		if err != nil {
			t.Fatalf("Parse(%s) failed: %v", test.FilePath, err)
		}
		if result.HasError {
			t.Fatalf("Parse(%s) unexpectedly had syntax errors", test.FilePath)
		}
		result.Close()
	}

	metrics := pool.SnapshotMetrics()
	if metrics.Total != len(tests) || metrics.Succeeded != len(tests) {
		t.Fatalf("unexpected parse metrics: %#v", metrics)
	}
	if metrics.CreatedParsers != 3 {
		t.Fatalf("CreatedParsers = %d, want 3 for javascript, typescript, and tsx grammars", metrics.CreatedParsers)
	}
}

func TestPoolSkipsNodeCountByDefault(t *testing.T) {
	pool := NewPool(nil, PoolOptions{ParseTimeout: time.Second})
	defer pool.Close()

	result, err := pool.Parse(context.Background(), Request{
		FilePath: "src/app.ts",
		Language: scanner.TypeScript,
		Source:   []byte("export function add(a: number, b: number): number { return a + b; }\n"),
	})
	if err != nil {
		t.Fatalf("Parse TypeScript failed: %v", err)
	}
	defer result.Close()
	if result.NodeCount != 0 {
		t.Fatalf("NodeCount = %d, want 0 when CountNodes is disabled", result.NodeCount)
	}
}

func BenchmarkPoolParseNodeCount(b *testing.B) {
	source := []byte(`
export class Service {
  private cache = new Map<string, number>();

  save(input: string): number {
    const next = input
      .split("")
      .map((value, index) => value.charCodeAt(0) + index)
      .reduce((sum, value) => sum + value, 0);
    this.cache.set(input, next);
    return next;
  }
}
`)
	benchmarks := []struct {
		name       string
		countNodes bool
	}{
		{name: "disabled", countNodes: false},
		{name: "enabled", countNodes: true},
	}
	for _, benchmark := range benchmarks {
		b.Run(benchmark.name, func(b *testing.B) {
			pool := NewPool(nil, PoolOptions{MaxParsersPerGrammar: 1, ParseTimeout: time.Second, CountNodes: benchmark.countNodes})
			defer pool.Close()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				result, err := pool.Parse(context.Background(), Request{
					FilePath: "src/service.ts",
					Language: scanner.TypeScript,
					Source:   source,
				})
				if err != nil {
					b.Fatalf("Parse TypeScript failed: %v", err)
				}
				result.Close()
			}
		})
	}
}

func TestPoolParsesGoFixture(t *testing.T) {
	pool := NewPool(nil, PoolOptions{ParseTimeout: time.Second})
	defer pool.Close()

	result, err := pool.Parse(context.Background(), Request{
		FilePath: "main.go",
		Language: scanner.Go,
		Source:   []byte("package main\nfunc add(a int, b int) int { return a + b }\n"),
	})
	if err != nil {
		t.Fatalf("Parse Go failed: %v", err)
	}
	defer result.Close()
	if result.RootKind != "source_file" || result.HasError {
		t.Fatalf("unexpected Go parse result: %#v", result)
	}
}

func TestPoolParsesJavaFixture(t *testing.T) {
	pool := NewPool(nil, PoolOptions{ParseTimeout: time.Second})
	defer pool.Close()

	result, err := pool.Parse(context.Background(), Request{
		FilePath: "src/app/Service.java",
		Language: scanner.Java,
		Source:   []byte("package app; public class Service { public boolean save() { return true; } }\n"),
	})
	if err != nil {
		t.Fatalf("Parse Java failed: %v", err)
	}
	defer result.Close()
	if result.RootKind != "program" || result.HasError {
		t.Fatalf("unexpected Java parse result: %#v", result)
	}
}

func TestPoolParsesKotlinFixture(t *testing.T) {
	pool := NewPool(nil, PoolOptions{ParseTimeout: time.Second})
	defer pool.Close()

	result, err := pool.Parse(context.Background(), Request{
		FilePath: "src/app/Service.kt",
		Language: scanner.Kotlin,
		Source: []byte(`package app

class Service {
    fun save(): Boolean {
        return true
    }
}
`),
	})
	if err != nil {
		t.Fatalf("Parse Kotlin failed: %v", err)
	}
	defer result.Close()
	if result.RootKind != "source_file" || result.HasError {
		t.Fatalf("unexpected Kotlin parse result: %#v", result)
	}
}

func TestPoolParsesCFixture(t *testing.T) {
	pool := NewPool(nil, PoolOptions{ParseTimeout: time.Second})
	defer pool.Close()

	result, err := pool.Parse(context.Background(), Request{
		FilePath: "src/service.c",
		Language: scanner.C,
		Source:   []byte("int helper(const char *value) { return 1; }\n"),
	})
	if err != nil {
		t.Fatalf("Parse C failed: %v", err)
	}
	defer result.Close()
	if result.RootKind != "translation_unit" || result.HasError {
		t.Fatalf("unexpected C parse result: %#v", result)
	}
}

func TestPoolParsesCSharpFixture(t *testing.T) {
	pool := NewPool(nil, PoolOptions{ParseTimeout: time.Second})
	defer pool.Close()

	result, err := pool.Parse(context.Background(), Request{
		FilePath: "src/Service.cs",
		Language: scanner.CSharp,
		Source:   []byte("namespace App; public class Service { public bool Save() { return true; } }\n"),
	})
	if err != nil {
		t.Fatalf("Parse C# failed: %v", err)
	}
	defer result.Close()
	if result.RootKind != "compilation_unit" || result.HasError {
		t.Fatalf("unexpected C# parse result: %#v", result)
	}
}

func TestPoolParsesCPPFixture(t *testing.T) {
	pool := NewPool(nil, PoolOptions{ParseTimeout: time.Second})
	defer pool.Close()

	result, err := pool.Parse(context.Background(), Request{
		FilePath: "src/service.cpp",
		Language: scanner.CPlusPlus,
		Source:   []byte("namespace app { class Service { public: bool save() { return true; } }; }\n"),
	})
	if err != nil {
		t.Fatalf("Parse C++ failed: %v", err)
	}
	defer result.Close()
	if result.RootKind != "translation_unit" || result.HasError {
		t.Fatalf("unexpected C++ parse result: %#v", result)
	}
}

func TestPoolParsesRustFixture(t *testing.T) {
	pool := NewPool(nil, PoolOptions{ParseTimeout: time.Second})
	defer pool.Close()

	result, err := pool.Parse(context.Background(), Request{
		FilePath: "src/service.rs",
		Language: scanner.Rust,
		Source:   []byte("mod app { pub struct Service; impl Service { pub fn save(&self) -> bool { true } } }\n"),
	})
	if err != nil {
		t.Fatalf("Parse Rust failed: %v", err)
	}
	defer result.Close()
	if result.RootKind != "source_file" || result.HasError {
		t.Fatalf("unexpected Rust parse result: %#v", result)
	}
}

func TestPoolParsesPHPFixture(t *testing.T) {
	pool := NewPool(nil, PoolOptions{ParseTimeout: time.Second})
	defer pool.Close()

	result, err := pool.Parse(context.Background(), Request{
		FilePath: "src/Service.php",
		Language: scanner.PHP,
		Source:   []byte("<?php namespace App; class Service { public function save(): bool { return true; } }\n"),
	})
	if err != nil {
		t.Fatalf("Parse PHP failed: %v", err)
	}
	defer result.Close()
	if result.RootKind != "program" || result.HasError {
		t.Fatalf("unexpected PHP parse result: %#v", result)
	}
}

func TestPoolParsesDartFixture(t *testing.T) {
	pool := NewPool(nil, PoolOptions{ParseTimeout: time.Second})
	defer pool.Close()

	result, err := pool.Parse(context.Background(), Request{
		FilePath: "lib/service.dart",
		Language: scanner.Dart,
		Source:   []byte("class Service { bool save() { return true; } }\n"),
	})
	if err != nil {
		t.Fatalf("Parse Dart failed: %v", err)
	}
	defer result.Close()
	if result.RootKind != "program" || result.HasError {
		t.Fatalf("unexpected Dart parse result: %#v", result)
	}
}

func TestPoolReportsSyntaxErrorsWithoutFailing(t *testing.T) {
	pool := NewPool(nil, PoolOptions{ParseTimeout: time.Second})
	defer pool.Close()

	result, err := pool.Parse(context.Background(), Request{
		FilePath: "broken.ts",
		Language: scanner.TypeScript,
		Source:   []byte("export function broken( {"),
	})
	if err != nil {
		t.Fatalf("syntax error should not fail parse: %v", err)
	}
	defer result.Close()
	if !result.HasError {
		t.Fatal("broken TypeScript did not report HasError")
	}
}

func TestPoolRejectsUnsupportedLanguage(t *testing.T) {
	pool := NewPool(nil, PoolOptions{ParseTimeout: time.Second})
	defer pool.Close()

	_, err := pool.Parse(context.Background(), Request{
		FilePath: "README.md",
		Language: scanner.Markdown,
		Source:   []byte("# not wired yet\n"),
	})
	if !errors.Is(err, ErrUnsupportedLanguage) {
		t.Fatalf("error = %v, want ErrUnsupportedLanguage", err)
	}
	metrics := pool.SnapshotMetrics()
	if metrics.Unsupported != 1 || metrics.Failed != 1 {
		t.Fatalf("unsupported metrics not recorded: %#v", metrics)
	}
}

func TestPoolHonorsCanceledContext(t *testing.T) {
	pool := NewPool(nil, PoolOptions{ParseTimeout: time.Second})
	defer pool.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := pool.Parse(ctx, Request{
		FilePath: "src/app.js",
		Language: scanner.JavaScript,
		Source:   []byte("const value = 1;\n"),
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
}

func TestPoolRecordsExpiredDeadlineAsTimeout(t *testing.T) {
	pool := NewPool(nil, PoolOptions{ParseTimeout: time.Second})
	defer pool.Close()

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Nanosecond))
	defer cancel()

	_, err := pool.Parse(ctx, Request{
		FilePath: "src/app.js",
		Language: scanner.JavaScript,
		Source:   []byte("const value = 1;\n"),
	})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error = %v, want context.DeadlineExceeded", err)
	}
	metrics := pool.SnapshotMetrics()
	if metrics.TimedOut != 1 || metrics.Failed != 1 {
		t.Fatalf("timeout metrics not recorded: %#v", metrics)
	}
}
