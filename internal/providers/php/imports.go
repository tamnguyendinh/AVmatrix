package php

import (
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

func (c *collector) emitImport(node *sitter.Node) {
	switch node.Kind() {
	case "namespace_use_declaration":
		if c.isFunctionOrConstUse(node) {
			return
		}
		if c.emitGroupedUseImports(node) {
			return
		}
		for _, clause := range directChildrenOfKind(node, "namespace_use_clause") {
			c.emitUseClauseImport(node, clause)
		}
	case "include_expression", "include_once_expression", "require_expression", "require_once_expression":
		c.emitIncludeImport(node)
	}
}

func (c *collector) emitGroupedUseImports(anchor *sitter.Node) bool {
	text := strings.TrimSpace(c.text(anchor))
	lower := strings.ToLower(text)
	if !strings.HasPrefix(lower, "use ") || !strings.Contains(text, "{") || !strings.Contains(text, "}") {
		return false
	}
	body := strings.TrimSpace(strings.TrimSuffix(text[len("use "):], ";"))
	open := strings.Index(body, "{")
	close := strings.LastIndex(body, "}")
	if open <= 0 || close <= open {
		return false
	}
	prefix := strings.Trim(strings.TrimSpace(body[:open]), "\\")
	if prefix == "" {
		return false
	}
	emitted := false
	for _, part := range strings.Split(body[open+1:close], ",") {
		name, alias := splitPHPUseAlias(strings.TrimSpace(part))
		if name == "" {
			continue
		}
		target := prefix + `\` + strings.Trim(name, "\\")
		imported := basePHPName(name)
		local := imported
		kind := scopeir.ImportNamed
		if alias != "" {
			local = alias
			kind = scopeir.ImportAlias
		}
		c.addImportFact(anchor, kind, local, imported, alias, target)
		emitted = true
	}
	return emitted
}

func (c *collector) isFunctionOrConstUse(node *sitter.Node) bool {
	text := strings.TrimSpace(c.text(node))
	lower := strings.ToLower(text)
	return strings.HasPrefix(lower, "use function ") || strings.HasPrefix(lower, "use const ")
}

func (c *collector) emitUseClauseImport(anchor *sitter.Node, clause *sitter.Node) {
	if clause == nil {
		return
	}
	targetNode := firstNamedChildOfType(clause, "qualified_name")
	if targetNode == nil {
		targetNode = firstNamedChildOfType(clause, "namespace_name")
	}
	if targetNode == nil {
		targetNode = firstIdentifierLikeChild(clause)
	}
	target := strings.TrimSpace(c.text(targetNode))
	if target == "" {
		return
	}
	imported := basePHPName(target)
	local := imported
	alias := ""
	kind := scopeir.ImportNamed
	if aliasNode := child(clause, "alias"); aliasNode != nil {
		alias = c.text(aliasNode)
		local = alias
		kind = scopeir.ImportAlias
	}
	c.addImportFact(anchor, kind, local, imported, alias, target)
}

func (c *collector) addImportFact(anchor *sitter.Node, kind scopeir.ImportKind, local string, imported string, alias string, target string) {
	rng := nodeRange(anchor)
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

func splitPHPUseAlias(text string) (string, string) {
	lower := strings.ToLower(text)
	index := strings.LastIndex(lower, " as ")
	if index < 0 {
		return strings.TrimSpace(text), ""
	}
	return strings.TrimSpace(text[:index]), strings.TrimSpace(text[index+4:])
}

func (c *collector) emitIncludeImport(node *sitter.Node) {
	target := c.includeTarget(node)
	if target == "" {
		return
	}
	rng := nodeRange(node)
	c.imports = append(c.imports, scopeir.ImportFact{
		ID:           importID(c.filePath, rng, target),
		FilePath:     c.filePath,
		FileHash:     c.fileHash,
		Kind:         scopeir.ImportDynamicUnresolved,
		LocalName:    target,
		ImportedName: target,
		TargetRaw:    stringPtr(target),
	})
}

func (c *collector) includeTarget(node *sitter.Node) string {
	for index := uint(0); index < node.NamedChildCount(); index++ {
		candidate := node.NamedChild(index)
		if candidate == nil {
			continue
		}
		if candidate.Kind() == "string" {
			return strings.Trim(c.text(candidate), `"'`)
		}
		if content := firstNamedChildOfType(candidate, "string_content"); content != nil {
			return c.text(content)
		}
	}
	return ""
}

func importID(filePath string, rng scopeir.Range, name string) string {
	return "import:" + filePath + "#" + rangeID(rng) + ":" + name
}
