package group

import (
	"path/filepath"
	"testing"
)

const testMonorepoGroupPath = "platform/monorepo"

func TestMonorepoFixtureExtractsContractsWithServiceAssignments(t *testing.T) {
	monorepoDir := filepath.Join("..", "..", "avmatrix", "test", "fixtures", "group", "test-monorepo")
	boundaries, err := DetectServiceBoundaries(monorepoDir)
	if err != nil {
		t.Fatalf("DetectServiceBoundaries() error = %v", err)
	}
	names := make([]string, 0, len(boundaries))
	for _, boundary := range boundaries {
		names = append(names, boundary.ServiceName)
	}
	for _, want := range []string{"auth", "gateway", "orders"} {
		if !containsString(names, want) {
			t.Fatalf("service boundary names missing %q: %v", want, names)
		}
	}

	contracts := extractMonorepoFixtureContracts(t, monorepoDir, boundaries)
	if len(contracts) == 0 {
		t.Fatal("expected monorepo fixture contracts")
	}
	if len(filterContractsByRole(contracts, "provider")) == 0 || len(filterContractsByRole(contracts, "consumer")) == 0 {
		t.Fatalf("expected both providers and consumers: %#v", contracts)
	}
	withService := 0
	for _, contract := range contracts {
		if contract.Service != "" {
			withService++
		}
	}
	if withService == 0 {
		t.Fatalf("expected service assignments: %#v", contracts)
	}
}

func TestMonorepoFixtureProducesIntraRepoCrossLinks(t *testing.T) {
	monorepoDir := filepath.Join("..", "..", "avmatrix", "test", "fixtures", "group", "test-monorepo")
	boundaries, err := DetectServiceBoundaries(monorepoDir)
	if err != nil {
		t.Fatalf("DetectServiceBoundaries() error = %v", err)
	}
	contracts := extractMonorepoFixtureContracts(t, monorepoDir, boundaries)
	matched, _ := runExactMatch(contracts)
	providers := providerIndex(contracts)
	wildcardMatched, _ := runWildcardMatch(contracts, providers)
	matched = append(matched, wildcardMatched...)

	for _, link := range matched {
		if link.From.Repo != testMonorepoGroupPath || link.To.Repo != testMonorepoGroupPath {
			t.Fatalf("cross-link should be intra-repo: %#v", link)
		}
		if link.From.Service == link.To.Service {
			t.Fatalf("cross-link should cross services: %#v", link)
		}
	}
	if !hasCrossLinkType(matched, "topic") {
		t.Fatalf("expected topic cross-link, got %#v", matched)
	}
	if !hasCrossLinkType(matched, "http") {
		t.Fatalf("expected HTTP cross-link, got %#v", matched)
	}
	if len(matched) < 2 {
		t.Fatalf("expected at least 2 cross-links, got %#v", matched)
	}
}

func extractMonorepoFixtureContracts(t *testing.T, monorepoDir string, boundaries []ServiceBoundary) []StoredContract {
	t.Helper()
	grpcContracts, err := ExtractGRPCContracts(monorepoDir)
	if err != nil {
		t.Fatalf("ExtractGRPCContracts() error = %v", err)
	}
	topicContracts, err := ExtractTopicContracts(monorepoDir)
	if err != nil {
		t.Fatalf("ExtractTopicContracts() error = %v", err)
	}
	httpContracts, err := ExtractHTTPContractsFromSource(monorepoDir)
	if err != nil {
		t.Fatalf("ExtractHTTPContractsFromSource() error = %v", err)
	}
	all := append(append(grpcContracts, topicContracts...), httpContracts...)
	for i := range all {
		all[i].Repo = testMonorepoGroupPath
		all[i].Service = AssignService(all[i].SymbolRef.FilePath, boundaries)
	}
	return all
}

func hasCrossLinkType(links []CrossLink, linkType string) bool {
	for _, link := range links {
		if link.Type == linkType {
			return true
		}
	}
	return false
}
