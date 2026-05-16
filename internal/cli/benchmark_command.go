package cli

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

type benchmarkMetrics struct {
	Label        string
	TotalWallMs  *float64
	PhaseMs      map[string]float64
	Counts       map[string]float64
	Relationship map[string]float64
	NodeLabels   map[string]float64
}

type benchmarkComparison struct {
	BeforeLabel   string                  `json:"beforeLabel,omitempty"`
	AfterLabel    string                  `json:"afterLabel,omitempty"`
	TotalWallMs   *numericDelta           `json:"totalWallMs,omitempty"`
	PhaseMs       map[string]numericDelta `json:"phaseMs,omitempty"`
	Counts        map[string]numericDelta `json:"counts,omitempty"`
	Relationships map[string]numericDelta `json:"relationshipCountsByType,omitempty"`
	NodeLabels    map[string]numericDelta `json:"nodeCountsByLabel,omitempty"`
}

type numericDelta struct {
	Before        *float64 `json:"before,omitempty"`
	After         *float64 `json:"after,omitempty"`
	Delta         *float64 `json:"delta,omitempty"`
	PercentChange *float64 `json:"percentChange,omitempty"`
}

func newBenchmarkCompareCommand() *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "benchmark-compare <before> <after>",
		Short: "Compare two analyze benchmark JSON artifacts",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			before, err := readBenchmarkMetrics(args[0])
			if err != nil {
				return err
			}
			after, err := readBenchmarkMetrics(args[1])
			if err != nil {
				return err
			}
			comparison := compareBenchmarks(before, after)
			if jsonOutput {
				raw, err := json.MarshalIndent(comparison, "", "  ")
				if err != nil {
					return err
				}
				_, err = fmt.Fprintln(cmd.OutOrStdout(), string(raw))
				return err
			}
			_, err = fmt.Fprint(cmd.OutOrStdout(), formatBenchmarkComparison(comparison))
			return err
		},
	}
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "print the full machine-readable comparison")
	return cmd
}

func readBenchmarkMetrics(path string) (benchmarkMetrics, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return benchmarkMetrics{}, err
	}
	var payload any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return benchmarkMetrics{}, err
	}
	root, ok := payload.(map[string]any)
	if !ok {
		return benchmarkMetrics{}, fmt.Errorf("invalid benchmark JSON: %s", path)
	}
	metrics := benchmarkMetrics{
		Label:        stringMapValue(root, "label"),
		PhaseMs:      map[string]float64{},
		Counts:       map[string]float64{},
		Relationship: map[string]float64{},
		NodeLabels:   map[string]float64{},
	}
	if keyMetrics, ok := root["keyMetrics"].(map[string]any); ok {
		if metrics.Label == "" {
			metrics.Label = stringMapValue(root, "label")
		}
		metrics.TotalWallMs = optionalFloat(keyMetrics, "totalWallMs")
		metrics.PhaseMs = numberRecord(keyMetrics, "phaseMs", false)
		metrics.Relationship = numberRecord(keyMetrics, "relationshipCountsByType", false)
		metrics.NodeLabels = numberRecord(keyMetrics, "nodeCountsByLabel", false)
		addOptionalCount(metrics.Counts, "nodes", optionalFloat(keyMetrics, "nodeCount"))
		addOptionalCount(metrics.Counts, "relationships", optionalFloat(keyMetrics, "relationshipCount"))
		return metrics, nil
	}
	if total := optionalDurationMs(root, "totalDuration"); total != nil {
		metrics.TotalWallMs = total
	}
	if phases, ok := root["phases"].([]any); ok {
		for _, item := range phases {
			phase, ok := item.(map[string]any)
			if !ok {
				continue
			}
			name := stringMapValue(phase, "name")
			value := durationMsFromAny(phase["duration"])
			if name != "" && value != nil {
				metrics.PhaseMs[name] = *value
			}
		}
	}
	if files, ok := root["files"].(map[string]any); ok {
		for _, key := range []string{"scanned", "parsed", "unsupported", "failed"} {
			addOptionalCount(metrics.Counts, "files."+key, optionalFloat(files, key))
		}
	}
	if dbLoad, ok := root["dbLoad"].(map[string]any); ok {
		for _, key := range []string{"nodeRows", "relationshipRows", "fallbackInsertFailures", "skippedRelationships"} {
			addOptionalCount(metrics.Counts, "dbLoad."+key, optionalFloat(dbLoad, key))
		}
	}
	if nodes, ok := root["nodes"].([]any); ok {
		value := float64(len(nodes))
		metrics.Counts["nodes"] = value
	}
	if relationships, ok := root["relationships"].([]any); ok {
		value := float64(len(relationships))
		metrics.Counts["relationships"] = value
	}
	if metrics.TotalWallMs == nil && len(metrics.Counts) == 0 && len(metrics.PhaseMs) == 0 {
		return benchmarkMetrics{}, fmt.Errorf("unsupported benchmark JSON shape: %s", path)
	}
	return metrics, nil
}

func compareBenchmarks(before benchmarkMetrics, after benchmarkMetrics) benchmarkComparison {
	return benchmarkComparison{
		BeforeLabel:   before.Label,
		AfterLabel:    after.Label,
		TotalWallMs:   compareOptionalNumbers(before.TotalWallMs, after.TotalWallMs),
		PhaseMs:       compareNumberMaps(before.PhaseMs, after.PhaseMs),
		Counts:        compareNumberMaps(before.Counts, after.Counts),
		Relationships: compareNumberMaps(before.Relationship, after.Relationship),
		NodeLabels:    compareNumberMaps(before.NodeLabels, after.NodeLabels),
	}
}

func formatBenchmarkComparison(comparison benchmarkComparison) string {
	var lines []string
	beforeLabel := firstNonEmpty(comparison.BeforeLabel, "before")
	afterLabel := firstNonEmpty(comparison.AfterLabel, "after")
	lines = append(lines, "AVmatrix benchmark comparison")
	lines = append(lines, fmt.Sprintf("labels: %s -> %s", beforeLabel, afterLabel))
	lines = append(lines, fmt.Sprintf("wall: %s", formatDelta(comparison.TotalWallMs)))
	appendDeltaSection(&lines, "phaseMs", comparison.PhaseMs)
	appendDeltaSection(&lines, "counts", comparison.Counts)
	appendDeltaSection(&lines, "relationshipCountsByType", comparison.Relationships)
	appendDeltaSection(&lines, "nodeCountsByLabel", comparison.NodeLabels)
	return strings.Join(lines, "\n") + "\n"
}

func appendDeltaSection(lines *[]string, title string, values map[string]numericDelta) {
	*lines = append(*lines, "", title+":")
	if len(values) == 0 {
		*lines = append(*lines, "  none")
		return
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		*lines = append(*lines, fmt.Sprintf("  %s: %s", key, formatDeltaValue(values[key])))
	}
}

func compareNumberMaps(before map[string]float64, after map[string]float64) map[string]numericDelta {
	keys := map[string]bool{}
	for key := range before {
		keys[key] = true
	}
	for key := range after {
		keys[key] = true
	}
	out := make(map[string]numericDelta, len(keys))
	for key := range keys {
		beforeValue, beforeOK := before[key]
		afterValue, afterOK := after[key]
		out[key] = numericDeltaFor(optionalNumber(beforeValue, beforeOK), optionalNumber(afterValue, afterOK))
	}
	return out
}

func compareOptionalNumbers(before *float64, after *float64) *numericDelta {
	if before == nil && after == nil {
		return nil
	}
	delta := numericDeltaFor(before, after)
	return &delta
}

func numericDeltaFor(before *float64, after *float64) numericDelta {
	out := numericDelta{Before: before, After: after}
	if before != nil && after != nil {
		delta := *after - *before
		out.Delta = &delta
		if *before != 0 {
			percent := (delta / *before) * 100
			out.PercentChange = &percent
		}
	}
	return out
}

func optionalNumber(value float64, ok bool) *float64 {
	if !ok {
		return nil
	}
	return &value
}

func optionalFloat(values map[string]any, key string) *float64 {
	value, ok := values[key]
	if !ok {
		return nil
	}
	switch typed := value.(type) {
	case float64:
		return &typed
	case int:
		converted := float64(typed)
		return &converted
	default:
		return nil
	}
}

func optionalDurationMs(values map[string]any, key string) *float64 {
	return durationMsFromAny(values[key])
}

func durationMsFromAny(value any) *float64 {
	number, ok := value.(float64)
	if !ok {
		return nil
	}
	if math.Abs(number) > 1_000_000 {
		number = number / 1_000_000
	}
	return &number
}

func numberRecord(values map[string]any, key string, durations bool) map[string]float64 {
	raw, ok := values[key].(map[string]any)
	if !ok {
		return map[string]float64{}
	}
	out := make(map[string]float64, len(raw))
	for itemKey, itemValue := range raw {
		var value *float64
		if durations {
			value = durationMsFromAny(itemValue)
		} else if numeric, ok := itemValue.(float64); ok {
			value = &numeric
		}
		if value != nil {
			out[itemKey] = *value
		}
	}
	return out
}

func addOptionalCount(values map[string]float64, key string, value *float64) {
	if value != nil {
		values[key] = *value
	}
}

func stringMapValue(values map[string]any, key string) string {
	value, _ := values[key].(string)
	return value
}

func formatDelta(delta *numericDelta) string {
	if delta == nil {
		return "n/a"
	}
	return formatDeltaValue(*delta)
}

func formatDeltaValue(delta numericDelta) string {
	before := formatFloatPtr(delta.Before)
	after := formatFloatPtr(delta.After)
	change := "n/a"
	if delta.Delta != nil {
		change = signedFloat(*delta.Delta)
	}
	pct := ""
	if delta.PercentChange != nil {
		pct = ", " + signedFloat(*delta.PercentChange) + "%"
	}
	return fmt.Sprintf("%s -> %s (%s%s)", before, after, change, pct)
}

func formatFloatPtr(value *float64) string {
	if value == nil {
		return "n/a"
	}
	if math.Abs(*value-math.Round(*value)) < 0.05 {
		return fmt.Sprintf("%.0f", *value)
	}
	return fmt.Sprintf("%.1f", *value)
}

func signedFloat(value float64) string {
	sign := ""
	if value > 0 {
		sign = "+"
	}
	if math.Abs(value-math.Round(value)) < 0.05 {
		return fmt.Sprintf("%s%.0f", sign, value)
	}
	return fmt.Sprintf("%s%.1f", sign, value)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
