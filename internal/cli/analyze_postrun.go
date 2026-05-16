package cli

import (
	"time"

	"github.com/tamnguyendinh/avmatrix-go/internal/aicontext"
	"github.com/tamnguyendinh/avmatrix-go/internal/analyze"
	"github.com/tamnguyendinh/avmatrix-go/internal/repo"
)

type analyzeRegistration struct {
	Name string
	Meta repo.Meta
}

type analyzeRecordOptions struct {
	Name               string
	AllowDuplicateName bool
}

type analyzeAIContextResult struct {
	Files           []string
	GeneratedSkills []aicontext.GeneratedSkillInfo
	BaseSkillCount  int
}

func recordAnalyzeResult(result analyze.Result, options analyzeRecordOptions) (analyzeRegistration, error) {
	meta := repo.Meta{
		RepoPath:   result.RepoPath,
		LastCommit: repo.CurrentCommit(result.RepoPath),
		IndexedAt:  time.Now().UTC().Format(time.RFC3339),
		Stats:      statsFromAnalyzeResult(result),
	}
	paths := repo.Paths(result.RepoPath)
	if err := repo.SaveMeta(paths.StoragePath, meta); err != nil {
		return analyzeRegistration{}, err
	}
	name, err := repo.NewEnvStore().Register(result.RepoPath, meta, repo.RegisterOptions{
		Name:               options.Name,
		AllowDuplicateName: options.AllowDuplicateName,
	})
	if err != nil {
		return analyzeRegistration{}, err
	}
	return analyzeRegistration{Name: name, Meta: meta}, nil
}

func statsFromAnalyzeResult(result analyze.Result) *repo.Stats {
	files := result.Metrics.Files.Scanned
	nodes := 0
	edges := 0
	if result.Graph != nil {
		nodes = len(result.Graph.Nodes)
		edges = len(result.Graph.Relationships)
	}
	communities := result.Metrics.Communities.CommunitiesEmitted
	processes := result.Metrics.Processes.ProcessesEmitted
	stats := repo.Stats{
		Files:       &files,
		Nodes:       &nodes,
		Edges:       &edges,
		Communities: &communities,
		Processes:   &processes,
	}
	if result.Metrics.Embeddings.TotalNodes > 0 {
		embeddings := result.Metrics.Embeddings.EmbeddedNodes + result.Metrics.Embeddings.SkippedFreshNodes
		stats.Embeddings = &embeddings
	}
	return &stats
}

func generateAnalyzeAIContext(result analyze.Result, projectName string, skipAgentsMD bool, noStats bool) (analyzeAIContextResult, error) {
	generated, err := aicontext.Generate(result.RepoPath, projectName, result, aicontext.Options{
		SkipAgentsMD: skipAgentsMD,
		NoStats:      noStats,
	})
	if err != nil {
		return analyzeAIContextResult{}, err
	}
	return analyzeAIContextResult{
		Files:           generated.Files,
		GeneratedSkills: generated.Skills,
		BaseSkillCount:  len(generated.BaseSkillIDs),
	}, nil
}
