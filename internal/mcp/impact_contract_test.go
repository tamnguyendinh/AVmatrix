package mcp

import (
	"reflect"
	"testing"
)

func TestParseImpactArgsAcceptsTargetUIDOnlyAndDefaults(t *testing.T) {
	options, validation := parseImpactArgs(map[string]any{
		"target_uid": "uid:1234",
		"direction":  "upstream",
	})
	if validation != nil {
		t.Fatalf("parseImpactArgs validation = %#v", validation)
	}
	if options.Target != "" {
		t.Fatalf("Target = %q, want empty", options.Target)
	}
	if options.TargetUID != "uid:1234" {
		t.Fatalf("TargetUID = %q", options.TargetUID)
	}
	if options.Direction != "upstream" || options.MaxDepth != 3 || options.IncludeTests {
		t.Fatalf("defaults = %#v", options)
	}
	if options.MinConfidence != 0 {
		t.Fatalf("MinConfidence = %v, want 0", options.MinConfidence)
	}
	if !reflect.DeepEqual(options.RelationTypes, impactDefaultRelationTypes) {
		t.Fatalf("RelationTypes = %#v, want %#v", options.RelationTypes, impactDefaultRelationTypes)
	}
}

func TestParseImpactArgsValidationContract(t *testing.T) {
	tests := []struct {
		name      string
		args      map[string]any
		wantField string
	}{
		{
			name:      "missing target",
			args:      map[string]any{"direction": "upstream"},
			wantField: "target",
		},
		{
			name:      "invalid direction",
			args:      map[string]any{"target": "AuthService", "direction": "upstrem"},
			wantField: "direction",
		},
		{
			name: "invalid relation type",
			args: map[string]any{
				"target":        "AuthService",
				"direction":     "upstream",
				"relationTypes": []any{"NOT_A_RELATION"},
			},
			wantField: "relationTypes",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, validation := parseImpactArgs(tt.args)
			if validation == nil {
				t.Fatalf("parseImpactArgs(%s) validation = nil", tt.name)
			}
			if validation["field"] != tt.wantField {
				t.Fatalf("field = %#v, want %q; validation = %#v", validation["field"], tt.wantField, validation)
			}
		})
	}
}

func TestParseImpactArgsRelationTypeAliasesAndScopeFilters(t *testing.T) {
	options, validation := parseImpactArgs(map[string]any{
		"target":        "AuthService",
		"direction":     "upstream",
		"relationTypes": []any{"OVERRIDES"},
	})
	if validation != nil {
		t.Fatalf("OVERRIDES validation = %#v", validation)
	}
	if !reflect.DeepEqual(options.RelationTypes, []string{"OVERRIDES", "METHOD_OVERRIDES"}) {
		t.Fatalf("OVERRIDES relation types = %#v", options.RelationTypes)
	}

	options, validation = parseImpactArgs(map[string]any{
		"target":        "User",
		"direction":     "upstream",
		"relationTypes": []any{"USES", "INHERITS"},
	})
	if validation != nil {
		t.Fatalf("USES/INHERITS validation = %#v", validation)
	}
	if !reflect.DeepEqual(options.RelationTypes, []string{"USES", "INHERITS"}) {
		t.Fatalf("USES/INHERITS relation types = %#v", options.RelationTypes)
	}
}

func TestImpactRelationTypeSecurityContract(t *testing.T) {
	expected := []string{
		"CALLS",
		"IMPORTS",
		"USES",
		"INHERITS",
		"EXTENDS",
		"IMPLEMENTS",
		"HAS_METHOD",
		"HAS_PROPERTY",
		"METHOD_OVERRIDES",
		"OVERRIDES",
		"METHOD_IMPLEMENTS",
		"ACCESSES",
		"HANDLES_ROUTE",
		"FETCHES",
		"HANDLES_TOOL",
		"ENTRY_POINT_OF",
		"WRAPS",
	}
	if len(impactAllowedRelationTypes) != len(expected) {
		t.Fatalf("allowed relation type count = %d, want %d", len(impactAllowedRelationTypes), len(expected))
	}
	for _, relationType := range expected {
		if !impactAllowedRelationTypes[relationType] {
			t.Fatalf("allowed relation types missing %s", relationType)
		}
	}
	for _, relationType := range []string{"CONTAINS", "calls", "DROP_TABLE", "STEP_IN_PROCESS", "MEMBER_OF"} {
		if impactAllowedRelationTypes[relationType] {
			t.Fatalf("allowed relation types unexpectedly include %s", relationType)
		}
	}
}

func TestImpactTestPathSecurityContract(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{path: "src/foo.test.ts", want: true},
		{path: "src/foo.spec.ts", want: true},
		{path: "src/__tests__/foo.ts", want: true},
		{path: "src/test/foo.ts", want: true},
		{path: `src\test\foo.ts`, want: true},
		{path: `src\__tests__\bar.ts`, want: true},
		{path: "SRC/TEST/Foo.ts", want: true},
		{path: "SRC/Foo.Test.ts", want: true},
		{path: "pkg/handler_test.go", want: true},
		{path: "tests/test_handler.py", want: true},
		{path: "pkg/handler_test.py", want: true},
		{path: "src/main.ts", want: false},
		{path: "src/utils/helper.ts", want: false},
	}
	for _, tt := range tests {
		if got := isImpactTestPath(tt.path); got != tt.want {
			t.Fatalf("isImpactTestPath(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}
