package group

import (
	"path/filepath"
	"regexp"
	"strings"
)

var httpSourceExtensions = map[string]bool{
	".ts":   true,
	".tsx":  true,
	".js":   true,
	".jsx":  true,
	".mjs":  true,
	".cjs":  true,
	".go":   true,
	".java": true,
	".py":   true,
	".php":  true,
}

func ExtractHTTPContractsFromSource(repoPath string) ([]StoredContract, error) {
	files, err := walkGroupSourceFiles(repoPath, httpSourceExtensions, false)
	if err != nil {
		return nil, err
	}
	out := make([]StoredContract, 0)
	for _, rel := range files {
		content := readGroupSourceFile(repoPath, rel)
		if content == "" {
			continue
		}
		for _, detection := range scanHTTPDetections(rel, content) {
			consumer := detection.Role == "consumer"
			contractPath := detection.Path
			contract := StoredContract{
				ContractID: extractorHTTPContractID(detection.Method, contractPath, consumer),
				Type:       "http",
				Role:       detection.Role,
				SymbolUID:  "",
				SymbolRef:  SymbolRef{FilePath: rel, Name: httpDetectionSymbolName(detection)},
				SymbolName: httpDetectionSymbolName(detection),
				Confidence: detection.Confidence,
				Meta: map[string]any{
					"method":             strings.ToUpper(detection.Method),
					"path":               normalizeHTTPDetectionPath(detection),
					"extractionStrategy": "source_scan",
					"framework":          detection.Framework,
				},
			}
			out = append(out, contract)
		}
	}
	return dedupeExtractedContracts(out), nil
}

func scanHTTPDetections(rel string, content string) []httpDetection {
	ext := strings.ToLower(filepath.Ext(rel))
	out := make([]httpDetection, 0)
	switch ext {
	case ".java":
		out = append(out, scanSpringProviders(content)...)
		out = append(out, scanJavaHTTPConsumers(content)...)
	case ".ts", ".tsx", ".js", ".jsx", ".mjs", ".cjs":
		out = append(out, scanExpressProviders(content)...)
		out = append(out, scanNestProviders(content)...)
		out = append(out, scanFetchConsumers(content)...)
		out = append(out, scanAxiosConsumers(content)...)
		out = append(out, scanJQueryConsumers(content)...)
	case ".go":
		out = append(out, scanGoHTTPProviders(content)...)
		out = append(out, scanGoHTTPConsumers(content)...)
	case ".php":
		out = append(out, scanLaravelProviders(content)...)
	case ".py":
		out = append(out, scanFastAPIProviders(content)...)
		out = append(out, scanPythonHTTPConsumers(content)...)
	}
	return out
}

func scanSpringProviders(content string) []httpDetection {
	base := ""
	if match := regexp.MustCompile(`@RequestMapping\(\s*["']([^"']+)["']`).FindStringSubmatch(content); len(match) == 2 {
		base = match[1]
	}
	re := regexp.MustCompile(`(?s)@(Get|Post|Put|Delete|Patch)Mapping\(\s*["']([^"']*)["'][^)]*\)\s*(?:public|private|protected)?\s*[\w<>\[\], ?]+\s+(\w+)\s*\(`)
	matches := re.FindAllStringSubmatch(content, -1)
	out := make([]httpDetection, 0, len(matches))
	for _, match := range matches {
		out = append(out, httpDetection{
			Role:       "provider",
			Framework:  "spring",
			Method:     strings.ToUpper(match[1]),
			Path:       joinHTTPPath(base, match[2]),
			Name:       match[3],
			Confidence: 0.8,
		})
	}
	return out
}

func scanExpressProviders(content string) []httpDetection {
	re := regexp.MustCompile("(?is)\\b(?:router|app)\\.(get|post|put|delete|patch)\\(\\s*[\\x22'\\x60]([^\\x22'\\x60]+)[\\x22'\\x60]\\s*,\\s*([A-Za-z_][A-Za-z0-9_]*)?")
	matches := re.FindAllStringSubmatch(content, -1)
	out := make([]httpDetection, 0, len(matches))
	for _, match := range matches {
		name := match[3]
		if name == "" || name == "async" || name == "function" {
			name = "handler"
		}
		out = append(out, httpDetection{
			Role:       "provider",
			Framework:  "express",
			Method:     strings.ToUpper(match[1]),
			Path:       match[2],
			Name:       name,
			Confidence: 0.75,
		})
	}
	return out
}

func scanNestProviders(content string) []httpDetection {
	base := ""
	if match := regexp.MustCompile(`@Controller\(\s*["']([^"']+)["']`).FindStringSubmatch(content); len(match) == 2 {
		base = match[1]
	}
	re := regexp.MustCompile(`(?s)@(Get|Post|Put|Delete|Patch)\(\s*["']([^"']*)["'][^)]*\)\s*(?:async\s+)?(\w+)\s*\(`)
	matches := re.FindAllStringSubmatch(content, -1)
	out := make([]httpDetection, 0, len(matches))
	for _, match := range matches {
		out = append(out, httpDetection{
			Role:       "provider",
			Framework:  "nestjs",
			Method:     strings.ToUpper(match[1]),
			Path:       joinHTTPPath(base, match[2]),
			Name:       match[3],
			Confidence: 0.8,
		})
	}
	return out
}

func scanGoHTTPProviders(content string) []httpDetection {
	out := make([]httpDetection, 0)
	frameworkRe := regexp.MustCompile(`(?m)\b\w+\.(GET|POST|PUT|DELETE|PATCH)\(\s*"([^"]+)"\s*,\s*(\w+)`)
	for _, match := range frameworkRe.FindAllStringSubmatch(content, -1) {
		out = append(out, httpDetection{
			Role:       "provider",
			Framework:  "go-router",
			Method:     match[1],
			Path:       match[2],
			Name:       match[3],
			Confidence: 0.75,
		})
	}
	stdlibRe := regexp.MustCompile(`http\.HandleFunc\(\s*"([^"]+)"\s*,\s*(\w+)`)
	for _, match := range stdlibRe.FindAllStringSubmatch(content, -1) {
		out = append(out, httpDetection{
			Role:       "provider",
			Framework:  "go-stdlib",
			Method:     "GET",
			Path:       match[1],
			Name:       match[2],
			Confidence: 0.65,
		})
	}
	return out
}

func scanLaravelProviders(content string) []httpDetection {
	re := regexp.MustCompile(`Route::(get|post|put|delete|patch)\(\s*['"]([^'"]+)['"]`)
	matches := re.FindAllStringSubmatch(content, -1)
	out := make([]httpDetection, 0, len(matches))
	for _, match := range matches {
		out = append(out, httpDetection{
			Role:       "provider",
			Framework:  "laravel",
			Method:     strings.ToUpper(match[1]),
			Path:       match[2],
			Name:       "handler",
			Confidence: 0.75,
		})
	}
	return out
}

func scanFastAPIProviders(content string) []httpDetection {
	re := regexp.MustCompile(`(?s)@app\.(get|post|put|delete|patch)\(\s*["']([^"']+)["'][^)]*\)\s*(?:async\s+)?def\s+(\w+)\s*\(`)
	matches := re.FindAllStringSubmatch(content, -1)
	out := make([]httpDetection, 0, len(matches))
	for _, match := range matches {
		out = append(out, httpDetection{
			Role:       "provider",
			Framework:  "fastapi",
			Method:     strings.ToUpper(match[1]),
			Path:       match[2],
			Name:       match[3],
			Confidence: 0.75,
		})
	}
	return out
}

func scanFetchConsumers(content string) []httpDetection {
	re := regexp.MustCompile("(?is)\\bfetch\\(\\s*[\\x22'\\x60]([^\\x22'\\x60]+)[\\x22'\\x60]\\s*(?:,\\s*\\{(.*?)\\})?")
	matches := re.FindAllStringSubmatch(content, -1)
	out := make([]httpDetection, 0, len(matches))
	for _, match := range matches {
		method := readHTTPMethodFromObject(match[2], "GET")
		out = append(out, httpDetection{
			Role:       "consumer",
			Framework:  "fetch",
			Method:     method,
			Path:       match[1],
			Name:       "fetch",
			Confidence: 0.75,
		})
	}
	return out
}

func scanAxiosConsumers(content string) []httpDetection {
	out := make([]httpDetection, 0)
	shorthand := regexp.MustCompile("(?is)\\baxios\\.(get|post|put|delete|patch)\\(\\s*[\\x22'\\x60]([^\\x22'\\x60]+)[\\x22'\\x60]")
	for _, match := range shorthand.FindAllStringSubmatch(content, -1) {
		out = append(out, httpDetection{
			Role:       "consumer",
			Framework:  "axios",
			Method:     strings.ToUpper(match[1]),
			Path:       match[2],
			Name:       "fetch",
			Confidence: 0.75,
		})
	}
	objectForm := regexp.MustCompile(`(?is)\baxios\(\s*\{(.*?)\}\s*\)`)
	for _, match := range objectForm.FindAllStringSubmatch(content, -1) {
		pathValue := readHTTPStringProp(match[1], "url")
		if pathValue == "" {
			continue
		}
		out = append(out, httpDetection{
			Role:       "consumer",
			Framework:  "axios",
			Method:     readHTTPMethodFromObject(match[1], "GET"),
			Path:       pathValue,
			Name:       "fetch",
			Confidence: 0.75,
		})
	}
	return out
}

func scanJQueryConsumers(content string) []httpDetection {
	out := make([]httpDetection, 0)
	shorthand := regexp.MustCompile("(?is)\\$\\.(get|post)\\(\\s*[\\x22'\\x60]([^\\x22'\\x60]+)[\\x22'\\x60]")
	for _, match := range shorthand.FindAllStringSubmatch(content, -1) {
		out = append(out, httpDetection{
			Role:       "consumer",
			Framework:  "jquery",
			Method:     strings.ToUpper(match[1]),
			Path:       match[2],
			Name:       "fetch",
			Confidence: 0.7,
		})
	}
	ajax := regexp.MustCompile(`(?is)\$\.ajax\(\s*\{(.*?)\}\s*\)`)
	for _, match := range ajax.FindAllStringSubmatch(content, -1) {
		pathValue := readHTTPStringProp(match[1], "url")
		if pathValue == "" {
			continue
		}
		method := readHTTPMethodFromObject(match[1], "GET")
		out = append(out, httpDetection{
			Role:       "consumer",
			Framework:  "jquery",
			Method:     method,
			Path:       pathValue,
			Name:       "fetch",
			Confidence: 0.7,
		})
	}
	return out
}

func scanPythonHTTPConsumers(content string) []httpDetection {
	re := regexp.MustCompile(`requests\.(get|post|put|delete|patch)\(\s*["']([^"']+)["']`)
	matches := re.FindAllStringSubmatch(content, -1)
	out := make([]httpDetection, 0, len(matches))
	for _, match := range matches {
		out = append(out, httpDetection{
			Role:       "consumer",
			Framework:  "requests",
			Method:     strings.ToUpper(match[1]),
			Path:       match[2],
			Name:       "fetch",
			Confidence: 0.7,
		})
	}
	return out
}

func scanJavaHTTPConsumers(content string) []httpDetection {
	out := make([]httpDetection, 0)
	for _, match := range regexp.MustCompile(`restTemplate\.getForObject\(\s*["']([^"']+)["']`).FindAllStringSubmatch(content, -1) {
		out = append(out, httpDetection{Role: "consumer", Framework: "resttemplate", Method: "GET", Path: match[1], Name: "fetch", Confidence: 0.7})
	}
	for _, match := range regexp.MustCompile(`webClient\.method\(\s*HttpMethod\.(GET|POST|PUT|DELETE|PATCH)\s*,\s*["']([^"']+)["']`).FindAllStringSubmatch(content, -1) {
		out = append(out, httpDetection{Role: "consumer", Framework: "webclient", Method: match[1], Path: match[2], Name: "fetch", Confidence: 0.7})
	}
	for _, match := range regexp.MustCompile(`Request\.Builder\(\)\.url\(\s*["']([^"']+)["']`).FindAllStringSubmatch(content, -1) {
		out = append(out, httpDetection{Role: "consumer", Framework: "okhttp", Method: "GET", Path: match[1], Name: "fetch", Confidence: 0.65})
	}
	return out
}

func scanGoHTTPConsumers(content string) []httpDetection {
	out := make([]httpDetection, 0)
	for _, match := range regexp.MustCompile(`http\.(Get|Post|Put|Delete|Patch)\(\s*"([^"]+)"`).FindAllStringSubmatch(content, -1) {
		out = append(out, httpDetection{Role: "consumer", Framework: "go-stdlib", Method: strings.ToUpper(match[1]), Path: match[2], Name: "fetch", Confidence: 0.7})
	}
	for _, match := range regexp.MustCompile(`\.R\(\)\.(Get|Post|Put|Delete|Patch)\(\s*"([^"]+)"`).FindAllStringSubmatch(content, -1) {
		out = append(out, httpDetection{Role: "consumer", Framework: "resty", Method: strings.ToUpper(match[1]), Path: match[2], Name: "fetch", Confidence: 0.7})
	}
	return out
}

func readHTTPMethodFromObject(object string, fallback string) string {
	if object == "" {
		return fallback
	}
	if value := readHTTPStringProp(object, "method"); value != "" {
		return strings.ToUpper(value)
	}
	if value := readHTTPStringProp(object, "type"); value != "" {
		return strings.ToUpper(value)
	}
	return fallback
}

func readHTTPStringProp(object string, key string) string {
	re := regexp.MustCompile("(?is)\\b" + regexp.QuoteMeta(key) + "\\s*:\\s*[\\x22'\\x60]([^\\x22'\\x60]+)[\\x22'\\x60]")
	match := re.FindStringSubmatch(object)
	if len(match) == 2 {
		return match[1]
	}
	return ""
}

func httpDetectionSymbolName(detection httpDetection) string {
	if detection.Role == "consumer" {
		return "fetch"
	}
	return firstNonEmptyGroupString(detection.Name, "handler")
}

func normalizeHTTPDetectionPath(detection httpDetection) string {
	if detection.Role == "consumer" {
		return normalizeExtractorConsumerPath(detection.Path)
	}
	return normalizeExtractorHTTPPath(detection.Path)
}
