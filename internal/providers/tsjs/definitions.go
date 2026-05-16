package tsjs

import (
	sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

func (c *collector) emitDefinition(node *sitter.Node) {
	c.emitDefinitionKind(node, node.Kind())
}

func (c *collector) emitDefinitionKind(node *sitter.Node, kind string) {
	switch kind {
	case "class_declaration", "abstract_class_declaration":
		c.addDefinition(node, scopeir.NodeClass, child(node, "name"), "", "", "", "")
	case "interface_declaration":
		c.addDefinition(node, scopeir.NodeInterface, child(node, "name"), "", "", "", "")
	case "type_alias_declaration":
		c.addDefinition(node, scopeir.NodeTypeAlias, child(node, "name"), "", "", "", "")
	case "enum_declaration":
		c.addDefinition(node, scopeir.NodeEnum, child(node, "name"), "", "", "", "")
	case "function_declaration", "function_signature":
		c.addDefinition(node, scopeir.NodeFunction, child(node, "name"), "", returnTypeNameForCallable(c, node), "", "")
	case "method_definition", "abstract_method_signature", "method_signature":
		nameNode := child(node, "name")
		label := scopeir.NodeMethod
		if c.text(nameNode) == "constructor" {
			label = scopeir.NodeConstructor
		}
		ownerName := c.ownerDeclarationNameFor(node)
		qualified := ""
		if ownerName != "" && nameNode != nil {
			qualified = ownerName + "." + c.text(nameNode)
		}
		c.addDefinition(
			node,
			label,
			nameNode,
			c.ownerDefIDFor(node),
			returnTypeNameForCallable(c, node),
			"",
			qualified,
		)
		if ownerName != "" {
			c.addTypeBinding(node, "this", ownerName, scopeir.TypeSourceAnnotation)
		}
	case "public_field_definition", "property_signature":
		nameNode := child(node, "name")
		ownerName := c.ownerDeclarationNameFor(node)
		qualified := ""
		if ownerName != "" && nameNode != nil {
			qualified = ownerName + "." + c.text(nameNode)
		}
		c.addDefinition(
			node,
			scopeir.NodeProperty,
			nameNode,
			c.ownerDefIDFor(node),
			"",
			declaredTypeNameForNode(c, node),
			qualified,
		)
	case "variable_declarator":
		nameNode := child(node, "name")
		if nameNode == nil || nameNode.Kind() != "identifier" {
			return
		}
		label := scopeir.NodeVariable
		returnType := ""
		if isFunctionExpression(child(node, "value")) {
			label = scopeir.NodeFunction
			returnType = returnTypeNameForCallable(c, child(node, "value"))
		}
		c.addDefinition(node, label, nameNode, "", returnType, declaredTypeNameForNode(c, node), "")
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
) {
	if nameNode == nil {
		return
	}
	name := c.text(nameNode)
	if name == "" {
		return
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

	if returnType != "" {
		if returnNode := child(node, "return_type"); returnNode != nil {
			c.returnTypes = append(c.returnTypes, scopeir.ReturnTypeFact{
				DefID:    id,
				FilePath: c.filePath,
				FileHash: c.fileHash,
				Range:    nodeRange(returnNode),
				Type: scopeir.TypeRef{
					RawName:         returnType,
					DeclaredAtScope: scopeID,
					Source:          scopeir.TypeSourceReturn,
				},
			})
		}
	}
}

func defID(filePath string, rng scopeir.Range, label scopeir.NodeLabel, name string) string {
	return "def:" + filePath + "#" + intString(rng.StartLine) + ":" + intString(rng.StartCol) + ":" + string(label) + ":" + name
}

func returnTypeNameForCallable(c *collector, node *sitter.Node) string {
	returnNode := child(node, "return_type")
	if returnNode == nil {
		return ""
	}
	return stripTypeAnnotation(c.text(returnNode))
}

func declaredTypeNameForNode(c *collector, node *sitter.Node) string {
	typeNode := child(node, "type")
	if typeNode == nil {
		return ""
	}
	return stripTypeAnnotation(c.text(typeNode))
}

func (c *collector) ownerDefIDFor(node *sitter.Node) string {
	current := node.Parent()
	for current != nil {
		if label, ok := ownerDeclarationLabel(current); ok {
			nameNode := child(current, "name")
			if nameNode == nil {
				return ""
			}
			return defID(c.filePath, nodeRange(current), label, c.text(nameNode))
		}
		current = current.Parent()
	}
	return ""
}

func (c *collector) ownerDeclarationNameFor(node *sitter.Node) string {
	current := node.Parent()
	for current != nil {
		if _, ok := ownerDeclarationLabel(current); ok {
			return c.text(child(current, "name"))
		}
		current = current.Parent()
	}
	return ""
}

func ownerDeclarationLabel(node *sitter.Node) (scopeir.NodeLabel, bool) {
	switch node.Kind() {
	case "class_declaration", "abstract_class_declaration":
		return scopeir.NodeClass, true
	case "interface_declaration":
		return scopeir.NodeInterface, true
	default:
		return "", false
	}
}
