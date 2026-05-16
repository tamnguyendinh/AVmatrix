package group

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestContractsFiltersRegistryAndUnmatched(t *testing.T) {
	homeDir := t.TempDir()
	groupDir := filepath.Join(homeDir, "groups", "fixture")
	if err := os.MkdirAll(groupDir, 0o755); err != nil {
		t.Fatalf("mkdir group: %v", err)
	}
	writeGroupTestConfig(t, groupDir)
	writeGroupTestContracts(t, groupDir)

	httpContracts, err := Contracts(homeDir, "fixture", ContractsOptions{Type: "http"})
	if err != nil {
		t.Fatalf("Contracts(type=http) error = %v", err)
	}
	if len(httpContracts.Contracts) != 2 || len(httpContracts.CrossLinks) != 1 {
		t.Fatalf("http contracts = %#v", httpContracts)
	}

	unmatched, err := Contracts(homeDir, "fixture", ContractsOptions{UnmatchedOnly: true})
	if err != nil {
		t.Fatalf("Contracts(unmatched) error = %v", err)
	}
	if len(unmatched.Contracts) != 1 || unmatched.Contracts[0].ContractID != "topic::orphan" {
		t.Fatalf("unmatched contracts = %#v", unmatched.Contracts)
	}
}

func writeGroupTestConfig(t *testing.T, groupDir string) {
	t.Helper()
	groupYAML := `version: 1
name: fixture
description: "fixture group"

repos:
  app/backend: backend
  app/frontend: frontend

links: []
`
	if err := os.WriteFile(filepath.Join(groupDir, "group.yaml"), []byte(groupYAML), 0o644); err != nil {
		t.Fatalf("write group.yaml: %v", err)
	}
}

func writeGroupTestContracts(t *testing.T, groupDir string) {
	t.Helper()
	registry := ContractRegistry{
		Version:     1,
		GeneratedAt: "2026-05-12T01:00:00Z",
		RepoSnapshots: map[string]RepoSnapshot{
			"app/backend": {IndexedAt: "2026-05-12T00:00:00Z", LastCommit: "abc123"},
		},
		MissingRepos: []string{"app/frontend"},
		Contracts: []StoredContract{
			{
				Repo:       "app/backend",
				ContractID: "http::GET::/users",
				Type:       "http",
				Role:       "provider",
				SymbolUID:  "Route:/api/users",
				SymbolRef:  SymbolRef{FilePath: "src/server.ts", Name: "/api/users"},
				SymbolName: "/api/users",
				Confidence: 1,
				Meta:       map[string]any{"source": "fixture"},
			},
			{
				Repo:       "app/frontend",
				ContractID: "http::GET::/users",
				Type:       "http",
				Role:       "consumer",
				SymbolUID:  "File:src/app.ts",
				SymbolRef:  SymbolRef{FilePath: "src/app.ts", Name: "fetchUsers"},
				SymbolName: "fetchUsers",
				Confidence: 0.95,
				Meta:       map[string]any{"source": "fixture"},
			},
			{
				Repo:       "app/backend",
				ContractID: "topic::orphan",
				Type:       "topic",
				Role:       "provider",
				SymbolUID:  "Function:helper",
				SymbolRef:  SymbolRef{FilePath: "src/helper.ts", Name: "helper"},
				SymbolName: "helper",
				Confidence: 0.75,
				Meta:       map[string]any{"source": "fixture"},
			},
		},
		CrossLinks: []CrossLink{
			{
				From:       CrossLinkEndpoint{Repo: "app/frontend", SymbolUID: "File:src/app.ts", SymbolRef: SymbolRef{FilePath: "src/app.ts", Name: "fetchUsers"}},
				To:         CrossLinkEndpoint{Repo: "app/backend", SymbolUID: "Route:/api/users", SymbolRef: SymbolRef{FilePath: "src/server.ts", Name: "/api/users"}},
				Type:       "http",
				ContractID: "http::GET::/users",
				MatchType:  "exact",
				Confidence: 1,
			},
		},
	}
	raw, err := json.MarshalIndent(registry, "", "  ")
	if err != nil {
		t.Fatalf("marshal contracts: %v", err)
	}
	if err := os.WriteFile(filepath.Join(groupDir, "contracts.json"), append(raw, '\n'), 0o644); err != nil {
		t.Fatalf("write contracts.json: %v", err)
	}
}
