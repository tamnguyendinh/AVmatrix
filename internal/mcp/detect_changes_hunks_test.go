package mcp

import (
	"reflect"
	"testing"
)

func TestParseDetectDiffHunksMatchesLegacyNewSideRanges(t *testing.T) {
	diff := joinDetectDiffLines(
		"diff --git a/src/foo.ts b/src/foo.ts",
		"--- a/src/foo.ts",
		"+++ b/src/foo.ts",
		"@@ -10,0 +11,3 @@ some context",
		"+line1",
		"+line2",
		"+line3",
		"diff --git a/src/bar.ts b/src/bar.ts",
		"--- a/src/bar.ts",
		"+++ b/src/bar.ts",
		"@@ -5,2 +5,4 @@ context",
		" unchanged",
		"+added",
		"@@ -20,0 +22,1 @@ more context",
		"+another line",
	)

	files := parseDetectDiffHunks(diff)
	if len(files) != 2 {
		t.Fatalf("len(files) = %d, want 2: %#v", len(files), files)
	}
	if files[0].FilePath != "src/foo.ts" || !reflect.DeepEqual(files[0].Hunks, []detectHunk{{StartLine: 11, EndLine: 13}}) {
		t.Fatalf("foo diff = %#v", files[0])
	}
	if files[1].FilePath != "src/bar.ts" || !reflect.DeepEqual(files[1].Hunks, []detectHunk{{StartLine: 5, EndLine: 8}, {StartLine: 22, EndLine: 22}}) {
		t.Fatalf("bar diff = %#v", files[1])
	}
}

func TestParseDetectDiffHunksHandlesSingleLinePureDeletionAndInterleavedFiles(t *testing.T) {
	single := parseDetectDiffHunks(joinDetectDiffLines(
		"+++ b/src/single.ts",
		"@@ -5,0 +6 @@ context",
		"+one line",
	))
	if len(single) != 1 || !reflect.DeepEqual(single[0].Hunks, []detectHunk{{StartLine: 6, EndLine: 6}}) {
		t.Fatalf("single hunk = %#v", single)
	}

	deletion := parseDetectDiffHunks(joinDetectDiffLines(
		"+++ b/src/del.ts",
		"@@ -10,3 +10,0 @@ context",
	))
	if len(deletion) != 0 {
		t.Fatalf("pure deletion hunks = %#v, want none", deletion)
	}

	deletedFile := parseDetectDiffHunks(joinDetectDiffLines(
		"--- a/src/deleted.ts",
		"+++ /dev/null",
		"@@ -10,3 +0,0 @@ context",
	))
	if len(deletedFile) != 1 || deletedFile[0].FilePath != "src/deleted.ts" || !deletedFile[0].Deleted || !reflect.DeepEqual(deletedFile[0].Hunks, []detectHunk{{StartLine: 10, EndLine: 12}}) {
		t.Fatalf("deleted file diff = %#v", deletedFile)
	}

	interleaved := parseDetectDiffHunks(joinDetectDiffLines(
		"diff --git a/src/alpha.ts b/src/alpha.ts",
		"index abc..def 100644",
		"--- a/src/alpha.ts",
		"+++ b/src/alpha.ts",
		"@@ -100,0 +101,2 @@ export function alpha() {",
		"+  const x = 1;",
		"+  return x;",
		"diff --git a/src/beta.ts b/src/beta.ts",
		"index 111..222 100644",
		"--- a/src/beta.ts",
		"+++ b/src/beta.ts",
		"@@ -50,0 +51,1 @@ export class Beta {",
		"+  private val = 0;",
		"@@ -80,0 +82,3 @@ export class Beta {",
		"+  doStuff() {",
		"+    return this.val;",
		"+  }",
	))
	if len(interleaved) != 2 {
		t.Fatalf("len(interleaved) = %d, want 2: %#v", len(interleaved), interleaved)
	}
	if interleaved[0].FilePath != "src/alpha.ts" || !reflect.DeepEqual(interleaved[0].Hunks, []detectHunk{{StartLine: 101, EndLine: 102}}) {
		t.Fatalf("alpha diff = %#v", interleaved[0])
	}
	if interleaved[1].FilePath != "src/beta.ts" || !reflect.DeepEqual(interleaved[1].Hunks, []detectHunk{{StartLine: 51, EndLine: 51}, {StartLine: 82, EndLine: 84}}) {
		t.Fatalf("beta diff = %#v", interleaved[1])
	}
}

func joinDetectDiffLines(lines ...string) string {
	out := ""
	for index, line := range lines {
		if index > 0 {
			out += "\n"
		}
		out += line
	}
	return out
}
