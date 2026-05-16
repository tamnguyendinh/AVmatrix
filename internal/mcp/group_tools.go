package mcp

import (
	"strings"

	groupcore "github.com/tamnguyendinh/avmatrix-go/internal/group"
)

func (s Server) groupListTool(args map[string]any) (map[string]any, error) {
	name := strings.TrimSpace(stringArg(args, "name", ""))
	if name == "" {
		groups, err := groupcore.List(s.store.HomeDir)
		if err != nil {
			return nil, err
		}
		return map[string]any{"groups": groups}, nil
	}
	config, err := groupcore.Load(s.store.HomeDir, name)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"name":        config.Name,
		"description": config.Description,
		"repos":       config.Repos,
		"links":       config.Links,
	}, nil
}

func (s Server) groupStatusTool(args map[string]any) (map[string]any, error) {
	name := strings.TrimSpace(stringArg(args, "name", ""))
	if name == "" {
		return map[string]any{"error": "name is required"}, nil
	}
	status, err := groupcore.Status(s.store.HomeDir, s.store, name)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"group":        status.Group,
		"lastSync":     status.LastSync,
		"missingRepos": status.MissingRepos,
		"repos":        status.Repos,
	}, nil
}

func (s Server) groupSyncTool(args map[string]any) (map[string]any, error) {
	name := strings.TrimSpace(stringArg(args, "name", ""))
	if name == "" {
		return map[string]any{"error": "name is required"}, nil
	}
	result, err := groupcore.Sync(s.store.HomeDir, s.store, name, groupcore.SyncOptions{
		AllowStale:     boolArg(args, "allowStale", false),
		Verbose:        boolArg(args, "verbose", false),
		ExactOnly:      boolArg(args, "exactOnly", false),
		SkipEmbeddings: boolArg(args, "skipEmbeddings", false),
	})
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"contracts":    len(result.Contracts),
		"crossLinks":   len(result.CrossLinks),
		"unmatched":    len(result.Unmatched),
		"missingRepos": result.MissingRepos,
	}, nil
}

func (s Server) groupContractsTool(args map[string]any) (map[string]any, error) {
	name := strings.TrimSpace(stringArg(args, "name", ""))
	if name == "" {
		return map[string]any{"error": "name is required"}, nil
	}
	result, err := groupcore.Contracts(s.store.HomeDir, name, groupcore.ContractsOptions{
		Type:          strings.TrimSpace(stringArg(args, "type", "")),
		Repo:          strings.TrimSpace(stringArg(args, "repo", "")),
		UnmatchedOnly: boolArg(args, "unmatchedOnly", false),
	})
	if err != nil {
		return map[string]any{"error": err.Error()}, nil
	}
	return map[string]any{
		"contracts":  result.Contracts,
		"crossLinks": result.CrossLinks,
	}, nil
}

func (s Server) groupQueryTool(args map[string]any) (map[string]any, error) {
	name := strings.TrimSpace(stringArg(args, "name", ""))
	query := strings.TrimSpace(stringArg(args, "query", ""))
	if name == "" || query == "" {
		return map[string]any{"error": "name and query are required"}, nil
	}
	result, err := groupcore.Query(
		s.store.HomeDir,
		s.store,
		name,
		query,
		intArg(args, "limit", 5, 1, 50),
		strings.TrimSpace(stringArg(args, "subgroup", "")),
	)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"group":    result.Group,
		"query":    result.Query,
		"results":  result.Results,
		"per_repo": result.PerRepo,
	}, nil
}
