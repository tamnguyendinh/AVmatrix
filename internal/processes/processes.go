package processes

import (
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/tamnguyendinh/avmatrix-go/internal/graph"
	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

type Config struct {
	MaxTraceDepth   int
	MaxBranching    int
	MaxProcesses    int
	MaxProcessesCap int
	MinSteps        int
}

const (
	defaultMaxProcessesCap = 700
	minDynamicProcesses    = 20
)

type Result struct {
	Processes []Process
	Steps     []Step
	Metrics   Metrics
}

type Process struct {
	ID             string
	HeuristicLabel string
	ProcessType    string
	StepCount      int
	Communities    []string
	EntryPointID   string
	TerminalID     string
	Trace          []string
}

type Step struct {
	NodeID    string
	ProcessID string
	Step      int
}

type Metrics struct {
	ProcessesEmitted     int     `json:"processesEmitted,omitempty"`
	StepsEmitted         int     `json:"stepsEmitted,omitempty"`
	EntryPointsFound     int     `json:"entryPointsFound,omitempty"`
	CrossCommunity       int     `json:"crossCommunity,omitempty"`
	AverageStepCount     float64 `json:"averageStepCount,omitempty"`
	CallsEdgesConsidered int     `json:"callsEdgesConsidered,omitempty"`
	EntryResourcesLinked int     `json:"entryResourcesLinked,omitempty"`
}

func Apply(g *graph.Graph, config Config) Result {
	if g == nil {
		return Result{}
	}
	cfg := config.withDefaults(g)
	membership := communityMemberships(g)
	calls := buildCallsGraph(g)
	reverseCalls := reverse(calls)
	entryPoints := findEntryPoints(g, calls, reverseCalls)

	traces := make([][]string, 0)
	for _, entryPoint := range entryPoints {
		if len(traces) >= cfg.MaxProcesses*2 {
			break
		}
		for _, trace := range traceFrom(entryPoint, calls, cfg) {
			if len(trace) >= cfg.MinSteps {
				traces = append(traces, trace)
			}
		}
	}
	traces = deduplicateByEndpoints(deduplicateSubsets(traces))
	sort.Slice(traces, func(i int, j int) bool {
		if len(traces[i]) != len(traces[j]) {
			return len(traces[i]) > len(traces[j])
		}
		return strings.Join(traces[i], "\x00") < strings.Join(traces[j], "\x00")
	})
	if len(traces) > cfg.MaxProcesses {
		traces = traces[:cfg.MaxProcesses]
	}

	result := Result{Metrics: Metrics{
		EntryPointsFound:     len(entryPoints),
		CallsEdgesConsidered: countEdges(calls),
	}}
	for index, trace := range traces {
		process := processFromTrace(g, membership, trace, index)
		result.Processes = append(result.Processes, process)
		g.AddNode(graph.Node{
			ID:    process.ID,
			Label: scopeir.NodeProcess,
			Properties: graph.NodeProperties{
				"label":          process.HeuristicLabel,
				"heuristicLabel": process.HeuristicLabel,
				"processType":    process.ProcessType,
				"stepCount":      process.StepCount,
				"communities":    append([]string(nil), process.Communities...),
				"entryPointId":   process.EntryPointID,
				"terminalId":     process.TerminalID,
			},
		})
		g.AddRelationship(graph.Relationship{
			ID:               graph.GenerateID(string(graph.RelEntryPointOf), process.EntryPointID+"->"+process.ID),
			SourceID:         process.EntryPointID,
			TargetID:         process.ID,
			Type:             graph.RelEntryPointOf,
			Confidence:       1,
			Reason:           "process entry point",
			ResolutionSource: "process-detection",
		})
		for stepIndex, nodeID := range trace {
			step := Step{NodeID: nodeID, ProcessID: process.ID, Step: stepIndex + 1}
			result.Steps = append(result.Steps, step)
			stepValue := step.Step
			g.AddRelationship(graph.Relationship{
				ID:               graph.GenerateID(string(graph.RelStepInProcess), nodeID+"->"+process.ID+":"+strconv.Itoa(step.Step)),
				SourceID:         nodeID,
				TargetID:         process.ID,
				Type:             graph.RelStepInProcess,
				Confidence:       1,
				Reason:           "process step",
				Step:             &stepValue,
				ResolutionSource: "process-detection",
			})
		}
		if process.ProcessType == "cross_community" {
			result.Metrics.CrossCommunity++
		}
	}
	result.Metrics.EntryResourcesLinked = linkEntryResourcesToProcesses(g, result.Processes)
	result.Metrics.ProcessesEmitted = len(result.Processes)
	result.Metrics.StepsEmitted = len(result.Steps)
	if len(result.Processes) > 0 {
		totalSteps := 0
		for _, process := range result.Processes {
			totalSteps += process.StepCount
		}
		result.Metrics.AverageStepCount = float64(totalSteps) / float64(len(result.Processes))
	}
	return result
}

func (config Config) withDefaults(g *graph.Graph) Config {
	if config.MaxTraceDepth <= 0 {
		config.MaxTraceDepth = 10
	}
	if config.MaxBranching <= 0 {
		config.MaxBranching = 4
	}
	if config.MaxProcesses <= 0 {
		config.MaxProcesses = dynamicMaxProcesses(symbolCount(g), config.MaxProcessesCap)
	}
	if config.MinSteps <= 0 {
		config.MinSteps = 3
	}
	return config
}

func dynamicMaxProcesses(symbols int, cap int) int {
	if cap <= 0 {
		cap = defaultMaxProcessesCap
	}
	scaled := int(math.Round(float64(symbols) / 10))
	if scaled < minDynamicProcesses {
		scaled = minDynamicProcesses
	}
	if scaled > cap {
		return cap
	}
	return scaled
}

func symbolCount(g *graph.Graph) int {
	if g == nil {
		return 0
	}
	count := 0
	for _, node := range g.Nodes {
		if node.Label != scopeir.NodeFile {
			count++
		}
	}
	return count
}

func linkEntryResourcesToProcesses(g *graph.Graph, processes []Process) int {
	if g == nil || len(processes) == 0 {
		return 0
	}
	resourcesByFile := entryResourcesByFile(g)
	linked := 0
	for _, process := range processes {
		entry, ok := g.GetNode(process.EntryPointID)
		if !ok {
			continue
		}
		entryFile := stringProperty(entry, "filePath")
		if entryFile == "" {
			continue
		}
		for _, resourceID := range resourcesByFile[entryFile] {
			g.AddRelationship(graph.Relationship{
				ID:               graph.GenerateID(string(graph.RelEntryPointOf), resourceID+"->"+process.ID),
				SourceID:         resourceID,
				TargetID:         process.ID,
				Type:             graph.RelEntryPointOf,
				Confidence:       0.85,
				Reason:           "entry-resource-process",
				ResolutionSource: "process-detection",
			})
			linked++
		}
	}
	return linked
}

func entryResourcesByFile(g *graph.Graph) map[string][]string {
	out := make(map[string][]string)
	for _, node := range g.Nodes {
		if node.Label != scopeir.NodeRoute && node.Label != scopeir.NodeTool {
			continue
		}
		filePath := stringProperty(node, "filePath")
		if filePath == "" {
			continue
		}
		out[filePath] = append(out[filePath], node.ID)
	}
	for filePath := range out {
		sort.Strings(out[filePath])
	}
	return out
}

func communityMemberships(g *graph.Graph) map[string]string {
	out := make(map[string]string)
	for _, rel := range g.Relationships {
		if rel.Type == graph.RelMemberOf {
			out[rel.SourceID] = rel.TargetID
		}
	}
	return out
}

type adjacency map[string][]string

func buildCallsGraph(g *graph.Graph) adjacency {
	out := make(adjacency)
	for _, rel := range g.Relationships {
		if rel.Type != graph.RelCalls {
			continue
		}
		if rel.Confidence > 0 && rel.Confidence < 0.5 {
			continue
		}
		out[rel.SourceID] = append(out[rel.SourceID], rel.TargetID)
	}
	for source := range out {
		sort.Strings(out[source])
	}
	return out
}

func reverse(calls adjacency) adjacency {
	out := make(adjacency)
	for source, targets := range calls {
		for _, target := range targets {
			out[target] = append(out[target], source)
		}
	}
	for target := range out {
		sort.Strings(out[target])
	}
	return out
}

func findEntryPoints(g *graph.Graph, calls adjacency, reverseCalls adjacency) []string {
	type candidate struct {
		id    string
		score float64
	}
	candidates := make([]candidate, 0)
	for _, node := range g.Nodes {
		if !isProcessSymbol(node.Label) || isTestFile(stringProperty(node, "filePath")) {
			continue
		}
		callees := calls[node.ID]
		if len(callees) == 0 {
			continue
		}
		callers := reverseCalls[node.ID]
		score := float64(len(callees)*3 - len(callers)*2 + entryNameScore(stringProperty(node, "name")))
		if boolProperty(node, "isExported") {
			score += 2
		}
		if multiplier := floatProperty(node, "astFrameworkMultiplier"); multiplier > 1 {
			score = math.Round(score*multiplier*100) / 100
		}
		if score <= 0 {
			continue
		}
		candidates = append(candidates, candidate{id: node.ID, score: score})
	}
	sort.Slice(candidates, func(i int, j int) bool {
		if candidates[i].score != candidates[j].score {
			return candidates[i].score > candidates[j].score
		}
		return candidates[i].id < candidates[j].id
	})
	if len(candidates) > 200 {
		candidates = candidates[:200]
	}
	out := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		out = append(out, candidate.id)
	}
	return out
}

func traceFrom(entryID string, calls adjacency, config Config) [][]string {
	type item struct {
		nodeID string
		path   []string
	}
	queue := []item{{nodeID: entryID, path: []string{entryID}}}
	traces := make([][]string, 0)
	for len(queue) > 0 && len(traces) < config.MaxBranching*3 {
		current := queue[0]
		queue = queue[1:]
		callees := calls[current.nodeID]
		if len(callees) == 0 || len(current.path) >= config.MaxTraceDepth {
			if len(current.path) >= config.MinSteps {
				traces = append(traces, append([]string(nil), current.path...))
			}
			continue
		}
		if len(callees) > config.MaxBranching {
			callees = callees[:config.MaxBranching]
		}
		added := false
		for _, callee := range callees {
			if contains(current.path, callee) {
				continue
			}
			nextPath := append(append([]string(nil), current.path...), callee)
			queue = append(queue, item{nodeID: callee, path: nextPath})
			added = true
		}
		if !added && len(current.path) >= config.MinSteps {
			traces = append(traces, append([]string(nil), current.path...))
		}
	}
	return traces
}

func deduplicateSubsets(traces [][]string) [][]string {
	sort.Slice(traces, func(i int, j int) bool {
		if len(traces[i]) != len(traces[j]) {
			return len(traces[i]) > len(traces[j])
		}
		return strings.Join(traces[i], "\x00") < strings.Join(traces[j], "\x00")
	})
	unique := make([][]string, 0, len(traces))
	for _, trace := range traces {
		key := strings.Join(trace, "->")
		subset := false
		for _, existing := range unique {
			if strings.Contains(strings.Join(existing, "->"), key) {
				subset = true
				break
			}
		}
		if !subset {
			unique = append(unique, trace)
		}
	}
	return unique
}

func deduplicateByEndpoints(traces [][]string) [][]string {
	byEndpoint := make(map[string][]string)
	for _, trace := range traces {
		key := trace[0] + "\x00" + trace[len(trace)-1]
		if existing, ok := byEndpoint[key]; !ok || len(trace) > len(existing) {
			byEndpoint[key] = trace
		}
	}
	out := make([][]string, 0, len(byEndpoint))
	for _, trace := range byEndpoint {
		out = append(out, trace)
	}
	return out
}

func processFromTrace(g *graph.Graph, memberships map[string]string, trace []string, index int) Process {
	entryID := trace[0]
	terminalID := trace[len(trace)-1]
	entryName := nodeName(g, entryID)
	terminalName := nodeName(g, terminalID)
	communities := traceCommunities(trace, memberships)
	processType := "intra_community"
	if len(communities) > 1 {
		processType = "cross_community"
	}
	return Process{
		ID:             "proc_" + strconv.Itoa(index) + "_" + sanitizeID(entryName),
		HeuristicLabel: capitalize(entryName) + " -> " + capitalize(terminalName),
		ProcessType:    processType,
		StepCount:      len(trace),
		Communities:    communities,
		EntryPointID:   entryID,
		TerminalID:     terminalID,
		Trace:          append([]string(nil), trace...),
	}
}

func traceCommunities(trace []string, memberships map[string]string) []string {
	seen := make(map[string]struct{})
	for _, nodeID := range trace {
		communityID := memberships[nodeID]
		if communityID != "" {
			seen[communityID] = struct{}{}
		}
	}
	out := make([]string, 0, len(seen))
	for communityID := range seen {
		out = append(out, communityID)
	}
	sort.Strings(out)
	return out
}

func nodeName(g *graph.Graph, nodeID string) string {
	node, ok := g.GetNode(nodeID)
	if !ok {
		return "Unknown"
	}
	if name := stringProperty(node, "name"); name != "" {
		return name
	}
	return nodeID
}

func countEdges(calls adjacency) int {
	total := 0
	for _, targets := range calls {
		total += len(targets)
	}
	return total
}

func isProcessSymbol(label scopeir.NodeLabel) bool {
	return label == scopeir.NodeFunction || label == scopeir.NodeMethod
}

func isTestFile(filePath string) bool {
	lower := strings.ToLower(strings.ReplaceAll(filePath, "\\", "/"))
	return strings.Contains(lower, ".test.") ||
		strings.Contains(lower, ".spec.") ||
		strings.Contains(lower, "/test/") ||
		strings.Contains(lower, "/tests/") ||
		strings.Contains(lower, "/__tests__/") ||
		strings.HasPrefix(lower, "test/") ||
		strings.HasPrefix(lower, "tests/") ||
		strings.HasPrefix(lower, "__tests__/") ||
		strings.HasSuffix(lower, "_test.go") ||
		strings.HasSuffix(lower, "_test.py")
}

func entryNameScore(name string) int {
	lower := strings.ToLower(name)
	switch {
	case lower == "main", lower == "run", lower == "start":
		return 5
	case strings.HasPrefix(lower, "handle"), strings.HasPrefix(lower, "on"):
		return 4
	case strings.HasPrefix(lower, "create"), strings.HasPrefix(lower, "update"), strings.HasPrefix(lower, "delete"):
		return 3
	case strings.Contains(lower, "controller"), strings.Contains(lower, "command"):
		return 3
	default:
		return 1
	}
}

func stringProperty(node graph.Node, key string) string {
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

func boolProperty(node graph.Node, key string) bool {
	value, ok := node.Properties[key]
	if !ok || value == nil {
		return false
	}
	typed, ok := value.(bool)
	return ok && typed
}

func floatProperty(node graph.Node, key string) float64 {
	value, ok := node.Properties[key]
	if !ok || value == nil {
		return 0
	}
	switch typed := value.(type) {
	case float64:
		return typed
	case float32:
		return float64(typed)
	case int:
		return float64(typed)
	case int64:
		return float64(typed)
	default:
		return 0
	}
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func sanitizeID(value string) string {
	value = strings.ToLower(value)
	builder := strings.Builder{}
	for _, char := range value {
		if (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') {
			builder.WriteRune(char)
			continue
		}
		builder.WriteByte('_')
	}
	text := strings.Trim(builder.String(), "_")
	if text == "" {
		return "unknown"
	}
	if len(text) > 20 {
		return text[:20]
	}
	return text
}

func capitalize(value string) string {
	if value == "" {
		return value
	}
	return strings.ToUpper(value[:1]) + value[1:]
}
