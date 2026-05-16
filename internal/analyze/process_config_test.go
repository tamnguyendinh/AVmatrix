package analyze

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tamnguyendinh/avmatrix-go/internal/processes"
	"github.com/tamnguyendinh/avmatrix-go/internal/repo"
)

func TestResolveConfiguredMaxProcessesCapDefaultsEnvAndSettings(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		dir := t.TempDir()
		if got := resolveConfiguredMaxProcessesCap(dir, ""); got != repo.DefaultMaxExecutionFlows {
			t.Fatalf("cap = %d, want default %d", got, repo.DefaultMaxExecutionFlows)
		}
	})

	t.Run("env wins over settings", func(t *testing.T) {
		dir := t.TempDir()
		if err := repo.SaveSettings(dir, repo.Settings{MaxExecutionFlows: 900}); err != nil {
			t.Fatalf("SaveSettings failed: %v", err)
		}
		if got := resolveConfiguredMaxProcessesCap(dir, "650"); got != 650 {
			t.Fatalf("cap = %d, want env cap 650", got)
		}
	})

	t.Run("settings used when env unset", func(t *testing.T) {
		dir := t.TempDir()
		if err := repo.SaveSettings(dir, repo.Settings{MaxExecutionFlows: 550}); err != nil {
			t.Fatalf("SaveSettings failed: %v", err)
		}
		if got := resolveConfiguredMaxProcessesCap(dir, ""); got != 550 {
			t.Fatalf("cap = %d, want settings cap 550", got)
		}
	})

	t.Run("invalid configured values fall back", func(t *testing.T) {
		dir := t.TempDir()
		settingsPath := repo.SettingsPath(dir)
		if err := os.MkdirAll(filepath.Dir(settingsPath), 0o755); err != nil {
			t.Fatalf("mkdir settings: %v", err)
		}
		if err := os.WriteFile(settingsPath, []byte(`{"maxExecutionFlows":-10}`), 0o644); err != nil {
			t.Fatalf("write settings: %v", err)
		}
		if got := resolveConfiguredMaxProcessesCap(dir, "abc"); got != repo.DefaultMaxExecutionFlows {
			t.Fatalf("cap = %d, want default %d", got, repo.DefaultMaxExecutionFlows)
		}
	})
}

func TestResolveProcessConfigPreservesExplicitOptions(t *testing.T) {
	dir := t.TempDir()
	if got := resolveProcessConfig(dir, processes.Config{MaxProcesses: 12}); got.MaxProcesses != 12 || got.MaxProcessesCap != 0 {
		t.Fatalf("explicit MaxProcesses config = %#v, want MaxProcesses only", got)
	}
	if got := resolveProcessConfig(dir, processes.Config{MaxProcessesCap: 44}); got.MaxProcessesCap != 44 {
		t.Fatalf("explicit MaxProcessesCap config = %#v, want cap 44", got)
	}
}
