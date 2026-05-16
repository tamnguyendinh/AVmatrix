package dart

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"sort"
	"strconv"
	"testing"
	"time"

	"github.com/tamnguyendinh/avmatrix-go/internal/graph"
	"github.com/tamnguyendinh/avmatrix-go/internal/parser"
	"github.com/tamnguyendinh/avmatrix-go/internal/resolution"
	"github.com/tamnguyendinh/avmatrix-go/internal/scanner"
	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

const dartParityFixture = `import 'repo.dart';
import 'base.dart' as base;

abstract class Named {
  bool save(User user);
}

class Base {
  final String id;
  Base(this.id);
}

class User {
  final String id;
  User(this.id);
}

class Repository {
  void write(String id) {}
}

class Service extends Base implements Named {
  final Repository repo;

  Service(this.repo) : super('root');

  @override
  bool save(User user) {
    final id = normalize(user);
    repo.write(id);
    return true;
  }

  String normalize(User user) {
    return user.id.trim();
  }
}
`

func TestExtractDartScopeIR(t *testing.T) {
	ir := parseAndExtract(t, "lib/service.dart", "hash-dart", []byte(dartParityFixture))

	for _, def := range ir.Definitions {
		if def.FileHash != "hash-dart" {
			t.Fatalf("definition %s missing file hash: %#v", def.Name, def)
		}
	}
	for _, call := range ir.Calls {
		if call.FileHash != "hash-dart" {
			t.Fatalf("call %s missing file hash: %#v", call.Name, call)
		}
	}

	named := requireDefinition(t, ir, "Named", scopeir.NodeInterface)
	base := requireDefinition(t, ir, "Base", scopeir.NodeClass)
	user := requireDefinition(t, ir, "User", scopeir.NodeClass)
	repository := requireDefinition(t, ir, "Repository", scopeir.NodeClass)
	service := requireDefinition(t, ir, "Service", scopeir.NodeClass)
	requireQualifiedDefinition(t, ir, "Named.save", scopeir.NodeMethod)
	requireQualifiedDefinition(t, ir, "Base.Base", scopeir.NodeConstructor)
	requireQualifiedDefinition(t, ir, "User.User", scopeir.NodeConstructor)
	requireQualifiedDefinition(t, ir, "Repository.write", scopeir.NodeMethod)
	constructor := requireQualifiedDefinition(t, ir, "Service.Service", scopeir.NodeConstructor)
	save := requireQualifiedDefinition(t, ir, "Service.save", scopeir.NodeMethod)
	normalize := requireQualifiedDefinition(t, ir, "Service.normalize", scopeir.NodeMethod)
	baseID := requireQualifiedDefinition(t, ir, "Base.id", scopeir.NodeProperty)
	userID := requireQualifiedDefinition(t, ir, "User.id", scopeir.NodeProperty)
	repo := requireQualifiedDefinition(t, ir, "Service.repo", scopeir.NodeProperty)
	id := requireDefinition(t, ir, "id", scopeir.NodeVariable)
	if named.ID == "" || base.ID == "" || user.ID == "" || repository.ID == "" || service.ID == "" || baseID.ID == "" || userID.ID == "" || repo.ID == "" {
		t.Fatal("expected all Dart definitions to have stable IDs")
	}
	if constructor.OwnerID != service.ID || save.OwnerID != service.ID || normalize.OwnerID != service.ID || repo.OwnerID != service.ID {
		t.Fatalf("owner mismatch: constructor=%#v save=%#v normalize=%#v repo=%#v", constructor, save, normalize, repo)
	}
	if save.ReturnType != "bool" || normalize.ReturnType != "String" || id.DeclaredType != "String" || repo.DeclaredType != "Repository" {
		t.Fatalf("type extraction mismatch: save=%#v normalize=%#v id=%#v repo=%#v", save, normalize, id, repo)
	}

	requireImport(t, ir, scopeir.ImportNamed, "repo", "repo", "repo.dart")
	requireImport(t, ir, scopeir.ImportAlias, "base", "base", "base.dart")
	requireCall(t, ir, "normalize", scopeir.CallMember)
	requireCall(t, ir, "write", scopeir.CallMember)
	requireCall(t, ir, "trim", scopeir.CallMember)
	requireAccess(t, ir, "repo", scopeir.AccessRead)
	requireAccess(t, ir, "id", scopeir.AccessRead)
	requireHeritage(t, ir, "Base", scopeir.HeritageExtends)
	requireHeritage(t, ir, "Named", scopeir.HeritageImplements)
	requireTypeBinding(t, ir, "this", "Service")
	requireTypeBinding(t, ir, "repo", "Repository")
	requireTypeBinding(t, ir, "user", "User")
	requireTypeBinding(t, ir, "id", "String")
	requireTypeAnnotation(t, ir, "repo", "Repository")
	requireReturnType(t, ir, save.ID, "bool")
	requireReturnType(t, ir, normalize.ID, "String")
}

func TestExtractDartImportAndTypeEdgeCases(t *testing.T) {
	const source = `import 'dart:async';
import 'package:my_app/models/user.dart';
import './repo.dart' as repo;
export 'src/public.dart';

class User {}

User getUser() {
  return User();
}

void demo(User? maybeUser, List<String> names) {
  User explicit = maybeUser;
  final inferred = getUser();
}
`

	ir := parseAndExtract(t, "lib/main.dart", "hash-dart-edge", []byte(source))

	requireImport(t, ir, scopeir.ImportNamed, "dart:async", "dart:async", "dart:async")
	requireImport(t, ir, scopeir.ImportNamed, "user", "user", "package:my_app/models/user.dart")
	requireImport(t, ir, scopeir.ImportAlias, "repo", "repo", "./repo.dart")
	requireImport(t, ir, scopeir.ImportNamed, "public", "public", "src/public.dart")

	getUser := requireDefinition(t, ir, "getUser", scopeir.NodeFunction)
	requireReturnType(t, ir, getUser.ID, "User")
	requireTypeBinding(t, ir, "maybeUser", "User")
	requireTypeBinding(t, ir, "names", "List")
	requireTypeBinding(t, ir, "explicit", "User")
	requireTypeBinding(t, ir, "inferred", "User")
	requireTypeAnnotation(t, ir, "inferred", "User")
}

func TestExtractDartScopeIRParityFixture(t *testing.T) {
	ir := parseAndExtract(t, "lib/service.dart", "hash-dart", []byte(dartParityFixture))
	signature := buildParitySignature(ir)
	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(signature); err != nil {
		t.Fatalf("marshal signature: %v", err)
	}
	raw := buffer.Bytes()
	golden, err := os.ReadFile("testdata/dart_scopeir_signature.golden.json")
	if err != nil {
		t.Fatalf("read golden: %v\ngot:\n%s", err, raw)
	}
	if string(raw) != string(golden) {
		t.Fatalf("parity signature mismatch\nwant:\n%s\ngot:\n%s", golden, raw)
	}
}

func TestResolveDartGraphParityCounts(t *testing.T) {
	ir := parseAndExtract(t, "lib/service.dart", "hash-dart", []byte(dartParityFixture))
	result, err := resolution.Resolve([]scopeir.ScopeIR{ir}, resolution.Options{})
	if err != nil {
		t.Fatalf("resolve failed: %v", err)
	}
	counts := stringRelationshipCounts(result.Graph.RelationshipCountsByType())
	expected := map[string]int{
		"ACCESSES":          2,
		"CALLS":             2,
		"DEFINES":           16,
		"EXTENDS":           1,
		"HAS_METHOD":        7,
		"HAS_PROPERTY":      3,
		"IMPLEMENTS":        1,
		"INHERITS":          2,
		"METHOD_IMPLEMENTS": 1,
		"USES":              4,
	}
	if !stringIntMapsEqual(counts, expected) {
		t.Fatalf("relationship counts mismatch\nwant: %#v\ngot:  %#v", expected, counts)
	}
	if result.Metrics.UnresolvedReferences == 0 {
		t.Fatalf("expected unresolved external references, metrics=%#v", result.Metrics)
	}
	if result.Metrics.ResolvedCalls == 0 || result.Metrics.ResolvedAccesses == 0 || result.Metrics.ResolvedInheritance == 0 {
		t.Fatalf("expected resolved calls/accesses/heritage, got metrics %#v", result.Metrics)
	}
}

func BenchmarkExtractDartScopeIR(b *testing.B) {
	source := []byte(dartParityFixture)
	pool := parser.NewPool(nil, parser.PoolOptions{ParseTimeout: time.Second})
	defer pool.Close()
	parsed, err := pool.Parse(context.Background(), parser.Request{
		FilePath: "lib/service.dart",
		Language: scanner.Dart,
		Source:   source,
	})
	if err != nil {
		b.Fatalf("parse failed: %v", err)
	}
	defer parsed.Close()
	root := parsed.Tree.RootNode()

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		ir, err := Extract(Request{
			FilePath: "lib/service.dart",
			FileHash: "hash-dart",
			Language: scanner.Dart,
			Source:   source,
			Root:     root,
		})
		if err != nil {
			b.Fatalf("extract failed: %v", err)
		}
		if len(ir.Definitions) == 0 || len(ir.Calls) == 0 {
			b.Fatalf("incomplete extraction: %#v", ir)
		}
	}
}

func BenchmarkParseAndExtractDartScopeIR(b *testing.B) {
	source := []byte(dartParityFixture)
	pool := parser.NewPool(nil, parser.PoolOptions{ParseTimeout: time.Second})
	defer pool.Close()

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		parsed, err := pool.Parse(context.Background(), parser.Request{
			FilePath: "lib/service.dart",
			Language: scanner.Dart,
			Source:   source,
		})
		if err != nil {
			b.Fatalf("parse failed: %v", err)
		}
		ir, err := Extract(Request{
			FilePath: "lib/service.dart",
			FileHash: "hash-dart",
			Language: scanner.Dart,
			Source:   source,
			Root:     parsed.Tree.RootNode(),
		})
		parsed.Close()
		if err != nil {
			b.Fatalf("extract failed: %v", err)
		}
		if len(ir.Definitions) == 0 || len(ir.Calls) == 0 {
			b.Fatalf("incomplete extraction: %#v", ir)
		}
	}
}

type paritySignature struct {
	Scopes         []string `json:"scopes"`
	Definitions    []string `json:"definitions"`
	Imports        []string `json:"imports"`
	Calls          []string `json:"calls"`
	Accesses       []string `json:"accesses"`
	Heritage       []string `json:"heritage"`
	TypeReferences []string `json:"typeReferences"`
	TypeBindings   []string `json:"typeBindings"`
}

func buildParitySignature(ir scopeir.ScopeIR) paritySignature {
	signature := paritySignature{}
	for _, scope := range ir.Scopes {
		signature.Scopes = append(signature.Scopes, string(scope.Kind)+":"+scope.ID)
		for _, binding := range scope.TypeBindings {
			signature.TypeBindings = append(signature.TypeBindings, binding.Name+":"+binding.Type.RawName+":"+string(binding.Type.Source))
		}
	}
	for _, def := range ir.Definitions {
		signature.Definitions = append(signature.Definitions, string(def.Label)+":"+def.QualifiedName+":"+def.ReturnType+":"+def.DeclaredType+":"+def.OwnerID)
	}
	for _, item := range ir.Imports {
		target := ""
		if item.TargetRaw != nil {
			target = *item.TargetRaw
		}
		signature.Imports = append(signature.Imports, string(item.Kind)+":"+item.LocalName+":"+item.ImportedName+":"+item.Alias+":"+target)
	}
	for _, call := range ir.Calls {
		signature.Calls = append(signature.Calls, call.Name+":"+string(call.CallForm)+":"+call.ExplicitReceiver+":"+formatOptionalInt(call.Arity))
	}
	for _, access := range ir.Accesses {
		signature.Accesses = append(signature.Accesses, string(access.Kind)+":"+access.Name+":"+access.ExplicitReceiver)
	}
	for _, item := range ir.Heritage {
		signature.Heritage = append(signature.Heritage, string(item.Kind)+":"+item.Name)
	}
	for _, item := range ir.TypeAnnotations {
		if item.Type.RawName != "" {
			signature.TypeReferences = append(signature.TypeReferences, item.Name+":"+item.Type.RawName+":"+string(item.Type.Source))
		}
	}
	sort.Strings(signature.Scopes)
	sort.Strings(signature.Definitions)
	sort.Strings(signature.Imports)
	sort.Strings(signature.Calls)
	sort.Strings(signature.Accesses)
	sort.Strings(signature.Heritage)
	sort.Strings(signature.TypeReferences)
	sort.Strings(signature.TypeBindings)
	return signature
}

func formatOptionalInt(value *int) string {
	if value == nil {
		return ""
	}
	return strconv.Itoa(*value)
}

func parseAndExtract(t *testing.T, filePath string, fileHash string, source []byte) scopeir.ScopeIR {
	t.Helper()
	pool := parser.NewPool(nil, parser.PoolOptions{ParseTimeout: time.Second})
	defer pool.Close()

	parsed, err := pool.Parse(context.Background(), parser.Request{
		FilePath: filePath,
		Language: scanner.Dart,
		Source:   source,
	})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	defer parsed.Close()

	ir, err := Extract(Request{
		FilePath: filePath,
		FileHash: fileHash,
		Language: scanner.Dart,
		Source:   source,
		Root:     parsed.Tree.RootNode(),
	})
	if err != nil {
		t.Fatalf("extract failed: %v", err)
	}
	return ir
}

func requireDefinition(t *testing.T, ir scopeir.ScopeIR, name string, label scopeir.NodeLabel) scopeir.DefinitionFact {
	t.Helper()
	for _, def := range ir.Definitions {
		if def.Name == name && def.Label == label {
			return def
		}
	}
	t.Fatalf("missing definition %s/%s in %#v", name, label, ir.Definitions)
	return scopeir.DefinitionFact{}
}

func requireQualifiedDefinition(t *testing.T, ir scopeir.ScopeIR, qualified string, label scopeir.NodeLabel) scopeir.DefinitionFact {
	t.Helper()
	for _, def := range ir.Definitions {
		if def.QualifiedName == qualified && def.Label == label {
			return def
		}
	}
	t.Fatalf("missing qualified definition %s/%s in %#v", qualified, label, ir.Definitions)
	return scopeir.DefinitionFact{}
}

func requireImport(t *testing.T, ir scopeir.ScopeIR, kind scopeir.ImportKind, local string, imported string, target string) {
	t.Helper()
	for _, item := range ir.Imports {
		if item.Kind == kind && item.LocalName == local && item.ImportedName == imported && item.TargetRaw != nil && *item.TargetRaw == target {
			return
		}
	}
	t.Fatalf("missing import kind=%s local=%s imported=%s target=%s in %#v", kind, local, imported, target, ir.Imports)
}

func requireCall(t *testing.T, ir scopeir.ScopeIR, name string, form scopeir.CallForm) {
	t.Helper()
	for _, call := range ir.Calls {
		if call.Name == name && call.CallForm == form {
			return
		}
	}
	t.Fatalf("missing call %s/%s in %#v", name, form, ir.Calls)
}

func requireAccess(t *testing.T, ir scopeir.ScopeIR, name string, kind scopeir.AccessKind) {
	t.Helper()
	for _, access := range ir.Accesses {
		if access.Name == name && access.Kind == kind {
			return
		}
	}
	t.Fatalf("missing access %s/%s in %#v", name, kind, ir.Accesses)
}

func requireHeritage(t *testing.T, ir scopeir.ScopeIR, name string, kind scopeir.HeritageKind) {
	t.Helper()
	for _, item := range ir.Heritage {
		if item.Name == name && item.Kind == kind {
			return
		}
	}
	t.Fatalf("missing heritage %s/%s in %#v", name, kind, ir.Heritage)
}

func requireTypeBinding(t *testing.T, ir scopeir.ScopeIR, name string, rawName string) {
	t.Helper()
	for _, scope := range ir.Scopes {
		for _, binding := range scope.TypeBindings {
			if binding.Name == name && binding.Type.RawName == rawName {
				return
			}
		}
	}
	t.Fatalf("missing type binding %s -> %s in %#v", name, rawName, ir.Scopes)
}

func requireTypeAnnotation(t *testing.T, ir scopeir.ScopeIR, name string, rawName string) {
	t.Helper()
	for _, item := range ir.TypeAnnotations {
		if item.Name == name && item.Type.RawName == rawName {
			return
		}
	}
	t.Fatalf("missing type annotation %s -> %s in %#v", name, rawName, ir.TypeAnnotations)
}

func requireReturnType(t *testing.T, ir scopeir.ScopeIR, defID string, rawName string) {
	t.Helper()
	for _, item := range ir.ReturnTypes {
		if item.DefID == defID && item.Type.RawName == rawName {
			return
		}
	}
	t.Fatalf("missing return type %s -> %s in %#v", defID, rawName, ir.ReturnTypes)
}

func stringRelationshipCounts(counts map[graph.RelationshipType]int) map[string]int {
	out := make(map[string]int, len(counts))
	for relType, count := range counts {
		out[string(relType)] = count
	}
	return out
}

func stringIntMapsEqual(left map[string]int, right map[string]int) bool {
	if len(left) != len(right) {
		return false
	}
	for key, leftValue := range left {
		if right[key] != leftValue {
			return false
		}
	}
	return true
}
