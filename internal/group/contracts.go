package group

import "fmt"

func Contracts(homeDir string, name string, options ContractsOptions) (*ContractsResult, error) {
	registry, err := ReadRegistry(homeDir, name)
	if err != nil {
		return nil, err
	}
	if registry == nil {
		return nil, fmt.Errorf("No contracts.json for group %q. Run group_sync first.", name)
	}

	contracts := make([]StoredContract, 0, len(registry.Contracts))
	for _, contract := range registry.Contracts {
		if options.Type != "" && contract.Type != options.Type {
			continue
		}
		if options.Repo != "" && contract.Repo != options.Repo {
			continue
		}
		contracts = append(contracts, contract)
	}

	if options.UnmatchedOnly {
		matched := matchedContractKeys(registry.CrossLinks)
		filtered := contracts[:0]
		for _, contract := range contracts {
			if !matched[contract.Repo+"::"+contract.ContractID] {
				filtered = append(filtered, contract)
			}
		}
		contracts = filtered
	}

	return &ContractsResult{
		Contracts:  contracts,
		CrossLinks: registry.CrossLinks,
	}, nil
}

func matchedContractKeys(links []CrossLink) map[string]bool {
	matched := make(map[string]bool, len(links)*2)
	for _, link := range links {
		matched[link.From.Repo+"::"+link.ContractID] = true
		matched[link.To.Repo+"::"+link.ContractID] = true
	}
	return matched
}
