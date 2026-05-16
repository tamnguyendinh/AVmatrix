package mcp

import (
	"strings"

	"github.com/tamnguyendinh/avmatrix-go/internal/graph"
)

type mcpRouteAnalysisRecord struct {
	ID           string
	Name         string
	Handler      string
	ResponseKeys []string
	ErrorKeys    []string
	Middleware   []string
	Consumers    []mcpRouteConsumer
	Flows        []string
}

type mcpShapeConsumer struct {
	Name               string   `json:"name"`
	FilePath           string   `json:"filePath"`
	AccessedKeys       []string `json:"accessedKeys,omitempty"`
	Mismatched         []string `json:"mismatched,omitempty"`
	MismatchConfidence string   `json:"mismatchConfidence,omitempty"`
	ErrorPathKeys      []string `json:"errorPathKeys,omitempty"`
	AttributionNote    string   `json:"attributionNote,omitempty"`
}

type mcpShapeRoute struct {
	Route        string             `json:"route"`
	Handler      string             `json:"handler"`
	ResponseKeys []string           `json:"responseKeys,omitempty"`
	ErrorKeys    []string           `json:"errorKeys,omitempty"`
	Consumers    []mcpShapeConsumer `json:"consumers"`
	Status       string             `json:"status,omitempty"`
}

type mcpAPIConsumer struct {
	Name            string   `json:"name"`
	File            string   `json:"file"`
	Accesses        []string `json:"accesses"`
	AttributionNote string   `json:"attributionNote,omitempty"`
}

type mcpAPIMismatch struct {
	Consumer   string `json:"consumer"`
	Field      string `json:"field"`
	Reason     string `json:"reason"`
	Confidence string `json:"confidence"`
}

type mcpAPIImpactRoute struct {
	Route               string              `json:"route"`
	Handler             string              `json:"handler"`
	ResponseShape       map[string][]string `json:"responseShape"`
	Middleware          []string            `json:"middleware"`
	MiddlewareDetection string              `json:"middlewareDetection,omitempty"`
	MiddlewareNote      string              `json:"middlewareNote,omitempty"`
	Consumers           []mcpAPIConsumer    `json:"consumers"`
	Mismatches          []mcpAPIMismatch    `json:"mismatches,omitempty"`
	ExecutionFlows      []string            `json:"executionFlows"`
	ImpactSummary       map[string]any      `json:"impactSummary"`
}

func (s Server) shapeCheckTool(args map[string]any) (map[string]any, error) {
	filter := strings.TrimSpace(stringArg(args, "route", ""))
	index, err := s.routeIndexForResource(stringArg(args, "repo", ""))
	if err != nil {
		return nil, err
	}
	records := index.analysisRecords(filter, "")
	routes := make([]mcpShapeRoute, 0)
	mismatchCount := 0
	for _, record := range records {
		if len(record.ResponseKeys) == 0 && len(record.ErrorKeys) == 0 {
			continue
		}
		if len(record.Consumers) == 0 {
			continue
		}
		route := mcpShapeRoute{
			Route:        record.Name,
			Handler:      record.Handler,
			ResponseKeys: record.ResponseKeys,
			ErrorKeys:    record.ErrorKeys,
			Consumers:    mcpShapeConsumers(record),
		}
		for _, consumer := range route.Consumers {
			if len(consumer.Mismatched) > 0 {
				route.Status = "MISMATCH"
				break
			}
		}
		if route.Status == "MISMATCH" {
			mismatchCount++
		}
		routes = append(routes, route)
	}
	message := "No routes with both response shapes and consumers found."
	if len(routes) > 0 && mismatchCount > 0 {
		message = "Found " + intString(len(routes)) + " route(s) with response shape data. " + intString(mismatchCount) + " route(s) have consumer/shape mismatches."
	} else if len(routes) > 0 {
		message = "Found " + intString(len(routes)) + " route(s) with response shape data and consumers."
	}
	result := map[string]any{
		"routes":           routes,
		"total":            len(routes),
		"routesWithShapes": len(routes),
		"message":          message,
	}
	if mismatchCount > 0 {
		result["mismatches"] = mismatchCount
	}
	return result, nil
}

func (s Server) apiImpactTool(args map[string]any) (map[string]any, error) {
	routeFilter := strings.TrimSpace(stringArg(args, "route", ""))
	fileFilter := strings.TrimSpace(stringArg(args, "file", ""))
	if routeFilter == "" && fileFilter == "" {
		return map[string]any{"error": `Either "route" or "file" parameter is required.`}, nil
	}
	index, err := s.routeIndexForResource(stringArg(args, "repo", ""))
	if err != nil {
		return nil, err
	}
	records := index.analysisRecords(routeFilter, fileFilter)
	if len(records) == 0 {
		target := routeFilter
		if target == "" {
			target = fileFilter
		}
		return map[string]any{"error": `No routes found matching "` + target + `".`}, nil
	}
	handlerCounts := make(map[string]int)
	for _, record := range records {
		if record.Handler != "" {
			handlerCounts[record.Handler]++
		}
	}
	routes := make([]mcpAPIImpactRoute, 0, len(records))
	for _, record := range records {
		routes = append(routes, mcpAPIImpactRecord(record, handlerCounts[record.Handler]))
	}
	if len(routes) == 1 {
		return structToMap(routes[0]), nil
	}
	return map[string]any{"routes": routes, "total": len(routes)}, nil
}

func mcpRouteAnalysisRecords(g *graph.Graph, routeFilter string, fileFilter string) []mcpRouteAnalysisRecord {
	return buildMCPRouteIndex(g).analysisRecords(routeFilter, fileFilter)
}

func mcpShapeConsumers(record mcpRouteAnalysisRecord) []mcpShapeConsumer {
	knownKeys := stringSliceSet(append(append([]string{}, record.ResponseKeys...), record.ErrorKeys...))
	responseKeySet := stringSliceSet(record.ResponseKeys)
	out := make([]mcpShapeConsumer, 0, len(record.Consumers))
	for _, consumer := range record.Consumers {
		item := mcpShapeConsumer{
			Name:     consumer.Name,
			FilePath: consumer.FilePath,
		}
		if len(consumer.AccessedKeys) > 0 {
			item.AccessedKeys = consumer.AccessedKeys
			item.Mismatched = missingKeys(consumer.AccessedKeys, knownKeys)
			item.ErrorPathKeys = errorPathKeys(consumer.AccessedKeys, knownKeys, responseKeySet)
			if len(item.Mismatched) > 0 {
				item.MismatchConfidence = "high"
				if consumer.FetchCount > 1 {
					item.MismatchConfidence = "low"
				}
			}
			if consumer.FetchCount > 1 {
				item.AttributionNote = "This file fetches " + intString(consumer.FetchCount) + " routes - accessed keys may belong to a different route."
			}
		}
		out = append(out, item)
	}
	return out
}

func mcpAPIImpactRecord(record mcpRouteAnalysisRecord, handlerRouteCount int) mcpAPIImpactRoute {
	knownKeys := stringSliceSet(append(append([]string{}, record.ResponseKeys...), record.ErrorKeys...))
	consumers := make([]mcpAPIConsumer, 0, len(record.Consumers))
	mismatches := make([]mcpAPIMismatch, 0)
	for _, consumer := range record.Consumers {
		item := mcpAPIConsumer{
			Name:     consumer.Name,
			File:     consumer.FilePath,
			Accesses: append([]string(nil), consumer.AccessedKeys...),
		}
		if consumer.FetchCount > 1 {
			item.AttributionNote = "This file fetches " + intString(consumer.FetchCount) + " routes - accessed keys may belong to a different route."
		}
		consumers = append(consumers, item)
		for _, key := range missingKeys(consumer.AccessedKeys, knownKeys) {
			confidence := "high"
			if consumer.FetchCount > 1 {
				confidence = "low"
			}
			mismatches = append(mismatches, mcpAPIMismatch{
				Consumer:   consumer.FilePath,
				Field:      key,
				Reason:     "accessed but not in response shape",
				Confidence: confidence,
			})
		}
	}
	riskLevel := apiRiskLevel(len(consumers), len(mismatches))
	summary := map[string]any{
		"directConsumers": len(consumers),
		"affectedFlows":   len(record.Flows),
		"riskLevel":       riskLevel,
	}
	if len(consumers) > 0 {
		summary["warning"] = "Changing response shape will affect " + intString(len(consumers)) + " component" + pluralSuffix(len(consumers))
	}
	route := mcpAPIImpactRoute{
		Route:          record.Name,
		Handler:        record.Handler,
		ResponseShape:  map[string][]string{"success": nonNilStringSlice(record.ResponseKeys), "error": nonNilStringSlice(record.ErrorKeys)},
		Middleware:     nonNilStringSlice(record.Middleware),
		Consumers:      nonNilAPIConsumers(consumers),
		Mismatches:     mismatches,
		ExecutionFlows: nonNilStringSlice(record.Flows),
		ImpactSummary:  summary,
	}
	if len(record.Middleware) > 0 && handlerRouteCount > 1 {
		route.MiddlewareDetection = "partial"
		route.MiddlewareNote = "Middleware captured from first HTTP method export only - other methods in this handler may use different middleware chains."
	}
	return route
}

func stringSliceSet(values []string) map[string]bool {
	out := make(map[string]bool, len(values))
	for _, value := range values {
		out[value] = true
	}
	return out
}

func missingKeys(values []string, known map[string]bool) []string {
	out := make([]string, 0)
	for _, value := range values {
		if !known[value] {
			out = append(out, value)
		}
	}
	return out
}

func errorPathKeys(values []string, known map[string]bool, responseKeys map[string]bool) []string {
	out := make([]string, 0)
	for _, value := range values {
		if known[value] && !responseKeys[value] {
			out = append(out, value)
		}
	}
	return out
}

func apiRiskLevel(consumerCount int, mismatchCount int) string {
	risk := "LOW"
	if consumerCount >= 10 {
		risk = "HIGH"
	} else if consumerCount >= 4 {
		risk = "MEDIUM"
	}
	if mismatchCount > 0 {
		if risk == "LOW" {
			return "MEDIUM"
		}
		return "HIGH"
	}
	return risk
}

func pluralSuffix(count int) string {
	if count == 1 {
		return ""
	}
	return "s"
}

func intString(value int) string {
	if value == 0 {
		return "0"
	}
	digits := make([]byte, 0, 10)
	for value > 0 {
		digits = append(digits, byte('0'+value%10))
		value /= 10
	}
	for left, right := 0, len(digits)-1; left < right; left, right = left+1, right-1 {
		digits[left], digits[right] = digits[right], digits[left]
	}
	return string(digits)
}

func structToMap(route mcpAPIImpactRoute) map[string]any {
	out := map[string]any{
		"route":          route.Route,
		"handler":        route.Handler,
		"responseShape":  route.ResponseShape,
		"middleware":     route.Middleware,
		"consumers":      route.Consumers,
		"executionFlows": route.ExecutionFlows,
		"impactSummary":  route.ImpactSummary,
	}
	if route.MiddlewareDetection != "" {
		out["middlewareDetection"] = route.MiddlewareDetection
	}
	if route.MiddlewareNote != "" {
		out["middlewareNote"] = route.MiddlewareNote
	}
	if len(route.Mismatches) > 0 {
		out["mismatches"] = route.Mismatches
	}
	return out
}

func nonNilAPIConsumers(values []mcpAPIConsumer) []mcpAPIConsumer {
	if values == nil {
		return []mcpAPIConsumer{}
	}
	return values
}
