package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tamnguyendinh/avmatrix-go/internal/repo"
)

func newListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List indexed repositories",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			entries, err := repo.NewEnvStore().ListRegistered(true)
			if err != nil {
				return err
			}
			if len(entries) == 0 {
				_, err = fmt.Fprintln(cmd.OutOrStdout(), "No indexed repositories.")
				return err
			}

			_, err = fmt.Fprintln(cmd.OutOrStdout(), "Indexed repositories:")
			if err != nil {
				return err
			}
			for _, entry := range entries {
				if _, err := fmt.Fprintf(cmd.OutOrStdout(), "- %s\n", entry.Name); err != nil {
					return err
				}
				if _, err := fmt.Fprintf(cmd.OutOrStdout(), "  Path: %s\n", entry.Path); err != nil {
					return err
				}
				if entry.IndexedAt != "" {
					if _, err := fmt.Fprintf(cmd.OutOrStdout(), "  Indexed: %s\n", formatIndexedAt(entry.IndexedAt)); err != nil {
						return err
					}
				}
				if entry.LastCommit != "" {
					if _, err := fmt.Fprintf(cmd.OutOrStdout(), "  Commit: %s\n", shortCommit(entry.LastCommit)); err != nil {
						return err
					}
				}
			}
			return nil
		},
	}
}
