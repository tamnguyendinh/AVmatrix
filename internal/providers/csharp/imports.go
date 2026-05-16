package csharp

import (
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

func (c *collector) emitImport(node *sitter.Node) {
	if node.Kind() != "using_directive" {
		return
	}
	target := strings.TrimSpace(c.text(node))
	target = strings.TrimSpace(strings.TrimPrefix(target, "using"))
	target = strings.TrimSpace(strings.TrimSuffix(target, ";"))
	if target == "" {
		return
	}
	alias := ""
	imported := ""
	local := ""
	kind := scopeir.ImportNamed
	if eq := strings.Index(target, "="); eq >= 0 {
		alias = strings.TrimSpace(target[:eq])
		target = strings.TrimSpace(target[eq+1:])
		if alias == "" || target == "" {
			return
		}
		local = alias
		imported = lastNamePart(target)
		kind = scopeir.ImportAlias
	} else {
		target = strings.TrimPrefix(target, "static ")
		local = lastNamePart(target)
		imported = local
	}
	if strings.HasSuffix(target, ".*") {
		kind = scopeir.ImportWildcard
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
