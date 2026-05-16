package dart

import (
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

func (c *collector) emitImport(node *sitter.Node) {
	if node.Kind() != "import_or_export" {
		return
	}
	uriNode := firstDescendantOfType(node, "uri")
	if uriNode == nil {
		return
	}
	target := strings.Trim(c.text(uriNode), `"'`)
	if target == "" {
		return
	}
	imported := importNameFromURI(target)
	local := imported
	alias := ""
	kind := scopeir.ImportNamed
	if aliasNode := lastDescendantOfType(node, "identifier"); aliasNode != nil {
		alias = c.text(aliasNode)
		local = alias
		kind = scopeir.ImportAlias
	}
	rng := nodeRange(node)
	c.imports = append(c.imports, scopeir.ImportFact{
		ID:           importID(c.filePath, rng, local),
		FilePath:     c.filePath,
		FileHash:     c.fileHash,
		Kind:         kind,
		LocalName:    local,
		ImportedName: imported,
		Alias:        alias,
		TargetRaw:    stringPtr(target),
	})
}

func importID(filePath string, rng scopeir.Range, name string) string {
	return "import:" + filePath + "#" + rangeID(rng) + ":" + name
}
