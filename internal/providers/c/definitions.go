package c

import (
	sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

func (c *collector) buildContext(root *sitter.Node) {
	walk(root, func(node *sitter.Node) {
		if node.Kind() != "struct_specifier" {
			return
		}
		nameNode := declarationName(node)
		if nameNode == nil {
			return
		}
		name := c.text(nameNode)
		c.typeDefIDsByName[name] = defID(c.filePath, nodeRange(node), scopeir.NodeStruct, name)
	})
}

func (c *collector) emitDefinition(node *sitter.Node) {
	switch node.Kind() {
	case "struct_specifier":
		c.addDefinition(node, scopeir.NodeStruct, declarationName(node), "", "", "", "")
	case "field_declaration":
		c.emitFieldDefinition(node)
	case "function_definition":
		c.emitFunctionDefinition(node)
	case "declaration":
		if c.insideFunction(node) {
			c.emitLocalDeclaration(node)
		}
	}
}

func (c *collector) emitFunctionDefinition(node *sitter.Node) {
	nameNode := declarationName(node)
	params := directChildOfKind(directChildOfKind(node, "function_declarator"), "parameter_list")
	paramTypes, count := parameterTypes(c, params)
	def := c.addDefinition(node, scopeir.NodeFunction, nameNode, "", returnTypeNameForDeclaration(c, node), "", "")
	if def == nil {
		return
	}
	def.ParameterCount = &count
	def.RequiredParameterCount = &count
	def.ParameterTypes = append([]string(nil), paramTypes...)
	if def.ReturnType != "" {
		c.addReturnType(node, def.ID, def.ReturnType)
	}
}

func (c *collector) emitFieldDefinition(node *sitter.Node) {
	ownerName := c.ownerStructNameFor(node)
	typeName := normalizeCType(c.text(typeNodeForDeclaration(node)))
	nameNode := declarationName(node)
	qualified := c.text(nameNode)
	if ownerName != "" && qualified != "" {
		qualified = ownerName + "." + qualified
	}
	c.addDefinition(node, scopeir.NodeProperty, nameNode, c.typeDefIDsByName[ownerName], "", typeName, qualified)
}

func (c *collector) emitLocalDeclaration(node *sitter.Node) {
	typeName := normalizeCType(c.text(typeNodeForDeclaration(node)))
	nameNode := declarationName(node)
	c.addDefinition(node, scopeir.NodeVariable, nameNode, "", "", typeName, "")
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

func declarationName(node *sitter.Node) *sitter.Node {
	if node == nil {
		return nil
	}
	switch node.Kind() {
	case "function_definition":
		return firstIdentifierLikeChild(directChildOfKind(node, "function_declarator"))
	case "parameter_declaration":
		return lastIdentifierLikeChild(node)
	case "field_declaration":
		if declarator := directChildOfKind(node, "function_declarator"); declarator != nil {
			return firstIdentifierLikeChild(declarator)
		}
		return lastIdentifierLikeChild(node)
	case "declaration":
		if declarator := directChildOfKind(node, "init_declarator"); declarator != nil {
			return firstIdentifierLikeChild(declarator)
		}
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
	for index := uint(0); index < params.NamedChildCount(); index++ {
		param := params.NamedChild(index)
		if param == nil || param.Kind() != "parameter_declaration" {
			continue
		}
		typeName := normalizeCType(c.text(typeNodeForDeclaration(param)))
		if typeName != "" {
			out = append(out, typeName)
		}
		count++
	}
	return out, count
}

func returnTypeNameForDeclaration(c *collector, node *sitter.Node) string {
	return normalizeCType(c.text(typeNodeForDeclaration(node)))
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
		case "primitive_type", "type_identifier", "struct_specifier":
			return candidate
		}
	}
	return nil
}

func (c *collector) ownerStructNameFor(node *sitter.Node) string {
	for current := node.Parent(); current != nil; current = current.Parent() {
		if current.Kind() == "struct_specifier" {
			return c.text(declarationName(current))
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
