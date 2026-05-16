package embeddings

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveRuntimeEmbedderPrefersHTTPMode(t *testing.T) {
	originalFactory := localEmbedderFactory
	localEmbedderFactory = func(context.Context, Config) (Embedder, error) {
		t.Fatal("local embedder should not be created in HTTP mode")
		return nil, nil
	}
	defer func() {
		localEmbedderFactory = originalFactory
	}()

	embedder, config, err := ResolveRuntimeEmbedder(context.Background(), Config{}, mapLookup(map[string]string{
		EnvEmbeddingURL:   "http://127.0.0.1:8080/v1",
		EnvEmbeddingModel: "local-embedder",
		EnvEmbeddingDims:  "768",
	}))
	if err != nil {
		t.Fatalf("ResolveRuntimeEmbedder() error = %v", err)
	}
	if _, ok := embedder.(HTTPEmbedder); !ok {
		t.Fatalf("embedder type = %T, want HTTPEmbedder", embedder)
	}
	if config.Dimensions != 768 {
		t.Fatalf("dimensions = %d, want 768", config.Dimensions)
	}
}

func TestResolveRuntimeEmbedderUsesLocalModeWhenHTTPUnset(t *testing.T) {
	originalFactory := localEmbedderFactory
	defer func() {
		localEmbedderFactory = originalFactory
	}()

	var got Config
	localEmbedderFactory = func(_ context.Context, config Config) (Embedder, error) {
		got = config
		return stubEmbedder{}, nil
	}

	embedder, config, err := ResolveRuntimeEmbedder(context.Background(), Config{Dimensions: 12}, mapLookup(nil))
	if err != nil {
		t.Fatalf("ResolveRuntimeEmbedder() error = %v", err)
	}
	if _, ok := embedder.(stubEmbedder); !ok {
		t.Fatalf("embedder type = %T, want stubEmbedder", embedder)
	}
	if got.ModelID != DefaultModelID || got.Dimensions != 12 || config.Dimensions != 12 {
		t.Fatalf("local config = %#v, resolved = %#v", got, config)
	}
}

func TestLocalModelCacheDirHonorsHFHome(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HF_HOME", root)

	got, err := LocalModelCacheDir()
	if err != nil {
		t.Fatalf("LocalModelCacheDir() error = %v", err)
	}
	if got != filepath.Join(root, localCacheSubdir) {
		t.Fatalf("cache dir = %q, want %q", got, filepath.Join(root, localCacheSubdir))
	}
}

func TestLocalModelReadyRequiresModelTokenizerAndConfig(t *testing.T) {
	dir := t.TempDir()
	if localModelReady(dir) {
		t.Fatal("localModelReady() = true for empty dir")
	}
	for _, name := range []string{"model.onnx", "tokenizer.json", "config.json"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("{}\n"), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	if !localModelReady(dir) {
		t.Fatal("localModelReady() = false for complete local model")
	}
}

func TestConfigureLocalModelCacheEnvSetsAbsoluteHubCache(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HF_HOME", "")
	t.Setenv("XDG_CACHE_HOME", "")

	if err := configureLocalModelCacheEnv(root); err != nil {
		t.Fatalf("configureLocalModelCacheEnv() error = %v", err)
	}
	if os.Getenv("HF_HOME") != root {
		t.Fatalf("HF_HOME = %q, want %q", os.Getenv("HF_HOME"), root)
	}
	if os.Getenv("XDG_CACHE_HOME") != filepath.Join(root, "go-cache") {
		t.Fatalf("XDG_CACHE_HOME = %q, want %q", os.Getenv("XDG_CACHE_HOME"), filepath.Join(root, "go-cache"))
	}
}

func TestEnsureLocalModelSkipsDownloadWhenCacheIsReady(t *testing.T) {
	cacheDir := t.TempDir()
	modelDir := filepath.Join(cacheDir, strings.ReplaceAll(DefaultModelID, "/", "_"))
	if err := os.MkdirAll(modelDir, 0o755); err != nil {
		t.Fatalf("mkdir model dir: %v", err)
	}
	for _, name := range []string{"model.onnx", "tokenizer.json", "config.json"} {
		if err := os.WriteFile(filepath.Join(modelDir, name), []byte("{}\n"), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}

	got, err := ensureLocalModel(context.Background(), DefaultConfig(), cacheDir)
	if err != nil {
		t.Fatalf("ensureLocalModel() error = %v", err)
	}
	if got != modelDir {
		t.Fatalf("model path = %q, want %q", got, modelDir)
	}
}

func TestEnsureLocalModelDownloadsMissingCacheWithoutHubSymlinks(t *testing.T) {
	cacheDir := t.TempDir()
	required := map[string]string{
		"/Snowflake/snowflake-arctic-embed-xs/resolve/main/onnx/model.onnx": "onnx",
		"/Snowflake/snowflake-arctic-embed-xs/resolve/main/tokenizer.json":  "{}",
		"/Snowflake/snowflake-arctic-embed-xs/resolve/main/config.json":     "{}",
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, ok := required[r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(body))
	}))
	defer server.Close()
	t.Setenv("HF_ENDPOINT", server.URL)

	modelDir, err := ensureLocalModel(context.Background(), DefaultConfig(), cacheDir)
	if err != nil {
		t.Fatalf("ensureLocalModel() error = %v", err)
	}
	for _, name := range []string{"model.onnx", "tokenizer.json", "config.json"} {
		if _, err := os.Stat(filepath.Join(modelDir, name)); err != nil {
			t.Fatalf("expected downloaded %s: %v", name, err)
		}
	}
	if _, err := os.Stat(filepath.Join(modelDir, "tokenizer_config.json")); !os.IsNotExist(err) {
		t.Fatalf("optional 404 file should not be created, stat error = %v", err)
	}
}

func TestParseLocalModelRefSupportsRevision(t *testing.T) {
	ref, err := parseLocalModelRef("owner/model:feature-branch")
	if err != nil {
		t.Fatalf("parseLocalModelRef() error = %v", err)
	}
	if ref.model != "owner/model" || ref.revision != "feature-branch" {
		t.Fatalf("ref = %#v", ref)
	}
}

type stubEmbedder struct{}

func (stubEmbedder) Embed(context.Context, []string) ([][]float32, error) {
	return nil, nil
}
