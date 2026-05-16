package golang

import (
	"strconv"
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

var goBuiltinTypeNames = map[string]struct{}{
	"any": {}, "bool": {}, "byte": {}, "complex64": {}, "complex128": {}, "error": {},
	"float32": {}, "float64": {}, "int": {}, "int8": {}, "int16": {}, "int32": {},
	"int64": {}, "rune": {}, "string": {}, "uint": {}, "uint8": {}, "uint16": {},
	"uint32": {}, "uint64": {}, "uintptr": {},
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

func namedChildrenOfType(node *sitter.Node, kind string) []*sitter.Node {
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
		if candidate != nil && isIdentifierLike(candidate) {
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
	case "identifier", "field_identifier", "package_identifier", "type_identifier":
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
		if (first == '"' && last == '"') || (first == '`' && last == '`') {
			return trimmed[1 : len(trimmed)-1]
		}
	}
	return trimmed
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

func normalizeGoType(raw string) string {
	value := strings.TrimSpace(raw)
	for {
		before := value
		value = strings.TrimSpace(strings.TrimPrefix(value, "*"))
		value = strings.TrimSpace(strings.TrimPrefix(value, "[]"))
		value = strings.TrimSpace(strings.TrimPrefix(value, "..."))
		value = strings.TrimSpace(strings.TrimPrefix(value, "<-chan "))
		value = strings.TrimSpace(strings.TrimPrefix(value, "chan<- "))
		value = strings.TrimSpace(strings.TrimPrefix(value, "chan "))
		if value == before {
			break
		}
	}
	return value
}

func baseGoType(raw string) string {
	value := normalizeGoType(raw)
	if strings.HasPrefix(value, "map[") {
		if end := strings.LastIndex(value, "]"); end >= 0 && end+1 < len(value) {
			return baseGoType(value[end+1:])
		}
	}
	if index := strings.LastIndex(value, "."); index >= 0 {
		return value[index+1:]
	}
	return value
}

func intString(value int) string {
	return strconv.Itoa(value)
}

func stringPtr(value string) *string {
	return &value
}
