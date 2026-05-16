package orm

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/tamnguyendinh/avmatrix-go/internal/graph"
	"github.com/tamnguyendinh/avmatrix-go/internal/scanner"
	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

var (
	prismaQueryPattern = regexp.MustCompile(`(?i)\bprisma\.([A-Za-z_][A-Za-z0-9_]*)\.(findMany|findFirst|findUnique|findUniqueOrThrow|findFirstOrThrow|create|createMany|update|updateMany|delete|deleteMany|upsert|count|aggregate|groupBy)\s*\(`)
	supabasePattern    = regexp.MustCompile(`(?i)\bsupabase\.from\s*\(\s*['"]([A-Za-z_][A-Za-z0-9_]*)['"]\s*\)\s*\.(select|insert|update|delete|upsert)\s*\(`)
)

type Result struct {
	Metrics Metrics
}

type Metrics struct {
	FilesScanned      int `json:"filesScanned,omitempty"`
	QueriesDetected   int `json:"queriesDetected,omitempty"`
	ModelCount        int `json:"modelCount,omitempty"`
	ModelNodesEmitted int `json:"modelNodesEmitted,omitempty"`
	QueriesEmitted    int `json:"queriesEmitted,omitempty"`
	Duplicates        int `json:"duplicates,omitempty"`
}

type query struct {
	FilePath string
	ORM      string
	Model    string
	Method   string
}

func Apply(g *graph.Graph, repoPath string, files []scanner.File) (Result, error) {
	if g == nil {
		return Result{}, nil
	}
	candidates := candidateFiles(files)
	result := Result{}
	queries := make([]query, 0)
	for _, file := range candidates {
		result.Metrics.FilesScanned++
		raw, err := os.ReadFile(filepath.Join(repoPath, filepath.FromSlash(file.Path)))
		if err != nil {
			return result, err
		}
		queries = append(queries, extractQueries(file.Path, string(raw))...)
	}
	result.Metrics.QueriesDetected = len(queries)
	processQueries(g, queries, &result.Metrics)
	return result, nil
}

func candidateFiles(files []scanner.File) []scanner.File {
	out := make([]scanner.File, 0)
	for _, file := range files {
		file.Path = normalizePath(file.Path)
		if file.Path == "" || !isSupportedLanguage(file.Language) {
			continue
		}
		lower := strings.ToLower(file.Path)
		if strings.Contains(lower, "node_modules/") || strings.Contains(lower, "/test/") || strings.Contains(lower, "__") {
			continue
		}
		out = append(out, file)
	}
	sort.Slice(out, func(i int, j int) bool { return out[i].Path < out[j].Path })
	return out
}

func extractQueries(filePath string, content string) []query {
	if !strings.Contains(content, "prisma.") && !strings.Contains(content, "supabase.from") {
		return nil
	}
	out := make([]query, 0)
	for _, match := range prismaQueryPattern.FindAllStringSubmatch(content, -1) {
		out = append(out, query{FilePath: filePath, ORM: "prisma", Model: match[1], Method: match[2]})
	}
	for _, match := range supabasePattern.FindAllStringSubmatch(content, -1) {
		out = append(out, query{FilePath: filePath, ORM: "supabase", Model: match[1], Method: match[2]})
	}
	return out
}

func processQueries(g *graph.Graph, queries []query, metrics *Metrics) {
	modelNodes := map[string]string{}
	seenEdges := map[string]struct{}{}
	for _, q := range queries {
		fileNodeID := graph.GenerateID(string(scopeir.NodeFile), q.FilePath)
		if _, ok := g.GetNode(fileNodeID); !ok {
			continue
		}
		modelKey := q.ORM + ":" + q.Model
		modelNodeID, ok := modelNodes[modelKey]
		if !ok {
			modelNodeID, ok = existingModelNode(g, q.Model)
			if !ok {
				modelNodeID = graph.GenerateID(string(scopeir.NodeCodeElement), modelKey)
				g.AddNode(graph.Node{
					ID:    modelNodeID,
					Label: scopeir.NodeCodeElement,
					Properties: graph.NodeProperties{
						"name":        q.Model,
						"filePath":    "",
						"description": q.ORM + " model/table: " + q.Model,
					},
				})
				metrics.ModelNodesEmitted++
			}
			modelNodes[modelKey] = modelNodeID
		}
		edgeKey := fileNodeID + "->" + modelNodeID + ":" + q.Method
		if _, seen := seenEdges[edgeKey]; seen {
			metrics.Duplicates++
			continue
		}
		seenEdges[edgeKey] = struct{}{}
		g.AddRelationship(graph.Relationship{
			ID:         graph.GenerateID(string(graph.RelQueries), edgeKey),
			SourceID:   fileNodeID,
			TargetID:   modelNodeID,
			Type:       graph.RelQueries,
			Confidence: 0.9,
			Reason:     q.ORM + "-" + q.Method,
		})
		metrics.QueriesEmitted++
	}
	metrics.ModelCount = len(modelNodes)
}

func existingModelNode(g *graph.Graph, model string) (string, bool) {
	for _, label := range []scopeir.NodeLabel{scopeir.NodeClass, scopeir.NodeInterface, scopeir.NodeCodeElement} {
		if id, ok := uniqueNodeByName(g, label, model); ok {
			return id, true
		}
	}
	return "", false
}

func uniqueNodeByName(g *graph.Graph, label scopeir.NodeLabel, name string) (string, bool) {
	if node, ok := g.GetNode(graph.GenerateID(string(label), name)); ok && node.Label == label {
		return node.ID, true
	}
	matches := make([]string, 0, 1)
	for _, node := range g.Nodes {
		if node.Label != label {
			continue
		}
		if nodeName, ok := node.Properties["name"].(string); ok && nodeName == name {
			matches = append(matches, node.ID)
		}
	}
	if len(matches) != 1 {
		return "", false
	}
	return matches[0], true
}

func isSupportedLanguage(language scanner.Language) bool {
	return language == scanner.JavaScript || language == scanner.TypeScript || language == scanner.Vue
}

func normalizePath(filePath string) string {
	return strings.ReplaceAll(filePath, "\\", "/")
}
