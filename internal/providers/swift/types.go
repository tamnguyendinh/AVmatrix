package swift

import (
	sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

func (c *collector) emitTypeBinding(node *sitter.Node) {
	switch node.Kind() {
	case "parameter":
		nameNode := declarationName(node)
		typeNode := typeNodeForDeclaration(node)
		if nameNode != nil && typeNode != nil {
			c.addTypeBinding(node, c.text(nameNode), normalizeSwiftType(c.text(typeNode)), scopeir.TypeSourceParameter)
		}
	case "property_declaration", "protocol_property_declaration":
		nameNode := declarationName(node)
		typeName := normalizeSwiftType(c.text(typeNodeForDeclaration(node)))
		if parentOfKind(node, "function_body") != nil && typeName == "" {
			typeName = c.inferredTypeFromValue(node)
		}
		if nameNode != nil && typeName != "" {
			source := scopeir.TypeSourceAnnotation
			if parentOfKind(node, "function_body") != nil && typeNodeForDeclaration(node) == nil {
				source = scopeir.TypeSourceAssignment
			}
			c.addTypeBinding(node, c.text(nameNode), typeName, source)
		}
	}
}

func (c *collector) emitTypeAnnotation(node *sitter.Node) {
	switch node.Kind() {
	case "parameter":
		nameNode := declarationName(node)
		typeNode := typeNodeForDeclaration(node)
		if nameNode != nil && typeNode != nil {
			c.addTypeAnnotation(node, c.text(nameNode), normalizeSwiftType(c.text(typeNode)), scopeir.TypeSourceParameter)
		}
	case "property_declaration", "protocol_property_declaration":
		nameNode := declarationName(node)
		typeNode := typeNodeForDeclaration(node)
		if nameNode != nil && typeNode != nil {
			c.addTypeAnnotation(node, c.text(nameNode), normalizeSwiftType(c.text(typeNode)), scopeir.TypeSourceAnnotation)
		}
	}
}

func (c *collector) addTypeBinding(anchor *sitter.Node, name string, rawType string, source scopeir.TypeRefSource) {
	c.addTypeBindingRange(nodeRange(anchor), name, rawType, source)
}

func (c *collector) addTypeBindingRange(rng scopeir.Range, name string, rawType string, source scopeir.TypeRefSource) {
	rawType = normalizeSwiftType(rawType)
	if name == "" || rawType == "" {
		return
	}
	scopeID := c.innermostScopeID(rng)
	if scope := c.scopeByID(scopeID); scope != nil {
		scope.TypeBindings = append(scope.TypeBindings, scopeir.TypeBindingFact{
			Name: name,
			Type: scopeir.TypeRef{
				RawName:         rawType,
				DeclaredAtScope: scopeID,
				Source:          source,
			},
		})
	}
}

func (c *collector) addTypeAnnotation(anchor *sitter.Node, name string, rawType string, source scopeir.TypeRefSource) {
	rawType = normalizeSwiftType(rawType)
	if name == "" || rawType == "" {
		return
	}
	rng := nodeRange(anchor)
	c.typeAnnotations = append(c.typeAnnotations, scopeir.TypeAnnotationFact{
		FilePath: c.filePath,
		FileHash: c.fileHash,
		Name:     name,
		Range:    rng,
		InScope:  c.innermostScopeID(rng),
		Type: scopeir.TypeRef{
			RawName:         rawType,
			DeclaredAtScope: c.innermostScopeID(rng),
			Source:          source,
		},
	})
}

func (c *collector) inferredTypeFromValue(node *sitter.Node) string {
	for _, candidate := range valueNodesAfterName(node) {
		switch candidate.Kind() {
		case "call_expression":
			if nameNode := callName(candidate); nameNode != nil {
				return c.returnTypesByCallable[c.text(nameNode)]
			}
		default:
			if isIdentifierLike(candidate) {
				return c.returnTypesByCallable[c.text(candidate)]
			}
		}
	}
	return ""
}

func valueNodesAfterName(node *sitter.Node) []*sitter.Node {
	if node == nil {
		return nil
	}
	nameNode := declarationName(node)
	out := []*sitter.Node{}
	seenName := false
	for index := uint(0); index < node.NamedChildCount(); index++ {
		candidate := node.NamedChild(index)
		if candidate == nil {
			continue
		}
		if !seenName {
			if nameNode != nil && candidate.Id() == nameNode.Id() {
				seenName = true
			}
			continue
		}
		out = append(out, candidate)
	}
	return out
}
