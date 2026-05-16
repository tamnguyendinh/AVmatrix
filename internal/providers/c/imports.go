package c

import (
	sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

func (c *collector) emitImport(node *sitter.Node) {
	if node.Kind() != "preproc_include" {
		return
	}
	targetNode := firstNamedChildOfType(node, "string_literal")
	if targetNode == nil {
		targetNode = firstNamedChildOfType(node, "system_lib_string")
	}
	target := includeName(c.text(targetNode))
	if target == "" {
		return
	}
	rng := nodeRange(node)
	c.imports = append(c.imports, scopeir.ImportFact{
		ID:           importID(c.filePath, rng, target),
		FilePath:     c.filePath,
		FileHash:     c.fileHash,
		Kind:         scopeir.ImportNamed,
		LocalName:    target,
		ImportedName: target,
		TargetRaw:    stringPtr(target),
	})
}

func importID(filePath string, rng scopeir.Range, name string) string {
	return "import:" + filePath + "#" + rangeID(rng) + ":" + name
}
