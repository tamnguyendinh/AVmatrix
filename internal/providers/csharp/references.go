package csharp

import (
	sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

func (c *collector) emitReference(node *sitter.Node) {
	switch node.Kind() {
	case "invocation_expression":
		nameNode := invocationName(node)
		if nameNode == nil {
			return
		}
		form := scopeir.CallMember
		receiver := invocationReceiver(node)
		if receiver == nil {
			form = scopeir.CallFree
		}
		c.addCall(node, nameNode, receiver, form, countArguments(directChildOfKind(node, "argument_list")))
	case "member_access_expression":
		if isInvocationTarget(node) {
			return
		}
		nameNode := lastIdentifierLikeChild(node)
		if nameNode != nil {
			c.addAccess(node, nameNode, memberAccessReceiver(node), memberAccessKind(node))
		}
	case "base_list":
		c.emitBaseListHeritage(node)
	}
}

func invocationName(node *sitter.Node) *sitter.Node {
	if node == nil {
		return nil
	}
	for index := uint(0); index < node.NamedChildCount(); index++ {
		candidate := node.NamedChild(index)
		if candidate == nil || candidate.Kind() == "argument_list" {
			continue
		}
		if candidate.Kind() == "member_access_expression" {
			return lastIdentifierLikeChild(candidate)
		}
		if isIdentifierLike(candidate) {
			return candidate
		}
	}
	return nil
}

func invocationReceiver(node *sitter.Node) *sitter.Node {
	if field := directChildOfKind(node, "member_access_expression"); field != nil {
		return memberAccessReceiver(field)
	}
	return nil
}

func memberAccessReceiver(node *sitter.Node) *sitter.Node {
	if node == nil || node.NamedChildCount() < 2 {
		return nil
	}
	return node.NamedChild(0)
}

func isInvocationTarget(node *sitter.Node) bool {
	parent := node.Parent()
	if parent == nil || parent.Kind() != "invocation_expression" {
		return false
	}
	for index := uint(0); index < parent.NamedChildCount(); index++ {
		candidate := parent.NamedChild(index)
		if candidate != nil && candidate.Id() == node.Id() {
			return true
		}
	}
	return false
}

func memberAccessKind(node *sitter.Node) scopeir.AccessKind {
	parent := node.Parent()
	if parent == nil || parent.Kind() != "assignment_expression" {
		return scopeir.AccessRead
	}
	left := parent.NamedChild(0)
	if left != nil && left.Id() == node.Id() {
		return scopeir.AccessWrite
	}
	return scopeir.AccessRead
}

func (c *collector) emitBaseListHeritage(node *sitter.Node) {
	owner := parentOfKind(node, "class_declaration")
	if owner == nil {
		owner = parentOfKind(node, "interface_declaration")
	}
	first := true
	for index := uint(0); index < node.NamedChildCount(); index++ {
		candidate := node.NamedChild(index)
		if !isIdentifierLike(candidate) {
			continue
		}
		kind := scopeir.HeritageImplements
		if first && owner != nil && owner.Kind() == "class_declaration" {
			kind = scopeir.HeritageExtends
		}
		c.addHeritage(candidate, candidate, kind)
		first = false
	}
}

func (c *collector) addCall(anchor *sitter.Node, nameNode *sitter.Node, receiver *sitter.Node, form scopeir.CallForm, arity *int) {
	name := baseCSharpType(c.text(nameNode))
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
	name := baseCSharpType(c.text(nameNode))
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

func parentOfKind(node *sitter.Node, kind string) *sitter.Node {
	for current := node.Parent(); current != nil; current = current.Parent() {
		if current.Kind() == kind {
			return current
		}
	}
	return nil
}
