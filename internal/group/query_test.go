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

func TestQuerySearchesIndexedGroupRepos(t *testing.T) {
	homeDir := t.TempDir()
	store := repo.NewStore(homeDir)
	repoPath := t.TempDir()
	meta := repo.Meta{
		RepoPath:   repoPath,
		IndexedAt:  "2026-05-12T00:00:00Z",
		LastCommit: "abc123",
		Stats:      &repo.Stats{},
	}
	if err := repo.SaveMeta(repo.StoragePath(repoPath), meta); err != nil {
		t.Fatalf("save meta: %v", err)
	}
	if _, err := store.Register(repoPath, meta, repo.RegisterOptions{Name: "backend"}); err != nil {
		t.Fatalf("register repo: %v", err)
	}
	writeGroupQueryGraph(t, repoPath)

	groupDir := filepath.Join(homeDir, "groups", "fixture")
	if err := os.MkdirAll(groupDir, 0o755); err != nil {
		t.Fatalf("mkdir group: %v", err)
	}
	writeGroupTestConfig(t, groupDir)

	result, err := Query(homeDir, store, "fixture", "Checkout", 5, "app")
	if err != nil {
		t.Fatalf("Query() error = %v", err)
	}
	if result.Group != "fixture" || result.Query != "Checkout" {
		t.Fatalf("query identity = %#v", result)
	}
	if len(result.Results) != 1 || result.Results[0]["_repo"] != "app/backend" || result.Results[0]["label"] != "CheckoutFlow" {
		t.Fatalf("query results = %#v", result.Results)
	}
	if len(result.PerRepo) != 2 || result.PerRepo[0].Repo != "app/backend" || result.PerRepo[0].Count != 1 || result.PerRepo[1].Count != 0 {
		t.Fatalf("per repo = %#v", result.PerRepo)
	}
}

func writeGroupQueryGraph(t *testing.T, repoPath string) {
	t.Helper()
	g := graph.New()
	g.AddNode(graph.Node{ID: "Function:checkout", Label: scopeir.NodeFunction, Properties: graph.NodeProperties{
		"name": "checkout", "filePath": "src/checkout.ts",
	}})
	g.AddNode(graph.Node{ID: "Process:checkout", Label: scopeir.NodeProcess, Properties: graph.NodeProperties{
		"name": "CheckoutFlow", "label": "CheckoutFlow", "heuristicLabel": "CheckoutFlow", "processType": "cross_community", "stepCount": 1,
	}})
	step := 1
	g.AddRelationship(graph.Relationship{ID: "rel:checkout-process", SourceID: "Function:checkout", TargetID: "Process:checkout", Type: graph.RelStepInProcess, Step: &step})
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
