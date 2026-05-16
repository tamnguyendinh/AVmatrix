package parser

import (
	"errors"
	"testing"

	"github.com/tamnguyendinh/avmatrix-go/internal/scanner"
)

func TestDefaultRegistryResolvesSupportedLanguages(t *testing.T) {
	registry := DefaultRegistry()
	tests := []struct {
		name        string
		language    scanner.Language
		filePath    string
		wantGrammar string
	}{
		{name: "JavaScript", language: scanner.JavaScript, filePath: "src/app.js", wantGrammar: "javascript"},
		{name: "TypeScript", language: scanner.TypeScript, filePath: "src/app.ts", wantGrammar: "typescript"},
		{name: "TSX", language: scanner.TypeScript, filePath: "src/App.TSX", wantGrammar: "tsx"},
		{name: "Python", language: scanner.Python, filePath: "src/app.py", wantGrammar: "python"},
		{name: "Java", language: scanner.Java, filePath: "src/App.java", wantGrammar: "java"},
		{name: "C", language: scanner.C, filePath: "src/app.c", wantGrammar: "c"},
		{name: "C++", language: scanner.CPlusPlus, filePath: "src/app.cpp", wantGrammar: "cpp"},
		{name: "C#", language: scanner.CSharp, filePath: "src/App.cs", wantGrammar: "csharp"},
		{name: "Go", language: scanner.Go, filePath: "src/app.go", wantGrammar: "go"},
		{name: "Rust", language: scanner.Rust, filePath: "src/lib.rs", wantGrammar: "rust"},
		{name: "PHP", language: scanner.PHP, filePath: "src/app.php", wantGrammar: "php"},
		{name: "Ruby", language: scanner.Ruby, filePath: "src/app.rb", wantGrammar: "ruby"},
		{name: "Swift", language: scanner.Swift, filePath: "src/App.swift", wantGrammar: "swift"},
		{name: "Kotlin", language: scanner.Kotlin, filePath: "src/App.kt", wantGrammar: "kotlin"},
		{name: "Dart", language: scanner.Dart, filePath: "lib/main.dart", wantGrammar: "dart"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			grammar, err := registry.Resolve(test.language, test.filePath)
			if err != nil {
				t.Fatalf("Resolve(%s) failed: %v", test.language, err)
			}
			if grammar.Key != test.wantGrammar || grammar.Language != test.language {
				t.Fatalf("grammar = %#v, want key %q language %q", grammar, test.wantGrammar, test.language)
			}
			if grammar.build == nil || grammar.build() == nil {
				t.Fatalf("grammar %s did not build a tree-sitter language", grammar.Key)
			}
		})
	}
}

func TestDefaultRegistryRejectsUnsupportedLanguages(t *testing.T) {
	registry := DefaultRegistry()
	for _, language := range []scanner.Language{"erlang", "haskell", scanner.Markdown} {
		t.Run(string(language), func(t *testing.T) {
			_, err := registry.Resolve(language, "src/file.txt")
			if !errors.Is(err, ErrUnsupportedLanguage) {
				t.Fatalf("Resolve(%s) error = %v, want ErrUnsupportedLanguage", language, err)
			}
		})
	}
}
