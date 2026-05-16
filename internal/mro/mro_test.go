package mro

import (
	"math"
	"testing"

	"github.com/tamnguyendinh/avmatrix-go/internal/graph"
	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

func TestApplyConvertedOverrideTopology(t *testing.T) {
	t.Run("leftmost base wins when both bases define the same method", func(t *testing.T) {
		g := graph.New()
		child := addMRONode(g, scopeir.NodeClass, "D", "cpp")
		baseA := addMRONode(g, scopeir.NodeClass, "B", "cpp")
		baseB := addMRONode(g, scopeir.NodeClass, "C", "cpp")
		addMRONode(g, scopeir.NodeClass, "A", "cpp")
		addMRORel(g, graph.RelExtends, baseA, graph.GenerateID(string(scopeir.NodeClass), "A"))
		addMRORel(g, graph.RelExtends, baseB, graph.GenerateID(string(scopeir.NodeClass), "A"))
		addMRORel(g, graph.RelExtends, child, baseA)
		addMRORel(g, graph.RelExtends, child, baseB)
		addMROMethod(g, graph.GenerateID(string(scopeir.NodeClass), "A"), "foo", 0, nil)
		baseAFoo := addMROMethod(g, baseA, "foo", 0, nil)
		baseBFoo := addMROMethod(g, baseB, "foo", 0, nil)

		result := Apply(g)

		if result.Metrics.ClassesAnalyzed != 3 || result.Metrics.MethodOverrides != 1 {
			t.Fatalf("metrics = %#v", result.Metrics)
		}
		requireMRORelationship(t, g, graph.RelMethodOverrides, child, baseAFoo, "first definition: B::foo")
		requireNoMRORelationship(t, g, graph.RelMethodOverrides, child, baseBFoo)
	})

	t.Run("diamond with only common ancestor method is not ambiguous", func(t *testing.T) {
		g := graph.New()
		child := addMRONode(g, scopeir.NodeClass, "D", "cpp")
		baseA := addMRONode(g, scopeir.NodeClass, "B", "cpp")
		baseB := addMRONode(g, scopeir.NodeClass, "C", "cpp")
		root := addMRONode(g, scopeir.NodeClass, "A", "cpp")
		addMRORel(g, graph.RelExtends, baseA, root)
		addMRORel(g, graph.RelExtends, baseB, root)
		addMRORel(g, graph.RelExtends, child, baseA)
		addMRORel(g, graph.RelExtends, child, baseB)
		addMROMethod(g, root, "foo", 0, nil)

		result := Apply(g)

		if result.Metrics.MethodOverrides != 0 || result.Metrics.AmbiguityCount != 0 {
			t.Fatalf("metrics = %#v", result.Metrics)
		}
	})

	t.Run("own method shadows ancestor collision", func(t *testing.T) {
		g := graph.New()
		child := addMRONode(g, scopeir.NodeClass, "Child", "cpp")
		baseA := addMRONode(g, scopeir.NodeClass, "BaseA", "cpp")
		baseB := addMRONode(g, scopeir.NodeClass, "BaseB", "cpp")
		addMRORel(g, graph.RelExtends, child, baseA)
		addMRORel(g, graph.RelExtends, child, baseB)
		addMROMethod(g, baseA, "foo", 0, nil)
		addMROMethod(g, baseB, "foo", 0, nil)
		addMROMethod(g, child, "foo", 0, nil)

		result := Apply(g)

		if result.Metrics.MethodOverrides != 0 {
			t.Fatalf("metrics = %#v", result.Metrics)
		}
	})

	t.Run("property collisions do not emit method overrides", func(t *testing.T) {
		g := graph.New()
		child := addMRONode(g, scopeir.NodeClass, "Child", "typescript")
		baseA := addMRONode(g, scopeir.NodeClass, "ParentA", "typescript")
		baseB := addMRONode(g, scopeir.NodeClass, "ParentB", "typescript")
		addMRORel(g, graph.RelExtends, child, baseA)
		addMRORel(g, graph.RelExtends, child, baseB)
		addMROProperty(g, baseA, "name")
		addMROProperty(g, baseB, "name")

		result := Apply(g)

		if result.Metrics.MethodOverrides != 0 {
			t.Fatalf("metrics = %#v", result.Metrics)
		}
	})
}

func TestApplyConvertedMethodImplements(t *testing.T) {
	t.Run("concrete class method implements interface method", func(t *testing.T) {
		g := graph.New()
		service := addMRONode(g, scopeir.NodeClass, "Service", "java")
		contract := addMRONode(g, scopeir.NodeInterface, "Saver", "java")
		serviceSave := addMROMethod(g, service, "save", 1, []string{"User"})
		contractSave := addMROMethod(g, contract, "save", 1, []string{"User"})
		addMRORel(g, graph.RelImplements, service, contract)

		result := Apply(g)

		if result.Metrics.MethodImplementsEdges != 1 {
			t.Fatalf("metrics = %#v", result.Metrics)
		}
		requireMRORelationship(t, g, graph.RelMethodImplements, serviceSave, contractSave, "")
		requireRelationshipConfidence(t, g, graph.RelMethodImplements, serviceSave, contractSave, 1)
	})

	t.Run("rust struct method implements trait method", func(t *testing.T) {
		g := graph.New()
		point := addMRONode(g, scopeir.NodeStruct, "Point", "rust")
		display := addMRONode(g, scopeir.NodeTrait, "Display", "rust")
		pointFmt := addMROMethod(g, point, "fmt", 1, []string{"Formatter"})
		displayFmt := addMROMethod(g, display, "fmt", 1, []string{"Formatter"})
		addMRORel(g, graph.RelImplements, point, display)

		Apply(g)

		requireMRORelationship(t, g, graph.RelMethodImplements, pointFmt, displayFmt, "")
	})

	t.Run("overloaded interface methods match by parameter types", func(t *testing.T) {
		g := graph.New()
		service := addMRONode(g, scopeir.NodeClass, "Service", "java")
		contract := addMRONode(g, scopeir.NodeInterface, "Repository", "java")
		saveUser := addMROMethodWithKey(g, service, "save", "Service.save#1~User", 1, []string{"User"}, false)
		saveOrder := addMROMethodWithKey(g, service, "save", "Service.save#1~Order", 1, []string{"Order"}, false)
		ifaceUser := addMROMethodWithKey(g, contract, "save", "Repository.save#1~User", 1, []string{"User"}, false)
		ifaceOrder := addMROMethodWithKey(g, contract, "save", "Repository.save#1~Order", 1, []string{"Order"}, false)
		addMRORel(g, graph.RelImplements, service, contract)

		result := Apply(g)

		if result.Metrics.MethodImplementsEdges != 2 {
			t.Fatalf("metrics = %#v", result.Metrics)
		}
		requireMRORelationship(t, g, graph.RelMethodImplements, saveUser, ifaceUser, "")
		requireMRORelationship(t, g, graph.RelMethodImplements, saveOrder, ifaceOrder, "")
	})

	t.Run("arity mismatch prevents false method implements edge", func(t *testing.T) {
		g := graph.New()
		service := addMRONode(g, scopeir.NodeClass, "Service", "java")
		contract := addMRONode(g, scopeir.NodeInterface, "Repository", "java")
		addMROMethod(g, service, "save", 1, nil)
		addMROMethod(g, contract, "save", 2, nil)
		addMRORel(g, graph.RelImplements, service, contract)

		result := Apply(g)

		if result.Metrics.MethodImplementsEdges != 0 {
			t.Fatalf("metrics = %#v", result.Metrics)
		}
	})

	t.Run("ambiguous same-name same-arity own candidates emit no edge", func(t *testing.T) {
		g := graph.New()
		service := addMRONode(g, scopeir.NodeClass, "Service", "java")
		contract := addMRONode(g, scopeir.NodeInterface, "Repository", "java")
		addMROMethodWithKey(g, service, "save", "Service.save#1~User", 1, nil, false)
		addMROMethodWithKey(g, service, "save", "Service.save#1~Order", 1, nil, false)
		addMROMethod(g, contract, "save", 1, nil)
		addMRORel(g, graph.RelImplements, service, contract)

		result := Apply(g)

		if result.Metrics.MethodImplementsEdges != 0 {
			t.Fatalf("metrics = %#v", result.Metrics)
		}
	})

	t.Run("abstract class method does not satisfy interface contract", func(t *testing.T) {
		g := graph.New()
		class := addMRONode(g, scopeir.NodeClass, "C", "java")
		contract := addMRONode(g, scopeir.NodeInterface, "I", "java")
		addMROMethodWithKey(g, class, "foo", "C.foo#0", 0, nil, true)
		addMROMethod(g, contract, "foo", 0, nil)
		addMRORel(g, graph.RelImplements, class, contract)

		result := Apply(g)

		if result.Metrics.MethodImplementsEdges != 0 {
			t.Fatalf("metrics = %#v", result.Metrics)
		}
	})

	t.Run("interface properties are skipped", func(t *testing.T) {
		g := graph.New()
		class := addMRONode(g, scopeir.NodeClass, "C", "java")
		contract := addMRONode(g, scopeir.NodeInterface, "I", "java")
		addMROMethod(g, class, "name", 0, nil)
		addMROProperty(g, contract, "name")
		addMRORel(g, graph.RelImplements, class, contract)

		result := Apply(g)

		if result.Metrics.MethodImplementsEdges != 0 {
			t.Fatalf("metrics = %#v", result.Metrics)
		}
	})
}

func TestApplyConvertedTransitiveMethodImplements(t *testing.T) {
	t.Run("transitive interface chain links class method to each contract", func(t *testing.T) {
		g := graph.New()
		class := addMRONode(g, scopeir.NodeClass, "C", "java")
		childIface := addMRONode(g, scopeir.NodeInterface, "B", "java")
		rootIface := addMRONode(g, scopeir.NodeInterface, "A", "java")
		classFoo := addMROMethod(g, class, "foo", 0, nil)
		childFoo := addMROMethod(g, childIface, "foo", 0, nil)
		rootFoo := addMROMethod(g, rootIface, "foo", 0, nil)
		addMRORel(g, graph.RelImplements, class, childIface)
		addMRORel(g, graph.RelExtends, childIface, rootIface)

		result := Apply(g)

		if result.Metrics.MethodImplementsEdges != 2 {
			t.Fatalf("metrics = %#v", result.Metrics)
		}
		requireMRORelationship(t, g, graph.RelMethodImplements, classFoo, childFoo, "")
		requireMRORelationship(t, g, graph.RelMethodImplements, classFoo, rootFoo, "")
	})

	t.Run("diamond interface ancestor deduplicates method implements", func(t *testing.T) {
		g := graph.New()
		class := addMRONode(g, scopeir.NodeClass, "E", "java")
		left := addMRONode(g, scopeir.NodeInterface, "B", "java")
		right := addMRONode(g, scopeir.NodeInterface, "C", "java")
		root := addMRONode(g, scopeir.NodeInterface, "A", "java")
		classFoo := addMROMethod(g, class, "foo", 0, nil)
		rootFoo := addMROMethod(g, root, "foo", 0, nil)
		addMRORel(g, graph.RelImplements, class, left)
		addMRORel(g, graph.RelImplements, class, right)
		addMRORel(g, graph.RelExtends, left, root)
		addMRORel(g, graph.RelExtends, right, root)

		Apply(g)

		if got := countMRORelationships(g, graph.RelMethodImplements, classFoo, rootFoo); got != 1 {
			t.Fatalf("METHOD_IMPLEMENTS count = %d, want 1", got)
		}
	})

	t.Run("class-only inheritance chain does not emit method implements", func(t *testing.T) {
		g := graph.New()
		child := addMRONode(g, scopeir.NodeClass, "Child", "java")
		base := addMRONode(g, scopeir.NodeClass, "Base", "java")
		addMROMethod(g, child, "foo", 0, nil)
		addMROMethod(g, base, "foo", 0, nil)
		addMRORel(g, graph.RelExtends, child, base)

		result := Apply(g)

		if result.Metrics.MethodImplementsEdges != 0 {
			t.Fatalf("metrics = %#v", result.Metrics)
		}
	})
}

func TestApplyConvertedInheritedImplementations(t *testing.T) {
	t.Run("inherited class method satisfies interface contract", func(t *testing.T) {
		g := graph.New()
		class := addMRONode(g, scopeir.NodeClass, "C", "java")
		base := addMRONode(g, scopeir.NodeClass, "Base", "java")
		contract := addMRONode(g, scopeir.NodeInterface, "I", "java")
		baseFoo := addMROMethod(g, base, "foo", 0, nil)
		ifaceFoo := addMROMethod(g, contract, "foo", 0, nil)
		addMRORel(g, graph.RelExtends, class, base)
		addMRORel(g, graph.RelImplements, class, contract)

		Apply(g)

		requireMRORelationship(t, g, graph.RelMethodImplements, baseFoo, ifaceFoo, "")
	})

	t.Run("nearest inherited class method wins", func(t *testing.T) {
		g := graph.New()
		class := addMRONode(g, scopeir.NodeClass, "C", "java")
		base := addMRONode(g, scopeir.NodeClass, "B", "java")
		grand := addMRONode(g, scopeir.NodeClass, "A", "java")
		contract := addMRONode(g, scopeir.NodeInterface, "I", "java")
		baseFoo := addMROMethod(g, base, "foo", 0, nil)
		grandFoo := addMROMethod(g, grand, "foo", 0, nil)
		ifaceFoo := addMROMethod(g, contract, "foo", 0, nil)
		addMRORel(g, graph.RelExtends, class, base)
		addMRORel(g, graph.RelExtends, base, grand)
		addMRORel(g, graph.RelImplements, class, contract)

		Apply(g)

		requireMRORelationship(t, g, graph.RelMethodImplements, baseFoo, ifaceFoo, "")
		requireNoMRORelationship(t, g, graph.RelMethodImplements, grandFoo, ifaceFoo)
	})

	t.Run("two extends parents with matching methods are ambiguous", func(t *testing.T) {
		g := graph.New()
		class := addMRONode(g, scopeir.NodeClass, "C", "java")
		baseA := addMRONode(g, scopeir.NodeClass, "A", "java")
		baseB := addMRONode(g, scopeir.NodeClass, "B", "java")
		contract := addMRONode(g, scopeir.NodeInterface, "I", "java")
		addMROMethod(g, baseA, "foo", 0, nil)
		addMROMethod(g, baseB, "foo", 0, nil)
		addMROMethod(g, contract, "foo", 0, nil)
		addMRORel(g, graph.RelExtends, class, baseA)
		addMRORel(g, graph.RelExtends, class, baseB)
		addMRORel(g, graph.RelImplements, class, contract)

		result := Apply(g)

		if result.Metrics.MethodImplementsEdges != 0 {
			t.Fatalf("metrics = %#v", result.Metrics)
		}
	})

	t.Run("interface default method satisfies grandparent interface contract", func(t *testing.T) {
		g := graph.New()
		class := addMRONode(g, scopeir.NodeClass, "C", "java")
		childIface := addMRONode(g, scopeir.NodeInterface, "I2", "java")
		rootIface := addMRONode(g, scopeir.NodeInterface, "I1", "java")
		defaultBar := addMROMethod(g, childIface, "bar", 0, nil)
		abstractBar := addMROMethodWithKey(g, rootIface, "bar", "I1.bar#0", 0, nil, true)
		addMRORel(g, graph.RelImplements, class, childIface)
		addMRORel(g, graph.RelExtends, childIface, rootIface)

		Apply(g)

		requireMRORelationship(t, g, graph.RelMethodImplements, defaultBar, abstractBar, "")
	})

	t.Run("implements default methods are ambiguous when two interfaces provide them", func(t *testing.T) {
		g := graph.New()
		class := addMRONode(g, scopeir.NodeClass, "C", "java")
		left := addMRONode(g, scopeir.NodeInterface, "Left", "java")
		right := addMRONode(g, scopeir.NodeInterface, "Right", "java")
		root := addMRONode(g, scopeir.NodeInterface, "Root", "java")
		rootProcess := addMROMethodWithKey(g, root, "process", "Root.process#0", 0, nil, true)
		addMROMethod(g, left, "process", 0, nil)
		addMROMethod(g, right, "process", 0, nil)
		addMRORel(g, graph.RelImplements, class, left)
		addMRORel(g, graph.RelImplements, class, right)
		addMRORel(g, graph.RelExtends, left, root)
		addMRORel(g, graph.RelExtends, right, root)

		Apply(g)

		for _, rel := range g.Relationships {
			if rel.Type == graph.RelMethodImplements && rel.TargetID == rootProcess {
				t.Fatalf("unexpected class fallback edge for ambiguous interface defaults: %#v", rel)
			}
		}
	})

	t.Run("dart implements class does not inherit concrete bodies", func(t *testing.T) {
		g := graph.New()
		class := addMRONode(g, scopeir.NodeClass, "DartImpl", "dart")
		base := addMRONode(g, scopeir.NodeClass, "AbstractBase", "dart")
		contract := addMRONode(g, scopeir.NodeInterface, "I", "dart")
		addMROMethod(g, base, "foo", 0, nil)
		ifaceFoo := addMROMethod(g, contract, "foo", 0, nil)
		addMRORel(g, graph.RelImplements, class, base)
		addMRORel(g, graph.RelImplements, class, contract)

		Apply(g)

		requireNoMRORelationshipFromTarget(t, g, graph.RelMethodImplements, ifaceFoo)
	})
}

func TestApplyConvertedConfidenceTiering(t *testing.T) {
	t.Run("fully typed and arity-only matches are confidence one", func(t *testing.T) {
		g := graph.New()
		class := addMRONode(g, scopeir.NodeClass, "C", "java")
		contract := addMRONode(g, scopeir.NodeInterface, "I", "java")
		typedClass := addMROMethod(g, class, "typed", 2, []string{"int", "String"})
		typedIface := addMROMethod(g, contract, "typed", 2, []string{"int", "String"})
		arityClass := addMROMethod(g, class, "arity", 2, nil)
		arityIface := addMROMethod(g, contract, "arity", 2, nil)
		addMRORel(g, graph.RelImplements, class, contract)

		Apply(g)

		requireRelationshipConfidence(t, g, graph.RelMethodImplements, typedClass, typedIface, 1)
		requireRelationshipConfidence(t, g, graph.RelMethodImplements, arityClass, arityIface, 1)
	})

	t.Run("missing arity on either side is lenient confidence", func(t *testing.T) {
		g := graph.New()
		class := addMRONode(g, scopeir.NodeClass, "C", "java")
		contract := addMRONode(g, scopeir.NodeInterface, "I", "java")
		lenientClass := addMROMethodWithOptionalCount(g, class, "loose", "C.loose", nil, nil, false)
		lenientIface := addMROMethodWithOptionalCount(g, contract, "loose", "I.loose", nil, nil, false)
		halfClass := addMROMethodWithOptionalCount(g, class, "half", "C.half", nil, nil, false)
		halfIface := addMROMethodWithOptionalCount(g, contract, "half", "I.half#2", intPtr(2), nil, false)
		addMRORel(g, graph.RelImplements, class, contract)

		Apply(g)

		requireRelationshipConfidence(t, g, graph.RelMethodImplements, lenientClass, lenientIface, 0.7)
		requireRelationshipConfidence(t, g, graph.RelMethodImplements, halfClass, halfIface, 0.7)
	})
}

func TestApplyConvertedCyclesAndEmptyGraph(t *testing.T) {
	t.Run("cyclic inheritance does not loop forever", func(t *testing.T) {
		g := graph.New()
		a := addMRONode(g, scopeir.NodeClass, "A", "python")
		b := addMRONode(g, scopeir.NodeClass, "B", "python")
		addMRORel(g, graph.RelExtends, a, b)
		addMRORel(g, graph.RelExtends, b, a)
		addMROMethod(g, a, "foo", 0, nil)
		addMROMethod(g, b, "foo", 0, nil)

		result := Apply(g)

		if result.Metrics.ClassesAnalyzed != 2 {
			t.Fatalf("metrics = %#v", result.Metrics)
		}
	})

	t.Run("very deep single inheritance chain is bounded by visited set", func(t *testing.T) {
		g := graph.New()
		previous := addMRONode(g, scopeir.NodeClass, "Level0", "python")
		for i := 1; i < 50; i++ {
			current := addMRONode(g, scopeir.NodeClass, "Level"+itoa(i), "python")
			addMRORel(g, graph.RelExtends, previous, current)
			previous = current
		}
		addMROMethod(g, previous, "terminal", 0, nil)

		result := Apply(g)

		if result.Metrics.ClassesAnalyzed != 49 {
			t.Fatalf("metrics = %#v", result.Metrics)
		}
	})

	t.Run("empty graph returns empty metrics", func(t *testing.T) {
		result := Apply(graph.New())
		if result.Metrics != (Metrics{}) {
			t.Fatalf("metrics = %#v", result.Metrics)
		}
	})
}

func addMRONode(g *graph.Graph, label scopeir.NodeLabel, name string, language string) string {
	id := graph.GenerateID(string(label), name)
	g.AddNode(graph.Node{
		ID:    id,
		Label: label,
		Properties: graph.NodeProperties{
			"name":     name,
			"filePath": "src/" + name,
			"language": language,
		},
	})
	return id
}

func addMROMethod(g *graph.Graph, ownerID string, name string, parameterCount int, parameterTypes []string) string {
	return addMROMethodWithKey(g, ownerID, name, ownerID+"."+name+"#"+itoa(parameterCount), parameterCount, parameterTypes, false)
}

func addMROMethodWithKey(g *graph.Graph, ownerID string, name string, key string, parameterCount int, parameterTypes []string, abstract bool) string {
	return addMROMethodWithOptionalCount(g, ownerID, name, key, &parameterCount, parameterTypes, abstract)
}

func addMROMethodWithOptionalCount(g *graph.Graph, ownerID string, name string, key string, parameterCount *int, parameterTypes []string, abstract bool) string {
	id := graph.GenerateID(string(scopeir.NodeMethod), key)
	props := graph.NodeProperties{
		"name":     name,
		"filePath": key,
	}
	if parameterCount != nil {
		props["parameterCount"] = *parameterCount
	}
	if parameterTypes != nil {
		props["parameterTypes"] = parameterTypes
	}
	if abstract {
		props["isAbstract"] = true
	}
	g.AddNode(graph.Node{ID: id, Label: scopeir.NodeMethod, Properties: props})
	addMRORel(g, graph.RelHasMethod, ownerID, id)
	return id
}

func addMROProperty(g *graph.Graph, ownerID string, name string) string {
	id := graph.GenerateID(string(scopeir.NodeProperty), ownerID+"."+name)
	g.AddNode(graph.Node{
		ID:    id,
		Label: scopeir.NodeProperty,
		Properties: graph.NodeProperties{
			"name":     name,
			"filePath": ownerID + "." + name,
		},
	})
	addMRORel(g, graph.RelHasProperty, ownerID, id)
	return id
}

func addMRORel(g *graph.Graph, relType graph.RelationshipType, sourceID string, targetID string) {
	g.AddRelationship(graph.Relationship{
		ID:         graph.GenerateID(string(relType), sourceID+"->"+targetID),
		SourceID:   sourceID,
		TargetID:   targetID,
		Type:       relType,
		Confidence: 1,
	})
}

func requireMRORelationship(t *testing.T, g *graph.Graph, relType graph.RelationshipType, sourceID string, targetID string, reason string) {
	t.Helper()
	for _, rel := range g.Relationships {
		if rel.Type == relType && rel.SourceID == sourceID && rel.TargetID == targetID && rel.Reason == reason {
			return
		}
	}
	t.Fatalf("missing %s %s -> %s reason %q", relType, sourceID, targetID, reason)
}

func requireNoMRORelationship(t *testing.T, g *graph.Graph, relType graph.RelationshipType, sourceID string, targetID string) {
	t.Helper()
	for _, rel := range g.Relationships {
		if rel.Type == relType && rel.SourceID == sourceID && rel.TargetID == targetID {
			t.Fatalf("unexpected %s %s -> %s", relType, sourceID, targetID)
		}
	}
}

func requireNoMRORelationshipFromTarget(t *testing.T, g *graph.Graph, relType graph.RelationshipType, targetID string) {
	t.Helper()
	for _, rel := range g.Relationships {
		if rel.Type == relType && rel.TargetID == targetID {
			t.Fatalf("unexpected %s target %s: %#v", relType, targetID, rel)
		}
	}
}

func countMRORelationships(g *graph.Graph, relType graph.RelationshipType, sourceID string, targetID string) int {
	count := 0
	for _, rel := range g.Relationships {
		if rel.Type == relType && rel.SourceID == sourceID && rel.TargetID == targetID {
			count++
		}
	}
	return count
}

func requireRelationshipConfidence(t *testing.T, g *graph.Graph, relType graph.RelationshipType, sourceID string, targetID string, confidence float64) {
	t.Helper()
	for _, rel := range g.Relationships {
		if rel.Type == relType && rel.SourceID == sourceID && rel.TargetID == targetID {
			if math.Abs(rel.Confidence-confidence) > 0.0001 {
				t.Fatalf("confidence = %f, want %f for %#v", rel.Confidence, confidence, rel)
			}
			return
		}
	}
	t.Fatalf("missing %s %s -> %s", relType, sourceID, targetID)
}

func intPtr(value int) *int {
	return &value
}

func itoa(value int) string {
	if value == 0 {
		return "0"
	}
	digits := [20]byte{}
	index := len(digits)
	for value > 0 {
		index--
		digits[index] = byte('0' + value%10)
		value /= 10
	}
	return string(digits[index:])
}
