package group

import "strings"

func runExactMatch(contracts []StoredContract) ([]CrossLink, []StoredContract) {
	providers := providerIndex(contracts)
	matched := make([]CrossLink, 0)
	matchedConsumers := make(map[string]bool)
	matchedProviders := make(map[string]bool)
	for _, consumer := range contracts {
		if consumer.Role != "consumer" || isGRPCWildcard(consumer.ContractID) {
			continue
		}
		for _, provider := range matchingProviders(consumer.ContractID, providers) {
			if sameRepoSameService(provider, consumer) {
				continue
			}
			matched = append(matched, CrossLink{
				From:       CrossLinkEndpoint{Repo: consumer.Repo, Service: consumer.Service, SymbolUID: consumer.SymbolUID, SymbolRef: consumer.SymbolRef},
				To:         CrossLinkEndpoint{Repo: provider.Repo, Service: provider.Service, SymbolUID: provider.SymbolUID, SymbolRef: provider.SymbolRef},
				Type:       consumer.Type,
				ContractID: consumer.ContractID,
				MatchType:  "exact",
				Confidence: minFloat64(nonZeroConfidence(consumer.Confidence), nonZeroConfidence(provider.Confidence)),
			})
			matchedConsumers[contractMatchKey(consumer)] = true
			matchedProviders[contractMatchKey(provider)] = true
		}
	}

	unmatched := make([]StoredContract, 0)
	for _, contract := range contracts {
		if isGRPCWildcard(contract.ContractID) {
			unmatched = append(unmatched, contract)
			continue
		}
		key := contractMatchKey(contract)
		if contract.Role == "provider" && !matchedProviders[key] {
			unmatched = append(unmatched, contract)
		}
		if contract.Role == "consumer" && !matchedConsumers[key] {
			unmatched = append(unmatched, contract)
		}
	}
	return matched, unmatched
}

func runWildcardMatch(unmatched []StoredContract, providers map[string][]StoredContract) ([]CrossLink, []StoredContract) {
	matched := make([]CrossLink, 0)
	matchedConsumers := make(map[string]bool)
	for _, consumer := range unmatched {
		if consumer.Role != "consumer" || !isGRPCWildcard(consumer.ContractID) {
			continue
		}
		normalized := normalizeContractID(consumer.ContractID)
		service := strings.TrimSuffix(strings.TrimPrefix(normalized, "grpc::"), "/*")
		for key, candidates := range providers {
			if !strings.HasPrefix(key, "grpc::") || strings.HasSuffix(key, "/*") {
				continue
			}
			providerService, _, ok := strings.Cut(strings.TrimPrefix(key, "grpc::"), "/")
			if !ok {
				continue
			}
			if providerService != service && (strings.Contains(service, ".") || !strings.HasSuffix(providerService, "."+service)) {
				continue
			}
			for _, provider := range candidates {
				if sameRepoSameService(provider, consumer) {
					continue
				}
				matched = append(matched, CrossLink{
					From:       CrossLinkEndpoint{Repo: consumer.Repo, Service: consumer.Service, SymbolUID: consumer.SymbolUID, SymbolRef: consumer.SymbolRef},
					To:         CrossLinkEndpoint{Repo: provider.Repo, Service: provider.Service, SymbolUID: provider.SymbolUID, SymbolRef: provider.SymbolRef},
					Type:       consumer.Type,
					ContractID: consumer.ContractID,
					MatchType:  "wildcard",
					Confidence: minFloat64(nonZeroConfidence(consumer.Confidence), nonZeroConfidence(provider.Confidence)),
				})
				matchedConsumers[contractMatchKey(consumer)] = true
			}
		}
	}
	remaining := make([]StoredContract, 0, len(unmatched))
	for _, contract := range unmatched {
		if contract.Role == "consumer" && isGRPCWildcard(contract.ContractID) && matchedConsumers[contractMatchKey(contract)] {
			continue
		}
		remaining = append(remaining, contract)
	}
	return matched, remaining
}

func providerIndex(contracts []StoredContract) map[string][]StoredContract {
	index := make(map[string][]StoredContract)
	for _, contract := range contracts {
		if contract.Role != "provider" {
			continue
		}
		key := normalizeContractID(contract.ContractID)
		index[key] = append(index[key], contract)
	}
	return index
}

func matchingProviders(contractID string, providers map[string][]StoredContract) []StoredContract {
	normalized := normalizeContractID(contractID)
	if matches := providers[normalized]; len(matches) > 0 {
		return matches
	}
	if strings.HasPrefix(normalized, "http::*::") {
		pathPart := strings.TrimPrefix(normalized, "http::*::")
		matches := make([]StoredContract, 0)
		for key, candidates := range providers {
			if strings.HasPrefix(key, "http::") && strings.HasSuffix(key, "::"+pathPart) {
				matches = append(matches, candidates...)
			}
		}
		return matches
	}
	return nil
}

func normalizeContractID(contractID string) string {
	contractType, rest, ok := strings.Cut(contractID, "::")
	if !ok {
		return contractID
	}
	switch contractType {
	case "http":
		method, path, ok := strings.Cut(rest, "::")
		if !ok {
			return contractID
		}
		return httpContractID(method, path)
	case "grpc":
		service, method, ok := strings.Cut(rest, "/")
		if !ok {
			return "grpc::" + strings.ToLower(rest)
		}
		return "grpc::" + strings.ToLower(service) + "/" + method
	case "topic":
		return "topic::" + strings.ToLower(strings.TrimSpace(rest))
	case "lib":
		return "lib::" + strings.ToLower(rest)
	default:
		return contractID
	}
}

func isGRPCWildcard(contractID string) bool {
	return strings.HasPrefix(contractID, "grpc::") && strings.HasSuffix(contractID, "/*")
}

func sameRepoSameService(provider StoredContract, consumer StoredContract) bool {
	if provider.Repo != consumer.Repo {
		return false
	}
	return provider.Service == "" || consumer.Service == "" || provider.Service == consumer.Service
}

func dedupeContracts(contracts []StoredContract) []StoredContract {
	seen := make(map[string]bool, len(contracts))
	out := make([]StoredContract, 0, len(contracts))
	for _, contract := range contracts {
		key := contract.Repo + "\x00" + contract.ContractID + "\x00" + contract.Role + "\x00" + contract.SymbolRef.FilePath
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, contract)
	}
	return out
}

func dedupeCrossLinks(links []CrossLink) []CrossLink {
	seen := make(map[string]bool, len(links))
	out := make([]CrossLink, 0, len(links))
	for _, link := range links {
		key := link.Type + "\x00" + link.ContractID + "\x00" + link.From.Repo + "\x00" + link.From.SymbolUID + "\x00" + link.To.Repo + "\x00" + link.To.SymbolUID
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, link)
	}
	return out
}

func contractMatchKey(contract StoredContract) string {
	return contract.Repo + "::" + contract.ContractID
}

func nonZeroConfidence(confidence float64) float64 {
	if confidence == 0 {
		return 1
	}
	return confidence
}

func minFloat64(left float64, right float64) float64 {
	if left < right {
		return left
	}
	return right
}
