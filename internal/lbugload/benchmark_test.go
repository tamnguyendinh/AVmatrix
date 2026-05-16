package lbugload

import (
	"path/filepath"
	"strconv"
	"testing"

	"github.com/tamnguyendinh/avmatrix-go/internal/graph"
	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

func BenchmarkExportGraphCSVs(b *testing.B) {
	g := benchmarkGraph(250, false)
	baseDir := b.TempDir()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := ExportGraphCSVs(g, filepath.Join(baseDir, strconv.Itoa(i))); err != nil {
			b.Fatalf("ExportGraphCSVs() error = %v", err)
		}
	}
}

func BenchmarkLoadCSVExportCopyPathNoop(b *testing.B) {
	export, err := ExportGraphCSVs(benchmarkGraph(250, false), filepath.Join(b.TempDir(), "csv"))
	if err != nil {
		b.Fatalf("ExportGraphCSVs() error = %v", err)
	}
	runner := benchmarkNoopRunner{}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := LoadCSVExport(runner, export); err != nil {
			b.Fatalf("LoadCSVExport() error = %v", err)
		}
	}
}

func BenchmarkDiagnosticFallbackPathNoop(b *testing.B) {
	export, err := ExportGraphCSVs(benchmarkGraph(250, true), filepath.Join(b.TempDir(), "csv"))
	if err != nil {
		b.Fatalf("ExportGraphCSVs() error = %v", err)
	}
	runner := benchmarkNoopRunner{}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := LoadCSVExportWithOptions(runner, export, LoadOptions{AllowRelationshipFallback: true}); err != nil {
			b.Fatalf("LoadCSVExport() error = %v", err)
		}
	}
}

type benchmarkNoopRunner struct{}

func (benchmarkNoopRunner) Query(string) error {
	return nil
}

func benchmarkGraph(count int, includeFallback bool) *graph.Graph {
	g := graph.New()
	for i := 0; i < count; i++ {
		fileID := "File:src/file" + strconv.Itoa(i) + ".ts"
		functionID := "Function:fn" + strconv.Itoa(i)
		g.AddNode(graph.Node{ID: fileID, Label: scopeir.NodeFile, Properties: graph.NodeProperties{
			"name": "file" + strconv.Itoa(i) + ".ts", "filePath": "src/file" + strconv.Itoa(i) + ".ts", "content": "export function fn() {}",
		}})
		g.AddNode(graph.Node{ID: functionID, Label: scopeir.NodeFunction, Properties: graph.NodeProperties{
			"name": "fn" + strconv.Itoa(i), "filePath": "src/file" + strconv.Itoa(i) + ".ts", "startLine": i + 1, "endLine": i + 2, "isExported": true, "content": "function fn() {}",
		}})
		g.AddRelationship(graph.Relationship{
			ID:               "rel:file-function:" + strconv.Itoa(i),
			SourceID:         fileID,
			TargetID:         functionID,
			Type:             graph.RelDefines,
			Confidence:       1,
			Reason:           "benchmark copy path",
			ResolutionSource: "benchmark",
			FileHash:         "hash-" + strconv.Itoa(i),
		})
		if includeFallback {
			communityID := "comm_" + strconv.Itoa(i)
			g.AddNode(graph.Node{ID: communityID, Label: scopeir.NodeCommunity, Properties: graph.NodeProperties{
				"name": "community " + strconv.Itoa(i),
			}})
			g.AddRelationship(graph.Relationship{
				ID:               "rel:file-community:" + strconv.Itoa(i),
				SourceID:         fileID,
				TargetID:         communityID,
				Type:             graph.RelMemberOf,
				Confidence:       1,
				Reason:           "benchmark fallback path",
				ResolutionSource: "benchmark",
				FileHash:         "hash-fallback-" + strconv.Itoa(i),
			})
		}
	}
	return g
}
