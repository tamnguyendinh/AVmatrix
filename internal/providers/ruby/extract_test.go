package ruby

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/tamnguyendinh/avmatrix-go/internal/graph"
	"github.com/tamnguyendinh/avmatrix-go/internal/parser"
	"github.com/tamnguyendinh/avmatrix-go/internal/resolution"
	"github.com/tamnguyendinh/avmatrix-go/internal/scanner"
	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

const rubyParityFixture = `require "json"

module App
  module Named
    def id
      raise NotImplementedError
    end
  end

  class Repository
    def write(value)
      value.to_s
    end
  end

  class Service < Repository
    include Named

    attr_reader :id

    def initialize(repo)
      @repo = repo
      @id = "service"
    end

    def save(user)
      formatted = user.id
      helper(formatted)
      @repo.write(formatted)
      formatted
    end

    def helper(value)
      value
    end
  end
end
`

func TestPoolParsesRubyFixture(t *testing.T) {
	pool := parser.NewPool(nil, parser.PoolOptions{ParseTimeout: time.Second})
	defer pool.Close()

	result, err := pool.Parse(context.Background(), parser.Request{
		FilePath: "lib/service.rb",
		Language: scanner.Ruby,
		Source:   []byte(rubyParityFixture),
	})
	if err != nil {
		t.Fatalf("Parse Ruby failed: %v", err)
	}
	defer result.Close()
	if result.RootKind != "program" || result.HasError {
		t.Fatalf("unexpected Ruby parse result: %#v", result)
	}
}

func TestExtractRubyScopeIR(t *testing.T) {
	ir := extract(t, "lib/service.rb", "hash-ruby", []byte(rubyParityFixture))

	if ir.Language != scanner.Ruby {
		t.Fatalf("language = %q, want %q", ir.Language, scanner.Ruby)
	}
	for _, def := range ir.Definitions {
		if def.FileHash != "hash-ruby" {
			t.Fatalf("definition %s missing file hash: %#v", def.Name, def)
		}
	}

	app := requireDefinition(t, ir, "App", scopeir.NodeTrait)
	named := requireDefinition(t, ir, "Named", scopeir.NodeTrait)
	repository := requireDefinition(t, ir, "Repository", scopeir.NodeClass)
	service := requireDefinition(t, ir, "Service", scopeir.NodeClass)
	requireQualifiedDefinition(t, ir, "Named.id", scopeir.NodeMethod)
	requireQualifiedDefinition(t, ir, "Repository.write", scopeir.NodeMethod)
	initialize := requireQualifiedDefinition(t, ir, "Service.initialize", scopeir.NodeMethod)
	save := requireQualifiedDefinition(t, ir, "Service.save", scopeir.NodeMethod)
	helper := requireQualifiedDefinition(t, ir, "Service.helper", scopeir.NodeMethod)
	id := requireQualifiedDefinition(t, ir, "Service.id", scopeir.NodeProperty)
	repo := requireQualifiedDefinition(t, ir, "Service.repo", scopeir.NodeProperty)
	formatted := requireDefinition(t, ir, "formatted", scopeir.NodeVariable)
	if app.ID == "" || named.ID == "" || repository.ID == "" || service.ID == "" || id.ID == "" || repo.ID == "" || formatted.ID == "" {
		t.Fatal("expected all Ruby definitions to have stable IDs")
	}
	if initialize.OwnerID != service.ID || save.OwnerID != service.ID || helper.OwnerID != service.ID || id.OwnerID != service.ID || repo.OwnerID != service.ID {
		t.Fatalf("owner mismatch: initialize=%#v save=%#v helper=%#v id=%#v repo=%#v", initialize, save, helper, id, repo)
	}

	requireImport(t, ir, scopeir.ImportNamed, "json", "json", "json")
	requireCall(t, ir, "helper", scopeir.CallMember)
	requireCall(t, ir, "id", scopeir.CallMember)
	requireCall(t, ir, "write", scopeir.CallMember)
	requireAccess(t, ir, "repo", scopeir.AccessWrite)
	requireAccess(t, ir, "id", scopeir.AccessWrite)
	requireHeritage(t, ir, "Repository", scopeir.HeritageExtends)
	requireHeritage(t, ir, "Named", scopeir.HeritageInclude)
}

func TestExtractRubyRequireAttrReaderAndImplicitMemberCalls(t *testing.T) {
	const source = `require "json"

class Account
  attr_reader :name, :email

  def initialize
    @name = "root"
  end

  def greet
    helper()
    puts "hello"
  end

  def helper
    name
  end
end
`

	ir := extract(t, "lib/account.rb", "hash-ruby-edge", []byte(source))

	requireImport(t, ir, scopeir.ImportNamed, "json", "json", "json")
	account := requireDefinition(t, ir, "Account", scopeir.NodeClass)
	name := requireQualifiedDefinition(t, ir, "Account.name", scopeir.NodeProperty)
	email := requireQualifiedDefinition(t, ir, "Account.email", scopeir.NodeProperty)
	if name.OwnerID != account.ID || email.OwnerID != account.ID {
		t.Fatalf("attr_reader owner mismatch: name=%#v email=%#v account=%#v", name, email, account)
	}
	requireQualifiedDefinition(t, ir, "Account.greet", scopeir.NodeMethod)
	requireQualifiedDefinition(t, ir, "Account.helper", scopeir.NodeMethod)
	requireCall(t, ir, "helper", scopeir.CallMember)
	requireCall(t, ir, "puts", scopeir.CallMember)
	requireAccess(t, ir, "name", scopeir.AccessWrite)
}

func TestExtractRubyRequireRelativeAccessorAndYARDLegacyRouting(t *testing.T) {
	longPath := strings.Repeat("a", 1025)
	source := `require "json"
require_relative "models/user"
require_relative "../shared/utils"
require "` + longPath + `"

class Account
  # @return [Address]
  attr_accessor :address
  attr_writer :email, :status
  attr_reader :name

  def greet
    helper()
  end

  def helper
  end
end
`

	ir := extract(t, "lib/account.rb", "hash-ruby-routing", []byte(source))

	requireImport(t, ir, scopeir.ImportNamed, "json", "json", "json")
	requireImport(t, ir, scopeir.ImportNamed, "user", "user", "./models/user")
	requireImport(t, ir, scopeir.ImportNamed, "utils", "utils", "../shared/utils")
	requireNoImportTarget(t, ir, longPath)

	account := requireDefinition(t, ir, "Account", scopeir.NodeClass)
	address := requireQualifiedDefinition(t, ir, "Account.address", scopeir.NodeProperty)
	email := requireQualifiedDefinition(t, ir, "Account.email", scopeir.NodeProperty)
	status := requireQualifiedDefinition(t, ir, "Account.status", scopeir.NodeProperty)
	name := requireQualifiedDefinition(t, ir, "Account.name", scopeir.NodeProperty)
	if address.OwnerID != account.ID || email.OwnerID != account.ID || status.OwnerID != account.ID || name.OwnerID != account.ID {
		t.Fatalf("accessor owner mismatch: account=%#v address=%#v email=%#v status=%#v name=%#v", account, address, email, status, name)
	}
	if address.DeclaredType != "Address" {
		t.Fatalf("address declared type = %q, want Address", address.DeclaredType)
	}
	requireCall(t, ir, "helper", scopeir.CallMember)
	requireNoCall(t, ir, "attr_accessor")
	requireNoCall(t, ir, "attr_writer")
}

func TestExtractRubySingletonOwnerAndImplicitSelfCalls(t *testing.T) {
	const source = `class Account
  class << self
    def factory
      log()
    end
  end

  def work
    helper()
  end

  def helper
  end
end
`

	ir := extract(t, "lib/account.rb", "hash-ruby-singleton", []byte(source))

	account := requireDefinition(t, ir, "Account", scopeir.NodeClass)
	factory := requireQualifiedDefinition(t, ir, "Account.factory", scopeir.NodeMethod)
	work := requireQualifiedDefinition(t, ir, "Account.work", scopeir.NodeMethod)
	helper := requireQualifiedDefinition(t, ir, "Account.helper", scopeir.NodeMethod)
	if factory.OwnerID != account.ID || work.OwnerID != account.ID || helper.OwnerID != account.ID {
		t.Fatalf("singleton/instance owner mismatch: account=%#v factory=%#v work=%#v helper=%#v", account, factory, work, helper)
	}
	requireCall(t, ir, "log", scopeir.CallMember)
	requireCall(t, ir, "helper", scopeir.CallMember)
}

func TestExtractRubyLegacyMixinHeritageKinds(t *testing.T) {
	const source = `module Greetable
  def greet
  end
end

module LoggerMixin
end

module PrependedOverride
end

class Account
  include Greetable
  extend LoggerMixin
  prepend PrependedOverride
end
`

	ir := extract(t, "lib/account.rb", "hash-ruby-mixins", []byte(source))

	requireDefinition(t, ir, "Greetable", scopeir.NodeTrait)
	requireDefinition(t, ir, "LoggerMixin", scopeir.NodeTrait)
	requireDefinition(t, ir, "PrependedOverride", scopeir.NodeTrait)
	requireHeritage(t, ir, "Greetable", scopeir.HeritageInclude)
	requireHeritage(t, ir, "LoggerMixin", scopeir.HeritageExtend)
	requireHeritage(t, ir, "PrependedOverride", scopeir.HeritagePrepend)
}

func TestResolveRubyLegacyMixinGraphParity(t *testing.T) {
	const source = `module Greetable
  def greet
  end
end

module LoggerMixin
  def log_event
  end
end

module PrependedOverride
  def prepended_marker
  end
end

class Account
  include Greetable
  extend LoggerMixin
  prepend PrependedOverride

  def call_greet
    greet()
  end

  def call_prepended_marker
    prepended_marker()
  end
end
`

	ir := extract(t, "lib/account.rb", "hash-ruby-mixin-graph", []byte(source))
	result, err := resolution.Resolve([]scopeir.ScopeIR{ir}, resolution.Options{})
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	account := requireGraphNode(t, result.Graph, scopeir.NodeClass, "lib/account.rb", "Account")
	greetable := requireGraphNode(t, result.Graph, scopeir.NodeTrait, "lib/account.rb", "Greetable")
	loggerMixin := requireGraphNode(t, result.Graph, scopeir.NodeTrait, "lib/account.rb", "LoggerMixin")
	prependedOverride := requireGraphNode(t, result.Graph, scopeir.NodeTrait, "lib/account.rb", "PrependedOverride")
	callGreet := requireGraphNode(t, result.Graph, scopeir.NodeMethod, "lib/account.rb", "Account.call_greet")
	callPrepended := requireGraphNode(t, result.Graph, scopeir.NodeMethod, "lib/account.rb", "Account.call_prepended_marker")
	greet := requireGraphNode(t, result.Graph, scopeir.NodeMethod, "lib/account.rb", "Greetable.greet")
	prependedMarker := requireGraphNode(t, result.Graph, scopeir.NodeMethod, "lib/account.rb", "PrependedOverride.prepended_marker")

	requireGraphRelationship(t, result.Graph, graph.RelImplements, account.ID, greetable.ID, string(scopeir.HeritageInclude))
	requireGraphRelationship(t, result.Graph, graph.RelImplements, account.ID, loggerMixin.ID, string(scopeir.HeritageExtend))
	requireGraphRelationship(t, result.Graph, graph.RelImplements, account.ID, prependedOverride.ID, string(scopeir.HeritagePrepend))
	requireGraphRelationship(t, result.Graph, graph.RelCalls, callGreet.ID, greet.ID, "")
	requireGraphRelationship(t, result.Graph, graph.RelCalls, callPrepended.ID, prependedMarker.ID, "")
}

func requireNoImportTarget(t *testing.T, ir scopeir.ScopeIR, target string) {
	t.Helper()
	for _, item := range ir.Imports {
		if item.TargetRaw != nil && *item.TargetRaw == target {
			t.Fatalf("unexpected import target %q in %#v", target, ir.Imports)
		}
	}
}

func requireNoCall(t *testing.T, ir scopeir.ScopeIR, name string) {
	t.Helper()
	for _, call := range ir.Calls {
		if call.Name == name {
			t.Fatalf("unexpected call %s in %#v", name, ir.Calls)
		}
	}
}

func TestExtractRubyRejectsNonRubyLanguage(t *testing.T) {
	_, err := Extract(Request{
		FilePath: "src/service.ts",
		FileHash: "hash-ts",
		Language: scanner.TypeScript,
		Source:   []byte("class Service {}"),
	})
	if err == nil {
		t.Fatal("expected non-Ruby language to fail")
	}
}

func TestExtractRubyScopeIRParityFixture(t *testing.T) {
	ir := extract(t, "lib/service.rb", "hash-ruby", []byte(rubyParityFixture))
	signature := buildParitySignature(ir)
	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(signature); err != nil {
		t.Fatalf("marshal signature: %v", err)
	}
	raw := buffer.Bytes()
	golden, err := os.ReadFile("testdata/ruby_scopeir_signature.golden.json")
	if err != nil {
		t.Fatalf("read golden: %v\ngot:\n%s", err, raw)
	}
	if string(raw) != string(golden) {
		t.Fatalf("parity signature mismatch\nwant:\n%s\ngot:\n%s", golden, raw)
	}
}

func TestResolveRubyGraphParityCounts(t *testing.T) {
	ir := extract(t, "lib/service.rb", "hash-ruby", []byte(rubyParityFixture))
	result, err := resolution.Resolve([]scopeir.ScopeIR{ir}, resolution.Options{})
	if err != nil {
		t.Fatalf("resolve failed: %v", err)
	}
	counts := stringRelationshipCounts(result.Graph.RelationshipCountsByType())
	expected := map[string]int{
		"ACCESSES":     2,
		"CALLS":        1,
		"DEFINES":      12,
		"EXTENDS":      1,
		"IMPLEMENTS":   1,
		"HAS_METHOD":   5,
		"HAS_PROPERTY": 2,
		"INHERITS":     2,
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

func BenchmarkExtractRubyScopeIR(b *testing.B) {
	parsed := parseRubyFixture(b)
	defer parsed.Close()
	source := []byte(rubyParityFixture)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		ir, err := Extract(Request{
			FilePath: "lib/service.rb",
			FileHash: "hash-ruby",
			Language: scanner.Ruby,
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

func BenchmarkParseAndExtractRubyScopeIR(b *testing.B) {
	source := []byte(rubyParityFixture)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		pool := parser.NewPool(nil, parser.PoolOptions{ParseTimeout: time.Second})
		parsed, err := pool.Parse(context.Background(), parser.Request{
			FilePath: "lib/service.rb",
			Language: scanner.Ruby,
			Source:   source,
		})
		if err != nil {
			pool.Close()
			b.Fatalf("parse failed: %v", err)
		}
		ir, err := Extract(Request{
			FilePath: "lib/service.rb",
			FileHash: "hash-ruby",
			Language: scanner.Ruby,
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
	Scopes      []string `json:"scopes"`
	Definitions []string `json:"definitions"`
	Imports     []string `json:"imports"`
	Calls       []string `json:"calls"`
	Accesses    []string `json:"accesses"`
	Heritage    []string `json:"heritage"`
}

func buildParitySignature(ir scopeir.ScopeIR) paritySignature {
	signature := paritySignature{}
	for _, scope := range ir.Scopes {
		signature.Scopes = append(signature.Scopes, string(scope.Kind)+":"+scope.ID)
	}
	for _, def := range ir.Definitions {
		signature.Definitions = append(signature.Definitions, string(def.Label)+":"+def.QualifiedName+":"+def.OwnerID)
	}
	for _, item := range ir.Imports {
		target := ""
		if item.TargetRaw != nil {
			target = *item.TargetRaw
		}
		signature.Imports = append(signature.Imports, string(item.Kind)+":"+item.LocalName+":"+item.ImportedName+":"+target)
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
	sort.Strings(signature.Scopes)
	sort.Strings(signature.Definitions)
	sort.Strings(signature.Imports)
	sort.Strings(signature.Calls)
	sort.Strings(signature.Accesses)
	sort.Strings(signature.Heritage)
	return signature
}

func extract(t *testing.T, filePath string, fileHash string, source []byte) scopeir.ScopeIR {
	t.Helper()
	pool := parser.NewPool(nil, parser.PoolOptions{ParseTimeout: time.Second})
	defer pool.Close()
	parsed, err := pool.Parse(context.Background(), parser.Request{FilePath: filePath, Language: scanner.Ruby, Source: source})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	defer parsed.Close()
	ir, err := Extract(Request{
		FilePath: filePath,
		FileHash: fileHash,
		Language: scanner.Ruby,
		Source:   source,
		Root:     parsed.Tree.RootNode(),
	})
	if err != nil {
		t.Fatalf("extract failed: %v", err)
	}
	return ir
}

func parseRubyFixture(b *testing.B) *parser.Result {
	b.Helper()
	pool := parser.NewPool(nil, parser.PoolOptions{ParseTimeout: time.Second})
	parsed, err := pool.Parse(context.Background(), parser.Request{
		FilePath: "lib/service.rb",
		Language: scanner.Ruby,
		Source:   []byte(rubyParityFixture),
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

func requireGraphNode(t *testing.T, g *graph.Graph, label scopeir.NodeLabel, filePath string, qualifiedName string) graph.Node {
	t.Helper()
	for _, node := range g.Nodes {
		if node.Label == label && node.Properties["filePath"] == filePath && node.Properties["qualifiedName"] == qualifiedName {
			return node
		}
	}
	t.Fatalf("missing graph node %s %s %s", label, filePath, qualifiedName)
	return graph.Node{}
}

func requireGraphRelationship(t *testing.T, g *graph.Graph, relType graph.RelationshipType, sourceID string, targetID string, reason string) graph.Relationship {
	t.Helper()
	for _, relationship := range g.Relationships {
		if relationship.Type != relType || relationship.SourceID != sourceID || relationship.TargetID != targetID {
			continue
		}
		if reason == "" || relationship.Reason == reason {
			return relationship
		}
	}
	t.Fatalf("missing graph relationship %s %s -> %s reason %q", relType, sourceID, targetID, reason)
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
