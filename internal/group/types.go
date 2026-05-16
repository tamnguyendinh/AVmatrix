package group

type ManifestLink struct {
	From     string `json:"from"`
	To       string `json:"to"`
	Type     string `json:"type"`
	Contract string `json:"contract"`
	Role     string `json:"role"`
}

type DetectConfig struct {
	HTTP              bool `json:"http"`
	GRPC              bool `json:"grpc"`
	Topics            bool `json:"topics"`
	SharedLibs        bool `json:"shared_libs"`
	EmbeddingFallback bool `json:"embedding_fallback"`
}

type MatchingConfig struct {
	BM25Threshold        float64 `json:"bm25_threshold"`
	EmbeddingThreshold   float64 `json:"embedding_threshold"`
	MaxCandidatesPerStep int     `json:"max_candidates_per_step"`
}

type Config struct {
	Version     int                          `json:"version"`
	Name        string                       `json:"name"`
	Description string                       `json:"description"`
	Repos       map[string]string            `json:"repos"`
	Links       []ManifestLink               `json:"links"`
	Packages    map[string]map[string]string `json:"packages"`
	Detect      DetectConfig                 `json:"detect"`
	Matching    MatchingConfig               `json:"matching"`
}

type RepoSnapshot struct {
	IndexedAt  string `json:"indexedAt"`
	LastCommit string `json:"lastCommit"`
}

type SymbolRef struct {
	FilePath string `json:"filePath"`
	Name     string `json:"name"`
}

type StoredContract struct {
	ContractID string         `json:"contractId"`
	Type       string         `json:"type"`
	Role       string         `json:"role"`
	SymbolUID  string         `json:"symbolUid"`
	SymbolRef  SymbolRef      `json:"symbolRef"`
	SymbolName string         `json:"symbolName"`
	Confidence float64        `json:"confidence"`
	Meta       map[string]any `json:"meta"`
	Service    string         `json:"service,omitempty"`
	Repo       string         `json:"repo"`
}

type CrossLinkEndpoint struct {
	Repo      string    `json:"repo"`
	Service   string    `json:"service,omitempty"`
	SymbolUID string    `json:"symbolUid"`
	SymbolRef SymbolRef `json:"symbolRef"`
}

type CrossLink struct {
	From       CrossLinkEndpoint `json:"from"`
	To         CrossLinkEndpoint `json:"to"`
	Type       string            `json:"type"`
	ContractID string            `json:"contractId"`
	MatchType  string            `json:"matchType"`
	Confidence float64           `json:"confidence"`
}

type ContractRegistry struct {
	Version       int                     `json:"version"`
	GeneratedAt   string                  `json:"generatedAt"`
	RepoSnapshots map[string]RepoSnapshot `json:"repoSnapshots"`
	MissingRepos  []string                `json:"missingRepos"`
	Contracts     []StoredContract        `json:"contracts"`
	CrossLinks    []CrossLink             `json:"crossLinks"`
}

type RepoStatus struct {
	IndexStale     bool `json:"indexStale"`
	ContractsStale bool `json:"contractsStale"`
	Missing        bool `json:"missing"`
	CommitsBehind  *int `json:"commitsBehind,omitempty"`
}

type StatusResult struct {
	Group        string                `json:"group"`
	LastSync     *string               `json:"lastSync"`
	MissingRepos []string              `json:"missingRepos"`
	Repos        map[string]RepoStatus `json:"repos"`
}

type ContractsOptions struct {
	Type          string
	Repo          string
	UnmatchedOnly bool
}

type ContractsResult struct {
	Contracts  []StoredContract `json:"contracts"`
	CrossLinks []CrossLink      `json:"crossLinks"`
}

type QueryResult struct {
	Group   string             `json:"group"`
	Query   string             `json:"query"`
	Results []map[string]any   `json:"results"`
	PerRepo []QueryRepoSummary `json:"per_repo"`
}

type QueryRepoSummary struct {
	Repo  string `json:"repo"`
	Count int    `json:"count"`
}

type SyncOptions struct {
	AllowStale     bool
	Verbose        bool
	ExactOnly      bool
	SkipEmbeddings bool
}

type SyncResult struct {
	Contracts     []StoredContract        `json:"contracts"`
	CrossLinks    []CrossLink             `json:"crossLinks"`
	Unmatched     []StoredContract        `json:"unmatched"`
	MissingRepos  []string                `json:"missingRepos"`
	RepoSnapshots map[string]RepoSnapshot `json:"repoSnapshots"`
}
