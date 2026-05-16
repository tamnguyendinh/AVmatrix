package tsjs

import (
	sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

func (c *collector) emitImport(node *sitter.Node) {
	c.emitImportKind(node, node.Kind())
}

func (c *collector) emitImportKind(node *sitter.Node, kind string) {
	switch kind {
	case "import_statement":
		c.emitImportStatement(node)
	case "export_statement":
		c.emitExportStatement(node)
	}
}

func (c *collector) emitImportStatement(node *sitter.Node) {
	sourceNode := child(node, "source")
	if sourceNode == nil {
		return
	}
	targetRaw := stripQuotes(c.text(sourceNode))
	importClause := firstNamedChildOfType(node, "import_clause")
	if importClause == nil {
		return
	}

	namespaceImport := firstDescendantOfType(importClause, "namespace_import")
	if namespaceImport != nil {
		localName := c.text(firstDescendantOfType(namespaceImport, "identifier"))
		if localName != "" {
			c.addImport(scopeir.ImportNamespace, localName, moduleNameFromTarget(targetRaw), "", targetRaw)
			c.importedLocalNames[localName] = struct{}{}
		}
		return
	}

	defaultName := c.text(firstNamedChildOfType(importClause, "identifier"))
	if defaultName != "" {
		c.addImport(scopeir.ImportNamed, defaultName, "default", "", targetRaw)
		c.importedLocalNames[defaultName] = struct{}{}
	}

	for _, specifier := range descendantsOfType(importClause, "import_specifier") {
		names := namedIdentifierChildren(specifier)
		imported := c.text(child(specifier, "name"))
		if imported == "" && len(names) > 0 {
			imported = c.text(names[0])
		}
		if imported == "" {
			continue
		}
		alias := ""
		if len(names) > 1 {
			alias = c.text(names[len(names)-1])
		}
		localName := imported
		kind := scopeir.ImportNamed
		if alias != "" && alias != imported {
			localName = alias
			kind = scopeir.ImportAlias
		}
		c.addImport(kind, localName, imported, alias, targetRaw)
		c.importedLocalNames[localName] = struct{}{}
	}
}

func (c *collector) emitExportStatement(node *sitter.Node) {
	sourceNode := child(node, "source")
	if sourceNode == nil {
		return
	}
	targetRaw := stripQuotes(c.text(sourceNode))
	if hasAnonymousChild(node, "*") {
		c.addImport(scopeir.ImportWildcard, "", "", "", targetRaw)
		return
	}
	for _, specifier := range descendantsOfType(node, "export_specifier") {
		names := namedIdentifierChildren(specifier)
		imported := c.text(child(specifier, "name"))
		if imported == "" && len(names) > 0 {
			imported = c.text(names[0])
		}
		if imported == "" {
			continue
		}
		alias := ""
		if len(names) > 1 {
			alias = c.text(names[len(names)-1])
		}
		localName := imported
		if alias != "" {
			localName = alias
		}
		c.addImport(scopeir.ImportReexport, localName, imported, alias, targetRaw)
	}
}

func (c *collector) addImport(kind scopeir.ImportKind, localName, importedName, alias, targetRaw string) {
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

func hasAnonymousChild(node *sitter.Node, text string) bool {
	if node == nil {
		return false
	}
	for index := uint(0); index < node.ChildCount(); index++ {
		child := node.Child(index)
		if child != nil && child.Kind() == text {
			return true
		}
	}
	return false
}
