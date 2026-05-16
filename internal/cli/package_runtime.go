package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

type packageRuntimeMetadata struct {
	Platform string   `json:"platform"`
	Arch     string   `json:"arch"`
	Binary   string   `json:"binary"`
	Source   string   `json:"source"`
	Tags     []string `json:"tags"`
}

func ensurePackagedRuntime(packageRoot string, output io.Writer) error {
	root, err := filepath.Abs(packageRoot)
	if err != nil {
		return err
	}
	outputPath := filepath.Join(root, "bin", "avmatrix.exe")
	metadataPath := filepath.Join(root, "bin", "avmatrix-runtime.json")
	stat, err := os.Stat(outputPath)
	if err != nil || stat.IsDir() {
		return fmt.Errorf("packaged Go runtime is missing: %s", outputPath)
	}
	raw, err := os.ReadFile(metadataPath)
	if err != nil {
		return fmt.Errorf("packaged Go runtime metadata is missing: %w", err)
	}
	var metadata packageRuntimeMetadata
	if err := json.Unmarshal(raw, &metadata); err != nil {
		return fmt.Errorf("packaged Go runtime metadata is invalid: %w", err)
	}
	if !platformMatches(metadata.Platform) || !archMatches(metadata.Arch) {
		return fmt.Errorf("packaged Go runtime is %s/%s, current platform is %s/%s", metadata.Platform, metadata.Arch, runtime.GOOS, runtime.GOARCH)
	}
	if err := os.Chmod(outputPath, 0o755); err != nil {
		return err
	}
	_, err = fmt.Fprintf(output, "[package-runtime] using packaged Go runtime %s/%s\n", metadata.Platform, metadata.Arch)
	return err
}

func buildGoRuntimePackage(packageRoot string, output io.Writer) error {
	root, err := filepath.Abs(packageRoot)
	if err != nil {
		return err
	}
	sourceRoot, err := resolvePackageSourceRoot(root)
	if err != nil {
		if ensureErr := ensurePackagedRuntime(root, output); ensureErr == nil {
			return nil
		}
		return err
	}
	if _, err := exec.LookPath("go"); err != nil {
		if ensureErr := ensurePackagedRuntime(root, output); ensureErr == nil {
			return nil
		}
		return fmt.Errorf("Go toolchain is required to build the packaged AVmatrix runtime: %w", err)
	}

	outputDir := filepath.Join(root, "bin")
	outputPath := filepath.Join(outputDir, "avmatrix.exe")
	metadataPath := filepath.Join(outputDir, "avmatrix-runtime.json")
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return err
	}

	nativeDir, err := resolvePackageNativeDir(sourceRoot)
	if err != nil {
		return err
	}
	env := os.Environ()
	env = setEnv(env, "CGO_ENABLED", "1")
	env = setEnv(env, "CGO_CFLAGS", "-I"+nativeDir)
	env = setEnv(env, "PATH", nativeDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	env = setEnv(env, "DYLD_LIBRARY_PATH", nativeDir+string(os.PathListSeparator)+os.Getenv("DYLD_LIBRARY_PATH"))
	switch runtime.GOOS {
	case "windows":
		env = setEnv(env, "CGO_LDFLAGS", "-L"+nativeDir+" -llbug_shared")
	case "darwin":
		env = setEnv(env, "CGO_LDFLAGS", "-L"+nativeDir+" -llbug -Wl,-rpath,@loader_path")
	default:
		env = setEnv(env, "CGO_LDFLAGS", "-L"+nativeDir+" -llbug -Wl,-rpath,$ORIGIN")
		env = setEnv(env, "LD_LIBRARY_PATH", nativeDir+string(os.PathListSeparator)+os.Getenv("LD_LIBRARY_PATH"))
	}

	fmt.Fprintf(output, "[package-runtime] building Go runtime for %s/%s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Fprintf(output, "[package-runtime] Go source root: %s\n", sourceRoot)
	fmt.Fprintf(output, "[package-runtime] LadybugDB native runtime: %s\n", nativeDir)
	if err := runPackageCommand(output, sourceRoot, env, "go", "build", "-tags", "ladybugdb", "-trimpath", "-ldflags=-s -w", "-o", outputPath, "./cmd/avmatrix"); err != nil {
		return err
	}
	if err := os.Chmod(outputPath, 0o755); err != nil {
		return err
	}
	if err := copyPackageNativeRuntime(nativeDir, outputDir); err != nil {
		return err
	}
	relativeSource, err := filepath.Rel(root, sourceRoot)
	if err != nil || strings.HasPrefix(relativeSource, "..") {
		relativeSource = sourceRoot
	}
	metadata := packageRuntimeMetadata{
		Platform: runtime.GOOS,
		Arch:     runtime.GOARCH,
		Binary:   "avmatrix.exe",
		Source:   filepath.ToSlash(relativeSource),
		Tags:     []string{"ladybugdb"},
	}
	raw, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(metadataPath, append(raw, '\n'), 0o644); err != nil {
		return err
	}
	fmt.Fprintf(output, "[package-runtime] wrote %s\n", outputPath)
	return nil
}

func prepareGoSourcePackage(packageRoot string, output io.Writer) error {
	root, err := filepath.Abs(packageRoot)
	if err != nil {
		return err
	}
	outputRoot := filepath.Join(root, "go-src")
	if err := assertPackageChild(root, outputRoot); err != nil {
		return err
	}
	if err := os.RemoveAll(outputRoot); err != nil {
		return err
	}

	repoRoot := filepath.Dir(root)
	copied := make([]string, 0, 256)
	for _, rel := range []string{"go.mod", "go.sum"} {
		if err := copyPackageFile(filepath.Join(repoRoot, rel), filepath.Join(outputRoot, rel), outputRoot, &copied); err != nil {
			return err
		}
	}
	for _, rel := range []string{"cmd", "internal"} {
		if err := copyPackageGoDir(repoRoot, rel, outputRoot, &copied); err != nil {
			return err
		}
	}
	for _, rel := range []string{"scripts/ensure-ladybug-native.ps1", "scripts/ensure-ladybug-native.sh"} {
		if err := copyPackageFile(filepath.Join(repoRoot, rel), filepath.Join(outputRoot, rel), outputRoot, &copied); err != nil {
			return err
		}
	}
	if err := os.Chmod(filepath.Join(outputRoot, "scripts", "ensure-ladybug-native.sh"), 0o755); err != nil {
		return err
	}
	sort.Strings(copied)
	manifest := map[string]any{
		"generatedBy": "avmatrix package prepare-go-source",
		"source":      "repo-root",
		"files":       len(copied),
	}
	raw, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(outputRoot, "avmatrix-go-source.json"), append(raw, '\n'), 0o644); err != nil {
		return err
	}
	_, err = fmt.Fprintf(output, "[prepare-go-source-package] copied %d files to %s\n", len(copied), outputRoot)
	return err
}

func resolvePackageSourceRoot(packageRoot string) (string, error) {
	repoRoot := filepath.Dir(packageRoot)
	if hasPackageGoSource(repoRoot) {
		return repoRoot, nil
	}
	packagedSourceRoot := filepath.Join(packageRoot, "go-src")
	if hasPackageGoSource(packagedSourceRoot) {
		return packagedSourceRoot, nil
	}
	return "", fmt.Errorf("Go source is not available and the packaged Go runtime does not match this platform")
}

func hasPackageGoSource(root string) bool {
	required := []string{"go.mod", "go.sum", "cmd/avmatrix/main.go", "internal/cli/command.go"}
	for _, rel := range required {
		if stat, err := os.Stat(filepath.Join(root, rel)); err != nil || stat.IsDir() {
			return false
		}
	}
	return true
}

func resolvePackageNativeDir(sourceRoot string) (string, error) {
	version := os.Getenv("AVMATRIX_LADYBUGDB_VERSION")
	if strings.TrimSpace(version) == "" {
		version = "auto"
	}
	outputRoot := filepath.Join(sourceRoot, ".tmp", "ladybug-native")
	if runtime.GOOS == "windows" {
		script, err := resolvePackageNativeScript(sourceRoot, "ensure-ladybug-native.ps1")
		if err != nil {
			return "", err
		}
		return commandPackageOutput(sourceRoot, "powershell", "-NoProfile", "-ExecutionPolicy", "Bypass", "-File", script, "-Version", version, "-OutputRoot", outputRoot)
	}
	script, err := resolvePackageNativeScript(sourceRoot, "ensure-ladybug-native.sh")
	if err != nil {
		return "", err
	}
	return commandPackageOutput(sourceRoot, "bash", script, version, outputRoot)
}

func resolvePackageNativeScript(sourceRoot, scriptName string) (string, error) {
	repoRoot := filepath.Dir(sourceRoot)
	candidates := []string{
		filepath.Join(sourceRoot, "scripts", scriptName),
		filepath.Join(repoRoot, "scripts", scriptName),
	}
	for _, candidate := range candidates {
		if stat, err := os.Stat(candidate); err == nil && !stat.IsDir() {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("missing LadybugDB native resolver: %s", scriptName)
}

func commandPackageOutput(cwd, command string, args ...string) (string, error) {
	cmd := exec.Command(command, args...)
	cmd.Dir = cwd
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	raw, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("%s failed: %w: %s", command, err, strings.TrimSpace(stderr.String()))
	}
	lines := strings.Split(strings.TrimSpace(string(raw)), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		if line := strings.TrimSpace(lines[i]); line != "" {
			return line, nil
		}
	}
	return "", fmt.Errorf("%s returned empty output", command)
}

func runPackageCommand(output io.Writer, cwd string, env []string, command string, args ...string) error {
	cmd := exec.Command(command, args...)
	cmd.Dir = cwd
	cmd.Env = env
	cmd.Stdout = output
	cmd.Stderr = output
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s failed: %w", strings.Join(append([]string{command}, args...), " "), err)
	}
	return nil
}

func copyPackageNativeRuntime(nativeDir, outputDir string) error {
	if runtime.GOOS == "windows" {
		return copyPackageFileIfExists(filepath.Join(nativeDir, "lbug_shared.dll"), filepath.Join(outputDir, "lbug_shared.dll"))
	}

	entries, err := os.ReadDir(nativeDir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if runtime.GOOS == "darwin" {
			if strings.HasPrefix(name, "liblbug") && strings.HasSuffix(name, ".dylib") {
				if err := copyPackageFileIfExists(filepath.Join(nativeDir, name), filepath.Join(outputDir, name)); err != nil {
					return err
				}
			}
			continue
		}
		if name == "liblbug.so" || strings.HasPrefix(name, "liblbug.so.") {
			if err := copyPackageFileIfExists(filepath.Join(nativeDir, name), filepath.Join(outputDir, name)); err != nil {
				return err
			}
		}
	}
	return nil
}

func copyPackageFileIfExists(source, destination string) error {
	if stat, err := os.Stat(source); err != nil || stat.IsDir() {
		return nil
	}
	raw, err := os.ReadFile(source)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(destination, raw, 0o755); err != nil {
		return err
	}
	return os.Chmod(destination, 0o755)
}

func copyPackageGoDir(repoRoot, relativeDir, outputRoot string, copied *[]string) error {
	sourceDir := filepath.Join(repoRoot, relativeDir)
	if stat, err := os.Stat(sourceDir); err != nil || !stat.IsDir() {
		return fmt.Errorf("missing required source directory: %s", sourceDir)
	}
	stack := []string{sourceDir}
	for len(stack) > 0 {
		current := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		entries, err := os.ReadDir(current)
		if err != nil {
			return err
		}
		sort.Slice(entries, func(i, j int) bool { return entries[i].Name() > entries[j].Name() })
		for _, entry := range entries {
			source := filepath.Join(current, entry.Name())
			if entry.IsDir() {
				stack = append(stack, source)
				continue
			}
			if !entry.Type().IsRegular() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
				continue
			}
			rel, err := filepath.Rel(repoRoot, source)
			if err != nil {
				return err
			}
			if err := copyPackageFile(source, filepath.Join(outputRoot, rel), outputRoot, copied); err != nil {
				return err
			}
		}
	}
	return nil
}

func copyPackageFile(source, destination, outputRoot string, copied *[]string) error {
	stat, err := os.Stat(source)
	if err != nil || stat.IsDir() {
		return fmt.Errorf("missing required source file: %s", source)
	}
	if err := assertPackageChild(outputRoot, destination); err != nil {
		return err
	}
	raw, err := os.ReadFile(source)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(destination, raw, 0o644); err != nil {
		return err
	}
	rel, err := filepath.Rel(outputRoot, destination)
	if err != nil {
		return err
	}
	*copied = append(*copied, filepath.ToSlash(rel))
	return nil
}

func assertPackageChild(parent, child string) error {
	parentAbs, err := filepath.Abs(parent)
	if err != nil {
		return err
	}
	childAbs, err := filepath.Abs(child)
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(parentAbs, childAbs)
	if err != nil {
		return err
	}
	if rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return fmt.Errorf("refusing to write outside package root: %s", childAbs)
	}
	return nil
}

func setEnv(env []string, key, value string) []string {
	prefix := key + "="
	for i, item := range env {
		if strings.HasPrefix(item, prefix) {
			env[i] = prefix + value
			return env
		}
	}
	return append(env, prefix+value)
}

func platformMatches(value string) bool {
	switch value {
	case runtime.GOOS:
		return true
	case "win32":
		return runtime.GOOS == "windows"
	case "darwin":
		return runtime.GOOS == "darwin"
	case "linux":
		return runtime.GOOS == "linux"
	default:
		return false
	}
}

func archMatches(value string) bool {
	switch value {
	case runtime.GOARCH:
		return true
	case "x64":
		return runtime.GOARCH == "amd64"
	case "arm64":
		return runtime.GOARCH == "arm64"
	default:
		return false
	}
}
