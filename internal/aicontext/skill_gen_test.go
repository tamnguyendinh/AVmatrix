package aicontext

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tamnguyendinh/avmatrix-go/internal/graph"
	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

func TestGenerateSkillFilesReturnsEmptyForNilOrSmallCommunities(t *testing.T) {
	dir := t.TempDir()

	skills, outputPath, err := GenerateSkillFiles(dir, "TestProject", nil)
	if err != nil {
		t.Fatalf("GenerateSkillFiles(nil): %v", err)
	}
	if len(skills) != 0 {
		t.Fatalf("nil graph skills = %#v, want none", skills)
	}
	if outputPath != filepath.Join(dir, ".claude", "skills", "generated") {
		t.Fatalf("output path = %q, want generated skills path", outputPath)
	}

	g := graph.New()
	for index := 0; index < 2; index++ {
		nodeID := "Function:small:" + string(rune('a'+index))
		g.AddNode(testSkillNode(nodeID, "smallFunc", scopeir.NodeFunction, filepath.Join(dir, "src", "small.go"), 1, false))
		g.AddRelationship(testSkillMemberRelationship(nodeID, "Community:small"))
	}
	g.AddNode(testSkillCommunity("Community:small", "Small", 2, 0.6))

	skills, _, err = GenerateSkillFiles(dir, "TestProject", g)
	if err != nil {
		t.Fatalf("GenerateSkillFiles(small): %v", err)
	}
	if len(skills) != 0 {
		t.Fatalf("small community skills = %#v, want none", skills)
	}
}

func TestGenerateSkillFilesSelectsSortsAndCapsCommunities(t *testing.T) {
	dir := t.TempDir()
	g := graph.New()
	sizes := []int{5, 10, 3}
	for communityIndex, size := range sizes {
		communityID := "Community:area-" + string(rune('a'+communityIndex))
		g.AddNode(testSkillCommunity(communityID, "Area"+string(rune('A'+communityIndex)), size, 0.75))
		for nodeIndex := 0; nodeIndex < size; nodeIndex++ {
			nodeID := communityID + ":fn:" + string(rune('a'+nodeIndex))
			g.AddNode(testSkillNode(nodeID, "func", scopeir.NodeFunction, filepath.Join(dir, "src", "area", "f.go"), nodeIndex+1, false))
			g.AddRelationship(testSkillMemberRelationship(nodeID, communityID))
		}
	}
	for communityIndex := 0; communityIndex < 25; communityIndex++ {
		communityID := "Community:cap-" + string(rune('a'+communityIndex))
		g.AddNode(testSkillCommunity(communityID, "Cap"+string(rune('A'+communityIndex)), 4, 0.5))
		for nodeIndex := 0; nodeIndex < 4; nodeIndex++ {
			nodeID := communityID + ":fn:" + string(rune('a'+nodeIndex))
			g.AddNode(testSkillNode(nodeID, "capFunc", scopeir.NodeFunction, filepath.Join(dir, "src", "cap", "f.go"), nodeIndex+1, false))
			g.AddRelationship(testSkillMemberRelationship(nodeID, communityID))
		}
	}

	skills, _, err := GenerateSkillFiles(dir, "TestProject", g)
	if err != nil {
		t.Fatalf("GenerateSkillFiles: %v", err)
	}
	if len(skills) != 20 {
		t.Fatalf("skill count = %d, want cap of 20", len(skills))
	}
	if skills[0].SymbolCount != 10 || skills[1].SymbolCount != 5 {
		t.Fatalf("skills not sorted by symbol count desc: %#v", skills[:3])
	}
}

func TestGenerateSkillFilesWritesSkillMarkdownWithProcesses(t *testing.T) {
	dir := t.TempDir()
	g := graph.New()
	communityID := "Community:auth"
	g.AddNode(testSkillCommunity(communityID, "Auth", 5, 0.82))
	for index := 0; index < 5; index++ {
		nodeID := "Function:auth:" + string(rune('a'+index))
		g.AddNode(testSkillNode(nodeID, "authFunc"+string(rune('A'+index)), scopeir.NodeFunction, filepath.Join(dir, "src", "auth", "file.go"), index+1, index < 2))
		g.AddRelationship(testSkillMemberRelationship(nodeID, communityID))
	}
	g.AddNode(graph.Node{
		ID:    "Process:auth-flow",
		Label: scopeir.NodeProcess,
		Properties: graph.NodeProperties{
			"heuristicLabel": "AuthFlow",
			"processType":    "intra_community",
			"stepCount":      5,
			"communities":    []string{communityID},
		},
	})

	skills, _, err := GenerateSkillFiles(dir, "TestProject", g)
	if err != nil {
		t.Fatalf("GenerateSkillFiles: %v", err)
	}
	if len(skills) != 1 {
		t.Fatalf("skills = %#v, want one skill", skills)
	}
	if skills[0].Name != "auth" || skills[0].Label != "Auth" || skills[0].SymbolCount != 5 || skills[0].FileCount != 1 {
		t.Fatalf("skill metadata = %#v, want auth metadata", skills[0])
	}

	content := readSkillFile(t, dir, "auth")
	for _, want := range []string{
		"---\nname: auth",
		"description: \"Skill for the Auth area of TestProject. 5 symbols across 1 files.\"",
		"5 symbols | 1 files | Cohesion: 82%",
		"## Key Files",
		"src/auth/file.go",
		"## Key Symbols",
		"authFuncA",
		"## Execution Flows",
		"AuthFlow",
		"## How to Explore",
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("SKILL.md missing %q:\n%s", want, content)
		}
	}
}

func TestGenerateSkillFilesCleansPreviousRunAndSanitizesNames(t *testing.T) {
	dir := t.TempDir()

	first := graph.New()
	first.AddNode(testSkillCommunity("Community:first", "First", 4, 0.7))
	for index := 0; index < 4; index++ {
		nodeID := "Function:first:" + string(rune('a'+index))
		first.AddNode(testSkillNode(nodeID, "firstFunc", scopeir.NodeFunction, filepath.Join(dir, "src", "first", "f.go"), 1, false))
		first.AddRelationship(testSkillMemberRelationship(nodeID, "Community:first"))
	}
	if _, _, err := GenerateSkillFiles(dir, "TestProject", first); err != nil {
		t.Fatalf("first GenerateSkillFiles: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".claude", "skills", "generated", "first", "SKILL.md")); err != nil {
		t.Fatalf("first skill missing: %v", err)
	}

	second := graph.New()
	second.AddNode(testSkillCommunity("Community:cpp", "C++ Core", 4, 0.7))
	second.AddNode(testSkillCommunity("Community:cpp2", "C++ Core", 4, 0.7))
	for communityIndex, communityID := range []string{"Community:cpp", "Community:cpp2"} {
		for nodeIndex := 0; nodeIndex < 4; nodeIndex++ {
			nodeID := communityID + ":fn:" + string(rune('a'+nodeIndex))
			second.AddNode(testSkillNode(nodeID, "cppFunc", scopeir.NodeFunction, filepath.Join(dir, "src", "cpp", "f.go"), communityIndex+nodeIndex+1, false))
			second.AddRelationship(testSkillMemberRelationship(nodeID, communityID))
		}
	}
	skills, _, err := GenerateSkillFiles(dir, "TestProject", second)
	if err != nil {
		t.Fatalf("second GenerateSkillFiles: %v", err)
	}
	if len(skills) != 2 || skills[0].Name != "c-core" || skills[1].Name != "c-core-2" {
		t.Fatalf("sanitized skill names = %#v, want c-core and c-core-2", skills)
	}
	if _, err := os.Stat(filepath.Join(dir, ".claude", "skills", "generated", "first")); !os.IsNotExist(err) {
		t.Fatalf("first run output was not cleaned: %v", err)
	}
}

func TestGenerateSkillFilesHandlesMissingAndWindowsStylePaths(t *testing.T) {
	dir := t.TempDir()
	g := graph.New()
	communityID := "Community:paths"
	g.AddNode(testSkillCommunity(communityID, "Paths", 4, 0.8))
	for index := 0; index < 2; index++ {
		nodeID := "Function:missing:" + string(rune('a'+index))
		g.AddNode(testSkillNode(nodeID, "missingPath", scopeir.NodeFunction, "", 0, false))
		g.AddRelationship(testSkillMemberRelationship(nodeID, communityID))
	}
	for index := 0; index < 2; index++ {
		nodeID := "Function:windows:" + string(rune('a'+index))
		g.AddNode(testSkillNode(nodeID, "windowsPath", scopeir.NodeFunction, dir+"\\src\\win\\f.go", index+1, false))
		g.AddRelationship(testSkillMemberRelationship(nodeID, communityID))
	}

	skills, _, err := GenerateSkillFiles(dir, "TestProject", g)
	if err != nil {
		t.Fatalf("GenerateSkillFiles: %v", err)
	}
	if len(skills) != 1 || skills[0].FileCount != 1 {
		t.Fatalf("skills = %#v, want one resolved file", skills)
	}
	content := readSkillFile(t, dir, "paths")
	keyFiles := content
	if start := strings.Index(content, "## Key Files"); start >= 0 {
		keyFiles = content[start:]
	}
	if strings.Contains(keyFiles, "\\") {
		t.Fatalf("Key Files section contains backslash paths:\n%s", keyFiles)
	}
	if !strings.Contains(keyFiles, "src/win/f.go") {
		t.Fatalf("Key Files section missing normalized path:\n%s", keyFiles)
	}
}

func testSkillNode(id string, name string, label scopeir.NodeLabel, filePath string, startLine int, exported bool) graph.Node {
	return graph.Node{
		ID:    id,
		Label: label,
		Properties: graph.NodeProperties{
			"name":       name,
			"filePath":   filePath,
			"startLine":  startLine,
			"endLine":    startLine + 1,
			"isExported": exported,
		},
	}
}

func testSkillCommunity(id string, label string, symbolCount int, cohesion float64) graph.Node {
	return graph.Node{
		ID:    id,
		Label: scopeir.NodeCommunity,
		Properties: graph.NodeProperties{
			"heuristicLabel": label,
			"symbolCount":    symbolCount,
			"cohesion":       cohesion,
		},
	}
}

func testSkillMemberRelationship(nodeID string, communityID string) graph.Relationship {
	return graph.Relationship{
		ID:       "rel:" + nodeID + "->" + communityID,
		SourceID: nodeID,
		TargetID: communityID,
		Type:     graph.RelMemberOf,
	}
}

func readSkillFile(t *testing.T, repoPath string, name string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(repoPath, ".claude", "skills", "generated", name, "SKILL.md"))
	if err != nil {
		t.Fatalf("read skill %s: %v", name, err)
	}
	return string(raw)
}
