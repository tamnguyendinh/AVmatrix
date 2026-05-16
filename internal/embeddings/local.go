package embeddings

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"

	"github.com/knights-analytics/hugot"
	hugotpipelines "github.com/knights-analytics/hugot/pipelines"
)

const (
	DefaultLocalONNXFilePath = "onnx/model.onnx"
	defaultLocalRevision     = "main"
	defaultHFEndpoint        = "https://huggingface.co"
	localPipelineName        = "avmatrix-local-embedding"
	localCacheSubdir         = "avmatrix-go"
)

var (
	localEmbedderFactory = cachedHugotLocalEmbedder
	localRuntimeMu       sync.Mutex
	localRuntimeKey      string
	localRuntime         *HugotLocalEmbedder
)

type localModelFile struct {
	remote   string
	local    string
	required bool
}

type localModelRef struct {
	model    string
	revision string
}

var localModelFiles = []localModelFile{
	{remote: DefaultLocalONNXFilePath, local: path.Base(DefaultLocalONNXFilePath), required: true},
	{remote: "tokenizer.json", local: "tokenizer.json", required: true},
	{remote: "config.json", local: "config.json", required: true},
	{remote: "tokenizer_config.json", local: "tokenizer_config.json"},
	{remote: "special_tokens_map.json", local: "special_tokens_map.json"},
	{remote: "vocab.txt", local: "vocab.txt"},
	{remote: "genai_config.json", local: "genai_config.json"},
	{remote: "chat_template.jinja", local: "chat_template.jinja"},
}

type HugotLocalEmbedder struct {
	config   Config
	session  *hugot.Session
	pipeline *hugotpipelines.FeatureExtractionPipeline
	mu       sync.Mutex
	closed   bool
}

func ResolveRuntimeEmbedder(ctx context.Context, config Config, lookup EnvLookup) (Embedder, Config, error) {
	config = NormalizeConfig(config)
	httpConfig, err := ReadHTTPConfig(lookup)
	if err != nil {
		return nil, config, err
	}
	if httpConfig != nil {
		if httpConfig.Dimensions > 0 {
			config.Dimensions = httpConfig.Dimensions
		}
		return NewHTTPEmbedder(*httpConfig, nil), config, nil
	}
	embedder, err := NewLocalEmbedder(ctx, config)
	if err != nil {
		return nil, config, err
	}
	return embedder, config, nil
}

func NewLocalEmbedder(ctx context.Context, config Config) (Embedder, error) {
	if localEmbedderFactory == nil {
		return nil, fmt.Errorf("local embedding factory is not configured")
	}
	return localEmbedderFactory(ctx, NormalizeConfig(config))
}

func cachedHugotLocalEmbedder(ctx context.Context, config Config) (Embedder, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	cacheDir, err := LocalModelCacheDir()
	if err != nil {
		return nil, err
	}
	key := strings.Join([]string{config.ModelID, DefaultLocalONNXFilePath, cacheDir, fmt.Sprint(config.Dimensions)}, "\x00")

	localRuntimeMu.Lock()
	defer localRuntimeMu.Unlock()
	if localRuntime != nil && localRuntimeKey == key {
		return localRuntime, nil
	}

	created, err := loadHugotLocalEmbedder(ctx, config, cacheDir)
	if err != nil {
		return nil, err
	}
	previous := localRuntime
	localRuntime = created
	localRuntimeKey = key
	if previous != nil {
		_ = previous.Close()
	}
	return created, nil
}

func loadHugotLocalEmbedder(ctx context.Context, config Config, cacheDir string) (*HugotLocalEmbedder, error) {
	modelPath, err := ensureLocalModel(ctx, config, cacheDir)
	if err != nil {
		return nil, err
	}
	session, err := hugot.NewGoSession(ctx)
	if err != nil {
		return nil, fmt.Errorf("initialize local embedding session: %w", err)
	}
	pipeline, err := hugot.NewPipeline(session, hugot.FeatureExtractionConfig{
		ModelPath:    modelPath,
		Name:         localPipelineName,
		OnnxFilename: path.Base(DefaultLocalONNXFilePath),
		Options: []hugot.FeatureExtractionOption{
			hugotpipelines.WithNormalization(),
		},
	})
	if err != nil {
		_ = session.Destroy()
		return nil, fmt.Errorf("initialize local embedding pipeline: %w", err)
	}
	return &HugotLocalEmbedder{
		config:   config,
		session:  session,
		pipeline: pipeline,
	}, nil
}

func ensureLocalModel(ctx context.Context, config Config, cacheDir string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if cacheDir == "" {
		return "", fmt.Errorf("local embedding cache directory is empty")
	}
	hfHome := filepath.Dir(cacheDir)
	if err := configureLocalModelCacheEnv(hfHome); err != nil {
		return "", err
	}
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return "", err
	}
	modelPath := filepath.Join(cacheDir, strings.ReplaceAll(strings.Split(config.ModelID, ":")[0], "/", "_"))
	if localModelReady(modelPath) {
		return modelPath, nil
	}
	if err := downloadLocalModel(ctx, config.ModelID, modelPath); err != nil {
		return "", fmt.Errorf("download local embedding model %q: %w", config.ModelID, err)
	}
	if !localModelReady(modelPath) {
		return "", fmt.Errorf("download local embedding model %q: required files are incomplete", config.ModelID)
	}
	return modelPath, nil
}

func localModelReady(modelPath string) bool {
	for _, name := range []string{path.Base(DefaultLocalONNXFilePath), "tokenizer.json", "config.json"} {
		if _, err := os.Stat(filepath.Join(modelPath, name)); err != nil {
			return false
		}
	}
	return true
}

func downloadLocalModel(ctx context.Context, modelID string, modelPath string) error {
	ref, err := parseLocalModelRef(modelID)
	if err != nil {
		return err
	}
	endpoint, err := localModelEndpoint()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(modelPath, 0o755); err != nil {
		return err
	}
	client := http.DefaultClient
	for _, file := range localModelFiles {
		if _, err := os.Stat(filepath.Join(modelPath, file.local)); err == nil {
			continue
		}
		fileURL, err := localModelFileURL(endpoint, ref, file.remote)
		if err != nil {
			return err
		}
		if err := downloadLocalModelFile(ctx, client, fileURL, filepath.Join(modelPath, file.local), file.required); err != nil {
			return err
		}
	}
	return nil
}

func parseLocalModelRef(modelID string) (localModelRef, error) {
	modelID = strings.TrimSpace(modelID)
	if modelID == "" {
		return localModelRef{}, fmt.Errorf("local embedding model id is empty")
	}
	ref := localModelRef{model: modelID, revision: defaultLocalRevision}
	if before, after, ok := strings.Cut(modelID, ":"); ok {
		ref.model = strings.TrimSpace(before)
		ref.revision = strings.TrimSpace(after)
	}
	if ref.model == "" || ref.revision == "" {
		return localModelRef{}, fmt.Errorf("local embedding model id %q must include a model and revision", modelID)
	}
	return ref, nil
}

func localModelEndpoint() (*url.URL, error) {
	rawEndpoint := strings.TrimRight(strings.TrimSpace(os.Getenv("HF_ENDPOINT")), "/")
	if rawEndpoint == "" {
		rawEndpoint = defaultHFEndpoint
	}
	endpoint, err := url.Parse(rawEndpoint)
	if err != nil || endpoint.Scheme == "" || endpoint.Host == "" {
		return nil, fmt.Errorf("invalid HF_ENDPOINT for local embedding model: %q", rawEndpoint)
	}
	return endpoint, nil
}

func localModelFileURL(endpoint *url.URL, ref localModelRef, remotePath string) (string, error) {
	if endpoint == nil {
		return "", fmt.Errorf("local embedding model endpoint is nil")
	}
	segments := strings.Split(strings.Trim(endpoint.Path, "/"), "/")
	segments = append(segments, strings.Split(ref.model, "/")...)
	segments = append(segments, "resolve", ref.revision)
	segments = append(segments, strings.Split(remotePath, "/")...)
	escaped := make([]string, 0, len(segments))
	for _, segment := range segments {
		if segment == "" {
			continue
		}
		escaped = append(escaped, url.PathEscape(segment))
	}
	clone := *endpoint
	clone.Path = "/" + strings.Join(escaped, "/")
	return clone.String(), nil
}

func downloadLocalModelFile(ctx context.Context, client *http.Client, fileURL string, destination string, required bool) error {
	if client == nil {
		client = http.DefaultClient
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, fileURL, nil)
	if err != nil {
		return err
	}
	if token := strings.TrimSpace(os.Getenv("HF_TOKEN")); token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusNotFound && !required {
		return nil
	}
	if response.StatusCode < 200 || response.StatusCode > 299 {
		return fmt.Errorf("download %s: unexpected status %s", fileURL, response.Status)
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return err
	}
	tmpPath := destination + ".downloading"
	output, err := os.Create(tmpPath)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(output, response.Body)
	closeErr := output.Close()
	if copyErr != nil || closeErr != nil {
		_ = os.Remove(tmpPath)
		if copyErr != nil {
			return copyErr
		}
		return closeErr
	}
	if err := os.Rename(tmpPath, destination); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	return nil
}

func LocalModelCacheDir() (string, error) {
	hfHome, err := localHFHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(hfHome, localCacheSubdir), nil
}

func localHFHomeDir() (string, error) {
	if hfHome := strings.TrimSpace(os.Getenv("HF_HOME")); hfHome != "" {
		absolute, err := filepath.Abs(hfHome)
		if err != nil {
			return "", fmt.Errorf("resolve HF_HOME for local embedding cache: %w", err)
		}
		return absolute, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory for local embedding cache: %w", err)
	}
	return filepath.Join(home, ".cache", "huggingface"), nil
}

func configureLocalModelCacheEnv(hfHome string) error {
	if hfHome == "" {
		return fmt.Errorf("local embedding HF_HOME is empty")
	}
	if err := os.Setenv("HF_HOME", hfHome); err != nil {
		return fmt.Errorf("set HF_HOME for local embedding model cache: %w", err)
	}
	xdgCache := strings.TrimSpace(os.Getenv("XDG_CACHE_HOME"))
	if xdgCache == "" {
		xdgCache = filepath.Join(hfHome, "go-cache")
	} else {
		absolute, err := filepath.Abs(xdgCache)
		if err != nil {
			return fmt.Errorf("resolve XDG_CACHE_HOME for local embedding model cache: %w", err)
		}
		xdgCache = absolute
	}
	if err := os.Setenv("XDG_CACHE_HOME", xdgCache); err != nil {
		return fmt.Errorf("set XDG_CACHE_HOME for local embedding model cache: %w", err)
	}
	return nil
}

func (e *HugotLocalEmbedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.closed {
		return nil, fmt.Errorf("local embedding runtime is closed")
	}
	result, err := e.pipeline.RunPipeline(ctx, texts)
	if err != nil {
		return nil, err
	}
	if len(result.Embeddings) != len(texts) {
		return nil, fmt.Errorf("local embedding model returned %d vectors for %d texts", len(result.Embeddings), len(texts))
	}
	vectors := make([][]float32, 0, len(result.Embeddings))
	for _, embedding := range result.Embeddings {
		if len(embedding) != e.config.Dimensions {
			return nil, fmt.Errorf("local embedding dimension mismatch: model returned %dd vector, but expected %dd", len(embedding), e.config.Dimensions)
		}
		vectors = append(vectors, append([]float32(nil), embedding...))
	}
	return vectors, nil
}

func (e *HugotLocalEmbedder) Close() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.closed {
		return nil
	}
	e.closed = true
	if e.session == nil {
		return nil
	}
	return e.session.Destroy()
}
