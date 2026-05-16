package group

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/tamnguyendinh/avmatrix-go/internal/graph"
	"github.com/tamnguyendinh/avmatrix-go/internal/repo"
	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

func TestSyncWritesHTTPContractsAndCrossLinks(t *testing.T) {
	homeDir := t.TempDir()
	store := repo.NewStore(homeDir)
	backendPath := t.TempDir()
	frontendPath := t.TempDir()
	registerGroupSyncRepo(t, store, backendPath, "backend")
	registerGroupSyncRepo(t, store, frontendPath, "frontend")
	writeGroupSyncProviderGraph(t, backendPath)
	writeGroupSyncConsumerGraph(t, frontendPath)

	groupDir := filepath.Join(homeDir, "groups", "fixture")
	if err := os.MkdirAll(groupDir, 0o755); err != nil {
		t.Fatalf("mkdir group: %v", err)
	}
	writeGroupTestConfig(t, groupDir)

	result, err := Sync(homeDir, store, "fixture", SyncOptions{})
	if err != nil {
		t.Fatalf("Sync() error = %v", err)
	}
	if len(result.Contracts) != 2 || len(result.CrossLinks) != 1 || len(result.Unmatched) != 0 || len(result.MissingRepos) != 0 {
		t.Fatalf("sync result = %#v", result)
	}
	if result.CrossLinks[0].From.Repo != "app/frontend" || result.CrossLinks[0].To.Repo != "app/backend" || result.CrossLinks[0].ContractID != "http::*::/api/users" {
		t.Fatalf("cross-link = %#v", result.CrossLinks[0])
	}

	registry, err := ReadRegistry(homeDir, "fixture")
	if err != nil {
		t.Fatalf("ReadRegistry() error = %v", err)
	}
	if registry == nil || len(registry.Contracts) != 2 || len(registry.CrossLinks) != 1 || registry.RepoSnapshots["app/backend"].IndexedAt == "" {
		t.Fatalf("written registry = %#v", registry)
	}
}

func TestSyncSkipsHTTPContractsWhenDetectHTTPDisabled(t *testing.T) {
	homeDir := t.TempDir()
	store := repo.NewStore(homeDir)
	backendPath := t.TempDir()
	frontendPath := t.TempDir()
	registerGroupSyncRepo(t, store, backendPath, "backend")
	registerGroupSyncRepo(t, store, frontendPath, "frontend")
	writeGroupSyncProviderGraph(t, backendPath)
	writeGroupSyncConsumerGraph(t, frontendPath)

	groupDir := filepath.Join(homeDir, "groups", "fixture")
	if err := os.MkdirAll(groupDir, 0o755); err != nil {
		t.Fatalf("mkdir group: %v", err)
	}
	groupYAML := `version: 1
name: fixture
description: "fixture group"

repos:
  app/backend: backend
  app/frontend: frontend

detect:
  http: false

links: []
`
	if err := os.WriteFile(filepath.Join(groupDir, "group.yaml"), []byte(groupYAML), 0o644); err != nil {
		t.Fatalf("write group.yaml: %v", err)
	}

	result, err := Sync(homeDir, store, "fixture", SyncOptions{})
	if err != nil {
		t.Fatalf("Sync() error = %v", err)
	}
	if len(result.Contracts) != 0 || len(result.CrossLinks) != 0 || len(result.Unmatched) != 0 {
		t.Fatalf("sync result with http disabled = %#v", result)
	}
}

func TestSyncReportsMissingReposAndHandlesEmptyRepos(t *testing.T) {
	homeDir := t.TempDir()
	store := repo.NewStore(homeDir)
	groupDir := filepath.Join(homeDir, "groups", "fixture")
	if err := os.MkdirAll(groupDir, 0o755); err != nil {
		t.Fatalf("mkdir group: %v", err)
	}
	if err := os.WriteFile(filepath.Join(groupDir, "group.yaml"), []byte(`version: 1
name: fixture
repos:
  app/backend: missing-repo
links: []
`), 0o644); err != nil {
		t.Fatalf("write group.yaml: %v", err)
	}
	result, err := Sync(homeDir, store, "fixture", SyncOptions{})
	if err != nil {
		t.Fatalf("Sync(missing) error = %v", err)
	}
	if len(result.MissingRepos) != 1 || result.MissingRepos[0] != "app/backend" || len(result.Contracts) != 0 {
		t.Fatalf("missing repo sync result = %#v", result)
	}

	emptyDir := filepath.Join(homeDir, "groups", "empty")
	if err := os.MkdirAll(emptyDir, 0o755); err != nil {
		t.Fatalf("mkdir empty group: %v", err)
	}
	if err := os.WriteFile(filepath.Join(emptyDir, "group.yaml"), []byte(`version: 1
name: empty
repos: {}
links: []
`), 0o644); err != nil {
		t.Fatalf("write empty group.yaml: %v", err)
	}
	empty, err := Sync(homeDir, store, "empty", SyncOptions{})
	if err != nil {
		t.Fatalf("Sync(empty) error = %v", err)
	}
	if len(empty.Contracts) != 0 || len(empty.CrossLinks) != 0 || len(empty.MissingRepos) != 0 {
		t.Fatalf("empty repo sync result = %#v", empty)
	}
}

func TestSyncManifestLinksProduceSyntheticCrossLinks(t *testing.T) {
	homeDir := t.TempDir()
	store := repo.NewStore(homeDir)
	groupDir := filepath.Join(homeDir, "groups", "fixture")
	if err := os.MkdirAll(groupDir, 0o755); err != nil {
		t.Fatalf("mkdir group: %v", err)
	}
	if err := os.WriteFile(filepath.Join(groupDir, "group.yaml"), []byte(`version: 1
name: fixture
repos:
  app/consumer: consumer-repo
  app/provider: provider-repo
links:
  - from: app/consumer
    to: app/provider
    type: http
    contract: GET::/api/orders
    role: consumer
`), 0o644); err != nil {
		t.Fatalf("write group.yaml: %v", err)
	}

	result, err := Sync(homeDir, store, "fixture", SyncOptions{})
	if err != nil {
		t.Fatalf("Sync(manifest) error = %v", err)
	}
	if len(result.Contracts) != 2 || len(result.CrossLinks) != 1 {
		t.Fatalf("manifest sync result = %#v", result)
	}
	link := result.CrossLinks[0]
	if link.MatchType != "manifest" || link.ContractID != "http::GET::/api/orders" || link.From.SymbolUID != "manifest::app/consumer::http::GET::/api/orders" || link.To.SymbolUID != "manifest::app/provider::http::GET::/api/orders" {
		t.Fatalf("manifest link = %#v", link)
	}
}

func TestParseFixtureGroupConfig(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "avmatrix", "test", "fixtures", "group", "group.yaml"))
	if err != nil {
		t.Fatalf("read fixture group.yaml: %v", err)
	}
	config, err := ParseConfig(string(raw))
	if err != nil {
		t.Fatalf("ParseConfig(fixture) error = %v", err)
	}
	if config.Name != "test-group" || config.Repos["app/backend"] != "test-backend" || config.Repos["app/frontend"] != "test-frontend" {
		t.Fatalf("fixture config = %#v", config)
	}
}

func registerGroupSyncRepo(t *testing.T, store repo.Store, repoPath string, name string) {
	t.Helper()
	meta := repo.Meta{
		RepoPath:   repoPath,
		IndexedAt:  "2026-05-12T00:00:00Z",
		LastCommit: "abc123",
		Stats:      &repo.Stats{},
	}
	if err := repo.SaveMeta(repo.StoragePath(repoPath), meta); err != nil {
		t.Fatalf("save meta: %v", err)
	}
	if _, err := store.Register(repoPath, meta, repo.RegisterOptions{Name: name}); err != nil {
		t.Fatalf("register repo: %v", err)
	}
}

func writeGroupSyncProviderGraph(t *testing.T, repoPath string) {
	t.Helper()
	g := graph.New()
	g.AddNode(graph.Node{ID: "Route:/api/users", Label: scopeir.NodeRoute, Properties: graph.NodeProperties{
		"name": "/api/users", "filePath": "src/server.ts",
	}})
	writeGroupSyncGraph(t, repoPath, g)
}

func writeGroupSyncConsumerGraph(t *testing.T, repoPath string) {
	t.Helper()
	g := graph.New()
	g.AddNode(graph.Node{ID: "File:src/app.ts", Label: scopeir.NodeFile, Properties: graph.NodeProperties{
		"name": "app.ts", "filePath": "src/app.ts",
	}})
	g.AddNode(graph.Node{ID: "Route:/api/users", Label: scopeir.NodeRoute, Properties: graph.NodeProperties{
		"name": "/api/users",
	}})
	g.AddRelationship(graph.Relationship{ID: "rel:fetch-users", SourceID: "File:src/app.ts", TargetID: "Route:/api/users", Type: graph.RelFetches, Confidence: 0.9, Reason: "fetch-route"})
	writeGroupSyncGraph(t, repoPath, g)
}

func writeGroupSyncGraph(t *testing.T, repoPath string, g *graph.Graph) {
	t.Helper()
	raw, err := json.MarshalIndent(g, "", "  ")
	if err != nil {
		t.Fatalf("marshal graph: %v", err)
	}
	graphPath := filepath.Join(repo.StoragePath(repoPath), "graph.json")
	if err := os.MkdirAll(filepath.Dir(graphPath), 0o755); err != nil {
		t.Fatalf("mkdir graph dir: %v", err)
	}
	if err := os.WriteFile(graphPath, append(raw, '\n'), 0o644); err != nil {
		t.Fatalf("write graph: %v", err)
	}
}
