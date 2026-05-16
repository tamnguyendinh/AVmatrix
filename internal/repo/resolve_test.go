package repo

import (
	"errors"
	"path/filepath"
	"testing"
)

func TestResolveEntryPrefersPathForDuplicateNames(t *testing.T) {
	first := t.TempDir()
	second := t.TempDir()
	entries := []RegistryEntry{
		{Name: "app", Path: absClean(first), StoragePath: Paths(first).StoragePath},
		{Name: "app", Path: absClean(second), StoragePath: Paths(second).StoragePath},
	}

	got, err := ResolveEntry(entries, filepath.Clean(second))
	if err != nil {
		t.Fatalf("ResolveEntry(path) error = %v", err)
	}
	if !SamePath(got.Path, second) {
		t.Fatalf("ResolveEntry(path) = %#v", got)
	}
}

func TestResolveEntryReportsAmbiguousName(t *testing.T) {
	entries := []RegistryEntry{
		{Name: "app", Path: "/repos/one/app"},
		{Name: "app", Path: "/repos/two/app"},
	}

	_, err := ResolveEntry(entries, "app")
	var ambiguous AmbiguousNameError
	if !errors.As(err, &ambiguous) {
		t.Fatalf("expected AmbiguousNameError, got %v", err)
	}
	if len(ambiguous.Matches) != 2 {
		t.Fatalf("ambiguous matches = %#v", ambiguous.Matches)
	}
}
