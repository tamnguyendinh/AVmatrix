package session

type Provider string

const (
	ProviderCodex Provider = "codex"
)

type Availability string

const (
	AvailabilityReady        Availability = "ready"
	AvailabilityNotInstalled Availability = "not_installed"
	AvailabilityNotSignedIn  Availability = "not_signed_in"
	AvailabilityError        Availability = "error"
)

type ExecutionMode string

const (
	ExecutionModeSandboxed ExecutionMode = "sandboxed"
	ExecutionModeBypass    ExecutionMode = "bypass"
)

type RuntimeEnvironment string

const (
	RuntimeNative RuntimeEnvironment = "native"
	RuntimeWSL2   RuntimeEnvironment = "wsl2"
)

type ErrorCode string

const (
	ErrorBadRequest                ErrorCode = "BAD_REQUEST"
	ErrorInvalidRepoBinding        ErrorCode = "INVALID_REPO_BINDING"
	ErrorInvalidRepoPath           ErrorCode = "INVALID_REPO_PATH"
	ErrorRepoNotFound              ErrorCode = "REPO_NOT_FOUND"
	ErrorIndexRequired             ErrorCode = "INDEX_REQUIRED"
	ErrorSessionNotFound           ErrorCode = "SESSION_NOT_FOUND"
	ErrorSessionRuntimeUnavailable ErrorCode = "SESSION_RUNTIME_UNAVAILABLE"
	ErrorSessionNotSignedIn        ErrorCode = "SESSION_NOT_SIGNED_IN"
	ErrorSessionStartFailed        ErrorCode = "SESSION_START_FAILED"
	ErrorSessionCancelled          ErrorCode = "SESSION_CANCELLED"
)

type RepoBinding struct {
	RepoName string `json:"repoName,omitempty"`
	RepoPath string `json:"repoPath,omitempty"`
}

type ResolvedRepo struct {
	RepoName    string `json:"repoName"`
	RepoPath    string `json:"repoPath"`
	Indexed     bool   `json:"indexed"`
	StoragePath string `json:"storagePath,omitempty"`
}

type RepoResolution struct {
	RepoName         string `json:"repoName,omitempty"`
	RepoPath         string `json:"repoPath,omitempty"`
	State            string `json:"state"`
	ResolvedRepoName string `json:"resolvedRepoName,omitempty"`
	ResolvedRepoPath string `json:"resolvedRepoPath,omitempty"`
	Message          string `json:"message,omitempty"`
}

type Status struct {
	Provider               Provider           `json:"provider"`
	Availability           Availability       `json:"availability"`
	Available              bool               `json:"available"`
	Authenticated          bool               `json:"authenticated"`
	ExecutablePath         string             `json:"executablePath,omitempty"`
	Version                string             `json:"version,omitempty"`
	Message                string             `json:"message,omitempty"`
	RecommendedEnvironment RuntimeEnvironment `json:"recommendedEnvironment,omitempty"`
	RuntimeEnvironment     RuntimeEnvironment `json:"runtimeEnvironment"`
	ExecutionMode          ExecutionMode      `json:"executionMode"`
	SupportsSSE            bool               `json:"supportsSse"`
	SupportsCancel         bool               `json:"supportsCancel"`
	SupportsMCP            bool               `json:"supportsMcp"`
	Repo                   *RepoResolution    `json:"repo,omitempty"`
}

type ChatRequest struct {
	RepoName string `json:"repoName,omitempty"`
	RepoPath string `json:"repoPath,omitempty"`
	Message  string `json:"message"`
}

func (r ChatRequest) Binding() RepoBinding {
	return RepoBinding{RepoName: r.RepoName, RepoPath: r.RepoPath}
}

type ToolCall struct {
	ID     string         `json:"id"`
	Name   string         `json:"name"`
	Args   map[string]any `json:"args,omitempty"`
	Result string         `json:"result,omitempty"`
	Status string         `json:"status"`
}

type Event struct {
	Type               string             `json:"type"`
	SessionID          string             `json:"sessionId"`
	Provider           Provider           `json:"provider"`
	RepoName           string             `json:"repoName"`
	RepoPath           string             `json:"repoPath"`
	Timestamp          int64              `json:"timestamp"`
	RuntimeEnvironment RuntimeEnvironment `json:"runtimeEnvironment,omitempty"`
	ExecutionMode      ExecutionMode      `json:"executionMode,omitempty"`
	Reasoning          string             `json:"reasoning,omitempty"`
	Content            string             `json:"content,omitempty"`
	ToolCall           *ToolCall          `json:"toolCall,omitempty"`
	Code               ErrorCode          `json:"code,omitempty"`
	Error              string             `json:"error,omitempty"`
	Reason             string             `json:"reason,omitempty"`
	Usage              map[string]int     `json:"usage,omitempty"`
}
