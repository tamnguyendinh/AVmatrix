package kotlin

import (
	sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

func (c *collector) emitTypeBinding(node *sitter.Node) {
	switch node.Kind() {
	case "parameter", "class_parameter":
		nameNode := declarationName(node)
		typeNode := typeNodeForDeclaration(node)
		if nameNode != nil && typeNode != nil {
			c.addTypeBinding(node, c.text(nameNode), normalizeKotlinType(c.text(typeNode)), scopeir.TypeSourceParameter)
		}
	case "property_declaration":
		variable := firstNamedChildOfType(node, "variable_declaration")
		nameNode := declarationName(variable)
		if nameNode == nil {
			return
		}
		typeName := normalizeKotlinType(c.text(typeNodeForDeclaration(variable)))
		if typeName == "" {
			typeName = c.inferredTypeFromValue(initializerForProperty(node))
		}
		if typeName != "" {
			c.addTypeBinding(node, c.text(nameNode), typeName, scopeir.TypeSourceAssignment)
		}
	}
}

func (c *collector) emitTypeAnnotation(node *sitter.Node) {
	switch node.Kind() {
	case "parameter", "class_parameter":
		nameNode := declarationName(node)
		typeNode := typeNodeForDeclaration(node)
		if nameNode != nil && typeNode != nil {
			c.addTypeAnnotation(node, c.text(nameNode), normalizeKotlinType(c.text(typeNode)), scopeir.TypeSourceParameter)
		}
	case "property_declaration":
		variable := firstNamedChildOfType(node, "variable_declaration")
		nameNode := declarationName(variable)
		typeName := normalizeKotlinType(c.text(typeNodeForDeclaration(variable)))
		if nameNode != nil && typeName != "" {
			c.addTypeAnnotation(node, c.text(nameNode), typeName, scopeir.TypeSourceAnnotation)
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
