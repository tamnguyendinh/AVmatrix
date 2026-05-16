package resolution

import (
	"testing"

	"github.com/tamnguyendinh/avmatrix-go/internal/graph"
	"github.com/tamnguyendinh/avmatrix-go/internal/scanner"
	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

func TestLegacyHeritageMapConversionCoversParentsAncestorsCyclesAndDuplicates(t *testing.T) {
	files := []scopeir.ScopeIR{
		legacyHeritageIR("src/a.ts", scanner.TypeScript, "A", scopeir.NodeClass,
			legacyHeritageItem("A", "B", scopeir.HeritageExtends),
		),
		legacyHeritageIR("src/b.ts", scanner.TypeScript, "B", scopeir.NodeClass,
			legacyHeritageItem("B", "C", scopeir.HeritageExtends),
			legacyHeritageItem("B", "Named", scopeir.HeritageImplements),
			legacyHeritageItem("B", "Named", scopeir.HeritageImplements),
		),
		legacyHeritageIR("src/c.ts", scanner.TypeScript, "C", scopeir.NodeClass,
			legacyHeritageItem("C", "A", scopeir.HeritageExtends),
		),
		legacyHeritageIR("src/named.ts", scanner.TypeScript, "Named", scopeir.NodeInterface),
		legacyHeritageIR("src/missing-parent.ts", scanner.TypeScript, "MissingParentChild", scopeir.NodeClass,
			legacyHeritageItem("MissingParentChild", "DoesNotExist", scopeir.HeritageExtends),
		),
		legacyHeritageIR("src/self.ts", scanner.TypeScript, "Self", scopeir.NodeClass,
			legacyHeritageItem("Self", "Self", scopeir.HeritageExtends),
		),
	}
	workspace, err := buildWorkspace(files)
	if err != nil {
		t.Fatalf("buildWorkspace() error = %v", err)
	}

	a := workspace.defsByID["def:src/a.ts:A"]
	b := workspace.defsByID["def:src/b.ts:B"]
	c := workspace.defsByID["def:src/c.ts:C"]
	named := workspace.defsByID["def:src/named.ts:Named"]

	requireHeritageResolution(t, workspace, a.Fact.ID, b.Fact.ID, scopeir.HeritageExtends)
	requireHeritageResolution(t, workspace, b.Fact.ID, c.Fact.ID, scopeir.HeritageExtends)
	requireHeritageResolution(t, workspace, b.Fact.ID, named.Fact.ID, scopeir.HeritageImplements)
	requireNoHeritageResolution(t, workspace, "def:src/missing-parent.ts:MissingParentChild", "DoesNotExist")
	requireNoHeritageResolution(t, workspace, "def:src/self.ts:Self", "def:src/self.ts:Self")

	ancestors := workspace.ancestorsOf(a.Fact.ID)
	requireAncestorIDs(t, ancestors, []string{b.Fact.ID, c.Fact.ID, named.Fact.ID})
}

func TestLegacyHeritageProcessorConversionEmitsCompatibilityEdges(t *testing.T) {
	files := []scopeir.ScopeIR{
		legacyHeritageIR("src/service.ts", scanner.TypeScript, "Service", scopeir.NodeClass,
			legacyHeritageItem("Service", "BaseService", scopeir.HeritageExtends),
			legacyHeritageItem("Service", "Named", scopeir.HeritageImplements),
		),
		legacyHeritageIR("src/base.ts", scanner.TypeScript, "BaseService", scopeir.NodeClass),
		legacyHeritageIR("src/named.ts", scanner.TypeScript, "Named", scopeir.NodeInterface),
		legacyHeritageIR("src/point.rs", scanner.Rust, "Point", scopeir.NodeStruct,
			legacyHeritageItem("Point", "Display", scopeir.HeritageTraitImpl),
		),
		legacyHeritageIR("src/display.rs", scanner.Rust, "Display", scopeir.NodeTrait),
	}

	result, err := Resolve(files, Options{})
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	requireRelationship(t, result.Graph, graph.RelExtends, "Class:src/service.ts:Service", "Class:src/base.ts:BaseService")
	requireRelationship(t, result.Graph, graph.RelImplements, "Class:src/service.ts:Service", "Interface:src/named.ts:Named")
	requireRelationship(t, result.Graph, graph.RelImplements, "Struct:src/point.rs:Point", "Trait:src/display.rs:Display")
	if result.Metrics.ResolvedInheritance != 3 {
		t.Fatalf("ResolvedInheritance = %d, want 3; metrics=%#v", result.Metrics.ResolvedInheritance, result.Metrics)
	}
}

func legacyHeritageIR(filePath string, language scanner.Language, name string, label scopeir.NodeLabel, heritage ...scopeir.HeritageFact) scopeir.ScopeIR {
	moduleScope := "scope:" + filePath + ":module"
	ownerScope := "scope:" + filePath + ":" + name
	def := scopeir.DefinitionFact{
		ID:            "def:" + filePath + ":" + name,
		FilePath:      filePath,
		Name:          name,
		Label:         label,
		QualifiedName: name,
		Range:         scopeir.Range{StartLine: 1, EndLine: 4},
	}
	for index := range heritage {
		heritage[index].FilePath = filePath
		heritage[index].InScope = ownerScope
	}
	return scopeir.ScopeIR{
		FilePath:    filePath,
		FileHash:    "hash-" + filePath,
		Language:    language,
		ModuleScope: moduleScope,
		Scopes: []scopeir.ScopeFact{
			{
				ID:          moduleScope,
				Kind:        scopeir.ScopeModule,
				FilePath:    filePath,
				Range:       scopeir.Range{StartLine: 1, EndLine: 4},
				Bindings:    []scopeir.BindingFact{{Name: name, DefID: def.ID, Origin: scopeir.BindingLocal}},
				OwnedDefIDs: []string{def.ID},
			},
			{
				ID:          ownerScope,
				Parent:      stringPtr(moduleScope),
				Kind:        scopeir.ScopeClass,
				FilePath:    filePath,
				Range:       scopeir.Range{StartLine: 1, EndLine: 4},
				OwnedDefIDs: []string{def.ID},
			},
		},
		Definitions: []scopeir.DefinitionFact{def},
		Heritage:    heritage,
	}
}

func legacyHeritageItem(owner string, parent string, kind scopeir.HeritageKind) scopeir.HeritageFact {
	return scopeir.HeritageFact{
		Name: parent,
		Kind: kind,
		Range: scopeir.Range{
			StartLine: 1,
			EndLine:   1,
		},
	}
}

func requireHeritageResolution(t *testing.T, workspace *workspace, ownerID string, targetID string, kind scopeir.HeritageKind) {
	t.Helper()
	for _, item := range workspace.heritage {
		if item.Owner.Fact.ID == ownerID && item.Target.Fact.ID == targetID && item.Fact.Kind == kind {
			return
		}
	}
	t.Fatalf("missing heritage owner=%s target=%s kind=%s in %#v", ownerID, targetID, kind, workspace.heritage)
}

func requireNoHeritageResolution(t *testing.T, workspace *workspace, ownerID string, targetID string) {
	t.Helper()
	for _, item := range workspace.heritage {
		if item.Owner.Fact.ID == ownerID && (item.Target.Fact.ID == targetID || item.Fact.Name == targetID) {
			t.Fatalf("unexpected heritage owner=%s target=%s in %#v", ownerID, targetID, workspace.heritage)
		}
	}
}

func requireAncestorIDs(t *testing.T, ancestors []defRef, want []string) {
	t.Helper()
	got := make(map[string]struct{}, len(ancestors))
	for _, ancestor := range ancestors {
		got[ancestor.Fact.ID] = struct{}{}
	}
	if len(got) != len(want) {
		t.Fatalf("ancestor count = %d, want %d; ancestors=%#v", len(got), len(want), ancestors)
	}
	for _, id := range want {
		if _, ok := got[id]; !ok {
			t.Fatalf("missing ancestor %s in %#v", id, ancestors)
		}
	}
}
