package swift

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

const swiftParityFixture = `import Foundation

protocol Named {
  var id: String { get }
  func label() -> String
}

class Repository {
  func write(_ value: String) {}
}

class Service: Named {
  let id: String
  let repo: Repository
  init(repo: Repository) {
    self.repo = repo
    self.id = "service"
  }
  func label() -> String {
    return id
  }
  func save(user: Named) -> String {
    let formatted = user.label()
    repo.write(formatted)
    return formatted
  }
}
`

func TestPoolParsesSwiftFixture(t *testing.T) {
	pool := parser.NewPool(nil, parser.PoolOptions{ParseTimeout: time.Second})
	defer pool.Close()

	result, err := pool.Parse(context.Background(), parser.Request{
		FilePath: "Sources/App/Service.swift",
		Language: scanner.Swift,
		Source:   []byte(swiftParityFixture),
	})
	if err != nil {
		t.Fatalf("Parse Swift failed: %v", err)
	}
	defer result.Close()
	if result.RootKind != "source_file" || result.HasError {
		t.Fatalf("unexpected Swift parse result: %#v", result)
	}
}

func TestExtractSwiftScopeIR(t *testing.T) {
	ir := extract(t, "Sources/App/Service.swift", "hash-swift", []byte(swiftParityFixture))

	if ir.Language != scanner.Swift {
		t.Fatalf("language = %q, want %q", ir.Language, scanner.Swift)
	}
	for _, def := range ir.Definitions {
		if def.FileHash != "hash-swift" {
			t.Fatalf("definition %s missing file hash: %#v", def.Name, def)
		}
	}
	for _, call := range ir.Calls {
		if call.FileHash != "hash-swift" {
			t.Fatalf("call %s missing file hash: %#v", call.Name, call)
		}
	}

	named := requireDefinition(t, ir, "Named", scopeir.NodeInterface)
	repository := requireDefinition(t, ir, "Repository", scopeir.NodeClass)
	service := requireDefinition(t, ir, "Service", scopeir.NodeClass)
	requireQualifiedDefinition(t, ir, "Repository.write", scopeir.NodeMethod)
	constructor := requireQualifiedDefinition(t, ir, "Service.init", scopeir.NodeConstructor)
	label := requireQualifiedDefinition(t, ir, "Service.label", scopeir.NodeMethod)
	save := requireQualifiedDefinition(t, ir, "Service.save", scopeir.NodeMethod)
	id := requireQualifiedDefinition(t, ir, "Service.id", scopeir.NodeProperty)
	repo := requireQualifiedDefinition(t, ir, "Service.repo", scopeir.NodeProperty)
	formatted := requireDefinition(t, ir, "formatted", scopeir.NodeVariable)
	if named.ID == "" || repository.ID == "" || service.ID == "" || label.ID == "" || id.ID == "" || repo.ID == "" || formatted.ID == "" {
		t.Fatal("expected all Swift definitions to have stable IDs")
	}
	if constructor.OwnerID != service.ID || save.OwnerID != service.ID || id.OwnerID != service.ID || repo.OwnerID != service.ID {
		t.Fatalf("owner mismatch: constructor=%#v save=%#v id=%#v repo=%#v", constructor, save, id, repo)
	}
	if save.ReturnType != "String" || label.ReturnType != "String" || repo.DeclaredType != "Repository" {
		t.Fatalf("type extraction mismatch: save=%#v label=%#v repo=%#v", save, label, repo)
	}

	requireImport(t, ir, scopeir.ImportNamed, "Foundation", "Foundation", "Foundation")
	requireCall(t, ir, "label", scopeir.CallMember)
	requireCall(t, ir, "write", scopeir.CallMember)
	requireAccess(t, ir, "repo", scopeir.AccessWrite)
	requireAccess(t, ir, "id", scopeir.AccessWrite)
	requireHeritage(t, ir, "Named", scopeir.HeritageImplements)
	requireTypeBinding(t, ir, "self", "Service")
	requireTypeBinding(t, ir, "repo", "Repository")
	requireTypeBinding(t, ir, "user", "Named")
	requireTypeAnnotation(t, ir, "user")
	requireReturnType(t, ir, save.ID, "String")
}

func TestExtractSwiftRejectsNonSwiftLanguage(t *testing.T) {
	_, err := Extract(Request{
		FilePath: "src/service.ts",
		FileHash: "hash-ts",
		Language: scanner.TypeScript,
		Source:   []byte("class Service {}"),
	})
	if err == nil {
		t.Fatal("expected non-Swift language to fail")
	}
}

func TestExtractSwiftScopeIRParityFixture(t *testing.T) {
	ir := extract(t, "Sources/App/Service.swift", "hash-swift", []byte(swiftParityFixture))
	signature := buildParitySignature(ir)
	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(signature); err != nil {
		t.Fatalf("marshal signature: %v", err)
	}
	raw := buffer.Bytes()
	golden, err := os.ReadFile("testdata/swift_scopeir_signature.golden.json")
	if err != nil {
		t.Fatalf("read golden: %v\ngot:\n%s", err, raw)
	}
	if string(raw) != string(golden) {
		t.Fatalf("parity signature mismatch\nwant:\n%s\ngot:\n%s", golden, raw)
	}
}

func TestResolveSwiftGraphParityCounts(t *testing.T) {
	ir := extract(t, "Sources/App/Service.swift", "hash-swift", []byte(swiftParityFixture))
	result, err := resolution.Resolve([]scopeir.ScopeIR{ir}, resolution.Options{})
	if err != nil {
		t.Fatalf("resolve failed: %v", err)
	}
	counts := stringRelationshipCounts(result.Graph.RelationshipCountsByType())
	expected := map[string]int{
		"ACCESSES":          2,
		"CALLS":             2,
		"DEFINES":           12,
		"HAS_METHOD":        5,
		"HAS_PROPERTY":      3,
		"IMPLEMENTS":        1,
		"INHERITS":          1,
		"METHOD_IMPLEMENTS": 1,
		"USES":              3,
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

func BenchmarkExtractSwiftScopeIR(b *testing.B) {
	parsed := parseSwiftFixture(b)
	defer parsed.Close()
	source := []byte(swiftParityFixture)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		ir, err := Extract(Request{
			FilePath: "Sources/App/Service.swift",
			FileHash: "hash-swift",
			Language: scanner.Swift,
			Source:   source,
			Root:     parsed.Tree.RootNode(),
		})
		if err != nil {
			b.Fatalf("extract failed: %v", err)
		}
		if len(ir.Definitions) == 0 || len(ir.Calls) == 0 {
			b.Fatalf("incomplete extraction: %#v", ir)
		}
	}
}

func BenchmarkParseAndExtractSwiftScopeIR(b *testing.B) {
	source := []byte(swiftParityFixture)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		pool := parser.NewPool(nil, parser.PoolOptions{ParseTimeout: time.Second})
		parsed, err := pool.Parse(context.Background(), parser.Request{
			FilePath: "Sources/App/Service.swift",
			Language: scanner.Swift,
			Source:   source,
		})
		if err != nil {
			pool.Close()
			b.Fatalf("parse failed: %v", err)
		}
		ir, err := Extract(Request{
			FilePath: "Sources/App/Service.swift",
			FileHash: "hash-swift",
			Language: scanner.Swift,
			Source:   source,
			Root:     parsed.Tree.RootNode(),
		})
		parsed.Close()
		pool.Close()
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
		if item.Name == item.Type.RawName {
			signature.TypeReferences = append(signature.TypeReferences, item.Name)
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

func extract(t *testing.T, filePath string, fileHash string, source []byte) scopeir.ScopeIR {
	t.Helper()
	pool := parser.NewPool(nil, parser.PoolOptions{ParseTimeout: time.Second})
	defer pool.Close()
	parsed, err := pool.Parse(context.Background(), parser.Request{FilePath: filePath, Language: scanner.Swift, Source: source})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	defer parsed.Close()
	ir, err := Extract(Request{
		FilePath: filePath,
		FileHash: fileHash,
		Language: scanner.Swift,
		Source:   source,
		Root:     parsed.Tree.RootNode(),
	})
	if err != nil {
		t.Fatalf("extract failed: %v", err)
	}
	return ir
}

func parseSwiftFixture(b *testing.B) *parser.Result {
	b.Helper()
	pool := parser.NewPool(nil, parser.PoolOptions{ParseTimeout: time.Second})
	parsed, err := pool.Parse(context.Background(), parser.Request{
		FilePath: "Sources/App/Service.swift",
		Language: scanner.Swift,
		Source:   []byte(swiftParityFixture),
	})
	pool.Close()
	if err != nil {
		b.Fatalf("parse failed: %v", err)
	}
	return parsed
}

func formatOptionalInt(value *int) string {
	if value == nil {
		return ""
	}
	return strconv.Itoa(*value)
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

func requireTypeAnnotation(t *testing.T, ir scopeir.ScopeIR, name string) {
	t.Helper()
	for _, item := range ir.TypeAnnotations {
		if item.Name == name {
			return
		}
	}
	t.Fatalf("missing type annotation %s in %#v", name, ir.TypeAnnotations)
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
