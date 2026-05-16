package ruby

import (
	"fmt"

	sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

func declarationName(node *sitter.Node) *sitter.Node {
	if node == nil {
		return nil
	}
	switch node.Kind() {
	case "module", "class":
		return firstNamedChildOfType(node, "constant")
	case "method":
		return firstNamedChildOfType(node, "identifier")
	default:
		return firstIdentifierLikeChild(node)
	}
}

func declarationLabel(node *sitter.Node) scopeir.NodeLabel {
	if node != nil && node.Kind() == "module" {
		return scopeir.NodeTrait
	}
	return scopeir.NodeClass
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

func firstNamedChild(node *sitter.Node) *sitter.Node {
	if node == nil || node.NamedChildCount() == 0 {
		return nil
	}
	return node.NamedChild(0)
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

func namedChildren(node *sitter.Node) []*sitter.Node {
	if node == nil {
		return nil
	}
	out := make([]*sitter.Node, 0, node.NamedChildCount())
	for index := uint(0); index < node.NamedChildCount(); index++ {
		if child := node.NamedChild(index); child != nil {
			out = append(out, child)
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

func isIdentifierLike(node *sitter.Node) bool {
	if node == nil {
		return false
	}
	switch node.Kind() {
	case "identifier", "constant":
		return true
	default:
		return false
	}
}

func walk(node *sitter.Node, visit func(*sitter.Node)) {
	if node == nil {
		return
	}
	visit(node)
	for index := uint(0); index < node.NamedChildCount(); index++ {
		walk(node.NamedChild(index), visit)
	}
}

func scopeID(filePath string, rng scopeir.Range, kind scopeir.ScopeKind) string {
	return "scope:" + filePath + "#" + rangeID(rng) + ":" + string(kind)
}

func rangeID(rng scopeir.Range) string {
	return fmt.Sprintf("%d:%d-%d:%d", rng.StartLine, rng.StartCol, rng.EndLine, rng.EndCol)
}

func defID(filePath string, rng scopeir.Range, label scopeir.NodeLabel, name string) string {
	return "def:" + filePath + "#" + fmt.Sprintf("%d:%d:%s:%s", rng.StartLine, rng.StartCol, string(label), name)
}

func importID(filePath string, rng scopeir.Range, name string) string {
	return "import:" + filePath + "#" + rangeID(rng) + ":" + name
}

func rangeStrictlyContains(outer, inner scopeir.Range) bool {
	if outer == inner {
		return false
	}
	return rangeContains(outer, inner)
}

func rangeContains(outer, inner scopeir.Range) bool {
	startsBefore := outer.StartLine < inner.StartLine ||
		(outer.StartLine == inner.StartLine && outer.StartCol <= inner.StartCol)
	endsAfter := outer.EndLine > inner.EndLine ||
		(outer.EndLine == inner.EndLine && outer.EndCol >= inner.EndCol)
	return startsBefore && endsAfter
}

func rangeSpan(rng scopeir.Range) int {
	return (rng.EndLine-rng.StartLine)*1_000_000 + (rng.EndCol - rng.StartCol)
}

func stringPtr(value string) *string {
	return &value
}

func parentOfKind(node *sitter.Node, kind string) *sitter.Node {
	for current := node.Parent(); current != nil; current = current.Parent() {
		if current.Kind() == kind {
			return current
		}
	}
	return nil
}
