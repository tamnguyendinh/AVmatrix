package repo

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var remoteURLPattern = regexp.MustCompile(`(?i)^[a-z][a-z0-9+.-]*://`)

func ResolveAnalyzePath(repoPath string) (string, error) {
	if isRemoteURL(repoPath) {
		return "", errors.New(`"path" must be a local filesystem path`)
	}
	if isUNCPath(repoPath) {
		return "", errors.New("UNC and network-share paths are not allowed")
	}
	if !filepath.IsAbs(repoPath) {
		return "", errors.New(`"path" must be an absolute path`)
	}

	realPath, err := filepath.EvalSymlinks(repoPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("repository path %q does not exist", repoPath)
		}
		return "", fmt.Errorf("failed to resolve repository path %q", repoPath)
	}

	info, err := os.Stat(realPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("repository path %q does not exist", repoPath)
		}
		return "", fmt.Errorf("failed to inspect repository path %q", repoPath)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("%q is not a directory", repoPath)
	}

	return filepath.Clean(realPath), nil
}

func isRemoteURL(value string) bool {
	return remoteURLPattern.MatchString(value) || strings.HasPrefix(strings.ToLower(value), "git@")
}

func isUNCPath(value string) bool {
	return strings.HasPrefix(value, `\\`) || strings.HasPrefix(value, "//")
}
