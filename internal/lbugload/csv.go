package lbugload

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/tamnguyendinh/avmatrix-go/internal/graph"
	"github.com/tamnguyendinh/avmatrix-go/internal/lbugschema"
)

const RelationshipCSVHeader = "from,to,type,confidence,reason,step,resolutionSource,evidence,fileHash"

var relationshipColumns = []string{"from", "to", "type", "confidence", "reason", "step", "resolutionSource", "evidence", "fileHash"}

var (
	fileNodeColumns      = []string{"id", "name", "filePath", "content"}
	folderNodeColumns    = []string{"id", "name", "filePath"}
	symbolNodeColumns    = []string{"id", "name", "filePath", "startLine", "endLine", "isExported", "content", "description"}
	methodNodeColumns    = []string{"id", "name", "filePath", "startLine", "endLine", "isExported", "content", "description", "parameterCount", "returnType"}
	communityNodeColumns = []string{"id", "label", "heuristicLabel", "keywords", "description", "enrichedBy", "cohesion", "symbolCount"}
	processNodeColumns   = []string{"id", "label", "heuristicLabel", "processType", "stepCount", "communities", "entryPointId", "terminalId"}
	sectionNodeColumns   = []string{"id", "name", "filePath", "startLine", "endLine", "level", "content", "description"}
	routeNodeColumns     = []string{"id", "name", "filePath", "responseKeys", "errorKeys", "middleware"}
	toolNodeColumns      = []string{"id", "name", "filePath", "description"}
	defaultNodeColumns   = []string{"id", "name", "filePath", "startLine", "endLine", "content", "description"}
	nodeColumnLookup     = map[string][]string{
		"File":        fileNodeColumns,
		"Folder":      folderNodeColumns,
		"Function":    symbolNodeColumns,
		"Class":       symbolNodeColumns,
		"Interface":   symbolNodeColumns,
		"CodeElement": symbolNodeColumns,
		"Method":      methodNodeColumns,
		"Community":   communityNodeColumns,
		"Process":     processNodeColumns,
		"Section":     sectionNodeColumns,
		"Route":       routeNodeColumns,
		"Tool":        toolNodeColumns,
	}
	validNodeTableLookup = makeValidNodeTableLookup()
	relationPairLookup   = makeRelationPairLookup()
)

type NodeCSVFile struct {
	Table   string
	CSVPath string
	Rows    int
	Bytes   int64
}

type RelationshipPairCSV struct {
	From          string
	To            string
	CSVPath       string
	Rows          int
	Bytes         int64
	CopySupported bool
}

type ExportMetrics struct {
	RowsByTable          map[string]int
	BytesByTable         map[string]int64
	SkippedNodes         int
	SkippedRelationships int
}

type CSVExport struct {
	CSVDir                string
	NodeFiles             []NodeCSVFile
	RelationshipCSVPath   string
	RelationshipRows      int
	RelationshipPairFiles []RelationshipPairCSV
	Metrics               ExportMetrics
}

func ExportGraphCSVs(g *graph.Graph, csvDir string) (*CSVExport, error) {
	if g == nil {
		return nil, fmt.Errorf("graph is nil")
	}
	if err := os.RemoveAll(csvDir); err != nil {
		return nil, fmt.Errorf("remove stale csv dir: %w", err)
	}
	if err := os.MkdirAll(csvDir, 0o755); err != nil {
		return nil, fmt.Errorf("create csv dir: %w", err)
	}

	result := &CSVExport{
		CSVDir:              csvDir,
		RelationshipCSVPath: filepath.Join(csvDir, "relations.csv"),
		Metrics: ExportMetrics{
			RowsByTable:  map[string]int{},
			BytesByTable: map[string]int64{},
		},
	}

	nodeWriters := map[string]*csvFileWriter{}
	seenNodeIDs := map[string]struct{}{}
	nodeLabels := map[string]string{}
	for _, node := range g.Nodes {
		nodeLabels[node.ID] = string(node.Label)
		if _, seen := seenNodeIDs[node.ID]; seen {
			continue
		}
		seenNodeIDs[node.ID] = struct{}{}
		table := string(node.Label)
		columns, ok := nodeColumns(table)
		if !ok {
			result.Metrics.SkippedNodes++
			continue
		}
		writer, err := writerForTable(nodeWriters, csvDir, table, columns)
		if err != nil {
			closeWriters(nodeWriters)
			return nil, err
		}
		row := nodeCSVRow(node, table)
		if err := writer.Write(row); err != nil {
			closeWriters(nodeWriters)
			return nil, err
		}
	}
	if err := closeWriters(nodeWriters); err != nil {
		return nil, err
	}

	for _, table := range lbugschema.NodeTables {
		writer, ok := nodeWriters[table]
		if !ok || writer.rows == 0 {
			continue
		}
		bytes, err := fileSize(writer.path)
		if err != nil {
			return nil, err
		}
		result.NodeFiles = append(result.NodeFiles, NodeCSVFile{
			Table:   table,
			CSVPath: writer.path,
			Rows:    writer.rows,
			Bytes:   bytes,
		})
		result.Metrics.RowsByTable[table] = writer.rows
		result.Metrics.BytesByTable[table] = bytes
	}

	relWriter, err := newCSVFileWriter(result.RelationshipCSVPath, relationshipColumns)
	if err != nil {
		return nil, err
	}
	pairWriters := map[string]*csvFileWriter{}
	pairMeta := map[string]RelationshipPairCSV{}
	validTables := validNodeTables()
	for _, rel := range g.SortedRelationships() {
		row := relationshipCSVRow(rel)
		if err := relWriter.Write(row); err != nil {
			closeWriters(pairWriters)
			relWriter.Close()
			return nil, err
		}

		fromLabel, fromOK := nodeLabels[rel.SourceID]
		toLabel, toOK := nodeLabels[rel.TargetID]
		if !fromOK || !toOK || !validTables[fromLabel] || !validTables[toLabel] {
			result.Metrics.SkippedRelationships++
			continue
		}
		key := pairKey(fromLabel, toLabel)
		writer, ok := pairWriters[key]
		if !ok {
			pairPath := filepath.Join(csvDir, fmt.Sprintf("rel_%s_%s.csv", fromLabel, toLabel))
			writer, err = newCSVFileWriter(pairPath, relationshipColumns)
			if err != nil {
				closeWriters(pairWriters)
				relWriter.Close()
				return nil, err
			}
			pairWriters[key] = writer
			pairMeta[key] = RelationshipPairCSV{
				From:          fromLabel,
				To:            toLabel,
				CSVPath:       pairPath,
				CopySupported: relationPairSupported(fromLabel, toLabel),
			}
		}
		if err := writer.Write(row); err != nil {
			closeWriters(pairWriters)
			relWriter.Close()
			return nil, err
		}
	}
	if err := relWriter.Close(); err != nil {
		closeWriters(pairWriters)
		return nil, err
	}
	if err := closeWriters(pairWriters); err != nil {
		return nil, err
	}

	relBytes, err := fileSize(result.RelationshipCSVPath)
	if err != nil {
		return nil, err
	}
	result.RelationshipRows = relWriter.rows
	result.Metrics.RowsByTable["Relationship"] = relWriter.rows
	result.Metrics.BytesByTable["Relationship"] = relBytes

	pairKeys := make([]string, 0, len(pairMeta))
	for key := range pairMeta {
		pairKeys = append(pairKeys, key)
	}
	sort.Strings(pairKeys)
	for _, key := range pairKeys {
		meta := pairMeta[key]
		writer := pairWriters[key]
		bytes, err := fileSize(meta.CSVPath)
		if err != nil {
			return nil, err
		}
		meta.Rows = writer.rows
		meta.Bytes = bytes
		result.RelationshipPairFiles = append(result.RelationshipPairFiles, meta)
	}

	return result, nil
}

type csvFileWriter struct {
	path   string
	file   *os.File
	writer *csv.Writer
	rows   int
}

func newCSVFileWriter(path string, header []string) (*csvFileWriter, error) {
	file, err := os.Create(path)
	if err != nil {
		return nil, fmt.Errorf("create csv %s: %w", path, err)
	}
	writer := csv.NewWriter(file)
	writer.UseCRLF = false
	if err := writer.Write(header); err != nil {
		file.Close()
		return nil, fmt.Errorf("write csv header %s: %w", path, err)
	}
	return &csvFileWriter{path: path, file: file, writer: writer}, nil
}

func (w *csvFileWriter) Write(row []string) error {
	if err := w.writer.Write(row); err != nil {
		return fmt.Errorf("write csv row %s: %w", w.path, err)
	}
	w.rows++
	return nil
}

func (w *csvFileWriter) Close() error {
	w.writer.Flush()
	writeErr := w.writer.Error()
	closeErr := w.file.Close()
	if writeErr != nil {
		return fmt.Errorf("flush csv %s: %w", w.path, writeErr)
	}
	if closeErr != nil {
		return fmt.Errorf("close csv %s: %w", w.path, closeErr)
	}
	return nil
}

func writerForTable(writers map[string]*csvFileWriter, csvDir string, table string, columns []string) (*csvFileWriter, error) {
	if writer, ok := writers[table]; ok {
		return writer, nil
	}
	csvPath := filepath.Join(csvDir, strings.ToLower(table)+".csv")
	writer, err := newCSVFileWriter(csvPath, columns)
	if err != nil {
		return nil, err
	}
	writers[table] = writer
	return writer, nil
}

func closeWriters(writers map[string]*csvFileWriter) error {
	var firstErr error
	for _, writer := range writers {
		if err := writer.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func nodeCSVRow(node graph.Node, table string) []string {
	props := node.Properties
	switch table {
	case "File":
		return []string{node.ID, stringProp(props, "name", ""), stringProp(props, "filePath", ""), stringProp(props, "content", "")}
	case "Folder":
		return []string{node.ID, stringProp(props, "name", ""), stringProp(props, "filePath", "")}
	case "Function", "Class", "Interface", "CodeElement":
		return []string{node.ID, stringProp(props, "name", ""), stringProp(props, "filePath", ""), intProp(props, "startLine", -1), intProp(props, "endLine", -1), boolProp(props, "isExported"), stringProp(props, "content", ""), stringProp(props, "description", "")}
	case "Method":
		return []string{node.ID, stringProp(props, "name", ""), stringProp(props, "filePath", ""), intProp(props, "startLine", -1), intProp(props, "endLine", -1), boolProp(props, "isExported"), stringProp(props, "content", ""), stringProp(props, "description", ""), intProp(props, "parameterCount", 0), stringProp(props, "returnType", "")}
	case "Community":
		return []string{node.ID, firstStringProp(props, []string{"label", "name"}, ""), stringProp(props, "heuristicLabel", ""), arrayLiteral(props["keywords"]), stringProp(props, "description", ""), stringProp(props, "enrichedBy", "heuristic"), floatProp(props, "cohesion", 0), intProp(props, "symbolCount", 0)}
	case "Process":
		return []string{node.ID, firstStringProp(props, []string{"label", "name"}, ""), stringProp(props, "heuristicLabel", ""), stringProp(props, "processType", ""), intProp(props, "stepCount", 0), arrayLiteral(props["communities"]), stringProp(props, "entryPointId", ""), stringProp(props, "terminalId", "")}
	case "Section":
		return []string{node.ID, stringProp(props, "name", ""), stringProp(props, "filePath", ""), intProp(props, "startLine", -1), intProp(props, "endLine", -1), intProp(props, "level", 1), stringProp(props, "content", ""), stringProp(props, "description", "")}
	case "Route":
		return []string{node.ID, stringProp(props, "name", ""), stringProp(props, "filePath", ""), arrayLiteral(props["responseKeys"]), arrayLiteral(props["errorKeys"]), arrayLiteral(props["middleware"])}
	case "Tool":
		return []string{node.ID, stringProp(props, "name", ""), stringProp(props, "filePath", ""), stringProp(props, "description", "")}
	default:
		return []string{node.ID, stringProp(props, "name", ""), stringProp(props, "filePath", ""), intProp(props, "startLine", -1), intProp(props, "endLine", -1), stringProp(props, "content", ""), stringProp(props, "description", "")}
	}
}

func relationshipCSVRow(rel graph.Relationship) []string {
	confidence := rel.Confidence
	if confidence == 0 {
		confidence = 1
	}
	step := 0
	if rel.Step != nil {
		step = *rel.Step
	}
	return []string{
		rel.SourceID,
		rel.TargetID,
		string(rel.Type),
		strconv.FormatFloat(confidence, 'f', -1, 64),
		rel.Reason,
		strconv.Itoa(step),
		rel.ResolutionSource,
		relationshipEvidence(rel),
		rel.FileHash,
	}
}

func relationshipEvidence(rel graph.Relationship) string {
	if len(rel.Evidence) == 0 {
		return ""
	}
	raw, err := json.Marshal(rel.Evidence)
	if err != nil {
		return ""
	}
	return string(raw)
}

func nodeColumns(table string) ([]string, bool) {
	if columns, ok := nodeColumnLookup[table]; ok {
		return columns, true
	}
	if validNodeTableLookup[table] {
		return defaultNodeColumns, true
	}
	return nil, false
}

func stringProp(props graph.NodeProperties, key string, fallback string) string {
	value, ok := props[key]
	if !ok || value == nil {
		return fallback
	}
	switch typed := value.(type) {
	case string:
		if typed == "" {
			return fallback
		}
		return sanitizeString(typed)
	case fmt.Stringer:
		return sanitizeString(typed.String())
	default:
		return sanitizeString(fmt.Sprint(typed))
	}
}

func firstStringProp(props graph.NodeProperties, keys []string, fallback string) string {
	for _, key := range keys {
		if value := stringProp(props, key, ""); value != "" {
			return value
		}
	}
	return fallback
}

func intProp(props graph.NodeProperties, key string, fallback int) string {
	value, ok := props[key]
	if !ok || value == nil {
		return strconv.Itoa(fallback)
	}
	switch typed := value.(type) {
	case int:
		return strconv.Itoa(typed)
	case int64:
		return strconv.FormatInt(typed, 10)
	case int32:
		return strconv.FormatInt(int64(typed), 10)
	case float64:
		return strconv.Itoa(int(typed))
	case float32:
		return strconv.Itoa(int(typed))
	case string:
		if typed == "" {
			return strconv.Itoa(fallback)
		}
		if parsed, err := strconv.Atoi(typed); err == nil {
			return strconv.Itoa(parsed)
		}
	}
	return strconv.Itoa(fallback)
}

func floatProp(props graph.NodeProperties, key string, fallback float64) string {
	value, ok := props[key]
	if !ok || value == nil {
		return strconv.FormatFloat(fallback, 'f', -1, 64)
	}
	switch typed := value.(type) {
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64)
	case float32:
		return strconv.FormatFloat(float64(typed), 'f', -1, 64)
	case int:
		return strconv.FormatFloat(float64(typed), 'f', -1, 64)
	case int64:
		return strconv.FormatFloat(float64(typed), 'f', -1, 64)
	case string:
		if parsed, err := strconv.ParseFloat(typed, 64); err == nil {
			return strconv.FormatFloat(parsed, 'f', -1, 64)
		}
	}
	return strconv.FormatFloat(fallback, 'f', -1, 64)
}

func boolProp(props graph.NodeProperties, key string) string {
	value, ok := props[key]
	if !ok || value == nil {
		return "false"
	}
	switch typed := value.(type) {
	case bool:
		if typed {
			return "true"
		}
	case string:
		if strings.EqualFold(typed, "true") {
			return "true"
		}
	}
	return "false"
}

func arrayLiteral(value any) string {
	if typed, ok := value.(string); ok && strings.HasPrefix(typed, "[") && strings.HasSuffix(typed, "]") {
		return sanitizeString(typed)
	}
	items := stringItems(value)
	if len(items) == 0 {
		return "[]"
	}
	quoted := make([]string, 0, len(items))
	for _, item := range items {
		clean := sanitizeString(item)
		clean = strings.ReplaceAll(clean, `\`, `\\`)
		clean = strings.ReplaceAll(clean, `'`, `''`)
		quoted = append(quoted, "'"+clean+"'")
	}
	return "[" + strings.Join(quoted, ",") + "]"
}

func stringItems(value any) []string {
	switch typed := value.(type) {
	case []string:
		return typed
	case []any:
		items := make([]string, 0, len(typed))
		for _, item := range typed {
			if item != nil {
				items = append(items, fmt.Sprint(item))
			}
		}
		return items
	case string:
		if typed == "" {
			return nil
		}
		if strings.HasPrefix(typed, "[") && strings.HasSuffix(typed, "]") {
			return []string{typed}
		}
		return []string{typed}
	default:
		if value == nil {
			return nil
		}
		return []string{fmt.Sprint(value)}
	}
}

func sanitizeString(value string) string {
	value = strings.ToValidUTF8(value, "")
	value = strings.ReplaceAll(value, "\r\n", "\n")
	value = strings.ReplaceAll(value, "\r", "\n")
	return strings.Map(func(r rune) rune {
		if r == '\n' || r == '\t' {
			return r
		}
		if r < 0x20 || r == 0x7f || r == '\uFFFE' || r == '\uFFFF' {
			return -1
		}
		return r
	}, value)
}

func validNodeTables() map[string]bool {
	return validNodeTableLookup
}

func makeValidNodeTableLookup() map[string]bool {
	tables := make(map[string]bool, len(lbugschema.NodeTables))
	for _, table := range lbugschema.NodeTables {
		tables[table] = true
	}
	return tables
}

func relationPairSupported(from string, to string) bool {
	return relationPairLookup[pairKey(from, to)]
}

func makeRelationPairLookup() map[string]bool {
	pairs := make(map[string]bool, len(lbugschema.RelationPairs))
	for _, pair := range lbugschema.RelationPairs {
		pairs[pairKey(pair.From, pair.To)] = true
	}
	return pairs
}

func pairKey(from string, to string) string {
	return from + "|" + to
}

func fileSize(path string) (int64, error) {
	stat, err := os.Stat(path)
	if err != nil {
		return 0, fmt.Errorf("stat csv %s: %w", path, err)
	}
	return stat.Size(), nil
}
