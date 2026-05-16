package httpapi

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	mcpserver "github.com/tamnguyendinh/avmatrix-go/internal/mcp"
	"github.com/tamnguyendinh/avmatrix-go/internal/repo"
)

const mcpHTTPSessionTTL = 30 * time.Minute

type mcpHTTPSession struct {
	server       mcpserver.Server
	lastActivity time.Time
	sseDone      chan struct{}
}

type mcpHTTPSessions struct {
	mu       sync.Mutex
	store    repo.Store
	now      func() time.Time
	sessions map[string]mcpHTTPSession
}

func newMCPHTTPSessions(store repo.Store) *mcpHTTPSessions {
	return &mcpHTTPSessions{
		store:    store,
		now:      time.Now,
		sessions: map[string]mcpHTTPSession{},
	}
}

func (s *mcpHTTPSessions) create() (string, mcpserver.Server, error) {
	id, err := newMCPHTTPSessionID()
	if err != nil {
		return "", mcpserver.Server{}, err
	}
	server := mcpserver.NewServer(mcpserver.Config{Store: s.store})

	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[id] = mcpHTTPSession{server: server, lastActivity: s.now()}
	return id, server, nil
}

func (s *mcpHTTPSessions) get(id string) (mcpserver.Server, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, ok := s.sessions[id]
	if !ok {
		return mcpserver.Server{}, false
	}
	session.lastActivity = s.now()
	s.sessions[id] = session
	return session.server, true
}

func (s *mcpHTTPSessions) startSSE(id string) (chan struct{}, bool, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, ok := s.sessions[id]
	if !ok {
		return nil, false, false
	}
	if session.sseDone != nil {
		return nil, true, true
	}
	done := make(chan struct{})
	session.sseDone = done
	session.lastActivity = s.now()
	s.sessions[id] = session
	return done, true, false
}

func (s *mcpHTTPSessions) endSSE(id string, done chan struct{}) {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, ok := s.sessions[id]
	if !ok || session.sseDone != done {
		return
	}
	session.sseDone = nil
	session.lastActivity = s.now()
	s.sessions[id] = session
}

func (s *mcpHTTPSessions) delete(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	session, ok := s.sessions[id]
	if !ok {
		return false
	}
	if session.sseDone != nil {
		close(session.sseDone)
	}
	delete(s.sessions, id)
	return true
}

func (s *mcpHTTPSessions) deleteExpired() {
	now := s.now()

	s.mu.Lock()
	defer s.mu.Unlock()
	for id, session := range s.sessions {
		if now.Sub(session.lastActivity) > mcpHTTPSessionTTL {
			if session.sseDone != nil {
				close(session.sseDone)
			}
			delete(s.sessions, id)
		}
	}
}

func (s Server) handleMCP(w http.ResponseWriter, r *http.Request) {
	s.mcpSessions.deleteExpired()

	sessionID := strings.TrimSpace(r.Header.Get("Mcp-Session-Id"))
	if sessionID != "" {
		s.handleMCPWithSession(w, r, sessionID)
		return
	}

	if r.Method != http.MethodPost {
		writeMCPError(w, http.StatusBadRequest, -32000, "No valid session. Send a POST to initialize.")
		return
	}
	if !validateMCPProtocolHeader(w, r) {
		return
	}
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		writeMCPError(w, http.StatusBadRequest, -32700, "Parse error")
		return
	}
	method, err := mcpRequestMethod(raw)
	if err != nil {
		writeMCPError(w, http.StatusBadRequest, -32700, "Parse error")
		return
	}
	if method != "initialize" {
		writeMCPError(w, http.StatusBadRequest, -32000, "Bad Request: initialize required before session use")
		return
	}

	newSessionID, server, err := s.mcpSessions.create()
	if err != nil {
		writeMCPError(w, http.StatusInternalServerError, -32000, "Internal MCP server error")
		return
	}
	w.Header().Set("Mcp-Session-Id", newSessionID)
	s.handleMCPPostRaw(w, r, server, raw)
}

func (s Server) handleMCPWithSession(w http.ResponseWriter, r *http.Request, sessionID string) {
	if r.Method == http.MethodDelete {
		if !validateMCPProtocolHeader(w, r) {
			return
		}
		if !s.mcpSessions.delete(sessionID) {
			writeMCPError(w, http.StatusNotFound, -32001, "Session not found. Re-initialize.")
			return
		}
		w.WriteHeader(http.StatusOK)
		return
	}

	server, ok := s.mcpSessions.get(sessionID)
	if !ok {
		writeMCPError(w, http.StatusNotFound, -32001, "Session not found. Re-initialize.")
		return
	}

	if r.Method == http.MethodGet {
		s.handleMCPGet(w, r, sessionID)
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "GET, POST, DELETE")
		writeMCPError(w, http.StatusMethodNotAllowed, -32000, "Method not allowed")
		return
	}
	if !validateMCPProtocolHeader(w, r) {
		return
	}

	w.Header().Set("Mcp-Session-Id", sessionID)
	s.handleMCPPost(w, r, server)
}

func (s Server) handleMCPGet(w http.ResponseWriter, r *http.Request, sessionID string) {
	if !mcpHeaderContains(r.Header.Get("Accept"), "text/event-stream") {
		writeMCPError(w, http.StatusNotAcceptable, -32000, "Not Acceptable: Client must accept text/event-stream")
		return
	}
	if !validateMCPProtocolHeader(w, r) {
		return
	}

	done, ok, conflict := s.mcpSessions.startSSE(sessionID)
	if !ok {
		writeMCPError(w, http.StatusNotFound, -32001, "Session not found. Re-initialize.")
		return
	}
	if conflict {
		writeMCPError(w, http.StatusConflict, -32000, "Conflict: Only one SSE stream is allowed per session")
		return
	}
	defer s.mcpSessions.endSSE(sessionID, done)

	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache, no-transform")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Mcp-Session-Id", sessionID)
	w.WriteHeader(http.StatusOK)
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}

	select {
	case <-r.Context().Done():
	case <-done:
	}
}

func (s Server) handleMCPPost(w http.ResponseWriter, r *http.Request, server mcpserver.Server) {
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		writeMCPError(w, http.StatusBadRequest, -32700, "Parse error")
		return
	}
	s.handleMCPPostRaw(w, r, server, raw)
}

func (s Server) handleMCPPostRaw(w http.ResponseWriter, r *http.Request, server mcpserver.Server, raw []byte) {
	response, ok, err := server.HandleJSONRPC(raw)
	if err != nil {
		writeMCPError(w, http.StatusInternalServerError, -32000, "Internal MCP server error")
		return
	}
	if !ok {
		w.WriteHeader(http.StatusAccepted)
		return
	}

	if mcpWantsSSE(r) {
		w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.WriteHeader(http.StatusOK)
		writeMCPSSEMessage(w, response)
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(response)
}

func mcpRequestMethod(raw []byte) (string, error) {
	var request struct {
		Method string `json:"method"`
	}
	if err := json.Unmarshal(raw, &request); err != nil {
		return "", err
	}
	return request.Method, nil
}

func validateMCPProtocolHeader(w http.ResponseWriter, r *http.Request) bool {
	protocol := strings.TrimSpace(r.Header.Get("Mcp-Protocol-Version"))
	if protocol == "" || mcpserver.SupportsProtocolVersion(protocol) {
		return true
	}
	writeMCPError(w, http.StatusBadRequest, -32000, "Bad Request: Unsupported protocol version: "+protocol)
	return false
}

func mcpWantsSSE(r *http.Request) bool {
	accept := r.Header.Get("Accept")
	return mcpHeaderContains(accept, "text/event-stream") &&
		mcpHeaderContains(accept, "application/json")
}

func mcpHeaderContains(header string, value string) bool {
	return strings.Contains(strings.ToLower(header), strings.ToLower(value))
}

func writeMCPSSEMessage(w io.Writer, payload []byte) {
	fmt.Fprintf(w, "event: message\n")
	fmt.Fprintf(w, "data: %s\n\n", payload)
}

func writeMCPError(w http.ResponseWriter, status int, code int, message string) {
	writeJSON(w, status, map[string]any{
		"jsonrpc": "2.0",
		"error": map[string]any{
			"code":    code,
			"message": message,
		},
		"id": nil,
	})
}

func newMCPHTTPSessionID() (string, error) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw[:]), nil
}
