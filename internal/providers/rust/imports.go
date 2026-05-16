package rust

import (
	sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

func (c *collector) emitImport(node *sitter.Node) {
	if node.Kind() != "use_declaration" {
		return
	}
	arg := child(node, "argument")
	if arg == nil {
		arg = node.NamedChild(0)
	}
	if arg == nil {
		return
	}
	kind := scopeir.ImportNamed
	localNode := lastIdentifierLikeChild(arg)
	importedNode := localNode
	alias := ""
	if arg.Kind() == "use_as_clause" {
		if aliasNode := child(arg, "alias"); aliasNode != nil {
			alias = c.text(aliasNode)
			localNode = aliasNode
		}
		if pathNode := child(arg, "path"); pathNode != nil {
			importedNode = lastIdentifierLikeChild(pathNode)
		}
		kind = scopeir.ImportAlias
	}
	if arg.Kind() == "use_wildcard" {
		kind = scopeir.ImportWildcard
	}
	local := c.text(localNode)
	imported := c.text(importedNode)
	if local == "" && imported == "" {
		local = c.text(arg)
		imported = local
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
		TargetRaw:    stringPtr(c.text(arg)),
	})
}

func importID(filePath string, rng scopeir.Range, name string) string {
	return "import:" + filePath + "#" + rangeID(rng) + ":" + name
}
