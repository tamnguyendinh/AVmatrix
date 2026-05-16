package embeddings

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	HTTPTimeout      = 30 * time.Second
	HTTPMaxRetries   = 2
	HTTPRetryBackoff = time.Second
	HTTPBatchSize    = 64
)

type HTTPEmbedder struct {
	Config       HTTPConfig
	Client       *http.Client
	Timeout      time.Duration
	MaxRetries   int
	RetryBackoff time.Duration
	BatchSize    int
	Sleep        func(context.Context, time.Duration) error
}

func NewHTTPEmbedder(config HTTPConfig, client *http.Client) HTTPEmbedder {
	return HTTPEmbedder{
		Config:       config,
		Client:       client,
		Timeout:      HTTPTimeout,
		MaxRetries:   HTTPMaxRetries,
		RetryBackoff: HTTPRetryBackoff,
		BatchSize:    HTTPBatchSize,
	}
}

func (e HTTPEmbedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}
	if err := e.validateConfig(); err != nil {
		return nil, err
	}
	if ctx == nil {
		ctx = context.Background()
	}

	batchSize := e.BatchSize
	if batchSize <= 0 {
		batchSize = HTTPBatchSize
	}

	vectors := make([][]float32, 0, len(texts))
	for start := 0; start < len(texts); start += batchSize {
		end := min(start+batchSize, len(texts))
		batchIndex := start / batchSize
		items, err := e.embedBatch(ctx, texts[start:end], batchIndex, 0)
		if err != nil {
			return nil, err
		}
		if len(items) != end-start {
			return nil, fmt.Errorf("embedding endpoint returned %d vectors for %d texts (%s, batch %d)", len(items), end-start, safeURL(e.endpoint()), batchIndex)
		}
		for _, item := range items {
			if err := e.validateVector(item.Embedding); err != nil {
				return nil, err
			}
			vectors = append(vectors, item.Embedding)
		}
	}
	return vectors, nil
}

func (e HTTPEmbedder) EmbedQuery(ctx context.Context, text string) ([]float32, error) {
	if err := e.validateConfig(); err != nil {
		return nil, err
	}
	if ctx == nil {
		ctx = context.Background()
	}
	items, err := e.embedBatch(ctx, []string{text}, 0, 0)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("embedding endpoint returned empty response (%s)", safeURL(e.endpoint()))
	}
	if err := e.validateVector(items[0].Embedding); err != nil {
		return nil, err
	}
	return items[0].Embedding, nil
}

func (e HTTPEmbedder) embedBatch(ctx context.Context, batch []string, batchIndex int, attempt int) ([]embeddingItem, error) {
	endpoint := e.endpoint()
	body, err := json.Marshal(embeddingRequest{Input: batch, Model: e.Config.Model})
	if err != nil {
		return nil, err
	}

	timeout := e.Timeout
	if timeout <= 0 {
		timeout = HTTPTimeout
	}
	requestCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(requestCtx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+e.Config.APIKey)

	resp, err := e.httpClient().Do(req)
	if err != nil {
		if errors.Is(requestCtx.Err(), context.DeadlineExceeded) {
			return nil, fmt.Errorf("embedding request timed out after %s (%s, batch %d)", timeout, safeURL(endpoint), batchIndex)
		}
		if attempt < e.maxRetries() {
			if err := e.sleep(ctx, attempt); err != nil {
				return nil, err
			}
			return e.embedBatch(ctx, batch, batchIndex, attempt+1)
		}
		return nil, fmt.Errorf("embedding request failed (%s, batch %d): %w", safeURL(endpoint), batchIndex, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		if isRetryableStatus(resp.StatusCode) && attempt < e.maxRetries() {
			if err := e.sleep(ctx, attempt); err != nil {
				return nil, err
			}
			return e.embedBatch(ctx, batch, batchIndex, attempt+1)
		}
		return nil, fmt.Errorf("embedding endpoint returned %d (%s, batch %d)", resp.StatusCode, safeURL(endpoint), batchIndex)
	}

	var payload embeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("embedding endpoint returned invalid JSON (%s, batch %d): %w", safeURL(endpoint), batchIndex, err)
	}
	return payload.Data, nil
}

func (e HTTPEmbedder) validateConfig() error {
	if strings.TrimSpace(e.Config.BaseURL) == "" {
		return fmt.Errorf("HTTP embedding base URL is required")
	}
	if strings.TrimSpace(e.Config.Model) == "" {
		return fmt.Errorf("HTTP embedding model is required")
	}
	if e.Config.Dimensions < 0 {
		return fmt.Errorf("embedding dimensions must be positive, got %d", e.Config.Dimensions)
	}
	return nil
}

func (e HTTPEmbedder) validateVector(vector []float32) error {
	expected := e.Config.ExpectedDimensions()
	if len(vector) == expected {
		return nil
	}
	hint := fmt.Sprintf("Set %s=%d to match your model output.", EnvEmbeddingDims, len(vector))
	if e.Config.Dimensions > 0 {
		hint = fmt.Sprintf("Update %s to match your model output.", EnvEmbeddingDims)
	}
	return fmt.Errorf("embedding dimension mismatch: endpoint returned %dd vector, but expected %dd. %s", len(vector), expected, hint)
}

func (e HTTPEmbedder) endpoint() string {
	return strings.TrimRight(e.Config.BaseURL, "/") + "/embeddings"
}

func (e HTTPEmbedder) httpClient() *http.Client {
	if e.Client != nil {
		return e.Client
	}
	return http.DefaultClient
}

func (e HTTPEmbedder) maxRetries() int {
	if e.MaxRetries < 0 {
		return 0
	}
	if e.MaxRetries == 0 {
		return HTTPMaxRetries
	}
	return e.MaxRetries
}

func (e HTTPEmbedder) sleep(ctx context.Context, attempt int) error {
	delay := e.RetryBackoff
	if delay == 0 {
		delay = HTTPRetryBackoff
	}
	delay *= time.Duration(attempt + 1)
	if e.Sleep != nil {
		return e.Sleep(ctx, delay)
	}
	if delay <= 0 {
		return nil
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func isRetryableStatus(status int) bool {
	return status == http.StatusTooManyRequests || status >= 500
}

func safeURL(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "<invalid-url>"
	}
	parsed.User = nil
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String()
}

type embeddingRequest struct {
	Input []string `json:"input"`
	Model string   `json:"model"`
}

type embeddingResponse struct {
	Data []embeddingItem `json:"data"`
}

type embeddingItem struct {
	Embedding []float32 `json:"embedding"`
}
