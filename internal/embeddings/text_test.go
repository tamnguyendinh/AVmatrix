package embeddings

import (
	"crypto/sha1"
	"encoding/hex"
	"strings"
	"testing"

	"github.com/tamnguyendinh/avmatrix-go/internal/graph"
	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

func TestNodesFromGraphSelectsEmbeddableLabelsAndContext(t *testing.T) {
	exported := true
	g := graph.New()
	g.AddNode(graph.Node{
		ID:    "Function:main",
		Label: scopeir.NodeFunction,
		Properties: graph.NodeProperties{
			"name":        "main",
			"filePath":    "src/main.ts",
			"content":     "export function main() {}",
			"startLine":   10,
			"endLine":     12,
			"isExported":  exported,
			"description": "entrypoint",
		},
	})
	g.AddNode(graph.Node{ID: "File:src/main.ts", Label: scopeir.NodeFile, Properties: graph.NodeProperties{"name": "main.ts"}})
	g.AddNode(graph.Node{ID: "CodeElement:ignored", Label: scopeir.NodeCodeElement, Properties: graph.NodeProperties{"name": "ignored"}})

	nodes := NodesFromGraph(g, RuntimeContext{RepoName: "repo", ServerName: "server"})
	if len(nodes) != 1 {
		t.Fatalf("len(nodes) = %d, want 1", len(nodes))
	}
	node := nodes[0]
	if node.ID != "Function:main" || node.RepoName != "repo" || node.ServerName != "server" || node.StartLine != 10 || node.EndLine != 12 {
		t.Fatalf("node = %#v", node)
	}
	if node.IsExported == nil || *node.IsExported != exported {
		t.Fatalf("IsExported = %#v, want true", node.IsExported)
	}
}

func TestGenerateTextBuildsMetadataAndCleansContent(t *testing.T) {
	exported := true
	text := GenerateText(EmbeddableNode{
		Name:        "main",
		Label:       scopeir.NodeFunction,
		FilePath:    "src/main.ts",
		Content:     "export function main() {\r\n  return 1\r\n}\n\n\n",
		IsExported:  &exported,
		Description: strings.Repeat("description ", 40),
		RepoName:    "repo",
		ServerName:  "server",
	}, "export function main() {\r\n  return 1\r\n}\n\n\n", Config{MaxDescriptionLength: 40})

	for _, want := range []string{"Function: main", "Repo: repo", "Server: server", "Path: src/main.ts", "Export: true", "export function main() {\n  return 1\n}"} {
		if !strings.Contains(text, want) {
			t.Fatalf("GenerateText() missing %q:\n%s", want, text)
		}
	}
}

func TestGenerateTextIncludesStructuralMetadataAndDeclarationOnly(t *testing.T) {
	text := GenerateText(EmbeddableNode{
		Name:        "Parser",
		Label:       scopeir.NodeClass,
		FilePath:    "src/utils/parser.ts",
		Content:     "class Parser {\n  options: ParserOptions;\n  private cache: Map<string, any>;\n  parseJSON(text: string) { return JSON.parse(text); }\n  validate() { return true; }\n}",
		MethodNames: []string{"parseJSON", "validate"},
		FieldNames:  []string{"options", "cache"},
	}, "class Parser {\n  options: ParserOptions;\n  private cache: Map<string, any>;\n  parseJSON(text: string) { return JSON.parse(text); }\n  validate() { return true; }\n}", Config{})

	for _, want := range []string{"Class: Parser", "Methods: parseJSON, validate", "Properties: options, cache", "class Parser {", "options: ParserOptions;"} {
		if !strings.Contains(text, want) {
			t.Fatalf("GenerateText() missing %q:\n%s", want, text)
		}
	}
	for _, unwanted := range []string{"return JSON.parse", "return true"} {
		if strings.Contains(text, unwanted) {
			t.Fatalf("GenerateText() kept method body %q:\n%s", unwanted, text)
		}
	}
}

func TestGenerateTextIncludesStructuralChunkBody(t *testing.T) {
	node := EmbeddableNode{
		Name:        "Parser",
		Label:       scopeir.NodeClass,
		FilePath:    "src/utils/parser.ts",
		Content:     "class Parser {\n  options: ParserOptions;\n  cache: Map<string, any>;\n  parseJSON(text: string) { return JSON.parse(text); }\n}",
		MethodNames: []string{"parseJSON"},
		FieldNames:  []string{"options", "cache"},
	}
	chunkBody := "parseJSON(text: string) { return JSON.parse(text); }"
	text := GenerateText(node, chunkBody, Config{})

	for _, want := range []string{"Class: Parser", "Methods: parseJSON", "class Parser {", chunkBody} {
		if !strings.Contains(text, want) {
			t.Fatalf("GenerateText() missing %q:\n%s", want, text)
		}
	}
}

func TestGenerateTextIncludesMetadataForStructuralChunks(t *testing.T) {
	exported := true
	node := EmbeddableNode{
		Name:        "Parser",
		Label:       scopeir.NodeClass,
		FilePath:    "src/test.ts",
		Content:     "class Parser {\n  options: ParserOptions;\n  cache: Map<string, any>;\n  parseJSON(text: string) { return JSON.parse(text); }\n  validate() { return true; }\n}",
		StartLine:   20,
		EndLine:     25,
		IsExported:  &exported,
		RepoName:    "my-project",
		ServerName:  "my-service",
		MethodNames: []string{"parseJSON", "validate"},
		FieldNames:  []string{"options", "cache"},
	}
	chunks := ChunkNode(node, Config{ChunkSize: 90, Overlap: 0})
	if len(chunks) != 2 {
		t.Fatalf("len(chunks) = %d, want 2: %#v", len(chunks), chunks)
	}

	text := GenerateText(node, chunks[1].Text, Config{})
	for _, want := range []string{"Class: Parser", "Repo: my-project", "Server: my-service", "Export: true", "Methods: parseJSON, validate", "Properties: options, cache", "parseJSON(text: string)"} {
		if !strings.Contains(text, want) {
			t.Fatalf("GenerateText() missing %q:\n%s", want, text)
		}
	}
}

func TestGenerateTextCoversShortTypeAliasConstructorAndOptionalServer(t *testing.T) {
	alias := GenerateText(EmbeddableNode{
		Name:     "Result",
		Label:    scopeir.NodeTypeAlias,
		FilePath: "src/result.ts",
		Content:  "type Result<T> = Success<T> | Error;",
	}, "ignored chunk", Config{})
	if !strings.Contains(alias, "TypeAlias: Result") || !strings.Contains(alias, "type Result<T> = Success<T> | Error;") {
		t.Fatalf("type alias text = %q", alias)
	}

	if !IsChunkableLabel(scopeir.NodeConstructor) || !IsEmbeddableLabel(scopeir.NodeConstructor) {
		t.Fatal("constructor label should be chunkable and embeddable")
	}
	constructor := GenerateText(EmbeddableNode{
		Name:     "constructor",
		Label:    scopeir.NodeConstructor,
		FilePath: "src/user.ts",
		Content:  "constructor(private service: ApiClient) {\n  this.service = service;\n}",
	}, "constructor(private service: ApiClient) {\n  this.service = service;\n}", Config{})
	if !strings.Contains(constructor, "Constructor: constructor") || !strings.Contains(constructor, "this.service = service") {
		t.Fatalf("constructor text = %q", constructor)
	}

	withoutServer := GenerateText(EmbeddableNode{Name: "main", Label: scopeir.NodeFunction, FilePath: "src/main.ts", Content: "function main() {}"}, "function main() {}", Config{})
	if strings.Contains(withoutServer, "Server:") {
		t.Fatalf("server line emitted without server name:\n%s", withoutServer)
	}
}

func TestContentHashForNodeUsesGeneratedTextAndStableInputs(t *testing.T) {
	node := EmbeddableNode{
		ID:       "Function:foo:src/main.ts",
		Name:     "foo",
		Label:    scopeir.NodeFunction,
		FilePath: "src/main.ts",
		Content:  "function foo() { return 1; }",
	}

	sum := sha1.Sum([]byte(GenerateText(node, node.Content, Config{})))
	expected := hex.EncodeToString(sum[:])
	got := ContentHashForNode(node, Config{})
	if got != expected {
		t.Fatalf("ContentHashForNode() = %q, want sha1(GenerateText(...)) %q", got, expected)
	}
	if len(got) != 40 {
		t.Fatalf("hash length = %d, want 40", len(got))
	}
	if ContentHashForNode(node, Config{}) != got {
		t.Fatal("ContentHashForNode() is not deterministic")
	}
	if ContentHashForNode(node, Config{}) != ContentHashForNode(node, DefaultConfig()) {
		t.Fatal("empty config and explicit default config produced different hashes")
	}

	edited := node
	edited.Content = "function foo() { return 42; }"
	if ContentHashForNode(edited, Config{}) == got {
		t.Fatal("hash did not change when content changed")
	}

	moved := node
	moved.FilePath = "src/other.ts"
	if ContentHashForNode(moved, Config{}) == got {
		t.Fatal("hash did not change when file path changed")
	}
}

func TestContentHashIgnoresStructuralNameEnrichment(t *testing.T) {
	node := EmbeddableNode{
		ID:          "Class:User",
		Name:        "User",
		Label:       scopeir.NodeClass,
		FilePath:    "src/user.ts",
		Content:     "class User {\n  id: string\n  save() {}\n}",
		MethodNames: []string{"save"},
		FieldNames:  []string{"id"},
	}
	hashWithNames := ContentHashForNode(node, Config{})
	node.MethodNames = []string{"save", "load"}
	node.FieldNames = []string{"id", "email"}
	hashWithDifferentNames := ContentHashForNode(node, Config{})
	if hashWithNames != hashWithDifferentNames {
		t.Fatalf("hash changed with structural names: %s != %s", hashWithNames, hashWithDifferentNames)
	}
}

func TestExtractDeclarationOnlySkipsMethodBodies(t *testing.T) {
	got := extractDeclarationOnly(`class User {
  id: string
  save() {
    return db.save(this)
  }
  email: string
}`)
	if strings.Contains(got, "return db.save") {
		t.Fatalf("extractDeclarationOnly() kept method body:\n%s", got)
	}
	for _, want := range []string{"class User {", "id: string", "email: string", "}"} {
		if !strings.Contains(got, want) {
			t.Fatalf("extractDeclarationOnly() missing %q:\n%s", want, got)
		}
	}
}

func TestExtractDeclarationOnlyPreservesInterfacesStructFieldsAndRejectsNonBraceLanguages(t *testing.T) {
	interfaceDecl := extractDeclarationOnly("interface Handler {\n  handle(event: Event): void;\n  validate(input: string): boolean;\n  readonly name: string;\n}")
	for _, want := range []string{"interface Handler {", "handle(event: Event): void;", "readonly name: string;"} {
		if !strings.Contains(interfaceDecl, want) {
			t.Fatalf("interface declaration missing %q:\n%s", want, interfaceDecl)
		}
	}

	rustStruct := extractDeclarationOnly("struct User {\n    name: String,\n    age: u32,\n}")
	for _, want := range []string{"struct User {", "name: String,", "age: u32,"} {
		if !strings.Contains(rustStruct, want) {
			t.Fatalf("struct declaration missing %q:\n%s", want, rustStruct)
		}
	}

	python := extractDeclarationOnly("class User:\n    def __init__(self, name):\n        self.name = name")
	if python != "" {
		t.Fatalf("Python declaration = %q, want empty", python)
	}
}

func TestExtractDeclarationOnlyKeepsSingleLinePropertyInitializers(t *testing.T) {
	got := extractDeclarationOnly("class Foo {\n  config = { timeout: 5000 };\n  count = 0;\n}")
	for _, want := range []string{"config = { timeout: 5000 };", "count = 0;"} {
		if !strings.Contains(got, want) {
			t.Fatalf("extractDeclarationOnly() missing %q:\n%s", want, got)
		}
	}
}

func TestTruncateDescriptionUsesSentenceOrWordBoundaries(t *testing.T) {
	if got := truncateDescription("short text", 150); got != "short text" {
		t.Fatalf("short description = %q", got)
	}
	sentence := truncateDescription("First sentence. Second sentence. Third very long sentence.", 40)
	if !strings.Contains(sentence, "First sentence.") || len(sentence) >= len("First sentence. Second sentence. Third very long sentence.") {
		t.Fatalf("sentence truncation = %q", sentence)
	}
	word := truncateDescription("this is a long description without any sentence ending punctuation marks at all", 30)
	if len(word) > 30 || strings.HasSuffix(word, " ") {
		t.Fatalf("word truncation = %q", word)
	}
}
