package lbugload

import "fmt"

type QueryRunner interface {
	Query(query string) error
}

type loadTransactionRunner interface {
	BeginLoadTransaction() error
	CommitLoadTransaction() error
	RollbackLoadTransaction() error
}

type LoadResult struct {
	NodeCopyCount          int
	RelationshipCopyCount  int
	FallbackInsertCount    int
	FallbackInsertFailures int
	SkippedRelationships   int
	Warnings               []string
}

type LoadOptions struct {
	AllowRelationshipFallback     bool
	AllowCopyRetryWithIgnoreError bool
	AllowSkippedRelationships     bool
}

func LoadCSVExport(runner QueryRunner, export *CSVExport) (LoadResult, error) {
	return LoadCSVExportWithOptions(runner, export, LoadOptions{})
}

func LoadCSVExportWithOptions(runner QueryRunner, export *CSVExport, options LoadOptions) (LoadResult, error) {
	var result LoadResult
	if runner == nil {
		return result, fmt.Errorf("query runner is nil")
	}
	if export == nil {
		return result, fmt.Errorf("csv export is nil")
	}

	result.SkippedRelationships = export.Metrics.SkippedRelationships
	if result.SkippedRelationships > 0 && !options.AllowSkippedRelationships {
		return result, fmt.Errorf("db load skipped %d relationships; refusing incomplete graph load", result.SkippedRelationships)
	}

	tx, err := beginLoadTransaction(runner)
	if err != nil {
		return result, err
	}

	for _, nodeFile := range export.NodeFiles {
		query, err := NodeCopyQuery(nodeFile.Table, nodeFile.CSVPath)
		if err != nil {
			if tx != nil {
				return result, rollbackLoadTransaction(tx, err)
			}
			return result, err
		}
		result.NodeCopyCount++
		if err := runCopy(runner, query, options.AllowCopyRetryWithIgnoreError); err != nil {
			err := fmt.Errorf("copy nodes %s: %w", nodeFile.Table, err)
			if tx != nil {
				return result, rollbackLoadTransaction(tx, err)
			}
			return result, err
		}
	}

	for _, pairFile := range export.RelationshipPairFiles {
		if pairFile.CopySupported {
			query := RelationshipCopyQuery(pairFile.From, pairFile.To, pairFile.CSVPath)
			result.RelationshipCopyCount++
			if err := runCopy(runner, query, options.AllowCopyRetryWithIgnoreError); err == nil {
				continue
			} else {
				if !options.AllowRelationshipFallback {
					err := fmt.Errorf("copy relationships %s->%s: %w", pairFile.From, pairFile.To, err)
					if tx != nil {
						return result, rollbackLoadTransaction(tx, err)
					}
					return result, err
				}
				result.Warnings = append(result.Warnings, fmt.Sprintf("%s->%s COPY failed, using fallback: %v", pairFile.From, pairFile.To, err))
			}
		} else {
			if !options.AllowRelationshipFallback {
				err := fmt.Errorf("copy relationships %s->%s: schema pair unsupported", pairFile.From, pairFile.To)
				if tx != nil {
					return result, rollbackLoadTransaction(tx, err)
				}
				return result, err
			}
			result.Warnings = append(result.Warnings, fmt.Sprintf("%s->%s missing schema pair, using fallback", pairFile.From, pairFile.To))
		}

		inserted, failed, err := fallbackRelationshipFile(runner, pairFile)
		result.FallbackInsertCount += inserted
		result.FallbackInsertFailures += failed
		if err != nil {
			if tx != nil {
				return result, rollbackLoadTransaction(tx, err)
			}
			return result, err
		}
	}
	if tx != nil {
		if err := tx.CommitLoadTransaction(); err != nil {
			return result, rollbackLoadTransaction(tx, fmt.Errorf("commit load transaction: %w", err))
		}
	}
	return result, nil
}

func beginLoadTransaction(runner QueryRunner) (loadTransactionRunner, error) {
	tx, ok := runner.(loadTransactionRunner)
	if !ok {
		return nil, nil
	}
	if err := tx.BeginLoadTransaction(); err != nil {
		return nil, fmt.Errorf("begin load transaction: %w", err)
	}
	return tx, nil
}

func rollbackLoadTransaction(tx loadTransactionRunner, cause error) error {
	if rollbackErr := tx.RollbackLoadTransaction(); rollbackErr != nil {
		return fmt.Errorf("%w (rollback failed: %v)", cause, rollbackErr)
	}
	return cause
}

func runCopy(runner QueryRunner, query string, allowRetryWithIgnoreErrors bool) error {
	if err := runner.Query(query); err == nil {
		return nil
	} else if !allowRetryWithIgnoreErrors {
		return err
	}
	retryQuery := RetryCopyQuery(query)
	if retryQuery == query {
		return fmt.Errorf("copy failed")
	}
	if err := runner.Query(retryQuery); err != nil {
		return err
	}
	return nil
}

func fallbackRelationshipFile(runner QueryRunner, pairFile RelationshipPairCSV) (int, int, error) {
	rows, err := ReadRelationshipCSVRows(pairFile.CSVPath)
	if err != nil {
		return 0, 0, err
	}
	inserted := 0
	failed := 0
	var firstFailure error
	for _, row := range rows {
		query := FallbackRelationshipInsertQuery(row, pairFile.From, pairFile.To)
		if err := runner.Query(query); err != nil {
			failed++
			if firstFailure == nil {
				firstFailure = err
			}
			continue
		}
		inserted++
	}
	if failed > 0 {
		return inserted, failed, fmt.Errorf("fallback relationship insert failed for %d of %d rows: %w", failed, len(rows), firstFailure)
	}
	return inserted, failed, nil
}
