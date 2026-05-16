package httpapi

import (
	"log/slog"
	"net/http"
	"runtime"

	"github.com/tamnguyendinh/avmatrix-go/internal/repo"
	"github.com/tamnguyendinh/avmatrix-go/internal/session"
	"github.com/tamnguyendinh/avmatrix-go/internal/version"
)

const (
	LaunchContextNPX    = "npx"
	LaunchContextGlobal = "global"
	LaunchContextLocal  = "local"
)

type Config struct {
	Store          repo.Store
	Version        string
	LaunchContext  string
	RuntimeVersion string
	Logger         *slog.Logger
	Searcher       Searcher
	EmbedRunner    EmbedRunner
	AnalyzeRunner  AnalyzeRunner
	SessionRuntime SessionRuntime
}

type Server struct {
	store          repo.Store
	version        string
	launchContext  string
	runtimeVersion string
	logger         *slog.Logger
	searcher       Searcher
	embedRunner    EmbedRunner
	analyzeRunner  AnalyzeRunner
	analyzeJobs    *JobManager
	embedJobs      *JobManager
	mcpSessions    *mcpHTTPSessions
	sessionRuntime SessionRuntime
}

func NewHandler(config Config) http.Handler {
	server := newServer(config)
	mux := http.NewServeMux()

	mux.HandleFunc("/api/heartbeat", server.handleHeartbeat)
	mux.HandleFunc("/api/info", server.handleInfo)
	mux.HandleFunc("/api/repos", server.handleRepos)
	mux.HandleFunc("/api/repo", server.handleRepo)
	mux.HandleFunc("/api/local/folder-picker", server.handleLocalFolderPicker)
	mux.HandleFunc("/api/graph", server.handleGraph)
	mux.HandleFunc("/api/file", server.handleFile)
	mux.HandleFunc("/api/grep", server.handleGrep)
	mux.HandleFunc("/api/query", server.handleQuery)
	mux.HandleFunc("/api/processes", server.handleProcesses)
	mux.HandleFunc("/api/process", server.handleProcess)
	mux.HandleFunc("/api/clusters", server.handleClusters)
	mux.HandleFunc("/api/cluster", server.handleCluster)
	mux.HandleFunc("/api/analyze", server.handleAnalyze)
	mux.HandleFunc("/api/analyze/", server.handleAnalyzeJob)
	mux.HandleFunc("/api/search", server.handleSearch)
	mux.HandleFunc("/api/embed", server.handleEmbed)
	mux.HandleFunc("/api/embed/", server.handleEmbedJob)
	mux.HandleFunc("/api/mcp", server.handleMCP)
	mux.HandleFunc("/api/session/status", server.handleSessionStatus)
	mux.HandleFunc("/api/session/chat", server.handleSessionChat)
	mux.HandleFunc("/api/session/", server.handleSession)

	return WithCORS(mux)
}

func newServer(config Config) Server {
	if config.Version == "" {
		config.Version = version.Version
	}
	if config.LaunchContext == "" {
		config.LaunchContext = LaunchContextGlobal
	}
	if config.RuntimeVersion == "" {
		config.RuntimeVersion = runtime.Version()
	}
	if config.Logger == nil {
		config.Logger = slog.Default()
	}
	if config.Searcher == nil {
		config.Searcher = SearchService{}
	}
	if config.EmbedRunner == nil {
		config.EmbedRunner = EmbedService{}
	}
	if config.AnalyzeRunner == nil {
		config.AnalyzeRunner = AnalyzeService{Store: config.Store}
	}
	if config.SessionRuntime == nil {
		config.SessionRuntime = session.NewController(nil, config.Store)
	}

	return Server{
		store:          config.Store,
		version:        config.Version,
		launchContext:  config.LaunchContext,
		runtimeVersion: config.RuntimeVersion,
		logger:         config.Logger,
		searcher:       config.Searcher,
		embedRunner:    config.EmbedRunner,
		analyzeRunner:  config.AnalyzeRunner,
		analyzeJobs:    NewJobManager(),
		embedJobs:      NewJobManager(),
		mcpSessions:    newMCPHTTPSessions(config.Store),
		sessionRuntime: config.SessionRuntime,
	}
}

func methodAllowed(w http.ResponseWriter, r *http.Request, allowed string) bool {
	if r.Method == allowed {
		return true
	}
	w.Header().Set("Allow", allowed)
	writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
	return false
}
