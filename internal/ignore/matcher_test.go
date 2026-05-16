package ignore

import (
	"os"
	"path/filepath"
	"testing"
)

func TestShouldIgnorePathMatchesBaselineHardcodedRules(t *testing.T) {
	for _, rel := range []string{
		"node_modules/pkg/index.js",
		".git/HEAD",
		".svn/wc.db",
		".hg/store",
		".idea/workspace.xml",
		".vscode/settings.json",
		"project/vendor/lib.js",
		"project/venv/site.py",
		"project/__pycache__/mod.pyc",
		"coverage/report.json",
		"src/generated/client.ts",
		"src/image.png",
		"src/bundle.zip",
		"src/native.dll",
		"src/module.wasm",
		"src/app.min.js",
		"dist/vendor.chunk.js",
		"dist/app.bundle.js",
		"package-lock.json",
		"Cargo.lock",
		".env.production",
		"LICENSE.md",
		"types/foo.d.ts",
		"node_modules\\pkg\\index.js",
		"project\\.git\\HEAD",
	} {
		if !ShouldIgnorePath(rel) {
			t.Fatalf("ShouldIgnorePath(%q) = false", rel)
		}
	}
	if ShouldIgnorePath("src/index.ts") {
		t.Fatal("src/index.ts was unexpectedly ignored")
	}
	for _, rel := range []string{
		"docs/spec.doc",
		"docs/spec.docx",
		"docs/ref.pdf",
		"data/book.xlsx",
		"data/macro.xlsm",
		"data/table.csv",
	} {
		if ShouldIgnorePath(rel) {
			t.Fatalf("ShouldIgnorePath(%q) = true, want document/spreadsheet file included", rel)
		}
	}
}

func TestIsHardcodedIgnoredDirectory(t *testing.T) {
	for _, name := range []string{"node_modules", ".git", "dist", "__pycache__"} {
		if !IsHardcodedIgnoredDirectory(name) {
			t.Fatalf("IsHardcodedIgnoredDirectory(%q) = false", name)
		}
	}
	for _, name := range []string{"src", "lib", "app", "local"} {
		if IsHardcodedIgnoredDirectory(name) {
			t.Fatalf("IsHardcodedIgnoredDirectory(%q) = true", name)
		}
	}
}

func TestMatcherCombinesGitignoreAndAvmatrixignore(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, ".gitignore", "data/\n*.log\n")
	writeFile(t, dir, ".avmatrixignore", "local/\n")

	matcher, err := Load(dir, Options{})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	for _, rel := range []string{"data/file.json", "debug.log", "local/plugin.js"} {
		if !matcher.Ignored(rel, false) {
			t.Fatalf("%q was not ignored", rel)
		}
	}
	if matcher.Ignored("src/index.ts", false) {
		t.Fatal("src/index.ts was unexpectedly ignored")
	}
}

func TestNoGitignoreStillReadsAvmatrixignore(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, ".gitignore", "data/\n")
	writeFile(t, dir, ".avmatrixignore", "local/\n")

	matcher, err := Load(dir, Options{NoGitignore: true})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if matcher.Ignored("data/file.json", false) {
		t.Fatal("gitignore rule applied despite NoGitignore")
	}
	if !matcher.Ignored("local/plugin.js", false) {
		t.Fatal("avmatrixignore rule was not applied")
	}
}

func TestNoGitignoreEnv(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, ".gitignore", "data/\n")
	t.Setenv(NoGitignoreEnv, "1")

	matcher, err := Load(dir, Options{})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if matcher.Ignored("data/file.json", false) {
		t.Fatal("gitignore rule applied despite AVMATRIX_NO_GITIGNORE")
	}
}

func TestMatcherHandlesCommentsBareDirsGlobsAndNegations(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, ".gitignore", "# comment\n\nlocal\n*.log\n")
	writeFile(t, dir, ".avmatrixignore", "*\n!iOS\n!iOS/**\n!backend/\n!backend/living_plan/**\n")

	matcher, err := Load(dir, Options{})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if !matcher.Ignored("local", true) {
		t.Fatal("bare directory pattern did not prune local/")
	}
	if !matcher.Ignored("app.log", false) {
		t.Fatal("file glob pattern did not ignore app.log")
	}
	if matcher.Ignored("iOS", true) {
		t.Fatal("bare negation did not unignore iOS/")
	}
	if matcher.Ignored("iOS/App.swift", false) {
		t.Fatal("whitelisted iOS file was ignored")
	}
	if matcher.Ignored("backend/living_plan/plan.md", false) {
		t.Fatal("nested whitelisted backend/living_plan file was ignored")
	}
	if !matcher.Ignored("scripts/main.py", false) {
		t.Fatal("non-whitelisted file was not ignored by wildcard rule")
	}
	if !matcher.Ignored("src", true) {
		t.Fatal("non-whitelisted directory was not pruned by wildcard rule")
	}
}

func writeFile(t *testing.T, root string, rel string, content string) {
	t.Helper()
	fullPath := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
}
