package swift

import (
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

func (c *collector) emitImport(node *sitter.Node) {
	if node.Kind() != "import_declaration" {
		return
	}
	nameNode := lastDescendantOfType(node, "identifier")
	if nameNode == nil {
		return
	}
	target := strings.TrimSpace(c.text(nameNode))
	if target == "" {
		return
	}
	imported := importNameFromPath(target)
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
