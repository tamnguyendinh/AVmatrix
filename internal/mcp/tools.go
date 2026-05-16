package mcp

import (
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/tamnguyendinh/avmatrix-go/internal/graph"
	"github.com/tamnguyendinh/avmatrix-go/internal/lbugnative"
	"github.com/tamnguyendinh/avmatrix-go/internal/lbugruntime"
	"github.com/tamnguyendinh/avmatrix-go/internal/repo"
	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

var (
	mcpProcessIDLiteralPattern = regexp.MustCompile(`Process\s*\{\s*id\s*:\s*'((?:[^']|'')*)'`)
	mcpProcessIDInPattern      = regexp.MustCompile(`p\.id\s+IN\s+\[([^\]]*)\]`)
	mcpFromIDInPattern         = regexp.MustCompile(`from\.id\s+IN\s+\[([^\]]*)\]`)
	mcpToIDInPattern           = regexp.MustCompile(`to\.id\s+IN\s+\[([^\]]*)\]`)
	mcpQuotedStringPattern     = regexp.MustCompile(`'((?:[^']|'')*)'`)
	mcpNodeLabelQueryPattern   = regexp.MustCompile(`MATCH\s+\(n:(Function|Class|Interface)\)`)
	mcpLimitPattern            = regexp.MustCompile(`(?i)\bLIMIT\s+(\d+)`)
)

var errUnsupportedMCPGraphQuery = errors.New("unsupported graph query in Go MCP graph adapter")

func mcpTools() []toolDefinition {
	return []toolDefinition{
		{
			Name:        "list_repos",
			Description: "List all indexed repositories available to AVmatrix.",
			InputSchema: map[string]any{
				"type":       "object",
				"properties": map[string]any{},
				"required":   []string{},
			},
		},
		{
			Name:        "query",
			Description: "Query the code knowledge graph for execution flows related to a concept.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query":           map[string]any{"type": "string", "description": "Natural language or keyword search query"},
					"task_context":    map[string]any{"type": "string", "description": "What you are working on. Helps ranking where supported."},
					"goal":            map[string]any{"type": "string", "description": "What you want to find. Helps ranking where supported."},
					"limit":           map[string]any{"type": "number", "description": "Max processes to return", "default": 5},
					"max_symbols":     map[string]any{"type": "number", "description": "Max symbols per process", "default": 10},
					"include_content": map[string]any{"type": "boolean", "description": "Include full symbol source code", "default": false},
					"repo":            map[string]any{"type": "string", "description": "Repository name or path"},
				},
				"required": []string{"query"},
			},
		},
		{
			Name:        "cypher",
			Description: "Execute a read-only Cypher query against the indexed graph.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query": map[string]any{"type": "string", "description": "Cypher query to execute"},
					"repo":  map[string]any{"type": "string", "description": "Repository name or path"},
				},
				"required": []string{"query"},
			},
		},
		{
			Name:        "context",
			Description: "360-degree view of a single code symbol with categorized refs and process participation.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"name":            map[string]any{"type": "string", "description": "Symbol name"},
					"uid":             map[string]any{"type": "string", "description": "Direct symbol UID from prior tool results"},
					"file_path":       map[string]any{"type": "string", "description": "File path to disambiguate common names"},
					"kind":            map[string]any{"type": "string", "description": "Kind filter such as Function, Class, Method, Interface, or Constructor"},
					"include_content": map[string]any{"type": "boolean", "description": "Include symbol source content", "default": false},
					"repo":            map[string]any{"type": "string", "description": "Repository name or path"},
				},
				"required": []string{},
			},
		},
		{
			Name:        "detect_changes",
			Description: "Analyze uncommitted git changes and find affected execution flows.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"scope":    map[string]any{"type": "string", "description": "unstaged, staged, all, or compare", "enum": []string{"unstaged", "staged", "all", "compare"}, "default": "unstaged"},
					"base_ref": map[string]any{"type": "string", "description": "Branch or commit for compare scope"},
					"repo":     map[string]any{"type": "string", "description": "Repository name or path"},
				},
				"required": []string{},
			},
		},
		{
			Name:        "rename",
			Description: "Multi-file coordinated rename using the knowledge graph.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"symbol_name": map[string]any{"type": "string", "description": "Current symbol name to rename"},
					"symbol_uid":  map[string]any{"type": "string", "description": "Direct symbol UID from prior tool results"},
					"new_name":    map[string]any{"type": "string", "description": "The new name for the symbol"},
					"file_path":   map[string]any{"type": "string", "description": "File path to disambiguate common names"},
					"dry_run":     map[string]any{"type": "boolean", "description": "Preview edits without modifying files", "default": true},
					"repo":        map[string]any{"type": "string", "description": "Repository name or path"},
				},
				"required": []string{"new_name"},
			},
		},
		{
			Name:        "impact",
			Description: "Analyze the blast radius of changing a code symbol.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"target":        map[string]any{"type": "string", "description": "Name of function, class, or file to analyze"},
					"target_uid":    map[string]any{"type": "string", "description": "Direct symbol UID from prior tool results"},
					"direction":     map[string]any{"type": "string", "description": "upstream or downstream", "enum": []string{"upstream", "downstream"}, "default": "upstream"},
					"file_path":     map[string]any{"type": "string", "description": "File path hint to disambiguate common names"},
					"kind":          map[string]any{"type": "string", "description": "Kind filter such as Function, Class, Method, Interface, or Constructor"},
					"maxDepth":      map[string]any{"type": "number", "description": "Max relationship depth", "default": 3},
					"relationTypes": map[string]any{"type": "array", "items": map[string]any{"type": "string", "enum": sortedImpactAllowedRelationTypes()}},
					"includeTests":  map[string]any{"type": "boolean", "description": "Include test files", "default": false},
					"minConfidence": map[string]any{"type": "number", "description": "Minimum confidence 0-1", "default": 0},
					"repo":          map[string]any{"type": "string", "description": "Repository name or path"},
				},
				"required": []string{"direction"},
				"oneOf":    []map[string]any{{"required": []string{"target"}}, {"required": []string{"target_uid"}}},
			},
		},
		{
			Name:        "route_map",
			Description: "Show API route mappings: which components/hooks fetch which API endpoints, and which handler files serve them.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"route": map[string]any{"type": "string", "description": "Filter by route path"},
					"repo":  map[string]any{"type": "string", "description": "Repository name or path"},
				},
				"required": []string{},
			},
		},
		{
			Name:        "tool_map",
			Description: "Show MCP/RPC tool definitions: which tools are defined, where they're handled, and their descriptions.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"tool": map[string]any{"type": "string", "description": "Filter by tool name"},
					"repo": map[string]any{"type": "string", "description": "Repository name or path"},
				},
				"required": []string{},
			},
		},
		{
			Name:        "shape_check",
			Description: "Check response shapes for API routes against their consumers' property accesses.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"route": map[string]any{"type": "string", "description": "Check a specific route"},
					"repo":  map[string]any{"type": "string", "description": "Repository name or path"},
				},
				"required": []string{},
			},
		},
		{
			Name:        "api_impact",
			Description: "Pre-change impact report for an API route handler.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"route": map[string]any{"type": "string", "description": "Route path"},
					"file":  map[string]any{"type": "string", "description": "Handler file path"},
					"repo":  map[string]any{"type": "string", "description": "Repository name or path"},
				},
				"required": []string{},
			},
		},
		{
			Name:        "group_list",
			Description: "List all configured repository groups, or return details for one group.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"name": map[string]any{"type": "string", "description": "Group name"},
				},
				"required": []string{},
			},
		},
		{
			Name:        "group_sync",
			Description: "Extract group contracts from indexed graph snapshots, exact-match cross-links, and write contracts.json.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"name":           map[string]any{"type": "string", "description": "Group name"},
					"allowStale":     map[string]any{"type": "boolean", "description": "Allow stale member indexes", "default": false},
					"verbose":        map[string]any{"type": "boolean", "description": "Include verbose sync behavior where supported", "default": false},
					"exactOnly":      map[string]any{"type": "boolean", "description": "Use exact matching only", "default": false},
					"skipEmbeddings": map[string]any{"type": "boolean", "description": "Skip embedding fallback", "default": false},
				},
				"required": []string{"name"},
			},
		},
		{
			Name:        "group_contracts",
			Description: "Inspect a group's Contract Registry with type, repo, and unmatched filters.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"name":          map[string]any{"type": "string", "description": "Group name"},
					"type":          map[string]any{"type": "string", "description": "Contract type filter such as http, grpc, topic, lib, or custom"},
					"repo":          map[string]any{"type": "string", "description": "Group repo path filter"},
					"unmatchedOnly": map[string]any{"type": "boolean", "description": "Only return contracts without matching cross-links", "default": false},
				},
				"required": []string{"name"},
			},
		},
		{
			Name:        "group_query",
			Description: "Search execution flows across all indexed repos in a group and merge ranked results.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"name":     map[string]any{"type": "string", "description": "Group name"},
					"query":    map[string]any{"type": "string", "description": "Natural language or keyword search query"},
					"subgroup": map[string]any{"type": "string", "description": "Optional group repo path prefix"},
					"limit":    map[string]any{"type": "number", "description": "Max merged results", "default": 5},
				},
				"required": []string{"name", "query"},
			},
		},
		{
			Name:        "group_status",
			Description: "Report index staleness and Contract Registry staleness for each repo in a group.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"name": map[string]any{"type": "string", "description": "Group name"},
				},
				"required": []string{"name"},
			},
		},
	}
}

func (s Server) queryTool(args map[string]any) (map[string]any, error) {
	query := strings.TrimSpace(stringArg(args, "query", ""))
	if query == "" {
		return nil, errors.New(`Missing "query" argument`)
	}
	g, err := s.graphForResource(stringArg(args, "repo", ""))
	if err != nil {
		return nil, err
	}
	limit := intArg(args, "limit", 5, 1, 50)
	processSteps := resourceProcessStepsByProcess(g)
	matches := rankedProcessMatches(g, query, limit, processSteps)
	symbols := make([]map[string]any, 0)
	for _, process := range matches {
		for _, step := range processSteps[process.ID] {
			symbols = append(symbols, map[string]any{
				"id":       step.ID,
				"process":  process.Label,
				"step":     step.Step,
				"name":     step.Name,
				"filePath": step.FilePath,
			})
		}
	}
	return map[string]any{
		"query":           query,
		"processes":       matches,
		"process_symbols": symbols,
		"definitions":     matchingDefinitionRows(g, query, limit),
	}, nil
}

func (s Server) cypherTool(args map[string]any) (map[string]any, error) {
	query := strings.TrimSpace(stringArg(args, "query", ""))
	if query == "" {
		query = strings.TrimSpace(stringArg(args, "cypher", ""))
	}
	if query == "" {
		return nil, errors.New(`Missing "query" argument`)
	}
	if err := lbugruntime.ValidateReadQuery(query); err != nil {
		return nil, err
	}
	entry, err := s.resolveResourceRepo(stringArg(args, "repo", ""))
	if err != nil {
		return nil, err
	}
	rows, err := s.runCypherRead(entry, query)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"row_count": len(rows),
		"rows":      rows,
		"markdown":  markdownRows(rows),
	}, nil
}

func (s Server) runCypherRead(entry repo.RegistryEntry, query string) ([]map[string]any, error) {
	if s.openReadRunner != nil {
		runner, err := s.openReadRunner(filepath.Join(storagePathForEntry(entry), "lbug"))
		if err == nil {
			defer runner.Close()
			rows, err := runner.QueryRows(query)
			if err != nil {
				return nil, err
			}
			return mcpRowsFromLbugRows(rows), nil
		}
		if !errors.Is(err, lbugnative.ErrUnavailable) {
			return nil, err
		}
	}

	g, err := loadResourceGraphSnapshot(filepath.Join(storagePathForEntry(entry), "graph.json"))
	if err != nil {
		return nil, err
	}
	return runMCPGraphQuery(g, query)
}

func mcpRowsFromLbugRows(rows []lbugruntime.Row) []map[string]any {
	out := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		copied := make(map[string]any, len(row))
		for key, value := range row {
			copied[key] = value
		}
		out = append(out, copied)
	}
	return out
}

func rankedProcessMatches(g *graph.Graph, query string, limit int, processSteps map[string][]resourceProcessStep) []resourceProcess {
	needle := strings.ToLower(query)
	processes := resourceProcessItems(g)
	type scored struct {
		process resourceProcess
		score   int
	}
	scoredItems := make([]scored, 0, len(processes))
	for _, process := range processes {
		score := containsScore(process.Label, needle) + containsScore(process.ProcessType, needle)
		for _, step := range processSteps[process.ID] {
			score += containsScore(step.Name, needle)
			score += containsScore(step.FilePath, needle)
		}
		if score > 0 {
			scoredItems = append(scoredItems, scored{process: process, score: score})
		}
	}
	if len(scoredItems) == 0 {
		for _, process := range processes[:minInt(len(processes), limit)] {
			scoredItems = append(scoredItems, scored{process: process, score: 0})
		}
	}
	sort.Slice(scoredItems, func(i, j int) bool {
		if scoredItems[i].score != scoredItems[j].score {
			return scoredItems[i].score > scoredItems[j].score
		}
		if scoredItems[i].process.StepCount != scoredItems[j].process.StepCount {
			return scoredItems[i].process.StepCount > scoredItems[j].process.StepCount
		}
		return scoredItems[i].process.Label < scoredItems[j].process.Label
	})
	out := make([]resourceProcess, 0, minInt(len(scoredItems), limit))
	for _, item := range scoredItems[:minInt(len(scoredItems), limit)] {
		out = append(out, item.process)
	}
	return out
}

func matchingDefinitionRows(g *graph.Graph, query string, limit int) []map[string]any {
	needle := strings.ToLower(query)
	rows := make([]map[string]any, 0)
	for _, node := range g.Nodes {
		if node.Label != scopeir.NodeClass && node.Label != scopeir.NodeInterface {
			continue
		}
		name := firstResourceNodeString(node, "name", "label", "heuristicLabel")
		filePath := resourceNodeString(node, "filePath")
		if containsScore(name, needle)+containsScore(filePath, needle) == 0 {
			continue
		}
		rows = append(rows, map[string]any{
			"id":       node.ID,
			"name":     name,
			"type":     string(node.Label),
			"filePath": filePath,
		})
		if len(rows) >= limit {
			break
		}
	}
	return rows
}

func runMCPGraphQuery(g *graph.Graph, cypher string) ([]map[string]any, error) {
	if strings.Contains(cypher, "STEP_IN_PROCESS") {
		return mcpProcessStepRows(g, cypher)
	}
	if strings.Contains(cypher, "CALLS") && strings.Contains(cypher, "from.id IN") && strings.Contains(cypher, "to.id IN") {
		return mcpCallEdgeRows(g, cypher)
	}
	if matches := mcpNodeLabelQueryPattern.FindStringSubmatch(cypher); len(matches) == 2 {
		if strings.Contains(strings.ToUpper(cypher), " WHERE ") {
			return nil, errUnsupportedMCPGraphQuery
		}
		return mcpNodeLabelRows(g, scopeir.NodeLabel(matches[1]), mcpQueryLimit(cypher, 50))
	}
	return nil, errUnsupportedMCPGraphQuery
}

func mcpProcessStepRows(g *graph.Graph, cypher string) ([]map[string]any, error) {
	processIDs := extractMCPProcessIDs(cypher)
	if len(processIDs) == 0 {
		return nil, errors.New("process query did not include process ids")
	}
	processSet := stringSet(processIDs)
	nodeByID := resourceGraphNodesByID(g)
	rows := make([]map[string]any, 0)
	for _, relationship := range g.Relationships {
		if relationship.Type != graph.RelStepInProcess || !processSet[relationship.TargetID] {
			continue
		}
		node, ok := nodeByID[relationship.SourceID]
		if !ok {
			continue
		}
		step := 0
		if relationship.Step != nil {
			step = *relationship.Step
		}
		rows = append(rows, map[string]any{
			"id":         node.ID,
			"name":       node.Properties["name"],
			"filePath":   node.Properties["filePath"],
			"stepNumber": step,
			"step":       step,
		})
	}
	sort.Slice(rows, func(i, j int) bool {
		left, _ := rows[i]["stepNumber"].(int)
		right, _ := rows[j]["stepNumber"].(int)
		if left != right {
			return left < right
		}
		return rows[i]["id"].(string) < rows[j]["id"].(string)
	})
	return rows, nil
}

func mcpCallEdgeRows(g *graph.Graph, cypher string) ([]map[string]any, error) {
	fromIDs := extractMCPQuotedList(cypher, mcpFromIDInPattern)
	toIDs := extractMCPQuotedList(cypher, mcpToIDInPattern)
	if len(fromIDs) == 0 || len(toIDs) == 0 {
		return nil, errors.New("CALLS query did not include from/to id lists")
	}
	fromSet := stringSet(fromIDs)
	toSet := stringSet(toIDs)
	rows := make([]map[string]any, 0)
	for _, relationship := range g.Relationships {
		if relationship.Type != graph.RelCalls || !fromSet[relationship.SourceID] || !toSet[relationship.TargetID] {
			continue
		}
		rows = append(rows, map[string]any{"fromId": relationship.SourceID, "toId": relationship.TargetID, "type": string(relationship.Type)})
	}
	return rows, nil
}

func mcpNodeLabelRows(g *graph.Graph, label scopeir.NodeLabel, limit int) ([]map[string]any, error) {
	rows := make([]map[string]any, 0)
	for _, node := range g.Nodes {
		if node.Label != label {
			continue
		}
		rows = append(rows, map[string]any{"id": node.ID, "name": node.Properties["name"], "filePath": node.Properties["filePath"], "path": node.Properties["filePath"]})
		if len(rows) >= limit {
			break
		}
	}
	return rows, nil
}

func extractMCPProcessIDs(cypher string) []string {
	if matches := mcpProcessIDLiteralPattern.FindStringSubmatch(cypher); len(matches) == 2 {
		return []string{unescapeMCPCypherString(matches[1])}
	}
	return extractMCPQuotedList(cypher, mcpProcessIDInPattern)
}

func extractMCPQuotedList(cypher string, listPattern *regexp.Regexp) []string {
	matches := listPattern.FindStringSubmatch(cypher)
	if len(matches) != 2 {
		return nil
	}
	values := make([]string, 0)
	for _, match := range mcpQuotedStringPattern.FindAllStringSubmatch(matches[1], -1) {
		if len(match) == 2 {
			values = append(values, unescapeMCPCypherString(match[1]))
		}
	}
	return values
}

func unescapeMCPCypherString(value string) string {
	return strings.ReplaceAll(value, "''", "'")
}

func mcpQueryLimit(cypher string, fallback int) int {
	matches := mcpLimitPattern.FindStringSubmatch(cypher)
	if len(matches) != 2 {
		return fallback
	}
	parsed, err := strconv.Atoi(matches[1])
	if err != nil || parsed < 1 {
		return fallback
	}
	return minInt(parsed, 500)
}

func markdownRows(rows []map[string]any) string {
	if len(rows) == 0 {
		return "_No rows_"
	}
	keys := make([]string, 0, len(rows[0]))
	for key := range rows[0] {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	lines := []string{"| " + strings.Join(keys, " | ") + " |", "| " + strings.Repeat("--- | ", len(keys))}
	for _, row := range rows {
		values := make([]string, 0, len(keys))
		for _, key := range keys {
			values = append(values, strings.ReplaceAll(fmt.Sprint(row[key]), "|", "\\|"))
		}
		lines = append(lines, "| "+strings.Join(values, " | ")+" |")
	}
	return strings.Join(lines, "\n")
}

func containsScore(value string, needle string) int {
	if value == "" || needle == "" {
		return 0
	}
	if strings.Contains(strings.ToLower(value), needle) {
		return 1
	}
	return 0
}

func intArg(args map[string]any, name string, fallback int, minValue int, maxValue int) int {
	if args == nil {
		return fallback
	}
	var value int
	switch raw := args[name].(type) {
	case int:
		value = raw
	case float64:
		value = int(raw)
	default:
		return fallback
	}
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

func stringSet(values []string) map[string]bool {
	out := make(map[string]bool, len(values))
	for _, value := range values {
		out[value] = true
	}
	return out
}
