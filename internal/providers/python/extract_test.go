package python

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

const pythonParityFixture = `from app.models import User as AppUser
import os.path as osp

class Base:
    pass

class Service(Base):
    repo: AppUser

    def __init__(self, repo: AppUser):
        self.repo = repo

    def save(self, user: AppUser) -> bool:
        self.repo.write(user.id)
        return True

def create_service(repo: AppUser) -> Service:
    service = Service(repo)
    service.save(repo)
    return service
`

func TestExtractPythonScopeIR(t *testing.T) {
	ir := parseAndExtract(t, "service/service.py", "hash-python", []byte(pythonParityFixture))

	for _, def := range ir.Definitions {
		if def.FileHash != "hash-python" {
			t.Fatalf("definition %s missing file hash: %#v", def.Name, def)
		}
	}
	for _, call := range ir.Calls {
		if call.FileHash != "hash-python" {
			t.Fatalf("call %s missing file hash: %#v", call.Name, call)
		}
	}

	base := requireDefinition(t, ir, "Base", scopeir.NodeClass)
	service := requireDefinition(t, ir, "Service", scopeir.NodeClass)
	initMethod := requireQualifiedDefinition(t, ir, "Service.__init__", scopeir.NodeMethod)
	saveMethod := requireQualifiedDefinition(t, ir, "Service.save", scopeir.NodeMethod)
	createService := requireDefinition(t, ir, "create_service", scopeir.NodeFunction)
	repo := requireDefinition(t, ir, "repo", scopeir.NodeProperty)
	serviceVar := requireDefinition(t, ir, "service", scopeir.NodeVariable)
	if initMethod.OwnerID != service.ID || saveMethod.OwnerID != service.ID || repo.OwnerID != service.ID {
		t.Fatalf("owner mismatch: base=%s service=%s init=%#v save=%#v repo=%#v", base.ID, service.ID, initMethod, saveMethod, repo)
	}
	if serviceVar.DeclaredType != "Service" || createService.ReturnType != "Service" {
		t.Fatalf("type extraction mismatch: service var=%#v create_service=%#v", serviceVar, createService)
	}

	requireImport(t, ir, scopeir.ImportAlias, "AppUser", "User", "app.models.User")
	requireImport(t, ir, scopeir.ImportAlias, "osp", "path", "os.path")
	requireCall(t, ir, "Service", scopeir.CallConstructor)
	requireCall(t, ir, "save", scopeir.CallMember)
	requireCall(t, ir, "write", scopeir.CallMember)
	requireAccess(t, ir, "repo", scopeir.AccessWrite)
	requireAccess(t, ir, "repo", scopeir.AccessRead)
	requireAccess(t, ir, "id", scopeir.AccessRead)
	requireHeritage(t, ir, "Base", scopeir.HeritageExtends)
	requireTypeBinding(t, ir, "self", "Service")
	requireTypeBinding(t, ir, "repo", "AppUser")
	requireTypeBinding(t, ir, "user", "AppUser")
	requireTypeBinding(t, ir, "service", "Service")
	requireNoTypeBinding(t, ir, "cls", "Service")
	requireTypeAnnotation(t, ir, "repo", "AppUser")
	requireTypeAnnotation(t, ir, "save", "bool")
	requireReturnType(t, ir, saveMethod.ID, "bool")
	requireReturnType(t, ir, createService.ID, "Service")
}

func TestExtractPythonScopeIRParityFixture(t *testing.T) {
	ir := parseAndExtract(t, "service/service.py", "hash-python", []byte(pythonParityFixture))
	signature := buildParitySignature(ir)
	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(signature); err != nil {
		t.Fatalf("marshal signature: %v", err)
	}
	raw := buffer.Bytes()
	golden, err := os.ReadFile("testdata/python_scopeir_signature.golden.json")
	if err != nil {
		t.Fatalf("read golden: %v\ngot:\n%s", err, raw)
	}
	if string(raw) != string(golden) {
		t.Fatalf("parity signature mismatch\nwant:\n%s\ngot:\n%s", golden, raw)
	}
}

func TestResolvePythonGraphParityCounts(t *testing.T) {
	ir := parseAndExtract(t, "service/service.py", "hash-python", []byte(pythonParityFixture))
	result, err := resolution.Resolve([]scopeir.ScopeIR{ir}, resolution.Options{})
	if err != nil {
		t.Fatalf("resolve failed: %v", err)
	}
	counts := stringRelationshipCounts(result.Graph.RelationshipCountsByType())
	expected := map[string]int{
		"ACCESSES":     2,
		"CALLS":        2,
		"DEFINES":      7,
		"EXTENDS":      1,
		"HAS_METHOD":   2,
		"HAS_PROPERTY": 1,
		"INHERITS":     1,
		"USES":         1,
	}
	if !stringIntMapsEqual(counts, expected) {
		t.Fatalf("relationship counts mismatch\nwant: %#v\ngot:  %#v", expected, counts)
	}
	if result.Metrics.UnresolvedReferences == 0 {
		t.Fatalf("expected unresolved external references, metrics=%#v", result.Metrics)
	}
	if result.Metrics.ResolvedCalls == 0 || result.Metrics.ResolvedAccesses == 0 || result.Metrics.ResolvedInheritance == 0 {
		t.Fatalf("expected resolved calls/accesses/inheritance, got metrics %#v", result.Metrics)
	}
}

func BenchmarkExtractPythonScopeIR(b *testing.B) {
	source := []byte(pythonParityFixture)
	pool := parser.NewPool(nil, parser.PoolOptions{ParseTimeout: time.Second})
	defer pool.Close()
	parsed, err := pool.Parse(context.Background(), parser.Request{
		FilePath: "service/service.py",
		Language: scanner.Python,
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
			FilePath: "service/service.py",
			FileHash: "hash-python",
			Language: scanner.Python,
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

func BenchmarkParseAndExtractPythonScopeIR(b *testing.B) {
	source := []byte(pythonParityFixture)
	pool := parser.NewPool(nil, parser.PoolOptions{ParseTimeout: time.Second})
	defer pool.Close()

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		parsed, err := pool.Parse(context.Background(), parser.Request{
			FilePath: "service/service.py",
			Language: scanner.Python,
			Source:   source,
		})
		if err != nil {
			b.Fatalf("parse failed: %v", err)
		}
		ir, err := Extract(Request{
			FilePath: "service/service.py",
			FileHash: "hash-python",
			Language: scanner.Python,
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
		if item.Name == item.Type.RawName || item.Name == basePythonType(item.Type.RawName) || item.Type.RawName != "" {
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
		Language: scanner.Python,
		Source:   source,
	})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	defer parsed.Close()

	ir, err := Extract(Request{
		FilePath: filePath,
		FileHash: fileHash,
		Language: scanner.Python,
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

func requireNoTypeBinding(t *testing.T, ir scopeir.ScopeIR, name string, rawName string) {
	t.Helper()
	for _, scope := range ir.Scopes {
		for _, binding := range scope.TypeBindings {
			if binding.Name == name && binding.Type.RawName == rawName {
				t.Fatalf("unexpected type binding %s -> %s in %#v", name, rawName, ir.Scopes)
			}
		}
	}
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
