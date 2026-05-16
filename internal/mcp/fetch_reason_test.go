package mcp

import (
	"reflect"
	"strings"
	"testing"
)

func TestMCPFetchReasonParsing(t *testing.T) {
	tests := []struct {
		name      string
		reason    string
		wantKeys  []string
		wantCount int
	}{
		{name: "basic reason with no keys or fetches", reason: "fetch-url-match"},
		{name: "keys without fetches suffix", reason: "fetch-url-match|keys:data,pagination", wantKeys: []string{"data", "pagination"}},
		{name: "keys with fetches suffix", reason: "fetch-url-match|keys:data,pagination|fetches:3", wantKeys: []string{"data", "pagination"}, wantCount: 3},
		{name: "fetches without keys", reason: "fetch-url-match|fetches:2", wantCount: 2},
		{name: "single fetch count omitted", reason: "fetch-url-match|keys:data|fetches:1", wantKeys: []string{"data"}},
		{name: "empty reason"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mcpFetchReasonKeys(tt.reason); !reflect.DeepEqual(got, tt.wantKeys) {
				t.Fatalf("mcpFetchReasonKeys(%q) = %#v, want %#v", tt.reason, got, tt.wantKeys)
			}
			if got := mcpFetchReasonCount(tt.reason); got != tt.wantCount {
				t.Fatalf("mcpFetchReasonCount(%q) = %d, want %d", tt.reason, got, tt.wantCount)
			}
		})
	}
}

func TestMCPFetchReasonKeysStopBeforeFetchesSuffix(t *testing.T) {
	keys := mcpFetchReasonKeys("fetch-url-match|keys:data,pagination|fetches:3")
	if reflect.DeepEqual(keys, []string{"data", "pagination|fetches:3"}) || containsMCPFetchReasonTestString(keys, "pagination|fetches:3") {
		t.Fatalf("keys included fetches suffix: %#v", keys)
	}
	if !reflect.DeepEqual(keys, []string{"data", "pagination"}) {
		t.Fatalf("keys = %#v, want data,pagination", keys)
	}
}

func TestMCPShapeConsumersDerivesConfidenceFromFetchCount(t *testing.T) {
	record := mcpRouteAnalysisRecord{
		ResponseKeys: []string{"data"},
		Consumers: []mcpRouteConsumer{
			{Name: "singleFetch", FilePath: "src/single.ts", AccessedKeys: []string{"missing"}},
			{Name: "multiFetch", FilePath: "src/multi.ts", AccessedKeys: []string{"missing"}, FetchCount: 3},
		},
	}
	consumers := mcpShapeConsumers(record)
	if len(consumers) != 2 {
		t.Fatalf("mcpShapeConsumers() = %#v", consumers)
	}
	if consumers[0].MismatchConfidence != "high" || len(consumers[0].Mismatched) != 1 || consumers[0].Mismatched[0] != "missing" {
		t.Fatalf("single-fetch consumer = %#v", consumers[0])
	}
	if consumers[1].MismatchConfidence != "low" || !strings.Contains(consumers[1].AttributionNote, "3 routes") {
		t.Fatalf("multi-fetch consumer = %#v", consumers[1])
	}
}

func TestMCPAPIImpactMiddlewareDetectionPartialFlag(t *testing.T) {
	record := mcpRouteAnalysisRecord{
		Name:       "/api/users",
		Handler:    "app/api/users/route.ts",
		Middleware: []string{"withAuth"},
	}
	partial := mcpAPIImpactRecord(record, 2)
	if partial.MiddlewareDetection != "partial" || partial.MiddlewareNote == "" {
		t.Fatalf("partial middleware impact = %#v", partial)
	}

	singleRoute := mcpAPIImpactRecord(record, 1)
	if singleRoute.MiddlewareDetection != "" || singleRoute.MiddlewareNote != "" {
		t.Fatalf("single route middleware impact = %#v", singleRoute)
	}

	record.Middleware = nil
	noMiddleware := mcpAPIImpactRecord(record, 3)
	if noMiddleware.MiddlewareDetection != "" || noMiddleware.MiddlewareNote != "" {
		t.Fatalf("no middleware impact = %#v", noMiddleware)
	}
}

func TestMCPAPIImpactRecordSeparatesShapesAndMismatches(t *testing.T) {
	record := mcpRouteAnalysisRecord{
		Name:         "/api/grants",
		Handler:      "app/api/grants/route.ts",
		ResponseKeys: []string{"data", "pagination"},
		ErrorKeys:    []string{"error", "message"},
		Middleware:   []string{"withAuth", "withRateLimit"},
		Consumers: []mcpRouteConsumer{
			{Name: "GrantsList", FilePath: "components/GrantsList.tsx", AccessedKeys: []string{"data", "pagination", "message"}},
			{Name: "useGrants", FilePath: "hooks/useGrants.ts", AccessedKeys: []string{"items"}},
			{Name: "useMulti", FilePath: "hooks/useMulti.ts", AccessedKeys: []string{"meta"}, FetchCount: 2},
		},
		Flows: []string{"GrantListFlow"},
	}

	impact := mcpAPIImpactRecord(record, 1)
	if !reflect.DeepEqual(impact.ResponseShape["success"], []string{"data", "pagination"}) {
		t.Fatalf("success response shape = %#v", impact.ResponseShape["success"])
	}
	if !reflect.DeepEqual(impact.ResponseShape["error"], []string{"error", "message"}) {
		t.Fatalf("error response shape = %#v", impact.ResponseShape["error"])
	}
	if len(impact.Consumers) != 3 || impact.Consumers[2].AttributionNote == "" {
		t.Fatalf("consumers = %#v", impact.Consumers)
	}
	if len(impact.Mismatches) != 2 {
		t.Fatalf("mismatches = %#v, want items and meta", impact.Mismatches)
	}
	if impact.Mismatches[0].Field != "items" || impact.Mismatches[0].Consumer != "hooks/useGrants.ts" || impact.Mismatches[0].Confidence != "high" {
		t.Fatalf("high-confidence mismatch = %#v", impact.Mismatches[0])
	}
	if impact.Mismatches[1].Field != "meta" || impact.Mismatches[1].Consumer != "hooks/useMulti.ts" || impact.Mismatches[1].Confidence != "low" {
		t.Fatalf("low-confidence mismatch = %#v", impact.Mismatches[1])
	}
	if impact.ImpactSummary["directConsumers"] != 3 || impact.ImpactSummary["affectedFlows"] != 1 || impact.ImpactSummary["riskLevel"] != "MEDIUM" {
		t.Fatalf("impact summary = %#v", impact.ImpactSummary)
	}
}

func TestMCPShapeConsumersTreatsErrorKeysAsErrorPathNotMismatch(t *testing.T) {
	record := mcpRouteAnalysisRecord{
		Name:         "/api/orders",
		Handler:      "app/api/orders/route.ts",
		ResponseKeys: []string{"items", "orderId", "status"},
		ErrorKeys:    []string{"code", "error"},
		Consumers: []mcpRouteConsumer{
			{Name: "OrderStatus", FilePath: "components/OrderStatus.tsx", AccessedKeys: []string{"orderId", "status", "error"}},
		},
	}

	consumers := mcpShapeConsumers(record)
	if len(consumers) != 1 {
		t.Fatalf("consumers = %#v", consumers)
	}
	if len(consumers[0].Mismatched) != 0 {
		t.Fatalf("mismatched = %#v, want none for error-path key", consumers[0].Mismatched)
	}
	if !reflect.DeepEqual(consumers[0].ErrorPathKeys, []string{"error"}) {
		t.Fatalf("errorPathKeys = %#v, want error", consumers[0].ErrorPathKeys)
	}
}

func TestMCPShapeConsumersKeepsDOMLikeAPIFieldsValid(t *testing.T) {
	record := mcpRouteAnalysisRecord{
		Name:         "/api/links",
		Handler:      "app/api/links/route.ts",
		ResponseKeys: []string{"href", "label", "target", "type"},
		Consumers: []mcpRouteConsumer{
			{Name: "LinkList", FilePath: "components/LinkList.tsx", AccessedKeys: []string{"type", "href", "target", "label"}},
		},
	}

	consumers := mcpShapeConsumers(record)
	if len(consumers) != 1 {
		t.Fatalf("consumers = %#v", consumers)
	}
	if len(consumers[0].Mismatched) != 0 {
		t.Fatalf("mismatched = %#v, want none for DOM-like response fields", consumers[0].Mismatched)
	}
	if len(consumers[0].ErrorPathKeys) != 0 {
		t.Fatalf("errorPathKeys = %#v, want none", consumers[0].ErrorPathKeys)
	}
}

func containsMCPFetchReasonTestString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
