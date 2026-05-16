package lbugschema

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestSchemaConstantsMatchFrozenContract(t *testing.T) {
	contract := readFrozenContract(t)

	if !reflect.DeepEqual(NodeTables, contract.LadybugDB.NodeTables) {
		t.Fatalf("NodeTables drift\nwant %#v\ngot  %#v", contract.LadybugDB.NodeTables, NodeTables)
	}
	if !reflect.DeepEqual(RelationshipTypes, contract.SchemaConstants.RelationshipTypes) {
		t.Fatalf("RelationshipTypes drift\nwant %#v\ngot  %#v", contract.SchemaConstants.RelationshipTypes, RelationshipTypes)
	}
	if RelTableName != contract.LadybugDB.RelationshipTable.Name {
		t.Fatalf("RelTableName = %q, want %q", RelTableName, contract.LadybugDB.RelationshipTable.Name)
	}

	gotColumns := map[string]string{}
	for _, col := range relationColumns() {
		gotColumns[col.Name] = col.Type
	}
	if !reflect.DeepEqual(gotColumns, contract.LadybugDB.RelationshipTable.Columns) {
		t.Fatalf("relation columns drift\nwant %#v\ngot  %#v", contract.LadybugDB.RelationshipTable.Columns, gotColumns)
	}
}

func TestSchemaQueriesPreserveDDLShape(t *testing.T) {
	queries, err := SchemaQueries(DefaultEmbeddingDims)
	if err != nil {
		t.Fatalf("SchemaQueries() error = %v", err)
	}
	if len(queries) != len(NodeTables)+2 {
		t.Fatalf("SchemaQueries() len = %d, want %d", len(queries), len(NodeTables)+2)
	}
	if len(RelationPairs) != 238 {
		t.Fatalf("RelationPairs len = %d, want 238", len(RelationPairs))
	}

	fileSchema := NodeSchema("File")
	for _, want := range []string{"CREATE NODE TABLE File", "content STRING", "PRIMARY KEY (id)"} {
		if !strings.Contains(fileSchema, want) {
			t.Fatalf("File schema missing %q:\n%s", want, fileSchema)
		}
	}
	methodSchema := NodeSchema("Method")
	for _, want := range []string{"parameterCount INT32", "returnType STRING"} {
		if !strings.Contains(methodSchema, want) {
			t.Fatalf("Method schema missing %q:\n%s", want, methodSchema)
		}
	}
	if !strings.Contains(NodeSchema("Struct"), "CREATE NODE TABLE `Struct`") {
		t.Fatalf("Struct schema must quote the table name:\n%s", NodeSchema("Struct"))
	}

	relationSchema := RelationSchema()
	for _, want := range []string{
		"CREATE REL TABLE CodeRelation",
		"FROM File TO File",
		"FROM File TO Package",
		"FROM File TO `Struct`",
		"FROM `Variable` TO Function",
		"FROM `Const` TO Function",
		"FROM `Constructor` TO `Property`",
		"FROM CodeElement TO CodeElement",
		"FROM CodeElement TO `Module`",
		"FROM `Module` TO `Namespace`",
		"FROM `Namespace` TO Function",
		"FROM `TypeAlias` TO Method",
		"type STRING",
		"confidence DOUBLE",
		"resolutionSource STRING",
		"evidence STRING",
		"fileHash STRING",
	} {
		if !strings.Contains(relationSchema, want) {
			t.Fatalf("relation schema missing %q", want)
		}
	}
}

func TestSchemaSurfaceCoversLegacyCoreAndModernNodeTypes(t *testing.T) {
	for _, table := range []string{
		"File", "Folder", "Function", "Class", "Interface", "Method", "CodeElement", "Community", "Process",
		"Package", "Section", "Struct", "Enum", "Macro", "Typedef", "Union", "Namespace", "Trait", "Impl",
		"TypeAlias", "Const", "Static", "Variable", "Property", "Record", "Delegate", "Annotation", "Constructor",
		"Template", "Module", "Route", "Tool",
	} {
		if !containsString(NodeTables, table) {
			t.Fatalf("NodeTables missing %q", table)
		}
	}
	if len(NodeTables) != 32 {
		t.Fatalf("NodeTables length = %d, want 32", len(NodeTables))
	}

	for _, relationType := range []string{
		"CONTAINS", "DEFINES", "IMPORTS", "CALLS", "USES", "INHERITS", "EXTENDS", "IMPLEMENTS",
		"HAS_METHOD", "HAS_PROPERTY", "ACCESSES", "METHOD_OVERRIDES", "OVERRIDES", "METHOD_IMPLEMENTS",
		"MEMBER_OF", "STEP_IN_PROCESS", "HANDLES_ROUTE", "FETCHES", "HANDLES_TOOL", "ENTRY_POINT_OF",
		"WRAPS", "QUERIES",
	} {
		if !containsString(RelationshipTypes, relationType) {
			t.Fatalf("RelationshipTypes missing %q", relationType)
		}
	}

	for _, want := range []string{"startLine INT64", "endLine INT64", "isExported BOOLEAN"} {
		if !strings.Contains(NodeSchema("Function"), want) {
			t.Fatalf("Function schema missing %q:\n%s", want, NodeSchema("Function"))
		}
	}
	for _, want := range []string{"heuristicLabel STRING", "cohesion DOUBLE"} {
		if !strings.Contains(NodeSchema("Community"), want) {
			t.Fatalf("Community schema missing %q:\n%s", want, NodeSchema("Community"))
		}
	}
	for _, want := range []string{"processType STRING", "stepCount INT32"} {
		if !strings.Contains(NodeSchema("Process"), want) {
			t.Fatalf("Process schema missing %q:\n%s", want, NodeSchema("Process"))
		}
	}

	relationSchema := RelationSchema()
	for _, want := range []string{
		"FROM Class TO Method",
		"FROM Class TO `Constructor`",
		"FROM `Struct` TO Method",
		"FROM `Trait` TO `Constructor`",
		"FROM `Impl` TO `Property`",
		"FROM `Record` TO `Property`",
		"FROM `Property` TO Method",
	} {
		if !strings.Contains(relationSchema, want) {
			t.Fatalf("relation schema missing %q", want)
		}
	}

	queries, err := SchemaQueries(DefaultEmbeddingDims)
	if err != nil {
		t.Fatalf("SchemaQueries() error = %v", err)
	}
	if got := len(queries); got != len(NodeTables)+2 {
		t.Fatalf("SchemaQueries() length = %d, want %d", got, len(NodeTables)+2)
	}
	joinedNodes := strings.Join(NodeSchemaQueries(), "\n")
	relationIndex := indexOfQuery(queries, RelationSchema())
	if relationIndex <= 0 || !strings.Contains(joinedNodes, queries[relationIndex-1]) {
		t.Fatalf("relation schema must come after node schemas")
	}
}

func TestEmbeddingAndIndexQueries(t *testing.T) {
	if _, err := EmbeddingSchema(0); err == nil {
		t.Fatalf("EmbeddingSchema(0) expected error")
	}
	schema, err := EmbeddingSchema(768)
	if err != nil {
		t.Fatalf("EmbeddingSchema(768) error = %v", err)
	}
	for _, want := range []string{"CREATE NODE TABLE CodeEmbedding", "embedding FLOAT[768]", "contentHash STRING"} {
		if !strings.Contains(schema, want) {
			t.Fatalf("embedding schema missing %q:\n%s", want, schema)
		}
	}
	if got := CreateVectorIndexQuery(); got != "CALL CREATE_VECTOR_INDEX('CodeEmbedding', 'code_embedding_idx', 'embedding', metric := 'cosine')" {
		t.Fatalf("CreateVectorIndexQuery() = %q", got)
	}
	if got := FTSIndexQueries(); len(got) != 5 || !strings.Contains(got[0], "file_fts") {
		t.Fatalf("unexpected FTSIndexQueries(): %#v", got)
	}
}

type frozenContract struct {
	LadybugDB struct {
		NodeTables        []string `json:"nodeTables"`
		RelationshipTable struct {
			Name    string            `json:"name"`
			Columns map[string]string `json:"columns"`
		} `json:"relationshipTable"`
	} `json:"ladybugdb"`
	SchemaConstants struct {
		RelationshipTypes []string `json:"relationshipTypes"`
	} `json:"schemaConstants"`
}

func readFrozenContract(t *testing.T) frozenContract {
	t.Helper()
	path := filepath.FromSlash("../../baseline/phase-1-contract-freeze/ladybugdb-graph-contract.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read frozen contract: %v", err)
	}
	var contract frozenContract
	if err := json.Unmarshal(raw, &contract); err != nil {
		t.Fatalf("decode frozen contract: %v", err)
	}
	return contract
}

func indexOfQuery(queries []string, query string) int {
	for index, candidate := range queries {
		if candidate == query {
			return index
		}
	}
	return -1
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
