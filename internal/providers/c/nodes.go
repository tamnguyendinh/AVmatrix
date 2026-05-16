package c

import (
	"strconv"
	"strings"

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

func firstIdentifierLikeChild(node *sitter.Node) *sitter.Node {
	if node == nil {
		return nil
	}
	for index := uint(0); index < node.NamedChildCount(); index++ {
		candidate := node.NamedChild(index)
		if isIdentifierLike(candidate) {
			return candidate
		}
		if nested := firstIdentifierLikeChild(candidate); nested != nil {
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
		candidate := node.NamedChild(uint(index))
		if nested := lastIdentifierLikeChild(candidate); nested != nil {
			return nested
		}
		if isIdentifierLike(candidate) {
			return candidate
		}
	}
	return nil
}

func isIdentifierLike(node *sitter.Node) bool {
	if node == nil {
		return false
	}
	switch node.Kind() {
	case "identifier", "type_identifier", "field_identifier":
		return true
	default:
		return false
	}
}

func countArguments(args *sitter.Node) *int {
	if args == nil {
		return nil
	}
	count := 0
	for index := uint(0); index < args.NamedChildCount(); index++ {
		if child := args.NamedChild(index); child != nil {
			count++
		}
	}
	return &count
}

func intString(value int) string {
	return strconv.Itoa(value)
}

func stringPtr(value string) *string {
	return &value
}

func normalizeCType(raw string) string {
	value := strings.TrimSpace(raw)
	value = strings.ReplaceAll(value, "const ", "")
	value = strings.ReplaceAll(value, "volatile ", "")
	return strings.TrimSpace(value)
}

func baseCType(raw string) string {
	value := normalizeCType(raw)
	value = strings.TrimLeft(value, "*")
	value = strings.TrimSpace(value)
	if index := strings.LastIndex(value, " "); index >= 0 {
		value = value[index+1:]
	}
	return value
}

func includeName(raw string) string {
	value := strings.TrimSpace(raw)
	return strings.Trim(value, "\"<>")
}
