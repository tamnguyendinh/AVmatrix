package mcp

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/tamnguyendinh/avmatrix-go/internal/graph"
)

type renameEdit struct {
	Line       int    `json:"line"`
	OldText    string `json:"old_text"`
	NewText    string `json:"new_text"`
	Confidence string `json:"confidence"`
	oldLine    string
	newLine    string
}

type renameChange struct {
	FilePath string       `json:"file_path"`
	Edits    []renameEdit `json:"edits"`
}

func (s Server) renameTool(args map[string]any) (map[string]any, error) {
	newName := strings.TrimSpace(stringArg(args, "new_name", ""))
	if newName == "" {
		return map[string]any{"error": "new_name is required."}, nil
	}
	symbolName := strings.TrimSpace(stringArg(args, "symbol_name", ""))
	symbolUID := strings.TrimSpace(stringArg(args, "symbol_uid", ""))
	if symbolName == "" && symbolUID == "" {
		return map[string]any{"error": "Either symbol_name or symbol_uid is required."}, nil
	}

	entry, err := s.resolveResourceRepo(stringArg(args, "repo", ""))
	if err != nil {
		return nil, err
	}
	g, err := loadResourceGraphSnapshot(filepath.Join(storagePathForEntry(entry), "graph.json"))
	if err != nil {
		return nil, err
	}
	candidates := contextCandidates(g, symbolName, symbolUID, stringArg(args, "file_path", ""), "")
	if len(candidates) == 0 {
		return map[string]any{"error": fmt.Sprintf("Symbol '%s' not found", firstNonEmptyString(symbolName, symbolUID))}, nil
	}
	if len(candidates) > 1 && candidates[0].Score >= 0.9 && candidates[0].Score-candidates[1].Score > 0.09 {
		candidates = candidates[:1]
	}
	if len(candidates) > 1 {
		return map[string]any{
			"status":     "ambiguous",
			"message":    fmt.Sprintf("Found %d symbols matching '%s'. Use symbol_uid or file_path to disambiguate.", len(candidates), symbolName),
			"candidates": contextCandidatePayloads(candidates),
		}, nil
	}

	target := candidates[0].Node
	oldName := firstResourceNodeString(target, "name", "label", "heuristicLabel")
	if oldName == newName {
		return map[string]any{"error": "New name is the same as the current name."}, nil
	}
	changes := collectRenameChanges(entry.Path, g, target, oldName, newName)
	dryRun := true
	if raw, ok := args["dry_run"].(bool); ok {
		dryRun = raw
	}
	if !dryRun {
		if err := applyRenameChanges(entry.Path, changes); err != nil {
			return map[string]any{"error": err.Error()}, nil
		}
	}
	return map[string]any{
		"status":            "success",
		"old_name":          oldName,
		"new_name":          newName,
		"files_affected":    len(changes),
		"total_edits":       countRenameEdits(changes),
		"graph_edits":       countRenameEditsByConfidence(changes, "graph"),
		"text_search_edits": countRenameEditsByConfidence(changes, "text_search"),
		"changes":           changes,
		"applied":           !dryRun,
	}, nil
}

func collectRenameChanges(repoPath string, g *graph.Graph, target graph.Node, oldName string, newName string) []renameChange {
	changes := make(map[string]*renameChange)
	definitionPath := resourceNodeString(target, "filePath")
	if definitionPath != "" && resourceNodeInt(target, "startLine") > 0 {
		addRenameLineEdit(repoPath, changes, definitionPath, resourceNodeInt(target, "startLine"), oldName, newName, "graph")
	}

	nodeByID := resourceGraphNodesByID(g)
	for _, relationship := range g.Relationships {
		if relationship.TargetID != target.ID {
			continue
		}
		switch relationship.Type {
		case graph.RelCalls, graph.RelImports, graph.RelExtends, graph.RelImplements:
		default:
			continue
		}
		source, ok := nodeByID[relationship.SourceID]
		if !ok {
			continue
		}
		filePath := resourceNodeString(source, "filePath")
		if filePath == "" {
			continue
		}
		if lineNumber := renameReferenceLine(relationship); lineNumber > 0 {
			addRenameLineEdit(repoPath, changes, filePath, lineNumber, oldName, newName, "graph")
			continue
		}
		addRenameAllMatchingLineEdits(repoPath, changes, filePath, oldName, newName, "text_search")
	}

	out := make([]renameChange, 0, len(changes))
	for _, change := range changes {
		out = append(out, *change)
	}
	return out
}

func addRenameLineEdit(repoPath string, changes map[string]*renameChange, filePath string, lineNumber int, oldName string, newName string, confidence string) {
	lines, err := readRenameFileLines(repoPath, filePath)
	if err != nil || lineNumber < 1 || lineNumber > len(lines) {
		return
	}
	line := lines[lineNumber-1]
	if !renameLineContainsWord(line, oldName) {
		return
	}
	addRenameEdit(changes, filePath, lineNumber, line, replaceRenameWord(line, oldName, newName), confidence)
}

func addRenameAllMatchingLineEdits(repoPath string, changes map[string]*renameChange, filePath string, oldName string, newName string, confidence string) {
	lines, err := readRenameFileLines(repoPath, filePath)
	if err != nil {
		return
	}
	for index, line := range lines {
		if renameLineContainsWord(line, oldName) {
			addRenameEdit(changes, filePath, index+1, line, replaceRenameWord(line, oldName, newName), confidence)
		}
	}
}

func addRenameEdit(changes map[string]*renameChange, filePath string, lineNumber int, oldText string, newText string, confidence string) {
	change := changes[filePath]
	if change == nil {
		change = &renameChange{FilePath: filePath}
		changes[filePath] = change
	}
	oldLine := strings.TrimRight(oldText, "\r")
	newLine := strings.TrimRight(newText, "\r")
	for _, existing := range change.Edits {
		if existing.Line == lineNumber && existing.oldLine == oldLine {
			return
		}
	}
	change.Edits = append(change.Edits, renameEdit{
		Line:       lineNumber,
		OldText:    strings.TrimSpace(oldLine),
		NewText:    strings.TrimSpace(newLine),
		Confidence: confidence,
		oldLine:    oldLine,
		newLine:    newLine,
	})
}

func renameReferenceLine(relationship graph.Relationship) int {
	parts := strings.Split(relationship.ID, ":")
	if len(parts) < 3 {
		return 0
	}
	line, err := strconv.Atoi(parts[len(parts)-2])
	if err != nil || line < 1 {
		return 0
	}
	return line
}

func readRenameFileLines(repoPath string, filePath string) ([]string, error) {
	fullPath, err := safeRenamePath(repoPath, filePath)
	if err != nil {
		return nil, err
	}
	raw, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, err
	}
	return strings.Split(string(raw), "\n"), nil
}

func applyRenameChanges(repoPath string, changes []renameChange) error {
	for _, change := range changes {
		fullPath, err := safeRenamePath(repoPath, change.FilePath)
		if err != nil {
			return err
		}
		raw, err := os.ReadFile(fullPath)
		if err != nil {
			return err
		}
		content := string(raw)
		lines := strings.Split(content, "\n")
		for _, edit := range change.Edits {
			if err := applyRenameLineEdit(lines, change.FilePath, edit); err != nil {
				return err
			}
		}
		content = strings.Join(lines, "\n")
		if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
			return err
		}
	}
	return nil
}

func applyRenameLineEdit(lines []string, filePath string, edit renameEdit) error {
	if edit.Line < 1 || edit.Line > len(lines) {
		return fmt.Errorf("rename edit line out of range for %s:%d", filePath, edit.Line)
	}
	index := edit.Line - 1
	oldLine := edit.oldLine
	if oldLine == "" {
		oldLine = edit.OldText
	}
	newLine := edit.newLine
	if newLine == "" {
		newLine = edit.NewText
	}
	if lines[index] != oldLine {
		return fmt.Errorf("rename edit mismatch at %s:%d", filePath, edit.Line)
	}
	lines[index] = newLine
	return nil
}

func safeRenamePath(repoPath string, filePath string) (string, error) {
	fullPath := filepath.Clean(filepath.Join(repoPath, filePath))
	repoRoot := filepath.Clean(repoPath)
	relative, err := filepath.Rel(repoRoot, fullPath)
	if err != nil {
		return "", err
	}
	if relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", errors.New("path traversal blocked: " + filePath)
	}
	return fullPath, nil
}

func replaceRenameWord(line string, oldName string, newName string) string {
	pattern := regexp.MustCompile(`\b` + regexp.QuoteMeta(oldName) + `\b`)
	return pattern.ReplaceAllString(line, newName)
}

func renameLineContainsWord(line string, oldName string) bool {
	pattern := regexp.MustCompile(`\b` + regexp.QuoteMeta(oldName) + `\b`)
	return pattern.MatchString(line)
}

func countRenameEdits(changes []renameChange) int {
	total := 0
	for _, change := range changes {
		total += len(change.Edits)
	}
	return total
}

func countRenameEditsByConfidence(changes []renameChange, confidence string) int {
	total := 0
	for _, change := range changes {
		for _, edit := range change.Edits {
			if edit.Confidence == confidence {
				total++
			}
		}
	}
	return total
}
