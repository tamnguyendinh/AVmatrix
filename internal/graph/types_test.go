package graph

import (
	"testing"

	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

func TestAddGetCountsAndSliceIteration(t *testing.T) {
	g := New()

	if node, ok := g.GetNode("Function:missing"); ok {
		t.Fatalf("GetNode(missing) = %#v, true; want false", node)
	}
	if relationship, ok := g.GetRelationship("rel:missing"); ok {
		t.Fatalf("GetRelationship(missing) = %#v, true; want false", relationship)
	}

	g.AddNode(testGraphNode("Function:src/a.go:run", scopeir.NodeFunction, "run", "src/a.go"))
	g.AddNode(testGraphNode("Function:src/a.go:save", scopeir.NodeFunction, "save", "src/a.go"))

	node, ok := g.GetNode("Function:src/a.go:run")
	if !ok || node.Properties["name"] != "run" {
		t.Fatalf("GetNode(run) = %#v, %v; want run node", node, ok)
	}
	if len(g.Nodes) != 2 {
		t.Fatalf("node count = %d, want 2", len(g.Nodes))
	}

	seenNodes := map[string]bool{}
	for _, node := range g.Nodes {
		seenNodes[node.ID] = true
	}
	if !seenNodes["Function:src/a.go:run"] || !seenNodes["Function:src/a.go:save"] {
		t.Fatalf("iterated node IDs = %#v, want run and save", seenNodes)
	}

	g.AddRelationship(Relationship{
		ID:         "rel:run-save",
		SourceID:   "Function:src/a.go:run",
		TargetID:   "Function:src/a.go:save",
		Type:       RelCalls,
		Confidence: 1,
		Reason:     "test",
	})

	relationship, ok := g.GetRelationship("rel:run-save")
	if !ok || relationship.SourceID != "Function:src/a.go:run" {
		t.Fatalf("GetRelationship(run-save) = %#v, %v; want run relationship", relationship, ok)
	}
	if len(g.Relationships) != 1 {
		t.Fatalf("relationship count = %d, want 1", len(g.Relationships))
	}

	seenRelationships := map[string]bool{}
	for _, relationship := range g.Relationships {
		seenRelationships[relationship.ID] = true
	}
	if !seenRelationships["rel:run-save"] {
		t.Fatalf("iterated relationship IDs = %#v, want rel:run-save", seenRelationships)
	}
}

func TestGenerateIDMatchesLegacyUtility(t *testing.T) {
	tests := map[string]string{
		GenerateID("Function", "main"):                          "Function:main",
		GenerateID("File", "src/index.ts"):                      "File:src/index.ts",
		GenerateID("Class", "UserService"):                      "Class:UserService",
		GenerateID("Method", "getData"):                         "Method:getData",
		GenerateID("Folder", "src"):                             "Folder:src",
		GenerateID("Interface", "IUser"):                        "Interface:IUser",
		GenerateID("Function", "path/to/file.ts:init"):          "Function:path/to/file.ts:init",
		GenerateID("", ""):                                      ":",
		GenerateID("", "name"):                                  ":name",
		GenerateID("label", ""):                                 "label:",
		GenerateID("CONTAINS", "Folder:src->File:src/index.ts"): "CONTAINS:Folder:src->File:src/index.ts",
		GenerateID("Struct", "Point"):                           "Struct:Point",
		GenerateID("Trait", "Display"):                          "Trait:Display",
		GenerateID("Impl", "Display for Point"):                 "Impl:Display for Point",
		GenerateID("Enum", "Color"):                             "Enum:Color",
		GenerateID("Namespace", "std"):                          "Namespace:std",
		GenerateID("Constructor", "User"):                       "Constructor:User",
	}
	for got, want := range tests {
		if got != want {
			t.Fatalf("GenerateID result = %q, want %q", got, want)
		}
	}
}

func TestDuplicateAddReplacesExistingEntry(t *testing.T) {
	g := New()

	g.AddNode(testGraphNode("Function:src/a.go:run", scopeir.NodeFunction, "run", "src/a.go"))
	g.AddNode(testGraphNode("Function:src/a.go:run", scopeir.NodeFunction, "runFast", "src/a.go"))

	if len(g.Nodes) != 1 {
		t.Fatalf("node count after duplicate add = %d, want 1", len(g.Nodes))
	}
	node, ok := g.GetNode("Function:src/a.go:run")
	if !ok || node.Properties["name"] != "runFast" {
		t.Fatalf("duplicate node result = %#v, %v; want replacement", node, ok)
	}

	g.AddRelationship(Relationship{
		ID:         "rel:run-save",
		SourceID:   "Function:src/a.go:run",
		TargetID:   "Function:src/a.go:save",
		Type:       RelCalls,
		Confidence: 1,
		Reason:     "old",
	})
	g.AddRelationship(Relationship{
		ID:         "rel:run-save",
		SourceID:   "Function:src/a.go:run",
		TargetID:   "Function:src/a.go:save",
		Type:       RelUses,
		Confidence: 0.75,
		Reason:     "new",
	})

	if len(g.Relationships) != 1 {
		t.Fatalf("relationship count after duplicate add = %d, want 1", len(g.Relationships))
	}
	relationship, ok := g.GetRelationship("rel:run-save")
	if !ok || relationship.Type != RelUses || relationship.Reason != "new" {
		t.Fatalf("duplicate relationship result = %#v, %v; want replacement", relationship, ok)
	}
}

func TestRemoveNodeRemovesIncidentRelationships(t *testing.T) {
	g := New()
	g.AddNode(testGraphNode("Function:a", scopeir.NodeFunction, "a", "src/a.go"))
	g.AddNode(testGraphNode("Function:b", scopeir.NodeFunction, "b", "src/b.go"))
	g.AddNode(testGraphNode("Function:c", scopeir.NodeFunction, "c", "src/c.go"))
	g.AddRelationship(Relationship{ID: "rel:a-b", SourceID: "Function:a", TargetID: "Function:b", Type: RelCalls})
	g.AddRelationship(Relationship{ID: "rel:b-c", SourceID: "Function:b", TargetID: "Function:c", Type: RelCalls})

	if !g.RemoveNode("Function:a") {
		t.Fatal("RemoveNode(Function:a) = false, want true")
	}
	if _, ok := g.GetNode("Function:a"); ok {
		t.Fatal("removed node still present")
	}
	if len(g.Nodes) != 2 || len(g.Relationships) != 1 {
		t.Fatalf("graph size after node removal = %d/%d, want 2/1", len(g.Nodes), len(g.Relationships))
	}
	if rel, ok := g.GetRelationship("rel:b-c"); !ok || rel.SourceID != "Function:b" {
		t.Fatalf("remaining relationship = %#v, %v; want rel:b-c", rel, ok)
	}
	if g.RemoveNode("Function:missing") {
		t.Fatal("RemoveNode(missing) = true, want false")
	}
}

func TestRemoveNodesByFileRemovesOnlyMatchingFileAndRelationships(t *testing.T) {
	g := New()
	g.AddNode(testGraphNode("Function:a", scopeir.NodeFunction, "a", "src/foo.go"))
	g.AddNode(testGraphNode("Function:b", scopeir.NodeFunction, "b", "src/foo.go"))
	g.AddNode(testGraphNode("Function:c", scopeir.NodeFunction, "c", "src/bar.go"))
	g.AddRelationship(Relationship{ID: "rel:a-b", SourceID: "Function:a", TargetID: "Function:b", Type: RelCalls})
	g.AddRelationship(Relationship{ID: "rel:b-c", SourceID: "Function:b", TargetID: "Function:c", Type: RelCalls})

	if removed := g.RemoveNodesByFile("src/foo.go"); removed != 2 {
		t.Fatalf("RemoveNodesByFile = %d, want 2", removed)
	}
	if len(g.Nodes) != 1 || len(g.Relationships) != 0 {
		t.Fatalf("graph size after file removal = %d/%d, want 1/0", len(g.Nodes), len(g.Relationships))
	}
	if node, ok := g.GetNode("Function:c"); !ok || node.Properties["name"] != "c" {
		t.Fatalf("remaining node = %#v, %v; want Function:c", node, ok)
	}
	if removed := g.RemoveNodesByFile("src/missing.go"); removed != 0 {
		t.Fatalf("RemoveNodesByFile(missing) = %d, want 0", removed)
	}
}

func TestRemoveRelationshipLeavesNodesAndOtherRelationships(t *testing.T) {
	g := New()
	g.AddNode(testGraphNode("Function:a", scopeir.NodeFunction, "a", "src/a.go"))
	g.AddNode(testGraphNode("Function:b", scopeir.NodeFunction, "b", "src/b.go"))
	g.AddNode(testGraphNode("Function:c", scopeir.NodeFunction, "c", "src/c.go"))
	g.AddRelationship(Relationship{ID: "rel:a-b", SourceID: "Function:a", TargetID: "Function:b", Type: RelCalls})
	g.AddRelationship(Relationship{ID: "rel:b-c", SourceID: "Function:b", TargetID: "Function:c", Type: RelCalls})

	if !g.RemoveRelationship("rel:a-b") {
		t.Fatal("RemoveRelationship(rel:a-b) = false, want true")
	}
	if len(g.Nodes) != 3 || len(g.Relationships) != 1 {
		t.Fatalf("graph size after relationship removal = %d/%d, want 3/1", len(g.Nodes), len(g.Relationships))
	}
	if rel, ok := g.GetRelationship("rel:b-c"); !ok || rel.TargetID != "Function:c" {
		t.Fatalf("remaining relationship = %#v, %v; want rel:b-c", rel, ok)
	}
	if g.RemoveRelationship("rel:a-b") {
		t.Fatal("RemoveRelationship(second call) = true, want false")
	}
}

func TestRelationshipCountsByTypeAndSortedRelationships(t *testing.T) {
	g := New()
	g.AddRelationship(Relationship{ID: "rel:z-call-b", SourceID: "Function:z", TargetID: "Function:b", Type: RelCalls})
	g.AddRelationship(Relationship{ID: "rel:a-use-c", SourceID: "Function:a", TargetID: "Class:c", Type: RelUses})
	g.AddRelationship(Relationship{ID: "rel:a-call-c", SourceID: "Function:a", TargetID: "Class:c", Type: RelCalls})
	g.AddRelationship(Relationship{ID: "rel:a-call-b", SourceID: "Function:a", TargetID: "Function:b", Type: RelCalls})

	counts := g.RelationshipCountsByType()
	if counts[RelCalls] != 3 || counts[RelUses] != 1 {
		t.Fatalf("relationship counts = %#v, want CALLS=3 USES=1", counts)
	}

	sorted := g.SortedRelationships()
	gotOrder := make([]string, 0, len(sorted))
	for _, relationship := range sorted {
		gotOrder = append(gotOrder, relationship.ID)
	}
	wantOrder := []string{"rel:a-call-c", "rel:a-call-b", "rel:z-call-b", "rel:a-use-c"}
	for index := range wantOrder {
		if gotOrder[index] != wantOrder[index] {
			t.Fatalf("sorted relationship order = %#v, want %#v", gotOrder, wantOrder)
		}
	}
	if g.Relationships[0].ID != "rel:z-call-b" {
		t.Fatalf("SortedRelationships mutated original order: first relationship = %q", g.Relationships[0].ID)
	}
}

func TestCompactTrimsSlicesAndRebuildsIndexesLazily(t *testing.T) {
	step := 1
	g := &Graph{
		Nodes:         make([]Node, 1, 8),
		Relationships: make([]Relationship, 1, 8),
		nodeIndex: map[string]int{
			"Function:src/main.ts:main": 0,
		},
		relIndex: map[string]int{
			"rel:CALLS:main->leaf": 0,
		},
	}
	g.Nodes[0] = Node{
		ID:    "Function:src/main.ts:main",
		Label: scopeir.NodeFunction,
		Properties: NodeProperties{
			"name": "main",
		},
	}
	g.Relationships[0] = Relationship{
		ID:         "rel:CALLS:main->leaf",
		SourceID:   "Function:src/main.ts:main",
		TargetID:   "Function:src/main.ts:leaf",
		Type:       RelCalls,
		Confidence: 1,
		Step:       &step,
	}

	g.Compact()

	if len(g.Nodes) != cap(g.Nodes) {
		t.Fatalf("node capacity = %d, want len %d", cap(g.Nodes), len(g.Nodes))
	}
	if len(g.Relationships) != cap(g.Relationships) {
		t.Fatalf("relationship capacity = %d, want len %d", cap(g.Relationships), len(g.Relationships))
	}
	if g.nodeIndex != nil || g.relIndex != nil {
		t.Fatalf("indexes retained after compact: node=%v rel=%v", g.nodeIndex != nil, g.relIndex != nil)
	}
	if node, ok := g.GetNode("Function:src/main.ts:main"); !ok || node.Properties["name"] != "main" {
		t.Fatalf("GetNode after compact = %#v, %v", node, ok)
	}
	if relationship, ok := g.GetRelationship("rel:CALLS:main->leaf"); !ok || relationship.Type != RelCalls {
		t.Fatalf("GetRelationship after compact = %#v, %v", relationship, ok)
	}

	g.AddNode(Node{
		ID:    "Function:src/main.ts:leaf",
		Label: scopeir.NodeFunction,
	})
	g.AddRelationship(Relationship{
		ID:         "rel:CALLS:leaf->main",
		SourceID:   "Function:src/main.ts:leaf",
		TargetID:   "Function:src/main.ts:main",
		Type:       RelCalls,
		Confidence: 1,
	})
	if len(g.Nodes) != 2 || len(g.Relationships) != 2 {
		t.Fatalf("graph size after add = %d/%d, want 2/2", len(g.Nodes), len(g.Relationships))
	}
}

func testGraphNode(id string, label scopeir.NodeLabel, name string, filePath string) Node {
	return Node{
		ID:    id,
		Label: label,
		Properties: NodeProperties{
			"name":     name,
			"filePath": filePath,
		},
	}
}
