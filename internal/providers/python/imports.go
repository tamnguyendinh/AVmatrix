package python

import (
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

func (c *collector) emitImport(node *sitter.Node) {
	switch node.Kind() {
	case "import_statement":
		for _, item := range importItems(node) {
			imported, alias := importNameAndAlias(c, item)
			if imported == "" {
				continue
			}
			local := lastNamePart(imported)
			kind := scopeir.ImportNamed
			if alias != "" {
				kind = scopeir.ImportAlias
				local = alias
			}
			c.addImport(kind, local, lastNamePart(imported), alias, imported)
		}
	case "import_from_statement":
		moduleNode := directChildOfKind(node, "dotted_name")
		moduleName := c.text(moduleNode)
		if moduleName == "" {
			return
		}
		for _, item := range importItems(node) {
			if item.Id() == moduleNode.Id() {
				continue
			}
			imported, alias := importNameAndAlias(c, item)
			if imported == "" || imported == moduleName {
				continue
			}
			local := lastNamePart(imported)
			kind := scopeir.ImportNamed
			if alias != "" {
				kind = scopeir.ImportAlias
				local = alias
			}
			c.addImport(kind, local, lastNamePart(imported), alias, moduleName+"."+imported)
		}
	}
}

func importItems(node *sitter.Node) []*sitter.Node {
	return directChildrenExcept(node, map[string]struct{}{
		"comment": {},
	})
}

func importNameAndAlias(c *collector, node *sitter.Node) (string, string) {
	if node == nil {
		return "", ""
	}
	if node.Kind() == "aliased_import" {
		nameNode := directChildOfKind(node, "dotted_name")
		if nameNode == nil {
			nameNode = firstIdentifierLikeChild(node)
		}
		aliasNode := lastIdentifierLikeChild(node)
		alias := ""
		if aliasNode != nil && nameNode != nil && aliasNode.Id() != nameNode.Id() {
			alias = c.text(aliasNode)
		}
		return c.text(nameNode), alias
	}
	if node.Kind() == "dotted_name" || node.Kind() == "identifier" {
		return c.text(node), ""
	}
	return "", ""
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

func lastNamePart(value string) string {
	parts := strings.Split(value, ".")
	for index := len(parts) - 1; index >= 0; index-- {
		if parts[index] != "" {
			return parts[index]
		}
	}
	return value
}
