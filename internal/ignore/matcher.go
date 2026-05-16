package ignore

import (
	"bufio"
	"os"
	"path"
	"path/filepath"
	"strings"
)

const NoGitignoreEnv = "AVMATRIX_NO_GITIGNORE"

type Options struct {
	NoGitignore bool
}

type Matcher struct {
	rules []rule
}

type rule struct {
	pattern string
	negated bool
}

func Load(repoPath string, options Options) (Matcher, error) {
	files := []string{".gitignore", ".avmatrixignore"}
	if options.NoGitignore || os.Getenv(NoGitignoreEnv) != "" {
		files = []string{".avmatrixignore"}
	}

	var matcher Matcher
	for _, name := range files {
		if err := matcher.addFile(filepath.Join(repoPath, name)); err != nil {
			return Matcher{}, err
		}
	}
	return matcher, nil
}

func (m Matcher) Ignored(relativePath string, isDir bool) bool {
	rel := NormalizePath(relativePath)
	if rel == "" {
		return false
	}
	if ShouldIgnorePath(rel) {
		return true
	}

	ignored := false
	for _, rule := range m.rules {
		if rule.matches(rel, isDir) {
			ignored = !rule.negated
		}
	}
	return ignored
}

func NormalizePath(p string) string {
	clean := strings.ReplaceAll(filepath.ToSlash(filepath.Clean(p)), "\\", "/")
	if clean == "." {
		return ""
	}
	clean = strings.TrimPrefix(clean, "./")
	return strings.Trim(clean, "/")
}

func (m *Matcher) addFile(filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		negated := strings.HasPrefix(line, "!")
		if negated {
			line = strings.TrimSpace(strings.TrimPrefix(line, "!"))
		}
		if line == "" {
			continue
		}
		m.rules = append(m.rules, rule{pattern: NormalizePath(line), negated: negated})
	}
	return scanner.Err()
}

func (r rule) matches(relativePath string, isDir bool) bool {
	pattern := r.pattern
	if pattern == "" {
		return false
	}

	dirOnly := strings.HasSuffix(pattern, "/")
	pattern = strings.TrimSuffix(pattern, "/")
	if dirOnly && !isDir && !strings.HasPrefix(relativePath, pattern+"/") {
		return false
	}

	if !strings.Contains(pattern, "/") {
		if segmentMatch(pattern, relativePath) {
			return true
		}
		base := path.Base(relativePath)
		if ok, _ := path.Match(pattern, base); ok {
			return true
		}
		return false
	}

	if strings.HasSuffix(pattern, "/**") {
		prefix := strings.TrimSuffix(pattern, "/**")
		return relativePath == prefix || strings.HasPrefix(relativePath, prefix+"/")
	}

	if ok, _ := path.Match(pattern, relativePath); ok {
		return true
	}
	return relativePath == pattern || strings.HasPrefix(relativePath, pattern+"/")
}

func segmentMatch(pattern, relativePath string) bool {
	for _, segment := range strings.Split(relativePath, "/") {
		if segment == pattern {
			return true
		}
		if ok, _ := path.Match(pattern, segment); ok {
			return true
		}
	}
	return false
}
