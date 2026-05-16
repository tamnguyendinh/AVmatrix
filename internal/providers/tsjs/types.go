package tsjs

import (
	sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

func (c *collector) buildContext(root *sitter.Node) {
	walkKind(root, func(node *sitter.Node, kind string) {
		c.collectContextForKind(node, kind)
	})
}

func (c *collector) collectContextForKind(node *sitter.Node, kind string) {
	if isFunctionScopeKind(kind) {
		returnType := returnTypeNameForCallable(c, node)
		name := c.text(child(node, "name"))
		if name != "" && returnType != "" {
			c.returnTypesByCallableName[name] = returnType
		}
	}
	if kind == "variable_declarator" {
		nameNode := child(node, "name")
		value := child(node, "value")
		returnType := returnTypeNameForCallable(c, value)
		if nameNode != nil && nameNode.Kind() == "identifier" && isFunctionExpression(value) && returnType != "" {
			c.returnTypesByCallableName[c.text(nameNode)] = returnType
		}
	}
	if kind == "import_statement" {
		c.collectImportedLocalNames(node)
	}
}

func (c *collector) collectImportedLocalNames(node *sitter.Node) {
	importClause := firstNamedChildOfType(node, "import_clause")
	if importClause == nil {
		return
	}
	defaultName := c.text(firstNamedChildOfType(importClause, "identifier"))
	if defaultName != "" {
		c.importedLocalNames[defaultName] = struct{}{}
	}
	for _, specifier := range descendantsOfType(importClause, "import_specifier") {
		names := namedIdentifierChildren(specifier)
		imported := c.text(child(specifier, "name"))
		if imported == "" && len(names) > 0 {
			imported = c.text(names[0])
		}
		if imported == "" {
			continue
		}
		localName := imported
		if len(names) > 1 {
			localName = c.text(names[len(names)-1])
		}
		c.importedLocalNames[localName] = struct{}{}
	}
}

func (c *collector) emitTypeBinding(node *sitter.Node) {
	c.emitTypeBindingKind(node, node.Kind())
}

func (c *collector) emitTypeBindingKind(node *sitter.Node, kind string) {
	switch kind {
	case "required_parameter", "optional_parameter":
		nameNode := child(node, "pattern")
		if nameNode == nil {
			nameNode = firstIdentifierChild(node)
		}
		typeNode := child(node, "type")
		if nameNode != nil && typeNode != nil {
			c.addTypeBinding(typeNode, c.text(nameNode), stripTypeAnnotation(c.text(typeNode)), scopeir.TypeSourceParameter)
			c.emitTypeReferences(typeNode, scopeir.TypeSourceParameter)
		}
	case "public_field_definition", "property_signature":
		nameNode := child(node, "name")
		typeNode := child(node, "type")
		if nameNode != nil && typeNode != nil {
			c.addTypeBinding(typeNode, c.text(nameNode), stripTypeAnnotation(c.text(typeNode)), scopeir.TypeSourceAnnotation)
			c.emitTypeReferences(typeNode, scopeir.TypeSourceAnnotation)
		}
	case "variable_declarator":
		c.emitVariableTypeBinding(node)
	case "type_alias_declaration":
		if valueNode := child(node, "value"); valueNode != nil {
			c.emitTypeReferences(valueNode, scopeir.TypeSourceAnnotation)
		}
	default:
		if isFunctionScopeKind(kind) {
			if returnTypeNode := child(node, "return_type"); returnTypeNode != nil {
				c.emitTypeReferences(returnTypeNode, scopeir.TypeSourceReturn)
			}
		}
	}
}

func (c *collector) emitVariableTypeBinding(node *sitter.Node) {
	nameNode := child(node, "name")
	if nameNode == nil || nameNode.Kind() != "identifier" {
		return
	}
	name := c.text(nameNode)
	typeNode := child(node, "type")
	if typeNode != nil {
		c.addTypeBinding(typeNode, name, stripTypeAnnotation(c.text(typeNode)), scopeir.TypeSourceAnnotation)
		c.emitTypeReferences(typeNode, scopeir.TypeSourceAnnotation)
		return
	}

	value := child(node, "value")
	if ctorName := c.constructorNameFromValue(value); ctorName != "" {
		c.addTypeBinding(node, name, ctorName, scopeir.TypeSourceConstructor)
		return
	}
	if returnType := c.returnTypeNameFromCallValue(value); returnType != "" {
		c.addTypeBinding(node, name, returnType, scopeir.TypeSourceReturn)
		return
	}
	if callName := c.callNameFromCallValue(value); callName != "" {
		if _, ok := c.importedLocalNames[callName]; ok {
			c.addTypeBinding(node, name, callName, scopeir.TypeSourceCallReturn)
			return
		}
	}
	if methodReturn := c.memberMethodNameFromCallValue(value); methodReturn != "" {
		c.addTypeBinding(node, name, methodReturn, scopeir.TypeSourceMethodReturn)
		return
	}
	if fieldAccess := c.memberFieldNameFromValue(value); fieldAccess != "" {
		c.addTypeBinding(node, name, fieldAccess, scopeir.TypeSourceFieldAccess)
		return
	}
	if propagated := c.receiverNameFromCopyValue(value); propagated != "" && propagated != name {
		c.addTypeBinding(node, name, propagated, scopeir.TypeSourceReceiver)
	}
}

func (c *collector) addTypeBinding(anchor *sitter.Node, name string, rawName string, source scopeir.TypeRefSource) {
	if name == "" || rawName == "" {
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
	walk(typeNode, func(candidate *sitter.Node) {
		if !isIdentifierLike(candidate) {
			return
		}
		name := c.text(candidate)
		if _, ok := builtinTypeNames[name]; ok {
			return
		}
		rng := nodeRange(candidate)
		scopeID := c.innermostScopeID(rng)
		c.typeAnnotations = append(c.typeAnnotations, scopeir.TypeAnnotationFact{
			FilePath: c.filePath,
			FileHash: c.fileHash,
			Name:     name,
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

func (c *collector) constructorNameFromValue(node *sitter.Node) string {
	value := unwrapExpression(node)
	if value == nil || value.Kind() != "new_expression" {
		return ""
	}
	return c.text(child(value, "constructor"))
}

func (c *collector) returnTypeNameFromCallValue(value *sitter.Node) string {
	expression := callExpressionFromValue(value)
	if expression == nil {
		return ""
	}
	fn := unwrapAwaitExpression(child(expression, "function"))
	if fn == nil {
		fn = expression
	}
	if fn.Kind() == "identifier" {
		return c.returnTypesByCallableName[c.text(fn)]
	}
	return ""
}

func (c *collector) callNameFromCallValue(value *sitter.Node) string {
	expression := callExpressionFromValue(value)
	if expression == nil {
		return ""
	}
	fn := unwrapAwaitExpression(child(expression, "function"))
	if fn == nil {
		fn = expression
	}
	if fn.Kind() == "identifier" {
		return c.text(fn)
	}
	return ""
}

func callExpressionFromValue(value *sitter.Node) *sitter.Node {
	expression := unwrapAwaitExpression(unwrapExpression(value))
	if expression != nil && expression.Kind() == "call_expression" {
		return expression
	}
	return nil
}

func (c *collector) memberMethodNameFromCallValue(value *sitter.Node) string {
	expression := callExpressionFromValue(value)
	if expression == nil {
		return ""
	}
	fn := unwrapAwaitExpression(child(expression, "function"))
	if fn == nil || fn.Kind() != "member_expression" {
		return ""
	}
	receiver := child(fn, "object")
	property := child(fn, "property")
	if receiver == nil || property == nil {
		return ""
	}
	if receiver.Kind() != "identifier" && receiver.Kind() != "this" {
		return ""
	}
	if property.Kind() != "property_identifier" {
		return ""
	}
	return c.text(receiver) + "." + c.text(property)
}

func (c *collector) memberFieldNameFromValue(value *sitter.Node) string {
	expression := unwrapExpression(value)
	if expression == nil || expression.Kind() != "member_expression" {
		return ""
	}
	receiver := child(expression, "object")
	property := child(expression, "property")
	if receiver == nil || property == nil {
		return ""
	}
	if receiver.Kind() != "identifier" && receiver.Kind() != "this" {
		return ""
	}
	if property.Kind() != "property_identifier" {
		return ""
	}
	return c.text(receiver) + "." + c.text(property)
}

func (c *collector) receiverNameFromCopyValue(value *sitter.Node) string {
	expression := unwrapExpression(value)
	if expression == nil {
		return ""
	}
	if expression.Kind() == "identifier" || expression.Kind() == "this" {
		return c.text(expression)
	}
	return ""
}
