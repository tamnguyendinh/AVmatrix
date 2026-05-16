package scanner

import (
	"path"
	"strings"

	"github.com/tamnguyendinh/avmatrix-go/internal/ignore"
)

func matchesSelection(relativePath string, options Options) bool {
	if len(options.Include) > 0 && !matchesAny(relativePath, options.Include) {
		return false
	}
	return !matchesAny(relativePath, options.Exclude)
}

func matchesAny(relativePath string, patterns []string) bool {
	rel := ignore.NormalizePath(relativePath)
	for _, pattern := range patterns {
		if matchPattern(rel, ignore.NormalizePath(pattern)) {
			return true
		}
	}
	return false
}

func matchPattern(relativePath string, pattern string) bool {
	if pattern == "" {
		return false
	}
	if strings.HasSuffix(pattern, "/**") {
		prefix := strings.TrimSuffix(pattern, "/**")
		return relativePath == prefix || strings.HasPrefix(relativePath, prefix+"/")
	}
	if !strings.Contains(pattern, "/") {
		if ok, _ := path.Match(pattern, path.Base(relativePath)); ok {
			return true
		}
		for _, segment := range strings.Split(relativePath, "/") {
			if segment == pattern {
				return true
			}
		}
	}
	if ok, _ := path.Match(pattern, relativePath); ok {
		return true
	}
	return relativePath == pattern || strings.HasPrefix(relativePath, strings.TrimSuffix(pattern, "/")+"/")
}
