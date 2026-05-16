package group

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseConfigReadsReposAndLinks(t *testing.T) {
	config, err := ParseConfig(`version: 1
name: company
description: "All company microservices"

repos:
  hr/hiring/backend: hr-hiring-backend
  hr/hiring/ui: hr-hiring-ui

links:
  - from: hr/hiring/backend
    to: hr/hiring/ui
    type: http
    contract: "/api/users"
    role: provider

detect:
  http: true
  grpc: false
  topics: false
  shared_libs: true
  embedding_fallback: false

matching:
  bm25_threshold: 0.7
  embedding_threshold: 0.65
  max_candidates_per_step: 3

packages:
  hr/common:
    npm: "@hr/common"
`)
	if err != nil {
		t.Fatalf("ParseConfig() error = %v", err)
	}
	if config.Name != "company" || config.Description != "All company microservices" || config.Version != 1 {
		t.Fatalf("config identity = %#v", config)
	}
	if len(config.Repos) != 2 || config.Repos["hr/hiring/backend"] != "hr-hiring-backend" || config.Repos["hr/hiring/ui"] != "hr-hiring-ui" {
		t.Fatalf("repos = %#v", config.Repos)
	}
	if len(config.Links) != 1 || config.Links[0].Type != "http" || config.Links[0].Role != "provider" {
		t.Fatalf("links = %#v", config.Links)
	}
	if !config.Detect.HTTP || config.Detect.GRPC || config.Detect.Topics || !config.Detect.SharedLibs || config.Detect.EmbeddingFallback {
		t.Fatalf("detect = %#v", config.Detect)
	}
	if config.Matching.BM25Threshold != 0.7 || config.Matching.EmbeddingThreshold != 0.65 || config.Matching.MaxCandidatesPerStep != 3 {
		t.Fatalf("matching = %#v", config.Matching)
	}
	if config.Packages["hr/common"]["npm"] != "@hr/common" {
		t.Fatalf("packages = %#v", config.Packages)
	}
}

func TestParseConfigAppliesDefaultsAndAllowsEmptyRepos(t *testing.T) {
	config, err := ParseConfig(`version: 1
name: test
repos:
  app: my-app
`)
	if err != nil {
		t.Fatalf("ParseConfig() error = %v", err)
	}
	if config.Description != "" || len(config.Links) != 0 || len(config.Packages) != 0 {
		t.Fatalf("defaults = %#v", config)
	}
	if !config.Detect.HTTP || config.Matching.BM25Threshold != 0.7 {
		t.Fatalf("detect/matching defaults = %#v %#v", config.Detect, config.Matching)
	}

	empty, err := ParseConfig(`version: 1
name: new-group
repos: {}
`)
	if err != nil {
		t.Fatalf("ParseConfig(empty repos) error = %v", err)
	}
	if len(empty.Repos) != 0 {
		t.Fatalf("empty repos = %#v", empty.Repos)
	}
}

func TestParseConfigRejectsMissingRequiredFields(t *testing.T) {
	tests := []struct {
		name string
		yaml string
		want string
	}{
		{"missing name", "version: 1\nrepos: {}", "name"},
		{"missing version", "name: test\nrepos: {}", "version"},
		{"missing repos", "version: 1\nname: test", "repos"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseConfig(tt.yaml)
			if err == nil || !strings.Contains(strings.ToLower(err.Error()), tt.want) {
				t.Fatalf("ParseConfig() error = %v, want containing %q", err, tt.want)
			}
		})
	}
}

func TestParseConfigRejectsInvalidLinks(t *testing.T) {
	_, err := ParseConfig(`version: 1
name: test-group

repos:
  app/backend: test-backend
  app/frontend: test-frontend

links:
  - from: app/frontend
    to: app/backend
    type: websocket
    contract: ""
    role: reader
`)
	if err == nil {
		t.Fatal("ParseConfig() expected invalid link error")
	}
}

func TestParseConfigRejectsInvalidVersionRoleAndRepoRefs(t *testing.T) {
	if _, err := ParseConfig("version: 2\nname: test\nrepos:\n  a: b"); err == nil || !strings.Contains(strings.ToLower(err.Error()), "version") {
		t.Fatalf("invalid version error = %v", err)
	}
	if _, err := ParseConfig(`version: 1
name: test
repos:
  a: repo-a
  b: repo-b
links:
  - from: a
    to: b
    type: http
    contract: "/api"
    role: invalid
`); err == nil || !strings.Contains(strings.ToLower(err.Error()), "role") {
		t.Fatalf("invalid role error = %v", err)
	}
	if _, err := ParseConfig(`version: 1
name: test
repos:
  a: repo-a
links:
  - from: a
    to: nonexistent
    type: http
    contract: "/api"
    role: provider
`); err == nil || !strings.Contains(err.Error(), "nonexistent") {
		t.Fatalf("invalid repo ref error = %v", err)
	}
}

func TestLoadReadsGroupConfigFromDisk(t *testing.T) {
	homeDir := t.TempDir()
	groupDir := filepath.Join(homeDir, "groups", "disk-test")
	if err := os.MkdirAll(groupDir, 0o755); err != nil {
		t.Fatalf("mkdir group: %v", err)
	}
	if err := os.WriteFile(filepath.Join(groupDir, "group.yaml"), []byte(`version: 1
name: disk-test
repos:
  a: repo-a
`), 0o644); err != nil {
		t.Fatalf("write group.yaml: %v", err)
	}
	config, err := Load(homeDir, "disk-test")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if config.Name != "disk-test" || config.Repos["a"] != "repo-a" {
		t.Fatalf("config = %#v", config)
	}
}

func TestGroupTypesHaveRequiredShape(t *testing.T) {
	config := Config{
		Version:     1,
		Name:        "company",
		Description: "All company microservices",
		Repos:       map[string]string{"hr/hiring/backend": "hr-hiring-backend"},
		Links:       []ManifestLink{},
		Packages:    map[string]map[string]string{},
		Detect:      DetectConfig{HTTP: true, GRPC: true, Topics: true, SharedLibs: true, EmbeddingFallback: true},
		Matching:    MatchingConfig{BM25Threshold: 0.7, EmbeddingThreshold: 0.65, MaxCandidatesPerStep: 3},
	}
	if config.Version != 1 || config.Name != "company" {
		t.Fatalf("config = %#v", config)
	}
	registry := ContractRegistry{
		Version: 1,
		RepoSnapshots: map[string]RepoSnapshot{
			"hr/hiring/backend": {IndexedAt: "2026-03-30T21:14:14Z", LastCommit: "5838fb8d"},
		},
		MissingRepos: []string{},
		Contracts:    []StoredContract{},
		CrossLinks:   []CrossLink{},
	}
	if registry.Version != 1 || len(registry.Contracts) != 0 {
		t.Fatalf("registry = %#v", registry)
	}
	for _, contractType := range []string{"http", "grpc", "topic", "lib", "custom"} {
		contract := StoredContract{ContractID: contractType + "::test", Type: contractType, Role: "provider", SymbolUID: "uid-123", SymbolRef: SymbolRef{FilePath: "src/test.ts", Name: "testFn"}, SymbolName: "testFn", Confidence: 1, Meta: map[string]any{}}
		if contract.Type != contractType {
			t.Fatalf("contract = %#v", contract)
		}
	}
	link := CrossLink{
		From:       CrossLinkEndpoint{Repo: "frontend", SymbolUID: "uid-1", SymbolRef: SymbolRef{FilePath: "src/api.ts", Name: "fetchUsers"}},
		To:         CrossLinkEndpoint{Repo: "backend", SymbolUID: "uid-2", SymbolRef: SymbolRef{FilePath: "src/ctrl.ts", Name: "UserController.list"}},
		Type:       "http",
		ContractID: "http::GET::/api/users",
		MatchType:  "exact",
		Confidence: 1,
	}
	if link.MatchType != "exact" {
		t.Fatalf("link = %#v", link)
	}
	manifest := ManifestLink{From: "a", To: "b", Type: "http", Contract: "/x", Role: "provider"}
	if manifest.Contract != "/x" {
		t.Fatalf("manifest = %#v", manifest)
	}
}
