package cobol

import (
	"regexp"
	"strings"
)

var (
	jclDDPattern      = regexp.MustCompile(`(?i)^//([A-Z0-9$#@-]+)\s+DD\b(.*)`)
	jclProcPattern    = regexp.MustCompile(`(?i)^//([A-Z0-9$#@-]+)\s+PROC\b`)
	jclIncludePattern = regexp.MustCompile(`(?i)^//\s+INCLUDE\s+MEMBER=([A-Z0-9$#@-]+)`)
	jclSetPattern     = regexp.MustCompile(`(?i)^//\s+SET\s+([A-Z0-9$#@-]+)\s*=\s*([^,\s]+)`)
	jclJCLLibPattern  = regexp.MustCompile(`(?i)^//\s+JCLLIB\s+ORDER=\(([^)]*)\)`)
	jclIfPattern      = regexp.MustCompile(`(?i)^//\s+IF\s+(.+?)\s+THEN\s*$`)
)

type jclParseResult struct {
	Jobs         []jclJob
	Steps        []jclStep
	DDStatements []jclDD
	Procs        []jclProc
	Includes     []jclInclude
	Sets         []jclSet
	JCLLib       []jclLib
	Conditionals []jclConditional
}

type jclJob struct {
	Name     string
	Line     int
	Class    string
	MsgClass string
}

type jclStep struct {
	Name    string
	Line    int
	JobName string
	Program string
	Proc    string
}

type jclDD struct {
	DDName   string
	Line     int
	StepName string
	Dataset  string
	Disp     string
}

type jclProc struct {
	Name       string
	Line       int
	IsInStream bool
}

type jclInclude struct {
	Member string
	Line   int
}

type jclSet struct {
	Variable string
	Value    string
	Line     int
}

type jclLib struct {
	Order []string
	Line  int
}

type jclConditional struct {
	Type      string
	Condition string
	Line      int
}

type jclLogicalLine struct {
	Text string
	Line int
}

func parseJCL(content string) jclParseResult {
	result := jclParseResult{}
	currentJob := ""
	currentStep := ""
	for _, logical := range joinJCLContinuationLines(content) {
		line := strings.TrimSpace(logical.Text)
		if line == "" || !strings.HasPrefix(line, "//") || strings.HasPrefix(line, "//*") {
			continue
		}
		if match := jclJobPattern.FindStringSubmatch(line); match != nil {
			job := jclJob{Name: strings.ToUpper(match[1]), Line: logical.Line}
			tail := ""
			if len(match) > 2 {
				tail = match[2]
			}
			job.Class = jclParam(tail, "CLASS")
			job.MsgClass = jclParam(tail, "MSGCLASS")
			result.Jobs = append(result.Jobs, job)
			currentJob = job.Name
			currentStep = ""
			continue
		}
		if match := jclExecPattern.FindStringSubmatch(line); match != nil {
			step := jclStep{Name: strings.ToUpper(match[1]), Line: logical.Line, JobName: currentJob}
			if len(match) > 2 {
				step.Program = strings.ToUpper(match[2])
			}
			if len(match) > 3 {
				step.Proc = strings.ToUpper(firstNonEmpty(match[3], match[4]))
			}
			result.Steps = append(result.Steps, step)
			currentStep = step.Name
			continue
		}
		if match := jclDDPattern.FindStringSubmatch(line); match != nil {
			tail := ""
			if len(match) > 2 {
				tail = match[2]
			}
			result.DDStatements = append(result.DDStatements, jclDD{
				DDName:   strings.ToUpper(match[1]),
				Line:     logical.Line,
				StepName: currentStep,
				Dataset:  jclParam(tail, "DSN"),
				Disp:     firstJCLListValue(jclParam(tail, "DISP")),
			})
			continue
		}
		if match := jclProcPattern.FindStringSubmatch(line); match != nil {
			result.Procs = append(result.Procs, jclProc{Name: strings.ToUpper(match[1]), Line: logical.Line, IsInStream: true})
			continue
		}
		if match := jclIncludePattern.FindStringSubmatch(line); match != nil {
			result.Includes = append(result.Includes, jclInclude{Member: strings.ToUpper(match[1]), Line: logical.Line})
			continue
		}
		if match := jclSetPattern.FindStringSubmatch(line); match != nil {
			result.Sets = append(result.Sets, jclSet{Variable: strings.ToUpper(match[1]), Value: trimJCLValue(match[2]), Line: logical.Line})
			continue
		}
		if match := jclJCLLibPattern.FindStringSubmatch(line); match != nil {
			libs := strings.Split(match[1], ",")
			order := make([]string, 0, len(libs))
			for _, lib := range libs {
				value := trimJCLValue(lib)
				if value != "" {
					order = append(order, value)
				}
			}
			result.JCLLib = append(result.JCLLib, jclLib{Order: order, Line: logical.Line})
			continue
		}
		if match := jclIfPattern.FindStringSubmatch(line); match != nil {
			result.Conditionals = append(result.Conditionals, jclConditional{Type: "IF", Condition: strings.TrimSpace(match[1]), Line: logical.Line})
			continue
		}
		switch strings.ToUpper(strings.TrimSpace(strings.TrimPrefix(line, "//"))) {
		case "ELSE":
			result.Conditionals = append(result.Conditionals, jclConditional{Type: "ELSE", Line: logical.Line})
		case "ENDIF":
			result.Conditionals = append(result.Conditionals, jclConditional{Type: "ENDIF", Line: logical.Line})
		}
	}
	return result
}

func joinJCLContinuationLines(content string) []jclLogicalLine {
	lines := strings.Split(content, "\n")
	out := make([]jclLogicalLine, 0, len(lines))
	for index := 0; index < len(lines); index++ {
		line := strings.TrimRight(lines[index], "\r")
		if len(line) >= 72 && line[71] != ' ' && index+1 < len(lines) && strings.HasPrefix(strings.TrimLeft(lines[index+1], " "), "//") {
			next := strings.TrimRight(lines[index+1], "\r")
			continued := strings.TrimRight(line[:71], " ")
			continued += strings.TrimSpace(strings.TrimPrefix(strings.TrimLeft(next, " "), "//"))
			out = append(out, jclLogicalLine{Text: continued, Line: index + 1})
			index++
			continue
		}
		out = append(out, jclLogicalLine{Text: line, Line: index + 1})
	}
	return out
}

func jclParam(text string, name string) string {
	re := regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(name) + `=('[^']*'|"[^"]*"|\([^)]*\)|[^,\s]+)`)
	match := re.FindStringSubmatch(text)
	if match == nil {
		return ""
	}
	return trimJCLValue(match[1])
}

func trimJCLValue(value string) string {
	value = strings.TrimSpace(value)
	value = strings.Trim(value, `"'`)
	return value
}

func firstJCLListValue(value string) string {
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(value, "(")
	value = strings.TrimSuffix(value, ")")
	if comma := strings.Index(value, ","); comma >= 0 {
		value = value[:comma]
	}
	return trimJCLValue(value)
}
