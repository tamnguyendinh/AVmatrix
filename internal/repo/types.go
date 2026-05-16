package repo

type Stats struct {
	Files       *int `json:"files,omitempty"`
	Nodes       *int `json:"nodes,omitempty"`
	Edges       *int `json:"edges,omitempty"`
	Communities *int `json:"communities,omitempty"`
	Processes   *int `json:"processes,omitempty"`
	Embeddings  *int `json:"embeddings,omitempty"`
}

type Meta struct {
	RepoPath   string `json:"repoPath"`
	LastCommit string `json:"lastCommit"`
	IndexedAt  string `json:"indexedAt"`
	Stats      *Stats `json:"stats,omitempty"`
}

type Indexed struct {
	RepoPath    string
	StoragePath string
	LbugPath    string
	MetaPath    string
	Meta        Meta
}

type RegistryEntry struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	StoragePath string `json:"storagePath"`
	IndexedAt   string `json:"indexedAt"`
	LastCommit  string `json:"lastCommit"`
	Stats       *Stats `json:"stats,omitempty"`
}

type RegisterOptions struct {
	Name               string
	AllowDuplicateName bool
	InferredName       string
}
