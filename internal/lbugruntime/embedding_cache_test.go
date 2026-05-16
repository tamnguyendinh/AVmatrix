package lbugruntime

import (
	"errors"
	"strings"
	"testing"

	"github.com/tamnguyendinh/avmatrix-go/internal/lbugschema"
)

func TestFetchExistingEmbeddingHashesReadsCurrentSchema(t *testing.T) {
	result, err := FetchExistingEmbeddingHashes(func(query string) ([]Row, error) {
		if !strings.Contains(query, "contentHash AS contentHash") {
			t.Fatalf("unexpected query: %s", query)
		}
		return []Row{
			{"nodeId": "Function:a", "chunkIndex": 0, "startLine": 1, "endLine": 2, "contentHash": "hash-a"},
			{"nodeId": "Function:b", "chunkIndex": 1, "startLine": 3, "endLine": 4, "contentHash": ""},
			{"nodeId": "Function:legacy-metadata", "chunkIndex": nil, "startLine": nil, "endLine": nil, "contentHash": "hash-b"},
		}, nil
	})
	if err != nil {
		t.Fatalf("FetchExistingEmbeddingHashes() error = %v", err)
	}
	if !result.Found || result.Legacy {
		t.Fatalf("result flags = found %v legacy %v, want found true legacy false", result.Found, result.Legacy)
	}
	if result.Hashes["Function:a"] != "hash-a" {
		t.Fatalf("Function:a hash = %q, want hash-a", result.Hashes["Function:a"])
	}
	if got, ok := result.Hashes["Function:b"]; !ok || got != lbugschema.StaleHashSentinel {
		t.Fatalf("Function:b stale hash = %q/%v, want sentinel", got, ok)
	}
	if got, ok := result.Hashes["Function:legacy-metadata"]; !ok || got != lbugschema.StaleHashSentinel {
		t.Fatalf("Function:legacy-metadata stale hash = %q/%v, want sentinel", got, ok)
	}
}

func TestFetchExistingEmbeddingHashesFallsBackForLegacyRows(t *testing.T) {
	calls := 0
	result, err := FetchExistingEmbeddingHashes(func(query string) ([]Row, error) {
		calls++
		if calls == 1 {
			return nil, errors.New("column contentHash does not exist")
		}
		if !strings.Contains(query, "RETURN e.nodeId AS nodeId") {
			t.Fatalf("unexpected fallback query: %s", query)
		}
		return []Row{{"nodeId": "Function:legacy"}}, nil
	})
	if err != nil {
		t.Fatalf("FetchExistingEmbeddingHashes() error = %v", err)
	}
	if calls != 2 {
		t.Fatalf("query calls = %d, want 2", calls)
	}
	if !result.Found || !result.Legacy {
		t.Fatalf("result flags = found %v legacy %v, want found true legacy true", result.Found, result.Legacy)
	}
	if got, ok := result.Hashes["Function:legacy"]; !ok || got != lbugschema.StaleHashSentinel {
		t.Fatalf("legacy stale hash = %q/%v, want sentinel", got, ok)
	}
}

func TestFetchExistingEmbeddingHashesMissingTableMeansNoCache(t *testing.T) {
	result, err := FetchExistingEmbeddingHashes(func(query string) ([]Row, error) {
		return nil, errors.New("table CodeEmbedding not found")
	})
	if err != nil {
		t.Fatalf("FetchExistingEmbeddingHashes() error = %v", err)
	}
	if result.Found || result.Legacy || len(result.Hashes) != 0 {
		t.Fatalf("result = %#v, want empty no-cache result", result)
	}
}
