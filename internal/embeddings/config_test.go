package embeddings

import (
	"testing"
)

func TestDefaultConfigMatchesCurrentEmbeddingContract(t *testing.T) {
	config := DefaultConfig()
	if config.ModelID != DefaultModelID {
		t.Fatalf("ModelID = %q, want %q", config.ModelID, DefaultModelID)
	}
	if config.BatchSize != 16 || config.Dimensions != 384 || config.Device != DeviceAuto {
		t.Fatalf("default config = %#v", config)
	}
	if config.MaxSnippetLength != 500 || config.ChunkSize != 1200 || config.Overlap != 120 || config.MaxDescriptionLength != 150 {
		t.Fatalf("default text config = %#v", config)
	}
}

func TestReadHTTPConfigRequiresURLAndModel(t *testing.T) {
	config, err := ReadHTTPConfig(mapLookup(map[string]string{
		EnvEmbeddingURL: "http://127.0.0.1:8080/v1",
	}))
	if err != nil {
		t.Fatalf("ReadHTTPConfig() error = %v", err)
	}
	if config != nil {
		t.Fatalf("ReadHTTPConfig() = %#v, want nil", config)
	}
}

func TestReadHTTPConfigParsesEnvSnapshot(t *testing.T) {
	config, err := ReadHTTPConfig(mapLookup(map[string]string{
		EnvEmbeddingURL:    "http://user:pass@127.0.0.1:8080/v1///",
		EnvEmbeddingModel:  "local-embedder",
		EnvEmbeddingAPIKey: "secret",
		EnvEmbeddingDims:   "768",
	}))
	if err != nil {
		t.Fatalf("ReadHTTPConfig() error = %v", err)
	}
	if config == nil {
		t.Fatal("ReadHTTPConfig() = nil")
	}
	if config.BaseURL != "http://user:pass@127.0.0.1:8080/v1" || config.Model != "local-embedder" || config.APIKey != "secret" || config.Dimensions != 768 {
		t.Fatalf("config = %#v", config)
	}
	if config.ExpectedDimensions() != 768 {
		t.Fatalf("ExpectedDimensions() = %d, want 768", config.ExpectedDimensions())
	}
}

func TestReadHTTPConfigDefaultsAPIKeyAndDimensions(t *testing.T) {
	config, err := ReadHTTPConfig(mapLookup(map[string]string{
		EnvEmbeddingURL:   "http://127.0.0.1:8080/v1",
		EnvEmbeddingModel: "local-embedder",
	}))
	if err != nil {
		t.Fatalf("ReadHTTPConfig() error = %v", err)
	}
	if config.APIKey != "unused" || config.Dimensions != 0 || config.ExpectedDimensions() != DefaultDimensions {
		t.Fatalf("config = %#v", config)
	}
}

func TestReadHTTPConfigRejectsInvalidDimensions(t *testing.T) {
	_, err := ReadHTTPConfig(mapLookup(map[string]string{
		EnvEmbeddingURL:   "http://127.0.0.1:8080/v1",
		EnvEmbeddingModel: "local-embedder",
		EnvEmbeddingDims:  "0",
	}))
	if err == nil {
		t.Fatal("ReadHTTPConfig() expected error")
	}
}

func TestIsHTTPModeAndHTTPDimensions(t *testing.T) {
	lookup := mapLookup(map[string]string{
		EnvEmbeddingURL:   "http://127.0.0.1:8080/v1",
		EnvEmbeddingModel: "local-embedder",
		EnvEmbeddingDims:  "1024",
	})
	httpMode, err := IsHTTPMode(lookup)
	if err != nil {
		t.Fatalf("IsHTTPMode() error = %v", err)
	}
	if !httpMode {
		t.Fatal("IsHTTPMode() = false, want true")
	}
	dimensions, ok, err := HTTPDimensions(lookup)
	if err != nil {
		t.Fatalf("HTTPDimensions() error = %v", err)
	}
	if !ok || dimensions != 1024 {
		t.Fatalf("HTTPDimensions() = %d, %v; want 1024, true", dimensions, ok)
	}
}

func mapLookup(values map[string]string) EnvLookup {
	return func(key string) (string, bool) {
		value, ok := values[key]
		return value, ok
	}
}
