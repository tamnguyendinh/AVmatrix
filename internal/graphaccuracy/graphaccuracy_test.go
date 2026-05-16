package graphaccuracy

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestRunReportsGoLocalAccuracyGaps(t *testing.T) {
	repo := writeAccuracyFixtureRepo(t)
	completeGraph := accuracyFixtureGraph(true)
	missingGraph := accuracyFixtureGraph(false)
	nodeGraphPath := writeGraphFixture(t, repo, "node.json", completeGraph)
	goGraphPath := writeGraphFixture(t, repo, "go.json", missingGraph)
	outPath := filepath.Join(repo, "accuracy.json")

	result, err := Run(Options{
		Repo:          repo,
		NodeGraphPath: nodeGraphPath,
		GoGraphPath:   goGraphPath,
		OutPath:       outPath,
		MaxExamples:   10,
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if _, err := os.Stat(outPath); err != nil {
		t.Fatalf("accuracy output missing: %v", err)
	}

	assertMetric(t, result.Definitions["TypeAlias"]["goLocal"], 1, 2)
	assertMetric(t, result.Definitions["Variable"]["goLocal"], 1, 2)
	assertMetric(t, result.Calls["goLocal"], 1, 2)

	failures := GoLocalFailures(result)
	for _, want := range []string{"Definition TypeAlias", "Definition Variable", "Direct CALLS subset"} {
		if !slices.ContainsFunc(failures, func(f GateFailure) bool { return f.Gate == want }) {
			t.Fatalf("GoLocalFailures() missing %q in %#v", want, failures)
		}
	}
}

func TestGoLocalFailuresPassWhenGraphMatchesGroundTruth(t *testing.T) {
	repo := writeAccuracyFixtureRepo(t)
	completeGraph := accuracyFixtureGraph(true)
	nodeGraphPath := writeGraphFixture(t, repo, "node.json", completeGraph)
	goGraph := completeGraph
	goGraph.Nodes = append(goGraph.Nodes,
		fileNode("new/gate.go"),
		defNode("Function", "new/gate.go", "GeneratedAfterBaseline"),
	)
	goGraphPath := writeGraphFixture(t, repo, "go.json", goGraph)

	result, err := Run(Options{
		Repo:          repo,
		NodeGraphPath: nodeGraphPath,
		GoGraphPath:   goGraphPath,
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if failures := GoLocalFailures(result); len(failures) != 0 {
		t.Fatalf("GoLocalFailures() = %#v, want none", failures)
	}
}

func writeAccuracyFixtureRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	writeAccuracyFixtureFile(t, dir, "go.mod", "module example.com/demo\n\ngo 1.26\n")
	writeAccuracyFixtureFile(t, dir, "a/a.go", `package a

const Limit = 1

var PackageValue = 2

type Alias = string
type Named int
type Shape struct{}
type Doer interface {
	Do()
}

func Caller() {
	local := 1
	_ = local
	Callee()
}

func Callee() {}
`)
	writeAccuracyFixtureFile(t, dir, "b/b.go", `package b

import "example.com/demo/a"

func Use() {
	a.Callee()
}
`)
	return dir
}

func writeAccuracyFixtureFile(t *testing.T, root string, rel string, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir fixture dir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write fixture file: %v", err)
	}
}

func writeGraphFixture(t *testing.T, root string, name string, g GraphFile) string {
	t.Helper()
	path := filepath.Join(root, name)
	raw, err := json.MarshalIndent(g, "", "  ")
	if err != nil {
		t.Fatalf("marshal graph fixture: %v", err)
	}
	if err := os.WriteFile(path, append(raw, '\n'), 0o644); err != nil {
		t.Fatalf("write graph fixture: %v", err)
	}
	return path
}

func accuracyFixtureGraph(complete bool) GraphFile {
	nodes := []GraphNode{
		fileNode("a/a.go"),
		fileNode("b/b.go"),
		defNode("Function", "a/a.go", "Caller"),
		defNode("Function", "a/a.go", "Callee"),
		defNode("Function", "b/b.go", "Use"),
		defNode("TypeAlias", "a/a.go", "Named"),
		defNode("Struct", "a/a.go", "Shape"),
		defNode("Interface", "a/a.go", "Doer"),
		methodNode("a/a.go", "Doer.Do"),
		defNode("Const", "a/a.go", "Limit"),
		defNode("Variable", "a/a.go", "PackageValue"),
	}
	relationships := []GraphRelationship{
		{ID: "imports:b-a", Type: "IMPORTS", SourceID: "File:b/b.go", TargetID: "File:a/a.go"},
		{ID: "calls:use-callee", Type: "CALLS", SourceID: "Function:b/b.go:Use", TargetID: "Function:a/a.go:Callee"},
	}
	if complete {
		nodes = append(nodes,
			defNode("TypeAlias", "a/a.go", "Alias"),
			defNode("Variable", "a/a.go", "local"),
		)
		relationships = append(relationships, GraphRelationship{
			ID:       "calls:caller-callee",
			Type:     "CALLS",
			SourceID: "Function:a/a.go:Caller",
			TargetID: "Function:a/a.go:Callee",
		})
	}
	return GraphFile{Nodes: nodes, Relationships: relationships}
}

func fileNode(rel string) GraphNode {
	return GraphNode{
		ID:    "File:" + rel,
		Label: "File",
		Properties: map[string]any{
			"name":     filepath.Base(rel),
			"filePath": rel,
		},
	}
}

func defNode(label string, rel string, name string) GraphNode {
	return GraphNode{
		ID:    label + ":" + rel + ":" + name,
		Label: label,
		Properties: map[string]any{
			"name":     name,
			"filePath": rel,
		},
	}
}

func methodNode(rel string, qualified string) GraphNode {
	return GraphNode{
		ID:    "Method:" + rel + ":" + qualified,
		Label: "Method",
		Properties: map[string]any{
			"name":          "Do",
			"qualifiedName": qualified,
			"filePath":      rel,
		},
	}
}

func assertMetric(t *testing.T, metric AnalyzerMetrics, matched int, expected int) {
	t.Helper()
	if metric.Matched != matched || metric.Expected != expected {
		t.Fatalf("metric = %d/%d, want %d/%d (%#v)", metric.Matched, metric.Expected, matched, expected, metric)
	}
}
