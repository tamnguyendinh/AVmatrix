package repo

import (
	"path/filepath"
	"testing"
)

func TestSaveLoadMetaRoundTrip(t *testing.T) {
	files := 3
	meta := Meta{
		RepoPath:   "C:/repo",
		LastCommit: "abc123",
		IndexedAt:  "2026-05-08T00:00:00Z",
		Stats:      &Stats{Files: &files},
	}
	storagePath := filepath.Join(t.TempDir(), ".avmatrix")

	if err := SaveMeta(storagePath, meta); err != nil {
		t.Fatalf("SaveMeta() error = %v", err)
	}
	got, err := LoadMeta(storagePath)
	if err != nil {
		t.Fatalf("LoadMeta() error = %v", err)
	}
	if got == nil {
		t.Fatal("LoadMeta() returned nil")
	}
	if got.RepoPath != meta.RepoPath || got.LastCommit != meta.LastCommit || got.IndexedAt != meta.IndexedAt {
		t.Fatalf("LoadMeta() = %#v", got)
	}
	if got.Stats == nil || got.Stats.Files == nil || *got.Stats.Files != files {
		t.Fatalf("LoadMeta().Stats = %#v", got.Stats)
	}
}

func TestLoadMetaMissingReturnsNil(t *testing.T) {
	got, err := LoadMeta(t.TempDir())
	if err != nil {
		t.Fatalf("LoadMeta() error = %v", err)
	}
	if got != nil {
		t.Fatalf("LoadMeta() = %#v, want nil", got)
	}
}

func TestLoadIndexedUsesRepoStoragePaths(t *testing.T) {
	repoPath := t.TempDir()
	meta := Meta{RepoPath: repoPath, LastCommit: "abc123", IndexedAt: "now"}
	if err := SaveMeta(Paths(repoPath).StoragePath, meta); err != nil {
		t.Fatalf("SaveMeta() error = %v", err)
	}

	indexed, err := LoadIndexed(repoPath)
	if err != nil {
		t.Fatalf("LoadIndexed() error = %v", err)
	}
	if indexed == nil {
		t.Fatal("LoadIndexed() returned nil")
	}
	if indexed.StoragePath != Paths(repoPath).StoragePath {
		t.Fatalf("StoragePath = %q", indexed.StoragePath)
	}
	if indexed.LbugPath != Paths(repoPath).LbugPath {
		t.Fatalf("LbugPath = %q", indexed.LbugPath)
	}
	if !HasIndex(repoPath) {
		t.Fatal("HasIndex() = false")
	}
}
