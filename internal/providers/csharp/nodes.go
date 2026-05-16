package csharp

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

func directChildrenOfKind(node *sitter.Node, kind string) []*sitter.Node {
	var out []*sitter.Node
	if node == nil {
		return out
	}
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
	for index := uint(0); index < node.NamedChildCount(); index++ {
		candidate := node.NamedChild(index)
		if isIdentifierLike(candidate) {
			return candidate
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
		if isIdentifierLike(candidate) {
			return candidate
		}
		if nested := lastIdentifierLikeChild(candidate); nested != nil {
			return nested
		}
	}
	return nil
}

func isIdentifierLike(node *sitter.Node) bool {
	if node == nil {
		return false
	}
	switch node.Kind() {
	case "identifier", "qualified_name", "generic_name", "predefined_type":
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
		child := args.NamedChild(index)
		if child != nil && child.Kind() == "argument" {
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

func normalizeCSharpType(raw string) string {
	return strings.TrimSpace(raw)
}

func baseCSharpType(raw string) string {
	value := normalizeCSharpType(raw)
	if index := strings.LastIndex(value, "."); index >= 0 {
		value = value[index+1:]
	}
	if index := strings.IndexAny(value, "<[("); index >= 0 {
		value = strings.TrimSpace(value[:index])
	}
	return value
}

func lastNamePart(value string) string {
	value = strings.TrimSpace(value)
	if index := strings.LastIndex(value, "."); index >= 0 {
		return value[index+1:]
	}
	return value
}
