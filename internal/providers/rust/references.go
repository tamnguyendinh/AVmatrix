package rust

import (
	sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

func (c *collector) emitReference(node *sitter.Node) {
	switch node.Kind() {
	case "call_expression":
		fn := child(node, "function")
		if fn == nil {
			fn = node.NamedChild(0)
		}
		if fn == nil {
			return
		}
		arity := countArguments(child(node, "arguments"))
		if fn.Kind() == "field_expression" {
			if property := child(fn, "field"); property != nil {
				c.addCall(node, property, child(fn, "value"), scopeir.CallMember, arity)
			}
			return
		}
		if name := lastIdentifierLikeChild(fn); name != nil {
			c.addCall(node, name, nil, scopeir.CallFree, arity)
		}
	case "field_expression":
		if isCallFunctionField(node) {
			return
		}
		property := child(node, "field")
		receiver := child(node, "value")
		if property != nil {
			c.addAccess(node, property, receiver, fieldAccessKind(node))
		}
	case "struct_expression":
		typeNode := child(node, "type")
		if typeNode == nil {
			typeNode = firstIdentifierLikeChild(node)
		}
		if typeNode != nil {
			c.addCall(node, typeNode, nil, scopeir.CallConstructor, countArguments(node))
		}
	case "impl_item":
		traitName := c.implTraitName(node)
		ownerName := c.implOwnerTypeName(node)
		if traitName == "" || ownerName == "" {
			return
		}
		ownerScope := c.typeScopeIDsByName[ownerName]
		if ownerScope == "" {
			return
		}
		c.addHeritageAtScope(node, traitName, scopeir.HeritageTraitImpl, ownerScope)
	}
}

func isCallFunctionField(node *sitter.Node) bool {
	parent := node.Parent()
	if parent == nil || parent.Kind() != "call_expression" {
		return false
	}
	fn := child(parent, "function")
	return fn != nil && fn.Id() == node.Id()
}

func fieldAccessKind(node *sitter.Node) scopeir.AccessKind {
	parent := node.Parent()
	if parent == nil || parent.Kind() != "assignment_expression" {
		return scopeir.AccessRead
	}
	left := child(parent, "left")
	if left == nil {
		left = parent.NamedChild(0)
	}
	if left != nil && containsNode(left, node) {
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

func (c *collector) addCall(anchor *sitter.Node, nameNode *sitter.Node, receiver *sitter.Node, form scopeir.CallForm, arity *int) {
	name := baseRustType(c.text(nameNode))
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

func (c *collector) addHeritageAtScope(anchor *sitter.Node, name string, kind scopeir.HeritageKind, scopeID string) {
	name = baseRustType(name)
	if name == "" || scopeID == "" {
		return
	}
	c.heritage = append(c.heritage, scopeir.HeritageFact{
		FilePath: c.filePath,
		FileHash: c.fileHash,
		Name:     name,
		Kind:     kind,
		Range:    nodeRange(anchor),
		InScope:  scopeID,
	})
}
