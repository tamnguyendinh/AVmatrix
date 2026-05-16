package resolution

import (
	"testing"

	"github.com/tamnguyendinh/avmatrix-go/internal/graph"
	"github.com/tamnguyendinh/avmatrix-go/internal/scanner"
	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

func TestLegacyModelResolutionParityCoversTierPrecedenceAndRegistryRouting(t *testing.T) {
	targetRaw := "./b"
	moduleScope := "scope:src/a.ts#1:0-8:1:Module"
	functionScope := "scope:src/a.ts#4:0-8:1:Function"
	localU := scopeir.DefinitionFact{
		ID:            "def:src/a.ts#2:0:Function:U",
		FilePath:      "src/a.ts",
		Name:          "U",
		Label:         scopeir.NodeFunction,
		QualifiedName: "U",
		Range:         scopeir.Range{StartLine: 2, EndLine: 2},
	}
	start := scopeir.DefinitionFact{
		ID:            "def:src/a.ts#4:0:Function:start",
		FilePath:      "src/a.ts",
		Name:          "start",
		Label:         scopeir.NodeFunction,
		QualifiedName: "start",
		Range:         scopeir.Range{StartLine: 4, EndLine: 8},
	}
	importedUser := scopeir.DefinitionFact{
		ID:            "def:src/b.ts#1:0:Class:User",
		FilePath:      "src/b.ts",
		Name:          "User",
		Label:         scopeir.NodeClass,
		QualifiedName: "models.User",
		Range:         scopeir.Range{StartLine: 1, EndLine: 1},
	}
	decoyU := scopeir.DefinitionFact{
		ID:            "def:src/b.ts#2:0:Class:U",
		FilePath:      "src/b.ts",
		Name:          "U",
		Label:         scopeir.NodeClass,
		QualifiedName: "models.U",
		Range:         scopeir.Range{StartLine: 2, EndLine: 2},
	}
	source := scopeir.ScopeIR{
		FilePath:    "src/a.ts",
		FileHash:    "hash-a",
		Language:    scanner.TypeScript,
		ModuleScope: moduleScope,
		Scopes: []scopeir.ScopeFact{
			{
				ID:       moduleScope,
				Kind:     scopeir.ScopeModule,
				FilePath: "src/a.ts",
				Range:    scopeir.Range{StartLine: 1, EndLine: 8},
				Bindings: []scopeir.BindingFact{
					{Name: "U", DefID: localU.ID, Origin: scopeir.BindingLocal},
					{Name: "start", DefID: start.ID, Origin: scopeir.BindingLocal},
				},
			},
			{
				ID:          functionScope,
				Parent:      stringPtr(moduleScope),
				Kind:        scopeir.ScopeFunction,
				FilePath:    "src/a.ts",
				Range:       scopeir.Range{StartLine: 4, EndLine: 8},
				OwnedDefIDs: []string{start.ID},
			},
		},
		Definitions: []scopeir.DefinitionFact{localU, start},
		Imports: []scopeir.ImportFact{{
			FilePath:     "src/a.ts",
			Kind:         scopeir.ImportAlias,
			LocalName:    "U",
			ImportedName: "User",
			TargetRaw:    &targetRaw,
		}},
		Calls: []scopeir.CallSiteFact{{
			FilePath: "src/a.ts",
			FileHash: "hash-a",
			Name:     "U",
			InScope:  functionScope,
			CallForm: scopeir.CallFree,
			Range:    scopeir.Range{StartLine: 5, StartCol: 2, EndLine: 5, EndCol: 5},
		}},
		TypeAnnotations: []scopeir.TypeAnnotationFact{{
			FilePath: "src/a.ts",
			FileHash: "hash-a",
			Name:     "value",
			InScope:  functionScope,
			Type: scopeir.TypeRef{
				RawName:         "U",
				DeclaredAtScope: functionScope,
				Source:          scopeir.TypeSourceAnnotation,
			},
			Range: scopeir.Range{StartLine: 6, StartCol: 15, EndLine: 6, EndCol: 16},
		}},
	}
	target := scopeir.ScopeIR{
		FilePath:    "src/b.ts",
		Language:    scanner.TypeScript,
		ModuleScope: "scope:src/b.ts#1:0-2:1:Module",
		Scopes: []scopeir.ScopeFact{{
			ID:       "scope:src/b.ts#1:0-2:1:Module",
			Kind:     scopeir.ScopeModule,
			FilePath: "src/b.ts",
			Range:    scopeir.Range{StartLine: 1, EndLine: 2},
			Bindings: []scopeir.BindingFact{
				{Name: "User", DefID: importedUser.ID, Origin: scopeir.BindingLocal},
				{Name: "U", DefID: decoyU.ID, Origin: scopeir.BindingLocal},
			},
		}},
		Definitions: []scopeir.DefinitionFact{importedUser, decoyU},
	}

	result, err := Resolve([]scopeir.ScopeIR{source, target}, Options{})
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	requireRelationship(t, result.Graph, graph.RelCalls, "Function:src/a.ts:start", "Function:src/a.ts:U")
	requireRelationship(t, result.Graph, graph.RelUses, "Function:src/a.ts:start", "Class:src/b.ts:models.User")
	requireNoRelationship(t, result.Graph, graph.RelUses, "Function:src/a.ts:start", "Class:src/b.ts:models.U")
	if result.Metrics.ResolvedCalls != 1 || result.Metrics.ResolvedTypeReferences != 1 || result.Metrics.ImportUsesEmitted != 1 {
		t.Fatalf("unexpected metrics: %#v", result.Metrics)
	}
}

func TestLegacyEmitReferencesParityCoversMemberCallsAccessesAndUnresolved(t *testing.T) {
	moduleScope := "scope:src/app.ts#1:0-8:1:Module"
	functionScope := "scope:src/app.ts#4:0-8:1:Function"
	userClass := scopeir.DefinitionFact{
		ID:            "def:src/app.ts#1:0:Class:User",
		FilePath:      "src/app.ts",
		Name:          "User",
		Label:         scopeir.NodeClass,
		QualifiedName: "User",
		Range:         scopeir.Range{StartLine: 1, EndLine: 3},
	}
	nameProperty := scopeir.DefinitionFact{
		ID:            "def:src/app.ts#2:2:Property:name",
		FilePath:      "src/app.ts",
		Name:          "name",
		Label:         scopeir.NodeProperty,
		QualifiedName: "User.name",
		OwnerID:       userClass.ID,
		DeclaredType:  "string",
		Range:         scopeir.Range{StartLine: 2, StartCol: 2, EndLine: 2, EndCol: 15},
	}
	saveMethod := scopeir.DefinitionFact{
		ID:            "def:src/app.ts#3:2:Method:save",
		FilePath:      "src/app.ts",
		Name:          "save",
		Label:         scopeir.NodeMethod,
		QualifiedName: "User.save",
		OwnerID:       userClass.ID,
		Range:         scopeir.Range{StartLine: 3, StartCol: 2, EndLine: 3, EndCol: 20},
	}
	start := scopeir.DefinitionFact{
		ID:            "def:src/app.ts#4:0:Function:start",
		FilePath:      "src/app.ts",
		Name:          "start",
		Label:         scopeir.NodeFunction,
		QualifiedName: "start",
		Range:         scopeir.Range{StartLine: 4, EndLine: 8},
	}
	userVar := scopeir.DefinitionFact{
		ID:            "def:src/app.ts#5:2:Variable:u",
		FilePath:      "src/app.ts",
		Name:          "u",
		Label:         scopeir.NodeVariable,
		QualifiedName: "u",
		DeclaredType:  "User",
		Range:         scopeir.Range{StartLine: 5, StartCol: 2, EndLine: 5, EndCol: 12},
	}
	ir := scopeir.ScopeIR{
		FilePath:    "src/app.ts",
		FileHash:    "hash-app",
		Language:    scanner.TypeScript,
		ModuleScope: moduleScope,
		Scopes: []scopeir.ScopeFact{
			{
				ID:       moduleScope,
				Kind:     scopeir.ScopeModule,
				FilePath: "src/app.ts",
				Range:    scopeir.Range{StartLine: 1, EndLine: 8},
				Bindings: []scopeir.BindingFact{
					{Name: "User", DefID: userClass.ID, Origin: scopeir.BindingLocal},
					{Name: "start", DefID: start.ID, Origin: scopeir.BindingLocal},
				},
				OwnedDefIDs: []string{userClass.ID},
			},
			{
				ID:       functionScope,
				Parent:   stringPtr(moduleScope),
				Kind:     scopeir.ScopeFunction,
				FilePath: "src/app.ts",
				Range:    scopeir.Range{StartLine: 4, EndLine: 8},
				Bindings: []scopeir.BindingFact{
					{Name: "u", DefID: userVar.ID, Origin: scopeir.BindingLocal},
					{Name: "start", DefID: start.ID, Origin: scopeir.BindingLocal},
				},
				OwnedDefIDs: []string{start.ID, userVar.ID},
				TypeBindings: []scopeir.TypeBindingFact{{
					Name: "u",
					Type: scopeir.TypeRef{RawName: "User", DeclaredAtScope: functionScope, Source: scopeir.TypeSourceAnnotation},
				}},
			},
		},
		Definitions: []scopeir.DefinitionFact{userClass, nameProperty, saveMethod, start, userVar},
		Calls: []scopeir.CallSiteFact{
			{FilePath: "src/app.ts", FileHash: "hash-app", Name: "save", ExplicitReceiver: "u", InScope: functionScope, CallForm: scopeir.CallMember, Range: scopeir.Range{StartLine: 6, StartCol: 2, EndLine: 6, EndCol: 10}},
			{FilePath: "src/app.ts", FileHash: "hash-app", Name: "missing", ExplicitReceiver: "u", InScope: functionScope, CallForm: scopeir.CallMember, Range: scopeir.Range{StartLine: 7, StartCol: 2, EndLine: 7, EndCol: 13}},
		},
		Accesses: []scopeir.AccessFact{
			{FilePath: "src/app.ts", FileHash: "hash-app", Name: "name", ExplicitReceiver: "u", InScope: functionScope, Kind: scopeir.AccessRead, Range: scopeir.Range{StartLine: 6, StartCol: 12, EndLine: 6, EndCol: 18}},
			{FilePath: "src/app.ts", FileHash: "hash-app", Name: "name", ExplicitReceiver: "u", InScope: functionScope, Kind: scopeir.AccessWrite, Range: scopeir.Range{StartLine: 7, StartCol: 2, EndLine: 7, EndCol: 8}},
		},
	}

	result, err := Resolve([]scopeir.ScopeIR{ir}, Options{})
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	requireRelationship(t, result.Graph, graph.RelCalls, "Function:src/app.ts:start", "Method:src/app.ts:User.save")
	read := requireRelationshipWithStep(t, result.Graph, graph.RelAccesses, "Function:src/app.ts:start", "Property:src/app.ts:User.name", 1)
	write := requireRelationshipWithStep(t, result.Graph, graph.RelAccesses, "Function:src/app.ts:start", "Property:src/app.ts:User.name", 2)
	if read.Reason != string(ReferenceRead) || write.Reason != string(ReferenceWrite) {
		t.Fatalf("access reasons = %q/%q, want read/write", read.Reason, write.Reason)
	}
	if result.Metrics.ResolvedCalls != 1 || result.Metrics.ResolvedAccesses != 2 || result.Metrics.UnresolvedReferences != 1 {
		t.Fatalf("unexpected metrics: %#v", result.Metrics)
	}
	if got := result.ReferenceIndex.ByTargetDef[saveMethod.ID]; len(got) != 1 || got[0].Kind != ReferenceCall {
		t.Fatalf("call reference index = %#v", got)
	}
	if got := result.ReferenceIndex.BySourceScope[functionScope]; len(got) != 3 {
		t.Fatalf("source reference index length = %d, want 3: %#v", len(got), got)
	}
}

func TestLegacyImportFinalizeParityCoversDefaultNamespaceAliasUnresolvedAndGoPackage(t *testing.T) {
	defaultRaw := "./account"
	namespaceRaw := "./utils"
	aliasRaw := "./repo"
	missingRaw := "./missing"
	goRaw := "github.com/tamnguyendinh/avmatrix-go/internal/pkg"
	moduleScope := "scope:src/service.ts#1:0-6:1:Module"
	functionScope := "scope:src/service.ts#2:0-6:1:Function"
	start := scopeir.DefinitionFact{
		ID:            "def:src/service.ts#2:0:Function:start",
		FilePath:      "src/service.ts",
		Name:          "start",
		Label:         scopeir.NodeFunction,
		QualifiedName: "start",
		Range:         scopeir.Range{StartLine: 2, EndLine: 6},
	}
	source := scopeir.ScopeIR{
		FilePath:    "src/service.ts",
		FileHash:    "hash-service",
		Language:    scanner.TypeScript,
		ModuleScope: moduleScope,
		Scopes: []scopeir.ScopeFact{
			{ID: moduleScope, Kind: scopeir.ScopeModule, FilePath: "src/service.ts", Range: scopeir.Range{StartLine: 1, EndLine: 6}, Bindings: []scopeir.BindingFact{{Name: "start", DefID: start.ID, Origin: scopeir.BindingLocal}}},
			{ID: functionScope, Parent: stringPtr(moduleScope), Kind: scopeir.ScopeFunction, FilePath: "src/service.ts", Range: scopeir.Range{StartLine: 2, EndLine: 6}, OwnedDefIDs: []string{start.ID}, Bindings: []scopeir.BindingFact{{Name: "start", DefID: start.ID, Origin: scopeir.BindingLocal}}},
		},
		Definitions: []scopeir.DefinitionFact{start},
		Imports: []scopeir.ImportFact{
			{FilePath: "src/service.ts", Kind: scopeir.ImportNamed, LocalName: "User", ImportedName: "default", TargetRaw: &defaultRaw},
			{FilePath: "src/service.ts", Kind: scopeir.ImportNamespace, LocalName: "utils", ImportedName: "utils", TargetRaw: &namespaceRaw},
			{FilePath: "src/service.ts", Kind: scopeir.ImportAlias, LocalName: "Repository", ImportedName: "Repo", TargetRaw: &aliasRaw},
			{FilePath: "src/service.ts", Kind: scopeir.ImportNamed, LocalName: "Ghost", ImportedName: "Ghost", TargetRaw: &missingRaw},
		},
		Calls: []scopeir.CallSiteFact{{
			FilePath:         "src/service.ts",
			FileHash:         "hash-service",
			Name:             "format",
			ExplicitReceiver: "utils",
			InScope:          functionScope,
			CallForm:         scopeir.CallMember,
			Range:            scopeir.Range{StartLine: 3, StartCol: 2, EndLine: 3, EndCol: 16},
		}},
		TypeAnnotations: []scopeir.TypeAnnotationFact{{
			FilePath: "src/service.ts",
			FileHash: "hash-service",
			Name:     "repo",
			InScope:  functionScope,
			Type:     scopeir.TypeRef{RawName: "Repository", DeclaredAtScope: functionScope, Source: scopeir.TypeSourceAnnotation},
			Range:    scopeir.Range{StartLine: 4, StartCol: 12, EndLine: 4, EndCol: 22},
		}},
	}
	accountClass := scopeir.DefinitionFact{ID: "def:src/account.ts#1:0:Class:Account", FilePath: "src/account.ts", Name: "Account", Label: scopeir.NodeClass, QualifiedName: "Account", Range: scopeir.Range{StartLine: 1, EndLine: 1}}
	formatFn := scopeir.DefinitionFact{ID: "def:src/utils.ts#1:0:Function:format", FilePath: "src/utils.ts", Name: "format", Label: scopeir.NodeFunction, QualifiedName: "format", Range: scopeir.Range{StartLine: 1, EndLine: 1}}
	repoClass := scopeir.DefinitionFact{ID: "def:src/repo.ts#1:0:Class:Repo", FilePath: "src/repo.ts", Name: "Repo", Label: scopeir.NodeClass, QualifiedName: "Repo", Range: scopeir.Range{StartLine: 1, EndLine: 1}}
	goSource := scopeir.ScopeIR{
		FilePath:    "cmd/app/main.go",
		Language:    scanner.Go,
		ModuleScope: "scope:cmd/app/main.go#1:0-1:1:Module",
		Imports: []scopeir.ImportFact{{
			FilePath:     "cmd/app/main.go",
			Kind:         scopeir.ImportNamed,
			LocalName:    "pkg",
			ImportedName: "pkg",
			TargetRaw:    &goRaw,
		}},
	}

	result, err := Resolve([]scopeir.ScopeIR{
		source,
		{FilePath: "src/account.ts", Language: scanner.TypeScript, Definitions: []scopeir.DefinitionFact{accountClass}},
		{FilePath: "src/utils.ts", Language: scanner.TypeScript, Definitions: []scopeir.DefinitionFact{formatFn}},
		{FilePath: "src/repo.ts", Language: scanner.TypeScript, Definitions: []scopeir.DefinitionFact{repoClass}},
		goSource,
		{FilePath: "internal/pkg/a.go", Language: scanner.Go},
		{FilePath: "internal/pkg/b.go", Language: scanner.Go},
		{FilePath: "internal/pkg/a_test.go", Language: scanner.Go},
	}, Options{})
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	requireRelationship(t, result.Graph, graph.RelImports, "File:src/service.ts", "File:src/account.ts")
	requireRelationship(t, result.Graph, graph.RelImports, "File:src/service.ts", "File:src/utils.ts")
	requireRelationship(t, result.Graph, graph.RelImports, "File:src/service.ts", "File:src/repo.ts")
	requireRelationship(t, result.Graph, graph.RelUses, "File:src/service.ts", "Class:src/account.ts:Account")
	requireRelationship(t, result.Graph, graph.RelUses, "Function:src/service.ts:start", "Class:src/repo.ts:Repo")
	requireRelationship(t, result.Graph, graph.RelCalls, "Function:src/service.ts:start", "Function:src/utils.ts:format")
	requireRelationship(t, result.Graph, graph.RelImports, "File:cmd/app/main.go", "File:internal/pkg/a.go")
	requireRelationship(t, result.Graph, graph.RelImports, "File:cmd/app/main.go", "File:internal/pkg/b.go")
	requireNoRelationship(t, result.Graph, graph.RelImports, "File:cmd/app/main.go", "File:internal/pkg/a_test.go")
	requireNoRelationship(t, result.Graph, graph.RelImports, "File:src/service.ts", "File:src/missing.ts")
	if result.Metrics.ImportsResolved != 5 || result.Metrics.FinalizedImportsEmitted != 5 {
		t.Fatalf("import metrics = %#v, want five resolved imports", result.Metrics)
	}
}

func TestLegacyMethodDispatchParityCoversOverridesAndImplements(t *testing.T) {
	moduleScope := "scope:src/service.ts#1:0-10:1:Module"
	baseScope := "scope:src/service.ts#1:0-3:1:Class"
	namedScope := "scope:src/service.ts#4:0-5:1:Interface"
	serviceScope := "scope:src/service.ts#6:0-10:1:Class"
	baseClass := scopeir.DefinitionFact{ID: "def:src/service.ts#1:0:Class:BaseService", FilePath: "src/service.ts", Name: "BaseService", Label: scopeir.NodeClass, QualifiedName: "BaseService", Range: scopeir.Range{StartLine: 1, EndLine: 3}}
	baseSave := scopeir.DefinitionFact{ID: "def:src/service.ts#2:2:Method:save", FilePath: "src/service.ts", Name: "save", Label: scopeir.NodeMethod, QualifiedName: "BaseService.save", OwnerID: baseClass.ID, ReturnType: "void", Range: scopeir.Range{StartLine: 2, EndLine: 2}}
	namedInterface := scopeir.DefinitionFact{ID: "def:src/service.ts#4:0:Interface:Named", FilePath: "src/service.ts", Name: "Named", Label: scopeir.NodeInterface, QualifiedName: "Named", Range: scopeir.Range{StartLine: 4, EndLine: 5}}
	namedSave := scopeir.DefinitionFact{ID: "def:src/service.ts#5:2:Method:save", FilePath: "src/service.ts", Name: "save", Label: scopeir.NodeMethod, QualifiedName: "Named.save", OwnerID: namedInterface.ID, ReturnType: "void", Range: scopeir.Range{StartLine: 5, EndLine: 5}}
	serviceClass := scopeir.DefinitionFact{ID: "def:src/service.ts#6:0:Class:Service", FilePath: "src/service.ts", Name: "Service", Label: scopeir.NodeClass, QualifiedName: "Service", Range: scopeir.Range{StartLine: 6, EndLine: 10}}
	serviceSave := scopeir.DefinitionFact{ID: "def:src/service.ts#8:2:Method:save", FilePath: "src/service.ts", Name: "save", Label: scopeir.NodeMethod, QualifiedName: "Service.save", OwnerID: serviceClass.ID, ReturnType: "void", Range: scopeir.Range{StartLine: 8, EndLine: 8}}
	ir := scopeir.ScopeIR{
		FilePath:    "src/service.ts",
		Language:    scanner.TypeScript,
		ModuleScope: moduleScope,
		Scopes: []scopeir.ScopeFact{
			{ID: moduleScope, Kind: scopeir.ScopeModule, FilePath: "src/service.ts", Range: scopeir.Range{StartLine: 1, EndLine: 10}, Bindings: []scopeir.BindingFact{{Name: "BaseService", DefID: baseClass.ID, Origin: scopeir.BindingLocal}, {Name: "Named", DefID: namedInterface.ID, Origin: scopeir.BindingLocal}, {Name: "Service", DefID: serviceClass.ID, Origin: scopeir.BindingLocal}}},
			{ID: baseScope, Parent: stringPtr(moduleScope), Kind: scopeir.ScopeClass, FilePath: "src/service.ts", Range: scopeir.Range{StartLine: 1, EndLine: 3}, OwnedDefIDs: []string{baseClass.ID}},
			{ID: namedScope, Parent: stringPtr(moduleScope), Kind: scopeir.ScopeClass, FilePath: "src/service.ts", Range: scopeir.Range{StartLine: 4, EndLine: 5}, OwnedDefIDs: []string{namedInterface.ID}},
			{ID: serviceScope, Parent: stringPtr(moduleScope), Kind: scopeir.ScopeClass, FilePath: "src/service.ts", Range: scopeir.Range{StartLine: 6, EndLine: 10}, OwnedDefIDs: []string{serviceClass.ID}},
		},
		Definitions: []scopeir.DefinitionFact{baseClass, baseSave, namedInterface, namedSave, serviceClass, serviceSave},
		Heritage: []scopeir.HeritageFact{
			{FilePath: "src/service.ts", Name: "BaseService", Kind: scopeir.HeritageExtends, InScope: serviceScope, Range: scopeir.Range{StartLine: 6, StartCol: 22, EndLine: 6, EndCol: 33}},
			{FilePath: "src/service.ts", Name: "Named", Kind: scopeir.HeritageImplements, InScope: serviceScope, Range: scopeir.Range{StartLine: 6, StartCol: 45, EndLine: 6, EndCol: 50}},
		},
	}

	result, err := Resolve([]scopeir.ScopeIR{ir}, Options{})
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	requireRelationship(t, result.Graph, graph.RelExtends, "Class:src/service.ts:Service", "Class:src/service.ts:BaseService")
	requireRelationship(t, result.Graph, graph.RelImplements, "Class:src/service.ts:Service", "Interface:src/service.ts:Named")
	requireRelationship(t, result.Graph, graph.RelInherits, "Class:src/service.ts:Service", "Class:src/service.ts:BaseService")
	requireRelationship(t, result.Graph, graph.RelInherits, "Class:src/service.ts:Service", "Interface:src/service.ts:Named")
	requireRelationship(t, result.Graph, graph.RelMethodOverrides, "Class:src/service.ts:Service", "Method:src/service.ts:BaseService.save")
	requireRelationship(t, result.Graph, graph.RelMethodImplements, "Method:src/service.ts:Service.save", "Method:src/service.ts:Named.save")
	if result.Metrics.ResolvedInheritance != 2 || result.Metrics.MethodOverridesEmitted != 1 || result.Metrics.MethodImplementsEmitted != 1 {
		t.Fatalf("unexpected metrics: %#v", result.Metrics)
	}
}

func requireRelationshipWithStep(t *testing.T, g *graph.Graph, relType graph.RelationshipType, sourceID string, targetID string, step int) graph.Relationship {
	t.Helper()
	for _, relationship := range g.Relationships {
		if relationship.Type != relType || relationship.SourceID != sourceID || relationship.TargetID != targetID {
			continue
		}
		if relationship.Step != nil && *relationship.Step == step {
			return relationship
		}
	}
	t.Fatalf("missing relationship %s %s -> %s with step %d", relType, sourceID, targetID, step)
	return graph.Relationship{}
}

func stringPtr(value string) *string {
	return &value
}
