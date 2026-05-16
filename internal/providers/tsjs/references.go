package tsjs

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
		member := unwrapAwaitExpression(fn)
		if member != nil && member.Kind() == "member_expression" {
			property := child(member, "property")
			receiver := child(member, "object")
			if property != nil {
				c.addCall(node, property, receiver, scopeir.CallMember, arity)
			}
			return
		}
		if member != nil && member.Kind() == "identifier" {
			c.addCall(node, member, nil, scopeir.CallFree, arity)
		}
	case "member_expression":
		if isCallFunctionMember(node) {
			return
		}
		property := child(node, "property")
		receiver := child(node, "object")
		if property != nil {
			c.addAccess(node, property, receiver, memberAccessKind(node))
		}
	case "new_expression":
		ctor := child(node, "constructor")
		if ctor != nil {
			c.addCall(node, ctor, nil, scopeir.CallConstructor, countArguments(child(node, "arguments")))
		}
	case "extends_clause":
		value := child(node, "value")
		if value == nil {
			value = firstIdentifierLikeChild(node)
		}
		if value != nil {
			c.addHeritage(node, value, scopeir.HeritageExtends)
		}
	case "implements_clause":
		for _, ident := range namedIdentifierChildren(node) {
			c.addHeritage(ident, ident, scopeir.HeritageImplements)
		}
	}
}

func (c *collector) addCall(anchor *sitter.Node, nameNode *sitter.Node, receiver *sitter.Node, form scopeir.CallForm, arity *int) {
	rng := nodeRange(anchor)
	fact := scopeir.CallSiteFact{
		FilePath: c.filePath,
		FileHash: c.fileHash,
		Name:     c.text(nameNode),
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
	rng := nodeRange(anchor)
	fact := scopeir.AccessFact{
		FilePath: c.filePath,
		FileHash: c.fileHash,
		Name:     c.text(nameNode),
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
	rng := nodeRange(anchor)
	c.heritage = append(c.heritage, scopeir.HeritageFact{
		FilePath: c.filePath,
		FileHash: c.fileHash,
		Name:     c.text(nameNode),
		Kind:     kind,
		Range:    rng,
		InScope:  c.innermostScopeID(rng),
	})
}

func isCallFunctionMember(node *sitter.Node) bool {
	parent := node.Parent()
	if parent == nil || parent.Kind() != "call_expression" {
		return false
	}
	fn := child(parent, "function")
	return fn != nil && fn.Id() == node.Id()
}

func memberAccessKind(node *sitter.Node) scopeir.AccessKind {
	parent := node.Parent()
	if parent == nil {
		return scopeir.AccessRead
	}
	switch parent.Kind() {
	case "assignment_expression", "augmented_assignment_expression":
		left := child(parent, "left")
		if left == nil {
			left = parent.NamedChild(0)
		}
		if left != nil && left.Id() == node.Id() {
			return scopeir.AccessWrite
		}
	case "update_expression":
		return scopeir.AccessWrite
	}
	return scopeir.AccessRead
}
