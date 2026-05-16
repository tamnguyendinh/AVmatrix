package java

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
			c.addTypeBinding(node, c.text(nameNode), normalizeJavaType(c.text(typeNode)), scopeir.TypeSourceParameter)
		}
	case "local_variable_declaration":
		typeName := normalizeJavaType(c.text(typeNodeForDeclaration(node)))
		for _, declarator := range namedChildrenOfType(node, "variable_declarator") {
			nameNode := declarationName(declarator)
			if nameNode == nil {
				continue
			}
			resolvedType := typeName
			if resolvedType == "" || resolvedType == "var" {
				resolvedType = c.inferredTypeFromValue(initializerForDeclarator(declarator))
			}
			if resolvedType != "" {
				c.addTypeBinding(declarator, c.text(nameNode), resolvedType, scopeir.TypeSourceAssignment)
			}
		}
	}
}

func (c *collector) emitTypeAnnotation(node *sitter.Node) {
	switch node.Kind() {
	case "formal_parameter":
		nameNode := declarationName(node)
		typeNode := typeNodeForDeclaration(node)
		if nameNode != nil && typeNode != nil {
			c.addTypeAnnotation(node, c.text(nameNode), normalizeJavaType(c.text(typeNode)), scopeir.TypeSourceParameter)
		}
	case "field_declaration":
		typeName := normalizeJavaType(c.text(typeNodeForDeclaration(node)))
		for _, declarator := range namedChildrenOfType(node, "variable_declarator") {
			nameNode := declarationName(declarator)
			if nameNode != nil && typeName != "" {
				c.addTypeAnnotation(declarator, c.text(nameNode), typeName, scopeir.TypeSourceAnnotation)
			}
		}
	case "local_variable_declaration":
		typeName := normalizeJavaType(c.text(typeNodeForDeclaration(node)))
		for _, declarator := range namedChildrenOfType(node, "variable_declarator") {
			nameNode := declarationName(declarator)
			if nameNode != nil && typeName != "" && typeName != "var" {
				c.addTypeAnnotation(declarator, c.text(nameNode), typeName, scopeir.TypeSourceAnnotation)
			}
		}
	}
}

func (c *collector) addTypeBinding(anchor *sitter.Node, name string, rawType string, source scopeir.TypeRefSource) {
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
