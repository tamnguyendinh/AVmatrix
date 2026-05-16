package repo

import (
	"encoding/json"
	"testing"
)

func TestRegistryEntryJSONShapeMatchesFrozenContract(t *testing.T) {
	statsCount := 1
	entry := RegistryEntry{
		Name:        "sample",
		Path:        "C:/sample",
		StoragePath: "C:/sample/.avmatrix",
		IndexedAt:   "2026-05-08T00:00:00Z",
		LastCommit:  "abc123",
		Stats:       &Stats{Files: &statsCount},
	}

	raw, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	for _, key := range []string{"name", "path", "storagePath", "indexedAt", "lastCommit", "stats"} {
		if _, ok := got[key]; !ok {
			t.Fatalf("registry entry JSON missing %q: %s", key, raw)
		}
	}
}

func TestMetaJSONShapeMatchesFrozenContract(t *testing.T) {
	statsCount := 1
	meta := Meta{
		RepoPath:   "C:/sample",
		LastCommit: "abc123",
		IndexedAt:  "2026-05-08T00:00:00Z",
		Stats:      &Stats{Nodes: &statsCount},
	}

	raw, err := json.Marshal(meta)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	for _, key := range []string{"repoPath", "lastCommit", "indexedAt", "stats"} {
		if _, ok := got[key]; !ok {
			t.Fatalf("meta JSON missing %q: %s", key, raw)
		}
	}
}
