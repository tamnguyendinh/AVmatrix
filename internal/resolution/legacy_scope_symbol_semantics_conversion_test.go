package resolution

import (
	"testing"

	"github.com/tamnguyendinh/avmatrix-go/internal/graph"
	"github.com/tamnguyendinh/avmatrix-go/internal/scanner"
	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

func TestLegacyScopeSymbolConversionMaterializesImportBindingsAndCycles(t *testing.T) {
	modelsRaw := "./models"
	auditRaw := "./audit"
	bRaw := "./b"
	aRaw := "./a"

	sourceModule := "scope:src/app.ts#1:0-8:0:Module"
	modelsModule := "scope:src/models.ts#1:0-4:0:Module"
	auditModule := "scope:src/audit.ts#1:0-2:0:Module"
	aModule := "scope:src/a.ts#1:0-2:0:Module"
	bModule := "scope:src/b.ts#1:0-2:0:Module"

	userDef := legacyScopeSymbolDef("def:src/models.ts#1:0:Class:User", "src/models.ts", "User", scopeir.NodeClass, "models.User")
	repoDef := legacyScopeSymbolDef("def:src/models.ts#2:0:Class:Repo", "src/models.ts", "Repo", scopeir.NodeClass, "models.Repo")
	helperDef := legacyScopeSymbolDef("def:src/models.ts#3:0:Function:Helper", "src/models.ts", "Helper", scopeir.NodeFunction, "models.Helper")
	auditDef := legacyScopeSymbolDef("def:src/audit.ts#1:0:Class:Audit", "src/audit.ts", "Audit", scopeir.NodeClass, "audit.Audit")
	xDef := legacyScopeSymbolDef("def:src/a.ts#1:0:Class:X", "src/a.ts", "X", scopeir.NodeClass, "a.X")
	yDef := legacyScopeSymbolDef("def:src/b.ts#1:0:Class:Y", "src/b.ts", "Y", scopeir.NodeClass, "b.Y")

	files := []scopeir.ScopeIR{
		{
			FilePath:    "src/app.ts",
			Language:    scanner.TypeScript,
			ModuleScope: sourceModule,
			Scopes: []scopeir.ScopeFact{legacyScopeSymbolModuleScope(sourceModule, "src/app.ts",
				scopeir.BindingFact{Name: "LocalOnly", DefID: "def:src/app.ts#1:0:Class:LocalOnly", Origin: scopeir.BindingLocal},
			)},
			Definitions: []scopeir.DefinitionFact{
				legacyScopeSymbolDef("def:src/app.ts#1:0:Class:LocalOnly", "src/app.ts", "LocalOnly", scopeir.NodeClass, "LocalOnly"),
			},
			Imports: []scopeir.ImportFact{
				{FilePath: "src/app.ts", Kind: scopeir.ImportAlias, LocalName: "Account", ImportedName: "User", Alias: "Account", TargetRaw: &modelsRaw},
				{FilePath: "src/app.ts", Kind: scopeir.ImportNamed, LocalName: "Repo", ImportedName: "Repo", TargetRaw: &modelsRaw},
				{FilePath: "src/app.ts", Kind: scopeir.ImportWildcardExpanded, LocalName: "Helper", ImportedName: "Helper", TargetRaw: &modelsRaw},
				{FilePath: "src/app.ts", Kind: scopeir.ImportReexport, LocalName: "AuditLog", ImportedName: "Audit", Alias: "AuditLog", TargetRaw: &auditRaw},
			},
		},
		{
			FilePath:    "src/models.ts",
			Language:    scanner.TypeScript,
			ModuleScope: modelsModule,
			Scopes:      []scopeir.ScopeFact{legacyScopeSymbolModuleScope(modelsModule, "src/models.ts", legacyScopeSymbolBinding("User", userDef), legacyScopeSymbolBinding("Repo", repoDef), legacyScopeSymbolBinding("Helper", helperDef))},
			Definitions: []scopeir.DefinitionFact{userDef, repoDef, helperDef},
		},
		{
			FilePath:    "src/audit.ts",
			Language:    scanner.TypeScript,
			ModuleScope: auditModule,
			Scopes:      []scopeir.ScopeFact{legacyScopeSymbolModuleScope(auditModule, "src/audit.ts", legacyScopeSymbolBinding("Audit", auditDef))},
			Definitions: []scopeir.DefinitionFact{auditDef},
		},
		{
			FilePath:    "src/a.ts",
			Language:    scanner.TypeScript,
			ModuleScope: aModule,
			Scopes:      []scopeir.ScopeFact{legacyScopeSymbolModuleScope(aModule, "src/a.ts", legacyScopeSymbolBinding("X", xDef))},
			Definitions: []scopeir.DefinitionFact{xDef},
			Imports:     []scopeir.ImportFact{{FilePath: "src/a.ts", Kind: scopeir.ImportNamed, LocalName: "Y", ImportedName: "Y", TargetRaw: &bRaw}},
		},
		{
			FilePath:    "src/b.ts",
			Language:    scanner.TypeScript,
			ModuleScope: bModule,
			Scopes:      []scopeir.ScopeFact{legacyScopeSymbolModuleScope(bModule, "src/b.ts", legacyScopeSymbolBinding("Y", yDef))},
			Definitions: []scopeir.DefinitionFact{yDef},
			Imports:     []scopeir.ImportFact{{FilePath: "src/b.ts", Kind: scopeir.ImportNamed, LocalName: "X", ImportedName: "X", TargetRaw: &aRaw}},
		},
	}

	workspace, err := buildWorkspace(files)
	if err != nil {
		t.Fatalf("buildWorkspace() error = %v", err)
	}

	legacyScopeSymbolRequireBinding(t, workspace, sourceModule, "Account", scopeir.BindingImport, userDef.ID)
	legacyScopeSymbolRequireBinding(t, workspace, sourceModule, "Repo", scopeir.BindingImport, repoDef.ID)
	legacyScopeSymbolRequireBinding(t, workspace, sourceModule, "Helper", scopeir.BindingWildcard, helperDef.ID)
	legacyScopeSymbolRequireBinding(t, workspace, sourceModule, "AuditLog", scopeir.BindingReexport, auditDef.ID)
	legacyScopeSymbolRequireBinding(t, workspace, aModule, "Y", scopeir.BindingImport, yDef.ID)
	legacyScopeSymbolRequireBinding(t, workspace, bModule, "X", scopeir.BindingImport, xDef.ID)

	result, err := Resolve(files, Options{})
	if err != nil {
		t.Fatalf("Resolve() cyclic import workspace error = %v", err)
	}
	requireRelationship(t, result.Graph, graph.RelImports, "File:src/a.ts", "File:src/b.ts")
	requireRelationship(t, result.Graph, graph.RelImports, "File:src/b.ts", "File:src/a.ts")
	if result.Metrics.ImportsResolved != 6 || result.Metrics.FinalizedImportsEmitted != 6 {
		t.Fatalf("import metrics = %#v, want 6 resolved/finalized imports", result.Metrics)
	}
}

func TestLegacyScopeSymbolConversionResolvesTiersOwnersAndArity(t *testing.T) {
	loggerRaw := "./logger"
	moduleScope := "scope:src/app.ts#1:0-8:0:Module"
	functionScope := "scope:src/app.ts#4:0-8:0:Function"
	loggerModule := "scope:src/logger.ts#1:0-2:0:Module"
	testingModule := "scope:src/testing/logger.ts#1:0-2:0:Module"

	runDef := legacyScopeSymbolDef("def:src/app.ts#4:0:Function:run", "src/app.ts", "run", scopeir.NodeFunction, "run")
	userDef := legacyScopeSymbolDef("def:src/app.ts#1:0:Class:User", "src/app.ts", "User", scopeir.NodeClass, "User")
	saveDef := legacyScopeSymbolDef("def:src/app.ts#2:2:Method:save", "src/app.ts", "save", scopeir.NodeMethod, "User.save")
	saveDef.OwnerID = userDef.ID
	saveDef.ReturnType = "bool"
	nameDef := legacyScopeSymbolDef("def:src/app.ts#3:2:Property:name", "src/app.ts", "name", scopeir.NodeProperty, "User.name")
	nameDef.OwnerID = userDef.ID
	nameDef.DeclaredType = "string"
	localHelper := legacyScopeSymbolDef("def:src/app.ts#5:0:Function:helper", "src/app.ts", "helper", scopeir.NodeFunction, "helper")
	oneParam := 1
	twoParams := 2
	localHelper.ParameterCount = &oneParam
	otherHelper := legacyScopeSymbolDef("def:src/other.ts#1:0:Function:helper", "src/other.ts", "helper", scopeir.NodeFunction, "helper")
	otherHelper.ParameterCount = &twoParams
	importedLogger := legacyScopeSymbolDef("def:src/logger.ts#1:0:Class:Logger", "src/logger.ts", "Logger", scopeir.NodeClass, "logger.Logger")
	globalLogger := legacyScopeSymbolDef("def:src/testing/logger.ts#1:0:Class:Logger", "src/testing/logger.ts", "Logger", scopeir.NodeClass, "testing.Logger")

	workspace, err := buildWorkspace([]scopeir.ScopeIR{
		{
			FilePath:    "src/app.ts",
			Language:    scanner.TypeScript,
			ModuleScope: moduleScope,
			Scopes: []scopeir.ScopeFact{
				legacyScopeSymbolModuleScope(moduleScope, "src/app.ts", legacyScopeSymbolBinding("run", runDef), legacyScopeSymbolBinding("User", userDef), legacyScopeSymbolBinding("helper", localHelper)),
				{ID: functionScope, Parent: stringPtr(moduleScope), Kind: scopeir.ScopeFunction, FilePath: "src/app.ts", Range: scopeir.Range{StartLine: 4, EndLine: 8}, OwnedDefIDs: []string{runDef.ID}},
			},
			Definitions: []scopeir.DefinitionFact{runDef, userDef, saveDef, nameDef, localHelper},
			Imports:     []scopeir.ImportFact{{FilePath: "src/app.ts", Kind: scopeir.ImportNamed, LocalName: "Logger", ImportedName: "Logger", TargetRaw: &loggerRaw}},
		},
		{
			FilePath:    "src/logger.ts",
			Language:    scanner.TypeScript,
			ModuleScope: loggerModule,
			Scopes:      []scopeir.ScopeFact{legacyScopeSymbolModuleScope(loggerModule, "src/logger.ts", legacyScopeSymbolBinding("Logger", importedLogger))},
			Definitions: []scopeir.DefinitionFact{importedLogger},
		},
		{
			FilePath:    "src/testing/logger.ts",
			Language:    scanner.TypeScript,
			ModuleScope: testingModule,
			Scopes:      []scopeir.ScopeFact{legacyScopeSymbolModuleScope(testingModule, "src/testing/logger.ts", legacyScopeSymbolBinding("Logger", globalLogger))},
			Definitions: []scopeir.DefinitionFact{globalLogger},
		},
		{
			FilePath:    "src/other.ts",
			Language:    scanner.TypeScript,
			ModuleScope: "scope:src/other.ts#1:0-1:0:Module",
			Scopes:      []scopeir.ScopeFact{legacyScopeSymbolModuleScope("scope:src/other.ts#1:0-1:0:Module", "src/other.ts", legacyScopeSymbolBinding("helper", otherHelper))},
			Definitions: []scopeir.DefinitionFact{otherHelper},
		},
	})
	if err != nil {
		t.Fatalf("buildWorkspace() error = %v", err)
	}

	importScoped, ok := workspace.resolveName("Logger", functionScope, dispatchOwnerLabels())
	if !ok || importScoped.Fact.ID != importedLogger.ID {
		t.Fatalf("resolveName(Logger) = %#v, %v; want imported logger", importScoped, ok)
	}
	if _, ok := workspace.resolveGlobalName("Logger", dispatchOwnerLabels()); ok {
		t.Fatalf("resolveGlobalName(Logger) unexpectedly resolved ambiguous global")
	}
	if got, ok := workspace.resolveOwnedMember(userDef.ID, "save", callableLabels()); !ok || got.Fact.ID != saveDef.ID {
		t.Fatalf("resolveOwnedMember(save) = %#v, %v; want save method", got, ok)
	}
	if got, ok := workspace.resolveOwnedMember(userDef.ID, "name", propertyLabels()); !ok || got.Fact.ID != nameDef.ID {
		t.Fatalf("resolveOwnedMember(name) = %#v, %v; want name property", got, ok)
	}
	if got, ok := workspace.resolveGlobalCallName("helper", callableLabels(), &oneParam); !ok || got.Fact.ID != localHelper.ID {
		t.Fatalf("resolveGlobalCallName(helper/1) = %#v, %v; want one-arg helper", got, ok)
	}
	if got := workspace.defsByID[nameDef.ID].Fact.DeclaredType; got != "string" {
		t.Fatalf("declaredType metadata = %q, want string", got)
	}
}

func TestLegacyScopeSymbolConversionResolvesTypeReferencesThroughGraph(t *testing.T) {
	modelsRaw := "./models"
	moduleScope := "scope:src/app.ts#1:0-10:0:Module"
	functionScope := "scope:src/app.ts#3:0-8:0:Function"
	modelsModule := "scope:src/models.ts#1:0-4:0:Module"
	otherModule := "scope:src/other.ts#1:0-2:0:Module"

	saveDef := legacyScopeSymbolDef("def:src/app.ts#3:0:Function:save", "src/app.ts", "save", scopeir.NodeFunction, "save")
	userDef := legacyScopeSymbolDef("def:src/models.ts#1:0:Class:User", "src/models.ts", "User", scopeir.NodeClass, "models.User")
	repoDef := legacyScopeSymbolDef("def:src/models.ts#2:0:Class:Repo", "src/models.ts", "Repo", scopeir.NodeClass, "models.Repo")
	ambiguousA := legacyScopeSymbolDef("def:src/models.ts#3:0:Class:Duplicate", "src/models.ts", "Duplicate", scopeir.NodeClass, "models.Duplicate")
	ambiguousB := legacyScopeSymbolDef("def:src/other.ts#1:0:Class:Duplicate", "src/other.ts", "Duplicate", scopeir.NodeClass, "models.Duplicate")

	source := scopeir.ScopeIR{
		FilePath:    "src/app.ts",
		FileHash:    "hash-app",
		Language:    scanner.TypeScript,
		ModuleScope: moduleScope,
		Scopes: []scopeir.ScopeFact{
			legacyScopeSymbolModuleScope(moduleScope, "src/app.ts", legacyScopeSymbolBinding("save", saveDef)),
			{ID: functionScope, Parent: stringPtr(moduleScope), Kind: scopeir.ScopeFunction, FilePath: "src/app.ts", FileHash: "hash-app", Range: scopeir.Range{StartLine: 3, EndLine: 8}, OwnedDefIDs: []string{saveDef.ID}},
		},
		Definitions: []scopeir.DefinitionFact{saveDef},
		Imports:     []scopeir.ImportFact{{FilePath: "src/app.ts", Kind: scopeir.ImportAlias, LocalName: "Account", ImportedName: "User", Alias: "Account", TargetRaw: &modelsRaw}},
		TypeAnnotations: []scopeir.TypeAnnotationFact{
			{FilePath: "src/app.ts", FileHash: "hash-app", Name: "account", InScope: functionScope, Type: scopeir.TypeRef{RawName: "Account", DeclaredAtScope: functionScope, Source: scopeir.TypeSourceParameter}},
			{FilePath: "src/app.ts", FileHash: "hash-app", Name: "repo", InScope: functionScope, Type: scopeir.TypeRef{RawName: "models.Repo", DeclaredAtScope: functionScope, Source: scopeir.TypeSourceAnnotation}},
			{FilePath: "src/app.ts", FileHash: "hash-app", Name: "duplicate", InScope: functionScope, Type: scopeir.TypeRef{RawName: "models.Duplicate", DeclaredAtScope: functionScope, Source: scopeir.TypeSourceAnnotation}},
		},
	}
	models := scopeir.ScopeIR{
		FilePath:    "src/models.ts",
		Language:    scanner.TypeScript,
		ModuleScope: modelsModule,
		Scopes:      []scopeir.ScopeFact{legacyScopeSymbolModuleScope(modelsModule, "src/models.ts", legacyScopeSymbolBinding("User", userDef), legacyScopeSymbolBinding("Repo", repoDef), legacyScopeSymbolBinding("Duplicate", ambiguousA))},
		Definitions: []scopeir.DefinitionFact{userDef, repoDef, ambiguousA},
	}
	other := scopeir.ScopeIR{
		FilePath:    "src/other.ts",
		Language:    scanner.TypeScript,
		ModuleScope: otherModule,
		Scopes:      []scopeir.ScopeFact{legacyScopeSymbolModuleScope(otherModule, "src/other.ts", legacyScopeSymbolBinding("Duplicate", ambiguousB))},
		Definitions: []scopeir.DefinitionFact{ambiguousB},
	}

	result, err := Resolve([]scopeir.ScopeIR{source, models, other}, Options{})
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	requireRelationship(t, result.Graph, graph.RelUses, "Function:src/app.ts:save", "Class:src/models.ts:models.User")
	requireRelationship(t, result.Graph, graph.RelUses, "Function:src/app.ts:save", "Class:src/models.ts:models.Repo")
	requireNoRelationship(t, result.Graph, graph.RelUses, "Function:src/app.ts:save", "Class:src/models.ts:models.Duplicate")
	requireNoRelationship(t, result.Graph, graph.RelUses, "Function:src/app.ts:save", "Class:src/other.ts:models.Duplicate")
	if result.Metrics.ResolvedTypeReferences != 2 || result.Metrics.UnresolvedReferences != 1 {
		t.Fatalf("type-reference metrics = %#v, want 2 resolved and 1 unresolved", result.Metrics)
	}
}

func legacyScopeSymbolDef(id string, filePath string, name string, label scopeir.NodeLabel, qualified string) scopeir.DefinitionFact {
	return scopeir.DefinitionFact{
		ID:            id,
		FilePath:      filePath,
		Name:          name,
		Label:         label,
		QualifiedName: qualified,
		Range:         scopeir.Range{StartLine: 1, EndLine: 1},
	}
}

func legacyScopeSymbolModuleScope(id string, filePath string, bindings ...scopeir.BindingFact) scopeir.ScopeFact {
	return scopeir.ScopeFact{
		ID:       id,
		Kind:     scopeir.ScopeModule,
		FilePath: filePath,
		Range:    scopeir.Range{StartLine: 1, EndLine: 20},
		Bindings: bindings,
	}
}

func legacyScopeSymbolBinding(name string, def scopeir.DefinitionFact) scopeir.BindingFact {
	return scopeir.BindingFact{Name: name, DefID: def.ID, Origin: scopeir.BindingLocal}
}

func legacyScopeSymbolRequireBinding(t *testing.T, workspace *workspace, scopeID string, name string, origin scopeir.BindingOrigin, defID string) {
	t.Helper()
	for _, binding := range workspace.scopeBindings[scopeID][name] {
		if binding.Origin == origin && binding.Def.Fact.ID == defID {
			if binding.Via == nil {
				t.Fatalf("binding %s/%s has nil Via", scopeID, name)
			}
			return
		}
	}
	t.Fatalf("missing binding scope=%s name=%s origin=%s def=%s in %#v", scopeID, name, origin, defID, workspace.scopeBindings[scopeID][name])
}
