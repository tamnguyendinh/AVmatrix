package java

import (
	sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

func (c *collector) emitReference(node *sitter.Node) {
	switch node.Kind() {
	case "method_invocation":
		nameNode := methodInvocationName(node)
		if nameNode == nil {
			return
		}
		c.addCall(node, nameNode, methodInvocationReceiver(node), scopeir.CallMember, countArguments(directChildOfKind(node, "argument_list")))
	case "object_creation_expression":
		typeNode := firstIdentifierLikeChild(node)
		if typeNode != nil {
			c.addCall(node, typeNode, nil, scopeir.CallConstructor, countArguments(directChildOfKind(node, "argument_list")))
		}
	case "field_access":
		if isMethodInvocationReceiver(node) {
			return
		}
		nameNode := lastIdentifierLikeChild(node)
		if nameNode != nil {
			c.addAccess(node, nameNode, fieldAccessReceiver(node), fieldAccessKind(node))
		}
	case "superclass":
		nameNode := firstIdentifierLikeChild(node)
		if nameNode != nil {
			c.addHeritage(node, nameNode, scopeir.HeritageExtends)
		}
	case "super_interfaces":
		for _, typeNode := range directChildrenOfKinds(node, map[string]struct{}{"type_identifier": {}, "scoped_type_identifier": {}, "generic_type": {}, "type_list": {}}) {
			if typeNode.Kind() == "type_list" {
				for index := uint(0); index < typeNode.NamedChildCount(); index++ {
					child := typeNode.NamedChild(index)
					if child != nil {
						c.addHeritage(child, child, scopeir.HeritageImplements)
					}
				}
				continue
			}
			c.addHeritage(typeNode, typeNode, scopeir.HeritageImplements)
		}
	}
}

func methodInvocationName(node *sitter.Node) *sitter.Node {
	if name := child(node, "name"); name != nil {
		return name
	}
	for index := int(node.NamedChildCount()) - 1; index >= 0; index-- {
		candidate := node.NamedChild(uint(index))
		if candidate == nil || candidate.Kind() == "argument_list" {
			continue
		}
		if isIdentifierLike(candidate) {
			return candidate
		}
	}
	return nil
}

func methodInvocationReceiver(node *sitter.Node) *sitter.Node {
	if receiver := child(node, "object"); receiver != nil {
		return receiver
	}
	nameNode := methodInvocationName(node)
	for index := uint(0); index < node.NamedChildCount(); index++ {
		candidate := node.NamedChild(index)
		if candidate == nil || candidate.Kind() == "argument_list" || (nameNode != nil && candidate.Id() == nameNode.Id()) {
			continue
		}
		return candidate
	}
	return nil
}

func fieldAccessReceiver(node *sitter.Node) *sitter.Node {
	if receiver := child(node, "object"); receiver != nil {
		return receiver
	}
	if node == nil || node.NamedChildCount() < 2 {
		return nil
	}
	return node.NamedChild(0)
}

func isMethodInvocationReceiver(node *sitter.Node) bool {
	parent := node.Parent()
	if parent == nil || parent.Kind() != "method_invocation" {
		return false
	}
	receiver := methodInvocationReceiver(parent)
	return receiver != nil && receiver.Id() == node.Id()
}

func fieldAccessKind(node *sitter.Node) scopeir.AccessKind {
	parent := node.Parent()
	if parent == nil {
		return scopeir.AccessRead
	}
	if parent.Kind() == "assignment_expression" {
		left := child(parent, "left")
		if left == nil {
			left = parent.NamedChild(0)
		}
		if left != nil && containsNode(left, node) {
			return scopeir.AccessWrite
		}
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

func (c *collector) addCall(anchor *sitter.Node, nameNode *sitter.Node, receiver *sitter.Node, form scopeir.CallForm, arity *int) {
	name := baseJavaType(c.text(nameNode))
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
	name := baseJavaType(c.text(nameNode))
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
