package mcp

import (
	"os"
	"path/filepath"
	"testing"

	groupcore "github.com/tamnguyendinh/avmatrix-go/internal/group"
	"github.com/tamnguyendinh/avmatrix-go/internal/repo"
)

func TestGroupToolsListStatusContractsAndQueryErrors(t *testing.T) {
	store := repo.NewStore(t.TempDir())
	groupDir := writeMCPGroupToolsConfig(t, store.HomeDir, "test-group")
	server := NewServer(Config{Store: store})

	listResult, err := server.groupListTool(map[string]any{})
	if err != nil {
		t.Fatalf("groupListTool() error = %v", err)
	}
	if groups, ok := listResult["groups"].([]string); !ok || len(groups) != 1 || groups[0] != "test-group" {
		t.Fatalf("group list = %#v", listResult)
	}

	detail, err := server.groupListTool(map[string]any{"name": "test-group"})
	if err != nil {
		t.Fatalf("groupListTool(detail) error = %v", err)
	}
	repos, _ := detail["repos"].(map[string]string)
	if detail["name"] != "test-group" || repos["app/backend"] != "test-backend" {
		t.Fatalf("group detail = %#v", detail)
	}

	if result, err := server.groupStatusTool(map[string]any{}); err != nil || result["error"] != "name is required" {
		t.Fatalf("groupStatusTool(empty) = %#v, %v", result, err)
	}
	status, err := server.groupStatusTool(map[string]any{"name": "test-group"})
	if err != nil {
		t.Fatalf("groupStatusTool() error = %v", err)
	}
	statusRepos, _ := status["repos"].(map[string]groupcore.RepoStatus)
	if !statusRepos["app/backend"].Missing || !statusRepos["app/frontend"].Missing {
		t.Fatalf("group status = %#v", status)
	}

	if result, err := server.groupContractsTool(map[string]any{}); err != nil || result["error"] != "name is required" {
		t.Fatalf("groupContractsTool(empty) = %#v, %v", result, err)
	}
	noRegistry, err := server.groupContractsTool(map[string]any{"name": "test-group"})
	if err != nil {
		t.Fatalf("groupContractsTool(no registry) error = %v", err)
	}
	if got, _ := noRegistry["error"].(string); got == "" {
		t.Fatalf("groupContractsTool(no registry) = %#v", noRegistry)
	}

	writeMCPGroupToolsRegistry(t, store.HomeDir, "test-group")
	contracts, err := server.groupContractsTool(map[string]any{"name": "test-group", "type": "grpc"})
	if err != nil {
		t.Fatalf("groupContractsTool(type) error = %v", err)
	}
	filtered, _ := contracts["contracts"].([]groupcore.StoredContract)
	if len(filtered) != 1 || filtered[0].Type != "grpc" {
		t.Fatalf("filtered contracts = %#v", contracts)
	}
	unmatched, err := server.groupContractsTool(map[string]any{"name": "test-group", "unmatchedOnly": true})
	if err != nil {
		t.Fatalf("groupContractsTool(unmatched) error = %v", err)
	}
	unmatchedContracts, _ := unmatched["contracts"].([]groupcore.StoredContract)
	if len(unmatchedContracts) != 1 || unmatchedContracts[0].ContractID != "grpc::auth.AuthService/Login" {
		t.Fatalf("unmatched contracts = %#v", unmatched)
	}

	if result, err := server.groupSyncTool(map[string]any{}); err != nil || result["error"] != "name is required" {
		t.Fatalf("groupSyncTool(empty) = %#v, %v", result, err)
	}
	if result, err := server.groupQueryTool(map[string]any{}); err != nil || result["error"] != "name and query are required" {
		t.Fatalf("groupQueryTool(empty) = %#v, %v", result, err)
	}
	query, err := server.groupQueryTool(map[string]any{"name": "test-group", "query": "auth", "subgroup": "app/backend"})
	if err != nil {
		t.Fatalf("groupQueryTool(subgroup) error = %v", err)
	}
	perRepo, _ := query["per_repo"].([]groupcore.QueryRepoSummary)
	if len(perRepo) != 1 || perRepo[0].Repo != "app/backend" || perRepo[0].Count != 0 {
		t.Fatalf("group query per_repo = %#v", query)
	}

	if groupDir == "" {
		t.Fatal("group fixture not written")
	}
}

func writeMCPGroupToolsConfig(t *testing.T, homeDir string, name string) string {
	t.Helper()
	groupDir := filepath.Join(homeDir, "groups", name)
	groupYAML := `version: 1
name: test-group
description: Test
repos:
  app/backend: test-backend
  app/frontend: test-frontend
`
	if err := os.MkdirAll(groupDir, 0o755); err != nil {
		t.Fatalf("mkdir group: %v", err)
	}
	if err := os.WriteFile(filepath.Join(groupDir, "group.yaml"), []byte(groupYAML), 0o644); err != nil {
		t.Fatalf("write group.yaml: %v", err)
	}
	return groupDir
}

func writeMCPGroupToolsRegistry(t *testing.T, homeDir string, name string) {
	t.Helper()
	provider := groupcore.StoredContract{
		Repo:       "app/backend",
		ContractID: "http::GET::/api/users",
		Type:       "http",
		Role:       "provider",
		SymbolUID:  "uid-provider",
		SymbolRef:  groupcore.SymbolRef{FilePath: "src/routes.ts", Name: "getUsers"},
		SymbolName: "getUsers",
		Confidence: 1,
		Meta:       map[string]any{},
	}
	consumer := groupcore.StoredContract{
		Repo:       "app/frontend",
		ContractID: "http::GET::/api/users",
		Type:       "http",
		Role:       "consumer",
		SymbolUID:  "uid-consumer",
		SymbolRef:  groupcore.SymbolRef{FilePath: "src/api.ts", Name: "fetchUsers"},
		SymbolName: "fetchUsers",
		Confidence: 1,
		Meta:       map[string]any{},
	}
	grpc := groupcore.StoredContract{
		Repo:       "app/backend",
		ContractID: "grpc::auth.AuthService/Login",
		Type:       "grpc",
		Role:       "provider",
		SymbolUID:  "uid-grpc",
		SymbolRef:  groupcore.SymbolRef{FilePath: "src/auth.proto", Name: "Login"},
		SymbolName: "Login",
		Confidence: 1,
		Meta:       map[string]any{},
	}
	registry := groupcore.ContractRegistry{
		Version:     1,
		GeneratedAt: "2026-05-15T00:00:00Z",
		RepoSnapshots: map[string]groupcore.RepoSnapshot{
			"app/backend": {IndexedAt: "2026-05-15T00:00:00Z", LastCommit: "abc123"},
		},
		MissingRepos: []string{"app/frontend"},
		Contracts:    []groupcore.StoredContract{provider, consumer, grpc},
		CrossLinks: []groupcore.CrossLink{{
			From:       groupcore.CrossLinkEndpoint{Repo: "app/frontend", SymbolUID: "uid-consumer", SymbolRef: consumer.SymbolRef},
			To:         groupcore.CrossLinkEndpoint{Repo: "app/backend", SymbolUID: "uid-provider", SymbolRef: provider.SymbolRef},
			Type:       "http",
			ContractID: "http::GET::/api/users",
			MatchType:  "exact",
			Confidence: 1,
		}},
	}
	if err := groupcore.WriteRegistry(homeDir, name, registry); err != nil {
		t.Fatalf("WriteRegistry() error = %v", err)
	}
}
