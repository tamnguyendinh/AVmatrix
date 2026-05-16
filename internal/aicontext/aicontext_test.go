package aicontext

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateAIContextFilesCreatesAndUpdatesManagedContext(t *testing.T) {
	dir := t.TempDir()
	stats := Stats{Nodes: 50, Edges: 100, Processes: 5}
	skills := []GeneratedSkillInfo{{Name: "core", Label: "Core", SymbolCount: 12, FileCount: 3}}

	files, baseSkills, err := GenerateAIContextFiles(dir, "TestProject", stats, skills, Options{})
	if err != nil {
		t.Fatalf("GenerateAIContextFiles: %v", err)
	}
	if len(files) == 0 {
		t.Fatalf("expected generated files")
	}
	if len(baseSkills) == 0 {
		t.Fatalf("expected base skills to be installed")
	}

	agentsPath := filepath.Join(dir, "AGENTS.md")
	content, err := os.ReadFile(agentsPath)
	if err != nil {
		t.Fatalf("read AGENTS.md: %v", err)
	}
	text := string(content)
	for _, want := range []string{
		startMarker,
		endMarker,
		"TestProject",
		"50 symbols, 100 relationships, 5 execution flows",
		"avmatrix://repo/TestProject/context",
		".claude/skills/generated/core/SKILL.md",
		"## Always Do",
		"## Never Do",
		"## Resources",
		"avmatrix-impact-analysis/SKILL.md",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("AGENTS.md missing %q:\n%s", want, text)
		}
	}
	for _, retired := range []string{
		"## Tools Quick Reference",
		"## Impact Risk Levels",
		"## Self-Check Before Finishing",
		"## When Debugging",
		"## When Refactoring",
		"## Keeping the Index Fresh",
		"`avmatrix analyze --force --skip-agents-md`",
		"avmatrix_impact",
		"avmatrix_detect_changes",
		"avmatrix_query",
		"avmatrix_context",
	} {
		if strings.Contains(text, retired) {
			t.Fatalf("AGENTS.md contains retired content %q:\n%s", retired, text)
		}
	}

	if _, _, err := GenerateAIContextFiles(dir, "TestProject", Stats{Nodes: 10}, nil, Options{}); err != nil {
		t.Fatalf("second GenerateAIContextFiles: %v", err)
	}
	updated, err := os.ReadFile(agentsPath)
	if err != nil {
		t.Fatalf("read updated AGENTS.md: %v", err)
	}
	if count := strings.Count(string(updated), "avmatrix:start"); count != 1 {
		t.Fatalf("expected one managed section after update, got %d:\n%s", count, updated)
	}

	for _, skill := range []string{"avmatrix-exploring", "avmatrix-cli"} {
		if _, err := os.Stat(filepath.Join(dir, ".claude", "skills", "avmatrix", skill, "SKILL.md")); err != nil {
			t.Fatalf("base skill %s was not installed: %v", skill, err)
		}
	}
}

func TestGenerateAIContextFilesSkipAgentsPreservesManualFiles(t *testing.T) {
	dir := t.TempDir()
	agentsPath := filepath.Join(dir, "AGENTS.md")
	claudePath := filepath.Join(dir, "CLAUDE.md")
	agentsContent := "# AGENTS\n\nCustom manual instructions only\n"
	claudeContent := "# CLAUDE\n\nCustom manual instructions only\n"
	if err := os.WriteFile(agentsPath, []byte(agentsContent), 0o644); err != nil {
		t.Fatalf("write AGENTS.md: %v", err)
	}
	if err := os.WriteFile(claudePath, []byte(claudeContent), 0o644); err != nil {
		t.Fatalf("write CLAUDE.md: %v", err)
	}

	files, baseSkills, err := GenerateAIContextFiles(dir, "TestProject", Stats{Nodes: 42}, nil, Options{SkipAgentsMD: true})
	if err != nil {
		t.Fatalf("GenerateAIContextFiles: %v", err)
	}
	if len(baseSkills) == 0 {
		t.Fatalf("expected base skills to be installed")
	}
	joined := strings.Join(files, "\n")
	for _, want := range []string{"AGENTS.md (skipped via --skip-agents-md)", "CLAUDE.md (skipped via --skip-agents-md)"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("generated file list missing %q: %v", want, files)
		}
	}
	if got, err := os.ReadFile(agentsPath); err != nil || string(got) != agentsContent {
		t.Fatalf("AGENTS.md changed: err=%v content=%q", err, got)
	}
	if got, err := os.ReadFile(claudePath); err != nil || string(got) != claudeContent {
		t.Fatalf("CLAUDE.md changed: err=%v content=%q", err, got)
	}
}

func TestRenderCrossRepoGroupsSectionMentionsSupportedToolsAndCommands(t *testing.T) {
	section := FormatCrossRepoGroupsSection([]string{"CorePlatform"})
	for _, want := range []string{
		"CorePlatform",
		"`group_list`",
		"`group_status`",
		"`group_sync`",
		"`group_contracts`",
		"`group_query`",
		"`avmatrix group list`",
		"`avmatrix group status <name>`",
		"`avmatrix group sync <name>`",
		"`avmatrix group contracts <name>`",
		"`avmatrix group query <name> \"<query>\"`",
	} {
		if !strings.Contains(section, want) {
			t.Fatalf("cross-repo section missing %q:\n%s", want, section)
		}
	}
	if strings.Contains(section, "group_impact") || strings.Contains(section, "avmatrix group impact") {
		t.Fatalf("cross-repo section mentions retired group impact:\n%s", section)
	}
	if got := FormatCrossRepoGroupsSection(nil); got != "" {
		t.Fatalf("nil groups should render empty section, got %q", got)
	}
}
