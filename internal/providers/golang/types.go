package golang

import (
	sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

func (c *collector) emitTypeBinding(node *sitter.Node) {
	c.emitTypeBindingKind(node, node.Kind())
}

func (c *collector) emitTypeBindingKind(node *sitter.Node, kind string) {
	switch kind {
	case "parameter_declaration":
		if isReceiverParameter(node) {
			return
		}
		typeNode := child(node, "type")
		if typeNode == nil {
			return
		}
		typeName := normalizeGoType(c.text(typeNode))
		for _, nameNode := range namedChildrenOfType(node, "identifier") {
			c.addTypeBinding(typeNode, c.text(nameNode), typeName, scopeir.TypeSourceParameter)
		}
		c.emitTypeReferences(typeNode, scopeir.TypeSourceParameter)
	case "field_declaration":
		typeNode := child(node, "type")
		if typeNode == nil {
			typeNode = firstIdentifierLikeChild(node)
		}
		if typeNode == nil {
			return
		}
		typeName := normalizeGoType(c.text(typeNode))
		for _, nameNode := range namedChildrenOfType(node, "field_identifier") {
			c.addTypeBinding(typeNode, c.text(nameNode), typeName, scopeir.TypeSourceAnnotation)
		}
		c.emitTypeReferences(typeNode, scopeir.TypeSourceAnnotation)
	case "var_spec", "const_spec":
		c.emitValueSpecTypeBindings(node)
	case "short_var_declaration":
		c.emitShortVarTypeBindings(node)
	case "type_spec", "type_alias":
		if typeNode := child(node, "type"); typeNode != nil {
			switch typeNode.Kind() {
			case "struct_type", "interface_type":
				return
			default:
				c.emitTypeReferences(typeNode, scopeir.TypeSourceAnnotation)
			}
		}
	case "type_elem":
		if typeNode := firstIdentifierLikeChild(node); typeNode != nil {
			c.emitTypeReferences(typeNode, scopeir.TypeSourceAnnotation)
		}
	default:
		if kind == "function_declaration" || kind == "method_declaration" || kind == "method_elem" {
			if result := child(node, "result"); result != nil {
				c.emitTypeReferences(result, scopeir.TypeSourceReturn)
			}
		}
	}
}

func (c *collector) emitValueSpecTypeBindings(node *sitter.Node) {
	typeNode := child(node, "type")
	names := namedChildrenOfType(node, "identifier")
	if typeNode != nil {
		typeName := normalizeGoType(c.text(typeNode))
		for _, nameNode := range names {
			c.addTypeBinding(typeNode, c.text(nameNode), typeName, scopeir.TypeSourceAnnotation)
		}
		c.emitTypeReferences(typeNode, scopeir.TypeSourceAnnotation)
		return
	}

	values := namedValueChildren(child(node, "value"))
	for index, nameNode := range names {
		if index >= len(values) {
			continue
		}
		if inferred := c.inferredTypeFromValue(values[index]); inferred != "" {
			c.addTypeBinding(values[index], c.text(nameNode), inferred, scopeir.TypeSourceAssignment)
		}
	}
}

func (c *collector) emitShortVarTypeBindings(node *sitter.Node) {
	left := child(node, "left")
	if left == nil {
		left = node.NamedChild(0)
	}
	right := child(node, "right")
	names := namedChildrenOfType(left, "identifier")
	values := namedValueChildren(right)
	for index, nameNode := range names {
		if index >= len(values) {
			continue
		}
		if inferred := c.inferredTypeFromValue(values[index]); inferred != "" {
			c.addTypeBinding(values[index], c.text(nameNode), inferred, scopeir.TypeSourceAssignment)
		}
	}
}

func (c *collector) addTypeBinding(anchor *sitter.Node, name string, rawName string, source scopeir.TypeRefSource) {
	rawName = normalizeGoType(rawName)
	if name == "" || name == "_" || rawName == "" {
		return
	}
	rng := nodeRange(anchor)
	scopeID := c.innermostScopeID(rng)
	ref := scopeir.TypeRef{RawName: rawName, DeclaredAtScope: scopeID, Source: source}
	if scope := c.scopeByID(scopeID); scope != nil {
		scope.TypeBindings = append(scope.TypeBindings, scopeir.TypeBindingFact{Name: name, Type: ref})
	}
	c.typeAnnotations = append(c.typeAnnotations, scopeir.TypeAnnotationFact{
		FilePath: c.filePath,
		FileHash: c.fileHash,
		Name:     name,
		Range:    rng,
		InScope:  scopeID,
		Type:     ref,
	})
}

func (c *collector) emitTypeReferences(typeNode *sitter.Node, source scopeir.TypeRefSource) {
	walkKind(typeNode, func(candidate *sitter.Node, kind string) {
		if candidate == nil {
			return
		}
		if kind == "type_identifier" {
			parent := candidate.Parent()
			if parent != nil && parent.Kind() == "qualified_type" {
				return
			}
		}
		if kind != "type_identifier" && kind != "qualified_type" {
			return
		}
		name := normalizeGoType(c.text(candidate))
		base := baseGoType(name)
		if base == "" {
			return
		}
		if _, ok := goBuiltinTypeNames[base]; ok {
			return
		}
		rng := nodeRange(candidate)
		scopeID := c.innermostScopeID(rng)
		c.typeAnnotations = append(c.typeAnnotations, scopeir.TypeAnnotationFact{
			FilePath: c.filePath,
			FileHash: c.fileHash,
			Name:     base,
			Range:    rng,
			InScope:  scopeID,
			Type: scopeir.TypeRef{
				RawName:         name,
				DeclaredAtScope: scopeID,
				Source:          source,
			},
		})
	})
}

func (c *collector) inferredTypeFromValue(node *sitter.Node) string {
	if node == nil {
		return ""
	}
	switch node.Kind() {
	case "composite_literal":
		return normalizeGoType(c.text(child(node, "type")))
	case "unary_expression":
		if operand := child(node, "operand"); operand != nil {
			return c.inferredTypeFromValue(operand)
		}
	case "call_expression":
		fn := child(node, "function")
		if fn != nil && fn.Kind() == "identifier" {
			return c.returnTypesByCallableName[c.text(fn)]
		}
	}
	return ""
}

func isReceiverParameter(node *sitter.Node) bool {
	parent := node.Parent()
	if parent == nil || parent.Kind() != "parameter_list" {
		return false
	}
	grandparent := parent.Parent()
	if grandparent == nil || grandparent.Kind() != "method_declaration" {
		return false
	}
	receiver := child(grandparent, "receiver")
	return receiver != nil && receiver.Id() == parent.Id()
}
