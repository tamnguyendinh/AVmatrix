package golang

import (
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

func (c *collector) buildContext(root *sitter.Node) {
	walkKind(root, func(node *sitter.Node, kind string) {
		c.collectContextForKind(node, kind)
	})
}

func (c *collector) collectContextForKind(node *sitter.Node, kind string) {
	switch kind {
	case "type_spec", "type_alias":
		nameNode := child(node, "name")
		if nameNode == nil {
			return
		}
		label := goTypeSpecLabelForKind(kind, child(node, "type"))
		c.typeDefIDsByName[c.text(nameNode)] = defID(c.filePath, nodeRange(node), label, c.text(nameNode))
	case "function_declaration", "method_declaration":
		nameNode := child(node, "name")
		returnType := returnTypeNameForCallable(c, node)
		if nameNode != nil && returnType != "" {
			c.returnTypesByCallableName[c.text(nameNode)] = returnType
		}
	}
}

func (c *collector) emitDefinition(node *sitter.Node) {
	c.emitDefinitionKind(node, node.Kind())
}

func (c *collector) emitDefinitionKind(node *sitter.Node, kind string) {
	switch kind {
	case "package_clause":
		c.addDefinition(node, scopeir.NodePackage, firstIdentifierLikeChild(node), "", "", "", "")
	case "type_spec", "type_alias":
		typeNode := child(node, "type")
		label := goTypeSpecLabelForKind(kind, typeNode)
		declaredType := ""
		if label == scopeir.NodeTypeAlias && typeNode != nil {
			declaredType = normalizeGoType(c.text(typeNode))
		}
		c.addDefinition(node, label, child(node, "name"), "", "", declaredType, "")
	case "function_declaration":
		params := parameterTypes(c, child(node, "parameters"))
		c.addCallableDefinition(node, scopeir.NodeFunction, child(node, "name"), "", "", params)
	case "method_declaration":
		nameNode := child(node, "name")
		ownerName, receiverName, receiverType := c.receiverInfo(node)
		qualified := ""
		if ownerName != "" && nameNode != nil {
			qualified = ownerName + "." + c.text(nameNode)
		}
		c.addCallableDefinition(node, scopeir.NodeMethod, nameNode, c.typeDefIDsByName[ownerName], qualified, parameterTypes(c, child(node, "parameters")))
		if receiverName != "" && receiverType != "" {
			c.addTypeBinding(child(node, "receiver"), receiverName, receiverType, scopeir.TypeSourceReceiver)
		}
	case "method_elem":
		ownerName := c.ownerTypeSpecNameFor(node)
		nameNode := child(node, "name")
		qualified := ""
		if ownerName != "" && nameNode != nil {
			qualified = ownerName + "." + c.text(nameNode)
		}
		c.addCallableDefinition(node, scopeir.NodeMethod, nameNode, c.typeDefIDsByName[ownerName], qualified, parameterTypes(c, child(node, "parameters")))
	case "field_declaration":
		c.emitFieldDefinitions(node)
	case "var_spec":
		c.emitValueSpecDefinitions(node, scopeir.NodeVariable)
	case "const_spec":
		c.emitValueSpecDefinitions(node, scopeir.NodeConst)
	case "short_var_declaration":
		c.emitShortVarDefinitions(node)
	case "range_clause":
		c.emitRangeClauseDefinitions(node)
	case "type_switch_statement":
		c.emitTypeSwitchStatementDefinitions(node)
	case "receive_statement":
		c.emitReceiveStatementDefinitions(node)
	}
}

func (c *collector) addCallableDefinition(
	node *sitter.Node,
	label scopeir.NodeLabel,
	nameNode *sitter.Node,
	ownerID string,
	qualifiedName string,
	paramTypes []string,
) {
	count := len(paramTypes)
	returnType := returnTypeNameForCallable(c, node)
	def := c.addDefinition(node, label, nameNode, ownerID, returnType, "", qualifiedName)
	if def == nil {
		return
	}
	def.ParameterCount = &count
	def.RequiredParameterCount = &count
	def.ParameterTypes = append([]string(nil), paramTypes...)
	if returnType != "" {
		c.addReturnType(node, def.ID, returnType)
	}
}

func (c *collector) emitFieldDefinitions(node *sitter.Node) {
	typeNode := child(node, "type")
	declaredType := ""
	if typeNode != nil {
		declaredType = normalizeGoType(c.text(typeNode))
	}
	ownerName := c.ownerTypeSpecNameFor(node)
	ownerID := c.typeDefIDsByName[ownerName]
	names := namedChildrenOfType(node, "field_identifier")
	for _, nameNode := range names {
		qualified := c.text(nameNode)
		if ownerName != "" {
			qualified = ownerName + "." + qualified
		}
		c.addDefinition(node, scopeir.NodeProperty, nameNode, ownerID, "", declaredType, qualified)
	}
}

func (c *collector) emitValueSpecDefinitions(node *sitter.Node, label scopeir.NodeLabel) {
	typeNode := child(node, "type")
	declaredType := ""
	if typeNode != nil {
		declaredType = normalizeGoType(c.text(typeNode))
	}
	for _, nameNode := range namedChildrenOfType(node, "identifier") {
		c.addDefinition(node, label, nameNode, "", "", declaredType, "")
	}
}

func (c *collector) emitShortVarDefinitions(node *sitter.Node) {
	left := child(node, "left")
	if left == nil {
		left = node.NamedChild(0)
	}
	right := child(node, "right")
	names := namedChildrenOfType(left, "identifier")
	values := namedValueChildren(right)
	for index, nameNode := range names {
		declaredType := ""
		if index < len(values) {
			declaredType = c.inferredTypeFromValue(values[index])
		}
		c.addDefinition(node, scopeir.NodeVariable, nameNode, "", "", declaredType, "")
	}
}

func (c *collector) emitRangeClauseDefinitions(node *sitter.Node) {
	if !rangeClauseDefines(c.text(node)) {
		return
	}
	left := child(node, "left")
	for _, nameNode := range definitionNameNodes(left) {
		c.addDefinition(node, scopeir.NodeVariable, nameNode, "", "", "", "")
	}
}

func rangeClauseDefines(text string) bool {
	rangeIndex := strings.Index(text, "range")
	if rangeIndex < 0 {
		return false
	}
	return strings.Contains(text[:rangeIndex], ":=")
}

func definitionNameNodes(node *sitter.Node) []*sitter.Node {
	if node == nil {
		return nil
	}
	if node.Kind() == "identifier" {
		return []*sitter.Node{node}
	}
	return namedChildrenOfType(node, "identifier")
}

func (c *collector) emitTypeSwitchStatementDefinitions(node *sitter.Node) {
	if !definesBefore(c.text(node), ".(type)") {
		return
	}
	alias := child(node, "alias")
	if alias == nil {
		return
	}
	for _, nameNode := range definitionNameNodes(alias) {
		c.addDefinition(node, scopeir.NodeVariable, nameNode, "", "", "", "")
	}
}

func (c *collector) emitReceiveStatementDefinitions(node *sitter.Node) {
	if !definesBefore(c.text(node), "<-") {
		return
	}
	left := child(node, "left")
	for _, nameNode := range definitionNameNodes(left) {
		c.addDefinition(node, scopeir.NodeVariable, nameNode, "", "", "", "")
	}
}

func definesBefore(text string, marker string) bool {
	index := strings.Index(text, marker)
	if index < 0 {
		return false
	}
	return strings.Contains(text[:index], ":=")
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

func (c *collector) addReturnType(node *sitter.Node, defID string, returnType string) {
	returnNode := child(node, "result")
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

func defID(filePath string, rng scopeir.Range, label scopeir.NodeLabel, name string) string {
	return "def:" + filePath + "#" + intString(rng.StartLine) + ":" + intString(rng.StartCol) + ":" + string(label) + ":" + name
}

func goTypeSpecLabel(typeNode *sitter.Node) scopeir.NodeLabel {
	return goTypeSpecLabelForKind("type_spec", typeNode)
}

func goTypeSpecLabelForKind(kind string, typeNode *sitter.Node) scopeir.NodeLabel {
	if kind == "type_alias" {
		return scopeir.NodeTypeAlias
	}
	if typeNode == nil {
		return scopeir.NodeTypeAlias
	}
	switch typeNode.Kind() {
	case "struct_type":
		return scopeir.NodeStruct
	case "interface_type":
		return scopeir.NodeInterface
	default:
		return scopeir.NodeTypeAlias
	}
}

func returnTypeNameForCallable(c *collector, node *sitter.Node) string {
	result := child(node, "result")
	if result == nil {
		return ""
	}
	return normalizeGoType(c.text(result))
}

func parameterTypes(c *collector, params *sitter.Node) []string {
	if params == nil {
		return nil
	}
	var out []string
	for _, param := range namedChildrenOfType(params, "parameter_declaration") {
		typeNode := child(param, "type")
		if typeNode == nil {
			continue
		}
		typeName := normalizeGoType(c.text(typeNode))
		names := namedChildrenOfType(param, "identifier")
		count := len(names)
		if count == 0 {
			count = 1
		}
		for i := 0; i < count; i++ {
			out = append(out, typeName)
		}
	}
	return out
}

func (c *collector) receiverInfo(node *sitter.Node) (ownerName string, receiverName string, receiverType string) {
	receiver := child(node, "receiver")
	param := firstNamedChildOfType(receiver, "parameter_declaration")
	if param == nil {
		return "", "", ""
	}
	if nameNode := child(param, "name"); nameNode != nil {
		receiverName = c.text(nameNode)
	}
	typeNode := child(param, "type")
	if typeNode == nil {
		return "", receiverName, ""
	}
	receiverType = normalizeGoType(c.text(typeNode))
	return baseGoType(receiverType), receiverName, receiverType
}

func (c *collector) ownerTypeSpecNameFor(node *sitter.Node) string {
	for current := node.Parent(); current != nil; current = current.Parent() {
		if current.Kind() == "type_spec" {
			return c.text(child(current, "name"))
		}
	}
	return ""
}

func namedValueChildren(node *sitter.Node) []*sitter.Node {
	var out []*sitter.Node
	if node == nil {
		return out
	}
	if node.Kind() == "expression_list" {
		for index := uint(0); index < node.NamedChildCount(); index++ {
			child := node.NamedChild(index)
			if child != nil {
				out = append(out, child)
			}
		}
		return out
	}
	for index := uint(0); index < node.NamedChildCount(); index++ {
		child := node.NamedChild(index)
		if child != nil {
			out = append(out, child)
		}
	}
	return out
}
