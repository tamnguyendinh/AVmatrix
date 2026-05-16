package embeddings

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/tamnguyendinh/avmatrix-go/internal/lbugruntime"
	"github.com/tamnguyendinh/avmatrix-go/internal/lbugschema"
)

const (
	defaultSearchLimit      = 10
	defaultMaxDistance      = 0.5
	defaultFetchMultiplier  = 4
	defaultFetchBuffer      = 8
	defaultMaxFetch         = 200
	defaultSearchDimensions = DefaultDimensions
)

type RowQueryRunner interface {
	QueryRows(string) ([]lbugruntime.Row, error)
}

type SearchOptions struct {
	Limit           int
	MaxDistance     float64
	FetchMultiplier int
	FetchBuffer     int
	MaxFetch        int
	Dimensions      int
}

type SearchResult struct {
	NodeID    string
	Name      string
	Label     string
	FilePath  string
	Distance  float64
	StartLine int
	EndLine   int
}

type ChunkSearchRow struct {
	NodeID     string
	ChunkIndex int
	StartLine  int
	EndLine    int
	Distance   float64
}

type BestChunkMatch struct {
	NodeID     string
	ChunkIndex int
	StartLine  int
	EndLine    int
	Distance   float64
}

func SemanticSearch(ctx context.Context, runner RowQueryRunner, embedder Embedder, query string, options SearchOptions) ([]SearchResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if runner == nil {
		return nil, fmt.Errorf("query runner is nil")
	}
	if embedder == nil {
		return nil, fmt.Errorf("embedder is nil")
	}
	options = normalizeSearchOptions(options)
	if options.Limit <= 0 {
		return nil, nil
	}

	vectors, err := embedder.Embed(ctx, []string{query})
	if err != nil {
		return nil, err
	}
	if len(vectors) != 1 {
		return nil, fmt.Errorf("embedder returned %d query vectors, want 1", len(vectors))
	}
	queryVector := vectors[0]
	if len(queryVector) != options.Dimensions {
		return nil, fmt.Errorf("query embedding dimension mismatch: embedder returned %dd vector, but expected %dd", len(queryVector), options.Dimensions)
	}

	matches, err := collectBestChunks(ctx, runner, queryVector, options)
	if err != nil {
		return nil, err
	}
	return hydrateSearchResults(runner, matches, options.Limit), nil
}

func DedupBestChunks(rows []ChunkSearchRow, limit int) []BestChunkMatch {
	if limit <= 0 {
		return nil
	}
	best := map[string]BestChunkMatch{}
	for _, row := range rows {
		if row.NodeID == "" {
			continue
		}
		existing, ok := best[row.NodeID]
		if !ok || row.Distance < existing.Distance {
			best[row.NodeID] = BestChunkMatch{
				NodeID:     row.NodeID,
				ChunkIndex: row.ChunkIndex,
				StartLine:  row.StartLine,
				EndLine:    row.EndLine,
				Distance:   row.Distance,
			}
		}
	}
	matches := make([]BestChunkMatch, 0, len(best))
	for _, match := range best {
		matches = append(matches, match)
	}
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].Distance != matches[j].Distance {
			return matches[i].Distance < matches[j].Distance
		}
		return matches[i].NodeID < matches[j].NodeID
	})
	if len(matches) > limit {
		matches = matches[:limit]
	}
	return matches
}

func collectBestChunks(ctx context.Context, runner RowQueryRunner, queryVector []float32, options SearchOptions) ([]BestChunkMatch, error) {
	fetchLimit := max(options.Limit*options.FetchMultiplier, options.Limit+options.FetchBuffer)
	previousFetchLimit := 0
	for fetchLimit > previousFetchLimit {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		rows, err := runner.QueryRows(vectorSearchQuery(queryVector, options, fetchLimit))
		if err != nil {
			return nil, err
		}
		chunks := chunkRows(rows)
		best := DedupBestChunks(chunks, options.Limit)
		if len(best) >= options.Limit || len(rows) < fetchLimit {
			return best, nil
		}
		previousFetchLimit = fetchLimit
		if fetchLimit >= options.MaxFetch {
			fetchLimit *= 2
		} else {
			fetchLimit = min(options.MaxFetch, fetchLimit*2)
		}
	}
	return nil, nil
}

func hydrateSearchResults(runner RowQueryRunner, matches []BestChunkMatch, limit int) []SearchResult {
	if len(matches) == 0 {
		return nil
	}
	byLabel := map[string][]BestChunkMatch{}
	for _, match := range matches {
		label := labelFromNodeID(match.NodeID)
		if label == "" {
			continue
		}
		byLabel[label] = append(byLabel[label], match)
	}

	results := make([]SearchResult, 0, len(matches))
	for label, labelMatches := range byLabel {
		rows, err := runner.QueryRows(metadataQuery(label, labelMatches))
		if err != nil {
			continue
		}
		metadata := map[string]lbugruntime.Row{}
		for _, row := range rows {
			id := rowStringValue(row["id"])
			if id != "" {
				metadata[id] = row
			}
		}
		for _, match := range labelMatches {
			row, ok := metadata[match.NodeID]
			if !ok {
				continue
			}
			results = append(results, SearchResult{
				NodeID:    match.NodeID,
				Name:      rowStringValue(row["name"]),
				Label:     label,
				FilePath:  rowStringValue(row["filePath"]),
				Distance:  match.Distance,
				StartLine: match.StartLine,
				EndLine:   match.EndLine,
			})
		}
	}
	sort.Slice(results, func(i, j int) bool {
		if results[i].Distance != results[j].Distance {
			return results[i].Distance < results[j].Distance
		}
		return results[i].NodeID < results[j].NodeID
	})
	if len(results) > limit {
		results = results[:limit]
	}
	return results
}

func vectorSearchQuery(queryVector []float32, options SearchOptions, fetchLimit int) string {
	return fmt.Sprintf(
		"CALL QUERY_VECTOR_INDEX('%s', '%s', CAST(%s AS FLOAT[%d]), %d) YIELD node AS emb, distance WITH emb, distance WHERE distance < %s RETURN emb.nodeId AS nodeId, emb.chunkIndex AS chunkIndex, emb.startLine AS startLine, emb.endLine AS endLine, distance ORDER BY distance",
		lbugschema.EmbeddingTableName,
		lbugschema.EmbeddingIndexName,
		vectorLiteral(queryVector),
		options.Dimensions,
		fetchLimit,
		strconv.FormatFloat(options.MaxDistance, 'f', -1, 64),
	)
}

func metadataQuery(label string, matches []BestChunkMatch) string {
	ids := make([]string, 0, len(matches))
	for _, match := range matches {
		ids = append(ids, cypherString(match.NodeID))
	}
	return fmt.Sprintf(
		"MATCH (n:%s) WHERE n.id IN [%s] RETURN n.id AS id, n.name AS name, n.filePath AS filePath, n.startLine AS startLine, n.endLine AS endLine",
		lbugschema.FormatIdent(label),
		strings.Join(ids, ", "),
	)
}

func chunkRows(rows []lbugruntime.Row) []ChunkSearchRow {
	chunks := make([]ChunkSearchRow, 0, len(rows))
	for _, row := range rows {
		chunks = append(chunks, ChunkSearchRow{
			NodeID:     rowStringValue(row["nodeId"]),
			ChunkIndex: intValue(row["chunkIndex"]),
			StartLine:  intValue(row["startLine"]),
			EndLine:    intValue(row["endLine"]),
			Distance:   floatValue(row["distance"]),
		})
	}
	return chunks
}

func normalizeSearchOptions(options SearchOptions) SearchOptions {
	if options.Limit == 0 {
		options.Limit = defaultSearchLimit
	}
	if options.MaxDistance == 0 {
		options.MaxDistance = defaultMaxDistance
	}
	if options.FetchMultiplier <= 0 {
		options.FetchMultiplier = defaultFetchMultiplier
	}
	if options.FetchBuffer <= 0 {
		options.FetchBuffer = defaultFetchBuffer
	}
	if options.MaxFetch <= 0 {
		options.MaxFetch = defaultMaxFetch
	}
	if options.Dimensions <= 0 {
		options.Dimensions = defaultSearchDimensions
	}
	return options
}

func labelFromNodeID(nodeID string) string {
	index := strings.Index(nodeID, ":")
	if index <= 0 {
		return ""
	}
	return nodeID[:index]
}

func intValue(value any) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int32:
		return int(typed)
	case int64:
		return int(typed)
	case float32:
		return int(typed)
	case float64:
		return int(typed)
	case string:
		parsed, err := strconv.Atoi(typed)
		if err == nil {
			return parsed
		}
	}
	return 0
}

func floatValue(value any) float64 {
	switch typed := value.(type) {
	case float64:
		return typed
	case float32:
		return float64(typed)
	case int:
		return float64(typed)
	case int32:
		return float64(typed)
	case int64:
		return float64(typed)
	case string:
		parsed, err := strconv.ParseFloat(typed, 64)
		if err == nil {
			return parsed
		}
	}
	return 0
}

func rowStringValue(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	default:
		return fmt.Sprint(typed)
	}
}
