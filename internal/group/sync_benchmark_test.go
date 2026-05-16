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

func BenchmarkSyncSmallFixture(b *testing.B) {
	homeDir := b.TempDir()
	store := repo.NewStore(homeDir)
	backendPath := b.TempDir()
	frontendPath := b.TempDir()
	registerGroupSyncBenchmarkRepo(b, store, backendPath, "backend")
	registerGroupSyncBenchmarkRepo(b, store, frontendPath, "frontend")
	writeGroupSyncBenchmarkProviderGraph(b, backendPath)
	writeGroupSyncBenchmarkConsumerGraph(b, frontendPath)

	groupDir := filepath.Join(homeDir, "groups", "fixture")
	if err := os.MkdirAll(groupDir, 0o755); err != nil {
		b.Fatalf("mkdir group: %v", err)
	}
	writeGroupSyncBenchmarkConfig(b, groupDir)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, err := Sync(homeDir, store, "fixture", SyncOptions{ExactOnly: true, SkipEmbeddings: true})
		if err != nil {
			b.Fatalf("Sync() error = %v", err)
		}
		if len(result.Contracts) != 2 || len(result.CrossLinks) != 1 || len(result.Unmatched) != 0 || len(result.MissingRepos) != 0 {
			b.Fatalf("sync result = %#v", result)
		}
	}
}

func registerGroupSyncBenchmarkRepo(tb testing.TB, store repo.Store, repoPath string, name string) {
	tb.Helper()
	meta := repo.Meta{
		RepoPath:   repoPath,
		IndexedAt:  "2026-05-12T00:00:00Z",
		LastCommit: "abc123",
		Stats:      &repo.Stats{},
	}
	if err := repo.SaveMeta(repo.StoragePath(repoPath), meta); err != nil {
		tb.Fatalf("save meta: %v", err)
	}
	if _, err := store.Register(repoPath, meta, repo.RegisterOptions{Name: name}); err != nil {
		tb.Fatalf("register repo: %v", err)
	}
}

func writeGroupSyncBenchmarkProviderGraph(tb testing.TB, repoPath string) {
	tb.Helper()
	g := graph.New()
	g.AddNode(graph.Node{ID: "Route:/api/users", Label: scopeir.NodeRoute, Properties: graph.NodeProperties{
		"name": "/api/users", "filePath": "src/server.ts",
	}})
	writeGroupSyncBenchmarkGraph(tb, repoPath, g)
}

func writeGroupSyncBenchmarkConsumerGraph(tb testing.TB, repoPath string) {
	tb.Helper()
	g := graph.New()
	g.AddNode(graph.Node{ID: "File:src/app.ts", Label: scopeir.NodeFile, Properties: graph.NodeProperties{
		"name": "app.ts", "filePath": "src/app.ts",
	}})
	g.AddNode(graph.Node{ID: "Route:/api/users", Label: scopeir.NodeRoute, Properties: graph.NodeProperties{
		"name": "/api/users",
	}})
	g.AddRelationship(graph.Relationship{ID: "rel:fetch-users", SourceID: "File:src/app.ts", TargetID: "Route:/api/users", Type: graph.RelFetches, Confidence: 0.9, Reason: "fetch-route"})
	writeGroupSyncBenchmarkGraph(tb, repoPath, g)
}

func writeGroupSyncBenchmarkGraph(tb testing.TB, repoPath string, g *graph.Graph) {
	tb.Helper()
	raw, err := json.MarshalIndent(g, "", "  ")
	if err != nil {
		tb.Fatalf("marshal graph: %v", err)
	}
	graphPath := filepath.Join(repo.StoragePath(repoPath), "graph.json")
	if err := os.MkdirAll(filepath.Dir(graphPath), 0o755); err != nil {
		tb.Fatalf("mkdir graph dir: %v", err)
	}
	if err := os.WriteFile(graphPath, append(raw, '\n'), 0o644); err != nil {
		tb.Fatalf("write graph: %v", err)
	}
}

func writeGroupSyncBenchmarkConfig(tb testing.TB, groupDir string) {
	tb.Helper()
	groupYAML := `version: 1
name: fixture
description: "fixture group"

repos:
  app/backend: backend
  app/frontend: frontend

links: []
`
	if err := os.WriteFile(filepath.Join(groupDir, "group.yaml"), []byte(groupYAML), 0o644); err != nil {
		tb.Fatalf("write group.yaml: %v", err)
	}
}
