package communities

import (
	"math"
	"strconv"
	"testing"

	"github.com/tamnguyendinh/avmatrix-go/internal/graph"
	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

func TestCohesionUsesInternalEdgeRatioWithBoundaryEdges(t *testing.T) {
	g := graph.New()
	clique := addCohesionNodes(t, g, "c", "cluster", 4)
	external := addCohesionNodes(t, g, "ext", "other", 2)
	addClique(t, g, clique)
	g.AddRelationship(graph.Relationship{ID: "rel:external", SourceID: external[0], TargetID: external[1], Type: graph.RelCalls, Confidence: 1})
	g.AddRelationship(graph.Relationship{ID: "rel:boundary0", SourceID: clique[0], TargetID: external[0], Type: graph.RelCalls, Confidence: 1})
	g.AddRelationship(graph.Relationship{ID: "rel:boundary1", SourceID: clique[1], TargetID: external[1], Type: graph.RelCalls, Confidence: 1})

	got := cohesion(clique, buildIndex(g))

	if got >= 1 {
		t.Fatalf("cohesion = %v, want edge ratio below graph density 1.0", got)
	}
	requireFloatNear(t, got, 12.0/14.0, 0.000001)
}

func TestCohesionIsOneForClosedCommunities(t *testing.T) {
	g := graph.New()
	pair := addCohesionNodes(t, g, "pair", "pair", 2)
	g.AddRelationship(graph.Relationship{ID: "rel:pair", SourceID: pair[0], TargetID: pair[1], Type: graph.RelCalls, Confidence: 1})

	got := cohesion(pair, buildIndex(g))

	requireFloatNear(t, got, 1, 0.000001)
}

func TestCohesionDecreasesAsBoundaryEdgesIncrease(t *testing.T) {
	oneBoundary := cohesionForCliqueWithBoundaryEdges(t, "one", 1)
	fourBoundaries := cohesionForCliqueWithBoundaryEdges(t, "four", 4)

	if fourBoundaries >= oneBoundary {
		t.Fatalf("cohesion with four boundary edges = %v, want less than one boundary edge %v", fourBoundaries, oneBoundary)
	}
}

func TestApplyReportsEdgeRatioCohesion(t *testing.T) {
	g := graph.New()
	triangle := addCohesionNodes(t, g, "tri", "tri", 3)
	external := addCohesionNodes(t, g, "ext", "ext", 2)
	addClique(t, g, triangle)
	g.AddRelationship(graph.Relationship{ID: "rel:external", SourceID: external[0], TargetID: external[1], Type: graph.RelCalls, Confidence: 1})
	g.AddRelationship(graph.Relationship{ID: "rel:boundary", SourceID: triangle[0], TargetID: external[0], Type: graph.RelCalls, Confidence: 1})

	result := Apply(g)
	membership := membershipByNode(result.Memberships)
	triangleCommunityID := membership[triangle[0]]
	if triangleCommunityID == "" {
		t.Fatalf("missing triangle membership: %#v", membership)
	}
	var triangleCommunity *Community
	for index := range result.Communities {
		if result.Communities[index].ID == triangleCommunityID {
			triangleCommunity = &result.Communities[index]
			break
		}
	}
	if triangleCommunity == nil {
		t.Fatalf("missing triangle community %q in %#v", triangleCommunityID, result.Communities)
	}

	requireFloatNear(t, triangleCommunity.Cohesion, 6.0/7.0, 0.000001)
	if math.Abs(triangleCommunity.Cohesion-1) < 0.000001 {
		t.Fatalf("cohesion = %v, want edge ratio rather than density 1.0", triangleCommunity.Cohesion)
	}
}

func TestApplyReturnsNoCommunitiesForEmptyGraph(t *testing.T) {
	result := Apply(graph.New())

	if len(result.Communities) != 0 || len(result.Memberships) != 0 {
		t.Fatalf("empty graph result = %#v, want no communities or memberships", result)
	}
	if result.Metrics.CommunitiesEmitted != 0 || result.Metrics.NodesConsidered != 0 {
		t.Fatalf("empty graph metrics = %#v, want zeroed metrics", result.Metrics)
	}
}

func cohesionForCliqueWithBoundaryEdges(t *testing.T, prefix string, boundaryEdges int) float64 {
	t.Helper()

	g := graph.New()
	clique := addCohesionNodes(t, g, prefix+"c", "cluster", 4)
	external := addCohesionNodes(t, g, prefix+"ext", "external", boundaryEdges)
	addClique(t, g, clique)
	for index, externalID := range external {
		g.AddRelationship(graph.Relationship{
			ID:         prefix + ":boundary:" + externalID,
			SourceID:   clique[index%len(clique)],
			TargetID:   externalID,
			Type:       graph.RelCalls,
			Confidence: 1,
		})
	}
	return cohesion(clique, buildIndex(g))
}

func addCohesionNodes(t *testing.T, g *graph.Graph, prefix string, folder string, count int) []string {
	t.Helper()

	ids := make([]string, 0, count)
	for index := 0; index < count; index++ {
		suffix := strconv.Itoa(index)
		id := "Function:" + prefix + "." + suffix
		ids = append(ids, id)
		g.AddNode(symbolNode(id, scopeir.NodeFunction, prefix+"Fn"+suffix, "src/"+folder+"/f.go"))
	}
	return ids
}

func requireFloatNear(t *testing.T, got float64, want float64, tolerance float64) {
	t.Helper()
	if math.Abs(got-want) > tolerance {
		t.Fatalf("value = %v, want %v +/- %v", got, want, tolerance)
	}
}
