package httpapi

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/tamnguyendinh/avmatrix-go/internal/repo"
)

type fileResponse struct {
	Content    string `json:"content"`
	StartLine  int    `json:"startLine,omitempty"`
	EndLine    int    `json:"endLine,omitempty"`
	TotalLines int    `json:"totalLines"`
}

func (s Server) handleFile(w http.ResponseWriter, r *http.Request) {
	if !methodAllowed(w, r, http.MethodGet) {
		return
	}

	entry, status, message, err := s.resolveRequestedRepo(r)
	if err != nil {
		writeError(w, status, message)
		return
	}
	relPath := strings.TrimSpace(r.URL.Query().Get("path"))
	if relPath == "" {
		writeError(w, http.StatusBadRequest, `Missing "path" query parameter`)
		return
	}

	fullPath, err := resolveRepoFilePath(entry.Path, relPath)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	raw, err := os.ReadFile(fullPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			writeError(w, http.StatusNotFound, "File not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	startLine, _ := optionalNonNegativeInt(r.URL.Query().Get("startLine"))
	endLine, hasEnd := optionalNonNegativeInt(r.URL.Query().Get("endLine"))
	response := fileSliceResponse(string(raw), startLine, endLine, hasEnd)
	writeJSON(w, http.StatusOK, response)
}

func resolveRepoFilePath(repoPath string, relPath string) (string, error) {
	normalized := filepath.Clean(filepath.FromSlash(relPath))
	if filepath.IsAbs(normalized) || normalized == "." || strings.HasPrefix(normalized, ".."+string(filepath.Separator)) || normalized == ".." {
		return "", errors.New("file path must be relative to the repository")
	}
	fullPath := filepath.Join(repoPath, normalized)
	if !repo.SamePath(repoPath, fullPath) {
		rel, err := filepath.Rel(repoPath, fullPath)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return "", errors.New("file path escapes the repository")
		}
	}
	return fullPath, nil
}

func optionalNonNegativeInt(value string) (int, bool) {
	if value == "" {
		return 0, false
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 0 {
		return 0, false
	}
	return parsed, true
}

func fileSliceResponse(content string, startLine int, endLine int, hasEnd bool) fileResponse {
	lines := strings.Split(content, "\n")
	totalLines := len(lines)
	if totalLines > 0 && lines[totalLines-1] == "" {
		totalLines--
		lines = lines[:totalLines]
	}
	if startLine > totalLines {
		startLine = totalLines
	}
	if !hasEnd || endLine >= totalLines {
		endLine = totalLines - 1
	}
	if endLine < startLine {
		return fileResponse{Content: "", StartLine: startLine, EndLine: startLine, TotalLines: totalLines}
	}
	return fileResponse{
		Content:    strings.Join(lines[startLine:endLine+1], "\n"),
		StartLine:  startLine,
		EndLine:    endLine,
		TotalLines: totalLines,
	}
}
