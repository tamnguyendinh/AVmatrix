package routes

import (
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/tamnguyendinh/avmatrix-go/internal/graph"
	"github.com/tamnguyendinh/avmatrix-go/internal/scanner"
	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

var (
	frameworkRoutePattern   = regexp.MustCompile(`(?i)\b(?:app|router)\.(get|post|put|patch|delete|options|head|all|use)\s*\(\s*["']([^"']+)["']`)
	fetchPattern            = regexp.MustCompile("(?i)\\bfetch\\s*\\(\\s*[\"'`]([^\"'`]+)[\"'`]")
	routerNavigationPattern = regexp.MustCompile(`(?i)\brouter\.(?:push|replace|navigate)\s*\(\s*["']([^"']+)["']`)
	hrefNavigationPattern   = regexp.MustCompile(`(?i)\bhref\s*=\s*["']([^"']+)["']`)
	templateExpression      = regexp.MustCompile(`\$\{[^}]+\}`)
	statusCallPattern       = regexp.MustCompile(`(?i)\.status\s*\(\s*(\d{3})\s*\)`)
	statusPropertyPattern   = regexp.MustCompile(`(?i)\bstatus\s*:\s*(\d{3})`)
	httpResponseCodePattern = regexp.MustCompile(`(?i)http_response_code\s*\(\s*(\d{3})\s*\)`)
	httpStatusHeaderPattern = regexp.MustCompile(`(?i)header\s*\(\s*['"](?:HTTP/\d(?:\.\d)?|Status:)\s+(\d{3})\b`)
	jsIdentifierPattern     = regexp.MustCompile(`[A-Za-z_$][A-Za-z0-9_$]*`)
	jsonVariablePattern     = regexp.MustCompile(`(?s)\b(?:const|let|var)\s+([A-Za-z_$][A-Za-z0-9_$]*)\s*=\s*(?:await\s+)?[^;\n]*\.json\s*\(`)
	destructureJSONPattern  = regexp.MustCompile(`(?s)\b(?:const|let|var)\s*\{([^}]+)\}\s*=\s*(?:await\s+)?[A-Za-z_$][A-Za-z0-9_$]*\.json\s*\(`)
)

var ignoredConsumerKeys = map[string]bool{
	"appendChild":   true,
	"createElement": true,
	"click":         true,
	"entries":       true,
	"filter":        true,
	"forEach":       true,
	"json":          true,
	"keys":          true,
	"length":        true,
	"map":           true,
	"push":          true,
	"reduce":        true,
	"removeChild":   true,
	"then":          true,
	"values":        true,
}

var phpIgnoredAPIFileNames = map[string]bool{
	"_helpers": true,
	"helper":   true,
	"helpers":  true,
}

type responseShapes struct {
	ResponseKeys []string
	ErrorKeys    []string
}

type projectMiddleware struct {
	Names    []string
	Matchers []string
}

type Result struct {
	Metrics Metrics
}

type Metrics struct {
	FilesScanned   int `json:"filesScanned,omitempty"`
	RoutesEmitted  int `json:"routesEmitted,omitempty"`
	HandlesEmitted int `json:"handlesEmitted,omitempty"`
	FetchesEmitted int `json:"fetchesEmitted,omitempty"`
	Duplicates     int `json:"duplicates,omitempty"`
}

type routeEntry struct {
	URL          string
	FilePath     string
	Source       string
	ResponseKeys []string
	ErrorKeys    []string
	Middleware   []string
}

type fetchCall struct {
	FilePath     string
	URL          string
	AccessedKeys []string
	FetchCount   int
}

func Apply(g *graph.Graph, repoPath string, files []scanner.File) (Result, error) {
	if g == nil {
		return Result{}, nil
	}
	ordered := routeCandidateFiles(files)
	result := Result{}
	routesByURL := map[string]routeEntry{}
	fetches := make([]fetchCall, 0)
	middleware := make([]projectMiddleware, 0)
	for _, file := range ordered {
		result.Metrics.FilesScanned++
		var content string
		if !canReadForRouteExtraction(file.Language) {
			for _, route := range fileSystemRoutes(file.Path) {
				addRoute(routesByURL, route, &result.Metrics)
			}
			continue
		}
		raw, err := os.ReadFile(filepath.Join(repoPath, filepath.FromSlash(file.Path)))
		if err != nil {
			return result, err
		}
		content = string(raw)
		for _, route := range fileSystemRoutes(file.Path) {
			enrichRouteEntry(&route, content)
			addRoute(routesByURL, route, &result.Metrics)
		}
		for _, route := range frameworkRoutes(file.Path, content) {
			enrichRouteEntry(&route, content)
			addRoute(routesByURL, route, &result.Metrics)
		}
		if config, ok := extractProjectMiddleware(file.Path, content); ok {
			middleware = append(middleware, config)
		}
		fetches = append(fetches, fetchCalls(file.Path, content)...)
	}

	applyProjectMiddleware(routesByURL, middleware)
	routes := make([]routeEntry, 0, len(routesByURL))
	for _, route := range routesByURL {
		routes = append(routes, route)
	}
	sort.Slice(routes, func(i int, j int) bool { return routes[i].URL < routes[j].URL })
	for _, route := range routes {
		if emitRoute(g, route) {
			result.Metrics.RoutesEmitted++
			result.Metrics.HandlesEmitted++
		}
	}
	for _, call := range fetches {
		targetURL := normalizeRouteURL(call.URL)
		if targetURL == "" {
			continue
		}
		route, ok := matchRoute(routesByURL, targetURL)
		if !ok {
			continue
		}
		if emitFetch(g, call.FilePath, route.URL, call.AccessedKeys, call.FetchCount) {
			result.Metrics.FetchesEmitted++
		}
	}
	return result, nil
}

func routeCandidateFiles(files []scanner.File) []scanner.File {
	out := make([]scanner.File, 0)
	for _, file := range files {
		file.Path = normalizePath(file.Path)
		if file.Path == "" {
			continue
		}
		if canReadForRouteExtraction(file.Language) || len(fileSystemRoutes(file.Path)) > 0 {
			out = append(out, file)
		}
	}
	sort.Slice(out, func(i int, j int) bool { return out[i].Path < out[j].Path })
	return out
}

func fileSystemRoutes(filePath string) []routeEntry {
	routeURL, source := nextRouteURL(filePath)
	if routeURL == "" {
		routeURL, source = phpRouteURL(filePath)
		if routeURL == "" {
			routeURL, source = expoRouteURL(filePath)
			if routeURL == "" {
				return nil
			}
		}
	}
	return []routeEntry{{URL: routeURL, FilePath: filePath, Source: source}}
}

func nextRouteURL(filePath string) (string, string) {
	segments := strings.Split(normalizePath(filePath), "/")
	if len(segments) == 0 {
		return "", ""
	}
	base := segments[len(segments)-1]
	ext := path.Ext(base)
	name := strings.TrimSuffix(base, ext)
	if !isRouteFileExtension(ext) {
		return "", ""
	}
	for index, segment := range segments {
		if segment == "app" && (name == "page" || name == "route") {
			return routeFromSegments(segments[index+1 : len(segments)-1]), "nextjs-filesystem-route"
		}
		if segment == "pages" {
			return routeFromSegments(segments[index+1:]), "nextjs-pages-route"
		}
	}
	return "", ""
}

func routeFromSegments(segments []string) string {
	out := make([]string, 0, len(segments))
	for _, segment := range segments {
		if !(strings.HasPrefix(segment, "[") && strings.HasSuffix(segment, "]")) {
			segment = strings.TrimSuffix(segment, path.Ext(segment))
		}
		if segment == "" || segment == "index" || strings.HasPrefix(segment, "(") && strings.HasSuffix(segment, ")") {
			continue
		}
		if strings.HasPrefix(segment, "[[...") && strings.HasSuffix(segment, "]]") {
			out = append(out, "*"+strings.TrimSuffix(strings.TrimPrefix(segment, "[[..."), "]]"))
			continue
		}
		if strings.HasPrefix(segment, "[...") && strings.HasSuffix(segment, "]") {
			out = append(out, "*"+strings.TrimSuffix(strings.TrimPrefix(segment, "[..."), "]"))
			continue
		}
		if strings.HasPrefix(segment, "[") && strings.HasSuffix(segment, "]") {
			out = append(out, ":"+strings.TrimSuffix(strings.TrimPrefix(segment, "["), "]"))
			continue
		}
		out = append(out, segment)
	}
	if len(out) == 0 {
		return "/"
	}
	return "/" + strings.Join(out, "/")
}

func phpRouteURL(filePath string) (string, string) {
	normalized := normalizePath(filePath)
	if strings.ToLower(path.Ext(normalized)) != ".php" {
		return "", ""
	}
	segments := strings.Split(normalized, "/")
	for index, segment := range segments {
		if segment != "api" || index != 0 || index == len(segments)-1 {
			continue
		}
		base := segments[len(segments)-1]
		name := strings.TrimSuffix(base, path.Ext(base))
		lowerName := strings.ToLower(name)
		if phpIgnoredAPIFileNames[lowerName] ||
			strings.HasPrefix(lowerName, "_") ||
			strings.HasPrefix(lowerName, "helper_") ||
			strings.HasSuffix(lowerName, "_helper") ||
			strings.HasSuffix(lowerName, "_helpers") ||
			strings.HasPrefix(lowerName, "test_") ||
			strings.HasSuffix(lowerName, "_test") ||
			strings.HasPrefix(lowerName, "fixture_") ||
			strings.HasSuffix(lowerName, "_fixture") {
			return "", ""
		}
		routeSegments := append([]string{"api"}, segments[index+1:]...)
		routeSegments[len(routeSegments)-1] = name
		return "/" + strings.Join(routeSegments, "/"), "php-api-file-route"
	}
	return "", ""
}

func expoRouteURL(filePath string) (string, string) {
	segments := strings.Split(normalizePath(filePath), "/")
	for index, segment := range segments {
		if segment != "app" {
			continue
		}
		if index >= len(segments)-1 {
			return "", ""
		}
		base := segments[len(segments)-1]
		ext := path.Ext(base)
		if !isRouteFileExtension(ext) || strings.HasSuffix(base, ".d.ts") {
			return "", ""
		}
		name := strings.TrimSuffix(base, ext)
		if name == "_layout" || name == "layout" || strings.HasPrefix(name, "+") && !strings.HasSuffix(name, "+api") {
			return "", ""
		}
		if strings.HasSuffix(name, "+api") {
			name = strings.TrimSuffix(name, "+api")
		}
		routeSegments := append([]string(nil), segments[index+1:len(segments)-1]...)
		routeSegments = append(routeSegments, name)
		return expoRouteFromSegments(routeSegments), "expo-filesystem-route"
	}
	return "", ""
}

func expoRouteFromSegments(segments []string) string {
	out := make([]string, 0, len(segments))
	for _, segment := range segments {
		if segment == "" || segment == "index" || strings.HasPrefix(segment, "(") && strings.HasSuffix(segment, ")") {
			continue
		}
		out = append(out, segment)
	}
	if len(out) == 0 {
		return "/"
	}
	return "/" + strings.Join(out, "/")
}

func frameworkRoutes(filePath string, content string) []routeEntry {
	matches := frameworkRoutePattern.FindAllStringSubmatch(content, -1)
	out := make([]routeEntry, 0, len(matches))
	for _, match := range matches {
		routeURL := normalizeRouteURL(match[2])
		if routeURL == "" {
			continue
		}
		out = append(out, routeEntry{URL: routeURL, FilePath: filePath, Source: "framework-route"})
	}
	return out
}

func fetchCalls(filePath string, content string) []fetchCall {
	matches := fetchPattern.FindAllStringSubmatch(content, -1)
	out := make([]fetchCall, 0, len(matches))
	accessedKeys := extractConsumerAccessedKeys(content)
	fetchCount := len(matches)
	for _, match := range matches {
		routeURL := normalizeRouteURL(match[1])
		if routeURL == "" {
			continue
		}
		out = append(out, fetchCall{FilePath: filePath, URL: routeURL, AccessedKeys: accessedKeys, FetchCount: fetchCount})
	}
	for _, match := range routerNavigationPattern.FindAllStringSubmatch(content, -1) {
		routeURL := normalizeRouteURL(match[1])
		if routeURL == "" {
			continue
		}
		out = append(out, fetchCall{FilePath: filePath, URL: routeURL})
	}
	for _, match := range hrefNavigationPattern.FindAllStringSubmatch(content, -1) {
		routeURL := normalizeRouteURL(match[1])
		if routeURL == "" {
			continue
		}
		out = append(out, fetchCall{FilePath: filePath, URL: routeURL})
	}
	return out
}

func addRoute(routesByURL map[string]routeEntry, route routeEntry, metrics *Metrics) {
	if route.URL == "" || route.FilePath == "" {
		return
	}
	if existing, exists := routesByURL[route.URL]; exists {
		existing.ResponseKeys = mergeStrings(existing.ResponseKeys, route.ResponseKeys)
		existing.ErrorKeys = mergeStrings(existing.ErrorKeys, route.ErrorKeys)
		existing.Middleware = mergeStrings(existing.Middleware, route.Middleware)
		routesByURL[route.URL] = existing
		metrics.Duplicates++
		return
	}
	routesByURL[route.URL] = route
}

func emitRoute(g *graph.Graph, route routeEntry) bool {
	fileNodeID := graph.GenerateID(string(scopeir.NodeFile), route.FilePath)
	if _, ok := g.GetNode(fileNodeID); !ok {
		return false
	}
	routeNodeID := graph.GenerateID(string(scopeir.NodeRoute), route.URL)
	g.AddNode(graph.Node{
		ID:    routeNodeID,
		Label: scopeir.NodeRoute,
		Properties: graph.NodeProperties{
			"name":         route.URL,
			"filePath":     route.FilePath,
			"responseKeys": route.ResponseKeys,
			"errorKeys":    route.ErrorKeys,
			"middleware":   route.Middleware,
		},
	})
	g.AddRelationship(graph.Relationship{
		ID:         graph.GenerateID(string(graph.RelHandlesRoute), fileNodeID+"->"+routeNodeID),
		SourceID:   fileNodeID,
		TargetID:   routeNodeID,
		Type:       graph.RelHandlesRoute,
		Confidence: 1,
		Reason:     route.Source,
	})
	return true
}

func emitFetch(g *graph.Graph, filePath string, routeURL string, accessedKeys []string, fetchCount int) bool {
	fileNodeID := graph.GenerateID(string(scopeir.NodeFile), filePath)
	routeNodeID := graph.GenerateID(string(scopeir.NodeRoute), routeURL)
	if _, ok := g.GetNode(fileNodeID); !ok {
		return false
	}
	if _, ok := g.GetNode(routeNodeID); !ok {
		return false
	}
	g.AddRelationship(graph.Relationship{
		ID:         graph.GenerateID(string(graph.RelFetches), fileNodeID+"->"+routeNodeID),
		SourceID:   fileNodeID,
		TargetID:   routeNodeID,
		Type:       graph.RelFetches,
		Confidence: 0.9,
		Reason:     fetchReason(accessedKeys, fetchCount),
	})
	return true
}

func normalizeRouteURL(rawURL string) string {
	if rawURL == "" || strings.HasPrefix(rawURL, "http://") || strings.HasPrefix(rawURL, "https://") {
		return ""
	}
	rawURL = strings.Trim(rawURL, "`\"'")
	rawURL = templateExpression.ReplaceAllString(rawURL, "[param]")
	if strings.Contains(rawURL, "+") || strings.Contains(rawURL, "(") {
		return ""
	}
	cleaned := strings.Split(rawURL, "?")[0]
	cleaned = strings.Split(cleaned, "#")[0]
	if cleaned == "" {
		return ""
	}
	if !strings.HasPrefix(cleaned, "/") {
		cleaned = "/" + cleaned
	}
	if len(cleaned) > 1 {
		cleaned = strings.TrimRight(cleaned, "/")
	}
	return cleaned
}

func canReadForRouteExtraction(language scanner.Language) bool {
	switch language {
	case scanner.JavaScript, scanner.TypeScript, scanner.PHP:
		return true
	default:
		return false
	}
}

func isRouteFileExtension(ext string) bool {
	switch strings.ToLower(ext) {
	case ".js", ".jsx", ".mjs", ".cjs", ".ts", ".tsx", ".mts", ".cts", ".php":
		return true
	default:
		return false
	}
}

func normalizePath(filePath string) string {
	return strings.ReplaceAll(filePath, "\\", "/")
}

func enrichRouteEntry(route *routeEntry, content string) {
	shapes := extractResponseShapes(content)
	route.ResponseKeys = shapes.ResponseKeys
	route.ErrorKeys = shapes.ErrorKeys
	route.Middleware = extractMiddlewareChain(content)
}

func extractResponseShapes(content string) responseShapes {
	if strings.Contains(content, "<?php") || strings.Contains(content, "json_encode") {
		return extractPHPResponseShapes(content)
	}
	return extractJSResponseShapes(content)
}

func extractJSResponseShapes(content string) responseShapes {
	result := responseShapes{}
	searchFrom := 0
	for {
		index := strings.Index(content[searchFrom:], ".json")
		if index < 0 {
			break
		}
		jsonIndex := searchFrom + index
		openParen := strings.Index(content[jsonIndex:], "(")
		if openParen < 0 {
			break
		}
		objectStart := findNextNonSpace(content, jsonIndex+openParen+1)
		if objectStart < 0 || content[objectStart] != '{' {
			searchFrom = jsonIndex + len(".json")
			continue
		}
		keys, closeBrace := extractObjectKeys(content, objectStart)
		if len(keys) > 0 {
			status := detectJSStatusCode(content, jsonIndex, closeBrace)
			if status >= 400 {
				result.ErrorKeys = mergeStrings(result.ErrorKeys, keys)
			} else {
				result.ResponseKeys = mergeStrings(result.ResponseKeys, keys)
			}
		}
		if closeBrace <= jsonIndex {
			searchFrom = jsonIndex + len(".json")
		} else {
			searchFrom = closeBrace + 1
		}
	}
	return result
}

func detectJSStatusCode(content string, jsonIndex int, closeBrace int) int {
	prefixStart := jsonIndex - 80
	if prefixStart < 0 {
		prefixStart = 0
	}
	if status := lastStatusMatch(statusCallPattern, content[prefixStart:jsonIndex]); status > 0 {
		return status
	}
	suffixEnd := closeBrace + 160
	if suffixEnd > len(content) {
		suffixEnd = len(content)
	}
	for index := closeBrace; index < suffixEnd; index++ {
		if content[index] == ')' || content[index] == ';' || content[index] == '\n' {
			suffixEnd = index + 1
			break
		}
	}
	if closeBrace >= 0 && closeBrace < suffixEnd {
		if status := lastStatusMatch(statusPropertyPattern, content[closeBrace:suffixEnd]); status > 0 {
			return status
		}
	}
	return 0
}

func extractPHPResponseShapes(content string) responseShapes {
	result := responseShapes{}
	searchFrom := 0
	for {
		index := strings.Index(content[searchFrom:], "json_encode")
		if index < 0 {
			break
		}
		callIndex := searchFrom + index
		openParen := strings.Index(content[callIndex:], "(")
		if openParen < 0 {
			break
		}
		argStart := findNextNonSpace(content, callIndex+openParen+1)
		if argStart < 0 {
			break
		}
		keys, end := extractPHPArrayKeys(content, argStart)
		if len(keys) > 0 {
			status := detectPHPStatusCode(content, callIndex)
			if status >= 400 {
				result.ErrorKeys = mergeStrings(result.ErrorKeys, keys)
			} else {
				result.ResponseKeys = mergeStrings(result.ResponseKeys, keys)
			}
		}
		if end <= callIndex {
			searchFrom = callIndex + len("json_encode")
		} else {
			searchFrom = end + 1
		}
	}
	return result
}

func detectPHPStatusCode(content string, callIndex int) int {
	segmentStart := 0
	before := content[:callIndex]
	for _, marker := range []string{"exit", "die"} {
		if index := strings.LastIndex(before, marker); index >= segmentStart {
			segmentStart = index + len(marker)
		}
	}
	segment := content[segmentStart:callIndex]
	status := lastStatusMatch(httpResponseCodePattern, segment)
	if headerStatus := lastStatusMatch(httpStatusHeaderPattern, segment); headerStatus > 0 {
		status = headerStatus
	}
	return status
}

func extractObjectKeys(content string, openBrace int) ([]string, int) {
	keys := make([]string, 0)
	depth := 0
	inString := byte(0)
	identifierStart := -1
	for index := openBrace; index < len(content); index++ {
		ch := content[index]
		if inString != 0 {
			if ch == '\\' {
				index++
				continue
			}
			if ch == inString {
				inString = 0
				if identifierStart >= 0 && depth == 1 && nextNonSpaceIsPropertySeparator(content, index+1) {
					value := content[identifierStart:index]
					keys = append(keys, value)
				}
				identifierStart = -1
			}
			continue
		}
		switch {
		case ch == '"' || ch == '\'' || ch == '`':
			inString = ch
			if previousNonSpace(content, index-1) != ':' {
				identifierStart = index + 1
			} else {
				identifierStart = -1
			}
		case ch == '{':
			depth++
		case ch == '}':
			if identifierStart >= 0 && depth == 1 {
				keys = append(keys, content[identifierStart:index])
				identifierStart = -1
			}
			depth--
			if depth == 0 {
				return cleanUniqueStrings(keys), index
			}
		case depth == 1 && identifierStart < 0 && isIdentifierStart(ch) && previousNonSpace(content, index-1) != ':' && !isIdentifierPart(previousNonSpace(content, index-1)):
			identifierStart = index
		case depth == 1 && identifierStart >= 0 && !isIdentifierPart(ch):
			value := content[identifierStart:index]
			identifierStart = -1
			if nextNonSpaceIsPropertySeparator(content, index) {
				keys = append(keys, value)
			}
		}
	}
	return cleanUniqueStrings(keys), len(content) - 1
}

func extractPHPArrayKeys(content string, start int) ([]string, int) {
	if start >= len(content) {
		return nil, start
	}
	if content[start] == '[' {
		return extractPHPArrayKeysDelimited(content, start, '[', ']')
	}
	if strings.HasPrefix(strings.ToLower(content[start:]), "array") {
		openParen := strings.Index(content[start:], "(")
		if openParen >= 0 {
			return extractPHPArrayKeysDelimited(content, start+openParen, '(', ')')
		}
	}
	return nil, start
}

func extractPHPArrayKeysDelimited(content string, openIndex int, open byte, close byte) ([]string, int) {
	keys := make([]string, 0)
	depth := 0
	inString := byte(0)
	stringStart := -1
	for index := openIndex; index < len(content); index++ {
		ch := content[index]
		if inString != 0 {
			if ch == '\\' {
				index++
				continue
			}
			if ch == inString {
				value := content[stringStart:index]
				inString = 0
				stringStart = -1
				if depth == 1 && strings.HasPrefix(strings.TrimSpace(content[index+1:]), "=>") {
					keys = append(keys, value)
				}
			}
			continue
		}
		switch ch {
		case '"', '\'':
			inString = ch
			stringStart = index + 1
		case open, '[':
			depth++
		case close, ']':
			depth--
			if depth == 0 {
				return cleanUniqueStrings(keys), index
			}
		}
	}
	return cleanUniqueStrings(keys), len(content) - 1
}

func extractMiddlewareChain(content string) []string {
	exportPattern := regexp.MustCompile(`(?s)export\s+(?:const\s+(?:GET|POST|PUT|PATCH|DELETE|OPTIONS|HEAD)\s*=\s*|default\s+)([^;]+)`)
	wrapperPattern := regexp.MustCompile(`\bwith[A-Za-z0-9_]*\b`)
	for _, match := range exportPattern.FindAllStringSubmatch(content, -1) {
		assignment := match[1]
		if asyncIndex := strings.Index(assignment, "async"); asyncIndex >= 0 {
			assignment = assignment[:asyncIndex]
		}
		middleware := cleanUniqueStrings(wrapperPattern.FindAllString(assignment, -1))
		if len(middleware) > 0 {
			return middleware
		}
	}
	return nil
}

func extractProjectMiddleware(filePath string, content string) (projectMiddleware, bool) {
	base := path.Base(normalizePath(filePath))
	if !strings.HasPrefix(base, "middleware.") {
		return projectMiddleware{}, false
	}
	names := make([]string, 0)
	if regexp.MustCompile(`(?s)export\s+(?:default\s+)?function\s+middleware\b`).MatchString(content) ||
		regexp.MustCompile(`(?s)export\s+const\s+middleware\b`).MatchString(content) {
		names = append(names, "middleware")
	}
	if match := regexp.MustCompile(`(?s)export\s+default\s+([A-Za-z_$][A-Za-z0-9_$]*)`).FindStringSubmatch(content); len(match) == 2 && match[1] != "function" {
		names = append(names, match[1])
	}
	for _, match := range regexp.MustCompile(`(?s)export\s+default\s+chain\s*\(\s*\[([^\]]*)\]`).FindAllStringSubmatch(content, -1) {
		for _, ident := range jsIdentifierPattern.FindAllString(match[1], -1) {
			names = append(names, ident)
		}
	}
	names = cleanUniqueStrings(names)
	if len(names) == 0 {
		return projectMiddleware{}, false
	}
	matchers := make([]string, 0)
	if match := regexp.MustCompile(`(?s)matcher\s*:\s*(\[[^\]]*\]|["'][^"']+["'])`).FindStringSubmatch(content); len(match) == 2 {
		for _, quoted := range regexp.MustCompile(`["']([^"']+)["']`).FindAllStringSubmatch(match[1], -1) {
			matchers = append(matchers, quoted[1])
		}
	}
	return projectMiddleware{Names: names, Matchers: cleanUniqueStrings(matchers)}, true
}

func applyProjectMiddleware(routesByURL map[string]routeEntry, middleware []projectMiddleware) {
	if len(middleware) == 0 {
		return
	}
	for url, route := range routesByURL {
		for _, config := range middleware {
			if middlewareApplies(config, route.URL) {
				route.Middleware = mergeStrings(route.Middleware, config.Names)
			}
		}
		routesByURL[url] = route
	}
}

func middlewareApplies(config projectMiddleware, routeURL string) bool {
	if len(config.Matchers) == 0 {
		return true
	}
	for _, matcher := range config.Matchers {
		if middlewareMatcherMatchesRoute(matcher, routeURL) {
			return true
		}
	}
	return false
}

func middlewareMatcherMatchesRoute(matcher string, routeURL string) bool {
	matcher = normalizeRouteURL(matcher)
	if matcher == "" {
		return false
	}
	if strings.Contains(matcher, "(?!") {
		if strings.Contains(matcher, "api") && strings.HasPrefix(routeURL, "/api") {
			return false
		}
		return true
	}
	if strings.HasSuffix(matcher, "/:path*") {
		prefix := strings.TrimSuffix(matcher, "/:path*")
		return routeURL == prefix || strings.HasPrefix(routeURL, prefix+"/")
	}
	return routeURL == matcher
}

func extractConsumerAccessedKeys(content string) []string {
	keys := make([]string, 0)
	for _, match := range destructureJSONPattern.FindAllStringSubmatch(content, -1) {
		for _, raw := range strings.Split(match[1], ",") {
			name := strings.TrimSpace(strings.Split(raw, ":")[0])
			if name != "" {
				keys = append(keys, name)
			}
		}
	}
	variables := make([]string, 0)
	for _, match := range jsonVariablePattern.FindAllStringSubmatch(content, -1) {
		variables = append(variables, match[1])
	}
	for _, variable := range cleanUniqueStrings(variables) {
		pattern := regexp.MustCompile(`\b` + regexp.QuoteMeta(variable) + `\.([A-Za-z_$][A-Za-z0-9_$]*)`)
		for _, match := range pattern.FindAllStringSubmatch(content, -1) {
			keys = append(keys, match[1])
		}
	}
	return filterConsumerKeys(keys)
}

func filterConsumerKeys(keys []string) []string {
	out := make([]string, 0, len(keys))
	for _, key := range cleanUniqueStrings(keys) {
		if !ignoredConsumerKeys[key] {
			out = append(out, key)
		}
	}
	return out
}

func matchRoute(routesByURL map[string]routeEntry, targetURL string) (routeEntry, bool) {
	if route, ok := routesByURL[targetURL]; ok {
		return route, true
	}
	keys := make([]string, 0, len(routesByURL))
	for routeURL := range routesByURL {
		keys = append(keys, routeURL)
	}
	sort.Strings(keys)
	for _, routeURL := range keys {
		if routeMatches(targetURL, routeURL) {
			return routesByURL[routeURL], true
		}
	}
	return routeEntry{}, false
}

func routeMatches(candidate string, routeURL string) bool {
	candidateSegments := routeSegments(candidate)
	routeSegments := routeSegments(routeURL)
	for i := 0; i < len(routeSegments); i++ {
		if isCatchAllRouteSegment(routeSegments[i]) {
			return len(candidateSegments) >= i
		}
		if i >= len(candidateSegments) {
			return false
		}
		if isDynamicRouteSegment(candidateSegments[i]) || isDynamicRouteSegment(routeSegments[i]) {
			continue
		}
		if candidateSegments[i] != routeSegments[i] {
			return false
		}
	}
	return len(candidateSegments) == len(routeSegments)
}

func routeSegments(routeURL string) []string {
	trimmed := strings.Trim(routeURL, "/")
	if trimmed == "" {
		return nil
	}
	return strings.Split(trimmed, "/")
}

func isDynamicRouteSegment(segment string) bool {
	return isCatchAllRouteSegment(segment) ||
		strings.HasPrefix(segment, ":") ||
		strings.HasPrefix(segment, "[") && strings.HasSuffix(segment, "]") ||
		strings.HasPrefix(segment, "*")
}

func isCatchAllRouteSegment(segment string) bool {
	return strings.HasPrefix(segment, "*") ||
		strings.HasPrefix(segment, "[...") && strings.HasSuffix(segment, "]") ||
		strings.HasPrefix(segment, "[[...") && strings.HasSuffix(segment, "]]")
}

func fetchReason(accessedKeys []string, fetchCount int) string {
	parts := []string{"fetch-route"}
	if len(accessedKeys) > 0 {
		parts = append(parts, "keys:"+strings.Join(cleanUniqueStrings(accessedKeys), ","))
	}
	if fetchCount > 1 {
		parts = append(parts, "fetches:"+intToString(fetchCount))
	}
	return strings.Join(parts, "|")
}

func lastStatusMatch(pattern *regexp.Regexp, text string) int {
	matches := pattern.FindAllStringSubmatch(text, -1)
	if len(matches) == 0 || len(matches[len(matches)-1]) < 2 {
		return 0
	}
	return parsePositiveInt(matches[len(matches)-1][1])
}

func findNextNonSpace(content string, start int) int {
	for index := start; index < len(content); index++ {
		if !isSpace(content[index]) {
			return index
		}
	}
	return -1
}

func nextNonSpaceIsPropertySeparator(content string, start int) bool {
	index := findNextNonSpace(content, start)
	if index < 0 {
		return false
	}
	return content[index] == ':' || content[index] == ',' || content[index] == '}'
}

func previousNonSpace(content string, start int) byte {
	for index := start; index >= 0; index-- {
		if !isSpace(content[index]) {
			return content[index]
		}
	}
	return 0
}

func isIdentifierStart(ch byte) bool {
	return ch == '_' || ch == '$' || ch >= 'A' && ch <= 'Z' || ch >= 'a' && ch <= 'z'
}

func isIdentifierPart(ch byte) bool {
	return isIdentifierStart(ch) || ch >= '0' && ch <= '9'
}

func isSpace(ch byte) bool {
	return ch == ' ' || ch == '\t' || ch == '\r' || ch == '\n'
}

func mergeStrings(existing []string, values []string) []string {
	if len(values) == 0 {
		return cleanUniqueStrings(existing)
	}
	return cleanUniqueStrings(append(append([]string{}, existing...), values...))
}

func cleanUniqueStrings(values []string) []string {
	seen := make(map[string]bool, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.Trim(strings.TrimSpace(value), `"'`)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func parsePositiveInt(raw string) int {
	value := 0
	for _, digit := range raw {
		if digit < '0' || digit > '9' {
			return value
		}
		value = value*10 + int(digit-'0')
	}
	return value
}

func intToString(value int) string {
	if value == 0 {
		return "0"
	}
	digits := make([]byte, 0, 10)
	for value > 0 {
		digits = append(digits, byte('0'+value%10))
		value /= 10
	}
	for left, right := 0, len(digits)-1; left < right; left, right = left+1, right-1 {
		digits[left], digits[right] = digits[right], digits[left]
	}
	return string(digits)
}
