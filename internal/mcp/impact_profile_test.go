package mcp

import (
	"fmt"
	"strings"
	"testing"

	"github.com/tamnguyendinh/avmatrix-go/internal/graph"
	"github.com/tamnguyendinh/avmatrix-go/internal/repo"
	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

func TestImpactToolProfiledKeepsPayloadShape(t *testing.T) {
	store := repo.NewStore(t.TempDir())
	repoPath := t.TempDir()
	meta := repo.Meta{RepoPath: repoPath, IndexedAt: "2026-05-13T00:00:00Z", LastCommit: "abc123", Stats: &repo.Stats{}}
	if err := repo.SaveMeta(repo.StoragePath(repoPath), meta); err != nil {
		t.Fatalf("save meta: %v", err)
	}
	if _, err := store.Register(repoPath, meta, repo.RegisterOptions{Name: "fixture"}); err != nil {
		t.Fatalf("register repo: %v", err)
	}
	writeMCPImpactProfileGraph(t, repoPath)

	server := NewServer(Config{Store: store})
	payload, profile, err := server.impactToolProfiled(map[string]any{
		"repo":      "fixture",
		"target":    "Target",
		"kind":      "Function",
		"direction": "upstream",
		"maxDepth":  2,
	})
	if err != nil {
		t.Fatalf("impactToolProfiled: %v", err)
	}
	if profile.RepoResolve <= 0 {
		t.Fatalf("profile did not record repo resolve timing: %#v", profile)
	}
	if payload["impactedCount"] != 2 {
		t.Fatalf("impact count = %#v, payload %#v", payload["impactedCount"], payload)
	}
	byDepth := payload["byDepth"].(map[string][]map[string]any)
	if len(byDepth["1"]) != 1 || byDepth["1"][0]["id"] != "Function:Caller" {
		t.Fatalf("depth 1 = %#v", byDepth["1"])
	}
	if len(byDepth["2"]) != 1 || byDepth["2"][0]["id"] != "Function:GrandCaller" {
		t.Fatalf("depth 2 = %#v", byDepth["2"])
	}
	processes := payload["affected_processes"].([]map[string]any)
	if len(processes) != 1 || processes[0]["name"] != "RequestFlow" {
		t.Fatalf("affected processes = %#v", processes)
	}
	modules := payload["affected_modules"].([]map[string]any)
	if len(modules) != 1 || modules[0]["name"] != "API" {
		t.Fatalf("affected modules = %#v", modules)
	}
}

func TestImpactToolByUIDReturnsUnknownForMissingTarget(t *testing.T) {
	store := repo.NewStore(t.TempDir())
	repoPath := t.TempDir()
	meta := repo.Meta{RepoPath: repoPath, IndexedAt: "2026-05-13T00:00:00Z", LastCommit: "abc123", Stats: &repo.Stats{}}
	if err := repo.SaveMeta(repo.StoragePath(repoPath), meta); err != nil {
		t.Fatalf("save meta: %v", err)
	}
	if _, err := store.Register(repoPath, meta, repo.RegisterOptions{Name: "fixture"}); err != nil {
		t.Fatalf("register repo: %v", err)
	}
	writeMCPImpactProfileGraph(t, repoPath)

	server := NewServer(Config{Store: store})
	payload, err := server.impactTool(map[string]any{
		"repo":          "fixture",
		"target_uid":    "fake-uid-123",
		"direction":     "upstream",
		"maxDepth":      float64(3),
		"relationTypes": []any{"CALLS", "IMPORTS"},
		"minConfidence": float64(0),
		"includeTests":  false,
	})
	if err != nil {
		t.Fatalf("impactTool(target_uid) error = %v", err)
	}
	if payload["risk"] != "UNKNOWN" || payload["impactedCount"] != 0 {
		t.Fatalf("impact missing uid payload = %#v", payload)
	}
	if errorText, _ := payload["error"].(string); !strings.Contains(errorText, "fake-uid-123") {
		t.Fatalf("impact missing uid error = %#v", payload)
	}
}

func TestImpactConfidenceUsesStoredValueAndGoFallbacks(t *testing.T) {
	cases := []struct {
		name         string
		relationship graph.Relationship
		want         float64
	}{
		{
			name:         "stored confidence wins even when lower than fallback",
			relationship: graph.Relationship{Type: graph.RelHasMethod, Confidence: 0.42},
			want:         0.42,
		},
		{
			name:         "calls fall back to direct-reference confidence",
			relationship: graph.Relationship{Type: graph.RelCalls},
			want:         0.95,
		},
		{
			name:         "imports fall back to direct-reference confidence",
			relationship: graph.Relationship{Type: graph.RelImports},
			want:         0.95,
		},
		{
			name:         "defines fall back to structural direct confidence",
			relationship: graph.Relationship{Type: graph.RelDefines},
			want:         0.95,
		},
		{
			name:         "uses fall back to resolved dependency confidence",
			relationship: graph.Relationship{Type: graph.RelUses},
			want:         0.85,
		},
		{
			name:         "implements fall back to resolved dependency confidence",
			relationship: graph.Relationship{Type: graph.RelImplements},
			want:         0.85,
		},
		{
			name:         "other allowed relations use conservative Go runtime fallback",
			relationship: graph.Relationship{Type: graph.RelHasProperty},
			want:         0.7,
		},
		{
			name:         "unknown relations use conservative Go runtime fallback",
			relationship: graph.Relationship{Type: graph.RelationshipType("UNKNOWN_EDGE")},
			want:         0.7,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := impactConfidence(tc.relationship); got != tc.want {
				t.Fatalf("impactConfidence() = %v, want %v", got, tc.want)
			}
		})
	}

	for _, relType := range []string{
		string(graph.RelCalls),
		string(graph.RelImports),
		string(graph.RelUses),
		string(graph.RelInherits),
		string(graph.RelExtends),
		string(graph.RelImplements),
		string(graph.RelHasMethod),
		string(graph.RelHasProperty),
		string(graph.RelAccesses),
	} {
		if !impactAllowedRelationTypes[relType] {
			t.Fatalf("allowed impact relation type missing %s", relType)
		}
	}
}

func TestImpactAffectedProcessesGroupsRowsByProcess(t *testing.T) {
	g := graph.New()
	for i := 0; i < 6; i++ {
		g.AddNode(graph.Node{ID: fmt.Sprintf("Function:node%d", i), Label: scopeir.NodeFunction, Properties: graph.NodeProperties{
			"name": fmt.Sprintf("node%d", i), "filePath": fmt.Sprintf("src/node%d.go", i),
		}})
	}
	for i, name := range []string{"EP1", "EP2", "EP3"} {
		g.AddNode(graph.Node{ID: fmt.Sprintf("Process:%d", i+1), Label: scopeir.NodeProcess, Properties: graph.NodeProperties{
			"heuristicLabel": name, "processType": "Function", "filePath": fmt.Sprintf("/p/%d", i+1),
		}})
	}
	steps := []int{1, 3, 2, 2, 1, 1}
	g.AddRelationship(graph.Relationship{ID: "rel:node0-ep1", SourceID: "Function:node0", TargetID: "Process:1", Type: graph.RelStepInProcess, Step: &steps[0]})
	g.AddRelationship(graph.Relationship{ID: "rel:node1-ep1", SourceID: "Function:node1", TargetID: "Process:1", Type: graph.RelStepInProcess, Step: &steps[1]})
	g.AddRelationship(graph.Relationship{ID: "rel:node2-ep1", SourceID: "Function:node2", TargetID: "Process:1", Type: graph.RelStepInProcess, Step: &steps[2]})
	g.AddRelationship(graph.Relationship{ID: "rel:node3-ep2", SourceID: "Function:node3", TargetID: "Process:2", Type: graph.RelStepInProcess, Step: &steps[3]})
	g.AddRelationship(graph.Relationship{ID: "rel:node4-ep2", SourceID: "Function:node4", TargetID: "Process:2", Type: graph.RelStepInProcess, Step: &steps[4]})
	g.AddRelationship(graph.Relationship{ID: "rel:node5-ep3", SourceID: "Function:node5", TargetID: "Process:3", Type: graph.RelStepInProcess, Step: &steps[5]})

	impacted := make([]map[string]any, 0, 6)
	for i := 0; i < 6; i++ {
		impacted = append(impacted, map[string]any{"id": fmt.Sprintf("Function:node%d", i), "depth": 1})
	}

	processes := impactAffectedProcesses(g, impacted)
	if len(processes) != 3 {
		t.Fatalf("affected processes = %#v", processes)
	}
	if processes[0]["name"] != "EP1" || processes[0]["total_hits"] != 3 || processes[0]["earliest_broken_step"] != 1 {
		t.Fatalf("EP1 grouping = %#v", processes[0])
	}
	if processes[1]["name"] != "EP2" || processes[1]["total_hits"] != 2 || processes[1]["earliest_broken_step"] != 1 {
		t.Fatalf("EP2 grouping = %#v", processes[1])
	}
	if processes[2]["name"] != "EP3" || processes[2]["total_hits"] != 1 || processes[2]["earliest_broken_step"] != 1 {
		t.Fatalf("EP3 grouping = %#v", processes[2])
	}
}

func TestImpactTraversalUsesCompleteInMemoryGraphWithoutChunkCap(t *testing.T) {
	g := graph.New()
	target := graph.Node{ID: "Function:ImpactTarget", Label: scopeir.NodeFunction, Properties: graph.NodeProperties{
		"name": "ImpactTarget", "filePath": "internal/mcp/impact.go",
	}}
	g.AddNode(target)
	for i := 0; i < 500; i++ {
		callerID := fmt.Sprintf("Function:impactCaller%d", i)
		g.AddNode(graph.Node{ID: callerID, Label: scopeir.NodeFunction, Properties: graph.NodeProperties{
			"name": fmt.Sprintf("impactCaller%d", i), "filePath": fmt.Sprintf("src/caller%d.go", i),
		}})
		g.AddRelationship(graph.Relationship{ID: fmt.Sprintf("rel:impact-caller%d", i), SourceID: callerID, TargetID: target.ID, Type: graph.RelCalls, Confidence: 1})
		if i < 5 {
			processID := fmt.Sprintf("Process:impactFlow%d", i)
			moduleID := fmt.Sprintf("Community:impactModule%d", i)
			step := i + 1
			g.AddNode(graph.Node{ID: processID, Label: scopeir.NodeProcess, Properties: graph.NodeProperties{"heuristicLabel": fmt.Sprintf("ImpactFlow%d", i), "processType": "cross_community"}})
			g.AddNode(graph.Node{ID: moduleID, Label: scopeir.NodeCommunity, Properties: graph.NodeProperties{"name": fmt.Sprintf("ImpactModule%d", i)}})
			g.AddRelationship(graph.Relationship{ID: fmt.Sprintf("rel:impact-process%d", i), SourceID: callerID, TargetID: processID, Type: graph.RelStepInProcess, Step: &step})
			g.AddRelationship(graph.Relationship{ID: fmt.Sprintf("rel:impact-module%d", i), SourceID: callerID, TargetID: moduleID, Type: graph.RelMemberOf})
		}
	}

	payload, _ := runImpactBFSProfiled(g, target, impactOptions{
		Direction:     "upstream",
		MaxDepth:      1,
		RelationTypes: []string{string(graph.RelCalls)},
		IncludeTests:  true,
	}, true)
	if payload["impactedCount"] != 500 {
		t.Fatalf("impactedCount = %#v", payload["impactedCount"])
	}
	if _, partial := payload["partial"]; partial {
		t.Fatalf("Go in-memory impact traversal unexpectedly marked partial: %#v", payload)
	}
	byDepth := payload["byDepth"].(map[string][]map[string]any)
	if len(byDepth["1"]) != 500 {
		t.Fatalf("depth 1 count = %d, want 500", len(byDepth["1"]))
	}
	if processes := payload["affected_processes"].([]map[string]any); len(processes) != 5 {
		t.Fatalf("affected processes = %#v", processes)
	}
	if modules := payload["affected_modules"].([]map[string]any); len(modules) != 5 {
		t.Fatalf("affected modules = %#v", modules)
	}
}

func BenchmarkImpactToolWarmTraversalProfile(b *testing.B) {
	store := repo.NewStore(b.TempDir())
	repoPath := b.TempDir()
	meta := repo.Meta{RepoPath: repoPath, IndexedAt: "2026-05-13T00:00:00Z", LastCommit: "abc123", Stats: &repo.Stats{}}
	if err := repo.SaveMeta(repo.StoragePath(repoPath), meta); err != nil {
		b.Fatalf("save meta: %v", err)
	}
	if _, err := store.Register(repoPath, meta, repo.RegisterOptions{Name: "fixture"}); err != nil {
		b.Fatalf("register repo: %v", err)
	}
	writeMCPBenchmarkImpactProfileGraph(b, repoPath, 2500, 750)

	server := NewServer(Config{Store: store})
	args := map[string]any{"repo": "fixture", "target": "ImpactTarget", "kind": "Function", "direction": "upstream", "maxDepth": 1}
	if _, _, err := server.impactToolProfiled(args); err != nil {
		b.Fatalf("warm impact: %v", err)
	}

	var total impactToolProfile
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		payload, profile, err := server.impactToolProfiled(args)
		if err != nil {
			b.Fatalf("impactToolProfiled: %v", err)
		}
		if payload["impactedCount"] != 2500 {
			b.Fatalf("impact payload = %#v", payload)
		}
		total.RepoResolve += profile.RepoResolve
		total.TargetLookup += profile.TargetLookup
		total.IndexBuild += profile.IndexBuild
		total.Traversal += profile.Traversal
		total.AffectedSummaries += profile.AffectedSummaries
		total.Formatting += profile.Formatting
	}
	b.StopTimer()

	if b.N > 0 {
		n := float64(b.N)
		b.ReportMetric(float64(total.RepoResolve.Nanoseconds())/n/1000, "repo_resolve_us/op")
		b.ReportMetric(float64(total.TargetLookup.Nanoseconds())/n/1000, "target_lookup_us/op")
		b.ReportMetric(float64(total.IndexBuild.Nanoseconds())/n/1000, "node_index_us/op")
		b.ReportMetric(float64(total.Traversal.Nanoseconds())/n/1000, "traversal_us/op")
		b.ReportMetric(float64(total.AffectedSummaries.Nanoseconds())/n/1000, "summaries_us/op")
		b.ReportMetric(float64(total.Formatting.Nanoseconds())/n/1000, "format_us/op")
	}
}

func writeMCPImpactProfileGraph(t testing.TB, repoPath string) {
	t.Helper()
	g := graph.New()
	g.AddNode(graph.Node{ID: "Function:Target", Label: scopeir.NodeFunction, Properties: graph.NodeProperties{"name": "Target", "filePath": "src/target.go"}})
	g.AddNode(graph.Node{ID: "Function:Caller", Label: scopeir.NodeFunction, Properties: graph.NodeProperties{"name": "Caller", "filePath": "src/caller.go"}})
	g.AddNode(graph.Node{ID: "Function:GrandCaller", Label: scopeir.NodeFunction, Properties: graph.NodeProperties{"name": "GrandCaller", "filePath": "src/grand.go"}})
	g.AddNode(graph.Node{ID: "Process:RequestFlow", Label: scopeir.NodeProcess, Properties: graph.NodeProperties{"heuristicLabel": "RequestFlow", "processType": "cross_community"}})
	g.AddNode(graph.Node{ID: "Community:API", Label: scopeir.NodeCommunity, Properties: graph.NodeProperties{"name": "API"}})
	step := 3
	g.AddRelationship(graph.Relationship{ID: "rel:caller-target", SourceID: "Function:Caller", TargetID: "Function:Target", Type: graph.RelCalls, Confidence: 1})
	g.AddRelationship(graph.Relationship{ID: "rel:grand-caller", SourceID: "Function:GrandCaller", TargetID: "Function:Caller", Type: graph.RelCalls, Confidence: 1})
	g.AddRelationship(graph.Relationship{ID: "rel:caller-process", SourceID: "Function:Caller", TargetID: "Process:RequestFlow", Type: graph.RelStepInProcess, Step: &step})
	g.AddRelationship(graph.Relationship{ID: "rel:caller-module", SourceID: "Function:Caller", TargetID: "Community:API", Type: graph.RelMemberOf})
	writeMCPGraphTB(t, repoPath, g)
}

func writeMCPBenchmarkImpactProfileGraph(t testing.TB, repoPath string, callerCount int, processCount int) {
	t.Helper()
	g := graph.New()
	g.AddNode(graph.Node{ID: "Function:ImpactTarget", Label: scopeir.NodeFunction, Properties: graph.NodeProperties{"name": "ImpactTarget", "filePath": "internal/mcp/impact.go"}})
	for i := 0; i < callerCount; i++ {
		id := fmt.Sprintf("Function:impactCaller%d", i)
		g.AddNode(graph.Node{ID: id, Label: scopeir.NodeFunction, Properties: graph.NodeProperties{"name": fmt.Sprintf("impactCaller%d", i), "filePath": fmt.Sprintf("src/caller%d.go", i)}})
		g.AddRelationship(graph.Relationship{ID: fmt.Sprintf("rel:impact-caller%d", i), SourceID: id, TargetID: "Function:ImpactTarget", Type: graph.RelCalls, Confidence: 1})
		if i < processCount {
			processID := fmt.Sprintf("Process:impactFlow%d", i)
			moduleID := fmt.Sprintf("Community:impactModule%d", i%10)
			step := i % 10
			g.AddNode(graph.Node{ID: processID, Label: scopeir.NodeProcess, Properties: graph.NodeProperties{"heuristicLabel": fmt.Sprintf("ImpactFlow%d", i), "processType": "cross_community"}})
			g.AddNode(graph.Node{ID: moduleID, Label: scopeir.NodeCommunity, Properties: graph.NodeProperties{"name": fmt.Sprintf("ImpactModule%d", i%10)}})
			g.AddRelationship(graph.Relationship{ID: fmt.Sprintf("rel:impact-process%d", i), SourceID: id, TargetID: processID, Type: graph.RelStepInProcess, Step: &step})
			g.AddRelationship(graph.Relationship{ID: fmt.Sprintf("rel:impact-module%d", i), SourceID: id, TargetID: moduleID, Type: graph.RelMemberOf})
		}
	}
	writeMCPGraphTB(t, repoPath, g)
}
