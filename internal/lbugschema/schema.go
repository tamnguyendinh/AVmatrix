package lbugschema

import (
	"fmt"
	"strings"
)

const (
	RelTableName         = "CodeRelation"
	EmbeddingTableName   = "CodeEmbedding"
	DefaultEmbeddingDims = 384
	EmbeddingIndexName   = "code_embedding_idx"
	StaleHashSentinel    = ""
)

var NodeTables = []string{
	"File",
	"Folder",
	"Function",
	"Class",
	"Interface",
	"Method",
	"CodeElement",
	"Community",
	"Process",
	"Package",
	"Section",
	"Struct",
	"Enum",
	"Macro",
	"Typedef",
	"Union",
	"Namespace",
	"Trait",
	"Impl",
	"TypeAlias",
	"Const",
	"Static",
	"Variable",
	"Property",
	"Record",
	"Delegate",
	"Annotation",
	"Constructor",
	"Template",
	"Module",
	"Route",
	"Tool",
}

var RelationshipTypes = []string{
	"CONTAINS",
	"DEFINES",
	"IMPORTS",
	"CALLS",
	"USES",
	"INHERITS",
	"EXTENDS",
	"IMPLEMENTS",
	"HAS_METHOD",
	"HAS_PROPERTY",
	"ACCESSES",
	"METHOD_OVERRIDES",
	"OVERRIDES",
	"METHOD_IMPLEMENTS",
	"MEMBER_OF",
	"STEP_IN_PROCESS",
	"HANDLES_ROUTE",
	"FETCHES",
	"HANDLES_TOOL",
	"ENTRY_POINT_OF",
	"WRAPS",
	"QUERIES",
}

type RelationPair struct {
	From string
	To   string
}

var RelationPairs = []RelationPair{
	{From: "File", To: "File"},
	{From: "File", To: "Folder"},
	{From: "File", To: "Function"},
	{From: "File", To: "Class"},
	{From: "File", To: "Interface"},
	{From: "File", To: "Method"},
	{From: "File", To: "CodeElement"},
	{From: "File", To: "Package"},
	{From: "File", To: "Struct"},
	{From: "File", To: "Enum"},
	{From: "File", To: "Macro"},
	{From: "File", To: "Typedef"},
	{From: "File", To: "Union"},
	{From: "File", To: "Namespace"},
	{From: "File", To: "Trait"},
	{From: "File", To: "Impl"},
	{From: "File", To: "TypeAlias"},
	{From: "File", To: "Const"},
	{From: "File", To: "Static"},
	{From: "File", To: "Variable"},
	{From: "File", To: "Property"},
	{From: "File", To: "Record"},
	{From: "File", To: "Delegate"},
	{From: "File", To: "Annotation"},
	{From: "File", To: "Constructor"},
	{From: "File", To: "Template"},
	{From: "File", To: "Module"},
	{From: "File", To: "Section"},
	{From: "Folder", To: "Folder"},
	{From: "Folder", To: "File"},
	{From: "Function", To: "Function"},
	{From: "Function", To: "Method"},
	{From: "Function", To: "Class"},
	{From: "Function", To: "Community"},
	{From: "Function", To: "Macro"},
	{From: "Function", To: "Struct"},
	{From: "Function", To: "Template"},
	{From: "Function", To: "Enum"},
	{From: "Function", To: "Namespace"},
	{From: "Function", To: "TypeAlias"},
	{From: "Function", To: "Module"},
	{From: "Function", To: "Impl"},
	{From: "Function", To: "Interface"},
	{From: "Function", To: "Constructor"},
	{From: "Function", To: "Const"},
	{From: "Function", To: "Typedef"},
	{From: "Function", To: "Union"},
	{From: "Function", To: "Property"},
	{From: "Function", To: "CodeElement"},
	{From: "Class", To: "Method"},
	{From: "Class", To: "Function"},
	{From: "Class", To: "Class"},
	{From: "Class", To: "Interface"},
	{From: "Class", To: "Community"},
	{From: "Class", To: "Template"},
	{From: "Class", To: "TypeAlias"},
	{From: "Class", To: "Struct"},
	{From: "Class", To: "Enum"},
	{From: "Class", To: "Annotation"},
	{From: "Class", To: "Constructor"},
	{From: "Class", To: "Trait"},
	{From: "Class", To: "Macro"},
	{From: "Class", To: "Impl"},
	{From: "Class", To: "Union"},
	{From: "Class", To: "Namespace"},
	{From: "Class", To: "Typedef"},
	{From: "Class", To: "Property"},
	{From: "Method", To: "Function"},
	{From: "Method", To: "Method"},
	{From: "Method", To: "Class"},
	{From: "Method", To: "Community"},
	{From: "Method", To: "Template"},
	{From: "Method", To: "Struct"},
	{From: "Method", To: "TypeAlias"},
	{From: "Method", To: "Enum"},
	{From: "Method", To: "Macro"},
	{From: "Method", To: "Namespace"},
	{From: "Method", To: "Module"},
	{From: "Method", To: "Impl"},
	{From: "Method", To: "Interface"},
	{From: "Method", To: "Constructor"},
	{From: "Method", To: "Property"},
	{From: "Method", To: "CodeElement"},
	{From: "Template", To: "Template"},
	{From: "Template", To: "Function"},
	{From: "Template", To: "Method"},
	{From: "Template", To: "Class"},
	{From: "Template", To: "Struct"},
	{From: "Template", To: "TypeAlias"},
	{From: "Template", To: "Enum"},
	{From: "Template", To: "Macro"},
	{From: "Template", To: "Interface"},
	{From: "Template", To: "Constructor"},
	{From: "Module", To: "Module"},
	{From: "Section", To: "Section"},
	{From: "Section", To: "File"},
	{From: "File", To: "Route"},
	{From: "Function", To: "Route"},
	{From: "Method", To: "Route"},
	{From: "File", To: "Tool"},
	{From: "Function", To: "Tool"},
	{From: "Method", To: "Tool"},
	{From: "CodeElement", To: "Community"},
	{From: "CodeElement", To: "CodeElement"},
	{From: "CodeElement", To: "Module"},
	{From: "Interface", To: "Community"},
	{From: "Interface", To: "Function"},
	{From: "Interface", To: "Method"},
	{From: "Interface", To: "Class"},
	{From: "Interface", To: "Enum"},
	{From: "Interface", To: "Interface"},
	{From: "Interface", To: "TypeAlias"},
	{From: "Interface", To: "Struct"},
	{From: "Interface", To: "Constructor"},
	{From: "Interface", To: "Property"},
	{From: "Struct", To: "Community"},
	{From: "Struct", To: "Trait"},
	{From: "Struct", To: "Struct"},
	{From: "Struct", To: "Class"},
	{From: "Struct", To: "Enum"},
	{From: "Struct", To: "Function"},
	{From: "Struct", To: "Method"},
	{From: "Struct", To: "Interface"},
	{From: "Struct", To: "TypeAlias"},
	{From: "Struct", To: "Constructor"},
	{From: "Struct", To: "Property"},
	{From: "Enum", To: "Enum"},
	{From: "Enum", To: "Community"},
	{From: "Enum", To: "Class"},
	{From: "Enum", To: "Interface"},
	{From: "Macro", To: "Community"},
	{From: "Macro", To: "Function"},
	{From: "Macro", To: "Method"},
	{From: "Package", To: "Interface"},
	{From: "Package", To: "Struct"},
	{From: "Package", To: "TypeAlias"},
	{From: "Module", To: "Function"},
	{From: "Module", To: "Method"},
	{From: "Module", To: "Namespace"},
	{From: "Typedef", To: "Community"},
	{From: "Union", To: "Community"},
	{From: "Namespace", To: "Community"},
	{From: "Namespace", To: "Function"},
	{From: "Namespace", To: "Struct"},
	{From: "Trait", To: "Method"},
	{From: "Trait", To: "Constructor"},
	{From: "Trait", To: "Property"},
	{From: "Trait", To: "Community"},
	{From: "Impl", To: "Method"},
	{From: "Impl", To: "Constructor"},
	{From: "Impl", To: "Property"},
	{From: "Impl", To: "Community"},
	{From: "Impl", To: "Trait"},
	{From: "Impl", To: "Struct"},
	{From: "Impl", To: "Impl"},
	{From: "TypeAlias", To: "Community"},
	{From: "TypeAlias", To: "Trait"},
	{From: "TypeAlias", To: "Class"},
	{From: "TypeAlias", To: "Enum"},
	{From: "TypeAlias", To: "Function"},
	{From: "TypeAlias", To: "Interface"},
	{From: "TypeAlias", To: "Method"},
	{From: "TypeAlias", To: "Struct"},
	{From: "TypeAlias", To: "TypeAlias"},
	{From: "Const", To: "Community"},
	{From: "Const", To: "Function"},
	{From: "Const", To: "Struct"},
	{From: "Const", To: "TypeAlias"},
	{From: "Static", To: "Community"},
	{From: "Variable", To: "Community"},
	{From: "Variable", To: "Class"},
	{From: "Variable", To: "Enum"},
	{From: "Variable", To: "Function"},
	{From: "Variable", To: "Interface"},
	{From: "Variable", To: "Method"},
	{From: "Variable", To: "Property"},
	{From: "Variable", To: "Struct"},
	{From: "Variable", To: "TypeAlias"},
	{From: "Property", To: "Community"},
	{From: "Property", To: "Class"},
	{From: "Property", To: "Enum"},
	{From: "Property", To: "Function"},
	{From: "Property", To: "Interface"},
	{From: "Property", To: "Method"},
	{From: "Property", To: "Property"},
	{From: "Property", To: "Struct"},
	{From: "Property", To: "TypeAlias"},
	{From: "Record", To: "Method"},
	{From: "Record", To: "Constructor"},
	{From: "Record", To: "Property"},
	{From: "Record", To: "Community"},
	{From: "Delegate", To: "Community"},
	{From: "Annotation", To: "Community"},
	{From: "Constructor", To: "Community"},
	{From: "Constructor", To: "Interface"},
	{From: "Constructor", To: "Class"},
	{From: "Constructor", To: "Method"},
	{From: "Constructor", To: "Function"},
	{From: "Constructor", To: "Constructor"},
	{From: "Constructor", To: "Struct"},
	{From: "Constructor", To: "Macro"},
	{From: "Constructor", To: "Template"},
	{From: "Constructor", To: "TypeAlias"},
	{From: "Constructor", To: "Enum"},
	{From: "Constructor", To: "Annotation"},
	{From: "Constructor", To: "Impl"},
	{From: "Constructor", To: "Namespace"},
	{From: "Constructor", To: "Module"},
	{From: "Constructor", To: "Property"},
	{From: "Constructor", To: "Typedef"},
	{From: "Template", To: "Community"},
	{From: "Module", To: "Community"},
	{From: "Function", To: "Process"},
	{From: "Method", To: "Process"},
	{From: "Class", To: "Process"},
	{From: "Interface", To: "Process"},
	{From: "Struct", To: "Process"},
	{From: "Constructor", To: "Process"},
	{From: "Module", To: "Process"},
	{From: "Macro", To: "Process"},
	{From: "Impl", To: "Process"},
	{From: "Typedef", To: "Process"},
	{From: "TypeAlias", To: "Process"},
	{From: "Enum", To: "Process"},
	{From: "Union", To: "Process"},
	{From: "Namespace", To: "Process"},
	{From: "Trait", To: "Process"},
	{From: "Const", To: "Process"},
	{From: "Static", To: "Process"},
	{From: "Variable", To: "Process"},
	{From: "Property", To: "Process"},
	{From: "Record", To: "Process"},
	{From: "Delegate", To: "Process"},
	{From: "Annotation", To: "Process"},
	{From: "Template", To: "Process"},
	{From: "CodeElement", To: "Process"},
	{From: "Route", To: "Process"},
	{From: "Tool", To: "Process"},
}

func NodeSchemaQueries() []string {
	queries := make([]string, 0, len(NodeTables))
	for _, table := range NodeTables {
		queries = append(queries, NodeSchema(table))
	}
	return queries
}

func NodeSchema(table string) string {
	switch table {
	case "File":
		return nodeTable(table, []column{{"id", "STRING"}, {"name", "STRING"}, {"filePath", "STRING"}, {"content", "STRING"}})
	case "Folder":
		return nodeTable(table, []column{{"id", "STRING"}, {"name", "STRING"}, {"filePath", "STRING"}})
	case "Function", "Class", "Interface", "CodeElement":
		return nodeTable(table, []column{{"id", "STRING"}, {"name", "STRING"}, {"filePath", "STRING"}, {"startLine", "INT64"}, {"endLine", "INT64"}, {"isExported", "BOOLEAN"}, {"content", "STRING"}, {"description", "STRING"}})
	case "Method":
		return nodeTable(table, []column{{"id", "STRING"}, {"name", "STRING"}, {"filePath", "STRING"}, {"startLine", "INT64"}, {"endLine", "INT64"}, {"isExported", "BOOLEAN"}, {"content", "STRING"}, {"description", "STRING"}, {"parameterCount", "INT32"}, {"returnType", "STRING"}})
	case "Community":
		return nodeTable(table, []column{{"id", "STRING"}, {"label", "STRING"}, {"heuristicLabel", "STRING"}, {"keywords", "STRING[]"}, {"description", "STRING"}, {"enrichedBy", "STRING"}, {"cohesion", "DOUBLE"}, {"symbolCount", "INT32"}})
	case "Process":
		return nodeTable(table, []column{{"id", "STRING"}, {"label", "STRING"}, {"heuristicLabel", "STRING"}, {"processType", "STRING"}, {"stepCount", "INT32"}, {"communities", "STRING[]"}, {"entryPointId", "STRING"}, {"terminalId", "STRING"}})
	case "Route":
		return nodeTable(table, []column{{"id", "STRING"}, {"name", "STRING"}, {"filePath", "STRING"}, {"responseKeys", "STRING[]"}, {"errorKeys", "STRING[]"}, {"middleware", "STRING[]"}})
	case "Tool":
		return nodeTable(table, []column{{"id", "STRING"}, {"name", "STRING"}, {"filePath", "STRING"}, {"description", "STRING"}})
	case "Section":
		return nodeTable(table, []column{{"id", "STRING"}, {"name", "STRING"}, {"filePath", "STRING"}, {"startLine", "INT64"}, {"endLine", "INT64"}, {"level", "INT64"}, {"content", "STRING"}, {"description", "STRING"}})
	default:
		return nodeTable(table, []column{{"id", "STRING"}, {"name", "STRING"}, {"filePath", "STRING"}, {"startLine", "INT64"}, {"endLine", "INT64"}, {"content", "STRING"}, {"description", "STRING"}})
	}
}

func RelationSchema() string {
	var builder strings.Builder
	builder.WriteString("CREATE REL TABLE ")
	builder.WriteString(RelTableName)
	builder.WriteString(" (\n")
	for _, pair := range RelationPairs {
		builder.WriteString("  FROM ")
		builder.WriteString(formatIdent(pair.From))
		builder.WriteString(" TO ")
		builder.WriteString(formatIdent(pair.To))
		builder.WriteString(",\n")
	}
	for index, col := range relationColumns() {
		builder.WriteString("  ")
		builder.WriteString(col.Name)
		builder.WriteString(" ")
		builder.WriteString(col.Type)
		if index < len(relationColumns())-1 {
			builder.WriteString(",")
		}
		builder.WriteString("\n")
	}
	builder.WriteString(")")
	return builder.String()
}

func EmbeddingSchema(dims int) (string, error) {
	if dims <= 0 {
		return "", fmt.Errorf("embedding dimensions must be positive, got %d", dims)
	}
	return nodeTable(EmbeddingTableName, []column{{"id", "STRING"}, {"nodeId", "STRING"}, {"chunkIndex", "INT32"}, {"startLine", "INT64"}, {"endLine", "INT64"}, {"embedding", fmt.Sprintf("FLOAT[%d]", dims)}, {"contentHash", "STRING"}}), nil
}

func SchemaQueries(dims int) ([]string, error) {
	embeddingSchema, err := EmbeddingSchema(dims)
	if err != nil {
		return nil, err
	}
	queries := NodeSchemaQueries()
	queries = append(queries, RelationSchema(), embeddingSchema)
	return queries, nil
}

func CreateVectorIndexQuery() string {
	return fmt.Sprintf("CALL CREATE_VECTOR_INDEX('%s', '%s', 'embedding', metric := 'cosine')", EmbeddingTableName, EmbeddingIndexName)
}

func FTSIndexQueries() []string {
	return []string{
		"CALL CREATE_FTS_INDEX('File', 'file_fts', ['name', 'content'])",
		"CALL CREATE_FTS_INDEX('Function', 'function_fts', ['name', 'content'])",
		"CALL CREATE_FTS_INDEX('Class', 'class_fts', ['name', 'content'])",
		"CALL CREATE_FTS_INDEX('Method', 'method_fts', ['name', 'content'])",
		"CALL CREATE_FTS_INDEX('Interface', 'interface_fts', ['name', 'content'])",
	}
}

func FormatIdent(name string) string {
	return formatIdent(name)
}

type column struct {
	Name string
	Type string
}

func relationColumns() []column {
	return []column{{"type", "STRING"}, {"confidence", "DOUBLE"}, {"reason", "STRING"}, {"step", "INT32"}, {"resolutionSource", "STRING"}, {"evidence", "STRING"}, {"fileHash", "STRING"}}
}

func nodeTable(name string, columns []column) string {
	var builder strings.Builder
	builder.WriteString("CREATE NODE TABLE ")
	builder.WriteString(formatIdent(name))
	builder.WriteString(" (\n")
	for _, col := range columns {
		builder.WriteString("  ")
		builder.WriteString(col.Name)
		builder.WriteString(" ")
		builder.WriteString(col.Type)
		builder.WriteString(",\n")
	}
	builder.WriteString("  PRIMARY KEY (id)\n")
	builder.WriteString(")")
	return builder.String()
}

func formatIdent(name string) string {
	if quoteTable[name] {
		return "`" + name + "`"
	}
	return name
}

var quoteTable = map[string]bool{
	"Struct": true, "Enum": true, "Macro": true, "Typedef": true, "Union": true,
	"Namespace": true, "Trait": true, "Impl": true, "TypeAlias": true, "Const": true,
	"Static": true, "Variable": true, "Property": true, "Record": true, "Delegate": true,
	"Annotation": true, "Constructor": true, "Template": true, "Module": true,
}
