package mcp

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/tamnguyendinh/avmatrix-go/internal/graph"
	"github.com/tamnguyendinh/avmatrix-go/internal/repo"
	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

func TestContextToolGraphCacheInvalidatesWhenGraphChanges(t *testing.T) {
	store := repo.NewStore(t.TempDir())
	repoPath := t.TempDir()
	meta := repo.Meta{RepoPath: repoPath, IndexedAt: "2026-05-13T00:00:00Z", LastCommit: "abc123", Stats: &repo.Stats{}}
	if err := repo.SaveMeta(repo.StoragePath(repoPath), meta); err != nil {
		t.Fatalf("save meta: %v", err)
	}
	if _, err := store.Register(repoPath, meta, repo.RegisterOptions{Name: "fixture"}); err != nil {
		t.Fatalf("register repo: %v", err)
	}

	writeMCPClassContextGraphWithCaller(t, repoPath, "makeUser")
	server := NewServer(Config{Store: store})
	first, err := server.contextTool(map[string]any{"repo": "fixture", "name": "User", "kind": "Class"})
	if err != nil {
		t.Fatalf("first context: %v", err)
	}
	assertContextIncomingCall(t, first, "Function:makeUser")

	time.Sleep(10 * time.Millisecond)
	writeMCPClassContextGraphWithCaller(t, repoPath, "makeOrder")
	second, err := server.contextTool(map[string]any{"repo": "fixture", "name": "User", "kind": "Class"})
	if err != nil {
		t.Fatalf("second context: %v", err)
	}
	assertContextIncomingCall(t, second, "Function:makeOrder")
}

func BenchmarkContextToolWarmNeighborhood(b *testing.B) {
	store := repo.NewStore(b.TempDir())
	repoPath := b.TempDir()
	meta := repo.Meta{RepoPath: repoPath, IndexedAt: "2026-05-13T00:00:00Z", LastCommit: "abc123", Stats: &repo.Stats{}}
	if err := repo.SaveMeta(repo.StoragePath(repoPath), meta); err != nil {
		b.Fatalf("save meta: %v", err)
	}
	if _, err := store.Register(repoPath, meta, repo.RegisterOptions{Name: "fixture"}); err != nil {
		b.Fatalf("register repo: %v", err)
	}
	writeMCPBenchmarkContextGraph(b, repoPath, 2500, 2500, 750)

	server := NewServer(Config{Store: store})
	args := map[string]any{"repo": "fixture", "name": "BenchmarkTarget", "kind": "Function"}
	if _, _, err := server.contextToolProfiled(args); err != nil {
		b.Fatalf("warm context: %v", err)
	}

	var total contextToolProfile
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		payload, profile, err := server.contextToolProfiled(args)
		if err != nil {
			b.Fatalf("contextToolProfiled: %v", err)
		}
		if payload["status"] != "found" {
			b.Fatalf("context payload = %#v", payload)
		}
		total.RepoResolve += profile.RepoResolve
		total.TargetLookup += profile.TargetLookup
		total.NeighborhoodRead += profile.NeighborhoodRead
		total.SymbolPayload += profile.SymbolPayload
		total.Formatting += profile.Formatting
	}
	b.StopTimer()

	if b.N > 0 {
		n := float64(b.N)
		b.ReportMetric(float64(total.RepoResolve.Nanoseconds())/n/1000, "repo_resolve_us/op")
		b.ReportMetric(float64(total.TargetLookup.Nanoseconds())/n/1000, "target_lookup_us/op")
		b.ReportMetric(float64(total.NeighborhoodRead.Nanoseconds())/n/1000, "neighborhood_us/op")
		b.ReportMetric(float64(total.SymbolPayload.Nanoseconds())/n/1000, "symbol_us/op")
		b.ReportMetric(float64(total.Formatting.Nanoseconds())/n/1000, "format_us/op")
	}
}

func assertContextIncomingCall(t *testing.T, payload map[string]any, uid string) {
	t.Helper()
	if payload["status"] != "found" {
		t.Fatalf("context status = %#v", payload["status"])
	}
	symbol, ok := payload["symbol"].(map[string]any)
	if !ok || symbol["uid"] != "Class:User" {
		t.Fatalf("context symbol = %#v", payload["symbol"])
	}
	incoming, ok := payload["incoming"].(map[string][]map[string]any)
	if !ok {
		t.Fatalf("incoming payload = %#v", payload["incoming"])
	}
	calls := incoming["calls"]
	if len(calls) != 1 || calls[0]["uid"] != uid {
		t.Fatalf("incoming calls = %#v, want uid %q", calls, uid)
	}
}

func writeMCPClassContextGraphWithCaller(t testing.TB, repoPath string, callerName string) {
	t.Helper()
	g := graph.New()
	g.AddNode(graph.Node{ID: "File:src/user.ts", Label: scopeir.NodeFile, Properties: graph.NodeProperties{
		"name": "user.ts", "filePath": "src/user.ts",
	}})
	g.AddNode(graph.Node{ID: "File:src/consumer.ts", Label: scopeir.NodeFile, Properties: graph.NodeProperties{
		"name": "consumer.ts", "filePath": "src/consumer.ts",
	}})
	g.AddNode(graph.Node{ID: "Class:User", Label: scopeir.NodeClass, Properties: graph.NodeProperties{
		"name": "User", "filePath": "src/user.ts", "startLine": 1, "endLine": 3,
	}})
	g.AddNode(graph.Node{ID: "Constructor:User", Label: scopeir.NodeConstructor, Properties: graph.NodeProperties{
		"name": "User", "filePath": "src/user.ts", "startLine": 2, "endLine": 2,
	}})
	callerID := "Function:" + callerName
	g.AddNode(graph.Node{ID: callerID, Label: scopeir.NodeFunction, Properties: graph.NodeProperties{
		"name": callerName, "filePath": "src/consumer.ts", "startLine": 4, "endLine": 6,
	}})
	g.AddRelationship(graph.Relationship{ID: "rel:file-defines-class", SourceID: "File:src/user.ts", TargetID: "Class:User", Type: graph.RelDefines})
	g.AddRelationship(graph.Relationship{ID: "rel:class-has-ctor", SourceID: "Class:User", TargetID: "Constructor:User", Type: graph.RelHasMethod})
	g.AddRelationship(graph.Relationship{ID: "rel:caller-calls-ctor", SourceID: callerID, TargetID: "Constructor:User", Type: graph.RelCalls})
	g.AddRelationship(graph.Relationship{ID: "rel:consumer-imports-user-file", SourceID: "File:src/consumer.ts", TargetID: "File:src/user.ts", Type: graph.RelImports})
	writeMCPGraphTB(t, repoPath, g)
}

func writeMCPBenchmarkContextGraph(t testing.TB, repoPath string, callerCount int, calleeCount int, processCount int) {
	t.Helper()
	g := graph.New()
	g.AddNode(graph.Node{ID: "Function:BenchmarkTarget", Label: scopeir.NodeFunction, Properties: graph.NodeProperties{
		"name": "BenchmarkTarget", "filePath": "internal/mcp/context.go", "startLine": 1, "endLine": 20,
	}})
	for i := 0; i < callerCount; i++ {
		id := fmt.Sprintf("Function:caller%d", i)
		g.AddNode(graph.Node{ID: id, Label: scopeir.NodeFunction, Properties: graph.NodeProperties{
			"name": fmt.Sprintf("caller%d", i), "filePath": fmt.Sprintf("src/caller%d.go", i), "startLine": i + 1, "endLine": i + 2,
		}})
		g.AddRelationship(graph.Relationship{ID: fmt.Sprintf("rel:caller%d", i), SourceID: id, TargetID: "Function:BenchmarkTarget", Type: graph.RelCalls})
	}
	for i := 0; i < calleeCount; i++ {
		id := fmt.Sprintf("Function:callee%d", i)
		g.AddNode(graph.Node{ID: id, Label: scopeir.NodeFunction, Properties: graph.NodeProperties{
			"name": fmt.Sprintf("callee%d", i), "filePath": fmt.Sprintf("src/callee%d.go", i), "startLine": i + 1, "endLine": i + 2,
		}})
		g.AddRelationship(graph.Relationship{ID: fmt.Sprintf("rel:callee%d", i), SourceID: "Function:BenchmarkTarget", TargetID: id, Type: graph.RelCalls})
	}
	for i := 0; i < processCount; i++ {
		id := fmt.Sprintf("Process:flow%d", i)
		step := i % 10
		g.AddNode(graph.Node{ID: id, Label: scopeir.NodeProcess, Properties: graph.NodeProperties{
			"heuristicLabel": fmt.Sprintf("Flow%d", i), "stepCount": 10,
		}})
		g.AddRelationship(graph.Relationship{ID: fmt.Sprintf("rel:process%d", i), SourceID: "Function:BenchmarkTarget", TargetID: id, Type: graph.RelStepInProcess, Step: &step})
	}
	writeMCPGraphTB(t, repoPath, g)
}

func writeMCPGraphTB(t testing.TB, repoPath string, g *graph.Graph) {
	t.Helper()
	raw, err := json.MarshalIndent(g, "", "  ")
	if err != nil {
		t.Fatalf("marshal graph: %v", err)
	}
	graphPath := filepath.Join(repo.StoragePath(repoPath), "graph.json")
	if err := os.MkdirAll(filepath.Dir(graphPath), 0o755); err != nil {
		t.Fatalf("mkdir graph: %v", err)
	}
	if err := os.WriteFile(graphPath, append(raw, '\n'), 0o644); err != nil {
		t.Fatalf("write graph: %v", err)
	}
}
