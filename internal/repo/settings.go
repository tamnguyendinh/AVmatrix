package repo

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

const DefaultMaxExecutionFlows = 700

type Settings struct {
	MaxExecutionFlows int `json:"maxExecutionFlows"`
}

func SettingsPath(repoPath string) string {
	return filepath.Join(StoragePath(repoPath), "settings.json")
}

func LoadSettings(repoPath string) (Settings, error) {
	raw, err := os.ReadFile(SettingsPath(repoPath))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			settings := NormalizeSettings(Settings{})
			if saveErr := SaveSettings(repoPath, settings); saveErr != nil {
				return Settings{}, saveErr
			}
			return settings, nil
		}
		return NormalizeSettings(Settings{}), nil
	}
	var settings Settings
	if err := json.Unmarshal(raw, &settings); err != nil {
		return NormalizeSettings(Settings{}), nil
	}
	return NormalizeSettings(settings), nil
}

func SaveSettings(repoPath string, settings Settings) error {
	if err := os.MkdirAll(StoragePath(repoPath), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(NormalizeSettings(settings), "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(SettingsPath(repoPath), append(raw, '\n'), 0o644)
}

func NormalizeSettings(settings Settings) Settings {
	if settings.MaxExecutionFlows <= 0 {
		settings.MaxExecutionFlows = DefaultMaxExecutionFlows
	}
	return settings
}
