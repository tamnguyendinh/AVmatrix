package python

import (
	sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

func (c *collector) buildContext(root *sitter.Node) {
	walk(root, func(node *sitter.Node) {
		if node.Kind() != "class_definition" {
			return
		}
		nameNode := firstNamedChildOfType(node, "identifier")
		if nameNode == nil {
			return
		}
		name := c.text(nameNode)
		c.classDefIDsByName[name] = defID(c.filePath, nodeRange(node), scopeir.NodeClass, name)
	})
}

func (c *collector) emitDefinition(node *sitter.Node) {
	switch node.Kind() {
	case "class_definition":
		c.addDefinition(node, scopeir.NodeClass, firstNamedChildOfType(node, "identifier"), "", "", "", "")
		c.emitClassHeritage(node)
	case "function_definition":
		c.emitFunctionDefinition(node)
	case "assignment":
		c.emitAssignmentDefinition(node)
	}
}

func (c *collector) emitFunctionDefinition(node *sitter.Node) {
	nameNode := firstNamedChildOfType(node, "identifier")
	ownerName := c.ownerClassNameFor(node)
	label := scopeir.NodeFunction
	ownerID := ""
	qualified := ""
	if ownerName != "" {
		label = scopeir.NodeMethod
		ownerID = c.classDefIDsByName[ownerName]
		if nameNode != nil {
			qualified = ownerName + "." + c.text(nameNode)
		}
	}
	paramTypes, count := parameterTypes(c, directChildOfKind(node, "parameters"), ownerName != "")
	def := c.addDefinition(node, label, nameNode, ownerID, returnTypeNameForCallable(c, node), "", qualified)
	if def == nil {
		return
	}
	def.ParameterCount = &count
	def.RequiredParameterCount = &count
	def.ParameterTypes = append([]string(nil), paramTypes...)
	if ownerName != "" {
		if receiverName := c.methodReceiverName(directChildOfKind(node, "parameters")); receiverName == "self" || receiverName == "cls" {
			c.addTypeBinding(node, receiverName, ownerName, scopeir.TypeSourceSelf)
		}
	}
}

func (c *collector) emitAssignmentDefinition(node *sitter.Node) {
	left := firstAssignmentTarget(node)
	if left == nil {
		return
	}
	declaredType := assignmentTypeName(c, node)
	ownerName := c.ownerClassNameForDirectClassBody(node)
	if ownerName != "" && left.Kind() == "identifier" {
		qualified := ownerName + "." + c.text(left)
		c.addDefinition(node, scopeir.NodeProperty, left, c.classDefIDsByName[ownerName], "", declaredType, qualified)
		return
	}
	if left.Kind() != "identifier" || c.ownerFunctionNameFor(node) == "" {
		return
	}
	if declaredType == "" {
		declaredType = c.inferredTypeFromValue(assignmentValue(node))
	}
	c.addDefinition(node, scopeir.NodeVariable, left, "", "", declaredType, "")
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
	if returnType != "" {
		if returnNode := returnTypeNode(node); returnNode != nil {
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
	return &c.definitions[len(c.definitions)-1]
}

func defID(filePath string, rng scopeir.Range, label scopeir.NodeLabel, name string) string {
	return "def:" + filePath + "#" + intString(rng.StartLine) + ":" + intString(rng.StartCol) + ":" + string(label) + ":" + name
}

func returnTypeNameForCallable(c *collector, node *sitter.Node) string {
	returnNode := returnTypeNode(node)
	if returnNode == nil {
		return ""
	}
	return normalizePythonType(c.text(returnNode))
}

func returnTypeNode(node *sitter.Node) *sitter.Node {
	if node == nil || node.Kind() != "function_definition" {
		return nil
	}
	paramsSeen := false
	for index := uint(0); index < node.NamedChildCount(); index++ {
		candidate := node.NamedChild(index)
		if candidate == nil {
			continue
		}
		if candidate.Kind() == "parameters" {
			paramsSeen = true
			continue
		}
		if paramsSeen && candidate.Kind() == "type" {
			return candidate
		}
	}
	return nil
}

func parameterTypes(c *collector, params *sitter.Node, method bool) ([]string, int) {
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
		name, typeName := parameterNameAndType(c, param)
		if method && (name == "self" || name == "cls") {
			continue
		}
		count++
		if typeName != "" {
			out = append(out, typeName)
		}
	}
	return out, count
}

func parameterNameAndType(c *collector, node *sitter.Node) (string, string) {
	switch node.Kind() {
	case "identifier":
		return c.text(node), ""
	case "typed_parameter", "default_parameter", "typed_default_parameter":
		nameNode := firstNamedChildOfType(node, "identifier")
		typeNode := directChildOfKind(node, "type")
		if typeNode == nil {
			for index := uint(0); index < node.NamedChildCount(); index++ {
				candidate := node.NamedChild(index)
				if candidate != nil && candidate.Kind() == "typed_parameter" {
					return parameterNameAndType(c, candidate)
				}
			}
		}
		return c.text(nameNode), normalizePythonType(c.text(typeNode))
	default:
		return c.text(firstNamedChildOfType(node, "identifier")), ""
	}
}

func (c *collector) methodReceiverName(params *sitter.Node) string {
	if params == nil || params.NamedChildCount() == 0 {
		return ""
	}
	name, _ := parameterNameAndType(c, params.NamedChild(0))
	return name
}

func (c *collector) ownerClassNameFor(node *sitter.Node) string {
	for current := node.Parent(); current != nil; current = current.Parent() {
		if current.Kind() == "class_definition" {
			return c.text(firstNamedChildOfType(current, "identifier"))
		}
		if current.Kind() == "module" {
			break
		}
	}
	return ""
}

func (c *collector) ownerFunctionNameFor(node *sitter.Node) string {
	for current := node.Parent(); current != nil; current = current.Parent() {
		if current.Kind() == "function_definition" {
			return c.text(firstNamedChildOfType(current, "identifier"))
		}
		if current.Kind() == "module" {
			break
		}
	}
	return ""
}

func (c *collector) ownerClassNameForDirectClassBody(node *sitter.Node) string {
	parent := node.Parent()
	if parent != nil && parent.Kind() == "expression_statement" {
		parent = parent.Parent()
	}
	if parent == nil || parent.Kind() != "block" {
		return ""
	}
	if classNode := parent.Parent(); classNode != nil && classNode.Kind() == "class_definition" {
		return c.text(firstNamedChildOfType(classNode, "identifier"))
	}
	return ""
}

func firstAssignmentTarget(node *sitter.Node) *sitter.Node {
	if node == nil || node.Kind() != "assignment" || node.NamedChildCount() == 0 {
		return nil
	}
	return node.NamedChild(0)
}

func assignmentValue(node *sitter.Node) *sitter.Node {
	if node == nil || node.Kind() != "assignment" {
		return nil
	}
	for index := int(node.NamedChildCount()) - 1; index >= 0; index-- {
		candidate := node.NamedChild(uint(index))
		if candidate == nil || candidate.Kind() == "type" {
			continue
		}
		return candidate
	}
	return nil
}

func assignmentTypeName(c *collector, node *sitter.Node) string {
	typeNode := directChildOfKind(node, "type")
	if typeNode == nil {
		return ""
	}
	return normalizePythonType(c.text(typeNode))
}

func (c *collector) inferredTypeFromValue(node *sitter.Node) string {
	if node == nil || node.Kind() != "call" {
		return ""
	}
	fn := callFunction(node)
	if fn == nil || fn.Kind() != "identifier" {
		return ""
	}
	name := c.text(fn)
	if name == "" || name[0] < 'A' || name[0] > 'Z' {
		return ""
	}
	return name
}
