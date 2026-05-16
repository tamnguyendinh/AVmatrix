package embeddings

import (
	"strings"
	"testing"

	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

func TestChunkNodeKeepsShortLabelsSingleChunk(t *testing.T) {
	chunks := ChunkNode(EmbeddableNode{
		Label:     scopeir.NodeVariable,
		Content:   "const value = 1",
		StartLine: 4,
		EndLine:   4,
	}, Config{ChunkSize: 5})
	if len(chunks) != 1 || chunks[0].Text != "const value = 1" || chunks[0].StartLine != 4 || chunks[0].EndLine != 4 {
		t.Fatalf("chunks = %#v", chunks)
	}
	if chunks[0].StartOffset != 0 || chunks[0].EndOffset != len("const value = 1") {
		t.Fatalf("chunk offsets = %#v", chunks[0])
	}
}

func TestCharacterChunksUsesOverlapAndLineRanges(t *testing.T) {
	chunks := CharacterChunks("aa\nbb\ncc\ndd", 10, 13, 5, 2)
	if len(chunks) != 3 {
		t.Fatalf("len(chunks) = %d, want 3: %#v", len(chunks), chunks)
	}
	if chunks[0].Text != "aa\nbb" || chunks[1].Text != "bb\ncc" || chunks[2].Text != "cc\ndd" {
		t.Fatalf("chunks = %#v", chunks)
	}
	if chunks[1].StartLine != 11 || chunks[1].EndLine != 12 {
		t.Fatalf("chunk line range = %#v", chunks[1])
	}
	for i, chunk := range chunks {
		if chunk.ChunkIndex != i {
			t.Fatalf("chunk %d index = %d", i, chunk.ChunkIndex)
		}
	}
}

func TestCharacterChunksKeepsStartLineAndNewlineBoundary(t *testing.T) {
	chunks := CharacterChunks("aaa\nbbb\nccc", 10, 12, 4, 0)
	if chunks[0].Text != "aaa\n" || chunks[0].StartLine != 10 || chunks[0].EndLine != 10 {
		t.Fatalf("first chunk = %#v", chunks[0])
	}
	if chunks[1].StartLine != 11 {
		t.Fatalf("second chunk start line = %d, want 11", chunks[1].StartLine)
	}
}

func TestChunkNodeSplitsClassByMembers(t *testing.T) {
	content := "class Parser {\n" +
		"  options: ParserOptions;\n" +
		"  cache: Map<string, any>;\n" +
		"  parseJSON() { return JSON.parse(\"{}\"); }\n" +
		"  validate() { return true; }\n" +
		"}"

	chunks := ChunkNode(EmbeddableNode{
		Label:     scopeir.NodeClass,
		Content:   content,
		StartLine: 1,
		EndLine:   6,
	}, Config{ChunkSize: 90, Overlap: 0})

	if len(chunks) != 2 {
		t.Fatalf("len(chunks) = %d, want 2: %#v", len(chunks), chunks)
	}
	if !containsAll(chunks[0].Text, "options: ParserOptions;", "cache: Map<string, any>;") {
		t.Fatalf("first chunk text = %q", chunks[0].Text)
	}
	if !containsAll(chunks[1].Text, "parseJSON()", "validate()") {
		t.Fatalf("second chunk text = %q", chunks[1].Text)
	}
	if chunks[0].StartLine != 2 || chunks[1].StartLine != 4 {
		t.Fatalf("chunk lines = %#v", chunks)
	}
}

func TestChunkNodePreservesInterfaceSignatures(t *testing.T) {
	content := "interface Handler {\n" +
		"  handle(event: Event): void;\n" +
		"  validate(input: string): boolean;\n" +
		"  readonly name: string;\n" +
		"}"

	chunks := ChunkNode(EmbeddableNode{
		Label:     scopeir.NodeInterface,
		Content:   content,
		StartLine: 10,
		EndLine:   14,
	}, Config{ChunkSize: 500, Overlap: 0})

	if len(chunks) != 1 {
		t.Fatalf("len(chunks) = %d, want 1: %#v", len(chunks), chunks)
	}
	if !containsAll(chunks[0].Text, "handle(event: Event): void;", "validate(input: string): boolean;", "readonly name: string;") {
		t.Fatalf("interface chunk text = %q", chunks[0].Text)
	}
}

func TestChunkNodeSplitsFunctionsAndConstructorsOnBodyLines(t *testing.T) {
	fn := "function example() {\n" +
		"  const first = 1;\n" +
		"\n" +
		"  const second = 2;\n" +
		"  return first + second;\n" +
		"}"
	fnChunks := ChunkNode(EmbeddableNode{
		Label:     scopeir.NodeFunction,
		Content:   fn,
		StartLine: 38,
		EndLine:   43,
	}, Config{ChunkSize: 68, Overlap: 0})
	if len(fnChunks) != 2 {
		t.Fatalf("len(fnChunks) = %d, want 2: %#v", len(fnChunks), fnChunks)
	}
	if fnChunks[0].StartOffset != 0 || !containsAll(fnChunks[0].Text, "function example() {", "const second = 2;") {
		t.Fatalf("first function chunk = %#v", fnChunks[0])
	}
	if fnChunks[1].StartLine < 42 || !containsAll(fnChunks[1].Text, "return first + second;", "}") {
		t.Fatalf("second function chunk = %#v", fnChunks[1])
	}

	constructor := "constructor() {\n" +
		"  this.ready = true;\n" +
		"  this.mode = \"prod\";\n" +
		"  this.start();\n" +
		"}"
	constructorChunks := ChunkNode(EmbeddableNode{
		Label:     scopeir.NodeConstructor,
		Content:   constructor,
		StartLine: 12,
		EndLine:   16,
	}, Config{ChunkSize: 55, Overlap: 0})
	if len(constructorChunks) != 2 {
		t.Fatalf("len(constructorChunks) = %d, want 2: %#v", len(constructorChunks), constructorChunks)
	}
	if !containsAll(constructorChunks[0].Text, "constructor() {", "this.ready = true;") || constructorChunks[0].StartLine != 12 {
		t.Fatalf("first constructor chunk = %#v", constructorChunks[0])
	}
	if constructorChunks[1].StartLine != 14 || !containsAll(constructorChunks[1].Text, "this.mode", "this.start") {
		t.Fatalf("second constructor chunk = %#v", constructorChunks[1])
	}
}

func TestChunkNodeFallsBackToCharacterChunksForUnsupportedShapes(t *testing.T) {
	content := strings.Repeat("x", 3000)
	chunks := ChunkNode(EmbeddableNode{
		Label:     scopeir.NodeFunction,
		Content:   content,
		StartLine: 1,
		EndLine:   100,
	}, Config{ChunkSize: 1200, Overlap: 120})
	if len(chunks) < 2 || chunks[0].StartOffset != 0 {
		t.Fatalf("chunks = %#v", chunks)
	}
}

func containsAll(text string, values ...string) bool {
	for _, value := range values {
		if !strings.Contains(text, value) {
			return false
		}
	}
	return true
}
