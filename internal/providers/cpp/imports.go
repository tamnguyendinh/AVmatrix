package cpp

import (
	sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

func (c *collector) emitImport(node *sitter.Node) {
	switch node.Kind() {
	case "preproc_include":
		c.emitIncludeImport(node)
	case "using_declaration":
		c.emitUsingImport(node)
	}
}

func (c *collector) emitIncludeImport(node *sitter.Node) {
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

func (c *collector) emitUsingImport(node *sitter.Node) {
	nameNode := lastIdentifierLikeChild(node)
	if nameNode == nil {
		return
	}
	name := c.text(nameNode)
	if name == "" {
		return
	}
	kind := scopeir.ImportNamed
	if c.text(node) == "using namespace "+name+";" {
		kind = scopeir.ImportNamespace
	}
	rng := nodeRange(node)
	c.imports = append(c.imports, scopeir.ImportFact{
		ID:           importID(c.filePath, rng, name),
		FilePath:     c.filePath,
		FileHash:     c.fileHash,
		Kind:         kind,
		LocalName:    name,
		ImportedName: name,
		TargetRaw:    stringPtr(c.text(node)),
	})
}

func importID(filePath string, rng scopeir.Range, name string) string {
	return "import:" + filePath + "#" + rangeID(rng) + ":" + name
}
