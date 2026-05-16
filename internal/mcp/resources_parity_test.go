package mcp

import (
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/tamnguyendinh/avmatrix-go/internal/repo"
)

func TestResourceDefinitionsAndTemplatesParity(t *testing.T) {
	definitions := resourceDefinitions()
	if len(definitions) != 2 {
		t.Fatalf("resourceDefinitions() length = %d, want 2", len(definitions))
	}
	assertResourceDefinition(t, definitions, "avmatrix://repos", "text/yaml")
	assertResourceDefinition(t, definitions, "avmatrix://setup", "text/markdown")

	templates := resourceTemplates()
	if len(templates) != 6 {
		t.Fatalf("resourceTemplates() length = %d, want 6", len(templates))
	}
	for _, want := range []string{
		"avmatrix://repo/{name}/context",
		"avmatrix://repo/{name}/clusters",
		"avmatrix://repo/{name}/processes",
		"avmatrix://repo/{name}/schema",
		"avmatrix://repo/{name}/cluster/{clusterName}",
		"avmatrix://repo/{name}/process/{processName}",
	} {
		assertResourceTemplate(t, templates, want, "text/yaml")
	}
}

func TestParseRepoResourceURIDecodesRepoAndDetailNames(t *testing.T) {
	request, ok, err := parseRepoResourceURI("avmatrix://repo/my%20project/context")
	if err != nil || !ok {
		t.Fatalf("parse context uri ok=%v err=%v", ok, err)
	}
	if request.RepoName != "my project" || request.ResourceType != "context" || request.Param != "" {
		t.Fatalf("context request = %#v", request)
	}

	request, ok, err = parseRepoResourceURI("avmatrix://repo/my%20project/cluster/Auth%20Module")
	if err != nil || !ok {
		t.Fatalf("parse cluster uri ok=%v err=%v", ok, err)
	}
	if request.RepoName != "my project" || request.ResourceType != "cluster" || request.Param != "Auth Module" {
		t.Fatalf("cluster request = %#v", request)
	}

	if _, ok, err := parseRepoResourceURI("avmatrix://repo/test/nonexistent/extra"); err != nil || ok {
		t.Fatalf("invalid deep resource ok=%v err=%v", ok, err)
	}
}

func TestReadResourceTextStaticReposAndSetup(t *testing.T) {
	store := repo.NewStore(t.TempDir())
	server := NewServer(Config{Store: store})

	text, mimeType, err := server.readResourceText("avmatrix://repos")
	if err != nil {
		t.Fatalf("read repos: %v", err)
	}
	if mimeType != "text/yaml" || !strings.Contains(text, "No repositories indexed") {
		t.Fatalf("empty repos resource mime=%q text=\n%s", mimeType, text)
	}

	text, mimeType, err = server.readResourceText("avmatrix://setup")
	if err != nil {
		t.Fatalf("read setup: %v", err)
	}
	if mimeType != "text/markdown" || !strings.Contains(text, "No repositories indexed") {
		t.Fatalf("empty setup resource mime=%q text=\n%s", mimeType, text)
	}

	files, nodes, edges, processes := 10, 50, 70, 5
	entries := []repo.RegistryEntry{
		{
			Name:       "my project",
			Path:       t.TempDir(),
			IndexedAt:  "2024-01-01",
			LastCommit: "abc1234",
			Stats:      &repo.Stats{Files: &files, Nodes: &nodes, Edges: &edges, Processes: &processes},
		},
		{
			Name:       "other",
			Path:       t.TempDir(),
			IndexedAt:  "2024-01-02",
			LastCommit: "def5678",
			Stats:      &repo.Stats{},
		},
	}
	if err := store.WriteRegistry(entries); err != nil {
		t.Fatalf("write registry: %v", err)
	}

	text, mimeType, err = server.readResourceText("avmatrix://repos")
	if err != nil {
		t.Fatalf("read populated repos: %v", err)
	}
	for _, want := range []string{"my project", "files: 10", "symbols: 50", "processes: 5", "Multiple repos indexed", "repo parameter"} {
		if !strings.Contains(text, want) {
			t.Fatalf("repos resource missing %q:\n%s", want, text)
		}
	}

	text, mimeType, err = server.readResourceText("avmatrix://setup")
	if err != nil {
		t.Fatalf("read populated setup: %v", err)
	}
	if mimeType != "text/markdown" {
		t.Fatalf("setup mimeType = %q, want text/markdown", mimeType)
	}
	for _, want := range []string{"AVmatrix MCP - my project", "50 symbols, 70 relationships, 5 execution flows", "avmatrix://repo/my%20project/context"} {
		if !strings.Contains(text, want) {
			t.Fatalf("setup resource missing %q:\n%s", want, text)
		}
	}
}

func TestReadResourceTextRejectsUnknownResources(t *testing.T) {
	server := NewServer(Config{Store: repo.NewStore(t.TempDir())})

	if _, _, err := server.readResourceText("avmatrix://unknown"); err == nil || !strings.Contains(err.Error(), "Unknown resource URI") {
		t.Fatalf("unknown uri error = %v", err)
	}
	if _, _, err := server.readResourceText("avmatrix://repo/test/nonexistent"); err == nil || !strings.Contains(err.Error(), "Unknown resource") {
		t.Fatalf("unknown repo-scoped resource error = %v", err)
	}
	if _, _, err := server.readResourceText("avmatrix://repo/%zz/context"); err == nil {
		t.Fatal("invalid URI escape unexpectedly succeeded")
	}
}

func TestMCPStalenessHintMatchesLegacyGitBehavior(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	repoPath := t.TempDir()
	runResourceTestGit(t, repoPath, "init")
	runResourceTestGit(t, repoPath, "config", "user.email", "test@example.com")
	runResourceTestGit(t, repoPath, "config", "user.name", "Test User")
	if err := os.WriteFile(repoPath+"/file.txt", []byte("one\n"), 0o644); err != nil {
		t.Fatalf("write first file: %v", err)
	}
	runResourceTestGit(t, repoPath, "add", "file.txt")
	runResourceTestGit(t, repoPath, "commit", "-m", "first")
	firstCommit := strings.TrimSpace(runResourceTestGit(t, repoPath, "rev-parse", "HEAD"))
	if err := os.WriteFile(repoPath+"/file.txt", []byte("two\n"), 0o644); err != nil {
		t.Fatalf("write second file: %v", err)
	}
	runResourceTestGit(t, repoPath, "commit", "-am", "second")
	headCommit := strings.TrimSpace(runResourceTestGit(t, repoPath, "rev-parse", "HEAD"))

	if hint := mcpStalenessHint(repoPath, headCommit); hint != "" {
		t.Fatalf("fresh staleness hint = %q, want empty", hint)
	}
	hint := mcpStalenessHint(repoPath, firstCommit)
	if !strings.Contains(hint, "1 commit behind HEAD") || !strings.Contains(hint, "Run analyze tool") {
		t.Fatalf("stale hint = %q, want behind HEAD guidance", hint)
	}
	if hint := mcpStalenessHint("/path/that/does/not/exist", "abc123"); hint != "" {
		t.Fatalf("invalid repo hint = %q, want empty", hint)
	}
	if hint := mcpStalenessHint(repoPath, "not-a-real-commit-hash"); hint != "" {
		t.Fatalf("invalid commit hint = %q, want empty", hint)
	}
}

func runResourceTestGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, output)
	}
	return string(output)
}

func assertResourceDefinition(t *testing.T, definitions []resourceDefinition, uri string, mimeType string) {
	t.Helper()
	for _, definition := range definitions {
		if definition.URI != uri {
			continue
		}
		if definition.Name == "" || definition.Description == "" {
			t.Fatalf("definition %s missing metadata: %#v", uri, definition)
		}
		if definition.MimeType != mimeType {
			t.Fatalf("definition %s mimeType = %q, want %q", uri, definition.MimeType, mimeType)
		}
		return
	}
	t.Fatalf("definition %s not found: %#v", uri, definitions)
}

func assertResourceTemplate(t *testing.T, templates []resourceTemplate, uriTemplate string, mimeType string) {
	t.Helper()
	for _, template := range templates {
		if template.URITemplate != uriTemplate {
			continue
		}
		if template.Name == "" || template.Description == "" {
			t.Fatalf("template %s missing metadata: %#v", uriTemplate, template)
		}
		if template.MimeType != mimeType {
			t.Fatalf("template %s mimeType = %q, want %q", uriTemplate, template.MimeType, mimeType)
		}
		return
	}
	t.Fatalf("template %s not found: %#v", uriTemplate, templates)
}
