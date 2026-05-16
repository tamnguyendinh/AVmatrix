package csharp

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
			c.addTypeBinding(node, c.text(nameNode), normalizeCSharpType(c.text(typeNode)), scopeir.TypeSourceParameter)
		}
	case "local_declaration_statement":
		c.emitVariableDeclarationTypeBinding(firstNamedChildOfType(node, "variable_declaration"), scopeir.TypeSourceAssignment)
	case "field_declaration":
		c.emitVariableDeclarationTypeBinding(firstNamedChildOfType(node, "variable_declaration"), scopeir.TypeSourceAnnotation)
	}
}

func (c *collector) emitTypeAnnotation(node *sitter.Node) {
	switch node.Kind() {
	case "parameter":
		nameNode := declarationName(node)
		typeNode := typeNodeForDeclaration(node)
		if nameNode != nil && typeNode != nil {
			c.addTypeAnnotation(node, c.text(nameNode), normalizeCSharpType(c.text(typeNode)), scopeir.TypeSourceParameter)
		}
	case "local_declaration_statement", "field_declaration":
		declaration := firstNamedChildOfType(node, "variable_declaration")
		typeName := normalizeCSharpType(c.text(typeNodeForDeclaration(declaration)))
		for _, declarator := range directChildrenOfKind(declaration, "variable_declarator") {
			nameNode := declarationName(declarator)
			if nameNode != nil && typeName != "" {
				c.addTypeAnnotation(declarator, c.text(nameNode), typeName, scopeir.TypeSourceAnnotation)
			}
		}
	}
}

func (c *collector) emitVariableDeclarationTypeBinding(node *sitter.Node, source scopeir.TypeRefSource) {
	typeName := normalizeCSharpType(c.text(typeNodeForDeclaration(node)))
	for _, declarator := range directChildrenOfKind(node, "variable_declarator") {
		nameNode := declarationName(declarator)
		if nameNode != nil && typeName != "" {
			c.addTypeBinding(declarator, c.text(nameNode), typeName, source)
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
