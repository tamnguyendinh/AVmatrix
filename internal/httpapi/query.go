package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/tamnguyendinh/avmatrix-go/internal/graph"
	"github.com/tamnguyendinh/avmatrix-go/internal/repo"
	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

type queryRequestBody struct {
	Cypher string `json:"cypher"`
	Repo   string `json:"repo"`
}

type queryResponse struct {
	Result []map[string]any `json:"result"`
}

var (
	processIDLiteralPattern = regexp.MustCompile(`Process\s*\{\s*id\s*:\s*'((?:[^']|'')*)'`)
	processIDInPattern      = regexp.MustCompile(`p\.id\s+IN\s+\[([^\]]*)\]`)
	fromIDInPattern         = regexp.MustCompile(`from\.id\s+IN\s+\[([^\]]*)\]`)
	toIDInPattern           = regexp.MustCompile(`to\.id\s+IN\s+\[([^\]]*)\]`)
	quotedStringPattern     = regexp.MustCompile(`'((?:[^']|'')*)'`)
	nodeLabelQueryPattern   = regexp.MustCompile(`MATCH\s+\(n:(Function|Class|Interface)\)`)
	relationTypePattern     = regexp.MustCompile(`CodeRelation\s*\{\s*type\s*:\s*'((?:[^']|'')*)'`)
	limitPattern            = regexp.MustCompile(`(?i)\bLIMIT\s+(\d+)`)
)

func (s Server) handleQuery(w http.ResponseWriter, r *http.Request) {
	if !methodAllowed(w, r, http.MethodPost) {
		return
	}

	var body queryRequestBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON request body")
		return
	}
	body.Cypher = strings.TrimSpace(body.Cypher)
	if body.Cypher == "" {
		writeError(w, http.StatusBadRequest, `Missing "cypher" in request body`)
		return
	}

	entry, status, message, err := s.resolveQueryRepo(r, body.Repo)
	if err != nil {
		writeError(w, status, message)
		return
	}
	g, err := loadGraphSnapshot(filepath.Join(storagePathFor(entry), "graph.json"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	rows, err := runGraphPanelQuery(g, body.Cypher)
	if err != nil {
		if errors.Is(err, errUnsupportedGraphQuery) {
			writeError(w, http.StatusNotImplemented, err.Error())
			return
		}
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, queryResponse{Result: rows})
}

func (s Server) resolveQueryRepo(r *http.Request, bodyRepo string) (repo.RegistryEntry, int, string, error) {
	entries, err := s.store.ListRegistered(false)
	if err != nil {
		return repo.RegistryEntry{}, http.StatusInternalServerError, err.Error(), err
	}
	repoQuery := strings.TrimSpace(bodyRepo)
	if repoQuery == "" {
		repoQuery = requestedRepo(r)
	}
	return resolveRepoQuery(entries, repoQuery)
}

var errUnsupportedGraphQuery = errors.New("unsupported graph query in Go HTTP panel adapter")

func runGraphPanelQuery(g *graph.Graph, cypher string) ([]map[string]any, error) {
	if strings.Contains(cypher, "STEP_IN_PROCESS") {
		return processStepRows(g, cypher)
	}
	if strings.Contains(cypher, "CALLS") && strings.Contains(cypher, "from.id IN") && strings.Contains(cypher, "to.id IN") {
		return callEdgeRows(g, cypher)
	}
	if matches := nodeLabelQueryPattern.FindStringSubmatch(cypher); len(matches) == 2 {
		return nodeLabelRows(g, scopeir.NodeLabel(matches[1]), queryLimit(cypher, 50))
	}
	if matches := relationTypePattern.FindStringSubmatch(cypher); len(matches) == 2 {
		return relationshipRows(g, graph.RelationshipType(unescapeCypherString(matches[1])), queryLimit(cypher, 50))
	}
	return nil, errUnsupportedGraphQuery
}

func processStepRows(g *graph.Graph, cypher string) ([]map[string]any, error) {
	processIDs := extractProcessIDs(cypher)
	if len(processIDs) == 0 {
		return nil, errors.New("process query did not include process ids")
	}
	processSet := stringSet(processIDs)
	nodeByID := graphNodesByID(g)
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

func callEdgeRows(g *graph.Graph, cypher string) ([]map[string]any, error) {
	fromIDs := extractQuotedList(cypher, fromIDInPattern)
	toIDs := extractQuotedList(cypher, toIDInPattern)
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
		rows = append(rows, map[string]any{
			"fromId": relationship.SourceID,
			"toId":   relationship.TargetID,
			"type":   string(relationship.Type),
		})
	}
	return rows, nil
}

func nodeLabelRows(g *graph.Graph, label scopeir.NodeLabel, limit int) ([]map[string]any, error) {
	rows := make([]map[string]any, 0)
	for _, node := range g.Nodes {
		if node.Label != label {
			continue
		}
		rows = append(rows, map[string]any{
			"id":       node.ID,
			"name":     node.Properties["name"],
			"filePath": node.Properties["filePath"],
			"path":     node.Properties["filePath"],
		})
		if len(rows) >= limit {
			break
		}
	}
	return rows, nil
}

func relationshipRows(g *graph.Graph, relType graph.RelationshipType, limit int) ([]map[string]any, error) {
	nodeByID := graphNodesByID(g)
	rows := make([]map[string]any, 0)
	for _, relationship := range g.Relationships {
		if relationship.Type != relType {
			continue
		}
		source, sourceOK := nodeByID[relationship.SourceID]
		target, targetOK := nodeByID[relationship.TargetID]
		if !sourceOK || !targetOK {
			continue
		}
		rows = append(rows, map[string]any{
			"id":      source.ID,
			"fromId":  source.ID,
			"toId":    target.ID,
			"caller":  firstNodeString(source, "name", "label", "heuristicLabel"),
			"callee":  firstNodeString(target, "name", "label", "heuristicLabel"),
			"from":    firstNodeString(source, "name", "label", "heuristicLabel"),
			"imports": firstNodeString(target, "name", "label", "heuristicLabel"),
			"type":    string(relationship.Type),
		})
		if len(rows) >= limit {
			break
		}
	}
	return rows, nil
}

func extractProcessIDs(cypher string) []string {
	if matches := processIDLiteralPattern.FindStringSubmatch(cypher); len(matches) == 2 {
		return []string{unescapeCypherString(matches[1])}
	}
	return extractQuotedList(cypher, processIDInPattern)
}

func extractQuotedList(cypher string, listPattern *regexp.Regexp) []string {
	matches := listPattern.FindStringSubmatch(cypher)
	if len(matches) != 2 {
		return nil
	}
	values := make([]string, 0)
	for _, match := range quotedStringPattern.FindAllStringSubmatch(matches[1], -1) {
		if len(match) == 2 {
			values = append(values, unescapeCypherString(match[1]))
		}
	}
	return values
}

func unescapeCypherString(value string) string {
	return strings.ReplaceAll(value, "''", "'")
}

func queryLimit(cypher string, fallback int) int {
	matches := limitPattern.FindStringSubmatch(cypher)
	if len(matches) != 2 {
		return fallback
	}
	parsed, err := strconv.Atoi(matches[1])
	if err != nil || parsed < 1 {
		return fallback
	}
	if parsed > 500 {
		return 500
	}
	return parsed
}

func stringSet(values []string) map[string]bool {
	out := make(map[string]bool, len(values))
	for _, value := range values {
		out[value] = true
	}
	return out
}

func graphNodesByID(g *graph.Graph) map[string]graph.Node {
	out := make(map[string]graph.Node, len(g.Nodes))
	for _, node := range g.Nodes {
		out[node.ID] = node
	}
	return out
}
