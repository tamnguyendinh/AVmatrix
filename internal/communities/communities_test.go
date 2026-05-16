package communities

import (
	"regexp"
	"strings"
	"testing"

	"github.com/tamnguyendinh/avmatrix-go/internal/graph"
	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

func TestCommunityColorPaletteMatchesLegacySurface(t *testing.T) {
	if len(CommunityColors) != 12 {
		t.Fatalf("CommunityColors length = %d, want 12", len(CommunityColors))
	}
	hexColor := regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)
	seen := map[string]bool{}
	for _, color := range CommunityColors {
		if !hexColor.MatchString(color) {
			t.Fatalf("invalid color %q", color)
		}
		if seen[color] {
			t.Fatalf("duplicate color %q", color)
		}
		seen[color] = true
	}
	if CommunityColor(0) != CommunityColors[0] || CommunityColor(12) != CommunityColors[0] || CommunityColor(13) != CommunityColors[1] {
		t.Fatalf("CommunityColor wrap mismatch: 0=%q 12=%q 13=%q", CommunityColor(0), CommunityColor(12), CommunityColor(13))
	}
	if CommunityColor(0) == CommunityColor(1) {
		t.Fatalf("CommunityColor(0) and CommunityColor(1) should differ")
	}
}

func TestApplyEmitsCommunityNodesAndMembershipEdges(t *testing.T) {
	g := graph.New()
	g.AddNode(symbolNode("Function:api.handle", scopeir.NodeFunction, "handle", "src/api/handler.ts"))
	g.AddNode(symbolNode("Function:api.validate", scopeir.NodeFunction, "validate", "src/api/handler.ts"))
	g.AddNode(symbolNode("Function:db.save", scopeir.NodeFunction, "save", "src/db/repo.ts"))
	g.AddRelationship(graph.Relationship{
		ID:         "rel:CALLS:handle->validate",
		SourceID:   "Function:api.handle",
		TargetID:   "Function:api.validate",
		Type:       graph.RelCalls,
		Confidence: 1,
	})

	result := Apply(g)
	if result.Metrics.CommunitiesEmitted != 1 || result.Metrics.MembershipsEmitted != 2 {
		t.Fatalf("metrics = %#v, want one community with two memberships", result.Metrics)
	}
	community, ok := g.GetNode("comm_0")
	if !ok {
		t.Fatal("community node comm_0 missing")
	}
	if community.Label != scopeir.NodeCommunity || community.Properties["heuristicLabel"] != "Api" {
		t.Fatalf("community node = %#v", community)
	}
	requireRelationship(t, g, graph.RelMemberOf, "Function:api.handle", "comm_0")
	requireRelationship(t, g, graph.RelMemberOf, "Function:api.validate", "comm_0")
	if _, ok := findRelationship(g, graph.RelMemberOf, "Function:db.save", "comm_0"); ok {
		t.Fatal("isolated symbol pulled into community")
	}
}

func TestApplySeparatesBridgeLinkedDenseCommunities(t *testing.T) {
	g := graph.New()
	api := []string{"Function:api.handle", "Function:api.validate", "Function:api.respond"}
	db := []string{"Function:db.load", "Function:db.save", "Function:db.commit"}
	for _, id := range api {
		g.AddNode(symbolNode(id, scopeir.NodeFunction, id[strings.LastIndex(id, ".")+1:], "src/api/handler.ts"))
	}
	for _, id := range db {
		g.AddNode(symbolNode(id, scopeir.NodeFunction, id[strings.LastIndex(id, ".")+1:], "src/db/repo.ts"))
	}
	addClique(t, g, api)
	addClique(t, g, db)
	g.AddRelationship(graph.Relationship{
		ID:         "rel:CALLS:api.respond->db.load",
		SourceID:   "Function:api.respond",
		TargetID:   "Function:db.load",
		Type:       graph.RelCalls,
		Confidence: 1,
	})

	result := Apply(g)
	if result.Metrics.CommunitiesEmitted != 2 {
		t.Fatalf("communities emitted = %d, want 2; memberships=%#v", result.Metrics.CommunitiesEmitted, result.Memberships)
	}
	membership := membershipByNode(result.Memberships)
	apiCommunity := membership["Function:api.handle"]
	dbCommunity := membership["Function:db.load"]
	if apiCommunity == "" || dbCommunity == "" {
		t.Fatalf("missing memberships: %#v", membership)
	}
	if apiCommunity == dbCommunity {
		t.Fatalf("bridge-linked dense communities merged into %s", apiCommunity)
	}
	for _, id := range api {
		if membership[id] != apiCommunity {
			t.Fatalf("api node %s in community %q, want %q", id, membership[id], apiCommunity)
		}
	}
	for _, id := range db {
		if membership[id] != dbCommunity {
			t.Fatalf("db node %s in community %q, want %q", id, membership[id], dbCommunity)
		}
	}
}

func TestApplySkipsSingletonCommunities(t *testing.T) {
	g := graph.New()
	g.AddNode(symbolNode("Function:solo", scopeir.NodeFunction, "solo", "src/solo.ts"))

	result := Apply(g)
	if result.Metrics.CommunitiesEmitted != 0 || result.Metrics.MembershipsEmitted != 0 {
		t.Fatalf("metrics = %#v, want no singleton community", result.Metrics)
	}
}

func TestApplyProducesDeterministicCommunityOutput(t *testing.T) {
	first := deterministicCommunityGraph()
	second := deterministicCommunityGraph()

	firstResult := Apply(first)
	secondResult := Apply(second)

	if got, want := communitySignature(firstResult), communitySignature(secondResult); got != want {
		t.Fatalf("community signature drifted\nfirst:  %s\nsecond: %s", got, want)
	}
	if firstResult.Metrics.CommunitiesEmitted != 2 || secondResult.Metrics.CommunitiesEmitted != 2 {
		t.Fatalf("community metrics first=%#v second=%#v, want two communities", firstResult.Metrics, secondResult.Metrics)
	}
}

func deterministicCommunityGraph() *graph.Graph {
	g := graph.New()
	api := []string{"Function:api.handle", "Function:api.validate", "Function:api.respond"}
	db := []string{"Function:db.load", "Function:db.save", "Function:db.commit"}
	for _, id := range api {
		g.AddNode(symbolNode(id, scopeir.NodeFunction, id[strings.LastIndex(id, ".")+1:], "src/api/handler.ts"))
	}
	for _, id := range db {
		g.AddNode(symbolNode(id, scopeir.NodeFunction, id[strings.LastIndex(id, ".")+1:], "src/db/repo.ts"))
	}
	addClique(nil, g, api)
	addClique(nil, g, db)
	return g
}

func communitySignature(result Result) string {
	parts := make([]string, 0, len(result.Communities)+len(result.Memberships))
	for _, community := range result.Communities {
		parts = append(parts, community.ID+":"+community.HeuristicLabel+":"+strings.Join(community.Members, ","))
	}
	for _, membership := range result.Memberships {
		parts = append(parts, membership.NodeID+"->"+membership.CommunityID)
	}
	return strings.Join(parts, "|")
}

func symbolNode(id string, label scopeir.NodeLabel, name string, filePath string) graph.Node {
	return graph.Node{
		ID:    id,
		Label: label,
		Properties: graph.NodeProperties{
			"name":     name,
			"filePath": filePath,
		},
	}
}

func addClique(t *testing.T, g *graph.Graph, nodes []string) {
	if t != nil {
		t.Helper()
	}
	for i := 0; i < len(nodes); i++ {
		for j := i + 1; j < len(nodes); j++ {
			g.AddRelationship(graph.Relationship{
				ID:         "rel:CALLS:" + nodes[i] + "->" + nodes[j],
				SourceID:   nodes[i],
				TargetID:   nodes[j],
				Type:       graph.RelCalls,
				Confidence: 1,
			})
		}
	}
}

func membershipByNode(memberships []Membership) map[string]string {
	out := make(map[string]string, len(memberships))
	for _, membership := range memberships {
		out[membership.NodeID] = membership.CommunityID
	}
	return out
}

func requireRelationship(t *testing.T, g *graph.Graph, relType graph.RelationshipType, sourceID string, targetID string) {
	t.Helper()
	if rel, ok := findRelationship(g, relType, sourceID, targetID); !ok || rel.ResolutionSource != "community-detection" {
		t.Fatalf("missing relationship %s %s -> %s", relType, sourceID, targetID)
	}
}

func findRelationship(g *graph.Graph, relType graph.RelationshipType, sourceID string, targetID string) (graph.Relationship, bool) {
	for _, rel := range g.Relationships {
		if rel.Type == relType && rel.SourceID == sourceID && rel.TargetID == targetID {
			return rel, true
		}
	}
	return graph.Relationship{}, false
}
