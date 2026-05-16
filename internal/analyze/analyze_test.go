package analyze

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/tamnguyendinh/avmatrix-go/internal/embeddings"
	"github.com/tamnguyendinh/avmatrix-go/internal/graph"
	"github.com/tamnguyendinh/avmatrix-go/internal/lbugload"
	"github.com/tamnguyendinh/avmatrix-go/internal/parser"
	"github.com/tamnguyendinh/avmatrix-go/internal/repo"
	"github.com/tamnguyendinh/avmatrix-go/internal/scanner"
	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

func TestRunOrchestratesScanParseResolutionWithMetricsProgressAndBenchmark(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "src/main.ts", `
export function leaf() {
  return 1;
}

export function helper() {
  return leaf();
}

export function main() {
  return helper();
}
`)
	writeFile(t, dir, "app/api/users/route.ts", "export async function GET() { return Response.json({ ok: true }) }\n")
	writeFile(t, dir, "src/client.ts", "export function load() { return fetch('/api/users') }\nserver.tool('query', 'Query tool', handler)\n")
	writeFile(t, dir, "src/db.ts", "export async function users() { return prisma.user.findMany() }\n")
	writeFile(t, dir, "src/inherit.ts", "class Base { run(): void {} }\nclass Child extends Base {}\n")
	writeFile(t, dir, "README.md", "# ignored by parser\n")
	writeFile(t, dir, "docs/spec.docx", "fake docx")
	writeFile(t, dir, "docs/ref.pdf", "fake pdf")
	writeFile(t, dir, "data/sheet.xlsx", "fake spreadsheet")
	writeFile(t, dir, "mainframe/main.cbl", `
       IDENTIFICATION DIVISION.
       PROGRAM-ID. MAINPGM.
       PROCEDURE DIVISION.
       MAIN-SECTION SECTION.
       START.
           COPY CUSTREC.
           PERFORM WORK-PARA.
           STOP RUN.
       WORK-PARA.
           DISPLAY 'DONE'.
`)
	writeFile(t, dir, "mainframe/CUSTREC.cpy", "       01 CUST-ID PIC X(10).\n")
	writeFile(t, dir, "mainframe/run.jcl", "//RUNJOB JOB\n//STEP1 EXEC PGM=MAINPGM\n")
	benchmarkPath := filepath.Join(dir, ".tmp", "analyze-benchmark.json")

	var events []Event
	dbRunner := &recordingDBRunner{}
	result, err := Run(context.Background(), dir, Options{
		Parser:             parser.PoolOptions{ParseTimeout: time.Second},
		BenchmarkPath:      benchmarkPath,
		WriteGraphSnapshot: true,
		DBRunner:           dbRunner,
		OnEvent: func(event Event) {
			events = append(events, event)
		},
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.Graph == nil || len(result.Graph.Nodes) == 0 || len(result.Graph.Relationships) == 0 {
		t.Fatalf("Run() did not produce a graph: %#v", result.Graph)
	}
	if result.Metrics.Files.Scanned != 12 || result.Metrics.Files.Parsed != 5 || result.Metrics.Files.Unsupported != 7 {
		t.Fatalf("file metrics = %#v", result.Metrics.Files)
	}
	if len(result.Metrics.Phases) != 14 {
		t.Fatalf("phase metrics len = %d, want 14", len(result.Metrics.Phases))
	}
	assertPhaseOrder(t, result.Metrics.Phases, []PhaseName{
		PhaseScan,
		PhaseStructure,
		PhaseDocuments,
		PhaseCobol,
		PhaseParse,
		PhaseRoutes,
		PhaseTools,
		PhaseORM,
		PhaseCrossFile,
		PhaseResolution,
		PhaseMRO,
		PhaseCommunities,
		PhaseProcesses,
		PhaseDBLoad,
	})
	if result.Metrics.Structure.FolderNodesAdded == 0 || result.Metrics.Structure.ContainsEmitted == 0 {
		t.Fatalf("missing structure metrics: %#v", result.Metrics.Structure)
	}
	if result.Metrics.Documents.MarkdownFiles != 1 || result.Metrics.Documents.WordFiles != 1 || result.Metrics.Documents.PDFFiles != 1 || result.Metrics.Documents.SpreadsheetFiles != 1 {
		t.Fatalf("missing document metrics: %#v", result.Metrics.Documents)
	}
	if result.Metrics.Cobol.Programs != 1 || result.Metrics.Cobol.Copybooks != 1 || result.Metrics.Cobol.JCLProgramLinks != 1 {
		t.Fatalf("missing COBOL metrics: %#v", result.Metrics.Cobol)
	}
	if result.Metrics.Routes.RoutesEmitted != 1 || result.Metrics.Routes.FetchesEmitted != 1 {
		t.Fatalf("missing route metrics: %#v", result.Metrics.Routes)
	}
	if result.Metrics.Tools.ToolsEmitted != 1 || result.Metrics.Tools.HandlesEmitted != 1 {
		t.Fatalf("missing tool metrics: %#v", result.Metrics.Tools)
	}
	if result.Metrics.ORM.QueriesDetected != 1 || result.Metrics.ORM.QueriesEmitted != 1 || result.Metrics.ORM.ModelNodesEmitted != 1 {
		t.Fatalf("missing ORM metrics: %#v", result.Metrics.ORM)
	}
	if result.Metrics.MRO.ClassesAnalyzed == 0 {
		t.Fatalf("missing MRO metrics: %#v", result.Metrics.MRO)
	}
	if result.Metrics.Communities.CommunitiesEmitted == 0 || result.Metrics.Processes.ProcessesEmitted == 0 {
		t.Fatalf("missing graph enrichment metrics: communities=%#v processes=%#v", result.Metrics.Communities, result.Metrics.Processes)
	}
	if result.Metrics.DBLoad.Skipped || result.Metrics.DBLoad.RelationshipCopyCount == 0 || result.Metrics.DBLoad.FallbackInsertCount != 0 {
		t.Fatalf("DB load metrics = %#v", result.Metrics.DBLoad)
	}
	if result.Metrics.DBLoad.NodeRows == 0 || result.Metrics.DBLoad.RelationshipRows == 0 {
		t.Fatalf("DB load row metrics = %#v", result.Metrics.DBLoad)
	}
	if len(dbRunner.queries) == 0 {
		t.Fatal("DB runner was not called")
	}
	if result.Metrics.Memory.StartAllocBytes == 0 || result.Metrics.Memory.MaxObservedSys == 0 {
		t.Fatalf("memory metrics were not recorded: %#v", result.Metrics.Memory)
	}
	if !hasEvent(events, EventPhaseStart, PhaseScan) || !hasEvent(events, EventPhaseDone, PhaseProcesses) {
		t.Fatalf("missing expected progress events: %#v", events)
	}
	if countNodes(result.Graph, scopeir.NodeCommunity) == 0 || countNodes(result.Graph, scopeir.NodeProcess) == 0 {
		t.Fatalf("graph missing enrichment nodes: %#v", result.Graph)
	}
	if countRelationships(result.Graph, graph.RelContains) == 0 || countRelationships(result.Graph, graph.RelMemberOf) == 0 || countRelationships(result.Graph, graph.RelStepInProcess) == 0 {
		t.Fatalf("graph missing enrichment relationships: %#v", result.Graph.RelationshipCountsByType())
	}

	raw, err := os.ReadFile(benchmarkPath)
	if err != nil {
		t.Fatalf("benchmark artifact missing: %v", err)
	}
	if !strings.Contains(string(raw), `"fallbackInsertCount": 0`) {
		t.Fatalf("benchmark artifact must record zero DB fallback inserts:\n%s", raw)
	}
	var metrics Metrics
	if err := json.Unmarshal(raw, &metrics); err != nil {
		t.Fatalf("benchmark artifact is not metrics JSON: %v", err)
	}
	if metrics.Files.Parsed != 5 {
		t.Fatalf("benchmark metrics files = %#v", metrics.Files)
	}
	if metrics.TotalDuration <= 0 || metrics.Memory.EndAllocBytes == 0 {
		t.Fatalf("benchmark final metrics were not recorded: duration=%s memory=%#v", metrics.TotalDuration, metrics.Memory)
	}
	if metrics.Structure.FolderNodesAdded == 0 || metrics.Structure.ContainsEmitted == 0 {
		t.Fatalf("benchmark structure metrics = %#v", metrics.Structure)
	}
	if metrics.Documents.WordFiles != 1 || metrics.Documents.PDFFiles != 1 || metrics.Documents.SpreadsheetFiles != 1 {
		t.Fatalf("benchmark document metrics = %#v", metrics.Documents)
	}
	if metrics.Cobol.Programs != 1 || metrics.Cobol.Copybooks != 1 || metrics.Cobol.JCLProgramLinks != 1 {
		t.Fatalf("benchmark COBOL metrics = %#v", metrics.Cobol)
	}
	if metrics.Routes.RoutesEmitted != 1 || metrics.Routes.FetchesEmitted != 1 {
		t.Fatalf("benchmark route metrics = %#v", metrics.Routes)
	}
	if metrics.Tools.ToolsEmitted != 1 || metrics.Tools.HandlesEmitted != 1 {
		t.Fatalf("benchmark tool metrics = %#v", metrics.Tools)
	}
	if metrics.ORM.QueriesDetected != 1 || metrics.ORM.QueriesEmitted != 1 || metrics.ORM.ModelNodesEmitted != 1 {
		t.Fatalf("benchmark ORM metrics = %#v", metrics.ORM)
	}
	if !metrics.CrossFile.BindingAccumulatorFinalized || metrics.CrossFile.DefinitionsIndexed == 0 {
		t.Fatalf("benchmark cross-file binding metrics = %#v", metrics.CrossFile)
	}
	if metrics.MRO.ClassesAnalyzed == 0 {
		t.Fatalf("benchmark MRO metrics = %#v", metrics.MRO)
	}
	if metrics.Communities.CommunitiesEmitted == 0 || metrics.Processes.ProcessesEmitted == 0 {
		t.Fatalf("benchmark enrichment metrics = communities %#v processes %#v", metrics.Communities, metrics.Processes)
	}
	if metrics.DBLoad.Skipped || metrics.DBLoad.RelationshipCopyCount == 0 || metrics.DBLoad.FallbackInsertCount != 0 {
		t.Fatalf("benchmark DB load metrics = %#v", metrics.DBLoad)
	}
	if result.GraphPath != repo.Paths(result.RepoPath).GraphPath {
		t.Fatalf("GraphPath = %q, want %q", result.GraphPath, repo.Paths(result.RepoPath).GraphPath)
	}
	if _, err := os.Stat(result.GraphPath); err != nil {
		t.Fatalf("graph snapshot missing: %v", err)
	}
	graphRaw, err := os.ReadFile(result.GraphPath)
	if err != nil {
		t.Fatalf("read graph snapshot: %v", err)
	}
	var snapshot graph.Graph
	if err := json.Unmarshal(graphRaw, &snapshot); err != nil {
		t.Fatalf("graph snapshot is not graph JSON: %v", err)
	}
	if len(snapshot.Nodes) != len(result.Graph.Nodes) || len(snapshot.Relationships) != len(result.Graph.Relationships) {
		t.Fatalf("graph snapshot size = %d/%d, want %d/%d", len(snapshot.Nodes), len(snapshot.Relationships), len(result.Graph.Nodes), len(result.Graph.Relationships))
	}
	if _, err := os.Stat(repo.Paths(dir).AnalyzeTempPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("analyze temp path stat error = %v, want cleanup", err)
	}
}

func TestRunCanReleaseScopeIRsAfterResolution(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "src/main.ts", `
export function leaf() {
  return 1;
}

export function main() {
  return leaf();
}
`)

	result, err := Run(context.Background(), dir, Options{
		Parser:                         parser.PoolOptions{ParseTimeout: time.Second},
		DBRunner:                       &recordingDBRunner{},
		ReleaseScopeIRsAfterResolution: true,
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.ScopeIRs != nil {
		t.Fatalf("ScopeIRs retained after release option: %d", len(result.ScopeIRs))
	}
	if result.Metrics.Files.Parsed != 1 {
		t.Fatalf("parsed files = %d, want 1", result.Metrics.Files.Parsed)
	}
	if result.Graph == nil || len(result.Graph.Nodes) == 0 || len(result.Graph.Relationships) == 0 {
		t.Fatalf("release option dropped graph output: %#v", result.Graph)
	}
}

func BenchmarkWriteGraphSnapshot(b *testing.B) {
	g := graph.New()
	for index := 0; index < 2500; index++ {
		id := "node-" + strconv.Itoa(index)
		g.AddNode(graph.Node{
			ID:    id,
			Label: scopeir.NodeFunction,
			Properties: graph.NodeProperties{
				"name":     id,
				"filePath": "src/file-" + strconv.Itoa(index%50) + ".ts",
				"content":  strings.Repeat("export function "+id+"() { return true }\n", 2),
			},
		})
		if index > 0 {
			relID := "rel-" + strconv.Itoa(index)
			g.AddRelationship(graph.Relationship{
				ID:         relID,
				SourceID:   "node-" + strconv.Itoa(index-1),
				TargetID:   id,
				Type:       graph.RelCalls,
				Confidence: 1,
				Reason:     "benchmark",
			})
		}
	}

	dir := b.TempDir()
	path := filepath.Join(dir, "graph.json")
	b.ReportAllocs()
	b.ResetTimer()
	for index := 0; index < b.N; index++ {
		if err := writeGraphSnapshot(path, g); err != nil {
			b.Fatalf("writeGraphSnapshot() error = %v", err)
		}
	}
}

func TestParseFilesRoutesGoFilesToGoProvider(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "service/service.go", `package service

type Repo struct{}

func (r *Repo) Write(value string) error {
	return nil
}
`)

	result, err := parseFiles(context.Background(), dir, []scanner.File{
		{Path: "service/service.go", Hash: "hash-go", Language: scanner.Go},
	}, Options{Parser: parser.PoolOptions{ParseTimeout: time.Second}})
	if err != nil {
		t.Fatalf("parseFiles() error = %v", err)
	}
	if result.Metrics.Parsed != 1 || result.Metrics.Unsupported != 0 || len(result.IRs) != 1 {
		t.Fatalf("parse result = %#v irs=%d", result.Metrics, len(result.IRs))
	}
	ir := result.IRs[0]
	if ir.Language != scanner.Go {
		t.Fatalf("language = %q, want %q", ir.Language, scanner.Go)
	}
	if !containsDefinition(ir, "Repo", scopeir.NodeStruct) || !containsDefinition(ir, "Repo.Write", scopeir.NodeMethod) {
		t.Fatalf("Go provider definitions missing: %#v", ir.Definitions)
	}
}

func TestParseFilesRoutesPythonFilesToPythonProvider(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "service/service.py", `class Service:
    def save(self):
        return True
`)

	result, err := parseFiles(context.Background(), dir, []scanner.File{
		{Path: "service/service.py", Hash: "hash-python", Language: scanner.Python},
	}, Options{Parser: parser.PoolOptions{ParseTimeout: time.Second}})
	if err != nil {
		t.Fatalf("parseFiles() error = %v", err)
	}
	if result.Metrics.Parsed != 1 || result.Metrics.Unsupported != 0 || len(result.IRs) != 1 {
		t.Fatalf("parse result = %#v irs=%d", result.Metrics, len(result.IRs))
	}
	ir := result.IRs[0]
	if ir.Language != scanner.Python {
		t.Fatalf("language = %q, want %q", ir.Language, scanner.Python)
	}
	if !containsDefinition(ir, "Service", scopeir.NodeClass) || !containsDefinition(ir, "Service.save", scopeir.NodeMethod) {
		t.Fatalf("Python provider definitions missing: %#v", ir.Definitions)
	}
}

func TestParseFilesRoutesJavaFilesToJavaProvider(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "src/app/Service.java", `package app;

public class Service {
    public boolean save() {
        return true;
    }
}
`)

	result, err := parseFiles(context.Background(), dir, []scanner.File{
		{Path: "src/app/Service.java", Hash: "hash-java", Language: scanner.Java},
	}, Options{Parser: parser.PoolOptions{ParseTimeout: time.Second}})
	if err != nil {
		t.Fatalf("parseFiles() error = %v", err)
	}
	if result.Metrics.Parsed != 1 || result.Metrics.Unsupported != 0 || len(result.IRs) != 1 {
		t.Fatalf("parse result = %#v irs=%d", result.Metrics, len(result.IRs))
	}
	ir := result.IRs[0]
	if ir.Language != scanner.Java {
		t.Fatalf("language = %q, want %q", ir.Language, scanner.Java)
	}
	if !containsDefinition(ir, "Service", scopeir.NodeClass) || !containsDefinition(ir, "Service.save", scopeir.NodeMethod) {
		t.Fatalf("Java provider definitions missing: %#v", ir.Definitions)
	}
}

func TestParseFilesRoutesKotlinFilesToKotlinProvider(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "src/app/Service.kt", `package app

class Service {
    fun save(): Boolean {
        return true
    }
}
`)

	result, err := parseFiles(context.Background(), dir, []scanner.File{
		{Path: "src/app/Service.kt", Hash: "hash-kotlin", Language: scanner.Kotlin},
	}, Options{Parser: parser.PoolOptions{ParseTimeout: time.Second}})
	if err != nil {
		t.Fatalf("parseFiles() error = %v", err)
	}
	if result.Metrics.Parsed != 1 || result.Metrics.Unsupported != 0 || len(result.IRs) != 1 {
		t.Fatalf("parse result = %#v irs=%d", result.Metrics, len(result.IRs))
	}
	ir := result.IRs[0]
	if ir.Language != scanner.Kotlin {
		t.Fatalf("language = %q, want %q", ir.Language, scanner.Kotlin)
	}
	if !containsDefinition(ir, "Service", scopeir.NodeClass) || !containsDefinition(ir, "Service.save", scopeir.NodeMethod) {
		t.Fatalf("Kotlin provider definitions missing: %#v", ir.Definitions)
	}
}

func TestParseFilesRoutesCFilesToCProvider(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "src/service.c", `int helper(const char *value) {
    return 1;
}
`)

	result, err := parseFiles(context.Background(), dir, []scanner.File{
		{Path: "src/service.c", Hash: "hash-c", Language: scanner.C},
	}, Options{Parser: parser.PoolOptions{ParseTimeout: time.Second}})
	if err != nil {
		t.Fatalf("parseFiles() error = %v", err)
	}
	if result.Metrics.Parsed != 1 || result.Metrics.Unsupported != 0 || len(result.IRs) != 1 {
		t.Fatalf("parse result = %#v irs=%d", result.Metrics, len(result.IRs))
	}
	ir := result.IRs[0]
	if ir.Language != scanner.C {
		t.Fatalf("language = %q, want %q", ir.Language, scanner.C)
	}
	if !containsDefinition(ir, "helper", scopeir.NodeFunction) {
		t.Fatalf("C provider definitions missing: %#v", ir.Definitions)
	}
}

func TestParseFilesRoutesCSharpFilesToCSharpProvider(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "src/Service.cs", `namespace App;

public class Service {
    public bool Save() {
        return true;
    }
}
`)

	result, err := parseFiles(context.Background(), dir, []scanner.File{
		{Path: "src/Service.cs", Hash: "hash-csharp", Language: scanner.CSharp},
	}, Options{Parser: parser.PoolOptions{ParseTimeout: time.Second}})
	if err != nil {
		t.Fatalf("parseFiles() error = %v", err)
	}
	if result.Metrics.Parsed != 1 || result.Metrics.Unsupported != 0 || len(result.IRs) != 1 {
		t.Fatalf("parse result = %#v irs=%d", result.Metrics, len(result.IRs))
	}
	ir := result.IRs[0]
	if ir.Language != scanner.CSharp {
		t.Fatalf("language = %q, want %q", ir.Language, scanner.CSharp)
	}
	if !containsDefinition(ir, "Service", scopeir.NodeClass) || !containsDefinition(ir, "Service.Save", scopeir.NodeMethod) {
		t.Fatalf("C# provider definitions missing: %#v", ir.Definitions)
	}
}

func TestParseFilesRoutesCPPFilesToCPPProvider(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "src/service.cpp", `namespace app {
class Service {
public:
    bool save() {
        return true;
    }
};
}
`)

	result, err := parseFiles(context.Background(), dir, []scanner.File{
		{Path: "src/service.cpp", Hash: "hash-cpp", Language: scanner.CPlusPlus},
	}, Options{Parser: parser.PoolOptions{ParseTimeout: time.Second}})
	if err != nil {
		t.Fatalf("parseFiles() error = %v", err)
	}
	if result.Metrics.Parsed != 1 || result.Metrics.Unsupported != 0 || len(result.IRs) != 1 {
		t.Fatalf("parse result = %#v irs=%d", result.Metrics, len(result.IRs))
	}
	ir := result.IRs[0]
	if ir.Language != scanner.CPlusPlus {
		t.Fatalf("language = %q, want %q", ir.Language, scanner.CPlusPlus)
	}
	if !containsDefinition(ir, "Service", scopeir.NodeClass) || !containsDefinition(ir, "Service.save", scopeir.NodeMethod) {
		t.Fatalf("C++ provider definitions missing: %#v", ir.Definitions)
	}
}

func TestParseFilesRoutesRustFilesToRustProvider(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "src/service.rs", `mod app {
pub struct Service;
impl Service {
    pub fn save(&self) -> bool {
        true
    }
}
}
`)

	result, err := parseFiles(context.Background(), dir, []scanner.File{
		{Path: "src/service.rs", Hash: "hash-rust", Language: scanner.Rust},
	}, Options{Parser: parser.PoolOptions{ParseTimeout: time.Second}})
	if err != nil {
		t.Fatalf("parseFiles() error = %v", err)
	}
	if result.Metrics.Parsed != 1 || result.Metrics.Unsupported != 0 || len(result.IRs) != 1 {
		t.Fatalf("parse result = %#v irs=%d", result.Metrics, len(result.IRs))
	}
	ir := result.IRs[0]
	if ir.Language != scanner.Rust {
		t.Fatalf("language = %q, want %q", ir.Language, scanner.Rust)
	}
	if !containsDefinition(ir, "Service", scopeir.NodeStruct) || !containsDefinition(ir, "Service.save", scopeir.NodeMethod) {
		t.Fatalf("Rust provider definitions missing: %#v", ir.Definitions)
	}
}

func TestParseFilesRoutesPHPFilesToPHPProvider(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "src/Service.php", `<?php
namespace App;

class Service {
    public function save(): bool {
        return true;
    }
}
`)

	result, err := parseFiles(context.Background(), dir, []scanner.File{
		{Path: "src/Service.php", Hash: "hash-php", Language: scanner.PHP},
	}, Options{Parser: parser.PoolOptions{ParseTimeout: time.Second}})
	if err != nil {
		t.Fatalf("parseFiles() error = %v", err)
	}
	if result.Metrics.Parsed != 1 || result.Metrics.Unsupported != 0 || len(result.IRs) != 1 {
		t.Fatalf("parse result = %#v irs=%d", result.Metrics, len(result.IRs))
	}
	ir := result.IRs[0]
	if ir.Language != scanner.PHP {
		t.Fatalf("language = %q, want %q", ir.Language, scanner.PHP)
	}
	if !containsDefinition(ir, "Service", scopeir.NodeClass) || !containsDefinition(ir, "Service.save", scopeir.NodeMethod) {
		t.Fatalf("PHP provider definitions missing: %#v", ir.Definitions)
	}
}

func TestParseFilesRoutesDartFilesToDartProvider(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "lib/service.dart", `class Service {
  bool save() {
    return true;
  }
}
`)

	result, err := parseFiles(context.Background(), dir, []scanner.File{
		{Path: "lib/service.dart", Hash: "hash-dart", Language: scanner.Dart},
	}, Options{Parser: parser.PoolOptions{ParseTimeout: time.Second}})
	if err != nil {
		t.Fatalf("parseFiles() error = %v", err)
	}
	if result.Metrics.Parsed != 1 || result.Metrics.Unsupported != 0 || len(result.IRs) != 1 {
		t.Fatalf("parse result = %#v irs=%d", result.Metrics, len(result.IRs))
	}
	ir := result.IRs[0]
	if ir.Language != scanner.Dart {
		t.Fatalf("language = %q, want %q", ir.Language, scanner.Dart)
	}
	if !containsDefinition(ir, "Service", scopeir.NodeClass) || !containsDefinition(ir, "Service.save", scopeir.NodeMethod) {
		t.Fatalf("Dart provider definitions missing: %#v", ir.Definitions)
	}
}

func TestParseFilesRoutesSwiftFilesToSwiftProvider(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "Sources/App/Service.swift", `class Service {
  func save() -> Bool {
    return true
  }
}
`)

	result, err := parseFiles(context.Background(), dir, []scanner.File{
		{Path: "Sources/App/Service.swift", Hash: "hash-swift", Language: scanner.Swift},
	}, Options{Parser: parser.PoolOptions{ParseTimeout: time.Second}})
	if err != nil {
		t.Fatalf("parseFiles() error = %v", err)
	}
	if result.Metrics.Parsed != 1 || result.Metrics.Unsupported != 0 || len(result.IRs) != 1 {
		t.Fatalf("parse result = %#v irs=%d", result.Metrics, len(result.IRs))
	}
	ir := result.IRs[0]
	if ir.Language != scanner.Swift {
		t.Fatalf("language = %q, want %q", ir.Language, scanner.Swift)
	}
	if !containsDefinition(ir, "Service", scopeir.NodeClass) || !containsDefinition(ir, "Service.save", scopeir.NodeMethod) {
		t.Fatalf("Swift provider definitions missing: %#v", ir.Definitions)
	}
}

func TestParseFilesRoutesRubyFilesToRubyProvider(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "lib/service.rb", `class Service
  def save
    true
  end
end
`)

	result, err := parseFiles(context.Background(), dir, []scanner.File{
		{Path: "lib/service.rb", Hash: "hash-ruby", Language: scanner.Ruby},
	}, Options{Parser: parser.PoolOptions{ParseTimeout: time.Second}})
	if err != nil {
		t.Fatalf("parseFiles() error = %v", err)
	}
	if result.Metrics.Parsed != 1 || result.Metrics.Unsupported != 0 || len(result.IRs) != 1 {
		t.Fatalf("parse result = %#v irs=%d", result.Metrics, len(result.IRs))
	}
	ir := result.IRs[0]
	if ir.Language != scanner.Ruby {
		t.Fatalf("language = %q, want %q", ir.Language, scanner.Ruby)
	}
	if !containsDefinition(ir, "Service", scopeir.NodeClass) || !containsDefinition(ir, "Service.save", scopeir.NodeMethod) {
		t.Fatalf("Ruby provider definitions missing: %#v", ir.Definitions)
	}
}

func TestParseFilesRoutesVueFilesToVueProvider(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "src/Service.vue", `<template><button>{{ label }}</button></template>
<script lang="ts">
class Service {
  save(): boolean {
    return true;
  }
}
</script>
`)

	result, err := parseFiles(context.Background(), dir, []scanner.File{
		{Path: "src/Service.vue", Hash: "hash-vue", Language: scanner.Vue},
	}, Options{Parser: parser.PoolOptions{ParseTimeout: time.Second}})
	if err != nil {
		t.Fatalf("parseFiles() error = %v", err)
	}
	if result.Metrics.Parsed != 1 || result.Metrics.Unsupported != 0 || len(result.IRs) != 1 {
		t.Fatalf("parse result = %#v irs=%d", result.Metrics, len(result.IRs))
	}
	ir := result.IRs[0]
	if ir.Language != scanner.Vue {
		t.Fatalf("language = %q, want %q", ir.Language, scanner.Vue)
	}
	if !containsDefinition(ir, "Service", scopeir.NodeClass) || !containsDefinition(ir, "Service.save", scopeir.NodeMethod) {
		t.Fatalf("Vue provider definitions missing: %#v", ir.Definitions)
	}
}

func TestParseFilesRoutesSvelteFilesToSvelteProvider(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "src/routes/+page.svelte", `<script lang="ts">
class Service {
  save(): boolean {
    return true;
  }
}
</script>
<button>Save</button>
`)

	result, err := parseFiles(context.Background(), dir, []scanner.File{
		{Path: "src/routes/+page.svelte", Hash: "hash-svelte", Language: scanner.Svelte},
	}, Options{Parser: parser.PoolOptions{ParseTimeout: time.Second}})
	if err != nil {
		t.Fatalf("parseFiles() error = %v", err)
	}
	if result.Metrics.Parsed != 1 || result.Metrics.Unsupported != 0 || len(result.IRs) != 1 {
		t.Fatalf("parse result = %#v irs=%d", result.Metrics, len(result.IRs))
	}
	ir := result.IRs[0]
	if ir.Language != scanner.Svelte {
		t.Fatalf("language = %q, want %q", ir.Language, scanner.Svelte)
	}
	if !containsDefinition(ir, "Service", scopeir.NodeClass) || !containsDefinition(ir, "Service.save", scopeir.NodeMethod) {
		t.Fatalf("Svelte provider definitions missing: %#v", ir.Definitions)
	}
}

func TestParseFilesRoutesAstroFilesToAstroProvider(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "src/pages/users.astro", `---
class Service {
  save(): boolean {
    return true;
  }
}
---
<div>Users</div>
`)

	result, err := parseFiles(context.Background(), dir, []scanner.File{
		{Path: "src/pages/users.astro", Hash: "hash-astro", Language: scanner.Astro},
	}, Options{Parser: parser.PoolOptions{ParseTimeout: time.Second}})
	if err != nil {
		t.Fatalf("parseFiles() error = %v", err)
	}
	if result.Metrics.Parsed != 1 || result.Metrics.Unsupported != 0 || len(result.IRs) != 1 {
		t.Fatalf("parse result = %#v irs=%d", result.Metrics, len(result.IRs))
	}
	ir := result.IRs[0]
	if ir.Language != scanner.Astro {
		t.Fatalf("language = %q, want %q", ir.Language, scanner.Astro)
	}
	if !containsDefinition(ir, "Service", scopeir.NodeClass) || !containsDefinition(ir, "Service.save", scopeir.NodeMethod) {
		t.Fatalf("Astro provider definitions missing: %#v", ir.Definitions)
	}
}

func TestRunHonorsCanceledContextBeforePhaseWork(t *testing.T) {
	dir := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := Run(ctx, dir, Options{})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Run() error = %v, want context.Canceled", err)
	}
}

func TestRunForceRemovesPreviousLbugOutput(t *testing.T) {
	dir := t.TempDir()
	resolvedDir, err := repo.ResolveAnalyzePath(dir)
	if err != nil {
		t.Fatalf("ResolveAnalyzePath() error = %v", err)
	}
	writeFile(t, dir, "src/main.ts", "export function main() { return 1; }\n")
	stalePath := filepath.Join(repo.Paths(resolvedDir).LbugPath, "stale.txt")
	staleGraphPath := repo.Paths(resolvedDir).GraphPath
	if err := os.MkdirAll(filepath.Dir(stalePath), 0o755); err != nil {
		t.Fatalf("mkdir stale lbug: %v", err)
	}
	if err := os.WriteFile(stalePath, []byte("stale"), 0o644); err != nil {
		t.Fatalf("write stale lbug: %v", err)
	}
	if err := os.WriteFile(staleGraphPath, []byte("stale graph"), 0o644); err != nil {
		t.Fatalf("write stale graph: %v", err)
	}

	if _, err := Run(context.Background(), dir, Options{Force: true}); err != nil {
		t.Fatalf("Run(force) error = %v", err)
	}
	if _, err := os.Stat(stalePath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("stale lbug stat error = %v, want removed", err)
	}
	if _, err := os.Stat(staleGraphPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("stale graph stat error = %v, want removed", err)
	}
}

func TestRunUsesDBRunnerFactoryWhenRunnerIsNotInjected(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "src/main.ts", "export function main() { return 1; }\n")
	factory := &recordingDBRunnerFactory{}

	result, err := Run(context.Background(), dir, Options{DBRunnerFactory: factory.Factory})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.Metrics.DBLoad.Skipped || result.Metrics.DBLoad.NodeCopyCount == 0 {
		t.Fatalf("DB load metrics = %#v", result.Metrics.DBLoad)
	}
	if factory.path == "" || factory.path != repo.Paths(result.RepoPath).LbugPath {
		t.Fatalf("factory path = %q, want %q", factory.path, repo.Paths(result.RepoPath).LbugPath)
	}
	if !factory.closed {
		t.Fatal("factory close was not called")
	}
}

func TestRunSkipsDBLoadWhenFactoryReturnsNilRunner(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "src/main.ts", "export function main() { return 1; }\n")
	result, err := Run(context.Background(), dir, Options{
		DBRunnerFactory: func(repo.StoragePaths) (lbugload.QueryRunner, func() error, error) {
			return nil, nil, nil
		},
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !result.Metrics.DBLoad.Skipped || result.Metrics.DBLoad.SkipReason != "query runner factory returned nil" {
		t.Fatalf("DB load metrics = %#v", result.Metrics.DBLoad)
	}
}

func TestRunExecutesEmbeddingsWhenEnabledWithInjectedEmbedder(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "src/main.ts", "export function main() { return 1; }\n")
	dbRunner := &recordingDBRunner{}

	result, err := Run(context.Background(), dir, Options{
		DBRunner:        dbRunner,
		Embeddings:      true,
		EmbeddingConfig: embeddings.Config{Dimensions: 3},
		EmbedderFactory: func(config embeddings.Config) (embeddings.Embedder, error) {
			if config.Dimensions != 3 {
				t.Fatalf("embedding dimensions = %d, want 3", config.Dimensions)
			}
			return &recordingEmbedder{dimensions: config.Dimensions}, nil
		},
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.Metrics.Embeddings.TotalNodes == 0 || result.Metrics.Embeddings.InsertQueries == 0 || !result.Metrics.Embeddings.VectorIndexCreated {
		t.Fatalf("embedding metrics = %#v", result.Metrics.Embeddings)
	}
	if !hasRecordedQuery(dbRunner.queries, "CREATE (e:CodeEmbedding") || !hasRecordedQuery(dbRunner.queries, "CALL CREATE_VECTOR_INDEX") {
		t.Fatalf("embedding queries missing: %#v", dbRunner.queries)
	}
	if len(result.Metrics.Phases) == 0 || result.Metrics.Phases[len(result.Metrics.Phases)-1].Name != PhaseEmbeddings {
		t.Fatalf("last phase = %#v, want embeddings", result.Metrics.Phases)
	}
}

func TestRunRejectsEmbeddingsWhenDBRunnerIsUnavailable(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "src/main.ts", "export function main() { return 1; }\n")

	_, err := Run(context.Background(), dir, Options{
		Embeddings:      true,
		EmbeddingConfig: embeddings.Config{Dimensions: 3},
		EmbedderFactory: func(config embeddings.Config) (embeddings.Embedder, error) {
			return &recordingEmbedder{dimensions: config.Dimensions}, nil
		},
	})
	if err == nil || !strings.Contains(err.Error(), "embeddings require a DB query runner") {
		t.Fatalf("Run() error = %v, want DB runner requirement", err)
	}
}

func TestRunRejectsConcurrentWriterLock(t *testing.T) {
	dir := t.TempDir()
	resolvedDir, err := repo.ResolveAnalyzePath(dir)
	if err != nil {
		t.Fatalf("ResolveAnalyzePath() error = %v", err)
	}
	lock, err := repo.AcquireStorageLock(repo.Paths(resolvedDir).AnalyzeLockPath)
	if err != nil {
		t.Fatalf("AcquireStorageLock() error = %v", err)
	}
	defer lock.Release()

	_, err = Run(context.Background(), dir, Options{})
	if !errors.Is(err, repo.ErrLockHeld) {
		t.Fatalf("Run() error = %v, want ErrLockHeld", err)
	}
}

func hasEvent(events []Event, kind EventKind, phase PhaseName) bool {
	for _, event := range events {
		if event.Kind == kind && event.Phase == phase {
			return true
		}
	}
	return false
}

func assertPhaseOrder(t *testing.T, phases []PhaseMetric, want []PhaseName) {
	t.Helper()
	if len(phases) != len(want) {
		t.Fatalf("phase count = %d, want %d", len(phases), len(want))
	}
	for index, phase := range phases {
		if phase.Name != want[index] {
			t.Fatalf("phase[%d] = %s, want %s; phases=%#v", index, phase.Name, want[index], phases)
		}
	}
}

func countNodes(g *graph.Graph, label scopeir.NodeLabel) int {
	count := 0
	for _, node := range g.Nodes {
		if node.Label == label {
			count++
		}
	}
	return count
}

func countRelationships(g *graph.Graph, relType graph.RelationshipType) int {
	count := 0
	for _, rel := range g.Relationships {
		if rel.Type == relType {
			count++
		}
	}
	return count
}

func containsDefinition(ir scopeir.ScopeIR, name string, label scopeir.NodeLabel) bool {
	for _, def := range ir.Definitions {
		if (def.Name == name || def.QualifiedName == name) && def.Label == label {
			return true
		}
	}
	return false
}

func writeFile(t *testing.T, root string, rel string, contents string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", rel, err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("write %s: %v", rel, err)
	}
}

type recordingDBRunner struct {
	queries []string
}

func (r *recordingDBRunner) Query(query string) error {
	r.queries = append(r.queries, query)
	return nil
}

type recordingEmbedder struct {
	dimensions int
}

func (r *recordingEmbedder) Embed(_ context.Context, texts []string) ([][]float32, error) {
	vectors := make([][]float32, 0, len(texts))
	for range texts {
		vector := make([]float32, r.dimensions)
		for index := range vector {
			vector[index] = float32(index + 1)
		}
		vectors = append(vectors, vector)
	}
	return vectors, nil
}

func hasRecordedQuery(queries []string, needle string) bool {
	for _, query := range queries {
		if strings.Contains(query, needle) {
			return true
		}
	}
	return false
}

type recordingDBRunnerFactory struct {
	runner recordingDBRunner
	path   string
	closed bool
}

func (f *recordingDBRunnerFactory) Factory(paths repo.StoragePaths) (lbugload.QueryRunner, func() error, error) {
	f.path = paths.LbugPath
	return &f.runner, func() error {
		f.closed = true
		return nil
	}, nil
}
