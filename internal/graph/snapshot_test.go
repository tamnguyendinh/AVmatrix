package graph

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

type testGraphCorrectnessSnapshot struct {
	NodeCount          int
	RelationshipCount  int
	ByNodeLabel        map[string]int
	ByRelationshipType map[string]int
	NodeDigest         string
	RelationshipDigest string
}

type testGraphCorrectnessDiff struct {
	Field    string
	Expected any
	Actual   any
}

func TestGraphCorrectnessSnapshotDetectsNodePropertyAndRelationshipChanges(t *testing.T) {
	before := New()
	before.AddNode(testFileNode("File:src/a.ts", "a.ts", "src/a.ts"))
	before.AddNode(Node{
		ID:    "Function:src/a.ts:run",
		Label: scopeir.NodeFunction,
		Properties: NodeProperties{
			"name":      "run",
			"filePath":  "src/a.ts",
			"startLine": 1,
			"endLine":   3,
		},
	})
	before.AddRelationship(Relationship{
		ID:         "rel-1",
		Type:       RelContains,
		SourceID:   "File:src/a.ts",
		TargetID:   "Function:src/a.ts:run",
		Confidence: 1,
		Reason:     "test",
	})

	after := New()
	after.AddNode(testFileNode("File:src/a.ts", "a.ts", "src/a.ts"))
	after.AddNode(Node{
		ID:    "Function:src/a.ts:run",
		Label: scopeir.NodeFunction,
		Properties: NodeProperties{
			"name":      "runFast",
			"filePath":  "src/a.ts",
			"startLine": 1,
			"endLine":   3,
		},
	})
	after.AddRelationship(Relationship{
		ID:         "rel-1",
		Type:       RelCalls,
		SourceID:   "File:src/a.ts",
		TargetID:   "Function:src/a.ts:run",
		Confidence: 1,
		Reason:     "test",
	})

	diffs := compareTestGraphCorrectnessSnapshots(
		createTestGraphCorrectnessSnapshot(t, before),
		createTestGraphCorrectnessSnapshot(t, after),
	)

	requireDiffFields(t, diffs, "byRelationshipType", "nodeDigest", "relationshipDigest")
}

func TestGraphCorrectnessSnapshotIgnoresInsertionOrder(t *testing.T) {
	first := New()
	second := New()
	fileNode := testFileNode("File:src/a.ts", "a.ts", "src/a.ts")
	functionNode := testGraphNode("Function:src/a.ts:run", scopeir.NodeFunction, "run", "src/a.ts")
	relationship := Relationship{
		ID:         "rel-1",
		Type:       RelContains,
		SourceID:   "File:src/a.ts",
		TargetID:   "Function:src/a.ts:run",
		Confidence: 1,
		Reason:     "test",
	}

	first.AddNode(fileNode)
	first.AddNode(functionNode)
	first.AddRelationship(relationship)
	second.AddNode(functionNode)
	second.AddNode(fileNode)
	second.AddRelationship(relationship)

	diffs := compareTestGraphCorrectnessSnapshots(
		createTestGraphCorrectnessSnapshot(t, first),
		createTestGraphCorrectnessSnapshot(t, second),
	)
	if len(diffs) != 0 {
		t.Fatalf("snapshot diffs for insertion-order equivalent graphs = %#v, want none", diffs)
	}
}

func createTestGraphCorrectnessSnapshot(t *testing.T, g *Graph) testGraphCorrectnessSnapshot {
	t.Helper()

	byNodeLabel := map[string]int{}
	byRelationshipType := map[string]int{}
	nodeLines := make([]string, 0, len(g.Nodes))
	relationshipLines := make([]string, 0, len(g.Relationships))

	for _, node := range g.Nodes {
		byNodeLabel[string(node.Label)]++
		nodeLines = append(nodeLines, mustStableJSON(t, map[string]any{
			"id":         node.ID,
			"label":      node.Label,
			"properties": node.Properties,
		}))
	}
	for _, relationship := range g.Relationships {
		byRelationshipType[string(relationship.Type)]++
		relationshipLines = append(relationshipLines, mustStableJSON(t, testRelationshipSnapshotLine(relationship)))
	}

	sort.Strings(nodeLines)
	sort.Strings(relationshipLines)

	return testGraphCorrectnessSnapshot{
		NodeCount:          len(g.Nodes),
		RelationshipCount:  len(g.Relationships),
		ByNodeLabel:        byNodeLabel,
		ByRelationshipType: byRelationshipType,
		NodeDigest:         hashTestSnapshotLines(nodeLines),
		RelationshipDigest: hashTestSnapshotLines(relationshipLines),
	}
}

func compareTestGraphCorrectnessSnapshots(expected testGraphCorrectnessSnapshot, actual testGraphCorrectnessSnapshot) []testGraphCorrectnessDiff {
	var diffs []testGraphCorrectnessDiff
	appendTestGraphCorrectnessDiff(&diffs, "nodeCount", expected.NodeCount, actual.NodeCount)
	appendTestGraphCorrectnessDiff(&diffs, "relationshipCount", expected.RelationshipCount, actual.RelationshipCount)
	appendTestGraphCorrectnessDiff(&diffs, "byNodeLabel", expected.ByNodeLabel, actual.ByNodeLabel)
	appendTestGraphCorrectnessDiff(&diffs, "byRelationshipType", expected.ByRelationshipType, actual.ByRelationshipType)
	appendTestGraphCorrectnessDiff(&diffs, "nodeDigest", expected.NodeDigest, actual.NodeDigest)
	appendTestGraphCorrectnessDiff(&diffs, "relationshipDigest", expected.RelationshipDigest, actual.RelationshipDigest)
	return diffs
}

func appendTestGraphCorrectnessDiff(diffs *[]testGraphCorrectnessDiff, field string, expected any, actual any) {
	if !reflect.DeepEqual(expected, actual) {
		*diffs = append(*diffs, testGraphCorrectnessDiff{Field: field, Expected: expected, Actual: actual})
	}
}

func testRelationshipSnapshotLine(relationship Relationship) map[string]any {
	line := map[string]any{
		"id":         relationship.ID,
		"type":       relationship.Type,
		"sourceId":   relationship.SourceID,
		"targetId":   relationship.TargetID,
		"confidence": relationship.Confidence,
		"reason":     relationship.Reason,
	}
	if relationship.Step != nil {
		line["step"] = *relationship.Step
	}
	if relationship.ResolutionSource != "" {
		line["resolutionSource"] = relationship.ResolutionSource
	}
	if relationship.FileHash != "" {
		line["fileHash"] = relationship.FileHash
	}
	if len(relationship.Evidence) > 0 {
		line["evidence"] = relationship.Evidence
	}
	return line
}

func mustStableJSON(t *testing.T, value any) string {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal snapshot line: %v", err)
	}
	return string(raw)
}

func hashTestSnapshotLines(lines []string) string {
	sum := sha256.Sum256([]byte(strings.Join(lines, "\n")))
	return hex.EncodeToString(sum[:])
}

func requireDiffFields(t *testing.T, diffs []testGraphCorrectnessDiff, fields ...string) {
	t.Helper()
	seen := map[string]bool{}
	for _, diff := range diffs {
		seen[diff.Field] = true
	}
	for _, field := range fields {
		if !seen[field] {
			t.Fatalf("diff fields = %#v, want field %q", seen, field)
		}
	}
}

func testFileNode(id string, name string, filePath string) Node {
	return Node{
		ID:    id,
		Label: scopeir.NodeFile,
		Properties: NodeProperties{
			"name":     name,
			"filePath": filePath,
		},
	}
}
