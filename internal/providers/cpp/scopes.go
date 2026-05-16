package cpp

import (
	"sort"

	sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

type scopeCandidate struct {
	id   string
	kind scopeir.ScopeKind
	rng  scopeir.Range
}

func (c *collector) collectScopes(root *sitter.Node) {
	c.addScopeCandidate(scopeir.ScopeModule, root)
	walk(root, func(node *sitter.Node) {
		switch node.Kind() {
		case "namespace_definition":
			c.addScopeCandidate(scopeir.ScopeNamespace, node)
		case "class_specifier", "struct_specifier":
			c.addScopeCandidate(scopeir.ScopeClass, node)
		case "function_definition":
			c.addScopeCandidate(scopeir.ScopeFunction, node)
		case "declaration", "field_declaration":
			if hasFunctionDeclarator(node) {
				c.addScopeCandidate(scopeir.ScopeFunction, node)
			}
		}
	})
}

func (c *collector) addScopeCandidate(kind scopeir.ScopeKind, node *sitter.Node) {
	rng := nodeRange(node)
	c.scopeCandidates = append(c.scopeCandidates, scopeCandidate{
		id:   scopeID(c.filePath, rng, kind),
		kind: kind,
		rng:  rng,
	})
}

func (c *collector) buildScopes() {
	candidates := append([]scopeCandidate(nil), c.scopeCandidates...)
	sort.Slice(candidates, func(i, j int) bool {
		a, b := candidates[i].rng, candidates[j].rng
		if a.StartLine != b.StartLine {
			return a.StartLine < b.StartLine
		}
		if a.StartCol != b.StartCol {
			return a.StartCol < b.StartCol
		}
		if a.EndLine != b.EndLine {
			return a.EndLine > b.EndLine
		}
		return a.EndCol > b.EndCol
	})

	stack := make([]scopeCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		for len(stack) > 0 && !rangeStrictlyContains(stack[len(stack)-1].rng, candidate.rng) {
			stack = stack[:len(stack)-1]
		}
		var parent *string
		if len(stack) > 0 {
			parent = stringPtr(stack[len(stack)-1].id)
		}
		scope := scopeir.ScopeFact{
			ID:       candidate.id,
			Parent:   parent,
			Kind:     candidate.kind,
			Range:    candidate.rng,
			FilePath: c.filePath,
			FileHash: c.fileHash,
		}
		c.scopeIndex[scope.ID] = len(c.scopes)
		c.scopes = append(c.scopes, scope)
		stack = append(stack, candidate)
	}
}

func (c *collector) innermostScopeID(rng scopeir.Range) string {
	bestID := ""
	bestSpan := int(^uint(0) >> 1)
	for _, scope := range c.scopes {
		if scope.FilePath != c.filePath || !rangeContains(scope.Range, rng) {
			continue
		}
		span := rangeSpan(scope.Range)
		if span < bestSpan {
			bestSpan = span
			bestID = scope.ID
		}
	}
	if bestID != "" {
		return bestID
	}
	for _, scope := range c.scopes {
		if scope.Kind == scopeir.ScopeModule {
			return scope.ID
		}
	}
	return ""
}

func (c *collector) scopeByID(id string) *scopeir.ScopeFact {
	index, ok := c.scopeIndex[id]
	if !ok {
		return nil
	}
	return &c.scopes[index]
}

func scopeID(filePath string, rng scopeir.Range, kind scopeir.ScopeKind) string {
	return "scope:" + filePath + "#" + rangeID(rng) + ":" + string(kind)
}

func rangeID(rng scopeir.Range) string {
	return intString(rng.StartLine) + ":" + intString(rng.StartCol) + "-" + intString(rng.EndLine) + ":" + intString(rng.EndCol)
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
