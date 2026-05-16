package resolution

import (
	"errors"

	"github.com/tamnguyendinh/avmatrix-go/internal/graph"
	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

func Resolve(files []scopeir.ScopeIR, options Options) (result Result, err error) {
	return ResolveInto(nil, files, options)
}

func ResolveInto(baseGraph *graph.Graph, files []scopeir.ScopeIR, options Options) (result Result, err error) {
	binding, err := BuildCrossFileBinding(files, options)
	if err != nil {
		return Result{}, err
	}
	return ResolveBoundInto(baseGraph, binding, options)
}

func BuildCrossFileBinding(files []scopeir.ScopeIR, options Options) (result BindingResult, err error) {
	w, err := buildWorkspace(files)
	if err != nil {
		return BindingResult{}, err
	}
	defer func() {
		if err != nil {
			w.bindingAccumulator.dispose()
		}
	}()
	metrics := Metrics{DefinitionsIndexed: len(w.defsByID), ImportsResolved: countResolvedImports(w.imports)}
	applyCrossFileCompatibilityMetrics(options, &metrics)
	metrics.BindingAccumulatorFiles = w.bindingAccumulator.fileCount()
	metrics.BindingAccumulatorEntries = w.bindingAccumulator.total()
	if err := w.bindingAccumulator.finalize(); err != nil {
		return BindingResult{Metrics: metrics}, err
	}
	metrics.BindingAccumulatorFinalized = w.bindingAccumulator.finalized
	return BindingResult{workspace: w, Metrics: metrics}, nil
}

func ResolveBoundInto(baseGraph *graph.Graph, binding BindingResult, options Options) (result Result, err error) {
	w := binding.workspace
	if w == nil {
		return Result{}, errors.New("resolution binding result is empty")
	}
	g := baseGraph
	if g == nil {
		g = graph.New()
	}
	metrics := binding.Metrics
	defer func() {
		w.bindingAccumulator.dispose()
		result.Metrics.BindingAccumulatorDisposed = w.bindingAccumulator.disposed
	}()
	e := newEmitter(g, &metrics)

	emitDefinitionNodes(w, e)
	emitImportEdges(w, e)

	emitInherits := !options.DisableScopeInheritsCompatibility
	for _, item := range w.heritage {
		emitHeritageCompatibilityEdges(e, item, emitInherits)
		metrics.ResolvedInheritance++
	}

	for _, ir := range w.files {
		for _, call := range ir.Calls {
			resolveCall(w, e, call)
		}
		for _, access := range ir.Accesses {
			resolveAccess(w, e, access)
		}
		for _, annotation := range ir.TypeAnnotations {
			resolveTypeAnnotation(w, e, annotation)
		}
	}
	emitMethodDispatchEdges(w, e)

	metrics.GraphNodesEmitted = len(g.Nodes)
	metrics.GraphRelationshipsEmitted = len(g.Relationships)
	return Result{Graph: g, ReferenceIndex: e.referenceIndex, Metrics: metrics}, nil
}

func applyCrossFileCompatibilityMetrics(options Options, metrics *Metrics) {
	metrics.CrossFileFilesReprocessed = 0
	metrics.CrossFileSkipped = true
	if options.SkipCompatibilityCrossFile {
		metrics.CrossFileSkipReason = "disabled-by-pipeline-option"
		return
	}
	metrics.CrossFileSkipReason = "covered-by-scopeir-single-pass-resolution"
}

func resolveCall(w *workspace, e *emitter, call scopeir.CallSiteFact) {
	source, ok := w.callerForScope(call.InScope)
	if !ok {
		source, ok = callerFileRef(call.FilePath)
		if !ok {
			e.metrics.UnresolvedReferences++
			return
		}
	}
	var target defRef
	confidence := 1.0
	switch call.CallForm {
	case scopeir.CallConstructor:
		target, ok = w.resolveName(call.Name, call.InScope, dispatchOwnerLabels())
		if !ok {
			target, ok = w.resolveSameFileName(call.FilePath, call.Name, dispatchOwnerLabels())
			if ok {
				confidence = 0.95
			}
		}
		if !ok {
			target, ok = w.resolveGlobalCallName(call.Name, dispatchOwnerLabels(), call.Arity)
			if ok {
				confidence = 0.5
			}
		}
	case scopeir.CallMember:
		target, ok = w.resolveMember(call.Name, call.ExplicitReceiver, call.InScope, callableLabels())
		if !ok {
			target, ok = w.resolveImportedMember(call.ExplicitReceiver, call.Name, call.InScope, callableLabels())
			if ok {
				confidence = 0.9
			}
		}
		if !ok && call.ExplicitReceiver == "" {
			target, ok = w.resolveGlobalCallName(call.Name, callableLabels(), call.Arity)
			if ok {
				confidence = 0.5
			}
		}
	default:
		target, ok = w.resolveName(call.Name, call.InScope, callableLabels())
		if !ok {
			target, ok = w.resolveSameFileName(call.FilePath, call.Name, callableLabels())
			if ok {
				confidence = 0.95
			}
		}
		if !ok {
			target, ok = w.resolveGoSamePackageFunction(call.FilePath, call.Name, call.Arity)
			if ok {
				confidence = 0.95
			}
		}
		if !ok {
			target, ok = w.resolveGlobalCallName(call.Name, callableLabels(), call.Arity)
			if ok {
				confidence = 0.5
			}
		}
	}
	if !ok {
		e.metrics.UnresolvedReferences++
		return
	}
	e.emitReference(source, target, Reference{
		FromScope:  call.InScope,
		ToDefID:    target.Fact.ID,
		FileHash:   call.FileHash,
		Range:      call.Range,
		Kind:       ReferenceCall,
		Confidence: confidence,
		Evidence: []graph.Evidence{{
			Kind:   callEvidenceKind(call.CallForm),
			Weight: 1,
			Note:   call.Name,
		}},
	})
	e.metrics.ResolvedReferences++
	e.metrics.ResolvedCalls++
}

func callerFileRef(filePath string) (defRef, bool) {
	filePath = cleanPath(filePath)
	if filePath == "" {
		return defRef{}, false
	}
	return defRef{
		Fact: scopeir.DefinitionFact{
			ID:       graph.GenerateID(string(scopeir.NodeFile), filePath),
			FilePath: filePath,
			Name:     filePath,
			Label:    scopeir.NodeFile,
		},
		GraphID: graph.GenerateID(string(scopeir.NodeFile), filePath),
	}, true
}

func resolveAccess(w *workspace, e *emitter, access scopeir.AccessFact) {
	source, ok := w.callerForScope(access.InScope)
	if !ok {
		e.metrics.UnresolvedReferences++
		return
	}
	target, ok := w.resolveMember(access.Name, access.ExplicitReceiver, access.InScope, propertyLabels())
	if !ok {
		e.metrics.UnresolvedReferences++
		return
	}
	kind := ReferenceRead
	if access.Kind == scopeir.AccessWrite {
		kind = ReferenceWrite
	}
	e.emitReference(source, target, Reference{
		FromScope:  access.InScope,
		ToDefID:    target.Fact.ID,
		FileHash:   access.FileHash,
		Range:      access.Range,
		Kind:       kind,
		Confidence: 1,
		Evidence: []graph.Evidence{{
			Kind:   "type-binding",
			Weight: 1,
			Note:   access.ExplicitReceiver + "." + access.Name,
		}},
	})
	e.metrics.ResolvedReferences++
	e.metrics.ResolvedAccesses++
}

func resolveTypeAnnotation(w *workspace, e *emitter, annotation scopeir.TypeAnnotationFact) {
	targetName := baseTypeName(annotation.Type.RawName)
	if targetName == "" || isBuiltinType(targetName) {
		return
	}
	source, ok := w.callerForScope(annotation.InScope)
	if !ok {
		e.metrics.UnresolvedReferences++
		return
	}
	target, ok := w.resolveName(targetName, annotation.InScope, typeLabels())
	if !ok {
		e.metrics.UnresolvedReferences++
		return
	}
	e.emitReference(source, target, Reference{
		FromScope:  annotation.InScope,
		ToDefID:    target.Fact.ID,
		FileHash:   annotation.FileHash,
		Range:      annotation.Range,
		Kind:       ReferenceTypeReference,
		Confidence: 1,
		Evidence: []graph.Evidence{{
			Kind:   "scope-chain",
			Weight: 1,
			Note:   annotation.Type.RawName,
		}},
	})
	e.metrics.ResolvedReferences++
	e.metrics.ResolvedTypeReferences++
}

func countResolvedImports(imports []resolvedImport) int {
	count := 0
	for _, item := range imports {
		if len(item.TargetFiles) > 0 && item.LinkStatus != "unresolved" {
			count += len(item.TargetFiles)
		}
	}
	return count
}

func callEvidenceKind(form scopeir.CallForm) string {
	switch form {
	case scopeir.CallMember:
		return "type-binding"
	case scopeir.CallConstructor:
		return "kind-match"
	default:
		return "scope-chain"
	}
}

func isBuiltinType(name string) bool {
	switch name {
	case "string", "number", "boolean", "bool", "void", "undefined", "null", "any", "unknown", "never", "object":
		return true
	default:
		return false
	}
}
