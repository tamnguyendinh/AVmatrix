package swift

import (
	sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

func (c *collector) buildContext(root *sitter.Node) {
	walk(root, func(node *sitter.Node) {
		switch node.Kind() {
		case "class_declaration", "protocol_declaration":
			nameNode := declarationName(node)
			if nameNode == nil {
				return
			}
			name := c.text(nameNode)
			label := swiftDeclarationLabel(node)
			c.typeDefIDsByName[name] = defID(c.filePath, nodeRange(node), label, name)
			c.typeLabelsByName[name] = label
		case "function_declaration", "protocol_function_declaration":
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
	case "class_declaration", "protocol_declaration":
		c.addDefinition(nodeRange(node), swiftDeclarationLabel(node), declarationName(node), "", "", "", "")
	case "function_declaration", "protocol_function_declaration":
		label := scopeir.NodeFunction
		if c.ownerTypeNameFor(node) != "" {
			label = scopeir.NodeMethod
		}
		c.emitCallableDefinition(node, label, declarationName(node))
	case "init_declaration":
		c.emitCallableDefinition(node, scopeir.NodeConstructor, firstIdentifierForInit(node))
	case "property_declaration", "protocol_property_declaration":
		c.emitPropertyDefinition(node)
	}
}

func (c *collector) emitCallableDefinition(node *sitter.Node, label scopeir.NodeLabel, nameNode *sitter.Node) {
	ownerName := c.ownerTypeNameFor(node)
	qualified := ""
	if ownerName != "" && nameNode != nil {
		qualified = ownerName + "." + c.text(nameNode)
	}
	paramTypes, count := parameterTypes(c, node)
	rng := callableRange(node)
	def := c.addDefinition(rng, label, nameNode, c.typeDefIDsByName[ownerName], returnTypeNameForCallable(c, node), "", qualified)
	if def == nil {
		return
	}
	def.ParameterCount = &count
	def.RequiredParameterCount = &count
	def.ParameterTypes = append([]string(nil), paramTypes...)
	if ownerName != "" {
		c.addTypeBindingRange(rng, "self", ownerName, scopeir.TypeSourceSelf)
	}
	if def.ReturnType != "" {
		c.addReturnType(node, def.ID, def.ReturnType)
	}
}

func (c *collector) emitPropertyDefinition(node *sitter.Node) {
	nameNode := declarationName(node)
	if nameNode == nil {
		return
	}
	typeName := normalizeSwiftType(c.text(typeNodeForDeclaration(node)))
	if isMemberProperty(node) {
		ownerName := c.ownerTypeNameFor(node)
		qualified := c.text(nameNode)
		if ownerName != "" {
			qualified = ownerName + "." + qualified
		}
		c.addDefinition(nodeRange(node), scopeir.NodeProperty, nameNode, c.typeDefIDsByName[ownerName], "", typeName, qualified)
		return
	}
	if parentOfKind(node, "function_body") != nil {
		if typeName == "" {
			typeName = c.inferredTypeFromValue(node)
		}
		c.addDefinition(nodeRange(node), scopeir.NodeVariable, nameNode, "", "", typeName, "")
	}
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

func swiftDeclarationLabel(node *sitter.Node) scopeir.NodeLabel {
	if node != nil && node.Kind() == "protocol_declaration" {
		return scopeir.NodeInterface
	}
	return scopeir.NodeClass
}

func declarationName(node *sitter.Node) *sitter.Node {
	if node == nil {
		return nil
	}
	switch node.Kind() {
	case "class_declaration", "protocol_declaration":
		return firstNamedChildOfType(node, "type_identifier")
	case "function_declaration", "protocol_function_declaration":
		return firstNamedChildOfType(node, "simple_identifier")
	case "property_declaration", "protocol_property_declaration":
		if pattern := firstNamedChildOfType(node, "pattern"); pattern != nil {
			return firstIdentifierLikeChild(pattern)
		}
		return firstIdentifierLikeChild(node)
	case "parameter":
		return parameterName(node)
	default:
		if name := child(node, "name"); name != nil {
			return name
		}
		return firstIdentifierLikeChild(node)
	}
}

func parameterName(node *sitter.Node) *sitter.Node {
	var identifiers []*sitter.Node
	for index := uint(0); index < node.NamedChildCount(); index++ {
		candidate := node.NamedChild(index)
		if candidate != nil && candidate.Kind() == "simple_identifier" {
			identifiers = append(identifiers, candidate)
		}
	}
	if len(identifiers) == 0 {
		return nil
	}
	if len(identifiers) > 1 {
		return identifiers[1]
	}
	return identifiers[0]
}

func firstIdentifierForInit(node *sitter.Node) *sitter.Node {
	for index := uint(0); index < node.ChildCount(); index++ {
		child := node.Child(index)
		if child != nil && cstring(child.Kind()) == "init" {
			return child
		}
	}
	return firstIdentifierLikeChild(node)
}

func cstring(value string) string {
	return value
}

func parameterTypes(c *collector, callable *sitter.Node) ([]string, int) {
	var out []string
	count := 0
	for _, param := range directChildrenOfKind(callable, "parameter") {
		typeName := normalizeSwiftType(c.text(typeNodeForDeclaration(param)))
		if typeName != "" {
			out = append(out, typeName)
		}
		count++
	}
	return out, count
}

func returnTypeNameForCallable(c *collector, node *sitter.Node) string {
	if node == nil || node.Kind() == "init_declaration" {
		return ""
	}
	return normalizeSwiftType(c.text(lastNamedChildOfType(node, "user_type")))
}

func (c *collector) addReturnType(node *sitter.Node, defID string, returnType string) {
	returnNode := lastNamedChildOfType(node, "user_type")
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
	switch node.Kind() {
	case "function_declaration", "protocol_function_declaration":
		return lastNamedChildOfType(node, "user_type")
	case "parameter", "property_declaration", "protocol_property_declaration":
		return firstDescendantOfType(node, "user_type")
	default:
		return firstDescendantOfType(node, "user_type")
	}
}

func (c *collector) ownerTypeNameFor(node *sitter.Node) string {
	for current := node.Parent(); current != nil; current = current.Parent() {
		switch current.Kind() {
		case "class_declaration", "protocol_declaration":
			return c.text(declarationName(current))
		case "source_file":
			return ""
		}
	}
	return ""
}

func isMemberProperty(node *sitter.Node) bool {
	parent := node.Parent()
	return parent != nil && (parent.Kind() == "class_body" || parent.Kind() == "protocol_body")
}

func parentOfKind(node *sitter.Node, kind string) *sitter.Node {
	for current := node.Parent(); current != nil; current = current.Parent() {
		if current.Kind() == kind {
			return current
		}
	}
	return nil
}
