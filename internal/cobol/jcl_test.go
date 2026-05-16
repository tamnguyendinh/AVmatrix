package cobol

import (
	"reflect"
	"testing"
)

func TestParseJCLJobStatements(t *testing.T) {
	result := parseJCL("//PAYJOB   JOB (ACCT),'PAYROLL',CLASS=A,MSGCLASS=X")
	if len(result.Jobs) != 1 {
		t.Fatalf("jobs = %d, want 1", len(result.Jobs))
	}
	want := jclJob{Name: "PAYJOB", Line: 1, Class: "A", MsgClass: "X"}
	if result.Jobs[0] != want {
		t.Fatalf("job = %#v, want %#v", result.Jobs[0], want)
	}
}

func TestParseJCLExecStatements(t *testing.T) {
	result := parseJCL(linesJCL(
		"//JOB1    JOB (ACCT)",
		"//STEPA   EXEC PGM=PROG1",
		"//JOB2    JOB (ACCT)",
		"//STEPB   EXEC MYPROC",
	))
	if len(result.Steps) != 2 {
		t.Fatalf("steps = %d, want 2", len(result.Steps))
	}
	if result.Steps[0].Name != "STEPA" || result.Steps[0].Program != "PROG1" || result.Steps[0].JobName != "JOB1" {
		t.Fatalf("first step = %#v", result.Steps[0])
	}
	if result.Steps[1].Name != "STEPB" || result.Steps[1].Proc != "MYPROC" || result.Steps[1].JobName != "JOB2" {
		t.Fatalf("second step = %#v", result.Steps[1])
	}
}

func TestParseJCLDDStatements(t *testing.T) {
	result := parseJCL(linesJCL(
		"//MYJOB   JOB (ACCT)",
		"//STEP1   EXEC PGM=PROG1",
		"//DD1     DD DSN=DS1,DISP=SHR",
		"//STEP2   EXEC PGM=PROG2",
		"//DD2     DD DSN=DS2,DISP=(NEW,CATLG,DELETE)",
	))
	if len(result.DDStatements) != 2 {
		t.Fatalf("ddStatements = %d, want 2", len(result.DDStatements))
	}
	if result.DDStatements[0].DDName != "DD1" || result.DDStatements[0].StepName != "STEP1" || result.DDStatements[0].Dataset != "DS1" || result.DDStatements[0].Disp != "SHR" {
		t.Fatalf("first dd = %#v", result.DDStatements[0])
	}
	if result.DDStatements[1].DDName != "DD2" || result.DDStatements[1].StepName != "STEP2" || result.DDStatements[1].Dataset != "DS2" || result.DDStatements[1].Disp != "NEW" {
		t.Fatalf("second dd = %#v", result.DDStatements[1])
	}
}

func TestParseJCLProcIncludeSetConditionalsAndJCLLib(t *testing.T) {
	result := parseJCL(linesJCL(
		"//MYPROC  PROC",
		"//STEP1   EXEC PGM=IEFBR14",
		"// PEND",
		"// JCLLIB ORDER=(SYS1.PROCLIB,USER.PROCLIB)",
		"// SET ENV=PROD",
		"// INCLUDE MEMBER=STDPARMS",
		"// IF STEP1.RC = 0 THEN",
		"// ELSE",
		"// ENDIF",
	))
	if len(result.Procs) != 1 || result.Procs[0].Name != "MYPROC" || !result.Procs[0].IsInStream {
		t.Fatalf("procs = %#v", result.Procs)
	}
	if len(result.Includes) != 1 || result.Includes[0].Member != "STDPARMS" || result.Includes[0].Line != 6 {
		t.Fatalf("includes = %#v", result.Includes)
	}
	if len(result.Sets) != 1 || result.Sets[0].Variable != "ENV" || result.Sets[0].Value != "PROD" || result.Sets[0].Line != 5 {
		t.Fatalf("sets = %#v", result.Sets)
	}
	if len(result.JCLLib) != 1 || !reflect.DeepEqual(result.JCLLib[0].Order, []string{"SYS1.PROCLIB", "USER.PROCLIB"}) {
		t.Fatalf("jcllib = %#v", result.JCLLib)
	}
	if got := conditionalTypes(result.Conditionals); !reflect.DeepEqual(got, []string{"IF", "ELSE", "ENDIF"}) {
		t.Fatalf("conditionals = %#v, want IF/ELSE/ENDIF", result.Conditionals)
	}
	if result.Conditionals[0].Condition != "STEP1.RC = 0" {
		t.Fatalf("if condition = %q", result.Conditionals[0].Condition)
	}
}

func TestParseJCLContinuationAndEdgeCases(t *testing.T) {
	base := "//DD1     DD DSN=MY.VERY.LONG.DATASET.NAME.THAT.KEEPS.GOING,"
	line1 := base + repeatSpaces(71-len(base)) + "X"
	result := parseJCL(linesJCL(
		"//* comment",
		"This is not JCL",
		"//MYJOB   JOB (ACCT)",
		"//STEP1   EXEC PGM=IEFBR14",
		line1,
		"//             DISP=SHR",
	))
	if len(result.Jobs) != 1 || result.Jobs[0].Line != 3 {
		t.Fatalf("jobs = %#v", result.Jobs)
	}
	if len(result.Steps) != 1 {
		t.Fatalf("steps = %#v", result.Steps)
	}
	if len(result.DDStatements) != 1 {
		t.Fatalf("ddStatements = %#v", result.DDStatements)
	}
	dd := result.DDStatements[0]
	if dd.DDName != "DD1" || dd.Dataset != "MY.VERY.LONG.DATASET.NAME.THAT.KEEPS.GOING" || dd.Disp != "SHR" {
		t.Fatalf("dd = %#v", dd)
	}
}

func TestParseJCLCompleteJob(t *testing.T) {
	result := parseJCL(linesJCL(
		"//* Complete payroll job",
		"//PAYJOB   JOB (ACCT123),'PAYROLL RUN',CLASS=A,MSGCLASS=X",
		"// JCLLIB ORDER=(PAY.PROCLIB,SYS1.PROCLIB)",
		"// SET ENV=PROD",
		"// INCLUDE MEMBER=STDPARMS",
		"//*",
		"// IF 1 = 1 THEN",
		"//STEP01   EXEC PGM=PAYEXT",
		"//INPUT    DD DSN=PAY.MASTER,DISP=SHR",
		"//OUTPUT   DD DSN=PAY.EXTRACT,DISP=(NEW,CATLG,DELETE)",
		"//SYSPRINT DD SYSOUT=*",
		"//*",
		"//STEP02   EXEC PAYCALC",
		"//INFILE   DD DSN=PAY.EXTRACT,DISP=SHR",
		"// ELSE",
		"//STEP03   EXEC PGM=IEFBR14",
		"// ENDIF",
	))
	if len(result.Jobs) != 1 || result.Jobs[0].Name != "PAYJOB" || result.Jobs[0].Class != "A" || result.Jobs[0].MsgClass != "X" {
		t.Fatalf("jobs = %#v", result.Jobs)
	}
	if len(result.Steps) != 3 {
		t.Fatalf("steps = %#v", result.Steps)
	}
	if result.Steps[0].Program != "PAYEXT" || result.Steps[1].Proc != "PAYCALC" || result.Steps[2].Program != "IEFBR14" {
		t.Fatalf("steps = %#v", result.Steps)
	}
	if len(result.DDStatements) != 4 {
		t.Fatalf("ddStatements = %#v", result.DDStatements)
	}
	if result.DDStatements[0].DDName != "INPUT" || result.DDStatements[0].Dataset != "PAY.MASTER" {
		t.Fatalf("first dd = %#v", result.DDStatements[0])
	}
	if result.DDStatements[2].DDName != "SYSPRINT" || result.DDStatements[2].Dataset != "" {
		t.Fatalf("sysprint dd = %#v", result.DDStatements[2])
	}
	if len(result.Conditionals) != 3 {
		t.Fatalf("conditionals = %#v", result.Conditionals)
	}
}

func conditionalTypes(conditionals []jclConditional) []string {
	out := make([]string, 0, len(conditionals))
	for _, conditional := range conditionals {
		out = append(out, conditional.Type)
	}
	return out
}

func linesJCL(lines ...string) string {
	return stringsJoin(lines, "\n")
}

func stringsJoin(values []string, sep string) string {
	if len(values) == 0 {
		return ""
	}
	out := values[0]
	for _, value := range values[1:] {
		out += sep + value
	}
	return out
}

func repeatSpaces(count int) string {
	out := ""
	for range count {
		out += " "
	}
	return out
}
