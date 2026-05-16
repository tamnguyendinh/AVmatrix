package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

func newPackageCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "package",
		Short:  "Run package lifecycle helpers",
		Hidden: true,
	}
	cmd.AddCommand(
		&cobra.Command{
			Use:    "build-runtime",
			Short:  "Build the packaged Go runtime binary",
			Hidden: true,
			Args:   cobra.NoArgs,
			RunE: func(cmd *cobra.Command, args []string) error {
				root, err := packageRootForLifecycle()
				if err != nil {
					return err
				}
				return buildGoRuntimePackage(root, cmd.OutOrStdout())
			},
		},
		&cobra.Command{
			Use:    "prepare-go-source",
			Short:  "Copy Go source files for npm package fallback builds",
			Hidden: true,
			Args:   cobra.NoArgs,
			RunE: func(cmd *cobra.Command, args []string) error {
				root, err := packageRootForLifecycle()
				if err != nil {
					return err
				}
				return prepareGoSourcePackage(root, cmd.OutOrStdout())
			},
		},
		&cobra.Command{
			Use:    "ensure-runtime",
			Short:  "Verify the packaged runtime matches this platform",
			Hidden: true,
			Args:   cobra.NoArgs,
			RunE: func(cmd *cobra.Command, args []string) error {
				root, err := packageRootForLifecycle()
				if err != nil {
					return err
				}
				return ensurePackagedRuntime(root, cmd.OutOrStdout())
			},
		},
		&cobra.Command{
			Use:    "clean-go-source",
			Short:  "Remove temporary Go source package output",
			Hidden: true,
			Args:   cobra.NoArgs,
			RunE: func(cmd *cobra.Command, args []string) error {
				root, err := packageRootForLifecycle()
				if err != nil {
					return err
				}
				return cleanGoSourcePackage(root, cmd.OutOrStdout())
			},
		},
	)
	return cmd
}

func packageRootForLifecycle() (string, error) {
	if exe, err := os.Executable(); err == nil && exe != "" {
		binDir := filepath.Dir(exe)
		root := filepath.Dir(binDir)
		if filepath.Base(binDir) == "bin" && packageJSONExists(root) {
			return root, nil
		}
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	if !packageJSONExists(cwd) {
		return "", fmt.Errorf("package root not found from executable or working directory")
	}
	return cwd, nil
}

func packageJSONExists(root string) bool {
	stat, err := os.Stat(filepath.Join(root, "package.json"))
	return err == nil && !stat.IsDir()
}

func cleanGoSourcePackage(packageRoot string, output io.Writer) error {
	root, err := filepath.Abs(packageRoot)
	if err != nil {
		return err
	}
	target := filepath.Join(root, "go-src")
	resolved, err := filepath.Abs(target)
	if err != nil {
		return err
	}
	relative, err := filepath.Rel(root, resolved)
	if err != nil {
		return err
	}
	if relative == "." || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) || filepath.IsAbs(relative) {
		return fmt.Errorf("refusing to remove outside package root: %s", resolved)
	}
	if err := os.RemoveAll(resolved); err != nil {
		return err
	}
	_, err = fmt.Fprintf(output, "[clean-go-source-package] removed %s\n", resolved)
	return err
}
