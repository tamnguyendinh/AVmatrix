package resolution

import (
	"reflect"
	"testing"
	"time"

	"github.com/tamnguyendinh/avmatrix-go/internal/scanner"
)

func TestLegacyShadowDiffAgreementAndEvidenceDelta(t *testing.T) {
	callsite := ShadowCallsite{FilePath: "src/app.ts", Line: 42, Col: 8, CalledName: "save"}

	tests := []struct {
		name      string
		baseline  []ShadowResolution
		next      []ShadowResolution
		agreement ShadowAgreement
		delta     []string
	}{
		{name: "both empty", agreement: ShadowBothEmpty},
		{name: "both agree", baseline: []ShadowResolution{shadowResolution("def:User.save", "local", "owner-match")}, next: []ShadowResolution{shadowResolution("def:User.save", "local", "kind-match")}, agreement: ShadowBothAgree},
		{name: "only new", next: []ShadowResolution{shadowResolution("def:User.save", "local", "owner-match")}, agreement: ShadowOnlyNew, delta: []string{"local", "owner-match"}},
		{name: "only baseline", baseline: []ShadowResolution{shadowResolution("def:User.save", "global-name")}, agreement: ShadowOnlyBaseline, delta: []string{"global-name"}},
		{name: "both disagree disjoint", baseline: []ShadowResolution{shadowResolution("def:A", "global-name")}, next: []ShadowResolution{shadowResolution("def:B", "local", "owner-match")}, agreement: ShadowBothDisagree, delta: []string{"global-name", "local", "owner-match"}},
		{name: "both disagree overlap", baseline: []ShadowResolution{shadowResolution("def:A", "local", "scope-chain", "global-name")}, next: []ShadowResolution{shadowResolution("def:B", "local", "import", "owner-match")}, agreement: ShadowBothDisagree, delta: []string{"scope-chain", "global-name", "import", "owner-match"}},
		{name: "both disagree same kinds", baseline: []ShadowResolution{shadowResolution("def:A", "local", "owner-match")}, next: []ShadowResolution{shadowResolution("def:B", "owner-match", "local")}, agreement: ShadowBothDisagree},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := DiffShadowResolutions(callsite, test.baseline, test.next)
			if got.Agreement != test.agreement {
				t.Fatalf("agreement = %s, want %s", got.Agreement, test.agreement)
			}
			if got.Callsite != callsite {
				t.Fatalf("callsite = %#v, want %#v", got.Callsite, callsite)
			}
			if kinds := shadowEvidenceKinds(got.EvidenceDelta); !reflect.DeepEqual(kinds, test.delta) {
				t.Fatalf("evidence delta = %#v, want %#v", kinds, test.delta)
			}
		})
	}
}

func TestLegacyShadowDiffUsesOnlyTopResolution(t *testing.T) {
	callsite := ShadowCallsite{FilePath: "src/app.ts", Line: 42, Col: 8, CalledName: "save"}
	got := DiffShadowResolutions(
		callsite,
		[]ShadowResolution{shadowResolution("def:User.save", "local"), shadowResolution("def:other", "global-name")},
		[]ShadowResolution{shadowResolution("def:User.save", "local"), shadowResolution("def:yet-another", "global-name")},
	)
	if got.Agreement != ShadowBothAgree {
		t.Fatalf("agreement = %s, want both-agree", got.Agreement)
	}
}

func TestLegacyShadowAggregateDiffs(t *testing.T) {
	fixedNow := time.Date(2026, 4, 18, 12, 0, 0, 0, time.UTC)
	report := AggregateShadowDiffs(nil, fixedNow)
	if report.GeneratedAt != "2026-04-18T12:00:00.000Z" {
		t.Fatalf("generatedAt = %q", report.GeneratedAt)
	}
	if len(report.PerLanguage) != 0 || report.Overall.TotalCalls != 0 || report.Overall.Parity != 0 {
		t.Fatalf("empty report = %#v", report)
	}

	diffs := []LanguageShadowDiff{
		{Language: scanner.TypeScript, Diff: shadowDiff(ShadowBothAgree)},
		{Language: scanner.TypeScript, Diff: shadowDiff(ShadowBothAgree)},
		{Language: scanner.TypeScript, Diff: shadowDiff(ShadowOnlyBaseline, "global-name")},
		{Language: scanner.TypeScript, Diff: shadowDiff(ShadowOnlyNew, "local")},
		{Language: scanner.TypeScript, Diff: shadowDiff(ShadowBothDisagree, "import")},
		{Language: scanner.TypeScript, Diff: shadowDiff(ShadowBothEmpty)},
		{Language: scanner.TypeScript, Diff: shadowDiff(ShadowBothEmpty)},
		{Language: scanner.C, Diff: shadowDiff(ShadowBothAgree)},
		{Language: scanner.Python, Diff: shadowDiff(ShadowBothAgree)},
		{Language: scanner.Go, Diff: shadowDiff(ShadowBothDisagree, "import", "owner-match")},
		{Language: scanner.Go, Diff: shadowDiff(ShadowOnlyBaseline, "import", "global-name")},
		{Language: scanner.Go, Diff: shadowDiff(ShadowOnlyNew, "local")},
	}

	report = AggregateShadowDiffs(diffs, fixedNow)
	if got := shadowRowLanguages(report.PerLanguage); !reflect.DeepEqual(got, []scanner.Language{scanner.C, scanner.Go, scanner.Python, scanner.TypeScript}) {
		t.Fatalf("languages = %#v", got)
	}
	ts := requireShadowRow(t, report.PerLanguage, scanner.TypeScript)
	if ts.TotalCalls != 7 || ts.BothAgree != 2 || ts.OnlyBaseline != 1 || ts.OnlyNew != 1 || ts.BothDisagree != 1 || ts.BothEmpty != 2 {
		t.Fatalf("typescript row = %#v", ts)
	}
	if ts.Parity != 0.4 {
		t.Fatalf("typescript parity = %v, want 0.4", ts.Parity)
	}
	goRow := requireShadowRow(t, report.PerLanguage, scanner.Go)
	if !reflect.DeepEqual(goRow.EvidenceBreakdown, map[string]int{"global-name": 1, "import": 2, "local": 1, "owner-match": 1}) {
		t.Fatalf("go evidence breakdown = %#v", goRow.EvidenceBreakdown)
	}
	if report.Overall.TotalCalls != len(diffs) || report.Overall.BothAgree != 4 || report.Overall.BothEmpty != 2 {
		t.Fatalf("overall = %#v", report.Overall)
	}
}

func shadowResolution(nodeID string, kinds ...string) ShadowResolution {
	evidence := make([]ShadowEvidence, 0, len(kinds))
	for _, kind := range kinds {
		evidence = append(evidence, ShadowEvidence{Kind: kind, Weight: 0.3})
	}
	return ShadowResolution{
		Def:        ShadowDefinition{NodeID: nodeID, FilePath: "src/models.ts", Type: "Method"},
		Confidence: min(1, float64(len(kinds))*0.3),
		Evidence:   evidence,
	}
}

func shadowDiff(agreement ShadowAgreement, kinds ...string) ShadowDiff {
	evidence := make([]ShadowEvidence, 0, len(kinds))
	for _, kind := range kinds {
		evidence = append(evidence, ShadowEvidence{Kind: kind, Weight: 0.3})
	}
	return ShadowDiff{
		Callsite:      ShadowCallsite{FilePath: "src/x.ts", Line: 1, Col: 0, CalledName: "foo"},
		Agreement:     agreement,
		EvidenceDelta: evidence,
	}
}

func shadowEvidenceKinds(values []ShadowEvidence) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, value.Kind)
	}
	return out
}

func shadowRowLanguages(rows []LanguageParityRow) []scanner.Language {
	out := make([]scanner.Language, 0, len(rows))
	for _, row := range rows {
		out = append(out, row.Language)
	}
	return out
}

func requireShadowRow(t *testing.T, rows []LanguageParityRow, language scanner.Language) LanguageParityRow {
	t.Helper()
	for _, row := range rows {
		if row.Language == language {
			return row
		}
	}
	t.Fatalf("missing row for %s in %#v", language, rows)
	return LanguageParityRow{}
}
