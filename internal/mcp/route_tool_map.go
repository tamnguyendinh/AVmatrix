package mcp

import (
	"sort"
	"strings"

	"github.com/tamnguyendinh/avmatrix-go/internal/graph"
	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

type mcpRouteConsumer struct {
	Name         string   `json:"name"`
	FilePath     string   `json:"filePath"`
	AccessedKeys []string `json:"accessedKeys,omitempty"`
	FetchCount   int      `json:"fetchCount,omitempty"`
}

type mcpRouteMapItem struct {
	Route      string             `json:"route"`
	Handler    string             `json:"handler"`
	Middleware []string           `json:"middleware"`
	Consumers  []mcpRouteConsumer `json:"consumers"`
	Flows      []string           `json:"flows"`
}

type mcpToolMapItem struct {
	Name        string   `json:"name"`
	FilePath    string   `json:"filePath"`
	Description string   `json:"description"`
	Flows       []string `json:"flows"`
}

type mcpRouteIndex struct {
	routes  []mcpRouteMapItem
	records []mcpRouteAnalysisRecord
}

func (s Server) routeMapTool(args map[string]any) (map[string]any, error) {
	filter := strings.TrimSpace(stringArg(args, "route", ""))
	index, err := s.routeIndexForResource(stringArg(args, "repo", ""))
	if err != nil {
		return nil, err
	}
	routes := index.routeMapItems(filter)
	if len(routes) == 0 {
		message := "No routes found in this project."
		if filter != "" {
			message = `No routes matching "` + filter + `"`
		}
		return map[string]any{"routes": []mcpRouteMapItem{}, "total": 0, "message": message}, nil
	}
	return map[string]any{"routes": routes, "total": len(routes)}, nil
}

func (s Server) toolMapTool(args map[string]any) (map[string]any, error) {
	filter := strings.TrimSpace(stringArg(args, "tool", ""))
	g, err := s.graphForResource(stringArg(args, "repo", ""))
	if err != nil {
		return nil, err
	}
	tools := mcpToolMapItems(g, filter)
	if len(tools) == 0 {
		message := "No tool definitions found."
		if filter != "" {
			message = `No tools matching "` + filter + `"`
		}
		return map[string]any{"tools": []mcpToolMapItem{}, "total": 0, "message": message}, nil
	}
	return map[string]any{"tools": tools, "total": len(tools)}, nil
}

func mcpRouteMapItems(g *graph.Graph, filter string) []mcpRouteMapItem {
	return buildMCPRouteIndex(g).routeMapItems(filter)
}

func buildMCPRouteIndex(g *graph.Graph) *mcpRouteIndex {
	nodeByID := resourceGraphNodesByID(g)
	consumerMap := mcpRouteConsumersByRoute(g, nodeByID)
	flowMap := mcpLinkedFlowsBySource(g, nodeByID)
	index := &mcpRouteIndex{
		routes:  make([]mcpRouteMapItem, 0),
		records: make([]mcpRouteAnalysisRecord, 0),
	}
	for _, node := range g.Nodes {
		if node.Label != scopeir.NodeRoute {
			continue
		}
		route := firstResourceNodeString(node, "name", "label")
		handler := resourceNodeString(node, "filePath")
		consumers := consumerMap[node.ID]
		sort.Slice(consumers, func(i, j int) bool {
			if consumers[i].FilePath != consumers[j].FilePath {
				return consumers[i].FilePath < consumers[j].FilePath
			}
			return consumers[i].Name < consumers[j].Name
		})
		middleware := nonNilStringSlice(resourceNodeStringSlice(node, "middleware"))
		flows := nonNilStringSlice(flowMap[node.ID])
		index.routes = append(index.routes, mcpRouteMapItem{
			Route:      route,
			Handler:    handler,
			Middleware: middleware,
			Consumers:  nonNilRouteConsumers(consumers),
			Flows:      flows,
		})
		index.records = append(index.records, mcpRouteAnalysisRecord{
			ID:           node.ID,
			Name:         route,
			Handler:      handler,
			ResponseKeys: resourceNodeStringSlice(node, "responseKeys"),
			ErrorKeys:    resourceNodeStringSlice(node, "errorKeys"),
			Middleware:   middleware,
			Consumers:    nonNilRouteConsumers(consumers),
			Flows:        flows,
		})
	}
	sort.Slice(index.routes, func(i, j int) bool {
		return index.routes[i].Route < index.routes[j].Route
	})
	sort.Slice(index.records, func(i, j int) bool {
		return index.records[i].Name < index.records[j].Name
	})
	return index
}

func (index *mcpRouteIndex) routeMapItems(filter string) []mcpRouteMapItem {
	if index == nil {
		return nil
	}
	needle := strings.ToLower(strings.TrimSpace(filter))
	items := make([]mcpRouteMapItem, 0, len(index.routes))
	for _, item := range index.routes {
		if needle != "" && !strings.Contains(strings.ToLower(item.Route), needle) {
			continue
		}
		items = append(items, cloneMCPRouteMapItem(item))
	}
	return items
}

func (index *mcpRouteIndex) analysisRecords(routeFilter string, fileFilter string) []mcpRouteAnalysisRecord {
	if index == nil {
		return nil
	}
	routeNeedle := strings.ToLower(strings.TrimSpace(routeFilter))
	fileNeedle := strings.ToLower(strings.TrimSpace(fileFilter))
	records := make([]mcpRouteAnalysisRecord, 0, len(index.records))
	for _, record := range index.records {
		if routeNeedle != "" && !strings.Contains(strings.ToLower(record.Name), routeNeedle) {
			continue
		}
		if fileNeedle != "" && !strings.Contains(strings.ToLower(record.Handler), fileNeedle) {
			continue
		}
		records = append(records, cloneMCPRouteAnalysisRecord(record))
	}
	return records
}

func cloneMCPRouteMapItem(item mcpRouteMapItem) mcpRouteMapItem {
	return mcpRouteMapItem{
		Route:      item.Route,
		Handler:    item.Handler,
		Middleware: cloneNonNilStringSlice(item.Middleware),
		Consumers:  cloneNonNilRouteConsumers(item.Consumers),
		Flows:      cloneNonNilStringSlice(item.Flows),
	}
}

func cloneMCPRouteAnalysisRecord(record mcpRouteAnalysisRecord) mcpRouteAnalysisRecord {
	return mcpRouteAnalysisRecord{
		ID:           record.ID,
		Name:         record.Name,
		Handler:      record.Handler,
		ResponseKeys: cloneStringSlice(record.ResponseKeys),
		ErrorKeys:    cloneStringSlice(record.ErrorKeys),
		Middleware:   cloneNonNilStringSlice(record.Middleware),
		Consumers:    cloneNonNilRouteConsumers(record.Consumers),
		Flows:        cloneNonNilStringSlice(record.Flows),
	}
}

func cloneStringSlice(values []string) []string {
	if values == nil {
		return nil
	}
	return append([]string(nil), values...)
}

func cloneNonNilStringSlice(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	return append([]string(nil), values...)
}

func cloneNonNilRouteConsumers(values []mcpRouteConsumer) []mcpRouteConsumer {
	if len(values) == 0 {
		return []mcpRouteConsumer{}
	}
	out := append([]mcpRouteConsumer(nil), values...)
	for i := range out {
		out[i].AccessedKeys = cloneStringSlice(out[i].AccessedKeys)
	}
	return out
}

func mcpToolMapItems(g *graph.Graph, filter string) []mcpToolMapItem {
	flowMap := mcpLinkedFlowsBySource(g, resourceGraphNodesByID(g))
	needle := strings.ToLower(filter)
	items := make([]mcpToolMapItem, 0)
	for _, node := range g.Nodes {
		if node.Label != scopeir.NodeTool {
			continue
		}
		name := firstResourceNodeString(node, "name", "label")
		if filter != "" && !strings.Contains(strings.ToLower(name), needle) {
			continue
		}
		items = append(items, mcpToolMapItem{
			Name:        name,
			FilePath:    resourceNodeString(node, "filePath"),
			Description: trimMCPToolDescription(resourceNodeString(node, "description")),
			Flows:       nonNilStringSlice(flowMap[node.ID]),
		})
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].Name < items[j].Name
	})
	return items
}

func mcpRouteConsumersByRoute(g *graph.Graph, nodeByID map[string]graph.Node) map[string][]mcpRouteConsumer {
	consumers := make(map[string][]mcpRouteConsumer)
	for _, relationship := range g.Relationships {
		if relationship.Type != graph.RelFetches {
			continue
		}
		source, ok := nodeByID[relationship.SourceID]
		if !ok {
			continue
		}
		consumer := mcpRouteConsumer{
			Name:         firstResourceNodeString(source, "name", "label", "heuristicLabel"),
			FilePath:     resourceNodeString(source, "filePath"),
			AccessedKeys: mcpFetchReasonKeys(relationship.Reason),
			FetchCount:   mcpFetchReasonCount(relationship.Reason),
		}
		consumers[relationship.TargetID] = append(consumers[relationship.TargetID], consumer)
	}
	return consumers
}

func mcpLinkedFlowsBySource(g *graph.Graph, nodeByID map[string]graph.Node) map[string][]string {
	flows := make(map[string][]string)
	for _, relationship := range g.Relationships {
		if relationship.Type != graph.RelEntryPointOf {
			continue
		}
		target, ok := nodeByID[relationship.TargetID]
		if !ok || target.Label != scopeir.NodeProcess {
			continue
		}
		name := firstResourceNodeString(target, "label", "heuristicLabel", "name")
		if name == "" {
			continue
		}
		flows[relationship.SourceID] = append(flows[relationship.SourceID], name)
	}
	for sourceID := range flows {
		sort.Strings(flows[sourceID])
	}
	return flows
}

func resourceNodeStringSlice(node graph.Node, key string) []string {
	raw, ok := node.Properties[key]
	if !ok || raw == nil {
		return nil
	}
	switch value := raw.(type) {
	case []string:
		return cleanMCPStringSlice(value)
	case []any:
		values := make([]string, 0, len(value))
		for _, item := range value {
			if text, ok := item.(string); ok {
				values = append(values, text)
			}
		}
		return cleanMCPStringSlice(values)
	default:
		return nil
	}
}

func cleanMCPStringSlice(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.Trim(strings.TrimSpace(value), `"'`)
		if value != "" {
			out = append(out, value)
		}
	}
	if len(out) == 0 {
		return nil
	}
	sort.Strings(out)
	return out
}

func mcpFetchReasonKeys(reason string) []string {
	keysPart := mcpReasonPart(reason, "keys:")
	if keysPart == "" {
		return nil
	}
	return cleanMCPStringSlice(strings.Split(keysPart, ","))
}

func mcpFetchReasonCount(reason string) int {
	raw := mcpReasonPart(reason, "fetches:")
	if raw == "" {
		return 0
	}
	value := 0
	for _, digit := range raw {
		if digit < '0' || digit > '9' {
			break
		}
		value = value*10 + int(digit-'0')
	}
	if value <= 1 {
		return 0
	}
	return value
}

func mcpReasonPart(reason string, prefix string) string {
	for _, part := range strings.Split(reason, "|") {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, prefix) {
			return strings.TrimSpace(strings.TrimPrefix(part, prefix))
		}
	}
	return ""
}

func trimMCPToolDescription(description string) string {
	description = strings.TrimSpace(description)
	if len(description) <= 200 {
		return description
	}
	return description[:200]
}

func nonNilStringSlice(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}

func nonNilRouteConsumers(values []mcpRouteConsumer) []mcpRouteConsumer {
	if values == nil {
		return []mcpRouteConsumer{}
	}
	return values
}
