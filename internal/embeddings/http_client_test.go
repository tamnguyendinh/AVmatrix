package embeddings

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestHTTPEmbedderPostsOpenAICompatibleBatches(t *testing.T) {
	var calls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		if r.URL.Path != "/v1/embeddings" {
			t.Fatalf("path = %q, want /v1/embeddings", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer secret" {
			t.Fatalf("Authorization = %q", r.Header.Get("Authorization"))
		}
		var request embeddingRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if request.Model != "local-embedder" {
			t.Fatalf("model = %q", request.Model)
		}
		response := embeddingResponse{Data: make([]embeddingItem, 0, len(request.Input))}
		for range request.Input {
			response.Data = append(response.Data, embeddingItem{Embedding: []float32{1, 2, 3}})
		}
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	embedder := NewHTTPEmbedder(HTTPConfig{
		BaseURL:    server.URL + "/v1",
		Model:      "local-embedder",
		APIKey:     "secret",
		Dimensions: 3,
	}, server.Client())
	embedder.BatchSize = 2

	vectors, err := embedder.Embed(context.Background(), []string{"a", "b", "c"})
	if err != nil {
		t.Fatalf("Embed() error = %v", err)
	}
	if len(vectors) != 3 {
		t.Fatalf("len(vectors) = %d, want 3", len(vectors))
	}
	if calls != 2 {
		t.Fatalf("calls = %d, want 2", calls)
	}
}

func TestHTTPEmbedderEmbedQueryRejectsEmptyResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(embeddingResponse{})
	}))
	defer server.Close()

	embedder := NewHTTPEmbedder(HTTPConfig{
		BaseURL:    server.URL,
		Model:      "local-embedder",
		Dimensions: 3,
	}, server.Client())

	_, err := embedder.EmbedQuery(context.Background(), "query")
	if err == nil || !strings.Contains(err.Error(), "empty response") {
		t.Fatalf("EmbedQuery() error = %v, want empty response", err)
	}
}

func TestHTTPEmbedderRejectsVectorCountMismatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(embeddingResponse{Data: []embeddingItem{{Embedding: []float32{1, 2, 3}}}})
	}))
	defer server.Close()

	embedder := NewHTTPEmbedder(HTTPConfig{
		BaseURL:    server.URL,
		Model:      "local-embedder",
		Dimensions: 3,
	}, server.Client())

	_, err := embedder.Embed(context.Background(), []string{"a", "b"})
	if err == nil || !strings.Contains(err.Error(), "returned 1 vectors for 2 texts") {
		t.Fatalf("Embed() error = %v, want vector count mismatch", err)
	}
}

func TestHTTPEmbedderRejectsDimensionMismatchWithHint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(embeddingResponse{Data: []embeddingItem{{Embedding: []float32{1, 2}}}})
	}))
	defer server.Close()

	embedder := NewHTTPEmbedder(HTTPConfig{
		BaseURL: server.URL,
		Model:   "local-embedder",
	}, server.Client())

	_, err := embedder.Embed(context.Background(), []string{"a"})
	if err == nil || !strings.Contains(err.Error(), "expected 384d") || !strings.Contains(err.Error(), "AVMATRIX_EMBEDDING_DIMS=2") {
		t.Fatalf("Embed() error = %v, want dimension mismatch hint", err)
	}
}

func TestHTTPEmbedderRetriesRetryableStatus(t *testing.T) {
	var calls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		call := atomic.AddInt32(&calls, 1)
		if call == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		_ = json.NewEncoder(w).Encode(embeddingResponse{Data: []embeddingItem{{Embedding: []float32{1, 2, 3}}}})
	}))
	defer server.Close()

	embedder := NewHTTPEmbedder(HTTPConfig{
		BaseURL:    server.URL,
		Model:      "local-embedder",
		Dimensions: 3,
	}, server.Client())
	embedder.RetryBackoff = -time.Nanosecond
	embedder.Sleep = func(context.Context, time.Duration) error { return nil }

	vectors, err := embedder.Embed(context.Background(), []string{"a"})
	if err != nil {
		t.Fatalf("Embed() error = %v", err)
	}
	if len(vectors) != 1 || calls != 2 {
		t.Fatalf("vectors=%d calls=%d, want vectors=1 calls=2", len(vectors), calls)
	}
}

func TestHTTPEmbedderDoesNotRetryTimeout(t *testing.T) {
	var calls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		time.Sleep(30 * time.Millisecond)
		_ = json.NewEncoder(w).Encode(embeddingResponse{Data: []embeddingItem{{Embedding: []float32{1, 2, 3}}}})
	}))
	defer server.Close()

	embedder := NewHTTPEmbedder(HTTPConfig{
		BaseURL:    server.URL,
		Model:      "local-embedder",
		Dimensions: 3,
	}, server.Client())
	embedder.Timeout = time.Millisecond
	embedder.MaxRetries = 3
	embedder.RetryBackoff = -time.Nanosecond
	embedder.Sleep = func(context.Context, time.Duration) error { return nil }

	_, err := embedder.Embed(context.Background(), []string{"a"})
	if err == nil || !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("Embed() error = %v, want timeout", err)
	}
	if calls != 1 {
		t.Fatalf("calls = %d, want 1", calls)
	}
}

func TestHTTPEmbedderStatusErrorsExcludeSecrets(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	baseURL := strings.Replace(server.URL, "http://", "http://user:pass@", 1) + "/v1?api_key=secret"
	embedder := NewHTTPEmbedder(HTTPConfig{
		BaseURL:    baseURL,
		Model:      "local-embedder",
		APIKey:     "secret-key",
		Dimensions: 3,
	}, server.Client())
	embedder.MaxRetries = -1

	_, err := embedder.Embed(context.Background(), []string{"a"})
	if err == nil || !strings.Contains(err.Error(), "returned 500") {
		t.Fatalf("Embed() error = %v, want status error", err)
	}
	for _, secret := range []string{"user:pass", "api_key", "secret", "secret-key"} {
		if strings.Contains(err.Error(), secret) {
			t.Fatalf("Embed() error leaked %q: %v", secret, err)
		}
	}
}

func TestSafeURLStripsSecrets(t *testing.T) {
	got := safeURL("https://user:pass@example.com/v1/embeddings?api_key=secret#frag")
	if got != "https://example.com/v1/embeddings" {
		t.Fatalf("safeURL() = %q", got)
	}
}
