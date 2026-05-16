package embeddings

import (
	"context"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/tamnguyendinh/avmatrix-go/internal/lbugruntime"
)

func TestSemanticSearchQueriesVectorIndexDedupsChunksAndHydratesMetadata(t *testing.T) {
	runner := &searchRunner{
		vectorRows: []lbugruntime.Row{
			{"nodeId": "Function:alpha", "chunkIndex": 0, "startLine": 1, "endLine": 4, "distance": 0.31},
			{"nodeId": "Function:alpha", "chunkIndex": 1, "startLine": 5, "endLine": 8, "distance": 0.20},
			{"nodeId": "Method:beta", "chunkIndex": 0, "startLine": 10, "endLine": 12, "distance": 0.25},
			{"nodeId": "Class:gamma", "chunkIndex": 0, "startLine": 20, "endLine": 30, "distance": 0.40},
		},
		metadataRows: map[string][]lbugruntime.Row{
			"Function": {{"id": "Function:alpha", "name": "alpha", "filePath": "src/a.ts"}},
			"Method":   {{"id": "Method:beta", "name": "beta", "filePath": "src/b.ts"}},
			"Class":    {{"id": "Class:gamma", "name": "gamma", "filePath": "src/c.ts"}},
		},
	}
	results, err := SemanticSearch(context.Background(), runner, &recordingEmbedder{dimensions: 3}, "alpha query", SearchOptions{
		Limit:       2,
		Dimensions:  3,
		MaxDistance: 0.5,
	})
	if err != nil {
		t.Fatalf("SemanticSearch() error = %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("len(results) = %d, want 2: %#v", len(results), results)
	}
	if results[0].NodeID != "Function:alpha" || results[0].Distance != 0.20 || results[0].StartLine != 5 || results[1].NodeID != "Method:beta" {
		t.Fatalf("results = %#v", results)
	}
	if !containsQuery(runner.queries, "CALL QUERY_VECTOR_INDEX('CodeEmbedding', 'code_embedding_idx'") || !containsQuery(runner.queries, "CAST([1,2,3] AS FLOAT[3])") {
		t.Fatalf("vector query missing expected shape: %#v", runner.queries)
	}
	if !containsQuery(runner.queries, "MATCH (n:Function)") || !containsQuery(runner.queries, "MATCH (n:Method)") {
		t.Fatalf("metadata queries missing: %#v", runner.queries)
	}
}

func TestSemanticSearchRejectsQueryDimensionMismatch(t *testing.T) {
	runner := &searchRunner{}
	_, err := SemanticSearch(context.Background(), runner, &recordingEmbedder{dimensions: 2}, "query", SearchOptions{Dimensions: 3})
	if err == nil || !strings.Contains(err.Error(), "dimension mismatch") {
		t.Fatalf("SemanticSearch() error = %v, want dimension mismatch", err)
	}
}

func TestDedupBestChunksKeepsNearestChunkPerNode(t *testing.T) {
	matches := DedupBestChunks([]ChunkSearchRow{
		{NodeID: "Function:a", ChunkIndex: 0, Distance: 0.7},
		{NodeID: "Function:a", ChunkIndex: 1, Distance: 0.2},
		{NodeID: "Function:b", ChunkIndex: 0, Distance: 0.3},
	}, 1)
	if len(matches) != 1 || matches[0].NodeID != "Function:a" || matches[0].ChunkIndex != 1 {
		t.Fatalf("matches = %#v", matches)
	}
}

func TestCollectBestChunksExpandsFetchWindowUntilEnoughUniqueNodes(t *testing.T) {
	runner := &windowedSearchRunner{rows: append(append(
		searchRowsForNode("Function:a", 6, 10, 0.10),
		searchRowsForNode("Function:b", 6, 20, 0.20)...),
		ChunkSearchRow{NodeID: "Function:c", ChunkIndex: 0, StartLine: 30, EndLine: 31, Distance: 0.30},
		ChunkSearchRow{NodeID: "Function:d", ChunkIndex: 0, StartLine: 40, EndLine: 41, Distance: 0.40},
	)}

	matches, err := collectBestChunks(context.Background(), runner, []float32{1, 2, 3}, normalizedChunkSearchOptions(3))
	if err != nil {
		t.Fatalf("collectBestChunks() error = %v", err)
	}
	if !reflect.DeepEqual(runner.fetchLimits, []int{12, 24}) {
		t.Fatalf("fetchLimits = %#v, want [12 24]", runner.fetchLimits)
	}
	if got := bestChunkNodeIDs(matches); !reflect.DeepEqual(got, []string{"Function:a", "Function:b", "Function:c"}) {
		t.Fatalf("node IDs = %#v", got)
	}
}

func TestCollectBestChunksContinuesBeyondDefaultMaxFetchWhenUniqueNodesAreMissing(t *testing.T) {
	rows := make([]ChunkSearchRow, 0, 203)
	for i := 0; i < 200; i++ {
		nodeID := "Function:a"
		if i >= 120 {
			nodeID = "Function:b"
		}
		rows = append(rows, ChunkSearchRow{NodeID: nodeID, ChunkIndex: i, StartLine: i + 1, EndLine: i + 2, Distance: 0.01 + float64(i)*0.001})
	}
	rows = append(rows,
		ChunkSearchRow{NodeID: "Function:c", ChunkIndex: 0, StartLine: 500, EndLine: 501, Distance: 0.50},
		ChunkSearchRow{NodeID: "Function:d", ChunkIndex: 0, StartLine: 600, EndLine: 601, Distance: 0.60},
		ChunkSearchRow{NodeID: "Function:e", ChunkIndex: 0, StartLine: 700, EndLine: 701, Distance: 0.70},
	)
	runner := &windowedSearchRunner{rows: rows}

	matches, err := collectBestChunks(context.Background(), runner, []float32{1, 2, 3}, normalizedChunkSearchOptions(5))
	if err != nil {
		t.Fatalf("collectBestChunks() error = %v", err)
	}
	if !reflect.DeepEqual(runner.fetchLimits, []int{20, 40, 80, 160, 200, 400}) {
		t.Fatalf("fetchLimits = %#v", runner.fetchLimits)
	}
	if got := bestChunkNodeIDs(matches); !reflect.DeepEqual(got, []string{"Function:a", "Function:b", "Function:c", "Function:d", "Function:e"}) {
		t.Fatalf("node IDs = %#v", got)
	}
}

func TestCollectBestChunksStopsWhenVectorSearchIsExhausted(t *testing.T) {
	runner := &windowedSearchRunner{rows: []ChunkSearchRow{
		{NodeID: "Function:a", ChunkIndex: 0, StartLine: 1, EndLine: 2, Distance: 0.10},
		{NodeID: "Function:a", ChunkIndex: 1, StartLine: 3, EndLine: 4, Distance: 0.20},
		{NodeID: "Function:b", ChunkIndex: 0, StartLine: 5, EndLine: 6, Distance: 0.30},
	}}

	matches, err := collectBestChunks(context.Background(), runner, []float32{1, 2, 3}, normalizedChunkSearchOptions(5))
	if err != nil {
		t.Fatalf("collectBestChunks() error = %v", err)
	}
	if !reflect.DeepEqual(runner.fetchLimits, []int{20}) {
		t.Fatalf("fetchLimits = %#v, want [20]", runner.fetchLimits)
	}
	if got := bestChunkNodeIDs(matches); !reflect.DeepEqual(got, []string{"Function:a", "Function:b"}) {
		t.Fatalf("node IDs = %#v", got)
	}
}

func TestCollectBestChunksLargeLimitContinuesPastDefaultFetchWindow(t *testing.T) {
	rows := make([]ChunkSearchRow, 0, 260)
	for i := 0; i < 200; i++ {
		rows = append(rows, ChunkSearchRow{
			NodeID:     "Function:" + strconv.Itoa(i/50),
			ChunkIndex: i,
			StartLine:  i + 1,
			EndLine:    i + 2,
			Distance:   0.01 + float64(i)*0.001,
		})
	}
	for i := 0; i < 60; i++ {
		rows = append(rows, ChunkSearchRow{
			NodeID:     "Function:extra-" + strconv.Itoa(i),
			ChunkIndex: 0,
			StartLine:  300 + i,
			EndLine:    301 + i,
			Distance:   1 + float64(i)*0.001,
		})
	}
	runner := &windowedSearchRunner{rows: rows}

	matches, err := collectBestChunks(context.Background(), runner, []float32{1, 2, 3}, normalizedChunkSearchOptions(60))
	if err != nil {
		t.Fatalf("collectBestChunks() error = %v", err)
	}
	if !reflect.DeepEqual(runner.fetchLimits, []int{240, 480}) {
		t.Fatalf("fetchLimits = %#v, want [240 480]", runner.fetchLimits)
	}
	if len(matches) != 60 {
		t.Fatalf("len(matches) = %d, want 60", len(matches))
	}
}

type searchRunner struct {
	vectorRows   []lbugruntime.Row
	metadataRows map[string][]lbugruntime.Row
	queries      []string
}

func (r *searchRunner) QueryRows(query string) ([]lbugruntime.Row, error) {
	r.queries = append(r.queries, query)
	if strings.Contains(query, "QUERY_VECTOR_INDEX") {
		return r.vectorRows, nil
	}
	for label, rows := range r.metadataRows {
		if strings.Contains(query, "MATCH (n:"+label+")") {
			return rows, nil
		}
	}
	return nil, nil
}

type windowedSearchRunner struct {
	rows        []ChunkSearchRow
	fetchLimits []int
}

func (r *windowedSearchRunner) QueryRows(query string) ([]lbugruntime.Row, error) {
	fetchLimit := vectorFetchLimit(query)
	r.fetchLimits = append(r.fetchLimits, fetchLimit)
	rows := r.rows
	if fetchLimit < len(rows) {
		rows = rows[:fetchLimit]
	}
	result := make([]lbugruntime.Row, 0, len(rows))
	for _, row := range rows {
		result = append(result, lbugruntime.Row{
			"nodeId":     row.NodeID,
			"chunkIndex": row.ChunkIndex,
			"startLine":  row.StartLine,
			"endLine":    row.EndLine,
			"distance":   row.Distance,
		})
	}
	return result, nil
}

var vectorFetchLimitPattern = regexp.MustCompile(`,\s*(\d+)\)\s+YIELD`)

func vectorFetchLimit(query string) int {
	matches := vectorFetchLimitPattern.FindStringSubmatch(query)
	if len(matches) != 2 {
		return 0
	}
	value, _ := strconv.Atoi(matches[1])
	return value
}

func searchRowsForNode(nodeID string, count int, startLine int, distance float64) []ChunkSearchRow {
	rows := make([]ChunkSearchRow, 0, count)
	for i := 0; i < count; i++ {
		rows = append(rows, ChunkSearchRow{
			NodeID:     nodeID,
			ChunkIndex: i,
			StartLine:  startLine + i,
			EndLine:    startLine + i + 1,
			Distance:   distance + float64(i)*0.01,
		})
	}
	return rows
}

func bestChunkNodeIDs(matches []BestChunkMatch) []string {
	nodeIDs := make([]string, 0, len(matches))
	for _, match := range matches {
		nodeIDs = append(nodeIDs, match.NodeID)
	}
	return nodeIDs
}

func normalizedChunkSearchOptions(limit int) SearchOptions {
	return normalizeSearchOptions(SearchOptions{Limit: limit, Dimensions: 3})
}
