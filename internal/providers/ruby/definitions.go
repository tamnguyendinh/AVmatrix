package ruby

import (
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

func (c *collector) buildContext(root *sitter.Node) {
	walk(root, func(node *sitter.Node) {
		switch node.Kind() {
		case "class", "module":
			nameNode := declarationName(node)
			if nameNode == nil {
				return
			}
			name := c.text(nameNode)
			c.typeDefIDsByName[name] = defID(c.filePath, nodeRange(node), declarationLabel(node), name)
		}
	})
}

func (c *collector) emitDefinition(node *sitter.Node) {
	switch node.Kind() {
	case "module", "class":
		c.addDefinition(nodeRange(node), declarationLabel(node), declarationName(node), "", "", c.qualifiedTypeName(node))
	case "method":
		label := scopeir.NodeFunction
		ownerName := c.ownerTypeNameFor(node)
		qualified := ""
		if ownerName != "" {
			label = scopeir.NodeMethod
			if nameNode := declarationName(node); nameNode != nil {
				qualified = ownerName + "." + c.text(nameNode)
			}
		}
		def := c.addDefinition(nodeRange(node), label, declarationName(node), c.typeDefIDsByName[ownerName], "", qualified)
		if def != nil {
			count := len(directChildrenOfKind(directChildOfKind(node, "method_parameters"), "identifier"))
			def.ParameterCount = &count
			def.RequiredParameterCount = &count
		}
	case "call":
		c.emitAttrReaderProperties(node)
	case "assignment":
		c.emitAssignmentDefinition(node)
	}
}

func (c *collector) qualifiedTypeName(node *sitter.Node) string {
	nameNode := declarationName(node)
	name := c.text(nameNode)
	if name == "" {
		return ""
	}
	names := []string{name}
	for current := node.Parent(); current != nil; current = current.Parent() {
		switch current.Kind() {
		case "class", "module":
			parentName := c.text(declarationName(current))
			if parentName != "" {
				names = append([]string{parentName}, names...)
			}
		case "program":
			return strings.Join(names, ".")
		}
	}
	return strings.Join(names, ".")
}

func (c *collector) emitAttrReaderProperties(node *sitter.Node) {
	if !isRubyAttributeAccessor(callName(c, node)) {
		return
	}
	ownerName := c.ownerTypeNameFor(node)
	if ownerName == "" {
		return
	}
	args := directChildOfKind(node, "argument_list")
	declaredType := c.yardReturnTypeBefore(node)
	for _, symbol := range directChildrenOfKind(args, "simple_symbol") {
		name := strings.TrimPrefix(c.text(symbol), ":")
		c.addPropertyDefinition(nodeRange(symbol), name, ownerName, declaredType)
	}
}

func isRubyAttributeAccessor(name string) bool {
	return name == "attr_reader" || name == "attr_accessor" || name == "attr_writer"
}

func (c *collector) emitAssignmentDefinition(node *sitter.Node) {
	left := firstNamedChild(node)
	if left == nil {
		return
	}
	switch left.Kind() {
	case "instance_variable":
		ownerName := c.ownerTypeNameFor(node)
		if ownerName != "" {
			c.addPropertyDefinition(nodeRange(left), strings.TrimPrefix(c.text(left), "@"), ownerName, "")
		}
	case "identifier":
		if parentOfKind(node, "method") != nil {
			c.addDefinition(nodeRange(left), scopeir.NodeVariable, left, "", "", "")
		}
	}
}

func (c *collector) addPropertyDefinition(rng scopeir.Range, name string, ownerName string, declaredType string) {
	if name == "" || ownerName == "" {
		return
	}
	key := ownerName + "." + name
	if _, ok := c.propertiesSeen[key]; ok {
		return
	}
	c.propertiesSeen[key] = struct{}{}
	nameNode := syntheticNameNode{name: name}
	c.addDefinition(rng, scopeir.NodeProperty, nameNode, c.typeDefIDsByName[ownerName], declaredType, key)
}

func (c *collector) yardReturnTypeBefore(node *sitter.Node) string {
	lines := strings.Split(string(c.source), "\n")
	for index := int(node.StartPosition().Row) - 1; index >= 0 && index < len(lines); index-- {
		line := strings.TrimSpace(strings.TrimSuffix(lines[index], "\r"))
		if line == "" {
			continue
		}
		if !strings.HasPrefix(line, "#") {
			break
		}
		if typeName := parseYARDReturnType(line); typeName != "" {
			return typeName
		}
	}
	return ""
}

func parseYARDReturnType(line string) string {
	at := strings.Index(line, "@return")
	if at < 0 {
		return ""
	}
	open := strings.Index(line[at:], "[")
	if open < 0 {
		return ""
	}
	open += at
	close := strings.Index(line[open+1:], "]")
	if close < 0 {
		return ""
	}
	close += open + 1
	raw := strings.TrimSpace(line[open+1 : close])
	if raw == "" || raw[0] < 'A' || raw[0] > 'Z' {
		return ""
	}
	end := 0
	for end < len(raw) {
		ch := raw[end]
		if (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') || ch == '_' {
			end++
			continue
		}
		break
	}
	return raw[:end]
}

type syntheticNameNode struct {
	name string
}

func (n syntheticNameNode) text() string {
	return n.name
}

type nameSource interface {
	text() string
}

func (c *collector) addDefinition(
	rng scopeir.Range,
	label scopeir.NodeLabel,
	nameNode any,
	ownerID string,
	declaredType string,
	qualifiedName string,
) *scopeir.DefinitionFact {
	name := c.nameText(nameNode)
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

func (c *collector) nameText(value any) string {
	switch typed := value.(type) {
	case *sitter.Node:
		return c.text(typed)
	case nameSource:
		return typed.text()
	default:
		return ""
	}
}

func (c *collector) ownerTypeNameFor(node *sitter.Node) string {
	for current := node.Parent(); current != nil; current = current.Parent() {
		switch current.Kind() {
		case "class", "module":
			return c.text(declarationName(current))
		case "program":
			return ""
		}
	}
	return ""
}
