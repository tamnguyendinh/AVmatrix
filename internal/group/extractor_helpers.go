package group

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

type httpDetection struct {
	Role       string
	Framework  string
	Method     string
	Path       string
	Name       string
	Confidence float64
}

type graphSymbolRow struct {
	UID      string
	Name     string
	FilePath string
	Labels   []string
}

type httpGraphRouteRow struct {
	FileID      string
	FilePath    string
	RoutePath   string
	RouteSource string
}

type httpGraphFetchRow struct {
	FileID      string
	FilePath    string
	RoutePath   string
	FetchReason string
}

var groupScanExcludedDirs = map[string]bool{
	"node_modules": true,
	".git":         true,
	"vendor":       true,
	"dist":         true,
	"build":        true,
	"target":       true,
	"__pycache__":  true,
}

func walkGroupSourceFiles(root string, extensions map[string]bool, skipTests bool) ([]string, error) {
	files := make([]string, 0)
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) || os.IsPermission(err) {
				return nil
			}
			return err
		}
		name := d.Name()
		if d.IsDir() {
			if path != root && (strings.HasPrefix(name, ".") || groupScanExcludedDirs[name]) {
				return filepath.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() {
			return nil
		}
		if skipTests && strings.HasSuffix(name, "_test.go") {
			return nil
		}
		if !extensions[strings.ToLower(filepath.Ext(name))] {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		files = append(files, filepath.ToSlash(rel))
		return nil
	})
	sort.Strings(files)
	return files, err
}

func readGroupSourceFile(root string, rel string) string {
	raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		return ""
	}
	return string(raw)
}

func normalizeExtractorHTTPPath(pathValue string) string {
	trimmed := strings.TrimSpace(pathValue)
	if index := strings.Index(trimmed, "?"); index >= 0 {
		trimmed = trimmed[:index]
	}
	trimmed = strings.ToLower(trimmed)
	trimmed = regexp.MustCompile(`:\w+`).ReplaceAllString(trimmed, "{param}")
	trimmed = regexp.MustCompile(`\{[^}]+\}`).ReplaceAllString(trimmed, "{param}")
	trimmed = regexp.MustCompile(`\[[^\]]+\]`).ReplaceAllString(trimmed, "{param}")
	if trimmed == "" {
		return "/"
	}
	if !strings.HasPrefix(trimmed, "/") {
		trimmed = "/" + trimmed
	}
	for strings.Contains(trimmed, "//") {
		trimmed = strings.ReplaceAll(trimmed, "//", "/")
	}
	if trimmed == "/" {
		return trimmed
	}
	return strings.TrimRight(trimmed, "/")
}

func normalizeExtractorConsumerPath(url string) string {
	value := regexp.MustCompile(`\$\{[^}]+\}`).ReplaceAllString(strings.TrimSpace(url), "{param}")
	if strings.HasPrefix(strings.ToLower(value), "http://") || strings.HasPrefix(strings.ToLower(value), "https://") {
		withoutScheme := value
		if index := strings.Index(withoutScheme, "://"); index >= 0 {
			withoutScheme = withoutScheme[index+3:]
		}
		if slash := strings.Index(withoutScheme, "/"); slash >= 0 {
			value = withoutScheme[slash:]
		}
	}
	normalized := normalizeExtractorHTTPPath(value)
	segments := strings.Split(normalized, "/")
	for i, segment := range segments {
		if segment != "" && regexp.MustCompile(`^\d+$`).MatchString(segment) {
			segments[i] = "{param}"
		}
	}
	result := strings.Join(segments, "/")
	result = strings.TrimRight(result, "/")
	if result == "" {
		return "/"
	}
	return result
}

func extractorHTTPContractID(method string, pathValue string, consumer bool) string {
	pathNorm := normalizeExtractorHTTPPath(pathValue)
	if consumer {
		pathNorm = normalizeExtractorConsumerPath(pathValue)
	}
	method = strings.ToUpper(strings.TrimSpace(method))
	if method == "" {
		method = "GET"
	}
	return "http::" + method + "::" + pathNorm
}

func joinHTTPPath(base string, child string) string {
	base = strings.TrimSpace(strings.Trim(base, `"'`))
	child = strings.TrimSpace(strings.Trim(child, `"'`))
	if base == "" {
		return child
	}
	if child == "" {
		return base
	}
	return strings.TrimRight(base, "/") + "/" + strings.TrimLeft(child, "/")
}

func methodFromRouteReason(reason string) string {
	switch {
	case regexp.MustCompile(`(?i)GetMapping|decorator-Get`).MatchString(reason):
		return "GET"
	case regexp.MustCompile(`(?i)PostMapping|decorator-Post`).MatchString(reason):
		return "POST"
	case regexp.MustCompile(`(?i)PutMapping|decorator-Put`).MatchString(reason):
		return "PUT"
	case regexp.MustCompile(`(?i)DeleteMapping|decorator-Delete`).MatchString(reason):
		return "DELETE"
	case regexp.MustCompile(`(?i)PatchMapping|decorator-Patch`).MatchString(reason):
		return "PATCH"
	default:
		return ""
	}
}

func pickGraphSymbol(rows []graphSymbolRow, preferredName string) graphSymbolRow {
	pool := make([]graphSymbolRow, 0, len(rows))
	for _, row := range rows {
		for _, label := range row.Labels {
			if label == "Method" || label == "Function" {
				pool = append(pool, row)
				break
			}
		}
	}
	if len(pool) == 0 {
		pool = rows
	}
	if preferredName != "" {
		for _, row := range pool {
			if row.Name == preferredName {
				return row
			}
		}
	}
	if len(pool) > 0 {
		return pool[0]
	}
	return graphSymbolRow{}
}

func httpProviderContractFromGraphRow(row httpGraphRouteRow, detections []httpDetection, symbols []graphSymbolRow) StoredContract {
	method := methodFromRouteReason(row.RouteSource)
	normalizedRoute := normalizeExtractorHTTPPath(row.RoutePath)
	providerDetections := make([]httpDetection, 0)
	for _, detection := range detections {
		if detection.Role == "provider" && normalizeExtractorHTTPPath(detection.Path) == normalizedRoute {
			providerDetections = append(providerDetections, detection)
		}
	}

	handlerName := ""
	ambiguous := method == "" && len(providerDetections) > 1
	if method != "" {
		for _, detection := range providerDetections {
			if strings.EqualFold(detection.Method, method) {
				handlerName = detection.Name
				break
			}
		}
	} else if len(providerDetections) == 1 {
		method = strings.ToUpper(providerDetections[0].Method)
		handlerName = providerDetections[0].Name
	}
	if method == "" {
		method = "GET"
	}

	symbolUID := ""
	symbolName := filepath.Base(row.FilePath)
	if symbolName == "" || symbolName == "." {
		symbolName = "handler"
	}
	symbolPath := row.FilePath
	if !ambiguous && len(symbols) > 0 {
		picked := pickGraphSymbol(symbols, handlerName)
		symbolUID = picked.UID
		symbolName = firstNonEmptyGroupString(picked.Name, symbolName)
		symbolPath = firstNonEmptyGroupString(picked.FilePath, symbolPath)
	}

	return StoredContract{
		ContractID: extractorHTTPContractID(method, row.RoutePath, false),
		Type:       "http",
		Role:       "provider",
		SymbolUID:  symbolUID,
		SymbolRef:  SymbolRef{FilePath: filepath.ToSlash(symbolPath), Name: symbolName},
		SymbolName: symbolName,
		Confidence: 0.9,
		Meta: map[string]any{
			"method":             method,
			"path":               normalizedRoute,
			"extractionStrategy": "graph_assisted",
			"routeSource":        row.RouteSource,
		},
	}
}

func httpConsumerContractFromGraphRow(row httpGraphFetchRow, detections []httpDetection, symbols []graphSymbolRow) StoredContract {
	method := "GET"
	normalizedRoute := normalizeExtractorHTTPPath(row.RoutePath)
	candidates := make([]httpDetection, 0)
	for _, detection := range detections {
		if detection.Role == "consumer" && normalizeExtractorConsumerPath(detection.Path) == normalizedRoute {
			candidates = append(candidates, detection)
		}
	}
	if len(candidates) == 1 {
		method = strings.ToUpper(candidates[0].Method)
	}

	symbolUID := ""
	symbolName := "fetch"
	symbolPath := row.FilePath
	if len(symbols) > 0 {
		picked := pickGraphSymbol(symbols, "")
		symbolUID = picked.UID
		symbolName = firstNonEmptyGroupString(picked.Name, symbolName)
		symbolPath = firstNonEmptyGroupString(picked.FilePath, symbolPath)
	}
	return StoredContract{
		ContractID: extractorHTTPContractID(method, row.RoutePath, false),
		Type:       "http",
		Role:       "consumer",
		SymbolUID:  symbolUID,
		SymbolRef:  SymbolRef{FilePath: filepath.ToSlash(symbolPath), Name: symbolName},
		SymbolName: symbolName,
		Confidence: 0.9,
		Meta: map[string]any{
			"method":             method,
			"path":               normalizedRoute,
			"extractionStrategy": "graph_assisted",
			"fetchReason":        row.FetchReason,
		},
	}
}

func dedupeExtractedContracts(items []StoredContract) []StoredContract {
	seen := make(map[string]bool, len(items))
	out := make([]StoredContract, 0, len(items))
	for _, item := range items {
		key := item.ContractID + "\x00" + item.Role + "\x00" + item.SymbolRef.FilePath + "\x00" + item.SymbolRef.Name
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, item)
	}
	return out
}
