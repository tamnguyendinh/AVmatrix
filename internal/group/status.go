package group

import (
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/tamnguyendinh/avmatrix-go/internal/repo"
)

func Status(homeDir string, store repo.Store, name string) (StatusResult, error) {
	config, err := Load(homeDir, name)
	if err != nil {
		return StatusResult{}, err
	}
	registry, err := ReadRegistry(homeDir, name)
	if err != nil {
		return StatusResult{}, err
	}
	entries, err := store.ListRegistered(false)
	if err != nil {
		return StatusResult{}, err
	}
	result := StatusResult{
		Group:        config.Name,
		MissingRepos: []string{},
		Repos:        map[string]RepoStatus{},
	}
	if registry != nil {
		result.LastSync = &registry.GeneratedAt
		result.MissingRepos = registry.MissingRepos
	}
	for groupRepoPath, registryName := range config.Repos {
		entry, err := repo.ResolveEntry(entries, registryName)
		if err != nil {
			result.Repos[groupRepoPath] = RepoStatus{Missing: true}
			continue
		}
		meta, err := repo.LoadMeta(entry.StoragePath)
		if err != nil || meta == nil {
			result.Repos[groupRepoPath] = RepoStatus{Missing: true}
			continue
		}
		indexStale, commitsBehind := indexStaleness(entry.Path, meta.LastCommit)
		contractsStale := true
		if registry != nil {
			snapshot, ok := registry.RepoSnapshots[groupRepoPath]
			contractsStale = !ok || snapshot.IndexedAt != meta.IndexedAt
		}
		result.Repos[groupRepoPath] = RepoStatus{
			IndexStale:     indexStale,
			ContractsStale: contractsStale,
			Missing:        false,
			CommitsBehind:  commitsBehind,
		}
	}
	return result, nil
}

func indexStaleness(repoPath string, lastCommit string) (bool, *int) {
	if strings.TrimSpace(lastCommit) == "" {
		behind := -1
		return true, &behind
	}
	output, err := exec.Command("git", "-C", filepath.Clean(repoPath), "rev-list", "--count", lastCommit+"..HEAD").Output()
	if err != nil {
		behind := 0
		return false, &behind
	}
	count, err := strconv.Atoi(strings.TrimSpace(string(output)))
	if err != nil {
		count = 0
	}
	return count > 0, &count
}
