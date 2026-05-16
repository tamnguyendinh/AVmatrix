package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestCleanGoSourcePackageRemovesOnlyPackageGoSrc(t *testing.T) {
	root := t.TempDir()
	goSrc := filepath.Join(root, "go-src")
	nested := filepath.Join(goSrc, "internal")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("mkdir go-src: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nested, "x.go"), []byte("package x\n"), 0o644); err != nil {
		t.Fatalf("write go-src file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "package.json"), []byte(`{"name":"avmatrix"}`), 0o644); err != nil {
		t.Fatalf("write package.json: %v", err)
	}

	var out bytes.Buffer
	if err := cleanGoSourcePackage(root, &out); err != nil {
		t.Fatalf("cleanGoSourcePackage returned error: %v", err)
	}
	if _, err := os.Stat(goSrc); !os.IsNotExist(err) {
		t.Fatalf("go-src still exists or stat returned unexpected error: %v", err)
	}
	if !strings.Contains(out.String(), "[clean-go-source-package] removed ") {
		t.Fatalf("cleanup output missing status line: %q", out.String())
	}
	if _, err := os.Stat(filepath.Join(root, "package.json")); err != nil {
		t.Fatalf("package root file was removed: %v", err)
	}
}

func TestPackageCleanGoSourceCommandUsesWorkingDirectoryPackageRoot(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "package.json"), []byte(`{"name":"avmatrix"}`), 0o644); err != nil {
		t.Fatalf("write package.json: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "go-src"), 0o755); err != nil {
		t.Fatalf("mkdir go-src: %v", err)
	}
	previous, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir package root: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(previous)
	})

	out, errOut, err := executeForTest(t, "package", "clean-go-source")
	if err != nil {
		t.Fatalf("package clean-go-source returned error: %v\nstdout:\n%s\nstderr:\n%s", err, out, errOut)
	}
	if errOut != "" {
		t.Fatalf("package clean-go-source wrote stderr: %q", errOut)
	}
	if _, err := os.Stat(filepath.Join(root, "go-src")); !os.IsNotExist(err) {
		t.Fatalf("go-src still exists or stat returned unexpected error: %v", err)
	}
}

func TestEnsurePackagedRuntimeAcceptsCurrentPlatformMetadata(t *testing.T) {
	root := t.TempDir()
	binDir := filepath.Join(root, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}
	if err := os.WriteFile(filepath.Join(binDir, "avmatrix.exe"), []byte("runtime"), 0o644); err != nil {
		t.Fatalf("write runtime: %v", err)
	}
	metadata := packageRuntimeMetadata{
		Platform: runtime.GOOS,
		Arch:     runtime.GOARCH,
		Binary:   "avmatrix.exe",
		Source:   "..",
		Tags:     []string{"ladybugdb"},
	}
	raw, err := json.Marshal(metadata)
	if err != nil {
		t.Fatalf("marshal metadata: %v", err)
	}
	if err := os.WriteFile(filepath.Join(binDir, "avmatrix-runtime.json"), raw, 0o644); err != nil {
		t.Fatalf("write metadata: %v", err)
	}

	var out bytes.Buffer
	if err := ensurePackagedRuntime(root, &out); err != nil {
		t.Fatalf("ensurePackagedRuntime returned error: %v", err)
	}
	if !strings.Contains(out.String(), "[package-runtime] using packaged Go runtime") {
		t.Fatalf("ensure output missing status: %q", out.String())
	}
}

func TestPrepareGoSourcePackageCopiesMinimalGoSource(t *testing.T) {
	parent := t.TempDir()
	packageRoot := filepath.Join(parent, "avmatrix")
	if err := os.MkdirAll(packageRoot, 0o755); err != nil {
		t.Fatalf("mkdir package root: %v", err)
	}
	if err := os.WriteFile(filepath.Join(packageRoot, "package.json"), []byte(`{"name":"avmatrix"}`), 0o644); err != nil {
		t.Fatalf("write package.json: %v", err)
	}
	writePackageTestFile(t, parent, "go.mod", "module example.com/avmatrix\n")
	writePackageTestFile(t, parent, "go.sum", "")
	writePackageTestFile(t, parent, "cmd/avmatrix/main.go", "package main\n")
	writePackageTestFile(t, parent, "cmd/avmatrix/main_test.go", "package main\n")
	writePackageTestFile(t, parent, "internal/cli/command.go", "package cli\n")
	writePackageTestFile(t, parent, "internal/cli/command_test.go", "package cli\n")
	writePackageTestFile(t, parent, "scripts/ensure-ladybug-native.ps1", "Write-Output native\n")
	writePackageTestFile(t, parent, "scripts/ensure-ladybug-native.sh", "#!/usr/bin/env bash\nprintf native\\n\n")
	if err := os.MkdirAll(filepath.Join(packageRoot, "go-src", "old"), 0o755); err != nil {
		t.Fatalf("mkdir old go-src: %v", err)
	}

	var out bytes.Buffer
	if err := prepareGoSourcePackage(packageRoot, &out); err != nil {
		t.Fatalf("prepareGoSourcePackage returned error: %v", err)
	}
	for _, rel := range []string{
		"go.mod",
		"go.sum",
		"cmd/avmatrix/main.go",
		"internal/cli/command.go",
		"scripts/ensure-ladybug-native.ps1",
		"scripts/ensure-ladybug-native.sh",
		"avmatrix-go-source.json",
	} {
		if _, err := os.Stat(filepath.Join(packageRoot, "go-src", filepath.FromSlash(rel))); err != nil {
			t.Fatalf("prepared source missing %s: %v", rel, err)
		}
	}
	for _, rel := range []string{"cmd/avmatrix/main_test.go", "internal/cli/command_test.go", "old"} {
		if _, err := os.Stat(filepath.Join(packageRoot, "go-src", filepath.FromSlash(rel))); !os.IsNotExist(err) {
			t.Fatalf("prepared source retained excluded path %s: %v", rel, err)
		}
	}
	if !strings.Contains(out.String(), "[prepare-go-source-package] copied 6 files") {
		t.Fatalf("prepare output missing copied count: %q", out.String())
	}
}

func TestPackageJSONUsesCanonicalGoBinaryAndPackageCleanupCommand(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "avmatrix", "package.json"))
	if err != nil {
		t.Fatalf("read package.json: %v", err)
	}

	var pkg struct {
		Bin     map[string]string `json:"bin"`
		Files   []string          `json:"files"`
		Scripts map[string]string `json:"scripts"`
	}
	if err := json.Unmarshal(raw, &pkg); err != nil {
		t.Fatalf("parse package.json: %v", err)
	}

	if got := pkg.Bin["avmatrix"]; got != "bin/avmatrix.exe" {
		t.Fatalf("pkg.bin.avmatrix = %q", got)
	}
	if len(pkg.Bin) != 1 {
		t.Fatalf("pkg.bin should expose only avmatrix: %#v", pkg.Bin)
	}
	for _, want := range []string{"bin", "go-src", "skills"} {
		if !containsString(pkg.Files, want) {
			t.Fatalf("pkg.files missing %q: %#v", want, pkg.Files)
		}
	}
	for _, retiredFile := range []string{"dist", "scripts", "vendor"} {
		if containsString(pkg.Files, retiredFile) {
			t.Fatalf("pkg.files still ships retired legacy path %q: %#v", retiredFile, pkg.Files)
		}
	}
	if got := pkg.Scripts["build"]; got != "go run ../cmd/avmatrix package build-runtime" {
		t.Fatalf("pkg.scripts.build = %q", got)
	}
	if got := pkg.Scripts["prepack"]; !strings.Contains(got, "package prepare-go-source") {
		t.Fatalf("pkg.scripts.prepack missing Go prepare helper: %q", got)
	}
	if got := pkg.Scripts["postinstall"]; !strings.Contains(got, "package build-runtime") || strings.Contains(got, "scripts/") {
		t.Fatalf("pkg.scripts.postinstall should use Go package lifecycle helpers only: %q", got)
	}
	if got := pkg.Scripts["postpack"]; !strings.Contains(got, "package clean-go-source") || strings.Contains(got, "scripts/") {
		t.Fatalf("pkg.scripts.postpack = %q", got)
	}
	if strings.Contains(pkg.Scripts["postpack"], "clean-go-source-package.cjs") {
		t.Fatalf("pkg.scripts.postpack still references deleted CJS cleanup helper: %q", pkg.Scripts["postpack"])
	}
	for _, script := range []string{"test", "build"} {
		if pkg.Scripts[script] == "" {
			t.Fatalf("pkg.scripts missing %q: %#v", script, pkg.Scripts)
		}
	}
	for _, retired := range []string{"test:integration", "test:unit", "test:watch", "test:coverage"} {
		if pkg.Scripts[retired] != "" {
			t.Fatalf("pkg.scripts still contains retired legacy Vitest script %q: %q", retired, pkg.Scripts[retired])
		}
	}
	if strings.Contains(strings.ToLower(pkg.Scripts["test"]), "vitest") {
		t.Fatalf("pkg.scripts.test still references legacy Vitest: %q", pkg.Scripts["test"])
	}
	for _, script := range pkg.Scripts {
		for _, retired := range []string{"build.js", "build-go-runtime.cjs", "prepare-go-source-package.cjs", "patch-tree-sitter-swift.cjs", "build-tree-sitter-proto.cjs", "dev:ts-baseline"} {
			if strings.Contains(script, retired) {
				t.Fatalf("pkg.scripts still references retired legacy helper %q in %q", retired, script)
			}
		}
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

func writePackageTestFile(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
