package swift

import (
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

func (c *collector) emitReference(node *sitter.Node) {
	switch node.Kind() {
	case "call_expression":
		c.emitCall(node)
	case "assignment":
		c.emitAssignmentAccess(node)
	case "inheritance_specifier":
		c.emitHeritage(node)
	}
}

func (c *collector) emitCall(node *sitter.Node) {
	nameNode := callName(node)
	if nameNode == nil {
		return
	}
	receiver := callReceiver(c, node)
	form := scopeir.CallFree
	if receiver != "" {
		form = scopeir.CallMember
	} else if startsWithUpper(c.text(nameNode)) {
		form = scopeir.CallConstructor
	} else if c.ownerTypeNameFor(node) != "" {
		form = scopeir.CallMember
	}
	c.addCall(node, nameNode, receiver, form, callArguments(node))
}

func callName(node *sitter.Node) *sitter.Node {
	if node == nil || node.Kind() != "call_expression" {
		return nil
	}
	if nav := directChildOfKind(node, "navigation_expression"); nav != nil {
		if suffix := lastNamedChildOfType(nav, "navigation_suffix"); suffix != nil {
			return lastIdentifierLikeChild(suffix)
		}
	}
	return firstIdentifierLikeChild(node)
}

func callReceiver(c *collector, node *sitter.Node) string {
	nav := directChildOfKind(node, "navigation_expression")
	if nav == nil {
		return ""
	}
	children := namedChildren(nav)
	if len(children) == 0 {
		return ""
	}
	if children[0].Kind() == "self_expression" {
		return "self"
	}
	if isIdentifierLike(children[0]) {
		return c.text(children[0])
	}
	return ""
}

func callArguments(node *sitter.Node) *int {
	suffix := directChildOfKind(node, "call_suffix")
	args := directChildOfKind(suffix, "value_arguments")
	if args == nil {
		return nil
	}
	count := 0
	for _, arg := range directChildrenOfKind(args, "value_argument") {
		if arg != nil {
			count++
		}
	}
	return &count
}

func (c *collector) emitAssignmentAccess(node *sitter.Node) {
	left := firstNamedChildOfType(node, "directly_assignable_expression")
	nav := firstNamedChildOfType(left, "navigation_expression")
	if nav == nil {
		return
	}
	nameNode := lastDescendantOfType(nav, "simple_identifier")
	if nameNode == nil {
		return
	}
	receiver := ""
	children := namedChildren(nav)
	if len(children) > 0 {
		switch {
		case children[0].Kind() == "self_expression":
			receiver = "self"
		case isIdentifierLike(children[0]):
			receiver = c.text(children[0])
		}
	}
	c.addAccess(nav, nameNode, receiver, scopeir.AccessWrite)
}

func (c *collector) emitHeritage(node *sitter.Node) {
	typeNode := firstDescendantOfType(node, "type_identifier")
	if typeNode == nil {
		return
	}
	kind := scopeir.HeritageExtends
	if c.typeLabelsByName[c.text(typeNode)] == scopeir.NodeInterface {
		kind = scopeir.HeritageImplements
	}
	c.addHeritage(node, typeNode, kind)
}

func (c *collector) addCall(anchor *sitter.Node, nameNode *sitter.Node, receiver string, form scopeir.CallForm, arity *int) {
	name := baseSwiftType(c.text(nameNode))
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
	if receiver != "" {
		fact.ExplicitReceiver = strings.TrimPrefix(receiver, "self.")
	}
	c.calls = append(c.calls, fact)
}

func (c *collector) addAccess(anchor *sitter.Node, nameNode *sitter.Node, receiver string, kind scopeir.AccessKind) {
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
	if receiver != "" {
		fact.ExplicitReceiver = strings.TrimPrefix(receiver, "self.")
	}
	c.accesses = append(c.accesses, fact)
}

func (c *collector) addHeritage(anchor *sitter.Node, nameNode *sitter.Node, kind scopeir.HeritageKind) {
	name := baseSwiftType(c.text(nameNode))
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
