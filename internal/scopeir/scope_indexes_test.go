package scopeir

import (
	"errors"
	"reflect"
	"testing"
)

func TestBuildDefinitionIndexLegacyParity(t *testing.T) {
	first := testDef("def:dup", "Original")
	second := testDef("def:dup", "Shadow")
	a := testDef("def:A", "")
	b := testDef("def:B", "")

	empty := BuildDefinitionIndex(nil)
	if empty.Size() != 0 || empty.Has("anything") {
		t.Fatalf("empty index size/has = %d/%v, want 0/false", empty.Size(), empty.Has("anything"))
	}
	if _, ok := empty.Get("anything"); ok {
		t.Fatalf("empty Get returned a definition")
	}

	defs := []DefinitionFact{a, b, first, second}
	idx := BuildDefinitionIndex(defs)
	if idx.Size() != 3 {
		t.Fatalf("Size() = %d, want 3", idx.Size())
	}
	if got, ok := idx.Get("def:A"); !ok || got.ID != "def:A" {
		t.Fatalf("Get(def:A) = %#v, %v; want def:A", got, ok)
	}
	if got, ok := idx.Get("def:dup"); !ok || got.ReturnType != "Original" {
		t.Fatalf("duplicate Get = %#v, %v; want first definition", got, ok)
	}
	if idx.Has("def:missing") {
		t.Fatalf("Has(def:missing) = true, want false")
	}
	byID := idx.ByID()
	if len(byID) != 3 || byID["def:B"].ID != "def:B" {
		t.Fatalf("ByID() = %#v, want copied map with def:B", byID)
	}
	delete(byID, "def:A")
	if !idx.Has("def:A") {
		t.Fatalf("mutating ByID copy changed index")
	}
}

func TestBuildPositionIndexLegacyParity(t *testing.T) {
	t.Run("empty and missing", func(t *testing.T) {
		idx := BuildPositionIndex(nil)
		if idx.Size() != 0 {
			t.Fatalf("empty Size() = %d, want 0", idx.Size())
		}
		if _, ok := idx.AtPosition("src/any.ts", 1, 0); ok {
			t.Fatalf("empty index returned a scope")
		}
		idx = BuildPositionIndex([]ScopeFact{testScope("scope:m", "a.ts", ScopeModule, testRange(5, 0, 10, 0), "")})
		if _, ok := idx.AtPosition("b.ts", 5, 0); ok {
			t.Fatalf("unindexed file returned a scope")
		}
		for _, query := range []struct {
			line int
			col  int
		}{{1, 0}, {4, 99}, {11, 0}, {10, 1}} {
			if _, ok := idx.AtPosition("a.ts", query.line, query.col); ok {
				t.Fatalf("position %d:%d returned a scope", query.line, query.col)
			}
		}
	})

	t.Run("boundaries and innermost", func(t *testing.T) {
		idx := BuildPositionIndex([]ScopeFact{
			testScope("scope:mod", "a.ts", ScopeModule, testRange(1, 0, 100, 0), ""),
			testScope("scope:cls", "a.ts", ScopeClass, testRange(5, 0, 80, 0), "scope:mod"),
			testScope("scope:fn", "a.ts", ScopeFunction, testRange(10, 0, 60, 0), "scope:cls"),
			testScope("scope:blk", "a.ts", ScopeBlock, testRange(20, 0, 50, 0), "scope:fn"),
		})
		requireScopeAt(t, idx, "a.ts", 30, 0, "scope:blk")
		requireScopeAt(t, idx, "a.ts", 15, 0, "scope:fn")
		requireScopeAt(t, idx, "a.ts", 7, 0, "scope:cls")
		requireScopeAt(t, idx, "a.ts", 2, 0, "scope:mod")

		startEnd := BuildPositionIndex([]ScopeFact{
			testScope("scope:m", "a.ts", ScopeModule, testRange(5, 2, 10, 5), ""),
		})
		requireScopeAt(t, startEnd, "a.ts", 5, 2, "scope:m")
		requireScopeAt(t, startEnd, "a.ts", 10, 5, "scope:m")
		requireNoScopeAt(t, startEnd, "a.ts", 5, 1)
		requireNoScopeAt(t, startEnd, "a.ts", 10, 6)
	})

	t.Run("tie breakers, files, columns, and duplicate ids", func(t *testing.T) {
		coStart := BuildPositionIndex([]ScopeFact{
			testScope("scope:outer", "a.ts", ScopeModule, testRange(5, 0, 50, 0), ""),
			testScope("scope:inner", "a.ts", ScopeFunction, testRange(5, 0, 20, 0), "scope:outer"),
		})
		requireScopeAt(t, coStart, "a.ts", 10, 0, "scope:inner")
		requireScopeAt(t, coStart, "a.ts", 30, 0, "scope:outer")

		touchingSiblings := BuildPositionIndex([]ScopeFact{
			testScope("scope:mod", "a.ts", ScopeModule, testRange(1, 0, 100, 0), ""),
			testScope("scope:left", "a.ts", ScopeBlock, testRange(5, 0, 10, 0), "scope:mod"),
			testScope("scope:right", "a.ts", ScopeBlock, testRange(10, 0, 15, 0), "scope:mod"),
		})
		requireScopeAt(t, touchingSiblings, "a.ts", 10, 0, "scope:right")
		requireScopeAt(t, touchingSiblings, "a.ts", 7, 0, "scope:left")
		requireScopeAt(t, touchingSiblings, "a.ts", 12, 0, "scope:right")

		multiFile := BuildPositionIndex([]ScopeFact{
			testScope("scope:a-mod", "a.ts", ScopeModule, testRange(1, 0, 50, 0), ""),
			testScope("scope:a-fn", "a.ts", ScopeFunction, testRange(10, 0, 20, 0), "scope:a-mod"),
			testScope("scope:b-mod", "b.ts", ScopeModule, testRange(1, 0, 50, 0), ""),
		})
		if multiFile.Size() != 3 {
			t.Fatalf("multi-file Size() = %d, want 3", multiFile.Size())
		}
		requireScopeAt(t, multiFile, "a.ts", 10, 0, "scope:a-fn")
		requireScopeAt(t, multiFile, "b.ts", 10, 0, "scope:b-mod")

		sameLine := BuildPositionIndex([]ScopeFact{
			testScope("scope:outer", "a.ts", ScopeExpression, testRange(5, 0, 5, 30), ""),
			testScope("scope:inner", "a.ts", ScopeExpression, testRange(5, 10, 5, 20), "scope:outer"),
		})
		requireScopeAt(t, sameLine, "a.ts", 5, 15, "scope:inner")
		requireScopeAt(t, sameLine, "a.ts", 5, 5, "scope:outer")
		requireScopeAt(t, sameLine, "a.ts", 5, 25, "scope:outer")
		requireNoScopeAt(t, sameLine, "a.ts", 5, 31)

		dup := testScope("scope:dup", "a.ts", ScopeModule, testRange(1, 0, 10, 0), "")
		dedup := BuildPositionIndex([]ScopeFact{dup, dup, dup})
		if dedup.Size() != 1 {
			t.Fatalf("dedup Size() = %d, want 1", dedup.Size())
		}
		requireScopeAt(t, dedup, "a.ts", 5, 0, "scope:dup")
	})
}

func TestBuildScopeTreeLegacyParity(t *testing.T) {
	t.Run("empty, single module, nesting, and siblings", func(t *testing.T) {
		empty, err := BuildScopeTree(nil)
		if err != nil {
			t.Fatalf("BuildScopeTree(nil) error = %v", err)
		}
		if empty.Size() != 0 || empty.Has("scope:missing") {
			t.Fatalf("empty tree size/has = %d/%v", empty.Size(), empty.Has("scope:missing"))
		}
		if children := empty.GetChildren("scope:missing"); len(children) != 0 {
			t.Fatalf("missing children = %#v, want empty", children)
		}
		if ancestors := empty.GetAncestors("scope:missing"); len(ancestors) != 0 {
			t.Fatalf("missing ancestors = %#v, want empty", ancestors)
		}

		mod := testScope("scope:m", "src/test.ts", ScopeModule, testRange(1, 0, 50, 0), "")
		cls := testScope("scope:c", "src/test.ts", ScopeClass, testRange(5, 0, 40, 0), "scope:m")
		fn := testScope("scope:f", "src/test.ts", ScopeFunction, testRange(10, 2, 30, 2), "scope:c")
		tree, err := BuildScopeTree([]ScopeFact{mod, cls, fn})
		if err != nil {
			t.Fatalf("BuildScopeTree(nested) error = %v", err)
		}
		if tree.Size() != 3 || !tree.Has("scope:m") {
			t.Fatalf("nested tree size/has = %d/%v", tree.Size(), tree.Has("scope:m"))
		}
		requireParent(t, tree, "scope:f", "scope:c")
		requireParent(t, tree, "scope:c", "scope:m")
		if parent, ok := tree.GetParent("scope:m"); ok {
			t.Fatalf("module parent = %#v, want none", parent)
		}
		requireStringSlice(t, tree.GetChildren("scope:m"), []string{"scope:c"})
		requireStringSlice(t, tree.GetChildren("scope:c"), []string{"scope:f"})
		requireStringSlice(t, tree.GetAncestors("scope:f"), []string{"scope:c", "scope:m"})

		fn1 := testScope("scope:f1", "src/test.ts", ScopeFunction, testRange(5, 0, 10, 0), "scope:m")
		fn2 := testScope("scope:f2", "src/test.ts", ScopeFunction, testRange(15, 0, 20, 0), "scope:m")
		fn3 := testScope("scope:f3", "src/test.ts", ScopeFunction, testRange(25, 0, 30, 0), "scope:m")
		siblings, err := BuildScopeTree([]ScopeFact{mod, fn2, fn1, fn3})
		if err != nil {
			t.Fatalf("BuildScopeTree(siblings) error = %v", err)
		}
		requireStringSlice(t, siblings.GetChildren("scope:m"), []string{"scope:f2", "scope:f1", "scope:f3"})
	})

	t.Run("returned slices are immutable copies", func(t *testing.T) {
		mod := testScope("scope:m", "src/test.ts", ScopeModule, testRange(1, 0, 50, 0), "")
		fn := testScope("scope:f", "src/test.ts", ScopeFunction, testRange(5, 0, 10, 0), "scope:m")
		tree, err := BuildScopeTree([]ScopeFact{mod, fn})
		if err != nil {
			t.Fatalf("BuildScopeTree() error = %v", err)
		}
		children := tree.GetChildren("scope:m")
		children[0] = "x"
		requireStringSlice(t, tree.GetChildren("scope:m"), []string{"scope:f"})

		ancestors := tree.GetAncestors("scope:f")
		ancestors[0] = "x"
		requireStringSlice(t, tree.GetAncestors("scope:f"), []string{"scope:m"})
	})

	t.Run("invariant violations", func(t *testing.T) {
		cases := []struct {
			name      string
			scopes    []ScopeFact
			invariant ScopeTreeInvariant
		}{
			{
				name: "non-module without parent",
				scopes: []ScopeFact{
					testScope("scope:f", "src/test.ts", ScopeFunction, testRange(1, 0, 5, 0), ""),
				},
				invariant: ScopeInvariantNonModuleRequiresParent,
			},
			{
				name: "parent not found",
				scopes: []ScopeFact{
					testScope("scope:f", "src/test.ts", ScopeFunction, testRange(1, 0, 5, 0), "scope:ghost"),
				},
				invariant: ScopeInvariantParentNotFound,
			},
			{
				name: "parent does not contain child",
				scopes: []ScopeFact{
					testScope("scope:m", "src/test.ts", ScopeModule, testRange(1, 0, 10, 0), ""),
					testScope("scope:f", "src/test.ts", ScopeFunction, testRange(5, 0, 50, 0), "scope:m"),
				},
				invariant: ScopeInvariantParentMustContainChild,
			},
			{
				name: "identical range",
				scopes: []ScopeFact{
					testScope("scope:m", "src/test.ts", ScopeModule, testRange(1, 0, 10, 0), ""),
					testScope("scope:f", "src/test.ts", ScopeFunction, testRange(1, 0, 10, 0), "scope:m"),
				},
				invariant: ScopeInvariantParentMustContainChild,
			},
			{
				name: "sibling overlap",
				scopes: []ScopeFact{
					testScope("scope:m", "src/test.ts", ScopeModule, testRange(1, 0, 100, 0), ""),
					testScope("scope:a", "src/test.ts", ScopeFunction, testRange(5, 0, 20, 0), "scope:m"),
					testScope("scope:b", "src/test.ts", ScopeFunction, testRange(15, 0, 30, 0), "scope:m"),
				},
				invariant: ScopeInvariantSiblingRangesOverlap,
			},
			{
				name: "cross file parent",
				scopes: []ScopeFact{
					testScope("scope:m", "a.ts", ScopeModule, testRange(1, 0, 100, 0), ""),
					testScope("scope:f", "b.ts", ScopeFunction, testRange(5, 0, 10, 0), "scope:m"),
				},
				invariant: ScopeInvariantParentMustShareFilePath,
			},
			{
				name: "duplicate scope id",
				scopes: []ScopeFact{
					testScope("scope:dup", "src/test.ts", ScopeModule, testRange(1, 0, 10, 0), ""),
					testScope("scope:dup", "src/test.ts", ScopeModule, testRange(1, 0, 10, 0), ""),
				},
				invariant: ScopeInvariantDuplicateScopeID,
			},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				_, err := BuildScopeTree(tc.scopes)
				requireScopeTreeError(t, err, tc.invariant)
			})
		}

		_, err := BuildScopeTree([]ScopeFact{
			testScope("scope:m", "src/test.ts", ScopeModule, testRange(1, 0, 100, 0), ""),
			testScope("scope:a", "src/test.ts", ScopeBlock, testRange(5, 0, 10, 0), "scope:m"),
			testScope("scope:b", "src/test.ts", ScopeBlock, testRange(10, 0, 15, 0), "scope:m"),
		})
		if err != nil {
			t.Fatalf("touching siblings error = %v, want nil", err)
		}
	})
}

func testDef(id string, returnType string) DefinitionFact {
	return DefinitionFact{
		ID:         id,
		FilePath:   "src/test.ts",
		Label:      NodeMethod,
		ReturnType: returnType,
	}
}

func testRange(startLine int, startCol int, endLine int, endCol int) Range {
	return Range{StartLine: startLine, StartCol: startCol, EndLine: endLine, EndCol: endCol}
}

func testScope(id string, filePath string, kind ScopeKind, scopeRange Range, parent string) ScopeFact {
	var parentPtr *string
	if parent != "" {
		parentPtr = &parent
	}
	return ScopeFact{
		ID:       id,
		Parent:   parentPtr,
		Kind:     kind,
		Range:    scopeRange,
		FilePath: filePath,
	}
}

func requireScopeAt(t *testing.T, idx PositionIndex, filePath string, line int, col int, want string) {
	t.Helper()
	got, ok := idx.AtPosition(filePath, line, col)
	if !ok || got != want {
		t.Fatalf("AtPosition(%q, %d, %d) = %q, %v; want %q true", filePath, line, col, got, ok, want)
	}
}

func requireNoScopeAt(t *testing.T, idx PositionIndex, filePath string, line int, col int) {
	t.Helper()
	got, ok := idx.AtPosition(filePath, line, col)
	if ok {
		t.Fatalf("AtPosition(%q, %d, %d) = %q, true; want no scope", filePath, line, col, got)
	}
}

func requireParent(t *testing.T, tree *ScopeTree, id string, want string) {
	t.Helper()
	parent, ok := tree.GetParent(id)
	if !ok || parent.ID != want {
		t.Fatalf("GetParent(%q) = %#v, %v; want %q true", id, parent, ok, want)
	}
}

func requireStringSlice(t *testing.T, got []string, want []string) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("slice = %#v, want %#v", got, want)
	}
}

func requireScopeTreeError(t *testing.T, err error, want ScopeTreeInvariant) {
	t.Helper()
	var invariantErr *ScopeTreeInvariantError
	if !errors.As(err, &invariantErr) {
		t.Fatalf("error = %v, want ScopeTreeInvariantError", err)
	}
	if invariantErr.Invariant != want {
		t.Fatalf("invariant = %q, want %q", invariantErr.Invariant, want)
	}
}
