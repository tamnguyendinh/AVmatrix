package group

import (
	"regexp"
	"strings"
)

var httpManifestMethodPattern = regexp.MustCompile(`^([A-Za-z]+)::`)

func manifestContractsAndLinks(links []ManifestLink) ([]StoredContract, []CrossLink) {
	contracts := make([]StoredContract, 0, len(links)*2)
	crossLinks := make([]CrossLink, 0, len(links))
	for _, link := range links {
		contractID := manifestContractID(link.Type, link.Contract)
		providerRepo := link.To
		consumerRepo := link.From
		if link.Role == "provider" {
			providerRepo = link.From
			consumerRepo = link.To
		}

		providerRef := SymbolRef{Name: link.Contract}
		consumerRef := SymbolRef{Name: link.Contract}
		providerUID := manifestSymbolUID(providerRepo, contractID)
		consumerUID := manifestSymbolUID(consumerRepo, contractID)
		contracts = append(contracts,
			StoredContract{
				Repo:       providerRepo,
				ContractID: contractID,
				Type:       link.Type,
				Role:       "provider",
				SymbolUID:  providerUID,
				SymbolRef:  providerRef,
				SymbolName: link.Contract,
				Confidence: 1,
				Meta:       map[string]any{"source": "manifest"},
			},
			StoredContract{
				Repo:       consumerRepo,
				ContractID: contractID,
				Type:       link.Type,
				Role:       "consumer",
				SymbolUID:  consumerUID,
				SymbolRef:  consumerRef,
				SymbolName: link.Contract,
				Confidence: 1,
				Meta:       map[string]any{"source": "manifest"},
			},
		)
		crossLinks = append(crossLinks, CrossLink{
			From:       CrossLinkEndpoint{Repo: consumerRepo, SymbolUID: consumerUID, SymbolRef: consumerRef},
			To:         CrossLinkEndpoint{Repo: providerRepo, SymbolUID: providerUID, SymbolRef: providerRef},
			Type:       link.Type,
			ContractID: contractID,
			MatchType:  "manifest",
			Confidence: 1,
		})
	}
	return contracts, crossLinks
}

func manifestSymbolUID(repoPath string, contractID string) string {
	return "manifest::" + repoPath + "::" + contractID
}

func manifestContractID(contractType string, contract string) string {
	switch contractType {
	case "http":
		method, path := parseHTTPManifestContract(contract)
		return httpContractID(method, path)
	case "grpc", "topic", "lib", "custom":
		return contractType + "::" + contract
	default:
		return contractType + "::" + contract
	}
}

func parseHTTPManifestContract(contract string) (string, string) {
	trimmed := strings.TrimSpace(contract)
	match := httpManifestMethodPattern.FindStringSubmatch(trimmed)
	if len(match) != 2 {
		return "", trimmed
	}
	return strings.ToUpper(match[1]), strings.TrimPrefix(trimmed, match[0])
}

func httpContractID(method string, path string) string {
	method = strings.ToUpper(strings.TrimSpace(method))
	if method == "" {
		method = "*"
	}
	return "http::" + method + "::" + normalizeHTTPPath(path)
}

func normalizeHTTPPath(path string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return "/"
	}
	if !strings.HasPrefix(trimmed, "/") {
		trimmed = "/" + trimmed
	}
	for strings.Contains(trimmed, "//") {
		trimmed = strings.ReplaceAll(trimmed, "//", "/")
	}
	if trimmed == "/" {
		return trimmed
	}
	return strings.TrimRight(trimmed, "/")
}
