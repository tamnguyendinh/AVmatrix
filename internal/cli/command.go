package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"

	"github.com/spf13/cobra"
	"github.com/tamnguyendinh/avmatrix-go/internal/analyze"
	"github.com/tamnguyendinh/avmatrix-go/internal/embeddings"
	"github.com/tamnguyendinh/avmatrix-go/internal/httpapi"
	"github.com/tamnguyendinh/avmatrix-go/internal/lbugload"
	"github.com/tamnguyendinh/avmatrix-go/internal/lbugnative"
	"github.com/tamnguyendinh/avmatrix-go/internal/logging"
	mcpserver "github.com/tamnguyendinh/avmatrix-go/internal/mcp"
	"github.com/tamnguyendinh/avmatrix-go/internal/repo"
	"github.com/tamnguyendinh/avmatrix-go/internal/resolution"
	"github.com/tamnguyendinh/avmatrix-go/internal/scanner"
	"github.com/tamnguyendinh/avmatrix-go/internal/version"
)

type Options struct {
	Out    io.Writer
	Err    io.Writer
	Logger *slog.Logger
}

func NewRootCommand(options Options) *cobra.Command {
	out := options.Out
	if out == nil {
		out = os.Stdout
	}
	errOut := options.Err
	if errOut == nil {
		errOut = os.Stderr
	}
	logger := options.Logger
	if logger == nil {
		logger = logging.NewTextLogger(errOut, slog.LevelWarn)
	}

	cmd := &cobra.Command{
		Use:           version.CommandName,
		Short:         "AVmatrix local CLI and MCP server",
		Version:       version.Version,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			logger.Debug("showing root help")
			return cmd.Help()
		},
	}
	cmd.SetOut(out)
	cmd.SetErr(errOut)
	cmd.SetVersionTemplate("{{.Version}}\n")
	cmd.AddCommand(
		newVersionCommand(),
		newServeCommand(logger),
		newAnalyzeCommand(logger),
		newBenchmarkCompareCommand(),
		newCleanCommand(),
		newGroupCommand(),
		newIndexCommand(),
		newMCPCommand(),
		newPackageCommand(),
		newListCommand(),
		newSetupCommand(),
		newStatusCommand(),
		newWikiCommand(),
		newWikiModeCommand(),
		newAugmentCommand(),
		newQueryCommand(),
		newContextCommand(),
		newImpactCommand(),
		newCypherCommand(),
		newDetectChangesCommand(),
		newHookCommand(),
	)

	return cmd
}

func newMCPCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "mcp",
		Short: "Start MCP server over stdio",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return mcpserver.Serve(contextFromCommand(cmd), os.Stdin, cmd.OutOrStdout(), mcpserver.Config{
				Store: repo.NewEnvStore(),
			})
		},
	}
}

func newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the AVmatrix version",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := fmt.Fprintln(cmd.OutOrStdout(), version.Version)
			return err
		},
	}
}

func newServeCommand(logger *slog.Logger) *cobra.Command {
	var host string
	var port int

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start local HTTP bridge for the web UI and shared session runtime",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			logger.Info("starting local HTTP bridge", "host", host, "port", port)
			return httpapi.ListenAndServe(contextFromCommand(cmd), httpapi.ListenConfig{
				Host: host,
				Port: port,
				API: httpapi.Config{
					Store:         repo.NewEnvStore(),
					Version:       version.Version,
					LaunchContext: httpapi.LaunchContextGlobal,
				},
			})
		},
	}
	cmd.Flags().StringVar(&host, "host", httpapi.DefaultHost, "host interface to bind")
	cmd.Flags().IntVar(&port, "port", httpapi.DefaultPort, "port to bind")

	return cmd
}

func newAnalyzeCommand(logger *slog.Logger) *cobra.Command {
	var benchmarkPath string
	var include []string
	var exclude []string
	var noGitignore bool
	var progress bool
	var force bool
	var enableEmbeddings bool
	var generateSkills bool
	var skipAgentsMD bool
	var noStats bool
	var skipGit bool
	var skipCompatibilityCrossFile bool
	var benchmarkLabel string
	var cpuProfilePath string
	var memProfilePath string
	var registryName string
	var allowDuplicateName bool
	var verbose bool

	cmd := &cobra.Command{
		Use:   "analyze [path]",
		Short: "Analyze a local repository with the Go pipeline",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			target, err := resolveAnalyzeArgument(args, skipGit)
			if err != nil {
				return err
			}
			logger.Info("starting analyze", "path", target)

			options := analyze.Options{
				Scanner: scanner.Options{
					NoGitignore: noGitignore,
					Include:     include,
					Exclude:     exclude,
				},
				Resolution: resolution.Options{
					SkipCompatibilityCrossFile: skipCompatibilityCrossFile,
				},
				BenchmarkPath:                  benchmarkPath,
				BenchmarkLabel:                 benchmarkLabel,
				Force:                          force || generateSkills,
				Embeddings:                     enableEmbeddings,
				WriteGraphSnapshot:             true,
				ReleaseScopeIRsAfterResolution: true,
			}
			if enableEmbeddings {
				options.EmbeddingConfig = embeddings.DefaultConfig()
				dimensions, ok, err := embeddings.HTTPDimensions(nil)
				if err != nil {
					return err
				}
				if ok {
					options.EmbeddingConfig.Dimensions = dimensions
				}
			}
			options.DBRunnerFactory = func(paths repo.StoragePaths) (lbugload.QueryRunner, func() error, error) {
				return nativeDBRunnerFactory(paths, options.EmbeddingConfig.Dimensions)
			}
			if progress || verbose {
				options.OnEvent = func(event analyze.Event) {
					if event.Kind == analyze.EventPhaseDone {
						_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "%s done\n", event.Phase)
					}
				}
			}

			stopCPUProfile, err := startAnalyzeCPUProfile(cpuProfilePath)
			if err != nil {
				return err
			}
			defer stopCPUProfile()

			result, err := analyze.Run(contextFromCommand(cmd), target, options)
			if err != nil {
				return err
			}
			if err := writeAnalyzeMemoryProfile(memProfilePath); err != nil {
				return err
			}
			registration, err := recordAnalyzeResult(result, analyzeRecordOptions{
				Name:               registryName,
				AllowDuplicateName: allowDuplicateName,
			})
			if err != nil {
				return err
			}
			var aiResult analyzeAIContextResult
			if generateSkills {
				aiResult, err = generateAnalyzeAIContext(result, registration.Name, skipAgentsMD, noStats)
				if err != nil {
					return err
				}
			}
			_, err = fmt.Fprintf(
				cmd.OutOrStdout(),
				"analyzed %s\nfiles: scanned=%d parsed=%d unsupported=%d failed=%d\ngraph: nodes=%d relationships=%d path=%s\n",
				result.RepoPath,
				result.Metrics.Files.Scanned,
				result.Metrics.Files.Parsed,
				result.Metrics.Files.Unsupported,
				result.Metrics.Files.Failed,
				len(result.Graph.Nodes),
				len(result.Graph.Relationships),
				result.GraphPath,
			)
			if err != nil {
				return err
			}
			if generateSkills {
				_, err = fmt.Fprintf(
					cmd.OutOrStdout(),
					"skills: generated=%d base=%d files=%d\n",
					len(aiResult.GeneratedSkills),
					aiResult.BaseSkillCount,
					len(aiResult.Files),
				)
			}
			return err
		},
	}
	cmd.Flags().StringVar(&benchmarkPath, "benchmark-json", "", "write analyze benchmark metrics JSON")
	cmd.Flags().StringVar(&benchmarkLabel, "benchmark-label", "", "attach a label to the benchmark JSON artifact")
	cmd.Flags().StringVar(&cpuProfilePath, "cpuprofile", "", "write Go CPU pprof profile for analyze")
	cmd.Flags().StringVar(&memProfilePath, "memprofile", "", "write Go heap pprof profile after analyze")
	cmd.Flags().StringArrayVar(&include, "include", nil, "include glob pattern")
	cmd.Flags().StringArrayVar(&exclude, "exclude", nil, "exclude glob pattern")
	cmd.Flags().BoolVar(&noGitignore, "no-gitignore", false, "ignore .gitignore rules")
	cmd.Flags().BoolVar(&progress, "progress", false, "write phase progress to stderr")
	cmd.Flags().BoolVar(&force, "force", false, "remove previous index output before analyze")
	cmd.Flags().BoolVar(&enableEmbeddings, "embeddings", false, "enable embedding generation for semantic search")
	cmd.Flags().BoolVar(&generateSkills, "skills", false, "generate repo-specific skill files from detected communities")
	cmd.Flags().BoolVar(&skipAgentsMD, "skip-agents-md", false, "skip updating the AVmatrix section in AGENTS.md and CLAUDE.md")
	cmd.Flags().BoolVar(&noStats, "no-stats", false, "omit volatile file/symbol counts from AGENTS.md and CLAUDE.md")
	cmd.Flags().BoolVar(&skipGit, "skip-git", false, "index a folder without requiring a .git directory")
	cmd.Flags().BoolVar(&skipCompatibilityCrossFile, "skip-compatibility-cross-file", false, "diagnostic benchmark mode: skip compatibility cross-file work")
	cmd.Flags().StringVar(&registryName, "name", "", "register this repo under a custom name in the global registry")
	cmd.Flags().BoolVar(&allowDuplicateName, "allow-duplicate-name", false, "allow registering another repo with the same --name alias")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose ingestion progress")
	return cmd
}

func startAnalyzeCPUProfile(path string) (func(), error) {
	if path == "" {
		return func() {}, nil
	}
	file, err := os.Create(path)
	if err != nil {
		return nil, err
	}
	if err := pprof.StartCPUProfile(file); err != nil {
		_ = file.Close()
		return nil, err
	}
	return func() {
		pprof.StopCPUProfile()
		_ = file.Close()
	}, nil
}

func writeAnalyzeMemoryProfile(path string) error {
	if path == "" {
		return nil
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	runtime.GC()
	return pprof.WriteHeapProfile(file)
}

func nativeDBRunnerFactory(paths repo.StoragePaths, embeddingDims int) (lbugload.QueryRunner, func() error, error) {
	runner, err := lbugnative.OpenWriteRunnerWithEmbeddingDims(paths.LbugPath, embeddingDims)
	if errors.Is(err, lbugnative.ErrUnavailable) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}
	return runner, runner.Close, nil
}

func resolveAnalyzeArgument(args []string, skipGit bool) (string, error) {
	if len(args) == 0 {
		cwd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		gitRoot := repo.GitRoot(cwd)
		if gitRoot == "" {
			if !skipGit {
				return "", fmt.Errorf("not inside a git repository; pass --skip-git to index any folder without a .git directory")
			}
			return cwd, nil
		}
		return gitRoot, nil
	}
	target := args[0]
	if !filepath.IsAbs(target) {
		absolute, err := filepath.Abs(target)
		if err != nil {
			return "", err
		}
		target = absolute
	}
	if !skipGit && !repo.IsGitRepo(target) {
		return "", fmt.Errorf("not a git repository: %s; pass --skip-git to index any folder without a .git directory", target)
	}
	return target, nil
}

func contextFromCommand(cmd *cobra.Command) context.Context {
	if ctx := cmd.Context(); ctx != nil {
		return ctx
	}
	return context.Background()
}
