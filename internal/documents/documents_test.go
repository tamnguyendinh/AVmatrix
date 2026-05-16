package documents

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tamnguyendinh/avmatrix-go/internal/graph"
	"github.com/tamnguyendinh/avmatrix-go/internal/scanner"
	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

func TestApplyEmitsMarkdownSectionsHierarchyAndLocalLinks(t *testing.T) {
	dir := t.TempDir()
	writeDocumentTestFile(t, dir, "README.md", "# Intro\nSee [setup](docs/setup.md#install).\n\n## Usage\nBody\n")
	writeDocumentTestFile(t, dir, "docs/setup.md", "# Setup\n")

	g := graph.New()
	addFileNode(g, "README.md")
	addFileNode(g, "docs/setup.md")

	result, err := Apply(g, dir, []scanner.File{{Path: "README.md", Language: scanner.Markdown}, {Path: "docs/setup.md", Language: scanner.Markdown}})
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if result.Metrics.MarkdownFiles != 2 || result.Metrics.Sections != 3 || result.Metrics.Links != 1 {
		t.Fatalf("metrics = %#v", result.Metrics)
	}
	introID := "Section:README.md:L1:Intro"
	usageID := "Section:README.md:L4:Usage"
	requireDocumentNode(t, g, introID, scopeir.NodeSection)
	requireDocumentNode(t, g, usageID, scopeir.NodeSection)
	requireDocumentRelationship(t, g, graph.RelContains, "File:README.md", introID)
	requireDocumentRelationship(t, g, graph.RelContains, introID, usageID)
	requireDocumentRelationship(t, g, graph.RelImports, "File:README.md", "File:docs/setup.md")
	readme, _ := g.GetNode("File:README.md")
	if readme.Properties["documentKind"] != "markdown" || readme.Properties["binary"] != false {
		t.Fatalf("markdown file metadata = %#v", readme.Properties)
	}
}

func TestApplyMarksWordPDFAndSpreadsheetFilesWithoutTextParsing(t *testing.T) {
	dir := t.TempDir()
	for _, rel := range []string{"docs/spec.doc", "docs/report.docx", "docs/ref.pdf", "data/book.xlsx", "data/macro.xlsm", "data/table.csv"} {
		writeDocumentTestFile(t, dir, rel, "fake-binary")
	}

	g := graph.New()
	files := []scanner.File{
		{Path: "docs/spec.doc", Language: scanner.Word},
		{Path: "docs/report.docx", Language: scanner.Word},
		{Path: "docs/ref.pdf", Language: scanner.PDF},
		{Path: "data/book.xlsx", Language: scanner.Spreadsheet},
		{Path: "data/macro.xlsm", Language: scanner.Spreadsheet},
		{Path: "data/table.csv", Language: scanner.Spreadsheet},
	}
	for _, file := range files {
		addFileNode(g, file.Path)
	}

	result, err := Apply(g, dir, files)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if result.Metrics.WordFiles != 2 || result.Metrics.PDFFiles != 1 || result.Metrics.SpreadsheetFiles != 3 {
		t.Fatalf("metrics = %#v", result.Metrics)
	}
	if result.Metrics.Sections != 0 || result.Metrics.Links != 0 {
		t.Fatalf("binary documents should not produce markdown sections/links: %#v", result.Metrics)
	}
	for _, file := range files {
		node, ok := g.GetNode("File:" + file.Path)
		if !ok {
			t.Fatalf("file node missing for %s", file.Path)
		}
		if node.Properties["binary"] != true || node.Properties["documentKind"] == "" {
			t.Fatalf("document metadata missing for %s: %#v", file.Path, node.Properties)
		}
	}
}

func TestApplySkipsExternalAndMissingLinks(t *testing.T) {
	dir := t.TempDir()
	writeDocumentTestFile(t, dir, "README.md", "# Intro\n[web](https://example.com)\n[missing](missing.md)\n")

	g := graph.New()
	addFileNode(g, "README.md")

	result, err := Apply(g, dir, []scanner.File{{Path: "README.md", Language: scanner.Markdown}})
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if result.Metrics.Links != 0 {
		t.Fatalf("links = %d, want none for external/missing links", result.Metrics.Links)
	}
}

func addFileNode(g *graph.Graph, filePath string) {
	g.AddNode(graph.Node{
		ID:    graph.GenerateID(string(scopeir.NodeFile), filePath),
		Label: scopeir.NodeFile,
		Properties: graph.NodeProperties{
			"name":     filepath.Base(filePath),
			"filePath": filePath,
		},
	})
}

func writeDocumentTestFile(t *testing.T, root string, rel string, content string) {
	t.Helper()
	fullPath := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", rel, err)
	}
	if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", rel, err)
	}
}

func requireDocumentNode(t *testing.T, g *graph.Graph, id string, label scopeir.NodeLabel) {
	t.Helper()
	node, ok := g.GetNode(id)
	if !ok {
		t.Fatalf("missing node %s", id)
	}
	if node.Label != label {
		t.Fatalf("node %s label = %s, want %s", id, node.Label, label)
	}
}

func requireDocumentRelationship(t *testing.T, g *graph.Graph, relType graph.RelationshipType, sourceID string, targetID string) {
	t.Helper()
	for _, rel := range g.Relationships {
		if rel.Type == relType && rel.SourceID == sourceID && rel.TargetID == targetID {
			return
		}
	}
	t.Fatalf("missing %s %s -> %s", relType, sourceID, targetID)
}
