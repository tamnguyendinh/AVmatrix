package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/tamnguyendinh/avmatrix-go/internal/embeddings"
	"github.com/tamnguyendinh/avmatrix-go/internal/lbugnative"
	"github.com/tamnguyendinh/avmatrix-go/internal/lbugruntime"
)

func TestSearchEndpointCallsSearchServiceAndReturnsResults(t *testing.T) {
	searcher := &recordingSearcher{
		results: []SearchResult{{
			FilePath:  "src/app.ts",
			Score:     0.91,
			Rank:      1,
			Sources:   []string{"semantic"},
			NodeID:    "Function:alpha",
			Name:      "alpha",
			Label:     "Function",
			StartLine: 3,
			EndLine:   9,
		}},
	}
	server, fixtures := newRepoServerWithConfig(t, []repoFixture{{name: "alpha"}}, func(config *Config) {
		config.Searcher = searcher
	})
	defer server.Close()

	var payload searchResponse
	postJSON(t, server.URL+"/api/search", `{"query":" alpha ","repo":"`+jsonEscape(fixtures[0].path)+`","mode":"semantic","limit":200,"enrich":false}`, http.StatusOK, &payload)

	if searcher.request.Query != "alpha" || searcher.request.Mode != searchModeSemantic {
		t.Fatalf("search request = %#v", searcher.request)
	}
	if searcher.request.Limit != maxAPISearchLimit || searcher.request.Enrich {
		t.Fatalf("search request limit/enrich = %#v", searcher.request)
	}
	if searcher.target.RepoPath != fixtures[0].path {
		t.Fatalf("target repo = %q, want %q", searcher.target.RepoPath, fixtures[0].path)
	}
	wantLbugPath := filepath.Join(fixtures[0].path, ".avmatrix", "lbug")
	if searcher.target.LbugPath != wantLbugPath {
		t.Fatalf("target lbug = %q, want %q", searcher.target.LbugPath, wantLbugPath)
	}
	if len(payload.Results) != 1 || payload.Results[0].NodeID != "Function:alpha" || payload.Results[0].Score != 0.91 {
		t.Fatalf("unexpected search payload: %#v", payload)
	}
}

func TestSearchEndpointRejectsMissingQuery(t *testing.T) {
	server, _ := newRepoServer(t, []repoFixture{{name: "alpha"}})
	defer server.Close()

	var payload map[string]string
	postJSON(t, server.URL+"/api/search", `{"query":"   "}`, http.StatusBadRequest, &payload)

	if payload["error"] != `Missing "query" in request body` {
		t.Fatalf("error = %#v", payload)
	}
}

func TestSearchEndpointReturnsUnavailableStatus(t *testing.T) {
	server, fixtures := newRepoServerWithConfig(t, []repoFixture{{name: "alpha"}}, func(config *Config) {
		config.Searcher = &recordingSearcher{err: fmt.Errorf("%w: no read runner", ErrSearchUnavailable)}
	})
	defer server.Close()

	var payload map[string]string
	postJSON(t, server.URL+"/api/search?repo="+url.QueryEscape(fixtures[0].path), `{"query":"alpha"}`, http.StatusNotImplemented, &payload)

	if !strings.Contains(payload["error"], "search runtime unavailable") {
		t.Fatalf("error = %#v", payload)
	}
}

func TestSearchServiceUsesSemanticRuntime(t *testing.T) {
	runner := &semanticSearchRunner{}
	service := SearchService{
		OpenReadRunner: func(path string) (lbugnative.ReadRunner, error) {
			if path != filepath.Join("repo", ".avmatrix", "lbug") {
				t.Fatalf("lbug path = %q", path)
			}
			return runner, nil
		},
		ResolveEmbedder: func() (embeddings.Embedder, embeddings.Config, error) {
			return staticSearchEmbedder{vector: []float32{0.1, 0.2}}, embeddings.Config{Dimensions: 2}, nil
		},
	}

	results, err := service.Search(context.Background(), SearchTarget{LbugPath: filepath.Join("repo", ".avmatrix", "lbug")}, SearchRequest{
		Query: "find alpha",
		Mode:  searchModeSemantic,
		Limit: 1,
	})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if !runner.closed {
		t.Fatalf("runner was not closed")
	}
	if len(results) != 1 {
		t.Fatalf("result count = %d", len(results))
	}
	result := results[0]
	if result.NodeID != "Function:alpha" || result.Score != 0.88 || result.Rank != 1 {
		t.Fatalf("unexpected semantic result: %#v", result)
	}
	if len(result.Sources) != 1 || result.Sources[0] != searchModeSemantic {
		t.Fatalf("sources = %#v", result.Sources)
	}
}

func TestSearchServiceUsesBM25Runtime(t *testing.T) {
	runner := &semanticSearchRunner{
		ftsRows: map[string][]lbugruntime.Row{
			"File": {
				{"filePath": "src/auth.ts", "score": "3"},
			},
			"Function": {
				{"filePath": "src/auth.ts", "score": 2.5},
				{"filePath": "src/router.ts", "score": 1},
			},
			"Class": {
				{"filePath": "src/auth.ts", "score": 4},
			},
			"Method": {
				{"filePath": "src/auth.ts", "score": 1.25},
			},
			"Interface": {
				{"filePath": "src/auth.ts", "score": 0.75},
			},
		},
	}
	service := SearchService{
		OpenReadRunner: func(path string) (lbugnative.ReadRunner, error) {
			if path != filepath.Join("repo", ".avmatrix", "lbug") {
				t.Fatalf("lbug path = %q", path)
			}
			return runner, nil
		},
	}

	results, err := service.Search(context.Background(), SearchTarget{LbugPath: filepath.Join("repo", ".avmatrix", "lbug")}, SearchRequest{
		Query: "user authentication",
		Mode:  searchModeBM25,
		Limit: 2,
	})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if !runner.closed {
		t.Fatalf("runner was not closed")
	}
	if len(results) != 2 {
		t.Fatalf("len(results) = %d, want 2: %#v", len(results), results)
	}
	if results[0].FilePath != "src/auth.ts" || results[0].Score != 11.5 || results[0].Rank != 1 {
		t.Fatalf("top result = %#v", results[0])
	}
	if results[1].FilePath != "src/router.ts" || results[1].Score != 1 || results[1].Rank != 2 {
		t.Fatalf("second result = %#v", results[1])
	}
	if !reflect.DeepEqual(results[0].Sources, []string{searchModeBM25}) {
		t.Fatalf("sources = %#v", results[0].Sources)
	}
	if got := bm25QueryTables(runner.queries); !reflect.DeepEqual(got, []string{"File", "Function", "Class", "Method", "Interface"}) {
		t.Fatalf("BM25 query tables = %#v", got)
	}
	for _, query := range runner.queries {
		if strings.Contains(query, "QUERY_FTS_INDEX") && !strings.Contains(query, "LIMIT 2") {
			t.Fatalf("BM25 query missing limit: %s", query)
		}
	}
}

func TestSearchServiceBM25HandlesEmptyInputAndIndexErrors(t *testing.T) {
	openCount := 0
	service := SearchService{
		OpenReadRunner: func(string) (lbugnative.ReadRunner, error) {
			openCount++
			return &semanticSearchRunner{}, nil
		},
	}

	results, err := service.Search(context.Background(), SearchTarget{LbugPath: "repo/.avmatrix/lbug"}, SearchRequest{
		Query: "   ",
		Mode:  searchModeBM25,
		Limit: 10,
	})
	if err != nil {
		t.Fatalf("Search(empty query) error = %v", err)
	}
	if len(results) != 0 || openCount != 0 {
		t.Fatalf("empty query results/openCount = %#v/%d, want empty/0", results, openCount)
	}

	results, err = service.Search(context.Background(), SearchTarget{LbugPath: "repo/.avmatrix/lbug"}, SearchRequest{
		Query: "user authentication",
		Mode:  searchModeBM25,
		Limit: 0,
	})
	if err != nil {
		t.Fatalf("Search(limit 0) error = %v", err)
	}
	if len(results) != 0 || openCount != 0 {
		t.Fatalf("limit 0 results/openCount = %#v/%d, want empty/0", results, openCount)
	}

	runner := &semanticSearchRunner{
		ftsRows: map[string][]lbugruntime.Row{
			"File": {{"filePath": "src/auth.ts", "score": 5}},
		},
		ftsErrors: map[string]error{
			"Function":  errors.New("index does not exist"),
			"Class":     errors.New("index does not exist"),
			"Method":    errors.New("index does not exist"),
			"Interface": errors.New("index does not exist"),
		},
	}
	service.OpenReadRunner = func(string) (lbugnative.ReadRunner, error) {
		return runner, nil
	}
	results, err = service.Search(context.Background(), SearchTarget{LbugPath: "repo/.avmatrix/lbug"}, SearchRequest{
		Query: "user* OR auth+' \\",
		Mode:  searchModeBM25,
		Limit: 3,
	})
	if err != nil {
		t.Fatalf("Search(index errors) error = %v", err)
	}
	if len(results) != 1 || results[0].FilePath != "src/auth.ts" {
		t.Fatalf("results = %#v", results)
	}
	if len(runner.queries) == 0 || !strings.Contains(runner.queries[0], "auth+''") || !strings.Contains(runner.queries[0], `\\`) {
		t.Fatalf("escaped FTS query = %q", runner.queries[0])
	}
}

func TestMergeSearchResultsWithRRFMatchesLegacyHybridBehavior(t *testing.T) {
	if got := mergeSearchResultsWithRRF(nil, nil, 10); len(got) != 0 {
		t.Fatalf("empty merge = %#v, want empty", got)
	}

	bm25Only := mergeSearchResultsWithRRF([]SearchResult{
		makeBM25SearchResult("src/a.ts", 10),
		makeBM25SearchResult("src/b.ts", 5),
	}, nil, 10)
	if len(bm25Only) != 2 || bm25Only[0].FilePath != "src/a.ts" || bm25Only[0].Rank != 1 || bm25Only[1].Rank != 2 {
		t.Fatalf("BM25-only merge = %#v", bm25Only)
	}
	if !reflect.DeepEqual(bm25Only[0].Sources, []string{searchModeBM25}) || bm25Only[0].BM25Score != 10 {
		t.Fatalf("BM25-only metadata = %#v", bm25Only[0])
	}

	semanticOnly := mergeSearchResultsWithRRF(nil, []SearchResult{
		makeSemanticSearchResult("src/a.ts", 0.9),
		makeSemanticSearchResult("src/b.ts", 0.8),
	}, 10)
	if len(semanticOnly) != 2 || semanticOnly[0].FilePath != "src/a.ts" || semanticOnly[0].SemanticScore != 0.9 {
		t.Fatalf("semantic-only merge = %#v", semanticOnly)
	}
	if !reflect.DeepEqual(semanticOnly[0].Sources, []string{searchModeSemantic}) {
		t.Fatalf("semantic sources = %#v", semanticOnly[0].Sources)
	}

	combined := mergeSearchResultsWithRRF(
		[]SearchResult{
			makeBM25SearchResult("src/shared.ts", 10),
			makeBM25SearchResult("src/bm25-only.ts", 5),
		},
		[]SearchResult{
			makeSemanticSearchResult("src/shared.ts", 0.9),
			makeSemanticSearchResult("src/semantic-only.ts", 0.8),
		},
		10,
	)
	if combined[0].FilePath != "src/shared.ts" || combined[0].Score <= combined[1].Score {
		t.Fatalf("combined merge = %#v", combined)
	}
	if !reflect.DeepEqual(combined[0].Sources, []string{searchModeBM25, searchModeSemantic}) {
		t.Fatalf("combined sources = %#v", combined[0].Sources)
	}
	if combined[0].NodeID != "node:src/shared.ts" || combined[0].Name != "shared" || combined[0].Label != "Function" {
		t.Fatalf("combined semantic metadata = %#v", combined[0])
	}
	if combined[0].BM25Score != 10 || combined[0].SemanticScore != 0.9 {
		t.Fatalf("combined original scores = %#v", combined[0])
	}

	limited := mergeSearchResultsWithRRF(makeManyBM25Results(20), nil, 5)
	if len(limited) != 5 {
		t.Fatalf("limited merge len = %d, want 5", len(limited))
	}
	defaultLimited := mergeSearchResultsWithRRF(makeManyBM25Results(20), nil, 0)
	if len(defaultLimited) != defaultAPISearchLimit {
		t.Fatalf("default limited merge len = %d, want %d", len(defaultLimited), defaultAPISearchLimit)
	}
	for index, result := range limited {
		if result.Rank != index+1 {
			t.Fatalf("rank[%d] = %d", index, result.Rank)
		}
	}
}

func TestSearchServiceHybridMergesBM25AndSemanticAndFallsBack(t *testing.T) {
	runner := &semanticSearchRunner{
		ftsRows: map[string][]lbugruntime.Row{
			"File": {
				{"filePath": "src/shared.ts", "score": 10},
				{"filePath": "src/bm25-only.ts", "score": 5},
			},
		},
		vectorRows: []lbugruntime.Row{
			{"nodeId": "Function:shared", "chunkIndex": 0, "startLine": 3, "endLine": 9, "distance": 0.10},
			{"nodeId": "Function:semantic-only", "chunkIndex": 0, "startLine": 11, "endLine": 15, "distance": 0.20},
		},
		metadataRows: map[string][]lbugruntime.Row{
			"Function": {
				{"id": "Function:shared", "name": "shared", "filePath": "src/shared.ts"},
				{"id": "Function:semantic-only", "name": "semanticOnly", "filePath": "src/semantic-only.ts"},
			},
		},
	}
	service := SearchService{
		OpenReadRunner: func(string) (lbugnative.ReadRunner, error) {
			return runner, nil
		},
		ResolveEmbedder: func() (embeddings.Embedder, embeddings.Config, error) {
			return staticSearchEmbedder{vector: []float32{0.1, 0.2}}, embeddings.Config{Dimensions: 2}, nil
		},
	}

	results, err := service.Search(context.Background(), SearchTarget{LbugPath: "repo/.avmatrix/lbug"}, SearchRequest{
		Query: "shared auth",
		Mode:  searchModeHybrid,
		Limit: 3,
	})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("results len = %d, want 3: %#v", len(results), results)
	}
	if results[0].FilePath != "src/shared.ts" {
		t.Fatalf("top hybrid result = %#v", results[0])
	}
	if !reflect.DeepEqual(results[0].Sources, []string{searchModeBM25, searchModeSemantic}) {
		t.Fatalf("hybrid sources = %#v", results[0].Sources)
	}
	if results[0].BM25Score != 10 || results[0].SemanticScore != 0.9 || results[0].NodeID != "Function:shared" {
		t.Fatalf("hybrid metadata = %#v", results[0])
	}

	fallbackRunner := &semanticSearchRunner{
		ftsRows: map[string][]lbugruntime.Row{
			"File": {{"filePath": "src/auth.ts", "score": 5}},
		},
	}
	fallbackService := SearchService{
		OpenReadRunner: func(string) (lbugnative.ReadRunner, error) {
			return fallbackRunner, nil
		},
		ResolveEmbedder: func() (embeddings.Embedder, embeddings.Config, error) {
			return nil, embeddings.Config{}, errors.New("embedder offline")
		},
	}
	results, err = fallbackService.Search(context.Background(), SearchTarget{LbugPath: "repo/.avmatrix/lbug"}, SearchRequest{
		Query: "authentication",
		Mode:  searchModeHybrid,
		Limit: 2,
	})
	if err != nil {
		t.Fatalf("fallback Search() error = %v", err)
	}
	if len(results) != 1 || results[0].FilePath != "src/auth.ts" || !reflect.DeepEqual(results[0].Sources, []string{searchModeBM25}) {
		t.Fatalf("fallback results = %#v", results)
	}
}

func TestSearchServiceMapsNativeUnavailable(t *testing.T) {
	service := SearchService{
		OpenReadRunner: func(string) (lbugnative.ReadRunner, error) {
			return nil, lbugnative.ErrUnavailable
		},
	}

	_, err := service.Search(context.Background(), SearchTarget{LbugPath: "missing"}, SearchRequest{
		Query: "alpha",
		Mode:  searchModeSemantic,
		Limit: 1,
	})
	if !errors.Is(err, ErrSearchUnavailable) {
		t.Fatalf("Search() error = %v, want ErrSearchUnavailable", err)
	}
}

type recordingSearcher struct {
	target  SearchTarget
	request SearchRequest
	results []SearchResult
	err     error
}

func (s *recordingSearcher) Search(_ context.Context, target SearchTarget, request SearchRequest) ([]SearchResult, error) {
	s.target = target
	s.request = request
	return s.results, s.err
}

type semanticSearchRunner struct {
	closed       bool
	queries      []string
	ftsRows      map[string][]lbugruntime.Row
	ftsErrors    map[string]error
	vectorRows   []lbugruntime.Row
	metadataRows map[string][]lbugruntime.Row
}

func (r *semanticSearchRunner) QueryRows(query string) ([]lbugruntime.Row, error) {
	r.queries = append(r.queries, query)
	if strings.Contains(query, "QUERY_FTS_INDEX") {
		table := bm25TableFromQuery(query)
		if err := r.ftsErrors[table]; err != nil {
			return nil, err
		}
		return r.ftsRows[table], nil
	}
	if strings.Contains(query, "QUERY_VECTOR_INDEX") {
		if r.vectorRows != nil {
			return r.vectorRows, nil
		}
		return []lbugruntime.Row{{
			"nodeId":     "Function:alpha",
			"chunkIndex": 0,
			"startLine":  3,
			"endLine":    9,
			"distance":   0.12,
		}}, nil
	}
	for label, rows := range r.metadataRows {
		if strings.Contains(query, "MATCH (n:"+label+")") {
			return rows, nil
		}
	}
	return []lbugruntime.Row{{
		"id":       "Function:alpha",
		"name":     "alpha",
		"filePath": "src/app.ts",
	}}, nil
}

func (r *semanticSearchRunner) Close() error {
	r.closed = true
	return nil
}

type staticSearchEmbedder struct {
	vector []float32
}

func (e staticSearchEmbedder) Embed(context.Context, []string) ([][]float32, error) {
	return [][]float32{e.vector}, nil
}

func makeBM25SearchResult(filePath string, score float64) SearchResult {
	return SearchResult{FilePath: filePath, Score: score}
}

func makeSemanticSearchResult(filePath string, score float64) SearchResult {
	name := filepath.Base(filePath)
	if extension := filepath.Ext(name); extension != "" {
		name = strings.TrimSuffix(name, extension)
	}
	return SearchResult{
		FilePath:  filePath,
		Score:     score,
		NodeID:    "node:" + filePath,
		Name:      name,
		Label:     "Function",
		StartLine: 1,
		EndLine:   10,
	}
}

func makeManyBM25Results(count int) []SearchResult {
	results := make([]SearchResult, 0, count)
	for index := 0; index < count; index++ {
		results = append(results, makeBM25SearchResult(fmt.Sprintf("src/%d.ts", index), float64(100-index)))
	}
	return results
}

func bm25QueryTables(queries []string) []string {
	tables := make([]string, 0, len(queries))
	for _, query := range queries {
		if !strings.Contains(query, "QUERY_FTS_INDEX") {
			continue
		}
		tables = append(tables, bm25TableFromQuery(query))
	}
	return tables
}

func bm25TableFromQuery(query string) string {
	for _, index := range bm25FTSIndexes {
		if strings.Contains(query, "'"+index.tableName+"'") {
			return index.tableName
		}
	}
	return ""
}

func postJSON(t *testing.T, targetURL string, body string, wantStatus int, target any) {
	t.Helper()

	resp, err := http.Post(targetURL, "application/json", bytes.NewBufferString(body))
	if err != nil {
		t.Fatalf("POST %s failed: %v", targetURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != wantStatus {
		raw, _ := io.ReadAll(resp.Body)
		t.Fatalf("POST %s status = %d, want %d; body=%s", targetURL, resp.StatusCode, wantStatus, raw)
	}
	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		t.Fatalf("decode %s JSON: %v", targetURL, err)
	}
}

func jsonEscape(value string) string {
	raw, _ := json.Marshal(value)
	return strings.Trim(string(raw), `"`)
}
