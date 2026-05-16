package testutil

import (
	"path/filepath"
	"runtime"
	"testing"
)

func TestAbsPathReturnsCleanAbsolutePath(t *testing.T) {
	got := AbsPath(t, ".", "internal", "..", "internal")
	if !filepath.IsAbs(got) {
		t.Fatalf("expected absolute path, got %q", got)
	}
	if filepath.Clean(got) != got {
		t.Fatalf("expected clean path, got %q", got)
	}
}

func TestCleanPathNormalizesSegments(t *testing.T) {
	got := CleanPath("alpha", "beta", "..", "gamma")
	want := filepath.Join("alpha", "gamma")
	if got != want {
		t.Fatalf("CleanPath() = %q, want %q", got, want)
	}
}

func TestSamePathForTestMatchesWindowsCaseRules(t *testing.T) {
	left := CleanPath("C:\\AVmatrix", "Repo")
	right := CleanPath("c:\\avmatrix", "repo")
	got := SamePathForTest(left, right)

	if runtime.GOOS == "windows" && !got {
		t.Fatal("expected Windows path comparison to be case-insensitive")
	}
	if runtime.GOOS != "windows" && got {
		t.Fatal("expected non-Windows path comparison to stay case-sensitive")
	}
}
