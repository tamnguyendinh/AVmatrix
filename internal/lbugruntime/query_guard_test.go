package lbugruntime

import (
	"strings"
	"testing"
)

func TestIsWriteQueryMatchesPoolAdapterContract(t *testing.T) {
	tests := []struct {
		name  string
		query string
		want  bool
	}{
		{name: "match return", query: "MATCH (n:File) RETURN n LIMIT 1", want: false},
		{name: "where return", query: `MATCH (n:Function) WHERE n.name = "foo" RETURN n`, want: false},
		{name: "relationship return", query: "MATCH (a)-[r]->(b) RETURN a, r, b", want: false},
		{name: "optional match", query: "OPTIONAL MATCH (n)-[r]->(m) RETURN n, r, m", want: false},
		{name: "with clause", query: "MATCH (n) WITH n RETURN n.name", want: false},
		{name: "unwind clause", query: "UNWIND [1,2,3] AS x RETURN x", want: false},
		{name: "count return", query: "MATCH (n) RETURN count(n)", want: false},
		{name: "contains test as data", query: `MATCH (n:Function) WHERE n.filePath CONTAINS "test" RETURN n`, want: false},
		{name: "fts call allowed", query: "CALL QUERY_FTS_INDEX('File', 'file_fts', 'repo') RETURN node", want: false},
		{name: "vector call allowed", query: "CALL QUERY_VECTOR_INDEX('CodeEmbedding', 'idx', [0.1]) RETURN node", want: false},
		{name: "label named create is not write", query: "MATCH (n:CREATE) RETURN n", want: false},
		{name: "label containing create is not write", query: "MATCH (n:CreateHelpers) RETURN n", want: false},
		{name: "relationship type calls is not write", query: "MATCH (a)-[:CALLS]->(b) RETURN a, b", want: false},
		{name: "defines relationship is not write", query: "MATCH (f:File)-[r:DEFINES]->(n) RETURN n", want: false},
		{name: "write keyword in data is not write", query: "MATCH (n) WHERE n.name = 'MERGEHelper' RETURN n", want: false},
		{name: "write keyword after colon in data is not write", query: `MATCH (n) WHERE n.content CONTAINS ":CREATE" RETURN n`, want: false},
		{name: "label containing set is not write", query: "MATCH (n:SomethingWithSET) RETURN n", want: false},
		{name: "partial write keyword is not write", query: "MATCH (n) WHERE n.name = 'CREATED' RETURN n", want: false},
		{name: "empty query is not write", query: "", want: false},
		{name: "whitespace query is not write", query: "   ", want: false},
		{name: "create blocked", query: "CREATE (n:File {id: 'x'})", want: true},
		{name: "lowercase create blocked", query: `create (n:Function {id: "x"})`, want: true},
		{name: "set blocked", query: "MATCH (n) SET n.x = 1", want: true},
		{name: "lowercase set blocked", query: `set n.name = "x"`, want: true},
		{name: "merge blocked", query: "MERGE (n:Foo {id: 1})", want: true},
		{name: "remove blocked", query: "MATCH (n) REMOVE n.oldProp", want: true},
		{name: "delete blocked", query: "DELETE n", want: true},
		{name: "lowercase delete blocked", query: "delete n", want: true},
		{name: "drop index blocked", query: "DROP INDEX ON :Foo(prop)", want: true},
		{name: "alter table blocked", query: "ALTER TABLE Something", want: true},
		{name: "copy blocked", query: "COPY File FROM 'file.csv'", want: true},
		{name: "copy to blocked", query: "COPY TO something", want: true},
		{name: "load extension blocked", query: "LOAD EXTENSION fts", want: true},
		{name: "install extension blocked", query: "INSTALL VECTOR", want: true},
		{name: "detach delete blocked", query: "MATCH (n) DETACH DELETE n", want: true},
		{name: "multiline delete blocked", query: "MATCH (n)\nDELETE n", want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsWriteQuery(tt.query); got != tt.want {
				t.Fatalf("IsWriteQuery(%q) = %v, want %v", tt.query, got, tt.want)
			}
		})
	}
}

func TestIsWriteQueryMatchesSecurityKeywordContract(t *testing.T) {
	for _, keyword := range []string{"CREATE", "DELETE", "SET", "MERGE", "REMOVE", "DROP", "ALTER", "COPY", "DETACH"} {
		mixed := keyword[:1] + strings.ToLower(keyword[1:])
		for _, query := range []string{keyword + " (n:Node)", strings.ToLower(keyword) + " (n:Node)", mixed + " (n:Node)"} {
			if !IsWriteQuery(query) {
				t.Fatalf("IsWriteQuery(%q) = false, want true", query)
			}
		}
	}
}

func TestIsWriteQueryConsecutiveCallsDoNotCarryState(t *testing.T) {
	tests := []struct {
		query string
		want  bool
	}{
		{query: "CREATE (n)", want: true},
		{query: "MATCH (n) RETURN n", want: false},
		{query: "DROP TABLE foo", want: true},
		{query: "MATCH (n) RETURN n", want: false},
		{query: "SET n.x = 1", want: true},
	}
	for _, tt := range tests {
		if got := IsWriteQuery(tt.query); got != tt.want {
			t.Fatalf("IsWriteQuery(%q) = %v, want %v", tt.query, got, tt.want)
		}
	}
}

func TestValidateReadQueryRejectsWrites(t *testing.T) {
	if err := ValidateReadQuery("MATCH (n) RETURN n"); err != nil {
		t.Fatalf("ValidateReadQuery(read) error = %v", err)
	}
	if err := ValidateReadQuery("MERGE (n:File {id: 'x'})"); err == nil {
		t.Fatalf("ValidateReadQuery(write) expected error")
	}
}
