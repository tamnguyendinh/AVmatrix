package resolution

import (
	"testing"

	"github.com/tamnguyendinh/avmatrix-go/internal/graph"
	"github.com/tamnguyendinh/avmatrix-go/internal/scanner"
	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

func TestLegacyP7TypeExtractorParityCoversResolutionTypeNames(t *testing.T) {
	tests := map[string]string{
		"User":              "User",
		"User | null":       "User",
		"User | undefined":  "User",
		"*User":             "User",
		"&User":             "User",
		"[]User":            "User",
		"User[]":            "User",
		"models.User":       "User",
		"Promise<User>":     "Promise",
		"Map<string, User>": "Map",
	}
	for raw, want := range tests {
		if got := baseTypeName(raw); got != want {
			t.Fatalf("baseTypeName(%q) = %q, want %q", raw, got, want)
		}
	}
}

func TestLegacyP7CallProcessorParityCoversConstructorReturnAndAccumulatorBinding(t *testing.T) {
	appFile := "src/app.ts"
	modelsFile := "src/models.ts"
	moduleScope := "scope:src/app.ts#1:0-9:0:Module"
	functionScope := "scope:src/app.ts#3:0-9:0:Function"
	modelsScope := "scope:src/models.ts#1:0-5:0:Module"
	modelsRaw := "./models"

	runDef := scopeir.DefinitionFact{ID: "def:src/app.ts#3:0:Function:run", FilePath: appFile, Name: "run", Label: scopeir.NodeFunction, QualifiedName: "run", Range: scopeir.Range{StartLine: 3, EndLine: 9}}
	firstVar := scopeir.DefinitionFact{ID: "def:src/app.ts#4:2:Variable:first", FilePath: appFile, Name: "first", Label: scopeir.NodeVariable, QualifiedName: "first", Range: scopeir.Range{StartLine: 4, StartCol: 2, EndLine: 4, EndCol: 28}}
	secondVar := scopeir.DefinitionFact{ID: "def:src/app.ts#5:2:Variable:second", FilePath: appFile, Name: "second", Label: scopeir.NodeVariable, QualifiedName: "second", Range: scopeir.Range{StartLine: 5, StartCol: 2, EndLine: 5, EndCol: 29}}
	userDef := scopeir.DefinitionFact{ID: "def:src/models.ts#1:0:Class:User", FilePath: modelsFile, Name: "User", Label: scopeir.NodeClass, QualifiedName: "User", Range: scopeir.Range{StartLine: 1, EndLine: 4}}
	makeUserDef := scopeir.DefinitionFact{ID: "def:src/models.ts#2:0:Function:makeUser", FilePath: modelsFile, Name: "makeUser", Label: scopeir.NodeFunction, QualifiedName: "makeUser", ReturnType: "User", Range: scopeir.Range{StartLine: 2, EndLine: 2}}
	saveDef := scopeir.DefinitionFact{ID: "def:src/models.ts#3:2:Method:save", FilePath: modelsFile, Name: "save", Label: scopeir.NodeMethod, QualifiedName: "User.save", OwnerID: userDef.ID, Range: scopeir.Range{StartLine: 3, EndLine: 3}}

	source := scopeir.ScopeIR{
		FilePath:    appFile,
		FileHash:    "hash-app",
		Language:    scanner.TypeScript,
		ModuleScope: moduleScope,
		Scopes: []scopeir.ScopeFact{
			{ID: moduleScope, Kind: scopeir.ScopeModule, FilePath: appFile, Range: scopeir.Range{StartLine: 1, EndLine: 9}, Bindings: []scopeir.BindingFact{{Name: "run", DefID: runDef.ID, Origin: scopeir.BindingLocal}}},
			{ID: functionScope, Parent: stringPtr(moduleScope), Kind: scopeir.ScopeFunction, FilePath: appFile, Range: scopeir.Range{StartLine: 3, EndLine: 9}, OwnedDefIDs: []string{runDef.ID, firstVar.ID, secondVar.ID}, Bindings: []scopeir.BindingFact{{Name: "first", DefID: firstVar.ID, Origin: scopeir.BindingLocal}, {Name: "second", DefID: secondVar.ID, Origin: scopeir.BindingLocal}}},
		},
		Definitions: []scopeir.DefinitionFact{runDef, firstVar, secondVar},
		Imports:     []scopeir.ImportFact{{FilePath: appFile, Kind: scopeir.ImportNamed, LocalName: "makeUser", ImportedName: "makeUser", TargetRaw: &modelsRaw}, {FilePath: appFile, Kind: scopeir.ImportNamed, LocalName: "User", ImportedName: "User", TargetRaw: &modelsRaw}},
		Calls: []scopeir.CallSiteFact{
			{FilePath: appFile, FileHash: "hash-app", Name: "makeUser", InScope: functionScope, CallForm: scopeir.CallFree, Range: scopeir.Range{StartLine: 4, StartCol: 16, EndLine: 4, EndCol: 26}},
			{FilePath: appFile, FileHash: "hash-app", Name: "User", InScope: functionScope, CallForm: scopeir.CallConstructor, Range: scopeir.Range{StartLine: 5, StartCol: 17, EndLine: 5, EndCol: 27}},
			{FilePath: appFile, FileHash: "hash-app", Name: "save", ExplicitReceiver: "first", InScope: functionScope, CallForm: scopeir.CallMember, Range: scopeir.Range{StartLine: 6, StartCol: 2, EndLine: 6, EndCol: 14}},
			{FilePath: appFile, FileHash: "hash-app", Name: "save", ExplicitReceiver: "second", InScope: functionScope, CallForm: scopeir.CallMember, Range: scopeir.Range{StartLine: 7, StartCol: 2, EndLine: 7, EndCol: 15}},
		},
	}
	models := scopeir.ScopeIR{
		FilePath:    modelsFile,
		Language:    scanner.TypeScript,
		ModuleScope: modelsScope,
		Scopes:      []scopeir.ScopeFact{{ID: modelsScope, Kind: scopeir.ScopeModule, FilePath: modelsFile, Range: scopeir.Range{StartLine: 1, EndLine: 5}, Bindings: []scopeir.BindingFact{{Name: "User", DefID: userDef.ID, Origin: scopeir.BindingLocal}, {Name: "makeUser", DefID: makeUserDef.ID, Origin: scopeir.BindingLocal}}}},
		Definitions: []scopeir.DefinitionFact{userDef, makeUserDef, saveDef},
	}

	binding, err := BuildCrossFileBinding([]scopeir.ScopeIR{source, models}, Options{})
	if err != nil {
		t.Fatalf("BuildCrossFileBinding() error = %v", err)
	}
	if got, ok := binding.workspace.lookupTypeBinding("first", functionScope); !ok || got.RawName != "User" {
		t.Fatalf("return-call type binding for first = %#v, %v; want User", got, ok)
	}
	if got, ok := binding.workspace.lookupTypeBinding("second", functionScope); !ok || got.RawName != "User" {
		t.Fatalf("constructor type binding for second = %#v, %v; want User", got, ok)
	}

	result, err := ResolveBoundInto(nil, binding, Options{})
	if err != nil {
		t.Fatalf("ResolveBoundInto() error = %v", err)
	}
	requireRelationship(t, result.Graph, graph.RelCalls, "Function:src/app.ts:run", "Function:src/models.ts:makeUser")
	requireRelationship(t, result.Graph, graph.RelCalls, "Function:src/app.ts:run", "Class:src/models.ts:User")
	saveRel := requireRelationship(t, result.Graph, graph.RelCalls, "Function:src/app.ts:run", "Method:src/models.ts:User.save")
	if saveRel.Confidence != 1 {
		t.Fatalf("member call confidence = %v, want 1", saveRel.Confidence)
	}
	if result.Metrics.ResolvedCalls != 4 || !result.Metrics.BindingAccumulatorFinalized || !result.Metrics.BindingAccumulatorDisposed {
		t.Fatalf("unexpected metrics: %#v", result.Metrics)
	}
}
