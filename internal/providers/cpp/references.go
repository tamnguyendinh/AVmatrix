package cpp

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
			c.addAccess(node, nameNode, fieldReceiver(node), fieldAccessKind(node))
		}
	case "base_class_clause":
		c.emitBaseClassHeritage(node)
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
		if candidate.Kind() == "qualified_identifier" {
			return lastIdentifierLikeChild(candidate)
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

func fieldAccessKind(node *sitter.Node) scopeir.AccessKind {
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

func (c *collector) emitBaseClassHeritage(node *sitter.Node) {
	for index := uint(0); index < node.NamedChildCount(); index++ {
		candidate := node.NamedChild(index)
		if candidate == nil {
			continue
		}
		if candidate.Kind() == "access_specifier" || candidate.Kind() == "attribute_declaration" {
			continue
		}
		nameNode := candidate
		if !isIdentifierLike(nameNode) && nameNode.Kind() != "qualified_identifier" {
			nameNode = lastIdentifierLikeChild(candidate)
		}
		if nameNode != nil {
			c.addHeritage(candidate, nameNode, scopeir.HeritageExtends)
		}
	}
}

func (c *collector) addCall(anchor *sitter.Node, nameNode *sitter.Node, receiver *sitter.Node, form scopeir.CallForm, arity *int) {
	name := baseCPPType(c.text(nameNode))
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
	name := baseCPPType(c.text(nameNode))
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
