package analyze

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/tamnguyendinh/avmatrix-go/internal/graph"
	"github.com/tamnguyendinh/avmatrix-go/internal/parser"
	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

type pipelineGraphSummary struct {
	TotalFileCount      int
	NodeCount           int
	RelationshipCount   int
	ByLabel             map[scopeir.NodeLabel]int
	ByRelationshipType  map[graph.RelationshipType]int
	SemanticEdgeDigest  string
	Processes           int
	Communities         int
	StructureEdges      int
	ScopeResolvedCalls  int
	BenchmarkFileExists bool
	GraphFileExists     bool
}

func TestRunProducesStablePipelineGraphSummary(t *testing.T) {
	first := runMiniPipelineSummary(t)
	second := runMiniPipelineSummary(t)

	if !reflect.DeepEqual(first, second) {
		t.Fatalf("pipeline summary drifted\nfirst:  %#v\nsecond: %#v", first, second)
	}
	if first.TotalFileCount < 2 || first.NodeCount == 0 || first.RelationshipCount == 0 {
		t.Fatalf("summary missing graph output: %#v", first)
	}
	if first.ByLabel[scopeir.NodeFile] < 2 || first.ByLabel[scopeir.NodeFunction] < 5 {
		t.Fatalf("summary missing file/function nodes: %#v", first.ByLabel)
	}
	if first.ByRelationshipType[graph.RelContains] == 0 || first.ByRelationshipType[graph.RelCalls] == 0 {
		t.Fatalf("summary missing structure/call edges: %#v", first.ByRelationshipType)
	}
	if first.Processes == 0 || first.StructureEdges == 0 || first.ScopeResolvedCalls == 0 {
		t.Fatalf("summary missing pipeline metrics: %#v", first)
	}
	if !first.BenchmarkFileExists || !first.GraphFileExists {
		t.Fatalf("analyze did not write benchmark/graph artifacts: %#v", first)
	}
}

func runMiniPipelineSummary(t *testing.T) pipelineGraphSummary {
	t.Helper()
	dir := t.TempDir()
	writeFile(t, dir, "src/handler.ts", `
export function validateInput() {
  return true;
}

export function saveToDb() {
  return true;
}

export function formatResponse() {
  return { ok: true };
}

export function handleRequest() {
  if (!validateInput()) {
    return { ok: false };
  }
  saveToDb();
  return formatResponse();
}

export function processRequest() {
  return handleRequest();
}
`)
	writeFile(t, dir, "src/index.ts", `
import { processRequest } from './handler';
export function main() {
  return processRequest();
}
`)
	benchmarkPath := filepath.Join(dir, ".tmp", "benchmark.json")
	result, err := Run(context.Background(), dir, Options{
		Parser:             parser.PoolOptions{ParseTimeout: time.Second},
		DBRunner:           &recordingDBRunner{},
		BenchmarkPath:      benchmarkPath,
		WriteGraphSnapshot: true,
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.Graph == nil {
		t.Fatal("Run() returned nil graph")
	}
	summary := pipelineGraphSummary{
		TotalFileCount:     result.Metrics.Files.Scanned,
		NodeCount:          len(result.Graph.Nodes),
		RelationshipCount:  len(result.Graph.Relationships),
		ByLabel:            map[scopeir.NodeLabel]int{},
		ByRelationshipType: map[graph.RelationshipType]int{},
		Processes:          result.Metrics.Processes.ProcessesEmitted,
		Communities:        result.Metrics.Communities.CommunitiesEmitted,
		StructureEdges:     result.Metrics.Structure.ContainsEmitted,
		ScopeResolvedCalls: result.Metrics.Resolution.ResolvedCalls,
	}
	for _, node := range result.Graph.Nodes {
		summary.ByLabel[node.Label]++
	}
	for _, relationship := range result.Graph.Relationships {
		summary.ByRelationshipType[relationship.Type]++
	}
	summary.SemanticEdgeDigest = semanticEdgeDigest(result.Graph)
	if _, err := os.Stat(benchmarkPath); err == nil {
		summary.BenchmarkFileExists = true
	}
	if _, err := os.Stat(result.GraphPath); err == nil {
		summary.GraphFileExists = true
	}
	return summary
}

func semanticEdgeDigest(g *graph.Graph) string {
	nodeKeys := make(map[string]string, len(g.Nodes))
	for _, node := range g.Nodes {
		nodeKeys[node.ID] = fmt.Sprintf("%s:%s@%s", node.Label, stringNodeProperty(node, "name"), stringNodeProperty(node, "filePath"))
	}
	lines := make([]string, 0, len(g.Relationships))
	for _, relationship := range g.Relationships {
		step := ""
		if relationship.Step != nil {
			step = fmt.Sprintf("#%d", *relationship.Step)
		}
		lines = append(lines, fmt.Sprintf("%s%s|%s|%s", relationship.Type, step, nodeKeys[relationship.SourceID], nodeKeys[relationship.TargetID]))
	}
	sort.Strings(lines)
	sum := sha256.Sum256([]byte(strings.Join(lines, "\n")))
	return hex.EncodeToString(sum[:])
}

func stringNodeProperty(node graph.Node, key string) string {
	value, ok := node.Properties[key]
	if !ok || value == nil {
		return ""
	}
	text, ok := value.(string)
	if !ok {
		return ""
	}
	return text
}
