package cpp

import (
	sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

func (c *collector) buildContext(root *sitter.Node) {
	walk(root, func(node *sitter.Node) {
		switch node.Kind() {
		case "class_specifier", "struct_specifier":
			nameNode := declarationName(node)
			if nameNode == nil {
				return
			}
			name := c.text(nameNode)
			c.typeDefIDsByName[name] = defID(c.filePath, nodeRange(node), cppDeclarationLabel(node), name)
		}
	})
}

func (c *collector) emitDefinition(node *sitter.Node) {
	switch node.Kind() {
	case "namespace_definition":
		c.addDefinition(node, scopeir.NodePackage, namespaceName(node), "", "", "", "")
	case "class_specifier", "struct_specifier":
		c.addDefinition(node, cppDeclarationLabel(node), declarationName(node), "", "", "", "")
	case "function_definition":
		c.emitCallableDefinition(node)
	case "declaration":
		if hasFunctionDeclarator(node) {
			c.emitCallableDefinition(node)
		} else if c.insideFunction(node) {
			c.emitVariableDefinition(node)
		}
	case "field_declaration":
		if hasFunctionDeclarator(node) {
			c.emitCallableDefinition(node)
		} else {
			c.emitFieldDefinition(node)
		}
	}
}

func (c *collector) emitCallableDefinition(node *sitter.Node) {
	nameNode := declarationName(node)
	ownerName := c.ownerTypeNameFor(node)
	label := scopeir.NodeFunction
	if ownerName != "" {
		label = scopeir.NodeMethod
		if nameNode != nil && c.text(nameNode) == ownerName {
			label = scopeir.NodeConstructor
		}
	}
	qualified := ""
	if ownerName != "" && nameNode != nil {
		qualified = ownerName + "." + c.text(nameNode)
	}
	params := directChildOfKind(functionDeclarator(node), "parameter_list")
	paramTypes, count := parameterTypes(c, params)
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
	typeName := normalizeCPPType(c.text(typeNodeForDeclaration(node)))
	ownerName := c.ownerTypeNameFor(node)
	ownerID := c.typeDefIDsByName[ownerName]
	for _, nameNode := range declaratorNames(node) {
		qualified := c.text(nameNode)
		if ownerName != "" && qualified != "" {
			qualified = ownerName + "." + qualified
		}
		c.addDefinition(node, scopeir.NodeProperty, nameNode, ownerID, "", typeName, qualified)
	}
}

func (c *collector) emitVariableDefinition(node *sitter.Node) {
	typeName := normalizeCPPType(c.text(typeNodeForDeclaration(node)))
	for _, nameNode := range declaratorNames(node) {
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

func cppDeclarationLabel(node *sitter.Node) scopeir.NodeLabel {
	if node != nil && node.Kind() == "struct_specifier" {
		return scopeir.NodeStruct
	}
	return scopeir.NodeClass
}

func declarationName(node *sitter.Node) *sitter.Node {
	if node == nil {
		return nil
	}
	switch node.Kind() {
	case "class_specifier", "struct_specifier":
		return firstNamedChildOfType(node, "type_identifier")
	case "namespace_definition":
		return namespaceName(node)
	case "parameter_declaration":
		return lastIdentifierLikeChild(node)
	case "declaration", "field_declaration", "function_definition":
		if declarator := functionDeclarator(node); declarator != nil {
			return functionDeclaratorName(declarator)
		}
		if declarator := directChildOfKind(node, "init_declarator"); declarator != nil {
			return firstIdentifierLikeChild(declarator)
		}
		return lastIdentifierLikeChild(node)
	default:
		return firstIdentifierLikeChild(node)
	}
}

func functionDeclaratorName(node *sitter.Node) *sitter.Node {
	if node == nil {
		return nil
	}
	var chosen *sitter.Node
	for index := uint(0); index < node.NamedChildCount(); index++ {
		candidate := node.NamedChild(index)
		if candidate == nil || candidate.Kind() == "parameter_list" {
			continue
		}
		if candidate.Kind() == "destructor_name" {
			return firstIdentifierLikeChild(candidate)
		}
		if candidate.Kind() == "operator_name" {
			return candidate
		}
		if isIdentifierLike(candidate) {
			chosen = candidate
			continue
		}
		if candidate.Kind() == "qualified_identifier" {
			chosen = lastIdentifierLikeChild(candidate)
		}
	}
	return chosen
}

func namespaceName(node *sitter.Node) *sitter.Node {
	if name := firstNamedChildOfType(node, "namespace_identifier"); name != nil {
		return name
	}
	if name := firstNamedChildOfType(node, "nested_namespace_specifier"); name != nil {
		return name
	}
	return firstIdentifierLikeChild(node)
}

func declaratorNames(node *sitter.Node) []*sitter.Node {
	if node == nil {
		return nil
	}
	var out []*sitter.Node
	for index := uint(0); index < node.NamedChildCount(); index++ {
		candidate := node.NamedChild(index)
		if candidate == nil {
			continue
		}
		switch candidate.Kind() {
		case "init_declarator", "pointer_declarator", "reference_declarator":
			if name := firstIdentifierLikeChild(candidate); name != nil {
				out = append(out, name)
			}
		case "field_identifier", "identifier":
			out = append(out, candidate)
		}
	}
	if len(out) == 0 {
		if name := declarationName(node); name != nil {
			out = append(out, name)
		}
	}
	return out
}

func parameterTypes(c *collector, params *sitter.Node) ([]string, int) {
	if params == nil {
		return nil, 0
	}
	var out []string
	count := 0
	for _, param := range directChildrenOfKind(params, "parameter_declaration") {
		typeName := normalizeCPPType(c.text(typeNodeForDeclaration(param)))
		if typeName != "" {
			out = append(out, typeName)
		}
		count++
	}
	return out, count
}

func returnTypeNameForCallable(c *collector, node *sitter.Node) string {
	if node == nil {
		return ""
	}
	nameNode := declarationName(node)
	ownerName := c.ownerTypeNameFor(node)
	if ownerName != "" && nameNode != nil && c.text(nameNode) == ownerName {
		return ""
	}
	return normalizeCPPType(c.text(typeNodeForDeclaration(node)))
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
		case "primitive_type", "type_identifier", "qualified_identifier", "template_type",
			"sized_type_specifier", "placeholder_type_specifier", "auto":
			return candidate
		}
	}
	return nil
}

func (c *collector) ownerTypeNameFor(node *sitter.Node) string {
	for current := node.Parent(); current != nil; current = current.Parent() {
		switch current.Kind() {
		case "class_specifier", "struct_specifier":
			return c.text(declarationName(current))
		case "translation_unit":
			return ""
		}
	}
	return ""
}

func (c *collector) insideFunction(node *sitter.Node) bool {
	for current := node.Parent(); current != nil; current = current.Parent() {
		switch current.Kind() {
		case "function_definition":
			return true
		case "translation_unit":
			return false
		}
	}
	return false
}
