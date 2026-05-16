package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tamnguyendinh/avmatrix-go/internal/repo"
)

func newWikiCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "wiki [path]",
		Short: "Show wiki capability status",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			config := repo.LoadRuntimeConfig()
			if _, err := fmt.Fprintln(cmd.OutOrStdout(), formatWikiModeStatus(config.WikiMode)); err != nil {
				return err
			}
			return ExitError{Code: 1}
		},
	}
}

func newWikiModeCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "wiki-mode [mode]",
		Short: "Show or set wiki capability mode (off or local)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				config := repo.LoadRuntimeConfig()
				_, err := fmt.Fprintln(cmd.OutOrStdout(), formatWikiModeStatus(config.WikiMode))
				return err
			}

			mode, ok := repo.ParseWikiMode(strings.TrimSpace(strings.ToLower(args[0])))
			if !ok {
				_, _ = fmt.Fprint(cmd.ErrOrStderr(), "\n  Invalid wiki mode. Use `off` or `local`.\n")
				return ExitError{Code: 1}
			}
			if err := repo.SaveRuntimeConfig(repo.RuntimeConfig{WikiMode: mode}); err != nil {
				return err
			}
			_, err := fmt.Fprintln(cmd.OutOrStdout(), formatWikiModeStatus(mode))
			return err
		},
	}
}

func formatWikiModeStatus(mode repo.WikiMode) string {
	if mode == repo.WikiModeLocal {
		return "\n" +
			"  Wiki capability mode: local\n\n" +
			"  Local wiki mode is reserved, but the local wiki engine is not available yet in this build.\n" +
			"  AVmatrix will not fall back to any remote wiki service.\n"
	}
	return "\n" +
		"  Wiki capability mode: off\n\n" +
		"  Wiki generation is disabled in local-only mode.\n" +
		"  Run `avmatrix wiki-mode local` later when the local wiki engine is ready.\n"
}
