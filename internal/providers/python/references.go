package python

import (
	sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

func (c *collector) emitReference(node *sitter.Node) {
	switch node.Kind() {
	case "call":
		fn := callFunction(node)
		if fn == nil {
			return
		}
		arity := countArguments(directChildOfKind(node, "argument_list"))
		if fn.Kind() == "attribute" {
			property := lastIdentifierLikeChild(fn)
			receiver := attributeReceiver(fn)
			if property != nil {
				c.addCall(node, property, receiver, scopeir.CallMember, arity)
			}
			return
		}
		if fn.Kind() == "identifier" {
			form := scopeir.CallFree
			name := c.text(fn)
			if name != "" && name[0] >= 'A' && name[0] <= 'Z' {
				form = scopeir.CallConstructor
			}
			c.addCall(node, fn, nil, form, arity)
		}
	case "attribute":
		if isCallFunctionAttribute(node) {
			return
		}
		property := lastIdentifierLikeChild(node)
		receiver := attributeReceiver(node)
		if property != nil {
			c.addAccess(node, property, receiver, attributeAccessKind(node))
		}
	}
}

func (c *collector) emitClassHeritage(node *sitter.Node) {
	args := directChildOfKind(node, "argument_list")
	if args == nil {
		return
	}
	for index := uint(0); index < args.NamedChildCount(); index++ {
		base := args.NamedChild(index)
		if base == nil {
			continue
		}
		switch base.Kind() {
		case "identifier", "dotted_name", "attribute":
			c.addHeritage(base, base, scopeir.HeritageExtends)
		}
	}
}

func callFunction(node *sitter.Node) *sitter.Node {
	if node == nil || node.Kind() != "call" || node.NamedChildCount() == 0 {
		return nil
	}
	return node.NamedChild(0)
}

func attributeReceiver(node *sitter.Node) *sitter.Node {
	if node == nil || node.Kind() != "attribute" || node.NamedChildCount() < 2 {
		return nil
	}
	return node.NamedChild(0)
}

func isCallFunctionAttribute(node *sitter.Node) bool {
	parent := node.Parent()
	if parent == nil || parent.Kind() != "call" {
		return false
	}
	fn := callFunction(parent)
	return fn != nil && fn.Id() == node.Id()
}

func attributeAccessKind(node *sitter.Node) scopeir.AccessKind {
	parent := node.Parent()
	for parent != nil && parent.Kind() == "expression_statement" {
		parent = parent.Parent()
	}
	if parent == nil || parent.Kind() != "assignment" {
		return scopeir.AccessRead
	}
	left := firstAssignmentTarget(parent)
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
	name := c.text(nameNode)
	if form == scopeir.CallConstructor {
		name = basePythonType(name)
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
	name := basePythonType(c.text(nameNode))
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
