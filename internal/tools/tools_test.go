package tools

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tamnguyendinh/avmatrix-go/internal/graph"
	"github.com/tamnguyendinh/avmatrix-go/internal/scanner"
	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
	"github.com/tamnguyendinh/avmatrix-go/internal/structure"
)

func TestApplyEmitsObjectRegistrationAndDecoratorTools(t *testing.T) {
	dir := t.TempDir()
	writeToolTestFile(t, dir, "src/tools.ts", `
export const tools = [{
  name: 'query',
  description: 'Run graph query',
  inputSchema: {}
}]
server.tool('impact', 'Impact analysis', handler)
@tool('Trace a bug')
export function traceBug() {}
`)
	files := []scanner.File{{Path: "src/tools.ts", Language: scanner.TypeScript}}
	g := graph.New()
	structure.Apply(g, files)

	result, err := Apply(g, dir, files)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if result.Metrics.ToolsEmitted != 3 || result.Metrics.HandlesEmitted != 3 {
		t.Fatalf("metrics = %#v", result.Metrics)
	}
	for _, toolName := range []string{"query", "impact", "traceBug"} {
		toolID := graph.GenerateID(string(scopeir.NodeTool), toolName)
		requireToolNode(t, g, toolID)
		requireToolRelationship(t, g, graph.GenerateID(string(scopeir.NodeFile), "src/tools.ts"), toolID)
	}
}

func TestApplyEmitsPythonMCPDecoratorTools(t *testing.T) {
	dir := t.TempDir()
	writeToolTestFile(t, dir, "server.py", `from mcp import tool

@mcp.tool()
def get_weather(city: str) -> str:
    """Get weather for a city."""
    return f"Weather in {city}: sunny"

@mcp.tool()
def search_docs(query: str) -> list:
    """Search documentation."""
    return []
`)
	files := []scanner.File{{Path: "server.py", Language: scanner.Python}}
	g := graph.New()
	structure.Apply(g, files)

	result, err := Apply(g, dir, files)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if result.Metrics.ToolsEmitted != 2 || result.Metrics.HandlesEmitted != 2 {
		t.Fatalf("metrics = %#v", result.Metrics)
	}
	for _, toolName := range []string{"get_weather", "search_docs"} {
		toolID := graph.GenerateID(string(scopeir.NodeTool), toolName)
		requireToolNode(t, g, toolID)
		requireToolRelationship(t, g, graph.GenerateID(string(scopeir.NodeFile), "server.py"), toolID)
	}
}

func TestApplyDeduplicatesToolsAndSkipsNonCodeFiles(t *testing.T) {
	dir := t.TempDir()
	writeToolTestFile(t, dir, "src/tools.ts", "server.tool('query', 'first', handler)\napp.tool('query', 'second', handler)\n")
	writeToolTestFile(t, dir, "README.md", "server.tool('ignored', 'doc', handler)\n")
	files := []scanner.File{
		{Path: "src/tools.ts", Language: scanner.TypeScript},
		{Path: "README.md", Language: scanner.Markdown},
	}
	g := graph.New()
	structure.Apply(g, files)

	result, err := Apply(g, dir, files)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if result.Metrics.ToolsEmitted != 1 || result.Metrics.Duplicates != 1 || result.Metrics.FilesScanned != 1 {
		t.Fatalf("metrics = %#v", result.Metrics)
	}
}

func writeToolTestFile(t *testing.T, root string, rel string, contents string) {
	t.Helper()
	fullPath := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", rel, err)
	}
	if err := os.WriteFile(fullPath, []byte(contents), 0o644); err != nil {
		t.Fatalf("write %s: %v", rel, err)
	}
}

func requireToolNode(t *testing.T, g *graph.Graph, id string) {
	t.Helper()
	node, ok := g.GetNode(id)
	if !ok {
		t.Fatalf("missing tool node %s", id)
	}
	if node.Label != scopeir.NodeTool {
		t.Fatalf("node %s label = %s, want %s", id, node.Label, scopeir.NodeTool)
	}
}

func requireToolRelationship(t *testing.T, g *graph.Graph, sourceID string, targetID string) {
	t.Helper()
	for _, rel := range g.Relationships {
		if rel.Type == graph.RelHandlesTool && rel.SourceID == sourceID && rel.TargetID == targetID {
			return
		}
	}
	t.Fatalf("missing HANDLES_TOOL %s -> %s", sourceID, targetID)
}
