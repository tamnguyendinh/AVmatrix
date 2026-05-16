package embeddings

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/tamnguyendinh/avmatrix-go/internal/lbugschema"
)

const (
	DefaultModelID              = "Snowflake/snowflake-arctic-embed-xs"
	DefaultBatchSize            = 16
	DefaultDimensions           = lbugschema.DefaultEmbeddingDims
	DefaultMaxSnippetLength     = 500
	DefaultChunkSize            = 1200
	DefaultOverlap              = 120
	DefaultMaxDescriptionLength = 150

	EnvEmbeddingURL    = "AVMATRIX_EMBEDDING_URL"
	EnvEmbeddingModel  = "AVMATRIX_EMBEDDING_MODEL"
	EnvEmbeddingAPIKey = "AVMATRIX_EMBEDDING_API_KEY"
	EnvEmbeddingDims   = "AVMATRIX_EMBEDDING_DIMS"
)

type Device string

const (
	DeviceAuto Device = "auto"
	DeviceDML  Device = "dml"
	DeviceCUDA Device = "cuda"
	DeviceCPU  Device = "cpu"
	DeviceWASM Device = "wasm"
)

type Config struct {
	ModelID              string
	BatchSize            int
	Dimensions           int
	Device               Device
	MaxSnippetLength     int
	ChunkSize            int
	Overlap              int
	MaxDescriptionLength int
}

type HTTPConfig struct {
	BaseURL    string
	Model      string
	APIKey     string
	Dimensions int
}

type EnvLookup func(string) (string, bool)

func DefaultConfig() Config {
	return Config{
		ModelID:              DefaultModelID,
		BatchSize:            DefaultBatchSize,
		Dimensions:           DefaultDimensions,
		Device:               DeviceAuto,
		MaxSnippetLength:     DefaultMaxSnippetLength,
		ChunkSize:            DefaultChunkSize,
		Overlap:              DefaultOverlap,
		MaxDescriptionLength: DefaultMaxDescriptionLength,
	}
}

func (c HTTPConfig) ExpectedDimensions() int {
	if c.Dimensions > 0 {
		return c.Dimensions
	}
	return DefaultDimensions
}

func ReadHTTPConfig(lookup EnvLookup) (*HTTPConfig, error) {
	if lookup == nil {
		lookup = os.LookupEnv
	}

	baseURL, okURL := lookup(EnvEmbeddingURL)
	model, okModel := lookup(EnvEmbeddingModel)
	if !okURL || !okModel || baseURL == "" || model == "" {
		return nil, nil
	}

	dimensions := 0
	if rawDims, ok := lookup(EnvEmbeddingDims); ok {
		parsed, err := strconv.Atoi(rawDims)
		if err != nil || parsed <= 0 {
			return nil, fmt.Errorf("%s must be a positive integer, got %q", EnvEmbeddingDims, rawDims)
		}
		dimensions = parsed
	}

	apiKey := "unused"
	if configuredAPIKey, ok := lookup(EnvEmbeddingAPIKey); ok {
		apiKey = configuredAPIKey
	}

	return &HTTPConfig{
		BaseURL:    strings.TrimRight(baseURL, "/"),
		Model:      model,
		APIKey:     apiKey,
		Dimensions: dimensions,
	}, nil
}

func IsHTTPMode(lookup EnvLookup) (bool, error) {
	config, err := ReadHTTPConfig(lookup)
	if err != nil {
		return false, err
	}
	return config != nil, nil
}

func HTTPDimensions(lookup EnvLookup) (int, bool, error) {
	config, err := ReadHTTPConfig(lookup)
	if err != nil {
		return 0, false, err
	}
	if config == nil || config.Dimensions == 0 {
		return 0, false, nil
	}
	return config.Dimensions, true, nil
}
