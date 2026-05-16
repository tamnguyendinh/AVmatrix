package dart

import (
	"path"
	"strconv"
	"strings"
	"unicode"

	sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

func nodeRange(node *sitter.Node) scopeir.Range {
	start := node.StartPosition()
	end := node.EndPosition()
	return scopeir.Range{
		StartLine: int(start.Row) + 1,
		StartCol:  int(start.Column),
		EndLine:   int(end.Row) + 1,
		EndCol:    int(end.Column),
	}
}

func (c *collector) text(node *sitter.Node) string {
	if node == nil {
		return ""
	}
	return node.Utf8Text(c.source)
}

func child(node *sitter.Node, field string) *sitter.Node {
	if node == nil {
		return nil
	}
	return node.ChildByFieldName(field)
}

func firstNamedChildOfType(node *sitter.Node, kind string) *sitter.Node {
	if node == nil {
		return nil
	}
	for index := uint(0); index < node.NamedChildCount(); index++ {
		candidate := node.NamedChild(index)
		if candidate != nil && candidate.Kind() == kind {
			return candidate
		}
	}
	return nil
}

func firstDescendantOfType(node *sitter.Node, kind string) *sitter.Node {
	if node == nil {
		return nil
	}
	if node.Kind() == kind {
		return node
	}
	for index := uint(0); index < node.NamedChildCount(); index++ {
		if nested := firstDescendantOfType(node.NamedChild(index), kind); nested != nil {
			return nested
		}
	}
	return nil
}

func lastDescendantOfType(node *sitter.Node, kind string) *sitter.Node {
	if node == nil {
		return nil
	}
	for index := int(node.NamedChildCount()) - 1; index >= 0; index-- {
		if nested := lastDescendantOfType(node.NamedChild(uint(index)), kind); nested != nil {
			return nested
		}
	}
	if node.Kind() == kind {
		return node
	}
	return nil
}

func directChildOfKind(node *sitter.Node, kind string) *sitter.Node {
	if node == nil {
		return nil
	}
	for index := uint(0); index < node.NamedChildCount(); index++ {
		candidate := node.NamedChild(index)
		if candidate != nil && candidate.Kind() == kind {
			return candidate
		}
	}
	return nil
}

func directChildrenOfKind(node *sitter.Node, kind string) []*sitter.Node {
	if node == nil {
		return nil
	}
	out := make([]*sitter.Node, 0)
	for index := uint(0); index < node.NamedChildCount(); index++ {
		candidate := node.NamedChild(index)
		if candidate != nil && candidate.Kind() == kind {
			out = append(out, candidate)
		}
	}
	return out
}

func firstIdentifierLikeChild(node *sitter.Node) *sitter.Node {
	if node == nil {
		return nil
	}
	if isIdentifierLike(node) {
		return node
	}
	for index := uint(0); index < node.NamedChildCount(); index++ {
		if nested := firstIdentifierLikeChild(node.NamedChild(index)); nested != nil {
			return nested
		}
	}
	return nil
}

func lastIdentifierLikeChild(node *sitter.Node) *sitter.Node {
	if node == nil {
		return nil
	}
	for index := int(node.NamedChildCount()) - 1; index >= 0; index-- {
		if nested := lastIdentifierLikeChild(node.NamedChild(uint(index))); nested != nil {
			return nested
		}
	}
	if isIdentifierLike(node) {
		return node
	}
	return nil
}

func isIdentifierLike(node *sitter.Node) bool {
	if node == nil {
		return false
	}
	switch node.Kind() {
	case "identifier", "type_identifier":
		return true
	default:
		return false
	}
}

func intString(value int) string {
	return strconv.Itoa(value)
}

func stringPtr(value string) *string {
	return &value
}

func normalizeDartType(raw string) string {
	value := strings.TrimSpace(raw)
	value = strings.TrimPrefix(value, "final ")
	value = strings.TrimPrefix(value, "const ")
	value = strings.TrimSuffix(value, "?")
	return strings.TrimSpace(value)
}

func baseDartType(raw string) string {
	value := normalizeDartType(raw)
	if index := strings.LastIndex(value, "."); index >= 0 {
		value = value[index+1:]
	}
	if index := strings.IndexAny(value, "<|&[]("); index >= 0 {
		value = strings.TrimSpace(value[:index])
	}
	return value
}

func startsWithUpper(value string) bool {
	if value == "" {
		return false
	}
	first := []rune(value)[0]
	return unicode.IsUpper(first)
}

func importNameFromURI(uri string) string {
	uri = strings.Trim(uri, `"'`)
	base := path.Base(strings.ReplaceAll(uri, "\\", "/"))
	ext := path.Ext(base)
	return strings.TrimSuffix(base, ext)
}

func nodeRangeWithEnd(start *sitter.Node, end *sitter.Node) scopeir.Range {
	rng := nodeRange(start)
	if end != nil {
		endRange := nodeRange(end)
		rng.EndLine = endRange.EndLine
		rng.EndCol = endRange.EndCol
	}
	return rng
}
