package httpapi

import (
	"fmt"
	"net/http"
	"time"
)

func (s Server) handleHeartbeat(w http.ResponseWriter, r *http.Request) {
	if !methodAllowed(w, r, http.MethodGet) {
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "Streaming unsupported")
		return
	}

	header := w.Header()
	header.Set("Content-Type", "text/event-stream")
	header.Set("Cache-Control", "no-cache")
	header.Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	writeHeartbeat := func() bool {
		if _, err := fmt.Fprint(w, ":ok\n\n"); err != nil {
			return false
		}
		flusher.Flush()
		return true
	}
	if !writeHeartbeat() {
		return
	}

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			if !writeHeartbeat() {
				return
			}
		}
	}
}
