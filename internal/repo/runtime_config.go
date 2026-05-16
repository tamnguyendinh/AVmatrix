package repo

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type WikiMode string

const (
	WikiModeOff   WikiMode = "off"
	WikiModeLocal WikiMode = "local"
)

type RuntimeConfig struct {
	WikiMode WikiMode `json:"wikiMode"`
}

func RuntimeConfigPath() string {
	return filepath.Join(GlobalDir(), "runtime.json")
}

func LoadRuntimeConfig() RuntimeConfig {
	raw, err := os.ReadFile(RuntimeConfigPath())
	if err != nil {
		return RuntimeConfig{WikiMode: WikiModeOff}
	}
	var config RuntimeConfig
	if err := json.Unmarshal(raw, &config); err != nil {
		return RuntimeConfig{WikiMode: WikiModeOff}
	}
	if config.WikiMode != WikiModeLocal {
		config.WikiMode = WikiModeOff
	}
	return config
}

func SaveRuntimeConfig(config RuntimeConfig) error {
	mode, ok := ParseWikiMode(string(config.WikiMode))
	if !ok {
		mode = WikiModeOff
	}
	if err := os.MkdirAll(GlobalDir(), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(RuntimeConfig{WikiMode: mode}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(RuntimeConfigPath(), append(raw, '\n'), 0o644)
}

func ParseWikiMode(value string) (WikiMode, bool) {
	switch value {
	case string(WikiModeOff):
		return WikiModeOff, true
	case string(WikiModeLocal):
		return WikiModeLocal, true
	default:
		return WikiModeOff, false
	}
}
