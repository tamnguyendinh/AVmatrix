package communities

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
)

type ClusterEnrichment struct {
	Name        string
	Keywords    []string
	Description string
}

type EnrichmentResult struct {
	Enrichments map[string]ClusterEnrichment
	TokensUsed  float64
}

type LLMClient interface {
	Generate(ctx context.Context, prompt string) (string, error)
}

type ClusterMemberInfo struct {
	Name     string
	FilePath string
	Type     string
}

func EnrichClusters(ctx context.Context, communities []Community, memberMap map[string][]ClusterMemberInfo, llmClient LLMClient, onProgress func(current int, total int)) EnrichmentResult {
	result := EnrichmentResult{Enrichments: make(map[string]ClusterEnrichment)}
	for i, community := range communities {
		members := memberMap[community.ID]
		if onProgress != nil {
			onProgress(i+1, len(communities))
		}
		if len(members) == 0 {
			result.Enrichments[community.ID] = heuristicEnrichment(community)
			continue
		}
		prompt := buildEnrichmentPrompt(members, community.HeuristicLabel)
		response, err := llmClient.Generate(ctx, prompt)
		if err != nil {
			result.Enrichments[community.ID] = heuristicEnrichment(community)
			continue
		}
		result.TokensUsed += tokenEstimate(prompt, response)
		result.Enrichments[community.ID] = parseEnrichmentResponse(response, community.HeuristicLabel)
	}
	return result
}

func EnrichClustersBatch(ctx context.Context, communities []Community, memberMap map[string][]ClusterMemberInfo, llmClient LLMClient, batchSize int, onProgress func(current int, total int)) EnrichmentResult {
	if batchSize <= 0 {
		batchSize = 5
	}
	result := EnrichmentResult{Enrichments: make(map[string]ClusterEnrichment)}
	for i := 0; i < len(communities); i += batchSize {
		end := i + batchSize
		if end > len(communities) {
			end = len(communities)
		}
		if onProgress != nil {
			onProgress(end, len(communities))
		}
		batch := communities[i:end]
		prompt := buildBatchEnrichmentPrompt(batch, memberMap)
		response, err := llmClient.Generate(ctx, prompt)
		if err != nil {
			for _, community := range batch {
				result.Enrichments[community.ID] = heuristicEnrichment(community)
			}
			continue
		}
		result.TokensUsed += tokenEstimate(prompt, response)
		for id, enrichment := range parseBatchEnrichmentResponse(response) {
			result.Enrichments[id] = enrichment
		}
	}
	for _, community := range communities {
		if _, ok := result.Enrichments[community.ID]; !ok {
			result.Enrichments[community.ID] = heuristicEnrichment(community)
		}
	}
	return result
}

func buildEnrichmentPrompt(members []ClusterMemberInfo, heuristicLabel string) string {
	limited := members
	if len(limited) > 20 {
		limited = limited[:20]
	}
	memberList := formatMembers(limited)
	more := ""
	if len(members) > 20 {
		more = " (+" + strconv.Itoa(len(members)-20) + " more)"
	}
	return "Analyze this code cluster and provide a semantic name and short description.\n\n" +
		"Heuristic: \"" + heuristicLabel + "\"\n" +
		"Members: " + memberList + more + "\n\n" +
		"Reply with JSON only:\n" +
		"{\"name\": \"2-4 word semantic name\", \"description\": \"One sentence describing purpose\"}"
}

func buildBatchEnrichmentPrompt(communities []Community, memberMap map[string][]ClusterMemberInfo) string {
	parts := make([]string, 0, len(communities))
	for index, community := range communities {
		members := memberMap[community.ID]
		if len(members) > 15 {
			members = members[:15]
		}
		parts = append(parts, "Cluster "+strconv.Itoa(index+1)+" (id: "+community.ID+"):\n"+
			"Heuristic: \""+community.HeuristicLabel+"\"\n"+
			"Members: "+formatMembers(members))
	}
	return "Analyze these code clusters and generate semantic names, keywords, and descriptions.\n\n" +
		strings.Join(parts, "\n\n") +
		"\n\nOutput JSON array:\n" +
		"[\n" +
		"  {\"id\": \"comm_X\", \"name\": \"...\", \"keywords\": [...], \"description\": \"...\"},\n" +
		"  ...\n" +
		"]"
}

func formatMembers(members []ClusterMemberInfo) string {
	parts := make([]string, 0, len(members))
	for _, member := range members {
		parts = append(parts, member.Name+" ("+member.Type+")")
	}
	return strings.Join(parts, ", ")
}

func parseEnrichmentResponse(response string, fallbackLabel string) ClusterEnrichment {
	start := strings.Index(response, "{")
	end := strings.LastIndex(response, "}")
	if start < 0 || end < start {
		return fallbackEnrichment(fallbackLabel)
	}
	var parsed struct {
		Name        string   `json:"name"`
		Keywords    []string `json:"keywords"`
		Description string   `json:"description"`
	}
	if err := json.Unmarshal([]byte(response[start:end+1]), &parsed); err != nil {
		return fallbackEnrichment(fallbackLabel)
	}
	if parsed.Name == "" {
		parsed.Name = fallbackLabel
	}
	if parsed.Keywords == nil {
		parsed.Keywords = []string{}
	}
	return ClusterEnrichment{
		Name:        parsed.Name,
		Keywords:    parsed.Keywords,
		Description: parsed.Description,
	}
}

func parseBatchEnrichmentResponse(response string) map[string]ClusterEnrichment {
	start := strings.Index(response, "[")
	end := strings.LastIndex(response, "]")
	if start < 0 || end < start {
		return map[string]ClusterEnrichment{}
	}
	var parsed []struct {
		ID          string   `json:"id"`
		Name        string   `json:"name"`
		Keywords    []string `json:"keywords"`
		Description string   `json:"description"`
	}
	if err := json.Unmarshal([]byte(response[start:end+1]), &parsed); err != nil {
		return map[string]ClusterEnrichment{}
	}
	out := make(map[string]ClusterEnrichment, len(parsed))
	for _, item := range parsed {
		if item.ID == "" {
			continue
		}
		if item.Keywords == nil {
			item.Keywords = []string{}
		}
		out[item.ID] = ClusterEnrichment{
			Name:        item.Name,
			Keywords:    item.Keywords,
			Description: item.Description,
		}
	}
	return out
}

func heuristicEnrichment(community Community) ClusterEnrichment {
	return fallbackEnrichment(community.HeuristicLabel)
}

func fallbackEnrichment(label string) ClusterEnrichment {
	return ClusterEnrichment{Name: label, Keywords: []string{}, Description: ""}
}

func tokenEstimate(prompt string, response string) float64 {
	return float64(len(prompt))/4 + float64(len(response))/4
}
