package resolution

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/tamnguyendinh/avmatrix-go/internal/scanner"
)

type ShadowAgreement string

const (
	ShadowBothAgree    ShadowAgreement = "both-agree"
	ShadowOnlyBaseline ShadowAgreement = "only-baseline"
	ShadowOnlyNew      ShadowAgreement = "only-new"
	ShadowBothDisagree ShadowAgreement = "both-disagree"
	ShadowBothEmpty    ShadowAgreement = "both-empty"
)

type ShadowCallsite struct {
	FilePath   string
	Line       int
	Col        int
	CalledName string
}

type ShadowDefinition struct {
	NodeID   string
	FilePath string
	Type     string
}

type ShadowEvidence struct {
	Kind   string
	Weight float64
}

type ShadowResolution struct {
	Def        ShadowDefinition
	Confidence float64
	Evidence   []ShadowEvidence
}

type ShadowDiff struct {
	Callsite      ShadowCallsite
	Baseline      *ShadowResolution
	NewResult     *ShadowResolution
	Agreement     ShadowAgreement
	EvidenceDelta []ShadowEvidence
}

type LanguageParityRow struct {
	Language          scanner.Language
	TotalCalls        int
	BothAgree         int
	OnlyBaseline      int
	OnlyNew           int
	BothDisagree      int
	BothEmpty         int
	Parity            float64
	EvidenceBreakdown map[string]int
}

type ShadowParityReport struct {
	GeneratedAt string
	PerLanguage []LanguageParityRow
	Overall     LanguageParityOverall
}

type LanguageParityOverall struct {
	TotalCalls   int
	BothAgree    int
	OnlyBaseline int
	OnlyNew      int
	BothDisagree int
	BothEmpty    int
	Parity       float64
}

type LanguageShadowDiff struct {
	Language scanner.Language
	Diff     ShadowDiff
}

type PrimarySide string

const (
	PrimaryBaseline PrimarySide = "baseline"
	PrimaryRegistry PrimarySide = "registry"
)

type ShadowHarnessRecord struct {
	Language  scanner.Language
	Callsite  ShadowCallsite
	Baseline  []ShadowResolution
	NewResult []ShadowResolution
	Primary   PrimarySide
}

type ShadowHarness struct {
	enabled           bool
	records           []LanguageShadowDiff
	primaryByLanguage map[scanner.Language]PrimarySide
}

type PersistedShadowReport struct {
	SchemaVersion     int                              `json:"schemaVersion"`
	RunID             string                           `json:"runId"`
	GeneratedAt       string                           `json:"generatedAt"`
	PrimaryByLanguage map[scanner.Language]PrimarySide `json:"primaryByLanguage"`
	Report            ShadowParityReport               `json:"report"`
}

func NewShadowHarness() *ShadowHarness {
	return &ShadowHarness{
		enabled:           shadowModeEnabled(),
		primaryByLanguage: make(map[scanner.Language]PrimarySide),
	}
}

func (h *ShadowHarness) Enabled() bool {
	return h != nil && h.enabled
}

func (h *ShadowHarness) Record(input ShadowHarnessRecord) {
	if h == nil || !h.enabled {
		return
	}
	diff := DiffShadowResolutions(input.Callsite, input.Baseline, input.NewResult)
	h.records = append(h.records, LanguageShadowDiff{Language: input.Language, Diff: diff})
	h.primaryByLanguage[input.Language] = input.Primary
}

func (h *ShadowHarness) Size() int {
	if h == nil {
		return 0
	}
	return len(h.records)
}

func (h *ShadowHarness) Clear() {
	if h == nil {
		return
	}
	h.records = nil
	h.primaryByLanguage = make(map[scanner.Language]PrimarySide)
}

func (h *ShadowHarness) Snapshot(now time.Time) ShadowParityReport {
	if h == nil {
		return AggregateShadowDiffs(nil, now)
	}
	return AggregateShadowDiffs(h.records, now)
}

func (h *ShadowHarness) Persist(outputDir string, now time.Time) (string, error) {
	if h == nil {
		h = NewShadowHarness()
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	report := h.Snapshot(now)
	runID := shadowRunID(now)
	payload := PersistedShadowReport{
		SchemaVersion:     1,
		RunID:             runID,
		GeneratedAt:       report.GeneratedAt,
		PrimaryByLanguage: h.primaryByLanguageCopy(),
		Report:            report,
	}
	raw, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return "", err
	}
	raw = append(raw, '\n')
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", err
	}
	perRunPath := filepath.Join(outputDir, runID+".json")
	if err := os.WriteFile(perRunPath, raw, 0o644); err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(outputDir, "latest.json"), raw, 0o644); err != nil {
		return "", err
	}
	return perRunPath, nil
}

func DiffShadowResolutions(callsite ShadowCallsite, baseline []ShadowResolution, newResult []ShadowResolution) ShadowDiff {
	var baselineTop *ShadowResolution
	var newTop *ShadowResolution
	if len(baseline) > 0 {
		baselineTop = &baseline[0]
	}
	if len(newResult) > 0 {
		newTop = &newResult[0]
	}

	agreement := ShadowBothEmpty
	switch {
	case baselineTop == nil && newTop == nil:
		agreement = ShadowBothEmpty
	case baselineTop == nil:
		agreement = ShadowOnlyNew
	case newTop == nil:
		agreement = ShadowOnlyBaseline
	case baselineTop.Def.NodeID == newTop.Def.NodeID:
		agreement = ShadowBothAgree
	default:
		agreement = ShadowBothDisagree
	}

	return ShadowDiff{
		Callsite:      callsite,
		Baseline:      baselineTop,
		NewResult:     newTop,
		Agreement:     agreement,
		EvidenceDelta: shadowEvidenceDelta(baselineTop, newTop, agreement),
	}
}

func shadowModeEnabled() bool {
	switch strings.TrimSpace(strings.ToLower(os.Getenv("AVMATRIX_SHADOW_MODE"))) {
	case "1", "true", "yes":
		return true
	default:
		return false
	}
}

func (h *ShadowHarness) primaryByLanguageCopy() map[scanner.Language]PrimarySide {
	out := make(map[scanner.Language]PrimarySide, len(h.primaryByLanguage))
	for language, primary := range h.primaryByLanguage {
		out[language] = primary
	}
	return out
}

func shadowRunID(now time.Time) string {
	var suffix [4]byte
	if _, err := rand.Read(suffix[:]); err != nil {
		binary := uint32(now.UnixNano())
		suffix[0] = byte(binary >> 24)
		suffix[1] = byte(binary >> 16)
		suffix[2] = byte(binary >> 8)
		suffix[3] = byte(binary)
	}
	return now.UTC().Format("20060102-150405") + "-" + hex.EncodeToString(suffix[:])
}

func shadowEvidenceDelta(baseline *ShadowResolution, newResult *ShadowResolution, agreement ShadowAgreement) []ShadowEvidence {
	switch agreement {
	case ShadowBothAgree, ShadowBothEmpty:
		return nil
	case ShadowOnlyBaseline:
		return append([]ShadowEvidence(nil), baseline.Evidence...)
	case ShadowOnlyNew:
		return append([]ShadowEvidence(nil), newResult.Evidence...)
	}

	baselineKinds := make(map[string]struct{}, len(baseline.Evidence))
	newKinds := make(map[string]struct{}, len(newResult.Evidence))
	for _, evidence := range baseline.Evidence {
		baselineKinds[evidence.Kind] = struct{}{}
	}
	for _, evidence := range newResult.Evidence {
		newKinds[evidence.Kind] = struct{}{}
	}
	out := make([]ShadowEvidence, 0, len(baseline.Evidence)+len(newResult.Evidence))
	for _, evidence := range baseline.Evidence {
		if _, ok := newKinds[evidence.Kind]; !ok {
			out = append(out, evidence)
		}
	}
	for _, evidence := range newResult.Evidence {
		if _, ok := baselineKinds[evidence.Kind]; !ok {
			out = append(out, evidence)
		}
	}
	return out
}

func AggregateShadowDiffs(diffs []LanguageShadowDiff, now time.Time) ShadowParityReport {
	countsByLanguage := make(map[scanner.Language]*shadowMutableCounts)
	for _, item := range diffs {
		counts := countsByLanguage[item.Language]
		if counts == nil {
			counts = &shadowMutableCounts{EvidenceBreakdown: make(map[string]int)}
			countsByLanguage[item.Language] = counts
		}
		tallyShadowDiff(counts, item.Diff)
	}

	languages := make([]scanner.Language, 0, len(countsByLanguage))
	for language := range countsByLanguage {
		languages = append(languages, language)
	}
	sort.Slice(languages, func(i, j int) bool {
		return languages[i] < languages[j]
	})

	rows := make([]LanguageParityRow, 0, len(languages))
	for _, language := range languages {
		rows = append(rows, buildShadowLanguageRow(language, countsByLanguage[language]))
	}
	return ShadowParityReport{
		GeneratedAt: now.UTC().Format("2006-01-02T15:04:05.000Z"),
		PerLanguage: rows,
		Overall:     buildShadowOverall(rows),
	}
}

type shadowMutableCounts struct {
	TotalCalls        int
	BothAgree         int
	OnlyBaseline      int
	OnlyNew           int
	BothDisagree      int
	BothEmpty         int
	EvidenceBreakdown map[string]int
}

func tallyShadowDiff(counts *shadowMutableCounts, diff ShadowDiff) {
	counts.TotalCalls++
	switch diff.Agreement {
	case ShadowBothAgree:
		counts.BothAgree++
	case ShadowOnlyBaseline:
		counts.OnlyBaseline++
	case ShadowOnlyNew:
		counts.OnlyNew++
	case ShadowBothDisagree:
		counts.BothDisagree++
	case ShadowBothEmpty:
		counts.BothEmpty++
	}
	if diff.Agreement == ShadowBothAgree || diff.Agreement == ShadowBothEmpty {
		return
	}
	for _, evidence := range diff.EvidenceDelta {
		counts.EvidenceBreakdown[evidence.Kind]++
	}
}

func buildShadowLanguageRow(language scanner.Language, counts *shadowMutableCounts) LanguageParityRow {
	breakdown := make(map[string]int, len(counts.EvidenceBreakdown))
	for kind, count := range counts.EvidenceBreakdown {
		breakdown[kind] = count
	}
	resolved := counts.TotalCalls - counts.BothEmpty
	parity := 0.0
	if resolved > 0 {
		parity = float64(counts.BothAgree) / float64(resolved)
	}
	return LanguageParityRow{
		Language:          language,
		TotalCalls:        counts.TotalCalls,
		BothAgree:         counts.BothAgree,
		OnlyBaseline:      counts.OnlyBaseline,
		OnlyNew:           counts.OnlyNew,
		BothDisagree:      counts.BothDisagree,
		BothEmpty:         counts.BothEmpty,
		Parity:            parity,
		EvidenceBreakdown: breakdown,
	}
}

func buildShadowOverall(rows []LanguageParityRow) LanguageParityOverall {
	var overall LanguageParityOverall
	for _, row := range rows {
		overall.TotalCalls += row.TotalCalls
		overall.BothAgree += row.BothAgree
		overall.OnlyBaseline += row.OnlyBaseline
		overall.OnlyNew += row.OnlyNew
		overall.BothDisagree += row.BothDisagree
		overall.BothEmpty += row.BothEmpty
	}
	resolved := overall.TotalCalls - overall.BothEmpty
	if resolved > 0 {
		overall.Parity = float64(overall.BothAgree) / float64(resolved)
	}
	return overall
}
