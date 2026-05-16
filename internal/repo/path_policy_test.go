package repo

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveAnalyzePathAcceptsExistingAbsoluteDirectory(t *testing.T) {
	dir := t.TempDir()

	got, err := ResolveAnalyzePath(dir)
	if err != nil {
		t.Fatalf("ResolveAnalyzePath() error = %v", err)
	}
	if !filepath.IsAbs(got) {
		t.Fatalf("expected absolute path, got %q", got)
	}
	if filepath.Clean(got) != got {
		t.Fatalf("expected clean path, got %q", got)
	}
}

func TestResolveAnalyzePathRejectsRemoteURL(t *testing.T) {
	for _, input := range []string{
		"https://github.com/owner/repo",
		"ssh://git@example.com/owner/repo.git",
		"git@github.com:owner/repo.git",
	} {
		_, err := ResolveAnalyzePath(input)
		if err == nil || !strings.Contains(err.Error(), "local filesystem path") {
			t.Fatalf("ResolveAnalyzePath(%q) error = %v", input, err)
		}
	}
}

func TestResolveAnalyzePathRejectsUNCPath(t *testing.T) {
	for _, input := range []string{`\\server\share\repo`, "//server/share/repo"} {
		_, err := ResolveAnalyzePath(input)
		if err == nil || !strings.Contains(err.Error(), "UNC") {
			t.Fatalf("ResolveAnalyzePath(%q) error = %v", input, err)
		}
	}
}

func TestResolveAnalyzePathRejectsRelativePath(t *testing.T) {
	_, err := ResolveAnalyzePath("relative/repo")
	if err == nil || !strings.Contains(err.Error(), "absolute path") {
		t.Fatalf("expected absolute path error, got %v", err)
	}
}

func TestResolveAnalyzePathRejectsFile(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "file.txt")
	if err := os.WriteFile(filePath, []byte("not a directory"), 0o644); err != nil {
		t.Fatalf("write file fixture: %v", err)
	}

	_, err := ResolveAnalyzePath(filePath)
	if err == nil || !strings.Contains(err.Error(), "not a directory") {
		t.Fatalf("expected not directory error, got %v", err)
	}
}
