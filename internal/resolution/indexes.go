package resolution

import (
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/tamnguyendinh/avmatrix-go/internal/graph"
	"github.com/tamnguyendinh/avmatrix-go/internal/scanner"
	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

type defRef struct {
	Fact    scopeir.DefinitionFact
	GraphID string
}

type bindingRef struct {
	Def    defRef
	Origin scopeir.BindingOrigin
	Via    *resolvedImport
}

type resolvedImport struct {
	Fact        scopeir.ImportFact
	SourceScope string
	TargetFile  string
	TargetFiles []string
	TargetDef   *defRef
	LinkStatus  string
}

type importReceiverKey struct {
	filePath  string
	localName string
}

type heritageResolution struct {
	Fact   scopeir.HeritageFact
	Owner  defRef
	Target defRef
}

type workspace struct {
	files []scopeir.ScopeIR

	fileSet           map[string]struct{}
	fileHashes        map[string]string
	fileLanguages     map[string]scanner.Language
	moduleScopeByFile map[string]string

	scopes     map[string]scopeir.ScopeFact
	defsByID   map[string]defRef
	scopeByDef map[string]string
	defsByFile map[string][]defRef
	defsByName map[string][]defRef

	scopeBindings     map[string]map[string][]bindingRef
	typeBindings      map[string]map[string][]scopeir.TypeRef
	ownerMembers      map[string]map[string][]defRef
	imports           []resolvedImport
	importsByReceiver map[importReceiverKey][]int
	heritage          []heritageResolution

	bindingAccumulator *bindingAccumulator
}

func buildWorkspace(files []scopeir.ScopeIR) (*workspace, error) {
	size := measureWorkspace(files)
	w := &workspace{
		files:              make([]scopeir.ScopeIR, 0, size.files),
		fileSet:            make(map[string]struct{}, size.files),
		fileHashes:         make(map[string]string, size.files),
		fileLanguages:      make(map[string]scanner.Language, size.files),
		moduleScopeByFile:  make(map[string]string, size.files),
		scopes:             make(map[string]scopeir.ScopeFact, size.scopes),
		defsByID:           make(map[string]defRef, size.definitions),
		scopeByDef:         make(map[string]string, size.ownedDefs),
		defsByFile:         make(map[string][]defRef, size.files),
		defsByName:         make(map[string][]defRef, size.definitions*2),
		scopeBindings:      make(map[string]map[string][]bindingRef, size.scopes),
		typeBindings:       make(map[string]map[string][]scopeir.TypeRef, size.scopes),
		ownerMembers:       make(map[string]map[string][]defRef, size.definitions),
		imports:            make([]resolvedImport, 0, size.imports),
		importsByReceiver:  make(map[importReceiverKey][]int, size.imports),
		heritage:           make([]heritageResolution, 0, size.heritage),
		bindingAccumulator: newBindingAccumulator(),
	}

	for _, input := range files {
		ir := input.Normalized()
		ir.FilePath = cleanPath(ir.FilePath)
		w.files = append(w.files, ir)
		w.fileSet[ir.FilePath] = struct{}{}
		w.fileLanguages[ir.FilePath] = ir.Language
		if ir.FileHash != "" {
			w.fileHashes[ir.FilePath] = ir.FileHash
		}
		if ir.ModuleScope != "" {
			w.moduleScopeByFile[ir.FilePath] = ir.ModuleScope
		}
		for _, scope := range ir.Scopes {
			scope.FilePath = cleanPath(scope.FilePath)
			w.scopes[scope.ID] = scope
			if scope.FileHash != "" {
				w.fileHashes[scope.FilePath] = scope.FileHash
			}
			if _, ok := w.scopeBindings[scope.ID]; !ok {
				w.scopeBindings[scope.ID] = make(map[string][]bindingRef)
			}
			if _, ok := w.typeBindings[scope.ID]; !ok {
				w.typeBindings[scope.ID] = make(map[string][]scopeir.TypeRef)
			}
			for _, binding := range scope.TypeBindings {
				w.typeBindings[scope.ID][binding.Name] = append(
					w.typeBindings[scope.ID][binding.Name],
					binding.Type,
				)
			}
			if scope.Kind == scopeir.ScopeModule {
				entries := make([]bindingEntry, 0, len(scope.TypeBindings))
				for _, binding := range scope.TypeBindings {
					entries = append(entries, bindingEntry{Name: binding.Name, Type: binding.Type})
				}
				if err := w.bindingAccumulator.appendFile(ir.FilePath, entries); err != nil {
					return nil, err
				}
			}
			for _, defID := range scope.OwnedDefIDs {
				w.scopeByDef[defID] = scope.ID
			}
		}
	}

	for _, ir := range w.files {
		for _, def := range ir.Definitions {
			def.FilePath = cleanPath(def.FilePath)
			ref := defRef{Fact: def, GraphID: graphIDForDef(def)}
			w.defsByID[def.ID] = ref
			w.defsByFile[def.FilePath] = append(w.defsByFile[def.FilePath], ref)
			lookupNames := definitionLookupNameSetFor(def)
			for index := 0; index < lookupNames.count; index++ {
				name := lookupNames.values[index]
				w.defsByName[name] = append(w.defsByName[name], ref)
			}
			if def.OwnerID != "" {
				if _, ok := w.ownerMembers[def.OwnerID]; !ok {
					w.ownerMembers[def.OwnerID] = make(map[string][]defRef)
				}
				w.ownerMembers[def.OwnerID][def.Name] = append(w.ownerMembers[def.OwnerID][def.Name], ref)
			}
		}
	}

	for _, scope := range w.scopes {
		for _, binding := range scope.Bindings {
			ref, ok := w.defsByID[binding.DefID]
			if !ok {
				continue
			}
			w.scopeBindings[scope.ID][binding.Name] = append(
				w.scopeBindings[scope.ID][binding.Name],
				bindingRef{Def: ref, Origin: binding.Origin},
			)
		}
	}

	w.resolveImports()
	w.resolveHeritage()
	w.enrichCallReturnTypeBindings()
	w.sort()
	return w, nil
}

type workspaceSize struct {
	files       int
	scopes      int
	definitions int
	imports     int
	heritage    int
	ownedDefs   int
}

func measureWorkspace(files []scopeir.ScopeIR) workspaceSize {
	size := workspaceSize{files: len(files)}
	for _, file := range files {
		size.scopes += len(file.Scopes)
		size.definitions += len(file.Definitions)
		size.imports += len(file.Imports)
		size.heritage += len(file.Heritage)
		for _, scope := range file.Scopes {
			size.ownedDefs += len(scope.OwnedDefIDs)
		}
	}
	return size
}

func (w *workspace) enrichCallReturnTypeBindings() {
	for _, ir := range w.files {
		for _, def := range ir.Definitions {
			if def.DeclaredType != "" || !isAnyLabel(def.Label, []scopeir.NodeLabel{scopeir.NodeVariable, scopeir.NodeConst}) {
				continue
			}
			call, ok := callWithinRange(ir.Calls, def.Range)
			if !ok {
				continue
			}
			returnType, ok := w.returnTypeForCall(call)
			if !ok {
				continue
			}
			scopeID := w.scopeByDef[def.ID]
			if scopeID == "" {
				continue
			}
			ref := scopeir.TypeRef{
				RawName:         returnType,
				DeclaredAtScope: scopeID,
				Source:          scopeir.TypeSourceCallReturn,
			}
			w.typeBindings[scopeID][def.Name] = append(w.typeBindings[scopeID][def.Name], ref)
			scope := w.scopes[scopeID]
			scope.TypeBindings = append(scope.TypeBindings, scopeir.TypeBindingFact{Name: def.Name, Type: ref})
			w.scopes[scopeID] = scope
		}
	}
}

func (w *workspace) returnTypeForCall(call scopeir.CallSiteFact) (string, bool) {
	target, ok := w.resolveCallTargetForTypeBinding(call)
	if !ok {
		return "", false
	}
	if call.CallForm == scopeir.CallConstructor && target.Fact.Name != "" {
		return target.Fact.Name, true
	}
	if target.Fact.ReturnType == "" {
		return "", false
	}
	return target.Fact.ReturnType, true
}

func (w *workspace) resolveCallTargetForTypeBinding(call scopeir.CallSiteFact) (defRef, bool) {
	switch call.CallForm {
	case scopeir.CallConstructor:
		if target, ok := w.resolveName(call.Name, call.InScope, dispatchOwnerLabels()); ok {
			return target, true
		}
		return w.resolveSameFileName(call.FilePath, call.Name, dispatchOwnerLabels())
	case scopeir.CallMember:
		if target, ok := w.resolveMember(call.Name, call.ExplicitReceiver, call.InScope, callableLabels()); ok {
			return target, true
		}
		return w.resolveImportedMember(call.ExplicitReceiver, call.Name, call.InScope, callableLabels())
	default:
		if target, ok := w.resolveName(call.Name, call.InScope, callableLabels()); ok {
			return target, true
		}
		return w.resolveSameFileName(call.FilePath, call.Name, callableLabels())
	}
}

func (w *workspace) resolveImports() {
	for _, ir := range w.files {
		sourceScope := w.moduleScopeByFile[ir.FilePath]
		for _, item := range ir.Imports {
			item.FilePath = cleanPath(item.FilePath)
			resolved := resolvedImport{Fact: item, SourceScope: sourceScope, LinkStatus: "unresolved"}
			if item.TargetRaw != nil {
				if targetFiles := w.resolveImportFiles(ir.Language, item.FilePath, *item.TargetRaw); len(targetFiles) > 0 {
					resolved.TargetFiles = targetFiles
					resolved.TargetFile = targetFiles[0]
					resolved.LinkStatus = ""
					if targetDef, ok := w.resolveImportedDef(resolved.TargetFile, item); ok {
						resolved.TargetDef = &targetDef
					}
				}
			}
			w.imports = append(w.imports, resolved)
			importIndex := len(w.imports) - 1
			if item.FilePath != "" && item.LocalName != "" && resolved.LinkStatus != "unresolved" {
				key := importReceiverKey{filePath: item.FilePath, localName: item.LocalName}
				w.importsByReceiver[key] = append(w.importsByReceiver[key], importIndex)
			}
			if sourceScope == "" || item.LocalName == "" || resolved.TargetDef == nil {
				continue
			}
			if _, ok := w.scopeBindings[sourceScope]; !ok {
				w.scopeBindings[sourceScope] = make(map[string][]bindingRef)
			}
			w.scopeBindings[sourceScope][item.LocalName] = append(
				w.scopeBindings[sourceScope][item.LocalName],
				bindingRef{Def: *resolved.TargetDef, Origin: importBindingOrigin(item.Kind), Via: &w.imports[importIndex]},
			)
		}
	}
	w.synthesizeWildcardImportBindings()
}

func (w *workspace) resolveHeritage() {
	for _, ir := range w.files {
		for _, item := range ir.Heritage {
			owner, ok := w.ownerForScope(item.InScope, dispatchOwnerLabels())
			if !ok {
				continue
			}
			targetLabels := dispatchOwnerLabels()
			if item.Kind == scopeir.HeritageImplements {
				targetLabels = []scopeir.NodeLabel{scopeir.NodeInterface, scopeir.NodeTrait}
			}
			target, ok := w.resolveName(item.Name, item.InScope, targetLabels)
			if !ok || target.Fact.ID == owner.Fact.ID {
				continue
			}
			w.heritage = append(w.heritage, heritageResolution{
				Fact:   item,
				Owner:  owner,
				Target: target,
			})
		}
	}
}

func (w *workspace) resolveImportFiles(language scanner.Language, sourceFile string, targetRaw string) []string {
	var ok bool
	targetRaw, ok = preprocessImportTarget(targetRaw)
	if !ok {
		return nil
	}
	if language == scanner.Go && !strings.HasPrefix(strings.TrimSpace(targetRaw), ".") {
		if files := w.resolveGoPackageImportFiles(targetRaw); len(files) > 0 {
			return files
		}
	}
	if files := w.resolveLanguageImportFiles(language, sourceFile, targetRaw); len(files) > 0 {
		return files
	}
	if targetFile, ok := w.resolveImportFile(sourceFile, targetRaw); ok {
		return []string{targetFile}
	}
	return nil
}

func (w *workspace) resolveImportFile(sourceFile string, targetRaw string) (string, bool) {
	targetRaw = strings.TrimSpace(targetRaw)
	if targetRaw == "" {
		return "", false
	}

	var base string
	if strings.HasPrefix(targetRaw, ".") {
		base = cleanPath(filepath.Join(path.Dir(cleanPath(sourceFile)), targetRaw))
	} else {
		base = cleanPath(targetRaw)
	}
	candidates := []string{
		base,
		base + ".ts",
		base + ".tsx",
		base + ".js",
		base + ".jsx",
		path.Join(base, "index.ts"),
		path.Join(base, "index.tsx"),
		path.Join(base, "index.js"),
		path.Join(base, "index.jsx"),
	}
	for _, candidate := range candidates {
		candidate = cleanPath(candidate)
		if _, ok := w.fileSet[candidate]; ok {
			return candidate, true
		}
	}
	return "", false
}

func (w *workspace) resolveGoPackageImportFiles(targetRaw string) []string {
	targetRaw = cleanPath(strings.TrimSpace(targetRaw))
	if targetRaw == "" {
		return nil
	}
	parts := strings.Split(targetRaw, "/")
	for index := 0; index < len(parts); index++ {
		suffix := path.Join(parts[index:]...)
		if suffix == "" || !looksLikeLocalGoPackageSuffix(suffix) {
			continue
		}
		files := w.goFilesInDir(suffix)
		if len(files) > 0 {
			return files
		}
	}
	return nil
}

func looksLikeLocalGoPackageSuffix(suffix string) bool {
	switch {
	case suffix == "cmd", strings.HasPrefix(suffix, "cmd/"):
		return true
	case suffix == "internal", strings.HasPrefix(suffix, "internal/"):
		return true
	case suffix == "avmatrix-launcher", strings.HasPrefix(suffix, "avmatrix-launcher/"):
		return true
	default:
		return false
	}
}

func (w *workspace) goFilesInDir(dir string) []string {
	files := make([]string, 0)
	for _, ir := range w.files {
		if ir.Language != scanner.Go {
			continue
		}
		if strings.HasSuffix(path.Base(ir.FilePath), "_test.go") {
			continue
		}
		if path.Dir(ir.FilePath) == dir {
			files = append(files, ir.FilePath)
		}
	}
	sort.Strings(files)
	return files
}

func (w *workspace) resolveImportedDef(targetFile string, item scopeir.ImportFact) (defRef, bool) {
	names := []string{item.ImportedName, item.LocalName}
	if item.ImportedName == "default" {
		names = []string{item.LocalName}
	}
	defs := w.defsByFile[targetFile]
	for _, name := range uniqueStrings(names) {
		if name == "" {
			continue
		}
		for _, def := range defs {
			if def.Fact.Name == name || def.Fact.QualifiedName == name || simpleName(def.Fact.QualifiedName) == name {
				return def, true
			}
		}
	}
	if item.ImportedName == "default" {
		for _, def := range defs {
			if isAnyLabel(def.Fact.Label, []scopeir.NodeLabel{scopeir.NodeClass, scopeir.NodeFunction, scopeir.NodeInterface}) {
				return def, true
			}
		}
	}
	return defRef{}, false
}

func (w *workspace) resolveName(name string, startScope string, labels []scopeir.NodeLabel) (defRef, bool) {
	if name == "" {
		return defRef{}, false
	}
	for scopeID := startScope; scopeID != ""; scopeID = w.parentScope(scopeID) {
		if bindings := w.scopeBindings[scopeID][name]; len(bindings) > 0 {
			filtered := filterBindingsByLabel(bindings, labels)
			if len(filtered) == 1 {
				return filtered[0].Def, true
			}
			if len(filtered) > 1 {
				return defRef{}, false
			}
		}
	}
	return w.resolveGlobalName(name, labels)
}

func (w *workspace) resolveGlobalName(name string, labels []scopeir.NodeLabel) (defRef, bool) {
	var candidates uniqueDefAccumulator
	forEachGlobalLookupName(name, func(lookup string) bool {
		for _, def := range w.defsByName[lookup] {
			if isAnyLabel(def.Fact.Label, labels) {
				if !candidates.add(def) {
					return false
				}
			}
		}
		return true
	})
	return candidates.result()
}

func (w *workspace) resolveGlobalCallName(name string, labels []scopeir.NodeLabel, arity *int) (defRef, bool) {
	var candidates uniqueDefAccumulator
	forEachGlobalLookupName(name, func(lookup string) bool {
		for _, def := range w.defsByName[lookup] {
			if isAnyLabel(def.Fact.Label, labels) && parameterCompatible(def.Fact, arity) {
				if !candidates.add(def) {
					return false
				}
			}
		}
		return true
	})
	return candidates.result()
}

func (w *workspace) resolveSameFileName(filePath string, name string, labels []scopeir.NodeLabel) (defRef, bool) {
	filePath = cleanPath(filePath)
	name = strings.TrimSpace(name)
	if filePath == "" || name == "" {
		return defRef{}, false
	}
	var candidates uniqueDefAccumulator
	for _, def := range w.defsByFile[filePath] {
		if !isAnyLabel(def.Fact.Label, labels) {
			continue
		}
		if definitionLookupNameMatches(def.Fact, name) {
			if !candidates.add(def) {
				return defRef{}, false
			}
		}
	}
	return candidates.result()
}

func (w *workspace) resolveGoSamePackageFunction(filePath string, name string, _ *int) (defRef, bool) {
	filePath = cleanPath(filePath)
	name = strings.TrimSpace(name)
	if filePath == "" || name == "" || w.fileLanguages[filePath] != scanner.Go {
		return defRef{}, false
	}
	dir := path.Dir(filePath)
	var candidates uniqueDefAccumulator
	for candidateFile, defs := range w.defsByFile {
		if w.fileLanguages[candidateFile] != scanner.Go || path.Dir(candidateFile) != dir {
			continue
		}
		for _, def := range defs {
			if def.Fact.Label != scopeir.NodeFunction || !definitionLookupNameMatches(def.Fact, name) {
				continue
			}
			if !candidates.add(def) {
				return defRef{}, false
			}
		}
	}
	return candidates.result()
}

func (w *workspace) resolveMember(name string, receiver string, startScope string, labels []scopeir.NodeLabel) (defRef, bool) {
	receiverType, ok := w.resolveReceiverType(receiver, startScope)
	if !ok {
		return defRef{}, false
	}
	owner, ok := w.resolveName(receiverType, startScope, dispatchOwnerLabels())
	if !ok {
		return defRef{}, false
	}
	if member, ok := w.resolveOwnedMember(owner.Fact.ID, name, labels); ok {
		return member, true
	}
	for _, ancestor := range w.ancestorsOf(owner.Fact.ID) {
		if member, ok := w.resolveOwnedMember(ancestor.Fact.ID, name, labels); ok {
			return member, true
		}
	}
	return defRef{}, false
}

func (w *workspace) resolveImportedMember(receiver string, name string, startScope string, labels []scopeir.NodeLabel) (defRef, bool) {
	receiver = strings.TrimSpace(receiver)
	name = strings.TrimSpace(name)
	if receiver == "" || name == "" {
		return defRef{}, false
	}
	sourceFile := w.scopeFilePath(startScope)
	if sourceFile == "" {
		return defRef{}, false
	}
	var candidates uniqueDefAccumulator
	key := importReceiverKey{filePath: sourceFile, localName: receiver}
	for _, importIndex := range w.importsByReceiver[key] {
		item := w.imports[importIndex]
		if item.LinkStatus == "unresolved" {
			continue
		}
		for _, targetFile := range item.TargetFiles {
			for _, def := range w.defsByFile[targetFile] {
				if !isAnyLabel(def.Fact.Label, labels) {
					continue
				}
				if definitionLookupNameMatches(def.Fact, name) {
					if !candidates.add(def) {
						return defRef{}, false
					}
				}
			}
		}
	}
	return candidates.result()
}

func (w *workspace) resolveReceiverType(receiver string, startScope string) (string, bool) {
	receiver = strings.TrimSpace(receiver)
	if receiver == "" || receiver == "this" || receiver == "self" {
		if typeName, ok := w.lookupTypeBinding(firstNonEmpty(receiver, "this"), startScope); ok {
			return baseTypeName(typeName.RawName), true
		}
		if owner, ok := w.ownerForScope(startScope, dispatchOwnerLabels()); ok {
			return owner.Fact.Name, true
		}
		return "", false
	}

	parts := strings.Split(receiver, ".")
	currentType, ok := w.lookupTypeBinding(parts[0], startScope)
	if !ok {
		return "", false
	}
	typeName := baseTypeName(currentType.RawName)
	for _, part := range parts[1:] {
		owner, ok := w.resolveName(typeName, startScope, dispatchOwnerLabels())
		if !ok {
			return "", false
		}
		member, ok := w.resolveOwnedMember(owner.Fact.ID, part, []scopeir.NodeLabel{scopeir.NodeProperty, scopeir.NodeVariable})
		if !ok || member.Fact.DeclaredType == "" {
			return "", false
		}
		typeName = baseTypeName(member.Fact.DeclaredType)
	}
	return typeName, true
}

func (w *workspace) lookupTypeBinding(name string, startScope string) (scopeir.TypeRef, bool) {
	for scopeID := startScope; scopeID != ""; scopeID = w.parentScope(scopeID) {
		bindings := w.typeBindings[scopeID][name]
		if len(bindings) > 0 {
			return bindings[len(bindings)-1], true
		}
	}
	return scopeir.TypeRef{}, false
}

func (w *workspace) resolveOwnedMember(ownerDefID string, name string, labels []scopeir.NodeLabel) (defRef, bool) {
	members := w.ownerMembers[ownerDefID][name]
	filtered := make([]defRef, 0, len(members))
	for _, member := range members {
		if isAnyLabel(member.Fact.Label, labels) {
			filtered = append(filtered, member)
		}
	}
	if len(filtered) != 1 {
		return defRef{}, false
	}
	return filtered[0], true
}

func (w *workspace) ownerForScope(startScope string, labels []scopeir.NodeLabel) (defRef, bool) {
	for scopeID := startScope; scopeID != ""; scopeID = w.parentScope(scopeID) {
		scope := w.scopes[scopeID]
		for _, defID := range scope.OwnedDefIDs {
			def, ok := w.defsByID[defID]
			if ok && isAnyLabel(def.Fact.Label, labels) {
				return def, true
			}
		}
	}
	return defRef{}, false
}

func (w *workspace) callerForScope(startScope string) (defRef, bool) {
	var fallback *defRef
	for scopeID := startScope; scopeID != ""; scopeID = w.parentScope(scopeID) {
		scope := w.scopes[scopeID]
		for _, defID := range scope.OwnedDefIDs {
			def, ok := w.defsByID[defID]
			if !ok {
				continue
			}
			if fallback == nil {
				copy := def
				fallback = &copy
			}
			if isAnyLabel(def.Fact.Label, []scopeir.NodeLabel{scopeir.NodeFunction, scopeir.NodeMethod, scopeir.NodeConstructor}) {
				return def, true
			}
		}
	}
	if fallback != nil {
		return *fallback, true
	}
	return defRef{}, false
}

func (w *workspace) scopeFilePath(startScope string) string {
	for scopeID := startScope; scopeID != ""; scopeID = w.parentScope(scopeID) {
		if scope := w.scopes[scopeID]; scope.FilePath != "" {
			return cleanPath(scope.FilePath)
		}
	}
	return ""
}

func (w *workspace) parentScope(scopeID string) string {
	scope, ok := w.scopes[scopeID]
	if !ok || scope.Parent == nil {
		return ""
	}
	return *scope.Parent
}

func (w *workspace) ancestorsOf(ownerDefID string) []defRef {
	out := []defRef{}
	seen := map[string]struct{}{ownerDefID: {}}
	queue := []string{ownerDefID}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		for _, heritage := range w.heritage {
			if heritage.Owner.Fact.ID != current {
				continue
			}
			if _, ok := seen[heritage.Target.Fact.ID]; ok {
				continue
			}
			seen[heritage.Target.Fact.ID] = struct{}{}
			out = append(out, heritage.Target)
			queue = append(queue, heritage.Target.Fact.ID)
		}
	}
	return out
}

func (w *workspace) sort() {
	sort.Slice(w.files, func(i, j int) bool { return w.files[i].FilePath < w.files[j].FilePath })
	for filePath := range w.defsByFile {
		sort.Slice(w.defsByFile[filePath], func(i, j int) bool {
			return w.defsByFile[filePath][i].GraphID < w.defsByFile[filePath][j].GraphID
		})
	}
	for name := range w.defsByName {
		sort.Slice(w.defsByName[name], func(i, j int) bool {
			return w.defsByName[name][i].GraphID < w.defsByName[name][j].GraphID
		})
	}
}

func graphIDForDef(def scopeir.DefinitionFact) string {
	name := def.QualifiedName
	if name == "" {
		name = def.Name
	}
	arity := ""
	if def.ParameterCount != nil {
		arity = "#" + intString(*def.ParameterCount)
	}
	return graph.GenerateID(string(def.Label), cleanPath(def.FilePath)+":"+name+arity)
}

func definitionLookupNames(def scopeir.DefinitionFact) []string {
	names := definitionLookupNameSetFor(def)
	out := make([]string, 0, names.count)
	for index := 0; index < names.count; index++ {
		out = append(out, names.values[index])
	}
	return out
}

type definitionLookupNameSet struct {
	values [3]string
	count  int
}

func definitionLookupNameSetFor(def scopeir.DefinitionFact) definitionLookupNameSet {
	var names definitionLookupNameSet
	names.add(def.Name)
	names.add(def.QualifiedName)
	names.add(simpleName(def.QualifiedName))
	return names
}

func (names *definitionLookupNameSet) add(value string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return
	}
	for index := 0; index < names.count; index++ {
		if names.values[index] == value {
			return
		}
	}
	names.values[names.count] = value
	names.count++
}

func (names definitionLookupNameSet) contains(name string) bool {
	name = strings.TrimSpace(name)
	if name == "" {
		return false
	}
	for index := 0; index < names.count; index++ {
		if names.values[index] == name {
			return true
		}
	}
	return false
}

func definitionLookupNameMatches(def scopeir.DefinitionFact, name string) bool {
	return definitionLookupNameSetFor(def).contains(name)
}

func forEachGlobalLookupName(name string, visit func(string) bool) {
	primary := strings.TrimSpace(name)
	if primary != "" && !visit(primary) {
		return
	}
	alternate := baseTypeName(name)
	if alternate != "" && alternate != primary {
		visit(alternate)
	}
}

func callWithinRange(calls []scopeir.CallSiteFact, target scopeir.Range) (scopeir.CallSiteFact, bool) {
	var found *scopeir.CallSiteFact
	for _, call := range calls {
		if !rangeContains(target, call.Range) {
			continue
		}
		if found != nil {
			return scopeir.CallSiteFact{}, false
		}
		copy := call
		found = &copy
	}
	if found == nil {
		return scopeir.CallSiteFact{}, false
	}
	return *found, true
}

func rangeContains(outer scopeir.Range, inner scopeir.Range) bool {
	if outer.StartLine == 0 || inner.StartLine == 0 {
		return false
	}
	if inner.StartLine < outer.StartLine || inner.EndLine > outer.EndLine {
		return false
	}
	if inner.StartLine == outer.StartLine && inner.StartCol < outer.StartCol {
		return false
	}
	if inner.EndLine == outer.EndLine && outer.EndCol > 0 && inner.EndCol > outer.EndCol {
		return false
	}
	return true
}

func cleanPath(value string) string {
	if value == "" {
		return ""
	}
	return filepath.ToSlash(filepath.Clean(strings.ReplaceAll(value, "\\", "/")))
}

func simpleName(value string) string {
	if value == "" {
		return ""
	}
	index := strings.LastIndex(value, ".")
	if index == -1 {
		return value
	}
	return value[index+1:]
}

func baseTypeName(value string) string {
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(value, "readonly ")
	for {
		value = strings.TrimSpace(value)
		switch {
		case strings.HasPrefix(value, "*"), strings.HasPrefix(value, "&"):
			value = value[1:]
		case strings.HasPrefix(value, "[]"):
			value = value[2:]
		case strings.HasPrefix(value, "..."):
			value = value[3:]
		default:
			goto trimmed
		}
	}
trimmed:
	if index := strings.IndexAny(value, "<|&[]("); index >= 0 {
		value = value[:index]
	}
	return simpleName(strings.TrimSpace(value))
}

func filterBindingsByLabel(bindings []bindingRef, labels []scopeir.NodeLabel) []bindingRef {
	out := []bindingRef{}
	for _, binding := range bindings {
		if isAnyLabel(binding.Def.Fact.Label, labels) {
			out = append(out, binding)
		}
	}
	return out
}

func isAnyLabel(label scopeir.NodeLabel, accepted []scopeir.NodeLabel) bool {
	for _, candidate := range accepted {
		if label == candidate {
			return true
		}
	}
	return false
}

func dispatchOwnerLabels() []scopeir.NodeLabel {
	return []scopeir.NodeLabel{
		scopeir.NodeClass,
		scopeir.NodeInterface,
		scopeir.NodeStruct,
		scopeir.NodeTrait,
		scopeir.NodeRecord,
	}
}

func callableLabels() []scopeir.NodeLabel {
	return []scopeir.NodeLabel{scopeir.NodeFunction, scopeir.NodeMethod, scopeir.NodeConstructor}
}

func typeLabels() []scopeir.NodeLabel {
	return []scopeir.NodeLabel{
		scopeir.NodeClass,
		scopeir.NodeInterface,
		scopeir.NodeStruct,
		scopeir.NodeTrait,
		scopeir.NodeRecord,
		scopeir.NodeTypeAlias,
		scopeir.NodeEnum,
	}
}

func propertyLabels() []scopeir.NodeLabel {
	return []scopeir.NodeLabel{scopeir.NodeProperty, scopeir.NodeVariable, scopeir.NodeConst, scopeir.NodeStatic}
}

func importBindingOrigin(kind scopeir.ImportKind) scopeir.BindingOrigin {
	switch kind {
	case scopeir.ImportReexport:
		return scopeir.BindingReexport
	case scopeir.ImportNamespace:
		return scopeir.BindingNamespace
	case scopeir.ImportWildcard, scopeir.ImportWildcardExpanded:
		return scopeir.BindingWildcard
	default:
		return scopeir.BindingImport
	}
}

func uniqueStrings(values []string) []string {
	switch len(values) {
	case 0:
		return nil
	case 1:
		value := strings.TrimSpace(values[0])
		if value == "" {
			return nil
		}
		return []string{value}
	case 2:
		first := strings.TrimSpace(values[0])
		second := strings.TrimSpace(values[1])
		switch {
		case first == "" && second == "":
			return nil
		case first == "":
			return []string{second}
		case second == "" || second == first:
			return []string{first}
		default:
			return []string{first, second}
		}
	}
	out := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func stringInSlice(value string, values []string) bool {
	for _, candidate := range values {
		if value == candidate {
			return true
		}
	}
	return false
}

func parameterCompatible(def scopeir.DefinitionFact, arity *int) bool {
	if arity == nil || def.ParameterCount == nil {
		return true
	}
	minimum := *def.ParameterCount
	if def.RequiredParameterCount != nil {
		minimum = *def.RequiredParameterCount
	}
	return *arity >= minimum && *arity <= *def.ParameterCount
}

func uniqueDefs(values []defRef) []defRef {
	switch len(values) {
	case 0:
		return nil
	case 1:
		return values
	case 2:
		if values[0].Fact.ID == values[1].Fact.ID {
			return values[:1]
		}
		return values
	}
	out := make([]defRef, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if _, ok := seen[value.Fact.ID]; ok {
			continue
		}
		seen[value.Fact.ID] = struct{}{}
		out = append(out, value)
	}
	return out
}

type uniqueDefAccumulator struct {
	value defRef
	id    string
	count int
}

func (acc *uniqueDefAccumulator) add(value defRef) bool {
	if acc.count == 0 {
		acc.value = value
		acc.id = value.Fact.ID
		acc.count = 1
		return true
	}
	if value.Fact.ID == acc.id {
		return true
	}
	acc.count = 2
	return false
}

func (acc uniqueDefAccumulator) result() (defRef, bool) {
	if acc.count != 1 {
		return defRef{}, false
	}
	return acc.value, true
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func intString(value int) string {
	if value == 0 {
		return "0"
	}
	var buffer [20]byte
	index := len(buffer)
	negative := value < 0
	if negative {
		value = -value
	}
	for value > 0 {
		index--
		buffer[index] = byte('0' + value%10)
		value /= 10
	}
	if negative {
		index--
		buffer[index] = '-'
	}
	return string(buffer[index:])
}
