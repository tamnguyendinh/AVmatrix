package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	mcpserver "github.com/tamnguyendinh/avmatrix-go/internal/mcp"
	"github.com/tamnguyendinh/avmatrix-go/internal/repo"
)

func newAugmentCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "augment <pattern>",
		Short: "Augment a search pattern with knowledge graph context",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pattern := strings.TrimSpace(args[0])
			if len(pattern) < 3 {
				return nil
			}
			cwd, _ := os.Getwd()
			text, err := callLocalMCPTool("query", map[string]any{
				"query":       pattern,
				"limit":       3,
				"max_symbols": 5,
				"repo":        emptyToNil(cwd),
			})
			if err != nil {
				return nil
			}
			if strings.TrimSpace(text) == "" {
				return nil
			}
			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "AVmatrix graph context for %q:\n%s\n", pattern, text)
			return nil
		},
	}
}

func newQueryCommand() *cobra.Command {
	var repoName string
	var taskContext string
	var goal string
	var limit string
	var includeContent bool

	cmd := &cobra.Command{
		Use:   "query <search_query>",
		Short: "Search the knowledge graph for execution flows related to a concept",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			toolArgs := map[string]any{
				"query":           args[0],
				"repo":            emptyToNil(repoName),
				"task_context":    emptyToNil(taskContext),
				"goal":            emptyToNil(goal),
				"include_content": includeContent,
			}
			if limit != "" {
				parsed, err := parsePositiveIntFlag("limit", limit)
				if err != nil {
					return err
				}
				toolArgs["limit"] = parsed
			}
			return printLocalMCPTool(cmd, "query", toolArgs)
		},
	}
	cmd.Flags().StringVarP(&repoName, "repo", "r", "", "target repository (omit if only one indexed)")
	cmd.Flags().StringVarP(&taskContext, "context", "c", "", "task context to improve ranking")
	cmd.Flags().StringVarP(&goal, "goal", "g", "", "what you want to find")
	cmd.Flags().StringVarP(&limit, "limit", "l", "", "max processes to return")
	cmd.Flags().BoolVar(&includeContent, "content", false, "include full symbol source code")
	return cmd
}

func newContextCommand() *cobra.Command {
	var repoName string
	var uid string
	var filePath string
	var includeContent bool

	cmd := &cobra.Command{
		Use:   "context [name]",
		Short: "360-degree view of a code symbol: callers, callees, processes",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := ""
			if len(args) > 0 {
				name = args[0]
			}
			if strings.TrimSpace(name) == "" && strings.TrimSpace(uid) == "" {
				return fmt.Errorf("usage: avmatrix context <symbol_name> [--uid <uid>] [--file <path>]")
			}
			return printLocalMCPTool(cmd, "context", map[string]any{
				"name":            emptyToNil(name),
				"uid":             emptyToNil(uid),
				"file_path":       emptyToNil(filePath),
				"include_content": includeContent,
				"repo":            emptyToNil(repoName),
			})
		},
	}
	cmd.Flags().StringVarP(&repoName, "repo", "r", "", "target repository")
	cmd.Flags().StringVarP(&uid, "uid", "u", "", "direct symbol UID")
	cmd.Flags().StringVarP(&filePath, "file", "f", "", "file path to disambiguate common names")
	cmd.Flags().BoolVar(&includeContent, "content", false, "include full symbol source code")
	return cmd
}

func newImpactCommand() *cobra.Command {
	var direction string
	var repoName string
	var uid string
	var depth string
	var includeTests bool

	cmd := &cobra.Command{
		Use:   "impact [target]",
		Short: "Blast radius analysis: what breaks if you change a symbol",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			target := ""
			if len(args) > 0 {
				target = args[0]
			}
			if strings.TrimSpace(target) == "" && strings.TrimSpace(uid) == "" {
				return fmt.Errorf("usage: avmatrix impact [symbol_name] [--uid <uid>] [--direction upstream|downstream]")
			}
			if direction != "upstream" && direction != "downstream" {
				return fmt.Errorf("direction must be upstream or downstream")
			}
			toolArgs := map[string]any{
				"target":       emptyToNil(target),
				"target_uid":   emptyToNil(uid),
				"direction":    direction,
				"includeTests": includeTests,
				"repo":         emptyToNil(repoName),
			}
			if depth != "" {
				parsed, err := parsePositiveIntFlag("depth", depth)
				if err != nil {
					return err
				}
				toolArgs["maxDepth"] = parsed
			}
			return printLocalMCPTool(cmd, "impact", toolArgs)
		},
	}
	cmd.Flags().StringVarP(&direction, "direction", "d", "upstream", "upstream or downstream")
	cmd.Flags().StringVarP(&repoName, "repo", "r", "", "target repository")
	cmd.Flags().StringVarP(&uid, "uid", "u", "", "direct symbol UID")
	cmd.Flags().StringVar(&depth, "depth", "", "max relationship depth")
	cmd.Flags().BoolVar(&includeTests, "include-tests", false, "include test files in results")
	return cmd
}

func newCypherCommand() *cobra.Command {
	var repoName string
	cmd := &cobra.Command{
		Use:   "cypher <query>",
		Short: "Execute raw Cypher query against the knowledge graph",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return printLocalMCPTool(cmd, "cypher", map[string]any{
				"query": args[0],
				"repo":  emptyToNil(repoName),
			})
		},
	}
	cmd.Flags().StringVarP(&repoName, "repo", "r", "", "target repository")
	return cmd
}

func newDetectChangesCommand() *cobra.Command {
	var scope string
	var baseRef string
	var repoName string

	cmd := &cobra.Command{
		Use:   "detect-changes",
		Short: "Analyze uncommitted git changes and find affected execution flows",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return printLocalMCPTool(cmd, "detect_changes", map[string]any{
				"scope":    scope,
				"base_ref": emptyToNil(baseRef),
				"repo":     emptyToNil(repoName),
			})
		},
	}
	cmd.Flags().StringVarP(&scope, "scope", "s", "unstaged", "unstaged, staged, all, or compare")
	cmd.Flags().StringVar(&baseRef, "base-ref", "", "branch/commit for compare scope")
	cmd.Flags().StringVarP(&repoName, "repo", "r", "", "target repository")
	return cmd
}

func printLocalMCPTool(cmd *cobra.Command, toolName string, args map[string]any) error {
	text, err := callLocalMCPTool(toolName, args)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(cmd.OutOrStdout(), text)
	return err
}

func callLocalMCPTool(toolName string, args map[string]any) (string, error) {
	request := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      toolName,
			"arguments": stripNilValues(args),
		},
	}
	raw, err := json.Marshal(request)
	if err != nil {
		return "", err
	}
	server := mcpserver.NewServer(mcpserver.Config{Store: repo.NewEnvStore()})
	responseRaw, ok, err := server.HandleJSONRPC(raw)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", fmt.Errorf("tool %q did not return a response", toolName)
	}
	var response struct {
		Result struct {
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		} `json:"result"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(responseRaw, &response); err != nil {
		return "", err
	}
	if response.Error != nil {
		return "", errors.New(response.Error.Message)
	}
	if len(response.Result.Content) == 0 {
		return "", nil
	}
	return response.Result.Content[0].Text, nil
}

func emptyToNil(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}

func stripNilValues(input map[string]any) map[string]any {
	out := make(map[string]any, len(input))
	for key, value := range input {
		if value != nil {
			out[key] = value
		}
	}
	return out
}

func parsePositiveIntFlag(name string, raw string) (int, error) {
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return 0, fmt.Errorf("%s must be a positive integer", name)
	}
	return value, nil
}
