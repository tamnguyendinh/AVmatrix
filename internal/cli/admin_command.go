package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/tamnguyendinh/avmatrix-go/internal/repo"
)

func newCleanCommand() *cobra.Command {
	var force bool
	var all bool

	cmd := &cobra.Command{
		Use:   "clean",
		Short: "Delete AVmatrix index for current repo",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			store := repo.NewEnvStore()
			if all {
				return runCleanAll(cmd, store, force)
			}
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			indexed, err := repo.FindIndexed(cwd)
			if err != nil {
				return err
			}
			if indexed == nil {
				_, err = fmt.Fprintln(cmd.OutOrStdout(), "No indexed repository found in this directory.")
				return err
			}
			if !force {
				if _, err := fmt.Fprintf(cmd.OutOrStdout(), "This will delete the AVmatrix index for: %s\n", filepath.Base(indexed.RepoPath)); err != nil {
					return err
				}
				if _, err := fmt.Fprintf(cmd.OutOrStdout(), "   Path: %s\n", indexed.StoragePath); err != nil {
					return err
				}
				_, err = fmt.Fprintln(cmd.OutOrStdout(), "\nRun with --force to confirm deletion.")
				return err
			}
			if err := cleanStoragePreservingSettings(indexed.StoragePath); err != nil {
				return err
			}
			if err := store.Unregister(indexed.RepoPath); err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Deleted: %s\n", indexed.StoragePath)
			return err
		},
	}
	cmd.Flags().BoolVarP(&force, "force", "f", false, "skip confirmation prompt")
	cmd.Flags().BoolVar(&all, "all", false, "clean all indexed repos")
	return cmd
}

func runCleanAll(cmd *cobra.Command, store repo.Store, force bool) error {
	entries, err := store.ListRegistered(false)
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		_, err = fmt.Fprintln(cmd.OutOrStdout(), "No indexed repositories found.")
		return err
	}
	if !force {
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "This will delete AVmatrix indexes for %d repo(s):\n", len(entries)); err != nil {
			return err
		}
		for _, entry := range entries {
			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "  - %s (%s)\n", entry.Name, entry.Path); err != nil {
				return err
			}
		}
		_, err = fmt.Fprintln(cmd.OutOrStdout(), "\nRun with --force to confirm deletion.")
		return err
	}
	for _, entry := range entries {
		storagePath := entry.StoragePath
		if storagePath == "" {
			storagePath = repo.StoragePath(entry.Path)
		}
		if err := cleanStoragePreservingSettings(storagePath); err != nil {
			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Failed to delete %s: %v\n", entry.Name, err)
			continue
		}
		if err := store.Unregister(entry.Path); err != nil {
			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Failed to unregister %s: %v\n", entry.Name, err)
			continue
		}
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Deleted: %s (%s)\n", entry.Name, storagePath); err != nil {
			return err
		}
	}
	return nil
}

func cleanStoragePreservingSettings(storagePath string) error {
	settingsPath := filepath.Join(storagePath, "settings.json")
	preserved, err := os.ReadFile(settingsPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	hasSettings := err == nil
	if err := os.RemoveAll(storagePath); err != nil {
		return err
	}
	if hasSettings {
		if err := os.MkdirAll(storagePath, 0o755); err != nil {
			return err
		}
		return os.WriteFile(settingsPath, preserved, 0o644)
	}
	return nil
}

func newIndexCommand() *cobra.Command {
	var force bool
	var allowNonGit bool

	cmd := &cobra.Command{
		Use:   "index [path...]",
		Short: "Register an existing local index folder into the global registry",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			repoPath, err := resolveIndexPath(args, allowNonGit)
			if err != nil {
				return err
			}
			paths := repo.Paths(repoPath)
			if _, err := os.Stat(paths.StoragePath); err != nil {
				if errors.Is(err, os.ErrNotExist) {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "No .avmatrix/ folder found at: %s\n", paths.StoragePath)
					_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Run `avmatrix analyze` to build the index first.")
					return ExitError{Code: 1}
				}
				return err
			}
			if _, err := os.Stat(paths.LbugPath); err != nil {
				if errors.Is(err, os.ErrNotExist) {
					_, _ = fmt.Fprintln(cmd.OutOrStdout(), ".avmatrix/ folder exists but contains no LadybugDB index.")
					_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Run `avmatrix analyze` to build the index.")
					return ExitError{Code: 1}
				}
				return err
			}
			meta, err := repo.LoadMeta(paths.StoragePath)
			if err != nil {
				return err
			}
			if meta == nil {
				if !force {
					_, _ = fmt.Fprintln(cmd.OutOrStdout(), ".avmatrix/ exists but meta.json is missing.")
					_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Use --force to register anyway, or run `avmatrix analyze` to rebuild properly.")
					return ExitError{Code: 1}
				}
				meta = &repo.Meta{
					RepoPath:   repoPath,
					LastCommit: repo.CurrentCommit(repoPath),
					IndexedAt:  time.Now().UTC().Format(time.RFC3339),
				}
				if err := repo.SaveMeta(paths.StoragePath, *meta); err != nil {
					return err
				}
			}
			if meta.RepoPath == "" {
				meta.RepoPath = repoPath
			}
			name, err := repo.NewEnvStore().Register(repoPath, *meta, repo.RegisterOptions{})
			if err != nil {
				return err
			}
			if repo.IsGitRepo(repoPath) {
				_ = ensureGitignoreEntry(repoPath, repo.StorageDirName+"/")
			}
			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Repository registered: %s\n", name); err != nil {
				return err
			}
			if stats := formatIndexStats(meta.Stats); stats != "" {
				if _, err := fmt.Fprintln(cmd.OutOrStdout(), stats); err != nil {
					return err
				}
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), repoPath)
			return err
		},
	}
	cmd.Flags().BoolVarP(&force, "force", "f", false, "register even if meta.json is missing")
	cmd.Flags().BoolVar(&allowNonGit, "allow-non-git", false, "allow registering folders that are not Git repositories")
	return cmd
}

func resolveIndexPath(args []string, allowNonGit bool) (string, error) {
	var input string
	if len(args) > 0 {
		input = strings.Join(args, " ")
		if len(args) > 1 {
			if _, err := os.Stat(input); err != nil {
				return "", fmt.Errorf("the index command accepts a single path only; if your path contains spaces, wrap it in quotes")
			}
		}
	} else {
		cwd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		gitRoot := repo.GitRoot(cwd)
		if gitRoot == "" {
			return "", fmt.Errorf("not inside a git repository, try to run git init")
		}
		input = gitRoot
	}
	if !filepath.IsAbs(input) {
		absolute, err := filepath.Abs(input)
		if err != nil {
			return "", err
		}
		input = absolute
	}
	resolved, err := repo.ResolveAnalyzePath(input)
	if err != nil {
		return "", err
	}
	if !allowNonGit && !repo.IsGitRepo(resolved) {
		return "", fmt.Errorf("not a git repository: %s; use --allow-non-git to register an existing .avmatrix index anyway", resolved)
	}
	return resolved, nil
}

func ensureGitignoreEntry(repoPath string, entry string) error {
	gitignorePath := filepath.Join(repoPath, ".gitignore")
	raw, err := os.ReadFile(gitignorePath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if strings.Contains("\n"+string(raw)+"\n", "\n"+entry+"\n") {
		return nil
	}
	file, err := os.OpenFile(gitignorePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()
	if len(raw) > 0 && !strings.HasSuffix(string(raw), "\n") {
		if _, err := file.WriteString("\n"); err != nil {
			return err
		}
	}
	_, err = file.WriteString(entry + "\n")
	return err
}

func formatIndexStats(stats *repo.Stats) string {
	if stats == nil {
		return ""
	}
	parts := make([]string, 0, 4)
	if stats.Nodes != nil {
		parts = append(parts, fmt.Sprintf("%d nodes", *stats.Nodes))
	}
	if stats.Edges != nil {
		parts = append(parts, fmt.Sprintf("%d edges", *stats.Edges))
	}
	if stats.Communities != nil {
		parts = append(parts, fmt.Sprintf("%d clusters", *stats.Communities))
	}
	if stats.Processes != nil {
		parts = append(parts, fmt.Sprintf("%d flows", *stats.Processes))
	}
	return strings.Join(parts, " | ")
}
