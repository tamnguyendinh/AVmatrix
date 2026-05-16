package ruby

import (
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

func (c *collector) emitReference(node *sitter.Node) {
	switch node.Kind() {
	case "call":
		c.emitCall(node)
		c.emitMixinHeritage(node)
	case "assignment":
		c.emitAssignmentAccess(node)
	case "superclass":
		if typeNode := firstDescendantOfType(node, "constant"); typeNode != nil {
			c.addHeritage(node, typeNode, scopeir.HeritageExtends)
		}
	}
}

func (c *collector) emitCall(node *sitter.Node) {
	name := callName(c, node)
	if name == "" || isRubyNonCallName(name) {
		return
	}
	nameNode := callNameNode(node)
	if nameNode == nil {
		return
	}
	receiver := callReceiver(c, node)
	form := scopeir.CallFree
	if receiver != "" || c.ownerTypeNameFor(node) != "" {
		form = scopeir.CallMember
	}
	c.addCall(node, nameNode, receiver, form, callArguments(node))
}

func isRubyNonCallName(name string) bool {
	switch name {
	case "require", "require_relative", "include", "extend", "prepend", "attr_reader", "attr_accessor", "attr_writer":
		return true
	default:
		return false
	}
}

func (c *collector) emitMixinHeritage(node *sitter.Node) {
	name := callName(c, node)
	var kind scopeir.HeritageKind
	switch name {
	case "include":
		kind = scopeir.HeritageInclude
	case "extend":
		kind = scopeir.HeritageExtend
	case "prepend":
		kind = scopeir.HeritagePrepend
	default:
		return
	}
	if typeNode := firstDescendantOfType(directChildOfKind(node, "argument_list"), "constant"); typeNode != nil {
		c.addHeritage(node, typeNode, kind)
	}
}

func (c *collector) emitAssignmentAccess(node *sitter.Node) {
	left := firstNamedChild(node)
	if left == nil || left.Kind() != "instance_variable" {
		return
	}
	c.addAccess(left, strings.TrimPrefix(c.text(left), "@"), "self", scopeir.AccessWrite)
}

func (c *collector) addCall(anchor *sitter.Node, nameNode *sitter.Node, receiver string, form scopeir.CallForm, arity *int) {
	name := c.text(nameNode)
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
		fact.ExplicitReceiver = strings.TrimPrefix(receiver, "@")
	}
	c.calls = append(c.calls, fact)
}

func (c *collector) addAccess(anchor *sitter.Node, name string, receiver string, kind scopeir.AccessKind) {
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
		fact.ExplicitReceiver = receiver
	}
	c.accesses = append(c.accesses, fact)
}

func (c *collector) addHeritage(anchor *sitter.Node, nameNode *sitter.Node, kind scopeir.HeritageKind) {
	name := c.text(nameNode)
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

func callName(c *collector, node *sitter.Node) string {
	if nameNode := callNameNode(node); nameNode != nil {
		return c.text(nameNode)
	}
	return ""
}

func callNameNode(node *sitter.Node) *sitter.Node {
	if node == nil || node.Kind() != "call" {
		return nil
	}
	children := namedChildren(node)
	if len(children) == 0 {
		return nil
	}
	for index := len(children) - 1; index >= 0; index-- {
		child := children[index]
		if child.Kind() == "identifier" && (index == len(children)-1 || children[index+1].Kind() == "argument_list") {
			return child
		}
	}
	return nil
}

func callReceiver(c *collector, node *sitter.Node) string {
	children := namedChildren(node)
	if len(children) < 2 {
		return ""
	}
	if children[1].Kind() != "identifier" {
		return ""
	}
	if children[0].Kind() == "identifier" || children[0].Kind() == "instance_variable" || children[0].Kind() == "constant" {
		return c.text(children[0])
	}
	return ""
}

func callArguments(node *sitter.Node) *int {
	args := directChildOfKind(node, "argument_list")
	if args == nil {
		return nil
	}
	count := 0
	for index := uint(0); index < args.NamedChildCount(); index++ {
		child := args.NamedChild(index)
		if child != nil {
			count++
		}
	}
	return &count
}
