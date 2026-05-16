package repo

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Store struct {
	HomeDir string
}

type RegistryNameCollisionError struct {
	RegistryName  string
	ExistingPath  string
	RequestedPath string
}

func (e RegistryNameCollisionError) Error() string {
	return fmt.Sprintf(
		"registry name %q is already used by %q; use a different name or allow duplicate names for %q",
		e.RegistryName,
		e.ExistingPath,
		e.RequestedPath,
	)
}

func NewStore(homeDir string) Store {
	return Store{HomeDir: homeDir}
}

func NewEnvStore() Store {
	return Store{HomeDir: GlobalDir()}
}

func (s Store) RegistryPath() string {
	return filepath.Join(s.homeDir(), "registry.json")
}

func (s Store) ReadRegistry() ([]RegistryEntry, error) {
	raw, err := os.ReadFile(s.RegistryPath())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []RegistryEntry{}, nil
		}
		return nil, err
	}

	var entries []RegistryEntry
	if err := json.Unmarshal(raw, &entries); err != nil {
		return []RegistryEntry{}, nil
	}

	changed := false
	for index := range entries {
		expected := StoragePath(entries[index].Path)
		if entries[index].StoragePath != expected {
			entries[index].StoragePath = expected
			changed = true
		}
	}
	if changed {
		if err := s.WriteRegistry(entries); err != nil {
			return nil, err
		}
	}

	return entries, nil
}

func (s Store) WriteRegistry(entries []RegistryEntry) error {
	if err := os.MkdirAll(s.homeDir(), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.RegistryPath(), append(raw, '\n'), 0o644)
}

func (s Store) Register(repoPath string, meta Meta, opts RegisterOptions) (string, error) {
	resolved := absClean(repoPath)
	paths := Paths(resolved)
	entries, err := s.ReadRegistry()
	if err != nil {
		return "", err
	}

	existingIndex := -1
	for index, entry := range entries {
		if SamePath(entry.Path, resolved) {
			existingIndex = index
			break
		}
	}

	name, preservedAlias := s.registryName(resolved, entries, existingIndex, opts)
	explicitName := opts.Name != "" || preservedAlias
	if explicitName && !opts.AllowDuplicateName {
		for index, entry := range entries {
			if index == existingIndex {
				continue
			}
			if strings.EqualFold(entry.Name, name) && !SamePath(entry.Path, resolved) {
				return "", RegistryNameCollisionError{
					RegistryName:  name,
					ExistingPath:  entry.Path,
					RequestedPath: resolved,
				}
			}
		}
	}

	entry := RegistryEntry{
		Name:        name,
		Path:        resolved,
		StoragePath: paths.StoragePath,
		IndexedAt:   meta.IndexedAt,
		LastCommit:  meta.LastCommit,
		Stats:       meta.Stats,
	}
	if existingIndex >= 0 {
		entries[existingIndex] = entry
	} else {
		entries = append(entries, entry)
	}

	if err := s.WriteRegistry(entries); err != nil {
		return "", err
	}
	return name, nil
}

func (s Store) Unregister(repoPath string) error {
	resolved := absClean(repoPath)
	entries, err := s.ReadRegistry()
	if err != nil {
		return err
	}
	filtered := entries[:0]
	for _, entry := range entries {
		if !SamePath(entry.Path, resolved) {
			filtered = append(filtered, entry)
		}
	}
	return s.WriteRegistry(filtered)
}

func (s Store) ListRegistered(validate bool) ([]RegistryEntry, error) {
	entries, err := s.ReadRegistry()
	if err != nil || !validate {
		return entries, err
	}

	valid := entries[:0]
	for _, entry := range entries {
		if _, err := os.Stat(filepath.Join(entry.StoragePath, "meta.json")); err == nil {
			valid = append(valid, entry)
		}
	}
	if len(valid) != len(entries) {
		if err := s.WriteRegistry(valid); err != nil {
			return nil, err
		}
	}
	return valid, nil
}

func (s Store) registryName(
	resolved string,
	entries []RegistryEntry,
	existingIndex int,
	opts RegisterOptions,
) (string, bool) {
	if opts.Name != "" {
		return opts.Name, false
	}

	inferred := opts.InferredName
	if inferred == "" {
		inferred = InferredName(resolved)
	}
	if existingIndex >= 0 && hasCustomAlias(entries[existingIndex], inferred) {
		return entries[existingIndex].Name, true
	}
	if inferred != "" {
		return inferred, false
	}
	return filepath.Base(resolved), false
}

func (s Store) homeDir() string {
	if s.HomeDir != "" {
		return s.HomeDir
	}
	return GlobalDir()
}

func hasCustomAlias(entry RegistryEntry, inferredName string) bool {
	if entry.Name == filepath.Base(absClean(entry.Path)) {
		return false
	}
	if inferredName != "" && entry.Name == inferredName {
		return false
	}
	return true
}
