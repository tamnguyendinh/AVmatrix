package parser

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/tamnguyendinh/avmatrix-go/internal/scanner"
)

func TestPoolParsesParserLoaderReadinessLanguages(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		language scanner.Language
		source   []byte
	}{
		{
			name:     "JavaScript",
			filePath: "src/app.js",
			language: scanner.JavaScript,
			source:   []byte("function add(a, b) { return a + b; }\n"),
		},
		{
			name:     "TypeScript",
			filePath: "src/app.ts",
			language: scanner.TypeScript,
			source:   []byte("export function add(a: number, b: number): number { return a + b; }\n"),
		},
		{
			name:     "Python",
			filePath: "src/app.py",
			language: scanner.Python,
			source:   []byte("def add(a, b):\n    return a + b\n"),
		},
		{
			name:     "Java",
			filePath: "src/app/Main.java",
			language: scanner.Java,
			source:   []byte("class Main { int add(int a, int b) { return a + b; } }\n"),
		},
		{
			name:     "C",
			filePath: "src/add.c",
			language: scanner.C,
			source:   []byte("int add(int a, int b) { return a + b; }\n"),
		},
		{
			name:     "C++",
			filePath: "src/add.cpp",
			language: scanner.CPlusPlus,
			source:   []byte("int add(int a, int b) { return a + b; }\n"),
		},
		{
			name:     "C#",
			filePath: "src/App.cs",
			language: scanner.CSharp,
			source:   []byte("class Main { int Add(int a, int b) { return a + b; } }\n"),
		},
		{
			name:     "Go",
			filePath: "src/main.go",
			language: scanner.Go,
			source:   []byte("package main\nfunc add(a int, b int) int { return a + b }\n"),
		},
		{
			name:     "Rust",
			filePath: "src/lib.rs",
			language: scanner.Rust,
			source:   []byte("fn add(a: i32, b: i32) -> i32 { a + b }\n"),
		},
		{
			name:     "PHP",
			filePath: "src/app.php",
			language: scanner.PHP,
			source:   []byte("<?php function add($a, $b) { return $a + $b; }\n"),
		},
		{
			name:     "Ruby",
			filePath: "src/app.rb",
			language: scanner.Ruby,
			source:   []byte("def add(a, b)\n  a + b\nend\n"),
		},
		{
			name:     "Swift",
			filePath: "src/App.swift",
			language: scanner.Swift,
			source:   []byte("func add(_ a: Int, _ b: Int) -> Int { return a + b }\n"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			pool := NewPool(nil, PoolOptions{ParseTimeout: time.Second, CountNodes: true})
			defer pool.Close()

			result, err := pool.Parse(context.Background(), Request{
				FilePath: test.filePath,
				Language: test.language,
				Source:   test.source,
			})
			if err != nil {
				t.Fatalf("Parse(%s) failed: %v", test.name, err)
			}
			defer result.Close()
			if result.RootKind == "" || result.HasError || result.NodeCount == 0 || result.Bytes != len(test.source) {
				t.Fatalf("unexpected %s parse result: %#v", test.name, result)
			}
		})
	}
}

func TestPoolDistinguishesTypeScriptAndTSXGrammars(t *testing.T) {
	tests := []struct {
		name        string
		filePath    string
		source      []byte
		wantGrammar string
	}{
		{
			name:        "TypeScript",
			filePath:    "src/utils.ts",
			source:      []byte("export const value: number = 1;\n"),
			wantGrammar: "typescript",
		},
		{
			name:        "TSX",
			filePath:    "src/Component.tsx",
			source:      []byte("export const Component = () => <main>{'ok'}</main>;\n"),
			wantGrammar: "tsx",
		},
		{
			name:        "uppercase TSX extension",
			filePath:    "src/Component.TSX",
			source:      []byte("export const Component = () => <main>{'ok'}</main>;\n"),
			wantGrammar: "tsx",
		},
	}

	pool := NewPool(nil, PoolOptions{ParseTimeout: time.Second})
	defer pool.Close()

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := pool.Parse(context.Background(), Request{
				FilePath: test.filePath,
				Language: scanner.TypeScript,
				Source:   test.source,
			})
			if err != nil {
				t.Fatalf("Parse(%s) failed: %v", test.filePath, err)
			}
			defer result.Close()
			if result.Grammar != test.wantGrammar {
				t.Fatalf("Grammar = %q, want %q", result.Grammar, test.wantGrammar)
			}
			if result.HasError {
				t.Fatalf("Parse(%s) unexpectedly had syntax errors", test.filePath)
			}
		})
	}
}

func TestPoolHandlesEmptyAndBinaryLikeSources(t *testing.T) {
	tests := []struct {
		name       string
		filePath   string
		source     []byte
		allowError bool
	}{
		{
			name:     "empty TypeScript",
			filePath: "empty.ts",
			source:   nil,
		},
		{
			name:       "binary-like TypeScript",
			filePath:   "binary.ts",
			source:     []byte{0xff, 0xfe, 0x00, 0x01, 0x1f, 0xef, 0xbf, 0xbd, 0xff, 0xfe},
			allowError: true,
		},
	}

	pool := NewPool(nil, PoolOptions{ParseTimeout: time.Second, CountNodes: true})
	defer pool.Close()

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := pool.Parse(context.Background(), Request{
				FilePath: test.filePath,
				Language: scanner.TypeScript,
				Source:   test.source,
			})
			if err != nil {
				t.Fatalf("Parse(%s) failed: %v", test.name, err)
			}
			defer result.Close()
			if result.RootKind == "" || result.NodeCount == 0 || result.Bytes != len(test.source) {
				t.Fatalf("unexpected %s parse result: %#v", test.name, result)
			}
			if result.HasError && !test.allowError {
				t.Fatalf("Parse(%s) unexpectedly had syntax errors", test.name)
			}
		})
	}
}

func TestSampleCodeFixturesExistForParserReadiness(t *testing.T) {
	fixtureDir := filepath.Join("..", "..", "avmatrix", "test", "fixtures", "sample-code")
	for _, fixture := range []string{
		"simple.js",
		"simple.ts",
		"simple.tsx",
		"simple.py",
		"simple.go",
		"simple.swift",
		"simple.php",
		"simple.rs",
		"simple.java",
		"simple.c",
		"simple.cpp",
		"simple.cs",
		"simple.dart",
	} {
		t.Run(fixture, func(t *testing.T) {
			content, err := os.ReadFile(filepath.Join(fixtureDir, fixture))
			if err != nil {
				t.Fatalf("ReadFile(%s) failed: %v", fixture, err)
			}
			if len(content) == 0 {
				t.Fatalf("%s is empty", fixture)
			}
		})
	}
}
