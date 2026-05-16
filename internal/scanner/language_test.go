package scanner

import "testing"

func TestDetectLanguageMatchesSharedContract(t *testing.T) {
	tests := map[string]Language{
		"src/index.js":         JavaScript,
		"src/index.jsx":        JavaScript,
		"src/index.mjs":        JavaScript,
		"src/index.ts":         TypeScript,
		"src/index.tsx":        TypeScript,
		"src/app.py":           Python,
		"src/Main.java":        Java,
		"src/main.c":           C,
		"src/main.cpp":         CPlusPlus,
		"src/main.cs":          CSharp,
		"src/main.go":          Go,
		"src/lib.rs":           Rust,
		"src/index.php":        PHP,
		"src/main.kt":          Kotlin,
		"src/App.swift":        Swift,
		"src/main.dart":        Dart,
		"src/App.vue":          Vue,
		"src/App.svelte":       Svelte,
		"src/Page.astro":       Astro,
		"src/program.cbl":      Cobol,
		"src/program.copybook": Cobol,
		"jobs/nightly.jcl":     Cobol,
		"jobs/nightly.job":     Cobol,
		"jobs/common.proc":     Cobol,
		"README.md":            Markdown,
		"docs/spec.doc":        Word,
		"docs/spec.docx":       Word,
		"docs/ref.pdf":         PDF,
		"data/book.xls":        Spreadsheet,
		"data/book.xlsx":       Spreadsheet,
		"data/book.xlsm":       Spreadsheet,
		"data/book.xlsb":       Spreadsheet,
		"data/template.xltx":   Spreadsheet,
		"data/sheet.ods":       Spreadsheet,
		"data/table.csv":       Spreadsheet,
		"Rakefile":             Ruby,
	}

	for filePath, want := range tests {
		got, ok := DetectLanguage(filePath)
		if !ok || got != want {
			t.Fatalf("DetectLanguage(%q) = %q, %v; want %q, true", filePath, got, ok, want)
		}
	}
	if _, ok := DetectLanguage("archive.zip"); ok {
		t.Fatal("archive.zip unexpectedly detected as supported language")
	}
}
