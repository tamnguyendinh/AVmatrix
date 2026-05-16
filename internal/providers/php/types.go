package php

import (
	sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

func (c *collector) emitTypeBinding(node *sitter.Node) {
	switch node.Kind() {
	case "simple_parameter":
		nameNode := declarationName(node)
		typeNode := typeNodeForDeclaration(node)
		if nameNode != nil && typeNode != nil {
			c.addTypeBinding(node, c.text(nameNode), normalizePHPType(c.text(typeNode)), scopeir.TypeSourceParameter)
		}
	case "property_declaration":
		c.emitPropertyTypeBinding(node, scopeir.TypeSourceAnnotation)
	case "assignment_expression":
		left := child(node, "left")
		if left == nil || left.Kind() != "variable_name" {
			return
		}
		nameNode := declarationName(left)
		if nameNode == nil {
			return
		}
		typeName := c.inferredTypeFromValue(child(node, "right"))
		c.addTypeBinding(node, c.text(nameNode), typeName, scopeir.TypeSourceAssignment)
	}
}

func (c *collector) emitTypeAnnotation(node *sitter.Node) {
	switch node.Kind() {
	case "simple_parameter":
		nameNode := declarationName(node)
		typeNode := typeNodeForDeclaration(node)
		if nameNode != nil && typeNode != nil {
			c.addTypeAnnotation(node, c.text(nameNode), normalizePHPType(c.text(typeNode)), scopeir.TypeSourceParameter)
		}
	case "property_declaration":
		typeName := normalizePHPType(c.text(typeNodeForDeclaration(node)))
		for _, property := range directChildrenOfKind(node, "property_element") {
			nameNode := declarationName(property)
			if nameNode != nil && typeName != "" {
				c.addTypeAnnotation(property, c.text(nameNode), typeName, scopeir.TypeSourceAnnotation)
			}
		}
	case "assignment_expression":
		left := child(node, "left")
		if left == nil || left.Kind() != "variable_name" {
			return
		}
		nameNode := declarationName(left)
		typeName := c.inferredTypeFromValue(child(node, "right"))
		if nameNode != nil && typeName != "" {
			c.addTypeAnnotation(node, c.text(nameNode), typeName, scopeir.TypeSourceAssignment)
		}
	}
}

func (c *collector) emitPropertyTypeBinding(node *sitter.Node, source scopeir.TypeRefSource) {
	typeName := normalizePHPType(c.text(typeNodeForDeclaration(node)))
	for _, property := range directChildrenOfKind(node, "property_element") {
		nameNode := declarationName(property)
		if nameNode != nil && typeName != "" {
			c.addTypeBinding(property, c.text(nameNode), typeName, source)
		}
	}
}

func (c *collector) addTypeBinding(anchor *sitter.Node, name string, rawType string, source scopeir.TypeRefSource) {
	rawType = normalizePHPType(rawType)
	if name == "" || rawType == "" {
		return
	}
	rng := nodeRange(anchor)
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
	rawType = normalizePHPType(rawType)
	if name == "" || name == "this" || rawType == "" {
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
	if node == nil {
		return ""
	}
	switch node.Kind() {
	case "function_call_expression", "member_call_expression":
		nameNode := child(node, "function")
		if nameNode == nil {
			nameNode = child(node, "name")
		}
		if nameNode != nil {
			return c.returnTypesByCallable[c.text(nameNode)]
		}
	case "object_creation_expression":
		if nameNode := child(node, "name"); nameNode != nil {
			return normalizePHPType(c.text(nameNode))
		}
	}
	return ""
}
