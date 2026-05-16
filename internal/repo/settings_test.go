package repo

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadSettingsCreatesRepoLocalDefaultsWhenMissing(t *testing.T) {
	repoPath := t.TempDir()

	settings, err := LoadSettings(repoPath)
	if err != nil {
		t.Fatalf("LoadSettings() error = %v", err)
	}
	if settings.MaxExecutionFlows != DefaultMaxExecutionFlows {
		t.Fatalf("LoadSettings() = %#v", settings)
	}

	raw, err := os.ReadFile(SettingsPath(repoPath))
	if err != nil {
		t.Fatalf("read settings: %v", err)
	}
	var decoded Settings
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("unmarshal settings: %v", err)
	}
	if decoded.MaxExecutionFlows != DefaultMaxExecutionFlows {
		t.Fatalf("settings file = %#v", decoded)
	}
	if filepath.Dir(SettingsPath(repoPath)) != StoragePath(repoPath) {
		t.Fatalf("SettingsPath dir = %q, want %q", filepath.Dir(SettingsPath(repoPath)), StoragePath(repoPath))
	}
}

func TestSaveSettingsPersistsNormalizedMaxExecutionFlows(t *testing.T) {
	repoPath := t.TempDir()

	if err := SaveSettings(repoPath, Settings{MaxExecutionFlows: 900}); err != nil {
		t.Fatalf("SaveSettings() error = %v", err)
	}
	settings, err := LoadSettings(repoPath)
	if err != nil {
		t.Fatalf("LoadSettings() error = %v", err)
	}
	if settings.MaxExecutionFlows != 900 {
		t.Fatalf("LoadSettings() = %#v", settings)
	}

	if err := SaveSettings(repoPath, Settings{MaxExecutionFlows: -1}); err != nil {
		t.Fatalf("SaveSettings(default) error = %v", err)
	}
	settings, err = LoadSettings(repoPath)
	if err != nil {
		t.Fatalf("LoadSettings(default) error = %v", err)
	}
	if settings.MaxExecutionFlows != DefaultMaxExecutionFlows {
		t.Fatalf("normalized settings = %#v", settings)
	}
}

func TestLoadSettingsFallsBackToDefaultForInvalidJSON(t *testing.T) {
	repoPath := t.TempDir()
	if err := os.MkdirAll(StoragePath(repoPath), 0o755); err != nil {
		t.Fatalf("mkdir storage: %v", err)
	}
	if err := os.WriteFile(SettingsPath(repoPath), []byte("{not-json"), 0o644); err != nil {
		t.Fatalf("write invalid settings: %v", err)
	}

	settings, err := LoadSettings(repoPath)
	if err != nil {
		t.Fatalf("LoadSettings(invalid) error = %v", err)
	}
	if settings.MaxExecutionFlows != DefaultMaxExecutionFlows {
		t.Fatalf("LoadSettings(invalid) = %#v", settings)
	}
}
