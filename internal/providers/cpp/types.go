package cpp

import (
	sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

func (c *collector) emitTypeBinding(node *sitter.Node) {
	switch node.Kind() {
	case "parameter_declaration":
		c.emitNamedTypeBinding(node, scopeir.TypeSourceParameter)
	case "declaration":
		if c.insideFunction(node) && !hasFunctionDeclarator(node) {
			c.emitNamedTypeBinding(node, scopeir.TypeSourceAssignment)
		}
	case "field_declaration":
		if !hasFunctionDeclarator(node) {
			c.emitNamedTypeBinding(node, scopeir.TypeSourceAnnotation)
		}
	}
}

func (c *collector) emitTypeAnnotation(node *sitter.Node) {
	switch node.Kind() {
	case "parameter_declaration":
	case "declaration", "field_declaration":
		if hasFunctionDeclarator(node) {
			return
		}
	default:
		return
	}
	switch node.Kind() {
	case "parameter_declaration", "declaration", "field_declaration":
		typeNode := typeNodeForDeclaration(node)
		if typeNode == nil {
			return
		}
		typeName := normalizeCPPType(c.text(typeNode))
		for _, nameNode := range declaratorNames(node) {
			c.addTypeAnnotation(node, c.text(nameNode), typeName, scopeir.TypeSourceAnnotation)
		}
	}
}

func (c *collector) emitNamedTypeBinding(node *sitter.Node, source scopeir.TypeRefSource) {
	typeNode := typeNodeForDeclaration(node)
	if typeNode == nil {
		return
	}
	typeName := normalizeCPPType(c.text(typeNode))
	for _, nameNode := range declaratorNames(node) {
		c.addTypeBinding(node, c.text(nameNode), typeName, source)
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
