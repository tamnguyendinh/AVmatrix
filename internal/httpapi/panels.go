package httpapi

import (
	"net/http"
	"path/filepath"
	"sort"
	"strconv"

	"github.com/tamnguyendinh/avmatrix-go/internal/graph"
	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

type processesResponse struct {
	Processes []processItem `json:"processes"`
}

type processItem struct {
	ID             string `json:"id"`
	Label          string `json:"label"`
	HeuristicLabel string `json:"heuristicLabel"`
	ProcessType    string `json:"processType"`
	StepCount      int    `json:"stepCount"`
}

type processDetailResponse struct {
	Process processItem   `json:"process"`
	Steps   []processStep `json:"steps"`
}

type processStep struct {
	Step     int    `json:"step"`
	Name     string `json:"name"`
	Type     string `json:"type"`
	FilePath string `json:"filePath"`
}

type clustersResponse struct {
	Clusters []clusterItem `json:"clusters"`
}

type clusterItem struct {
	ID             string  `json:"id"`
	Label          string  `json:"label"`
	HeuristicLabel string  `json:"heuristicLabel"`
	Cohesion       float64 `json:"cohesion"`
	SymbolCount    int     `json:"symbolCount"`
	SubCommunities int     `json:"subCommunities,omitempty"`
}

type clusterDetailResponse struct {
	Cluster clusterItem     `json:"cluster"`
	Members []clusterMember `json:"members"`
}

type clusterMember struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	FilePath string `json:"filePath"`
}

func (s Server) handleProcesses(w http.ResponseWriter, r *http.Request) {
	if !methodAllowed(w, r, http.MethodGet) {
		return
	}
	g, ok := s.graphForPanelRequest(w, r)
	if !ok {
		return
	}
	processes := processItems(g)
	limit := boundedQueryLimit(r.URL.Query().Get("limit"), 50, 1, 500)
	if len(processes) > limit {
		processes = processes[:limit]
	}
	writeJSON(w, http.StatusOK, processesResponse{Processes: processes})
}

func (s Server) handleProcess(w http.ResponseWriter, r *http.Request) {
	if !methodAllowed(w, r, http.MethodGet) {
		return
	}
	name := r.URL.Query().Get("name")
	if name == "" {
		writeError(w, http.StatusBadRequest, `Missing "name" query parameter`)
		return
	}
	g, ok := s.graphForPanelRequest(w, r)
	if !ok {
		return
	}
	node, found := findProcessNode(g, name)
	if !found {
		writeError(w, http.StatusNotFound, "Process '"+name+"' not found")
		return
	}
	writeJSON(w, http.StatusOK, processDetailResponse{
		Process: processItemForNode(node),
		Steps:   processStepsFor(g, node.ID),
	})
}

func (s Server) handleClusters(w http.ResponseWriter, r *http.Request) {
	if !methodAllowed(w, r, http.MethodGet) {
		return
	}
	g, ok := s.graphForPanelRequest(w, r)
	if !ok {
		return
	}
	clusters := aggregateClusterItems(rawClusterItems(g), true)
	limit := boundedQueryLimit(r.URL.Query().Get("limit"), 100, 1, 500)
	if len(clusters) > limit {
		clusters = clusters[:limit]
	}
	writeJSON(w, http.StatusOK, clustersResponse{Clusters: clusters})
}

func (s Server) handleCluster(w http.ResponseWriter, r *http.Request) {
	if !methodAllowed(w, r, http.MethodGet) {
		return
	}
	name := r.URL.Query().Get("name")
	if name == "" {
		writeError(w, http.StatusBadRequest, `Missing "name" query parameter`)
		return
	}
	g, ok := s.graphForPanelRequest(w, r)
	if !ok {
		return
	}
	rawClusters := matchingClusterItems(g, name)
	if len(rawClusters) == 0 {
		writeError(w, http.StatusNotFound, "Cluster '"+name+"' not found")
		return
	}
	aggregated := aggregateClusterItems(rawClusters, false)
	writeJSON(w, http.StatusOK, clusterDetailResponse{
		Cluster: aggregated[0],
		Members: clusterMembersFor(g, rawClusters, 30),
	})
}

func (s Server) graphForPanelRequest(w http.ResponseWriter, r *http.Request) (*graph.Graph, bool) {
	entry, status, message, err := s.resolveRequestedRepo(r)
	if err != nil {
		if status == http.StatusNotFound {
			message = "Repository not found"
		}
		writeError(w, status, message)
		return nil, false
	}
	g, err := loadGraphSnapshot(filepath.Join(storagePathFor(entry), "graph.json"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return nil, false
	}
	return g, true
}

func processItems(g *graph.Graph) []processItem {
	items := make([]processItem, 0)
	for _, node := range g.Nodes {
		if node.Label == scopeir.NodeProcess {
			items = append(items, processItemForNode(node))
		}
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].StepCount != items[j].StepCount {
			return items[i].StepCount > items[j].StepCount
		}
		return items[i].ID < items[j].ID
	})
	return items
}

func processItemForNode(node graph.Node) processItem {
	label := firstNodeString(node, "label", "heuristicLabel", "name")
	return processItem{
		ID:             node.ID,
		Label:          label,
		HeuristicLabel: firstNonEmpty(nodeString(node, "heuristicLabel"), label),
		ProcessType:    nodeString(node, "processType"),
		StepCount:      nodeInt(node, "stepCount"),
	}
}

func findProcessNode(g *graph.Graph, name string) (graph.Node, bool) {
	for _, node := range g.Nodes {
		if node.Label != scopeir.NodeProcess {
			continue
		}
		if node.ID == name || nodeString(node, "label") == name || nodeString(node, "heuristicLabel") == name {
			return node, true
		}
	}
	return graph.Node{}, false
}

func processStepsFor(g *graph.Graph, processID string) []processStep {
	nodeByID := graphNodesByID(g)
	steps := make([]processStep, 0)
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
		steps = append(steps, processStep{
			Step:     step,
			Name:     firstNodeString(node, "name", "label", "heuristicLabel"),
			Type:     string(node.Label),
			FilePath: nodeString(node, "filePath"),
		})
	}
	sort.Slice(steps, func(i, j int) bool {
		if steps[i].Step != steps[j].Step {
			return steps[i].Step < steps[j].Step
		}
		return steps[i].Name < steps[j].Name
	})
	return steps
}

func rawClusterItems(g *graph.Graph) []clusterItem {
	items := make([]clusterItem, 0)
	for _, node := range g.Nodes {
		if node.Label == scopeir.NodeCommunity {
			items = append(items, clusterItemForNode(node))
		}
	}
	return items
}

func matchingClusterItems(g *graph.Graph, name string) []clusterItem {
	items := make([]clusterItem, 0)
	for _, node := range g.Nodes {
		if node.Label != scopeir.NodeCommunity {
			continue
		}
		item := clusterItemForNode(node)
		if item.ID == name || item.Label == name || item.HeuristicLabel == name {
			items = append(items, item)
		}
	}
	return items
}

func clusterItemForNode(node graph.Node) clusterItem {
	label := firstNodeString(node, "heuristicLabel", "label", "name")
	return clusterItem{
		ID:             node.ID,
		Label:          label,
		HeuristicLabel: label,
		Cohesion:       nodeFloat(node, "cohesion"),
		SymbolCount:    nodeInt(node, "symbolCount"),
	}
}

func aggregateClusterItems(raw []clusterItem, filterTiny bool) []clusterItem {
	type group struct {
		ids              []string
		totalSymbols     int
		weightedCohesion float64
		largest          clusterItem
	}
	groups := make(map[string]*group)
	for _, item := range raw {
		label := firstNonEmpty(item.HeuristicLabel, item.Label, "Unknown")
		current := groups[label]
		if current == nil {
			groups[label] = &group{
				ids:              []string{item.ID},
				totalSymbols:     item.SymbolCount,
				weightedCohesion: item.Cohesion * float64(item.SymbolCount),
				largest:          item,
			}
			continue
		}
		current.ids = append(current.ids, item.ID)
		current.totalSymbols += item.SymbolCount
		current.weightedCohesion += item.Cohesion * float64(item.SymbolCount)
		if item.SymbolCount > current.largest.SymbolCount {
			current.largest = item
		}
	}

	out := make([]clusterItem, 0, len(groups))
	for label, group := range groups {
		if filterTiny && group.totalSymbols < 5 {
			continue
		}
		item := clusterItem{
			ID:             group.largest.ID,
			Label:          label,
			HeuristicLabel: label,
			SymbolCount:    group.totalSymbols,
			SubCommunities: len(group.ids),
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

func clusterMembersFor(g *graph.Graph, clusters []clusterItem, limit int) []clusterMember {
	clusterIDs := make(map[string]bool, len(clusters))
	for _, cluster := range clusters {
		clusterIDs[cluster.ID] = true
	}
	nodeByID := graphNodesByID(g)
	seen := make(map[string]bool)
	members := make([]clusterMember, 0)
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
		members = append(members, clusterMember{
			Name:     firstNodeString(node, "name", "label", "heuristicLabel"),
			Type:     string(node.Label),
			FilePath: nodeString(node, "filePath"),
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

func boundedQueryLimit(value string, fallback int, min int, max int) int {
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	if parsed < min {
		return min
	}
	if parsed > max {
		return max
	}
	return parsed
}

func firstNodeString(node graph.Node, keys ...string) string {
	values := make([]string, 0, len(keys))
	for _, key := range keys {
		values = append(values, nodeString(node, key))
	}
	return firstNonEmpty(values...)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func nodeString(node graph.Node, key string) string {
	value, _ := node.Properties[key].(string)
	return value
}

func nodeInt(node graph.Node, key string) int {
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

func nodeFloat(node graph.Node, key string) float64 {
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
