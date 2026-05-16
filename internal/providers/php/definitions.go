package php

import (
	sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

func (c *collector) buildContext(root *sitter.Node) {
	walk(root, func(node *sitter.Node) {
		switch node.Kind() {
		case "class_declaration", "interface_declaration":
			nameNode := declarationName(node)
			if nameNode == nil {
				return
			}
			name := c.text(nameNode)
			c.typeDefIDsByName[name] = defID(c.filePath, nodeRange(node), phpDeclarationLabel(node), name)
		case "function_definition", "method_declaration":
			if nameNode := declarationName(node); nameNode != nil {
				if returnType := returnTypeNameForCallable(c, node); returnType != "" {
					c.returnTypesByCallable[c.text(nameNode)] = returnType
				}
			}
		}
	})
}

func (c *collector) emitDefinition(node *sitter.Node) {
	switch node.Kind() {
	case "namespace_definition":
		c.addDefinition(node, scopeir.NodePackage, child(node, "name"), "", "", "", "")
	case "class_declaration", "interface_declaration":
		c.addDefinition(node, phpDeclarationLabel(node), declarationName(node), "", "", "", "")
	case "function_definition":
		c.emitCallableDefinition(node, scopeir.NodeFunction, declarationName(node))
	case "method_declaration":
		label := scopeir.NodeMethod
		if name := c.text(declarationName(node)); name == "__construct" {
			label = scopeir.NodeConstructor
		}
		c.emitCallableDefinition(node, label, declarationName(node))
	case "property_declaration":
		c.emitPropertyDefinition(node)
	case "assignment_expression":
		c.emitLocalAssignmentDefinition(node)
	}
}

func (c *collector) emitCallableDefinition(node *sitter.Node, label scopeir.NodeLabel, nameNode *sitter.Node) {
	ownerName := c.ownerTypeNameFor(node)
	qualified := ""
	if ownerName != "" && nameNode != nil {
		qualified = ownerName + "." + c.text(nameNode)
	}
	paramTypes, count := parameterTypes(c, child(node, "parameters"))
	def := c.addDefinition(node, label, nameNode, c.typeDefIDsByName[ownerName], returnTypeNameForCallable(c, node), "", qualified)
	if def == nil {
		return
	}
	def.ParameterCount = &count
	def.RequiredParameterCount = &count
	def.ParameterTypes = append([]string(nil), paramTypes...)
	if ownerName != "" {
		c.addTypeBinding(node, "this", ownerName, scopeir.TypeSourceSelf)
	}
	if def.ReturnType != "" {
		c.addReturnType(node, def.ID, def.ReturnType)
	}
}

func (c *collector) emitPropertyDefinition(node *sitter.Node) {
	typeName := normalizePHPType(c.text(typeNodeForDeclaration(node)))
	ownerName := c.ownerTypeNameFor(node)
	ownerID := c.typeDefIDsByName[ownerName]
	for _, property := range directChildrenOfKind(node, "property_element") {
		nameNode := declarationName(property)
		qualified := c.text(nameNode)
		if ownerName != "" && qualified != "" {
			qualified = ownerName + "." + qualified
		}
		c.addDefinition(node, scopeir.NodeProperty, nameNode, ownerID, "", typeName, qualified)
	}
}

func (c *collector) emitLocalAssignmentDefinition(node *sitter.Node) {
	left := child(node, "left")
	if left == nil || left.Kind() != "variable_name" {
		return
	}
	nameNode := declarationName(left)
	if nameNode == nil {
		return
	}
	name := c.text(nameNode)
	if name == "this" {
		return
	}
	c.addDefinition(node, scopeir.NodeVariable, nameNode, "", "", c.inferredTypeFromValue(child(node, "right")), "")
}

func (c *collector) addDefinition(
	node *sitter.Node,
	label scopeir.NodeLabel,
	nameNode *sitter.Node,
	ownerID string,
	returnType string,
	declaredType string,
	qualifiedName string,
) *scopeir.DefinitionFact {
	if nameNode == nil {
		return nil
	}
	name := c.text(nameNode)
	if name == "" || name == "_" {
		return nil
	}
	rng := nodeRange(node)
	id := defID(c.filePath, rng, label, name)
	if qualifiedName == "" {
		qualifiedName = name
	}
	fact := scopeir.DefinitionFact{
		ID:            id,
		FilePath:      c.filePath,
		FileHash:      c.fileHash,
		Name:          name,
		Label:         label,
		Range:         rng,
		QualifiedName: qualifiedName,
		ReturnType:    returnType,
		DeclaredType:  declaredType,
		OwnerID:       ownerID,
	}
	c.definitions = append(c.definitions, fact)

	scopeID := c.innermostScopeID(rng)
	if scope := c.scopeByID(scopeID); scope != nil {
		scope.OwnedDefIDs = append(scope.OwnedDefIDs, id)
		scope.Bindings = append(scope.Bindings, scopeir.BindingFact{
			Name:   name,
			DefID:  id,
			Origin: scopeir.BindingLocal,
		})
	}
	return &c.definitions[len(c.definitions)-1]
}

func defID(filePath string, rng scopeir.Range, label scopeir.NodeLabel, name string) string {
	return "def:" + filePath + "#" + intString(rng.StartLine) + ":" + intString(rng.StartCol) + ":" + string(label) + ":" + name
}

func phpDeclarationLabel(node *sitter.Node) scopeir.NodeLabel {
	if node != nil && node.Kind() == "interface_declaration" {
		return scopeir.NodeInterface
	}
	return scopeir.NodeClass
}

func declarationName(node *sitter.Node) *sitter.Node {
	if node == nil {
		return nil
	}
	if node.Kind() == "variable_name" {
		return firstNamedChildOfType(node, "name")
	}
	if name := child(node, "name"); name != nil {
		if name.Kind() == "variable_name" {
			return declarationName(name)
		}
		return name
	}
	switch node.Kind() {
	case "property_element", "simple_parameter":
		return declarationName(firstNamedChildOfType(node, "variable_name"))
	default:
		return firstIdentifierLikeChild(node)
	}
}

func parameterTypes(c *collector, params *sitter.Node) ([]string, int) {
	if params == nil {
		return nil, 0
	}
	var out []string
	count := 0
	for _, param := range directChildrenOfKind(params, "simple_parameter") {
		typeName := normalizePHPType(c.text(typeNodeForDeclaration(param)))
		if typeName != "" {
			out = append(out, typeName)
		}
		count++
	}
	return out, count
}

func returnTypeNameForCallable(c *collector, node *sitter.Node) string {
	if node == nil || c.text(declarationName(node)) == "__construct" {
		return ""
	}
	return normalizePHPType(c.text(child(node, "return_type")))
}

func (c *collector) addReturnType(node *sitter.Node, defID string, returnType string) {
	returnNode := child(node, "return_type")
	if returnNode == nil {
		return
	}
	rng := nodeRange(returnNode)
	scopeID := c.innermostScopeID(rng)
	c.returnTypes = append(c.returnTypes, scopeir.ReturnTypeFact{
		DefID:    defID,
		FilePath: c.filePath,
		FileHash: c.fileHash,
		Range:    rng,
		Type: scopeir.TypeRef{
			RawName:         returnType,
			DeclaredAtScope: scopeID,
			Source:          scopeir.TypeSourceReturn,
		},
	})
}

func typeNodeForDeclaration(node *sitter.Node) *sitter.Node {
	if node == nil {
		return nil
	}
	if typeNode := child(node, "type"); typeNode != nil {
		return typeNode
	}
	if returnNode := child(node, "return_type"); returnNode != nil {
		return returnNode
	}
	for index := uint(0); index < node.NamedChildCount(); index++ {
		candidate := node.NamedChild(index)
		if isPHPTypeNode(candidate) {
			return candidate
		}
	}
	return nil
}

func isPHPTypeNode(node *sitter.Node) bool {
	if node == nil {
		return false
	}
	switch node.Kind() {
	case "named_type", "primitive_type", "qualified_name", "namespace_name", "union_type", "intersection_type", "optional_type":
		return true
	default:
		return false
	}
}

func (c *collector) ownerTypeNameFor(node *sitter.Node) string {
	for current := node.Parent(); current != nil; current = current.Parent() {
		switch current.Kind() {
		case "class_declaration", "interface_declaration":
			return c.text(declarationName(current))
		case "program":
			return ""
		}
	}
	return ""
}
