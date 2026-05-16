package scopeir

import (
	"errors"
	"testing"
)

func TestLegacyP7ScopeExtractorConversionCoversScopeTreeInvariants(t *testing.T) {
	module := ScopeFact{
		ID:       "scope:src/app.ts#1:0-40:0:Module",
		Kind:     ScopeModule,
		FilePath: "src/app.ts",
		Range:    Range{StartLine: 1, StartCol: 0, EndLine: 40, EndCol: 0},
	}
	class := ScopeFact{
		ID:       "scope:src/app.ts#5:0-30:0:Class",
		Parent:   &module.ID,
		Kind:     ScopeClass,
		FilePath: "src/app.ts",
		Range:    Range{StartLine: 5, StartCol: 0, EndLine: 30, EndCol: 0},
	}
	method := ScopeFact{
		ID:       "scope:src/app.ts#10:2-20:2:Function",
		Parent:   &class.ID,
		Kind:     ScopeFunction,
		FilePath: "src/app.ts",
		Range:    Range{StartLine: 10, StartCol: 2, EndLine: 20, EndCol: 2},
	}
	sibling := ScopeFact{
		ID:       "scope:src/app.ts#31:0-35:0:Function",
		Parent:   &module.ID,
		Kind:     ScopeFunction,
		FilePath: "src/app.ts",
		Range:    Range{StartLine: 31, StartCol: 0, EndLine: 35, EndCol: 0},
	}

	tree, err := BuildScopeTree([]ScopeFact{method, sibling, module, class})
	if err != nil {
		t.Fatalf("BuildScopeTree() error = %v", err)
	}
	if tree.Size() != 4 {
		t.Fatalf("tree size = %d, want 4", tree.Size())
	}
	if parent, ok := tree.GetParent(method.ID); !ok || parent.ID != class.ID {
		t.Fatalf("method parent = %#v, %v; want class", parent, ok)
	}
	if ancestors := tree.GetAncestors(method.ID); len(ancestors) != 2 || ancestors[0] != class.ID || ancestors[1] != module.ID {
		t.Fatalf("ancestors = %#v, want class -> module", ancestors)
	}

	overlap := ScopeFact{
		ID:       "scope:src/app.ts#15:0-25:0:Function",
		Parent:   &class.ID,
		Kind:     ScopeFunction,
		FilePath: "src/app.ts",
		Range:    Range{StartLine: 15, StartCol: 0, EndLine: 25, EndCol: 0},
	}
	_, err = BuildScopeTree([]ScopeFact{module, class, method, overlap})
	var invariant *ScopeTreeInvariantError
	if !errors.As(err, &invariant) || invariant.Invariant != ScopeInvariantSiblingRangesOverlap {
		t.Fatalf("overlap error = %#v, want sibling overlap invariant", err)
	}

	_, err = BuildScopeTree([]ScopeFact{{ID: "scope:src/app.ts#1:0-2:0:Function", Kind: ScopeFunction, FilePath: "src/app.ts", Range: Range{StartLine: 1, EndLine: 2}}})
	if !errors.As(err, &invariant) || invariant.Invariant != ScopeInvariantNonModuleRequiresParent {
		t.Fatalf("missing module error = %#v, want non-module parent invariant", err)
	}
}

func TestLegacyP7ScopeExtractorConversionCoversDefinitionIndexFirstWriterWins(t *testing.T) {
	defs := []DefinitionFact{
		{ID: "def:one", FilePath: "src/a.ts", Name: "User", Label: NodeClass, QualifiedName: "User"},
		{ID: "def:one", FilePath: "src/b.ts", Name: "OtherUser", Label: NodeClass, QualifiedName: "OtherUser"},
		{ID: "def:two", FilePath: "src/a.ts", Name: "save", Label: NodeMethod, QualifiedName: "User.save"},
	}

	index := BuildDefinitionIndex(defs)
	if index.Size() != 2 {
		t.Fatalf("definition index size = %d, want 2", index.Size())
	}
	first, ok := index.Get("def:one")
	if !ok || first.Name != "User" {
		t.Fatalf("duplicate definition lookup = %#v, %v; want first writer", first, ok)
	}
	copy := index.ByID()
	delete(copy, "def:one")
	if !index.Has("def:one") {
		t.Fatalf("ByID() returned mutable backing map")
	}
}
