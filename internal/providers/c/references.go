package c

import (
	sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

func (c *collector) emitReference(node *sitter.Node) {
	switch node.Kind() {
	case "call_expression":
		nameNode := callName(node)
		if nameNode == nil {
			return
		}
		form := scopeir.CallFree
		if callReceiver(node) != nil {
			form = scopeir.CallMember
		}
		c.addCall(node, nameNode, callReceiver(node), form, countArguments(directChildOfKind(node, "argument_list")))
	case "field_expression":
		nameNode := lastIdentifierLikeChild(node)
		if nameNode != nil {
			c.addAccess(node, nameNode, fieldReceiver(node), scopeir.AccessRead)
		}
	}
}

func callName(node *sitter.Node) *sitter.Node {
	if node == nil {
		return nil
	}
	for index := uint(0); index < node.NamedChildCount(); index++ {
		candidate := node.NamedChild(index)
		if candidate == nil || candidate.Kind() == "argument_list" {
			continue
		}
		if candidate.Kind() == "field_expression" {
			return lastIdentifierLikeChild(candidate)
		}
		if isIdentifierLike(candidate) {
			return candidate
		}
	}
	return nil
}

func callReceiver(node *sitter.Node) *sitter.Node {
	if node == nil {
		return nil
	}
	if field := directChildOfKind(node, "field_expression"); field != nil {
		return fieldReceiver(field)
	}
	return nil
}

func fieldReceiver(node *sitter.Node) *sitter.Node {
	if node == nil || node.NamedChildCount() < 2 {
		return nil
	}
	return node.NamedChild(0)
}

func (c *collector) addCall(anchor *sitter.Node, nameNode *sitter.Node, receiver *sitter.Node, form scopeir.CallForm, arity *int) {
	name := c.text(nameNode)
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
