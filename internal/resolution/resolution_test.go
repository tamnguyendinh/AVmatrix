package resolution

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"sort"
	"testing"
	"time"

	"github.com/tamnguyendinh/avmatrix-go/internal/graph"
	"github.com/tamnguyendinh/avmatrix-go/internal/parser"
	"github.com/tamnguyendinh/avmatrix-go/internal/providers/tsjs"
	"github.com/tamnguyendinh/avmatrix-go/internal/scanner"
	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

func TestResolveTypeScriptGraphFixture(t *testing.T) {
	irs := parseFixtureWorkspace(t)
	result, err := Resolve(irs, Options{})
	if err != nil {
		t.Fatalf("resolve failed: %v", err)
	}

	serviceSave := requireNode(t, result.Graph, "Method", "src/service.ts", "Service.save")
	format := requireNode(t, result.Graph, "Function", "src/user.ts", "format")
	repoWrite := requireNode(t, result.Graph, "Method", "src/repo.ts", "Repo.write")
	repoClass := requireNode(t, result.Graph, "Class", "src/repo.ts", "Repo")
	modelClass := requireNode(t, result.Graph, "Class", "src/model.ts", "Model")
	userID := requireNode(t, result.Graph, "Property", "src/user.ts", "User.id")
	serviceRepo := requireNode(t, result.Graph, "Property", "src/service.ts", "Service.repo")
	named := requireNode(t, result.Graph, "Interface", "src/contracts.ts", "Named")
	baseService := requireNode(t, result.Graph, "Class", "src/contracts.ts", "BaseService")

	requireRelationship(t, result.Graph, graph.RelCalls, serviceSave.ID, format.ID)
	requireRelationship(t, result.Graph, graph.RelCalls, serviceSave.ID, repoWrite.ID)
	requireRelationship(t, result.Graph, graph.RelCalls, serviceSave.ID, modelClass.ID)
	requireRelationship(t, result.Graph, graph.RelCalls, requireNode(t, result.Graph, "Function", "src/service.ts", "makeRepo").ID, repoClass.ID)
	requireRelationship(t, result.Graph, graph.RelAccesses, serviceSave.ID, userID.ID)
	requireRelationship(t, result.Graph, graph.RelAccesses, serviceSave.ID, serviceRepo.ID)
	requireRelationship(t, result.Graph, graph.RelExtends, requireNode(t, result.Graph, "Class", "src/service.ts", "Service").ID, baseService.ID)
	requireRelationship(t, result.Graph, graph.RelImplements, requireNode(t, result.Graph, "Class", "src/service.ts", "Service").ID, named.ID)
	requireRelationship(t, result.Graph, graph.RelMethodOverrides, requireNode(t, result.Graph, "Class", "src/service.ts", "Service").ID, requireNode(t, result.Graph, "Method", "src/contracts.ts", "BaseService.save").ID)
	requireRelationship(t, result.Graph, graph.RelMethodImplements, serviceSave.ID, requireNode(t, result.Graph, "Method", "src/contracts.ts", "Named.save").ID)

	counts := result.Graph.RelationshipCountsByType()
	for _, relType := range []graph.RelationshipType{
		graph.RelCalls,
		graph.RelImports,
		graph.RelAccesses,
		graph.RelExtends,
		graph.RelImplements,
		graph.RelInherits,
		graph.RelUses,
		graph.RelHasMethod,
		graph.RelHasProperty,
		graph.RelMethodOverrides,
		graph.RelMethodImplements,
	} {
		if counts[relType] == 0 {
			t.Fatalf("expected %s relationships in graph, counts=%v", relType, counts)
		}
	}
	if result.Metrics.ResolvedCalls < 5 || result.Metrics.ResolvedAccesses < 2 || result.Metrics.ResolvedInheritance != 2 {
		t.Fatalf("unexpected metrics: %#v", result.Metrics)
	}
	if len(result.ReferenceIndex.BySourceScope) == 0 || len(result.ReferenceIndex.ByTargetDef) == 0 {
		t.Fatalf("expected reference index to be populated: %#v", result.ReferenceIndex)
	}
}

func TestResolveIntoPreservesExistingFileNodeMetadata(t *testing.T) {
	base := graph.New()
	base.AddNode(graph.Node{
		ID:    graph.GenerateID("File", "README.md"),
		Label: scopeir.NodeFile,
		Properties: graph.NodeProperties{
			"name":         "README.md",
			"filePath":     "README.md",
			"documentKind": "markdown",
		},
	})
	base.AddRelationship(graph.Relationship{
		ID:         graph.GenerateID(string(graph.RelContains), "Folder:docs->File:README.md"),
		SourceID:   "Folder:docs",
		TargetID:   graph.GenerateID("File", "README.md"),
		Type:       graph.RelContains,
		Confidence: 1,
		Reason:     "project structure",
	})
	ir := scopeir.ScopeIR{
		FilePath: "README.md",
		Language: scanner.Markdown,
	}

	result, err := ResolveInto(base, []scopeir.ScopeIR{ir}, Options{})
	if err != nil {
		t.Fatalf("ResolveInto() error = %v", err)
	}
	node, ok := result.Graph.GetNode(graph.GenerateID("File", "README.md"))
	if !ok {
		t.Fatalf("file node missing")
	}
	if node.Properties["documentKind"] != "markdown" || node.Properties["language"] != string(scanner.Markdown) {
		t.Fatalf("file properties = %#v", node.Properties)
	}
	if _, ok := result.Graph.GetRelationship(graph.GenerateID(string(graph.RelContains), "Folder:docs->File:README.md")); !ok {
		t.Fatalf("pre-existing relationship was not preserved")
	}
}

func TestResolveAnnotatesFrameworkHintProperties(t *testing.T) {
	moduleScope := "scope:app/api/users/route.ts#1:0-3:1:Module"
	functionScope := "scope:app/api/users/route.ts#1:0-3:1:Function"
	functionDef := scopeir.DefinitionFact{
		ID:            "def:app/api/users/route.ts#1:0:Function:GET",
		FilePath:      "app/api/users/route.ts",
		Name:          "GET",
		Label:         scopeir.NodeFunction,
		QualifiedName: "GET",
		Range:         scopeir.Range{StartLine: 1, EndLine: 3},
	}
	ir := scopeir.ScopeIR{
		FilePath:    "app/api/users/route.ts",
		FileHash:    "hash-route",
		Language:    scanner.TypeScript,
		ModuleScope: moduleScope,
		Scopes: []scopeir.ScopeFact{
			{ID: moduleScope, Kind: scopeir.ScopeModule, FilePath: "app/api/users/route.ts", Range: scopeir.Range{StartLine: 1, EndLine: 4}},
			{ID: functionScope, Parent: &[]string{moduleScope}[0], Kind: scopeir.ScopeFunction, FilePath: "app/api/users/route.ts", Range: scopeir.Range{StartLine: 1, EndLine: 3}, OwnedDefIDs: []string{functionDef.ID}, Bindings: []scopeir.BindingFact{{Name: "GET", DefID: functionDef.ID, Origin: scopeir.BindingLocal}}},
		},
		Definitions: []scopeir.DefinitionFact{functionDef},
	}

	result, err := Resolve([]scopeir.ScopeIR{ir}, Options{})
	if err != nil {
		t.Fatalf("resolve failed: %v", err)
	}
	fileNode, ok := result.Graph.GetNode(graph.GenerateID("File", "app/api/users/route.ts"))
	if !ok {
		t.Fatalf("file node missing")
	}
	functionNode := requireNode(t, result.Graph, "Function", "app/api/users/route.ts", "GET")
	for _, node := range []graph.Node{fileNode, functionNode} {
		if node.Properties["framework"] != "nextjs-api" || node.Properties["frameworkReason"] != "nextjs-api-route" {
			t.Fatalf("framework properties missing on %s: %#v", node.ID, node.Properties)
		}
		if node.Properties["astFrameworkMultiplier"] != 3.0 || node.Properties["astFrameworkReason"] != "nextjs-api-route" {
			t.Fatalf("process framework properties missing on %s: %#v", node.ID, node.Properties)
		}
	}
}

func TestResolveAppliesScopeIRFrameworkFacts(t *testing.T) {
	moduleScope := "scope:src/users.controller.ts#1:0-4:1:Module"
	classScope := "scope:src/users.controller.ts#2:0-4:1:Class"
	classDef := scopeir.DefinitionFact{
		ID:            "def:src/users.controller.ts#2:0:Class:UsersController",
		FilePath:      "src/users.controller.ts",
		Name:          "UsersController",
		Label:         scopeir.NodeClass,
		QualifiedName: "UsersController",
		Range:         scopeir.Range{StartLine: 2, EndLine: 4},
	}
	ir := scopeir.ScopeIR{
		FilePath:    "src/users.controller.ts",
		FileHash:    "hash-users-controller",
		Language:    scanner.TypeScript,
		ModuleScope: moduleScope,
		Scopes: []scopeir.ScopeFact{
			{ID: moduleScope, Kind: scopeir.ScopeModule, FilePath: "src/users.controller.ts", Range: scopeir.Range{StartLine: 1, EndLine: 4}, OwnedDefIDs: []string{classDef.ID}},
			{ID: classScope, Parent: &[]string{moduleScope}[0], Kind: scopeir.ScopeClass, FilePath: "src/users.controller.ts", Range: scopeir.Range{StartLine: 2, EndLine: 4}, OwnedDefIDs: []string{classDef.ID}},
		},
		Definitions: []scopeir.DefinitionFact{classDef},
		Frameworks: []scopeir.FrameworkFact{{
			DefID:                classDef.ID,
			FilePath:             classDef.FilePath,
			FileHash:             "hash-users-controller",
			Framework:            "nestjs",
			Reason:               "nestjs-decorator",
			EntryPointMultiplier: 3.2,
			Range:                classDef.Range,
		}},
	}

	result, err := Resolve([]scopeir.ScopeIR{ir}, Options{})
	if err != nil {
		t.Fatalf("resolve failed: %v", err)
	}
	classNode := requireNode(t, result.Graph, "Class", "src/users.controller.ts", "UsersController")
	if classNode.Properties["framework"] != "nestjs" || classNode.Properties["frameworkReason"] != "nestjs-decorator" {
		t.Fatalf("framework fact properties missing: %#v", classNode.Properties)
	}
	if classNode.Properties["astFrameworkMultiplier"] != 3.2 || classNode.Properties["astFrameworkReason"] != "nestjs-decorator" {
		t.Fatalf("AST framework fact properties missing: %#v", classNode.Properties)
	}
}

func TestBuildCrossFileBindingFeedsResolveBoundInto(t *testing.T) {
	binding, err := BuildCrossFileBinding(parseFixtureWorkspace(t), Options{})
	if err != nil {
		t.Fatalf("BuildCrossFileBinding() error = %v", err)
	}
	if binding.Metrics.DefinitionsIndexed == 0 || binding.Metrics.ImportsResolved == 0 {
		t.Fatalf("binding metrics missing index/import data: %#v", binding.Metrics)
	}
	if !binding.Metrics.BindingAccumulatorFinalized {
		t.Fatalf("binding accumulator metrics not finalized: %#v", binding.Metrics)
	}

	result, err := ResolveBoundInto(nil, binding, Options{})
	if err != nil {
		t.Fatalf("ResolveBoundInto() error = %v", err)
	}
	if !result.Metrics.BindingAccumulatorDisposed {
		t.Fatalf("binding accumulator was not disposed: %#v", result.Metrics)
	}
	if result.Metrics.ResolvedCalls == 0 || result.Metrics.GraphRelationshipsEmitted == 0 {
		t.Fatalf("resolution metrics missing semantic output: %#v", result.Metrics)
	}
}

func TestResolveExpandsGoPackageImportsToPackageFiles(t *testing.T) {
	targetRaw := "github.com/tamnguyendinh/avmatrix-go/internal/pkg"
	source := scopeir.ScopeIR{
		FilePath:    "cmd/app/main.go",
		Language:    scanner.Go,
		ModuleScope: "scope:cmd/app/main.go#1:0-4:1:Module",
		Imports: []scopeir.ImportFact{{
			FilePath:     "cmd/app/main.go",
			Kind:         scopeir.ImportNamed,
			LocalName:    "pkg",
			ImportedName: "pkg",
			TargetRaw:    &targetRaw,
		}},
	}
	targetA := scopeir.ScopeIR{FilePath: "internal/pkg/a.go", Language: scanner.Go}
	targetB := scopeir.ScopeIR{FilePath: "internal/pkg/b.go", Language: scanner.Go}
	targetTest := scopeir.ScopeIR{FilePath: "internal/pkg/a_test.go", Language: scanner.Go}

	result, err := Resolve([]scopeir.ScopeIR{source, targetTest, targetB, targetA}, Options{})
	if err != nil {
		t.Fatalf("resolve failed: %v", err)
	}
	requireRelationship(t, result.Graph, graph.RelImports, "File:cmd/app/main.go", "File:internal/pkg/a.go")
	requireRelationship(t, result.Graph, graph.RelImports, "File:cmd/app/main.go", "File:internal/pkg/b.go")
	requireNoRelationship(t, result.Graph, graph.RelImports, "File:cmd/app/main.go", "File:internal/pkg/a_test.go")
	if result.Metrics.FinalizedImportsEmitted != 2 || result.Metrics.ImportsResolved != 2 {
		t.Fatalf("import metrics = %#v, want two package-file imports", result.Metrics)
	}
}

func TestResolveTypeScriptGraphSignatureFixture(t *testing.T) {
	result, err := Resolve(parseFixtureWorkspace(t), Options{})
	if err != nil {
		t.Fatalf("resolve failed: %v", err)
	}
	signature := buildGraphSignature(result.Graph)
	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(signature); err != nil {
		t.Fatalf("marshal signature: %v", err)
	}
	golden, err := os.ReadFile("testdata/typescript_graph_signature.golden.json")
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	if string(buffer.Bytes()) != string(golden) {
		t.Fatalf("graph signature mismatch\nwant:\n%s\ngot:\n%s", golden, buffer.Bytes())
	}
}

func TestResolveMergesDuplicateSemanticEdgesAndCountsUnresolved(t *testing.T) {
	moduleScope := "scope:src/app.ts#1:0-3:1:Module"
	functionScope := "scope:src/app.ts#1:0-3:1:Function"
	functionDef := scopeir.DefinitionFact{
		ID:            "def:src/app.ts#1:0:Function:start",
		FilePath:      "src/app.ts",
		Name:          "start",
		Label:         scopeir.NodeFunction,
		QualifiedName: "start",
		Range:         scopeir.Range{StartLine: 1, EndLine: 3},
	}
	targetDef := scopeir.DefinitionFact{
		ID:            "def:src/app.ts#4:0:Function:target",
		FilePath:      "src/app.ts",
		Name:          "target",
		Label:         scopeir.NodeFunction,
		QualifiedName: "target",
		Range:         scopeir.Range{StartLine: 4, EndLine: 4},
	}
	ir := scopeir.ScopeIR{
		FilePath:    "src/app.ts",
		FileHash:    "hash-app",
		Language:    scanner.TypeScript,
		ModuleScope: moduleScope,
		Scopes: []scopeir.ScopeFact{
			{ID: moduleScope, Kind: scopeir.ScopeModule, FilePath: "src/app.ts", Range: scopeir.Range{StartLine: 1, EndLine: 4}},
			{ID: functionScope, Parent: &[]string{moduleScope}[0], Kind: scopeir.ScopeFunction, FilePath: "src/app.ts", Range: scopeir.Range{StartLine: 1, EndLine: 3}, OwnedDefIDs: []string{functionDef.ID}, Bindings: []scopeir.BindingFact{{Name: "start", DefID: functionDef.ID, Origin: scopeir.BindingLocal}, {Name: "target", DefID: targetDef.ID, Origin: scopeir.BindingLocal}, {Name: "aliasTarget", DefID: targetDef.ID, Origin: scopeir.BindingLocal}}},
		},
		Definitions: []scopeir.DefinitionFact{functionDef, targetDef},
		Calls: []scopeir.CallSiteFact{
			{FilePath: "src/app.ts", FileHash: "hash-app", Name: "target", InScope: functionScope, CallForm: scopeir.CallFree, Range: scopeir.Range{StartLine: 2, StartCol: 2}},
			{FilePath: "src/app.ts", FileHash: "hash-app", Name: "target", InScope: functionScope, CallForm: scopeir.CallFree, Range: scopeir.Range{StartLine: 3, StartCol: 2}},
			{FilePath: "src/app.ts", FileHash: "hash-app", Name: "aliasTarget", InScope: functionScope, CallForm: scopeir.CallFree, Range: scopeir.Range{StartLine: 3, StartCol: 6}},
			{FilePath: "src/app.ts", FileHash: "hash-app", Name: "missing", InScope: functionScope, CallForm: scopeir.CallFree, Range: scopeir.Range{StartLine: 3, StartCol: 10}},
		},
	}
	result, err := Resolve([]scopeir.ScopeIR{ir}, Options{})
	if err != nil {
		t.Fatalf("resolve failed: %v", err)
	}
	counts := result.Graph.RelationshipCountsByType()
	if counts[graph.RelCalls] != 2 {
		t.Fatalf("expected same calledName duplicate to merge while alias call remains distinct, got counts=%v", counts)
	}
	if result.Metrics.DuplicateEdgesMerged != 1 || result.Metrics.UnresolvedReferences != 1 {
		t.Fatalf("unexpected duplicate/unresolved metrics: %#v", result.Metrics)
	}
}

func TestDefinitionLookupNamesDeduplicatesTrimmedNames(t *testing.T) {
	def := scopeir.DefinitionFact{
		Name:          " Service ",
		QualifiedName: "pkg.Service",
	}
	names := definitionLookupNames(def)
	want := []string{"Service", "pkg.Service"}
	if len(names) != len(want) {
		t.Fatalf("lookup name count = %d, want %d: %#v", len(names), len(want), names)
	}
	for index := range want {
		if names[index] != want[index] {
			t.Fatalf("lookup name %d = %q, want %q", index, names[index], want[index])
		}
	}
	if !definitionLookupNameMatches(def, "Service") || !definitionLookupNameMatches(def, "pkg.Service") {
		t.Fatalf("expected lookup-name matcher to find simple and qualified names")
	}
	if definitionLookupNameMatches(def, "Other") {
		t.Fatalf("unexpected lookup-name match")
	}
}

func TestUniqueDefsFastPathKeepsFirstCandidate(t *testing.T) {
	first := defRef{Fact: scopeir.DefinitionFact{ID: "same", Name: "first"}}
	duplicate := defRef{Fact: scopeir.DefinitionFact{ID: "same", Name: "duplicate"}}
	other := defRef{Fact: scopeir.DefinitionFact{ID: "other", Name: "other"}}

	one := uniqueDefs([]defRef{first, duplicate})
	if len(one) != 1 || one[0].Fact.Name != "first" {
		t.Fatalf("unique duplicate defs = %#v, want first only", one)
	}
	two := uniqueDefs([]defRef{first, other})
	if len(two) != 2 || two[0].Fact.Name != "first" || two[1].Fact.Name != "other" {
		t.Fatalf("unique distinct defs = %#v, want both in order", two)
	}
}

func TestResolveFallsBackToSameFileCallTargetAndFileCaller(t *testing.T) {
	moduleScope := "scope:src/app.ts#1:0-3:1:Module"
	targetDef := scopeir.DefinitionFact{
		ID:            "def:src/app.ts#4:0:Function:target",
		FilePath:      "src/app.ts",
		Name:          "target",
		Label:         scopeir.NodeFunction,
		QualifiedName: "target",
		Range:         scopeir.Range{StartLine: 4, EndLine: 4},
	}
	otherTargetDef := scopeir.DefinitionFact{
		ID:            "def:src/other.ts#1:0:Function:target",
		FilePath:      "src/other.ts",
		Name:          "target",
		Label:         scopeir.NodeFunction,
		QualifiedName: "target",
		Range:         scopeir.Range{StartLine: 1, EndLine: 1},
	}
	ir := scopeir.ScopeIR{
		FilePath:    "src/app.ts",
		FileHash:    "hash-app",
		Language:    scanner.TypeScript,
		ModuleScope: moduleScope,
		Scopes: []scopeir.ScopeFact{
			{ID: moduleScope, Kind: scopeir.ScopeModule, FilePath: "src/app.ts", Range: scopeir.Range{StartLine: 1, EndLine: 4}},
		},
		Definitions: []scopeir.DefinitionFact{targetDef},
		Calls: []scopeir.CallSiteFact{
			{FilePath: "src/app.ts", FileHash: "hash-app", Name: "target", InScope: moduleScope, CallForm: scopeir.CallFree, Range: scopeir.Range{StartLine: 2, StartCol: 2}},
		},
	}
	otherIR := scopeir.ScopeIR{
		FilePath:    "src/other.ts",
		Language:    scanner.TypeScript,
		ModuleScope: "scope:src/other.ts#1:0-1:1:Module",
		Scopes: []scopeir.ScopeFact{
			{ID: "scope:src/other.ts#1:0-1:1:Module", Kind: scopeir.ScopeModule, FilePath: "src/other.ts", Range: scopeir.Range{StartLine: 1, EndLine: 1}},
		},
		Definitions: []scopeir.DefinitionFact{otherTargetDef},
	}

	result, err := Resolve([]scopeir.ScopeIR{ir, otherIR}, Options{})
	if err != nil {
		t.Fatalf("resolve failed: %v", err)
	}
	relationship := requireRelationship(t, result.Graph, graph.RelCalls, "File:src/app.ts", "Function:src/app.ts:target")
	if relationship.Confidence != 0.95 {
		t.Fatalf("same-file fallback confidence = %v, want 0.95", relationship.Confidence)
	}
	if result.Metrics.ResolvedCalls != 1 || result.Metrics.UnresolvedReferences != 0 {
		t.Fatalf("unexpected metrics: %#v", result.Metrics)
	}
}

func TestResolveGoSamePackageDirectCallAcrossFilesBeforeGlobalAmbiguity(t *testing.T) {
	moduleScope := "scope:internal/pkg/app.go#1:0-5:1:Module"
	functionScope := "scope:internal/pkg/app.go#3:0-5:1:Function"
	runDef := scopeir.DefinitionFact{
		ID:            "def:internal/pkg/app.go#3:0:Function:Run",
		FilePath:      "internal/pkg/app.go",
		Name:          "Run",
		Label:         scopeir.NodeFunction,
		QualifiedName: "Run",
		Range:         scopeir.Range{StartLine: 3, EndLine: 5},
	}
	zeroParams := 0
	samePackageHelper := scopeir.DefinitionFact{
		ID:             "def:internal/pkg/helper.go#1:0:Function:helper",
		FilePath:       "internal/pkg/helper.go",
		Name:           "helper",
		Label:          scopeir.NodeFunction,
		QualifiedName:  "helper",
		Range:          scopeir.Range{StartLine: 1, EndLine: 1},
		ParameterCount: &zeroParams,
	}
	otherPackageHelper := scopeir.DefinitionFact{
		ID:            "def:internal/other/helper.go#1:0:Function:helper",
		FilePath:      "internal/other/helper.go",
		Name:          "helper",
		Label:         scopeir.NodeFunction,
		QualifiedName: "helper",
		Range:         scopeir.Range{StartLine: 1, EndLine: 1},
	}
	source := scopeir.ScopeIR{
		FilePath:    "internal/pkg/app.go",
		FileHash:    "hash-app",
		Language:    scanner.Go,
		ModuleScope: moduleScope,
		Scopes: []scopeir.ScopeFact{
			{ID: moduleScope, Kind: scopeir.ScopeModule, FilePath: "internal/pkg/app.go", Range: scopeir.Range{StartLine: 1, EndLine: 5}},
			{ID: functionScope, Parent: &[]string{moduleScope}[0], Kind: scopeir.ScopeFunction, FilePath: "internal/pkg/app.go", Range: scopeir.Range{StartLine: 3, EndLine: 5}, OwnedDefIDs: []string{runDef.ID}, Bindings: []scopeir.BindingFact{{Name: "Run", DefID: runDef.ID, Origin: scopeir.BindingLocal}}},
		},
		Definitions: []scopeir.DefinitionFact{runDef},
		Calls: []scopeir.CallSiteFact{
			{FilePath: "internal/pkg/app.go", FileHash: "hash-app", Name: "helper", InScope: functionScope, CallForm: scopeir.CallFree, Arity: intPtr(2), Range: scopeir.Range{StartLine: 4, StartCol: 2}},
		},
	}
	samePackage := scopeir.ScopeIR{
		FilePath: "internal/pkg/helper.go",
		Language: scanner.Go,
		Definitions: []scopeir.DefinitionFact{
			samePackageHelper,
		},
	}
	otherPackage := scopeir.ScopeIR{
		FilePath: "internal/other/helper.go",
		Language: scanner.Go,
		Definitions: []scopeir.DefinitionFact{
			otherPackageHelper,
		},
	}

	result, err := Resolve([]scopeir.ScopeIR{source, samePackage, otherPackage}, Options{})
	if err != nil {
		t.Fatalf("resolve failed: %v", err)
	}
	relationship := requireRelationship(t, result.Graph, graph.RelCalls, "Function:internal/pkg/app.go:Run", "Function:internal/pkg/helper.go:helper#0")
	if relationship.Confidence != 0.95 {
		t.Fatalf("same-package fallback confidence = %v, want 0.95", relationship.Confidence)
	}
	requireNoRelationship(t, result.Graph, graph.RelCalls, "Function:internal/pkg/app.go:Run", "Function:internal/other/helper.go:helper")
	if result.Metrics.ResolvedCalls != 1 || result.Metrics.UnresolvedReferences != 0 {
		t.Fatalf("unexpected metrics: %#v", result.Metrics)
	}
}

func TestResolveImportedPackageMemberCall(t *testing.T) {
	targetRaw := "github.com/tamnguyendinh/avmatrix-go/internal/pkg"
	moduleScope := "scope:cmd/app/main.go#1:0-5:1:Module"
	functionScope := "scope:cmd/app/main.go#3:0-5:1:Function"
	mainDef := scopeir.DefinitionFact{
		ID:            "def:cmd/app/main.go#3:0:Function:main",
		FilePath:      "cmd/app/main.go",
		Name:          "main",
		Label:         scopeir.NodeFunction,
		QualifiedName: "main",
		Range:         scopeir.Range{StartLine: 3, EndLine: 5},
	}
	helperDef := scopeir.DefinitionFact{
		ID:            "def:internal/pkg/helper.go#1:0:Function:Helper",
		FilePath:      "internal/pkg/helper.go",
		Name:          "Helper",
		Label:         scopeir.NodeFunction,
		QualifiedName: "Helper",
		Range:         scopeir.Range{StartLine: 1, EndLine: 1},
	}
	source := scopeir.ScopeIR{
		FilePath:    "cmd/app/main.go",
		FileHash:    "hash-main",
		Language:    scanner.Go,
		ModuleScope: moduleScope,
		Scopes: []scopeir.ScopeFact{
			{ID: moduleScope, Kind: scopeir.ScopeModule, FilePath: "cmd/app/main.go", Range: scopeir.Range{StartLine: 1, EndLine: 5}},
			{ID: functionScope, Parent: &[]string{moduleScope}[0], Kind: scopeir.ScopeFunction, FilePath: "cmd/app/main.go", Range: scopeir.Range{StartLine: 3, EndLine: 5}, OwnedDefIDs: []string{mainDef.ID}, Bindings: []scopeir.BindingFact{{Name: "main", DefID: mainDef.ID, Origin: scopeir.BindingLocal}}},
		},
		Definitions: []scopeir.DefinitionFact{mainDef},
		Imports: []scopeir.ImportFact{{
			FilePath:     "cmd/app/main.go",
			Kind:         scopeir.ImportNamed,
			LocalName:    "pkg",
			ImportedName: "pkg",
			TargetRaw:    &targetRaw,
		}},
		Calls: []scopeir.CallSiteFact{
			{FilePath: "cmd/app/main.go", FileHash: "hash-main", Name: "Helper", ExplicitReceiver: "pkg", InScope: functionScope, CallForm: scopeir.CallMember, Range: scopeir.Range{StartLine: 4, StartCol: 2}},
		},
	}
	target := scopeir.ScopeIR{
		FilePath:    "internal/pkg/helper.go",
		Language:    scanner.Go,
		ModuleScope: "scope:internal/pkg/helper.go#1:0-1:1:Module",
		Scopes: []scopeir.ScopeFact{
			{ID: "scope:internal/pkg/helper.go#1:0-1:1:Module", Kind: scopeir.ScopeModule, FilePath: "internal/pkg/helper.go", Range: scopeir.Range{StartLine: 1, EndLine: 1}},
		},
		Definitions: []scopeir.DefinitionFact{helperDef},
	}

	result, err := Resolve([]scopeir.ScopeIR{source, target}, Options{})
	if err != nil {
		t.Fatalf("resolve failed: %v", err)
	}
	relationship := requireRelationship(t, result.Graph, graph.RelCalls, "Function:cmd/app/main.go:main", "Function:internal/pkg/helper.go:Helper")
	if relationship.Confidence != 0.9 {
		t.Fatalf("imported package member confidence = %v, want 0.9", relationship.Confidence)
	}
	if result.Metrics.ResolvedCalls != 1 || result.Metrics.UnresolvedReferences != 0 {
		t.Fatalf("unexpected metrics: %#v", result.Metrics)
	}
}

func TestResolveMemberCallThroughImportedCallReturnBinding(t *testing.T) {
	targetRaw := "github.com/tamnguyendinh/avmatrix-go/internal/graph"
	moduleScope := "scope:cmd/app/main.go#1:0-5:1:Module"
	functionScope := "scope:cmd/app/main.go#2:0-5:1:Function"
	mainDef := scopeir.DefinitionFact{
		ID:             "def:cmd/app/main.go#2:0:Function:main",
		FilePath:       "cmd/app/main.go",
		Name:           "main",
		Label:          scopeir.NodeFunction,
		QualifiedName:  "main",
		Range:          scopeir.Range{StartLine: 2, EndLine: 5},
		ParameterCount: intPtr(0),
	}
	graphVar := scopeir.DefinitionFact{
		ID:            "def:cmd/app/main.go#3:2:Variable:g",
		FilePath:      "cmd/app/main.go",
		Name:          "g",
		Label:         scopeir.NodeVariable,
		QualifiedName: "g",
		Range:         scopeir.Range{StartLine: 3, StartCol: 2, EndLine: 3, EndCol: 18},
	}
	graphDef := scopeir.DefinitionFact{
		ID:            "def:internal/graph/types.go#1:0:Struct:Graph",
		FilePath:      "internal/graph/types.go",
		Name:          "Graph",
		Label:         scopeir.NodeStruct,
		QualifiedName: "Graph",
		Range:         scopeir.Range{StartLine: 1, EndLine: 1},
	}
	newDef := scopeir.DefinitionFact{
		ID:             "def:internal/graph/types.go#2:0:Function:New",
		FilePath:       "internal/graph/types.go",
		Name:           "New",
		Label:          scopeir.NodeFunction,
		QualifiedName:  "New",
		Range:          scopeir.Range{StartLine: 2, EndLine: 2},
		ParameterCount: intPtr(0),
		ReturnType:     "*Graph",
	}
	addNodeDef := scopeir.DefinitionFact{
		ID:             "def:internal/graph/types.go#3:0:Method:AddNode",
		FilePath:       "internal/graph/types.go",
		Name:           "AddNode",
		Label:          scopeir.NodeMethod,
		QualifiedName:  "Graph.AddNode",
		Range:          scopeir.Range{StartLine: 3, EndLine: 3},
		OwnerID:        graphDef.ID,
		ParameterCount: intPtr(1),
	}
	ir := scopeir.ScopeIR{
		FilePath:    "cmd/app/main.go",
		FileHash:    "hash-main",
		Language:    scanner.Go,
		ModuleScope: moduleScope,
		Scopes: []scopeir.ScopeFact{
			{ID: moduleScope, Kind: scopeir.ScopeModule, FilePath: "cmd/app/main.go", Range: scopeir.Range{StartLine: 1, EndLine: 5}},
			{ID: functionScope, Parent: &[]string{moduleScope}[0], Kind: scopeir.ScopeFunction, FilePath: "cmd/app/main.go", Range: scopeir.Range{StartLine: 2, EndLine: 5}, OwnedDefIDs: []string{mainDef.ID, graphVar.ID}, Bindings: []scopeir.BindingFact{{Name: "main", DefID: mainDef.ID, Origin: scopeir.BindingLocal}, {Name: "g", DefID: graphVar.ID, Origin: scopeir.BindingLocal}}},
		},
		Definitions: []scopeir.DefinitionFact{mainDef, graphVar},
		Imports: []scopeir.ImportFact{{
			FilePath:     "cmd/app/main.go",
			Kind:         scopeir.ImportNamed,
			LocalName:    "graph",
			ImportedName: "graph",
			TargetRaw:    &targetRaw,
		}},
		Calls: []scopeir.CallSiteFact{
			{FilePath: "cmd/app/main.go", FileHash: "hash-main", Name: "New", ExplicitReceiver: "graph", InScope: functionScope, CallForm: scopeir.CallMember, Arity: intPtr(0), Range: scopeir.Range{StartLine: 3, StartCol: 7, EndLine: 3, EndCol: 18}},
			{FilePath: "cmd/app/main.go", FileHash: "hash-main", Name: "AddNode", ExplicitReceiver: "g", InScope: functionScope, CallForm: scopeir.CallMember, Arity: intPtr(1), Range: scopeir.Range{StartLine: 4, StartCol: 2, EndLine: 4, EndCol: 16}},
		},
	}
	graphIR := scopeir.ScopeIR{
		FilePath: "internal/graph/types.go",
		Language: scanner.Go,
		Definitions: []scopeir.DefinitionFact{
			{ID: "def:internal/graph/types.go#1:0:Package:graph", FilePath: "internal/graph/types.go", Name: "graph", Label: scopeir.NodePackage, QualifiedName: "graph", Range: scopeir.Range{StartLine: 1, EndLine: 1}},
			graphDef,
			newDef,
			addNodeDef,
		},
	}

	result, err := Resolve([]scopeir.ScopeIR{ir, graphIR}, Options{})
	if err != nil {
		t.Fatalf("resolve failed: %v", err)
	}
	requireRelationship(t, result.Graph, graph.RelCalls, "Function:cmd/app/main.go:main#0", "Function:internal/graph/types.go:New#0")
	requireRelationship(t, result.Graph, graph.RelCalls, "Function:cmd/app/main.go:main#0", "Method:internal/graph/types.go:Graph.AddNode#1")
	if result.Metrics.ResolvedCalls != 2 || result.Metrics.UnresolvedReferences != 0 {
		t.Fatalf("unexpected metrics: %#v", result.Metrics)
	}
}

func TestResolveGlobalCallFallbackUsesArityToAvoidAmbiguity(t *testing.T) {
	moduleScope := "scope:src/app.ts#1:0-3:1:Module"
	functionScope := "scope:src/app.ts#1:0-3:1:Function"
	callerDef := scopeir.DefinitionFact{
		ID:            "def:src/app.ts#1:0:Function:start",
		FilePath:      "src/app.ts",
		Name:          "start",
		Label:         scopeir.NodeFunction,
		QualifiedName: "start",
		Range:         scopeir.Range{StartLine: 1, EndLine: 3},
	}
	oneParam := 1
	twoParams := 2
	targetOne := scopeir.DefinitionFact{
		ID:             "def:src/one.ts#1:0:Function:target",
		FilePath:       "src/one.ts",
		Name:           "target",
		Label:          scopeir.NodeFunction,
		QualifiedName:  "target",
		Range:          scopeir.Range{StartLine: 1, EndLine: 1},
		ParameterCount: &oneParam,
	}
	targetTwo := scopeir.DefinitionFact{
		ID:             "def:src/two.ts#1:0:Function:target",
		FilePath:       "src/two.ts",
		Name:           "target",
		Label:          scopeir.NodeFunction,
		QualifiedName:  "target",
		Range:          scopeir.Range{StartLine: 1, EndLine: 1},
		ParameterCount: &twoParams,
	}
	ir := scopeir.ScopeIR{
		FilePath:    "src/app.ts",
		FileHash:    "hash-app",
		Language:    scanner.TypeScript,
		ModuleScope: moduleScope,
		Scopes: []scopeir.ScopeFact{
			{ID: moduleScope, Kind: scopeir.ScopeModule, FilePath: "src/app.ts", Range: scopeir.Range{StartLine: 1, EndLine: 3}},
			{ID: functionScope, Parent: &[]string{moduleScope}[0], Kind: scopeir.ScopeFunction, FilePath: "src/app.ts", Range: scopeir.Range{StartLine: 1, EndLine: 3}, OwnedDefIDs: []string{callerDef.ID}, Bindings: []scopeir.BindingFact{{Name: "start", DefID: callerDef.ID, Origin: scopeir.BindingLocal}}},
		},
		Definitions: []scopeir.DefinitionFact{callerDef},
		Calls: []scopeir.CallSiteFact{
			{FilePath: "src/app.ts", FileHash: "hash-app", Name: "target", InScope: functionScope, CallForm: scopeir.CallFree, Arity: &oneParam, Range: scopeir.Range{StartLine: 2, StartCol: 2}},
		},
	}
	oneIR := scopeir.ScopeIR{FilePath: "src/one.ts", Language: scanner.TypeScript, Definitions: []scopeir.DefinitionFact{targetOne}}
	twoIR := scopeir.ScopeIR{FilePath: "src/two.ts", Language: scanner.TypeScript, Definitions: []scopeir.DefinitionFact{targetTwo}}

	result, err := Resolve([]scopeir.ScopeIR{ir, oneIR, twoIR}, Options{})
	if err != nil {
		t.Fatalf("resolve failed: %v", err)
	}
	relationship := requireRelationship(t, result.Graph, graph.RelCalls, "Function:src/app.ts:start", "Function:src/one.ts:target#1")
	if relationship.Confidence != 0.5 {
		t.Fatalf("global fallback confidence = %v, want 0.5", relationship.Confidence)
	}
	requireNoRelationship(t, result.Graph, graph.RelCalls, "Function:src/app.ts:start", "Function:src/two.ts:target#2")
	if result.Metrics.ResolvedCalls != 1 || result.Metrics.UnresolvedReferences != 0 {
		t.Fatalf("unexpected metrics: %#v", result.Metrics)
	}
}

func BenchmarkResolveImportedMemberManyImports(b *testing.B) {
	const importCount = 1200
	const sourceFile = "src/main.ts"
	moduleScope := "scope:src/main.ts#1:0-4:1:Module"
	functionScope := "scope:src/main.ts#2:0-4:1:Function"
	mainDef := scopeir.DefinitionFact{
		ID:            "def:src/main.ts#2:0:Function:main",
		FilePath:      sourceFile,
		Name:          "main",
		Label:         scopeir.NodeFunction,
		QualifiedName: "main",
		Range:         scopeir.Range{StartLine: 2, EndLine: 4},
	}
	source := scopeir.ScopeIR{
		FilePath:    sourceFile,
		Language:    scanner.TypeScript,
		ModuleScope: moduleScope,
		Scopes: []scopeir.ScopeFact{
			{ID: moduleScope, Kind: scopeir.ScopeModule, FilePath: sourceFile, Range: scopeir.Range{StartLine: 1, EndLine: 4}},
			{ID: functionScope, Parent: &[]string{moduleScope}[0], Kind: scopeir.ScopeFunction, FilePath: sourceFile, Range: scopeir.Range{StartLine: 2, EndLine: 4}, OwnedDefIDs: []string{mainDef.ID}, Bindings: []scopeir.BindingFact{{Name: "main", DefID: mainDef.ID, Origin: scopeir.BindingLocal}}},
		},
		Definitions: []scopeir.DefinitionFact{mainDef},
		Imports:     make([]scopeir.ImportFact, 0, importCount),
	}
	files := make([]scopeir.ScopeIR, 0, importCount+1)
	files = append(files, source)
	for index := 0; index < importCount; index++ {
		receiver := "pkg" + intString(index)
		targetRaw := "./" + receiver
		targetFile := "src/" + receiver + ".ts"
		files[0].Imports = append(files[0].Imports, scopeir.ImportFact{
			FilePath:     sourceFile,
			Kind:         scopeir.ImportNamespace,
			LocalName:    receiver,
			ImportedName: receiver,
			TargetRaw:    &targetRaw,
		})
		files = append(files, scopeir.ScopeIR{
			FilePath: targetFile,
			Language: scanner.TypeScript,
			Definitions: []scopeir.DefinitionFact{{
				ID:            "def:" + targetFile + "#1:0:Function:Helper",
				FilePath:      targetFile,
				Name:          "Helper",
				Label:         scopeir.NodeFunction,
				QualifiedName: "Helper",
				Range:         scopeir.Range{StartLine: 1, EndLine: 1},
			}},
		})
	}

	w, err := buildWorkspace(files)
	if err != nil {
		b.Fatalf("buildWorkspace failed: %v", err)
	}
	receiver := "pkg" + intString(importCount-1)
	labels := callableLabels()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		target, ok := w.resolveImportedMember(receiver, "Helper", functionScope, labels)
		if !ok || target.Fact.Name != "Helper" {
			b.Fatalf("resolveImportedMember() = %#v, %v", target, ok)
		}
	}
}

func BenchmarkResolveTypeScriptGraphFixture(b *testing.B) {
	irs := parseFixtureWorkspaceForBenchmark(b)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		result, err := Resolve(irs, Options{})
		if err != nil {
			b.Fatalf("resolve failed: %v", err)
		}
		if len(result.Graph.Relationships) == 0 {
			b.Fatalf("empty graph")
		}
	}
}

type graphSignature struct {
	Nodes         []string `json:"nodes"`
	Relationships []string `json:"relationships"`
}

func buildGraphSignature(g *graph.Graph) graphSignature {
	signature := graphSignature{}
	for _, node := range g.Nodes {
		signature.Nodes = append(signature.Nodes, string(node.Label)+":"+node.ID)
	}
	for _, relationship := range g.SortedRelationships() {
		signature.Relationships = append(signature.Relationships,
			string(relationship.Type)+":"+relationship.SourceID+"->"+relationship.TargetID+":"+relationship.Reason,
		)
	}
	sort.Strings(signature.Nodes)
	sort.Strings(signature.Relationships)
	return signature
}

func parseFixtureWorkspace(t *testing.T) []scopeir.ScopeIR {
	t.Helper()
	pool := parser.NewPool(nil, parser.PoolOptions{ParseTimeout: time.Second})
	defer pool.Close()
	return parseFixtureWorkspaceWithPool(t, pool)
}

func parseFixtureWorkspaceForBenchmark(b *testing.B) []scopeir.ScopeIR {
	b.Helper()
	pool := parser.NewPool(nil, parser.PoolOptions{ParseTimeout: time.Second})
	defer pool.Close()
	return parseFixtureWorkspaceWithPool(b, pool)
}

type testingFataler interface {
	Helper()
	Fatalf(string, ...any)
}

func intPtr(value int) *int {
	return &value
}

func parseFixtureWorkspaceWithPool(t testingFataler, pool *parser.Pool) []scopeir.ScopeIR {
	t.Helper()
	sources := map[string]string{
		"src/user.ts":      `export default class User { id: string; } export function format(id: string): string { return id; }`,
		"src/repo.ts":      `export class Repo { write(value: string): void {} }`,
		"src/model.ts":     `export class Model {}`,
		"src/contracts.ts": `import User from './user'; export interface Named { id: string; save(user: User): Promise<void>; } export class BaseService { save(user: User): Promise<void> {} }`,
		"src/service.ts": `import User, { format as fmt } from './user';
import { Repo } from './repo';
import { Model } from './model';
import { BaseService, Named } from './contracts';

function makeRepo(): Repo { return new Repo(); }

class Service extends BaseService implements Named {
  public repo: Repo;
  constructor(repo: Repo) { this.repo = repo; }
  save(user: User): Promise<void> {
    const model = new Model();
    const made = makeRepo();
    const formatted = fmt(user.id);
    this.repo.write(formatted);
  }
}
`,
	}
	paths := []string{"src/user.ts", "src/repo.ts", "src/model.ts", "src/contracts.ts", "src/service.ts"}
	irs := make([]scopeir.ScopeIR, 0, len(paths))
	for _, filePath := range paths {
		source := []byte(sources[filePath])
		parsed, err := pool.Parse(context.Background(), parser.Request{
			FilePath: filePath,
			Language: scanner.TypeScript,
			Source:   source,
		})
		if err != nil {
			t.Fatalf("parse %s failed: %v", filePath, err)
		}
		ir, err := tsjs.Extract(tsjs.Request{
			FilePath: filePath,
			FileHash: "hash-" + filePath,
			Language: scanner.TypeScript,
			Source:   source,
			Root:     parsed.Tree.RootNode(),
		})
		parsed.Close()
		if err != nil {
			t.Fatalf("extract %s failed: %v", filePath, err)
		}
		irs = append(irs, ir)
	}
	return irs
}

func requireNode(t *testing.T, g *graph.Graph, label string, filePath string, qualifiedName string) graph.Node {
	t.Helper()
	for _, node := range g.Nodes {
		if string(node.Label) != label {
			continue
		}
		if node.Properties["filePath"] == filePath && node.Properties["qualifiedName"] == qualifiedName {
			return node
		}
	}
	t.Fatalf("missing node %s %s %s", label, filePath, qualifiedName)
	return graph.Node{}
}

func requireRelationship(t *testing.T, g *graph.Graph, relType graph.RelationshipType, sourceID string, targetID string) graph.Relationship {
	t.Helper()
	for _, relationship := range g.Relationships {
		if relationship.Type == relType && relationship.SourceID == sourceID && relationship.TargetID == targetID {
			return relationship
		}
	}
	t.Fatalf("missing relationship %s %s -> %s", relType, sourceID, targetID)
	return graph.Relationship{}
}

func requireNoRelationship(t *testing.T, g *graph.Graph, relType graph.RelationshipType, sourceID string, targetID string) {
	t.Helper()
	for _, relationship := range g.Relationships {
		if relationship.Type == relType && relationship.SourceID == sourceID && relationship.TargetID == targetID {
			t.Fatalf("unexpected relationship %s %s -> %s", relType, sourceID, targetID)
		}
	}
}
