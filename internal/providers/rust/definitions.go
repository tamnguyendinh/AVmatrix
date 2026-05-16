package rust

import (
	sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

func (c *collector) buildContext(root *sitter.Node) {
	walk(root, func(node *sitter.Node) {
		switch node.Kind() {
		case "struct_item", "trait_item":
			nameNode := declarationName(node)
			if nameNode == nil {
				return
			}
			name := c.text(nameNode)
			label := rustDeclarationLabel(node)
			rng := nodeRange(node)
			c.typeDefIDsByName[name] = defID(c.filePath, rng, label, name)
			c.typeScopeIDsByName[name] = scopeID(c.filePath, rng, scopeir.ScopeClass)
		case "function_item":
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
	case "mod_item":
		c.addDefinition(node, scopeir.NodePackage, declarationName(node), "", "", "", "")
	case "struct_item", "trait_item":
		c.addDefinition(node, rustDeclarationLabel(node), declarationName(node), "", "", "", "")
	case "function_item", "function_signature_item":
		c.emitCallableDefinition(node)
	case "field_declaration":
		c.emitFieldDefinition(node)
	case "let_declaration":
		c.emitLocalDefinition(node)
	}
}

func (c *collector) emitCallableDefinition(node *sitter.Node) {
	nameNode := declarationName(node)
	ownerName := c.ownerTypeNameFor(node)
	label := scopeir.NodeFunction
	if ownerName != "" {
		label = scopeir.NodeMethod
	}
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
		c.addTypeBinding(node, "self", ownerName, scopeir.TypeSourceSelf)
	}
	if def.ReturnType != "" {
		c.addReturnType(node, def.ID, def.ReturnType)
	}
}

func (c *collector) emitFieldDefinition(node *sitter.Node) {
	ownerName := c.ownerTypeNameFor(node)
	ownerID := c.typeDefIDsByName[ownerName]
	typeName := normalizeRustType(c.text(typeNodeForDeclaration(node)))
	for _, nameNode := range declaratorNames(node) {
		qualified := c.text(nameNode)
		if ownerName != "" && qualified != "" {
			qualified = ownerName + "." + qualified
		}
		c.addDefinition(node, scopeir.NodeProperty, nameNode, ownerID, "", typeName, qualified)
	}
}

func (c *collector) emitLocalDefinition(node *sitter.Node) {
	typeName := normalizeRustType(c.text(typeNodeForDeclaration(node)))
	if typeName == "" {
		typeName = c.inferredTypeFromValue(child(node, "value"))
	}
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

func rustDeclarationLabel(node *sitter.Node) scopeir.NodeLabel {
	if node != nil && node.Kind() == "trait_item" {
		return scopeir.NodeTrait
	}
	return scopeir.NodeStruct
}

func declarationName(node *sitter.Node) *sitter.Node {
	if node == nil {
		return nil
	}
	if name := child(node, "name"); name != nil {
		return name
	}
	switch node.Kind() {
	case "mod_item", "function_item", "function_signature_item":
		return firstNamedChildOfType(node, "identifier")
	case "struct_item", "trait_item":
		return firstNamedChildOfType(node, "type_identifier")
	case "field_declaration":
		if name := child(node, "name"); name != nil {
			return name
		}
		return firstNamedChildOfType(node, "field_identifier")
	case "parameter", "let_declaration":
		return firstIdentifierLikeChild(node)
	default:
		return firstIdentifierLikeChild(node)
	}
}

func declaratorNames(node *sitter.Node) []*sitter.Node {
	if node == nil {
		return nil
	}
	if name := declarationName(node); name != nil {
		return []*sitter.Node{name}
	}
	return nil
}

func parameterTypes(c *collector, params *sitter.Node) ([]string, int) {
	if params == nil {
		return nil, 0
	}
	var out []string
	count := 0
	for index := uint(0); index < params.NamedChildCount(); index++ {
		param := params.NamedChild(index)
		if param == nil {
			continue
		}
		switch param.Kind() {
		case "self_parameter":
			out = append(out, "self")
			count++
		case "parameter":
			typeName := normalizeRustType(c.text(typeNodeForDeclaration(param)))
			if typeName != "" {
				out = append(out, typeName)
			}
			count++
		}
	}
	return out, count
}

func returnTypeNameForCallable(c *collector, node *sitter.Node) string {
	if node == nil {
		return ""
	}
	if returnNode := child(node, "return_type"); returnNode != nil {
		return normalizeRustType(c.text(returnNode))
	}
	params := child(node, "parameters")
	for index := uint(0); index < node.NamedChildCount(); index++ {
		candidate := node.NamedChild(index)
		if candidate == nil || (params != nil && candidate.EndByte() <= params.EndByte()) {
			continue
		}
		if isRustTypeNode(candidate) {
			return normalizeRustType(c.text(candidate))
		}
	}
	return ""
}

func (c *collector) addReturnType(node *sitter.Node, defID string, returnType string) {
	returnNode := child(node, "return_type")
	if returnNode == nil {
		params := child(node, "parameters")
		for index := uint(0); index < node.NamedChildCount(); index++ {
			candidate := node.NamedChild(index)
			if candidate != nil && (params == nil || candidate.StartByte() > params.EndByte()) && isRustTypeNode(candidate) {
				returnNode = candidate
				break
			}
		}
	}
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
		if isRustTypeNode(candidate) {
			return candidate
		}
	}
	return nil
}

func isRustTypeNode(node *sitter.Node) bool {
	if node == nil {
		return false
	}
	switch node.Kind() {
	case "type_identifier", "primitive_type", "reference_type", "generic_type", "scoped_type_identifier", "unit_type", "tuple_type", "array_type":
		return true
	default:
		return false
	}
}

func (c *collector) ownerTypeNameFor(node *sitter.Node) string {
	for current := node.Parent(); current != nil; current = current.Parent() {
		switch current.Kind() {
		case "struct_item", "trait_item":
			return c.text(declarationName(current))
		case "impl_item":
			return c.implOwnerTypeName(current)
		case "source_file":
			return ""
		}
	}
	return ""
}

func (c *collector) implOwnerTypeName(node *sitter.Node) string {
	if typeNode := child(node, "type"); typeNode != nil {
		return baseRustType(c.text(typeNode))
	}
	for index := int(node.NamedChildCount()) - 1; index >= 0; index-- {
		candidate := node.NamedChild(uint(index))
		if isRustTypeNode(candidate) {
			return baseRustType(c.text(candidate))
		}
	}
	return ""
}

func (c *collector) implTraitName(node *sitter.Node) string {
	if traitNode := child(node, "trait"); traitNode != nil {
		return baseRustType(c.text(traitNode))
	}
	return ""
}
