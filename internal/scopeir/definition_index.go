package scopeir

type DefinitionIndex struct {
	byID map[string]*DefinitionFact
}

func BuildDefinitionIndex(defs []DefinitionFact) DefinitionIndex {
	byID := make(map[string]*DefinitionFact, len(defs))
	for index := range defs {
		def := &defs[index]
		if _, ok := byID[def.ID]; ok {
			continue
		}
		byID[def.ID] = def
	}
	return DefinitionIndex{byID: byID}
}

func (idx DefinitionIndex) Size() int {
	return len(idx.byID)
}

func (idx DefinitionIndex) Get(id string) (*DefinitionFact, bool) {
	def, ok := idx.byID[id]
	return def, ok
}

func (idx DefinitionIndex) Has(id string) bool {
	_, ok := idx.byID[id]
	return ok
}

func (idx DefinitionIndex) ByID() map[string]*DefinitionFact {
	out := make(map[string]*DefinitionFact, len(idx.byID))
	for id, def := range idx.byID {
		out[id] = def
	}
	return out
}
