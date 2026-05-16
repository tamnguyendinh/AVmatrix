package ruby

import (
	"path"
	"strings"
	"unicode"

	sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

func (c *collector) emitImport(node *sitter.Node) {
	if node.Kind() != "call" {
		return
	}
	name := callName(c, node)
	if name != "require" && name != "require_relative" {
		return
	}
	args := directChildOfKind(node, "argument_list")
	targetNode := firstDescendantOfType(args, "string_content")
	if targetNode == nil {
		return
	}
	target := c.text(targetNode)
	if invalidRubyImportPath(target) {
		return
	}
	if name == "require_relative" && !strings.HasPrefix(target, ".") {
		target = "./" + target
	}
	imported := importNameFromTarget(target)
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

func invalidRubyImportPath(target string) bool {
	if target == "" || len(target) > 1024 {
		return true
	}
	return strings.IndexFunc(target, unicode.IsControl) >= 0
}

func importNameFromTarget(target string) string {
	base := path.Base(strings.ReplaceAll(target, "\\", "/"))
	ext := path.Ext(base)
	return strings.TrimSuffix(base, ext)
}
