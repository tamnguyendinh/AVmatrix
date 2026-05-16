package mcp

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tamnguyendinh/avmatrix-go/internal/repo"
)

func TestGitDiffForDetectChangesHandlesLargeOutput(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git executable is not available")
	}

	repoPath := t.TempDir()
	runMCPTestGit(t, repoPath, "init")
	runMCPTestGit(t, repoPath, "config", "user.email", "test@example.com")
	runMCPTestGit(t, repoPath, "config", "user.name", "Test User")

	sourcePath := filepath.Join(repoPath, "large.txt")
	if err := os.WriteFile(sourcePath, []byte("seed\n"), 0o644); err != nil {
		t.Fatalf("write seed file: %v", err)
	}
	runMCPTestGit(t, repoPath, "add", "large.txt")
	runMCPTestGit(t, repoPath, "commit", "-m", "initial")

	largeLine := strings.Repeat("x", 2*1024*1024)
	if err := os.WriteFile(sourcePath, []byte(largeLine+"\n"), 0o644); err != nil {
		t.Fatalf("write large diff file: %v", err)
	}

	output, err := gitDiffForDetectChanges(repo.RegistryEntry{Path: repoPath}, map[string]any{"scope": "unstaged"})
	if err != nil {
		t.Fatalf("gitDiffForDetectChanges() error = %v", err)
	}
	if len(output) < 1024*1024 {
		t.Fatalf("diff output length = %d, want > 1 MiB", len(output))
	}
	if !strings.Contains(output, largeLine[:256]) {
		t.Fatalf("diff output missing large changed line prefix")
	}
}
