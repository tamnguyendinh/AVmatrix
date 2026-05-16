package mcp

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"
)

type baselineMCPSurface struct {
	Tools             []baselineMCPTool   `json:"tools"`
	Resources         []string            `json:"resources"`
	ResourceTemplates []string            `json:"resourceTemplates"`
	Prompts           []baselineMCPPrompt `json:"prompts"`
}

type baselineMCPTool struct {
	Name       string   `json:"name"`
	Properties []string `json:"properties"`
	Required   []string `json:"required"`
}

type baselineMCPPrompt struct {
	Name      string   `json:"name"`
	Arguments []string `json:"arguments"`
}

func TestMCPSurfaceMatchesTypeScriptBaselineSnapshot(t *testing.T) {
	var baseline baselineMCPSurface
	raw, err := os.ReadFile("testdata/typescript_baseline_surface.json")
	if err != nil {
		t.Fatalf("read baseline snapshot: %v", err)
	}
	if err := json.Unmarshal(raw, &baseline); err != nil {
		t.Fatalf("parse baseline snapshot: %v", err)
	}

	tools := mcpTools()
	actualToolNames := make([]string, 0, len(tools))
	actualToolsByName := make(map[string]toolDefinition, len(tools))
	for _, tool := range tools {
		actualToolNames = append(actualToolNames, tool.Name)
		actualToolsByName[tool.Name] = tool
	}
	if !reflect.DeepEqual(actualToolNames, baselineToolNames(baseline.Tools)) {
		t.Fatalf("tool discovery order drifted\nactual:   %#v\nbaseline: %#v", actualToolNames, baselineToolNames(baseline.Tools))
	}
	for _, expectedTool := range baseline.Tools {
		actualTool := actualToolsByName[expectedTool.Name]
		assertContainsAll(t, schemaRequiredNames(actualTool.InputSchema), expectedTool.Required, expectedTool.Name+" required")
		assertContainsAll(t, schemaPropertyNames(actualTool.InputSchema), expectedTool.Properties, expectedTool.Name+" properties")
	}

	if !reflect.DeepEqual(resourceURIs(resourceDefinitions()), baseline.Resources) {
		t.Fatalf("resources drifted\nactual:   %#v\nbaseline: %#v", resourceURIs(resourceDefinitions()), baseline.Resources)
	}
	if !reflect.DeepEqual(resourceTemplateURIs(resourceTemplates()), baseline.ResourceTemplates) {
		t.Fatalf("resource templates drifted\nactual:   %#v\nbaseline: %#v", resourceTemplateURIs(resourceTemplates()), baseline.ResourceTemplates)
	}
	if !reflect.DeepEqual(promptSnapshots(promptDefinitions()), baseline.Prompts) {
		t.Fatalf("prompts drifted\nactual:   %#v\nbaseline: %#v", promptSnapshots(promptDefinitions()), baseline.Prompts)
	}
}

func TestGroupMCPToolsAreRegisteredWithSchemas(t *testing.T) {
	toolsByName := make(map[string]toolDefinition)
	for _, tool := range mcpTools() {
		toolsByName[tool.Name] = tool
	}
	for _, name := range []string{"group_list", "group_sync", "group_contracts", "group_query", "group_status"} {
		tool, ok := toolsByName[name]
		if !ok {
			t.Fatalf("group MCP tool %q is not registered", name)
		}
		if len(tool.Description) <= 10 {
			t.Fatalf("group MCP tool %q description is too short: %q", name, tool.Description)
		}
		if tool.InputSchema["type"] != "object" {
			t.Fatalf("group MCP tool %q schema type = %#v", name, tool.InputSchema["type"])
		}
	}
	required := schemaRequiredNames(toolsByName["group_sync"].InputSchema)
	if !containsString(required, "name") {
		t.Fatalf("group_sync required = %#v, want name", required)
	}
}

func TestMCPToolSchemasKeepRuntimeContract(t *testing.T) {
	toolsByName := make(map[string]toolDefinition)
	for _, tool := range mcpTools() {
		toolsByName[tool.Name] = tool
		if tool.Name == "" || tool.Description == "" {
			t.Fatalf("tool missing name/description: %#v", tool)
		}
		if tool.InputSchema["type"] != "object" {
			t.Fatalf("%s schema type = %#v, want object", tool.Name, tool.InputSchema["type"])
		}
		if _, ok := tool.InputSchema["properties"].(map[string]any); !ok {
			t.Fatalf("%s schema missing properties", tool.Name)
		}
		if tool.InputSchema["required"] == nil {
			t.Fatalf("%s schema missing required", tool.Name)
		}
	}

	if len(toolsByName) != 16 {
		t.Fatalf("tool count = %d, want 16", len(toolsByName))
	}
	for _, name := range []string{"list_repos", "query", "cypher", "context", "detect_changes", "rename", "impact", "route_map", "tool_map", "shape_check", "api_impact"} {
		if _, ok := toolsByName[name]; !ok {
			t.Fatalf("missing MCP tool %q", name)
		}
	}

	assertContainsAll(t, schemaRequiredNames(toolsByName["query"].InputSchema), []string{"query"}, "query required")
	assertContainsAll(t, schemaRequiredNames(toolsByName["cypher"].InputSchema), []string{"query"}, "cypher required")
	assertContainsAll(t, schemaRequiredNames(toolsByName["rename"].InputSchema), []string{"new_name"}, "rename required")
	if required := schemaRequiredNames(toolsByName["context"].InputSchema); len(required) != 0 {
		t.Fatalf("context required = %#v, want none", required)
	}
	if required := schemaRequiredNames(toolsByName["detect_changes"].InputSchema); len(required) != 0 {
		t.Fatalf("detect_changes required = %#v, want none", required)
	}
	if properties := schemaProperties(toolsByName["list_repos"].InputSchema); len(properties) != 0 {
		t.Fatalf("list_repos properties = %#v, want none", properties)
	}

	impact := toolsByName["impact"]
	assertContainsAll(t, schemaRequiredNames(impact.InputSchema), []string{"direction"}, "impact required")
	impactProperties := schemaProperties(impact.InputSchema)
	if impactProperties["target"] == nil || impactProperties["target_uid"] == nil {
		t.Fatalf("impact target properties = %#v", impactProperties)
	}
	if oneOf, ok := impact.InputSchema["oneOf"].([]map[string]any); !ok || len(oneOf) != 2 {
		t.Fatalf("impact oneOf = %#v", impact.InputSchema["oneOf"])
	}
	direction, _ := impactProperties["direction"].(map[string]any)
	if !reflect.DeepEqual(stringSliceFromSchema(direction["enum"]), []string{"upstream", "downstream"}) {
		t.Fatalf("impact direction enum = %#v", direction["enum"])
	}
	if direction["default"] != "upstream" {
		t.Fatalf("impact direction default = %#v", direction["default"])
	}
	maxDepth, _ := impactProperties["maxDepth"].(map[string]any)
	if maxDepth["default"] != 3 {
		t.Fatalf("impact maxDepth default = %#v", maxDepth["default"])
	}
	relationTypes, _ := impactProperties["relationTypes"].(map[string]any)
	items, _ := relationTypes["items"].(map[string]any)
	if relationTypes["type"] != "array" || items["type"] != "string" || len(stringSliceFromSchema(items["enum"])) == 0 {
		t.Fatalf("impact relationTypes schema = %#v", relationTypes)
	}

	for _, name := range []string{"query", "cypher", "context", "detect_changes", "rename", "impact", "route_map", "tool_map", "shape_check", "api_impact"} {
		properties := schemaProperties(toolsByName[name].InputSchema)
		if repoProp, ok := properties["repo"].(map[string]any); !ok || repoProp["type"] != "string" {
			t.Fatalf("%s repo property = %#v", name, properties["repo"])
		}
		if containsString(schemaRequiredNames(toolsByName[name].InputSchema), "repo") {
			t.Fatalf("%s should not require repo", name)
		}
	}
	for _, name := range []string{"group_list", "group_status", "group_sync", "group_query"} {
		if _, ok := schemaProperties(toolsByName[name].InputSchema)["repo"]; ok {
			t.Fatalf("%s should not expose backend repo property", name)
		}
	}
	if _, ok := schemaProperties(toolsByName["group_contracts"].InputSchema)["repo"]; !ok {
		t.Fatal("group_contracts should expose contract repo filter")
	}
	assertContainsAll(t, schemaRequiredNames(toolsByName["group_query"].InputSchema), []string{"name", "query"}, "group_query required")
	scope, _ := schemaProperties(toolsByName["detect_changes"].InputSchema)["scope"].(map[string]any)
	if !reflect.DeepEqual(stringSliceFromSchema(scope["enum"]), []string{"unstaged", "staged", "all", "compare"}) {
		t.Fatalf("detect_changes scope enum = %#v", scope["enum"])
	}
}

func baselineToolNames(tools []baselineMCPTool) []string {
	names := make([]string, 0, len(tools))
	for _, tool := range tools {
		names = append(names, tool.Name)
	}
	return names
}

func schemaPropertyNames(schema map[string]any) []string {
	properties := schemaProperties(schema)
	names := make([]string, 0, len(properties))
	for name := range properties {
		names = append(names, name)
	}
	return names
}

func schemaProperties(schema map[string]any) map[string]any {
	properties, _ := schema["properties"].(map[string]any)
	return properties
}

func schemaRequiredNames(schema map[string]any) []string {
	return stringSliceFromSchema(schema["required"])
}

func stringSliceFromSchema(value any) []string {
	switch typed := value.(type) {
	case []string:
		return typed
	case []any:
		out := make([]string, 0, len(typed))
		for _, entry := range typed {
			if text, ok := entry.(string); ok {
				out = append(out, text)
			}
		}
		return out
	default:
		return nil
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func resourceURIs(resources []resourceDefinition) []string {
	uris := make([]string, 0, len(resources))
	for _, resource := range resources {
		uris = append(uris, resource.URI)
	}
	return uris
}

func resourceTemplateURIs(templates []resourceTemplate) []string {
	uris := make([]string, 0, len(templates))
	for _, template := range templates {
		uris = append(uris, template.URITemplate)
	}
	return uris
}

func promptSnapshots(prompts []promptDefinition) []baselineMCPPrompt {
	snapshots := make([]baselineMCPPrompt, 0, len(prompts))
	for _, prompt := range prompts {
		arguments := make([]string, 0, len(prompt.Arguments))
		for _, argument := range prompt.Arguments {
			arguments = append(arguments, argument.Name)
		}
		snapshots = append(snapshots, baselineMCPPrompt{Name: prompt.Name, Arguments: arguments})
	}
	return snapshots
}

func assertContainsAll(t *testing.T, actual []string, expected []string, label string) {
	t.Helper()
	seen := make(map[string]bool, len(actual))
	for _, item := range actual {
		seen[item] = true
	}
	for _, item := range expected {
		if !seen[item] {
			t.Fatalf("%s missing %q\nactual:   %#v\nbaseline: %#v", label, item, actual, expected)
		}
	}
}
