package structure

import (
	"testing"

	"github.com/tamnguyendinh/avmatrix-go/internal/graph"
	"github.com/tamnguyendinh/avmatrix-go/internal/scanner"
	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

func TestApplyEmitsFolderNodesAndContainsEdges(t *testing.T) {
	g := graph.New()
	result := Apply(g, []scanner.File{
		{Path: "src/api/handler.ts", Language: scanner.TypeScript},
		{Path: "src/api/model.ts", Language: scanner.TypeScript},
	})

	if result.Metrics.FileNodesSeen != 2 || result.Metrics.FolderNodesAdded != 2 || result.Metrics.ContainsEmitted != 4 {
		t.Fatalf("metrics = %#v", result.Metrics)
	}
	requireNode(t, g, "Folder:src", scopeir.NodeFolder)
	requireNode(t, g, "Folder:src/api", scopeir.NodeFolder)
	requireNode(t, g, "File:src/api/handler.ts", scopeir.NodeFile)
	requireStructureRelationship(t, g, "Folder:src", "Folder:src/api")
	requireStructureRelationship(t, g, "Folder:src/api", "File:src/api/handler.ts")
	requireStructureRelationship(t, g, "Folder:src/api", "File:src/api/model.ts")
}

func TestApplyPreservesExistingFileNodeProperties(t *testing.T) {
	g := graph.New()
	g.AddNode(graph.Node{
		ID:    "File:src/app.ts",
		Label: scopeir.NodeFile,
		Properties: graph.NodeProperties{
			"name":     "app.ts",
			"filePath": "src/app.ts",
			"language": "typescript",
		},
	})

	Apply(g, []scanner.File{{Path: "src/app.ts", Language: scanner.TypeScript}})
	node, ok := g.GetNode("File:src/app.ts")
	if !ok {
		t.Fatal("file node missing")
	}
	if node.Properties["language"] != "typescript" {
		t.Fatalf("existing file metadata was not preserved: %#v", node)
	}
}

func TestApplyHandlesSingleFileEmptyPathsDeepPathsAndDedup(t *testing.T) {
	empty := graph.New()
	emptyResult := Apply(empty, nil)
	if emptyResult.Metrics != (Metrics{}) || len(empty.Nodes) != 0 || len(empty.Relationships) != 0 {
		t.Fatalf("empty apply result = %#v graph=%#v", emptyResult.Metrics, empty)
	}

	g := graph.New()
	result := Apply(g, []scanner.File{
		{Path: "README.md", Language: scanner.Markdown},
		{Path: "src/api/v1/users/handler.ts", Language: scanner.TypeScript},
		{Path: "src/api/v1/users/model.ts", Language: scanner.TypeScript},
	})

	if result.Metrics.FileNodesSeen != 3 {
		t.Fatalf("file nodes seen = %d, want 3", result.Metrics.FileNodesSeen)
	}
	for _, want := range []struct {
		id       string
		label    scopeir.NodeLabel
		filePath string
	}{
		{id: "File:README.md", label: scopeir.NodeFile, filePath: "README.md"},
		{id: "Folder:src", label: scopeir.NodeFolder, filePath: "src"},
		{id: "Folder:src/api", label: scopeir.NodeFolder, filePath: "src/api"},
		{id: "Folder:src/api/v1", label: scopeir.NodeFolder, filePath: "src/api/v1"},
		{id: "Folder:src/api/v1/users", label: scopeir.NodeFolder, filePath: "src/api/v1/users"},
		{id: "File:src/api/v1/users/handler.ts", label: scopeir.NodeFile, filePath: "src/api/v1/users/handler.ts"},
		{id: "File:src/api/v1/users/model.ts", label: scopeir.NodeFile, filePath: "src/api/v1/users/model.ts"},
	} {
		node, ok := g.GetNode(want.id)
		if !ok {
			t.Fatalf("missing node %s", want.id)
		}
		if node.Label != want.label || node.Properties["filePath"] != want.filePath {
			t.Fatalf("node %s = %#v, want label %s filePath %s", want.id, node, want.label, want.filePath)
		}
	}
	requireStructureRelationship(t, g, "Folder:src/api/v1/users", "File:src/api/v1/users/handler.ts")
	requireStructureRelationship(t, g, "Folder:src/api/v1/users", "File:src/api/v1/users/model.ts")
	for _, rel := range g.Relationships {
		if rel.Type == graph.RelContains && rel.Confidence != 1 {
			t.Fatalf("CONTAINS confidence = %f, want 1: %#v", rel.Confidence, rel)
		}
	}
	if result.Metrics.FolderNodesAdded != 4 {
		t.Fatalf("folder nodes added = %d, want 4 after shared folder dedup", result.Metrics.FolderNodesAdded)
	}
}

func requireNode(t *testing.T, g *graph.Graph, id string, label scopeir.NodeLabel) {
	t.Helper()
	node, ok := g.GetNode(id)
	if !ok {
		t.Fatalf("missing node %s", id)
	}
	if node.Label != label {
		t.Fatalf("node %s label = %s, want %s", id, node.Label, label)
	}
}

func requireStructureRelationship(t *testing.T, g *graph.Graph, sourceID string, targetID string) {
	t.Helper()
	for _, rel := range g.Relationships {
		if rel.Type == graph.RelContains && rel.SourceID == sourceID && rel.TargetID == targetID {
			return
		}
	}
	t.Fatalf("missing CONTAINS %s -> %s", sourceID, targetID)
}
