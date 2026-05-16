package tools

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
	namedToolPattern     = regexp.MustCompile(`(?is)name\s*:\s*['"]([A-Za-z_][A-Za-z0-9_-]*)['"]\s*,.*?description\s*:\s*[` + "`" + `'"]([^` + "`" + `'"]*)[` + "`" + `'"]`)
	registrationPattern  = regexp.MustCompile(`(?is)\b[A-Za-z_][A-Za-z0-9_]*\.tool\s*\(\s*['"]([A-Za-z_][A-Za-z0-9_-]*)['"](?:\s*,\s*['"]([^'"]*)['"])?`)
	decoratorToolPattern = regexp.MustCompile(`(?is)@tool(?:\s*\(\s*['"]([^'"]*)['"]\s*\))?\s*(?:export\s+)?(?:async\s+)?function\s+([A-Za-z_][A-Za-z0-9_]*)`)
	pythonToolPattern    = regexp.MustCompile(`(?is)@(?:[A-Za-z_][A-Za-z0-9_]*\.)?tool\s*\([^)]*\)\s*def\s+([A-Za-z_][A-Za-z0-9_]*)`)
)

type Result struct {
	Metrics Metrics
}

type Metrics struct {
	FilesScanned   int `json:"filesScanned,omitempty"`
	ToolsEmitted   int `json:"toolsEmitted,omitempty"`
	HandlesEmitted int `json:"handlesEmitted,omitempty"`
	Duplicates     int `json:"duplicates,omitempty"`
}

type definition struct {
	Name        string
	FilePath    string
	Description string
}

func Apply(g *graph.Graph, repoPath string, files []scanner.File) (Result, error) {
	if g == nil {
		return Result{}, nil
	}
	candidates := candidateFiles(files)
	result := Result{}
	defsByName := map[string]definition{}
	for _, file := range candidates {
		result.Metrics.FilesScanned++
		raw, err := os.ReadFile(filepath.Join(repoPath, filepath.FromSlash(file.Path)))
		if err != nil {
			return result, err
		}
		for _, def := range extractDefinitions(file.Path, string(raw)) {
			addDefinition(defsByName, def, &result.Metrics)
		}
	}
	defs := make([]definition, 0, len(defsByName))
	for _, def := range defsByName {
		defs = append(defs, def)
	}
	sort.Slice(defs, func(i int, j int) bool { return defs[i].Name < defs[j].Name })
	for _, def := range defs {
		if emitTool(g, def) {
			result.Metrics.ToolsEmitted++
			result.Metrics.HandlesEmitted++
		}
	}
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

func extractDefinitions(filePath string, content string) []definition {
	out := make([]definition, 0)
	if strings.Contains(content, "inputSchema") {
		for _, match := range namedToolPattern.FindAllStringSubmatch(content, -1) {
			out = append(out, definition{Name: match[1], FilePath: filePath, Description: cleanDescription(match[2])})
		}
	}
	for _, match := range registrationPattern.FindAllStringSubmatch(content, -1) {
		out = append(out, definition{Name: match[1], FilePath: filePath, Description: cleanDescription(match[2])})
	}
	for _, match := range decoratorToolPattern.FindAllStringSubmatch(content, -1) {
		description := cleanDescription(match[1])
		if description == "" {
			description = "decorated tool"
		}
		out = append(out, definition{Name: match[2], FilePath: filePath, Description: description})
	}
	for _, match := range pythonToolPattern.FindAllStringSubmatch(content, -1) {
		out = append(out, definition{Name: match[1], FilePath: filePath, Description: "decorated tool"})
	}
	return out
}

func addDefinition(defsByName map[string]definition, def definition, metrics *Metrics) {
	if def.Name == "" || def.FilePath == "" {
		return
	}
	if _, exists := defsByName[def.Name]; exists {
		metrics.Duplicates++
		return
	}
	defsByName[def.Name] = def
}

func emitTool(g *graph.Graph, def definition) bool {
	fileNodeID := graph.GenerateID(string(scopeir.NodeFile), def.FilePath)
	if _, ok := g.GetNode(fileNodeID); !ok {
		return false
	}
	toolNodeID := graph.GenerateID(string(scopeir.NodeTool), def.Name)
	g.AddNode(graph.Node{
		ID:    toolNodeID,
		Label: scopeir.NodeTool,
		Properties: graph.NodeProperties{
			"name":        def.Name,
			"filePath":    def.FilePath,
			"description": def.Description,
		},
	})
	g.AddRelationship(graph.Relationship{
		ID:         graph.GenerateID(string(graph.RelHandlesTool), fileNodeID+"->"+toolNodeID),
		SourceID:   fileNodeID,
		TargetID:   toolNodeID,
		Type:       graph.RelHandlesTool,
		Confidence: 1,
		Reason:     "tool-definition",
	})
	return true
}

func cleanDescription(value string) string {
	return strings.TrimSpace(strings.ReplaceAll(value, "\n", " "))
}

func isSupportedLanguage(language scanner.Language) bool {
	return language == scanner.JavaScript || language == scanner.TypeScript || language == scanner.Python
}

func normalizePath(filePath string) string {
	return strings.ReplaceAll(filePath, "\\", "/")
}
