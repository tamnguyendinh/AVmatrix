package kotlin

import (
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

func (c *collector) buildContext(root *sitter.Node) {
	walk(root, func(node *sitter.Node) {
		if node.Kind() != "class_declaration" {
			return
		}
		nameNode := declarationName(node)
		if nameNode == nil {
			return
		}
		name := c.text(nameNode)
		c.typeDefIDsByName[name] = defID(c.filePath, nodeRange(node), kotlinDeclarationLabel(c, node), name)
	})
}

func (c *collector) emitDefinition(node *sitter.Node) {
	switch node.Kind() {
	case "package_header":
		c.addDefinition(node, scopeir.NodePackage, firstNamedChildOfType(node, "qualified_identifier"), "", "", "", "")
	case "class_declaration":
		c.addDefinition(node, kotlinDeclarationLabel(c, node), declarationName(node), "", "", "", "")
	case "primary_constructor":
		c.emitPrimaryConstructorDefinition(node)
	case "function_declaration":
		label := scopeir.NodeFunction
		if c.ownerTypeNameFor(node) != "" {
			label = scopeir.NodeMethod
		}
		c.emitCallableDefinition(node, label, declarationName(node))
	case "class_parameter":
		if isPropertyClassParameter(c, node) {
			c.emitClassParameterProperty(node)
		}
	case "property_declaration":
		c.emitPropertyDeclaration(node)
	}
}

func (c *collector) emitCallableDefinition(node *sitter.Node, label scopeir.NodeLabel, nameNode *sitter.Node) {
	ownerName := c.ownerTypeNameFor(node)
	qualified := ""
	if ownerName != "" && nameNode != nil {
		qualified = ownerName + "." + c.text(nameNode)
	}
	paramTypes, count := parameterTypes(c, directChildOfKind(node, "function_value_parameters"))
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

func (c *collector) emitPrimaryConstructorDefinition(node *sitter.Node) {
	classNode := parentOfKind(node, "class_declaration")
	nameNode := declarationName(classNode)
	if nameNode == nil {
		return
	}
	ownerName := c.text(nameNode)
	paramTypes, count := classParameterTypes(c, directChildOfKind(node, "class_parameters"))
	def := c.addDefinition(node, scopeir.NodeConstructor, nameNode, c.typeDefIDsByName[ownerName], "", "", ownerName+"."+ownerName)
	if def == nil {
		return
	}
	def.ParameterCount = &count
	def.RequiredParameterCount = &count
	def.ParameterTypes = append([]string(nil), paramTypes...)
	c.addTypeBinding(node, "this", ownerName, scopeir.TypeSourceSelf)
}

func (c *collector) emitClassParameterProperty(node *sitter.Node) {
	ownerName := c.ownerTypeNameFor(node)
	typeName := normalizeKotlinType(c.text(typeNodeForDeclaration(node)))
	nameNode := declarationName(node)
	qualified := c.text(nameNode)
	if ownerName != "" && qualified != "" {
		qualified = ownerName + "." + qualified
	}
	c.addDefinition(node, scopeir.NodeProperty, nameNode, c.typeDefIDsByName[ownerName], "", typeName, qualified)
}

func (c *collector) emitPropertyDeclaration(node *sitter.Node) {
	variable := firstNamedChildOfType(node, "variable_declaration")
	nameNode := declarationName(variable)
	typeName := normalizeKotlinType(c.text(typeNodeForDeclaration(variable)))
	if typeName == "" {
		typeName = c.inferredTypeFromValue(initializerForProperty(node))
	}
	if c.insideFunction(node) {
		c.addDefinition(node, scopeir.NodeVariable, nameNode, "", "", typeName, "")
		return
	}
	ownerName := c.ownerTypeNameFor(node)
	qualified := c.text(nameNode)
	if ownerName != "" && qualified != "" {
		qualified = ownerName + "." + qualified
	}
	c.addDefinition(node, scopeir.NodeProperty, nameNode, c.typeDefIDsByName[ownerName], "", typeName, qualified)
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

func kotlinDeclarationLabel(c *collector, node *sitter.Node) scopeir.NodeLabel {
	if strings.HasPrefix(strings.TrimSpace(c.text(node)), "interface ") {
		return scopeir.NodeInterface
	}
	return scopeir.NodeClass
}

func declarationName(node *sitter.Node) *sitter.Node {
	return firstIdentifierLikeChild(node)
}

func parameterTypes(c *collector, params *sitter.Node) ([]string, int) {
	if params == nil {
		return nil, 0
	}
	var out []string
	count := 0
	for _, param := range namedChildrenOfType(params, "parameter") {
		typeName := normalizeKotlinType(c.text(typeNodeForDeclaration(param)))
		if typeName != "" {
			out = append(out, typeName)
		}
		count++
	}
	return out, count
}

func classParameterTypes(c *collector, params *sitter.Node) ([]string, int) {
	if params == nil {
		return nil, 0
	}
	var out []string
	count := 0
	for _, param := range namedChildrenOfType(params, "class_parameter") {
		typeName := normalizeKotlinType(c.text(typeNodeForDeclaration(param)))
		if typeName != "" {
			out = append(out, typeName)
		}
		count++
	}
	return out, count
}

func returnTypeNameForCallable(c *collector, node *sitter.Node) string {
	if returnNode := returnTypeNodeForFunction(node); returnNode != nil {
		return normalizeKotlinType(c.text(returnNode))
	}
	if body := directChildOfKind(node, "function_body"); firstNamedChildOfType(body, "block") != nil {
		return "Unit"
	}
	return ""
}

func (c *collector) addReturnType(node *sitter.Node, defID string, returnType string) {
	returnNode := returnTypeNodeForFunction(node)
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
	for index := int(node.NamedChildCount()) - 1; index >= 0; index-- {
		candidate := node.NamedChild(uint(index))
		if candidate != nil && candidate.Kind() == "user_type" {
			return candidate
		}
	}
	return nil
}

func returnTypeNodeForFunction(node *sitter.Node) *sitter.Node {
	if node == nil {
		return nil
	}
	params := directChildOfKind(node, "function_value_parameters")
	for index := uint(0); index < node.NamedChildCount(); index++ {
		candidate := node.NamedChild(index)
		if candidate == nil || candidate.Kind() != "user_type" {
			continue
		}
		if params == nil || candidate.StartByte() > params.EndByte() {
			return candidate
		}
	}
	return nil
}

func initializerForProperty(node *sitter.Node) *sitter.Node {
	if node == nil || node.Kind() != "property_declaration" {
		return nil
	}
	variable := firstNamedChildOfType(node, "variable_declaration")
	for index := int(node.NamedChildCount()) - 1; index >= 0; index-- {
		candidate := node.NamedChild(uint(index))
		if candidate == nil || (variable != nil && candidate.Id() == variable.Id()) {
			continue
		}
		return candidate
	}
	return nil
}

func (c *collector) inferredTypeFromValue(node *sitter.Node) string {
	if node == nil || node.Kind() != "call_expression" {
		return ""
	}
	return baseKotlinType(c.text(callName(node)))
}

func (c *collector) ownerTypeNameFor(node *sitter.Node) string {
	for current := node.Parent(); current != nil; current = current.Parent() {
		switch current.Kind() {
		case "class_declaration":
			return c.text(declarationName(current))
		case "source_file":
			return ""
		}
	}
	return ""
}

func (c *collector) insideFunction(node *sitter.Node) bool {
	for current := node.Parent(); current != nil; current = current.Parent() {
		switch current.Kind() {
		case "function_declaration":
			return true
		case "class_declaration", "source_file":
			return false
		}
	}
	return false
}

func parentOfKind(node *sitter.Node, kind string) *sitter.Node {
	for current := node.Parent(); current != nil; current = current.Parent() {
		if current.Kind() == kind {
			return current
		}
	}
	return nil
}

func isPropertyClassParameter(c *collector, node *sitter.Node) bool {
	text := " " + strings.TrimSpace(c.text(node)) + " "
	return strings.Contains(text, " val ") || strings.Contains(text, " var ")
}
