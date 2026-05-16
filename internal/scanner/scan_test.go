package scanner

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWalkRepositoryPathsDiscoversSourceAndSkipsIgnored(t *testing.T) {
	dir := t.TempDir()
	writeScannerFile(t, dir, "src/index.ts", "export const main = () => {}")
	writeScannerFile(t, dir, "src/components/Button.tsx", "export const Button = () => null")
	writeScannerFile(t, dir, "node_modules/pkg/index.js", "module.exports = {}")
	writeScannerFile(t, dir, ".git/HEAD", "ref: refs/heads/main")
	writeScannerFile(t, dir, "src/image.png", "png")
	writeScannerFile(t, dir, ".hidden.ts", "hidden")

	var progress []string
	files, metrics, err := WalkRepositoryPaths(dir, Options{}, func(_ int, _ int, filePath string) {
		progress = append(progress, filePath)
	})
	if err != nil {
		t.Fatalf("WalkRepositoryPaths failed: %v", err)
	}
	paths := filePaths(files)

	for _, want := range []string{"src/components/Button.tsx", "src/index.ts"} {
		if !contains(paths, want) {
			t.Fatalf("missing %q in %#v", want, paths)
		}
	}
	for _, unwanted := range []string{"node_modules/pkg/index.js", ".git/HEAD", "src/image.png", ".hidden.ts"} {
		if contains(paths, unwanted) {
			t.Fatalf("unexpected %q in %#v", unwanted, paths)
		}
	}
	if metrics.Included != 2 || len(progress) != 2 {
		t.Fatalf("unexpected metrics/progress: %#v progress=%#v", metrics, progress)
	}
}

func TestWalkRepositoryPathsIncludesDocumentAndSpreadsheetFiles(t *testing.T) {
	dir := t.TempDir()
	writeScannerFile(t, dir, "docs/spec.doc", "doc")
	writeScannerFile(t, dir, "docs/spec.docx", "docx")
	writeScannerFile(t, dir, "docs/ref.pdf", "pdf")
	writeScannerFile(t, dir, "data/book.xlsx", "xlsx")
	writeScannerFile(t, dir, "data/macro.xlsm", "xlsm")
	writeScannerFile(t, dir, "data/table.csv", "a,b\n1,2\n")

	files, _, err := WalkRepositoryPaths(dir, Options{}, nil)
	if err != nil {
		t.Fatalf("WalkRepositoryPaths failed: %v", err)
	}
	byPath := make(map[string]File)
	for _, file := range files {
		byPath[file.Path] = file
	}
	for _, want := range []struct {
		path string
		lang Language
	}{
		{"docs/spec.doc", Word},
		{"docs/spec.docx", Word},
		{"docs/ref.pdf", PDF},
		{"data/book.xlsx", Spreadsheet},
		{"data/macro.xlsm", Spreadsheet},
		{"data/table.csv", Spreadsheet},
	} {
		got, ok := byPath[want.path]
		if !ok {
			t.Fatalf("missing %s in %#v", want.path, filePaths(files))
		}
		if got.Language != want.lang {
			t.Fatalf("%s language = %s, want %s", want.path, got.Language, want.lang)
		}
	}
}

func TestWalkRepositoryPathsIncludesMainframeFiles(t *testing.T) {
	dir := t.TempDir()
	writeScannerFile(t, dir, "src/program.cbl", "IDENTIFICATION DIVISION.\n")
	writeScannerFile(t, dir, "copy/customer.copybook", "01 CUSTOMER-REC.\n")
	writeScannerFile(t, dir, "jobs/nightly.jcl", "//JOB1 JOB\n")
	writeScannerFile(t, dir, "jobs/common.proc", "//PROC1 PROC\n")

	files, _, err := WalkRepositoryPaths(dir, Options{}, nil)
	if err != nil {
		t.Fatalf("WalkRepositoryPaths failed: %v", err)
	}
	byPath := make(map[string]File)
	for _, file := range files {
		byPath[file.Path] = file
	}
	for _, want := range []string{"src/program.cbl", "copy/customer.copybook", "jobs/nightly.jcl", "jobs/common.proc"} {
		got, ok := byPath[want]
		if !ok {
			t.Fatalf("missing %s in %#v", want, filePaths(files))
		}
		if got.Language != Cobol {
			t.Fatalf("%s language = %s, want %s", want, got.Language, Cobol)
		}
	}
}

func TestWalkRepositoryPathsAppliesGitignoreAvmatrixignoreAndEnv(t *testing.T) {
	dir := t.TempDir()
	writeScannerFile(t, dir, ".gitignore", "data/\n*.log\n")
	writeScannerFile(t, dir, ".avmatrixignore", "vendor/\n")
	writeScannerFile(t, dir, "src/index.ts", "source")
	writeScannerFile(t, dir, "src/App.swift", "class App {}\n")
	writeScannerFile(t, dir, "data/dump.json", "{}")
	writeScannerFile(t, dir, "vendor/lib.js", "var x = 1")
	writeScannerFile(t, dir, "debug.log", "debug")

	files, _, err := WalkRepositoryPaths(dir, Options{}, nil)
	if err != nil {
		t.Fatalf("WalkRepositoryPaths failed: %v", err)
	}
	paths := filePaths(files)
	if !contains(paths, "src/index.ts") {
		t.Fatalf("source file missing: %#v", paths)
	}
	if !contains(paths, "src/App.swift") {
		t.Fatalf("Swift source should be discovered before parser availability checks: %#v", paths)
	}
	for _, unwanted := range []string{"data/dump.json", "vendor/lib.js", "debug.log"} {
		if contains(paths, unwanted) {
			t.Fatalf("unexpected ignored path %q in %#v", unwanted, paths)
		}
	}

	t.Setenv("AVMATRIX_NO_GITIGNORE", "1")
	files, _, err = WalkRepositoryPaths(dir, Options{}, nil)
	if err != nil {
		t.Fatalf("WalkRepositoryPaths with env failed: %v", err)
	}
	paths = filePaths(files)
	if !contains(paths, "data/dump.json") {
		t.Fatalf("gitignored path not restored by AVMATRIX_NO_GITIGNORE: %#v", paths)
	}
	if contains(paths, "vendor/lib.js") {
		t.Fatalf("avmatrixignore path was restored unexpectedly: %#v", paths)
	}
}

func TestWalkRepositoryPathsSkipsLargeFilesAndHashesIncludedFiles(t *testing.T) {
	dir := t.TempDir()
	writeScannerFile(t, dir, "src/a.ts", "a")
	writeScannerFile(t, dir, "src/b.ts", "b")
	large := strings.Repeat("x", int(MaxFileSize)+1)
	writeScannerFile(t, dir, "src/large.ts", large)

	files, metrics, err := WalkRepositoryPaths(dir, Options{}, nil)
	if err != nil {
		t.Fatalf("WalkRepositoryPaths failed: %v", err)
	}
	paths := filePaths(files)
	if contains(paths, "src/large.ts") {
		t.Fatalf("large file was included: %#v", paths)
	}
	if metrics.SkippedLarge != 1 {
		t.Fatalf("SkippedLarge = %d, want 1", metrics.SkippedLarge)
	}
	if len(files) != 2 || files[0].Hash == "" || files[1].Hash == "" || files[0].Hash == files[1].Hash {
		t.Fatalf("unexpected hash result: %#v", files)
	}
}

func TestWalkRepositoryPathsSelectionFilters(t *testing.T) {
	dir := t.TempDir()
	writeScannerFile(t, dir, "src/index.ts", "source")
	writeScannerFile(t, dir, "src/generated/client.ts", "generated")
	writeScannerFile(t, dir, "test/index.ts", "test")

	files, _, err := WalkRepositoryPaths(dir, Options{
		Include: []string{"src/**"},
		Exclude: []string{"src/generated/**"},
	}, nil)
	if err != nil {
		t.Fatalf("WalkRepositoryPaths failed: %v", err)
	}
	paths := filePaths(files)
	if len(paths) != 1 || paths[0] != "src/index.ts" {
		t.Fatalf("selection result = %#v", paths)
	}
}

func TestReadFileContentsSkipsMissingFiles(t *testing.T) {
	dir := t.TempDir()
	writeScannerFile(t, dir, "src/index.ts", "source")

	contents, err := ReadFileContents(dir, []string{"src/index.ts", "missing.ts"})
	if err != nil {
		t.Fatalf("ReadFileContents failed: %v", err)
	}
	if contents["src/index.ts"] != "source" || len(contents) != 1 {
		t.Fatalf("unexpected contents: %#v", contents)
	}
}

func writeScannerFile(t *testing.T, root string, rel string, content string) {
	t.Helper()
	fullPath := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
}

func filePaths(files []File) []string {
	paths := make([]string, 0, len(files))
	for _, file := range files {
		paths = append(paths, file.Path)
	}
	return paths
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
