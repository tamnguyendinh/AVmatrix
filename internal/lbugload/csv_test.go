package lbugload

import (
	"encoding/csv"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/tamnguyendinh/avmatrix-go/internal/graph"
	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

func TestExportGraphCSVsWritesNodeRelationshipAndSplitContracts(t *testing.T) {
	step := 3
	g := graph.New()
	g.AddNode(graph.Node{ID: "File:src/app.ts", Label: scopeir.NodeFile, Properties: graph.NodeProperties{
		"name": "app.ts", "filePath": "src/app.ts", "content": "export const quoted = \"value\"",
	}})
	g.AddNode(graph.Node{ID: "Function:doWork", Label: scopeir.NodeFunction, Properties: graph.NodeProperties{
		"name": "doWork", "filePath": "src/app.ts", "startLine": 7, "endLine": 12, "isExported": true, "content": "function doWork() {}", "description": "main worker",
	}})
	g.AddNode(graph.Node{ID: "comm_api", Label: scopeir.NodeCommunity, Properties: graph.NodeProperties{
		"name": "API", "heuristicLabel": "api", "keywords": []string{"route", "worker's"}, "symbolCount": 2,
	}})
	g.AddRelationship(graph.Relationship{
		ID:               "rel:file-function",
		SourceID:         "File:src/app.ts",
		TargetID:         "Function:doWork",
		Type:             graph.RelDefines,
		Confidence:       0.9,
		Reason:           "declares\nsymbol",
		Step:             &step,
		ResolutionSource: "resolver",
		FileHash:         "hash-1",
		Evidence:         []graph.Evidence{{Kind: "definition", Weight: 0.7, Note: "direct"}},
	})

	export, err := ExportGraphCSVs(g, filepath.Join(t.TempDir(), "csv"))
	if err != nil {
		t.Fatalf("ExportGraphCSVs() error = %v", err)
	}
	if export.RelationshipRows != 1 {
		t.Fatalf("RelationshipRows = %d, want 1", export.RelationshipRows)
	}
	if export.Metrics.RowsByTable["File"] != 1 || export.Metrics.RowsByTable["Function"] != 1 || export.Metrics.RowsByTable["Community"] != 1 {
		t.Fatalf("unexpected node row metrics: %#v", export.Metrics.RowsByTable)
	}

	fileRows := readCSV(t, filepath.Join(export.CSVDir, "file.csv"))
	wantFileHeader := []string{"id", "name", "filePath", "content"}
	if !reflect.DeepEqual(fileRows[0], wantFileHeader) {
		t.Fatalf("file.csv header = %#v, want %#v", fileRows[0], wantFileHeader)
	}
	if fileRows[1][3] != `export const quoted = "value"` {
		t.Fatalf("file content row not preserved: %#v", fileRows[1])
	}

	functionRows := readCSV(t, filepath.Join(export.CSVDir, "function.csv"))
	wantFunctionHeader := []string{"id", "name", "filePath", "startLine", "endLine", "isExported", "content", "description"}
	if !reflect.DeepEqual(functionRows[0], wantFunctionHeader) {
		t.Fatalf("function.csv header = %#v, want %#v", functionRows[0], wantFunctionHeader)
	}
	if functionRows[1][5] != "true" {
		t.Fatalf("function isExported = %q, want true", functionRows[1][5])
	}

	relationshipRows := readCSV(t, export.RelationshipCSVPath)
	if strings.Join(relationshipRows[0], ",") != RelationshipCSVHeader {
		t.Fatalf("relationship header = %#v", relationshipRows[0])
	}
	if relationshipRows[1][6] != "resolver" || relationshipRows[1][8] != "hash-1" {
		t.Fatalf("relationship audit columns not preserved: %#v", relationshipRows[1])
	}
	if !strings.Contains(relationshipRows[1][7], `"kind":"definition"`) {
		t.Fatalf("relationship evidence JSON not preserved: %q", relationshipRows[1][7])
	}

	pairRows := readCSV(t, filepath.Join(export.CSVDir, "rel_File_Function.csv"))
	if len(pairRows) != 2 {
		t.Fatalf("rel_File_Function.csv rows = %d, want 2 including header", len(pairRows))
	}
	if len(export.RelationshipPairFiles) != 1 || !export.RelationshipPairFiles[0].CopySupported {
		t.Fatalf("relationship pair metadata = %#v", export.RelationshipPairFiles)
	}
}

func TestCSVWriterEscapesFieldsAndSanitizesText(t *testing.T) {
	path := filepath.Join(t.TempDir(), "escape.csv")
	writer, err := newCSVFileWriter(path, []string{"value"})
	if err != nil {
		t.Fatalf("newCSVFileWriter() error = %v", err)
	}
	for _, value := range []string{
		"hello",
		`say "hello"`,
		"a,b,c",
		"line1\nline2",
		`path\to\file`,
		"function foo() {\n  return \"bar\";\n}",
	} {
		if err := writer.Write([]string{value}); err != nil {
			t.Fatalf("Write(%q) error = %v", value, err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read escaped csv: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		`"hello"`,
		`"say ""hello"""`,
		`"a,b,c"`,
		"\"line1\nline2\"",
		`path\to\file`,
		`return ""bar"";`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("escaped CSV missing %q:\n%s", want, text)
		}
	}
}

func TestCSVScalarFormattingAndSanitizationContract(t *testing.T) {
	props := graph.NodeProperties{
		"string":     "line1\r\nline2\rline3\x00\x01\tok",
		"empty":      "",
		"number":     42,
		"float":      3.14,
		"boolTrue":   true,
		"boolString": "true",
		"keywords":   []string{"auth", "pass,word", `path\to\file`, "worker's"},
	}
	if got := stringProp(props, "string", "fallback"); got != "line1\nline2\nline3\tok" {
		t.Fatalf("stringProp sanitized = %q", got)
	}
	if got := stringProp(props, "empty", "fallback"); got != "fallback" {
		t.Fatalf("empty string fallback = %q", got)
	}
	if got := intProp(props, "missing", -1); got != "-1" {
		t.Fatalf("missing int fallback = %q", got)
	}
	if got := intProp(props, "number", -1); got != "42" {
		t.Fatalf("intProp = %q", got)
	}
	if got := floatProp(props, "float", -1); got != "3.14" {
		t.Fatalf("floatProp = %q", got)
	}
	if got := boolProp(props, "boolTrue"); got != "true" {
		t.Fatalf("boolProp(true) = %q", got)
	}
	if got := boolProp(props, "boolString"); got != "true" {
		t.Fatalf("boolProp(string true) = %q", got)
	}
	keywords := arrayLiteral(props["keywords"])
	for _, want := range []string{`'auth'`, `'pass,word'`, `'path\\to\\file'`, `'worker''s'`} {
		if !strings.Contains(keywords, want) {
			t.Fatalf("arrayLiteral missing %q in %q", want, keywords)
		}
	}
	if got := sanitizeString("hello\uFFFEworld\uFFFF"); got != "helloworld" {
		t.Fatalf("sanitizeString noncharacters = %q", got)
	}
	if got := sanitizeString(string([]byte{'o', 'k', 0xff, 'x'})); got != "okx" {
		t.Fatalf("sanitizeString invalid UTF-8 = %q", got)
	}
}

func TestExportGraphCSVsSplitsRelationshipPairsAndSkipsUnknownEndpoints(t *testing.T) {
	g := graph.New()
	g.AddNode(graph.Node{ID: "Function:a", Label: scopeir.NodeFunction, Properties: graph.NodeProperties{"name": "a", "filePath": "src/a.ts"}})
	g.AddNode(graph.Node{ID: "Function:c", Label: scopeir.NodeFunction, Properties: graph.NodeProperties{"name": "c", "filePath": "src/c.ts"}})
	g.AddNode(graph.Node{ID: "Class:b", Label: scopeir.NodeClass, Properties: graph.NodeProperties{"name": "b", "filePath": "src/b.ts"}})
	g.AddNode(graph.Node{ID: "Class:d", Label: scopeir.NodeClass, Properties: graph.NodeProperties{"name": "d", "filePath": "src/d.ts"}})
	g.AddNode(graph.Node{ID: "File:e", Label: scopeir.NodeFile, Properties: graph.NodeProperties{"name": "e.ts", "filePath": "src/e.ts"}})
	g.AddNode(graph.Node{ID: "Method:f", Label: scopeir.NodeMethod, Properties: graph.NodeProperties{"name": "f", "filePath": "src/f.ts"}})
	g.AddRelationship(graph.Relationship{ID: "rel:a-b", SourceID: "Function:a", TargetID: "Class:b", Type: graph.RelCalls})
	g.AddRelationship(graph.Relationship{ID: "rel:c-d", SourceID: "Function:c", TargetID: "Class:d", Type: graph.RelCalls})
	g.AddRelationship(graph.Relationship{ID: "rel:e-f", SourceID: "File:e", TargetID: "Method:f", Type: graph.RelDefines})
	g.AddRelationship(graph.Relationship{ID: "rel:unknown", SourceID: "Unknown:x", TargetID: "Class:d", Type: graph.RelCalls})

	export, err := ExportGraphCSVs(g, filepath.Join(t.TempDir(), "csv"))
	if err != nil {
		t.Fatalf("ExportGraphCSVs() error = %v", err)
	}
	if export.RelationshipRows != 4 || export.Metrics.SkippedRelationships != 1 {
		t.Fatalf("relationship metrics = rows %d skipped %d", export.RelationshipRows, export.Metrics.SkippedRelationships)
	}
	pairs := map[string]RelationshipPairCSV{}
	for _, pair := range export.RelationshipPairFiles {
		pairs[pairKey(pair.From, pair.To)] = pair
	}
	if pairs["Function|Class"].Rows != 2 {
		t.Fatalf("Function|Class rows = %#v, want 2", pairs["Function|Class"])
	}
	if pairs["File|Method"].Rows != 1 {
		t.Fatalf("File|Method rows = %#v, want 1", pairs["File|Method"])
	}
	if _, ok := pairs["Unknown|Class"]; ok {
		t.Fatalf("unexpected pair file for unknown endpoint: %#v", pairs)
	}
}

func TestExportGraphCSVsHandlesEmptyGraphAndHeaderOnlyRelations(t *testing.T) {
	export, err := ExportGraphCSVs(graph.New(), filepath.Join(t.TempDir(), "csv"))
	if err != nil {
		t.Fatalf("ExportGraphCSVs(empty) error = %v", err)
	}
	if len(export.NodeFiles) != 0 || export.RelationshipRows != 0 || len(export.RelationshipPairFiles) != 0 {
		t.Fatalf("empty export = %#v", export)
	}
	rows := readCSV(t, export.RelationshipCSVPath)
	if len(rows) != 1 || strings.Join(rows[0], ",") != RelationshipCSVHeader {
		t.Fatalf("relationship header rows = %#v", rows)
	}
}

func readCSV(t *testing.T, path string) [][]string {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("open csv %s: %v", path, err)
	}
	defer file.Close()
	rows, err := csv.NewReader(file).ReadAll()
	if err != nil {
		t.Fatalf("read csv %s: %v", path, err)
	}
	return rows
}
