package embeddings

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"

	"github.com/tamnguyendinh/avmatrix-go/internal/graph"
	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

type RuntimeContext struct {
	RepoName   string
	ServerName string
}

type EmbeddableNode struct {
	ID             string
	Name           string
	Label          scopeir.NodeLabel
	FilePath       string
	Content        string
	StartLine      int
	EndLine        int
	IsExported     *bool
	Description    string
	ParameterCount int
	ReturnType     string
	RepoName       string
	ServerName     string
	MethodNames    []string
	FieldNames     []string
}

func NodesFromGraph(g *graph.Graph, runtime RuntimeContext) []EmbeddableNode {
	if g == nil {
		return nil
	}
	nodes := make([]EmbeddableNode, 0)
	for _, node := range g.Nodes {
		if !IsEmbeddableLabel(node.Label) {
			continue
		}
		props := node.Properties
		nodes = append(nodes, EmbeddableNode{
			ID:             node.ID,
			Name:           stringProperty(props, "name"),
			Label:          node.Label,
			FilePath:       stringProperty(props, "filePath"),
			Content:        stringProperty(props, "content"),
			StartLine:      intProperty(props, "startLine", 0),
			EndLine:        intProperty(props, "endLine", 0),
			IsExported:     boolPointerProperty(props, "isExported"),
			Description:    stringProperty(props, "description"),
			ParameterCount: intProperty(props, "parameterCount", 0),
			ReturnType:     stringProperty(props, "returnType"),
			RepoName:       runtime.RepoName,
			ServerName:     runtime.ServerName,
			MethodNames:    stringSliceProperty(props, "methodNames"),
			FieldNames:     stringSliceProperty(props, "fieldNames"),
		})
	}
	return nodes
}

func GenerateText(node EmbeddableNode, codeBody string, config Config) string {
	config = NormalizeConfig(config)
	if IsShortLabel(node.Label) {
		header := metadataHeader(node, config)
		return header + "\n\n" + cleanContent(node.Content)
	}
	if node.Label == scopeir.NodeClass || node.Label == scopeir.NodeInterface {
		return generateStructuralTypeText(node, codeBody, config)
	}
	return metadataHeader(node, config) + "\n\n" + cleanContent(codeBody)
}

func ContentHashForNode(node EmbeddableNode, config Config) string {
	hashNode := node
	hashNode.MethodNames = nil
	hashNode.FieldNames = nil
	text := GenerateText(hashNode, hashNode.Content, config)
	sum := sha1.Sum([]byte(text))
	return hex.EncodeToString(sum[:])
}

func metadataHeader(node EmbeddableNode, config Config) string {
	parts := []string{fmt.Sprintf("%s: %s", node.Label, node.Name)}
	if node.RepoName != "" {
		parts = append(parts, "Repo: "+node.RepoName)
	}
	if node.ServerName != "" {
		parts = append(parts, "Server: "+node.ServerName)
	}
	parts = append(parts, "Path: "+node.FilePath)
	if node.IsExported != nil {
		parts = append(parts, fmt.Sprintf("Export: %t", *node.IsExported))
	}
	if node.Description != "" {
		if truncated := truncateDescription(node.Description, config.MaxDescriptionLength); truncated != "" {
			parts = append(parts, truncated)
		}
	}
	return strings.Join(parts, "\n")
}

func generateStructuralTypeText(node EmbeddableNode, codeBody string, config Config) string {
	parts := []string{metadataHeader(node, config)}
	if len(node.MethodNames) > 0 {
		parts = append(parts, "Methods: "+strings.Join(node.MethodNames, ", "))
	}
	if len(node.FieldNames) > 0 {
		parts = append(parts, "Properties: "+strings.Join(node.FieldNames, ", "))
	}
	if declaration := extractDeclarationOnly(cleanContent(node.Content)); declaration != "" {
		parts = append(parts, "", declaration)
	}
	cleanedChunk := cleanContent(codeBody)
	if cleanedChunk != "" && cleanedChunk != cleanContent(node.Content) {
		parts = append(parts, "", cleanedChunk)
	}
	return strings.Join(parts, "\n")
}

func truncateDescription(text string, maxLength int) string {
	if maxLength <= 0 || len(text) <= maxLength {
		return text
	}
	truncated := text[:maxLength]
	sentenceEnd := max(
		strings.LastIndex(truncated, ". "),
		strings.LastIndex(truncated, "! "),
		strings.LastIndex(truncated, "? "),
	)
	if sentenceEnd > maxLength/2 {
		return truncated[:sentenceEnd+1]
	}
	lastSpace := strings.LastIndex(truncated, " ")
	if lastSpace > maxLength/2 {
		return truncated[:lastSpace]
	}
	return truncated
}

func cleanContent(content string) string {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	for strings.Contains(content, "\n\n\n") {
		content = strings.ReplaceAll(content, "\n\n\n", "\n\n")
	}
	lines := strings.Split(content, "\n")
	for i := range lines {
		lines[i] = strings.TrimRight(lines[i], " \t")
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

var declarationStartPattern = regexp.MustCompile(`^(?:(?:export|pub|data|abstract)\s+)*(?:type\s+\w+\s+struct|(?:class|struct|enum|interface)\s)`)

func extractDeclarationOnly(content string) string {
	lines := strings.Split(content, "\n")
	declLines := make([]string, 0, len(lines))
	depth := 0
	started := false
	classDepth := 0
	skipDepth := 0

	for index, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !started {
			if declarationStartPattern.MatchString(trimmed) {
				nextEnd := min(index+4, len(lines))
				nextLines := lines[index+1 : nextEnd]
				if !strings.Contains(trimmed, "{") && !anyLineContains(nextLines, "{") {
					return ""
				}
				started = true
				declLines = append(declLines, trimmed)
				depth += strings.Count(trimmed, "{")
				depth -= strings.Count(trimmed, "}")
				if depth > 0 {
					classDepth = depth
				}
			}
			continue
		}

		opens := strings.Count(trimmed, "{")
		closes := strings.Count(trimmed, "}")
		prevDepth := depth
		depth += opens - closes

		if skipDepth > 0 {
			if depth <= classDepth {
				skipDepth = 0
				if depth <= 0 {
					declLines = append(declLines, trimmed)
					break
				}
			}
			continue
		}

		if strings.Contains(trimmed, "(") && opens > 0 && prevDepth+opens > classDepth {
			if opens == closes && strings.HasSuffix(trimmed, ";") {
				declLines = append(declLines, trimmed)
			}
			if opens != closes {
				skipDepth = classDepth
			}
			continue
		}

		declLines = append(declLines, trimmed)
		if depth <= 0 && len(declLines) > 1 {
			break
		}
	}

	return strings.TrimSpace(strings.Join(declLines, "\n"))
}

func anyLineContains(lines []string, needle string) bool {
	for _, line := range lines {
		if strings.Contains(line, needle) {
			return true
		}
	}
	return false
}

func stringProperty(props graph.NodeProperties, key string) string {
	if props == nil {
		return ""
	}
	value, ok := props[key]
	if !ok || value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return typed
	case fmt.Stringer:
		return typed.String()
	default:
		return fmt.Sprint(typed)
	}
}

func intProperty(props graph.NodeProperties, key string, fallback int) int {
	if props == nil || props[key] == nil {
		return fallback
	}
	switch typed := props[key].(type) {
	case int:
		return typed
	case int32:
		return int(typed)
	case int64:
		return int(typed)
	case float32:
		return int(typed)
	case float64:
		return int(typed)
	case string:
		var parsed int
		if _, err := fmt.Sscanf(typed, "%d", &parsed); err == nil {
			return parsed
		}
	}
	return fallback
}

func boolPointerProperty(props graph.NodeProperties, key string) *bool {
	if props == nil || props[key] == nil {
		return nil
	}
	switch typed := props[key].(type) {
	case bool:
		value := typed
		return &value
	case string:
		if strings.EqualFold(typed, "true") || strings.EqualFold(typed, "false") {
			value := strings.EqualFold(typed, "true")
			return &value
		}
	}
	return nil
}

func stringSliceProperty(props graph.NodeProperties, key string) []string {
	if props == nil || props[key] == nil {
		return nil
	}
	switch typed := props[key].(type) {
	case []string:
		return append([]string(nil), typed...)
	case []any:
		values := make([]string, 0, len(typed))
		for _, item := range typed {
			if item != nil {
				values = append(values, fmt.Sprint(item))
			}
		}
		return values
	case string:
		if typed == "" {
			return nil
		}
		return []string{typed}
	default:
		return []string{fmt.Sprint(typed)}
	}
}
