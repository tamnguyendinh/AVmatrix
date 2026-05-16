package repo

import (
	"encoding/json"
	"os"
	"testing"
)

func TestRuntimeConfigDefaultsToOff(t *testing.T) {
	t.Setenv(HomeEnvName, t.TempDir())

	if got := RuntimeConfigPath(); got == "" {
		t.Fatal("RuntimeConfigPath() returned empty path")
	}
	if config := LoadRuntimeConfig(); config.WikiMode != WikiModeOff {
		t.Fatalf("LoadRuntimeConfig() = %#v, want off", config)
	}
}

func TestRuntimeConfigPersistsLocalMode(t *testing.T) {
	t.Setenv(HomeEnvName, t.TempDir())

	if err := SaveRuntimeConfig(RuntimeConfig{WikiMode: WikiModeLocal}); err != nil {
		t.Fatalf("SaveRuntimeConfig(local) error = %v", err)
	}
	if config := LoadRuntimeConfig(); config.WikiMode != WikiModeLocal {
		t.Fatalf("LoadRuntimeConfig() = %#v, want local", config)
	}

	raw, err := os.ReadFile(RuntimeConfigPath())
	if err != nil {
		t.Fatalf("read runtime config: %v", err)
	}
	var decoded RuntimeConfig
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("runtime config JSON invalid: %v", err)
	}
	if decoded.WikiMode != WikiModeLocal {
		t.Fatalf("runtime config JSON = %#v, want local", decoded)
	}
}

func TestRuntimeConfigNormalizesInvalidValuesToOff(t *testing.T) {
	t.Setenv(HomeEnvName, t.TempDir())

	if err := SaveRuntimeConfig(RuntimeConfig{WikiMode: WikiMode("remote")}); err != nil {
		t.Fatalf("SaveRuntimeConfig(invalid) error = %v", err)
	}
	if config := LoadRuntimeConfig(); config.WikiMode != WikiModeOff {
		t.Fatalf("LoadRuntimeConfig(invalid saved) = %#v, want off", config)
	}

	if err := os.WriteFile(RuntimeConfigPath(), []byte(`{"wikiMode":"remote"}`), 0o644); err != nil {
		t.Fatalf("write invalid runtime config: %v", err)
	}
	if config := LoadRuntimeConfig(); config.WikiMode != WikiModeOff {
		t.Fatalf("LoadRuntimeConfig(invalid raw) = %#v, want off", config)
	}
}
