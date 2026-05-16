package python

import (
	sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

func (c *collector) emitTypeBinding(node *sitter.Node) {
	switch node.Kind() {
	case "typed_parameter", "default_parameter", "typed_default_parameter":
		name, typeName := parameterNameAndType(c, node)
		if name != "" && typeName != "" {
			c.addTypeBinding(node, name, typeName, scopeir.TypeSourceParameter)
		}
	case "assignment":
		left := firstAssignmentTarget(node)
		if left == nil || left.Kind() != "identifier" {
			return
		}
		typeName := assignmentTypeName(c, node)
		source := scopeir.TypeSourceAnnotation
		if typeName == "" {
			typeName = c.inferredTypeFromValue(assignmentValue(node))
			source = scopeir.TypeSourceConstructor
		}
		if typeName != "" {
			c.addTypeBinding(node, c.text(left), typeName, source)
		}
	}
}

func (c *collector) emitTypeAnnotation(node *sitter.Node) {
	switch node.Kind() {
	case "typed_parameter", "default_parameter", "typed_default_parameter":
		name, typeName := parameterNameAndType(c, node)
		if name != "" && typeName != "" {
			c.addTypeAnnotation(node, name, typeName, scopeir.TypeSourceParameter)
		}
	case "assignment":
		left := firstAssignmentTarget(node)
		typeName := assignmentTypeName(c, node)
		if left != nil && left.Kind() == "identifier" && typeName != "" {
			c.addTypeAnnotation(node, c.text(left), typeName, scopeir.TypeSourceAnnotation)
		}
	case "function_definition":
		typeName := returnTypeNameForCallable(c, node)
		nameNode := firstNamedChildOfType(node, "identifier")
		if nameNode != nil && typeName != "" {
			c.addTypeAnnotation(returnTypeNode(node), c.text(nameNode), typeName, scopeir.TypeSourceReturn)
		}
	}
}

func (c *collector) addTypeBinding(anchor *sitter.Node, name string, typeName string, source scopeir.TypeRefSource) {
	if name == "" || typeName == "" {
		return
	}
	rng := nodeRange(anchor)
	scopeID := c.innermostScopeID(rng)
	if scope := c.scopeByID(scopeID); scope != nil {
		scope.TypeBindings = append(scope.TypeBindings, scopeir.TypeBindingFact{
			Name: name,
			Type: scopeir.TypeRef{
				RawName:         typeName,
				DeclaredAtScope: scopeID,
				Source:          source,
			},
		})
	}
}

func (c *collector) addTypeAnnotation(anchor *sitter.Node, name string, typeName string, source scopeir.TypeRefSource) {
	if anchor == nil || name == "" || typeName == "" {
		return
	}
	rng := nodeRange(anchor)
	scopeID := c.innermostScopeID(rng)
	c.typeAnnotations = append(c.typeAnnotations, scopeir.TypeAnnotationFact{
		FilePath: c.filePath,
		FileHash: c.fileHash,
		Name:     name,
		Range:    rng,
		InScope:  scopeID,
		Type: scopeir.TypeRef{
			RawName:         typeName,
			DeclaredAtScope: scopeID,
			Source:          source,
		},
	})
}
