package repo

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

func LoadMeta(storagePath string) (*Meta, error) {
	raw, err := os.ReadFile(filepath.Join(storagePath, "meta.json"))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	var meta Meta
	if err := json.Unmarshal(raw, &meta); err != nil {
		return nil, err
	}
	return &meta, nil
}

func SaveMeta(storagePath string, meta Meta) error {
	if err := os.MkdirAll(storagePath, 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(storagePath, "meta.json"), append(raw, '\n'), 0o644)
}

func HasIndex(repoPath string) bool {
	_, err := os.Stat(Paths(repoPath).MetaPath)
	return err == nil
}

func LoadIndexed(repoPath string) (*Indexed, error) {
	paths := Paths(repoPath)
	meta, err := LoadMeta(paths.StoragePath)
	if err != nil || meta == nil {
		return nil, err
	}
	return &Indexed{
		RepoPath:    absClean(repoPath),
		StoragePath: paths.StoragePath,
		LbugPath:    paths.LbugPath,
		MetaPath:    paths.MetaPath,
		Meta:        *meta,
	}, nil
}

func FindIndexed(startPath string) (*Indexed, error) {
	current := absClean(startPath)
	for {
		indexed, err := LoadIndexed(current)
		if err != nil || indexed != nil {
			return indexed, err
		}
		parent := filepath.Dir(current)
		if parent == current {
			return nil, nil
		}
		current = parent
	}
}

func HasLegacyKuzuIndex(storagePath string) bool {
	_, err := os.Stat(filepath.Join(storagePath, "kuzu"))
	return err == nil
}
