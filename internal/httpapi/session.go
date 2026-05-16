package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/tamnguyendinh/avmatrix-go/internal/session"
)

type SessionRuntime interface {
	GetStatus(ctx context.Context, binding session.RepoBinding) (session.Status, error)
	StartChat(ctx context.Context, request session.ChatRequest) (*session.Job, session.ResolvedRepo, error)
	CancelSession(sessionID string, reason string) bool
}

type sessionErrorResponse struct {
	Code    session.ErrorCode `json:"code"`
	Error   string            `json:"error"`
	Details map[string]any    `json:"details,omitempty"`
}

func (s Server) handleSessionStatus(w http.ResponseWriter, r *http.Request) {
	if !methodAllowed(w, r, http.MethodGet) {
		return
	}
	status, err := s.sessionRuntime.GetStatus(r.Context(), session.RepoBinding{
		RepoName: r.URL.Query().Get("repoName"),
		RepoPath: r.URL.Query().Get("repoPath"),
	})
	if err != nil {
		writeSessionError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, status)
}

func (s Server) handleSessionChat(w http.ResponseWriter, r *http.Request) {
	if !methodAllowed(w, r, http.MethodPost) {
		return
	}
	defer r.Body.Close()

	var request session.ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeJSON(w, http.StatusBadRequest, sessionErrorResponse{
			Code:  session.ErrorBadRequest,
			Error: "Request body must be valid JSON",
		})
		return
	}
	if strings.TrimSpace(request.Message) == "" {
		writeJSON(w, http.StatusBadRequest, sessionErrorResponse{
			Code:  session.ErrorBadRequest,
			Error: `Request body must include a non-empty "message"`,
		})
		return
	}

	job, _, err := s.sessionRuntime.StartChat(r.Context(), request)
	if err != nil {
		writeSessionError(w, err)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	flusher, _ := w.(http.Flusher)
	if flusher != nil {
		flusher.Flush()
	}

	events, unsubscribe := job.Subscribe(true)
	defer unsubscribe()

	heartbeat := time.NewTicker(15 * time.Second)
	defer heartbeat.Stop()

	for {
		select {
		case event, ok := <-events:
			if !ok {
				return
			}
			writeSessionSSE(w, event)
			if flusher != nil {
				flusher.Flush()
			}
			if isTerminalSessionEvent(event.Type) {
				return
			}
		case <-heartbeat.C:
			_, _ = w.Write([]byte(":heartbeat\n\n"))
			if flusher != nil {
				flusher.Flush()
			}
		case <-r.Context().Done():
			s.sessionRuntime.CancelSession(job.ID, "Client disconnected")
			return
		}
	}
}

func (s Server) handleSession(w http.ResponseWriter, r *http.Request) {
	if !methodAllowed(w, r, http.MethodDelete) {
		return
	}
	sessionID := strings.TrimPrefix(r.URL.Path, "/api/session/")
	if sessionID == "" || strings.Contains(sessionID, "/") {
		writeJSON(w, http.StatusNotFound, sessionErrorResponse{
			Code:  session.ErrorSessionNotFound,
			Error: "Session was not found",
		})
		return
	}
	if !s.sessionRuntime.CancelSession(sessionID, "Cancelled by user") {
		writeJSON(w, http.StatusNotFound, sessionErrorResponse{
			Code:  session.ErrorSessionNotFound,
			Error: `Session "` + sessionID + `" was not found or is no longer running`,
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"sessionId": sessionID, "status": "cancelled"})
}

func writeSessionSSE(w http.ResponseWriter, event session.Event) {
	_, _ = w.Write([]byte("event: " + event.Type + "\n"))
	raw, err := json.Marshal(event)
	if err != nil {
		return
	}
	_, _ = w.Write([]byte("data: "))
	_, _ = w.Write(raw)
	_, _ = w.Write([]byte("\n\n"))
}

func writeSessionError(w http.ResponseWriter, err error) {
	var runtimeErr *session.RuntimeError
	if errors.As(err, &runtimeErr) {
		writeJSON(w, runtimeErr.Status, sessionErrorResponse{
			Code:    runtimeErr.Code,
			Error:   runtimeErr.Message,
			Details: runtimeErr.Details,
		})
		return
	}
	writeJSON(w, http.StatusInternalServerError, sessionErrorResponse{
		Code:  session.ErrorSessionStartFailed,
		Error: err.Error(),
	})
}

func isTerminalSessionEvent(eventType string) bool {
	return eventType == "done" || eventType == "error" || eventType == "cancelled"
}
