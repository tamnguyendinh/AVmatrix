package mcp

import (
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/tamnguyendinh/avmatrix-go/internal/graph"
	"github.com/tamnguyendinh/avmatrix-go/internal/repo"
)

var diffHunkPattern = regexp.MustCompile(`@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@`)

type detectFileDiff struct {
	FilePath string
	Hunks    []detectHunk
	Deleted  bool
}

type detectHunk struct {
	StartLine int
	EndLine   int
}

func (s Server) detectChangesTool(args map[string]any) (map[string]any, error) {
	entry, err := s.resolveResourceRepo(stringArg(args, "repo", ""))
	if err != nil {
		return nil, err
	}
	diffOutput, err := gitDiffForDetectChanges(entry, args)
	if err != nil {
		return map[string]any{"error": "Git diff failed: " + err.Error()}, nil
	}
	fileDiffs := parseDetectDiffHunks(diffOutput)
	if len(fileDiffs) == 0 {
		return map[string]any{
			"summary": map[string]any{
				"changed_count":  0,
				"affected_count": 0,
				"risk_level":     "none",
				"message":        "No changes detected.",
			},
			"changed_symbols":    []map[string]any{},
			"affected_processes": []map[string]any{},
		}, nil
	}

	g, err := loadResourceGraphSnapshot(filepath.Join(storagePathForEntry(entry), "graph.json"))
	if err != nil {
		return nil, err
	}
	changedSymbols := detectChangedSymbols(g, fileDiffs)
	affectedProcesses := detectAffectedProcesses(g, changedSymbols)
	return map[string]any{
		"summary": map[string]any{
			"changed_count":  len(changedSymbols),
			"affected_count": len(affectedProcesses),
			"changed_files":  len(fileDiffs),
			"risk_level":     detectRisk(len(affectedProcesses)),
		},
		"changed_symbols":    changedSymbols,
		"affected_processes": affectedProcesses,
	}, nil
}

func gitDiffForDetectChanges(entry repo.RegistryEntry, args map[string]any) (string, error) {
	scope := firstNonEmptyString(stringArg(args, "scope", ""), "unstaged")
	diffArgs := []string{"-C", entry.Path, "diff", "-U0"}
	switch scope {
	case "staged":
		diffArgs = []string{"-C", entry.Path, "diff", "--staged", "-U0"}
	case "all":
		diffArgs = []string{"-C", entry.Path, "diff", "HEAD", "-U0"}
	case "compare":
		baseRef := strings.TrimSpace(stringArg(args, "base_ref", ""))
		if baseRef == "" {
			return "", errors.New(`base_ref is required for "compare" scope`)
		}
		diffArgs = []string{"-C", entry.Path, "diff", baseRef, "-U0"}
	case "unstaged":
	default:
		scope = "unstaged"
	}
	output, err := exec.Command("git", diffArgs...).Output()
	return string(output), err
}

func parseDetectDiffHunks(diffOutput string) []detectFileDiff {
	var files []detectFileDiff
	currentIndex := -1
	oldPath := ""
	for _, line := range strings.Split(diffOutput, "\n") {
		switch {
		case strings.HasPrefix(line, "--- "):
			oldPath = strings.TrimSpace(strings.TrimPrefix(line, "--- "))
			if oldPath != "/dev/null" {
				oldPath = strings.TrimPrefix(oldPath, "a/")
			}
		case strings.HasPrefix(line, "+++ "):
			newPath := strings.TrimSpace(strings.TrimPrefix(line, "+++ "))
			deleted := newPath == "/dev/null"
			filePath := newPath
			if deleted {
				filePath = oldPath
			}
			if filePath == "" || filePath == "/dev/null" {
				currentIndex = -1
				continue
			}
			filePath = strings.TrimPrefix(filePath, "b/")
			filePath = strings.TrimPrefix(filePath, "a/")
			files = append(files, detectFileDiff{FilePath: normalizeContextPath(filePath), Deleted: deleted})
			currentIndex = len(files) - 1
		case strings.HasPrefix(line, "@@ ") && currentIndex >= 0:
			hunk, ok := parseDetectHunk(line, files[currentIndex].Deleted)
			if ok {
				files[currentIndex].Hunks = append(files[currentIndex].Hunks, hunk)
			}
		}
	}
	out := files[:0]
	for _, file := range files {
		if file.FilePath != "" && len(file.Hunks) > 0 {
			out = append(out, file)
		}
	}
	return out
}

func parseDetectHunk(line string, deleted bool) (detectHunk, bool) {
	matches := diffHunkPattern.FindStringSubmatch(line)
	if len(matches) == 0 {
		return detectHunk{}, false
	}
	startIndex := 3
	countIndex := 4
	if deleted {
		startIndex = 1
		countIndex = 2
	}
	start, err := strconv.Atoi(matches[startIndex])
	if err != nil {
		return detectHunk{}, false
	}
	count := 1
	if len(matches) > countIndex && matches[countIndex] != "" {
		parsed, parseErr := strconv.Atoi(matches[countIndex])
		if parseErr == nil {
			count = parsed
		}
	}
	if count == 0 {
		return detectHunk{}, false
	}
	end := start
	if count > 0 {
		end = start + count - 1
	}
	return detectHunk{StartLine: start, EndLine: end}, true
}

func detectChangedSymbols(g *graph.Graph, fileDiffs []detectFileDiff) []map[string]any {
	seen := make(map[string]bool)
	out := make([]map[string]any, 0)
	for _, fileDiff := range fileDiffs {
		for _, node := range g.Nodes {
			filePath := normalizeContextPath(resourceNodeString(node, "filePath"))
			if filePath == "" || !(filePath == fileDiff.FilePath || strings.HasSuffix(filePath, "/"+fileDiff.FilePath)) {
				continue
			}
			startLine := resourceNodeInt(node, "startLine")
			endLine := resourceNodeInt(node, "endLine")
			if startLine == 0 || endLine == 0 || !detectOverlapsAnyHunk(startLine, endLine, fileDiff.Hunks) {
				continue
			}
			if seen[node.ID] {
				continue
			}
			seen[node.ID] = true
			changeType := "touched"
			if fileDiff.Deleted {
				changeType = "deleted"
			}
			out = append(out, map[string]any{
				"id":          node.ID,
				"name":        firstResourceNodeString(node, "name", "label", "heuristicLabel"),
				"type":        string(node.Label),
				"filePath":    resourceNodeString(node, "filePath"),
				"change_type": changeType,
			})
		}
	}
	return out
}

func detectOverlapsAnyHunk(startLine int, endLine int, hunks []detectHunk) bool {
	for _, hunk := range hunks {
		if startLine <= hunk.EndLine && endLine >= hunk.StartLine {
			return true
		}
	}
	return false
}

func detectAffectedProcesses(g *graph.Graph, changedSymbols []map[string]any) []map[string]any {
	if len(changedSymbols) == 0 {
		return []map[string]any{}
	}
	changedSet := make(map[string]string, len(changedSymbols))
	for _, symbol := range changedSymbols {
		changedSet[fmt.Sprint(symbol["id"])] = fmt.Sprint(symbol["name"])
	}
	nodeByID := resourceGraphNodesByID(g)
	processes := make(map[string]map[string]any)
	for _, relationship := range g.Relationships {
		symbolName, ok := changedSet[relationship.SourceID]
		if relationship.Type != graph.RelStepInProcess || !ok {
			continue
		}
		process, exists := nodeByID[relationship.TargetID]
		if !exists {
			continue
		}
		current := processes[process.ID]
		if current == nil {
			current = map[string]any{
				"id":            process.ID,
				"name":          firstResourceNodeString(process, "heuristicLabel", "label", "name"),
				"process_type":  resourceNodeString(process, "processType"),
				"step_count":    resourceNodeInt(process, "stepCount"),
				"changed_steps": []map[string]any{},
			}
			processes[process.ID] = current
		}
		step := 0
		if relationship.Step != nil {
			step = *relationship.Step
		}
		current["changed_steps"] = append(current["changed_steps"].([]map[string]any), map[string]any{"symbol": symbolName, "step": step})
	}
	out := make([]map[string]any, 0, len(processes))
	for _, process := range processes {
		out = append(out, process)
	}
	return out
}

func detectRisk(processCount int) string {
	switch {
	case processCount == 0:
		return "low"
	case processCount <= 5:
		return "medium"
	case processCount <= 15:
		return "high"
	default:
		return "critical"
	}
}
