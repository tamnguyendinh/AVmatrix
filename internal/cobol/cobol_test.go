package cobol

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tamnguyendinh/avmatrix-go/internal/graph"
	"github.com/tamnguyendinh/avmatrix-go/internal/scanner"
	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
	"github.com/tamnguyendinh/avmatrix-go/internal/structure"
)

func TestApplyEmitsCobolProgramsCopyPerformCallAndJCLLinks(t *testing.T) {
	dir := t.TempDir()
	writeCobolTestFile(t, dir, "src/main.cbl", `
       IDENTIFICATION DIVISION.
       PROGRAM-ID. MAINPGM.
       PROCEDURE DIVISION.
       MAIN-SECTION SECTION.
       START.
           COPY CUSTREC.
           PERFORM WORK-PARA.
           CALL 'PAYPGM'.
           STOP RUN.
       WORK-PARA.
           DISPLAY 'DONE'.
`)
	writeCobolTestFile(t, dir, "src/pay.cbl", `
       IDENTIFICATION DIVISION.
       PROGRAM-ID. PAYPGM.
       PROCEDURE DIVISION.
       PAY-START.
           DISPLAY 'PAY'.
`)
	writeCobolTestFile(t, dir, "copy/CUSTREC.cpy", "       01 CUST-ID PIC X(10).\n")
	writeCobolTestFile(t, dir, "jobs/nightly.jcl", "//NIGHTLY JOB\n//STEP1 EXEC PGM=MAINPGM\n")

	files := []scanner.File{
		{Path: "src/main.cbl", Language: scanner.Cobol},
		{Path: "src/pay.cbl", Language: scanner.Cobol},
		{Path: "copy/CUSTREC.cpy", Language: scanner.Cobol},
		{Path: "jobs/nightly.jcl", Language: scanner.Cobol},
	}
	g := graph.New()
	structure.Apply(g, files)

	result, err := Apply(g, dir, files)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if result.Metrics.Programs != 2 || result.Metrics.Copybooks != 1 || result.Metrics.JCLJobs != 1 || result.Metrics.JCLSteps != 1 {
		t.Fatalf("unexpected metrics: %#v", result.Metrics)
	}
	if result.Metrics.Sections != 1 || result.Metrics.Paragraphs != 3 || result.Metrics.Performs != 1 || result.Metrics.Calls != 1 || result.Metrics.Copies != 1 || result.Metrics.JCLProgramLinks != 1 {
		t.Fatalf("missing COBOL/JCL metrics: %#v", result.Metrics)
	}

	mainModule := graph.GenerateID(string(scopeir.NodeModule), "src/main.cbl:MAINPGM")
	payModule := graph.GenerateID(string(scopeir.NodeModule), "src/pay.cbl:PAYPGM")
	section := graph.GenerateID(string(scopeir.NodeNamespace), "src/main.cbl:MAINPGM:MAIN-SECTION")
	start := graph.GenerateID(string(scopeir.NodeFunction), "src/main.cbl:MAINPGM:START")
	work := graph.GenerateID(string(scopeir.NodeFunction), "src/main.cbl:MAINPGM:WORK-PARA")
	copyFile := graph.GenerateID(string(scopeir.NodeFile), "copy/CUSTREC.cpy")
	jclStep := graph.GenerateID(string(scopeir.NodeCodeElement), "jobs/nightly.jcl:step:NIGHTLY:STEP1")

	requireCobolNode(t, g, mainModule, scopeir.NodeModule)
	requireCobolNode(t, g, section, scopeir.NodeNamespace)
	requireCobolNode(t, g, start, scopeir.NodeFunction)
	requireCobolRelationship(t, g, graph.RelContains, section, start, "cobol-paragraph")
	requireCobolRelationship(t, g, graph.RelCalls, start, work, "cobol-perform")
	requireCobolRelationship(t, g, graph.RelCalls, mainModule, payModule, "cobol-call")
	requireCobolRelationship(t, g, graph.RelImports, graph.GenerateID(string(scopeir.NodeFile), "src/main.cbl"), copyFile, "cobol-copy")
	requireCobolRelationship(t, g, graph.RelCalls, jclStep, mainModule, "jcl-exec-pgm")
}

func TestApplyReturnsZeroForNonMainframeFiles(t *testing.T) {
	dir := t.TempDir()
	writeCobolTestFile(t, dir, "src/main.ts", "export const value = 1\n")
	files := []scanner.File{{Path: "src/main.ts", Language: scanner.TypeScript}}
	g := graph.New()
	structure.Apply(g, files)

	result, err := Apply(g, dir, files)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if result.Metrics != (Metrics{}) {
		t.Fatalf("metrics = %#v, want zero", result.Metrics)
	}
}

func TestApplyEmitsJCLJobMetadataAndProcSteps(t *testing.T) {
	dir := t.TempDir()
	writeCobolTestFile(t, dir, "jobs/payroll.jcl", "//PAYJOB JOB (ACCT),'PAY',CLASS=A,MSGCLASS=X\n//STEP1 EXEC PAYPROC\n")
	files := []scanner.File{{Path: "jobs/payroll.jcl", Language: scanner.Cobol}}
	g := graph.New()
	structure.Apply(g, files)

	result, err := Apply(g, dir, files)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if result.Metrics.JCLJobs != 1 || result.Metrics.JCLSteps != 1 || result.Metrics.JCLProgramLinks != 0 {
		t.Fatalf("unexpected metrics: %#v", result.Metrics)
	}

	jobID := graph.GenerateID(string(scopeir.NodeCodeElement), "jobs/payroll.jcl:job:PAYJOB")
	stepID := graph.GenerateID(string(scopeir.NodeCodeElement), "jobs/payroll.jcl:step:PAYJOB:STEP1")
	job, ok := g.GetNode(jobID)
	if !ok {
		t.Fatalf("missing JCL job node")
	}
	if job.Properties["description"] != "jcl-job class:A msgclass:X" {
		t.Fatalf("job description = %q", job.Properties["description"])
	}
	step, ok := g.GetNode(stepID)
	if !ok {
		t.Fatalf("missing JCL step node")
	}
	if step.Properties["description"] != "jcl-step proc:PAYPROC" {
		t.Fatalf("step description = %q", step.Properties["description"])
	}
	requireCobolRelationship(t, g, graph.RelContains, jobID, stepID, "jcl-step")
}

func BenchmarkApplyCobolEnrichment(b *testing.B) {
	dir := b.TempDir()
	writeCobolTestFile(b, dir, "src/main.cbl", `
       IDENTIFICATION DIVISION.
       PROGRAM-ID. MAINPGM.
       PROCEDURE DIVISION.
       MAIN-SECTION SECTION.
       START.
           COPY CUSTREC.
           PERFORM WORK-PARA.
           CALL 'PAYPGM'.
           STOP RUN.
       WORK-PARA.
           DISPLAY 'DONE'.
`)
	writeCobolTestFile(b, dir, "src/pay.cbl", `
       IDENTIFICATION DIVISION.
       PROGRAM-ID. PAYPGM.
       PROCEDURE DIVISION.
       PAY-START.
           DISPLAY 'PAY'.
`)
	writeCobolTestFile(b, dir, "copy/CUSTREC.cpy", "       01 CUST-ID PIC X(10).\n")
	writeCobolTestFile(b, dir, "jobs/nightly.jcl", "//NIGHTLY JOB\n//STEP1 EXEC PGM=MAINPGM\n")

	files := []scanner.File{
		{Path: "src/main.cbl", Language: scanner.Cobol},
		{Path: "src/pay.cbl", Language: scanner.Cobol},
		{Path: "copy/CUSTREC.cpy", Language: scanner.Cobol},
		{Path: "jobs/nightly.jcl", Language: scanner.Cobol},
	}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		g := graph.New()
		structure.Apply(g, files)
		result, err := Apply(g, dir, files)
		if err != nil {
			b.Fatalf("Apply() error = %v", err)
		}
		if result.Metrics.Programs != 2 || result.Metrics.JCLProgramLinks != 1 {
			b.Fatalf("incomplete COBOL/JCL metrics: %#v", result.Metrics)
		}
	}
}

func writeCobolTestFile(t testing.TB, root string, rel string, contents string) {
	t.Helper()
	fullPath := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", rel, err)
	}
	if err := os.WriteFile(fullPath, []byte(contents), 0o644); err != nil {
		t.Fatalf("write %s: %v", rel, err)
	}
}

func requireCobolNode(t *testing.T, g *graph.Graph, id string, label scopeir.NodeLabel) {
	t.Helper()
	node, ok := g.GetNode(id)
	if !ok {
		t.Fatalf("missing node %s", id)
	}
	if node.Label != label {
		t.Fatalf("node %s label = %s, want %s", id, node.Label, label)
	}
}

func requireCobolRelationship(t *testing.T, g *graph.Graph, relType graph.RelationshipType, sourceID string, targetID string, reason string) {
	t.Helper()
	for _, rel := range g.Relationships {
		if rel.Type == relType && rel.SourceID == sourceID && rel.TargetID == targetID && rel.Reason == reason {
			return
		}
	}
	t.Fatalf("missing %s %s -> %s reason %s", relType, sourceID, targetID, reason)
}
