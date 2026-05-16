package java

import (
	sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

func (c *collector) buildContext(root *sitter.Node) {
	walk(root, func(node *sitter.Node) {
		switch node.Kind() {
		case "class_declaration", "interface_declaration", "enum_declaration", "record_declaration":
			nameNode := declarationName(node)
			if nameNode == nil {
				return
			}
			name := c.text(nameNode)
			c.typeDefIDsByName[name] = defID(c.filePath, nodeRange(node), javaDeclarationLabel(node), name)
		}
	})
}

func (c *collector) emitDefinition(node *sitter.Node) {
	switch node.Kind() {
	case "package_declaration":
		nameNode := lastIdentifierLikeChild(node)
		c.packageName = c.text(firstIdentifierLikeChild(node))
		c.addDefinition(node, scopeir.NodePackage, nameNode, "", "", "", c.packageName)
	case "class_declaration", "interface_declaration", "enum_declaration", "record_declaration":
		c.addDefinition(node, javaDeclarationLabel(node), declarationName(node), "", "", "", c.qualifiedTypeName(node))
	case "method_declaration":
		c.emitCallableDefinition(node, scopeir.NodeMethod, declarationName(node))
	case "constructor_declaration", "compact_constructor_declaration":
		c.emitCallableDefinition(node, scopeir.NodeConstructor, declarationName(node))
	case "field_declaration":
		c.emitFieldDefinitions(node)
	case "local_variable_declaration":
		c.emitLocalVariableDefinitions(node)
	}
}

func (c *collector) qualifiedTypeName(node *sitter.Node) string {
	nameNode := declarationName(node)
	name := c.text(nameNode)
	if name == "" || c.packageName == "" {
		return name
	}
	return c.packageName + "." + name
}

func (c *collector) emitCallableDefinition(node *sitter.Node, label scopeir.NodeLabel, nameNode *sitter.Node) {
	ownerName := c.ownerTypeNameFor(node)
	qualified := ""
	if ownerName != "" && nameNode != nil {
		qualified = ownerName + "." + c.text(nameNode)
	}
	paramTypes, count := parameterTypes(c, formalParameters(node))
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

func (c *collector) emitFieldDefinitions(node *sitter.Node) {
	typeName := normalizeJavaType(c.text(typeNodeForDeclaration(node)))
	ownerName := c.ownerTypeNameFor(node)
	ownerID := c.typeDefIDsByName[ownerName]
	for _, declarator := range namedChildrenOfType(node, "variable_declarator") {
		nameNode := declarationName(declarator)
		qualified := c.text(nameNode)
		if ownerName != "" && qualified != "" {
			qualified = ownerName + "." + qualified
		}
		c.addDefinition(node, scopeir.NodeProperty, nameNode, ownerID, "", typeName, qualified)
	}
}

func (c *collector) emitLocalVariableDefinitions(node *sitter.Node) {
	typeName := normalizeJavaType(c.text(typeNodeForDeclaration(node)))
	for _, declarator := range namedChildrenOfType(node, "variable_declarator") {
		nameNode := declarationName(declarator)
		declaredType := typeName
		if declaredType == "var" || declaredType == "" {
			declaredType = c.inferredTypeFromValue(initializerForDeclarator(declarator))
		}
		c.addDefinition(node, scopeir.NodeVariable, nameNode, "", "", declaredType, "")
	}
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

func javaDeclarationLabel(node *sitter.Node) scopeir.NodeLabel {
	if node == nil {
		return scopeir.NodeClass
	}
	switch node.Kind() {
	case "interface_declaration":
		return scopeir.NodeInterface
	case "enum_declaration":
		return scopeir.NodeEnum
	case "record_declaration":
		return scopeir.NodeRecord
	default:
		return scopeir.NodeClass
	}
}

func declarationName(node *sitter.Node) *sitter.Node {
	if node == nil {
		return nil
	}
	if name := child(node, "name"); name != nil {
		return name
	}
	return firstIdentifierLikeChild(node)
}

func formalParameters(node *sitter.Node) *sitter.Node {
	if node == nil {
		return nil
	}
	if params := child(node, "parameters"); params != nil {
		return params
	}
	return directChildOfKind(node, "formal_parameters")
}

func parameterTypes(c *collector, params *sitter.Node) ([]string, int) {
	if params == nil {
		return nil, 0
	}
	var out []string
	count := 0
	for _, param := range namedChildrenOfType(params, "formal_parameter") {
		typeName := normalizeJavaType(c.text(typeNodeForDeclaration(param)))
		if typeName != "" {
			out = append(out, typeName)
		}
		count++
	}
	return out, count
}

func returnTypeNameForCallable(c *collector, node *sitter.Node) string {
	if node == nil || node.Kind() == "constructor_declaration" || node.Kind() == "compact_constructor_declaration" {
		return ""
	}
	return normalizeJavaType(c.text(typeNodeForDeclaration(node)))
}

func (c *collector) addReturnType(node *sitter.Node, defID string, returnType string) {
	returnNode := typeNodeForDeclaration(node)
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
	for index := uint(0); index < node.NamedChildCount(); index++ {
		candidate := node.NamedChild(index)
		if candidate == nil {
			continue
		}
		switch candidate.Kind() {
		case "void_type", "boolean_type", "integral_type", "floating_point_type", "type_identifier", "scoped_type_identifier", "generic_type", "array_type":
			return candidate
		}
	}
	return nil
}

func initializerForDeclarator(node *sitter.Node) *sitter.Node {
	if node == nil || node.Kind() != "variable_declarator" {
		return nil
	}
	if value := child(node, "value"); value != nil {
		return value
	}
	for index := int(node.NamedChildCount()) - 1; index >= 0; index-- {
		candidate := node.NamedChild(uint(index))
		nameNode := declarationName(node)
		if candidate == nil || (nameNode != nil && candidate.Id() == nameNode.Id()) {
			continue
		}
		return candidate
	}
	return nil
}

func (c *collector) inferredTypeFromValue(node *sitter.Node) string {
	if node == nil {
		return ""
	}
	if node.Kind() == "object_creation_expression" {
		return baseJavaType(c.text(firstIdentifierLikeChild(node)))
	}
	return ""
}

func (c *collector) ownerTypeNameFor(node *sitter.Node) string {
	for current := node.Parent(); current != nil; current = current.Parent() {
		switch current.Kind() {
		case "class_declaration", "interface_declaration", "enum_declaration", "record_declaration":
			return c.text(declarationName(current))
		case "program":
			return ""
		}
	}
	return ""
}
