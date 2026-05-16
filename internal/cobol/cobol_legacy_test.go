package cobol

import (
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/tamnguyendinh/avmatrix-go/internal/graph"
	"github.com/tamnguyendinh/avmatrix-go/internal/scanner"
	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
	"github.com/tamnguyendinh/avmatrix-go/internal/structure"
)

func TestPreprocessCobolSourceLegacyFixedColumns(t *testing.T) {
	input := cobolLines(
		"mzADD  IDENTIFICATION DIVISION.",
		"estero PROGRAM-ID. TEST1.",
		"000300 PROCEDURE DIVISION.",
		"SHORT",
		"      ",
	)
	output := preprocessCobolSource(input)
	lines := strings.Split(output, "\n")
	if lines[0][:6] != "      " || lines[0][6:] != " IDENTIFICATION DIVISION." {
		t.Fatalf("patch marker line = %q", lines[0])
	}
	if lines[1][:6] != "      " || lines[2] != "       PROCEDURE DIVISION." {
		t.Fatalf("sequence columns not stripped: %#v", lines[:3])
	}
	if lines[3] != "SHORT" || lines[4] != "      " {
		t.Fatalf("short lines changed: %#v", lines[3:])
	}
	if len(lines) != len(strings.Split(input, "\n")) {
		t.Fatalf("line count changed from %d to %d", len(strings.Split(input, "\n")), len(lines))
	}
}

func TestExtractProgramLegacyFixedFormatControlFlowSubset(t *testing.T) {
	program := extractProgram("test.cbl", cobolLines(
		"000100 IDENTIFICATION DIVISION.",
		"000200 PROGRAM-ID. TESTPROG.",
		"000300 PROCEDURE DIVISION.",
		"000400 INIT-SECTION SECTION 01.",
		"000500 MAIN-PARA.",
		"000600     PERFORM SUB-PARA.",
		"000700     PERFORM STEP-A THROUGH STEP-Z.",
		"000800     PERFORM WS-COUNT TIMES.",
		"000900     CALL \"SUBPROG\".",
		"001000     CALL WS-PROG-NAME.",
		"001100     COPY \"MY-COPY\".",
		"001200 SUB-PARA.",
		"001300     DISPLAY \"OK\".",
		"001400     NOT-A-PARA.",
	))

	if program.ProgramName != "TESTPROG" {
		t.Fatalf("ProgramName = %q", program.ProgramName)
	}
	if got := sectionNames(program.Sections); !reflect.DeepEqual(got, []string{"INIT-SECTION"}) {
		t.Fatalf("sections = %#v", got)
	}
	if got := paragraphNames(program.Paragraphs); !reflect.DeepEqual(got, []string{"MAIN-PARA", "SUB-PARA"}) {
		t.Fatalf("paragraphs = %#v", got)
	}
	if len(program.Performs) != 2 {
		t.Fatalf("performs = %#v", program.Performs)
	}
	if program.Performs[0].Target != "SUB-PARA" || program.Performs[1].Target != "STEP-A" || program.Performs[1].ThruTarget != "STEP-Z" {
		t.Fatalf("perform targets = %#v", program.Performs)
	}
	if got := callTargets(program.Calls); !reflect.DeepEqual(got, []string{"SUBPROG", "WS-PROG-NAME"}) {
		t.Fatalf("calls = %#v", got)
	}
	if got := copyTargets(program.Copies); !reflect.DeepEqual(got, []string{"MY-COPY"}) {
		t.Fatalf("copies = %#v", got)
	}
}

func TestExtractProgramsLegacyNestedProgramScopes(t *testing.T) {
	programs := extractPrograms("nested.cbl", cobolLines(
		"       IDENTIFICATION DIVISION.",
		"       PROGRAM-ID. OUTER-PROG.",
		"       PROCEDURE DIVISION.",
		"       OUTER-MAIN.",
		"           PERFORM OUTER-PROCESS",
		"           CALL \"INNER-PROG\"",
		"       OUTER-PROCESS.",
		"           DISPLAY 'OUTER'.",
		"       IDENTIFICATION DIVISION.",
		"       PROGRAM-ID. INNER-PROG.",
		"       PROCEDURE DIVISION.",
		"       INNER-MAIN.",
		"           PERFORM INNER-PROCESS",
		"       INNER-PROCESS.",
		"           DISPLAY 'INNER'.",
		"       END PROGRAM INNER-PROG.",
		"       END PROGRAM OUTER-PROG.",
	))
	if len(programs) != 2 {
		t.Fatalf("programs = %#v", programs)
	}
	outer := programs[0]
	inner := programs[1]
	if outer.ProgramName != "OUTER-PROG" || outer.ParentName != "" || outer.NestingDepth != 0 {
		t.Fatalf("outer = %#v", outer)
	}
	if inner.ProgramName != "INNER-PROG" || inner.ParentName != "OUTER-PROG" || inner.NestingDepth != 1 {
		t.Fatalf("inner = %#v", inner)
	}
	if got := paragraphNames(outer.Paragraphs); !reflect.DeepEqual(got, []string{"OUTER-MAIN", "OUTER-PROCESS"}) {
		t.Fatalf("outer paragraphs = %#v", got)
	}
	if got := paragraphNames(inner.Paragraphs); !reflect.DeepEqual(got, []string{"INNER-MAIN", "INNER-PROCESS"}) {
		t.Fatalf("inner paragraphs = %#v", got)
	}
	if got := callTargets(outer.Calls); !reflect.DeepEqual(got, []string{"INNER-PROG"}) {
		t.Fatalf("outer calls = %#v", got)
	}
	if len(inner.Performs) != 1 || inner.Performs[0].Target != "INNER-PROCESS" {
		t.Fatalf("inner performs = %#v", inner.Performs)
	}
}

func TestExtractProgramLegacyFreeFormatSubset(t *testing.T) {
	program := extractProgram("free.cbl", cobolLines(
		">>SOURCE FORMAT IS FREE",
		"IDENTIFICATION DIVISION.",
		"PROGRAM-ID. FREEPROG.",
		"PROCEDURE DIVISION.",
		"MAIN-PARA.",
		"    PERFORM PROCESS-DATA.",
		"PROCESS-DATA.",
		"    STOP RUN.",
	))
	if program.ProgramName != "FREEPROG" {
		t.Fatalf("ProgramName = %q", program.ProgramName)
	}
	if got := paragraphNames(program.Paragraphs); !reflect.DeepEqual(got, []string{"MAIN-PARA", "PROCESS-DATA"}) {
		t.Fatalf("paragraphs = %#v", got)
	}
	if len(program.Performs) != 1 || program.Performs[0].Target != "PROCESS-DATA" {
		t.Fatalf("performs = %#v", program.Performs)
	}
}

func TestExtractProgramLegacyMultiplePerformsCommentsAndSiblingProgram(t *testing.T) {
	programs := extractPrograms("multi.cbl", cobolLines(
		"       IDENTIFICATION DIVISION.",
		"       PROGRAM-ID. OUTER.",
		"       PROCEDURE DIVISION.",
		"       MAIN-PARA.",
		"      *    PERFORM COMMENTED-PARA.",
		"           IF WS-COUNT > 0 PERFORM FETCH-DATA ELSE PERFORM SEND-SCREEN",
		"           PERFORM WS-COUNT TIMES",
		"           CALL 'SUBPROG'",
		"       FETCH-DATA.",
		"           DISPLAY 'FETCH'.",
		"       SEND-SCREEN.",
		"           DISPLAY 'SEND'.",
		"       PROGRAM-ID. SIBLING.",
		"       PROCEDURE DIVISION.",
		"       SIB-MAIN.",
		"           PERFORM SIB-WORK.",
		"       SIB-WORK.",
		"           DISPLAY 'SIB'.",
		"       END PROGRAM SIBLING.",
		"       END PROGRAM OUTER.",
	))

	if len(programs) != 2 {
		t.Fatalf("programs = %#v", programs)
	}
	outer := programs[0]
	sibling := programs[1]
	if outer.ProgramName != "OUTER" || sibling.ProgramName != "SIBLING" || sibling.ParentName != "OUTER" {
		t.Fatalf("program nesting = %#v", programs)
	}
	if got := paragraphNames(outer.Paragraphs); !reflect.DeepEqual(got, []string{"MAIN-PARA", "FETCH-DATA", "SEND-SCREEN"}) {
		t.Fatalf("outer paragraphs = %#v", got)
	}
	if got := performTargets(outer.Performs); !reflect.DeepEqual(got, []string{"FETCH-DATA", "SEND-SCREEN"}) {
		t.Fatalf("outer performs = %#v", got)
	}
	if got := callTargets(outer.Calls); !reflect.DeepEqual(got, []string{"SUBPROG"}) {
		t.Fatalf("outer calls = %#v", got)
	}
	if got := performTargets(sibling.Performs); !reflect.DeepEqual(got, []string{"SIB-WORK"}) {
		t.Fatalf("sibling performs = %#v", got)
	}
}

func TestApplyLegacyCobolAppControlFlowAndJCLSubset(t *testing.T) {
	dir := t.TempDir()
	writeCobolTestFile(t, dir, "CUSTUPDT.cbl", cobolLines(
		"       IDENTIFICATION DIVISION.",
		"       PROGRAM-ID. CUSTUPDT.",
		"       DATA DIVISION.",
		"       WORKING-STORAGE SECTION.",
		"           COPY COPYLIB.",
		"       PROCEDURE DIVISION.",
		"       INIT-SECTION SECTION.",
		"       MAIN-PARAGRAPH.",
		"           PERFORM INIT-PARAGRAPH",
		"           PERFORM PROCESS-PARAGRAPH",
		"       INIT-PARAGRAPH.",
		"           DISPLAY 'INIT'.",
		"       PROCESSING-SECTION SECTION.",
		"       PROCESS-PARAGRAPH.",
		"           PERFORM READ-CUSTOMER THRU WRITE-CUSTOMER",
		"           CALL \"AUDITLOG\" USING CUST-ID",
		"           CALL WS-PROG-NAME.",
		"       READ-CUSTOMER.",
		"           DISPLAY 'READ'.",
		"       WRITE-CUSTOMER.",
		"           DISPLAY 'WRITE'.",
	))
	writeCobolTestFile(t, dir, "AUDITLOG.cbl", cobolLines(
		"       IDENTIFICATION DIVISION.",
		"       PROGRAM-ID. AUDITLOG.",
		"       PROCEDURE DIVISION.",
		"       MAIN-PARAGRAPH.",
		"           PERFORM WRITE-LOG",
		"       WRITE-LOG.",
		"           DISPLAY 'AUDIT'.",
	))
	writeCobolTestFile(t, dir, "RPTGEN.cbl", cobolLines(
		"       IDENTIFICATION DIVISION.",
		"       PROGRAM-ID. RPTGEN.",
		"       PROCEDURE DIVISION.",
		"       MAIN-PARAGRAPH.",
		"           PERFORM FETCH-DATA",
		"           PERFORM FORMAT-REPORT",
		"           CALL \"CUSTUPDT\"",
		"       FETCH-DATA.",
		"           DISPLAY 'FETCH'.",
		"       FORMAT-REPORT.",
		"           IF WS-COUNT > 0 PERFORM FETCH-DATA ELSE PERFORM SEND-SCREEN",
		"           PERFORM MAIN-PARAGRAPH THRU FORMAT-REPORT",
		"       SEND-SCREEN.",
		"           DISPLAY 'SEND'.",
	))
	writeCobolTestFile(t, dir, "NESTED.cbl", cobolLines(
		"       IDENTIFICATION DIVISION.",
		"       PROGRAM-ID. OUTER-PROG.",
		"       PROCEDURE DIVISION.",
		"       OUTER-MAIN.",
		"           PERFORM OUTER-PROCESS",
		"           CALL \"INNER-PROG\"",
		"       OUTER-PROCESS.",
		"           DISPLAY 'OUTER'.",
		"       IDENTIFICATION DIVISION.",
		"       PROGRAM-ID. INNER-PROG.",
		"       PROCEDURE DIVISION.",
		"       INNER-MAIN.",
		"           PERFORM INNER-PROCESS",
		"       INNER-PROCESS.",
		"           DISPLAY 'INNER'.",
		"       END PROGRAM INNER-PROG.",
		"       END PROGRAM OUTER-PROG.",
	))
	writeCobolTestFile(t, dir, "COPYLIB.cpy", "       01 COPY-REC PIC X(10).\n")
	writeCobolTestFile(t, dir, "RUNJOBS.jcl", cobolLines(
		"//CUSTJOB  JOB (ACCT),'CUSTOMER UPDATE',CLASS=A,MSGCLASS=X",
		"//STEP1    EXEC PGM=CUSTUPDT",
		"//CUSTFILE DD DSN=PROD.CUSTOMER.MASTER,DISP=SHR",
		"//STEP2    EXEC PGM=RPTGEN",
	))

	files := []scanner.File{
		{Path: "AUDITLOG.cbl", Language: scanner.Cobol},
		{Path: "COPYLIB.cpy", Language: scanner.Cobol},
		{Path: "CUSTUPDT.cbl", Language: scanner.Cobol},
		{Path: "NESTED.cbl", Language: scanner.Cobol},
		{Path: "RPTGEN.cbl", Language: scanner.Cobol},
		{Path: "RUNJOBS.jcl", Language: scanner.Cobol},
	}
	g := graph.New()
	structure.Apply(g, files)
	result, err := Apply(g, dir, files)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if result.Metrics.Programs != 5 || result.Metrics.Sections != 2 || result.Metrics.JCLProgramLinks != 2 {
		t.Fatalf("unexpected metrics: %#v", result.Metrics)
	}
	if got := cobolNodeNames(g, scopeir.NodeModule); !reflect.DeepEqual(got, []string{"AUDITLOG", "CUSTUPDT", "INNER-PROG", "OUTER-PROG", "RPTGEN"}) {
		t.Fatalf("module names = %#v", got)
	}

	custupdt := graph.GenerateID(string(scopeir.NodeModule), "CUSTUPDT.cbl:CUSTUPDT")
	auditlog := graph.GenerateID(string(scopeir.NodeModule), "AUDITLOG.cbl:AUDITLOG")
	rptgen := graph.GenerateID(string(scopeir.NodeModule), "RPTGEN.cbl:RPTGEN")
	outer := graph.GenerateID(string(scopeir.NodeModule), "NESTED.cbl:OUTER-PROG")
	inner := graph.GenerateID(string(scopeir.NodeModule), "NESTED.cbl:INNER-PROG")
	main := graph.GenerateID(string(scopeir.NodeFunction), "CUSTUPDT.cbl:CUSTUPDT:MAIN-PARAGRAPH")
	process := graph.GenerateID(string(scopeir.NodeFunction), "CUSTUPDT.cbl:CUSTUPDT:PROCESS-PARAGRAPH")
	initPara := graph.GenerateID(string(scopeir.NodeFunction), "CUSTUPDT.cbl:CUSTUPDT:INIT-PARAGRAPH")
	readCustomer := graph.GenerateID(string(scopeir.NodeFunction), "CUSTUPDT.cbl:CUSTUPDT:READ-CUSTOMER")
	writeCustomer := graph.GenerateID(string(scopeir.NodeFunction), "CUSTUPDT.cbl:CUSTUPDT:WRITE-CUSTOMER")
	step1 := graph.GenerateID(string(scopeir.NodeCodeElement), "RUNJOBS.jcl:step:CUSTJOB:STEP1")
	dataset := graph.GenerateID(string(scopeir.NodeCodeElement), "RUNJOBS.jcl:dataset:PROD.CUSTOMER.MASTER")

	requireCobolRelationship(t, g, graph.RelContains, outer, inner, "cobol-nested-program")
	requireCobolRelationship(t, g, graph.RelCalls, main, initPara, "cobol-perform")
	requireCobolRelationship(t, g, graph.RelCalls, process, readCustomer, "cobol-perform")
	requireCobolRelationship(t, g, graph.RelCalls, process, writeCustomer, "cobol-perform-thru")
	requireCobolRelationship(t, g, graph.RelCalls, custupdt, auditlog, "cobol-call")
	requireCobolRelationship(t, g, graph.RelCalls, rptgen, custupdt, "cobol-call")
	requireCobolRelationship(t, g, graph.RelCalls, outer, inner, "cobol-call")
	requireCobolRelationship(t, g, graph.RelCalls, step1, custupdt, "jcl-exec-pgm")
	requireCobolRelationship(t, g, graph.RelCalls, step1, dataset, "jcl-dd:CUSTFILE")
	requireCobolRelationship(t, g, graph.RelImports, graph.GenerateID(string(scopeir.NodeFile), "CUSTUPDT.cbl"), graph.GenerateID(string(scopeir.NodeFile), "COPYLIB.cpy"), "cobol-copy")
}

func cobolLines(lines ...string) string {
	return stringsJoin(lines, "\n")
}

func sectionNames(sections []sectionFact) []string {
	out := make([]string, 0, len(sections))
	for _, section := range sections {
		out = append(out, section.Name)
	}
	return out
}

func paragraphNames(paragraphs []paragraphFact) []string {
	out := make([]string, 0, len(paragraphs))
	for _, paragraph := range paragraphs {
		out = append(out, paragraph.Name)
	}
	return out
}

func callTargets(calls []callFact) []string {
	out := make([]string, 0, len(calls))
	for _, call := range calls {
		out = append(out, call.Target)
	}
	return out
}

func performTargets(performs []performFact) []string {
	out := make([]string, 0, len(performs))
	for _, perform := range performs {
		out = append(out, perform.Target)
	}
	return out
}

func copyTargets(copies []copyFact) []string {
	out := make([]string, 0, len(copies))
	for _, copyRef := range copies {
		out = append(out, copyRef.Target)
	}
	return out
}

func cobolNodeNames(g *graph.Graph, label scopeir.NodeLabel) []string {
	out := []string{}
	for _, node := range g.Nodes {
		if node.Label != label {
			continue
		}
		if name, ok := node.Properties["name"].(string); ok {
			out = append(out, name)
		}
	}
	sort.Strings(out)
	return out
}
