package dart

import (
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

func (c *collector) emitReference(node *sitter.Node) {
	switch node.Kind() {
	case "initialized_variable_definition":
		c.emitInitializedValueReference(node)
	case "expression_statement", "return_statement":
		c.emitSelectorExpression(node)
	case "superclass":
		if typeNode := firstNamedChildOfType(node, "type_identifier"); typeNode != nil {
			c.addHeritage(typeNode, typeNode, scopeir.HeritageExtends)
		}
	case "interfaces":
		for _, typeNode := range directChildrenOfKind(node, "type_identifier") {
			c.addHeritage(typeNode, typeNode, scopeir.HeritageImplements)
		}
	}
}

func (c *collector) emitInitializedValueReference(node *sitter.Node) {
	values := valueNodesAfterName(node)
	if len(values) == 0 {
		return
	}
	if len(values) >= 2 && isIdentifierLike(values[0]) && selectorIsCall(values[1]) {
		c.addCall(node, values[0], "", c.defaultCallForm(values[0]), selectorArguments(values[1]))
	}
}

func (c *collector) emitSelectorExpression(node *sitter.Node) {
	children := namedChildren(node)
	if len(children) == 0 || !isIdentifierLike(children[0]) {
		return
	}
	base := children[0]
	receiver := c.text(base)
	currentReceiver := receiver
	for index := 1; index < len(children); index++ {
		selector := children[index]
		if selector.Kind() != "selector" {
			continue
		}
		nameNode := selectorName(selector)
		if nameNode == nil {
			if selectorIsCall(selector) {
				c.addCall(selector, base, "", c.defaultCallForm(base), selectorArguments(selector))
			}
			continue
		}
		if index+1 < len(children) && selectorIsCall(children[index+1]) {
			callReceiver := currentReceiver
			if index == 1 && c.hasTypeBinding(callReceiver, nodeRange(node)) {
				c.addAccess(base, base, "", scopeir.AccessRead)
			}
			if c.ownerTypeNameFor(node) != "" && callReceiver == c.text(base) && !c.hasTypeBinding(callReceiver, nodeRange(node)) {
				callReceiver = ""
			}
			c.addCall(selector, nameNode, callReceiver, scopeir.CallMember, selectorArguments(children[index+1]))
			if currentReceiver == "" {
				currentReceiver = c.text(nameNode)
			} else {
				currentReceiver += "." + c.text(nameNode)
			}
			index++
			continue
		}
		accessReceiver := currentReceiver
		if accessReceiver == c.text(nameNode) {
			accessReceiver = ""
		}
		c.addAccess(selector, nameNode, accessReceiver, scopeir.AccessRead)
		if currentReceiver == "" {
			currentReceiver = c.text(nameNode)
		} else {
			currentReceiver += "." + c.text(nameNode)
		}
	}
	if len(children) >= 2 && selectorIsCall(children[1]) {
		c.addCall(children[1], base, "", c.defaultCallForm(base), selectorArguments(children[1]))
	}
}

func namedChildren(node *sitter.Node) []*sitter.Node {
	if node == nil {
		return nil
	}
	out := make([]*sitter.Node, 0, node.NamedChildCount())
	for index := uint(0); index < node.NamedChildCount(); index++ {
		if child := node.NamedChild(index); child != nil {
			out = append(out, child)
		}
	}
	return out
}

func selectorName(node *sitter.Node) *sitter.Node {
	if node == nil || node.Kind() != "selector" {
		return nil
	}
	for index := uint(0); index < node.NamedChildCount(); index++ {
		candidate := node.NamedChild(index)
		if candidate == nil || candidate.Kind() == "argument_part" {
			continue
		}
		if name := lastIdentifierLikeChild(candidate); name != nil {
			return name
		}
	}
	return nil
}

func selectorIsCall(node *sitter.Node) bool {
	return node != nil && node.Kind() == "selector" && directChildOfKind(node, "argument_part") != nil
}

func selectorArguments(node *sitter.Node) *int {
	if node == nil {
		return nil
	}
	args := directChildOfKind(directChildOfKind(node, "argument_part"), "arguments")
	if args == nil {
		return nil
	}
	count := 0
	for index := uint(0); index < args.NamedChildCount(); index++ {
		if child := args.NamedChild(index); child != nil && child.Kind() == "argument" {
			count++
		}
	}
	return &count
}

func (c *collector) defaultCallForm(nameNode *sitter.Node) scopeir.CallForm {
	name := baseDartType(c.text(nameNode))
	if startsWithUpper(name) {
		return scopeir.CallConstructor
	}
	if c.ownerTypeNameFor(nameNode) != "" {
		return scopeir.CallMember
	}
	return scopeir.CallFree
}

func (c *collector) hasTypeBinding(name string, rng scopeir.Range) bool {
	if name == "" {
		return false
	}
	for scopeID := c.innermostScopeID(rng); scopeID != ""; {
		scope := c.scopeByID(scopeID)
		if scope == nil {
			return false
		}
		for _, binding := range scope.TypeBindings {
			if binding.Name == name {
				return true
			}
		}
		if scope.Parent == nil {
			return false
		}
		scopeID = *scope.Parent
	}
	return false
}

func (c *collector) addCall(anchor *sitter.Node, nameNode *sitter.Node, receiver string, form scopeir.CallForm, arity *int) {
	name := baseDartType(c.text(nameNode))
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
		fact.ExplicitReceiver = strings.TrimPrefix(receiver, "this.")
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
		fact.ExplicitReceiver = strings.TrimPrefix(receiver, "this.")
	}
	c.accesses = append(c.accesses, fact)
}

func (c *collector) addHeritage(anchor *sitter.Node, nameNode *sitter.Node, kind scopeir.HeritageKind) {
	name := baseDartType(c.text(nameNode))
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
