package httpapi

import (
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

type grepResponse struct {
	Results []grepResult `json:"results"`
}

type grepResult struct {
	FilePath string `json:"filePath"`
	Line     int    `json:"line"`
	Text     string `json:"text"`
}

func (s Server) handleGrep(w http.ResponseWriter, r *http.Request) {
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
	pattern := r.URL.Query().Get("pattern")
	if pattern == "" {
		writeError(w, http.StatusBadRequest, `Missing "pattern" query parameter`)
		return
	}
	if len(pattern) > 200 {
		writeError(w, http.StatusBadRequest, "Pattern too long (max 200 characters)")
		return
	}
	regex, err := regexp.Compile("(?im)" + pattern)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid regex pattern")
		return
	}

	g, err := loadGraphSnapshot(filepath.Join(storagePathFor(entry), "graph.json"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	limit := boundedQueryLimit(r.URL.Query().Get("limit"), 50, 1, 200)
	results := make([]grepResult, 0)
	seenFiles := make(map[string]bool)
	for _, node := range g.Nodes {
		if len(results) >= limit || node.Label != scopeir.NodeFile {
			continue
		}
		filePath := nodeString(node, "filePath")
		if filePath == "" || seenFiles[filePath] {
			continue
		}
		seenFiles[filePath] = true
		fullPath, err := resolveRepoFilePath(entry.Path, filePath)
		if err != nil {
			continue
		}
		raw, err := os.ReadFile(fullPath)
		if err != nil {
			continue
		}
		lines := strings.Split(string(raw), "\n")
		for index, line := range lines {
			if len(results) >= limit {
				break
			}
			line = strings.TrimSuffix(line, "\r")
			if regex.MatchString(line) {
				results = append(results, grepResult{
					FilePath: filePath,
					Line:     index + 1,
					Text:     truncateRunes(strings.TrimSpace(line), 200),
				})
			}
		}
	}

	writeJSON(w, http.StatusOK, grepResponse{Results: results})
}

func truncateRunes(value string, maxRunes int) string {
	if utf8.RuneCountInString(value) <= maxRunes {
		return value
	}
	out := make([]rune, 0, maxRunes)
	for _, r := range value {
		if len(out) >= maxRunes {
			break
		}
		out = append(out, r)
	}
	return string(out)
}
