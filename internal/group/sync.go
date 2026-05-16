package group

import (
	"path/filepath"
	"time"

	"github.com/tamnguyendinh/avmatrix-go/internal/repo"
)

func Sync(homeDir string, store repo.Store, name string, options SyncOptions) (SyncResult, error) {
	config, err := Load(homeDir, name)
	if err != nil {
		return SyncResult{}, err
	}
	entries, err := store.ListRegistered(false)
	if err != nil {
		return SyncResult{}, err
	}

	missingRepos := make([]string, 0)
	repoSnapshots := make(map[string]RepoSnapshot)
	contracts := make([]StoredContract, 0)
	for _, groupRepoPath := range sortedGroupRepoPaths(config.Repos) {
		entry, err := repo.ResolveEntry(entries, config.Repos[groupRepoPath])
		if err != nil {
			missingRepos = append(missingRepos, groupRepoPath)
			continue
		}
		storagePath := storagePathForEntry(entry)
		meta, err := repo.LoadMeta(storagePath)
		if err != nil || meta == nil {
			missingRepos = append(missingRepos, groupRepoPath)
			continue
		}
		repoSnapshots[groupRepoPath] = RepoSnapshot{
			IndexedAt:  meta.IndexedAt,
			LastCommit: meta.LastCommit,
		}
		g, err := loadGroupGraphSnapshot(filepath.Join(storagePath, "graph.json"))
		if err != nil {
			missingRepos = append(missingRepos, groupRepoPath)
			continue
		}
		if config.Detect.HTTP {
			contracts = append(contracts, extractHTTPContracts(g, groupRepoPath)...)
		}
	}

	manifestContracts, manifestCrossLinks := manifestContractsAndLinks(config.Links)
	contracts = dedupeContracts(append(contracts, manifestContracts...))
	matched, unmatched := runExactMatch(contracts)
	if !options.ExactOnly {
		providers := providerIndex(contracts)
		wildcardMatched, remaining := runWildcardMatch(unmatched, providers)
		matched = append(matched, wildcardMatched...)
		unmatched = remaining
	}
	crossLinks := dedupeCrossLinks(append(manifestCrossLinks, matched...))
	result := SyncResult{
		Contracts:     contracts,
		CrossLinks:    crossLinks,
		Unmatched:     unmatched,
		MissingRepos:  missingRepos,
		RepoSnapshots: repoSnapshots,
	}
	registry := ContractRegistry{
		Version:       1,
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339Nano),
		RepoSnapshots: repoSnapshots,
		MissingRepos:  missingRepos,
		Contracts:     contracts,
		CrossLinks:    crossLinks,
	}
	if err := WriteRegistry(homeDir, name, registry); err != nil {
		return SyncResult{}, err
	}
	return result, nil
}
