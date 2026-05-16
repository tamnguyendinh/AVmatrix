package tsjs

import (
	"testing"

	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

func TestScopeIDParityShapeAndDeterminism(t *testing.T) {
	rng := scopeir.Range{StartLine: 1, StartCol: 0, EndLine: 100, EndCol: 0}
	id := scopeID("src/app.ts", rng, scopeir.ScopeModule)
	if id != "scope:src/app.ts#1:0-100:0:Module" {
		t.Fatalf("scopeID() = %q, want canonical shape", id)
	}
	if again := scopeID("src/app.ts", rng, scopeir.ScopeModule); again != id {
		t.Fatalf("scopeID() was not deterministic: %q then %q", id, again)
	}
}

func TestScopeIDParityIncludesScopeKindVerbatim(t *testing.T) {
	rng := scopeir.Range{StartLine: 1, StartCol: 0, EndLine: 2, EndCol: 0}
	kinds := []scopeir.ScopeKind{
		scopeir.ScopeModule,
		scopeir.ScopeNamespace,
		scopeir.ScopeClass,
		scopeir.ScopeFunction,
		scopeir.ScopeBlock,
		scopeir.ScopeExpression,
	}
	for _, kind := range kinds {
		id := scopeID("f.ts", rng, kind)
		want := "scope:f.ts#1:0-2:0:" + string(kind)
		if id != want {
			t.Fatalf("scopeID(%s) = %q, want %q", kind, id, want)
		}
	}
}

func TestScopeIDParityDistinguishesInputs(t *testing.T) {
	module := scopeID("src/a.ts", scopeir.Range{StartLine: 1, StartCol: 0, EndLine: 2, EndCol: 0}, scopeir.ScopeModule)
	otherFile := scopeID("src/b.ts", scopeir.Range{StartLine: 1, StartCol: 0, EndLine: 2, EndCol: 0}, scopeir.ScopeModule)
	otherRange := scopeID("src/a.ts", scopeir.Range{StartLine: 1, StartCol: 0, EndLine: 3, EndCol: 0}, scopeir.ScopeModule)
	otherKind := scopeID("src/a.ts", scopeir.Range{StartLine: 1, StartCol: 0, EndLine: 2, EndCol: 0}, scopeir.ScopeFunction)

	for label, candidate := range map[string]string{
		"file":  otherFile,
		"range": otherRange,
		"kind":  otherKind,
	} {
		if candidate == module {
			t.Fatalf("scopeID did not distinguish %s input: %q", label, candidate)
		}
	}
}
