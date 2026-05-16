package cpp

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
	case "identifier", "type_identifier", "field_identifier", "namespace_identifier":
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

func normalizeCPPType(raw string) string {
	value := strings.TrimSpace(raw)
	replacements := []string{
		"const ",
		"volatile ",
		"constexpr ",
		"static ",
		"inline ",
		"virtual ",
	}
	for _, replacement := range replacements {
		value = strings.ReplaceAll(value, replacement, "")
	}
	value = strings.Trim(value, "&* ")
	return strings.TrimSpace(value)
}

func baseCPPType(raw string) string {
	value := normalizeCPPType(raw)
	value = strings.TrimLeft(value, "*&")
	value = strings.TrimSpace(value)
	if index := strings.LastIndex(value, " "); index >= 0 {
		value = value[index+1:]
	}
	if index := strings.LastIndex(value, "::"); index >= 0 {
		value = value[index+2:]
	}
	return value
}

func includeName(raw string) string {
	value := strings.TrimSpace(raw)
	return strings.Trim(value, "\"<>")
}

func hasFunctionDeclarator(node *sitter.Node) bool {
	return functionDeclarator(node) != nil
}

func functionDeclarator(node *sitter.Node) *sitter.Node {
	if node == nil {
		return nil
	}
	if node.Kind() == "function_declarator" {
		return node
	}
	for index := uint(0); index < node.NamedChildCount(); index++ {
		candidate := node.NamedChild(index)
		if candidate == nil {
			continue
		}
		switch candidate.Kind() {
		case "function_declarator":
			return candidate
		case "pointer_declarator", "reference_declarator", "parenthesized_declarator":
			if nested := functionDeclarator(candidate); nested != nil {
				return nested
			}
		}
	}
	return nil
}
