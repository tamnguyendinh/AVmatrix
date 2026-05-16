package mcp

import (
	"fmt"
	"testing"

	"github.com/tamnguyendinh/avmatrix-go/internal/graph"
	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

func TestImpactClassLikeTraversalCoversJavaConstructorAndFileImportTopology(t *testing.T) {
	g := graph.New()
	addClassTopology(t, g, "java", true, "SessionTracker", "api/session/SessionTracker.java")
	addCaller(t, g, "Method:registerSessionTracker", scopeir.NodeMethod, "registerSessionTracker", "core/bootstrap/ServerBootstrap.java", "Constructor:java:SessionTracker")
	addCaller(t, g, "Method:setUp", scopeir.NodeMethod, "setUp", "src/test/java/api/session/SessionTrackerTest.java", "Constructor:java:SessionTracker")
	addImporter(t, g, "File:core/bootstrap/ServerBootstrap.java", "ServerBootstrap.java", "core/bootstrap/ServerBootstrap.java", "File:java:SessionTracker")
	addImporter(t, g, "File:src/test/java/api/session/SessionTrackerTest.java", "SessionTrackerTest.java", "src/test/java/api/session/SessionTrackerTest.java", "File:java:SessionTracker")

	target, ok := g.GetNode("Class:java:SessionTracker")
	if !ok {
		t.Fatalf("missing SessionTracker class")
	}
	prodOnly, _ := runImpactBFSProfiled(g, target, classImpactOptions(false), false)
	if prodOnly["impactedCount"] != 2 {
		t.Fatalf("prod impact count = %#v payload=%#v", prodOnly["impactedCount"], prodOnly)
	}
	assertImpactNames(t, prodOnly, []string{"registerSessionTracker", "ServerBootstrap.java"}, []string{"setUp", "SessionTrackerTest.java"})

	withTests, _ := runImpactBFSProfiled(g, target, classImpactOptions(true), false)
	if withTests["impactedCount"] != 4 {
		t.Fatalf("includeTests impact count = %#v payload=%#v", withTests["impactedCount"], withTests)
	}
	assertImpactNames(t, withTests, []string{"registerSessionTracker", "ServerBootstrap.java", "setUp", "SessionTrackerTest.java"}, nil)
}

func TestImpactClassLikeTraversalCoversSupportedLanguageTopologies(t *testing.T) {
	tests := []struct {
		lang      string
		jvm       bool
		className string
		callLabel scopeir.NodeLabel
	}{
		{lang: "java", jvm: true, className: "PaymentService", callLabel: scopeir.NodeMethod},
		{lang: "kotlin", jvm: true, className: "UserRepository", callLabel: scopeir.NodeMethod},
		{lang: "ts", className: "AuthService", callLabel: scopeir.NodeFunction},
		{lang: "js", className: "EventEmitter", callLabel: scopeir.NodeFunction},
		{lang: "python", className: "DatabaseClient", callLabel: scopeir.NodeFunction},
		{lang: "csharp", className: "OrderProcessor", callLabel: scopeir.NodeMethod},
		{lang: "ruby", className: "SessionManager", callLabel: scopeir.NodeFunction},
		{lang: "php", className: "CacheService", callLabel: scopeir.NodeMethod},
		{lang: "rust", className: "HttpClient", callLabel: scopeir.NodeFunction},
		{lang: "go", className: "Router", callLabel: scopeir.NodeFunction},
		{lang: "swift", className: "NetworkManager", callLabel: scopeir.NodeMethod},
		{lang: "c", className: "Connection", callLabel: scopeir.NodeFunction},
		{lang: "cpp", className: "Widget", callLabel: scopeir.NodeFunction},
	}

	for _, tc := range tests {
		t.Run(tc.lang, func(t *testing.T) {
			g := graph.New()
			classPath := fmt.Sprintf("src/%s/%s.ext", tc.lang, tc.className)
			addClassTopology(t, g, tc.lang, tc.jvm, tc.className, classPath)
			targetID := "Class:" + tc.lang + ":" + tc.className
			callTargetID := targetID
			if tc.jvm {
				callTargetID = "Constructor:" + tc.lang + ":" + tc.className
			}
			callerName := "use" + tc.className
			addCaller(t, g, "Caller:"+tc.lang, tc.callLabel, callerName, fmt.Sprintf("src/%s/caller.ext", tc.lang), callTargetID)
			addImporter(t, g, "Importer:"+tc.lang, tc.className+"Importer.ext", fmt.Sprintf("src/%s/importer.ext", tc.lang), "File:"+tc.lang+":"+tc.className)

			target, ok := g.GetNode(targetID)
			if !ok {
				t.Fatalf("missing target %s", targetID)
			}
			payload, _ := runImpactBFSProfiled(g, target, classImpactOptions(false), false)
			if payload["impactedCount"] != 2 {
				t.Fatalf("impact count = %#v payload=%#v", payload["impactedCount"], payload)
			}
			assertImpactNames(t, payload, []string{callerName, tc.className + "Importer.ext"}, nil)
			incoming, _, _ := contextNeighborhood(g, target)
			assertContextRefNames(t, incoming, "calls", []string{callerName})
			assertContextRefNames(t, incoming, "imports", []string{tc.className + "Importer.ext"})
		})
	}
}

func addClassTopology(t *testing.T, g *graph.Graph, lang string, jvm bool, className string, filePath string) {
	t.Helper()
	fileID := "File:" + lang + ":" + className
	classID := "Class:" + lang + ":" + className
	g.AddNode(graph.Node{ID: fileID, Label: scopeir.NodeFile, Properties: graph.NodeProperties{
		"name": className + ".ext", "filePath": filePath,
	}})
	g.AddNode(graph.Node{ID: classID, Label: scopeir.NodeClass, Properties: graph.NodeProperties{
		"name": className, "filePath": filePath, "startLine": 1, "endLine": 80,
	}})
	g.AddRelationship(graph.Relationship{ID: "rel:defines:" + classID, SourceID: fileID, TargetID: classID, Type: graph.RelDefines, Confidence: 1})
	if jvm {
		constructorID := "Constructor:" + lang + ":" + className
		g.AddNode(graph.Node{ID: constructorID, Label: scopeir.NodeConstructor, Properties: graph.NodeProperties{
			"name": className, "filePath": filePath, "startLine": 5, "endLine": 10,
		}})
		g.AddRelationship(graph.Relationship{ID: "rel:ctor:" + classID, SourceID: classID, TargetID: constructorID, Type: graph.RelHasMethod, Confidence: 1})
	}
}

func addCaller(t *testing.T, g *graph.Graph, id string, label scopeir.NodeLabel, name string, filePath string, targetID string) {
	t.Helper()
	g.AddNode(graph.Node{ID: id, Label: label, Properties: graph.NodeProperties{
		"name": name, "filePath": filePath, "startLine": 10, "endLine": 20,
	}})
	g.AddRelationship(graph.Relationship{ID: "rel:calls:" + id + ":" + targetID, SourceID: id, TargetID: targetID, Type: graph.RelCalls, Confidence: 0.9})
}

func addImporter(t *testing.T, g *graph.Graph, id string, name string, filePath string, targetFileID string) {
	t.Helper()
	g.AddNode(graph.Node{ID: id, Label: scopeir.NodeFile, Properties: graph.NodeProperties{
		"name": name, "filePath": filePath,
	}})
	g.AddRelationship(graph.Relationship{ID: "rel:imports:" + id + ":" + targetFileID, SourceID: id, TargetID: targetFileID, Type: graph.RelImports, Confidence: 0.9})
}

func classImpactOptions(includeTests bool) impactOptions {
	return impactOptions{
		Direction:     "upstream",
		MaxDepth:      2,
		RelationTypes: []string{string(graph.RelCalls), string(graph.RelImports), string(graph.RelDefines), string(graph.RelHasMethod)},
		IncludeTests:  includeTests,
	}
}

func assertImpactNames(t *testing.T, payload map[string]any, wantPresent []string, wantAbsent []string) {
	t.Helper()
	names := map[string]bool{}
	byDepth := payload["byDepth"].(map[string][]map[string]any)
	for _, items := range byDepth {
		for _, item := range items {
			names[fmt.Sprint(item["name"])] = true
		}
	}
	for _, name := range wantPresent {
		if !names[name] {
			t.Fatalf("impact names missing %q in %#v", name, names)
		}
	}
	for _, name := range wantAbsent {
		if names[name] {
			t.Fatalf("impact names unexpectedly include %q in %#v", name, names)
		}
	}
}

func assertContextRefNames(t *testing.T, incoming map[string][]map[string]any, key string, want []string) {
	t.Helper()
	seen := map[string]bool{}
	for _, ref := range incoming[key] {
		seen[fmt.Sprint(ref["name"])] = true
	}
	for _, name := range want {
		if !seen[name] {
			t.Fatalf("incoming[%s] missing %q in %#v", key, name, incoming[key])
		}
	}
}
