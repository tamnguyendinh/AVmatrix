package group

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/tamnguyendinh/avmatrix-go/internal/graph"
	"github.com/tamnguyendinh/avmatrix-go/internal/repo"
	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

type groupProcess struct {
	ID          string
	Label       string
	ProcessType string
	StepCount   int
}

type groupProcessStep struct {
	Name     string
	FilePath string
}

func Query(homeDir string, store repo.Store, name string, queryText string, limit int, subgroup string) (QueryResult, error) {
	config, err := Load(homeDir, name)
	if err != nil {
		return QueryResult{}, err
	}
	entries, err := store.ListRegistered(false)
	if err != nil {
		return QueryResult{}, err
	}
	if limit < 1 {
		limit = 5
	}
	if limit > 50 {
		limit = 50
	}

	repoPaths := sortedGroupRepoPaths(config.Repos)
	perRepo := make([]QueryRepoSummary, 0, len(repoPaths))
	allProcesses := make([]map[string]any, 0)
	for _, groupRepoPath := range repoPaths {
		if !repoInSubgroup(groupRepoPath, subgroup) {
			continue
		}
		registryName := config.Repos[groupRepoPath]
		entry, err := repo.ResolveEntry(entries, registryName)
		if err != nil {
			perRepo = append(perRepo, QueryRepoSummary{Repo: groupRepoPath, Count: 0})
			continue
		}
		g, err := loadGroupGraphSnapshot(filepath.Join(storagePathForEntry(entry), "graph.json"))
		if err != nil {
			perRepo = append(perRepo, QueryRepoSummary{Repo: groupRepoPath, Count: 0})
			continue
		}

		processes := rankedGroupProcesses(g, queryText, limit)
		scored := make([]map[string]any, 0, len(processes))
		for index, process := range processes {
			item := map[string]any{
				"id":             process.ID,
				"label":          process.Label,
				"name":           process.Label,
				"summary":        process.Label,
				"heuristicLabel": process.Label,
				"processType":    process.ProcessType,
				"stepCount":      process.StepCount,
				"_rrf_score":     1.0 / float64(index+1+60),
				"_repo":          groupRepoPath,
			}
			scored = append(scored, item)
		}
		perRepo = append(perRepo, QueryRepoSummary{Repo: groupRepoPath, Count: len(scored)})
		allProcesses = append(allProcesses, scored...)
	}

	sort.SliceStable(allProcesses, func(i, j int) bool {
		leftScore, _ := allProcesses[i]["_rrf_score"].(float64)
		rightScore, _ := allProcesses[j]["_rrf_score"].(float64)
		if leftScore != rightScore {
			return leftScore > rightScore
		}
		leftRepo, _ := allProcesses[i]["_repo"].(string)
		rightRepo, _ := allProcesses[j]["_repo"].(string)
		if leftRepo != rightRepo {
			return leftRepo < rightRepo
		}
		leftID, _ := allProcesses[i]["id"].(string)
		rightID, _ := allProcesses[j]["id"].(string)
		return leftID < rightID
	})
	if len(allProcesses) > limit {
		allProcesses = allProcesses[:limit]
	}

	return QueryResult{
		Group:   name,
		Query:   queryText,
		Results: allProcesses,
		PerRepo: perRepo,
	}, nil
}

func sortedGroupRepoPaths(repos map[string]string) []string {
	paths := make([]string, 0, len(repos))
	for repoPath := range repos {
		paths = append(paths, repoPath)
	}
	sort.Strings(paths)
	return paths
}

func repoInSubgroup(repoPath string, subgroup string) bool {
	subgroup = strings.TrimRight(strings.ReplaceAll(strings.TrimSpace(subgroup), "\\", "/"), "/")
	if subgroup == "" {
		return true
	}
	normalizedRepoPath := strings.ReplaceAll(repoPath, "\\", "/")
	return normalizedRepoPath == subgroup || strings.HasPrefix(normalizedRepoPath, subgroup+"/")
}

func loadGroupGraphSnapshot(path string) (*graph.Graph, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var g graph.Graph
	if err := json.Unmarshal(raw, &g); err != nil {
		return nil, err
	}
	return &g, nil
}

func rankedGroupProcesses(g *graph.Graph, query string, limit int) []groupProcess {
	needle := strings.ToLower(query)
	processes := groupProcessItems(g)
	type scoredProcess struct {
		process groupProcess
		score   int
	}
	scored := make([]scoredProcess, 0, len(processes))
	for _, process := range processes {
		score := groupContainsScore(process.Label, needle) + groupContainsScore(process.ProcessType, needle)
		for _, step := range groupProcessSteps(g, process.ID) {
			score += groupContainsScore(step.Name, needle)
			score += groupContainsScore(step.FilePath, needle)
		}
		if score > 0 {
			scored = append(scored, scoredProcess{process: process, score: score})
		}
	}
	if len(scored) == 0 {
		for _, process := range processes[:minGroupInt(len(processes), limit)] {
			scored = append(scored, scoredProcess{process: process, score: 0})
		}
	}
	sort.Slice(scored, func(i, j int) bool {
		if scored[i].score != scored[j].score {
			return scored[i].score > scored[j].score
		}
		if scored[i].process.StepCount != scored[j].process.StepCount {
			return scored[i].process.StepCount > scored[j].process.StepCount
		}
		return scored[i].process.Label < scored[j].process.Label
	})
	out := make([]groupProcess, 0, minGroupInt(len(scored), limit))
	for _, item := range scored[:minGroupInt(len(scored), limit)] {
		out = append(out, item.process)
	}
	return out
}

func groupProcessItems(g *graph.Graph) []groupProcess {
	items := make([]groupProcess, 0)
	for _, node := range g.Nodes {
		if node.Label != scopeir.NodeProcess {
			continue
		}
		items = append(items, groupProcess{
			ID:          node.ID,
			Label:       firstNonEmptyGroupString(groupNodeString(node, "heuristicLabel"), groupNodeString(node, "label"), groupNodeString(node, "name"), node.ID),
			ProcessType: groupNodeString(node, "processType"),
			StepCount:   groupNodeInt(node, "stepCount"),
		})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].StepCount != items[j].StepCount {
			return items[i].StepCount > items[j].StepCount
		}
		return items[i].ID < items[j].ID
	})
	return items
}

func groupProcessSteps(g *graph.Graph, processID string) []groupProcessStep {
	nodes := groupNodesByID(g)
	steps := make([]groupProcessStep, 0)
	for _, relationship := range g.Relationships {
		if relationship.Type != graph.RelStepInProcess || relationship.TargetID != processID {
			continue
		}
		node, ok := nodes[relationship.SourceID]
		if !ok {
			continue
		}
		steps = append(steps, groupProcessStep{
			Name:     firstNonEmptyGroupString(groupNodeString(node, "name"), groupNodeString(node, "label"), groupNodeString(node, "heuristicLabel")),
			FilePath: groupNodeString(node, "filePath"),
		})
	}
	return steps
}

func groupNodesByID(g *graph.Graph) map[string]graph.Node {
	nodes := make(map[string]graph.Node, len(g.Nodes))
	for _, node := range g.Nodes {
		nodes[node.ID] = node
	}
	return nodes
}

func groupNodeString(node graph.Node, key string) string {
	value, _ := node.Properties[key].(string)
	return value
}

func groupNodeInt(node graph.Node, key string) int {
	switch value := node.Properties[key].(type) {
	case int:
		return value
	case int32:
		return int(value)
	case int64:
		return int(value)
	case float64:
		return int(value)
	default:
		return 0
	}
}

func firstNonEmptyGroupString(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func groupContainsScore(value string, needle string) int {
	if value == "" || needle == "" {
		return 0
	}
	if strings.Contains(strings.ToLower(value), needle) {
		return 1
	}
	return 0
}

func minGroupInt(left int, right int) int {
	if left < right {
		return left
	}
	return right
}
