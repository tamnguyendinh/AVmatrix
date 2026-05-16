package mcp

import (
	"fmt"
	"testing"

	"github.com/tamnguyendinh/avmatrix-go/internal/graph"
	"github.com/tamnguyendinh/avmatrix-go/internal/repo"
	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

func TestQueryToolUsesProcessStepIndex(t *testing.T) {
	store, repoPath := newMCPQueryBenchmarkRepo(t)
	writeMCPQueryBenchmarkGraph(t, repoPath, 25, 4)

	server := NewServer(Config{Store: store})
	payload, err := server.queryTool(map[string]any{"repo": "fixture", "query": "needle", "limit": 3})
	if err != nil {
		t.Fatalf("queryTool: %v", err)
	}
	processes := payload["processes"].([]resourceProcess)
	if len(processes) == 0 || processes[0].Label != "NeedleFlow" {
		t.Fatalf("processes = %#v", processes)
	}
	symbols := payload["process_symbols"].([]map[string]any)
	if len(symbols) != 4 {
		t.Fatalf("process symbols = %#v", symbols)
	}
	if symbols[0]["id"] != "Function:needle0" {
		t.Fatalf("first symbol = %#v", symbols[0])
	}
}

func BenchmarkQueryToolWarmProcessIndex(b *testing.B) {
	store, repoPath := newMCPQueryBenchmarkRepo(b)
	writeMCPQueryBenchmarkGraph(b, repoPath, 700, 4)

	server := NewServer(Config{Store: store})
	args := map[string]any{"repo": "fixture", "query": "needle", "limit": 5}
	if _, err := server.queryTool(args); err != nil {
		b.Fatalf("warm query: %v", err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		payload, err := server.queryTool(args)
		if err != nil {
			b.Fatalf("queryTool: %v", err)
		}
		processes := payload["processes"].([]resourceProcess)
		if len(processes) == 0 || processes[0].Label != "NeedleFlow" {
			b.Fatalf("query processes = %#v", processes)
		}
	}
}

func newMCPQueryBenchmarkRepo(tb testing.TB) (repo.Store, string) {
	tb.Helper()
	store := repo.NewStore(tb.TempDir())
	repoPath := tb.TempDir()
	meta := repo.Meta{RepoPath: repoPath, IndexedAt: "2026-05-14T00:00:00Z", LastCommit: "abc123", Stats: &repo.Stats{}}
	if err := repo.SaveMeta(repo.StoragePath(repoPath), meta); err != nil {
		tb.Fatalf("save meta: %v", err)
	}
	if _, err := store.Register(repoPath, meta, repo.RegisterOptions{Name: "fixture"}); err != nil {
		tb.Fatalf("register repo: %v", err)
	}
	return store, repoPath
}

func writeMCPQueryBenchmarkGraph(tb testing.TB, repoPath string, processCount int, stepsPerProcess int) {
	tb.Helper()
	g := graph.New()
	for processIndex := 0; processIndex < processCount; processIndex++ {
		processID := fmt.Sprintf("Process:flow%d", processIndex)
		processLabel := fmt.Sprintf("Flow%d", processIndex)
		if processIndex == 0 {
			processLabel = "NeedleFlow"
		}
		g.AddNode(graph.Node{ID: processID, Label: scopeir.NodeProcess, Properties: graph.NodeProperties{
			"heuristicLabel": processLabel,
			"processType":    "cross_community",
			"stepCount":      stepsPerProcess,
		}})
		for stepIndex := 0; stepIndex < stepsPerProcess; stepIndex++ {
			functionID := fmt.Sprintf("Function:flow%dstep%d", processIndex, stepIndex)
			functionName := fmt.Sprintf("flow%dstep%d", processIndex, stepIndex)
			if processIndex == 0 {
				functionID = fmt.Sprintf("Function:needle%d", stepIndex)
				functionName = fmt.Sprintf("needle%d", stepIndex)
			}
			step := stepIndex + 1
			g.AddNode(graph.Node{ID: functionID, Label: scopeir.NodeFunction, Properties: graph.NodeProperties{
				"name":     functionName,
				"filePath": fmt.Sprintf("src/flow%d.go", processIndex),
			}})
			g.AddRelationship(graph.Relationship{
				ID:       fmt.Sprintf("rel:flow%dstep%d", processIndex, stepIndex),
				SourceID: functionID,
				TargetID: processID,
				Type:     graph.RelStepInProcess,
				Step:     &step,
			})
		}
	}
	writeMCPGraphTB(tb, repoPath, g)
}
