package group

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"

	"github.com/tamnguyendinh/avmatrix-go/internal/repo"
)

var groupNamePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]*$`)

func List(homeDir string) ([]string, error) {
	groupsDir := GroupsBaseDir(homeDir)
	entries, err := os.ReadDir(groupsDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []string{}, nil
		}
		return nil, err
	}
	names := make([]string, 0)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if _, err := os.Stat(filepath.Join(groupsDir, entry.Name(), "group.yaml")); err == nil {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)
	return names, nil
}

func Load(homeDir string, name string) (Config, error) {
	dir, err := Dir(homeDir, name)
	if err != nil {
		return Config{}, err
	}
	raw, err := os.ReadFile(filepath.Join(dir, "group.yaml"))
	if err != nil {
		return Config{}, err
	}
	return ParseConfig(string(raw))
}

func ReadRegistry(homeDir string, name string) (*ContractRegistry, error) {
	dir, err := Dir(homeDir, name)
	if err != nil {
		return nil, err
	}
	raw, err := os.ReadFile(filepath.Join(dir, "contracts.json"))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var registry ContractRegistry
	if err := json.Unmarshal(raw, &registry); err != nil {
		return nil, err
	}
	if registry.RepoSnapshots == nil {
		registry.RepoSnapshots = map[string]RepoSnapshot{}
	}
	if registry.MissingRepos == nil {
		registry.MissingRepos = []string{}
	}
	if registry.Contracts == nil {
		registry.Contracts = []StoredContract{}
	}
	if registry.CrossLinks == nil {
		registry.CrossLinks = []CrossLink{}
	}
	return &registry, nil
}

func WriteRegistry(homeDir string, name string, registry ContractRegistry) error {
	dir, err := Dir(homeDir, name)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(registry, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "contracts.json"), append(raw, '\n'), 0o644)
}

func GroupsBaseDir(homeDir string) string {
	return filepath.Join(home(homeDir), "groups")
}

func Dir(homeDir string, name string) (string, error) {
	if !groupNamePattern.MatchString(name) {
		return "", fmt.Errorf("Invalid group name %q. Names must start with a letter or digit and contain only [a-zA-Z0-9_-].", name)
	}
	return filepath.Join(GroupsBaseDir(homeDir), name), nil
}

func home(homeDir string) string {
	if homeDir != "" {
		return homeDir
	}
	return repo.GlobalDir()
}

func storagePathForEntry(entry repo.RegistryEntry) string {
	if entry.StoragePath != "" {
		return entry.StoragePath
	}
	return repo.StoragePath(entry.Path)
}
