package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	groupcore "github.com/tamnguyendinh/avmatrix-go/internal/group"
	"github.com/tamnguyendinh/avmatrix-go/internal/repo"
)

func newGroupCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "group",
		Short: "Manage repository groups for cross-index impact analysis",
	}
	cmd.AddCommand(
		newGroupCreateCommand(),
		newGroupAddCommand(),
		newGroupRemoveCommand(),
		newGroupListCommand(),
		newGroupStatusCommand(),
		newGroupSyncCommand(),
		newGroupQueryCommand(),
		newGroupContractsCommand(),
	)
	return cmd
}

func newGroupCreateCommand() *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a new group with template group.yaml",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			dir, err := groupcore.Dir(repo.GlobalDir(), name)
			if err != nil {
				return err
			}
			if _, err := os.Stat(dir); err == nil {
				if !force {
					return fmt.Errorf("group %q already exists; use --force to overwrite", name)
				}
				if err := os.RemoveAll(dir); err != nil {
					return err
				}
			} else if !os.IsNotExist(err) {
				return err
			}
			config := groupcore.Config{
				Version:     1,
				Name:        name,
				Description: "",
				Repos:       map[string]string{},
				Links:       []groupcore.ManifestLink{},
			}
			if err := writeGroupConfigFile(dir, config); err != nil {
				return err
			}
			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Created group %q at %s\n", name, dir); err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Edit group.yaml to add repos, then run: avmatrix group sync %s\n", name)
			return err
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "overwrite existing group")
	return cmd
}

func newGroupAddCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "add <group> <groupPath> <registryName>",
		Short: "Add a repo to a group",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			groupName, groupPath, registryName := args[0], normalizeGroupRepoPath(args[1]), args[2]
			config, dir, err := loadGroupConfigAndDir(groupName)
			if err != nil {
				return err
			}
			if config.Repos == nil {
				config.Repos = map[string]string{}
			}
			config.Repos[groupPath] = registryName
			if err := writeGroupConfigFile(dir, config); err != nil {
				return err
			}
			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Added %s as %q to group %q\n", registryName, groupPath, groupName); err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Run: avmatrix group sync %s\n", groupName)
			return err
		},
	}
}

func newGroupRemoveCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "remove <group> <path>",
		Short: "Remove a repo from a group",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			groupName, groupPath := args[0], normalizeGroupRepoPath(args[1])
			config, dir, err := loadGroupConfigAndDir(groupName)
			if err != nil {
				return err
			}
			if _, ok := config.Repos[groupPath]; !ok {
				return fmt.Errorf("repo path %q not found in group %q", groupPath, groupName)
			}
			delete(config.Repos, groupPath)
			if err := writeGroupConfigFile(dir, config); err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Removed %q from group %q\n", groupPath, groupName)
			return err
		},
	}
}

func newGroupListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list [name]",
		Short: "List all groups or details of one",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				groups, err := groupcore.List(repo.GlobalDir())
				if err != nil {
					return err
				}
				if len(groups) == 0 {
					_, err = fmt.Fprintln(cmd.OutOrStdout(), "No groups configured. Create one with: avmatrix group create <name>")
					return err
				}
				if _, err := fmt.Fprintln(cmd.OutOrStdout(), "Groups:"); err != nil {
					return err
				}
				for _, groupName := range groups {
					if _, err := fmt.Fprintf(cmd.OutOrStdout(), "  %s\n", groupName); err != nil {
						return err
					}
				}
				return nil
			}
			config, err := groupcore.Load(repo.GlobalDir(), args[0])
			if err != nil {
				return err
			}
			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Group: %s\n", config.Name); err != nil {
				return err
			}
			if config.Description != "" {
				if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Description: %s\n", config.Description); err != nil {
					return err
				}
			}
			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "\nRepos (%d):\n", len(config.Repos)); err != nil {
				return err
			}
			for _, groupPath := range sortedStringKeys(config.Repos) {
				if _, err := fmt.Fprintf(cmd.OutOrStdout(), "  %s -> %s\n", groupPath, config.Repos[groupPath]); err != nil {
					return err
				}
			}
			if len(config.Links) > 0 {
				if _, err := fmt.Fprintf(cmd.OutOrStdout(), "\nManifest links (%d):\n", len(config.Links)); err != nil {
					return err
				}
				for _, link := range config.Links {
					if _, err := fmt.Fprintf(cmd.OutOrStdout(), "  %s -> %s [%s: %s]\n", link.From, link.To, link.Type, link.Contract); err != nil {
						return err
					}
				}
			}
			return nil
		},
	}
}

func newGroupStatusCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "status <name>",
		Short: "Check staleness of group and repos",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			status, err := groupcore.Status(repo.GlobalDir(), repo.NewEnvStore(), args[0])
			if err != nil {
				return err
			}
			lastSync := "never synced"
			if status.LastSync != nil {
				lastSync = *status.LastSync
			}
			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Group: %s (last sync: %s)\n\n", status.Group, lastSync); err != nil {
				return err
			}
			if _, err := fmt.Fprintln(cmd.OutOrStdout(), "Repo index / contracts staleness:"); err != nil {
				return err
			}
			for _, groupPath := range sortedStatusKeys(status.Repos) {
				row := status.Repos[groupPath]
				if row.Missing {
					if _, err := fmt.Fprintf(cmd.OutOrStdout(), "  %s MISSING\n", groupPath); err != nil {
						return err
					}
					continue
				}
				behind := ""
				if row.CommitsBehind != nil {
					behind = fmt.Sprintf(" (%d commits behind)", *row.CommitsBehind)
				}
				if _, err := fmt.Fprintf(cmd.OutOrStdout(), "  %s indexStale=%t%s contractsStale=%t\n", groupPath, row.IndexStale, behind, row.ContractsStale); err != nil {
					return err
				}
			}
			if len(status.MissingRepos) > 0 {
				_, err = fmt.Fprintf(cmd.OutOrStdout(), "\nLast sync missing repos: %s\n", strings.Join(status.MissingRepos, ", "))
				return err
			}
			return nil
		},
	}
}

func newGroupSyncCommand() *cobra.Command {
	var skipEmbeddings bool
	var exactOnly bool
	var allowStale bool
	var verbose bool
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "sync <name>",
		Short: "Sync Contract Registry from indexed group repos",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := groupcore.Sync(repo.GlobalDir(), repo.NewEnvStore(), args[0], groupcore.SyncOptions{
				AllowStale:     allowStale,
				Verbose:        verbose,
				ExactOnly:      exactOnly,
				SkipEmbeddings: skipEmbeddings,
			})
			if err != nil {
				return err
			}
			if jsonOutput {
				return printJSON(cmd, result)
			}
			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Matching cascade:\n"); err != nil {
				return err
			}
			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "  exact: %d cross-links\n", len(result.CrossLinks)); err != nil {
				return err
			}
			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "  unmatched: %d contracts\n", len(result.Unmatched)); err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Wrote contracts.json (%d contracts, %d cross-links)\n", len(result.Contracts), len(result.CrossLinks))
			return err
		},
	}
	cmd.Flags().BoolVar(&skipEmbeddings, "skip-embeddings", false, "exact and BM25 only")
	cmd.Flags().BoolVar(&exactOnly, "exact-only", false, "exact match only")
	cmd.Flags().BoolVar(&allowStale, "allow-stale", false, "skip stale index warnings")
	cmd.Flags().BoolVar(&verbose, "verbose", false, "show each cross-link detail")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "print JSON output")
	return cmd
}

func newGroupQueryCommand() *cobra.Command {
	var subgroup string
	var limit string
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "query <name> <query>",
		Short: "Search execution flows across all repos in a group",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			parsedLimit := 5
			if limit != "" {
				value, err := parsePositiveIntFlag("limit", limit)
				if err != nil {
					return err
				}
				parsedLimit = value
			}
			result, err := groupcore.Query(repo.GlobalDir(), repo.NewEnvStore(), args[0], args[1], parsedLimit, subgroup)
			if err != nil {
				return err
			}
			if jsonOutput {
				return printJSON(cmd, result)
			}
			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Results (top %d):\n", len(result.Results)); err != nil {
				return err
			}
			for _, item := range result.Results {
				label := firstNonEmptyStringForGroup(anyString(item["summary"]), anyString(item["heuristicLabel"]), anyString(item["name"]), "unnamed")
				repoName := anyString(item["_repo"])
				score, _ := item["_rrf_score"].(float64)
				if _, err := fmt.Fprintf(cmd.OutOrStdout(), "  [%s] %s (rrf: %.4f)\n", repoName, label, score); err != nil {
					return err
				}
			}
			if len(result.Results) == 0 {
				_, err = fmt.Fprintln(cmd.OutOrStdout(), "  No matching execution flows found.")
				return err
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&subgroup, "subgroup", "", "limit search scope")
	cmd.Flags().StringVar(&limit, "limit", "5", "max merged results")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "print JSON output")
	return cmd
}

func newGroupContractsCommand() *cobra.Command {
	var contractType string
	var repoName string
	var unmatched bool
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "contracts <name>",
		Short: "Inspect Contract Registry",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := groupcore.Contracts(repo.GlobalDir(), args[0], groupcore.ContractsOptions{
				Type:          contractType,
				Repo:          repoName,
				UnmatchedOnly: unmatched,
			})
			if err != nil {
				return err
			}
			if jsonOutput {
				return printJSON(cmd, result)
			}
			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Contracts (%d):\n", len(result.Contracts)); err != nil {
				return err
			}
			for _, contract := range result.Contracts {
				if _, err := fmt.Fprintf(cmd.OutOrStdout(), "  [%s] %s  (%s)  %s\n", contract.Role, contract.ContractID, contract.Repo, contract.SymbolRef.Name); err != nil {
					return err
				}
			}
			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "\nCross-links (%d):\n", len(result.CrossLinks)); err != nil {
				return err
			}
			for _, link := range result.CrossLinks {
				if _, err := fmt.Fprintf(cmd.OutOrStdout(), "  %s -> %s  [%s, conf=%.2f]  %s\n", link.From.Repo, link.To.Repo, link.MatchType, link.Confidence, link.ContractID); err != nil {
					return err
				}
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&contractType, "type", "", "filter by contract type")
	cmd.Flags().StringVar(&repoName, "repo", "", "filter by repo")
	cmd.Flags().BoolVar(&unmatched, "unmatched", false, "show only unmatched contracts")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "print JSON output")
	return cmd
}

func loadGroupConfigAndDir(name string) (groupcore.Config, string, error) {
	dir, err := groupcore.Dir(repo.GlobalDir(), name)
	if err != nil {
		return groupcore.Config{}, "", err
	}
	config, err := groupcore.Load(repo.GlobalDir(), name)
	if err != nil {
		return groupcore.Config{}, "", err
	}
	return config, dir, nil
}

func writeGroupConfigFile(dir string, config groupcore.Config) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	raw := renderGroupConfig(config)
	return os.WriteFile(filepath.Join(dir, "group.yaml"), []byte(raw), 0o644)
}

func renderGroupConfig(config groupcore.Config) string {
	var builder strings.Builder
	builder.WriteString("version: 1\n")
	builder.WriteString("name: " + yamlScalar(config.Name) + "\n")
	if config.Description != "" {
		builder.WriteString("description: " + yamlScalar(config.Description) + "\n")
	}
	builder.WriteString("\nrepos:\n")
	for _, groupPath := range sortedStringKeys(config.Repos) {
		builder.WriteString("  " + yamlScalar(groupPath) + ": " + yamlScalar(config.Repos[groupPath]) + "\n")
	}
	if len(config.Links) == 0 {
		builder.WriteString("\nlinks: []\n")
	} else {
		builder.WriteString("\nlinks:\n")
		for _, link := range config.Links {
			builder.WriteString("  - from: " + yamlScalar(link.From) + "\n")
			builder.WriteString("    to: " + yamlScalar(link.To) + "\n")
			builder.WriteString("    type: " + yamlScalar(link.Type) + "\n")
			builder.WriteString("    contract: " + yamlScalar(link.Contract) + "\n")
			if link.Role != "" {
				builder.WriteString("    role: " + yamlScalar(link.Role) + "\n")
			}
		}
	}
	return builder.String()
}

func yamlScalar(value string) string {
	if value == "" {
		return `""`
	}
	if strings.ContainsAny(value, ":#'\"\n\r\t") || strings.HasPrefix(value, " ") || strings.HasSuffix(value, " ") {
		escaped := strings.ReplaceAll(value, `\`, `\\`)
		escaped = strings.ReplaceAll(escaped, `"`, `\"`)
		return `"` + escaped + `"`
	}
	return value
}

func normalizeGroupRepoPath(value string) string {
	return strings.Trim(strings.ReplaceAll(value, "\\", "/"), "/")
}

func sortedStringKeys(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func sortedStatusKeys(values map[string]groupcore.RepoStatus) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func printJSON(cmd *cobra.Command, value any) error {
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(cmd.OutOrStdout(), string(raw))
	return err
}

func anyString(value any) string {
	text, _ := value.(string)
	return text
}

func firstNonEmptyStringForGroup(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
