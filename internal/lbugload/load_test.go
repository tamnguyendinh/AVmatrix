package lbugload

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tamnguyendinh/avmatrix-go/internal/graph"
	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

func TestLoadCSVExportUsesCopyForSupportedNodeAndRelationshipPairs(t *testing.T) {
	g := graph.New()
	g.AddNode(graph.Node{ID: "File:src/app.ts", Label: scopeir.NodeFile, Properties: graph.NodeProperties{"name": "app.ts", "filePath": "src/app.ts"}})
	g.AddNode(graph.Node{ID: "Function:doWork", Label: scopeir.NodeFunction, Properties: graph.NodeProperties{"name": "doWork", "filePath": "src/app.ts", "startLine": 1, "endLine": 2}})
	g.AddNode(graph.Node{ID: "Variable:config", Label: scopeir.NodeVariable, Properties: graph.NodeProperties{"name": "config", "filePath": "src/app.ts", "startLine": 3, "endLine": 3}})
	g.AddNode(graph.Node{ID: "TypeAlias:Handler", Label: scopeir.NodeTypeAlias, Properties: graph.NodeProperties{"name": "Handler", "filePath": "src/app.ts", "startLine": 4, "endLine": 4}})
	g.AddNode(graph.Node{ID: "Method:Handler.Run", Label: scopeir.NodeMethod, Properties: graph.NodeProperties{"name": "Handler.Run", "filePath": "src/app.ts", "startLine": 5, "endLine": 7, "parameterCount": 1, "returnType": "void"}})
	g.AddRelationship(graph.Relationship{ID: "rel:file-function", SourceID: "File:src/app.ts", TargetID: "Function:doWork", Type: graph.RelDefines, Confidence: 0.8, Reason: "copy path", ResolutionSource: "resolver", FileHash: "hash-a"})
	g.AddRelationship(graph.Relationship{ID: "rel:file-variable", SourceID: "File:src/app.ts", TargetID: "Variable:config", Type: graph.RelDefines, Confidence: 0.9, Reason: "second copy path", ResolutionSource: "resolver", FileHash: "hash-b"})
	g.AddRelationship(graph.Relationship{ID: "rel:typealias-method", SourceID: "TypeAlias:Handler", TargetID: "Method:Handler.Run", Type: graph.RelHasMethod, Confidence: 1, Reason: "type alias method copy path", ResolutionSource: "resolver", FileHash: "hash-c"})

	export, err := ExportGraphCSVs(g, filepath.Join(t.TempDir(), "csv"))
	if err != nil {
		t.Fatalf("ExportGraphCSVs() error = %v", err)
	}
	if len(export.RelationshipPairFiles) != 3 {
		t.Fatalf("RelationshipPairFiles len = %d, want 3", len(export.RelationshipPairFiles))
	}

	runner := &recordingRunner{}
	result, err := LoadCSVExport(runner, export)
	if err != nil {
		t.Fatalf("LoadCSVExport() error = %v", err)
	}
	if result.NodeCopyCount != 5 {
		t.Fatalf("NodeCopyCount = %d, want 5", result.NodeCopyCount)
	}
	if result.RelationshipCopyCount != 3 {
		t.Fatalf("RelationshipCopyCount = %d, want 3", result.RelationshipCopyCount)
	}
	if result.FallbackInsertCount != 0 || result.FallbackInsertFailures != 0 {
		t.Fatalf("fallback counts = inserted %d failed %d, want no fallback on supported COPY path", result.FallbackInsertCount, result.FallbackInsertFailures)
	}
	if len(result.Warnings) != 0 {
		t.Fatalf("warnings = %#v, want none on supported COPY path", result.Warnings)
	}

	joined := strings.Join(runner.queries, "\n")
	for _, want := range []string{
		`COPY File(id, name, filePath, content)`,
		`COPY Function(id, name, filePath, startLine, endLine, isExported, content, description)`,
		`COPY Method(id, name, filePath, startLine, endLine, isExported, content, description, parameterCount, returnType)`,
		`COPY ` + "`TypeAlias`" + `(id, name, filePath, startLine, endLine, content, description)`,
		`COPY ` + "`Variable`" + `(id, name, filePath, startLine, endLine, content, description)`,
		`from="TypeAlias", to="Method"`,
		`COPY CodeRelation FROM`,
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("queries missing %q:\n%s", want, joined)
		}
	}
	if strings.Contains(joined, `CREATE (a)-[:CodeRelation`) {
		t.Fatalf("fallback insert must not run on supported COPY path:\n%s", joined)
	}
}

func TestLoadCSVExportReportsDiagnosticFallbackOnlyForSchemaOrCopyGaps(t *testing.T) {
	g := graph.New()
	g.AddNode(graph.Node{ID: "File:src/app.ts", Label: scopeir.NodeFile, Properties: graph.NodeProperties{"name": "app.ts", "filePath": "src/app.ts"}})
	g.AddNode(graph.Node{ID: "Function:doWork", Label: scopeir.NodeFunction, Properties: graph.NodeProperties{"name": "doWork", "filePath": "src/app.ts", "startLine": 1, "endLine": 2}})
	g.AddNode(graph.Node{ID: "comm_api", Label: scopeir.NodeCommunity, Properties: graph.NodeProperties{"name": "API"}})
	g.AddRelationship(graph.Relationship{ID: "rel:file-function", SourceID: "File:src/app.ts", TargetID: "Function:doWork", Type: graph.RelDefines, Confidence: 0.8, Reason: "copy failure diagnostic", ResolutionSource: "resolver", FileHash: "hash-a"})
	g.AddRelationship(graph.Relationship{ID: "rel:file-community", SourceID: "File:src/app.ts", TargetID: "comm_api", Type: graph.RelMemberOf, Confidence: 1, Reason: "schema gap diagnostic", ResolutionSource: "resolver", FileHash: "hash-b"})

	export, err := ExportGraphCSVs(g, filepath.Join(t.TempDir(), "csv"))
	if err != nil {
		t.Fatalf("ExportGraphCSVs() error = %v", err)
	}
	if len(export.RelationshipPairFiles) != 2 {
		t.Fatalf("RelationshipPairFiles len = %d, want 2", len(export.RelationshipPairFiles))
	}

	runner := &recordingRunner{failContains: "rel_File_Function.csv"}
	result, err := LoadCSVExportWithOptions(runner, export, LoadOptions{
		AllowRelationshipFallback:     true,
		AllowCopyRetryWithIgnoreError: true,
	})
	if err != nil {
		t.Fatalf("LoadCSVExport() error = %v", err)
	}
	if result.RelationshipCopyCount != 1 {
		t.Fatalf("RelationshipCopyCount = %d, want 1", result.RelationshipCopyCount)
	}
	if result.FallbackInsertCount != 2 || result.FallbackInsertFailures != 0 {
		t.Fatalf("fallback counts = inserted %d failed %d, want diagnostic fallback only", result.FallbackInsertCount, result.FallbackInsertFailures)
	}
	if len(result.Warnings) != 2 {
		t.Fatalf("warnings = %#v, want copy failure and schema gap warnings", result.Warnings)
	}

	joined := strings.Join(runner.queries, "\n")
	if !strings.Contains(joined, `COPY CodeRelation FROM`) {
		t.Fatalf("queries missing relationship COPY:\n%s", joined)
	}
	if !strings.Contains(joined, `CREATE (a)-[:CodeRelation`) {
		t.Fatalf("diagnostic fallback insert did not run for injected gaps:\n%s", joined)
	}
	for _, want := range []string{
		`fileHash: 'hash-a'`,
		`fileHash: 'hash-b'`,
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("queries missing %q:\n%s", want, joined)
		}
	}
}

func TestLoadCSVExportFailsClosedWhenRelationshipCopyFails(t *testing.T) {
	g := graph.New()
	g.AddNode(graph.Node{ID: "File:src/app.ts", Label: scopeir.NodeFile, Properties: graph.NodeProperties{"name": "app.ts", "filePath": "src/app.ts"}})
	g.AddNode(graph.Node{ID: "Function:doWork", Label: scopeir.NodeFunction, Properties: graph.NodeProperties{"name": "doWork", "filePath": "src/app.ts", "startLine": 1, "endLine": 2}})
	g.AddRelationship(graph.Relationship{ID: "rel:file-function", SourceID: "File:src/app.ts", TargetID: "Function:doWork", Type: graph.RelDefines, Confidence: 0.8, Reason: "copy failure must fail closed", ResolutionSource: "resolver", FileHash: "hash-a"})

	export, err := ExportGraphCSVs(g, filepath.Join(t.TempDir(), "csv"))
	if err != nil {
		t.Fatalf("ExportGraphCSVs() error = %v", err)
	}
	runner := &recordingRunner{failContains: "rel_File_Function.csv"}
	result, err := LoadCSVExport(runner, export)
	if err == nil {
		t.Fatalf("LoadCSVExport() error = nil, want fail-closed relationship COPY error")
	}
	if result.FallbackInsertCount != 0 || result.FallbackInsertFailures != 0 {
		t.Fatalf("fallback counts = inserted %d failed %d, want no fallback on normal path", result.FallbackInsertCount, result.FallbackInsertFailures)
	}
	joined := strings.Join(runner.queries, "\n")
	if strings.Contains(joined, "IGNORE_ERRORS=true") {
		t.Fatalf("normal COPY path must not retry with IGNORE_ERRORS:\n%s", joined)
	}
	if strings.Contains(joined, `CREATE (a)-[:CodeRelation`) {
		t.Fatalf("normal COPY path must not run fallback insert:\n%s", joined)
	}
}

func TestLoadCSVExportFailsClosedWhenSchemaPairUnsupported(t *testing.T) {
	g := graph.New()
	g.AddNode(graph.Node{ID: "File:src/app.ts", Label: scopeir.NodeFile, Properties: graph.NodeProperties{"name": "app.ts", "filePath": "src/app.ts"}})
	g.AddNode(graph.Node{ID: "comm_api", Label: scopeir.NodeCommunity, Properties: graph.NodeProperties{"name": "API"}})
	g.AddRelationship(graph.Relationship{ID: "rel:file-community", SourceID: "File:src/app.ts", TargetID: "comm_api", Type: graph.RelMemberOf, Confidence: 1, Reason: "schema gap must fail closed", ResolutionSource: "resolver", FileHash: "hash-b"})

	export, err := ExportGraphCSVs(g, filepath.Join(t.TempDir(), "csv"))
	if err != nil {
		t.Fatalf("ExportGraphCSVs() error = %v", err)
	}
	runner := &recordingRunner{}
	result, err := LoadCSVExport(runner, export)
	if err == nil {
		t.Fatalf("LoadCSVExport() error = nil, want fail-closed unsupported schema pair error")
	}
	if result.FallbackInsertCount != 0 || result.FallbackInsertFailures != 0 {
		t.Fatalf("fallback counts = inserted %d failed %d, want no fallback on normal path", result.FallbackInsertCount, result.FallbackInsertFailures)
	}
	if joined := strings.Join(runner.queries, "\n"); strings.Contains(joined, `CREATE (a)-[:CodeRelation`) {
		t.Fatalf("normal unsupported schema path must not run fallback insert:\n%s", joined)
	}
}

func TestLoadCSVExportUsesTransactionWhenRunnerSupportsIt(t *testing.T) {
	g := graph.New()
	g.AddNode(graph.Node{ID: "File:src/app.ts", Label: scopeir.NodeFile, Properties: graph.NodeProperties{"name": "app.ts", "filePath": "src/app.ts"}})
	g.AddNode(graph.Node{ID: "Function:doWork", Label: scopeir.NodeFunction, Properties: graph.NodeProperties{"name": "doWork", "filePath": "src/app.ts", "startLine": 1, "endLine": 2}})
	g.AddRelationship(graph.Relationship{ID: "rel:file-function", SourceID: "File:src/app.ts", TargetID: "Function:doWork", Type: graph.RelDefines, Confidence: 0.8, Reason: "transaction copy path", ResolutionSource: "resolver", FileHash: "hash-a"})

	export, err := ExportGraphCSVs(g, filepath.Join(t.TempDir(), "csv"))
	if err != nil {
		t.Fatalf("ExportGraphCSVs() error = %v", err)
	}
	runner := &transactionRecordingRunner{}
	result, err := LoadCSVExport(runner, export)
	if err != nil {
		t.Fatalf("LoadCSVExport() error = %v", err)
	}
	if result.NodeCopyCount != 2 || result.RelationshipCopyCount != 1 {
		t.Fatalf("copy counts = nodes %d relationships %d, want 2/1", result.NodeCopyCount, result.RelationshipCopyCount)
	}
	if got := runner.events[0]; got != "BEGIN TRANSACTION" {
		t.Fatalf("first event = %q, want BEGIN TRANSACTION; all events:\n%s", got, strings.Join(runner.events, "\n"))
	}
	if got := runner.events[len(runner.events)-1]; got != "COMMIT" {
		t.Fatalf("last event = %q, want COMMIT; all events:\n%s", got, strings.Join(runner.events, "\n"))
	}
	if strings.Contains(strings.Join(runner.events, "\n"), "ROLLBACK") {
		t.Fatalf("successful load rolled back:\n%s", strings.Join(runner.events, "\n"))
	}
}

func TestLoadCSVExportRollsBackTransactionWhenCopyFails(t *testing.T) {
	g := graph.New()
	g.AddNode(graph.Node{ID: "File:src/app.ts", Label: scopeir.NodeFile, Properties: graph.NodeProperties{"name": "app.ts", "filePath": "src/app.ts"}})
	g.AddNode(graph.Node{ID: "Function:doWork", Label: scopeir.NodeFunction, Properties: graph.NodeProperties{"name": "doWork", "filePath": "src/app.ts", "startLine": 1, "endLine": 2}})
	g.AddRelationship(graph.Relationship{ID: "rel:file-function", SourceID: "File:src/app.ts", TargetID: "Function:doWork", Type: graph.RelDefines, Confidence: 0.8, Reason: "transaction rollback path", ResolutionSource: "resolver", FileHash: "hash-a"})

	export, err := ExportGraphCSVs(g, filepath.Join(t.TempDir(), "csv"))
	if err != nil {
		t.Fatalf("ExportGraphCSVs() error = %v", err)
	}
	runner := &transactionRecordingRunner{failContains: "rel_File_Function.csv"}
	result, err := LoadCSVExport(runner, export)
	if err == nil {
		t.Fatalf("LoadCSVExport() error = nil, want relationship COPY failure")
	}
	if result.FallbackInsertCount != 0 || result.FallbackInsertFailures != 0 {
		t.Fatalf("fallback counts = inserted %d failed %d, want no fallback on rollback path", result.FallbackInsertCount, result.FallbackInsertFailures)
	}
	joined := strings.Join(runner.events, "\n")
	if !strings.Contains(joined, "ROLLBACK") {
		t.Fatalf("failed transaction did not roll back:\n%s", joined)
	}
	if strings.Contains(joined, "COMMIT") {
		t.Fatalf("failed transaction committed:\n%s", joined)
	}
}

func TestDiagnosticFallbackReturnsErrorWhenAnyInsertFails(t *testing.T) {
	g := graph.New()
	g.AddNode(graph.Node{ID: "File:src/app.ts", Label: scopeir.NodeFile, Properties: graph.NodeProperties{"name": "app.ts", "filePath": "src/app.ts"}})
	g.AddNode(graph.Node{ID: "comm_api", Label: scopeir.NodeCommunity, Properties: graph.NodeProperties{"name": "API"}})
	g.AddRelationship(graph.Relationship{ID: "rel:file-community", SourceID: "File:src/app.ts", TargetID: "comm_api", Type: graph.RelMemberOf, Confidence: 1, Reason: "diagnostic fallback failure", ResolutionSource: "resolver", FileHash: "hash-b"})

	export, err := ExportGraphCSVs(g, filepath.Join(t.TempDir(), "csv"))
	if err != nil {
		t.Fatalf("ExportGraphCSVs() error = %v", err)
	}
	runner := &recordingRunner{failContains: `CREATE (a)-[:CodeRelation`}
	result, err := LoadCSVExportWithOptions(runner, export, LoadOptions{AllowRelationshipFallback: true})
	if err == nil {
		t.Fatalf("LoadCSVExportWithOptions() error = nil, want fallback insert failure")
	}
	if result.FallbackInsertCount != 0 || result.FallbackInsertFailures != 1 {
		t.Fatalf("fallback counts = inserted %d failed %d, want failed fallback recorded", result.FallbackInsertCount, result.FallbackInsertFailures)
	}
}

func TestCopyQueriesMatchLadybugCSVContract(t *testing.T) {
	query, err := NodeCopyQuery("Method", `C:\tmp\method.csv`)
	if err != nil {
		t.Fatalf("NodeCopyQuery(Method) error = %v", err)
	}
	want := `COPY Method(id, name, filePath, startLine, endLine, isExported, content, description, parameterCount, returnType) FROM "C:/tmp/method.csv" (HEADER=true, ESCAPE='"', DELIM=',', QUOTE='"', PARALLEL=false, auto_detect=false)`
	if query != want {
		t.Fatalf("NodeCopyQuery(Method) = %q, want %q", query, want)
	}
	structQuery, err := NodeCopyQuery("Struct", `C:\tmp\struct.csv`)
	if err != nil {
		t.Fatalf("NodeCopyQuery(Struct) error = %v", err)
	}
	if !strings.HasPrefix(structQuery, "COPY `Struct`(") {
		t.Fatalf("Struct COPY query must quote identifier: %q", structQuery)
	}
	relQuery := RelationshipCopyQuery("File", "Function", `C:\tmp\rel_File_Function.csv`)
	if !strings.Contains(relQuery, `from="File", to="Function"`) || !strings.Contains(relQuery, `C:/tmp/rel_File_Function.csv`) {
		t.Fatalf("RelationshipCopyQuery() = %q", relQuery)
	}
}

type recordingRunner struct {
	failContains string
	queries      []string
}

func (r *recordingRunner) Query(query string) error {
	r.queries = append(r.queries, query)
	if r.failContains != "" && strings.Contains(query, r.failContains) {
		return errors.New("injected copy failure")
	}
	return nil
}

type transactionRecordingRunner struct {
	failContains string
	events       []string
}

func (r *transactionRecordingRunner) Query(query string) error {
	r.events = append(r.events, query)
	if r.failContains != "" && strings.Contains(query, r.failContains) {
		return errors.New("injected copy failure")
	}
	return nil
}

func (r *transactionRecordingRunner) BeginLoadTransaction() error {
	r.events = append(r.events, "BEGIN TRANSACTION")
	return nil
}

func (r *transactionRecordingRunner) CommitLoadTransaction() error {
	r.events = append(r.events, "COMMIT")
	return nil
}

func (r *transactionRecordingRunner) RollbackLoadTransaction() error {
	r.events = append(r.events, "ROLLBACK")
	return nil
}
