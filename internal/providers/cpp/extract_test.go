package cpp

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

const cppParityFixture = `#include "repo.hpp"
#include <string>

namespace app {
class Named {
public:
    virtual bool save(const std::string& value) = 0;
};

class Base {
public:
    int id;
};

class Repository {
public:
    void write(const std::string& value) {}
};

class Service : public Base, public Named {
private:
    Repository repo;

public:
    Service(Repository repo) : repo(repo) {}

    bool save(const std::string& value) {
        auto formatted = normalize(value);
        this->repo.write(formatted);
        return true;
    }

private:
    std::string normalize(const std::string& value) {
        return value;
    }
};
}
`

func TestExtractCPPScopeIR(t *testing.T) {
	ir := parseAndExtract(t, "src/service.cpp", "hash-cpp", []byte(cppParityFixture))

	for _, def := range ir.Definitions {
		if def.FileHash != "hash-cpp" {
			t.Fatalf("definition %s missing file hash: %#v", def.Name, def)
		}
	}
	for _, call := range ir.Calls {
		if call.FileHash != "hash-cpp" {
			t.Fatalf("call %s missing file hash: %#v", call.Name, call)
		}
	}

	requireDefinition(t, ir, "app", scopeir.NodePackage)
	named := requireDefinition(t, ir, "Named", scopeir.NodeClass)
	base := requireDefinition(t, ir, "Base", scopeir.NodeClass)
	repository := requireDefinition(t, ir, "Repository", scopeir.NodeClass)
	service := requireDefinition(t, ir, "Service", scopeir.NodeClass)
	saveDecl := requireQualifiedDefinition(t, ir, "Named.save", scopeir.NodeMethod)
	id := requireQualifiedDefinition(t, ir, "Base.id", scopeir.NodeProperty)
	write := requireQualifiedDefinition(t, ir, "Repository.write", scopeir.NodeMethod)
	repo := requireQualifiedDefinition(t, ir, "Service.repo", scopeir.NodeProperty)
	ctor := requireQualifiedDefinition(t, ir, "Service.Service", scopeir.NodeConstructor)
	save := requireQualifiedDefinition(t, ir, "Service.save", scopeir.NodeMethod)
	normalize := requireQualifiedDefinition(t, ir, "Service.normalize", scopeir.NodeMethod)
	formatted := requireDefinition(t, ir, "formatted", scopeir.NodeVariable)
	if named.ID == "" || base.ID == "" || repository.ID == "" || service.ID == "" || saveDecl.ID == "" || id.ID == "" || write.ID == "" || repo.ID == "" || ctor.ID == "" || save.ID == "" || normalize.ID == "" || formatted.ID == "" {
		t.Fatal("expected all C++ definitions to have stable IDs")
	}
	if id.OwnerID != base.ID || write.OwnerID != repository.ID || repo.OwnerID != service.ID || ctor.OwnerID != service.ID || save.OwnerID != service.ID || normalize.OwnerID != service.ID {
		t.Fatalf("owner mismatch: id=%#v write=%#v repo=%#v ctor=%#v save=%#v normalize=%#v", id, write, repo, ctor, save, normalize)
	}
	if save.ReturnType != "bool" || normalize.ReturnType != "std::string" || formatted.DeclaredType != "auto" {
		t.Fatalf("type extraction mismatch: save=%#v normalize=%#v formatted=%#v", save, normalize, formatted)
	}

	requireImport(t, ir, scopeir.ImportNamed, "repo.hpp", "repo.hpp", "repo.hpp")
	requireImport(t, ir, scopeir.ImportNamed, "string", "string", "string")
	requireCall(t, ir, "normalize", scopeir.CallFree)
	requireCall(t, ir, "write", scopeir.CallMember)
	requireAccess(t, ir, "repo", scopeir.AccessRead)
	requireAccess(t, ir, "write", scopeir.AccessRead)
	requireHeritage(t, ir, "Base", scopeir.HeritageExtends)
	requireHeritage(t, ir, "Named", scopeir.HeritageExtends)
	requireTypeBinding(t, ir, "this", "Service")
	requireTypeBinding(t, ir, "repo", "Repository")
	requireTypeBinding(t, ir, "value", "std::string")
	requireTypeBinding(t, ir, "formatted", "auto")
	requireTypeAnnotation(t, ir, "repo", "Repository")
	requireReturnType(t, ir, save.ID, "bool")
	requireReturnType(t, ir, normalize.ID, "std::string")
}

func TestExtractCPPScopeIRParityFixture(t *testing.T) {
	ir := parseAndExtract(t, "src/service.cpp", "hash-cpp", []byte(cppParityFixture))
	signature := buildParitySignature(ir)
	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(signature); err != nil {
		t.Fatalf("marshal signature: %v", err)
	}
	raw := buffer.Bytes()
	golden, err := os.ReadFile("testdata/cpp_scopeir_signature.golden.json")
	if err != nil {
		t.Fatalf("read golden: %v\ngot:\n%s", err, raw)
	}
	if string(raw) != string(golden) {
		t.Fatalf("parity signature mismatch\nwant:\n%s\ngot:\n%s", golden, raw)
	}
}

func TestResolveCPPGraphParityCounts(t *testing.T) {
	ir := parseAndExtract(t, "src/service.cpp", "hash-cpp", []byte(cppParityFixture))
	result, err := resolution.Resolve([]scopeir.ScopeIR{ir}, resolution.Options{})
	if err != nil {
		t.Fatalf("resolve failed: %v", err)
	}
	counts := stringRelationshipCounts(result.Graph.RelationshipCountsByType())
	expected := map[string]int{
		"ACCESSES":         1,
		"CALLS":            1,
		"DEFINES":          13,
		"EXTENDS":          2,
		"HAS_METHOD":       5,
		"HAS_PROPERTY":     2,
		"INHERITS":         2,
		"METHOD_OVERRIDES": 1,
		"USES":             2,
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

func BenchmarkExtractCPPScopeIR(b *testing.B) {
	source := []byte(cppParityFixture)
	pool := parser.NewPool(nil, parser.PoolOptions{ParseTimeout: time.Second})
	defer pool.Close()
	parsed, err := pool.Parse(context.Background(), parser.Request{
		FilePath: "src/service.cpp",
		Language: scanner.CPlusPlus,
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
			FilePath: "src/service.cpp",
			FileHash: "hash-cpp",
			Language: scanner.CPlusPlus,
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

func BenchmarkParseAndExtractCPPScopeIR(b *testing.B) {
	source := []byte(cppParityFixture)
	pool := parser.NewPool(nil, parser.PoolOptions{ParseTimeout: time.Second})
	defer pool.Close()

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		parsed, err := pool.Parse(context.Background(), parser.Request{
			FilePath: "src/service.cpp",
			Language: scanner.CPlusPlus,
			Source:   source,
		})
		if err != nil {
			b.Fatalf("parse failed: %v", err)
		}
		ir, err := Extract(Request{
			FilePath: "src/service.cpp",
			FileHash: "hash-cpp",
			Language: scanner.CPlusPlus,
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
		signature.Heritage = append(signature.Heritage, string(item.Kind)+":"+item.Name+":"+item.InScope)
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
		Language: scanner.CPlusPlus,
		Source:   source,
	})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	defer parsed.Close()

	ir, err := Extract(Request{
		FilePath: filePath,
		FileHash: fileHash,
		Language: scanner.CPlusPlus,
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
