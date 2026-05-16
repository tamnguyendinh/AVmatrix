package repo

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestRegisterRepoWritesRegistryEntry(t *testing.T) {
	store := NewStore(t.TempDir())
	repoPath := t.TempDir()
	files := 10
	meta := Meta{
		RepoPath:   repoPath,
		LastCommit: "abc123",
		IndexedAt:  "2026-05-08T00:00:00Z",
		Stats:      &Stats{Files: &files},
	}

	name, err := store.Register(repoPath, meta, RegisterOptions{})
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if name != filepath.Base(repoPath) {
		t.Fatalf("Register() name = %q, want %q", name, filepath.Base(repoPath))
	}

	entries, err := store.ReadRegistry()
	if err != nil {
		t.Fatalf("ReadRegistry() error = %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("registry length = %d", len(entries))
	}
	entry := entries[0]
	if entry.Path != absClean(repoPath) {
		t.Fatalf("entry.Path = %q", entry.Path)
	}
	if entry.StoragePath != Paths(repoPath).StoragePath {
		t.Fatalf("entry.StoragePath = %q", entry.StoragePath)
	}
	if entry.Stats == nil || entry.Stats.Files == nil || *entry.Stats.Files != files {
		t.Fatalf("entry.Stats = %#v", entry.Stats)
	}
}

func TestRegisterRepoExplicitAliasOverwritesSamePath(t *testing.T) {
	store := NewStore(t.TempDir())
	repoPath := t.TempDir()
	meta := Meta{RepoPath: repoPath, LastCommit: "abc123", IndexedAt: "now"}

	if _, err := store.Register(repoPath, meta, RegisterOptions{Name: "before"}); err != nil {
		t.Fatalf("Register(before) error = %v", err)
	}
	name, err := store.Register(repoPath, meta, RegisterOptions{Name: "after"})
	if err != nil {
		t.Fatalf("Register(after) error = %v", err)
	}
	if name != "after" {
		t.Fatalf("name = %q", name)
	}
	entries, err := store.ReadRegistry()
	if err != nil {
		t.Fatalf("ReadRegistry() error = %v", err)
	}
	if len(entries) != 1 || entries[0].Name != "after" {
		t.Fatalf("entries = %#v", entries)
	}
}

func TestRegisterRepoRejectsDuplicateExplicitAlias(t *testing.T) {
	store := NewStore(t.TempDir())
	first := t.TempDir()
	second := t.TempDir()
	meta := Meta{LastCommit: "abc123", IndexedAt: "now"}

	if _, err := store.Register(first, meta, RegisterOptions{Name: "shared"}); err != nil {
		t.Fatalf("Register(first) error = %v", err)
	}
	_, err := store.Register(second, meta, RegisterOptions{Name: "shared"})
	var collision RegistryNameCollisionError
	if !errors.As(err, &collision) {
		t.Fatalf("expected RegistryNameCollisionError, got %v", err)
	}
	if collision.RegistryName != "shared" || !SamePath(collision.RequestedPath, absClean(second)) {
		t.Fatalf("collision = %#v", collision)
	}
}

func TestRegisterRepoAllowsDuplicateAliasWhenRequested(t *testing.T) {
	store := NewStore(t.TempDir())
	first := t.TempDir()
	second := t.TempDir()
	meta := Meta{LastCommit: "abc123", IndexedAt: "now"}

	if _, err := store.Register(first, meta, RegisterOptions{Name: "shared"}); err != nil {
		t.Fatalf("Register(first) error = %v", err)
	}
	if _, err := store.Register(second, meta, RegisterOptions{Name: "shared", AllowDuplicateName: true}); err != nil {
		t.Fatalf("Register(second) error = %v", err)
	}
	entries, err := store.ReadRegistry()
	if err != nil {
		t.Fatalf("ReadRegistry() error = %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("registry length = %d", len(entries))
	}
}

func TestRegisterRepoPreservesCustomAlias(t *testing.T) {
	store := NewStore(t.TempDir())
	repoPath := t.TempDir()
	meta := Meta{LastCommit: "abc123", IndexedAt: "now"}

	if _, err := store.Register(repoPath, meta, RegisterOptions{Name: "custom"}); err != nil {
		t.Fatalf("Register(custom) error = %v", err)
	}
	name, err := store.Register(repoPath, meta, RegisterOptions{})
	if err != nil {
		t.Fatalf("Register(default) error = %v", err)
	}
	if name != "custom" {
		t.Fatalf("name = %q, want custom", name)
	}
}

func TestRegisterRepoUsesInferredNameAndPreservesAlias(t *testing.T) {
	store := NewStore(t.TempDir())
	repoPath := t.TempDir()
	meta := Meta{RepoPath: repoPath, LastCommit: "abc123", IndexedAt: "now"}

	name, err := store.Register(repoPath, meta, RegisterOptions{InferredName: "from-remote"})
	if err != nil {
		t.Fatalf("Register(inferred) error = %v", err)
	}
	if name != "from-remote" {
		t.Fatalf("Register(inferred) name = %q", name)
	}

	name, err = store.Register(repoPath, meta, RegisterOptions{Name: "sticky-alias"})
	if err != nil {
		t.Fatalf("Register(alias) error = %v", err)
	}
	if name != "sticky-alias" {
		t.Fatalf("Register(alias) name = %q", name)
	}

	name, err = store.Register(repoPath, meta, RegisterOptions{InferredName: "from-remote"})
	if err != nil {
		t.Fatalf("Register(reanalyze inferred) error = %v", err)
	}
	if name != "sticky-alias" {
		t.Fatalf("Register(reanalyze inferred) name = %q, want preserved alias", name)
	}
}

func TestRegisterRepoAllowsBasenameCollisionsWithoutExplicitName(t *testing.T) {
	store := NewStore(t.TempDir())
	parentA := t.TempDir()
	parentB := t.TempDir()
	repoA := filepath.Join(parentA, "app")
	repoB := filepath.Join(parentB, "app")
	if err := os.MkdirAll(repoA, 0o755); err != nil {
		t.Fatalf("mkdir repoA: %v", err)
	}
	if err := os.MkdirAll(repoB, 0o755); err != nil {
		t.Fatalf("mkdir repoB: %v", err)
	}
	meta := Meta{LastCommit: "abc123", IndexedAt: "now"}

	if _, err := store.Register(repoA, meta, RegisterOptions{}); err != nil {
		t.Fatalf("Register(repoA) error = %v", err)
	}
	if _, err := store.Register(repoB, meta, RegisterOptions{}); err != nil {
		t.Fatalf("Register(repoB) error = %v", err)
	}

	entries, err := store.ReadRegistry()
	if err != nil {
		t.Fatalf("ReadRegistry() error = %v", err)
	}
	if len(entries) != 2 || entries[0].Name != "app" || entries[1].Name != "app" {
		t.Fatalf("entries = %#v", entries)
	}
}

func TestReadRegistryMissingReturnsEmpty(t *testing.T) {
	store := NewStore(t.TempDir())
	entries, err := store.ReadRegistry()
	if err != nil {
		t.Fatalf("ReadRegistry() error = %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("ReadRegistry() length = %d, want 0", len(entries))
	}
}

func TestReadRegistryRepairsStoragePath(t *testing.T) {
	store := NewStore(t.TempDir())
	repoPath := t.TempDir()
	bad := []byte(`[{"name":"repo","path":"` + filepath.ToSlash(repoPath) + `","storagePath":"old","indexedAt":"now","lastCommit":"abc"}]`)
	if err := os.MkdirAll(store.HomeDir, 0o755); err != nil {
		t.Fatalf("mkdir home: %v", err)
	}
	if err := os.WriteFile(store.RegistryPath(), bad, 0o644); err != nil {
		t.Fatalf("write registry: %v", err)
	}

	entries, err := store.ReadRegistry()
	if err != nil {
		t.Fatalf("ReadRegistry() error = %v", err)
	}
	if entries[0].StoragePath != StoragePath(repoPath) {
		t.Fatalf("StoragePath = %q", entries[0].StoragePath)
	}
}

func TestListRegisteredPrunesMissingMeta(t *testing.T) {
	store := NewStore(t.TempDir())
	validRepo := t.TempDir()
	missingRepo := t.TempDir()
	if err := SaveMeta(Paths(validRepo).StoragePath, Meta{RepoPath: validRepo, LastCommit: "abc", IndexedAt: "now"}); err != nil {
		t.Fatalf("SaveMeta() error = %v", err)
	}
	if err := store.WriteRegistry([]RegistryEntry{
		{Name: "valid", Path: validRepo, StoragePath: Paths(validRepo).StoragePath, IndexedAt: "now", LastCommit: "abc"},
		{Name: "missing", Path: missingRepo, StoragePath: Paths(missingRepo).StoragePath, IndexedAt: "now", LastCommit: "abc"},
	}); err != nil {
		t.Fatalf("WriteRegistry() error = %v", err)
	}

	entries, err := store.ListRegistered(true)
	if err != nil {
		t.Fatalf("ListRegistered() error = %v", err)
	}
	if len(entries) != 1 || entries[0].Name != "valid" {
		t.Fatalf("entries = %#v", entries)
	}
}
