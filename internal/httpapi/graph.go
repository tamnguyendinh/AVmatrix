package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"

	"github.com/tamnguyendinh/avmatrix-go/internal/graph"
)

type graphResponse struct {
	Nodes         []graph.Node         `json:"nodes"`
	Relationships []graph.Relationship `json:"relationships"`
}

const graphNDJSONFlushInterval = 512

func (s Server) handleGraph(w http.ResponseWriter, r *http.Request) {
	if !methodAllowed(w, r, http.MethodGet) {
		return
	}

	entry, status, message, err := s.resolveRequestedRepo(r)
	if err != nil {
		if status == http.StatusNotFound {
			message = "Repository not found"
		}
		writeError(w, status, message)
		return
	}

	g, err := loadGraphSnapshot(filepath.Join(storagePathFor(entry), "graph.json"))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			writeError(w, http.StatusNotFound, "Graph not found. Run: avmatrix analyze")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	includeContent := r.URL.Query().Get("includeContent") == "true"

	if r.URL.Query().Get("stream") == "true" {
		header := w.Header()
		header.Set("Content-Type", "application/x-ndjson; charset=utf-8")
		header.Set("Cache-Control", "no-cache")
		header.Set("X-Accel-Buffering", "no")
		w.WriteHeader(http.StatusOK)
		streamGraphNDJSON(w, g, includeContent)
		return
	}

	writeJSON(w, http.StatusOK, graphPayload(g, includeContent))
}

func graphPayload(g *graph.Graph, includeContent bool) graphResponse {
	nodes := make([]graph.Node, 0, len(g.Nodes))
	for _, node := range g.Nodes {
		nodes = append(nodes, graphNodeForResponse(node, includeContent))
	}
	return graphResponse{Nodes: nodes, Relationships: g.Relationships}
}

func streamGraphNDJSON(w http.ResponseWriter, g *graph.Graph, includeContent bool) {
	encoder := json.NewEncoder(w)
	flusher, _ := w.(http.Flusher)
	written := 0
	flush := func(force bool) {
		if flusher == nil {
			return
		}
		if force || written%graphNDJSONFlushInterval == 0 {
			flusher.Flush()
		}
	}
	for _, node := range g.Nodes {
		_ = encoder.Encode(map[string]any{
			"type": "node",
			"data": graphNodeForResponse(node, includeContent),
		})
		written++
		flush(false)
	}
	for _, relationship := range g.Relationships {
		_ = encoder.Encode(map[string]any{
			"type": "relationship",
			"data": relationship,
		})
		written++
		flush(false)
	}
	flush(true)
}

func graphNodeForResponse(node graph.Node, includeContent bool) graph.Node {
	if includeContent {
		return node
	}
	if _, ok := node.Properties["content"]; !ok {
		return node
	}
	properties := make(graph.NodeProperties, len(node.Properties)-1)
	for key, value := range node.Properties {
		if key != "content" {
			properties[key] = value
		}
	}
	node.Properties = properties
	return node
}
