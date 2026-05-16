//go:build ladybugdb

package lbugnative

import (
	"path/filepath"
	"testing"

	"github.com/tamnguyendinh/avmatrix-go/internal/graph"
	"github.com/tamnguyendinh/avmatrix-go/internal/lbugload"
	"github.com/tamnguyendinh/avmatrix-go/internal/lbugruntime"
	"github.com/tamnguyendinh/avmatrix-go/internal/lbugschema"
	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

func TestNativeLadybugPersistenceReadbackAndStream(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "avmatrix-test.lbug")

	db, err := openNativeDatabase(dbPath, false)
	if err != nil {
		t.Fatalf("open writable db: %v", err)
	}
	conn, err := db.OpenConnection()
	if err != nil {
		db.Close()
		t.Fatalf("open writable connection: %v", err)
	}

	writeRunner := nativeWriteRunner{conn: conn}
	for _, query := range schemaQueries(t) {
		if err := writeRunner.Query(query); err != nil {
			conn.Close()
			db.Close()
			t.Fatalf("schema query failed:\n%s\n%v", query, err)
		}
	}

	export, err := lbugload.ExportGraphCSVs(nativeFixtureGraph(), filepath.Join(t.TempDir(), "csv"))
	if err != nil {
		conn.Close()
		db.Close()
		t.Fatalf("ExportGraphCSVs() error = %v", err)
	}
	loadResult, err := lbugload.LoadCSVExport(writeRunner, export)
	if err != nil {
		conn.Close()
		db.Close()
		t.Fatalf("LoadCSVExport() error = %v", err)
	}
	if loadResult.NodeCopyCount != 2 || loadResult.RelationshipCopyCount != 1 || loadResult.FallbackInsertCount != 0 {
		conn.Close()
		db.Close()
		t.Fatalf("unexpected load result: %#v", loadResult)
	}
	conn.Close()
	db.Close()

	readDB, err := openNativeDatabase(dbPath, true)
	if err != nil {
		t.Fatalf("reopen read-only db: %v", err)
	}
	defer readDB.Close()
	readConn, err := readDB.OpenConnection()
	if err != nil {
		t.Fatalf("open read-only connection: %v", err)
	}
	defer readConn.Close()

	readRunner := nativeReadRunner{conn: readConn, silencer: &lbugruntime.StdioSilencer{}}
	rows, err := readRunner.Query("MATCH (f:File) RETURN f.id AS id, f.name AS name, f.content AS content")
	if err != nil {
		t.Fatalf("read file row: %v", err)
	}
	if len(rows) != 1 || rows[0]["id"] != "File:src/app.ts" || rows[0]["name"] != "app.ts" {
		t.Fatalf("unexpected file rows: %#v", rows)
	}
	functionRows, err := readRunner.Query("MATCH (n:Function) RETURN n.id AS id")
	if err != nil {
		t.Fatalf("read function rows: %v", err)
	}
	if len(functionRows) != 1 || functionRows[0]["id"] != "Function:doWork" {
		t.Fatalf("unexpected function rows: %#v", functionRows)
	}
	missingRows, err := readRunner.Query("MATCH (n:Function) WHERE n.id = '__nonexistent_id__' RETURN n.id AS id")
	if err != nil {
		t.Fatalf("read missing function rows: %v", err)
	}
	if len(missingRows) != 0 {
		t.Fatalf("missing function rows = %#v, want empty", missingRows)
	}
	if _, err := readRunner.Query("MATCH RETURN 1"); err == nil {
		t.Fatalf("malformed Cypher query error = nil, want error")
	}
	if _, err := readRunner.Query("MATCH (n:GhostTable) RETURN n"); err == nil {
		t.Fatalf("non-existent node label query error = nil, want error")
	}
	if _, err := readRunner.Query("CREATE (n:File {id: 'new'})"); err == nil {
		t.Fatalf("read runner write query error = nil, want read-only rejection")
	}

	var streamed []lbugruntime.Row
	count, err := readRunner.Stream("MATCH (a:File)-[r:CodeRelation]->(b:Function) RETURN a.id AS fromId, b.id AS toId, r.type AS type, r.fileHash AS fileHash", func(row lbugruntime.Row) error {
		streamed = append(streamed, row)
		return nil
	})
	if err != nil {
		t.Fatalf("stream relationship rows: %v", err)
	}
	if count != 1 || streamed[0]["fromId"] != "File:src/app.ts" || streamed[0]["toId"] != "Function:doWork" || streamed[0]["type"] != "DEFINES" || streamed[0]["fileHash"] != "hash-native" {
		t.Fatalf("unexpected streamed rows: count=%d rows=%#v", count, streamed)
	}
}

type nativeWriteRunner struct {
	conn *nativeConnection
}

func (r nativeWriteRunner) Query(query string) error {
	result, err := r.conn.Query(query)
	if result != nil {
		result.Close()
	}
	return err
}

type nativeReadRunner struct {
	conn     *nativeConnection
	silencer *lbugruntime.StdioSilencer
}

func (r nativeReadRunner) Query(query string) ([]lbugruntime.Row, error) {
	if err := lbugruntime.ValidateReadQuery(query); err != nil {
		return nil, err
	}
	var rows []lbugruntime.Row
	err := r.silencer.Run(func() error {
		result, err := r.conn.Query(query)
		if err != nil {
			return err
		}
		defer result.Close()
		rows, err = result.Rows()
		return err
	})
	return rows, err
}

func (r nativeReadRunner) Stream(query string, onRow func(lbugruntime.Row) error) (int, error) {
	if err := lbugruntime.ValidateReadQuery(query); err != nil {
		return 0, err
	}
	count := 0
	err := r.silencer.Run(func() error {
		result, err := r.conn.Query(query)
		if err != nil {
			return err
		}
		defer result.Close()
		rows, err := result.Rows()
		if err != nil {
			return err
		}
		for _, row := range rows {
			if err := onRow(row); err != nil {
				return err
			}
			count++
		}
		return nil
	})
	return count, err
}

func schemaQueries(t *testing.T) []string {
	t.Helper()
	queries, err := lbugschema.SchemaQueries(lbugschema.DefaultEmbeddingDims)
	if err != nil {
		t.Fatalf("SchemaQueries() error = %v", err)
	}
	return queries
}

func nativeFixtureGraph() *graph.Graph {
	step := 1
	g := graph.New()
	g.AddNode(graph.Node{ID: "File:src/app.ts", Label: scopeir.NodeFile, Properties: graph.NodeProperties{
		"name": "app.ts", "filePath": "src/app.ts", "content": "export function doWork() {}",
	}})
	g.AddNode(graph.Node{ID: "Function:doWork", Label: scopeir.NodeFunction, Properties: graph.NodeProperties{
		"name": "doWork", "filePath": "src/app.ts", "startLine": 1, "endLine": 1, "isExported": true, "content": "function doWork() {}",
	}})
	g.AddRelationship(graph.Relationship{
		ID:               "rel:file-function",
		SourceID:         "File:src/app.ts",
		TargetID:         "Function:doWork",
		Type:             graph.RelDefines,
		Confidence:       1,
		Reason:           "native persistence fixture",
		Step:             &step,
		ResolutionSource: "native-test",
		FileHash:         "hash-native",
	})
	return g
}
