package mcp

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/tamnguyendinh/avmatrix-go/internal/graph"
	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

var contextRelationshipTypes = map[string]bool{
	string(graph.RelCalls):            true,
	string(graph.RelImports):          true,
	string(graph.RelUses):             true,
	string(graph.RelInherits):         true,
	string(graph.RelExtends):          true,
	string(graph.RelImplements):       true,
	string(graph.RelHasMethod):        true,
	string(graph.RelHasProperty):      true,
	string(graph.RelMethodOverrides):  true,
	"OVERRIDES":                       true,
	string(graph.RelMethodImplements): true,
	string(graph.RelAccesses):         true,
}

type contextCandidate struct {
	Node  graph.Node
	Score float64
}

func (s Server) contextTool(args map[string]any) (map[string]any, error) {
	payload, _, err := s.contextToolInternal(args, false)
	return payload, err
}

type contextToolProfile struct {
	RepoResolve      time.Duration
	TargetLookup     time.Duration
	NeighborhoodRead time.Duration
	SymbolPayload    time.Duration
	Formatting       time.Duration
}

func (s Server) contextToolProfiled(args map[string]any) (map[string]any, contextToolProfile, error) {
	return s.contextToolInternal(args, true)
}

func (s Server) contextToolInternal(args map[string]any, collectProfile bool) (map[string]any, contextToolProfile, error) {
	var profile contextToolProfile
	last := time.Time{}
	mark := func(target *time.Duration) {
		if !collectProfile {
			return
		}
		now := time.Now()
		if last.IsZero() {
			last = now
			return
		}
		*target += now.Sub(last)
		last = now
	}
	mark(nil)

	name := strings.TrimSpace(stringArg(args, "name", ""))
	uid := strings.TrimSpace(stringArg(args, "uid", ""))
	if name == "" && uid == "" {
		return map[string]any{"error": `Either "name" or "uid" parameter is required.`}, profile, nil
	}

	g, err := s.graphForResource(stringArg(args, "repo", ""))
	if err != nil {
		return nil, profile, err
	}
	mark(&profile.RepoResolve)

	candidates := contextCandidates(g, name, uid, stringArg(args, "file_path", ""), stringArg(args, "kind", ""))
	mark(&profile.TargetLookup)
	if len(candidates) == 0 {
		target := firstNonEmptyString(name, uid)
		return map[string]any{"error": fmt.Sprintf("Symbol '%s' not found", target)}, profile, nil
	}
	if len(candidates) > 1 && candidates[0].Score >= 0.9 && candidates[0].Score-candidates[1].Score > 0.09 {
		candidates = candidates[:1]
	}
	if len(candidates) > 1 {
		return map[string]any{
			"status":     "ambiguous",
			"message":    fmt.Sprintf("Found %d symbols matching '%s'. Use uid, file_path, or kind to disambiguate.", len(candidates), name),
			"candidates": contextCandidatePayloads(candidates),
		}, profile, nil
	}

	node := candidates[0].Node
	incoming, outgoing, processes := contextNeighborhood(g, node)
	mark(&profile.NeighborhoodRead)
	symbol := contextSymbolPayload(node, boolArg(args, "include_content", false))
	mark(&profile.SymbolPayload)
	payload := map[string]any{
		"status":    "found",
		"symbol":    symbol,
		"incoming":  incoming,
		"outgoing":  outgoing,
		"processes": processes,
	}
	mark(&profile.Formatting)
	return payload, profile, nil
}

func contextCandidates(g *graph.Graph, name string, uid string, filePathHint string, kindHint string) []contextCandidate {
	filePathHint = normalizeContextPath(filePathHint)
	kindHint = strings.TrimSpace(kindHint)
	candidates := make([]contextCandidate, 0)
	for _, node := range g.Nodes {
		score, ok := contextCandidateScore(node, name, uid, filePathHint, kindHint)
		if !ok {
			continue
		}
		candidates = append(candidates, contextCandidate{Node: node, Score: score})
	}
	sortContextCandidates(candidates)
	return candidates
}

func contextCandidateScore(node graph.Node, name string, uid string, filePathHint string, kindHint string) (float64, bool) {
	if uid != "" {
		if node.ID != uid {
			return 0, false
		}
		return 1, true
	}

	nodeName := firstResourceNodeString(node, "name", "label", "heuristicLabel")
	if nodeName == "" {
		return 0, false
	}
	score := 0.0
	switch {
	case nodeName == name:
		score = 0.9
	case strings.EqualFold(nodeName, name):
		score = 0.75
	case strings.Contains(strings.ToLower(nodeName), strings.ToLower(name)):
		score = 0.5
	default:
		return 0, false
	}

	if filePathHint != "" {
		nodePath := normalizeContextPath(resourceNodeString(node, "filePath"))
		if nodePath == "" || !strings.Contains(nodePath, filePathHint) {
			return 0, false
		}
		score += 0.2
	}
	if kindHint != "" {
		if string(node.Label) != kindHint {
			return 0, false
		}
		score += 0.2
	}
	return score, true
}

func contextCandidatePayloads(candidates []contextCandidate) []map[string]any {
	out := make([]map[string]any, 0, len(candidates))
	for _, candidate := range candidates {
		node := candidate.Node
		out = append(out, map[string]any{
			"uid":      node.ID,
			"name":     firstResourceNodeString(node, "name", "label", "heuristicLabel"),
			"kind":     string(node.Label),
			"filePath": resourceNodeString(node, "filePath"),
			"line":     resourceNodeInt(node, "startLine"),
			"score":    candidate.Score,
		})
	}
	return out
}

func contextSymbolPayload(node graph.Node, includeContent bool) map[string]any {
	payload := map[string]any{
		"uid":       node.ID,
		"name":      firstResourceNodeString(node, "name", "label", "heuristicLabel"),
		"kind":      string(node.Label),
		"filePath":  resourceNodeString(node, "filePath"),
		"startLine": resourceNodeInt(node, "startLine"),
		"endLine":   resourceNodeInt(node, "endLine"),
	}
	if includeContent {
		if content := resourceNodeString(node, "content"); content != "" {
			payload["content"] = content
		}
	}
	if metadata := contextMethodMetadata(node); len(metadata) > 0 {
		payload["methodMetadata"] = metadata
	}
	return payload
}

func contextMethodMetadata(node graph.Node) map[string]any {
	if node.Label != scopeir.NodeMethod && node.Label != scopeir.NodeFunction && node.Label != scopeir.NodeConstructor {
		return nil
	}
	keys := []string{
		"visibility",
		"isStatic",
		"isAbstract",
		"isFinal",
		"isVirtual",
		"isOverride",
		"isAsync",
		"isPartial",
		"returnType",
		"parameterCount",
		"isVariadic",
		"requiredParameterCount",
		"parameterTypes",
		"annotations",
	}
	out := make(map[string]any)
	for _, key := range keys {
		value, ok := node.Properties[key]
		if ok && value != nil {
			out[key] = value
		}
	}
	return out
}

func contextCategorizedRefs(g *graph.Graph, symbolID string, outgoing bool) map[string][]map[string]any {
	nodeByID := resourceGraphNodesByID(g)
	categories := make(map[string][]map[string]any)
	for _, relationship := range g.Relationships {
		if !contextRelationshipTypes[string(relationship.Type)] {
			continue
		}
		var relatedID string
		if outgoing {
			if relationship.SourceID != symbolID {
				continue
			}
			relatedID = relationship.TargetID
		} else {
			if relationship.TargetID != symbolID {
				continue
			}
			relatedID = relationship.SourceID
		}
		related, ok := nodeByID[relatedID]
		if !ok {
			continue
		}
		key := strings.ToLower(string(relationship.Type))
		categories[key] = append(categories[key], contextRefPayload(related, relationship))
	}
	for key := range categories {
		sortContextRefCategories(map[string][]map[string]any{key: categories[key]})
	}
	return categories
}

func isContextClassLike(node graph.Node) bool {
	return node.Label == scopeir.NodeClass || node.Label == scopeir.NodeInterface
}

func contextClassLikeIncomingRefs(g *graph.Graph, symbolID string) map[string][]map[string]any {
	nodeByID := resourceGraphNodesByID(g)
	constructorIDs := make(map[string]bool)
	fileIDs := make(map[string]bool)
	for _, relationship := range g.Relationships {
		switch relationship.Type {
		case graph.RelHasMethod:
			if relationship.SourceID == symbolID {
				if node, ok := nodeByID[relationship.TargetID]; ok && node.Label == scopeir.NodeConstructor {
					constructorIDs[relationship.TargetID] = true
				}
			}
		case graph.RelDefines:
			if relationship.TargetID == symbolID {
				if node, ok := nodeByID[relationship.SourceID]; ok && node.Label == scopeir.NodeFile {
					fileIDs[relationship.SourceID] = true
				}
			}
		}
	}

	categories := make(map[string][]map[string]any)
	for _, relationship := range g.Relationships {
		if constructorIDs[relationship.TargetID] && contextConstructorIncomingType(relationship.Type) {
			if related, ok := nodeByID[relationship.SourceID]; ok {
				key := strings.ToLower(string(relationship.Type))
				categories[key] = append(categories[key], contextRefPayload(related, relationship))
			}
			continue
		}
		if fileIDs[relationship.TargetID] && contextFileIncomingType(relationship.Type) {
			if related, ok := nodeByID[relationship.SourceID]; ok {
				key := strings.ToLower(string(relationship.Type))
				categories[key] = append(categories[key], contextRefPayload(related, relationship))
			}
		}
	}
	return categories
}

func contextConstructorIncomingType(relationshipType graph.RelationshipType) bool {
	switch relationshipType {
	case graph.RelCalls, graph.RelImports, graph.RelExtends, graph.RelImplements, graph.RelAccesses:
		return true
	default:
		return false
	}
}

func contextFileIncomingType(relationshipType graph.RelationshipType) bool {
	return relationshipType == graph.RelCalls || relationshipType == graph.RelImports
}

func mergeContextRefs(target map[string][]map[string]any, source map[string][]map[string]any) {
	for key, refs := range source {
		seen := make(map[string]bool, len(target[key])+len(refs))
		for _, existing := range target[key] {
			seen[contextRefKey(existing)] = true
		}
		for _, ref := range refs {
			dedupeKey := contextRefKey(ref)
			if seen[dedupeKey] {
				continue
			}
			seen[dedupeKey] = true
			target[key] = append(target[key], ref)
		}
		sortContextRefCategories(map[string][]map[string]any{key: target[key]})
	}
}

func contextRefKey(ref map[string]any) string {
	return fmt.Sprint(ref["uid"]) + "\x00" + fmt.Sprint(ref["kind"]) + "\x00" + fmt.Sprint(ref["filePath"])
}

func contextRefPayload(node graph.Node, relationship graph.Relationship) map[string]any {
	payload := map[string]any{
		"uid":      node.ID,
		"name":     firstResourceNodeString(node, "name", "label", "heuristicLabel"),
		"filePath": resourceNodeString(node, "filePath"),
		"kind":     string(node.Label),
	}
	if relationship.Confidence != 0 {
		payload["confidence"] = relationship.Confidence
	}
	if relationship.Reason != "" {
		payload["reason"] = relationship.Reason
	}
	if relationship.ResolutionSource != "" {
		payload["resolutionSource"] = relationship.ResolutionSource
	}
	if relationship.FileHash != "" {
		payload["fileHash"] = relationship.FileHash
	}
	if len(relationship.Evidence) > 0 {
		payload["evidence"] = relationship.Evidence
	}
	return payload
}

func contextProcessParticipation(g *graph.Graph, symbolID string) []map[string]any {
	nodeByID := resourceGraphNodesByID(g)
	processes := make([]map[string]any, 0)
	for _, relationship := range g.Relationships {
		if relationship.Type != graph.RelStepInProcess || relationship.SourceID != symbolID {
			continue
		}
		process, ok := nodeByID[relationship.TargetID]
		if !ok {
			continue
		}
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
	sort.Slice(processes, func(i, j int) bool {
		leftStep, _ := processes[i]["step_index"].(int)
		rightStep, _ := processes[j]["step_index"].(int)
		if leftStep != rightStep {
			return leftStep < rightStep
		}
		return fmt.Sprint(processes[i]["id"]) < fmt.Sprint(processes[j]["id"])
	})
	return processes
}

func normalizeContextPath(path string) string {
	return strings.ToLower(strings.ReplaceAll(strings.TrimSpace(path), "\\", "/"))
}

func boolArg(args map[string]any, name string, fallback bool) bool {
	if args == nil {
		return fallback
	}
	value, ok := args[name].(bool)
	if !ok {
		return fallback
	}
	return value
}
