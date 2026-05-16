package kotlin

import (
	sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

func (c *collector) emitReference(node *sitter.Node) {
	switch node.Kind() {
	case "call_expression":
		if parentOfKind(node, "delegation_specifier") != nil {
			return
		}
		nameNode := callName(node)
		if nameNode == nil {
			return
		}
		form := scopeir.CallMember
		if startsWithUpper(baseKotlinType(c.text(nameNode))) {
			form = scopeir.CallConstructor
		}
		c.addCall(node, nameNode, callReceiver(node), form, countArguments(directChildOfKind(node, "value_arguments")))
	case "navigation_expression":
		if isCallCalleeNavigation(node) {
			return
		}
		nameNode := lastIdentifierLikeChild(node)
		if nameNode != nil {
			c.addAccess(node, nameNode, navigationReceiver(node), scopeir.AccessRead)
		}
	case "delegation_specifier":
		c.emitDelegationHeritage(node)
	}
}

func callName(node *sitter.Node) *sitter.Node {
	if node == nil {
		return nil
	}
	for index := uint(0); index < node.NamedChildCount(); index++ {
		candidate := node.NamedChild(index)
		if candidate == nil || candidate.Kind() == "value_arguments" {
			continue
		}
		if candidate.Kind() == "navigation_expression" {
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
	for index := uint(0); index < node.NamedChildCount(); index++ {
		candidate := node.NamedChild(index)
		if candidate != nil && candidate.Kind() == "navigation_expression" {
			return navigationReceiver(candidate)
		}
	}
	return nil
}

func navigationReceiver(node *sitter.Node) *sitter.Node {
	if node == nil || node.NamedChildCount() < 2 {
		return nil
	}
	return node.NamedChild(0)
}

func isCallCalleeNavigation(node *sitter.Node) bool {
	parent := node.Parent()
	if parent == nil || parent.Kind() != "call_expression" {
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

func (c *collector) emitDelegationHeritage(node *sitter.Node) {
	if constructor := directChildOfKind(node, "constructor_invocation"); constructor != nil {
		typeNode := firstNamedChildOfType(constructor, "user_type")
		if typeNode != nil {
			c.addHeritage(node, typeNode, scopeir.HeritageExtends)
		}
		return
	}
	typeNode := firstNamedChildOfType(node, "user_type")
	if typeNode != nil {
		c.addHeritage(node, typeNode, scopeir.HeritageImplements)
	}
}

func (c *collector) addCall(anchor *sitter.Node, nameNode *sitter.Node, receiver *sitter.Node, form scopeir.CallForm, arity *int) {
	name := baseKotlinType(c.text(nameNode))
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
	name := baseKotlinType(c.text(nameNode))
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
