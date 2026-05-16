package lbugload

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/tamnguyendinh/avmatrix-go/internal/graph"
	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

func TestLegacyScopeAuditMetadataSurvivesCSVExportAndLoad(t *testing.T) {
	g := graph.New()
	g.AddNode(graph.Node{
		ID:    "Function:src/app.ts:run",
		Label: scopeir.NodeFunction,
		Properties: graph.NodeProperties{
			"name": "run", "filePath": "src/app.ts", "startLine": 1, "endLine": 3,
		},
	})
	g.AddNode(graph.Node{
		ID:    "Method:src/app.ts:A.save#0",
		Label: scopeir.NodeMethod,
		Properties: graph.NodeProperties{
			"name": "save", "filePath": "src/app.ts", "startLine": 5, "endLine": 5,
		},
	})
	g.AddRelationship(graph.Relationship{
		ID:               "scope-call",
		SourceID:         "Function:src/app.ts:run",
		TargetID:         "Method:src/app.ts:A.save#0",
		Type:             graph.RelCalls,
		Confidence:       0.95,
		Reason:           "scope-resolution: call | confidence 0.950",
		ResolutionSource: "scope-resolution",
		FileHash:         "sha256:scope",
		Evidence: []graph.Evidence{{
			Kind: "type-binding", Weight: 0.35, Note: "receiver A",
		}},
	})

	export, err := ExportGraphCSVs(g, filepath.Join(t.TempDir(), "csv"))
	if err != nil {
		t.Fatalf("ExportGraphCSVs() error = %v", err)
	}
	rows := readCSV(t, export.RelationshipCSVPath)
	if len(rows) != 2 {
		t.Fatalf("relationship rows = %d, want header plus one row", len(rows))
	}
	relationship := strings.Join(rows[1], "|")
	for _, want := range []string{
		"CALLS",
		"0.95",
		"scope-resolution: call | confidence 0.950",
		"scope-resolution",
		"sha256:scope",
		`"kind":"type-binding"`,
		`"note":"receiver A"`,
	} {
		if !strings.Contains(relationship, want) {
			t.Fatalf("relationship CSV missing %q: %#v", want, rows[1])
		}
	}

	runner := &recordingRunner{}
	result, err := LoadCSVExport(runner, export)
	if err != nil {
		t.Fatalf("LoadCSVExport() error = %v", err)
	}
	if result.RelationshipCopyCount != 1 || result.FallbackInsertCount != 0 || result.SkippedRelationships != 0 {
		t.Fatalf("load result = %#v", result)
	}
	if joined := strings.Join(runner.queries, "\n"); !strings.Contains(joined, "COPY CodeRelation FROM") {
		t.Fatalf("load queries missing relationship COPY:\n%s", joined)
	}
}
