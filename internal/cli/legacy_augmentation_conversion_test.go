package cli

import (
	"strings"
	"testing"

	"github.com/tamnguyendinh/avmatrix-go/internal/repo"
)

func TestAugmentLegacyConversionUsesLocalGraphContext(t *testing.T) {
	home := t.TempDir()
	t.Setenv(repo.HomeEnvName, home)
	repoPath := t.TempDir()
	store := repo.NewStore(home)
	registerDirectToolCommandRepo(t, store, repoPath, "fixture")
	writeDirectToolCommandGraph(t, repoPath)

	withWorkingDir(t, repoPath, func() {
		out, errOut, err := executeForTest(t, "augment", "MainFlow")
		if err != nil {
			t.Fatalf("augment returned error: %v\nstdout:\n%s\nstderr:\n%s", err, out, errOut)
		}
		if out != "" {
			t.Fatalf("augment wrote stdout: %q", out)
		}
		for _, want := range []string{`AVmatrix graph context for "MainFlow"`, `"Label": "MainFlow"`, `"Function:main"`} {
			if !strings.Contains(errOut, want) {
				t.Fatalf("augment stderr missing %q:\n%s", want, errOut)
			}
		}
	})
}

func TestAugmentLegacyConversionReturnsEmptyForShortAndWhitespacePatterns(t *testing.T) {
	home := t.TempDir()
	t.Setenv(repo.HomeEnvName, home)
	repoPath := t.TempDir()
	store := repo.NewStore(home)
	registerDirectToolCommandRepo(t, store, repoPath, "fixture")
	writeDirectToolCommandGraph(t, repoPath)

	withWorkingDir(t, repoPath, func() {
		for _, pattern := range []string{"   ", "go"} {
			t.Run(pattern, func(t *testing.T) {
				out, errOut, err := executeForTest(t, "augment", pattern)
				if err != nil {
					t.Fatalf("augment %q returned error: %v\nstdout:\n%s\nstderr:\n%s", pattern, err, out, errOut)
				}
				if out != "" || errOut != "" {
					t.Fatalf("augment %q wrote output: stdout=%q stderr=%q", pattern, out, errOut)
				}
			})
		}
	})
}

func TestAugmentLegacyConversionHandlesOddPatternsWithoutError(t *testing.T) {
	home := t.TempDir()
	t.Setenv(repo.HomeEnvName, home)
	repoPath := t.TempDir()
	store := repo.NewStore(home)
	registerDirectToolCommandRepo(t, store, repoPath, "fixture")
	writeDirectToolCommandGraph(t, repoPath)

	withWorkingDir(t, repoPath, func() {
		for _, pattern := range []string{"nonexistent_xyz", "func()", strings.Repeat("a", 500), "日本語テスト"} {
			t.Run(pattern, func(t *testing.T) {
				out, _, err := executeForTest(t, "augment", pattern)
				if err != nil {
					t.Fatalf("augment %q returned error: %v\nstdout:\n%s", pattern, err, out)
				}
				if out != "" {
					t.Fatalf("augment %q wrote stdout: %q", pattern, out)
				}
			})
		}
	})
}
