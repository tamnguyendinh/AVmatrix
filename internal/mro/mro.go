package mro

import (
	"github.com/tamnguyendinh/avmatrix-go/internal/graph"
	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

type Result struct {
	Metrics Metrics
}

type Metrics struct {
	ClassesAnalyzed       int `json:"classesAnalyzed,omitempty"`
	AmbiguityCount        int `json:"ambiguityCount,omitempty"`
	MethodOverrides       int `json:"methodOverrides,omitempty"`
	MethodImplementsEdges int `json:"methodImplementsEdges,omitempty"`
}

type adjacency struct {
	parentMap       map[string][]string
	methodMap       map[string][]string
	parentEdgeTypes map[string]map[string]graph.RelationshipType
}

type methodDef struct {
	ClassID   string
	ClassName string
	MethodID  string
}

func Apply(g *graph.Graph) Result {
	if g == nil {
		return Result{}
	}
	adj := buildAdjacency(g)
	metrics := Metrics{}
	metrics.MethodOverrides, metrics.AmbiguityCount, metrics.ClassesAnalyzed = emitMethodOverrides(g, adj)
	metrics.MethodImplementsEdges = emitMethodImplementsEdges(g, adj)
	return Result{Metrics: metrics}
}

func buildAdjacency(g *graph.Graph) adjacency {
	adj := adjacency{
		parentMap:       map[string][]string{},
		methodMap:       map[string][]string{},
		parentEdgeTypes: map[string]map[string]graph.RelationshipType{},
	}
	for _, rel := range g.Relationships {
		switch rel.Type {
		case graph.RelExtends, graph.RelImplements:
			adj.parentMap[rel.SourceID] = appendUnique(adj.parentMap[rel.SourceID], rel.TargetID)
			if adj.parentEdgeTypes[rel.SourceID] == nil {
				adj.parentEdgeTypes[rel.SourceID] = map[string]graph.RelationshipType{}
			}
			adj.parentEdgeTypes[rel.SourceID][rel.TargetID] = rel.Type
		case graph.RelHasMethod:
			adj.methodMap[rel.SourceID] = appendUnique(adj.methodMap[rel.SourceID], rel.TargetID)
		}
	}
	return adj
}

func emitMethodOverrides(g *graph.Graph, adj adjacency) (int, int, int) {
	overrideEdges := 0
	ambiguities := 0
	classesAnalyzed := 0
	for classID, parents := range adj.parentMap {
		if len(parents) == 0 {
			continue
		}
		classNode, ok := g.GetNode(classID)
		if !ok || !isOwnerLabel(classNode.Label) {
			continue
		}
		classesAnalyzed++
		mroOrder := gatherAncestors(classID, adj.parentMap)
		methodsByName := methodsByName(g, adj.methodMap, mroOrder)
		for methodName, defs := range methodsByName {
			if len(defs) < 2 || ownsMethod(g, adj.methodMap[classID], methodName) {
				continue
			}
			winner := defs[0]
			if winner.MethodID == "" {
				ambiguities++
				continue
			}
			rel := graph.Relationship{
				ID:         graph.GenerateID(string(graph.RelMethodOverrides), classID+"->"+winner.MethodID),
				SourceID:   classID,
				TargetID:   winner.MethodID,
				Type:       graph.RelMethodOverrides,
				Confidence: 0.9,
				Reason:     "first definition: " + winner.ClassName + "::" + methodName,
			}
			if addRelationship(g, rel) {
				overrideEdges++
			}
		}
	}
	return overrideEdges, ambiguities, classesAnalyzed
}

func methodsByName(g *graph.Graph, methodMap map[string][]string, ancestors []string) map[string][]methodDef {
	result := map[string][]methodDef{}
	seenMethods := map[string]struct{}{}
	for _, ancestorID := range ancestors {
		ancestor, ok := g.GetNode(ancestorID)
		if !ok {
			continue
		}
		for _, methodID := range methodMap[ancestorID] {
			if _, seen := seenMethods[methodID]; seen {
				continue
			}
			method, ok := g.GetNode(methodID)
			if !ok || method.Label == scopeir.NodeProperty {
				continue
			}
			name := stringProperty(method, "name")
			if name == "" {
				continue
			}
			seenMethods[methodID] = struct{}{}
			result[name] = append(result[name], methodDef{
				ClassID:   ancestorID,
				ClassName: firstNonEmpty(stringProperty(ancestor, "name"), ancestorID),
				MethodID:  methodID,
			})
		}
	}
	return result
}

func emitMethodImplementsEdges(g *graph.Graph, adj adjacency) int {
	edgeCount := 0
	for classID := range adj.parentMap {
		classNode, ok := g.GetNode(classID)
		if !ok || classNode.Label == scopeir.NodeInterface || classNode.Label == scopeir.NodeTrait {
			continue
		}
		ownMethods := ownMethodsByName(g, adj.methodMap[classID])
		ancestors := gatherAncestors(classID, adj.parentMap)
		edgeTypes := transitiveEdgeTypes(classID, adj)
		emitted := map[string]struct{}{}
		for _, ancestorID := range ancestors {
			ancestorNode, ok := g.GetNode(ancestorID)
			if !ok {
				continue
			}
			if !isInterfaceLike(ancestorNode.Label) && edgeTypes[ancestorID] != graph.RelImplements {
				continue
			}
			for _, ancestorMethodID := range adj.methodMap[ancestorID] {
				ancestorMethod, ok := g.GetNode(ancestorMethodID)
				if !ok || ancestorMethod.Label == scopeir.NodeProperty {
					continue
				}
				name := stringProperty(ancestorMethod, "name")
				if name == "" {
					continue
				}
				winner, ok := matchingOwnMethod(g, ownMethods[name], ancestorMethod)
				if !ok {
					inherited, inheritedOK := inheritedMethod(g, adj, classID, ancestorMethod)
					if !inheritedOK {
						continue
					}
					winner = inherited
				}
				edgeKey := winner.MethodID + "->" + ancestorMethodID
				if _, seen := emitted[edgeKey]; seen {
					continue
				}
				emitted[edgeKey] = struct{}{}
				rel := graph.Relationship{
					ID:         graph.GenerateID(string(graph.RelMethodImplements), edgeKey),
					SourceID:   winner.MethodID,
					TargetID:   ancestorMethodID,
					Type:       graph.RelMethodImplements,
					Confidence: winner.Confidence,
					Reason:     "",
				}
				if addRelationship(g, rel) {
					edgeCount++
				}
			}
		}
	}
	return edgeCount
}

type candidateMethod struct {
	MethodID       string
	ParameterTypes []string
	ParameterCount *int
	Confidence     float64
}

func ownMethodsByName(g *graph.Graph, methodIDs []string) map[string][]candidateMethod {
	result := map[string][]candidateMethod{}
	for _, methodID := range methodIDs {
		method, ok := g.GetNode(methodID)
		if !ok || method.Label == scopeir.NodeProperty || boolProperty(method, "isAbstract") {
			continue
		}
		name := stringProperty(method, "name")
		if name == "" {
			continue
		}
		result[name] = append(result[name], candidateMethod{
			MethodID:       methodID,
			ParameterTypes: stringSliceProperty(method, "parameterTypes"),
			ParameterCount: intProperty(method, "parameterCount"),
			Confidence:     1,
		})
	}
	return result
}

func matchingOwnMethod(g *graph.Graph, candidates []candidateMethod, target graph.Node) (candidateMethod, bool) {
	targetTypes := stringSliceProperty(target, "parameterTypes")
	targetCount := intProperty(target, "parameterCount")
	matches := make([]candidateMethod, 0, len(candidates))
	for _, candidate := range candidates {
		ok, confident := parameterTypesMatch(candidate.ParameterTypes, targetTypes, candidate.ParameterCount, targetCount)
		if !ok {
			continue
		}
		if !confident {
			candidate.Confidence = 0.7
		}
		matches = append(matches, candidate)
	}
	if len(matches) != 1 {
		return candidateMethod{}, false
	}
	return matches[0], true
}

func inheritedMethod(g *graph.Graph, adj adjacency, classID string, target graph.Node) (candidateMethod, bool) {
	name := stringProperty(target, "name")
	if name == "" {
		return candidateMethod{}, false
	}
	targetTypes := stringSliceProperty(target, "parameterTypes")
	targetCount := intProperty(target, "parameterCount")
	excludeMethodID := target.ID
	queue := make([]string, 0)
	for _, parentID := range adj.parentMap[classID] {
		if adj.parentEdgeTypes[classID][parentID] == graph.RelExtends {
			if parent, ok := g.GetNode(parentID); ok && !isInterfaceLike(parent.Label) {
				queue = append(queue, parentID)
			}
		}
	}
	visited := map[string]struct{}{}
	for len(queue) > 0 {
		level := queue
		queue = nil
		matches := map[string]candidateMethod{}
		for _, ancestorID := range level {
			if _, seen := visited[ancestorID]; seen {
				continue
			}
			visited[ancestorID] = struct{}{}
			for _, methodID := range adj.methodMap[ancestorID] {
				method, ok := g.GetNode(methodID)
				if !ok || method.Label == scopeir.NodeProperty || boolProperty(method, "isAbstract") || stringProperty(method, "name") != name {
					continue
				}
				ok, confident := parameterTypesMatch(stringSliceProperty(method, "parameterTypes"), targetTypes, intProperty(method, "parameterCount"), targetCount)
				if ok {
					confidence := 1.0
					if !confident {
						confidence = 0.7
					}
					matches[methodID] = candidateMethod{MethodID: methodID, ParameterTypes: stringSliceProperty(method, "parameterTypes"), ParameterCount: intProperty(method, "parameterCount"), Confidence: confidence}
				}
			}
			for _, nextID := range adj.parentMap[ancestorID] {
				if adj.parentEdgeTypes[ancestorID][nextID] == graph.RelExtends {
					if parent, ok := g.GetNode(nextID); ok && isInterfaceLike(parent.Label) {
						continue
					}
					queue = append(queue, nextID)
				}
			}
		}
		if len(matches) == 1 {
			for _, match := range matches {
				return match, true
			}
		}
		if len(matches) > 1 {
			return candidateMethod{}, false
		}
	}

	implQueue := make([]string, 0)
	for _, parentID := range adj.parentMap[classID] {
		if adj.parentEdgeTypes[classID][parentID] == graph.RelImplements {
			implQueue = append(implQueue, parentID)
		}
	}
	implVisited := map[string]struct{}{}
	matches := map[string]candidateMethod{}
	for len(implQueue) > 0 {
		ancestorID := implQueue[0]
		implQueue = implQueue[1:]
		if _, seen := implVisited[ancestorID]; seen {
			continue
		}
		implVisited[ancestorID] = struct{}{}
		ancestor, ok := g.GetNode(ancestorID)
		if !ok || !isInterfaceLike(ancestor.Label) {
			continue
		}
		for _, methodID := range adj.methodMap[ancestorID] {
			if methodID == excludeMethodID {
				continue
			}
			method, ok := g.GetNode(methodID)
			if !ok || method.Label == scopeir.NodeProperty || boolProperty(method, "isAbstract") || stringProperty(method, "name") != name {
				continue
			}
			ok, confident := parameterTypesMatch(stringSliceProperty(method, "parameterTypes"), targetTypes, intProperty(method, "parameterCount"), targetCount)
			if ok {
				confidence := 1.0
				if !confident {
					confidence = 0.7
				}
				matches[methodID] = candidateMethod{MethodID: methodID, ParameterTypes: stringSliceProperty(method, "parameterTypes"), ParameterCount: intProperty(method, "parameterCount"), Confidence: confidence}
			}
		}
		for _, nextID := range adj.parentMap[ancestorID] {
			if _, seen := implVisited[nextID]; !seen {
				implQueue = append(implQueue, nextID)
			}
		}
	}
	if len(matches) == 1 {
		for _, match := range matches {
			return match, true
		}
	}
	return candidateMethod{}, false
}

func parameterTypesMatch(left []string, right []string, leftCount *int, rightCount *int) (bool, bool) {
	if (leftCount == nil) != (rightCount == nil) {
		return true, false
	}
	if len(left) == 0 || len(right) == 0 {
		if leftCount != nil && rightCount != nil {
			return *leftCount == *rightCount, *leftCount == *rightCount
		}
		return true, false
	}
	if len(left) != len(right) {
		return false, false
	}
	for index := range left {
		if left[index] != right[index] {
			return false, false
		}
	}
	return true, true
}

func gatherAncestors(classID string, parentMap map[string][]string) []string {
	result := make([]string, 0)
	seen := map[string]struct{}{}
	queue := append([]string(nil), parentMap[classID]...)
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		if _, ok := seen[current]; ok {
			continue
		}
		seen[current] = struct{}{}
		result = append(result, current)
		queue = append(queue, parentMap[current]...)
	}
	return result
}

func transitiveEdgeTypes(classID string, adj adjacency) map[string]graph.RelationshipType {
	result := map[string]graph.RelationshipType{}
	type queued struct {
		ID      string
		RelType graph.RelationshipType
	}
	queue := make([]queued, 0)
	for _, parentID := range adj.parentMap[classID] {
		relType := adj.parentEdgeTypes[classID][parentID]
		if relType == "" {
			relType = graph.RelExtends
		}
		result[parentID] = relType
		queue = append(queue, queued{ID: parentID, RelType: relType})
	}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		for _, parentID := range adj.parentMap[current.ID] {
			if _, ok := result[parentID]; ok {
				continue
			}
			result[parentID] = current.RelType
			queue = append(queue, queued{ID: parentID, RelType: current.RelType})
		}
	}
	return result
}

func ownsMethod(g *graph.Graph, methodIDs []string, methodName string) bool {
	for _, methodID := range methodIDs {
		method, ok := g.GetNode(methodID)
		if ok && stringProperty(method, "name") == methodName {
			return true
		}
	}
	return false
}

func addRelationship(g *graph.Graph, rel graph.Relationship) bool {
	_, existed := g.GetRelationship(rel.ID)
	g.AddRelationship(rel)
	return !existed
}

func appendUnique(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

func isOwnerLabel(label scopeir.NodeLabel) bool {
	switch label {
	case scopeir.NodeClass, scopeir.NodeInterface, scopeir.NodeStruct, scopeir.NodeTrait, scopeir.NodeRecord:
		return true
	default:
		return false
	}
}

func isInterfaceLike(label scopeir.NodeLabel) bool {
	return label == scopeir.NodeInterface || label == scopeir.NodeTrait
}

func stringProperty(node graph.Node, key string) string {
	if value, ok := node.Properties[key].(string); ok {
		return value
	}
	return ""
}

func stringSliceProperty(node graph.Node, key string) []string {
	switch value := node.Properties[key].(type) {
	case []string:
		return value
	case []any:
		out := make([]string, 0, len(value))
		for _, item := range value {
			if typed, ok := item.(string); ok {
				out = append(out, typed)
			}
		}
		return out
	default:
		return nil
	}
}

func intProperty(node graph.Node, key string) *int {
	switch value := node.Properties[key].(type) {
	case int:
		return &value
	case int32:
		converted := int(value)
		return &converted
	case int64:
		converted := int(value)
		return &converted
	case float64:
		converted := int(value)
		return &converted
	default:
		return nil
	}
}

func boolProperty(node graph.Node, key string) bool {
	if value, ok := node.Properties[key].(bool); ok {
		return value
	}
	return false
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
