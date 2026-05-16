package cli

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/tamnguyendinh/avmatrix-go/internal/repo"
)

func newStatusCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show index status for current repo",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			if !repo.IsGitRepo(cwd) {
				_, err := fmt.Fprintln(cmd.OutOrStdout(), "Not a git repository.")
				return err
			}

			indexed, err := repo.FindIndexed(cwd)
			if err != nil {
				return err
			}
			if indexed == nil {
				repoRoot := repo.GitRoot(cwd)
				if repoRoot == "" {
					repoRoot = cwd
				}
				if repo.HasLegacyKuzuIndex(repo.Paths(repoRoot).StoragePath) {
					_, err = fmt.Fprintln(cmd.OutOrStdout(), "Repository has a stale KuzuDB index from a previous version.")
					if err != nil {
						return err
					}
					_, err = fmt.Fprintln(cmd.OutOrStdout(), "Run: avmatrix analyze   (rebuilds the index with LadybugDB)")
					return err
				}
				_, err = fmt.Fprintln(cmd.OutOrStdout(), "Repository not indexed.")
				if err != nil {
					return err
				}
				_, err = fmt.Fprintln(cmd.OutOrStdout(), "Run: avmatrix analyze")
				return err
			}

			currentCommit := repo.CurrentCommit(indexed.RepoPath)
			status := "✅ up-to-date"
			if currentCommit != indexed.Meta.LastCommit {
				status = "⚠️ stale (re-run avmatrix analyze)"
			}

			_, err = fmt.Fprintf(
				cmd.OutOrStdout(),
				"Repository: %s\nIndexed: %s\nIndexed commit: %s\nCurrent commit: %s\nStatus: %s\n",
				indexed.RepoPath,
				formatIndexedAt(indexed.Meta.IndexedAt),
				shortCommit(indexed.Meta.LastCommit),
				shortCommit(currentCommit),
				status,
			)
			return err
		},
	}
}

func shortCommit(commit string) string {
	if len(commit) <= 7 {
		return commit
	}
	return commit[:7]
}

func formatIndexedAt(value string) string {
	if parsed, err := time.Parse(time.RFC3339Nano, value); err == nil {
		return parsed.Local().Format("1/2/2006, 3:04:05 PM")
	}
	return value
}
