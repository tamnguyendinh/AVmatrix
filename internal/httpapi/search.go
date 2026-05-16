package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/tamnguyendinh/avmatrix-go/internal/embeddings"
	"github.com/tamnguyendinh/avmatrix-go/internal/lbugnative"
	"github.com/tamnguyendinh/avmatrix-go/internal/repo"
)

const (
	defaultAPISearchLimit = 10
	maxAPISearchLimit     = 100
	searchRRFK            = 60.0

	searchModeHybrid   = "hybrid"
	searchModeSemantic = "semantic"
	searchModeBM25     = "bm25"
)

var ErrSearchUnavailable = errors.New("search runtime unavailable")

type Searcher interface {
	Search(context.Context, SearchTarget, SearchRequest) ([]SearchResult, error)
}

type SearchTarget struct {
	RepoPath    string
	StoragePath string
	LbugPath    string
}

type SearchRequest struct {
	Query  string
	Mode   string
	Limit  int
	Enrich bool
}

type SearchResult struct {
	FilePath      string   `json:"filePath"`
	Score         float64  `json:"score"`
	Rank          int      `json:"rank,omitempty"`
	Sources       []string `json:"sources,omitempty"`
	NodeID        string   `json:"nodeId,omitempty"`
	Name          string   `json:"name,omitempty"`
	Label         string   `json:"label,omitempty"`
	StartLine     int      `json:"startLine,omitempty"`
	EndLine       int      `json:"endLine,omitempty"`
	BM25Score     float64  `json:"bm25Score,omitempty"`
	SemanticScore float64  `json:"semanticScore,omitempty"`
}

type SearchService struct {
	OpenReadRunner  func(string) (lbugnative.ReadRunner, error)
	ResolveEmbedder func() (embeddings.Embedder, embeddings.Config, error)
}

type bm25FTSIndex struct {
	tableName string
	indexName string
}

var bm25FTSIndexes = []bm25FTSIndex{
	{tableName: "File", indexName: "file_fts"},
	{tableName: "Function", indexName: "function_fts"},
	{tableName: "Class", indexName: "class_fts"},
	{tableName: "Method", indexName: "method_fts"},
	{tableName: "Interface", indexName: "interface_fts"},
}

type searchRequestBody struct {
	Query  string `json:"query"`
	Repo   string `json:"repo"`
	Mode   string `json:"mode"`
	Limit  int    `json:"limit"`
	Enrich *bool  `json:"enrich"`
}

type searchResponse struct {
	Results []SearchResult `json:"results"`
}

func (s Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	if !methodAllowed(w, r, http.MethodPost) {
		return
	}

	var body searchRequestBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil && !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "Invalid JSON request body")
		return
	}

	body.Query = strings.TrimSpace(body.Query)
	if body.Query == "" {
		writeError(w, http.StatusBadRequest, `Missing "query" in request body`)
		return
	}

	mode, ok := normalizeSearchMode(body.Mode)
	if !ok {
		writeError(w, http.StatusBadRequest, "Unsupported search mode")
		return
	}

	entries, err := s.store.ListRegistered(false)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	repoQuery := strings.TrimSpace(body.Repo)
	if repoQuery == "" {
		repoQuery = requestedRepo(r)
	}
	entry, status, message, err := resolveRepoQuery(entries, repoQuery)
	if err != nil {
		s.logger.Debug("resolve search repo failed", "error", err)
		writeError(w, status, message)
		return
	}

	enrich := true
	if body.Enrich != nil {
		enrich = *body.Enrich
	}
	results, err := s.searcher.Search(r.Context(), searchTargetFor(entry), SearchRequest{
		Query:  body.Query,
		Mode:   mode,
		Limit:  normalizeSearchLimit(body.Limit),
		Enrich: enrich,
	})
	if err != nil {
		if errors.Is(err, ErrSearchUnavailable) {
			writeError(w, http.StatusNotImplemented, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, searchResponse{Results: results})
}

func (s SearchService) Search(ctx context.Context, target SearchTarget, request SearchRequest) ([]SearchResult, error) {
	request.Query = strings.TrimSpace(request.Query)
	if request.Query == "" {
		return nil, nil
	}

	mode := request.Mode
	if mode == "" {
		mode = searchModeHybrid
	}
	if (mode == searchModeBM25 || mode == searchModeHybrid) && request.Limit <= 0 {
		return nil, nil
	}

	openReadRunner := s.OpenReadRunner
	if openReadRunner == nil {
		openReadRunner = lbugnative.OpenReadRunner
	}
	runner, err := openReadRunner(target.LbugPath)
	if err != nil {
		if errors.Is(err, lbugnative.ErrUnavailable) {
			return nil, fmt.Errorf("%w: native LadybugDB read runner is unavailable", ErrSearchUnavailable)
		}
		return nil, err
	}
	defer runner.Close()

	switch mode {
	case searchModeBM25:
		return searchBM25FromRunner(ctx, runner, request.Query, request.Limit)
	case searchModeSemantic:
		return s.searchSemantic(ctx, runner, request)
	case searchModeHybrid:
		bm25Results, err := searchBM25FromRunner(ctx, runner, request.Query, request.Limit)
		if err != nil {
			return nil, err
		}
		semanticResults, err := s.searchSemantic(ctx, runner, request)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return nil, err
			}
			return bm25Results, nil
		}
		return mergeSearchResultsWithRRF(bm25Results, semanticResults, request.Limit), nil
	default:
		return nil, fmt.Errorf("unsupported search mode %q", request.Mode)
	}
}

func (s SearchService) searchSemantic(ctx context.Context, runner embeddings.RowQueryRunner, request SearchRequest) ([]SearchResult, error) {
	resolveEmbedder := s.ResolveEmbedder
	if resolveEmbedder == nil {
		resolveEmbedder = defaultSearchEmbedder
	}
	embedder, config, err := resolveEmbedder()
	if err != nil {
		return nil, err
	}

	results, err := embeddings.SemanticSearch(ctx, runner, embedder, request.Query, embeddings.SearchOptions{
		Limit:      request.Limit,
		Dimensions: config.Dimensions,
	})
	if err != nil {
		return nil, err
	}
	return searchResultsFromSemantic(results), nil
}

func defaultSearchEmbedder() (embeddings.Embedder, embeddings.Config, error) {
	return resolveRuntimeEmbedder()
}

func resolveRuntimeEmbedder() (embeddings.Embedder, embeddings.Config, error) {
	config := embeddings.DefaultConfig()
	return embeddings.ResolveRuntimeEmbedder(context.Background(), config, nil)
}

func normalizeSearchMode(mode string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "", searchModeHybrid:
		return searchModeHybrid, true
	case searchModeSemantic:
		return searchModeSemantic, true
	case searchModeBM25:
		return searchModeBM25, true
	default:
		return "", false
	}
}

func normalizeSearchLimit(limit int) int {
	if limit <= 0 {
		return defaultAPISearchLimit
	}
	if limit > maxAPISearchLimit {
		return maxAPISearchLimit
	}
	return limit
}

func searchTargetFor(entry repo.RegistryEntry) SearchTarget {
	storagePath := storagePathFor(entry)
	return SearchTarget{
		RepoPath:    entry.Path,
		StoragePath: storagePath,
		LbugPath:    filepath.Join(storagePath, "lbug"),
	}
}

func searchResultsFromSemantic(results []embeddings.SearchResult) []SearchResult {
	items := make([]SearchResult, 0, len(results))
	for index, result := range results {
		score := 1 - result.Distance
		if math.IsNaN(score) || math.IsInf(score, 0) {
			score = 0
		}
		items = append(items, SearchResult{
			FilePath:  result.FilePath,
			Score:     score,
			Rank:      index + 1,
			Sources:   []string{searchModeSemantic},
			NodeID:    result.NodeID,
			Name:      result.Name,
			Label:     result.Label,
			StartLine: result.StartLine,
			EndLine:   result.EndLine,
		})
	}
	return items
}

func searchBM25FromRunner(ctx context.Context, runner embeddings.RowQueryRunner, query string, limit int) ([]SearchResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if runner == nil {
		return nil, fmt.Errorf("query runner is nil")
	}
	query = strings.TrimSpace(query)
	if query == "" || limit <= 0 {
		return nil, nil
	}

	merged := map[string]float64{}
	for _, index := range bm25FTSIndexes {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		rows, err := runner.QueryRows(bm25FTSQuery(index, query, limit))
		if err != nil {
			continue
		}
		for _, row := range rows {
			filePath := searchRowStringValue(row["filePath"])
			if filePath == "" {
				continue
			}
			merged[filePath] += searchRowFloatValue(row["score"])
		}
	}
	if len(merged) == 0 {
		return nil, nil
	}

	results := make([]SearchResult, 0, len(merged))
	for filePath, score := range merged {
		results = append(results, SearchResult{
			FilePath: filePath,
			Score:    score,
			Sources:  []string{searchModeBM25},
		})
	}
	sort.Slice(results, func(i, j int) bool {
		if results[i].Score != results[j].Score {
			return results[i].Score > results[j].Score
		}
		return results[i].FilePath < results[j].FilePath
	})
	if len(results) > limit {
		results = results[:limit]
	}
	for index := range results {
		results[index].Rank = index + 1
	}
	return results, nil
}

func bm25FTSQuery(index bm25FTSIndex, query string, limit int) string {
	return fmt.Sprintf(
		"CALL QUERY_FTS_INDEX(%s, %s, %s, conjunctive := false) RETURN node.filePath AS filePath, score ORDER BY score DESC LIMIT %d",
		searchCypherString(index.tableName),
		searchCypherString(index.indexName),
		searchCypherString(query),
		limit,
	)
}

func mergeSearchResultsWithRRF(bm25Results []SearchResult, semanticResults []SearchResult, limit int) []SearchResult {
	if limit <= 0 {
		limit = defaultAPISearchLimit
	}
	merged := map[string]*SearchResult{}

	for index, result := range bm25Results {
		if result.FilePath == "" {
			continue
		}
		item := SearchResult{
			FilePath:  result.FilePath,
			Score:     reciprocalRankScore(index),
			Sources:   []string{searchModeBM25},
			BM25Score: result.Score,
		}
		merged[result.FilePath] = &item
	}

	for index, result := range semanticResults {
		if result.FilePath == "" {
			continue
		}
		semanticScore := result.Score
		existing, ok := merged[result.FilePath]
		if ok {
			existing.Score += reciprocalRankScore(index)
			existing.Sources = appendSearchSource(existing.Sources, searchModeSemantic)
			existing.SemanticScore = semanticScore
			copySemanticMetadata(existing, result)
			continue
		}
		item := result
		item.Score = reciprocalRankScore(index)
		item.Rank = 0
		item.Sources = []string{searchModeSemantic}
		item.SemanticScore = semanticScore
		merged[result.FilePath] = &item
	}

	results := make([]SearchResult, 0, len(merged))
	for _, result := range merged {
		results = append(results, *result)
	}
	sort.Slice(results, func(i, j int) bool {
		if results[i].Score != results[j].Score {
			return results[i].Score > results[j].Score
		}
		return results[i].FilePath < results[j].FilePath
	})
	if len(results) > limit {
		results = results[:limit]
	}
	for index := range results {
		results[index].Rank = index + 1
	}
	return results
}

func reciprocalRankScore(index int) float64 {
	return 1 / (searchRRFK + float64(index) + 1)
}

func copySemanticMetadata(target *SearchResult, source SearchResult) {
	target.NodeID = source.NodeID
	target.Name = source.Name
	target.Label = source.Label
	target.StartLine = source.StartLine
	target.EndLine = source.EndLine
}

func appendSearchSource(sources []string, source string) []string {
	for _, existing := range sources {
		if existing == source {
			return sources
		}
	}
	return append(sources, source)
}

func searchCypherString(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `'`, `''`)
	value = strings.ReplaceAll(value, "\n", `\n`)
	value = strings.ReplaceAll(value, "\r", `\r`)
	return "'" + value + "'"
}

func searchRowStringValue(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	default:
		return fmt.Sprint(typed)
	}
}

func searchRowFloatValue(value any) float64 {
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
