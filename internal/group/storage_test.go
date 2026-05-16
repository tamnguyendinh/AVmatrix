package group

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStoragePathsUseConfiguredHome(t *testing.T) {
	homeDir := t.TempDir()
	if got := GroupsBaseDir(homeDir); got != filepath.Join(homeDir, "groups") {
		t.Fatalf("GroupsBaseDir() = %q", got)
	}
	dir, err := Dir(homeDir, "company")
	if err != nil {
		t.Fatalf("Dir() error = %v", err)
	}
	if dir != filepath.Join(homeDir, "groups", "company") {
		t.Fatalf("Dir() = %q", dir)
	}
}

func TestRegistryRoundTripsAndMissingReturnsNil(t *testing.T) {
	homeDir := t.TempDir()
	registry := ContractRegistry{
		Version:     1,
		GeneratedAt: "2026-03-31T10:00:00Z",
		RepoSnapshots: map[string]RepoSnapshot{
			"app/backend": {IndexedAt: "2026-03-30T21:14:14Z", LastCommit: "5838fb8d"},
		},
		MissingRepos: []string{},
		Contracts:    []StoredContract{},
		CrossLinks:   []CrossLink{},
	}
	if err := WriteRegistry(homeDir, "test-group", registry); err != nil {
		t.Fatalf("WriteRegistry() error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(homeDir, "groups", "test-group", "contracts.json")); err != nil {
		t.Fatalf("contracts.json stat error = %v", err)
	}
	loaded, err := ReadRegistry(homeDir, "test-group")
	if err != nil {
		t.Fatalf("ReadRegistry() error = %v", err)
	}
	if loaded == nil || loaded.Version != 1 || loaded.GeneratedAt != "2026-03-31T10:00:00Z" {
		t.Fatalf("loaded registry = %#v", loaded)
	}
	missing, err := ReadRegistry(homeDir, "missing-group")
	if err != nil {
		t.Fatalf("ReadRegistry(missing) error = %v", err)
	}
	if missing != nil {
		t.Fatalf("ReadRegistry(missing) = %#v, want nil", missing)
	}
}

func TestRegistryPersistsOverwritesAndNormalizesSlices(t *testing.T) {
	homeDir := t.TempDir()
	registry := ContractRegistry{
		Version:     1,
		GeneratedAt: "2026-05-12T01:00:00Z",
		RepoSnapshots: map[string]RepoSnapshot{
			"app/backend":  {IndexedAt: "2026-05-12T00:00:00Z", LastCommit: "abc123"},
			"app/frontend": {IndexedAt: "2026-05-12T00:05:00Z", LastCommit: "def456"},
		},
		MissingRepos: []string{"app/mobile"},
		Contracts: []StoredContract{
			{
				Repo:       "app/backend",
				ContractID: "http::GET::/api/users",
				Type:       "http",
				Role:       "provider",
				SymbolUID:  "Route:/api/users",
				SymbolRef:  SymbolRef{FilePath: "src/server.ts", Name: "/api/users"},
				SymbolName: "/api/users",
				Confidence: 1,
				Meta:       map[string]any{"source": "fixture"},
			},
			{
				Repo:       "app/frontend",
				ContractID: "http::GET::/api/users",
				Type:       "http",
				Role:       "consumer",
				SymbolUID:  "File:src/app.ts",
				SymbolRef:  SymbolRef{FilePath: "src/app.ts", Name: "fetchUsers"},
				SymbolName: "fetchUsers",
				Confidence: 0.95,
				Meta:       map[string]any{"source": "fixture"},
			},
		},
		CrossLinks: []CrossLink{{
			From:       CrossLinkEndpoint{Repo: "app/frontend", SymbolUID: "File:src/app.ts", SymbolRef: SymbolRef{FilePath: "src/app.ts", Name: "fetchUsers"}},
			To:         CrossLinkEndpoint{Repo: "app/backend", SymbolUID: "Route:/api/users", SymbolRef: SymbolRef{FilePath: "src/server.ts", Name: "/api/users"}},
			Type:       "http",
			ContractID: "http::GET::/api/users",
			MatchType:  "exact",
			Confidence: 1,
		}},
	}
	if err := WriteRegistry(homeDir, "fixture", registry); err != nil {
		t.Fatalf("WriteRegistry() error = %v", err)
	}
	loaded, err := ReadRegistry(homeDir, "fixture")
	if err != nil {
		t.Fatalf("ReadRegistry() error = %v", err)
	}
	if loaded == nil || loaded.GeneratedAt != registry.GeneratedAt || loaded.MissingRepos[0] != "app/mobile" {
		t.Fatalf("loaded registry metadata = %#v", loaded)
	}
	if len(loaded.RepoSnapshots) != 2 || loaded.RepoSnapshots["app/backend"].LastCommit != "abc123" {
		t.Fatalf("loaded repo snapshots = %#v", loaded.RepoSnapshots)
	}
	if len(loaded.Contracts) != 2 || loaded.Contracts[1].Repo != "app/frontend" || len(loaded.CrossLinks) != 1 {
		t.Fatalf("loaded contracts/cross-links = %#v", loaded)
	}

	if err := WriteRegistry(homeDir, "fixture", ContractRegistry{
		Version:     1,
		GeneratedAt: "2026-05-12T02:00:00Z",
		Contracts:   []StoredContract{{Repo: "app/new", ContractID: "topic::created", Type: "topic", Role: "provider"}},
	}); err != nil {
		t.Fatalf("WriteRegistry(overwrite) error = %v", err)
	}
	overwritten, err := ReadRegistry(homeDir, "fixture")
	if err != nil {
		t.Fatalf("ReadRegistry(overwrite) error = %v", err)
	}
	if len(overwritten.Contracts) != 1 || overwritten.Contracts[0].Repo != "app/new" {
		t.Fatalf("registry was not overwritten: %#v", overwritten.Contracts)
	}
	if overwritten.RepoSnapshots == nil || overwritten.MissingRepos == nil || overwritten.CrossLinks == nil {
		t.Fatalf("registry slices/maps were not normalized: %#v", overwritten)
	}
}

func TestListReturnsGroupsWithConfig(t *testing.T) {
	homeDir := t.TempDir()
	for _, name := range []string{"company", "personal"} {
		dir, err := Dir(homeDir, name)
		if err != nil {
			t.Fatalf("Dir(%s) error = %v", name, err)
		}
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir group: %v", err)
		}
		if err := os.WriteFile(filepath.Join(dir, "group.yaml"), []byte("version: 1\nname: "+name+"\nrepos:\n  a: b\n"), 0o644); err != nil {
			t.Fatalf("write group.yaml: %v", err)
		}
	}
	ignoredDir := filepath.Join(GroupsBaseDir(homeDir), "no-config")
	if err := os.MkdirAll(ignoredDir, 0o755); err != nil {
		t.Fatalf("mkdir ignored group: %v", err)
	}
	groups, err := List(homeDir)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if strings.Join(groups, ",") != "company,personal" {
		t.Fatalf("List() = %#v", groups)
	}
}

func TestValidateGroupNameThroughPathHelpers(t *testing.T) {
	homeDir := t.TempDir()
	invalid := []string{"../../evil", "foo/bar", "", "-leading-dash", "_leading", "com.example"}
	for _, name := range invalid {
		if _, err := Dir(homeDir, name); err == nil || !strings.Contains(err.Error(), "Invalid group name") {
			t.Fatalf("Dir(%q) error = %v, want invalid group name", name, err)
		}
		if err := WriteRegistry(homeDir, name, ContractRegistry{Version: 1}); err == nil || !strings.Contains(err.Error(), "Invalid group name") {
			t.Fatalf("WriteRegistry(%q) error = %v, want invalid group name", name, err)
		}
	}
	for _, name := range []string{"my-group_01", "A", "123"} {
		if _, err := Dir(homeDir, name); err != nil {
			t.Fatalf("Dir(%q) error = %v", name, err)
		}
	}
}
