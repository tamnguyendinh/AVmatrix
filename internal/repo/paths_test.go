package repo

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPathsMatchStorageContract(t *testing.T) {
	repoPath := t.TempDir()

	paths := Paths(repoPath)
	wantStorage := filepath.Join(absClean(repoPath), ".avmatrix")
	if paths.StoragePath != wantStorage {
		t.Fatalf("StoragePath = %q, want %q", paths.StoragePath, wantStorage)
	}
	if paths.LbugPath != filepath.Join(wantStorage, "lbug") {
		t.Fatalf("LbugPath = %q", paths.LbugPath)
	}
	if paths.GraphPath != filepath.Join(wantStorage, "graph.json") {
		t.Fatalf("GraphPath = %q", paths.GraphPath)
	}
	if paths.MetaPath != filepath.Join(wantStorage, "meta.json") {
		t.Fatalf("MetaPath = %q", paths.MetaPath)
	}
	if paths.AnalyzeLockPath != filepath.Join(wantStorage, "analyze.lock") {
		t.Fatalf("AnalyzeLockPath = %q", paths.AnalyzeLockPath)
	}
	if paths.AnalyzeTempPath != filepath.Join(wantStorage, "analyze.tmp") {
		t.Fatalf("AnalyzeTempPath = %q", paths.AnalyzeTempPath)
	}
}

func TestGlobalDirUsesAVmatrixHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv(HomeEnvName, home)

	if got := GlobalDir(); got != home {
		t.Fatalf("GlobalDir() = %q, want %q", got, home)
	}
}

func TestGlobalRegistryPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv(HomeEnvName, home)

	got := GlobalRegistryPath()
	want := filepath.Join(home, "registry.json")
	if got != want {
		t.Fatalf("GlobalRegistryPath() = %q, want %q", got, want)
	}
}

func TestStoragePathCleansRelativeInput(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get cwd: %v", err)
	}
	got := StoragePath(".")
	want := filepath.Join(cwd, ".avmatrix")
	if got != want {
		t.Fatalf("StoragePath(.) = %q, want %q", got, want)
	}
}
