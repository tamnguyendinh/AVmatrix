package golang

import (
	sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

func (c *collector) emitReference(node *sitter.Node) {
	c.emitReferenceKind(node, node.Kind())
}

func (c *collector) emitReferenceKind(node *sitter.Node, kind string) {
	switch kind {
	case "call_expression":
		fn := child(node, "function")
		if fn == nil {
			return
		}
		arity := countArguments(child(node, "arguments"))
		if fn.Kind() == "selector_expression" {
			property := child(fn, "field")
			receiver := child(fn, "operand")
			if property != nil {
				c.addCall(node, property, receiver, scopeir.CallMember, arity)
			}
			return
		}
		if fn.Kind() == "identifier" {
			c.addCall(node, fn, nil, scopeir.CallFree, arity)
		}
	case "selector_expression":
		if isCallFunctionSelector(node) {
			return
		}
		property := child(node, "field")
		receiver := child(node, "operand")
		if property != nil {
			c.addAccess(node, property, receiver, selectorAccessKind(node))
		}
	case "composite_literal":
		typeNode := child(node, "type")
		if typeNode != nil {
			c.addCall(node, typeNode, nil, scopeir.CallConstructor, countCompositeLiteralElements(child(node, "body")))
		}
	case "field_declaration":
		c.emitEmbeddedFieldHeritage(node)
	case "type_elem":
		if nameNode := firstIdentifierLikeChild(node); nameNode != nil {
			c.addHeritage(node, nameNode, scopeir.HeritageExtends)
		}
	}
}

func (c *collector) addCall(anchor *sitter.Node, nameNode *sitter.Node, receiver *sitter.Node, form scopeir.CallForm, arity *int) {
	name := c.text(nameNode)
	if form == scopeir.CallConstructor {
		name = baseGoType(name)
	}
	if name == "" {
		return
	}
	rng := nodeRange(anchor)
	fact := scopeir.CallSiteFact{
		FilePath: c.filePath,
		FileHash: c.fileHash,
		Name:     name,
		Range:    rng,
		InScope:  c.innermostScopeID(rng),
		CallForm: form,
		Arity:    arity,
	}
	if receiver != nil {
		fact.ExplicitReceiver = c.text(receiver)
	}
	c.calls = append(c.calls, fact)
}

func (c *collector) addAccess(anchor *sitter.Node, nameNode *sitter.Node, receiver *sitter.Node, kind scopeir.AccessKind) {
	name := c.text(nameNode)
	if name == "" {
		return
	}
	rng := nodeRange(anchor)
	fact := scopeir.AccessFact{
		FilePath: c.filePath,
		FileHash: c.fileHash,
		Name:     name,
		Kind:     kind,
		Range:    rng,
		InScope:  c.innermostScopeID(rng),
	}
	if receiver != nil {
		fact.ExplicitReceiver = c.text(receiver)
	}
	c.accesses = append(c.accesses, fact)
}

func (c *collector) addHeritage(anchor *sitter.Node, nameNode *sitter.Node, kind scopeir.HeritageKind) {
	name := baseGoType(c.text(nameNode))
	if name == "" {
		return
	}
	rng := nodeRange(anchor)
	c.heritage = append(c.heritage, scopeir.HeritageFact{
		FilePath: c.filePath,
		FileHash: c.fileHash,
		Name:     name,
		Kind:     kind,
		Range:    rng,
		InScope:  c.innermostScopeID(rng),
	})
}

func (c *collector) emitEmbeddedFieldHeritage(node *sitter.Node) {
	if len(namedChildrenOfType(node, "field_identifier")) > 0 {
		return
	}
	typeNode := child(node, "type")
	if typeNode == nil {
		typeNode = firstIdentifierLikeChild(node)
	}
	if typeNode == nil {
		return
	}
	c.addHeritage(node, typeNode, scopeir.HeritageExtends)
}

func isCallFunctionSelector(node *sitter.Node) bool {
	parent := node.Parent()
	if parent == nil || parent.Kind() != "call_expression" {
		return false
	}
	fn := child(parent, "function")
	return fn != nil && fn.Id() == node.Id()
}

func selectorAccessKind(node *sitter.Node) scopeir.AccessKind {
	parent := node.Parent()
	if parent == nil {
		return scopeir.AccessRead
	}
	switch parent.Kind() {
	case "assignment_statement", "short_var_declaration":
		left := child(parent, "left")
		if left == nil {
			left = parent.NamedChild(0)
		}
		if left != nil && containsNode(left, node) {
			return scopeir.AccessWrite
		}
	case "inc_statement", "dec_statement":
		return scopeir.AccessWrite
	}
	return scopeir.AccessRead
}

func containsNode(root *sitter.Node, target *sitter.Node) bool {
	if root == nil || target == nil {
		return false
	}
	found := false
	walk(root, func(candidate *sitter.Node) {
		if candidate.Id() == target.Id() {
			found = true
		}
	})
	return found
}

func countCompositeLiteralElements(body *sitter.Node) *int {
	if body == nil {
		return nil
	}
	count := 0
	for index := uint(0); index < body.NamedChildCount(); index++ {
		child := body.NamedChild(index)
		if child != nil && child.Kind() != "comment" {
			count++
		}
	}
	return &count
}
