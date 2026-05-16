package dart

import (
	sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

func (c *collector) buildContext(root *sitter.Node) {
	walk(root, func(node *sitter.Node) {
		switch node.Kind() {
		case "class_definition":
			nameNode := declarationName(node)
			if nameNode == nil {
				return
			}
			name := c.text(nameNode)
			c.typeDefIDsByName[name] = defID(c.filePath, nodeRange(node), dartDeclarationLabel(node), name)
		case "function_signature":
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
	case "class_definition":
		c.addDefinition(nodeRange(node), scopeir.NodeLabel(dartDeclarationLabel(node)), declarationName(node), "", "", "", "")
	case "function_signature":
		label := scopeir.NodeFunction
		if c.ownerTypeNameFor(node) != "" {
			label = scopeir.NodeMethod
		}
		c.emitCallableDefinition(node, label, declarationName(node))
	case "constructor_signature":
		c.emitCallableDefinition(node, scopeir.NodeConstructor, declarationName(node))
	case "declaration":
		c.emitFieldDefinition(node)
	case "initialized_variable_definition":
		c.emitLocalDefinition(node)
	}
}

func (c *collector) emitCallableDefinition(node *sitter.Node, label scopeir.NodeLabel, nameNode *sitter.Node) {
	ownerName := c.ownerTypeNameFor(node)
	qualified := ""
	if ownerName != "" && nameNode != nil {
		qualified = ownerName + "." + c.text(nameNode)
	}
	paramTypes, count := parameterTypes(c, directChildOfKind(node, "formal_parameter_list"))
	rng := callableRange(node)
	def := c.addDefinition(rng, label, nameNode, c.typeDefIDsByName[ownerName], returnTypeNameForCallable(c, node), "", qualified)
	if def == nil {
		return
	}
	def.ParameterCount = &count
	def.RequiredParameterCount = &count
	def.ParameterTypes = append([]string(nil), paramTypes...)
	if ownerName != "" {
		c.addTypeBindingRange(rng, "this", ownerName, scopeir.TypeSourceSelf)
	}
	if def.ReturnType != "" {
		c.addReturnType(node, def.ID, def.ReturnType)
	}
}

func (c *collector) emitFieldDefinition(node *sitter.Node) {
	if c.ownerTypeNameFor(node) == "" || directChildOfKind(node, "initialized_identifier_list") == nil {
		return
	}
	typeName := normalizeDartType(c.text(typeNodeForDeclaration(node)))
	ownerName := c.ownerTypeNameFor(node)
	ownerID := c.typeDefIDsByName[ownerName]
	for _, item := range directChildrenOfKind(directChildOfKind(node, "initialized_identifier_list"), "initialized_identifier") {
		nameNode := declarationName(item)
		qualified := c.text(nameNode)
		if ownerName != "" && qualified != "" {
			qualified = ownerName + "." + qualified
		}
		c.addDefinition(nodeRange(node), scopeir.NodeProperty, nameNode, ownerID, "", typeName, qualified)
	}
}

func (c *collector) emitLocalDefinition(node *sitter.Node) {
	if parentOfKind(node, "local_variable_declaration") == nil {
		return
	}
	nameNode := declarationName(node)
	typeName := normalizeDartType(c.text(typeNodeForDeclaration(node)))
	if typeName == "" {
		typeName = c.inferredTypeFromValue(node)
	}
	c.addDefinition(nodeRange(node), scopeir.NodeVariable, nameNode, "", "", typeName, "")
}

func (c *collector) addDefinition(
	rng scopeir.Range,
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

func dartDeclarationLabel(node *sitter.Node) scopeir.NodeLabel {
	if node == nil || node.Kind() != "class_definition" {
		return scopeir.NodeClass
	}
	if directChildOfKind(node, "abstract") != nil {
		return scopeir.NodeInterface
	}
	return scopeir.NodeClass
}

func declarationName(node *sitter.Node) *sitter.Node {
	if node == nil {
		return nil
	}
	if name := child(node, "name"); name != nil {
		return name
	}
	switch node.Kind() {
	case "class_definition", "function_signature", "constructor_signature", "formal_parameter", "initialized_identifier", "initialized_variable_definition":
		return firstIdentifierLikeChild(node)
	case "constructor_param":
		return lastIdentifierLikeChild(node)
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
	for _, param := range directChildrenOfKind(params, "formal_parameter") {
		typeName := normalizeDartType(c.text(typeNodeForDeclaration(param)))
		if typeName != "" {
			out = append(out, typeName)
		}
		count++
	}
	return out, count
}

func returnTypeNameForCallable(c *collector, node *sitter.Node) string {
	if node == nil || node.Kind() == "constructor_signature" {
		return ""
	}
	return normalizeDartType(c.text(typeNodeForDeclaration(node)))
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
	for index := uint(0); index < node.NamedChildCount(); index++ {
		candidate := node.NamedChild(index)
		if candidate == nil {
			continue
		}
		if isDartTypeNode(candidate) {
			return candidate
		}
		if candidate.Kind() == "initialized_identifier_list" || candidate.Kind() == "formal_parameter_list" {
			return nil
		}
	}
	return nil
}

func isDartTypeNode(node *sitter.Node) bool {
	if node == nil {
		return false
	}
	switch node.Kind() {
	case "type_identifier", "void_type", "type_name":
		return true
	default:
		return false
	}
}

func (c *collector) ownerTypeNameFor(node *sitter.Node) string {
	for current := node.Parent(); current != nil; current = current.Parent() {
		switch current.Kind() {
		case "class_definition":
			return c.text(declarationName(current))
		case "program":
			return ""
		}
	}
	return ""
}

func parentOfKind(node *sitter.Node, kind string) *sitter.Node {
	for current := node.Parent(); current != nil; current = current.Parent() {
		if current.Kind() == kind {
			return current
		}
	}
	return nil
}
