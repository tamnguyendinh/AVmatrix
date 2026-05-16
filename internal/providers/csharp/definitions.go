package csharp

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
			c.typeDefIDsByName[name] = defID(c.filePath, nodeRange(node), csharpDeclarationLabel(node), name)
		}
	})
}

func (c *collector) emitDefinition(node *sitter.Node) {
	switch node.Kind() {
	case "file_scoped_namespace_declaration", "namespace_declaration":
		nameNode := firstNamedChildOfType(node, "qualified_name")
		c.namespaceName = c.text(nameNode)
		c.addDefinition(node, scopeir.NodePackage, nameNode, "", "", "", "")
	case "class_declaration", "interface_declaration":
		c.addDefinition(node, csharpDeclarationLabel(node), declarationName(node), "", "", "", c.qualifiedTypeName(node))
	case "method_declaration":
		c.emitCallableDefinition(node, scopeir.NodeMethod, declarationName(node))
	case "constructor_declaration":
		c.emitCallableDefinition(node, scopeir.NodeConstructor, declarationName(node))
	case "field_declaration":
		c.emitFieldDefinition(node)
	case "local_declaration_statement":
		c.emitLocalDeclaration(node)
	}
}

func (c *collector) qualifiedTypeName(node *sitter.Node) string {
	nameNode := declarationName(node)
	name := c.text(nameNode)
	if name == "" || c.namespaceName == "" {
		return name
	}
	return c.namespaceName + "." + name
}

func (c *collector) emitCallableDefinition(node *sitter.Node, label scopeir.NodeLabel, nameNode *sitter.Node) {
	ownerName := c.ownerTypeNameFor(node)
	qualified := ""
	if ownerName != "" && nameNode != nil {
		qualified = ownerName + "." + c.text(nameNode)
	}
	paramTypes, count := parameterTypes(c, directChildOfKind(node, "parameter_list"))
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

func (c *collector) emitFieldDefinition(node *sitter.Node) {
	declaration := firstNamedChildOfType(node, "variable_declaration")
	typeName := normalizeCSharpType(c.text(typeNodeForDeclaration(declaration)))
	ownerName := c.ownerTypeNameFor(node)
	ownerID := c.typeDefIDsByName[ownerName]
	for _, declarator := range directChildrenOfKind(declaration, "variable_declarator") {
		nameNode := declarationName(declarator)
		qualified := c.text(nameNode)
		if ownerName != "" && qualified != "" {
			qualified = ownerName + "." + qualified
		}
		c.addDefinition(node, scopeir.NodeProperty, nameNode, ownerID, "", typeName, qualified)
	}
}

func (c *collector) emitLocalDeclaration(node *sitter.Node) {
	declaration := firstNamedChildOfType(node, "variable_declaration")
	typeName := normalizeCSharpType(c.text(typeNodeForDeclaration(declaration)))
	for _, declarator := range directChildrenOfKind(declaration, "variable_declarator") {
		nameNode := declarationName(declarator)
		c.addDefinition(node, scopeir.NodeVariable, nameNode, "", "", typeName, "")
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

func csharpDeclarationLabel(node *sitter.Node) scopeir.NodeLabel {
	if node != nil && node.Kind() == "interface_declaration" {
		return scopeir.NodeInterface
	}
	return scopeir.NodeClass
}

func declarationName(node *sitter.Node) *sitter.Node {
	if node == nil {
		return nil
	}
	switch node.Kind() {
	case "parameter":
		return lastIdentifierLikeChild(node)
	case "variable_declarator":
		return firstIdentifierLikeChild(node)
	case "method_declaration":
		params := directChildOfKind(node, "parameter_list")
		return identifierBefore(node, params)
	case "constructor_declaration", "class_declaration", "interface_declaration":
		return firstIdentifierLikeChild(node)
	default:
		return firstIdentifierLikeChild(node)
	}
}

func identifierBefore(node *sitter.Node, before *sitter.Node) *sitter.Node {
	if node == nil {
		return nil
	}
	var chosen *sitter.Node
	for index := uint(0); index < node.NamedChildCount(); index++ {
		candidate := node.NamedChild(index)
		if candidate == nil {
			continue
		}
		if before != nil && candidate.StartByte() >= before.StartByte() {
			break
		}
		if candidate.Kind() == "identifier" {
			chosen = candidate
		}
	}
	return chosen
}

func parameterTypes(c *collector, params *sitter.Node) ([]string, int) {
	if params == nil {
		return nil, 0
	}
	var out []string
	count := 0
	for _, param := range directChildrenOfKind(params, "parameter") {
		typeName := normalizeCSharpType(c.text(typeNodeForDeclaration(param)))
		if typeName != "" {
			out = append(out, typeName)
		}
		count++
	}
	return out, count
}

func returnTypeNameForCallable(c *collector, node *sitter.Node) string {
	if node == nil || node.Kind() == "constructor_declaration" {
		return ""
	}
	return normalizeCSharpType(c.text(typeNodeForDeclaration(node)))
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
		switch candidate.Kind() {
		case "predefined_type", "identifier", "qualified_name", "generic_name":
			return candidate
		}
	}
	return nil
}

func (c *collector) ownerTypeNameFor(node *sitter.Node) string {
	for current := node.Parent(); current != nil; current = current.Parent() {
		switch current.Kind() {
		case "class_declaration", "interface_declaration":
			return c.text(declarationName(current))
		case "compilation_unit":
			return ""
		}
	}
	return ""
}
