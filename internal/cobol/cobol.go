package cobol

import (
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/tamnguyendinh/avmatrix-go/internal/graph"
	"github.com/tamnguyendinh/avmatrix-go/internal/scanner"
	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

var (
	programIDPattern  = regexp.MustCompile(`(?i)\bPROGRAM-ID\.\s*([A-Z][A-Z0-9-]*)`)
	sectionPattern    = regexp.MustCompile(`(?i)^\s*([A-Z][A-Z0-9-]*)\s+SECTION(?:\s+\d+)?\.\s*$`)
	paragraphPattern  = regexp.MustCompile(`(?i)^\s*([A-Z][A-Z0-9-]*)\.\s*$`)
	performPattern    = regexp.MustCompile(`(?i)\bPERFORM\s+([A-Z][A-Z0-9-]*)(?:\s+(?:THRU|THROUGH)\s+([A-Z][A-Z0-9-]*))?`)
	callPattern       = regexp.MustCompile(`(?i)\bCALL\s+["']?([A-Z][A-Z0-9-]*)["']?`)
	copyPattern       = regexp.MustCompile(`(?i)\bCOPY\s+["']?([A-Z][A-Z0-9-]*)["']?`)
	endProgramPattern = regexp.MustCompile(`(?i)^\s*END\s+PROGRAM(?:\s+([A-Z][A-Z0-9-]*))?\.\s*$`)
	jclJobPattern     = regexp.MustCompile(`(?i)^//([A-Z0-9$#@-]+)\s+JOB\b(.*)`)
	jclExecPattern    = regexp.MustCompile(`(?i)^//([A-Z0-9$#@-]+)\s+EXEC\s+(?:PGM=([A-Z0-9$#@-]+)|PROC=([A-Z0-9$#@-]+)|([A-Z0-9$#@-]+))`)
)

type Result struct {
	Metrics Metrics
}

type Metrics struct {
	CobolFiles        int `json:"cobolFiles,omitempty"`
	Copybooks         int `json:"copybooks,omitempty"`
	JCLFiles          int `json:"jclFiles,omitempty"`
	Programs          int `json:"programs,omitempty"`
	Sections          int `json:"sections,omitempty"`
	Paragraphs        int `json:"paragraphs,omitempty"`
	Performs          int `json:"performs,omitempty"`
	Calls             int `json:"calls,omitempty"`
	Copies            int `json:"copies,omitempty"`
	JCLJobs           int `json:"jclJobs,omitempty"`
	JCLSteps          int `json:"jclSteps,omitempty"`
	JCLProgramLinks   int `json:"jclProgramLinks,omitempty"`
	MetadataFileNodes int `json:"metadataFileNodes,omitempty"`
}

type fileKind string

const (
	kindProgram  fileKind = "program"
	kindCopybook fileKind = "copybook"
	kindJCL      fileKind = "jcl"
)

type sourceFile struct {
	Path    string
	Kind    fileKind
	Content string
	Scanner scanner.File
}

type programInfo struct {
	FilePath     string
	Content      string
	ProgramName  string
	ParentName   string
	NestingDepth int
	StartLine    int
	EndLine      int
	Lines        []string
	Sections     []sectionFact
	Paragraphs   []paragraphFact
	Performs     []performFact
	Calls        []callFact
	Copies       []copyFact
}

type sectionFact struct {
	Name string
	Line int
}

type paragraphFact struct {
	Name string
	Line int
}

type performFact struct {
	Target     string
	ThruTarget string
	Line       int
}

type callFact struct {
	Target string
	Line   int
}

type copyFact struct {
	Target string
	Line   int
}

func Apply(g *graph.Graph, repoPath string, files []scanner.File) (Result, error) {
	if g == nil {
		return Result{}, nil
	}
	ordered := collectMainframeFiles(files)
	result := Result{}
	sources := make([]sourceFile, 0, len(ordered))
	for _, file := range ordered {
		kind := mainframeKind(file.Path)
		result.count(kind)
		if markFileNode(g, file, kind) {
			result.Metrics.MetadataFileNodes++
		}
		raw, err := os.ReadFile(filepath.Join(repoPath, filepath.FromSlash(file.Path)))
		if err != nil {
			return result, err
		}
		sources = append(sources, sourceFile{
			Path:    file.Path,
			Kind:    kind,
			Content: string(raw),
			Scanner: file,
		})
	}

	copybookPaths := map[string]string{}
	programs := make([]programInfo, 0)
	moduleIDs := map[string]string{}
	for _, source := range sources {
		switch source.Kind {
		case kindCopybook:
			copybookPaths[strings.ToUpper(basenameNoExt(source.Path))] = source.Path
		case kindProgram:
			for _, program := range extractPrograms(source.Path, source.Content) {
				if program.ProgramName == "" {
					continue
				}
				moduleID := addProgramModule(g, program)
				if moduleID == "" {
					continue
				}
				moduleIDs[strings.ToUpper(program.ProgramName)] = moduleID
				programs = append(programs, program)
				result.Metrics.Programs++
			}
		}
	}

	emitNestedProgramRelationships(g, programs, moduleIDs)
	for _, program := range programs {
		emitProgramDetails(g, program, moduleIDs, copybookPaths, &result.Metrics)
	}
	for _, source := range sources {
		if source.Kind == kindJCL {
			processJCL(g, source.Path, source.Content, moduleIDs, &result.Metrics)
		}
	}
	return result, nil
}

func collectMainframeFiles(files []scanner.File) []scanner.File {
	out := make([]scanner.File, 0)
	for _, file := range files {
		file.Path = normalizePath(file.Path)
		if file.Path == "" || mainframeKind(file.Path) == "" {
			continue
		}
		out = append(out, file)
	}
	sort.Slice(out, func(i int, j int) bool { return out[i].Path < out[j].Path })
	return out
}

func (result *Result) count(kind fileKind) {
	switch kind {
	case kindProgram:
		result.Metrics.CobolFiles++
	case kindCopybook:
		result.Metrics.CobolFiles++
		result.Metrics.Copybooks++
	case kindJCL:
		result.Metrics.JCLFiles++
	}
}

func markFileNode(g *graph.Graph, file scanner.File, kind fileKind) bool {
	fileNodeID := graph.GenerateID(string(scopeir.NodeFile), file.Path)
	node, ok := g.GetNode(fileNodeID)
	if !ok {
		return false
	}
	if node.Properties == nil {
		node.Properties = graph.NodeProperties{}
	}
	node.Properties["language"] = string(scanner.Cobol)
	node.Properties["mainframeKind"] = string(kind)
	node.Properties["fileExtension"] = strings.TrimPrefix(strings.ToLower(path.Ext(file.Path)), ".")
	node.Properties["binary"] = false
	g.AddNode(node)
	return true
}

func extractProgram(filePath string, content string) programInfo {
	programs := extractPrograms(filePath, content)
	if len(programs) == 0 {
		return programInfo{FilePath: filePath, Content: content, Lines: strings.Split(preprocessCobolSource(content), "\n")}
	}
	return programs[0]
}

func extractPrograms(filePath string, content string) []programInfo {
	normalized := preprocessCobolSource(content)
	lines := strings.Split(normalized, "\n")
	programs := make([]*programInfo, 0)
	stack := make([]*programInfo, 0)
	inProcedure := map[*programInfo]bool{}
	for index, line := range lines {
		line = strings.TrimRight(line, "\r")
		if isCobolComment(line) {
			continue
		}
		lineNumber := index + 1
		if match := programIDPattern.FindStringSubmatch(line); match != nil {
			name := strings.ToUpper(match[1])
			parentName := ""
			if len(stack) > 0 {
				parentName = stack[len(stack)-1].ProgramName
			}
			program := &programInfo{
				FilePath:     filePath,
				Content:      normalized,
				ProgramName:  name,
				ParentName:   parentName,
				NestingDepth: len(stack),
				StartLine:    lineNumber,
				EndLine:      len(lines),
				Lines:        lines,
			}
			programs = append(programs, program)
			stack = append(stack, program)
			inProcedure[program] = false
			continue
		}
		if len(stack) == 0 {
			continue
		}
		current := stack[len(stack)-1]
		if match := endProgramPattern.FindStringSubmatch(line); match != nil {
			current.EndLine = lineNumber
			stack = popProgramStack(stack, match[1])
			continue
		}
		upperLine := strings.ToUpper(strings.TrimSpace(line))
		if strings.HasPrefix(upperLine, "PROCEDURE DIVISION") {
			inProcedure[current] = true
		} else if strings.HasSuffix(upperLine, " DIVISION.") {
			inProcedure[current] = false
		}
		if match := sectionPattern.FindStringSubmatch(line); match != nil && inProcedure[current] && isCobolHeaderArea(line) {
			current.Sections = append(current.Sections, sectionFact{Name: strings.ToUpper(match[1]), Line: lineNumber})
		}
		if match := paragraphPattern.FindStringSubmatch(line); match != nil && inProcedure[current] && isCobolHeaderArea(line) {
			name := strings.ToUpper(match[1])
			if !isReservedParagraphName(name) {
				current.Paragraphs = append(current.Paragraphs, paragraphFact{Name: name, Line: lineNumber})
			}
		}
		if inProcedure[current] {
			for _, match := range performPattern.FindAllStringSubmatchIndex(line, -1) {
				target := strings.ToUpper(line[match[2]:match[3]])
				thruTarget := ""
				if match[4] >= 0 {
					thruTarget = strings.ToUpper(line[match[4]:match[5]])
				}
				if isInlinePerformLoop(line[match[3]:], thruTarget) {
					continue
				}
				current.Performs = append(current.Performs, performFact{Target: target, ThruTarget: thruTarget, Line: lineNumber})
			}
			for _, match := range callPattern.FindAllStringSubmatch(line, -1) {
				current.Calls = append(current.Calls, callFact{Target: strings.ToUpper(match[1]), Line: lineNumber})
			}
		}
		for _, match := range copyPattern.FindAllStringSubmatch(line, -1) {
			current.Copies = append(current.Copies, copyFact{Target: strings.ToUpper(match[1]), Line: lineNumber})
		}
	}
	out := make([]programInfo, 0, len(programs))
	for _, program := range programs {
		out = append(out, *program)
	}
	return out
}

func addProgramModule(g *graph.Graph, program programInfo) string {
	fileNodeID := graph.GenerateID(string(scopeir.NodeFile), program.FilePath)
	if _, ok := g.GetNode(fileNodeID); !ok {
		return ""
	}
	moduleID := graph.GenerateID(string(scopeir.NodeModule), program.FilePath+":"+program.ProgramName)
	g.AddNode(graph.Node{
		ID:    moduleID,
		Label: scopeir.NodeModule,
		Properties: graph.NodeProperties{
			"name":       program.ProgramName,
			"filePath":   program.FilePath,
			"startLine":  program.StartLine,
			"endLine":    program.EndLine,
			"language":   string(scanner.Cobol),
			"isExported": true,
		},
	})
	if program.ParentName != "" {
		return moduleID
	}
	g.AddRelationship(graph.Relationship{
		ID:         graph.GenerateID(string(graph.RelContains), fileNodeID+"->"+moduleID),
		SourceID:   fileNodeID,
		TargetID:   moduleID,
		Type:       graph.RelContains,
		Confidence: 1,
		Reason:     "cobol-program-id",
	})
	return moduleID
}

func emitNestedProgramRelationships(g *graph.Graph, programs []programInfo, moduleIDs map[string]string) {
	for _, program := range programs {
		if program.ParentName == "" {
			continue
		}
		parentID := moduleIDs[strings.ToUpper(program.ParentName)]
		childID := moduleIDs[strings.ToUpper(program.ProgramName)]
		if parentID == "" || childID == "" {
			continue
		}
		g.AddRelationship(graph.Relationship{
			ID:         graph.GenerateID(string(graph.RelContains), parentID+"->"+childID),
			SourceID:   parentID,
			TargetID:   childID,
			Type:       graph.RelContains,
			Confidence: 1,
			Reason:     "cobol-nested-program",
		})
	}
}

func emitProgramDetails(g *graph.Graph, program programInfo, moduleIDs map[string]string, copybookPaths map[string]string, metrics *Metrics) {
	moduleID := moduleIDs[strings.ToUpper(program.ProgramName)]
	if moduleID == "" {
		return
	}
	sectionIDs := emitSections(g, program, moduleID, metrics)
	paragraphIDs := emitParagraphs(g, program, moduleID, sectionIDs, metrics)
	emitPerforms(g, program, moduleID, paragraphIDs, sectionIDs, metrics)
	emitCalls(g, program, moduleID, moduleIDs, metrics)
	emitCopies(g, program, copybookPaths, metrics)
}

func emitSections(g *graph.Graph, program programInfo, moduleID string, metrics *Metrics) map[string]string {
	sectionIDs := map[string]string{}
	for index, section := range program.Sections {
		endLine := programEndLine(program)
		if index+1 < len(program.Sections) {
			endLine = program.Sections[index+1].Line - 1
		}
		sectionID := graph.GenerateID(string(scopeir.NodeNamespace), program.FilePath+":"+program.ProgramName+":"+section.Name)
		g.AddNode(graph.Node{
			ID:    sectionID,
			Label: scopeir.NodeNamespace,
			Properties: graph.NodeProperties{
				"name":       section.Name,
				"filePath":   program.FilePath,
				"startLine":  section.Line,
				"endLine":    endLine,
				"language":   string(scanner.Cobol),
				"isExported": true,
			},
		})
		g.AddRelationship(graph.Relationship{
			ID:         graph.GenerateID(string(graph.RelContains), moduleID+"->"+sectionID),
			SourceID:   moduleID,
			TargetID:   sectionID,
			Type:       graph.RelContains,
			Confidence: 1,
			Reason:     "cobol-section",
		})
		sectionIDs[section.Name] = sectionID
		metrics.Sections++
	}
	return sectionIDs
}

func emitParagraphs(g *graph.Graph, program programInfo, moduleID string, sectionIDs map[string]string, metrics *Metrics) map[string]string {
	paragraphIDs := map[string]string{}
	for index, paragraph := range program.Paragraphs {
		endLine := programEndLine(program)
		if index+1 < len(program.Paragraphs) {
			endLine = program.Paragraphs[index+1].Line - 1
		}
		paragraphID := graph.GenerateID(string(scopeir.NodeFunction), program.FilePath+":"+program.ProgramName+":"+paragraph.Name)
		g.AddNode(graph.Node{
			ID:    paragraphID,
			Label: scopeir.NodeFunction,
			Properties: graph.NodeProperties{
				"name":       paragraph.Name,
				"filePath":   program.FilePath,
				"startLine":  paragraph.Line,
				"endLine":    endLine,
				"language":   string(scanner.Cobol),
				"isExported": true,
			},
		})
		parentID := sectionForLine(program.Sections, sectionIDs, paragraph.Line)
		if parentID == "" {
			parentID = moduleID
		}
		g.AddRelationship(graph.Relationship{
			ID:         graph.GenerateID(string(graph.RelContains), parentID+"->"+paragraphID),
			SourceID:   parentID,
			TargetID:   paragraphID,
			Type:       graph.RelContains,
			Confidence: 1,
			Reason:     "cobol-paragraph",
		})
		paragraphIDs[paragraph.Name] = paragraphID
		metrics.Paragraphs++
	}
	return paragraphIDs
}

func emitPerforms(g *graph.Graph, program programInfo, moduleID string, paragraphIDs map[string]string, sectionIDs map[string]string, metrics *Metrics) {
	for _, perform := range program.Performs {
		targetID := paragraphIDs[perform.Target]
		if targetID == "" {
			targetID = sectionIDs[perform.Target]
		}
		if targetID == "" {
			continue
		}
		sourceID := paragraphForLine(program.Paragraphs, paragraphIDs, perform.Line)
		if sourceID == "" {
			sourceID = moduleID
		}
		g.AddRelationship(graph.Relationship{
			ID:         graph.GenerateID(string(graph.RelCalls), sourceID+"->perform->"+targetID+":L"+itoa(perform.Line)),
			SourceID:   sourceID,
			TargetID:   targetID,
			Type:       graph.RelCalls,
			Confidence: 1,
			Reason:     "cobol-perform",
		})
		metrics.Performs++
		if perform.ThruTarget == "" {
			continue
		}
		thruTargetID := paragraphIDs[perform.ThruTarget]
		if thruTargetID == "" {
			thruTargetID = sectionIDs[perform.ThruTarget]
		}
		if thruTargetID == "" || thruTargetID == targetID {
			continue
		}
		g.AddRelationship(graph.Relationship{
			ID:         graph.GenerateID(string(graph.RelCalls), sourceID+"->perform-thru->"+thruTargetID+":L"+itoa(perform.Line)),
			SourceID:   sourceID,
			TargetID:   thruTargetID,
			Type:       graph.RelCalls,
			Confidence: 1,
			Reason:     "cobol-perform-thru",
		})
	}
}

func emitCalls(g *graph.Graph, program programInfo, moduleID string, moduleIDs map[string]string, metrics *Metrics) {
	for _, call := range program.Calls {
		targetID := moduleIDs[call.Target]
		if targetID == "" {
			continue
		}
		g.AddRelationship(graph.Relationship{
			ID:         graph.GenerateID(string(graph.RelCalls), moduleID+"->call->"+targetID+":L"+itoa(call.Line)),
			SourceID:   moduleID,
			TargetID:   targetID,
			Type:       graph.RelCalls,
			Confidence: 0.95,
			Reason:     "cobol-call",
		})
		metrics.Calls++
	}
}

func emitCopies(g *graph.Graph, program programInfo, copybookPaths map[string]string, metrics *Metrics) {
	fileNodeID := graph.GenerateID(string(scopeir.NodeFile), program.FilePath)
	for _, copyRef := range program.Copies {
		targetPath := copybookPaths[copyRef.Target]
		if targetPath == "" {
			continue
		}
		targetFileID := graph.GenerateID(string(scopeir.NodeFile), targetPath)
		if _, ok := g.GetNode(targetFileID); !ok {
			continue
		}
		g.AddRelationship(graph.Relationship{
			ID:         graph.GenerateID(string(graph.RelImports), fileNodeID+"->"+targetFileID+":"+copyRef.Target),
			SourceID:   fileNodeID,
			TargetID:   targetFileID,
			Type:       graph.RelImports,
			Confidence: 1,
			Reason:     "cobol-copy",
		})
		metrics.Copies++
	}
}

func processJCL(g *graph.Graph, filePath string, content string, moduleIDs map[string]string, metrics *Metrics) {
	fileNodeID := graph.GenerateID(string(scopeir.NodeFile), filePath)
	if _, ok := g.GetNode(fileNodeID); !ok {
		return
	}
	parsed := parseJCL(content)
	jobIDs := map[string]string{}
	stepIDs := map[string]string{}
	for _, job := range parsed.Jobs {
		jobID := graph.GenerateID(string(scopeir.NodeCodeElement), filePath+":job:"+job.Name)
		description := "jcl-job"
		if job.Class != "" {
			description += " class:" + job.Class
		}
		if job.MsgClass != "" {
			description += " msgclass:" + job.MsgClass
		}
		g.AddNode(graph.Node{
			ID:    jobID,
			Label: scopeir.NodeCodeElement,
			Properties: graph.NodeProperties{
				"name":        job.Name,
				"filePath":    filePath,
				"startLine":   job.Line,
				"endLine":     job.Line,
				"language":    "jcl",
				"description": description,
			},
		})
		g.AddRelationship(graph.Relationship{
			ID:         graph.GenerateID(string(graph.RelContains), fileNodeID+"->"+jobID),
			SourceID:   fileNodeID,
			TargetID:   jobID,
			Type:       graph.RelContains,
			Confidence: 1,
			Reason:     "jcl-job",
		})
		jobIDs[job.Name] = jobID
		metrics.JCLJobs++
	}
	for _, step := range parsed.Steps {
		targetProgram := firstNonEmpty(step.Program, step.Proc)
		stepID := graph.GenerateID(string(scopeir.NodeCodeElement), filePath+":step:"+step.JobName+":"+step.Name)
		description := "jcl-step"
		if step.Program != "" {
			description += " pgm:" + step.Program
		}
		if step.Proc != "" {
			description += " proc:" + step.Proc
		}
		g.AddNode(graph.Node{
			ID:    stepID,
			Label: scopeir.NodeCodeElement,
			Properties: graph.NodeProperties{
				"name":        step.Name,
				"filePath":    filePath,
				"startLine":   step.Line,
				"endLine":     step.Line,
				"language":    "jcl",
				"description": description,
			},
		})
		parentID := jobIDs[step.JobName]
		if parentID == "" {
			parentID = fileNodeID
		}
		g.AddRelationship(graph.Relationship{
			ID:         graph.GenerateID(string(graph.RelContains), parentID+"->"+stepID),
			SourceID:   parentID,
			TargetID:   stepID,
			Type:       graph.RelContains,
			Confidence: 1,
			Reason:     "jcl-step",
		})
		stepIDs[step.Name] = stepID
		metrics.JCLSteps++
		targetID := moduleIDs[strings.ToUpper(targetProgram)]
		if targetID == "" || step.Program == "" {
			continue
		}
		g.AddRelationship(graph.Relationship{
			ID:         graph.GenerateID(string(graph.RelCalls), stepID+"->"+targetID),
			SourceID:   stepID,
			TargetID:   targetID,
			Type:       graph.RelCalls,
			Confidence: 0.95,
			Reason:     "jcl-exec-pgm",
		})
		metrics.JCLProgramLinks++
	}
	for _, dd := range parsed.DDStatements {
		if dd.Dataset == "" {
			continue
		}
		sourceID := stepIDs[dd.StepName]
		if sourceID == "" {
			continue
		}
		datasetID := graph.GenerateID(string(scopeir.NodeCodeElement), filePath+":dataset:"+dd.Dataset)
		description := "jcl-dd dd:" + dd.DDName
		if dd.Disp != "" {
			description += " disp:" + dd.Disp
		}
		g.AddNode(graph.Node{
			ID:    datasetID,
			Label: scopeir.NodeCodeElement,
			Properties: graph.NodeProperties{
				"name":        dd.Dataset,
				"filePath":    filePath,
				"startLine":   dd.Line,
				"endLine":     dd.Line,
				"language":    "jcl",
				"description": description,
			},
		})
		g.AddRelationship(graph.Relationship{
			ID:         graph.GenerateID(string(graph.RelCalls), sourceID+"->dd->"+datasetID+":"+dd.DDName),
			SourceID:   sourceID,
			TargetID:   datasetID,
			Type:       graph.RelCalls,
			Confidence: 0.95,
			Reason:     "jcl-dd:" + dd.DDName,
		})
	}
}

func preprocessCobolSource(content string) string {
	if isFreeFormatCobol(content) {
		return content
	}
	lines := strings.Split(content, "\n")
	for index, line := range lines {
		if len(line) < 7 {
			continue
		}
		lines[index] = "      " + line[6:]
	}
	return strings.Join(lines, "\n")
}

func sectionForLine(sections []sectionFact, sectionIDs map[string]string, line int) string {
	current := ""
	for _, section := range sections {
		if section.Line > line {
			break
		}
		current = sectionIDs[section.Name]
	}
	return current
}

func paragraphForLine(paragraphs []paragraphFact, paragraphIDs map[string]string, line int) string {
	current := ""
	for _, paragraph := range paragraphs {
		if paragraph.Line > line {
			break
		}
		current = paragraphIDs[paragraph.Name]
	}
	return current
}

func mainframeKind(filePath string) fileKind {
	switch strings.ToLower(path.Ext(filePath)) {
	case ".cob", ".cbl", ".cobol":
		return kindProgram
	case ".cpy", ".copybook":
		return kindCopybook
	case ".jcl", ".job", ".proc":
		return kindJCL
	default:
		return ""
	}
}

func isCobolComment(line string) bool {
	trimmed := strings.TrimSpace(line)
	if strings.HasPrefix(trimmed, "*>") || strings.HasPrefix(trimmed, "*") {
		return true
	}
	return len(line) > 6 && (line[6] == '*' || line[6] == '/')
}

func isFreeFormatCobol(content string) bool {
	for _, line := range strings.Split(content, "\n") {
		upper := strings.ToUpper(strings.TrimSpace(line))
		if strings.HasPrefix(upper, ">>SOURCE FREE") || strings.HasPrefix(upper, ">>SOURCE FORMAT IS FREE") {
			return true
		}
	}
	return false
}

func isCobolHeaderArea(line string) bool {
	leading := 0
	for leading < len(line) && (line[leading] == ' ' || line[leading] == '\t') {
		leading++
	}
	return leading < 11
}

func isInlinePerformLoop(remainder string, thruTarget string) bool {
	if thruTarget != "" {
		return false
	}
	word, ok := firstWord(remainder)
	if !ok {
		return false
	}
	switch strings.ToUpper(word) {
	case "TIMES", "UNTIL", "VARYING":
		return true
	default:
		return false
	}
}

func firstWord(text string) (string, bool) {
	text = strings.TrimLeft(text, " \t\r\n")
	if text == "" {
		return "", false
	}
	end := 0
	for end < len(text) && ((text[end] >= 'A' && text[end] <= 'Z') || (text[end] >= 'a' && text[end] <= 'z') || text[end] == '-') {
		end++
	}
	if end == 0 {
		return "", false
	}
	return text[:end], true
}

func popProgramStack(stack []*programInfo, name string) []*programInfo {
	if len(stack) == 0 {
		return stack
	}
	if name == "" {
		return stack[:len(stack)-1]
	}
	upperName := strings.ToUpper(name)
	for index := len(stack) - 1; index >= 0; index-- {
		if stack[index].ProgramName == upperName {
			return stack[:index]
		}
	}
	return stack[:len(stack)-1]
}

func programEndLine(program programInfo) int {
	if program.EndLine > 0 {
		return program.EndLine
	}
	return len(program.Lines)
}

func isReservedParagraphName(name string) bool {
	switch name {
	case "IDENTIFICATION", "ENVIRONMENT", "DATA", "PROCEDURE", "CONFIGURATION", "INPUT-OUTPUT",
		"FILE", "WORKING-STORAGE", "LOCAL-STORAGE", "LINKAGE", "REPORT", "SCREEN", "END":
		return true
	default:
		return false
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return strings.ToUpper(value)
		}
	}
	return ""
}

func basenameNoExt(filePath string) string {
	base := path.Base(filePath)
	return strings.TrimSuffix(base, path.Ext(base))
}

func normalizePath(filePath string) string {
	return strings.ReplaceAll(filePath, "\\", "/")
}

func itoa(value int) string {
	const digits = "0123456789"
	if value == 0 {
		return "0"
	}
	buf := make([]byte, 0, 12)
	for value > 0 {
		buf = append(buf, digits[value%10])
		value /= 10
	}
	for left, right := 0, len(buf)-1; left < right; left, right = left+1, right-1 {
		buf[left], buf[right] = buf[right], buf[left]
	}
	return string(buf)
}
