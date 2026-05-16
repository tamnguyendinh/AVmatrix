package java

import (
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

func (c *collector) emitImport(node *sitter.Node) {
	if node.Kind() != "import_declaration" {
		return
	}
	target := importTarget(c, node)
	if target == "" {
		return
	}
	kind := scopeir.ImportNamed
	imported := lastNamePart(target)
	local := imported
	if strings.HasSuffix(target, ".*") {
		kind = scopeir.ImportWildcard
		imported = "*"
		local = ""
	}
	c.imports = append(c.imports, scopeir.ImportFact{
		ID:           "import:" + c.filePath + ":" + rangeID(nodeRange(node)) + ":" + target,
		FilePath:     c.filePath,
		FileHash:     c.fileHash,
		Kind:         kind,
		LocalName:    local,
		ImportedName: imported,
		TargetRaw:    stringPtr(target),
	})
	if local == "" {
		return
	}
	scopeID := c.innermostScopeID(nodeRange(node))
	if scope := c.scopeByID(scopeID); scope != nil {
		scope.Bindings = append(scope.Bindings, scopeir.BindingFact{
			Name:   local,
			DefID:  "",
			Origin: scopeir.BindingImport,
		})
	}
}

func importTarget(c *collector, node *sitter.Node) string {
	name := c.text(lastIdentifierLikeChild(node))
	if name == "" {
		return ""
	}
	text := strings.TrimSpace(c.text(node))
	text = strings.TrimPrefix(text, "import")
	text = strings.TrimSpace(text)
	text = strings.TrimPrefix(text, "static")
	text = strings.TrimSpace(text)
	text = strings.TrimSuffix(text, ";")
	text = strings.TrimSpace(text)
	return text
}
