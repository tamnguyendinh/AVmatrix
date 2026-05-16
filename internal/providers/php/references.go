package php

import (
	sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

func (c *collector) emitReference(node *sitter.Node) {
	switch node.Kind() {
	case "member_call_expression":
		nameNode := child(node, "name")
		if nameNode == nil {
			return
		}
		c.addCall(node, nameNode, child(node, "object"), scopeir.CallMember, countArguments(child(node, "arguments")))
	case "function_call_expression":
		nameNode := child(node, "function")
		if nameNode == nil {
			nameNode = firstIdentifierLikeChild(node)
		}
		c.addCall(node, nameNode, nil, scopeir.CallFree, countArguments(child(node, "arguments")))
	case "object_creation_expression":
		nameNode := child(node, "name")
		if nameNode == nil {
			nameNode = firstIdentifierLikeChild(node)
		}
		c.addCall(node, nameNode, nil, scopeir.CallConstructor, countArguments(child(node, "arguments")))
	case "member_access_expression":
		nameNode := child(node, "name")
		if nameNode != nil {
			c.addAccess(node, nameNode, child(node, "object"), memberAccessKind(node))
		}
	case "base_clause":
		for index := uint(0); index < node.NamedChildCount(); index++ {
			candidate := node.NamedChild(index)
			if isIdentifierLike(candidate) {
				c.addHeritage(candidate, candidate, scopeir.HeritageExtends)
			}
		}
	case "class_interface_clause":
		for index := uint(0); index < node.NamedChildCount(); index++ {
			candidate := node.NamedChild(index)
			if isIdentifierLike(candidate) {
				c.addHeritage(candidate, candidate, scopeir.HeritageImplements)
			}
		}
	}
}

func memberAccessKind(node *sitter.Node) scopeir.AccessKind {
	parent := node.Parent()
	if parent == nil || parent.Kind() != "assignment_expression" {
		return scopeir.AccessRead
	}
	left := child(parent, "left")
	if left != nil && left.Id() == node.Id() {
		return scopeir.AccessWrite
	}
	return scopeir.AccessRead
}

func (c *collector) addCall(anchor *sitter.Node, nameNode *sitter.Node, receiver *sitter.Node, form scopeir.CallForm, arity *int) {
	name := basePHPName(c.text(nameNode))
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
		fact.ExplicitReceiver = c.receiverText(receiver)
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
		fact.ExplicitReceiver = c.receiverText(receiver)
	}
	c.accesses = append(c.accesses, fact)
}

func (c *collector) addHeritage(anchor *sitter.Node, nameNode *sitter.Node, kind scopeir.HeritageKind) {
	name := basePHPName(c.text(nameNode))
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
