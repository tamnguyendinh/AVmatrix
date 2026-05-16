package mcp

import (
	"encoding/json"
	"fmt"
)

type promptDefinition struct {
	Name        string           `json:"name"`
	Description string           `json:"description"`
	Arguments   []promptArgument `json:"arguments"`
}

type promptArgument struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
}

type promptMessage struct {
	Role    string        `json:"role"`
	Content promptContent `json:"content"`
}

type promptContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func promptDefinitions() []promptDefinition {
	return []promptDefinition{
		{
			Name:        "detect_impact",
			Description: "Analyze the impact of your current changes before committing. Guides through scope selection, change detection, process analysis, and risk assessment.",
			Arguments: []promptArgument{
				{Name: "scope", Description: "What to analyze: unstaged, staged, all, or compare", Required: false},
				{Name: "base_ref", Description: "Branch/commit for compare scope", Required: false},
			},
		},
		{
			Name:        "generate_map",
			Description: "Generate architecture documentation from the knowledge graph. Creates a codebase overview with execution flows and mermaid diagrams.",
			Arguments: []promptArgument{
				{Name: "repo", Description: "Repository name (omit if only one indexed)", Required: false},
			},
		},
	}
}

func getPrompt(raw json.RawMessage) (map[string]any, error) {
	var params struct {
		Name      string         `json:"name"`
		Arguments map[string]any `json:"arguments"`
	}
	if err := json.Unmarshal(raw, &params); err != nil {
		return nil, fmt.Errorf("Invalid prompt params: %w", err)
	}

	switch params.Name {
	case "detect_impact":
		return map[string]any{"messages": []promptMessage{detectImpactPrompt(params.Arguments)}}, nil
	case "generate_map":
		return map[string]any{"messages": []promptMessage{generateMapPrompt(params.Arguments)}}, nil
	default:
		return nil, fmt.Errorf("Unknown prompt: %s", params.Name)
	}
}

func detectImpactPrompt(args map[string]any) promptMessage {
	scope := stringArg(args, "scope", "all")
	baseRef := stringArg(args, "base_ref", "")
	payload := map[string]any{"scope": scope}
	if baseRef != "" {
		payload["base_ref"] = baseRef
	}
	raw, _ := json.Marshal(payload)

	return promptMessage{
		Role: "user",
		Content: promptContent{
			Type: "text",
			Text: "Analyze the impact of my current code changes before committing.\n\n" +
				"Follow these steps:\n" +
				"1. Run `detect_changes(" + string(raw) + ")` to find what changed and affected processes\n" +
				"2. For each changed symbol in critical processes, run `context({name: \"<symbol>\"})` to see its full reference graph\n" +
				"3. For any high-risk items, run `impact({target: \"<symbol>\", direction: \"upstream\"})` for blast radius\n" +
				"4. Summarize: changes, affected processes, risk level, and recommended actions\n\n" +
				"Present the analysis as a clear risk report.",
		},
	}
}

func generateMapPrompt(args map[string]any) promptMessage {
	repo := stringArg(args, "repo", "")
	repoName := repo
	if repoName == "" {
		repoName = "{name}"
	}

	return promptMessage{
		Role: "user",
		Content: promptContent{
			Type: "text",
			Text: "Generate architecture documentation for this codebase using the knowledge graph.\n\n" +
				"Follow these steps:\n" +
				"1. READ `" + canonicalResourceScheme + "://repo/" + repoName + "/context` for codebase stats\n" +
				"2. READ `" + canonicalResourceScheme + "://repo/" + repoName + "/clusters` to see all functional areas\n" +
				"3. READ `" + canonicalResourceScheme + "://repo/" + repoName + "/processes` to see all execution flows\n" +
				"4. For the top 5 most important processes, READ `" + canonicalResourceScheme + "://repo/" + repoName + "/process/{name}` for step-by-step traces\n" +
				"5. Generate a mermaid architecture diagram showing the major areas and their connections\n" +
				"6. Write an ARCHITECTURE.md file with: overview, functional areas, key execution flows, and the mermaid diagram",
		},
	}
}

func stringArg(args map[string]any, name string, fallback string) string {
	if args == nil {
		return fallback
	}
	value, ok := args[name].(string)
	if !ok || value == "" {
		return fallback
	}
	return value
}
