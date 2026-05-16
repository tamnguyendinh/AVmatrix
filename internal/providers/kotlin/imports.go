package kotlin

import (
	sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

func (c *collector) emitImport(node *sitter.Node) {
	if node.Kind() != "import" {
		return
	}
	targetNode := firstNamedChildOfType(node, "qualified_identifier")
	target := c.text(targetNode)
	if target == "" {
		return
	}
	imported := lastNamePart(target)
	rng := nodeRange(node)
	c.imports = append(c.imports, scopeir.ImportFact{
		ID:           importID(c.filePath, rng, imported),
		FilePath:     c.filePath,
		FileHash:     c.fileHash,
		Kind:         scopeir.ImportNamed,
		LocalName:    imported,
		ImportedName: imported,
		TargetRaw:    stringPtr(target),
	})
}

func importID(filePath string, rng scopeir.Range, name string) string {
	return "import:" + filePath + "#" + rangeID(rng) + ":" + name
}
