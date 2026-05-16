package group

import (
	"strings"

	"github.com/tamnguyendinh/avmatrix-go/internal/graph"
	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

func extractHTTPContracts(g *graph.Graph, groupRepoPath string) []StoredContract {
	nodes := groupNodesByID(g)
	contracts := make([]StoredContract, 0)
	for _, node := range g.Nodes {
		if node.Label != scopeir.NodeRoute {
			continue
		}
		routePath := routePathForNode(node)
		handlerPath := groupNodeString(node, "filePath")
		if routePath == "" || handlerPath == "" {
			continue
		}
		contracts = append(contracts, StoredContract{
			Repo:       groupRepoPath,
			ContractID: httpContractID("", routePath),
			Type:       "http",
			Role:       "provider",
			SymbolUID:  node.ID,
			SymbolRef:  SymbolRef{FilePath: handlerPath, Name: routePath},
			SymbolName: routePath,
			Confidence: 1,
			Meta:       map[string]any{"source": "graph-route"},
		})
	}

	for _, relationship := range g.Relationships {
		if relationship.Type != graph.RelFetches {
			continue
		}
		source, sourceOK := nodes[relationship.SourceID]
		route, routeOK := nodes[relationship.TargetID]
		if !sourceOK || !routeOK || route.Label != scopeir.NodeRoute {
			continue
		}
		routePath := routePathForNode(route)
		if routePath == "" {
			continue
		}
		confidence := relationship.Confidence
		if confidence == 0 {
			confidence = 0.8
		}
		meta := map[string]any{"source": "graph-fetch"}
		if relationship.Reason != "" {
			meta["reason"] = relationship.Reason
		}
		contracts = append(contracts, StoredContract{
			Repo:       groupRepoPath,
			ContractID: httpContractID("", routePath),
			Type:       "http",
			Role:       "consumer",
			SymbolUID:  source.ID,
			SymbolRef: SymbolRef{
				FilePath: groupNodeString(source, "filePath"),
				Name:     firstNonEmptyGroupString(groupNodeString(source, "name"), source.ID),
			},
			SymbolName: firstNonEmptyGroupString(groupNodeString(source, "name"), source.ID),
			Confidence: confidence,
			Meta:       meta,
		})
	}
	return contracts
}

func routePathForNode(node graph.Node) string {
	name := firstNonEmptyGroupString(groupNodeString(node, "name"), groupNodeString(node, "label"))
	if name == "" && strings.HasPrefix(node.ID, "Route:") {
		name = strings.TrimPrefix(node.ID, "Route:")
	}
	return normalizeHTTPPath(name)
}
