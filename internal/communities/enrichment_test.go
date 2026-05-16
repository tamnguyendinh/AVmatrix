package communities

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

type fakeLLM struct {
	responses []string
	err       error
	calls     []string
}

func (client *fakeLLM) Generate(_ context.Context, prompt string) (string, error) {
	client.calls = append(client.calls, prompt)
	if client.err != nil {
		return "", client.err
	}
	if len(client.responses) == 0 {
		return "", nil
	}
	response := client.responses[0]
	client.responses = client.responses[1:]
	return response, nil
}

func TestEnrichClustersUsesLLMResponses(t *testing.T) {
	client := &fakeLLM{responses: []string{
		`{"name": "Auth Module", "description": "Handles authentication"}`,
		`{"name": "Utility Helpers", "description": "Common utilities"}`,
	}}

	result := EnrichClusters(context.Background(), enrichmentCommunities(), enrichmentMemberMap(), client, nil)

	if len(result.Enrichments) != 2 {
		t.Fatalf("enrichments = %#v", result.Enrichments)
	}
	if got := result.Enrichments["comm_0"]; got.Name != "Auth Module" || got.Description != "Handles authentication" {
		t.Fatalf("auth enrichment = %#v", got)
	}
	if got := result.Enrichments["comm_1"]; got.Name != "Utility Helpers" || got.Description != "Common utilities" {
		t.Fatalf("utils enrichment = %#v", got)
	}
	if result.TokensUsed <= 0 {
		t.Fatalf("TokensUsed = %f, want > 0", result.TokensUsed)
	}
	if len(client.calls) != 2 {
		t.Fatalf("LLM calls = %d, want 2", len(client.calls))
	}
}

func TestEnrichClustersFallsBackOnInvalidOrEmptyResponses(t *testing.T) {
	tests := []struct {
		name      string
		responses []string
	}{
		{name: "invalid json", responses: []string{"this is not json at all", "also bad"}},
		{name: "empty string", responses: []string{"", ""}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &fakeLLM{responses: tt.responses}
			result := EnrichClusters(context.Background(), enrichmentCommunities(), enrichmentMemberMap(), client, nil)
			if got := result.Enrichments["comm_0"]; got.Name != "Authentication" || len(got.Keywords) != 0 || got.Description != "" {
				t.Fatalf("auth fallback = %#v", got)
			}
			if got := result.Enrichments["comm_1"]; got.Name != "Utilities" || len(got.Keywords) != 0 || got.Description != "" {
				t.Fatalf("utils fallback = %#v", got)
			}
		})
	}
}

func TestEnrichClustersSkipsLLMForEmptyMembers(t *testing.T) {
	client := &fakeLLM{responses: []string{`{"name": "Should Not Appear", "description": "nope"}`}}
	result := EnrichClusters(context.Background(), enrichmentCommunities(), map[string][]ClusterMemberInfo{
		"comm_0": {},
		"comm_1": {},
	}, client, nil)

	if len(result.Enrichments) != 2 {
		t.Fatalf("enrichments = %#v", result.Enrichments)
	}
	if got := result.Enrichments["comm_0"]; got.Name != "Authentication" || len(got.Keywords) != 0 || got.Description != "" {
		t.Fatalf("auth fallback = %#v", got)
	}
	if len(client.calls) != 0 {
		t.Fatalf("LLM calls = %d, want 0", len(client.calls))
	}
}

func TestEnrichClustersProgressAndZeroCommunities(t *testing.T) {
	client := &fakeLLM{responses: []string{
		`{"name": "X", "description": "Y"}`,
		`{"name": "Z", "description": "W"}`,
	}}
	progress := [][2]int{}

	EnrichClusters(context.Background(), enrichmentCommunities(), enrichmentMemberMap(), client, func(current int, total int) {
		progress = append(progress, [2]int{current, total})
	})

	if want := [][2]int{{1, 2}, {2, 2}}; !reflect.DeepEqual(progress, want) {
		t.Fatalf("progress = %#v, want %#v", progress, want)
	}

	emptyClient := &fakeLLM{}
	empty := EnrichClusters(context.Background(), nil, nil, emptyClient, nil)
	if len(empty.Enrichments) != 0 || len(emptyClient.calls) != 0 {
		t.Fatalf("empty result = %#v calls=%d", empty, len(emptyClient.calls))
	}
}

func TestEnrichClustersAllowsMissingDescription(t *testing.T) {
	client := &fakeLLM{responses: []string{
		`{"name": "Auth Only"}`,
		`{"name": "Utils Only"}`,
	}}
	result := EnrichClusters(context.Background(), enrichmentCommunities(), enrichmentMemberMap(), client, nil)
	if got := result.Enrichments["comm_0"]; got.Name != "Auth Only" || got.Description != "" {
		t.Fatalf("auth partial = %#v", got)
	}
}

func TestEnrichClustersBatchUsesLLMResponses(t *testing.T) {
	client := &fakeLLM{responses: []string{
		`[
		  {"id": "comm_0", "name": "Auth Module", "keywords": ["auth", "login"], "description": "Authentication logic"},
		  {"id": "comm_1", "name": "Utility Helpers", "keywords": ["utils"], "description": "Common utilities"}
		]`,
		`[
		  {"id": "comm_2", "name": "HTTP Router", "keywords": ["routing"], "description": "Request routing"}
		]`,
	}}

	result := EnrichClustersBatch(context.Background(), batchCommunities(), batchMemberMap(), client, 2, nil)

	if len(result.Enrichments) != 3 {
		t.Fatalf("enrichments = %#v", result.Enrichments)
	}
	if got := result.Enrichments["comm_0"]; got.Name != "Auth Module" || !reflect.DeepEqual(got.Keywords, []string{"auth", "login"}) || got.Description != "Authentication logic" {
		t.Fatalf("auth batch = %#v", got)
	}
	if got := result.Enrichments["comm_1"]; got.Name != "Utility Helpers" {
		t.Fatalf("utils batch = %#v", got)
	}
	if got := result.Enrichments["comm_2"]; got.Name != "HTTP Router" {
		t.Fatalf("router batch = %#v", got)
	}
	if result.TokensUsed <= 0 || len(client.calls) != 2 {
		t.Fatalf("TokensUsed=%f calls=%d, want tokens and two calls", result.TokensUsed, len(client.calls))
	}
}

func TestEnrichClustersBatchFallsBackOnFailureAndMissingItems(t *testing.T) {
	client := &fakeLLM{err: errors.New("LLM unavailable")}
	result := EnrichClustersBatch(context.Background(), batchCommunities(), batchMemberMap(), client, 5, nil)
	for id, want := range map[string]string{"comm_0": "Authentication", "comm_1": "Utilities", "comm_2": "Routing"} {
		if got := result.Enrichments[id]; got.Name != want || len(got.Keywords) != 0 || got.Description != "" {
			t.Fatalf("fallback %s = %#v, want %s", id, got, want)
		}
	}

	partialClient := &fakeLLM{responses: []string{`[{"id": "comm_0", "name": "Authentication", "description": "Auth"}]`}}
	partial := EnrichClustersBatch(context.Background(), batchCommunities(), batchMemberMap(), partialClient, 5, nil)
	if partial.Enrichments["comm_1"].Name != "Utilities" || partial.Enrichments["comm_2"].Name != "Routing" {
		t.Fatalf("missing batch items were not backfilled: %#v", partial.Enrichments)
	}
}

func enrichmentCommunities() []Community {
	return []Community{
		{ID: "comm_0", HeuristicLabel: "Authentication", Cohesion: 0.8, SymbolCount: 3},
		{ID: "comm_1", HeuristicLabel: "Utilities", Cohesion: 0.5, SymbolCount: 2},
	}
}

func enrichmentMemberMap() map[string][]ClusterMemberInfo {
	return map[string][]ClusterMemberInfo{
		"comm_0": {
			{Name: "login", FilePath: "src/auth.ts", Type: "Function"},
			{Name: "validate", FilePath: "src/auth.ts", Type: "Function"},
			{Name: "AuthService", FilePath: "src/auth.ts", Type: "Class"},
		},
		"comm_1": {
			{Name: "hash", FilePath: "src/utils.ts", Type: "Function"},
			{Name: "format", FilePath: "src/utils.ts", Type: "Function"},
		},
	}
}

func batchCommunities() []Community {
	return []Community{
		{ID: "comm_0", HeuristicLabel: "Authentication", Cohesion: 0.8, SymbolCount: 3},
		{ID: "comm_1", HeuristicLabel: "Utilities", Cohesion: 0.5, SymbolCount: 2},
		{ID: "comm_2", HeuristicLabel: "Routing", Cohesion: 0.6, SymbolCount: 2},
	}
}

func batchMemberMap() map[string][]ClusterMemberInfo {
	return map[string][]ClusterMemberInfo{
		"comm_0": {{Name: "login", FilePath: "src/auth.ts", Type: "Function"}},
		"comm_1": {{Name: "hash", FilePath: "src/utils.ts", Type: "Function"}},
		"comm_2": {{Name: "route", FilePath: "src/router.ts", Type: "Function"}},
	}
}
