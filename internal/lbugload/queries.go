package lbugload

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/tamnguyendinh/avmatrix-go/internal/lbugschema"
)

const copyCSVOptions = `(HEADER=true, ESCAPE='"', DELIM=',', QUOTE='"', PARALLEL=false, auto_detect=false)`

type RelationshipCSVRow struct {
	FromID           string
	ToID             string
	Type             string
	Confidence       string
	Reason           string
	Step             string
	ResolutionSource string
	Evidence         string
	FileHash         string
}

func NodeCopyQuery(table string, csvPath string) (string, error) {
	columns, ok := nodeColumns(table)
	if !ok {
		return "", fmt.Errorf("unsupported node table %q", table)
	}
	return fmt.Sprintf(
		`COPY %s(%s) FROM "%s" %s`,
		lbugschema.FormatIdent(table),
		strings.Join(columns, ", "),
		NormalizeCopyPath(csvPath),
		copyCSVOptions,
	), nil
}

func RelationshipCopyQuery(fromLabel string, toLabel string, csvPath string) string {
	return fmt.Sprintf(
		`COPY %s FROM "%s" (from="%s", to="%s", HEADER=true, ESCAPE='"', DELIM=',', QUOTE='"', PARALLEL=false, auto_detect=false)`,
		lbugschema.RelTableName,
		NormalizeCopyPath(csvPath),
		fromLabel,
		toLabel,
	)
}

func RetryCopyQuery(query string) string {
	return strings.Replace(query, "auto_detect=false)", "auto_detect=false, IGNORE_ERRORS=true)", 1)
}

func FallbackRelationshipInsertQuery(row RelationshipCSVRow, fromLabel string, toLabel string) string {
	confidence := parseFloatString(row.Confidence, 1)
	step := parseIntString(row.Step, 0)
	return fmt.Sprintf(
		`MATCH (a:%s {id: %s}), (b:%s {id: %s}) CREATE (a)-[:%s {type: %s, confidence: %s, reason: %s, step: %d, resolutionSource: %s, evidence: %s, fileHash: %s}]->(b)`,
		lbugschema.FormatIdent(fromLabel),
		cypherString(row.FromID),
		lbugschema.FormatIdent(toLabel),
		cypherString(row.ToID),
		lbugschema.RelTableName,
		cypherString(row.Type),
		strconv.FormatFloat(confidence, 'f', -1, 64),
		cypherString(row.Reason),
		step,
		cypherString(row.ResolutionSource),
		cypherString(row.Evidence),
		cypherString(row.FileHash),
	)
}

func NormalizeCopyPath(path string) string {
	return strings.ReplaceAll(path, `\`, `/`)
}

func ReadRelationshipCSVRows(csvPath string) ([]RelationshipCSVRow, error) {
	file, err := os.Open(csvPath)
	if err != nil {
		return nil, fmt.Errorf("open relationship csv %s: %w", csvPath, err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1
	if _, err := reader.Read(); err != nil {
		if err == io.EOF {
			return nil, nil
		}
		return nil, fmt.Errorf("read relationship csv header %s: %w", csvPath, err)
	}

	var rows []RelationshipCSVRow
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read relationship csv row %s: %w", csvPath, err)
		}
		row, ok := relationshipRowFromRecord(record)
		if !ok {
			continue
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func relationshipRowFromRecord(record []string) (RelationshipCSVRow, bool) {
	if len(record) < len(relationshipColumns) {
		return RelationshipCSVRow{}, false
	}
	return RelationshipCSVRow{
		FromID:           record[0],
		ToID:             record[1],
		Type:             record[2],
		Confidence:       record[3],
		Reason:           record[4],
		Step:             record[5],
		ResolutionSource: record[6],
		Evidence:         record[7],
		FileHash:         record[8],
	}, true
}

func cypherString(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `'`, `''`)
	value = strings.ReplaceAll(value, "\n", `\n`)
	value = strings.ReplaceAll(value, "\r", `\r`)
	return "'" + value + "'"
}

func parseFloatString(value string, fallback float64) float64 {
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return fallback
	}
	return parsed
}

func parseIntString(value string, fallback int) int {
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}
