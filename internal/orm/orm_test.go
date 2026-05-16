package orm

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tamnguyendinh/avmatrix-go/internal/graph"
	"github.com/tamnguyendinh/avmatrix-go/internal/scanner"
	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
	"github.com/tamnguyendinh/avmatrix-go/internal/structure"
)

func TestApplyEmitsPrismaAndSupabaseQueryEdges(t *testing.T) {
	dir := t.TempDir()
	writeORMTestFile(t, dir, "src/prisma-service.ts", `
export async function load() {
  await prisma.user.findMany()
  await prisma.user.create({ data: {} })
  await prisma.post.findMany()
}
`)
	writeORMTestFile(t, dir, "src/supabase-service.ts", `
export async function loadBookings() {
  await supabase.from('bookings').select('*')
  await supabase.from('interpreters').insert({})
  await supabase.from('sessions').select('*')
  return supabase.from('events').select('*')
}
`)
	files := []scanner.File{
		{Path: "src/prisma-service.ts", Language: scanner.TypeScript},
		{Path: "src/supabase-service.ts", Language: scanner.TypeScript},
	}
	g := graph.New()
	structure.Apply(g, files)

	result, err := Apply(g, dir, files)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if result.Metrics.FilesScanned != 2 || result.Metrics.QueriesDetected != 7 || result.Metrics.QueriesEmitted != 7 {
		t.Fatalf("metrics = %#v", result.Metrics)
	}
	if result.Metrics.ModelCount != 6 || result.Metrics.ModelNodesEmitted != 6 {
		t.Fatalf("model metrics = %#v", result.Metrics)
	}
	userModel := graph.GenerateID(string(scopeir.NodeCodeElement), "prisma:user")
	postModel := graph.GenerateID(string(scopeir.NodeCodeElement), "prisma:post")
	bookingsModel := graph.GenerateID(string(scopeir.NodeCodeElement), "supabase:bookings")
	interpretersModel := graph.GenerateID(string(scopeir.NodeCodeElement), "supabase:interpreters")
	sessionsModel := graph.GenerateID(string(scopeir.NodeCodeElement), "supabase:sessions")
	eventsModel := graph.GenerateID(string(scopeir.NodeCodeElement), "supabase:events")
	requireORMNode(t, g, userModel, "user")
	requireORMNode(t, g, postModel, "post")
	requireORMNode(t, g, bookingsModel, "bookings")
	requireORMNode(t, g, interpretersModel, "interpreters")
	requireORMNode(t, g, sessionsModel, "sessions")
	requireORMNode(t, g, eventsModel, "events")
	prismaFile := graph.GenerateID(string(scopeir.NodeFile), "src/prisma-service.ts")
	supabaseFile := graph.GenerateID(string(scopeir.NodeFile), "src/supabase-service.ts")
	requireORMRelationship(t, g, prismaFile, userModel, "prisma-findMany")
	requireORMRelationship(t, g, prismaFile, userModel, "prisma-create")
	requireORMRelationship(t, g, prismaFile, postModel, "prisma-findMany")
	requireORMRelationship(t, g, supabaseFile, bookingsModel, "supabase-select")
	requireORMRelationship(t, g, supabaseFile, interpretersModel, "supabase-insert")
	requireORMRelationship(t, g, supabaseFile, sessionsModel, "supabase-select")
	requireORMRelationship(t, g, supabaseFile, eventsModel, "supabase-select")
}

func TestApplyReusesUniqueExistingModelNodeAndDeduplicates(t *testing.T) {
	dir := t.TempDir()
	writeORMTestFile(t, dir, "src/account.ts", `
await prisma.Account.findFirst()
await prisma.Account.findFirst()
await supabase.from('logs').delete()
`)
	files := []scanner.File{{Path: "src/account.ts", Language: scanner.TypeScript}}
	g := graph.New()
	structure.Apply(g, files)
	accountNode := graph.GenerateID(string(scopeir.NodeClass), "src/models.ts:Account")
	g.AddNode(graph.Node{
		ID:    accountNode,
		Label: scopeir.NodeClass,
		Properties: graph.NodeProperties{
			"name":     "Account",
			"filePath": "src/models.ts",
		},
	})

	result, err := Apply(g, dir, files)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if result.Metrics.QueriesDetected != 3 || result.Metrics.QueriesEmitted != 2 || result.Metrics.Duplicates != 1 {
		t.Fatalf("metrics = %#v", result.Metrics)
	}
	if result.Metrics.ModelCount != 2 || result.Metrics.ModelNodesEmitted != 1 {
		t.Fatalf("model metrics = %#v", result.Metrics)
	}
	fileNode := graph.GenerateID(string(scopeir.NodeFile), "src/account.ts")
	requireORMRelationship(t, g, fileNode, accountNode, "prisma-findFirst")
	requireORMRelationship(t, g, fileNode, graph.GenerateID(string(scopeir.NodeCodeElement), "supabase:logs"), "supabase-delete")
}

func writeORMTestFile(t *testing.T, root string, rel string, contents string) {
	t.Helper()
	fullPath := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", rel, err)
	}
	if err := os.WriteFile(fullPath, []byte(contents), 0o644); err != nil {
		t.Fatalf("write %s: %v", rel, err)
	}
}

func requireORMNode(t *testing.T, g *graph.Graph, id string, name string) {
	t.Helper()
	node, ok := g.GetNode(id)
	if !ok {
		t.Fatalf("missing ORM node %s", id)
	}
	if node.Label != scopeir.NodeCodeElement {
		t.Fatalf("node %s label = %s, want %s", id, node.Label, scopeir.NodeCodeElement)
	}
	if node.Properties["name"] != name {
		t.Fatalf("node %s name = %#v, want %s", id, node.Properties["name"], name)
	}
}

func requireORMRelationship(t *testing.T, g *graph.Graph, sourceID string, targetID string, reason string) {
	t.Helper()
	for _, rel := range g.Relationships {
		if rel.Type == graph.RelQueries && rel.SourceID == sourceID && rel.TargetID == targetID && rel.Reason == reason {
			return
		}
	}
	t.Fatalf("missing QUERIES %s -> %s reason %s", sourceID, targetID, reason)
}
