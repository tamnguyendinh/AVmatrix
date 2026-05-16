package structure

import (
	"path"
	"sort"
	"strings"

	"github.com/tamnguyendinh/avmatrix-go/internal/graph"
	"github.com/tamnguyendinh/avmatrix-go/internal/scanner"
	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

type Result struct {
	Metrics Metrics
}

type Metrics struct {
	FileNodesSeen    int `json:"fileNodesSeen,omitempty"`
	FolderNodesAdded int `json:"folderNodesAdded,omitempty"`
	ContainsEmitted  int `json:"containsEmitted,omitempty"`
}

func Apply(g *graph.Graph, files []scanner.File) Result {
	if g == nil {
		return Result{}
	}
	paths := make([]string, 0, len(files))
	for _, file := range files {
		if file.Path != "" {
			paths = append(paths, strings.ReplaceAll(file.Path, "\\", "/"))
		}
	}
	sort.Strings(paths)

	result := Result{}
	for _, filePath := range paths {
		applyPath(g, filePath, &result.Metrics)
	}
	return result
}

func applyPath(g *graph.Graph, filePath string, metrics *Metrics) {
	parts := strings.Split(filePath, "/")
	currentPath := ""
	parentID := ""
	for index, part := range parts {
		if part == "" {
			continue
		}
		isFile := index == len(parts)-1
		if currentPath == "" {
			currentPath = part
		} else {
			currentPath = currentPath + "/" + part
		}

		label := scopeir.NodeFolder
		if isFile {
			label = scopeir.NodeFile
			metrics.FileNodesSeen++
		}
		nodeID := graph.GenerateID(string(label), currentPath)
		addStructureNode(g, nodeID, label, part, currentPath, metrics)
		if parentID != "" {
			g.AddRelationship(graph.Relationship{
				ID:         graph.GenerateID(string(graph.RelContains), parentID+"->"+nodeID),
				SourceID:   parentID,
				TargetID:   nodeID,
				Type:       graph.RelContains,
				Confidence: 1,
				Reason:     "project structure",
			})
			metrics.ContainsEmitted++
		}
		parentID = nodeID
	}
}

func addStructureNode(g *graph.Graph, nodeID string, label scopeir.NodeLabel, name string, filePath string, metrics *Metrics) {
	if existing, ok := g.GetNode(nodeID); ok {
		if existing.Properties == nil {
			existing.Properties = graph.NodeProperties{}
		}
		if _, ok := existing.Properties["name"]; !ok {
			existing.Properties["name"] = name
		}
		if _, ok := existing.Properties["filePath"]; !ok {
			existing.Properties["filePath"] = filePath
		}
		g.AddNode(existing)
		return
	}
	g.AddNode(graph.Node{
		ID:    nodeID,
		Label: label,
		Properties: graph.NodeProperties{
			"name":     name,
			"filePath": cleanPath(filePath),
		},
	})
	if label == scopeir.NodeFolder {
		metrics.FolderNodesAdded++
	}
}

func cleanPath(value string) string {
	cleaned := path.Clean(strings.ReplaceAll(value, "\\", "/"))
	if cleaned == "." {
		return ""
	}
	return cleaned
}
