package golang

import (
	sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

func (c *collector) emitImport(node *sitter.Node) {
	c.emitImportKind(node, node.Kind())
}

func (c *collector) emitImportKind(node *sitter.Node, nodeKind string) {
	if nodeKind != "import_spec" {
		return
	}
	pathNode := child(node, "path")
	if pathNode == nil {
		return
	}
	targetRaw := stripQuotes(c.text(pathNode))
	importedName := moduleNameFromTarget(targetRaw)
	localName := importedName
	alias := ""
	kind := scopeir.ImportNamed

	if nameNode := child(node, "name"); nameNode != nil {
		alias = c.text(nameNode)
		switch alias {
		case ".":
			kind = scopeir.ImportWildcardExpanded
			localName = ""
		case "_":
			kind = scopeir.ImportWildcard
			localName = "_"
		default:
			kind = scopeir.ImportAlias
			localName = alias
		}
	}
	c.addImport(kind, localName, importedName, alias, targetRaw)
}

func (c *collector) addImport(kind scopeir.ImportKind, localName string, importedName string, alias string, targetRaw string) {
	target := targetRaw
	c.imports = append(c.imports, scopeir.ImportFact{
		FilePath:     c.filePath,
		FileHash:     c.fileHash,
		Kind:         kind,
		LocalName:    localName,
		ImportedName: importedName,
		Alias:        alias,
		TargetRaw:    &target,
	})
}
