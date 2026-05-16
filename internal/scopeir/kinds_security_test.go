package scopeir

import "testing"

func TestNodeLabelSecurityContract(t *testing.T) {
	known := map[NodeLabel]bool{
		NodeProject:     true,
		NodePackage:     true,
		NodeModule:      true,
		NodeFolder:      true,
		NodeFile:        true,
		NodeClass:       true,
		NodeFunction:    true,
		NodeMethod:      true,
		NodeVariable:    true,
		NodeInterface:   true,
		NodeEnum:        true,
		NodeDecorator:   true,
		NodeImport:      true,
		NodeType:        true,
		NodeCodeElement: true,
		NodeCommunity:   true,
		NodeProcess:     true,
		NodeStruct:      true,
		NodeMacro:       true,
		NodeTypedef:     true,
		NodeUnion:       true,
		NodeNamespace:   true,
		NodeTrait:       true,
		NodeImpl:        true,
		NodeTypeAlias:   true,
		NodeConst:       true,
		NodeStatic:      true,
		NodeProperty:    true,
		NodeRecord:      true,
		NodeDelegate:    true,
		NodeAnnotation:  true,
		NodeConstructor: true,
		NodeTemplate:    true,
		NodeSection:     true,
		NodeRoute:       true,
		NodeTool:        true,
	}

	for _, label := range []NodeLabel{
		NodeFile,
		NodeFolder,
		NodeFunction,
		NodeClass,
		NodeInterface,
		NodeMethod,
		NodeCodeElement,
		NodeCommunity,
		NodeProcess,
		NodeStruct,
		NodeEnum,
		NodeMacro,
		NodeTrait,
		NodeImpl,
		NodeNamespace,
	} {
		if !known[label] {
			t.Fatalf("known node labels missing %s", label)
		}
	}
	for _, label := range []NodeLabel{"InvalidType", "function"} {
		if known[label] {
			t.Fatalf("known node labels unexpectedly include %s", label)
		}
	}
}
