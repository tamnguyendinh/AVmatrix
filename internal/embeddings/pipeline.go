package embeddings

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/tamnguyendinh/avmatrix-go/internal/graph"
	"github.com/tamnguyendinh/avmatrix-go/internal/lbugruntime"
	"github.com/tamnguyendinh/avmatrix-go/internal/lbugschema"
)

const embedSubBatchSize = 8

type Embedder interface {
	Embed(context.Context, []string) ([][]float32, error)
}

type QueryRunner interface {
	Query(string) error
}

type Phase string

const (
	PhaseEmbedding Phase = "embedding"
	PhaseIndexing  Phase = "indexing"
	PhaseReady     Phase = "ready"
	PhaseError     Phase = "error"
)

type Progress struct {
	Phase          Phase
	Percent        int
	NodesProcessed int
	TotalNodes     int
	CurrentBatch   int
	TotalBatches   int
	Error          string
}

type RunOptions struct {
	Config         Config
	RuntimeContext RuntimeContext
	ExistingHashes map[string]string
	OnProgress     func(Progress)
}

type RunResult struct {
	TotalNodes         int
	EmbeddedNodes      int
	SkippedFreshNodes  int
	StaleNodes         int
	Chunks             int
	DeleteQueries      int
	InsertQueries      int
	VectorIndexCreated bool
}

type EmbeddingUpdate struct {
	NodeID      string
	ChunkIndex  int
	StartLine   int
	EndLine     int
	Embedding   []float32
	ContentHash string
}

func Run(ctx context.Context, g *graph.Graph, runner QueryRunner, embedder Embedder, options RunOptions) (RunResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if g == nil {
		return RunResult{}, fmt.Errorf("graph is nil")
	}
	if runner == nil {
		return RunResult{}, fmt.Errorf("query runner is nil")
	}
	if embedder == nil {
		return RunResult{}, fmt.Errorf("embedder is nil")
	}
	config := NormalizeConfig(options.Config)
	nodes := NodesFromGraph(g, options.RuntimeContext)
	result := RunResult{TotalNodes: len(nodes)}
	if err := ctx.Err(); err != nil {
		return result, err
	}

	hashes := make(map[string]string, len(nodes))
	toEmbed := make([]EmbeddableNode, 0, len(nodes))
	staleNodeIDs := make([]string, 0)
	for _, node := range nodes {
		hash := ContentHashForNode(node, config)
		hashes[node.ID] = hash
		if existingHash, ok := options.ExistingHashes[node.ID]; ok {
			if existingHash == hash {
				result.SkippedFreshNodes++
				continue
			}
			result.StaleNodes++
			staleNodeIDs = append(staleNodeIDs, node.ID)
		}
		toEmbed = append(toEmbed, node)
	}

	for _, nodeID := range staleNodeIDs {
		if err := runner.Query(DeleteEmbeddingRowsQuery(nodeID)); err != nil {
			return fail(result, options.OnProgress, err)
		}
		result.DeleteQueries++
	}

	if len(toEmbed) == 0 {
		if err := createVectorIndex(runner); err != nil {
			return fail(result, options.OnProgress, err)
		}
		result.VectorIndexCreated = true
		emitProgress(options.OnProgress, Progress{Phase: PhaseReady, Percent: 100, NodesProcessed: 0, TotalNodes: 0})
		return result, nil
	}

	totalBatches := (len(toEmbed) + config.BatchSize - 1) / config.BatchSize
	emitProgress(options.OnProgress, Progress{Phase: PhaseEmbedding, Percent: 20, NodesProcessed: 0, TotalNodes: len(toEmbed), TotalBatches: totalBatches})
	processed := 0
	for start := 0; start < len(toEmbed); start += config.BatchSize {
		if err := ctx.Err(); err != nil {
			return fail(result, options.OnProgress, err)
		}
		end := min(start+config.BatchSize, len(toEmbed))
		batch := toEmbed[start:end]
		texts, updates := prepareBatch(batch, hashes, config)
		result.Chunks += len(updates)
		for subStart := 0; subStart < len(texts); subStart += embedSubBatchSize {
			subEnd := min(subStart+embedSubBatchSize, len(texts))
			vectors, err := embedder.Embed(ctx, texts[subStart:subEnd])
			if err != nil {
				return fail(result, options.OnProgress, err)
			}
			if len(vectors) != subEnd-subStart {
				return fail(result, options.OnProgress, fmt.Errorf("embedder returned %d vectors for %d texts", len(vectors), subEnd-subStart))
			}
			for index, vector := range vectors {
				if len(vector) != config.Dimensions {
					return fail(result, options.OnProgress, fmt.Errorf("embedding dimension mismatch: embedder returned %dd vector, but expected %dd", len(vector), config.Dimensions))
				}
				update := updates[subStart+index]
				update.Embedding = vector
				if err := runner.Query(CreateEmbeddingQuery(update)); err != nil {
					return fail(result, options.OnProgress, err)
				}
				result.InsertQueries++
			}
		}
		processed += len(batch)
		result.EmbeddedNodes += len(batch)
		percent := 20 + int(float64(processed)/float64(len(toEmbed))*70)
		emitProgress(options.OnProgress, Progress{
			Phase:          PhaseEmbedding,
			Percent:        percent,
			NodesProcessed: processed,
			TotalNodes:     len(toEmbed),
			CurrentBatch:   (start / config.BatchSize) + 1,
			TotalBatches:   totalBatches,
		})
	}

	emitProgress(options.OnProgress, Progress{Phase: PhaseIndexing, Percent: 90, NodesProcessed: len(toEmbed), TotalNodes: len(toEmbed)})
	if err := createVectorIndex(runner); err != nil {
		return fail(result, options.OnProgress, err)
	}
	result.VectorIndexCreated = true
	emitProgress(options.OnProgress, Progress{Phase: PhaseReady, Percent: 100, NodesProcessed: len(toEmbed), TotalNodes: len(toEmbed)})
	return result, nil
}

func prepareBatch(nodes []EmbeddableNode, hashes map[string]string, config Config) ([]string, []EmbeddingUpdate) {
	texts := make([]string, 0, len(nodes))
	updates := make([]EmbeddingUpdate, 0, len(nodes))
	for _, node := range nodes {
		for _, chunk := range ChunkNode(node, config) {
			texts = append(texts, GenerateText(node, chunk.Text, config))
			updates = append(updates, EmbeddingUpdate{
				NodeID:      node.ID,
				ChunkIndex:  chunk.ChunkIndex,
				StartLine:   chunk.StartLine,
				EndLine:     chunk.EndLine,
				ContentHash: hashes[node.ID],
			})
		}
	}
	return texts, updates
}

func createVectorIndex(runner QueryRunner) error {
	adapter := queryExecAdapter{runner: runner}
	var extensions lbugruntime.ExtensionState
	if err := extensions.EnsureVector(adapter); err != nil {
		return err
	}
	return runner.Query(lbugschema.CreateVectorIndexQuery())
}

func DeleteEmbeddingRowsQuery(nodeID string) string {
	return fmt.Sprintf("MATCH (e:%s {nodeId: %s}) DELETE e", lbugschema.EmbeddingTableName, cypherString(nodeID))
}

func CreateEmbeddingQuery(update EmbeddingUpdate) string {
	return fmt.Sprintf(
		"CREATE (e:%s {id: %s, nodeId: %s, chunkIndex: %d, startLine: %d, endLine: %d, embedding: %s, contentHash: %s})",
		lbugschema.EmbeddingTableName,
		cypherString(fmt.Sprintf("%s:%d", update.NodeID, update.ChunkIndex)),
		cypherString(update.NodeID),
		update.ChunkIndex,
		update.StartLine,
		update.EndLine,
		vectorLiteral(update.Embedding),
		cypherString(update.ContentHash),
	)
}

func vectorLiteral(vector []float32) string {
	values := make([]string, 0, len(vector))
	for _, value := range vector {
		values = append(values, strconv.FormatFloat(float64(value), 'g', -1, 32))
	}
	return "[" + strings.Join(values, ",") + "]"
}

func cypherString(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `'`, `''`)
	value = strings.ReplaceAll(value, "\n", `\n`)
	value = strings.ReplaceAll(value, "\r", `\r`)
	return "'" + value + "'"
}

func emitProgress(callback func(Progress), progress Progress) {
	if callback != nil {
		callback(progress)
	}
}

func fail(result RunResult, callback func(Progress), err error) (RunResult, error) {
	emitProgress(callback, Progress{Phase: PhaseError, Error: err.Error()})
	return result, err
}

type queryExecAdapter struct {
	runner QueryRunner
}

func (a queryExecAdapter) Exec(query string) error {
	return a.runner.Query(query)
}
