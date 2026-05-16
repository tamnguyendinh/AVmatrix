package scanner

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWalkRepositoryPathsReportsSizesAndProgress(t *testing.T) {
	dir := t.TempDir()
	writeScannerFile(t, dir, "src/index.ts", "export const main = () => {}")
	writeScannerFile(t, dir, "src/components/Button.tsx", "export const Button = () => null")

	var calls []struct {
		current int
		total   int
		path    string
	}
	files, metrics, err := WalkRepositoryPaths(dir, Options{}, func(current int, total int, filePath string) {
		calls = append(calls, struct {
			current int
			total   int
			path    string
		}{current: current, total: total, path: filePath})
	})
	if err != nil {
		t.Fatalf("WalkRepositoryPaths failed: %v", err)
	}
	if metrics.Included != 2 || len(files) != 2 {
		t.Fatalf("unexpected files/metrics: files=%#v metrics=%#v", files, metrics)
	}
	for _, file := range files {
		if file.Size <= 0 {
			t.Fatalf("%s Size = %d, want positive", file.Path, file.Size)
		}
		if file.Hash == "" {
			t.Fatalf("%s missing hash", file.Path)
		}
	}
	if len(calls) != 2 {
		t.Fatalf("progress calls = %d, want 2", len(calls))
	}
	for _, call := range calls {
		if call.total != 2 || call.current < 1 || call.current > 2 || call.path == "" {
			t.Fatalf("unexpected progress call: %#v", call)
		}
	}
}

func TestWalkRepositoryPathsReturnsEmptyForEmptyAndIgnoredOnlyDirectories(t *testing.T) {
	t.Run("empty directory", func(t *testing.T) {
		dir := t.TempDir()
		files, metrics, err := WalkRepositoryPaths(dir, Options{}, nil)
		if err != nil {
			t.Fatalf("WalkRepositoryPaths failed: %v", err)
		}
		if len(files) != 0 || metrics.Included != 0 || metrics.Visited != 0 {
			t.Fatalf("unexpected empty directory result: files=%#v metrics=%#v", files, metrics)
		}
	})

	t.Run("ignored-only directory", func(t *testing.T) {
		dir := t.TempDir()
		writeScannerFile(t, dir, ".git/HEAD", "ref: refs/heads/main")
		writeScannerFile(t, dir, "node_modules/pkg/index.js", "module.exports = {}")
		writeScannerFile(t, dir, "fixtures/sample.ts", "export const fixture = true")
		writeScannerFile(t, dir, ".env", "TOKEN=placeholder")
		writeScannerFile(t, dir, "src/image.png", "png")

		files, metrics, err := WalkRepositoryPaths(dir, Options{}, nil)
		if err != nil {
			t.Fatalf("WalkRepositoryPaths failed: %v", err)
		}
		if len(files) != 0 || metrics.Included != 0 {
			t.Fatalf("unexpected ignored-only result: files=%#v metrics=%#v", files, metrics)
		}
	})
}

func TestWalkRepositoryPathsReturnsEmptyForMissingDirectory(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing")
	files, metrics, err := WalkRepositoryPaths(missing, Options{}, nil)
	if err != nil {
		t.Fatalf("WalkRepositoryPaths(%q) failed: %v", missing, err)
	}
	if len(files) != 0 || metrics.SkippedErrored != 1 {
		t.Fatalf("WalkRepositoryPaths(%q) returned files=%#v metrics=%#v, want empty files with skipped error", missing, files, metrics)
	}
}

func TestReadFileContentsHandlesEmptyMissingAndBinaryInputs(t *testing.T) {
	dir := t.TempDir()
	writeScannerFile(t, dir, "src/index.ts", "export const main = () => {}")
	binary := []byte{0x89, 0x50, 0x4e, 0x47, 0x00, 0xff, 0xfe}
	binaryPath := filepath.Join(dir, "src", "image.png")
	if err := os.WriteFile(binaryPath, binary, 0o644); err != nil {
		t.Fatalf("write binary file: %v", err)
	}

	contents, err := ReadFileContents(dir, nil)
	if err != nil {
		t.Fatalf("ReadFileContents(empty) failed: %v", err)
	}
	if len(contents) != 0 {
		t.Fatalf("empty path list contents = %#v, want empty", contents)
	}

	contents, err = ReadFileContents(dir, []string{"missing.ts", "also/missing.ts"})
	if err != nil {
		t.Fatalf("ReadFileContents(missing) failed: %v", err)
	}
	if len(contents) != 0 {
		t.Fatalf("missing files contents = %#v, want empty", contents)
	}

	contents, err = ReadFileContents(dir, []string{"src/index.ts", "src/image.png"})
	if err != nil {
		t.Fatalf("ReadFileContents(binary) failed: %v", err)
	}
	if contents["src/index.ts"] == "" {
		t.Fatalf("source content missing: %#v", contents)
	}
	if got, ok := contents["src/image.png"]; !ok || len(got) != len(binary) {
		t.Fatalf("binary content length = %d, ok=%v; want %d", len(got), ok, len(binary))
	}
}
