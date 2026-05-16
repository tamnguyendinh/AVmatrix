package session

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/tamnguyendinh/avmatrix-go/internal/repo"
)

type RepoResolver interface {
	Resolve(binding RepoBinding) (ResolvedRepo, error)
}

type StoreResolver struct {
	Store repo.Store
}

var remotePattern = regexp.MustCompile(`(?i)^[a-z][a-z0-9+.-]*://`)

func NewStoreResolver(store repo.Store) StoreResolver {
	return StoreResolver{Store: store}
}

func (r StoreResolver) Resolve(binding RepoBinding) (ResolvedRepo, error) {
	if binding.RepoName == "" && binding.RepoPath == "" {
		return ResolvedRepo{}, NewRuntimeError(
			ErrorInvalidRepoBinding,
			`Provide either "repoName" or "repoPath" for session binding`,
			400,
			nil,
		)
	}

	var fromName *ResolvedRepo
	if binding.RepoName != "" {
		resolved, err := r.resolveName(binding.RepoName)
		if err != nil {
			return ResolvedRepo{}, err
		}
		fromName = &resolved
	}

	var fromPath *ResolvedRepo
	if binding.RepoPath != "" {
		resolved, err := r.resolvePath(binding.RepoPath)
		if err != nil {
			return ResolvedRepo{}, err
		}
		fromPath = &resolved
	}

	if fromName != nil && fromPath != nil {
		if !samePath(fromName.RepoPath, fromPath.RepoPath) {
			return ResolvedRepo{}, NewRuntimeError(
				ErrorInvalidRepoBinding,
				`"repoName" and "repoPath" refer to different repositories`,
				400,
				map[string]any{"repoName": fromName.RepoName, "repoPath": fromPath.RepoPath},
			)
		}
		return *fromName, nil
	}
	if fromName != nil {
		return *fromName, nil
	}
	return *fromPath, nil
}

func (r StoreResolver) resolveName(repoName string) (ResolvedRepo, error) {
	entries, err := r.Store.ReadRegistry()
	if err != nil {
		return ResolvedRepo{}, err
	}

	var matches []repo.RegistryEntry
	for _, entry := range entries {
		if strings.EqualFold(entry.Name, repoName) {
			matches = append(matches, entry)
		}
	}
	if len(matches) == 0 {
		return ResolvedRepo{}, NewRuntimeError(
			ErrorRepoNotFound,
			fmt.Sprintf("Indexed repository %q was not found", repoName),
			404,
			map[string]any{"repoName": repoName},
		)
	}
	if len(matches) > 1 {
		return ResolvedRepo{}, NewRuntimeError(
			ErrorInvalidRepoBinding,
			fmt.Sprintf("Indexed repository %q is ambiguous", repoName),
			400,
			map[string]any{"repoName": repoName},
		)
	}

	entry := matches[0]
	realPath, err := resolveExistingDirectory(
		entry.Path,
		fmt.Sprintf("Indexed repository %q no longer exists at %q", entry.Name, entry.Path),
		fmt.Sprintf("Failed to inspect repository path %q for %q", entry.Path, entry.Name),
		map[string]any{"repoName": entry.Name, "repoPath": entry.Path},
	)
	if err != nil {
		return ResolvedRepo{}, err
	}

	indexed := repo.HasIndex(realPath)
	storagePath := ""
	if indexed {
		storagePath = entry.StoragePath
		if storagePath == "" {
			storagePath = repo.StoragePath(realPath)
		}
	}
	return ResolvedRepo{RepoName: entry.Name, RepoPath: realPath, Indexed: indexed, StoragePath: storagePath}, nil
}

func (r StoreResolver) resolvePath(repoPath string) (ResolvedRepo, error) {
	if isRemoteURL(repoPath) {
		return ResolvedRepo{}, NewRuntimeError(
			ErrorInvalidRepoPath,
			"Remote URLs are not allowed for local session runtime",
			400,
			nil,
		)
	}
	if isUNCPath(repoPath) {
		return ResolvedRepo{}, NewRuntimeError(
			ErrorInvalidRepoPath,
			"UNC and network-share paths are not allowed",
			400,
			nil,
		)
	}
	if !filepath.IsAbs(repoPath) {
		return ResolvedRepo{}, NewRuntimeError(
			ErrorInvalidRepoPath,
			`"repoPath" must be an absolute local path`,
			400,
			map[string]any{"repoPath": repoPath},
		)
	}

	realPath, err := resolveExistingDirectory(
		repoPath,
		fmt.Sprintf("Repository path %q does not exist", repoPath),
		fmt.Sprintf("Failed to inspect repository path %q", repoPath),
		map[string]any{"repoPath": repoPath},
	)
	if err != nil {
		return ResolvedRepo{}, err
	}

	entries, err := r.Store.ReadRegistry()
	if err != nil {
		return ResolvedRepo{}, err
	}
	name := filepath.Base(realPath)
	storagePath := ""
	for _, entry := range entries {
		if samePath(absClean(entry.Path), realPath) {
			name = entry.Name
			storagePath = entry.StoragePath
			break
		}
	}
	indexed := repo.HasIndex(realPath)
	if indexed && storagePath == "" {
		storagePath = repo.StoragePath(realPath)
	}
	return ResolvedRepo{RepoName: name, RepoPath: realPath, Indexed: indexed, StoragePath: storagePath}, nil
}

func resolveExistingDirectory(targetPath string, notFoundMessage string, invalidMessage string, details map[string]any) (string, error) {
	realPath, err := filepath.EvalSymlinks(targetPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", NewRuntimeError(ErrorRepoNotFound, notFoundMessage, 404, details)
		}
		return "", NewRuntimeError(ErrorInvalidRepoPath, invalidMessage, 400, details)
	}
	info, err := os.Stat(realPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", NewRuntimeError(ErrorRepoNotFound, notFoundMessage, 404, details)
		}
		return "", NewRuntimeError(ErrorInvalidRepoPath, invalidMessage, 400, details)
	}
	if !info.IsDir() {
		return "", NewRuntimeError(ErrorInvalidRepoPath, invalidMessage, 400, details)
	}
	return absClean(realPath), nil
}

func isRemoteURL(value string) bool {
	return remotePattern.MatchString(value) || strings.HasPrefix(strings.ToLower(value), "git@")
}

func isUNCPath(value string) bool {
	return strings.HasPrefix(value, `\\`) || strings.HasPrefix(value, "//")
}

func samePath(left string, right string) bool {
	left = filepath.Clean(left)
	right = filepath.Clean(right)
	if runtime.GOOS == "windows" {
		return strings.EqualFold(left, right)
	}
	return left == right
}

func absClean(path string) string {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return filepath.Clean(path)
	}
	return filepath.Clean(absolute)
}
