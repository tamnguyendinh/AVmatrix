package scopeir

type DefinitionFact struct {
	ID                     string    `json:"id"`
	FilePath               string    `json:"filePath"`
	FileHash               string    `json:"fileHash,omitempty"`
	Name                   string    `json:"name"`
	Label                  NodeLabel `json:"label"`
	Range                  Range     `json:"range"`
	QualifiedName          string    `json:"qualifiedName,omitempty"`
	ParameterCount         *int      `json:"parameterCount,omitempty"`
	RequiredParameterCount *int      `json:"requiredParameterCount,omitempty"`
	ParameterTypes         []string  `json:"parameterTypes,omitempty"`
	ReturnType             string    `json:"returnType,omitempty"`
	DeclaredType           string    `json:"declaredType,omitempty"`
	OwnerID                string    `json:"ownerId,omitempty"`
	Visibility             string    `json:"visibility,omitempty"`
	Static                 *bool     `json:"isStatic,omitempty"`
	Readonly               *bool     `json:"isReadonly,omitempty"`
	Abstract               *bool     `json:"isAbstract,omitempty"`
	Final                  *bool     `json:"isFinal,omitempty"`
	Virtual                *bool     `json:"isVirtual,omitempty"`
	Override               *bool     `json:"isOverride,omitempty"`
	Async                  *bool     `json:"isAsync,omitempty"`
	Partial                *bool     `json:"isPartial,omitempty"`
	Annotations            []string  `json:"annotations,omitempty"`
	Description            string    `json:"description,omitempty"`
}

type BindingFact struct {
	Name    string        `json:"name"`
	DefID   string        `json:"defId"`
	Origin  BindingOrigin `json:"origin"`
	ViaID   string        `json:"viaId,omitempty"`
	ViaKind ImportKind    `json:"viaKind,omitempty"`
}

type TypeRef struct {
	RawName         string        `json:"rawName"`
	DeclaredAtScope string        `json:"declaredAtScope"`
	Source          TypeRefSource `json:"source"`
	TypeArgs        []TypeRef     `json:"typeArgs,omitempty"`
}

type TypeBindingFact struct {
	Name string  `json:"name"`
	Type TypeRef `json:"type"`
}

type ScopeFact struct {
	ID           string            `json:"id"`
	Parent       *string           `json:"parent"`
	Kind         ScopeKind         `json:"kind"`
	Range        Range             `json:"range"`
	FilePath     string            `json:"filePath"`
	FileHash     string            `json:"fileHash,omitempty"`
	Bindings     []BindingFact     `json:"bindings,omitempty"`
	OwnedDefIDs  []string          `json:"ownedDefIds,omitempty"`
	TypeBindings []TypeBindingFact `json:"typeBindings,omitempty"`
}

type ImportFact struct {
	ID                 string     `json:"id,omitempty"`
	FilePath           string     `json:"filePath"`
	FileHash           string     `json:"fileHash,omitempty"`
	Kind               ImportKind `json:"kind"`
	LocalName          string     `json:"localName,omitempty"`
	ImportedName       string     `json:"importedName,omitempty"`
	Alias              string     `json:"alias,omitempty"`
	TargetRaw          *string    `json:"targetRaw"`
	TargetFile         *string    `json:"targetFile,omitempty"`
	TargetExportedName string     `json:"targetExportedName,omitempty"`
	TargetModuleScope  string     `json:"targetModuleScope,omitempty"`
	TargetDefID        string     `json:"targetDefId,omitempty"`
	TransitiveVia      []string   `json:"transitiveVia,omitempty"`
	LinkStatus         string     `json:"linkStatus,omitempty"`
}

type CallSiteFact struct {
	FilePath         string   `json:"filePath"`
	FileHash         string   `json:"fileHash,omitempty"`
	Name             string   `json:"name"`
	Range            Range    `json:"range"`
	InScope          string   `json:"inScope"`
	CallForm         CallForm `json:"callForm,omitempty"`
	ExplicitReceiver string   `json:"explicitReceiver,omitempty"`
	Arity            *int     `json:"arity,omitempty"`
	ArgTypes         []string `json:"argTypes,omitempty"`
}

type AccessFact struct {
	FilePath         string     `json:"filePath"`
	FileHash         string     `json:"fileHash,omitempty"`
	Name             string     `json:"name"`
	Kind             AccessKind `json:"kind"`
	Range            Range      `json:"range"`
	InScope          string     `json:"inScope"`
	ExplicitReceiver string     `json:"explicitReceiver,omitempty"`
}

type HeritageFact struct {
	FilePath string       `json:"filePath"`
	FileHash string       `json:"fileHash,omitempty"`
	Name     string       `json:"name"`
	Kind     HeritageKind `json:"kind"`
	Range    Range        `json:"range"`
	InScope  string       `json:"inScope"`
}

type TypeAnnotationFact struct {
	FilePath string  `json:"filePath"`
	FileHash string  `json:"fileHash,omitempty"`
	Name     string  `json:"name"`
	Range    Range   `json:"range"`
	InScope  string  `json:"inScope"`
	Type     TypeRef `json:"type"`
}

type ReturnTypeFact struct {
	DefID    string  `json:"defId"`
	FilePath string  `json:"filePath"`
	FileHash string  `json:"fileHash,omitempty"`
	Range    Range   `json:"range"`
	Type     TypeRef `json:"type"`
}

type FrameworkFact struct {
	DefID                string  `json:"defId"`
	FilePath             string  `json:"filePath"`
	FileHash             string  `json:"fileHash,omitempty"`
	Framework            string  `json:"framework,omitempty"`
	Reason               string  `json:"reason"`
	EntryPointMultiplier float64 `json:"entryPointMultiplier"`
	Range                Range   `json:"range"`
}

type DomainFact struct {
	DefID    string `json:"defId"`
	FilePath string `json:"filePath"`
	FileHash string `json:"fileHash,omitempty"`
	Domain   string `json:"domain"`
	Role     string `json:"role,omitempty"`
	Reason   string `json:"reason,omitempty"`
	Range    Range  `json:"range"`
}
