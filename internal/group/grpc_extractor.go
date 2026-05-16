package group

import (
	"fmt"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

type ProtoServiceInfo struct {
	Package     string
	ServiceName string
	Methods     []string
	ProtoPath   string
}

var grpcSourceExtensions = map[string]bool{
	".proto": true,
	".go":    true,
	".java":  true,
	".py":    true,
	".ts":    true,
	".tsx":   true,
	".js":    true,
	".jsx":   true,
}

func ExtractGRPCContracts(repoPath string) ([]StoredContract, error) {
	protoMap, err := BuildProtoMap(repoPath)
	if err != nil {
		return nil, err
	}
	out := make([]StoredContract, 0)
	for _, serviceName := range sortedProtoServiceNames(protoMap) {
		for _, info := range protoMap[serviceName] {
			for _, method := range info.Methods {
				out = append(out, makeGRPCContract(
					grpcMethodContractID(info.Package, info.ServiceName, method),
					"provider",
					info.ProtoPath,
					info.ServiceName+"."+method,
					0.85,
					map[string]any{"package": info.Package, "service": info.ServiceName, "method": method, "source": "proto"},
				))
			}
		}
	}

	files, err := walkGroupSourceFiles(repoPath, grpcSourceExtensions, false)
	if err != nil {
		return nil, err
	}
	for _, rel := range files {
		if strings.HasSuffix(rel, ".proto") {
			continue
		}
		content := readGroupSourceFile(repoPath, rel)
		if content == "" {
			continue
		}
		out = append(out, scanGRPCSourceFile(rel, content, protoMap)...)
	}
	return dedupeGRPCContracts(out), nil
}

func BuildProtoMap(repoPath string) (map[string][]ProtoServiceInfo, error) {
	files, err := walkGroupSourceFiles(repoPath, map[string]bool{".proto": true}, false)
	if err != nil {
		return nil, err
	}
	contents := make(map[string]string, len(files))
	for _, rel := range files {
		contents[rel] = readGroupSourceFile(repoPath, rel)
	}
	packages := make(map[string]string, len(files))
	var resolvePackage func(string, map[string]bool) string
	resolvePackage = func(protoPath string, seen map[string]bool) string {
		if pkg, ok := packages[protoPath]; ok {
			return pkg
		}
		if seen[protoPath] {
			return ""
		}
		seen[protoPath] = true
		content := contents[protoPath]
		if match := regexp.MustCompile(`(?m)^\s*package\s+([\w.]+)\s*;`).FindStringSubmatch(content); len(match) == 2 {
			packages[protoPath] = match[1]
			return match[1]
		}
		for _, imported := range extractProtoImports(content) {
			candidates := []string{
				filepath.ToSlash(path.Clean(path.Join(path.Dir(protoPath), imported))),
				filepath.ToSlash(path.Clean(imported)),
			}
			for _, candidate := range candidates {
				if _, ok := contents[candidate]; !ok {
					continue
				}
				if pkg := resolvePackage(candidate, seen); pkg != "" {
					packages[protoPath] = pkg
					return pkg
				}
			}
		}
		packages[protoPath] = ""
		return ""
	}

	servicesByName := make(map[string][]ProtoServiceInfo)
	for _, rel := range files {
		content := contents[rel]
		pkg := resolvePackage(rel, map[string]bool{})
		for _, block := range extractProtoServiceBlocks(content) {
			methods := make([]string, 0)
			for _, match := range regexp.MustCompile(`rpc\s+(\w+)\s*\(`).FindAllStringSubmatch(block.body, -1) {
				methods = append(methods, match[1])
			}
			info := ProtoServiceInfo{
				Package:     pkg,
				ServiceName: block.name,
				Methods:     methods,
				ProtoPath:   rel,
			}
			servicesByName[block.name] = append(servicesByName[block.name], info)
		}
	}
	return servicesByName, nil
}

func ResolveProtoConflict(serviceName string, sourceFilePath string, candidates []ProtoServiceInfo) *ProtoServiceInfo {
	if len(candidates) == 0 {
		return nil
	}
	if len(candidates) == 1 {
		candidate := candidates[0]
		return &candidate
	}
	sourceDir := filepath.ToSlash(path.Dir(sourceFilePath))
	type scoredCandidate struct {
		info  ProtoServiceInfo
		score int
	}
	scored := make([]scoredCandidate, 0, len(candidates))
	best := -1
	for _, candidate := range candidates {
		score := longestSharedSegmentRun(sourceDir, filepath.ToSlash(path.Dir(candidate.ProtoPath)))
		if score > best {
			best = score
		}
		scored = append(scored, scoredCandidate{info: candidate, score: score})
	}
	winners := make([]ProtoServiceInfo, 0)
	for _, scoredItem := range scored {
		if scoredItem.score == best {
			winners = append(winners, scoredItem.info)
		}
	}
	if len(winners) != 1 {
		paths := make([]string, 0, len(candidates))
		for _, candidate := range candidates {
			paths = append(paths, candidate.ProtoPath)
		}
		fmt.Printf("[grpc-extractor] Ambiguous proto resolution for service %q from %s: %d candidates tied at score %d among [%s]\n", serviceName, sourceFilePath, len(winners), best, strings.Join(paths, ", "))
		return nil
	}
	winner := winners[0]
	return &winner
}

func ServiceContractID(pkg string, serviceName string) string {
	if pkg == "" {
		return "grpc::" + serviceName + "/*"
	}
	return "grpc::" + pkg + "." + serviceName + "/*"
}

func scanGRPCSourceFile(rel string, content string, protoMap map[string][]ProtoServiceInfo) []StoredContract {
	ext := strings.ToLower(filepath.Ext(rel))
	switch ext {
	case ".go":
		return scanGoGRPC(rel, content, protoMap)
	case ".java":
		return scanJavaGRPC(rel, content, protoMap)
	case ".py":
		return scanPythonGRPC(rel, content, protoMap)
	case ".ts", ".tsx", ".js", ".jsx":
		return scanTSGRPC(rel, content, protoMap)
	default:
		return nil
	}
}

func scanGoGRPC(rel string, content string, protoMap map[string][]ProtoServiceInfo) []StoredContract {
	out := make([]StoredContract, 0)
	for _, match := range regexp.MustCompile(`Register(\w+)Server\(`).FindAllStringSubmatch(content, -1) {
		out = append(out, grpcServiceDetectionContract(rel, match[1], "", "provider", "go_register", protoMap, 0.8, 0.65))
	}
	for _, match := range regexp.MustCompile(`Unimplemented(\w+)Server`).FindAllStringSubmatch(content, -1) {
		out = append(out, grpcServiceDetectionContract(rel, match[1], "", "provider", "go_unimplemented", protoMap, 0.8, 0.65))
	}
	for _, match := range regexp.MustCompile(`New(\w+)Client\(`).FindAllStringSubmatch(content, -1) {
		out = append(out, grpcServiceDetectionContract(rel, match[1], "", "consumer", "go_client", protoMap, 0.75, 0.55))
	}
	return out
}

func scanJavaGRPC(rel string, content string, protoMap map[string][]ProtoServiceInfo) []StoredContract {
	out := make([]StoredContract, 0)
	for _, match := range regexp.MustCompile(`(\w+)Grpc\.(\w+)ImplBase`).FindAllStringSubmatch(content, -1) {
		out = append(out, grpcServiceDetectionContract(rel, match[1], "", "provider", "java_grpc_service", protoMap, 0.8, 0.65))
	}
	for _, match := range regexp.MustCompile(`(\w+)Grpc\.(?:\w+BlockingStub|newBlockingStub)`).FindAllStringSubmatch(content, -1) {
		out = append(out, grpcServiceDetectionContract(rel, match[1], "", "consumer", "java_stub", protoMap, 0.75, 0.55))
	}
	return out
}

func scanPythonGRPC(rel string, content string, protoMap map[string][]ProtoServiceInfo) []StoredContract {
	out := make([]StoredContract, 0)
	for _, match := range regexp.MustCompile(`add_(\w+)Servicer_to_server\(`).FindAllStringSubmatch(content, -1) {
		out = append(out, grpcServiceDetectionContract(rel, match[1], "", "provider", "python_servicer", protoMap, 0.8, 0.65))
	}
	for _, match := range regexp.MustCompile(`(\w+)Stub\(`).FindAllStringSubmatch(content, -1) {
		out = append(out, grpcServiceDetectionContract(rel, match[1], "", "consumer", "python_stub", protoMap, 0.75, 0.55))
	}
	return out
}

func scanTSGRPC(rel string, content string, protoMap map[string][]ProtoServiceInfo) []StoredContract {
	out := make([]StoredContract, 0)
	for _, match := range regexp.MustCompile(`@GrpcMethod\(\s*['"](\w+)['"]\s*,\s*['"](\w+)['"]`).FindAllStringSubmatch(content, -1) {
		out = append(out, grpcServiceDetectionContract(rel, match[1], match[2], "provider", "ts_grpc_method", protoMap, 0.8, 0.8))
	}
	for _, match := range regexp.MustCompile(`getService<(\w+)>\(\s*['"](\w+)['"]`).FindAllStringSubmatch(content, -1) {
		service := strings.TrimSuffix(match[1], "Client")
		if service == "" {
			service = match[2]
		}
		out = append(out, grpcServiceDetectionContract(rel, service, "", "consumer", "ts_get_service", protoMap, 0.75, 0.55))
	}
	for _, match := range regexp.MustCompile(`new\s+(\w+Service)Client\(`).FindAllStringSubmatch(content, -1) {
		out = append(out, grpcServiceDetectionContract(rel, match[1], "", "consumer", "ts_client_constructor", protoMap, 0.75, 0.55))
	}
	for _, match := range regexp.MustCompile(`new\s+[\w.]+\.([A-Za-z0-9_.]+Service)\(`).FindAllStringSubmatch(content, -1) {
		parts := strings.Split(match[1], ".")
		service := parts[len(parts)-1]
		out = append(out, grpcServiceDetectionContract(rel, service, "", "consumer", "ts_load_package_definition", protoMap, 0.75, 0.55))
	}
	if strings.Contains(content, "@GrpcClient") {
		for serviceName := range protoMap {
			out = append(out, grpcServiceDetectionContract(rel, serviceName, "", "consumer", "ts_grpc_client", protoMap, 0.75, 0.55))
			break
		}
	}
	return out
}

func grpcServiceDetectionContract(rel string, serviceName string, methodName string, role string, source string, protoMap map[string][]ProtoServiceInfo, confidenceWithProto float64, confidenceWithoutProto float64) StoredContract {
	candidates := protoMap[serviceName]
	proto := ResolveProtoConflict(serviceName, rel, candidates)
	if len(candidates) > 0 && proto == nil {
		return StoredContract{}
	}
	pkg := ""
	confidence := confidenceWithoutProto
	if proto != nil {
		pkg = proto.Package
		confidence = confidenceWithProto
	}
	contractID := ServiceContractID(pkg, serviceName)
	if methodName != "" {
		contractID = grpcMethodContractID(pkg, serviceName, methodName)
	}
	meta := map[string]any{"service": serviceName, "source": source, "extractionStrategy": "source_scan"}
	if methodName != "" {
		meta["method"] = methodName
	}
	return makeGRPCContract(contractID, role, rel, grpcSymbolName(serviceName, methodName), confidence, meta)
}

func makeGRPCContract(contractID string, role string, filePath string, symbolName string, confidence float64, meta map[string]any) StoredContract {
	if contractID == "" {
		return StoredContract{}
	}
	return StoredContract{
		ContractID: contractID,
		Type:       "grpc",
		Role:       role,
		SymbolUID:  "",
		SymbolRef:  SymbolRef{FilePath: filepath.ToSlash(filePath), Name: symbolName},
		SymbolName: symbolName,
		Confidence: confidence,
		Meta:       meta,
	}
}

func grpcMethodContractID(pkg string, service string, method string) string {
	prefix := service
	if pkg != "" {
		prefix = pkg + "." + service
	}
	return "grpc::" + prefix + "/" + method
}

func grpcSymbolName(service string, method string) string {
	if method == "" {
		return service
	}
	return service + "." + method
}

func dedupeGRPCContracts(items []StoredContract) []StoredContract {
	byKey := make(map[string]StoredContract, len(items))
	for _, item := range items {
		if item.ContractID == "" {
			continue
		}
		key := item.ContractID + "\x00" + item.Role + "\x00" + item.SymbolRef.FilePath
		existing, ok := byKey[key]
		if !ok || item.Confidence > existing.Confidence || (item.Confidence == existing.Confidence && fmt.Sprint(item.Meta["source"]) < fmt.Sprint(existing.Meta["source"])) {
			byKey[key] = item
		}
	}
	keys := make([]string, 0, len(byKey))
	for key := range byKey {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]StoredContract, 0, len(keys))
	for _, key := range keys {
		out = append(out, byKey[key])
	}
	return out
}

func sortedProtoServiceNames(protoMap map[string][]ProtoServiceInfo) []string {
	names := make([]string, 0, len(protoMap))
	for name := range protoMap {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func extractProtoImports(content string) []string {
	imports := make([]string, 0)
	for _, match := range regexp.MustCompile(`(?m)^\s*import\s+"([^"]+)"\s*;`).FindAllStringSubmatch(content, -1) {
		imports = append(imports, match[1])
	}
	return imports
}

func stripProtoCommentsAndStrings(content string) string {
	out := []rune(content)
	runes := []rune(content)
	for i := 0; i < len(runes); {
		ch := runes[i]
		next := rune(0)
		if i+1 < len(runes) {
			next = runes[i+1]
		}
		if ch == '/' && next == '/' {
			out[i], out[i+1] = ' ', ' '
			i += 2
			for i < len(runes) && runes[i] != '\n' {
				if runes[i] != '\r' {
					out[i] = ' '
				}
				i++
			}
			continue
		}
		if ch == '/' && next == '*' {
			out[i], out[i+1] = ' ', ' '
			i += 2
			for i < len(runes) {
				if runes[i] == '*' && i+1 < len(runes) && runes[i+1] == '/' {
					out[i], out[i+1] = ' ', ' '
					i += 2
					break
				}
				if runes[i] != '\n' && runes[i] != '\r' {
					out[i] = ' '
				}
				i++
			}
			continue
		}
		if ch == '"' || ch == '\'' {
			quote := ch
			out[i] = ' '
			i++
			for i < len(runes) {
				if runes[i] == '\\' && i+1 < len(runes) {
					out[i], out[i+1] = ' ', ' '
					i += 2
					continue
				}
				if runes[i] == quote {
					out[i] = ' '
					i++
					break
				}
				if runes[i] != '\n' && runes[i] != '\r' {
					out[i] = ' '
				}
				i++
			}
			continue
		}
		i++
	}
	return string(out)
}

type protoServiceBlock struct {
	name string
	body string
}

func extractProtoServiceBlocks(content string) []protoServiceBlock {
	sanitized := stripProtoCommentsAndStrings(content)
	headerRe := regexp.MustCompile(`service\s+(\w+)\s*\{`)
	matches := headerRe.FindAllStringSubmatchIndex(sanitized, -1)
	out := make([]protoServiceBlock, 0, len(matches))
	for _, match := range matches {
		serviceName := sanitized[match[2]:match[3]]
		bodyStart := match[1]
		depth := 1
		pos := bodyStart
		for pos < len(sanitized) && depth > 0 {
			switch sanitized[pos] {
			case '{':
				depth++
			case '}':
				depth--
			}
			pos++
		}
		if depth != 0 {
			continue
		}
		out = append(out, protoServiceBlock{name: serviceName, body: content[bodyStart : pos-1]})
	}
	return out
}

func longestSharedSegmentRun(aPath string, bPath string) int {
	a := pathSegments(aPath)
	b := pathSegments(bPath)
	best := 0
	for i := range a {
		for j := range b {
			run := 0
			for i+run < len(a) && j+run < len(b) && a[i+run] == b[j+run] {
				run++
			}
			if run > best {
				best = run
			}
		}
	}
	return best
}

func pathSegments(value string) []string {
	parts := strings.Split(filepath.ToSlash(value), "/")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if part != "" && part != "." {
			out = append(out, part)
		}
	}
	return out
}
