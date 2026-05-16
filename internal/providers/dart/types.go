package dart

import (
	sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

func (c *collector) emitTypeBinding(node *sitter.Node) {
	switch node.Kind() {
	case "formal_parameter":
		nameNode := declarationName(node)
		typeNode := typeNodeForDeclaration(node)
		if nameNode != nil && typeNode != nil {
			c.addTypeBinding(node, c.text(nameNode), normalizeDartType(c.text(typeNode)), scopeir.TypeSourceParameter)
		}
	case "declaration":
		c.emitFieldTypeBinding(node, scopeir.TypeSourceAnnotation)
	case "initialized_variable_definition":
		if parentOfKind(node, "local_variable_declaration") == nil {
			return
		}
		nameNode := declarationName(node)
		typeName := normalizeDartType(c.text(typeNodeForDeclaration(node)))
		if typeName == "" {
			typeName = c.inferredTypeFromValue(node)
		}
		if nameNode != nil && typeName != "" {
			c.addTypeBinding(node, c.text(nameNode), typeName, scopeir.TypeSourceAssignment)
		}
	}
}

func (c *collector) emitTypeAnnotation(node *sitter.Node) {
	switch node.Kind() {
	case "formal_parameter":
		nameNode := declarationName(node)
		typeNode := typeNodeForDeclaration(node)
		if nameNode != nil && typeNode != nil {
			c.addTypeAnnotation(node, c.text(nameNode), normalizeDartType(c.text(typeNode)), scopeir.TypeSourceParameter)
		}
	case "declaration":
		if c.ownerTypeNameFor(node) == "" {
			return
		}
		typeName := normalizeDartType(c.text(typeNodeForDeclaration(node)))
		for _, item := range directChildrenOfKind(directChildOfKind(node, "initialized_identifier_list"), "initialized_identifier") {
			nameNode := declarationName(item)
			if nameNode != nil && typeName != "" {
				c.addTypeAnnotation(item, c.text(nameNode), typeName, scopeir.TypeSourceAnnotation)
			}
		}
	case "initialized_variable_definition":
		if parentOfKind(node, "local_variable_declaration") == nil {
			return
		}
		nameNode := declarationName(node)
		typeName := normalizeDartType(c.text(typeNodeForDeclaration(node)))
		if typeName == "" {
			typeName = c.inferredTypeFromValue(node)
		}
		if nameNode != nil && typeName != "" {
			c.addTypeAnnotation(node, c.text(nameNode), typeName, scopeir.TypeSourceAssignment)
		}
	}
}

func (c *collector) emitFieldTypeBinding(node *sitter.Node, source scopeir.TypeRefSource) {
	if c.ownerTypeNameFor(node) == "" {
		return
	}
	typeName := normalizeDartType(c.text(typeNodeForDeclaration(node)))
	for _, item := range directChildrenOfKind(directChildOfKind(node, "initialized_identifier_list"), "initialized_identifier") {
		nameNode := declarationName(item)
		if nameNode != nil && typeName != "" {
			c.addTypeBinding(item, c.text(nameNode), typeName, source)
		}
	}
}

func (c *collector) addTypeBinding(anchor *sitter.Node, name string, rawType string, source scopeir.TypeRefSource) {
	c.addTypeBindingRange(nodeRange(anchor), name, rawType, source)
}

func (c *collector) addTypeBindingRange(rng scopeir.Range, name string, rawType string, source scopeir.TypeRefSource) {
	rawType = normalizeDartType(rawType)
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
	rawType = normalizeDartType(rawType)
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
		if isIdentifierLike(candidate) {
			return c.returnTypesByCallable[c.text(candidate)]
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
