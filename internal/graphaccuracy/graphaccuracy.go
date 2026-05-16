package graphaccuracy

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

type Options struct {
	Repo          string
	NodeGraphPath string
	GoGraphPath   string
	OutPath       string
	MaxExamples   int
}

type GraphFile struct {
	Nodes         []GraphNode         `json:"nodes"`
	Relationships []GraphRelationship `json:"relationships"`
}

type GraphNode struct {
	ID         string         `json:"id"`
	Label      string         `json:"label"`
	Properties map[string]any `json:"properties"`
}

type GraphRelationship struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	SourceID string `json:"sourceId"`
	TargetID string `json:"targetId"`
	Reason   string `json:"reason"`
}

type AnalyzerMetrics struct {
	Matched         int      `json:"matched"`
	Expected        int      `json:"expected"`
	GraphCandidates int      `json:"graphCandidates,omitempty"`
	RecallPct       float64  `json:"recallPct"`
	PrecisionPct    float64  `json:"precisionPct,omitempty"`
	PrecisionNote   string   `json:"precisionNote,omitempty"`
	MissingExamples []string `json:"missingExamples,omitempty"`
	ExtraExamples   []string `json:"extraExamples,omitempty"`
}

type Result struct {
	GeneratedAt string `json:"generatedAt"`
	Inputs      struct {
		Repo      string `json:"repo"`
		NodeGraph string `json:"nodeGraph"`
		GoGraph   string `json:"goGraph"`
	} `json:"inputs"`
	Scope struct {
		CommonGoFiles int      `json:"commonGoFiles"`
		NodeGoFiles   int      `json:"nodeGoFiles"`
		GoGoFiles     int      `json:"goGoFiles"`
		Modules       []string `json:"modules"`
	} `json:"scope"`
	Definitions map[string]map[string]AnalyzerMetrics `json:"definitions"`
	Imports     map[string]AnalyzerMetrics            `json:"imports"`
	Calls       map[string]AnalyzerMetrics            `json:"calls"`
	Notes       []string                              `json:"notes"`
}

type GateFailure struct {
	Gate            string   `json:"gate"`
	Matched         int      `json:"matched"`
	Expected        int      `json:"expected"`
	GraphCandidates int      `json:"graphCandidates,omitempty"`
	RecallPct       float64  `json:"recallPct"`
	PrecisionPct    float64  `json:"precisionPct,omitempty"`
	MissingExamples []string `json:"missingExamples,omitempty"`
	ExtraExamples   []string `json:"extraExamples,omitempty"`
}

type moduleInfo struct {
	Path   string
	DirRel string
}

type fileInfo struct {
	RelPath     string
	AbsPath     string
	DirRel      string
	PackageName string
	Imports     map[string]string
	DotImports  []string
	AST         *ast.File
	Fset        *token.FileSet
}

type packageInfo struct {
	DirRel       string
	PackageName  string
	ImportPath   string
	Files        []string
	NonTestFiles []string
	Functions    map[string]string
	AllFunctions map[string][]string
}

var hashSuffix = regexp.MustCompile(`#\d+$`)

func Run(options Options) (Result, error) {
	if options.NodeGraphPath == "" || options.GoGraphPath == "" {
		return Result{}, fmt.Errorf("node and go graph paths are required")
	}
	if options.MaxExamples <= 0 {
		options.MaxExamples = 50
	}
	repo := options.Repo
	if strings.TrimSpace(repo) == "" {
		repo = "."
	}
	repoAbs, err := filepath.Abs(repo)
	if err != nil {
		return Result{}, fmt.Errorf("resolve repo: %w", err)
	}
	nodeGraph, err := ReadGraph(options.NodeGraphPath)
	if err != nil {
		return Result{}, err
	}
	goGraph, err := ReadGraph(options.GoGraphPath)
	if err != nil {
		return Result{}, err
	}

	nodeGoFiles := graphGoFiles(nodeGraph)
	goGoFiles := graphGoFiles(goGraph)
	commonGoFiles := intersect(nodeGoFiles, goGoFiles)

	modules, err := discoverModules(repoAbs)
	if err != nil {
		return Result{}, err
	}
	files := parseGoFiles(repoAbs, commonGoFiles)
	packages := buildPackages(files, modules)

	expectedDefs := expectedDefinitions(files)
	expectedImports := expectedImportEdges(files, packages, modules, commonGoFiles)
	expectedCalls := expectedDirectCallEdges(files, packages, modules)

	nodeFacts := buildGraphFacts(nodeGraph, commonGoFiles)
	goFacts := buildGraphFacts(goGraph, commonGoFiles)

	var r Result
	r.GeneratedAt = time.Now().Format(time.RFC3339)
	r.Inputs.Repo = repoAbs
	r.Inputs.NodeGraph = options.NodeGraphPath
	r.Inputs.GoGraph = options.GoGraphPath
	r.Scope.CommonGoFiles = len(commonGoFiles)
	r.Scope.NodeGoFiles = len(nodeGoFiles)
	r.Scope.GoGoFiles = len(goGoFiles)
	for _, m := range modules {
		r.Scope.Modules = append(r.Scope.Modules, m.Path+" => "+m.DirRel)
	}
	sort.Strings(r.Scope.Modules)

	r.Definitions = map[string]map[string]AnalyzerMetrics{}
	for _, label := range definitionLabels() {
		expected := filterByPrefix(expectedDefs, label+"|")
		nodeCandidates := filterGraphDefCandidates(nodeFacts.DefinitionKeys, label+"|")
		goCandidates := filterGraphDefCandidates(goFacts.DefinitionKeys, label+"|")
		r.Definitions[label] = map[string]AnalyzerMetrics{
			"nodeMcp": compareSets(expected, nodeCandidates, true, options.MaxExamples),
			"goLocal": compareSets(expected, goCandidates, true, options.MaxExamples),
		}
	}

	r.Imports = map[string]AnalyzerMetrics{
		"nodeMcp": compareSets(expectedImports, nodeFacts.ImportEdges, true, options.MaxExamples),
		"goLocal": compareSets(expectedImports, goFacts.ImportEdges, true, options.MaxExamples),
	}

	nodeCallMetric := compareSets(expectedCalls, nodeFacts.CallEdges, false, options.MaxExamples)
	nodeCallMetric.PrecisionNote = "Precision is not reported for CALLS because the ground truth only covers direct identifier and imported package function calls; valid method/type-driven CALLS are outside this subset."
	goCallMetric := compareSets(expectedCalls, goFacts.CallEdges, false, options.MaxExamples)
	goCallMetric.PrecisionNote = nodeCallMetric.PrecisionNote
	r.Calls = map[string]AnalyzerMetrics{
		"nodeMcp": nodeCallMetric,
		"goLocal": goCallMetric,
	}

	r.Notes = []string{
		fmt.Sprintf("Definition ground truth is built from Go standard parser over the %d Go files present in both API graphs.", len(commonGoFiles)),
		"Import ground truth includes local package imports only, expanded to non-test Go files in the imported package directory.",
		"CALLS ground truth is a high-confidence subset: direct same-package function calls, dot-import function calls, and imported-package function calls. Receiver method dispatch and type-driven calls are intentionally excluded.",
		"Const and Variable are compared at file/name level because graph node IDs do not encode lexical scope in a way this probe can compare without ambiguity.",
	}

	if options.OutPath != "" {
		if err := WriteResult(options.OutPath, r); err != nil {
			return Result{}, err
		}
	}
	return r, nil
}

func ReadGraph(path string) (GraphFile, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return GraphFile{}, fmt.Errorf("read graph %s: %w", path, err)
	}
	var g GraphFile
	if err := json.Unmarshal(raw, &g); err != nil {
		return GraphFile{}, fmt.Errorf("decode graph %s: %w", path, err)
	}
	return g, nil
}

func WriteResult(path string, result Result) error {
	raw, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal output: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, append(raw, '\n'), 0o644)
}

func GoLocalFailures(result Result) []GateFailure {
	var failures []GateFailure
	for _, label := range definitionLabels() {
		metric, ok := result.Definitions[label]["goLocal"]
		if !ok {
			continue
		}
		if metric.Matched != metric.Expected || metric.Matched != metric.GraphCandidates {
			failures = append(failures, gateFailure("Definition "+label, metric))
		}
	}
	if metric, ok := result.Imports["goLocal"]; ok {
		if metric.Matched != metric.Expected || metric.Matched != metric.GraphCandidates {
			failures = append(failures, gateFailure("Local IMPORTS", metric))
		}
	}
	if metric, ok := result.Calls["goLocal"]; ok {
		if metric.Matched != metric.Expected {
			failures = append(failures, gateFailure("Direct CALLS subset", metric))
		}
	}
	return failures
}

func SummaryLines(result Result) []string {
	lines := []string{
		fmt.Sprintf("common_go_files=%d expected_defs=%d expected_import_edges=%d expected_direct_calls=%d", result.Scope.CommonGoFiles, expectedDefinitionCount(result), result.Imports["goLocal"].Expected, result.Calls["goLocal"].Expected),
	}
	for _, label := range definitionLabels() {
		if metric, ok := result.Definitions[label]["goLocal"]; ok {
			lines = append(lines, fmt.Sprintf("definition.%s.goLocal=%d/%d recall=%.2f precision=%.2f graphCandidates=%d", label, metric.Matched, metric.Expected, metric.RecallPct, metric.PrecisionPct, metric.GraphCandidates))
		}
	}
	if metric, ok := result.Imports["goLocal"]; ok {
		lines = append(lines, fmt.Sprintf("imports.goLocal=%d/%d recall=%.2f precision=%.2f graphCandidates=%d", metric.Matched, metric.Expected, metric.RecallPct, metric.PrecisionPct, metric.GraphCandidates))
	}
	if metric, ok := result.Calls["goLocal"]; ok {
		lines = append(lines, fmt.Sprintf("calls.goLocal=%d/%d recall=%.2f graphCandidates=%d", metric.Matched, metric.Expected, metric.RecallPct, metric.GraphCandidates))
	}
	return lines
}

func gateFailure(gate string, metric AnalyzerMetrics) GateFailure {
	return GateFailure{
		Gate:            gate,
		Matched:         metric.Matched,
		Expected:        metric.Expected,
		GraphCandidates: metric.GraphCandidates,
		RecallPct:       metric.RecallPct,
		PrecisionPct:    metric.PrecisionPct,
		MissingExamples: metric.MissingExamples,
		ExtraExamples:   metric.ExtraExamples,
	}
}

func expectedDefinitionCount(result Result) int {
	total := 0
	for _, label := range definitionLabels() {
		total += result.Definitions[label]["goLocal"].Expected
	}
	return total
}

func definitionLabels() []string {
	return []string{"Function", "Method", "Struct", "Interface", "TypeAlias", "Const", "Variable"}
}

type graphFacts struct {
	DefinitionKeys map[string]bool
	ImportEdges    map[string]bool
	CallEdges      map[string]bool
}

func buildGraphFacts(g GraphFile, fileSet map[string]bool) graphFacts {
	idToKey := map[string]string{}
	definitionKeys := map[string]bool{}
	for _, n := range g.Nodes {
		key := nodeCanonicalKey(n, fileSet)
		if key != "" {
			idToKey[n.ID] = key
			definitionKeys[key] = true
		}
	}
	imports := map[string]bool{}
	calls := map[string]bool{}
	for _, rel := range g.Relationships {
		switch rel.Type {
		case "IMPORTS":
			src := strings.TrimPrefix(rel.SourceID, "File:")
			dst := strings.TrimPrefix(rel.TargetID, "File:")
			if strings.HasSuffix(src, ".go") && strings.HasSuffix(dst, ".go") && fileSet[filepath.ToSlash(src)] && fileSet[filepath.ToSlash(dst)] {
				imports["File:"+src+"->File:"+dst] = true
			}
		case "CALLS":
			src := idToKey[rel.SourceID]
			dst := idToKey[rel.TargetID]
			if src != "" && dst != "" {
				calls[src+"->"+dst] = true
			}
		}
	}
	return graphFacts{DefinitionKeys: definitionKeys, ImportEdges: imports, CallEdges: calls}
}

func nodeCanonicalKey(n GraphNode, fileSet map[string]bool) string {
	label := n.Label
	if !isDefinitionLabel(label) {
		return ""
	}
	filePath := propString(n.Properties, "filePath")
	if !strings.HasSuffix(filePath, ".go") {
		return ""
	}
	filePath = filepath.ToSlash(filePath)
	if !fileSet[filePath] {
		return ""
	}
	name := propString(n.Properties, "name")
	if label == "Method" {
		if qualified := propString(n.Properties, "qualifiedName"); qualified != "" {
			name = qualified
		} else if parsed := idName(n.ID); parsed != "" {
			name = parsed
		}
	} else if parsed := idName(n.ID); name == "" && parsed != "" {
		name = parsed
	}
	name = hashSuffix.ReplaceAllString(name, "")
	if filePath == "" || name == "" {
		return ""
	}
	return label + "|" + filepath.ToSlash(filePath) + "|" + name
}

func idName(id string) string {
	idx := strings.LastIndex(id, ":")
	if idx < 0 || idx+1 >= len(id) {
		return ""
	}
	return hashSuffix.ReplaceAllString(id[idx+1:], "")
}

func isDefinitionLabel(label string) bool {
	switch label {
	case "Function", "Method", "Struct", "Interface", "TypeAlias", "Const", "Variable":
		return true
	default:
		return false
	}
}

func expectedDefinitions(files map[string]*fileInfo) map[string]bool {
	out := map[string]bool{}
	for _, info := range files {
		ast.Inspect(info.AST, func(n ast.Node) bool {
			switch d := n.(type) {
			case *ast.FuncDecl:
				if d.Recv == nil {
					out["Function|"+info.RelPath+"|"+d.Name.Name] = true
				} else {
					recv := receiverName(d.Recv)
					if recv != "" {
						out["Method|"+info.RelPath+"|"+recv+"."+d.Name.Name] = true
					}
				}
			case *ast.GenDecl:
				addGenDeclDefinitions(info.RelPath, d, out)
			case *ast.AssignStmt:
				if d.Tok == token.DEFINE {
					for _, expr := range d.Lhs {
						if ident, ok := expr.(*ast.Ident); ok && ident.Name != "_" {
							out["Variable|"+info.RelPath+"|"+ident.Name] = true
						}
					}
				}
			case *ast.RangeStmt:
				if d.Tok == token.DEFINE {
					for _, expr := range []ast.Expr{d.Key, d.Value} {
						if ident, ok := expr.(*ast.Ident); ok && ident.Name != "_" {
							out["Variable|"+info.RelPath+"|"+ident.Name] = true
						}
					}
				}
			}
			return true
		})
	}
	return out
}

func addGenDeclDefinitions(relPath string, d *ast.GenDecl, out map[string]bool) {
	for _, spec := range d.Specs {
		switch s := spec.(type) {
		case *ast.TypeSpec:
			switch typ := s.Type.(type) {
			case *ast.StructType:
				out["Struct|"+relPath+"|"+s.Name.Name] = true
			case *ast.InterfaceType:
				out["Interface|"+relPath+"|"+s.Name.Name] = true
				for _, method := range typ.Methods.List {
					for _, name := range method.Names {
						out["Method|"+relPath+"|"+s.Name.Name+"."+name.Name] = true
					}
				}
			default:
				out["TypeAlias|"+relPath+"|"+s.Name.Name] = true
			}
		case *ast.ValueSpec:
			label := ""
			if d.Tok == token.CONST {
				label = "Const"
			} else if d.Tok == token.VAR {
				label = "Variable"
			}
			if label != "" {
				for _, name := range s.Names {
					if name.Name != "_" {
						out[label+"|"+relPath+"|"+name.Name] = true
					}
				}
			}
		}
	}
}

func expectedImportEdges(files map[string]*fileInfo, packages map[string]*packageInfo, modules []moduleInfo, fileSet map[string]bool) map[string]bool {
	out := map[string]bool{}
	for _, info := range files {
		for _, importPath := range info.Imports {
			targetDir := localImportDir(importPath, modules)
			if targetDir == "" {
				continue
			}
			pkg := packages[targetDir]
			if pkg == nil {
				continue
			}
			for _, targetFile := range pkg.NonTestFiles {
				if fileSet[targetFile] {
					out["File:"+info.RelPath+"->File:"+targetFile] = true
				}
			}
		}
	}
	return out
}

func expectedDirectCallEdges(files map[string]*fileInfo, packages map[string]*packageInfo, modules []moduleInfo) map[string]bool {
	out := map[string]bool{}
	for _, info := range files {
		sourcePkg := packages[info.DirRel]
		if sourcePkg == nil {
			continue
		}
		importAliasToPkg := map[string]*packageInfo{}
		dotImportPkgs := []*packageInfo{}
		for alias, importPath := range info.Imports {
			targetDir := localImportDir(importPath, modules)
			if targetDir == "" {
				continue
			}
			pkg := packages[targetDir]
			if pkg == nil {
				continue
			}
			if alias == "." {
				dotImportPkgs = append(dotImportPkgs, pkg)
			} else if alias != "_" {
				importAliasToPkg[alias] = pkg
			}
		}
		for _, decl := range info.AST.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			sourceKey := funcDeclKey(info.RelPath, fn)
			if sourceKey == "" {
				continue
			}
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				switch f := call.Fun.(type) {
				case *ast.Ident:
					if target := sourcePkg.Functions[f.Name]; target != "" {
						out[sourceKey+"->"+target] = true
						return true
					}
					for _, pkg := range dotImportPkgs {
						if target := pkg.Functions[f.Name]; target != "" {
							out[sourceKey+"->"+target] = true
						}
					}
				case *ast.SelectorExpr:
					x, ok := f.X.(*ast.Ident)
					if !ok {
						return true
					}
					if pkg := importAliasToPkg[x.Name]; pkg != nil {
						if target := pkg.Functions[f.Sel.Name]; target != "" {
							out[sourceKey+"->"+target] = true
						}
					}
				}
				return true
			})
		}
	}
	return out
}

func funcDeclKey(relPath string, fn *ast.FuncDecl) string {
	if fn.Recv == nil {
		return "Function|" + relPath + "|" + fn.Name.Name
	}
	recv := receiverName(fn.Recv)
	if recv == "" {
		return ""
	}
	return "Method|" + relPath + "|" + recv + "." + fn.Name.Name
}

func receiverName(recv *ast.FieldList) string {
	if recv == nil || len(recv.List) == 0 {
		return ""
	}
	return exprTypeName(recv.List[0].Type)
}

func exprTypeName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return exprTypeName(t.X)
	case *ast.SelectorExpr:
		return exprTypeName(t.X) + "." + t.Sel.Name
	case *ast.IndexExpr:
		return exprTypeName(t.X)
	case *ast.IndexListExpr:
		return exprTypeName(t.X)
	default:
		return ""
	}
}

func buildPackages(files map[string]*fileInfo, modules []moduleInfo) map[string]*packageInfo {
	packages := map[string]*packageInfo{}
	for _, info := range files {
		pkg := packages[info.DirRel]
		if pkg == nil {
			pkg = &packageInfo{
				DirRel:       info.DirRel,
				PackageName:  info.PackageName,
				ImportPath:   importPathForDir(info.DirRel, modules),
				Functions:    map[string]string{},
				AllFunctions: map[string][]string{},
			}
			packages[info.DirRel] = pkg
		}
		pkg.Files = append(pkg.Files, info.RelPath)
		if !strings.HasSuffix(info.RelPath, "_test.go") && info.PackageName == pkg.PackageName {
			pkg.NonTestFiles = append(pkg.NonTestFiles, info.RelPath)
		}
		for _, decl := range info.AST.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv != nil {
				continue
			}
			key := "Function|" + info.RelPath + "|" + fn.Name.Name
			pkg.AllFunctions[fn.Name.Name] = append(pkg.AllFunctions[fn.Name.Name], key)
		}
	}
	for _, pkg := range packages {
		sort.Strings(pkg.Files)
		sort.Strings(pkg.NonTestFiles)
		for name, keys := range pkg.AllFunctions {
			if len(keys) == 1 {
				pkg.Functions[name] = keys[0]
			}
		}
	}
	return packages
}

func importPathForDir(dirRel string, modules []moduleInfo) string {
	best := moduleInfo{}
	bestLen := -1
	for _, m := range modules {
		if dirRel == m.DirRel || strings.HasPrefix(dirRel, m.DirRel+"/") {
			if len(m.DirRel) > bestLen {
				best = m
				bestLen = len(m.DirRel)
			}
		}
	}
	if best.Path == "" {
		return ""
	}
	suffix := strings.TrimPrefix(dirRel, best.DirRel)
	suffix = strings.TrimPrefix(suffix, "/")
	if suffix == "" {
		return best.Path
	}
	return best.Path + "/" + suffix
}

func localImportDir(importPath string, modules []moduleInfo) string {
	best := moduleInfo{}
	for _, m := range modules {
		if importPath == m.Path || strings.HasPrefix(importPath, m.Path+"/") {
			if len(m.Path) > len(best.Path) {
				best = m
			}
		}
	}
	if best.Path == "" {
		return ""
	}
	suffix := strings.TrimPrefix(importPath, best.Path)
	suffix = strings.TrimPrefix(suffix, "/")
	if suffix == "" {
		return best.DirRel
	}
	if best.DirRel == "." {
		return filepath.ToSlash(suffix)
	}
	return filepath.ToSlash(best.DirRel + "/" + suffix)
}

func parseGoFiles(repoAbs string, relFiles map[string]bool) map[string]*fileInfo {
	out := map[string]*fileInfo{}
	for rel := range relFiles {
		abs := filepath.Join(repoAbs, filepath.FromSlash(rel))
		fset := token.NewFileSet()
		parsed, err := parser.ParseFile(fset, abs, nil, parser.ParseComments)
		if err != nil {
			continue
		}
		info := &fileInfo{
			RelPath:     rel,
			AbsPath:     abs,
			DirRel:      dirRel(rel),
			PackageName: parsed.Name.Name,
			Imports:     map[string]string{},
			AST:         parsed,
			Fset:        fset,
		}
		for _, spec := range parsed.Imports {
			importPath := strings.Trim(spec.Path.Value, `"`)
			alias := ""
			if spec.Name != nil {
				alias = spec.Name.Name
			}
			if alias == "" {
				alias = pathBase(importPath)
			}
			info.Imports[alias] = importPath
			if alias == "." {
				info.DotImports = append(info.DotImports, importPath)
			}
		}
		out[rel] = info
	}
	return out
}

func dirRel(rel string) string {
	dir := filepath.ToSlash(filepath.Dir(rel))
	if dir == "." {
		return "."
	}
	return dir
}

func pathBase(p string) string {
	p = strings.TrimSuffix(p, "/")
	idx := strings.LastIndex(p, "/")
	if idx >= 0 {
		return p[idx+1:]
	}
	return p
}

func discoverModules(repoAbs string) ([]moduleInfo, error) {
	var modules []moduleInfo
	err := filepath.WalkDir(repoAbs, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			name := d.Name()
			if name == ".git" || name == "node_modules" || name == ".tmp" || name == "coverage" || name == "dist" {
				return filepath.SkipDir
			}
			return nil
		}
		if d.Name() != "go.mod" {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		modulePath := parseModulePath(string(raw))
		if modulePath == "" {
			return nil
		}
		rel, err := filepath.Rel(repoAbs, filepath.Dir(path))
		if err != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)
		if rel == "." {
			rel = "."
		}
		modules = append(modules, moduleInfo{Path: modulePath, DirRel: rel})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("discover modules: %w", err)
	}
	sort.Slice(modules, func(i, j int) bool {
		return len(modules[i].Path) > len(modules[j].Path)
	})
	return modules, nil
}

func parseModulePath(raw string) string {
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module "))
		}
	}
	return ""
}

func graphGoFiles(g GraphFile) map[string]bool {
	out := map[string]bool{}
	for _, n := range g.Nodes {
		if n.Label != "File" {
			continue
		}
		filePath := propString(n.Properties, "filePath")
		if strings.HasSuffix(filePath, ".go") {
			out[filepath.ToSlash(filePath)] = true
		}
	}
	return out
}

func propString(props map[string]any, key string) string {
	if props == nil {
		return ""
	}
	v, ok := props[key]
	if !ok || v == nil {
		return ""
	}
	switch x := v.(type) {
	case string:
		return x
	default:
		return fmt.Sprint(x)
	}
}

func filterByPrefix(set map[string]bool, prefix string) map[string]bool {
	out := map[string]bool{}
	for k := range set {
		if strings.HasPrefix(k, prefix) {
			out[k] = true
		}
	}
	return out
}

func filterGraphDefCandidates(set map[string]bool, prefix string) map[string]bool {
	return filterByPrefix(set, prefix)
}

func compareSets(expected map[string]bool, actual map[string]bool, includePrecision bool, maxExamples int) AnalyzerMetrics {
	matched := 0
	var missing []string
	for key := range expected {
		if actual[key] {
			matched++
		} else if len(missing) < maxExamples {
			missing = append(missing, key)
		}
	}
	var extra []string
	if includePrecision {
		for key := range actual {
			if !expected[key] && len(extra) < maxExamples {
				extra = append(extra, key)
			}
		}
	}
	sort.Strings(missing)
	sort.Strings(extra)
	metric := AnalyzerMetrics{
		Matched:         matched,
		Expected:        len(expected),
		GraphCandidates: len(actual),
		RecallPct:       pct(matched, len(expected)),
		MissingExamples: missing,
	}
	if includePrecision {
		metric.PrecisionPct = pct(matched, len(actual))
		metric.ExtraExamples = extra
	}
	return metric
}

func pct(numerator int, denominator int) float64 {
	if denominator == 0 {
		return 0
	}
	return float64(int((float64(numerator)/float64(denominator))*10000+0.5)) / 100
}

func intersect(a, b map[string]bool) map[string]bool {
	out := map[string]bool{}
	for k := range a {
		if b[k] {
			out[k] = true
		}
	}
	return out
}
