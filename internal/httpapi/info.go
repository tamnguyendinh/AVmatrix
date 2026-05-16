package httpapi

import "net/http"

type infoResponse struct {
	Version       string `json:"version"`
	LaunchContext string `json:"launchContext"`
	NodeVersion   string `json:"nodeVersion"`
}

func (s Server) handleInfo(w http.ResponseWriter, r *http.Request) {
	if !methodAllowed(w, r, http.MethodGet) {
		return
	}

	writeJSON(w, http.StatusOK, infoResponse{
		Version:       s.version,
		LaunchContext: s.launchContext,
		NodeVersion:   s.runtimeVersion,
	})
}
