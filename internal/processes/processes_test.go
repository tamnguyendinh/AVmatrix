package processes

import (
	"strconv"
	"testing"

	"github.com/tamnguyendinh/avmatrix-go/internal/graph"
	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

func TestApplyEmitsProcessesAndStepRelationships(t *testing.T) {
	g := graph.New()
	g.AddNode(processSymbol("Function:handle", "handleRequest", "src/api/handler.ts"))
	g.AddNode(processSymbol("Function:validate", "validateInput", "src/api/handler.ts"))
	g.AddNode(processSymbol("Function:save", "saveRecord", "src/db/repo.ts"))
	g.AddNode(graph.Node{ID: "comm_api", Label: scopeir.NodeCommunity, Properties: graph.NodeProperties{"label": "Api"}})
	g.AddNode(graph.Node{ID: "comm_db", Label: scopeir.NodeCommunity, Properties: graph.NodeProperties{"label": "Db"}})
	g.AddRelationship(callRelationship("rel:CALLS:handle->validate", "Function:handle", "Function:validate", 1))
	g.AddRelationship(callRelationship("rel:CALLS:validate->save", "Function:validate", "Function:save", 1))
	g.AddRelationship(memberRelationship("Function:handle", "comm_api"))
	g.AddRelationship(memberRelationship("Function:validate", "comm_api"))
	g.AddRelationship(memberRelationship("Function:save", "comm_db"))

	result := Apply(g, Config{})
	if result.Metrics.ProcessesEmitted != 1 || result.Metrics.StepsEmitted != 3 || result.Metrics.CrossCommunity != 1 {
		t.Fatalf("metrics = %#v, want one cross-community process with three steps", result.Metrics)
	}
	process, ok := g.GetNode("proc_0_handlerequest")
	if !ok {
		t.Fatal("process node missing")
	}
	if process.Label != scopeir.NodeProcess || process.Properties["processType"] != "cross_community" {
		t.Fatalf("process node = %#v", process)
	}
	requireProcessRelationship(t, g, graph.RelEntryPointOf, "Function:handle", "proc_0_handlerequest", 0)
	requireProcessRelationship(t, g, graph.RelStepInProcess, "Function:handle", "proc_0_handlerequest", 1)
	requireProcessRelationship(t, g, graph.RelStepInProcess, "Function:validate", "proc_0_handlerequest", 2)
	requireProcessRelationship(t, g, graph.RelStepInProcess, "Function:save", "proc_0_handlerequest", 3)
}

func TestApplySkipsShortAndTestFileTraces(t *testing.T) {
	g := graph.New()
	g.AddNode(processSymbol("Function:testHandle", "handleThing", "src/api/handler.test.ts"))
	g.AddNode(processSymbol("Function:helper", "helper", "src/api/helper.ts"))
	g.AddRelationship(callRelationship("rel:CALLS:test->helper", "Function:testHandle", "Function:helper", 1))

	result := Apply(g, Config{})
	if result.Metrics.ProcessesEmitted != 0 {
		t.Fatalf("metrics = %#v, want no process from test entry point", result.Metrics)
	}
}

func TestIsTestFileSecurityContract(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{path: "src/foo.test.ts", want: true},
		{path: "src/foo.spec.ts", want: true},
		{path: "src/__tests__/foo.ts", want: true},
		{path: "src/test/foo.ts", want: true},
		{path: `src\test\foo.ts`, want: true},
		{path: `src\__tests__\bar.ts`, want: true},
		{path: "SRC/TEST/Foo.ts", want: true},
		{path: "SRC/Foo.Test.ts", want: true},
		{path: "pkg/handler_test.go", want: true},
		{path: "tests/test_handler.py", want: true},
		{path: "pkg/handler_test.py", want: true},
		{path: "src/main.ts", want: false},
		{path: "src/utils/helper.ts", want: false},
	}
	for _, tt := range tests {
		if got := isTestFile(tt.path); got != tt.want {
			t.Fatalf("isTestFile(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}

func TestApplyUsesFrameworkMultiplierWhenOrderingEntryPoints(t *testing.T) {
	g := graph.New()
	plain := processSymbol("Function:plain", "task", "src/workers/task.ts")
	frameworkEntry := processSymbol("Function:framework", "task", "app/api/users/route.ts")
	frameworkEntry.Properties["astFrameworkMultiplier"] = 3.0
	frameworkEntry.Properties["astFrameworkReason"] = "nextjs-api-route"
	g.AddNode(plain)
	g.AddNode(frameworkEntry)
	g.AddNode(processSymbol("Function:plainLeaf", "plainLeaf", "src/workers/task.ts"))
	g.AddNode(processSymbol("Function:frameworkLeaf", "frameworkLeaf", "app/api/users/route.ts"))
	g.AddRelationship(callRelationship("rel:CALLS:plain->plainLeaf", "Function:plain", "Function:plainLeaf", 1))
	g.AddRelationship(callRelationship("rel:CALLS:framework->frameworkLeaf", "Function:framework", "Function:frameworkLeaf", 1))

	result := Apply(g, Config{MinSteps: 2, MaxProcesses: 1})
	if result.Metrics.ProcessesEmitted != 1 {
		t.Fatalf("metrics = %#v, want one process", result.Metrics)
	}
	requireProcessRelationship(t, g, graph.RelEntryPointOf, "Function:framework", "proc_0_task", 0)
}

func TestFindEntryPointsScoresGraphShapeNamesExportsAndFrameworks(t *testing.T) {
	g := graph.New()
	handle := processSymbol("Function:handle", "handleRequest", "src/api/handler.ts")
	handle.Properties["isExported"] = true
	frameworkEntry := processSymbol("Function:framework", "task", "app/api/users/route.ts")
	frameworkEntry.Properties["astFrameworkMultiplier"] = 3.0
	testEntry := processSymbol("Function:testHandle", "handleRequest", "src/api/handler.test.ts")

	for _, node := range []graph.Node{
		handle,
		frameworkEntry,
		processSymbol("Function:start", "start", "src/start.ts"),
		processSymbol("Function:helper", "helper", "src/helper.ts"),
		testEntry,
		processSymbol("Function:h1", "h1", "src/api/handler.ts"),
		processSymbol("Function:h2", "h2", "src/api/handler.ts"),
		processSymbol("Function:h3", "h3", "src/api/handler.ts"),
		processSymbol("Function:frameworkLeaf", "frameworkLeaf", "app/api/users/route.ts"),
		processSymbol("Function:startLeaf", "startLeaf", "src/start.ts"),
		processSymbol("Function:helperLeaf", "helperLeaf", "src/helper.ts"),
		processSymbol("Function:testLeaf", "testLeaf", "src/api/handler.test.ts"),
	} {
		g.AddNode(node)
	}
	g.AddRelationship(callRelationship("rel:CALLS:handle->h1", "Function:handle", "Function:h1", 1))
	g.AddRelationship(callRelationship("rel:CALLS:handle->h2", "Function:handle", "Function:h2", 1))
	g.AddRelationship(callRelationship("rel:CALLS:handle->h3", "Function:handle", "Function:h3", 1))
	g.AddRelationship(callRelationship("rel:CALLS:framework->leaf", "Function:framework", "Function:frameworkLeaf", 1))
	g.AddRelationship(callRelationship("rel:CALLS:start->leaf", "Function:start", "Function:startLeaf", 1))
	g.AddRelationship(callRelationship("rel:CALLS:helper->leaf", "Function:helper", "Function:helperLeaf", 1))
	g.AddRelationship(callRelationship("rel:CALLS:test->leaf", "Function:testHandle", "Function:testLeaf", 1))
	g.AddRelationship(callRelationship("rel:CALLS:handle->helper", "Function:handle", "Function:helper", 1))
	g.AddRelationship(callRelationship("rel:CALLS:start->helper", "Function:start", "Function:helper", 1))

	calls := buildCallsGraph(g)
	entryPoints := findEntryPoints(g, calls, reverse(calls))
	wantPrefix := []string{"Function:handle", "Function:framework", "Function:start"}
	if len(entryPoints) < len(wantPrefix) {
		t.Fatalf("entryPoints = %#v, want prefix %#v", entryPoints, wantPrefix)
	}
	for index, want := range wantPrefix {
		if entryPoints[index] != want {
			t.Fatalf("entryPoints = %#v, want prefix %#v", entryPoints, wantPrefix)
		}
	}
	for _, excluded := range []string{"Function:helper", "Function:testHandle", "Function:h1"} {
		for _, got := range entryPoints {
			if got == excluded {
				t.Fatalf("entryPoints = %#v, unexpectedly included %s", entryPoints, excluded)
			}
		}
	}
}

func TestApplyUsesDynamicProcessBudgetWhenUnconfigured(t *testing.T) {
	g := graph.New()
	for i := 0; i < 80; i++ {
		entryID := "Function:handle" + strconv.Itoa(i)
		middleID := "Function:middle" + strconv.Itoa(i)
		leafID := "Function:leaf" + strconv.Itoa(i)
		g.AddNode(processSymbol(entryID, "handle"+strconv.Itoa(i), "src/flow"+strconv.Itoa(i)+".ts"))
		g.AddNode(processSymbol(middleID, "middle"+strconv.Itoa(i), "src/flow"+strconv.Itoa(i)+".ts"))
		g.AddNode(processSymbol(leafID, "leaf"+strconv.Itoa(i), "src/flow"+strconv.Itoa(i)+".ts"))
		g.AddRelationship(callRelationship("rel:CALLS:handle"+strconv.Itoa(i)+"->middle"+strconv.Itoa(i), entryID, middleID, 1))
		g.AddRelationship(callRelationship("rel:CALLS:middle"+strconv.Itoa(i)+"->leaf"+strconv.Itoa(i), middleID, leafID, 1))
	}
	for i := 0; i < 700; i++ {
		g.AddNode(processSymbol("Function:unused"+strconv.Itoa(i), "unused"+strconv.Itoa(i), "src/unused.ts"))
	}

	result := Apply(g, Config{})
	if result.Metrics.ProcessesEmitted != 80 {
		t.Fatalf("processes emitted = %d, want all 80 when dynamic budget exceeds old 75 cap", result.Metrics.ProcessesEmitted)
	}
}

func TestApplyUsesConfiguredProcessCapAfterDynamicScaling(t *testing.T) {
	g := graph.New()
	for i := 0; i < 50; i++ {
		entryID := "Function:handleCap" + strconv.Itoa(i)
		leafID := "Function:leafCap" + strconv.Itoa(i)
		g.AddNode(processSymbol(entryID, "handleCap"+strconv.Itoa(i), "src/flow"+strconv.Itoa(i)+".ts"))
		g.AddNode(processSymbol(leafID, "leafCap"+strconv.Itoa(i), "src/flow"+strconv.Itoa(i)+".ts"))
		g.AddRelationship(callRelationship("rel:CALLS:handleCap"+strconv.Itoa(i)+"->leafCap"+strconv.Itoa(i), entryID, leafID, 1))
	}

	result := Apply(g, Config{MinSteps: 2, MaxProcessesCap: 7})
	if result.Metrics.ProcessesEmitted != 7 {
		t.Fatalf("processes emitted = %d, want configured cap 7", result.Metrics.ProcessesEmitted)
	}
}

func TestApplyHandlesEmptyNoCallsLowConfidenceCyclesAndTraceLimits(t *testing.T) {
	if result := Apply(nil, Config{}); result.Metrics.ProcessesEmitted != 0 || len(result.Processes) != 0 {
		t.Fatalf("nil graph result = %#v, want empty", result)
	}

	noCalls := graph.New()
	noCalls.AddNode(processSymbol("Function:handleNoCalls", "handleNoCalls", "src/no-calls.ts"))
	if result := Apply(noCalls, Config{}); result.Metrics.ProcessesEmitted != 0 {
		t.Fatalf("no-calls processes emitted = %d, want 0", result.Metrics.ProcessesEmitted)
	}

	lowConfidence := graph.New()
	lowConfidence.AddNode(processSymbol("Function:handleLow", "handleLow", "src/low.ts"))
	lowConfidence.AddNode(processSymbol("Function:middleLow", "middleLow", "src/low.ts"))
	lowConfidence.AddNode(processSymbol("Function:leafLow", "leafLow", "src/low.ts"))
	lowConfidence.AddRelationship(callRelationship("rel:CALLS:handleLow->middleLow", "Function:handleLow", "Function:middleLow", 0.49))
	lowConfidence.AddRelationship(callRelationship("rel:CALLS:middleLow->leafLow", "Function:middleLow", "Function:leafLow", 1))
	if result := Apply(lowConfidence, Config{}); result.Metrics.ProcessesEmitted != 0 || result.Metrics.CallsEdgesConsidered != 1 {
		t.Fatalf("low-confidence result = %#v, want no process and one considered call", result.Metrics)
	}

	cyclic := graph.New()
	cyclic.AddNode(processSymbol("Function:handleCycle", "handleCycle", "src/cycle.ts"))
	cyclic.AddNode(processSymbol("Function:middleCycle", "middleCycle", "src/cycle.ts"))
	cyclic.AddNode(processSymbol("Function:leafCycle", "leafCycle", "src/cycle.ts"))
	cyclic.AddRelationship(callRelationship("rel:CALLS:handleCycle->middleCycle", "Function:handleCycle", "Function:middleCycle", 1))
	cyclic.AddRelationship(callRelationship("rel:CALLS:middleCycle->leafCycle", "Function:middleCycle", "Function:leafCycle", 1))
	cyclic.AddRelationship(callRelationship("rel:CALLS:leafCycle->handleCycle", "Function:leafCycle", "Function:handleCycle", 1))
	cyclicResult := Apply(cyclic, Config{MaxProcesses: 1})
	if cyclicResult.Metrics.ProcessesEmitted != 1 || cyclicResult.Processes[0].StepCount != 3 {
		t.Fatalf("cycle result = %#v processes=%#v, want one three-step process", cyclicResult.Metrics, cyclicResult.Processes)
	}

	depthLimited := graph.New()
	depthLimited.AddNode(processSymbol("Function:handleDepth", "handleDepth", "src/depth.ts"))
	depthLimited.AddNode(processSymbol("Function:stepOne", "stepOne", "src/depth.ts"))
	depthLimited.AddNode(processSymbol("Function:stepTwo", "stepTwo", "src/depth.ts"))
	depthLimited.AddNode(processSymbol("Function:stepThree", "stepThree", "src/depth.ts"))
	depthLimited.AddRelationship(callRelationship("rel:CALLS:handleDepth->stepOne", "Function:handleDepth", "Function:stepOne", 1))
	depthLimited.AddRelationship(callRelationship("rel:CALLS:stepOne->stepTwo", "Function:stepOne", "Function:stepTwo", 1))
	depthLimited.AddRelationship(callRelationship("rel:CALLS:stepTwo->stepThree", "Function:stepTwo", "Function:stepThree", 1))
	depthResult := Apply(depthLimited, Config{MaxTraceDepth: 3, MaxProcesses: 1})
	if depthResult.Metrics.ProcessesEmitted != 1 || depthResult.Processes[0].StepCount != 3 || depthResult.Processes[0].TerminalID != "Function:stepTwo" {
		t.Fatalf("depth-limited result = %#v processes=%#v, want terminal at stepTwo", depthResult.Metrics, depthResult.Processes)
	}
}

func TestApplyLinksRouteAndToolEntryResourcesToProcesses(t *testing.T) {
	g := graph.New()
	g.AddNode(processSymbol("Function:handle", "handleRequest", "src/api/handler.ts"))
	g.AddNode(processSymbol("Function:validate", "validateInput", "src/api/handler.ts"))
	g.AddNode(processSymbol("Function:save", "saveRecord", "src/db/repo.ts"))
	g.AddNode(graph.Node{ID: "Route:/api/users", Label: scopeir.NodeRoute, Properties: graph.NodeProperties{
		"name":     "/api/users",
		"filePath": "src/api/handler.ts",
	}})
	g.AddNode(graph.Node{ID: "Tool:user_search", Label: scopeir.NodeTool, Properties: graph.NodeProperties{
		"name":     "user_search",
		"filePath": "src/api/handler.ts",
	}})
	g.AddRelationship(callRelationship("rel:CALLS:handle->validate", "Function:handle", "Function:validate", 1))
	g.AddRelationship(callRelationship("rel:CALLS:validate->save", "Function:validate", "Function:save", 1))

	result := Apply(g, Config{})
	if result.Metrics.EntryResourcesLinked != 2 {
		t.Fatalf("entry resources linked = %d, want 2", result.Metrics.EntryResourcesLinked)
	}
	requireProcessRelationship(t, g, graph.RelEntryPointOf, "Route:/api/users", "proc_0_handlerequest", 0)
	requireProcessRelationship(t, g, graph.RelEntryPointOf, "Tool:user_search", "proc_0_handlerequest", 0)
}

func processSymbol(id string, name string, filePath string) graph.Node {
	return graph.Node{
		ID:    id,
		Label: scopeir.NodeFunction,
		Properties: graph.NodeProperties{
			"name":     name,
			"filePath": filePath,
		},
	}
}

func callRelationship(id string, sourceID string, targetID string, confidence float64) graph.Relationship {
	return graph.Relationship{
		ID:         id,
		SourceID:   sourceID,
		TargetID:   targetID,
		Type:       graph.RelCalls,
		Confidence: confidence,
	}
}

func memberRelationship(sourceID string, targetID string) graph.Relationship {
	return graph.Relationship{
		ID:         graph.GenerateID(string(graph.RelMemberOf), sourceID+"->"+targetID),
		SourceID:   sourceID,
		TargetID:   targetID,
		Type:       graph.RelMemberOf,
		Confidence: 1,
	}
}

func requireProcessRelationship(t *testing.T, g *graph.Graph, relType graph.RelationshipType, sourceID string, targetID string, step int) {
	t.Helper()
	for _, rel := range g.Relationships {
		if rel.Type != relType || rel.SourceID != sourceID || rel.TargetID != targetID {
			continue
		}
		if step == 0 {
			return
		}
		if rel.Step != nil && *rel.Step == step {
			return
		}
	}
	t.Fatalf("missing relationship %s %s -> %s step %d", relType, sourceID, targetID, step)
}
