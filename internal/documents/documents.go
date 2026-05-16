package documents

import (
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/tamnguyendinh/avmatrix-go/internal/graph"
	"github.com/tamnguyendinh/avmatrix-go/internal/scanner"
	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

var (
	headingPattern = regexp.MustCompile(`^(#{1,6})\s+(.+)$`)
	linkPattern    = regexp.MustCompile(`\[([^\]]*)\]\(([^)]+)\)`)
)

type Result struct {
	Metrics Metrics
}

type Metrics struct {
	MarkdownFiles     int `json:"markdownFiles,omitempty"`
	WordFiles         int `json:"wordFiles,omitempty"`
	PDFFiles          int `json:"pdfFiles,omitempty"`
	SpreadsheetFiles  int `json:"spreadsheetFiles,omitempty"`
	Sections          int `json:"sections,omitempty"`
	Links             int `json:"links,omitempty"`
	MetadataFileNodes int `json:"metadataFileNodes,omitempty"`
}

type heading struct {
	Level   int
	Text    string
	LineNum int
}

func Apply(g *graph.Graph, repoPath string, files []scanner.File) (Result, error) {
	if g == nil {
		return Result{}, nil
	}
	allPaths := make(map[string]struct{}, len(files))
	ordered := make([]scanner.File, 0, len(files))
	for _, file := range files {
		file.Path = normalizePath(file.Path)
		if file.Path == "" {
			continue
		}
		allPaths[file.Path] = struct{}{}
		if documentKind(file.Path) != "" {
			ordered = append(ordered, file)
		}
	}
	sort.Slice(ordered, func(i int, j int) bool { return ordered[i].Path < ordered[j].Path })

	result := Result{}
	for _, file := range ordered {
		kind := documentKind(file.Path)
		result.count(kind)
		if markFileNode(g, file, kind) {
			result.Metrics.MetadataFileNodes++
		}
		if kind != "markdown" {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(repoPath, filepath.FromSlash(file.Path)))
		if err != nil {
			return result, err
		}
		sections, links := processMarkdown(g, file.Path, string(raw), allPaths)
		result.Metrics.Sections += sections
		result.Metrics.Links += links
	}
	return result, nil
}

func (result *Result) count(kind string) {
	switch kind {
	case "markdown":
		result.Metrics.MarkdownFiles++
	case "word":
		result.Metrics.WordFiles++
	case "pdf":
		result.Metrics.PDFFiles++
	case "spreadsheet":
		result.Metrics.SpreadsheetFiles++
	}
}

func markFileNode(g *graph.Graph, file scanner.File, kind string) bool {
	fileNodeID := graph.GenerateID(string(scopeir.NodeFile), file.Path)
	node, ok := g.GetNode(fileNodeID)
	if !ok {
		return false
	}
	if node.Properties == nil {
		node.Properties = graph.NodeProperties{}
	}
	node.Properties["documentKind"] = kind
	node.Properties["fileExtension"] = strings.TrimPrefix(strings.ToLower(path.Ext(file.Path)), ".")
	node.Properties["binary"] = kind != "markdown"
	if file.Language != "" {
		node.Properties["language"] = string(file.Language)
	}
	g.AddNode(node)
	return true
}

func processMarkdown(g *graph.Graph, filePath string, content string, allPaths map[string]struct{}) (int, int) {
	fileNodeID := graph.GenerateID(string(scopeir.NodeFile), filePath)
	if _, ok := g.GetNode(fileNodeID); !ok {
		return 0, 0
	}
	lines := strings.Split(content, "\n")
	headings := collectHeadings(lines)
	sectionCount := emitSections(g, fileNodeID, filePath, lines, headings)
	linkCount := emitLinks(g, fileNodeID, filePath, content, allPaths)
	return sectionCount, linkCount
}

func collectHeadings(lines []string) []heading {
	out := make([]heading, 0)
	for index, line := range lines {
		match := headingPattern.FindStringSubmatch(line)
		if match == nil {
			continue
		}
		out = append(out, heading{
			Level:   len(match[1]),
			Text:    strings.TrimSpace(match[2]),
			LineNum: index + 1,
		})
	}
	return out
}

func emitSections(g *graph.Graph, fileNodeID string, filePath string, lines []string, headings []heading) int {
	type stackItem struct {
		level int
		id    string
	}
	stack := make([]stackItem, 0)
	for index, item := range headings {
		endLine := len(lines)
		for next := index + 1; next < len(headings); next++ {
			if headings[next].Level <= item.Level {
				endLine = headings[next].LineNum - 1
				break
			}
		}

		sectionID := graph.GenerateID(string(scopeir.NodeSection), filePath+":L"+strconv.Itoa(item.LineNum)+":"+item.Text)
		g.AddNode(graph.Node{
			ID:    sectionID,
			Label: scopeir.NodeSection,
			Properties: graph.NodeProperties{
				"name":        item.Text,
				"filePath":    filePath,
				"startLine":   item.LineNum,
				"endLine":     endLine,
				"level":       item.Level,
				"description": "h" + strconv.Itoa(item.Level),
			},
		})

		for len(stack) > 0 && stack[len(stack)-1].level >= item.Level {
			stack = stack[:len(stack)-1]
		}
		parentID := fileNodeID
		if len(stack) > 0 {
			parentID = stack[len(stack)-1].id
		}
		g.AddRelationship(graph.Relationship{
			ID:         graph.GenerateID(string(graph.RelContains), parentID+"->"+sectionID),
			SourceID:   parentID,
			TargetID:   sectionID,
			Type:       graph.RelContains,
			Confidence: 1,
			Reason:     "markdown-heading",
		})
		stack = append(stack, stackItem{level: item.Level, id: sectionID})
	}
	return len(headings)
}

func emitLinks(g *graph.Graph, fileNodeID string, filePath string, content string, allPaths map[string]struct{}) int {
	fileDir := path.Dir(filePath)
	if fileDir == "." {
		fileDir = ""
	}
	seen := make(map[string]struct{})
	links := 0
	for _, match := range linkPattern.FindAllStringSubmatch(content, -1) {
		href := match[2]
		if skipLink(href) {
			continue
		}
		cleanHref := strings.Split(href, "#")[0]
		if cleanHref == "" {
			continue
		}
		resolved := path.Clean(path.Join(fileDir, cleanHref))
		if resolved == "." {
			continue
		}
		if _, ok := allPaths[resolved]; !ok {
			continue
		}
		targetFileID := graph.GenerateID(string(scopeir.NodeFile), resolved)
		if _, ok := g.GetNode(targetFileID); !ok {
			continue
		}
		key := fileNodeID + "->" + targetFileID
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		g.AddRelationship(graph.Relationship{
			ID:         graph.GenerateID(string(graph.RelImports), key),
			SourceID:   fileNodeID,
			TargetID:   targetFileID,
			Type:       graph.RelImports,
			Confidence: 0.8,
			Reason:     "markdown-link",
		})
		links++
	}
	return links
}

func skipLink(href string) bool {
	lower := strings.ToLower(href)
	return strings.HasPrefix(lower, "http://") ||
		strings.HasPrefix(lower, "https://") ||
		strings.HasPrefix(lower, "#") ||
		strings.HasPrefix(lower, "mailto:")
}

func documentKind(filePath string) string {
	switch strings.ToLower(path.Ext(filePath)) {
	case ".md", ".mdx":
		return "markdown"
	case ".doc", ".docx", ".odt", ".rtf":
		return "word"
	case ".pdf":
		return "pdf"
	case ".xls", ".xlsx", ".xlsm", ".xlsb", ".xlt", ".xltx", ".xltm", ".xlam", ".ods", ".csv", ".tsv":
		return "spreadsheet"
	default:
		return ""
	}
}

func normalizePath(filePath string) string {
	return strings.ReplaceAll(filePath, "\\", "/")
}
