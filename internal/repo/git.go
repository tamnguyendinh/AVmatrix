package repo

import (
	"os/exec"
	"regexp"
	"strings"
)

var remoteNamePattern = regexp.MustCompile(`[/:]([^/:]+)$`)

func RemoteOriginURL(repoPath string) string {
	output, err := exec.Command("git", "-C", repoPath, "config", "--get", "remote.origin.url").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

func IsGitRepo(repoPath string) bool {
	output, err := exec.Command("git", "-C", repoPath, "rev-parse", "--is-inside-work-tree").Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(output)) == "true"
}

func GitRoot(repoPath string) string {
	output, err := exec.Command("git", "-C", repoPath, "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

func CurrentCommit(repoPath string) string {
	output, err := exec.Command("git", "-C", repoPath, "rev-parse", "HEAD").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

func ParseRepoNameFromURL(url string) string {
	trimmed := strings.TrimSpace(url)
	if trimmed == "" {
		return ""
	}
	withoutSuffix := regexp.MustCompile(`(?i)\.git/*$`).ReplaceAllString(trimmed, "")
	withoutSuffix = strings.TrimRight(withoutSuffix, "/")
	match := remoteNamePattern.FindStringSubmatch(withoutSuffix)
	if len(match) == 2 {
		return match[1]
	}
	return withoutSuffix
}

func InferredName(repoPath string) string {
	return ParseRepoNameFromURL(RemoteOriginURL(repoPath))
}
