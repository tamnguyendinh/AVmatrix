package embeddings

import "github.com/tamnguyendinh/avmatrix-go/internal/scopeir"

var chunkableLabels = map[scopeir.NodeLabel]bool{
	scopeir.NodeFunction:    true,
	scopeir.NodeMethod:      true,
	scopeir.NodeConstructor: true,
	scopeir.NodeClass:       true,
	scopeir.NodeInterface:   true,
	scopeir.NodeStruct:      true,
	scopeir.NodeEnum:        true,
	scopeir.NodeTrait:       true,
	scopeir.NodeImpl:        true,
	scopeir.NodeMacro:       true,
	scopeir.NodeNamespace:   true,
}

var shortLabels = map[scopeir.NodeLabel]bool{
	scopeir.NodeTypeAlias: true,
	scopeir.NodeTypedef:   true,
	scopeir.NodeConst:     true,
	scopeir.NodeProperty:  true,
	scopeir.NodeRecord:    true,
	scopeir.NodeUnion:     true,
	scopeir.NodeStatic:    true,
	scopeir.NodeVariable:  true,
}

func IsEmbeddableLabel(label scopeir.NodeLabel) bool {
	return chunkableLabels[label] || shortLabels[label]
}

func IsChunkableLabel(label scopeir.NodeLabel) bool {
	return chunkableLabels[label]
}

func IsShortLabel(label scopeir.NodeLabel) bool {
	return shortLabels[label]
}
