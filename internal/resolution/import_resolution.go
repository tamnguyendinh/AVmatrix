package resolution

import (
	"path"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/tamnguyendinh/avmatrix-go/internal/scanner"
	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

const (
	maxSyntheticImportBindings = 1000
	maxTransitiveIncludeFiles  = 5000
)

var importResolutionExtensions = []string{
	"",
	".tsx", ".ts", ".jsx", ".js", ".vue",
	"/index.tsx", "/index.ts", "/index.jsx", "/index.js",
	".py", "/__init__.py",
	".java", ".kt", ".kts",
	".c", ".h", ".cpp", ".hpp", ".cc", ".cxx", ".hxx", ".hh",
	".cs", ".go", ".rs", "/mod.rs", "/lib.rs",
	".php", ".phtml", ".swift", ".rb", ".dart",
}

func preprocessImportTarget(raw string) (string, bool) {
	cleaned := strings.TrimSpace(strings.Map(func(r rune) rune {
		switch r {
		case '\'', '"', '<', '>':
			return -1
		default:
			return r
		}
	}, raw))
	if cleaned == "" || len(cleaned) > 2048 || !utf8.ValidString(cleaned) {
		return "", false
	}
	for _, r := range cleaned {
		if r >= 0 && r < 0x20 {
			return "", false
		}
	}
	return cleaned, true
}

func (w *workspace) resolveLanguageImportFiles(language scanner.Language, sourceFile string, targetRaw string) []string {
	switch language {
	case scanner.Dart:
		return w.resolveDartImportFiles(sourceFile, targetRaw)
	case scanner.Java:
		return w.resolveJvmImportFiles(targetRaw, []string{".java"})
	case scanner.Kotlin:
		return w.resolveJvmImportFiles(targetRaw, []string{".kt", ".kts"})
	case scanner.CSharp:
		return w.resolveCSharpImportFiles(targetRaw)
	case scanner.PHP:
		return w.resolvePHPImportFiles(targetRaw)
	case scanner.Swift:
		return w.resolveSwiftImportFiles(targetRaw)
	case scanner.Ruby:
		return w.resolveRubyImportFiles(sourceFile, targetRaw)
	case scanner.Python:
		return w.resolvePythonImportFiles(sourceFile, targetRaw)
	case scanner.Rust:
		return w.resolveRustImportFiles(sourceFile, targetRaw)
	case scanner.C, scanner.CPlusPlus:
		if file, ok := w.resolveSuffixImport(targetRaw, []string{"", ".h", ".hpp", ".hh", ".hxx", ".c", ".cpp", ".cc", ".cxx"}); ok {
			return []string{file}
		}
	}
	return nil
}

func (w *workspace) resolveDartImportFiles(sourceFile string, targetRaw string) []string {
	switch {
	case strings.HasPrefix(targetRaw, "dart:"):
		return nil
	case strings.HasPrefix(targetRaw, "package:"):
		rest := strings.TrimPrefix(targetRaw, "package:")
		slash := strings.Index(rest, "/")
		if slash < 0 || slash == len(rest)-1 {
			return nil
		}
		if file, ok := w.resolveCandidatePath("lib/"+rest[slash+1:], []string{"", ".dart"}); ok {
			return []string{file}
		}
		return nil
	case strings.HasPrefix(targetRaw, "."):
		return w.resolveRelativeImport(sourceFile, targetRaw, []string{"", ".dart"})
	default:
		return nil
	}
}

func (w *workspace) resolveJvmImportFiles(targetRaw string, extensions []string) []string {
	if strings.HasSuffix(targetRaw, ".*") {
		dirSuffix := strings.ReplaceAll(strings.TrimSuffix(targetRaw, ".*"), ".", "/")
		return w.filesInDirSuffix(dirSuffix, extensions)
	}
	if file, ok := w.resolveSuffixImport(strings.ReplaceAll(targetRaw, ".", "/"), extensions); ok {
		return []string{file}
	}
	segments := strings.Split(targetRaw, ".")
	if len(segments) > 2 {
		classPath := strings.Join(segments[:len(segments)-1], "/")
		if file, ok := w.resolveSuffixImport(classPath, extensions); ok {
			return []string{file}
		}
	}
	return nil
}

func (w *workspace) resolveCSharpImportFiles(targetRaw string) []string {
	namespacePath := strings.ReplaceAll(targetRaw, ".", "/")
	if file, ok := w.resolveSuffixImport(namespacePath, []string{".cs"}); ok {
		return []string{file}
	}
	return w.filesInDirSuffix(namespacePath, []string{".cs"})
}

func (w *workspace) resolvePHPImportFiles(targetRaw string) []string {
	normalized := strings.ReplaceAll(targetRaw, "\\", "/")
	if strings.Contains(normalized, "..") {
		return nil
	}
	if file, ok := w.resolveSuffixImport(normalized, []string{".php", ".phtml"}); ok {
		return []string{file}
	}
	if slash := strings.LastIndex(normalized, "/"); slash > 0 {
		files := w.filesInDirSuffix(normalized[:slash], []string{".php", ".phtml"})
		if len(files) > 0 {
			return files[:1]
		}
	}
	return nil
}

func (w *workspace) resolveSwiftImportFiles(targetRaw string) []string {
	if files := w.filesInDirSuffix(path.Join("Sources", targetRaw), []string{".swift"}); len(files) > 0 {
		return files
	}
	if file, ok := w.resolveSuffixImport(targetRaw, []string{".swift"}); ok {
		return []string{file}
	}
	return nil
}

func (w *workspace) resolveRubyImportFiles(sourceFile string, targetRaw string) []string {
	if strings.HasPrefix(targetRaw, ".") {
		return w.resolveRelativeImport(sourceFile, targetRaw, []string{"", ".rb"})
	}
	if file, ok := w.resolveSuffixImport(strings.TrimPrefix(targetRaw, "./"), []string{"", ".rb"}); ok {
		return []string{file}
	}
	return nil
}

func (w *workspace) resolvePythonImportFiles(sourceFile string, targetRaw string) []string {
	if strings.HasPrefix(targetRaw, ".") {
		dots := 0
		for dots < len(targetRaw) && targetRaw[dots] == '.' {
			dots++
		}
		module := strings.TrimPrefix(targetRaw[dots:], ".")
		dirParts := splitCleanPath(path.Dir(cleanPath(sourceFile)))
		if dots-1 > len(dirParts) {
			return nil
		}
		dirParts = dirParts[:len(dirParts)-(dots-1)]
		if module != "" {
			dirParts = append(dirParts, strings.Split(strings.ReplaceAll(module, ".", "/"), "/")...)
		}
		if file, ok := w.resolveCandidatePath(path.Join(dirParts...), []string{".py", "/__init__.py"}); ok {
			return []string{file}
		}
		return nil
	}
	if strings.Contains(targetRaw, ".") {
		modulePath := strings.ReplaceAll(targetRaw, ".", "/")
		if file, ok := w.resolveSuffixImport(modulePath, []string{".py", "/__init__.py"}); ok {
			return []string{file}
		}
		return nil
	}
	importerDir := path.Dir(cleanPath(sourceFile))
	for dir := importerDir; ; dir = path.Dir(dir) {
		prefix := ""
		if dir != "." && dir != "/" {
			prefix = dir + "/"
		}
		if file, ok := w.resolveCandidatePath(prefix+targetRaw, []string{"/__init__.py", ".py"}); ok {
			return []string{file}
		}
		if dir == "." || dir == "/" {
			break
		}
	}
	if file, ok := w.resolveSuffixImport(targetRaw, []string{".py", "/__init__.py"}); ok {
		return []string{file}
	}
	return nil
}

func (w *workspace) resolveRustImportFiles(sourceFile string, targetRaw string) []string {
	if brace := strings.Index(targetRaw, "::{"); brace >= 0 {
		targetRaw = targetRaw[:brace]
	}
	var modulePath string
	switch {
	case strings.HasPrefix(targetRaw, "crate::"):
		modulePath = "src/" + strings.ReplaceAll(strings.TrimPrefix(targetRaw, "crate::"), "::", "/")
	case strings.HasPrefix(targetRaw, "self::"):
		modulePath = path.Join(path.Dir(cleanPath(sourceFile)), strings.ReplaceAll(strings.TrimPrefix(targetRaw, "self::"), "::", "/"))
	case strings.HasPrefix(targetRaw, "super::"):
		modulePath = path.Join(path.Dir(path.Dir(cleanPath(sourceFile))), strings.ReplaceAll(strings.TrimPrefix(targetRaw, "super::"), "::", "/"))
	case strings.Contains(targetRaw, "::"):
		modulePath = strings.ReplaceAll(targetRaw, "::", "/")
	default:
		return nil
	}
	if file, ok := w.resolveRustModulePath(modulePath); ok {
		return []string{file}
	}
	return nil
}

func (w *workspace) resolveRustModulePath(modulePath string) (string, bool) {
	if file, ok := w.resolveCandidatePath(modulePath, []string{".rs", "/mod.rs", "/lib.rs"}); ok {
		return file, true
	}
	if slash := strings.LastIndex(modulePath, "/"); slash > 0 {
		return w.resolveCandidatePath(modulePath[:slash], []string{".rs", "/mod.rs"})
	}
	return "", false
}

func (w *workspace) resolveRelativeImport(sourceFile string, targetRaw string, extensions []string) []string {
	base := cleanPath(path.Join(path.Dir(cleanPath(sourceFile)), targetRaw))
	if file, ok := w.resolveCandidatePath(base, extensions); ok {
		return []string{file}
	}
	return nil
}

func (w *workspace) resolveSuffixImport(targetRaw string, extensions []string) (string, bool) {
	parts := splitCleanPath(targetRaw)
	for index := 0; index < len(parts); index++ {
		suffix := path.Join(parts[index:]...)
		if file, ok := w.resolveCandidatePathSuffix(suffix, extensions); ok {
			return file, true
		}
	}
	return "", false
}

func (w *workspace) resolveCandidatePath(base string, extensions []string) (string, bool) {
	base = cleanPath(base)
	for _, extension := range extensions {
		candidate := cleanPath(base + extension)
		if _, ok := w.fileSet[candidate]; ok {
			return candidate, true
		}
	}
	return "", false
}

func (w *workspace) resolveCandidatePathSuffix(suffix string, extensions []string) (string, bool) {
	suffix = cleanPath(suffix)
	for _, file := range w.sortedFilePaths() {
		for _, extension := range extensions {
			candidate := cleanPath(suffix + extension)
			if file == candidate || strings.HasSuffix(file, "/"+candidate) || strings.EqualFold(file, candidate) || strings.HasSuffix(strings.ToLower(file), "/"+strings.ToLower(candidate)) {
				return file, true
			}
		}
	}
	return "", false
}

func (w *workspace) filesInDirSuffix(dirSuffix string, extensions []string) []string {
	dirSuffix = strings.Trim(cleanPath(dirSuffix), "/")
	files := make([]string, 0)
	for _, file := range w.sortedFilePaths() {
		dir := path.Dir(file)
		if dir != dirSuffix &&
			!strings.HasSuffix(dir, "/"+dirSuffix) &&
			!strings.EqualFold(dir, dirSuffix) &&
			!strings.HasSuffix(strings.ToLower(dir), "/"+strings.ToLower(dirSuffix)) {
			continue
		}
		if stringInSlice(path.Ext(file), extensions) {
			files = append(files, file)
		}
	}
	return files
}

func (w *workspace) sortedFilePaths() []string {
	files := make([]string, 0, len(w.fileSet))
	for file := range w.fileSet {
		files = append(files, file)
	}
	sort.Strings(files)
	return files
}

func splitCleanPath(value string) []string {
	value = strings.Trim(cleanPath(value), "/")
	if value == "" || value == "." {
		return nil
	}
	return strings.Split(value, "/")
}

func (w *workspace) synthesizeWildcardImportBindings() {
	importsBySource := make(map[string][]resolvedImport, len(w.imports))
	for _, item := range w.imports {
		if item.LinkStatus == "unresolved" || len(item.TargetFiles) == 0 {
			continue
		}
		importsBySource[item.Fact.FilePath] = append(importsBySource[item.Fact.FilePath], item)
	}
	bindingCounts := make(map[string]int, len(w.scopeBindings))
	for _, item := range w.imports {
		if item.SourceScope == "" || !needsWildcardSynthesis(item.Fact) {
			continue
		}
		targetFiles := item.TargetFiles
		if item.Fact.Kind == scopeir.ImportWildcard && isTransitiveIncludeLanguage(w.languageForFile(item.Fact.FilePath)) {
			targetFiles = expandTransitiveIncludeClosure(item.TargetFiles, importsBySource)
		}
		for _, targetFile := range targetFiles {
			for _, def := range w.defsByFile[targetFile] {
				if !isImportableSyntheticBinding(def.Fact) {
					continue
				}
				if bindingCounts[item.SourceScope] >= maxSyntheticImportBindings {
					return
				}
				if _, exists := w.scopeBindings[item.SourceScope][def.Fact.Name]; exists {
					continue
				}
				w.scopeBindings[item.SourceScope][def.Fact.Name] = append(
					w.scopeBindings[item.SourceScope][def.Fact.Name],
					bindingRef{Def: def, Origin: scopeir.BindingWildcard, Via: resolvedImportPtr(item)},
				)
				bindingCounts[item.SourceScope]++
			}
		}
	}
}

func resolvedImportPtr(item resolvedImport) *resolvedImport {
	return &item
}

func expandTransitiveIncludeClosure(direct []string, importsBySource map[string][]resolvedImport) []string {
	closure := make([]string, 0, len(direct))
	seen := make(map[string]struct{}, len(direct))
	queue := make([]string, 0, len(direct))
	enqueue := func(file string) bool {
		if _, ok := seen[file]; ok {
			return true
		}
		if len(seen) >= maxTransitiveIncludeFiles {
			return false
		}
		seen[file] = struct{}{}
		closure = append(closure, file)
		queue = append(queue, file)
		return true
	}
	for _, file := range direct {
		if !enqueue(file) {
			return closure
		}
	}
	for head := 0; head < len(queue); head++ {
		for _, item := range importsBySource[queue[head]] {
			for _, targetFile := range item.TargetFiles {
				if !enqueue(targetFile) {
					return closure
				}
			}
		}
	}
	return closure
}

func needsWildcardSynthesis(item scopeir.ImportFact) bool {
	return item.Kind == scopeir.ImportWildcard || item.Kind == scopeir.ImportWildcardExpanded || (item.Kind == scopeir.ImportNamed && item.LocalName == "" && item.ImportedName == "*")
}

func isTransitiveIncludeLanguage(language scanner.Language) bool {
	return language == scanner.C || language == scanner.CPlusPlus
}

func (w *workspace) languageForFile(filePath string) scanner.Language {
	filePath = cleanPath(filePath)
	for _, ir := range w.files {
		if ir.FilePath == filePath {
			return ir.Language
		}
	}
	return ""
}

func isImportableSyntheticBinding(def scopeir.DefinitionFact) bool {
	return isAnyLabel(def.Label, []scopeir.NodeLabel{
		scopeir.NodeFunction,
		scopeir.NodeClass,
		scopeir.NodeInterface,
		scopeir.NodeStruct,
		scopeir.NodeEnum,
		scopeir.NodeTrait,
		scopeir.NodeTypeAlias,
		scopeir.NodeConst,
		scopeir.NodeStatic,
		scopeir.NodeRecord,
		scopeir.NodeUnion,
		scopeir.NodeTypedef,
		scopeir.NodeMacro,
	})
}
