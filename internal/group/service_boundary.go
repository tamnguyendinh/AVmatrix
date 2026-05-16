package group

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type ServiceBoundary struct {
	ServicePath string
	ServiceName string
	Markers     []string
	Confidence  float64
}

var serviceBoundaryMarkers = map[string]bool{
	"package.json":     true,
	"go.mod":           true,
	"Dockerfile":       true,
	"pom.xml":          true,
	"build.gradle":     true,
	"build.gradle.kts": true,
	"Cargo.toml":       true,
	"pyproject.toml":   true,
	"requirements.txt": true,
	"mix.exs":          true,
}

var serviceBoundarySourceExtensions = map[string]bool{
	".ts":    true,
	".tsx":   true,
	".js":    true,
	".jsx":   true,
	".mjs":   true,
	".cjs":   true,
	".go":    true,
	".java":  true,
	".kt":    true,
	".kts":   true,
	".py":    true,
	".pyi":   true,
	".rs":    true,
	".c":     true,
	".cpp":   true,
	".h":     true,
	".hpp":   true,
	".cs":    true,
	".rb":    true,
	".php":   true,
	".swift": true,
	".dart":  true,
	".ex":    true,
	".exs":   true,
	".erl":   true,
	".proto": true,
}

var serviceBoundaryExcludedDirs = map[string]bool{
	"node_modules": true,
	"vendor":       true,
	"target":       true,
	"build":        true,
	"dist":         true,
	"__pycache__":  true,
	".venv":        true,
	"venv":         true,
	".tox":         true,
	".mypy_cache":  true,
	".gradle":      true,
	".mvn":         true,
	"out":          true,
	"bin":          true,
}

func DetectServiceBoundaries(repoPath string) ([]ServiceBoundary, error) {
	boundaries := make([]ServiceBoundary, 0)
	if err := walkForServiceBoundaries(repoPath, repoPath, &boundaries); err != nil {
		return nil, err
	}
	sort.Slice(boundaries, func(i, j int) bool {
		return boundaries[i].ServicePath < boundaries[j].ServicePath
	})
	return boundaries, nil
}

func AssignService(filePath string, boundaries []ServiceBoundary) string {
	normalized := filepath.ToSlash(filePath)
	var best string
	bestLength := 0
	for _, boundary := range boundaries {
		prefix := strings.TrimRight(boundary.ServicePath, "/") + "/"
		if strings.HasPrefix(normalized, prefix) && len(boundary.ServicePath) > bestLength {
			best = boundary.ServicePath
			bestLength = len(boundary.ServicePath)
		}
	}
	return best
}

func walkForServiceBoundaries(dir string, repoRoot string, results *[]ServiceBoundary) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) || os.IsPermission(err) {
			return nil
		}
		return err
	}

	rootAbs, err := filepath.Abs(repoRoot)
	if err != nil {
		return err
	}
	dirAbs, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	isRoot := dirAbs == rootAbs

	markers := make([]string, 0)
	subdirs := make([]string, 0)
	hasSource := false
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() {
			if shouldSkipGroupScanDir(name) {
				continue
			}
			subdirs = append(subdirs, filepath.Join(dir, name))
			continue
		}
		if !entry.Type().IsRegular() {
			continue
		}
		if serviceBoundaryMarkers[name] {
			markers = append(markers, name)
		}
		if serviceBoundarySourceExtensions[strings.ToLower(filepath.Ext(name))] {
			hasSource = true
		}
	}

	if !hasSource && len(markers) > 0 {
		found, err := hasSourceFilesInServiceSubdirs(subdirs)
		if err != nil {
			return err
		}
		hasSource = found
	}

	if !isRoot && len(markers) > 0 && hasSource {
		rel, err := filepath.Rel(repoRoot, dir)
		if err != nil {
			return err
		}
		sort.Strings(markers)
		*results = append(*results, ServiceBoundary{
			ServicePath: filepath.ToSlash(rel),
			ServiceName: filepath.Base(dir),
			Markers:     markers,
			Confidence:  serviceBoundaryConfidence(len(markers)),
		})
	}

	for _, subdir := range subdirs {
		if err := walkForServiceBoundaries(subdir, repoRoot, results); err != nil {
			return err
		}
	}
	return nil
}

func hasSourceFilesInServiceSubdirs(subdirs []string) (bool, error) {
	for _, subdir := range subdirs {
		entries, err := os.ReadDir(subdir)
		if err != nil {
			if os.IsNotExist(err) || os.IsPermission(err) {
				continue
			}
			return false, err
		}
		nested := make([]string, 0)
		for _, entry := range entries {
			name := entry.Name()
			if entry.IsDir() {
				if !shouldSkipGroupScanDir(name) {
					nested = append(nested, filepath.Join(subdir, name))
				}
				continue
			}
			if entry.Type().IsRegular() && serviceBoundarySourceExtensions[strings.ToLower(filepath.Ext(name))] {
				return true, nil
			}
		}
		found, err := hasSourceFilesInServiceSubdirs(nested)
		if err != nil || found {
			return found, err
		}
	}
	return false, nil
}

func shouldSkipGroupScanDir(name string) bool {
	return strings.HasPrefix(name, ".") || serviceBoundaryExcludedDirs[name]
}

func serviceBoundaryConfidence(markerCount int) float64 {
	if markerCount >= 3 {
		return 1
	}
	if markerCount == 2 {
		return 0.9
	}
	return 0.75
}
