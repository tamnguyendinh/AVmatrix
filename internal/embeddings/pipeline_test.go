package embeddings

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/tamnguyendinh/avmatrix-go/internal/graph"
	"github.com/tamnguyendinh/avmatrix-go/internal/lbugschema"
	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

func TestRunEmbedsNewAndStaleNodesAndCreatesVectorIndex(t *testing.T) {
	g := graph.New()
	fresh := EmbeddableNode{
		ID:        "Function:fresh",
		Name:      "fresh",
		Label:     scopeir.NodeFunction,
		FilePath:  "src/fresh.ts",
		Content:   "export function fresh() { return 1 }",
		StartLine: 1,
		EndLine:   1,
	}
	g.AddNode(graph.Node{ID: fresh.ID, Label: fresh.Label, Properties: graph.NodeProperties{"name": fresh.Name, "filePath": fresh.FilePath, "content": fresh.Content, "startLine": fresh.StartLine, "endLine": fresh.EndLine}})
	g.AddNode(graph.Node{ID: "Variable:stale", Label: scopeir.NodeVariable, Properties: graph.NodeProperties{"name": "stale", "filePath": "src/stale.ts", "content": "export const stale = 1", "startLine": 3, "endLine": 3}})
	g.AddNode(graph.Node{ID: "Function:new", Label: scopeir.NodeFunction, Properties: graph.NodeProperties{"name": "new", "filePath": "src/new.ts", "content": "export function created() { return 2 }", "startLine": 5, "endLine": 5}})

	runner := &recordingRunner{}
	embedder := &recordingEmbedder{dimensions: 3}
	var progress []Progress
	result, err := Run(context.Background(), g, runner, embedder, RunOptions{
		Config: Config{Dimensions: 3, BatchSize: 2},
		ExistingHashes: map[string]string{
			fresh.ID:         ContentHashForNode(fresh, Config{Dimensions: 3}),
			"Variable:stale": "old-hash",
		},
		OnProgress: func(event Progress) {
			progress = append(progress, event)
		},
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.TotalNodes != 3 || result.EmbeddedNodes != 2 || result.SkippedFreshNodes != 1 || result.StaleNodes != 1 || result.DeleteQueries != 1 || result.InsertQueries != 2 || !result.VectorIndexCreated {
		t.Fatalf("result = %#v", result)
	}
	if len(embedder.batches) != 1 || len(embedder.batches[0]) != 2 {
		t.Fatalf("embedder batches = %#v", embedder.batches)
	}
	if !containsQuery(runner.queries, "MATCH (e:CodeEmbedding {nodeId: 'Variable:stale'}) DELETE e") {
		t.Fatalf("missing stale delete query: %#v", runner.queries)
	}
	if !containsQuery(runner.queries, "CREATE (e:CodeEmbedding") || !containsQuery(runner.queries, lbugschema.CreateVectorIndexQuery()) {
		t.Fatalf("missing insert/vector index queries: %#v", runner.queries)
	}
	if !containsQuery(runner.queries, "INSTALL VECTOR") || !containsQuery(runner.queries, "LOAD EXTENSION VECTOR") {
		t.Fatalf("missing vector extension lifecycle: %#v", runner.queries)
	}
	if progress[len(progress)-1].Phase != PhaseReady || progress[len(progress)-1].Percent != 100 {
		t.Fatalf("progress = %#v", progress)
	}
}

func TestRunCreatesVectorIndexWhenAllNodesAreFresh(t *testing.T) {
	g := graph.New()
	node := EmbeddableNode{
		ID:       "Function:fresh",
		Name:     "fresh",
		Label:    scopeir.NodeFunction,
		FilePath: "src/fresh.ts",
		Content:  "export function fresh() { return 1 }",
	}
	g.AddNode(graph.Node{ID: node.ID, Label: node.Label, Properties: graph.NodeProperties{"name": node.Name, "filePath": node.FilePath, "content": node.Content}})

	runner := &recordingRunner{}
	embedder := &recordingEmbedder{dimensions: 3}
	result, err := Run(context.Background(), g, runner, embedder, RunOptions{
		Config:         Config{Dimensions: 3},
		ExistingHashes: map[string]string{node.ID: ContentHashForNode(node, Config{Dimensions: 3})},
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.EmbeddedNodes != 0 || result.SkippedFreshNodes != 1 || result.InsertQueries != 0 || !result.VectorIndexCreated {
		t.Fatalf("result = %#v", result)
	}
	if len(embedder.batches) != 0 {
		t.Fatalf("embedder was called: %#v", embedder.batches)
	}
	if !containsQuery(runner.queries, lbugschema.CreateVectorIndexQuery()) {
		t.Fatalf("missing vector index query: %#v", runner.queries)
	}
}

func TestRunRejectsEmbeddingDimensionMismatch(t *testing.T) {
	g := graph.New()
	g.AddNode(graph.Node{ID: "Function:main", Label: scopeir.NodeFunction, Properties: graph.NodeProperties{"name": "main", "filePath": "src/main.ts", "content": "export function main() {}"}})
	runner := &recordingRunner{}
	embedder := &recordingEmbedder{dimensions: 2}
	var last Progress
	_, err := Run(context.Background(), g, runner, embedder, RunOptions{
		Config: Config{Dimensions: 3},
		OnProgress: func(event Progress) {
			last = event
		},
	})
	if err == nil || !strings.Contains(err.Error(), "dimension mismatch") {
		t.Fatalf("Run() error = %v, want dimension mismatch", err)
	}
	if last.Phase != PhaseError || !strings.Contains(last.Error, "dimension mismatch") {
		t.Fatalf("last progress = %#v", last)
	}
	if containsQuery(runner.queries, "CREATE (e:CodeEmbedding") {
		t.Fatalf("insert should not run on mismatch: %#v", runner.queries)
	}
}

func TestRunReturnsErrorWhenStaleDeleteFails(t *testing.T) {
	g := graph.New()
	node := EmbeddableNode{
		ID:       "Function:stale",
		Name:     "stale",
		Label:    scopeir.NodeFunction,
		FilePath: "src/stale.ts",
		Content:  "export function stale() { return 42 }",
	}
	g.AddNode(graph.Node{ID: node.ID, Label: node.Label, Properties: graph.NodeProperties{"name": node.Name, "filePath": node.FilePath, "content": node.Content}})

	runner := &failingRunner{failOn: "DELETE"}
	embedder := &recordingEmbedder{dimensions: 3}
	var last Progress
	_, err := Run(context.Background(), g, runner, embedder, RunOptions{
		Config:         Config{Dimensions: 3},
		ExistingHashes: map[string]string{node.ID: "stale-hash"},
		OnProgress: func(event Progress) {
			last = event
		},
	})
	if err == nil || !strings.Contains(err.Error(), "connection lost") {
		t.Fatalf("Run() error = %v, want stale delete error", err)
	}
	if !containsQuery(runner.queries, "DELETE") {
		t.Fatalf("missing stale delete query: %#v", runner.queries)
	}
	if len(embedder.batches) != 0 {
		t.Fatalf("embedder should not run after stale delete error: %#v", embedder.batches)
	}
	if last.Phase != PhaseError || !strings.Contains(last.Error, "connection lost") {
		t.Fatalf("last progress = %#v", last)
	}
}

func TestCreateEmbeddingQueryEscapesStringsAndFormatsVector(t *testing.T) {
	query := CreateEmbeddingQuery(EmbeddingUpdate{
		NodeID:      "Function:it's\\ok",
		ChunkIndex:  2,
		StartLine:   7,
		EndLine:     9,
		Embedding:   []float32{0.1, 2},
		ContentHash: "hash\nvalue",
	})
	for _, want := range []string{"Function:it''s\\\\ok:2", "embedding: [0.1,2]", "contentHash: 'hash\\nvalue'"} {
		if !strings.Contains(query, want) {
			t.Fatalf("query missing %q:\n%s", want, query)
		}
	}
}

type recordingRunner struct {
	queries []string
}

func (r *recordingRunner) Query(query string) error {
	r.queries = append(r.queries, query)
	return nil
}

type failingRunner struct {
	queries []string
	failOn  string
}

func (r *failingRunner) Query(query string) error {
	r.queries = append(r.queries, query)
	if strings.Contains(query, r.failOn) {
		return errors.New("connection lost")
	}
	return nil
}

type recordingEmbedder struct {
	dimensions int
	batches    [][]string
}

func (e *recordingEmbedder) Embed(_ context.Context, texts []string) ([][]float32, error) {
	e.batches = append(e.batches, append([]string(nil), texts...))
	vectors := make([][]float32, 0, len(texts))
	for range texts {
		vector := make([]float32, e.dimensions)
		for i := range vector {
			vector[i] = float32(i + 1)
		}
		vectors = append(vectors, vector)
	}
	return vectors, nil
}

func containsQuery(queries []string, needle string) bool {
	for _, query := range queries {
		if strings.Contains(query, needle) {
			return true
		}
	}
	return false
}
