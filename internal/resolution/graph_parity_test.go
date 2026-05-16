package resolution

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/tamnguyendinh/avmatrix-go/internal/graph"
)

type graphCountBaseline struct {
	ResolutionOnly          graphCountSnapshot `json:"resolutionOnly"`
	FullGraph               graphCountSnapshot `json:"fullGraph"`
	MustMatchResolutionOnly []string           `json:"mustMatchResolutionOnly"`
	MustMatchFullGraph      []string           `json:"mustMatchFullGraph"`
	ReconciledDeltas        []graphCountDelta  `json:"reconciledDeltas"`
}

type graphCountSnapshot struct {
	Nodes         int            `json:"nodes"`
	Relationships int            `json:"relationships"`
	Counts        map[string]int `json:"counts"`
}

type graphCountDelta struct {
	Type                     string `json:"type"`
	Go                       int    `json:"go"`
	TypeScriptResolutionOnly int    `json:"typescriptResolutionOnly"`
	TypeScriptFullGraph      int    `json:"typescriptFullGraph"`
	Classification           string `json:"classification"`
}

func TestResolveTypeScriptGraphBaselineCountsAreReconciled(t *testing.T) {
	baseline := readGraphCountBaseline(t)
	result, err := Resolve(parseFixtureWorkspace(t), Options{})
	if err != nil {
		t.Fatalf("resolve failed: %v", err)
	}

	if len(result.Graph.Nodes) != baseline.ResolutionOnly.Nodes {
		t.Fatalf("Go node count = %d, TypeScript resolution-only node count = %d", len(result.Graph.Nodes), baseline.ResolutionOnly.Nodes)
	}

	goCounts := stringRelationshipCounts(result.Graph.RelationshipCountsByType())
	for _, relType := range baseline.MustMatchResolutionOnly {
		if goCounts[relType] != baseline.ResolutionOnly.Counts[relType] {
			t.Fatalf("%s count = %d, TypeScript resolution-only count = %d", relType, goCounts[relType], baseline.ResolutionOnly.Counts[relType])
		}
	}
	for _, relType := range baseline.MustMatchFullGraph {
		if goCounts[relType] != baseline.FullGraph.Counts[relType] {
			t.Fatalf("%s count = %d, TypeScript full-graph count = %d", relType, goCounts[relType], baseline.FullGraph.Counts[relType])
		}
	}
	for _, delta := range baseline.ReconciledDeltas {
		if delta.Classification == "" {
			t.Fatalf("%s delta is missing classification", delta.Type)
		}
		if goCounts[delta.Type] != delta.Go {
			t.Fatalf("%s Go count = %d, want reconciled count %d (%s)", delta.Type, goCounts[delta.Type], delta.Go, delta.Classification)
		}
		if baseline.ResolutionOnly.Counts[delta.Type] != delta.TypeScriptResolutionOnly {
			t.Fatalf("%s TypeScript resolution-only count drifted: got %d, fixture says %d", delta.Type, baseline.ResolutionOnly.Counts[delta.Type], delta.TypeScriptResolutionOnly)
		}
		if baseline.FullGraph.Counts[delta.Type] != delta.TypeScriptFullGraph {
			t.Fatalf("%s TypeScript full-graph count drifted: got %d, fixture says %d", delta.Type, baseline.FullGraph.Counts[delta.Type], delta.TypeScriptFullGraph)
		}
	}
}

func readGraphCountBaseline(t *testing.T) graphCountBaseline {
	t.Helper()
	raw, err := os.ReadFile("testdata/typescript_graph_baseline_counts.golden.json")
	if err != nil {
		t.Fatalf("read graph baseline: %v", err)
	}
	var baseline graphCountBaseline
	if err := json.Unmarshal(raw, &baseline); err != nil {
		t.Fatalf("decode graph baseline: %v", err)
	}
	return baseline
}

func stringRelationshipCounts(counts map[graph.RelationshipType]int) map[string]int {
	out := make(map[string]int, len(counts))
	for relType, count := range counts {
		out[string(relType)] = count
	}
	return out
}
