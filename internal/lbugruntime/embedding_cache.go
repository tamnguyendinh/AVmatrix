package lbugruntime

import (
	"fmt"

	"github.com/tamnguyendinh/avmatrix-go/internal/lbugschema"
)

type Row map[string]any

type QueryRows func(query string) ([]Row, error)

type EmbeddingHashResult struct {
	Hashes map[string]string
	Found  bool
	Legacy bool
}

func FetchExistingEmbeddingHashes(exec QueryRows) (EmbeddingHashResult, error) {
	if exec == nil {
		return EmbeddingHashResult{}, fmt.Errorf("query executor is nil")
	}

	rows, err := exec(fmt.Sprintf(
		"MATCH (e:%s) RETURN e.nodeId AS nodeId, e.chunkIndex AS chunkIndex, e.startLine AS startLine, e.endLine AS endLine, e.contentHash AS contentHash",
		lbugschema.EmbeddingTableName,
	))
	if err == nil {
		if len(rows) == 0 {
			return EmbeddingHashResult{}, nil
		}
		hashes := make(map[string]string, len(rows))
		for _, row := range rows {
			nodeID := stringValue(row["nodeId"])
			if nodeID == "" {
				continue
			}
			hash := stringValue(row["contentHash"])
			if hash == "" || row["chunkIndex"] == nil || row["startLine"] == nil || row["endLine"] == nil {
				hash = lbugschema.StaleHashSentinel
			}
			hashes[nodeID] = hash
		}
		return EmbeddingHashResult{Hashes: hashes, Found: len(hashes) > 0}, nil
	}
	if !IsMissingColumnOrTableError(err) {
		return EmbeddingHashResult{}, err
	}

	rows, fallbackErr := exec(fmt.Sprintf(
		"MATCH (e:%s) RETURN e.nodeId AS nodeId",
		lbugschema.EmbeddingTableName,
	))
	if fallbackErr != nil {
		if IsMissingColumnOrTableError(fallbackErr) {
			return EmbeddingHashResult{}, nil
		}
		return EmbeddingHashResult{}, fallbackErr
	}
	if len(rows) == 0 {
		return EmbeddingHashResult{}, nil
	}

	hashes := make(map[string]string, len(rows))
	for _, row := range rows {
		nodeID := stringValue(row["nodeId"])
		if nodeID != "" {
			hashes[nodeID] = lbugschema.StaleHashSentinel
		}
	}
	return EmbeddingHashResult{Hashes: hashes, Found: len(hashes) > 0, Legacy: len(hashes) > 0}, nil
}

func stringValue(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	default:
		return fmt.Sprint(typed)
	}
}
