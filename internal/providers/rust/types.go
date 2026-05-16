package rust

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
			c.addTypeBinding(node, c.text(nameNode), normalizeRustType(c.text(typeNode)), scopeir.TypeSourceParameter)
		}
	case "field_declaration":
		typeNode := typeNodeForDeclaration(node)
		for _, nameNode := range declaratorNames(node) {
			if typeNode != nil {
				c.addTypeBinding(node, c.text(nameNode), normalizeRustType(c.text(typeNode)), scopeir.TypeSourceAnnotation)
			}
		}
	case "let_declaration":
		typeName := normalizeRustType(c.text(typeNodeForDeclaration(node)))
		if typeName == "" {
			typeName = c.inferredTypeFromValue(child(node, "value"))
		}
		for _, nameNode := range declaratorNames(node) {
			c.addTypeBinding(node, c.text(nameNode), typeName, scopeir.TypeSourceAssignment)
		}
	}
}

func (c *collector) emitTypeAnnotation(node *sitter.Node) {
	switch node.Kind() {
	case "parameter", "field_declaration", "let_declaration":
		typeName := normalizeRustType(c.text(typeNodeForDeclaration(node)))
		if typeName == "" && node.Kind() == "let_declaration" {
			typeName = c.inferredTypeFromValue(child(node, "value"))
		}
		for _, nameNode := range declaratorNames(node) {
			c.addTypeAnnotation(node, c.text(nameNode), typeName, scopeir.TypeSourceAnnotation)
		}
	}
}

func (c *collector) addTypeBinding(anchor *sitter.Node, name string, rawType string, source scopeir.TypeRefSource) {
	rawType = normalizeRustType(rawType)
	if name == "" || name == "_" || rawType == "" {
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
	rawType = normalizeRustType(rawType)
	if name == "" || name == "_" || rawType == "" {
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
	case "call_expression":
		fn := child(node, "function")
		if fn == nil {
			fn = node.NamedChild(0)
		}
		if name := lastIdentifierLikeChild(fn); name != nil {
			return c.returnTypesByCallable[c.text(name)]
		}
	case "struct_expression":
		if typeNode := child(node, "type"); typeNode != nil {
			return normalizeRustType(c.text(typeNode))
		}
	}
	return ""
}
