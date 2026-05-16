package mcp

import (
	"fmt"
	"sort"
	"strings"

	"github.com/tamnguyendinh/avmatrix-go/internal/graph"
	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

func contextNeighborhood(g *graph.Graph, node graph.Node) (map[string][]map[string]any, map[string][]map[string]any, []map[string]any) {
	nodeByID := resourceGraphNodesByID(g)
	symbolID := node.ID
	incoming := make(map[string][]map[string]any)
	outgoing := make(map[string][]map[string]any)
	processes := make([]map[string]any, 0)
	constructorIDs := make(map[string]bool)
	fileIDs := make(map[string]bool)
	classLike := isContextClassLike(node)

	for _, relationship := range g.Relationships {
		if contextRelationshipTypes[string(relationship.Type)] {
			if relationship.SourceID == symbolID {
				if related, ok := nodeByID[relationship.TargetID]; ok {
					key := stringsLowerRelationship(relationship.Type)
					outgoing[key] = append(outgoing[key], contextRefPayload(related, relationship))
				}
			}
			if relationship.TargetID == symbolID {
				if related, ok := nodeByID[relationship.SourceID]; ok {
					key := stringsLowerRelationship(relationship.Type)
					incoming[key] = append(incoming[key], contextRefPayload(related, relationship))
				}
			}
		}
		if relationship.Type == graph.RelStepInProcess && relationship.SourceID == symbolID {
			if process, ok := nodeByID[relationship.TargetID]; ok {
				step := 0
				if relationship.Step != nil {
					step = *relationship.Step
				}
				processes = append(processes, map[string]any{
					"id":         process.ID,
					"name":       firstResourceNodeString(process, "heuristicLabel", "label", "name"),
					"step_index": step,
					"step_count": resourceNodeInt(process, "stepCount"),
				})
			}
		}
		if classLike {
			switch relationship.Type {
			case graph.RelHasMethod:
				if relationship.SourceID == symbolID {
					if related, ok := nodeByID[relationship.TargetID]; ok && related.Label == scopeir.NodeConstructor {
						constructorIDs[relationship.TargetID] = true
					}
				}
			case graph.RelDefines:
				if relationship.TargetID == symbolID {
					if related, ok := nodeByID[relationship.SourceID]; ok && related.Label == scopeir.NodeFile {
						fileIDs[relationship.SourceID] = true
					}
				}
			}
		}
	}

	if classLike && (len(constructorIDs) > 0 || len(fileIDs) > 0) {
		mergeContextRefs(incoming, contextClassLikeIncomingRefsFromSets(g, nodeByID, constructorIDs, fileIDs))
	}
	sortContextRefCategories(incoming)
	sortContextRefCategories(outgoing)
	sortContextProcesses(processes)
	return incoming, outgoing, processes
}

func contextClassLikeIncomingRefsFromSets(g *graph.Graph, nodeByID map[string]graph.Node, constructorIDs map[string]bool, fileIDs map[string]bool) map[string][]map[string]any {
	categories := make(map[string][]map[string]any)
	for _, relationship := range g.Relationships {
		if constructorIDs[relationship.TargetID] && contextConstructorIncomingType(relationship.Type) {
			if related, ok := nodeByID[relationship.SourceID]; ok {
				key := stringsLowerRelationship(relationship.Type)
				categories[key] = append(categories[key], contextRefPayload(related, relationship))
			}
			continue
		}
		if fileIDs[relationship.TargetID] && contextFileIncomingType(relationship.Type) {
			if related, ok := nodeByID[relationship.SourceID]; ok {
				key := stringsLowerRelationship(relationship.Type)
				categories[key] = append(categories[key], contextRefPayload(related, relationship))
			}
		}
	}
	sortContextRefCategories(categories)
	return categories
}

func sortContextCandidates(candidates []contextCandidate) {
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].Score != candidates[j].Score {
			return candidates[i].Score > candidates[j].Score
		}
		leftPath := resourceNodeString(candidates[i].Node, "filePath")
		rightPath := resourceNodeString(candidates[j].Node, "filePath")
		if leftPath != rightPath {
			return leftPath < rightPath
		}
		return candidates[i].Node.ID < candidates[j].Node.ID
	})
}

func sortContextRefCategories(categories map[string][]map[string]any) {
	for key := range categories {
		sort.Slice(categories[key], func(i, j int) bool {
			left, right := categories[key][i], categories[key][j]
			if left["filePath"] != right["filePath"] {
				return fmt.Sprint(left["filePath"]) < fmt.Sprint(right["filePath"])
			}
			return fmt.Sprint(left["uid"]) < fmt.Sprint(right["uid"])
		})
	}
}

func sortContextProcesses(processes []map[string]any) {
	sort.Slice(processes, func(i, j int) bool {
		leftStep, _ := processes[i]["step_index"].(int)
		rightStep, _ := processes[j]["step_index"].(int)
		if leftStep != rightStep {
			return leftStep < rightStep
		}
		return fmt.Sprint(processes[i]["id"]) < fmt.Sprint(processes[j]["id"])
	})
}

func stringsLowerRelationship(relationshipType graph.RelationshipType) string {
	return strings.ToLower(string(relationshipType))
}
