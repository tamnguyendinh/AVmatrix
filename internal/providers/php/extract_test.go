package php

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

const phpParityFixture = `<?php
require_once 'vendor/autoload.php';

namespace App\Service;

use App\Repository;
use App\Base;
use App\Named;

interface Named {
    public function save(User $user): bool;
}

class Base {
    public string $id;
}

class User {
    public string $id;
}

class Repository {
    public function write(string $id): void {}
}

class Service extends Base implements Named {
    private Repository $repo;

    public function __construct(Repository $repo) {
        $this->repo = $repo;
    }

    public function save(User $user): bool {
        $id = $this->normalize($user);
        $this->repo->write($id);
        return true;
    }

    private function normalize(User $user): string {
        return trim($user->id);
    }
}
`

func TestExtractPHPScopeIR(t *testing.T) {
	ir := parseAndExtract(t, "src/Service.php", "hash-php", []byte(phpParityFixture))

	for _, def := range ir.Definitions {
		if def.FileHash != "hash-php" {
			t.Fatalf("definition %s missing file hash: %#v", def.Name, def)
		}
	}
	for _, call := range ir.Calls {
		if call.FileHash != "hash-php" {
			t.Fatalf("call %s missing file hash: %#v", call.Name, call)
		}
	}

	requireDefinition(t, ir, "App\\Service", scopeir.NodePackage)
	named := requireDefinition(t, ir, "Named", scopeir.NodeInterface)
	base := requireDefinition(t, ir, "Base", scopeir.NodeClass)
	requireDefinition(t, ir, "User", scopeir.NodeClass)
	repository := requireDefinition(t, ir, "Repository", scopeir.NodeClass)
	service := requireDefinition(t, ir, "Service", scopeir.NodeClass)
	requireQualifiedDefinition(t, ir, "Named.save", scopeir.NodeMethod)
	requireQualifiedDefinition(t, ir, "Repository.write", scopeir.NodeMethod)
	constructor := requireQualifiedDefinition(t, ir, "Service.__construct", scopeir.NodeConstructor)
	save := requireQualifiedDefinition(t, ir, "Service.save", scopeir.NodeMethod)
	normalize := requireQualifiedDefinition(t, ir, "Service.normalize", scopeir.NodeMethod)
	baseID := requireQualifiedDefinition(t, ir, "Base.id", scopeir.NodeProperty)
	userID := requireQualifiedDefinition(t, ir, "User.id", scopeir.NodeProperty)
	repo := requireQualifiedDefinition(t, ir, "Service.repo", scopeir.NodeProperty)
	id := requireDefinition(t, ir, "id", scopeir.NodeVariable)
	if named.ID == "" || base.ID == "" || repository.ID == "" || service.ID == "" || baseID.ID == "" || userID.ID == "" || repo.ID == "" {
		t.Fatal("expected all PHP definitions to have stable IDs")
	}
	if constructor.OwnerID != service.ID || save.OwnerID != service.ID || normalize.OwnerID != service.ID || repo.OwnerID != service.ID {
		t.Fatalf("owner mismatch: constructor=%#v save=%#v normalize=%#v repo=%#v", constructor, save, normalize, repo)
	}
	if save.ReturnType != "bool" || normalize.ReturnType != "string" || id.DeclaredType != "string" || repo.DeclaredType != "Repository" {
		t.Fatalf("type extraction mismatch: save=%#v normalize=%#v id=%#v repo=%#v", save, normalize, id, repo)
	}

	requireImport(t, ir, scopeir.ImportDynamicUnresolved, "vendor/autoload.php", "vendor/autoload.php", "vendor/autoload.php")
	requireImport(t, ir, scopeir.ImportNamed, "Repository", "Repository", "App\\Repository")
	requireImport(t, ir, scopeir.ImportNamed, "Base", "Base", "App\\Base")
	requireImport(t, ir, scopeir.ImportNamed, "Named", "Named", "App\\Named")
	requireCall(t, ir, "normalize", scopeir.CallMember)
	requireCall(t, ir, "write", scopeir.CallMember)
	requireCall(t, ir, "trim", scopeir.CallFree)
	requireAccess(t, ir, "repo", scopeir.AccessWrite)
	requireAccess(t, ir, "repo", scopeir.AccessRead)
	requireAccess(t, ir, "id", scopeir.AccessRead)
	requireHeritage(t, ir, "Base", scopeir.HeritageExtends)
	requireHeritage(t, ir, "Named", scopeir.HeritageImplements)
	requireTypeBinding(t, ir, "this", "Service")
	requireTypeBinding(t, ir, "repo", "Repository")
	requireTypeBinding(t, ir, "user", "User")
	requireTypeBinding(t, ir, "id", "string")
	requireTypeAnnotation(t, ir, "repo", "Repository")
	requireReturnType(t, ir, save.ID, "bool")
	requireReturnType(t, ir, normalize.ID, "string")
}

func TestExtractPHPScopeIRParityFixture(t *testing.T) {
	ir := parseAndExtract(t, "src/Service.php", "hash-php", []byte(phpParityFixture))
	signature := buildParitySignature(ir)
	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(signature); err != nil {
		t.Fatalf("marshal signature: %v", err)
	}
	raw := buffer.Bytes()
	golden, err := os.ReadFile("testdata/php_scopeir_signature.golden.json")
	if err != nil {
		t.Fatalf("read golden: %v\ngot:\n%s", err, raw)
	}
	if string(raw) != string(golden) {
		t.Fatalf("parity signature mismatch\nwant:\n%s\ngot:\n%s", golden, raw)
	}
}

func TestResolvePHPGraphParityCounts(t *testing.T) {
	ir := parseAndExtract(t, "src/Service.php", "hash-php", []byte(phpParityFixture))
	result, err := resolution.Resolve([]scopeir.ScopeIR{ir}, resolution.Options{})
	if err != nil {
		t.Fatalf("resolve failed: %v", err)
	}
	counts := stringRelationshipCounts(result.Graph.RelationshipCountsByType())
	expected := map[string]int{
		"ACCESSES":          3,
		"CALLS":             2,
		"DEFINES":           15,
		"EXTENDS":           1,
		"HAS_METHOD":        5,
		"HAS_PROPERTY":      3,
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
	if result.Metrics.ResolvedCalls == 0 || result.Metrics.ResolvedAccesses == 0 || result.Metrics.ResolvedInheritance == 0 {
		t.Fatalf("expected resolved calls/accesses/heritage, got metrics %#v", result.Metrics)
	}
}

func TestExtractPHPLegacyUseFunctionAndConstDoNotCreateClassImports(t *testing.T) {
	ir := parseAndExtract(t, "app/Services/Calculator.php", "hash-php-use-function", []byte(`<?php
namespace App\Services;

use App\Models\User;
use function App\Utils\formatName;
use const App\Utils\MAX_USERS;

class Calculator {
    public function process(User $user): void {
        formatName($user->name);
    }
}
`))

	requireImport(t, ir, scopeir.ImportNamed, "User", "User", `App\Models\User`)
	for _, item := range ir.Imports {
		if item.LocalName == "formatName" || item.LocalName == "MAX_USERS" {
			t.Fatalf("unexpected function/const import emitted: %#v", item)
		}
	}
}

func TestResolvePHPLegacyAliasGroupedImportsAndReceiverBinding(t *testing.T) {
	files := parsePHPWorkspace(t, map[string]string{
		"app/Models/User.php": `<?php
namespace App\Models;

class User {
    public function save(): void {}
}
`,
		"app/Models/Repo.php": `<?php
namespace App\Models;

class Repo {
    public function persist(): void {}
}
`,
		"app/Services/Main.php": `<?php
namespace App\Services;

use App\Models\{User, Repo as R};

class Main {
    public function run(User $u, R $r): void {
        $u->save();
        $r->persist();
    }
}
`,
	})
	result, err := resolution.Resolve(files, resolution.Options{})
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	main := requirePHPGraphNode(t, result.Graph, scopeir.NodeClass, "app/Services/Main.php", "Main")
	user := requirePHPGraphNode(t, result.Graph, scopeir.NodeClass, "app/Models/User.php", "User")
	repo := requirePHPGraphNode(t, result.Graph, scopeir.NodeClass, "app/Models/Repo.php", "Repo")
	run := requirePHPGraphNode(t, result.Graph, scopeir.NodeMethod, "app/Services/Main.php", "Main.run")
	save := requirePHPGraphNode(t, result.Graph, scopeir.NodeMethod, "app/Models/User.php", "User.save")
	persist := requirePHPGraphNode(t, result.Graph, scopeir.NodeMethod, "app/Models/Repo.php", "Repo.persist")

	requirePHPGraphRelationship(t, result.Graph, graph.RelUses, graph.GenerateID(string(scopeir.NodeFile), "app/Services/Main.php"), user.ID)
	requirePHPGraphRelationship(t, result.Graph, graph.RelUses, graph.GenerateID(string(scopeir.NodeFile), "app/Services/Main.php"), repo.ID)
	requirePHPGraphRelationship(t, result.Graph, graph.RelCalls, run.ID, save.ID)
	requirePHPGraphRelationship(t, result.Graph, graph.RelCalls, run.ID, persist.ID)
	requirePHPGraphRelationship(t, result.Graph, graph.RelDefines, graph.GenerateID(string(scopeir.NodeFile), "app/Services/Main.php"), main.ID)
}

func TestResolvePHPLegacyConstructorReturnBindingAndArityNarrowing(t *testing.T) {
	files := parsePHPWorkspace(t, map[string]string{
		"app/Models/User.php": `<?php
namespace App\Models;

class User {
    public function save(): void {}
}
`,
		"app/Utils/OneArg/log.php": `<?php
namespace App\Utils\OneArg;

function write_audit(string $message): void {}
`,
		"app/Utils/TwoArg/log.php": `<?php
namespace App\Utils\TwoArg;

function write_audit(string $message, string $context): void {}
`,
		"app/Services/Main.php": `<?php
namespace App\Services;

use App\Models\User;
use function App\Utils\OneArg\write_audit;

class Main {
    public function run(): void {
        $user = new User();
        $user->save();
        write_audit("created");
    }
}
`,
	})
	result, err := resolution.Resolve(files, resolution.Options{})
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	run := requirePHPGraphNode(t, result.Graph, scopeir.NodeMethod, "app/Services/Main.php", "Main.run")
	userClass := requirePHPGraphNode(t, result.Graph, scopeir.NodeClass, "app/Models/User.php", "User")
	save := requirePHPGraphNode(t, result.Graph, scopeir.NodeMethod, "app/Models/User.php", "User.save")
	oneArgAudit := requirePHPGraphNode(t, result.Graph, scopeir.NodeFunction, "app/Utils/OneArg/log.php", "write_audit")
	twoArgAudit := requirePHPGraphNode(t, result.Graph, scopeir.NodeFunction, "app/Utils/TwoArg/log.php", "write_audit")

	requirePHPGraphRelationship(t, result.Graph, graph.RelCalls, run.ID, userClass.ID)
	requirePHPGraphRelationship(t, result.Graph, graph.RelCalls, run.ID, save.ID)
	requirePHPGraphRelationship(t, result.Graph, graph.RelCalls, run.ID, oneArgAudit.ID)
	for _, relationship := range result.Graph.Relationships {
		if relationship.Type == graph.RelCalls && relationship.SourceID == run.ID && relationship.TargetID == twoArgAudit.ID {
			t.Fatalf("unexpected arity-mismatched CALLS edge to two-arg write_audit: %#v", relationship)
		}
	}
}

func BenchmarkExtractPHPScopeIR(b *testing.B) {
	source := []byte(phpParityFixture)
	pool := parser.NewPool(nil, parser.PoolOptions{ParseTimeout: time.Second})
	defer pool.Close()
	parsed, err := pool.Parse(context.Background(), parser.Request{
		FilePath: "src/Service.php",
		Language: scanner.PHP,
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
			FilePath: "src/Service.php",
			FileHash: "hash-php",
			Language: scanner.PHP,
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

func BenchmarkParseAndExtractPHPScopeIR(b *testing.B) {
	source := []byte(phpParityFixture)
	pool := parser.NewPool(nil, parser.PoolOptions{ParseTimeout: time.Second})
	defer pool.Close()

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		parsed, err := pool.Parse(context.Background(), parser.Request{
			FilePath: "src/Service.php",
			Language: scanner.PHP,
			Source:   source,
		})
		if err != nil {
			b.Fatalf("parse failed: %v", err)
		}
		ir, err := Extract(Request{
			FilePath: "src/Service.php",
			FileHash: "hash-php",
			Language: scanner.PHP,
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
		Language: scanner.PHP,
		Source:   source,
	})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	defer parsed.Close()

	ir, err := Extract(Request{
		FilePath: filePath,
		FileHash: fileHash,
		Language: scanner.PHP,
		Source:   source,
		Root:     parsed.Tree.RootNode(),
	})
	if err != nil {
		t.Fatalf("extract failed: %v", err)
	}
	return ir
}

func parsePHPWorkspace(t *testing.T, files map[string]string) []scopeir.ScopeIR {
	t.Helper()
	paths := make([]string, 0, len(files))
	for filePath := range files {
		paths = append(paths, filePath)
	}
	sort.Strings(paths)
	irs := make([]scopeir.ScopeIR, 0, len(paths))
	for _, filePath := range paths {
		irs = append(irs, parseAndExtract(t, filePath, "hash-"+filePath, []byte(files[filePath])))
	}
	return irs
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

func requirePHPGraphNode(t *testing.T, g *graph.Graph, label scopeir.NodeLabel, filePath string, qualifiedName string) graph.Node {
	t.Helper()
	for _, node := range g.Nodes {
		if node.Label == label && node.Properties["filePath"] == filePath && node.Properties["qualifiedName"] == qualifiedName {
			return node
		}
	}
	t.Fatalf("missing graph node %s %s %s", label, filePath, qualifiedName)
	return graph.Node{}
}

func requirePHPGraphRelationship(t *testing.T, g *graph.Graph, relType graph.RelationshipType, sourceID string, targetID string) graph.Relationship {
	t.Helper()
	for _, relationship := range g.Relationships {
		if relationship.Type == relType && relationship.SourceID == sourceID && relationship.TargetID == targetID {
			return relationship
		}
	}
	t.Fatalf("missing graph relationship %s %s -> %s", relType, sourceID, targetID)
	return graph.Relationship{}
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
