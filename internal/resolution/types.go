package resolution

import (
	"github.com/tamnguyendinh/avmatrix-go/internal/graph"
	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

type Options struct {
	DisableScopeInheritsCompatibility bool
	SkipCompatibilityCrossFile        bool
}

type Metrics struct {
	DefinitionsIndexed          int
	ImportsResolved             int
	ImportUsesEmitted           int
	ResolvedReferences          int
	UnresolvedReferences        int
	ResolvedCalls               int
	ResolvedAccesses            int
	ResolvedTypeReferences      int
	ResolvedInheritance         int
	DuplicateEdgesMerged        int
	MethodOverridesEmitted      int
	MethodImplementsEmitted     int
	FinalizedImportsEmitted     int
	CrossFileFilesReprocessed   int
	CrossFileSkipped            bool
	CrossFileSkipReason         string
	BindingAccumulatorFiles     int
	BindingAccumulatorEntries   int
	BindingAccumulatorFinalized bool
	BindingAccumulatorDisposed  bool
	GraphNodesEmitted           int
	GraphRelationshipsEmitted   int
}

type Result struct {
	Graph          *graph.Graph
	ReferenceIndex ReferenceIndex
	Metrics        Metrics
}

type BindingResult struct {
	workspace *workspace
	Metrics   Metrics
}

type ReferenceIndex struct {
	BySourceScope map[string][]Reference
	ByTargetDef   map[string][]Reference
}

func newReferenceIndex() ReferenceIndex {
	return ReferenceIndex{
		BySourceScope: make(map[string][]Reference),
		ByTargetDef:   make(map[string][]Reference),
	}
}

func (index ReferenceIndex) add(reference Reference) {
	index.BySourceScope[reference.FromScope] = append(index.BySourceScope[reference.FromScope], reference)
	index.ByTargetDef[reference.ToDefID] = append(index.ByTargetDef[reference.ToDefID], reference)
}

type ReferenceKind string

const (
	ReferenceCall          ReferenceKind = "call"
	ReferenceRead          ReferenceKind = "read"
	ReferenceWrite         ReferenceKind = "write"
	ReferenceTypeReference ReferenceKind = "type-reference"
	ReferenceInherits      ReferenceKind = "inherits"
	ReferenceImportUse     ReferenceKind = "import-use"
)

type Reference struct {
	FromScope  string
	ToDefID    string
	FileHash   string
	Range      scopeir.Range
	Kind       ReferenceKind
	Confidence float64
	Evidence   []graph.Evidence
}
