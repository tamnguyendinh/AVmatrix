package resolution

import (
	"reflect"
	"testing"

	"github.com/tamnguyendinh/avmatrix-go/internal/graph"
	"github.com/tamnguyendinh/avmatrix-go/internal/scanner"
	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

func TestLegacyImportResolutionConversionCoversLanguageStrategies(t *testing.T) {
	workspace, err := buildWorkspace([]scopeir.ScopeIR{
		{FilePath: "src/app.ts", Language: scanner.TypeScript},
		{FilePath: "src/models/index.ts", Language: scanner.TypeScript},
		{FilePath: "src/main/java/com/example/models/Repo.java", Language: scanner.Java},
		{FilePath: "src/main/java/com/example/models/User.java", Language: scanner.Java},
		{FilePath: "src/main/java/com/example/util/FormatUtil.java", Language: scanner.Java},
		{FilePath: "src/main/kotlin/com/example/models/User.kt", Language: scanner.Kotlin},
		{FilePath: "src/App/Models/Repo.cs", Language: scanner.CSharp},
		{FilePath: "src/App/Models/User.cs", Language: scanner.CSharp},
		{FilePath: "app/Models/User.php", Language: scanner.PHP},
		{FilePath: "app/Models/functions.php", Language: scanner.PHP},
		{FilePath: "Sources/App/Service.swift", Language: scanner.Swift},
		{FilePath: "lib/app.rb", Language: scanner.Ruby},
		{FilePath: "lib/models/user.rb", Language: scanner.Ruby},
		{FilePath: "src/pkg/config/app.py", Language: scanner.Python},
		{FilePath: "src/pkg/models.py", Language: scanner.Python},
		{FilePath: "src/pkg/middleware.py", Language: scanner.Python},
		{FilePath: "src/models.rs", Language: scanner.Rust},
		{FilePath: "lib/main.dart", Language: scanner.Dart},
		{FilePath: "lib/models/user.dart", Language: scanner.Dart},
		{FilePath: "lib/repo.dart", Language: scanner.Dart},
		{FilePath: "include/models/user.hpp", Language: scanner.CPlusPlus},
	})
	if err != nil {
		t.Fatalf("buildWorkspace() error = %v", err)
	}

	tests := []struct {
		name     string
		language scanner.Language
		source   string
		target   string
		want     []string
	}{
		{name: "typescript relative index", language: scanner.TypeScript, source: "src/app.ts", target: "./models", want: []string{"src/models/index.ts"}},
		{name: "java wildcard", language: scanner.Java, source: "src/Main.java", target: "com.example.models.*", want: []string{"src/main/java/com/example/models/Repo.java", "src/main/java/com/example/models/User.java"}},
		{name: "java member import", language: scanner.Java, source: "src/Main.java", target: "com.example.util.FormatUtil.format", want: []string{"src/main/java/com/example/util/FormatUtil.java"}},
		{name: "kotlin wildcard", language: scanner.Kotlin, source: "src/App.kt", target: "com.example.models.*", want: []string{"src/main/kotlin/com/example/models/User.kt"}},
		{name: "csharp namespace", language: scanner.CSharp, source: "src/App.cs", target: "App.Models", want: []string{"src/App/Models/Repo.cs", "src/App/Models/User.cs"}},
		{name: "php psr suffix", language: scanner.PHP, source: "public/index.php", target: `App\Models\User`, want: []string{"app/Models/User.php"}},
		{name: "php function namespace fallback", language: scanner.PHP, source: "public/index.php", target: `App\Models\getUser`, want: []string{"app/Models/User.php"}},
		{name: "swift package target", language: scanner.Swift, source: "Sources/App/main.swift", target: "App", want: []string{"Sources/App/Service.swift"}},
		{name: "ruby require_relative", language: scanner.Ruby, source: "lib/app.rb", target: "./models/user", want: []string{"lib/models/user.rb"}},
		{name: "python relative", language: scanner.Python, source: "src/pkg/config/app.py", target: "..models", want: []string{"src/pkg/models.py"}},
		{name: "python nearest ancestor bare import", language: scanner.Python, source: "src/pkg/config/app.py", target: "middleware", want: []string{"src/pkg/middleware.py"}},
		{name: "rust crate module", language: scanner.Rust, source: "src/main.rs", target: "crate::models::{User, Repo}", want: []string{"src/models.rs"}},
		{name: "dart package local", language: scanner.Dart, source: "lib/main.dart", target: "package:my_app/models/user.dart", want: []string{"lib/models/user.dart"}},
		{name: "dart relative", language: scanner.Dart, source: "lib/main.dart", target: "./repo.dart", want: []string{"lib/repo.dart"}},
		{name: "cpp include suffix", language: scanner.CPlusPlus, source: "src/app.cpp", target: "models/user.hpp", want: []string{"include/models/user.hpp"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := workspace.resolveImportFiles(test.language, test.source, test.target); !reflect.DeepEqual(got, test.want) {
				t.Fatalf("resolveImportFiles(%s, %s) = %#v, want %#v", test.language, test.target, got, test.want)
			}
		})
	}

	if got := workspace.resolveImportFiles(scanner.Dart, "lib/main.dart", "dart:async"); got != nil {
		t.Fatalf("dart sdk import files = %#v, want nil", got)
	}
	if got := workspace.resolveImportFiles(scanner.Python, "src/pkg/config/app.py", "django.urls"); got != nil {
		t.Fatalf("external dotted python import files = %#v, want nil", got)
	}
}

func TestLegacyImportPreprocessingConversionRejectsUnsafePaths(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
		ok   bool
	}{
		{name: "double quotes", raw: `"foo/bar"`, want: "foo/bar", ok: true},
		{name: "single quotes", raw: `'foo/bar'`, want: "foo/bar", ok: true},
		{name: "angle include", raw: `<stdio.h>`, want: "stdio.h", ok: true},
		{name: "mixed delimiters", raw: `"'hello<>"`, want: "hello", ok: true},
		{name: "empty", raw: `""`, ok: false},
		{name: "control character", raw: "bad\npath", ok: false},
		{name: "null byte", raw: "bad\x00path", ok: false},
		{name: "max length", raw: stringOfLength(2048), want: stringOfLength(2048), ok: true},
		{name: "too long", raw: stringOfLength(2049), ok: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, ok := preprocessImportTarget(test.raw)
			if ok != test.ok || got != test.want {
				t.Fatalf("preprocessImportTarget(%q) = %q/%v, want %q/%v", test.raw, got, ok, test.want, test.ok)
			}
		})
	}
}

func TestLegacyWildcardSynthesisConversionCoversGoFallbackAndTransitiveIncludes(t *testing.T) {
	goRaw := "github.com/tamnguyendinh/avmatrix-go/internal/pkg"
	headerRaw := "include/a.h"
	deepRaw := "include/b.h"
	goModuleScope := "scope:cmd/app/main.go#1:0-4:1:Module"
	cModuleScope := "scope:src/app.c#1:0-4:1:Module"
	aHeaderScope := "scope:include/a.h#1:0-2:1:Module"
	helper := legacyImportConversionDef("def:internal/pkg/helper.go#1:0:Function:Helper", "internal/pkg/helper.go", "Helper", scopeir.NodeFunction)
	deep := legacyImportConversionDef("def:include/b.h#1:0:Function:Deep", "include/b.h", "Deep", scopeir.NodeFunction)

	workspace, err := buildWorkspace([]scopeir.ScopeIR{
		{
			FilePath:    "cmd/app/main.go",
			Language:    scanner.Go,
			ModuleScope: goModuleScope,
			Scopes:      []scopeir.ScopeFact{{ID: goModuleScope, Kind: scopeir.ScopeModule, FilePath: "cmd/app/main.go"}},
			Imports:     []scopeir.ImportFact{{FilePath: "cmd/app/main.go", Kind: scopeir.ImportWildcard, ImportedName: "*", TargetRaw: &goRaw}},
		},
		{FilePath: "internal/pkg/helper.go", Language: scanner.Go, Definitions: []scopeir.DefinitionFact{helper}},
		{
			FilePath:    "src/app.c",
			Language:    scanner.C,
			ModuleScope: cModuleScope,
			Scopes:      []scopeir.ScopeFact{{ID: cModuleScope, Kind: scopeir.ScopeModule, FilePath: "src/app.c"}},
			Imports:     []scopeir.ImportFact{{FilePath: "src/app.c", Kind: scopeir.ImportWildcard, ImportedName: "*", TargetRaw: &headerRaw}},
		},
		{
			FilePath:    "include/a.h",
			Language:    scanner.C,
			ModuleScope: aHeaderScope,
			Scopes:      []scopeir.ScopeFact{{ID: aHeaderScope, Kind: scopeir.ScopeModule, FilePath: "include/a.h"}},
			Imports:     []scopeir.ImportFact{{FilePath: "include/a.h", Kind: scopeir.ImportWildcard, ImportedName: "*", TargetRaw: &deepRaw}},
		},
		{FilePath: "include/b.h", Language: scanner.C, Definitions: []scopeir.DefinitionFact{deep}},
	})
	if err != nil {
		t.Fatalf("buildWorkspace() error = %v", err)
	}

	legacyImportConversionRequireBinding(t, workspace, goModuleScope, "Helper", scopeir.BindingWildcard, helper.ID)
	legacyImportConversionRequireBinding(t, workspace, cModuleScope, "Deep", scopeir.BindingWildcard, deep.ID)
}

func TestLegacyCrossFileBindingConversionResolvesCallsAcrossLanguageImports(t *testing.T) {
	tests := []struct {
		name     string
		language scanner.Language
		source   string
		target   string
		raw      string
		local    string
		imported string
		receiver string
		form     scopeir.CallForm
	}{
		{name: "typescript", language: scanner.TypeScript, source: "src/app.ts", target: "src/helper.ts", raw: "./helper", local: "Helper", imported: "Helper", form: scopeir.CallFree},
		{name: "java", language: scanner.Java, source: "src/main/java/com/example/App.java", target: "src/main/java/com/example/Helper.java", raw: "com.example.Helper", local: "Helper", imported: "Helper", form: scopeir.CallFree},
		{name: "kotlin", language: scanner.Kotlin, source: "src/main/kotlin/com/example/App.kt", target: "src/main/kotlin/com/example/Helper.kt", raw: "com.example.Helper", local: "Helper", imported: "Helper", form: scopeir.CallFree},
		{name: "csharp", language: scanner.CSharp, source: "src/App/App.cs", target: "src/App/Helper.cs", raw: "App.Helper", local: "Helper", imported: "Helper", form: scopeir.CallFree},
		{name: "php", language: scanner.PHP, source: "public/index.php", target: "app/Helper.php", raw: `App\Helper`, local: "Helper", imported: "Helper", form: scopeir.CallFree},
		{name: "dart", language: scanner.Dart, source: "lib/main.dart", target: "lib/helper.dart", raw: "package:my_app/helper.dart", local: "Helper", imported: "Helper", form: scopeir.CallFree},
		{name: "rust", language: scanner.Rust, source: "src/main.rs", target: "src/helper.rs", raw: "crate::helper", local: "Helper", imported: "Helper", form: scopeir.CallFree},
		{name: "python", language: scanner.Python, source: "src/pkg/app.py", target: "src/pkg/helper.py", raw: ".helper", local: "Helper", imported: "Helper", form: scopeir.CallFree},
		{name: "ruby", language: scanner.Ruby, source: "lib/app.rb", target: "lib/helper.rb", raw: "./helper", local: "Helper", imported: "Helper", form: scopeir.CallFree},
		{name: "go package receiver", language: scanner.Go, source: "cmd/app/main.go", target: "internal/pkg/helper.go", raw: "github.com/tamnguyendinh/avmatrix-go/internal/pkg", local: "pkg", imported: "pkg", receiver: "pkg", form: scopeir.CallMember},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := Resolve(legacyImportConversionCrossFileIR(test.language, test.source, test.target, test.raw, test.local, test.imported, test.receiver, test.form), Options{})
			if err != nil {
				t.Fatalf("Resolve() error = %v", err)
			}
			requireRelationship(t, result.Graph, graph.RelCalls, "Function:"+test.source+":Run", "Function:"+test.target+":Helper")
			if result.Metrics.ResolvedCalls != 1 || result.Metrics.UnresolvedReferences != 0 {
				t.Fatalf("unexpected metrics: %#v", result.Metrics)
			}
		})
	}
}

func legacyImportConversionCrossFileIR(language scanner.Language, sourceFile string, targetFile string, raw string, local string, imported string, receiver string, form scopeir.CallForm) []scopeir.ScopeIR {
	sourceModule := "scope:" + sourceFile + "#module"
	sourceFunction := "scope:" + sourceFile + "#run"
	run := legacyImportConversionDef("def:"+sourceFile+"#1:0:Function:Run", sourceFile, "Run", scopeir.NodeFunction)
	helper := legacyImportConversionDef("def:"+targetFile+"#1:0:Function:Helper", targetFile, "Helper", scopeir.NodeFunction)
	return []scopeir.ScopeIR{
		{
			FilePath:    sourceFile,
			FileHash:    "hash-" + sourceFile,
			Language:    language,
			ModuleScope: sourceModule,
			Scopes: []scopeir.ScopeFact{
				{ID: sourceModule, Kind: scopeir.ScopeModule, FilePath: sourceFile, Bindings: []scopeir.BindingFact{{Name: "Run", DefID: run.ID, Origin: scopeir.BindingLocal}}},
				{ID: sourceFunction, Parent: stringPtr(sourceModule), Kind: scopeir.ScopeFunction, FilePath: sourceFile, OwnedDefIDs: []string{run.ID}, Bindings: []scopeir.BindingFact{{Name: "Run", DefID: run.ID, Origin: scopeir.BindingLocal}}},
			},
			Definitions: []scopeir.DefinitionFact{run},
			Imports: []scopeir.ImportFact{{
				FilePath:     sourceFile,
				Kind:         scopeir.ImportNamed,
				LocalName:    local,
				ImportedName: imported,
				TargetRaw:    &raw,
			}},
			Calls: []scopeir.CallSiteFact{{
				FilePath:         sourceFile,
				FileHash:         "hash-" + sourceFile,
				Name:             "Helper",
				ExplicitReceiver: receiver,
				InScope:          sourceFunction,
				CallForm:         form,
				Range:            scopeir.Range{StartLine: 2, StartCol: 2, EndLine: 2, EndCol: 10},
			}},
		},
		{
			FilePath:    targetFile,
			Language:    language,
			ModuleScope: "scope:" + targetFile + "#module",
			Scopes:      []scopeir.ScopeFact{{ID: "scope:" + targetFile + "#module", Kind: scopeir.ScopeModule, FilePath: targetFile, Bindings: []scopeir.BindingFact{{Name: "Helper", DefID: helper.ID, Origin: scopeir.BindingLocal}}}},
			Definitions: []scopeir.DefinitionFact{helper},
		},
	}
}

func legacyImportConversionDef(id string, filePath string, name string, label scopeir.NodeLabel) scopeir.DefinitionFact {
	return scopeir.DefinitionFact{
		ID:            id,
		FilePath:      filePath,
		Name:          name,
		Label:         label,
		QualifiedName: name,
		Range:         scopeir.Range{StartLine: 1, EndLine: 1},
	}
}

func legacyImportConversionRequireBinding(t *testing.T, workspace *workspace, scopeID string, name string, origin scopeir.BindingOrigin, defID string) {
	t.Helper()
	for _, binding := range workspace.scopeBindings[scopeID][name] {
		if binding.Origin == origin && binding.Def.Fact.ID == defID {
			return
		}
	}
	t.Fatalf("missing binding scope=%s name=%s origin=%s def=%s in %#v", scopeID, name, origin, defID, workspace.scopeBindings[scopeID][name])
}

func stringOfLength(length int) string {
	if length <= 0 {
		return ""
	}
	return stringsRepeat("a", length)
}

func stringsRepeat(value string, count int) string {
	out := ""
	for index := 0; index < count; index++ {
		out += value
	}
	return out
}
