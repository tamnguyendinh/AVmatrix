package php

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
	case "name", "variable_name":
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
		if child := args.NamedChild(index); child != nil && child.Kind() == "argument" {
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

func normalizePHPType(raw string) string {
	value := strings.TrimSpace(raw)
	value = strings.TrimPrefix(value, "?")
	value = strings.TrimPrefix(value, "\\")
	value = strings.ReplaceAll(value, "\\", ".")
	return strings.TrimSpace(value)
}

func basePHPName(raw string) string {
	value := normalizePHPType(raw)
	value = strings.TrimPrefix(value, "$")
	if index := strings.LastIndex(value, "."); index >= 0 {
		value = value[index+1:]
	}
	if index := strings.IndexAny(value, "<|&[]("); index >= 0 {
		value = strings.TrimSpace(value[:index])
	}
	return strings.TrimPrefix(value, "$")
}

func (c *collector) receiverText(node *sitter.Node) string {
	if node == nil {
		return ""
	}
	switch node.Kind() {
	case "variable_name":
		return c.text(declarationName(node))
	case "member_access_expression":
		object := c.receiverText(child(node, "object"))
		name := c.text(child(node, "name"))
		if object == "" {
			return name
		}
		if name == "" {
			return object
		}
		return object + "." + name
	default:
		return strings.TrimPrefix(c.text(node), "$")
	}
}
