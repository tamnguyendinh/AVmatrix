package resolution

import (
	"path"
	"sort"

	"github.com/tamnguyendinh/avmatrix-go/internal/frameworks"
	"github.com/tamnguyendinh/avmatrix-go/internal/graph"
	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

type emitter struct {
	graph          *graph.Graph
	referenceIndex ReferenceIndex
	edgeKeys       map[string]graph.Relationship
	metrics        *Metrics
	sourceLabel    string
}

func newEmitter(g *graph.Graph, metrics *Metrics) *emitter {
	return &emitter{
		graph:          g,
		referenceIndex: newReferenceIndex(),
		edgeKeys:       make(map[string]graph.Relationship),
		metrics:        metrics,
		sourceLabel:    "scope-resolution",
	}
}

func (e *emitter) emitNode(node graph.Node) {
	e.graph.AddNode(node)
	e.metrics.GraphNodesEmitted = len(e.graph.Nodes)
}

func (e *emitter) emitRelationship(relationship graph.Relationship) {
	key := semanticEdgeKey(relationship)
	if existing, ok := e.edgeKeys[key]; ok {
		merged := mergeRelationship(existing, relationship)
		e.graph.ReplaceRelationship(existing.ID, merged)
		e.edgeKeys[key] = merged
		e.metrics.DuplicateEdgesMerged++
		e.metrics.GraphRelationshipsEmitted = len(e.graph.Relationships)
		return
	}
	e.graph.AddRelationship(relationship)
	e.edgeKeys[key] = relationship
	e.metrics.GraphRelationshipsEmitted = len(e.graph.Relationships)
}

func (e *emitter) emitReference(source defRef, target defRef, reference Reference) {
	e.referenceIndex.add(reference)
	relType := relationshipTypeForReference(reference.Kind)
	reason := string(reference.Kind)
	if reference.Kind != ReferenceRead && reference.Kind != ReferenceWrite {
		reason = e.sourceLabel + ": " + string(reference.Kind) + " | confidence " + confidenceString(reference.Confidence)
	}
	relationship := graph.Relationship{
		ID:               "rel:" + string(relType) + ":" + source.GraphID + "->" + target.GraphID + ":" + intString(reference.Range.StartLine) + ":" + intString(reference.Range.StartCol),
		SourceID:         source.GraphID,
		TargetID:         target.GraphID,
		Type:             relType,
		Confidence:       reference.Confidence,
		Reason:           reason,
		ResolutionSource: e.sourceLabel,
		FileHash:         reference.FileHash,
		Evidence:         reference.Evidence,
	}
	if reference.Kind == ReferenceRead {
		step := 1
		relationship.Step = &step
	}
	if reference.Kind == ReferenceWrite {
		step := 2
		relationship.Step = &step
	}
	e.emitRelationship(relationship)
}

func emitDefinitionNodes(w *workspace, e *emitter) {
	for _, ir := range w.files {
		frameworkFacts := frameworkFactsByDefID(ir.Frameworks)
		fileID := graph.GenerateID("File", ir.FilePath)
		fileProperties := graph.NodeProperties{
			"name":     path.Base(ir.FilePath),
			"filePath": ir.FilePath,
			"language": string(ir.Language),
		}
		applyFrameworkHint(fileProperties, ir.FilePath)
		if existing, ok := e.graph.GetNode(fileID); ok && existing.Properties != nil {
			for key, value := range existing.Properties {
				if _, exists := fileProperties[key]; !exists {
					fileProperties[key] = value
				}
			}
		}
		e.emitNode(graph.Node{
			ID:         fileID,
			Label:      scopeir.NodeFile,
			Properties: fileProperties,
		})
		for _, def := range w.defsByFile[ir.FilePath] {
			props := graph.NodeProperties{
				"name":          def.Fact.Name,
				"filePath":      def.Fact.FilePath,
				"qualifiedName": def.Fact.QualifiedName,
				"startLine":     def.Fact.Range.StartLine,
				"endLine":       def.Fact.Range.EndLine,
				"language":      string(ir.Language),
			}
			applyFrameworkHint(props, def.Fact.FilePath)
			if fact, ok := frameworkFacts[def.Fact.ID]; ok {
				applyFrameworkFact(props, fact)
			}
			addNonEmpty(props, "returnType", def.Fact.ReturnType)
			addNonEmpty(props, "declaredType", def.Fact.DeclaredType)
			addNonEmpty(props, "visibility", def.Fact.Visibility)
			if def.Fact.ParameterCount != nil {
				props["parameterCount"] = *def.Fact.ParameterCount
			}
			if len(def.Fact.ParameterTypes) > 0 {
				props["parameterTypes"] = append([]string(nil), def.Fact.ParameterTypes...)
			}
			e.emitNode(graph.Node{ID: def.GraphID, Label: def.Fact.Label, Properties: props})
			e.emitRelationship(graph.Relationship{
				ID:         graph.GenerateID(string(graph.RelDefines), fileID+"->"+def.GraphID),
				SourceID:   fileID,
				TargetID:   def.GraphID,
				Type:       graph.RelDefines,
				Confidence: 1,
				Reason:     "",
			})
			if def.Fact.OwnerID != "" {
				owner, ok := w.defsByID[def.Fact.OwnerID]
				if !ok {
					continue
				}
				relType := graph.RelHasMethod
				if def.Fact.Label == scopeir.NodeProperty {
					relType = graph.RelHasProperty
				}
				e.emitRelationship(graph.Relationship{
					ID:         graph.GenerateID(string(relType), owner.GraphID+"->"+def.GraphID),
					SourceID:   owner.GraphID,
					TargetID:   def.GraphID,
					Type:       relType,
					Confidence: 1,
					Reason:     "",
				})
			}
		}
	}
}

func frameworkFactsByDefID(facts []scopeir.FrameworkFact) map[string]scopeir.FrameworkFact {
	out := make(map[string]scopeir.FrameworkFact, len(facts))
	for _, fact := range facts {
		if fact.DefID == "" {
			continue
		}
		current, ok := out[fact.DefID]
		if !ok || fact.EntryPointMultiplier > current.EntryPointMultiplier {
			out[fact.DefID] = fact
		}
	}
	return out
}

func applyFrameworkHint(props graph.NodeProperties, filePath string) {
	hint, ok := frameworks.DetectFromPath(filePath)
	if !ok {
		return
	}
	props["framework"] = hint.Framework
	props["frameworkReason"] = hint.Reason
	props["frameworkEntryPointMultiplier"] = hint.EntryPointMultiplier
	props["astFrameworkReason"] = hint.Reason
	props["astFrameworkMultiplier"] = hint.EntryPointMultiplier
}

func applyFrameworkFact(props graph.NodeProperties, fact scopeir.FrameworkFact) {
	if fact.Framework != "" {
		props["framework"] = fact.Framework
	}
	props["frameworkReason"] = fact.Reason
	props["frameworkEntryPointMultiplier"] = fact.EntryPointMultiplier
	props["astFrameworkReason"] = fact.Reason
	props["astFrameworkMultiplier"] = fact.EntryPointMultiplier
}

func emitImportEdges(w *workspace, e *emitter) {
	for _, item := range w.imports {
		if len(item.TargetFiles) == 0 || item.LinkStatus == "unresolved" {
			continue
		}
		sourceFileID := graph.GenerateID("File", cleanPath(item.Fact.FilePath))
		for _, targetFile := range item.TargetFiles {
			targetFileID := graph.GenerateID("File", targetFile)
			e.emitRelationship(graph.Relationship{
				ID:               graph.GenerateID(string(graph.RelImports), cleanPath(item.Fact.FilePath)+"->"+targetFile),
				SourceID:         sourceFileID,
				TargetID:         targetFileID,
				Type:             graph.RelImports,
				Confidence:       1,
				Reason:           "scope-finalize import " + string(item.Fact.Kind) + " " + item.Fact.LocalName,
				ResolutionSource: "scope-finalize",
				FileHash:         w.fileHashes[cleanPath(item.Fact.FilePath)],
				Evidence: []graph.Evidence{{
					Kind:   "import",
					Weight: 1,
					Note:   string(item.Fact.Kind) + " " + item.Fact.LocalName + " -> " + targetFile,
				}},
			})
			e.metrics.FinalizedImportsEmitted++
		}
		if item.TargetDef == nil {
			continue
		}
		e.emitRelationship(graph.Relationship{
			ID:               graph.GenerateID(string(graph.RelUses), cleanPath(item.Fact.FilePath)+"->"+item.TargetDef.Fact.ID+":import:"+item.Fact.LocalName),
			SourceID:         sourceFileID,
			TargetID:         item.TargetDef.GraphID,
			Type:             graph.RelUses,
			Confidence:       1,
			Reason:           "scope-finalize import-use " + string(item.Fact.Kind) + " " + item.Fact.LocalName,
			ResolutionSource: "scope-finalize",
			FileHash:         w.fileHashes[cleanPath(item.Fact.FilePath)],
			Evidence: []graph.Evidence{{
				Kind:   "import",
				Weight: 1,
				Note:   string(item.Fact.Kind) + " " + item.Fact.LocalName + " -> " + item.TargetDef.Fact.Name,
			}},
		})
		e.metrics.ImportUsesEmitted++
	}
}

func emitHeritageCompatibilityEdges(e *emitter, item heritageResolution, emitInherits bool) {
	relType := graph.RelExtends
	if item.Fact.Kind == scopeir.HeritageImplements ||
		item.Fact.Kind == scopeir.HeritageTraitImpl ||
		item.Fact.Kind == scopeir.HeritageInclude ||
		item.Fact.Kind == scopeir.HeritageExtend ||
		item.Fact.Kind == scopeir.HeritagePrepend {
		relType = graph.RelImplements
	}
	e.emitRelationship(graph.Relationship{
		ID:         graph.GenerateID(string(relType), item.Owner.GraphID+"->"+item.Target.GraphID),
		SourceID:   item.Owner.GraphID,
		TargetID:   item.Target.GraphID,
		Type:       relType,
		Confidence: 1,
		Reason:     string(item.Fact.Kind),
		FileHash:   item.Fact.FileHash,
	})
	if !emitInherits {
		return
	}
	e.emitReference(item.Owner, item.Target, Reference{
		FromScope:  item.Fact.InScope,
		ToDefID:    item.Target.Fact.ID,
		FileHash:   item.Fact.FileHash,
		Range:      item.Fact.Range,
		Kind:       ReferenceInherits,
		Confidence: 1,
		Evidence: []graph.Evidence{{
			Kind:   "scope-chain",
			Weight: 1,
			Note:   string(item.Fact.Kind) + " " + item.Fact.Name,
		}},
	})
}

func emitMethodDispatchEdges(w *workspace, e *emitter) {
	ownMethodsByOwner := make(map[string]map[string][]defRef)
	for ownerID, members := range w.ownerMembers {
		ownMethodsByOwner[ownerID] = make(map[string][]defRef)
		for name, bucket := range members {
			for _, member := range bucket {
				if member.Fact.Label == scopeir.NodeMethod {
					ownMethodsByOwner[ownerID][name] = append(ownMethodsByOwner[ownerID][name], member)
				}
			}
		}
	}

	for _, item := range w.heritage {
		ownMethods := ownMethodsByOwner[item.Owner.Fact.ID]
		parentMethods := ownMethodsByOwner[item.Target.Fact.ID]
		for name, ownBucket := range ownMethods {
			parentBucket := parentMethods[name]
			if len(ownBucket) != 1 || len(parentBucket) != 1 {
				continue
			}
			if item.Fact.Kind == scopeir.HeritageImplements || item.Target.Fact.Label == scopeir.NodeInterface || item.Target.Fact.Label == scopeir.NodeTrait {
				e.emitRelationship(graph.Relationship{
					ID:         graph.GenerateID(string(graph.RelMethodImplements), ownBucket[0].GraphID+"->"+parentBucket[0].GraphID),
					SourceID:   ownBucket[0].GraphID,
					TargetID:   parentBucket[0].GraphID,
					Type:       graph.RelMethodImplements,
					Confidence: methodMatchConfidence(ownBucket[0], parentBucket[0]),
					Reason:     "method implements interface contract",
				})
				e.metrics.MethodImplementsEmitted++
				continue
			}
			e.emitRelationship(graph.Relationship{
				ID:         graph.GenerateID(string(graph.RelMethodOverrides), item.Owner.GraphID+"->"+parentBucket[0].GraphID),
				SourceID:   item.Owner.GraphID,
				TargetID:   parentBucket[0].GraphID,
				Type:       graph.RelMethodOverrides,
				Confidence: methodMatchConfidence(ownBucket[0], parentBucket[0]),
				Reason:     "first definition",
			})
			e.metrics.MethodOverridesEmitted++
		}
	}
}

func semanticEdgeKey(relationship graph.Relationship) string {
	accessKind := ""
	if relationship.Type == graph.RelAccesses {
		if relationship.Step != nil && *relationship.Step == 1 {
			accessKind = ":read"
		} else if relationship.Step != nil && *relationship.Step == 2 {
			accessKind = ":write"
		} else {
			accessKind = ":unknown"
		}
	}
	callName := ""
	if relationship.Type == graph.RelCalls {
		callName = ":call:" + relationshipCallName(relationship)
	}
	return relationship.SourceID + "\x00" + relationship.TargetID + "\x00" + string(relationship.Type) + accessKind + callName
}

func relationshipCallName(relationship graph.Relationship) string {
	for _, evidence := range relationship.Evidence {
		if evidence.Note != "" {
			return evidence.Note
		}
	}
	return relationship.ID
}

func mergeRelationship(existing graph.Relationship, incoming graph.Relationship) graph.Relationship {
	merged := existing
	if incoming.Confidence >= existing.Confidence {
		merged.Reason = incoming.Reason
	}
	if incoming.Confidence > merged.Confidence {
		merged.Confidence = incoming.Confidence
	}
	if merged.Step == nil && incoming.Step != nil {
		step := *incoming.Step
		merged.Step = &step
	}
	if incoming.ResolutionSource != "" {
		merged.ResolutionSource = incoming.ResolutionSource
	}
	if incoming.FileHash != "" {
		merged.FileHash = incoming.FileHash
	}
	if len(incoming.Evidence) > 0 {
		merged.Evidence = append([]graph.Evidence(nil), incoming.Evidence...)
	}
	return merged
}

func relationshipTypeForReference(kind ReferenceKind) graph.RelationshipType {
	switch kind {
	case ReferenceCall:
		return graph.RelCalls
	case ReferenceRead, ReferenceWrite:
		return graph.RelAccesses
	case ReferenceInherits:
		return graph.RelInherits
	case ReferenceTypeReference, ReferenceImportUse:
		return graph.RelUses
	default:
		return graph.RelUses
	}
}

func methodMatchConfidence(left defRef, right defRef) float64 {
	if len(left.Fact.ParameterTypes) > 0 && len(right.Fact.ParameterTypes) > 0 {
		if stringSlicesEqual(left.Fact.ParameterTypes, right.Fact.ParameterTypes) {
			return 1
		}
		return 0.7
	}
	if left.Fact.ParameterCount != nil && right.Fact.ParameterCount != nil && *left.Fact.ParameterCount == *right.Fact.ParameterCount {
		return 1
	}
	return 0.7
}

func addNonEmpty(props graph.NodeProperties, key string, value string) {
	if value != "" {
		props[key] = value
	}
}

func stringSlicesEqual(left []string, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	leftCopy := append([]string(nil), left...)
	rightCopy := append([]string(nil), right...)
	sort.Strings(leftCopy)
	sort.Strings(rightCopy)
	for index := range leftCopy {
		if leftCopy[index] != rightCopy[index] {
			return false
		}
	}
	return true
}

func confidenceString(value float64) string {
	if value >= 1 {
		return "1.000"
	}
	if value <= 0 {
		return "0.000"
	}
	scaled := int(value*1000 + 0.5)
	return "0." + leftPad3(scaled)
}

func leftPad3(value int) string {
	if value >= 100 {
		return intString(value)
	}
	if value >= 10 {
		return "0" + intString(value)
	}
	return "00" + intString(value)
}
