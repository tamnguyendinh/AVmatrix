package repo

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseRepoNameFromURL(t *testing.T) {
	tests := map[string]string{
		"https://github.com/owner/project.git":       "project",
		"https://github.com/owner/project.GIT/":      "project",
		"git@github.com:owner/project.git":           "project",
		"git@gitlab.com:group/sub/project.git":       "project",
		"ssh://git@example.com/owner/project":        "project",
		"git://host.example/owner/project.git":       "project",
		"file:///tmp/project/":                       "project",
		"  https://github.com/owner/project.git  \n": "project",
		"project": "project",
		"":        "",
		"   ":     "",
	}
	for input, want := range tests {
		if got := ParseRepoNameFromURL(input); got != want {
			t.Fatalf("ParseRepoNameFromURL(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestGitHelpersReturnEmptyValuesOutsideGitRepo(t *testing.T) {
	dir := t.TempDir()
	if IsGitRepo(dir) {
		t.Fatal("plain temp directory was reported as git repo")
	}
	if got := CurrentCommit(dir); got != "" {
		t.Fatalf("CurrentCommit(non-git) = %q, want empty", got)
	}
	if got := GitRoot(dir); got != "" {
		t.Fatalf("GitRoot(non-git) = %q, want empty", got)
	}
	if got := RemoteOriginURL(dir); got != "" {
		t.Fatalf("RemoteOriginURL(non-git) = %q, want empty", got)
	}
	if got := InferredName(dir); got != "" {
		t.Fatalf("InferredName(non-git) = %q, want empty", got)
	}
}

func TestGitHelpersReadRealRepository(t *testing.T) {
	requireGit(t)

	dir := t.TempDir()
	runGit(t, dir, "init", "-q")
	runGit(t, dir, "config", "user.email", "test@example.com")
	runGit(t, dir, "config", "user.name", "Test")
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# repo\n"), 0o644); err != nil {
		t.Fatalf("write readme: %v", err)
	}
	runGit(t, dir, "add", "README.md")
	runGit(t, dir, "commit", "-q", "-m", "initial")
	runGit(t, dir, "remote", "add", "origin", "https://github.com/owner/lume_spark.git")

	if !IsGitRepo(filepath.Join(dir, "subdir", "..")) {
		t.Fatal("initialized repository was not reported as git repo")
	}
	gotRoot, err := filepath.EvalSymlinks(GitRoot(dir))
	if err != nil {
		t.Fatalf("resolve GitRoot(): %v", err)
	}
	wantRoot, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatalf("resolve temp dir: %v", err)
	}
	if !SamePath(gotRoot, wantRoot) {
		t.Fatalf("GitRoot() = %q, want %q", gotRoot, wantRoot)
	}
	if commit := CurrentCommit(dir); len(commit) < 7 || strings.TrimSpace(commit) != commit {
		t.Fatalf("CurrentCommit() = %q, want non-empty trimmed hash", commit)
	}
	if remote := RemoteOriginURL(dir); remote != "https://github.com/owner/lume_spark.git" {
		t.Fatalf("RemoteOriginURL() = %q", remote)
	}
	if inferred := InferredName(dir); inferred != "lume_spark" {
		t.Fatalf("InferredName() = %q, want lume_spark", inferred)
	}
}

func requireGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git executable not available")
	}
}

func runGit(t *testing.T, cwd string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = cwd
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, output)
	}
}
