package java

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

const javaParityFixture = `package app.service;

import java.util.UUID;
import static app.util.Names.format;

public interface Named {
    boolean save(User user);
}

public class Base {}

public class User {
    public String id() {
        return "id";
    }
}

public class Repository {
    public void write(String id) {}
}

public class Service extends Base implements Named {
    private final Repository repo;

    public Service(Repository repo) {
        this.repo = repo;
    }

    @Override
    public boolean save(User user) {
        String id = format(normalize(user));
        this.repo.write(id);
        return true;
    }

    private String normalize(User user) {
        return user.id();
    }
}
`

func TestExtractJavaScopeIR(t *testing.T) {
	ir := parseAndExtract(t, "src/app/service/Service.java", "hash-java", []byte(javaParityFixture))

	for _, def := range ir.Definitions {
		if def.FileHash != "hash-java" {
			t.Fatalf("definition %s missing file hash: %#v", def.Name, def)
		}
	}
	for _, call := range ir.Calls {
		if call.FileHash != "hash-java" {
			t.Fatalf("call %s missing file hash: %#v", call.Name, call)
		}
	}

	requireDefinition(t, ir, "app.service", scopeir.NodePackage)
	named := requireDefinition(t, ir, "Named", scopeir.NodeInterface)
	base := requireDefinition(t, ir, "Base", scopeir.NodeClass)
	requireDefinition(t, ir, "User", scopeir.NodeClass)
	requireDefinition(t, ir, "Repository", scopeir.NodeClass)
	service := requireDefinition(t, ir, "Service", scopeir.NodeClass)
	requireQualifiedDefinition(t, ir, "Named.save", scopeir.NodeMethod)
	requireQualifiedDefinition(t, ir, "User.id", scopeir.NodeMethod)
	requireQualifiedDefinition(t, ir, "Repository.write", scopeir.NodeMethod)
	constructor := requireQualifiedDefinition(t, ir, "Service.Service", scopeir.NodeConstructor)
	save := requireQualifiedDefinition(t, ir, "Service.save", scopeir.NodeMethod)
	normalize := requireQualifiedDefinition(t, ir, "Service.normalize", scopeir.NodeMethod)
	repo := requireQualifiedDefinition(t, ir, "Service.repo", scopeir.NodeProperty)
	id := requireDefinition(t, ir, "id", scopeir.NodeVariable)
	if constructor.OwnerID != service.ID || save.OwnerID != service.ID || normalize.OwnerID != service.ID || repo.OwnerID != service.ID {
		t.Fatalf("owner mismatch: named=%s base=%s service=%s constructor=%#v save=%#v normalize=%#v repo=%#v", named.ID, base.ID, service.ID, constructor, save, normalize, repo)
	}
	if id.DeclaredType != "String" || save.ReturnType != "boolean" {
		t.Fatalf("type extraction mismatch: id=%#v save=%#v", id, save)
	}

	requireImport(t, ir, scopeir.ImportNamed, "UUID", "UUID", "java.util.UUID")
	requireImport(t, ir, scopeir.ImportNamed, "format", "format", "app.util.Names.format")
	requireCall(t, ir, "format", scopeir.CallMember)
	requireCall(t, ir, "normalize", scopeir.CallMember)
	requireCall(t, ir, "id", scopeir.CallMember)
	requireCall(t, ir, "write", scopeir.CallMember)
	requireAccess(t, ir, "repo", scopeir.AccessWrite)
	requireHeritage(t, ir, "Base", scopeir.HeritageExtends)
	requireHeritage(t, ir, "Named", scopeir.HeritageImplements)
	requireTypeBinding(t, ir, "this", "Service")
	requireTypeBinding(t, ir, "repo", "Repository")
	requireTypeBinding(t, ir, "user", "User")
	requireTypeBinding(t, ir, "id", "String")
	requireTypeAnnotation(t, ir, "repo", "Repository")
	requireReturnType(t, ir, save.ID, "boolean")
}

func TestExtractJavaScopeIRParityFixture(t *testing.T) {
	ir := parseAndExtract(t, "src/app/service/Service.java", "hash-java", []byte(javaParityFixture))
	signature := buildParitySignature(ir)
	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(signature); err != nil {
		t.Fatalf("marshal signature: %v", err)
	}
	raw := buffer.Bytes()
	golden, err := os.ReadFile("testdata/java_scopeir_signature.golden.json")
	if err != nil {
		t.Fatalf("read golden: %v\ngot:\n%s", err, raw)
	}
	if string(raw) != string(golden) {
		t.Fatalf("parity signature mismatch\nwant:\n%s\ngot:\n%s", golden, raw)
	}
}

func TestResolveJavaGraphParityCounts(t *testing.T) {
	ir := parseAndExtract(t, "src/app/service/Service.java", "hash-java", []byte(javaParityFixture))
	result, err := resolution.Resolve([]scopeir.ScopeIR{ir}, resolution.Options{})
	if err != nil {
		t.Fatalf("resolve failed: %v", err)
	}
	counts := stringRelationshipCounts(result.Graph.RelationshipCountsByType())
	expected := map[string]int{
		"ACCESSES":          1,
		"CALLS":             3,
		"DEFINES":           14,
		"EXTENDS":           1,
		"HAS_METHOD":        6,
		"HAS_PROPERTY":      1,
		"IMPLEMENTS":        1,
		"INHERITS":          2,
		"METHOD_IMPLEMENTS": 1,
		"USES":              5,
	}
	if !stringIntMapsEqual(counts, expected) {
		t.Fatalf("relationship counts mismatch\nwant: %#v\ngot:  %#v", expected, counts)
	}
	if result.Metrics.UnresolvedReferences == 0 {
		t.Fatalf("expected unresolved external references, metrics=%#v", result.Metrics)
	}
	if result.Metrics.ResolvedCalls == 0 || result.Metrics.ResolvedInheritance == 0 {
		t.Fatalf("expected resolved calls/inheritance, got metrics %#v", result.Metrics)
	}
}

func BenchmarkExtractJavaScopeIR(b *testing.B) {
	source := []byte(javaParityFixture)
	pool := parser.NewPool(nil, parser.PoolOptions{ParseTimeout: time.Second})
	defer pool.Close()
	parsed, err := pool.Parse(context.Background(), parser.Request{
		FilePath: "src/app/service/Service.java",
		Language: scanner.Java,
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
			FilePath: "src/app/service/Service.java",
			FileHash: "hash-java",
			Language: scanner.Java,
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

func BenchmarkParseAndExtractJavaScopeIR(b *testing.B) {
	source := []byte(javaParityFixture)
	pool := parser.NewPool(nil, parser.PoolOptions{ParseTimeout: time.Second})
	defer pool.Close()

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		parsed, err := pool.Parse(context.Background(), parser.Request{
			FilePath: "src/app/service/Service.java",
			Language: scanner.Java,
			Source:   source,
		})
		if err != nil {
			b.Fatalf("parse failed: %v", err)
		}
		ir, err := Extract(Request{
			FilePath: "src/app/service/Service.java",
			FileHash: "hash-java",
			Language: scanner.Java,
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
		if item.Name == item.Type.RawName || item.Name == baseJavaType(item.Type.RawName) || item.Type.RawName != "" {
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
		Language: scanner.Java,
		Source:   source,
	})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	defer parsed.Close()

	ir, err := Extract(Request{
		FilePath: filePath,
		FileHash: fileHash,
		Language: scanner.Java,
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
