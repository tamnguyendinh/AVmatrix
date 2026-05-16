package analyze

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"time"

	sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/tamnguyendinh/avmatrix-go/internal/cobol"
	"github.com/tamnguyendinh/avmatrix-go/internal/communities"
	"github.com/tamnguyendinh/avmatrix-go/internal/documents"
	"github.com/tamnguyendinh/avmatrix-go/internal/embeddings"
	"github.com/tamnguyendinh/avmatrix-go/internal/frameworks"
	"github.com/tamnguyendinh/avmatrix-go/internal/graph"
	"github.com/tamnguyendinh/avmatrix-go/internal/lbugload"
	"github.com/tamnguyendinh/avmatrix-go/internal/lbugruntime"
	"github.com/tamnguyendinh/avmatrix-go/internal/mro"
	"github.com/tamnguyendinh/avmatrix-go/internal/orm"
	"github.com/tamnguyendinh/avmatrix-go/internal/parser"
	"github.com/tamnguyendinh/avmatrix-go/internal/processes"
	"github.com/tamnguyendinh/avmatrix-go/internal/providers/astro"
	cprovider "github.com/tamnguyendinh/avmatrix-go/internal/providers/c"
	"github.com/tamnguyendinh/avmatrix-go/internal/providers/cpp"
	"github.com/tamnguyendinh/avmatrix-go/internal/providers/csharp"
	"github.com/tamnguyendinh/avmatrix-go/internal/providers/dart"
	"github.com/tamnguyendinh/avmatrix-go/internal/providers/golang"
	"github.com/tamnguyendinh/avmatrix-go/internal/providers/java"
	"github.com/tamnguyendinh/avmatrix-go/internal/providers/kotlin"
	"github.com/tamnguyendinh/avmatrix-go/internal/providers/php"
	"github.com/tamnguyendinh/avmatrix-go/internal/providers/python"
	"github.com/tamnguyendinh/avmatrix-go/internal/providers/ruby"
	"github.com/tamnguyendinh/avmatrix-go/internal/providers/rust"
	"github.com/tamnguyendinh/avmatrix-go/internal/providers/svelte"
	"github.com/tamnguyendinh/avmatrix-go/internal/providers/swift"
	"github.com/tamnguyendinh/avmatrix-go/internal/providers/tsjs"
	"github.com/tamnguyendinh/avmatrix-go/internal/providers/vue"
	"github.com/tamnguyendinh/avmatrix-go/internal/repo"
	"github.com/tamnguyendinh/avmatrix-go/internal/resolution"
	"github.com/tamnguyendinh/avmatrix-go/internal/routes"
	"github.com/tamnguyendinh/avmatrix-go/internal/scanner"
	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
	"github.com/tamnguyendinh/avmatrix-go/internal/structure"
	"github.com/tamnguyendinh/avmatrix-go/internal/tools"
)

type DBRunnerFactory func(repo.StoragePaths) (lbugload.QueryRunner, func() error, error)
type EmbeddingEmbedderFactory func(embeddings.Config) (embeddings.Embedder, error)

type PhaseName string

const (
	PhaseScan        PhaseName = "scan"
	PhaseStructure   PhaseName = "structure"
	PhaseDocuments   PhaseName = "documents"
	PhaseCobol       PhaseName = "cobol"
	PhaseRoutes      PhaseName = "routes"
	PhaseTools       PhaseName = "tools"
	PhaseORM         PhaseName = "orm"
	PhaseCrossFile   PhaseName = "cross_file_binding"
	PhaseMRO         PhaseName = "mro"
	PhaseParse       PhaseName = "parse"
	PhaseResolution  PhaseName = "resolution"
	PhaseCommunities PhaseName = "communities"
	PhaseProcesses   PhaseName = "processes"
	PhaseDBLoad      PhaseName = "db_load"
	PhaseEmbeddings  PhaseName = "embeddings"
)

const maxProcessesEnv = "AVMATRIX_MAX_PROCESSES"

type EventKind string

const (
	EventPhaseStart EventKind = "phase_start"
	EventPhaseDone  EventKind = "phase_done"
	EventProgress   EventKind = "progress"
)

type Options struct {
	Scanner                        scanner.Options
	Parser                         parser.PoolOptions
	Resolution                     resolution.Options
	Processes                      processes.Config
	OnEvent                        func(Event)
	BenchmarkPath                  string
	BenchmarkLabel                 string
	Force                          bool
	Embeddings                     bool
	EmbeddingConfig                embeddings.Config
	WriteGraphSnapshot             bool
	DBRunner                       lbugload.QueryRunner
	DBRunnerFactory                DBRunnerFactory
	EmbedderFactory                EmbeddingEmbedderFactory
	ExistingEmbeddings             map[string]string
	ReleaseScopeIRsAfterResolution bool
}

type Event struct {
	Kind    EventKind `json:"kind"`
	Phase   PhaseName `json:"phase"`
	Current int       `json:"current,omitempty"`
	Total   int       `json:"total,omitempty"`
	File    string    `json:"file,omitempty"`
}

type Result struct {
	RepoPath  string            `json:"repoPath"`
	GraphPath string            `json:"graphPath,omitempty"`
	Graph     *graph.Graph      `json:"graph,omitempty"`
	ScopeIRs  []scopeir.ScopeIR `json:"-"`
	Metrics   Metrics           `json:"metrics"`
}

type Metrics struct {
	Label         string              `json:"label,omitempty"`
	TotalDuration time.Duration       `json:"totalDuration"`
	Phases        []PhaseMetric       `json:"phases"`
	Scanner       scanner.Metrics     `json:"scanner"`
	Structure     structure.Metrics   `json:"structure,omitempty"`
	Documents     documents.Metrics   `json:"documents,omitempty"`
	Cobol         cobol.Metrics       `json:"cobol,omitempty"`
	Routes        routes.Metrics      `json:"routes,omitempty"`
	Tools         tools.Metrics       `json:"tools,omitempty"`
	ORM           orm.Metrics         `json:"orm,omitempty"`
	CrossFile     resolution.Metrics  `json:"crossFileBinding,omitempty"`
	MRO           mro.Metrics         `json:"mro,omitempty"`
	Parser        parser.Metrics      `json:"parser"`
	Resolution    resolution.Metrics  `json:"resolution"`
	Communities   communities.Metrics `json:"communities,omitempty"`
	Processes     processes.Metrics   `json:"processes,omitempty"`
	DBLoad        DBLoadMetrics       `json:"dbLoad,omitempty"`
	Embeddings    EmbeddingMetrics    `json:"embeddings,omitempty"`
	Files         FileMetrics         `json:"files"`
	Memory        MemoryMetrics       `json:"memory"`
}

type PhaseMetric struct {
	Name     PhaseName     `json:"name"`
	Duration time.Duration `json:"duration"`
}

type FileMetrics struct {
	Scanned     int `json:"scanned"`
	Parsed      int `json:"parsed"`
	Unsupported int `json:"unsupported"`
	Failed      int `json:"failed"`
}

type MemoryMetrics struct {
	StartAllocBytes uint64 `json:"startAllocBytes"`
	EndAllocBytes   uint64 `json:"endAllocBytes"`
	MaxObservedSys  uint64 `json:"maxObservedSys"`
}

type DBLoadMetrics struct {
	NodeCopyCount          int    `json:"nodeCopyCount"`
	RelationshipCopyCount  int    `json:"relationshipCopyCount"`
	FallbackInsertCount    int    `json:"fallbackInsertCount"`
	FallbackInsertFailures int    `json:"fallbackInsertFailures"`
	SkippedRelationships   int    `json:"skippedRelationships"`
	NodeRows               int    `json:"nodeRows"`
	RelationshipRows       int    `json:"relationshipRows"`
	Skipped                bool   `json:"skipped,omitempty"`
	SkipReason             string `json:"skipReason,omitempty"`
}

type EmbeddingMetrics struct {
	TotalNodes         int    `json:"totalNodes,omitempty"`
	EmbeddedNodes      int    `json:"embeddedNodes,omitempty"`
	SkippedFreshNodes  int    `json:"skippedFreshNodes,omitempty"`
	StaleNodes         int    `json:"staleNodes,omitempty"`
	Chunks             int    `json:"chunks,omitempty"`
	DeleteQueries      int    `json:"deleteQueries,omitempty"`
	InsertQueries      int    `json:"insertQueries,omitempty"`
	VectorIndexCreated bool   `json:"vectorIndexCreated,omitempty"`
	Skipped            bool   `json:"skipped,omitempty"`
	SkipReason         string `json:"skipReason,omitempty"`
}

type scanResult struct {
	Files   []scanner.File
	Metrics scanner.Metrics
}

func Run(ctx context.Context, repoPath string, options Options) (result Result, err error) {
	if ctx == nil {
		ctx = context.Background()
	}
	resolvedPath, err := repo.ResolveAnalyzePath(repoPath)
	if err != nil {
		return Result{}, err
	}
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	paths := repo.Paths(resolvedPath)
	lock, err := repo.AcquireStorageLock(paths.AnalyzeLockPath)
	if err != nil {
		return Result{}, err
	}
	defer lock.Release()
	if err := prepareStorage(paths, options.Force); err != nil {
		return Result{}, err
	}
	defer os.RemoveAll(paths.AnalyzeTempPath)

	start := time.Now()
	result = Result{RepoPath: resolvedPath}
	result.Metrics.Memory.StartAllocBytes = currentAlloc()
	result.Metrics.Memory.MaxObservedSys = currentSys()
	defer func() {
		finalizeMetrics(&result.Metrics, start)
	}()

	scan, err := runPhase(ctx, &result.Metrics, options.OnEvent, PhaseScan, func() (scanResult, error) {
		files, metrics, err := scanner.WalkRepositoryPaths(resolvedPath, options.Scanner, func(current int, total int, filePath string) {
			emit(options.OnEvent, Event{Kind: EventProgress, Phase: PhaseScan, Current: current, Total: total, File: filePath})
		})
		return scanResult{Files: files, Metrics: metrics}, err
	})
	if err != nil {
		return result, err
	}
	result.Metrics.Scanner = scan.Metrics
	result.Metrics.Files.Scanned = len(scan.Files)
	result.Graph = graph.New()

	structureResult, err := runPhase(ctx, &result.Metrics, options.OnEvent, PhaseStructure, func() (structure.Result, error) {
		if err := ctx.Err(); err != nil {
			return structure.Result{}, err
		}
		return structure.Apply(result.Graph, scan.Files), nil
	})
	if err != nil {
		return result, err
	}
	result.Metrics.Structure = structureResult.Metrics

	documentResult, err := runPhase(ctx, &result.Metrics, options.OnEvent, PhaseDocuments, func() (documents.Result, error) {
		if err := ctx.Err(); err != nil {
			return documents.Result{}, err
		}
		return documents.Apply(result.Graph, resolvedPath, scan.Files)
	})
	if err != nil {
		return result, err
	}
	result.Metrics.Documents = documentResult.Metrics

	cobolResult, err := runPhase(ctx, &result.Metrics, options.OnEvent, PhaseCobol, func() (cobol.Result, error) {
		if err := ctx.Err(); err != nil {
			return cobol.Result{}, err
		}
		return cobol.Apply(result.Graph, resolvedPath, scan.Files)
	})
	if err != nil {
		return result, err
	}
	result.Metrics.Cobol = cobolResult.Metrics

	parsedFiles, err := runPhase(ctx, &result.Metrics, options.OnEvent, PhaseParse, func() (parseRunResult, error) {
		return parseFiles(ctx, resolvedPath, scan.Files, options)
	})
	if err != nil {
		return result, err
	}
	result.ScopeIRs = parsedFiles.IRs
	result.Metrics.Parser = parsedFiles.Metrics.Parser
	result.Metrics.Files.Parsed = parsedFiles.Metrics.Parsed
	result.Metrics.Files.Unsupported = parsedFiles.Metrics.Unsupported
	result.Metrics.Files.Failed = parsedFiles.Metrics.Failed

	routesResult, err := runPhase(ctx, &result.Metrics, options.OnEvent, PhaseRoutes, func() (routes.Result, error) {
		if err := ctx.Err(); err != nil {
			return routes.Result{}, err
		}
		return routes.Apply(result.Graph, resolvedPath, scan.Files)
	})
	if err != nil {
		return result, err
	}
	result.Metrics.Routes = routesResult.Metrics

	toolsResult, err := runPhase(ctx, &result.Metrics, options.OnEvent, PhaseTools, func() (tools.Result, error) {
		if err := ctx.Err(); err != nil {
			return tools.Result{}, err
		}
		return tools.Apply(result.Graph, resolvedPath, scan.Files)
	})
	if err != nil {
		return result, err
	}
	result.Metrics.Tools = toolsResult.Metrics

	ormResult, err := runPhase(ctx, &result.Metrics, options.OnEvent, PhaseORM, func() (orm.Result, error) {
		if err := ctx.Err(); err != nil {
			return orm.Result{}, err
		}
		return orm.Apply(result.Graph, resolvedPath, scan.Files)
	})
	if err != nil {
		return result, err
	}
	result.Metrics.ORM = ormResult.Metrics

	crossFileResult, err := runPhase(ctx, &result.Metrics, options.OnEvent, PhaseCrossFile, func() (resolution.BindingResult, error) {
		if err := ctx.Err(); err != nil {
			return resolution.BindingResult{}, err
		}
		return resolution.BuildCrossFileBinding(parsedFiles.IRs, options.Resolution)
	})
	if err != nil {
		return result, err
	}
	result.Metrics.CrossFile = crossFileResult.Metrics

	resolutionResult, err := runPhase(ctx, &result.Metrics, options.OnEvent, PhaseResolution, func() (resolution.Result, error) {
		if err := ctx.Err(); err != nil {
			return resolution.Result{}, err
		}
		return resolution.ResolveBoundInto(result.Graph, crossFileResult, options.Resolution)
	})
	if err != nil {
		return result, err
	}
	result.Graph = resolutionResult.Graph
	result.Metrics.Resolution = resolutionResult.Metrics
	result.Metrics.Memory.MaxObservedSys = maxUint64(result.Metrics.Memory.MaxObservedSys, currentSys())
	if options.ReleaseScopeIRsAfterResolution {
		result.ScopeIRs = nil
		parsedFiles.IRs = nil
		crossFileResult = resolution.BindingResult{}
	}

	mroResult, err := runPhase(ctx, &result.Metrics, options.OnEvent, PhaseMRO, func() (mro.Result, error) {
		if err := ctx.Err(); err != nil {
			return mro.Result{}, err
		}
		return mro.Apply(result.Graph), nil
	})
	if err != nil {
		return result, err
	}
	result.Metrics.MRO = mroResult.Metrics

	communityResult, err := runPhase(ctx, &result.Metrics, options.OnEvent, PhaseCommunities, func() (communities.Result, error) {
		if err := ctx.Err(); err != nil {
			return communities.Result{}, err
		}
		return communities.Apply(result.Graph), nil
	})
	if err != nil {
		return result, err
	}
	result.Metrics.Communities = communityResult.Metrics

	processConfig := resolveProcessConfig(resolvedPath, options.Processes)
	processResult, err := runPhase(ctx, &result.Metrics, options.OnEvent, PhaseProcesses, func() (processes.Result, error) {
		if err := ctx.Err(); err != nil {
			return processes.Result{}, err
		}
		return processes.Apply(result.Graph, processConfig), nil
	})
	if err != nil {
		return result, err
	}
	result.Metrics.Processes = processResult.Metrics
	result.Graph.Compact()
	result.Metrics.Memory.MaxObservedSys = maxUint64(result.Metrics.Memory.MaxObservedSys, currentSys())

	dbRunner, closeDBRunner, skipReason, err := resolveDBRunner(paths, options)
	if err != nil {
		return result, err
	}
	if dbRunner != nil {
		dbLoad, err := runPhase(ctx, &result.Metrics, options.OnEvent, PhaseDBLoad, func() (DBLoadMetrics, error) {
			return loadGraph(ctx, paths, result.Graph, dbRunner)
		})
		if err != nil {
			if closeDBRunner != nil {
				_ = closeDBRunner()
			}
			return result, err
		}
		result.Metrics.DBLoad = dbLoad
		if options.Embeddings {
			embeddingMetrics, err := runPhase(ctx, &result.Metrics, options.OnEvent, PhaseEmbeddings, func() (EmbeddingMetrics, error) {
				return runEmbeddings(ctx, result.Graph, dbRunner, options)
			})
			if err != nil {
				if closeDBRunner != nil {
					_ = closeDBRunner()
				}
				return result, err
			}
			result.Metrics.Embeddings = embeddingMetrics
		}
		if closeDBRunner != nil {
			if err := closeDBRunner(); err != nil {
				return result, err
			}
		}
	} else {
		result.Metrics.DBLoad.Skipped = true
		result.Metrics.DBLoad.SkipReason = skipReason
		if options.Embeddings {
			result.Metrics.Embeddings.Skipped = true
			result.Metrics.Embeddings.SkipReason = skipReason
			return result, fmt.Errorf("embeddings require a DB query runner: %s", skipReason)
		}
	}

	if options.WriteGraphSnapshot {
		if err := writeGraphSnapshot(paths.GraphPath, result.Graph); err != nil {
			return result, err
		}
		result.GraphPath = paths.GraphPath
	}
	if options.BenchmarkPath != "" {
		finalizeMetrics(&result.Metrics, start)
		result.Metrics.Label = options.BenchmarkLabel
		if err := WriteBenchmark(options.BenchmarkPath, result); err != nil {
			return result, err
		}
	}
	return result, nil
}

func resolveProcessConfig(repoPath string, config processes.Config) processes.Config {
	if config.MaxProcesses > 0 || config.MaxProcessesCap > 0 {
		return config
	}
	config.MaxProcessesCap = resolveConfiguredMaxProcessesCap(repoPath, os.Getenv(maxProcessesEnv))
	return config
}

func resolveConfiguredMaxProcessesCap(repoPath string, envValue string) int {
	if parsed := positiveInt(envValue); parsed > 0 {
		return parsed
	}
	settings, err := repo.LoadSettings(repoPath)
	if err == nil && settings.MaxExecutionFlows > 0 {
		return settings.MaxExecutionFlows
	}
	return repo.DefaultMaxExecutionFlows
}

func positiveInt(value string) int {
	if value == "" {
		return 0
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return 0
	}
	return parsed
}

func finalizeMetrics(metrics *Metrics, start time.Time) {
	metrics.TotalDuration = time.Since(start)
	metrics.Memory.EndAllocBytes = currentAlloc()
	metrics.Memory.MaxObservedSys = maxUint64(metrics.Memory.MaxObservedSys, currentSys())
}

func resolveDBRunner(paths repo.StoragePaths, options Options) (lbugload.QueryRunner, func() error, string, error) {
	if options.DBRunner != nil {
		return options.DBRunner, nil, "", nil
	}
	if options.DBRunnerFactory == nil {
		return nil, nil, "no query runner configured", nil
	}
	runner, closeRunner, err := options.DBRunnerFactory(paths)
	if err != nil {
		return nil, nil, "", err
	}
	if runner == nil {
		return nil, nil, "query runner factory returned nil", nil
	}
	return runner, closeRunner, "", nil
}

func prepareStorage(paths repo.StoragePaths, force bool) error {
	if err := os.MkdirAll(paths.StoragePath, 0o755); err != nil {
		return err
	}
	if force {
		if err := os.RemoveAll(paths.LbugPath); err != nil {
			return err
		}
		if err := os.Remove(paths.GraphPath); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	if err := os.RemoveAll(paths.AnalyzeTempPath); err != nil {
		return err
	}
	return os.MkdirAll(paths.AnalyzeTempPath, 0o755)
}

func loadGraph(ctx context.Context, paths repo.StoragePaths, graph *graph.Graph, runner lbugload.QueryRunner) (DBLoadMetrics, error) {
	if err := ctx.Err(); err != nil {
		return DBLoadMetrics{}, err
	}
	export, err := lbugload.ExportGraphCSVs(graph, filepath.Join(paths.AnalyzeTempPath, "csv"))
	if err != nil {
		return DBLoadMetrics{}, err
	}
	load, err := lbugload.LoadCSVExport(runner, export)
	if err != nil {
		return DBLoadMetrics{}, err
	}
	return DBLoadMetrics{
		NodeCopyCount:          load.NodeCopyCount,
		RelationshipCopyCount:  load.RelationshipCopyCount,
		FallbackInsertCount:    load.FallbackInsertCount,
		FallbackInsertFailures: load.FallbackInsertFailures,
		SkippedRelationships:   load.SkippedRelationships,
		NodeRows:               nodeRows(export),
		RelationshipRows:       export.RelationshipRows,
	}, nil
}

type embeddingHashReader interface {
	QueryRows(string) ([]lbugruntime.Row, error)
}

func runEmbeddings(ctx context.Context, graph *graph.Graph, runner lbugload.QueryRunner, options Options) (EmbeddingMetrics, error) {
	embedder, config, err := resolveEmbedder(ctx, options)
	if err != nil {
		return EmbeddingMetrics{}, err
	}
	existingHashes := options.ExistingEmbeddings
	if existingHashes == nil {
		existingHashes = map[string]string{}
		if reader, ok := runner.(embeddingHashReader); ok {
			result, err := lbugruntime.FetchExistingEmbeddingHashes(reader.QueryRows)
			if err != nil {
				return EmbeddingMetrics{}, err
			}
			if result.Hashes != nil {
				existingHashes = result.Hashes
			}
		}
	}
	runResult, err := embeddings.Run(ctx, graph, runner, embedder, embeddings.RunOptions{
		Config:         config,
		ExistingHashes: existingHashes,
	})
	if err != nil {
		return EmbeddingMetrics{}, err
	}
	return embeddingMetricsFromRunResult(runResult), nil
}

func resolveEmbedder(ctx context.Context, options Options) (embeddings.Embedder, embeddings.Config, error) {
	config := embeddings.NormalizeConfig(options.EmbeddingConfig)
	if options.EmbedderFactory != nil {
		embedder, err := options.EmbedderFactory(config)
		return embedder, config, err
	}
	return embeddings.ResolveRuntimeEmbedder(ctx, config, nil)
}

func embeddingMetricsFromRunResult(result embeddings.RunResult) EmbeddingMetrics {
	return EmbeddingMetrics{
		TotalNodes:         result.TotalNodes,
		EmbeddedNodes:      result.EmbeddedNodes,
		SkippedFreshNodes:  result.SkippedFreshNodes,
		StaleNodes:         result.StaleNodes,
		Chunks:             result.Chunks,
		DeleteQueries:      result.DeleteQueries,
		InsertQueries:      result.InsertQueries,
		VectorIndexCreated: result.VectorIndexCreated,
	}
}

func nodeRows(export *lbugload.CSVExport) int {
	total := 0
	for table, rows := range export.Metrics.RowsByTable {
		if table != "Relationship" {
			total += rows
		}
	}
	return total
}

func writeGraphSnapshot(path string, graph *graph.Graph) error {
	if graph == nil {
		return fmt.Errorf("graph snapshot is nil")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tempPath := path + ".tmp"
	file, err := os.Create(tempPath)
	if err != nil {
		return err
	}
	closed := false
	committed := false
	defer func() {
		if !closed {
			_ = file.Close()
		}
		if !committed {
			_ = os.Remove(tempPath)
		}
	}()

	writer := bufio.NewWriter(file)
	if err := writeGraphSnapshotJSON(writer, graph); err != nil {
		return err
	}
	if err := writer.Flush(); err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	closed = true
	if err := os.Rename(tempPath, path); err != nil {
		return err
	}
	committed = true
	return nil
}

func writeGraphSnapshotJSON(writer *bufio.Writer, graph *graph.Graph) error {
	if _, err := writer.WriteString("{\n  \"nodes\": "); err != nil {
		return err
	}
	if err := writeGraphSnapshotArray(writer, graph.Nodes, "    ", "  "); err != nil {
		return err
	}
	if _, err := writer.WriteString(",\n  \"relationships\": "); err != nil {
		return err
	}
	if err := writeGraphSnapshotArray(writer, graph.Relationships, "    ", "  "); err != nil {
		return err
	}
	_, err := writer.WriteString("\n}\n")
	return err
}

func writeGraphSnapshotArray[T any](writer *bufio.Writer, values []T, itemPrefix string, closingPrefix string) error {
	if len(values) == 0 {
		_, err := writer.WriteString("[]")
		return err
	}
	if _, err := writer.WriteString("[\n"); err != nil {
		return err
	}
	for index, value := range values {
		if index > 0 {
			if _, err := writer.WriteString(",\n"); err != nil {
				return err
			}
		}
		if err := writeIndentedJSONValue(writer, value, itemPrefix); err != nil {
			return err
		}
	}
	if _, err := writer.WriteString("\n"); err != nil {
		return err
	}
	if _, err := writer.WriteString(closingPrefix); err != nil {
		return err
	}
	_, err := writer.WriteString("]")
	return err
}

func writeIndentedJSONValue(writer *bufio.Writer, value any, prefix string) error {
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	if _, err := writer.WriteString(prefix); err != nil {
		return err
	}
	start := 0
	for index, char := range raw {
		if char != '\n' {
			continue
		}
		if _, err := writer.Write(raw[start : index+1]); err != nil {
			return err
		}
		start = index + 1
		if start < len(raw) {
			if _, err := writer.WriteString(prefix); err != nil {
				return err
			}
		}
	}
	if start < len(raw) {
		if _, err := writer.Write(raw[start:]); err != nil {
			return err
		}
	}
	return nil
}

func WriteBenchmark(path string, result Result) error {
	raw, err := json.MarshalIndent(result.Metrics, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, append(raw, '\n'), 0o644)
}

type parseResult struct {
	Parser      parser.Metrics
	Parsed      int
	Unsupported int
	Failed      int
}

type parseRunResult struct {
	IRs     []scopeir.ScopeIR
	Metrics parseResult
}

func parseFiles(ctx context.Context, repoPath string, files []scanner.File, options Options) (parseRunResult, error) {
	pool := parser.NewPool(nil, options.Parser)
	defer pool.Close()

	irs := make([]scopeir.ScopeIR, 0, len(files))
	metrics := parseResult{}
	for index, file := range files {
		if err := ctx.Err(); err != nil {
			return parseRunResult{IRs: irs, Metrics: metrics}, err
		}
		if !hasExtractor(file.Language) {
			metrics.Unsupported++
			continue
		}
		emit(options.OnEvent, Event{Kind: EventProgress, Phase: PhaseParse, Current: index + 1, Total: len(files), File: file.Path})

		source, err := os.ReadFile(filepath.Join(repoPath, filepath.FromSlash(file.Path)))
		if err != nil {
			metrics.Failed++
			return parseRunResult{IRs: irs, Metrics: metrics}, err
		}
		if isScriptContainer(file.Language) {
			ir, err := extractScriptContainerScopeIR(file, source)
			if err != nil {
				metrics.Failed++
				return parseRunResult{IRs: irs, Metrics: metrics}, err
			}
			ir = frameworks.AnnotateScopeIR(ir, source)
			irs = append(irs, ir)
			metrics.Parsed++
			continue
		}

		parsed, err := pool.Parse(ctx, parser.Request{FilePath: file.Path, Language: file.Language, Source: source})
		if err != nil {
			metrics.Failed++
			if errors.Is(err, parser.ErrUnsupportedLanguage) {
				metrics.Unsupported++
				continue
			}
			return parseRunResult{IRs: irs, Metrics: metrics}, err
		}

		ir, err := extractScopeIR(file, source, parsed.Tree.RootNode())
		parsed.Close()
		if err != nil {
			metrics.Failed++
			return parseRunResult{IRs: irs, Metrics: metrics}, err
		}
		ir = frameworks.AnnotateScopeIR(ir, source)
		irs = append(irs, ir)
		metrics.Parsed++
	}
	metrics.Parser = pool.SnapshotMetrics()
	return parseRunResult{IRs: irs, Metrics: metrics}, nil
}

func isScriptContainer(language scanner.Language) bool {
	return language == scanner.Vue || language == scanner.Svelte || language == scanner.Astro
}

func extractScriptContainerScopeIR(file scanner.File, source []byte) (scopeir.ScopeIR, error) {
	switch file.Language {
	case scanner.Vue:
		return vue.Extract(vue.Request{
			FilePath: file.Path,
			FileHash: file.Hash,
			Language: file.Language,
			Source:   source,
		})
	case scanner.Svelte:
		return svelte.Extract(svelte.Request{
			FilePath: file.Path,
			FileHash: file.Hash,
			Language: file.Language,
			Source:   source,
		})
	case scanner.Astro:
		return astro.Extract(astro.Request{
			FilePath: file.Path,
			FileHash: file.Hash,
			Language: file.Language,
			Source:   source,
		})
	default:
		return scopeir.ScopeIR{}, fmt.Errorf("unsupported script container language %q", file.Language)
	}
}

func hasExtractor(language scanner.Language) bool {
	switch language {
	case scanner.JavaScript, scanner.TypeScript, scanner.Go, scanner.Python, scanner.Java, scanner.Kotlin, scanner.C, scanner.CSharp, scanner.CPlusPlus, scanner.Rust, scanner.PHP, scanner.Dart, scanner.Vue, scanner.Svelte, scanner.Astro, scanner.Swift, scanner.Ruby:
		return true
	default:
		return false
	}
}

func extractScopeIR(file scanner.File, source []byte, root *sitter.Node) (scopeir.ScopeIR, error) {
	switch file.Language {
	case scanner.Go:
		return golang.Extract(golang.Request{
			FilePath: file.Path,
			FileHash: file.Hash,
			Language: file.Language,
			Source:   source,
			Root:     root,
		})
	case scanner.Python:
		return python.Extract(python.Request{
			FilePath: file.Path,
			FileHash: file.Hash,
			Language: file.Language,
			Source:   source,
			Root:     root,
		})
	case scanner.Java:
		return java.Extract(java.Request{
			FilePath: file.Path,
			FileHash: file.Hash,
			Language: file.Language,
			Source:   source,
			Root:     root,
		})
	case scanner.Kotlin:
		return kotlin.Extract(kotlin.Request{
			FilePath: file.Path,
			FileHash: file.Hash,
			Language: file.Language,
			Source:   source,
			Root:     root,
		})
	case scanner.C:
		return cprovider.Extract(cprovider.Request{
			FilePath: file.Path,
			FileHash: file.Hash,
			Language: file.Language,
			Source:   source,
			Root:     root,
		})
	case scanner.CSharp:
		return csharp.Extract(csharp.Request{
			FilePath: file.Path,
			FileHash: file.Hash,
			Language: file.Language,
			Source:   source,
			Root:     root,
		})
	case scanner.CPlusPlus:
		return cpp.Extract(cpp.Request{
			FilePath: file.Path,
			FileHash: file.Hash,
			Language: file.Language,
			Source:   source,
			Root:     root,
		})
	case scanner.Rust:
		return rust.Extract(rust.Request{
			FilePath: file.Path,
			FileHash: file.Hash,
			Language: file.Language,
			Source:   source,
			Root:     root,
		})
	case scanner.PHP:
		return php.Extract(php.Request{
			FilePath: file.Path,
			FileHash: file.Hash,
			Language: file.Language,
			Source:   source,
			Root:     root,
		})
	case scanner.Dart:
		return dart.Extract(dart.Request{
			FilePath: file.Path,
			FileHash: file.Hash,
			Language: file.Language,
			Source:   source,
			Root:     root,
		})
	case scanner.Swift:
		return swift.Extract(swift.Request{
			FilePath: file.Path,
			FileHash: file.Hash,
			Language: file.Language,
			Source:   source,
			Root:     root,
		})
	case scanner.Ruby:
		return ruby.Extract(ruby.Request{
			FilePath: file.Path,
			FileHash: file.Hash,
			Language: file.Language,
			Source:   source,
			Root:     root,
		})
	default:
		return tsjs.Extract(tsjs.Request{
			FilePath: file.Path,
			FileHash: file.Hash,
			Language: file.Language,
			Source:   source,
			Root:     root,
		})
	}
}

func runPhase[T any](ctx context.Context, metrics *Metrics, onEvent func(Event), phase PhaseName, run func() (T, error)) (T, error) {
	var zero T
	if err := ctx.Err(); err != nil {
		return zero, err
	}
	emit(onEvent, Event{Kind: EventPhaseStart, Phase: phase})
	start := time.Now()
	value, err := run()
	metrics.Phases = append(metrics.Phases, PhaseMetric{Name: phase, Duration: time.Since(start)})
	metrics.Memory.MaxObservedSys = maxUint64(metrics.Memory.MaxObservedSys, currentSys())
	emit(onEvent, Event{Kind: EventPhaseDone, Phase: phase})
	if err != nil {
		return zero, fmt.Errorf("%s phase: %w", phase, err)
	}
	return value, nil
}

func emit(onEvent func(Event), event Event) {
	if onEvent != nil {
		onEvent(event)
	}
}

func currentAlloc() uint64 {
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)
	return stats.Alloc
}

func currentSys() uint64 {
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)
	return stats.Sys
}

func maxUint64(left uint64, right uint64) uint64 {
	if left > right {
		return left
	}
	return right
}
