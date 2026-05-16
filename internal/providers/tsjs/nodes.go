package tsjs

import (
	"strconv"
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

var builtinTypeNames = map[string]struct{}{
	"any": {}, "unknown": {}, "never": {}, "void": {}, "string": {}, "number": {},
	"boolean": {}, "bigint": {}, "symbol": {}, "object": {}, "undefined": {},
	"null": {}, "true": {}, "false": {},
}

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
	matches := descendantsOfType(node, kind)
	if len(matches) == 0 {
		return nil
	}
	return matches[0]
}

func descendantsOfType(node *sitter.Node, kind string) []*sitter.Node {
	var out []*sitter.Node
	walk(node, func(candidate *sitter.Node) {
		if candidate == nil || candidate.Id() == node.Id() {
			return
		}
		if candidate.Kind() == kind {
			out = append(out, candidate)
		}
	})
	return out
}

func namedIdentifierChildren(node *sitter.Node) []*sitter.Node {
	var out []*sitter.Node
	if node == nil {
		return out
	}
	for index := uint(0); index < node.NamedChildCount(); index++ {
		candidate := node.NamedChild(index)
		if candidate != nil && isIdentifierLike(candidate) {
			out = append(out, candidate)
		}
	}
	return out
}

func firstIdentifierChild(node *sitter.Node) *sitter.Node {
	children := namedIdentifierChildren(node)
	if len(children) == 0 {
		return nil
	}
	return children[0]
}

func firstIdentifierLikeChild(node *sitter.Node) *sitter.Node {
	if node == nil {
		return nil
	}
	for index := uint(0); index < node.NamedChildCount(); index++ {
		candidate := node.NamedChild(index)
		if candidate != nil && isIdentifierLike(candidate) {
			return candidate
		}
	}
	return nil
}

func isIdentifierLike(node *sitter.Node) bool {
	switch node.Kind() {
	case "identifier", "type_identifier", "property_identifier", "private_property_identifier":
		return true
	default:
		return false
	}
}

func isFunctionScopeNode(node *sitter.Node) bool {
	return node != nil && isFunctionScopeKind(node.Kind())
}

func isFunctionScopeKind(kind string) bool {
	switch kind {
	case "function_declaration", "function_signature", "method_definition",
		"abstract_method_signature", "method_signature", "arrow_function", "function_expression":
		return true
	default:
		return false
	}
}

func isFunctionExpression(node *sitter.Node) bool {
	return node != nil && (node.Kind() == "arrow_function" || node.Kind() == "function_expression")
}

func unwrapExpression(node *sitter.Node) *sitter.Node {
	current := node
	for current != nil {
		switch current.Kind() {
		case "as_expression", "non_null_expression", "parenthesized_expression":
			next := current.NamedChild(0)
			if next == nil {
				return current
			}
			current = next
		default:
			return current
		}
	}
	return nil
}

func unwrapAwaitExpression(node *sitter.Node) *sitter.Node {
	if node != nil && node.Kind() == "await_expression" {
		if child := node.NamedChild(0); child != nil {
			return child
		}
	}
	return node
}

func countArguments(args *sitter.Node) *int {
	if args == nil {
		return nil
	}
	count := 0
	for index := uint(0); index < args.NamedChildCount(); index++ {
		child := args.NamedChild(index)
		if child != nil && child.Kind() != "comment" {
			count++
		}
	}
	return &count
}

func stripQuotes(value string) string {
	trimmed := strings.TrimSpace(value)
	if len(trimmed) >= 2 {
		first, last := trimmed[0], trimmed[len(trimmed)-1]
		if (first == '\'' && last == '\'') || (first == '"' && last == '"') {
			return trimmed[1 : len(trimmed)-1]
		}
	}
	return trimmed
}

func stripTypeAnnotation(value string) string {
	trimmed := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(value), ":"))
	if trimmed == "" {
		return ""
	}
	if strings.HasSuffix(trimmed, "?") {
		trimmed = strings.TrimSpace(strings.TrimSuffix(trimmed, "?"))
	}
	if !strings.Contains(trimmed, "|") {
		return trimmed
	}
	parts := strings.Split(trimmed, "|")
	kept := make([]string, 0, len(parts))
	for _, part := range parts {
		name := strings.TrimSpace(part)
		switch name {
		case "", "null", "undefined":
			continue
		default:
			kept = append(kept, name)
		}
	}
	if len(kept) != 1 {
		return ""
	}
	return kept[0]
}

func moduleNameFromTarget(targetRaw string) string {
	normalized := strings.ReplaceAll(targetRaw, "\\", "/")
	parts := strings.Split(normalized, "/")
	for index := len(parts) - 1; index >= 0; index-- {
		if parts[index] != "" {
			return parts[index]
		}
	}
	return targetRaw
}

func intString(value int) string {
	return strconv.Itoa(value)
}

func stringPtr(value string) *string {
	return &value
}
