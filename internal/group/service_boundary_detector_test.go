package group

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

func TestDetectServiceBoundariesMarkersAndExclusions(t *testing.T) {
	tmpDir := t.TempDir()
	writeGroupFile(t, tmpDir, "services/auth/package.json", "{}")
	writeGroupFile(t, tmpDir, "services/auth/Dockerfile", "FROM node:20")
	writeGroupFile(t, tmpDir, "services/auth/src/index.ts", "export default {}")
	writeGroupFile(t, tmpDir, "services/orders/package.json", "{}")
	writeGroupFile(t, tmpDir, "services/orders/src/main.ts", "console.log('ok')")
	writeGroupFile(t, tmpDir, "services/api/go.mod", "module example.com/api")
	writeGroupFile(t, tmpDir, "services/api/main.go", "package main")
	writeGroupFile(t, tmpDir, "microservices/billing/pom.xml", "<project/>")
	writeGroupFile(t, tmpDir, "microservices/billing/src/Main.java", "class Main {}")
	writeGroupFile(t, tmpDir, "apps/worker/Dockerfile", "FROM python:3.12")
	writeGroupFile(t, tmpDir, "apps/worker/requirements.txt", "flask")
	writeGroupFile(t, tmpDir, "apps/worker/app.py", "print('worker')")
	writeGroupFile(t, tmpDir, "crates/parser/Cargo.toml", "[package]")
	writeGroupFile(t, tmpDir, "crates/parser/src/lib.rs", "pub fn parse() {}")
	writeGroupFile(t, tmpDir, "modules/gateway/build.gradle", "apply plugin: 'java'")
	writeGroupFile(t, tmpDir, "modules/gateway/src/Main.java", "class Main {}")
	writeGroupFile(t, tmpDir, "services/ml/pyproject.toml", "[project]")
	writeGroupFile(t, tmpDir, "services/ml/src/model.py", "")
	writeGroupFile(t, tmpDir, "package.json", "{}")
	writeGroupFile(t, tmpDir, "src/index.ts", "export default {}")
	writeGroupFile(t, tmpDir, "vendor/some-dep/package.json", "{}")
	writeGroupFile(t, tmpDir, "vendor/some-dep/src/lib.go", "")
	writeGroupFile(t, tmpDir, "target/classes/Main.java", "")
	writeGroupFile(t, tmpDir, "target/pom.xml", "<project/>")
	writeGroupFile(t, tmpDir, "__pycache__/package.json", "{}")
	writeGroupFile(t, tmpDir, "__pycache__/cached.py", "")
	writeGroupFile(t, tmpDir, ".hidden/package.json", "{}")
	writeGroupFile(t, tmpDir, ".hidden/src/index.ts", "")

	boundaries, err := DetectServiceBoundaries(tmpDir)
	if err != nil {
		t.Fatalf("DetectServiceBoundaries() error = %v", err)
	}
	paths := make([]string, 0, len(boundaries))
	for _, boundary := range boundaries {
		paths = append(paths, boundary.ServicePath)
	}
	sort.Strings(paths)
	for _, want := range []string{
		"apps/worker",
		"crates/parser",
		"microservices/billing",
		"modules/gateway",
		"services/api",
		"services/auth",
		"services/ml",
		"services/orders",
	} {
		if !containsString(paths, want) {
			t.Fatalf("boundaries missing %q: %v", want, paths)
		}
	}
	for _, disallowed := range []string{"vendor/some-dep", "target", "__pycache__", ".hidden"} {
		if containsString(paths, disallowed) {
			t.Fatalf("boundaries included excluded path %q: %v", disallowed, paths)
		}
	}

	auth := serviceBoundaryByPath(boundaries, "services/auth")
	api := serviceBoundaryByPath(boundaries, "services/api")
	if auth == nil || api == nil {
		t.Fatalf("missing auth/api boundaries: %#v", boundaries)
	}
	if auth.Confidence <= api.Confidence {
		t.Fatalf("multi-marker confidence = %v, single-marker confidence = %v", auth.Confidence, api.Confidence)
	}
	if !containsString(auth.Markers, "Dockerfile") || !containsString(auth.Markers, "package.json") {
		t.Fatalf("auth markers = %#v", auth.Markers)
	}
}

func TestDetectServiceBoundariesRootAndEmptyRepo(t *testing.T) {
	rootOnly := t.TempDir()
	writeGroupFile(t, rootOnly, "package.json", "{}")
	writeGroupFile(t, rootOnly, "src/index.ts", "export default {}")
	boundaries, err := DetectServiceBoundaries(rootOnly)
	if err != nil {
		t.Fatalf("DetectServiceBoundaries(rootOnly) error = %v", err)
	}
	if len(boundaries) != 0 {
		t.Fatalf("root package should not be a service boundary: %#v", boundaries)
	}

	empty := t.TempDir()
	boundaries, err = DetectServiceBoundaries(empty)
	if err != nil {
		t.Fatalf("DetectServiceBoundaries(empty) error = %v", err)
	}
	if len(boundaries) != 0 {
		t.Fatalf("empty repo boundaries = %#v", boundaries)
	}
}

func TestAssignServiceUsesDeepestBoundary(t *testing.T) {
	boundaries := []ServiceBoundary{
		{ServicePath: "platform", ServiceName: "platform"},
		{ServicePath: "platform/services/auth", ServiceName: "auth"},
		{ServicePath: "services/orders", ServiceName: "orders"},
	}
	tests := []struct {
		filePath string
		want     string
	}{
		{"platform/services/auth/src/index.ts", "platform/services/auth"},
		{"platform/shared/utils.ts", "platform"},
		{"services/orders/src/main.ts", "services/orders"},
		{"libs/shared/utils.ts", ""},
		{"README.md", ""},
	}
	for _, tt := range tests {
		if got := AssignService(tt.filePath, boundaries); got != tt.want {
			t.Fatalf("AssignService(%q) = %q, want %q", tt.filePath, got, tt.want)
		}
	}
}

func TestDetectServiceBoundariesNestedIncludesDeepestMatch(t *testing.T) {
	tmpDir := t.TempDir()
	writeGroupFile(t, tmpDir, "platform/package.json", "{}")
	writeGroupFile(t, tmpDir, "platform/src/shared.ts", "")
	writeGroupFile(t, tmpDir, "platform/services/auth/package.json", "{}")
	writeGroupFile(t, tmpDir, "platform/services/auth/src/index.ts", "")
	boundaries, err := DetectServiceBoundaries(tmpDir)
	if err != nil {
		t.Fatalf("DetectServiceBoundaries() error = %v", err)
	}
	paths := make([]string, 0, len(boundaries))
	for _, boundary := range boundaries {
		paths = append(paths, boundary.ServicePath)
	}
	sort.Strings(paths)
	if !reflect.DeepEqual(paths, []string{"platform", "platform/services/auth"}) {
		t.Fatalf("nested boundaries = %v", paths)
	}
}

func writeGroupFile(t *testing.T, root string, rel string, content string) {
	t.Helper()
	full := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(full), err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", rel, err)
	}
}

func serviceBoundaryByPath(boundaries []ServiceBoundary, servicePath string) *ServiceBoundary {
	for i := range boundaries {
		if boundaries[i].ServicePath == servicePath {
			return &boundaries[i]
		}
	}
	return nil
}

func containsString(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}
