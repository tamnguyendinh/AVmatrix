package testutil

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func AbsPath(t testing.TB, parts ...string) string {
	t.Helper()

	path := filepath.Join(parts...)
	absolute, err := filepath.Abs(path)
	if err != nil {
		t.Fatalf("resolve absolute path for %q: %v", path, err)
	}
	return filepath.Clean(absolute)
}

func CleanPath(parts ...string) string {
	return filepath.Clean(filepath.Join(parts...))
}

func SamePathForTest(left string, right string) bool {
	left = filepath.Clean(left)
	right = filepath.Clean(right)
	if runtime.GOOS == "windows" {
		return strings.EqualFold(left, right)
	}
	return left == right
}
