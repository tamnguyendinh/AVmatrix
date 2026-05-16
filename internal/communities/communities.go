package communities

import (
	"math"
	"path"
	"sort"
	"strconv"
	"strings"

	"github.com/tamnguyendinh/avmatrix-go/internal/graph"
	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

var CommunityColors = []string{
	"#2563eb",
	"#16a34a",
	"#dc2626",
	"#9333ea",
	"#ea580c",
	"#0891b2",
	"#4f46e5",
	"#be123c",
	"#0d9488",
	"#ca8a04",
	"#7c3aed",
	"#15803d",
}

func CommunityColor(index int) string {
	if len(CommunityColors) == 0 {
		return ""
	}
	if index < 0 {
		index = -index
	}
	return CommunityColors[index%len(CommunityColors)]
}

type Result struct {
	Communities []Community
	Memberships []Membership
	Metrics     Metrics
}

type Community struct {
	ID             string
	HeuristicLabel string
	Cohesion       float64
	SymbolCount    int
	Members        []string
}

type Membership struct {
	NodeID      string
	CommunityID string
}

type Metrics struct {
	CommunitiesEmitted int     `json:"communitiesEmitted,omitempty"`
	MembershipsEmitted int     `json:"membershipsEmitted,omitempty"`
	NodesConsidered    int     `json:"nodesConsidered,omitempty"`
	EdgesConsidered    int     `json:"edgesConsidered,omitempty"`
	Modularity         float64 `json:"modularity,omitempty"`
}

func Apply(g *graph.Graph) Result {
	if g == nil {
		return Result{}
	}
	index := buildIndex(g)
	partitions := modularityPartitions(index)

	result := Result{Metrics: Metrics{
		NodesConsidered: len(index.nodes),
		EdgesConsidered: index.edgeCount,
		Modularity:      partitionModularity(partitions, index),
	}}
	for partitionIndex, members := range partitions {
		communityID := "comm_" + strconv.Itoa(partitionIndex)
		for _, memberID := range members {
			membership := Membership{NodeID: memberID, CommunityID: communityID}
			result.Memberships = append(result.Memberships, membership)
			g.AddRelationship(graph.Relationship{
				ID:               graph.GenerateID(string(graph.RelMemberOf), memberID+"->"+communityID),
				SourceID:         memberID,
				TargetID:         communityID,
				Type:             graph.RelMemberOf,
				Confidence:       1,
				Reason:           "leiden-algorithm",
				ResolutionSource: "community-detection",
			})
		}
		if len(members) < 2 {
			continue
		}
		community := Community{
			ID:             communityID,
			HeuristicLabel: heuristicLabel(members, g),
			Cohesion:       cohesion(members, index),
			SymbolCount:    len(members),
			Members:        append([]string(nil), members...),
		}
		result.Communities = append(result.Communities, community)
		g.AddNode(graph.Node{
			ID:    community.ID,
			Label: scopeir.NodeCommunity,
			Properties: graph.NodeProperties{
				"label":          community.HeuristicLabel,
				"heuristicLabel": community.HeuristicLabel,
				"keywords":       keywords(members, g),
				"description":    "Detected symbol community",
				"enrichedBy":     "leiden-modularity",
				"cohesion":       community.Cohesion,
				"symbolCount":    community.SymbolCount,
			},
		})
	}
	result.Metrics.CommunitiesEmitted = len(result.Communities)
	result.Metrics.MembershipsEmitted = len(result.Memberships)
	return result
}

const (
	largeGraphSymbolThreshold = 10_000
	largeGraphMinConfidence   = 0.5
	modularityEpsilon         = 1e-10
	leidenRNGSeed             = 0x5eedc0de
)

type communityIndex struct {
	nodes     map[string]graph.Node
	adjacency map[string]map[string]struct{}
	nodeIDs   []string
	edgeCount int
	large     bool
}

func buildIndex(g *graph.Graph) communityIndex {
	symbols := make(map[string]graph.Node)
	for _, node := range g.Nodes {
		if isCommunitySymbol(node.Label) {
			symbols[node.ID] = node
		}
	}
	large := len(symbols) > largeGraphSymbolThreshold

	type edge struct {
		source string
		target string
	}
	edges := make(map[string]edge)
	degrees := make(map[string]int)
	for _, relationship := range g.Relationships {
		if !isClusteringRelationship(relationship, large) {
			continue
		}
		if _, ok := symbols[relationship.SourceID]; !ok {
			continue
		}
		if _, ok := symbols[relationship.TargetID]; !ok {
			continue
		}
		source, target := relationship.SourceID, relationship.TargetID
		if source > target {
			source, target = target, source
		}
		key := source + "\x00" + target
		if _, exists := edges[key]; exists {
			continue
		}
		edges[key] = edge{source: source, target: target}
		degrees[source]++
		degrees[target]++
	}

	index := communityIndex{
		nodes:     make(map[string]graph.Node),
		adjacency: make(map[string]map[string]struct{}),
		large:     large,
	}
	for id, node := range symbols {
		degree := degrees[id]
		if degree == 0 {
			continue
		}
		if large && degree < 2 {
			continue
		}
		index.nodes[id] = node
		index.adjacency[id] = make(map[string]struct{})
		index.nodeIDs = append(index.nodeIDs, id)
	}
	sort.Strings(index.nodeIDs)

	edgeKeys := make([]string, 0, len(edges))
	for key := range edges {
		edgeKeys = append(edgeKeys, key)
	}
	sort.Strings(edgeKeys)
	for _, key := range edgeKeys {
		edge := edges[key]
		if _, ok := index.nodes[edge.source]; !ok {
			continue
		}
		if _, ok := index.nodes[edge.target]; !ok {
			continue
		}
		index.adjacency[edge.source][edge.target] = struct{}{}
		index.adjacency[edge.target][edge.source] = struct{}{}
		index.edgeCount++
	}
	return index
}

func modularityPartitions(index communityIndex) [][]string {
	if len(index.nodeIDs) == 0 || index.edgeCount == 0 {
		return nil
	}

	positionByID := make(map[string]int, len(index.nodeIDs))
	for position, nodeID := range index.nodeIDs {
		positionByID[nodeID] = position
	}
	adjacency := make([][]int, len(index.nodeIDs))
	degrees := make([]float64, len(index.nodeIDs))
	for position, nodeID := range index.nodeIDs {
		neighbors := sortedNeighbors(index.adjacency[nodeID])
		for _, neighborID := range neighbors {
			neighborPosition, ok := positionByID[neighborID]
			if ok {
				adjacency[position] = append(adjacency[position], neighborPosition)
			}
		}
		degrees[position] = float64(len(adjacency[position]))
	}

	communities := make([]int, len(index.nodeIDs))
	totals := make([]float64, len(index.nodeIDs))
	sizes := make([]int, len(index.nodeIDs))
	for i := range index.nodeIDs {
		communities[i] = i
		totals[i] = degrees[i]
		sizes[i] = 1
	}

	maxPasses := 20
	resolution := 1.0
	if index.large {
		maxPasses = 3
		resolution = 2.0
	}
	doubleEdgeWeight := float64(index.edgeCount * 2)

	for pass := 0; pass < maxPasses; pass++ {
		moves := 0
		start := seededStart(pass, len(index.nodeIDs))
		for offset := range index.nodeIDs {
			nodeIndex := (start + offset) % len(index.nodeIDs)
			currentCommunity := communities[nodeIndex]
			degree := degrees[nodeIndex]

			neighborWeights := make(map[int]float64)
			for _, neighbor := range adjacency[nodeIndex] {
				neighborWeights[communities[neighbor]]++
			}

			totals[currentCommunity] -= degree
			sizes[currentCommunity]--

			bestCommunity := currentCommunity
			bestGain := insertionGain(neighborWeights[currentCommunity], degree, totals[currentCommunity], doubleEdgeWeight, resolution)
			targets := sortedCommunityTargets(neighborWeights)
			for _, targetCommunity := range targets {
				if targetCommunity == currentCommunity {
					continue
				}
				gain := insertionGain(neighborWeights[targetCommunity], degree, totals[targetCommunity], doubleEdgeWeight, resolution)
				if gain > bestGain+modularityEpsilon ||
					(math.Abs(gain-bestGain) < modularityEpsilon && tieBreakCommunity(bestCommunity, currentCommunity, targetCommunity)) {
					bestGain = gain
					bestCommunity = targetCommunity
				}
			}

			totals[bestCommunity] += degree
			sizes[bestCommunity]++
			if bestCommunity != currentCommunity {
				communities[nodeIndex] = bestCommunity
				moves++
			}
		}
		if moves == 0 {
			break
		}
	}

	grouped := make(map[int][]string)
	for nodeIndex, community := range communities {
		grouped[community] = append(grouped[community], index.nodeIDs[nodeIndex])
	}

	partitions := make([][]string, 0, len(grouped))
	for _, members := range grouped {
		sort.Strings(members)
		partitions = append(partitions, members)
	}
	sort.Slice(partitions, func(i int, j int) bool {
		if len(partitions[i]) != len(partitions[j]) {
			return len(partitions[i]) > len(partitions[j])
		}
		return partitions[i][0] < partitions[j][0]
	})
	return partitions
}

func insertionGain(weightToTarget float64, degree float64, targetTotal float64, doubleEdgeWeight float64, resolution float64) float64 {
	if doubleEdgeWeight == 0 {
		return 0
	}
	return weightToTarget - resolution*degree*targetTotal/doubleEdgeWeight
}

func sortedCommunityTargets(weights map[int]float64) []int {
	targets := make([]int, 0, len(weights))
	for target := range weights {
		targets = append(targets, target)
	}
	sort.Ints(targets)
	return targets
}

func tieBreakCommunity(bestCommunity int, currentCommunity int, targetCommunity int) bool {
	if bestCommunity == currentCommunity {
		return false
	}
	return targetCommunity > bestCommunity
}

func seededStart(pass int, nodeCount int) int {
	if nodeCount <= 1 {
		return 0
	}
	state := uint32(leidenRNGSeed) + uint32(pass)*0x6d2b79f5
	state += 0x6d2b79f5
	value := state
	value = (value ^ (value >> 15)) * (value | 1)
	value ^= value + (value^(value>>7))*(value|61)
	value = value ^ (value >> 14)
	return int(value % uint32(nodeCount))
}

func partitionModularity(partitions [][]string, index communityIndex) float64 {
	if len(partitions) == 0 || index.edgeCount == 0 {
		return 0
	}
	membership := make(map[string]int)
	for communityIndex, members := range partitions {
		for _, member := range members {
			membership[member] = communityIndex
		}
	}
	internalEdges := 0
	degreeTotals := make([]float64, len(partitions))
	for source, neighbors := range index.adjacency {
		sourceCommunity, ok := membership[source]
		if !ok {
			continue
		}
		degreeTotals[sourceCommunity] += float64(len(neighbors))
		for target := range neighbors {
			if source >= target {
				continue
			}
			if membership[target] == sourceCommunity {
				internalEdges++
			}
		}
	}
	edgeWeight := float64(index.edgeCount)
	score := float64(internalEdges) / edgeWeight
	resolution := 1.0
	if index.large {
		resolution = 2.0
	}
	for _, total := range degreeTotals {
		fraction := total / (2 * edgeWeight)
		score -= resolution * fraction * fraction
	}
	return score
}

func sortedNeighbors(neighbors map[string]struct{}) []string {
	out := make([]string, 0, len(neighbors))
	for neighbor := range neighbors {
		out = append(out, neighbor)
	}
	sort.Strings(out)
	return out
}

func heuristicLabel(members []string, g *graph.Graph) string {
	folderCounts := make(map[string]int)
	names := make([]string, 0, len(members))
	for _, memberID := range members {
		node, ok := g.GetNode(memberID)
		if !ok {
			continue
		}
		filePath := stringProperty(node, "filePath")
		folder := specificFolder(filePath)
		if folder != "" {
			folderCounts[folder]++
		}
		name := stringProperty(node, "name")
		if name != "" {
			names = append(names, name)
		}
	}
	if folder := mostCommon(folderCounts); folder != "" {
		return capitalize(folder)
	}
	sort.Strings(names)
	if prefix := commonPrefix(names); len(prefix) > 2 {
		return capitalize(prefix)
	}
	return "Cluster"
}

func cohesion(members []string, index communityIndex) float64 {
	memberSet := make(map[string]struct{}, len(members))
	for _, member := range members {
		memberSet[member] = struct{}{}
	}
	internal := 0
	total := 0
	for _, member := range members {
		for neighbor := range index.adjacency[member] {
			total++
			if _, ok := memberSet[neighbor]; ok {
				internal++
			}
		}
	}
	if total == 0 {
		return 1
	}
	return float64(internal) / float64(total)
}

func keywords(members []string, g *graph.Graph) []string {
	seen := make(map[string]struct{})
	for _, memberID := range members {
		node, ok := g.GetNode(memberID)
		if !ok {
			continue
		}
		if name := stringProperty(node, "name"); name != "" {
			seen[strings.ToLower(name)] = struct{}{}
		}
		if folder := specificFolder(stringProperty(node, "filePath")); folder != "" {
			seen[strings.ToLower(folder)] = struct{}{}
		}
	}
	out := make([]string, 0, len(seen))
	for keyword := range seen {
		out = append(out, keyword)
	}
	sort.Strings(out)
	if len(out) > 8 {
		return out[:8]
	}
	return out
}

func specificFolder(filePath string) string {
	if filePath == "" {
		return ""
	}
	dir := path.Dir(strings.ReplaceAll(filePath, "\\", "/"))
	if dir == "." || dir == "/" {
		return ""
	}
	folder := path.Base(dir)
	switch strings.ToLower(folder) {
	case "src", "lib", "core", "utils", "common", "shared", "helpers":
		return ""
	default:
		return folder
	}
}

func mostCommon(counts map[string]int) string {
	best := ""
	bestCount := 0
	keys := make([]string, 0, len(counts))
	for key := range counts {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if counts[key] > bestCount {
			best = key
			bestCount = counts[key]
		}
	}
	return best
}

func commonPrefix(values []string) string {
	if len(values) == 0 {
		return ""
	}
	first := values[0]
	last := values[len(values)-1]
	index := 0
	for index < len(first) && index < len(last) && first[index] == last[index] {
		index++
	}
	return first[:index]
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

func isCommunitySymbol(label scopeir.NodeLabel) bool {
	switch label {
	case scopeir.NodeFunction, scopeir.NodeClass, scopeir.NodeMethod, scopeir.NodeInterface:
		return true
	default:
		return false
	}
}

func isClusteringRelationship(relationship graph.Relationship, large bool) bool {
	if relationship.SourceID == relationship.TargetID {
		return false
	}
	if large && relationship.Confidence > 0 && relationship.Confidence < largeGraphMinConfidence {
		return false
	}
	switch relationship.Type {
	case graph.RelCalls, graph.RelExtends, graph.RelImplements:
		return true
	default:
		return false
	}
}

func capitalize(value string) string {
	if value == "" {
		return value
	}
	return strings.ToUpper(value[:1]) + value[1:]
}
