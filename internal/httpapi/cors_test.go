package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestIsAllowedOriginMatchesLoopbackContract(t *testing.T) {
	tests := []struct {
		name   string
		origin string
		want   bool
	}{
		{name: "absent", origin: "", want: true},
		{name: "localhost", origin: "http://localhost", want: true},
		{name: "localhost port", origin: "http://localhost:5228", want: true},
		{name: "https localhost", origin: "https://localhost:5228", want: true},
		{name: "backend ipv4", origin: "http://127.0.0.1:4848", want: true},
		{name: "web ipv4", origin: "http://127.0.0.1:5228", want: true},
		{name: "https ipv4", origin: "https://127.0.0.1", want: true},
		{name: "ipv6", origin: "http://[::1]:5228", want: true},
		{name: "https ipv6", origin: "https://[::1]", want: true},
		{name: "hosted app", origin: "https://avmatrix.vercel.app", want: false},
		{name: "private 10 range", origin: "http://10.0.0.5:3000", want: false},
		{name: "private 172 range", origin: "http://172.16.5.1:3000", want: false},
		{name: "private 192 range", origin: "http://192.168.1.10:3000", want: false},
		{name: "external", origin: "https://example.com", want: false},
		{name: "file", origin: "file://localhost", want: false},
		{name: "malformed", origin: "not-a-url", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsAllowedOrigin(tt.origin); got != tt.want {
				t.Fatalf("IsAllowedOrigin(%q) = %v, want %v", tt.origin, got, tt.want)
			}
		})
	}
}

func TestCORSAllowsLoopbackAndPrivateNetworkPreflight(t *testing.T) {
	handler := WithCORS(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	request := httptest.NewRequest(http.MethodOptions, "/api/repos", nil)
	request.Header.Set("Origin", "http://localhost:5228")
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("OPTIONS status = %d, want %d", recorder.Code, http.StatusNoContent)
	}
	if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:5228" {
		t.Fatalf("allow origin = %q", got)
	}
	if got := recorder.Header().Get("Access-Control-Allow-Private-Network"); got != "true" {
		t.Fatalf("private network header = %q", got)
	}
	if got := recorder.Header().Get("Access-Control-Allow-Headers"); !containsHeaderValue(got, "Mcp-Session-Id") {
		t.Fatalf("allow headers missing Mcp-Session-Id: %q", got)
	}
	if got := recorder.Header().Get("Access-Control-Allow-Headers"); !containsHeaderValue(got, "Mcp-Protocol-Version") {
		t.Fatalf("allow headers missing Mcp-Protocol-Version: %q", got)
	}
	if got := recorder.Header().Get("Access-Control-Expose-Headers"); !containsHeaderValue(got, "Mcp-Session-Id") {
		t.Fatalf("expose headers missing Mcp-Session-Id: %q", got)
	}
}

func TestCORSLeavesDisallowedOriginUnreflected(t *testing.T) {
	handler := WithCORS(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	request := httptest.NewRequest(http.MethodGet, "/api/repos", nil)
	request.Header.Set("Origin", "https://example.com")
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("GET status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("disallowed origin was reflected: %q", got)
	}
	if got := recorder.Header().Get("Access-Control-Allow-Private-Network"); got != "true" {
		t.Fatalf("private network header = %q", got)
	}
}

func containsHeaderValue(header string, value string) bool {
	for _, part := range strings.Split(header, ",") {
		if strings.EqualFold(strings.TrimSpace(part), value) {
			return true
		}
	}
	return false
}
