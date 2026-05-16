//go:build ladybugdb

package lbugnative

import (
	"fmt"

	"github.com/tamnguyendinh/avmatrix-go/internal/lbugruntime"
	"github.com/tamnguyendinh/avmatrix-go/internal/lbugschema"
)

type writeRunner struct {
	db   *nativeDatabase
	conn *nativeConnection
}

type readRunner struct {
	db   *nativeDatabase
	conn *nativeConnection
}

func OpenWriteRunner(path string) (WriteRunner, error) {
	return OpenWriteRunnerWithEmbeddingDims(path, lbugschema.DefaultEmbeddingDims)
}

func OpenWriteRunnerWithEmbeddingDims(path string, embeddingDims int) (WriteRunner, error) {
	var runner WriteRunner
	err := lbugruntime.RunWithWALRecovery(path, func() error {
		next, err := openWriteRunnerWithEmbeddingDimsOnce(path, embeddingDims)
		if err != nil {
			return err
		}
		runner = next
		return nil
	})
	return runner, err
}

func openWriteRunnerWithEmbeddingDimsOnce(path string, embeddingDims int) (WriteRunner, error) {
	if embeddingDims <= 0 {
		embeddingDims = lbugschema.DefaultEmbeddingDims
	}
	db, err := openNativeDatabase(path, false)
	if err != nil {
		return nil, err
	}
	conn, err := db.OpenConnection()
	if err != nil {
		db.Close()
		return nil, err
	}
	runner := &writeRunner{db: db, conn: conn}
	if err := runner.initializeSchema(embeddingDims); err != nil {
		_ = runner.Close()
		return nil, err
	}
	return runner, nil
}

func OpenReadRunner(path string) (ReadRunner, error) {
	db, err := openNativeDatabase(path, true)
	if err != nil {
		return nil, err
	}
	conn, err := db.OpenConnection()
	if err != nil {
		db.Close()
		return nil, err
	}
	return &readRunner{db: db, conn: conn}, nil
}

func (r *writeRunner) Query(query string) error {
	result, err := r.conn.Query(query)
	if result != nil {
		result.Close()
	}
	return err
}

func (r *writeRunner) BeginLoadTransaction() error {
	return r.Query("BEGIN TRANSACTION")
}

func (r *writeRunner) CommitLoadTransaction() error {
	return r.Query("COMMIT")
}

func (r *writeRunner) RollbackLoadTransaction() error {
	return r.Query("ROLLBACK")
}

func (r *writeRunner) QueryRows(query string) ([]lbugruntime.Row, error) {
	result, err := r.conn.Query(query)
	if err != nil {
		if result != nil {
			result.Close()
		}
		return nil, err
	}
	defer result.Close()
	return result.Rows()
}

func (r *writeRunner) Close() error {
	if r == nil {
		return nil
	}
	if r.conn != nil {
		r.conn.Close()
	}
	if r.db != nil {
		r.db.Close()
	}
	return nil
}

func (r *readRunner) QueryRows(query string) ([]lbugruntime.Row, error) {
	if err := lbugruntime.ValidateReadQuery(query); err != nil {
		return nil, err
	}
	result, err := r.conn.Query(query)
	if err != nil {
		if result != nil {
			result.Close()
		}
		return nil, err
	}
	defer result.Close()
	return result.Rows()
}

func (r *readRunner) Close() error {
	if r == nil {
		return nil
	}
	if r.conn != nil {
		r.conn.Close()
	}
	if r.db != nil {
		r.db.Close()
	}
	return nil
}

func (r *writeRunner) initializeSchema(embeddingDims int) error {
	queries, err := lbugschema.SchemaQueries(embeddingDims)
	if err != nil {
		return err
	}
	for _, query := range queries {
		if err := r.Query(query); err != nil {
			return fmt.Errorf("initialize native LadybugDB schema: %w", err)
		}
	}
	return nil
}
