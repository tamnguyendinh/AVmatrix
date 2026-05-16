package graph

import (
	"sort"

	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

type RelationshipType string

const (
	RelContains         RelationshipType = "CONTAINS"
	RelCalls            RelationshipType = "CALLS"
	RelInherits         RelationshipType = "INHERITS"
	RelMethodOverrides  RelationshipType = "METHOD_OVERRIDES"
	RelMethodImplements RelationshipType = "METHOD_IMPLEMENTS"
	RelImports          RelationshipType = "IMPORTS"
	RelUses             RelationshipType = "USES"
	RelDefines          RelationshipType = "DEFINES"
	RelDecorates        RelationshipType = "DECORATES"
	RelImplements       RelationshipType = "IMPLEMENTS"
	RelExtends          RelationshipType = "EXTENDS"
	RelHasMethod        RelationshipType = "HAS_METHOD"
	RelHasProperty      RelationshipType = "HAS_PROPERTY"
	RelAccesses         RelationshipType = "ACCESSES"
	RelMemberOf         RelationshipType = "MEMBER_OF"
	RelStepInProcess    RelationshipType = "STEP_IN_PROCESS"
	RelHandlesRoute     RelationshipType = "HANDLES_ROUTE"
	RelFetches          RelationshipType = "FETCHES"
	RelHandlesTool      RelationshipType = "HANDLES_TOOL"
	RelEntryPointOf     RelationshipType = "ENTRY_POINT_OF"
	RelWraps            RelationshipType = "WRAPS"
	RelQueries          RelationshipType = "QUERIES"
)

type NodeProperties map[string]any

type Node struct {
	ID         string            `json:"id"`
	Label      scopeir.NodeLabel `json:"label"`
	Properties NodeProperties    `json:"properties"`
}

type Evidence struct {
	Kind   string  `json:"kind"`
	Weight float64 `json:"weight"`
	Note   string  `json:"note,omitempty"`
}

type Relationship struct {
	ID               string           `json:"id"`
	SourceID         string           `json:"sourceId"`
	TargetID         string           `json:"targetId"`
	Type             RelationshipType `json:"type"`
	Confidence       float64          `json:"confidence"`
	Reason           string           `json:"reason"`
	Step             *int             `json:"step,omitempty"`
	ResolutionSource string           `json:"resolutionSource,omitempty"`
	FileHash         string           `json:"fileHash,omitempty"`
	Evidence         []Evidence       `json:"evidence,omitempty"`
}

type Graph struct {
	Nodes         []Node         `json:"nodes"`
	Relationships []Relationship `json:"relationships"`

	nodeIndex map[string]int
	relIndex  map[string]int
}

func New() *Graph {
	return &Graph{
		nodeIndex: make(map[string]int),
		relIndex:  make(map[string]int),
	}
}

func GenerateID(label string, name string) string {
	return label + ":" + name
}

func (g *Graph) AddNode(node Node) {
	g.init()
	if index, ok := g.nodeIndex[node.ID]; ok {
		g.Nodes[index] = node
		return
	}
	g.nodeIndex[node.ID] = len(g.Nodes)
	g.Nodes = append(g.Nodes, node)
}

func (g *Graph) AddRelationship(relationship Relationship) {
	g.init()
	if index, ok := g.relIndex[relationship.ID]; ok {
		g.Relationships[index] = relationship
		return
	}
	g.relIndex[relationship.ID] = len(g.Relationships)
	g.Relationships = append(g.Relationships, relationship)
}

func (g *Graph) GetNode(id string) (Node, bool) {
	g.init()
	index, ok := g.nodeIndex[id]
	if !ok {
		return Node{}, false
	}
	return g.Nodes[index], true
}

func (g *Graph) GetRelationship(id string) (Relationship, bool) {
	g.init()
	index, ok := g.relIndex[id]
	if !ok {
		return Relationship{}, false
	}
	return g.Relationships[index], true
}

func (g *Graph) RemoveNode(id string) bool {
	g.init()
	index, ok := g.nodeIndex[id]
	if !ok {
		return false
	}
	g.Nodes = append(g.Nodes[:index], g.Nodes[index+1:]...)
	delete(g.nodeIndex, id)
	for nextIndex := index; nextIndex < len(g.Nodes); nextIndex++ {
		g.nodeIndex[g.Nodes[nextIndex].ID] = nextIndex
	}

	filtered := g.Relationships[:0]
	g.relIndex = make(map[string]int, len(g.Relationships))
	for _, relationship := range g.Relationships {
		if relationship.SourceID == id || relationship.TargetID == id {
			continue
		}
		g.relIndex[relationship.ID] = len(filtered)
		filtered = append(filtered, relationship)
	}
	g.Relationships = filtered
	return true
}

func (g *Graph) RemoveNodesByFile(filePath string) int {
	g.init()
	remove := make(map[string]struct{})
	for _, node := range g.Nodes {
		if node.Properties["filePath"] == filePath {
			remove[node.ID] = struct{}{}
		}
	}
	if len(remove) == 0 {
		return 0
	}

	filteredNodes := g.Nodes[:0]
	g.nodeIndex = make(map[string]int, len(g.Nodes)-len(remove))
	for _, node := range g.Nodes {
		if _, ok := remove[node.ID]; ok {
			continue
		}
		g.nodeIndex[node.ID] = len(filteredNodes)
		filteredNodes = append(filteredNodes, node)
	}
	g.Nodes = filteredNodes

	filteredRelationships := g.Relationships[:0]
	g.relIndex = make(map[string]int, len(g.Relationships))
	for _, relationship := range g.Relationships {
		if _, ok := remove[relationship.SourceID]; ok {
			continue
		}
		if _, ok := remove[relationship.TargetID]; ok {
			continue
		}
		g.relIndex[relationship.ID] = len(filteredRelationships)
		filteredRelationships = append(filteredRelationships, relationship)
	}
	g.Relationships = filteredRelationships
	return len(remove)
}

func (g *Graph) RemoveRelationship(id string) bool {
	g.init()
	index, ok := g.relIndex[id]
	if !ok {
		return false
	}
	g.Relationships = append(g.Relationships[:index], g.Relationships[index+1:]...)
	delete(g.relIndex, id)
	for nextIndex := index; nextIndex < len(g.Relationships); nextIndex++ {
		g.relIndex[g.Relationships[nextIndex].ID] = nextIndex
	}
	return true
}

func (g *Graph) ReplaceRelationship(id string, relationship Relationship) bool {
	g.init()
	index, ok := g.relIndex[id]
	if !ok {
		return false
	}
	g.Relationships[index] = relationship
	g.relIndex[relationship.ID] = index
	if relationship.ID != id {
		delete(g.relIndex, id)
	}
	return true
}

func (g *Graph) Compact() {
	if g == nil {
		return
	}
	if len(g.Nodes) < cap(g.Nodes) {
		nodes := make([]Node, len(g.Nodes))
		copy(nodes, g.Nodes)
		g.Nodes = nodes
	}
	if len(g.Relationships) < cap(g.Relationships) {
		relationships := make([]Relationship, len(g.Relationships))
		copy(relationships, g.Relationships)
		g.Relationships = relationships
	}
	g.nodeIndex = nil
	g.relIndex = nil
}

func (g *Graph) RelationshipCountsByType() map[RelationshipType]int {
	counts := make(map[RelationshipType]int)
	for _, relationship := range g.Relationships {
		counts[relationship.Type]++
	}
	return counts
}

func (g *Graph) SortedRelationships() []Relationship {
	out := append([]Relationship(nil), g.Relationships...)
	sort.Slice(out, func(i, j int) bool {
		left, right := out[i], out[j]
		if left.Type != right.Type {
			return left.Type < right.Type
		}
		if left.SourceID != right.SourceID {
			return left.SourceID < right.SourceID
		}
		if left.TargetID != right.TargetID {
			return left.TargetID < right.TargetID
		}
		return left.ID < right.ID
	})
	return out
}

func (g *Graph) init() {
	if g.nodeIndex == nil {
		g.nodeIndex = make(map[string]int, len(g.Nodes))
		for index, node := range g.Nodes {
			g.nodeIndex[node.ID] = index
		}
	}
	if g.relIndex == nil {
		g.relIndex = make(map[string]int, len(g.Relationships))
		for index, relationship := range g.Relationships {
			g.relIndex[relationship.ID] = index
		}
	}
}
