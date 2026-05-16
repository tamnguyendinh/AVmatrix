package resolution

import (
	"reflect"
	"testing"

	"github.com/tamnguyendinh/avmatrix-go/internal/scanner"
	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

func TestLegacyScopeIndexParityBuildsWorkspaceLookupSurfaces(t *testing.T) {
	targetRaw := "./models"
	moduleScope := "scope:src/app.ts#1:0-20:0:Module"
	functionScope := "scope:src/app.ts#8:0-16:0:Function"
	userClass := scopeir.DefinitionFact{
		ID:            "def:src/models.ts#1:0:Class:User",
		FilePath:      "src/models.ts",
		Name:          "User",
		Label:         scopeir.NodeClass,
		QualifiedName: "models.User",
		Range:         scopeir.Range{StartLine: 1, EndLine: 4},
	}
	saveMethod := scopeir.DefinitionFact{
		ID:            "def:src/models.ts#2:2:Method:save",
		FilePath:      "src/models.ts",
		Name:          "save",
		Label:         scopeir.NodeMethod,
		QualifiedName: "models.User.save",
		OwnerID:       userClass.ID,
		Range:         scopeir.Range{StartLine: 2, EndLine: 2},
	}
	startFn := scopeir.DefinitionFact{
		ID:            "def:src/app.ts#8:0:Function:start",
		FilePath:      "src/app.ts",
		Name:          "start",
		Label:         scopeir.NodeFunction,
		QualifiedName: "start",
		Range:         scopeir.Range{StartLine: 8, EndLine: 16},
	}
	currentVar := scopeir.DefinitionFact{
		ID:            "def:src/app.ts#9:2:Variable:current",
		FilePath:      "src/app.ts",
		Name:          "current",
		Label:         scopeir.NodeVariable,
		QualifiedName: "current",
		Range:         scopeir.Range{StartLine: 9, StartCol: 2, EndLine: 9, EndCol: 20},
	}
	source := scopeir.ScopeIR{
		FilePath:    "src\\app.ts",
		FileHash:    "hash-app",
		Language:    scanner.TypeScript,
		ModuleScope: moduleScope,
		Scopes: []scopeir.ScopeFact{
			{
				ID:       moduleScope,
				Kind:     scopeir.ScopeModule,
				FilePath: "src\\app.ts",
				FileHash: "hash-app",
				Range:    scopeir.Range{StartLine: 1, EndLine: 20},
				Bindings: []scopeir.BindingFact{
					{Name: "start", DefID: startFn.ID, Origin: scopeir.BindingLocal},
				},
			},
			{
				ID:       functionScope,
				Parent:   stringPtr(moduleScope),
				Kind:     scopeir.ScopeFunction,
				FilePath: "src\\app.ts",
				FileHash: "hash-app",
				Range:    scopeir.Range{StartLine: 8, EndLine: 16},
				Bindings: []scopeir.BindingFact{
					{Name: "current", DefID: currentVar.ID, Origin: scopeir.BindingLocal},
				},
				OwnedDefIDs: []string{startFn.ID, currentVar.ID},
				TypeBindings: []scopeir.TypeBindingFact{{
					Name: "current",
					Type: scopeir.TypeRef{
						RawName:         "U",
						DeclaredAtScope: functionScope,
						Source:          scopeir.TypeSourceParameter,
					},
				}},
			},
		},
		Definitions: []scopeir.DefinitionFact{startFn, currentVar},
		Imports: []scopeir.ImportFact{{
			FilePath:     "src\\app.ts",
			Kind:         scopeir.ImportAlias,
			LocalName:    "U",
			ImportedName: "User",
			Alias:        "U",
			TargetRaw:    &targetRaw,
		}},
	}
	target := scopeir.ScopeIR{
		FilePath:    "src/models.ts",
		FileHash:    "hash-models",
		Language:    scanner.TypeScript,
		ModuleScope: "scope:src/models.ts#1:0-4:0:Module",
		Scopes: []scopeir.ScopeFact{{
			ID:       "scope:src/models.ts#1:0-4:0:Module",
			Kind:     scopeir.ScopeModule,
			FilePath: "src/models.ts",
			FileHash: "hash-models",
			Range:    scopeir.Range{StartLine: 1, EndLine: 4},
			Bindings: []scopeir.BindingFact{
				{Name: "User", DefID: userClass.ID, Origin: scopeir.BindingLocal},
			},
			OwnedDefIDs: []string{userClass.ID},
		}},
		Definitions: []scopeir.DefinitionFact{userClass, saveMethod},
	}

	workspace, err := buildWorkspace([]scopeir.ScopeIR{source, target})
	if err != nil {
		t.Fatalf("buildWorkspace() error = %v", err)
	}

	if _, ok := workspace.fileSet["src/app.ts"]; !ok {
		t.Fatalf("source path was not normalized into fileSet: %#v", workspace.fileSet)
	}
	if got := workspace.moduleScopeByFile["src/app.ts"]; got != moduleScope {
		t.Fatalf("moduleScopeByFile[src/app.ts] = %q, want %q", got, moduleScope)
	}
	if got := workspace.fileHashes["src/app.ts"]; got != "hash-app" {
		t.Fatalf("fileHashes[src/app.ts] = %q, want hash-app", got)
	}
	if got := workspace.defsByID[startFn.ID].Fact.Name; got != "start" {
		t.Fatalf("defsByID[%s].Name = %q, want start", startFn.ID, got)
	}
	if got := workspace.scopeByDef[currentVar.ID]; got != functionScope {
		t.Fatalf("scopeByDef[%s] = %q, want %q", currentVar.ID, got, functionScope)
	}
	if got := idsForDefs(workspace.defsByName["models.User"]); !reflect.DeepEqual(got, []string{userClass.ID}) {
		t.Fatalf("defsByName[models.User] = %#v, want %s", got, userClass.ID)
	}
	if got := idsForDefs(workspace.defsByName["User"]); !reflect.DeepEqual(got, []string{userClass.ID}) {
		t.Fatalf("defsByName[User] = %#v, want %s", got, userClass.ID)
	}
	if got := idsForDefs(workspace.ownerMembers[userClass.ID]["save"]); !reflect.DeepEqual(got, []string{saveMethod.ID}) {
		t.Fatalf("ownerMembers[%s][save] = %#v, want %s", userClass.ID, got, saveMethod.ID)
	}
	if got := workspace.typeBindings[functionScope]["current"]; len(got) != 1 || got[0].RawName != "U" {
		t.Fatalf("type binding current = %#v, want U", got)
	}

	importBindings := workspace.scopeBindings[moduleScope]["U"]
	if len(importBindings) != 1 {
		t.Fatalf("scopeBindings[%s][U] = %#v, want one import binding", moduleScope, importBindings)
	}
	if importBindings[0].Origin != scopeir.BindingImport || importBindings[0].Def.Fact.ID != userClass.ID {
		t.Fatalf("import binding = %#v, want imported User", importBindings[0])
	}
	if importBindings[0].Via == nil || importBindings[0].Via.TargetFile != "src/models.ts" || importBindings[0].Via.LinkStatus != "" {
		t.Fatalf("import binding via = %#v, want linked src/models.ts", importBindings[0].Via)
	}
}

func TestLegacyImportTargetAdapterParityUsesGoAndRelativeImportResolution(t *testing.T) {
	workspace, err := buildWorkspace([]scopeir.ScopeIR{
		{FilePath: "src/app.ts", Language: scanner.TypeScript},
		{FilePath: "src/models.ts", Language: scanner.TypeScript},
		{FilePath: "src/models/index.ts", Language: scanner.TypeScript},
		{FilePath: "cmd/app/main.go", Language: scanner.Go},
		{FilePath: "internal/pkg/b.go", Language: scanner.Go},
		{FilePath: "internal/pkg/a.go", Language: scanner.Go},
		{FilePath: "internal/pkg/a_test.go", Language: scanner.Go},
	})
	if err != nil {
		t.Fatalf("buildWorkspace() error = %v", err)
	}

	if got := workspace.resolveImportFiles(scanner.TypeScript, "src/app.ts", "./models"); !reflect.DeepEqual(got, []string{"src/models.ts"}) {
		t.Fatalf("TypeScript relative import files = %#v, want src/models.ts", got)
	}
	if got := workspace.resolveImportFiles(scanner.Go, "cmd/app/main.go", "github.com/tamnguyendinh/avmatrix-go/internal/pkg"); !reflect.DeepEqual(got, []string{"internal/pkg/a.go", "internal/pkg/b.go"}) {
		t.Fatalf("Go package import files = %#v, want sorted non-test package files", got)
	}
	if got := workspace.resolveImportFiles(scanner.TypeScript, "src/app.ts", "external-pkg"); got != nil {
		t.Fatalf("external import files = %#v, want nil", got)
	}
}

func idsForDefs(defs []defRef) []string {
	ids := make([]string, 0, len(defs))
	for _, def := range defs {
		ids = append(ids, def.Fact.ID)
	}
	return ids
}
