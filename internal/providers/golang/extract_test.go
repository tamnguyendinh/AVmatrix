package golang

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

const goParityFixture = `package service

import (
	"context"
	repoPkg "example.com/app/repo"
)

type Named interface {
	Save(ctx context.Context) error
	Base
}

type Base struct {
	ID string
}

type User struct {
	ID string
}

type Repo struct{}

func (r *Repo) Write(id string) error {
	return nil
}

type Service struct {
	Base
	repo *Repo
	external *repoPkg.Repo
}

func NewService(repo *Repo) *Service {
	return &Service{repo: repo}
}

func (s *Service) Save(ctx context.Context, user User) error {
	s.repo.Write(user.ID)
	return nil
}
`

func TestExtractGoScopeIR(t *testing.T) {
	ir := parseAndExtract(t, "service/service.go", "hash-go", []byte(goParityFixture))

	for _, def := range ir.Definitions {
		if def.FileHash != "hash-go" {
			t.Fatalf("definition %s missing file hash: %#v", def.Name, def)
		}
	}
	for _, call := range ir.Calls {
		if call.FileHash != "hash-go" {
			t.Fatalf("call %s missing file hash: %#v", call.Name, call)
		}
	}

	requireDefinition(t, ir, "service", scopeir.NodePackage)
	requireDefinition(t, ir, "Named", scopeir.NodeInterface)
	requireDefinition(t, ir, "Base", scopeir.NodeStruct)
	requireDefinition(t, ir, "User", scopeir.NodeStruct)
	repo := requireDefinition(t, ir, "Repo", scopeir.NodeStruct)
	service := requireDefinition(t, ir, "Service", scopeir.NodeStruct)
	write := requireQualifiedDefinition(t, ir, "Repo.Write", scopeir.NodeMethod)
	save := requireQualifiedDefinition(t, ir, "Service.Save", scopeir.NodeMethod)
	requireDefinition(t, ir, "NewService", scopeir.NodeFunction)
	requireDefinition(t, ir, "repo", scopeir.NodeProperty)
	requireDefinition(t, ir, "external", scopeir.NodeProperty)
	requireDefinition(t, ir, "ID", scopeir.NodeProperty)
	if write.OwnerID != repo.ID || save.OwnerID != service.ID {
		t.Fatalf("method owners mismatch: write=%#v save=%#v repo=%s service=%s", write, save, repo.ID, service.ID)
	}

	requireImport(t, ir, scopeir.ImportNamed, "context", "context", "context")
	requireImport(t, ir, scopeir.ImportAlias, "repoPkg", "repo", "example.com/app/repo")
	requireCall(t, ir, "Service", scopeir.CallConstructor)
	requireCall(t, ir, "Write", scopeir.CallMember)
	requireAccess(t, ir, "repo", scopeir.AccessRead)
	requireAccess(t, ir, "ID", scopeir.AccessRead)
	requireHeritage(t, ir, "Base", scopeir.HeritageExtends)
	requireTypeBinding(t, ir, "s", "Service")
	requireTypeBinding(t, ir, "repo", "Repo")
	requireTypeBinding(t, ir, "user", "User")
	requireTypeBinding(t, ir, "ctx", "context.Context")
	requireTypeAnnotation(t, ir, "Service")
	requireReturnType(t, ir, save.ID, "error")
}

func TestExtractGoScopeIRParityFixture(t *testing.T) {
	ir := parseAndExtract(t, "service/service.go", "hash-go", []byte(goParityFixture))
	signature := buildParitySignature(ir)
	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(signature); err != nil {
		t.Fatalf("marshal signature: %v", err)
	}
	raw := buffer.Bytes()
	golden, err := os.ReadFile("testdata/go_scopeir_signature.golden.json")
	if err != nil {
		t.Fatalf("read golden: %v\ngot:\n%s", err, raw)
	}
	if string(raw) != string(golden) {
		t.Fatalf("parity signature mismatch\nwant:\n%s\ngot:\n%s", golden, raw)
	}
}

func TestResolveGoGraphParityCounts(t *testing.T) {
	ir := parseAndExtract(t, "service/service.go", "hash-go", []byte(goParityFixture))
	result, err := resolution.Resolve([]scopeir.ScopeIR{ir}, resolution.Options{})
	if err != nil {
		t.Fatalf("resolve failed: %v", err)
	}
	counts := stringRelationshipCounts(result.Graph.RelationshipCountsByType())
	expected := map[string]int{
		"ACCESSES":     2,
		"CALLS":        2,
		"DEFINES":      14,
		"EXTENDS":      2,
		"HAS_METHOD":   3,
		"HAS_PROPERTY": 4,
		"INHERITS":     2,
		"USES":         8,
	}
	if !stringIntMapsEqual(counts, expected) {
		t.Fatalf("relationship counts mismatch\nwant: %#v\ngot:  %#v", expected, counts)
	}
	if result.Metrics.UnresolvedReferences != 4 {
		t.Fatalf("unresolved references = %d, want 4; metrics=%#v", result.Metrics.UnresolvedReferences, result.Metrics)
	}
	if result.Metrics.ResolvedCalls == 0 || result.Metrics.ResolvedAccesses == 0 || result.Metrics.ResolvedInheritance == 0 {
		t.Fatalf("expected resolved calls/accesses/inheritance, got metrics %#v", result.Metrics)
	}
}

func TestExtractGoTypeAliasDeclarations(t *testing.T) {
	const source = `package service

import sfc "github.com/tamnguyendinh/avmatrix-go/internal/providers/sfc"

type Request = sfc.Request
type Named string
type Shape struct{}
type Contract interface {
	Do()
}
`
	ir := parseAndExtract(t, "service/aliases.go", "hash-go", []byte(source))

	request := requireDefinition(t, ir, "Request", scopeir.NodeTypeAlias)
	if request.DeclaredType != "sfc.Request" {
		t.Fatalf("Request declared type = %q, want sfc.Request", request.DeclaredType)
	}
	named := requireDefinition(t, ir, "Named", scopeir.NodeTypeAlias)
	if named.DeclaredType != "string" {
		t.Fatalf("Named declared type = %q, want string", named.DeclaredType)
	}
	requireDefinition(t, ir, "Shape", scopeir.NodeStruct)
	requireDefinition(t, ir, "Contract", scopeir.NodeInterface)
	requireQualifiedDefinition(t, ir, "Contract.Do", scopeir.NodeMethod)
}

func TestExtractGoRangeVariableDeclarations(t *testing.T) {
	const source = `package service

func Use(values []string, table map[string]int) {
	for _, value := range values {
		_ = value
	}
	for key := range table {
		_ = key
	}
	for index, item := range values {
		for nestedKey, nestedValue := range table {
			_, _, _, _ = index, item, nestedKey, nestedValue
		}
	}
	var assignedKey string
	var assignedValue int
	for assignedKey, assignedValue = range table {
		_, _ = assignedKey, assignedValue
	}
}
`
	ir := parseAndExtract(t, "service/range_test.go", "hash-go", []byte(source))

	for _, name := range []string{"value", "key", "index", "item", "nestedKey", "nestedValue"} {
		requireDefinition(t, ir, name, scopeir.NodeVariable)
	}
	if got := countDefinitions(ir, "assignedKey", scopeir.NodeVariable); got != 1 {
		t.Fatalf("assignedKey definitions = %d, want only the var declaration", got)
	}
	if got := countDefinitions(ir, "assignedValue", scopeir.NodeVariable); got != 1 {
		t.Fatalf("assignedValue definitions = %d, want only the var declaration", got)
	}
}

func TestExtractGoSwitchAndReceiveVariableDeclarations(t *testing.T) {
	const source = `package service

type Event struct {
	Type string
}

func Use(value any, events <-chan Event) {
	switch typed := value.(type) {
	case string:
		_ = typed
	default:
		_ = typed
	}
	select {
	case event, ok := <-events:
		_, _ = event, ok
	default:
	}
}
`
	ir := parseAndExtract(t, "service/control_test.go", "hash-go", []byte(source))

	for _, name := range []string{"typed", "event", "ok"} {
		requireDefinition(t, ir, name, scopeir.NodeVariable)
	}
}

func BenchmarkExtractGoScopeIR(b *testing.B) {
	source := []byte(goParityFixture)
	pool := parser.NewPool(nil, parser.PoolOptions{ParseTimeout: time.Second})
	defer pool.Close()
	parsed, err := pool.Parse(context.Background(), parser.Request{
		FilePath: "service/service.go",
		Language: scanner.Go,
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
			FilePath: "service/service.go",
			FileHash: "hash-go",
			Language: scanner.Go,
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

func BenchmarkParseAndExtractGoScopeIR(b *testing.B) {
	source := []byte(goParityFixture)
	pool := parser.NewPool(nil, parser.PoolOptions{ParseTimeout: time.Second})
	defer pool.Close()

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		parsed, err := pool.Parse(context.Background(), parser.Request{
			FilePath: "service/service.go",
			Language: scanner.Go,
			Source:   source,
		})
		if err != nil {
			b.Fatalf("parse failed: %v", err)
		}
		ir, err := Extract(Request{
			FilePath: "service/service.go",
			FileHash: "hash-go",
			Language: scanner.Go,
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
		if item.Name == item.Type.RawName || item.Name == baseGoType(item.Type.RawName) {
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
		Language: scanner.Go,
		Source:   source,
	})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	defer parsed.Close()

	ir, err := Extract(Request{
		FilePath: filePath,
		FileHash: fileHash,
		Language: scanner.Go,
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

func countDefinitions(ir scopeir.ScopeIR, name string, label scopeir.NodeLabel) int {
	count := 0
	for _, def := range ir.Definitions {
		if def.Name == name && def.Label == label {
			count++
		}
	}
	return count
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
