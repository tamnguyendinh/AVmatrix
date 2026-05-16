package mcp

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/tamnguyendinh/avmatrix-go/internal/graph"
	"github.com/tamnguyendinh/avmatrix-go/internal/repo"
	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

const canonicalResourceScheme = "avmatrix"

type resourceDefinition struct {
	URI         string `json:"uri"`
	Name        string `json:"name"`
	Description string `json:"description"`
	MimeType    string `json:"mimeType"`
}

type resourceTemplate struct {
	URITemplate string `json:"uriTemplate"`
	Name        string `json:"name"`
	Description string `json:"description"`
	MimeType    string `json:"mimeType"`
}

type resourceContent struct {
	URI      string `json:"uri"`
	MimeType string `json:"mimeType"`
	Text     string `json:"text"`
}

type resourceCluster struct {
	ID          string
	Label       string
	Cohesion    float64
	SymbolCount int
}

type resourceProcess struct {
	ID          string
	Label       string
	ProcessType string
	StepCount   int
}

type resourceClusterMember struct {
	Name     string
	Type     string
	FilePath string
}

type resourceProcessStep struct {
	ID       string
	Step     int
	Name     string
	FilePath string
}

type repoResourceRequest struct {
	RepoName     string
	ResourceType string
	Param        string
}

func resourceDefinitions() []resourceDefinition {
	return []resourceDefinition{
		{
			URI:         canonicalResourceScheme + "://repos",
			Name:        "All Indexed Repositories",
			Description: "List of all indexed repos with stats. Read this first to discover available repos.",
			MimeType:    "text/yaml",
		},
		{
			URI:         canonicalResourceScheme + "://setup",
			Name:        "AVmatrix Setup Content",
			Description: "Returns AGENTS.md content for all indexed repos. Useful for setup/onboarding.",
			MimeType:    "text/markdown",
		},
	}
}

func resourceTemplates() []resourceTemplate {
	return []resourceTemplate{
		{
			URITemplate: canonicalResourceScheme + "://repo/{name}/context",
			Name:        "Repo Overview",
			Description: "Codebase stats, staleness check, and available tools",
			MimeType:    "text/yaml",
		},
		{
			URITemplate: canonicalResourceScheme + "://repo/{name}/clusters",
			Name:        "Repo Modules",
			Description: "All functional areas (Leiden clusters)",
			MimeType:    "text/yaml",
		},
		{
			URITemplate: canonicalResourceScheme + "://repo/{name}/processes",
			Name:        "Repo Processes",
			Description: "All execution flows",
			MimeType:    "text/yaml",
		},
		{
			URITemplate: canonicalResourceScheme + "://repo/{name}/schema",
			Name:        "Graph Schema",
			Description: "Node/edge schema for Cypher queries",
			MimeType:    "text/yaml",
		},
		{
			URITemplate: canonicalResourceScheme + "://repo/{name}/cluster/{clusterName}",
			Name:        "Module Detail",
			Description: "Deep dive into a specific functional area",
			MimeType:    "text/yaml",
		},
		{
			URITemplate: canonicalResourceScheme + "://repo/{name}/process/{processName}",
			Name:        "Process Trace",
			Description: "Step-by-step execution trace",
			MimeType:    "text/yaml",
		},
	}
}

func (s Server) readResource(raw json.RawMessage) (map[string]any, error) {
	var params struct {
		URI string `json:"uri"`
	}
	if err := json.Unmarshal(raw, &params); err != nil {
		return nil, fmt.Errorf("Invalid resource read params: %w", err)
	}
	text, mimeType, err := s.readResourceText(params.URI)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"contents": []resourceContent{{
			URI:      params.URI,
			MimeType: mimeType,
			Text:     text,
		}},
	}, nil
}

func (s Server) readResourceText(uri string) (string, string, error) {
	if uri == canonicalResourceScheme+"://repos" {
		text, err := s.reposResource()
		return text, "text/yaml", err
	}
	if uri == canonicalResourceScheme+"://setup" {
		text, err := s.setupResource()
		return text, "text/markdown", err
	}

	request, ok, err := parseRepoResourceURI(uri)
	if err != nil {
		return "", "", err
	}
	if !ok {
		return "", "", fmt.Errorf("Unknown resource URI: %s", uri)
	}
	var text string
	switch request.ResourceType {
	case "context":
		text, err = s.contextResource(request.RepoName)
	case "clusters":
		text, err = s.clustersResource(request.RepoName)
	case "processes":
		text, err = s.processesResource(request.RepoName)
	case "schema":
		text = schemaResource()
	case "cluster":
		text, err = s.clusterDetailResource(request.RepoName, request.Param)
	case "process":
		text, err = s.processDetailResource(request.RepoName, request.Param)
	default:
		return "", "", fmt.Errorf("Unknown resource: %s", uri)
	}
	if err != nil {
		return "", "", err
	}
	return text, "text/yaml", nil
}

func parseRepoResourceURI(uri string) (repoResourceRequest, bool, error) {
	const prefix = canonicalResourceScheme + "://repo/"
	if !strings.HasPrefix(uri, prefix) {
		return repoResourceRequest{}, false, nil
	}
	rest := strings.TrimPrefix(uri, prefix)
	encodedName, resourcePath, found := strings.Cut(rest, "/")
	if !found || encodedName == "" || resourcePath == "" {
		return repoResourceRequest{}, false, nil
	}
	name, err := url.PathUnescape(encodedName)
	if err != nil {
		return repoResourceRequest{}, false, err
	}
	parts := strings.Split(resourcePath, "/")
	switch {
	case len(parts) == 1:
		return repoResourceRequest{RepoName: name, ResourceType: parts[0]}, true, nil
	case len(parts) == 2 && (parts[0] == "cluster" || parts[0] == "process"):
		param, err := url.PathUnescape(parts[1])
		if err != nil {
			return repoResourceRequest{}, false, err
		}
		return repoResourceRequest{RepoName: name, ResourceType: parts[0], Param: param}, true, nil
	default:
		return repoResourceRequest{}, false, nil
	}
}

func (s Server) contextResource(repoName string) (string, error) {
	entry, err := s.resolveResourceRepo(repoName)
	if err != nil {
		return "", err
	}

	stats := entry.Stats
	indexedAt := entry.IndexedAt
	lastCommit := entry.LastCommit
	meta, err := repo.LoadMeta(storagePathForEntry(entry))
	if err != nil {
		return "", err
	}
	if meta != nil {
		if meta.Stats != nil {
			stats = meta.Stats
		}
		if meta.IndexedAt != "" {
			indexedAt = meta.IndexedAt
		}
		if meta.LastCommit != "" {
			lastCommit = meta.LastCommit
		}
	}

	stalenessHint := mcpStalenessHint(entry.Path, lastCommit)
	resourceName := url.PathEscape(entry.Name)
	lines := []string{
		"project: " + entry.Name,
	}
	if stalenessHint != "" {
		lines = append(lines, "", fmt.Sprintf("staleness: %q", stalenessHint))
	}
	lines = append(lines,
		"repo_path: "+entry.Path,
		"indexed: "+valueOrUnknown(indexedAt),
		"commit: "+shortCommit(lastCommit),
		"",
		"stats:",
		fmt.Sprintf("  files: %d", statValue(stats, func(s *repo.Stats) *int { return s.Files })),
		fmt.Sprintf("  symbols: %d", statValue(stats, func(s *repo.Stats) *int { return s.Nodes })),
		fmt.Sprintf("  processes: %d", statValue(stats, func(s *repo.Stats) *int { return s.Processes })),
		"",
		"tools_available:",
		"  - query: Process-grouped code intelligence (execution flows related to a concept)",
		"  - context: 360-degree symbol view (categorized refs, process participation)",
		"  - impact: Blast radius analysis (what breaks if you change a symbol)",
		"  - detect_changes: Git-diff impact analysis (what do your changes affect)",
		"  - rename: Multi-file coordinated rename with confidence tags",
		"  - route_map: API route handlers and consumers",
		"  - tool_map: MCP/RPC tool definitions and handler files",
		"  - shape_check: API response shape drift against route consumers",
		"  - api_impact: Route pre-change risk, consumers, shape mismatches, and flows",
		"  - group_list: Configured repository groups and group details",
		"  - group_status: Group member index and contract registry staleness",
		"  - group_sync: Build group Contract Registry from indexed graph snapshots",
		"  - group_contracts: Contract Registry contracts and cross-links",
		"  - group_query: Cross-repo group execution flow search",
		"  - cypher: Raw graph queries",
		"  - list_repos: Discover all indexed repositories",
		"",
		"re_index: Run `avmatrix analyze` in terminal if data is stale",
		"",
		"resources_available:",
		"  - "+canonicalResourceScheme+"://repos: All indexed repositories",
		"  - "+canonicalResourceScheme+"://repo/"+resourceName+"/clusters: All functional areas",
		"  - "+canonicalResourceScheme+"://repo/"+resourceName+"/processes: All execution flows",
		"  - "+canonicalResourceScheme+"://repo/"+resourceName+"/cluster/{name}: Module details",
		"  - "+canonicalResourceScheme+"://repo/"+resourceName+"/process/{name}: Process trace",
	)
	return strings.Join(lines, "\n"), nil
}

func (s Server) reposResource() (string, error) {
	entries, err := s.store.ListRegistered(false)
	if err != nil {
		return "", err
	}
	if len(entries) == 0 {
		return "repos: []\n# No repositories indexed. Run: avmatrix analyze", nil
	}

	lines := []string{"repos:"}
	for _, entry := range entries {
		lines = append(lines, fmt.Sprintf("  - name: %q", entry.Name))
		lines = append(lines, fmt.Sprintf("    path: %q", entry.Path))
		lines = append(lines, fmt.Sprintf("    indexed: %q", entry.IndexedAt))
		lines = append(lines, fmt.Sprintf("    commit: %q", shortCommit(entry.LastCommit)))
		if entry.Stats != nil {
			lines = append(lines, fmt.Sprintf("    files: %d", statValue(entry.Stats, func(s *repo.Stats) *int { return s.Files })))
			lines = append(lines, fmt.Sprintf("    symbols: %d", statValue(entry.Stats, func(s *repo.Stats) *int { return s.Nodes })))
			lines = append(lines, fmt.Sprintf("    processes: %d", statValue(entry.Stats, func(s *repo.Stats) *int { return s.Processes })))
		}
	}
	if len(entries) > 1 {
		lines = append(lines, "")
		lines = append(lines, "# Multiple repos indexed. Use repo parameter in tool calls:")
		lines = append(lines, fmt.Sprintf("# query({query: %q, repo: %q})", "auth", entries[0].Name))
	}
	return strings.Join(lines, "\n"), nil
}

func (s Server) setupResource() (string, error) {
	entries, err := s.store.ListRegistered(false)
	if err != nil {
		return "", err
	}
	if len(entries) == 0 {
		return "# AVmatrix\n\nNo repositories indexed. Run: `avmatrix analyze` in a repository.", nil
	}

	sections := make([]string, 0, len(entries))
	for _, entry := range entries {
		stats := entry.Stats
		resourceName := url.PathEscape(entry.Name)
		lines := []string{
			"# AVmatrix MCP - " + entry.Name,
			"",
			fmt.Sprintf("This project is indexed by AVmatrix as **%s** (%d symbols, %d relationships, %d execution flows).",
				entry.Name,
				statValue(stats, func(s *repo.Stats) *int { return s.Nodes }),
				statValue(stats, func(s *repo.Stats) *int { return s.Edges }),
				statValue(stats, func(s *repo.Stats) *int { return s.Processes }),
			),
			"",
			"## Tools",
			"",
			"| Tool | What it gives you |",
			"|------|-------------------|",
			"| `query` | Process-grouped code intelligence - execution flows related to a concept |",
			"| `context` | 360-degree symbol view - categorized refs, processes it participates in |",
			"| `impact` | Symbol blast radius - what breaks at depth 1/2/3 with confidence |",
			"| `detect_changes` | Git-diff impact - what do your current changes affect |",
			"| `rename` | Multi-file coordinated rename with confidence-tagged edits |",
			"| `route_map` | API route handlers, consumers, and linked flows |",
			"| `tool_map` | MCP/RPC tool definitions and handler files |",
			"| `shape_check` | API response shape drift against route consumers |",
			"| `api_impact` | Route pre-change risk, consumers, shape mismatches, and flows |",
			"| `group_list` | Configured repository groups and group details |",
			"| `group_status` | Group member index and contract registry staleness |",
			"| `group_sync` | Build group Contract Registry from indexed graph snapshots |",
			"| `group_contracts` | Contract Registry contracts and cross-links |",
			"| `group_query` | Cross-repo group execution flow search |",
			"| `cypher` | Raw graph queries |",
			"| `list_repos` | Discover indexed repos |",
			"",
			"## Resources",
			"",
			"- `" + canonicalResourceScheme + "://repo/" + resourceName + "/context` - Stats, staleness check",
			"- `" + canonicalResourceScheme + "://repo/" + resourceName + "/clusters` - All functional areas",
			"- `" + canonicalResourceScheme + "://repo/" + resourceName + "/processes` - All execution flows",
			"- `" + canonicalResourceScheme + "://repo/" + resourceName + "/schema` - Graph schema for Cypher",
		}
		sections = append(sections, strings.Join(lines, "\n"))
	}
	return strings.Join(sections, "\n\n---\n\n"), nil
}

func schemaResource() string {
	return `# AVmatrix Graph Schema

nodes:
  - File: Source code files
  - Folder: Directory containers
  - Function: Functions and arrow functions
  - Class: Class definitions
  - Interface: Interface/type definitions
  - Method: Class methods
  - CodeElement: Catch-all for other code elements
  - Community: Auto-detected functional area
  - Process: Execution flow trace

additional_node_types: "Multi-language: Struct, Enum, Macro, Typedef, Union, Namespace, Trait, Impl, TypeAlias, Const, Static, Property, Record, Delegate, Annotation, Constructor, Template, Module"

node_properties:
  common: "name (STRING), filePath (STRING), startLine (INT32), endLine (INT32)"
  Community: "heuristicLabel (STRING), cohesion (DOUBLE), symbolCount (INT32), keywords (STRING[]), description (STRING)"
  Process: "heuristicLabel (STRING), processType (STRING), stepCount (INT32), communities (STRING[]), entryPointId (STRING), terminalId (STRING)"

relationships:
  - CONTAINS: File/Folder contains child
  - DEFINES: File defines a symbol
  - CALLS: Function/method invocation
  - IMPORTS: Module imports
  - USES: Symbol uses a type/imported symbol dependency
  - INHERITS: Normalized scope-resolved inheritance or heritage dependency
  - EXTENDS: Class inheritance
  - IMPLEMENTS: Interface implementation
  - HAS_METHOD: Class/Struct/Interface owns a Method
  - HAS_PROPERTY: Class/Struct/Interface owns a Property
  - ACCESSES: Function/Method reads or writes a Property
  - METHOD_OVERRIDES: Method overrides another Method
  - METHOD_IMPLEMENTS: ConcreteMethod implements InterfaceMethod
  - MEMBER_OF: Symbol belongs to community
  - STEP_IN_PROCESS: Symbol is step N in process

relationship_table: "All relationships use a single CodeRelation table with a 'type' property. Properties: type, confidence, reason, step, resolutionSource, evidence, fileHash"

example_queries:
  find_callers: |
    MATCH (caller)-[:CodeRelation {type: 'CALLS'}]->(f:Function {name: "myFunc"})
    RETURN caller.name, caller.filePath

  trace_process: |
    MATCH (s)-[r:CodeRelation {type: 'STEP_IN_PROCESS'}]->(p:Process)
    WHERE p.heuristicLabel = "LoginFlow"
    RETURN s.name, r.step
    ORDER BY r.step
`
}

func (s Server) clustersResource(repoName string) (string, error) {
	g, err := s.graphForResource(repoName)
	if err != nil {
		return "", err
	}
	clusters := aggregateResourceClusters(resourceClusterItems(g), true)
	if len(clusters) == 0 {
		return "modules: []\n# No functional areas detected. Run: avmatrix analyze", nil
	}

	lines := []string{"modules:"}
	for _, cluster := range clusters[:minInt(len(clusters), 20)] {
		lines = append(lines, fmt.Sprintf("  - name: %q", cluster.Label))
		lines = append(lines, fmt.Sprintf("    symbols: %d", cluster.SymbolCount))
		if cluster.Cohesion > 0 {
			lines = append(lines, fmt.Sprintf("    cohesion: %.0f%%", cluster.Cohesion*100))
		}
	}
	if len(clusters) > 20 {
		lines = append(lines, fmt.Sprintf("\n# Showing top 20 of %d modules. Use query() for deeper search.", len(clusters)))
	}
	return strings.Join(lines, "\n"), nil
}

func (s Server) clusterDetailResource(repoName string, clusterName string) (string, error) {
	g, err := s.graphForResource(repoName)
	if err != nil {
		return "", err
	}
	clusters := matchingResourceClusters(g, clusterName)
	if len(clusters) == 0 {
		return "error: Cluster not found", nil
	}
	aggregated := aggregateResourceClusters(clusters, false)
	cluster := aggregated[0]
	lines := []string{
		fmt.Sprintf("module: %q", cluster.Label),
		fmt.Sprintf("symbols: %d", cluster.SymbolCount),
	}
	if cluster.Cohesion > 0 {
		lines = append(lines, fmt.Sprintf("cohesion: %.0f%%", cluster.Cohesion*100))
	}

	members := resourceClusterMembers(g, clusters, 20)
	if len(members) > 0 {
		lines = append(lines, "", "members:")
		for _, member := range members {
			lines = append(lines, fmt.Sprintf("  - name: %s", member.Name))
			lines = append(lines, fmt.Sprintf("    type: %s", member.Type))
			lines = append(lines, fmt.Sprintf("    file: %s", member.FilePath))
		}
	}
	return strings.Join(lines, "\n"), nil
}

func (s Server) processesResource(repoName string) (string, error) {
	g, err := s.graphForResource(repoName)
	if err != nil {
		return "", err
	}
	processes := resourceProcessItems(g)
	if len(processes) == 0 {
		return "processes: []\n# No processes detected. Run: avmatrix analyze", nil
	}

	lines := []string{"processes:"}
	for _, process := range processes[:minInt(len(processes), 20)] {
		lines = append(lines, fmt.Sprintf("  - name: %q", process.Label))
		lines = append(lines, fmt.Sprintf("    type: %s", valueOrUnknown(process.ProcessType)))
		lines = append(lines, fmt.Sprintf("    steps: %d", process.StepCount))
	}
	if len(processes) > 20 {
		lines = append(lines, fmt.Sprintf("\n# Showing top 20 of %d processes. Use query() for deeper search.", len(processes)))
	}
	return strings.Join(lines, "\n"), nil
}

func (s Server) processDetailResource(repoName string, processName string) (string, error) {
	g, err := s.graphForResource(repoName)
	if err != nil {
		return "", err
	}
	process, found := findResourceProcess(g, processName)
	if !found {
		return "error: Process not found", nil
	}
	lines := []string{
		fmt.Sprintf("name: %q", process.Label),
		fmt.Sprintf("type: %s", valueOrUnknown(process.ProcessType)),
		fmt.Sprintf("step_count: %d", process.StepCount),
	}
	steps := resourceProcessSteps(g, process.ID)
	if len(steps) > 0 {
		lines = append(lines, "", "trace:")
		for _, step := range steps {
			lines = append(lines, fmt.Sprintf("  %d: %s (%s)", step.Step, step.Name, step.FilePath))
		}
	}
	return strings.Join(lines, "\n"), nil
}

func (s Server) graphForResource(repoName string) (*graph.Graph, error) {
	entry, err := s.resolveResourceRepo(repoName)
	if err != nil {
		return nil, err
	}
	graphPath := filepath.Join(storagePathForEntry(entry), "graph.json")
	if s.graphCache == nil {
		return loadResourceGraphSnapshot(graphPath)
	}
	return s.graphCache.graph(graphPath)
}

func (s Server) routeIndexForResource(repoName string) (*mcpRouteIndex, error) {
	entry, err := s.resolveResourceRepo(repoName)
	if err != nil {
		return nil, err
	}
	graphPath := filepath.Join(storagePathForEntry(entry), "graph.json")
	if s.graphCache == nil {
		g, err := loadResourceGraphSnapshot(graphPath)
		if err != nil {
			return nil, err
		}
		return buildMCPRouteIndex(g), nil
	}
	return s.graphCache.routeIndex(graphPath)
}

func loadResourceGraphSnapshot(path string) (*graph.Graph, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var g graph.Graph
	if err := json.NewDecoder(file).Decode(&g); err != nil {
		return nil, err
	}
	return &g, nil
}

func (s Server) resolveResourceRepo(query string) (repo.RegistryEntry, error) {
	entries, err := s.store.ListRegistered(false)
	if err != nil {
		return repo.RegistryEntry{}, err
	}
	if query == "" {
		if len(entries) == 1 {
			return entries[0], nil
		}
		return repo.RegistryEntry{}, fmt.Errorf("Repository not found. Run: avmatrix analyze")
	}
	entry, err := repo.ResolveEntry(entries, query)
	if err != nil {
		return repo.RegistryEntry{}, err
	}
	return entry, nil
}

func resourceClusterItems(g *graph.Graph) []resourceCluster {
	items := make([]resourceCluster, 0)
	for _, node := range g.Nodes {
		if node.Label != scopeir.NodeCommunity {
			continue
		}
		items = append(items, resourceClusterForNode(node))
	}
	return items
}

func matchingResourceClusters(g *graph.Graph, name string) []resourceCluster {
	items := make([]resourceCluster, 0)
	for _, node := range g.Nodes {
		if node.Label != scopeir.NodeCommunity {
			continue
		}
		item := resourceClusterForNode(node)
		if item.ID == name || item.Label == name {
			items = append(items, item)
		}
	}
	return items
}

func resourceClusterForNode(node graph.Node) resourceCluster {
	label := firstResourceNodeString(node, "heuristicLabel", "label", "name")
	return resourceCluster{
		ID:          node.ID,
		Label:       firstNonEmptyString(label, node.ID),
		Cohesion:    resourceNodeFloat(node, "cohesion"),
		SymbolCount: resourceNodeInt(node, "symbolCount"),
	}
}

func aggregateResourceClusters(raw []resourceCluster, filterTiny bool) []resourceCluster {
	type group struct {
		totalSymbols     int
		weightedCohesion float64
		largest          resourceCluster
	}
	groups := make(map[string]*group)
	for _, item := range raw {
		label := firstNonEmptyString(item.Label, "Unknown")
		current := groups[label]
		if current == nil {
			groups[label] = &group{
				totalSymbols:     item.SymbolCount,
				weightedCohesion: item.Cohesion * float64(item.SymbolCount),
				largest:          item,
			}
			continue
		}
		current.totalSymbols += item.SymbolCount
		current.weightedCohesion += item.Cohesion * float64(item.SymbolCount)
		if item.SymbolCount > current.largest.SymbolCount {
			current.largest = item
		}
	}

	out := make([]resourceCluster, 0, len(groups))
	for label, group := range groups {
		if filterTiny && group.totalSymbols < 5 {
			continue
		}
		item := resourceCluster{
			ID:          group.largest.ID,
			Label:       label,
			SymbolCount: group.totalSymbols,
		}
		if group.totalSymbols > 0 {
			item.Cohesion = group.weightedCohesion / float64(group.totalSymbols)
		}
		out = append(out, item)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].SymbolCount != out[j].SymbolCount {
			return out[i].SymbolCount > out[j].SymbolCount
		}
		return out[i].Label < out[j].Label
	})
	return out
}

func resourceClusterMembers(g *graph.Graph, clusters []resourceCluster, limit int) []resourceClusterMember {
	clusterIDs := make(map[string]bool, len(clusters))
	for _, cluster := range clusters {
		clusterIDs[cluster.ID] = true
	}
	nodeByID := resourceGraphNodesByID(g)
	seen := make(map[string]bool)
	members := make([]resourceClusterMember, 0)
	for _, relationship := range g.Relationships {
		if len(members) >= limit {
			break
		}
		if relationship.Type != graph.RelMemberOf || !clusterIDs[relationship.TargetID] || seen[relationship.SourceID] {
			continue
		}
		node, ok := nodeByID[relationship.SourceID]
		if !ok {
			continue
		}
		seen[relationship.SourceID] = true
		members = append(members, resourceClusterMember{
			Name:     firstResourceNodeString(node, "name", "label", "heuristicLabel"),
			Type:     string(node.Label),
			FilePath: resourceNodeString(node, "filePath"),
		})
	}
	sort.Slice(members, func(i, j int) bool {
		if members[i].FilePath != members[j].FilePath {
			return members[i].FilePath < members[j].FilePath
		}
		return members[i].Name < members[j].Name
	})
	return members
}

func resourceProcessItems(g *graph.Graph) []resourceProcess {
	items := make([]resourceProcess, 0)
	for _, node := range g.Nodes {
		if node.Label != scopeir.NodeProcess {
			continue
		}
		items = append(items, resourceProcessForNode(node))
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].StepCount != items[j].StepCount {
			return items[i].StepCount > items[j].StepCount
		}
		return items[i].ID < items[j].ID
	})
	return items
}

func findResourceProcess(g *graph.Graph, name string) (resourceProcess, bool) {
	for _, node := range g.Nodes {
		if node.Label != scopeir.NodeProcess {
			continue
		}
		item := resourceProcessForNode(node)
		if item.ID == name || item.Label == name {
			return item, true
		}
	}
	return resourceProcess{}, false
}

func resourceProcessForNode(node graph.Node) resourceProcess {
	label := firstResourceNodeString(node, "heuristicLabel", "label", "name")
	return resourceProcess{
		ID:          node.ID,
		Label:       firstNonEmptyString(label, node.ID),
		ProcessType: resourceNodeString(node, "processType"),
		StepCount:   resourceNodeInt(node, "stepCount"),
	}
}

func resourceProcessSteps(g *graph.Graph, processID string) []resourceProcessStep {
	nodeByID := resourceGraphNodesByID(g)
	steps := make([]resourceProcessStep, 0)
	for _, relationship := range g.Relationships {
		if relationship.Type != graph.RelStepInProcess || relationship.TargetID != processID {
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
		steps = append(steps, resourceProcessStep{
			ID:       node.ID,
			Step:     step,
			Name:     firstResourceNodeString(node, "name", "label", "heuristicLabel"),
			FilePath: resourceNodeString(node, "filePath"),
		})
	}
	sortResourceProcessSteps(steps)
	return steps
}

func resourceProcessStepsByProcess(g *graph.Graph) map[string][]resourceProcessStep {
	nodeByID := resourceGraphNodesByID(g)
	stepsByProcess := make(map[string][]resourceProcessStep)
	for _, relationship := range g.Relationships {
		if relationship.Type != graph.RelStepInProcess {
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
		stepsByProcess[relationship.TargetID] = append(stepsByProcess[relationship.TargetID], resourceProcessStep{
			ID:       node.ID,
			Step:     step,
			Name:     firstResourceNodeString(node, "name", "label", "heuristicLabel"),
			FilePath: resourceNodeString(node, "filePath"),
		})
	}
	for processID := range stepsByProcess {
		sortResourceProcessSteps(stepsByProcess[processID])
	}
	return stepsByProcess
}

func sortResourceProcessSteps(steps []resourceProcessStep) {
	sort.Slice(steps, func(i, j int) bool {
		if steps[i].Step != steps[j].Step {
			return steps[i].Step < steps[j].Step
		}
		return steps[i].Name < steps[j].Name
	})
}

func resourceGraphNodesByID(g *graph.Graph) map[string]graph.Node {
	out := make(map[string]graph.Node, len(g.Nodes))
	for _, node := range g.Nodes {
		out[node.ID] = node
	}
	return out
}

func storagePathForEntry(entry repo.RegistryEntry) string {
	if entry.StoragePath != "" {
		return entry.StoragePath
	}
	return repo.StoragePath(entry.Path)
}

func firstResourceNodeString(node graph.Node, keys ...string) string {
	values := make([]string, 0, len(keys))
	for _, key := range keys {
		values = append(values, resourceNodeString(node, key))
	}
	return firstNonEmptyString(values...)
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func resourceNodeString(node graph.Node, key string) string {
	value, _ := node.Properties[key].(string)
	return value
}

func resourceNodeInt(node graph.Node, key string) int {
	switch value := node.Properties[key].(type) {
	case int:
		return value
	case int32:
		return int(value)
	case int64:
		return int(value)
	case float64:
		return int(value)
	default:
		return 0
	}
}

func resourceNodeFloat(node graph.Node, key string) float64 {
	switch value := node.Properties[key].(type) {
	case float64:
		return value
	case float32:
		return float64(value)
	case int:
		return float64(value)
	default:
		return 0
	}
}

func statValue(stats *repo.Stats, selector func(*repo.Stats) *int) int {
	if stats == nil {
		return 0
	}
	value := selector(stats)
	if value == nil {
		return 0
	}
	return *value
}

func valueOrUnknown(value string) string {
	if value == "" {
		return "unknown"
	}
	return value
}

func mcpStalenessHint(repoPath string, lastCommit string) string {
	indexedCommit := strings.TrimSpace(lastCommit)
	if indexedCommit == "" {
		indexedCommit = "HEAD"
	}
	output, err := exec.Command("git", "-C", filepath.Clean(repoPath), "rev-list", "--count", indexedCommit+"..HEAD").Output()
	if err != nil {
		return ""
	}
	commitsBehind, err := strconv.Atoi(strings.TrimSpace(string(output)))
	if err != nil || commitsBehind <= 0 {
		return ""
	}
	plural := ""
	if commitsBehind > 1 {
		plural = "s"
	}
	return fmt.Sprintf("⚠️ Index is %d commit%s behind HEAD. Run analyze tool to update.", commitsBehind, plural)
}

func shortCommit(commit string) string {
	if commit == "" {
		return "unknown"
	}
	if len(commit) > 7 {
		return commit[:7]
	}
	return commit
}

func minInt(left int, right int) int {
	if left < right {
		return left
	}
	return right
}
