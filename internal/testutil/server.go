// Package testutil provides test helpers for the zh CLI.
package testutil

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// GraphQLRequest represents a decoded GraphQL request body.
type GraphQLRequest struct {
	Query     string          `json:"query"`
	Variables json.RawMessage `json:"variables"`
}

// MockServer is a test HTTP server that serves canned GraphQL and REST responses.
type MockServer struct {
	Server       *httptest.Server
	handlers     []handler
	restHandlers []restHandler
}

type handler struct {
	match   func(GraphQLRequest) bool
	respond func(http.ResponseWriter, GraphQLRequest)
}

type restHandler struct {
	pathSubstring string
	respond       func(http.ResponseWriter, *http.Request)
}

// NewMockServer creates a new mock GraphQL server.
// Call Close() when done (or use t.Cleanup).
func NewMockServer(t *testing.T) *MockServer {
	t.Helper()

	ms := &MockServer{}

	ms.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "expected POST", http.StatusMethodNotAllowed)
			return
		}

		// Check REST handlers first (path-based routing)
		for _, h := range ms.restHandlers {
			if containsSubstring(r.URL.Path, h.pathSubstring) {
				h.respond(w, r)
				return
			}
		}

		var req GraphQLRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad request body", http.StatusBadRequest)
			return
		}

		for _, h := range ms.handlers {
			if h.match(req) {
				h.respond(w, req)
				return
			}
		}

		http.Error(w, "no handler matched request", http.StatusNotFound)
	}))

	t.Cleanup(ms.Server.Close)
	return ms
}

// URL returns the mock server's base URL.
func (ms *MockServer) URL() string {
	return ms.Server.URL
}

// Handle registers a handler that matches requests where match returns true.
func (ms *MockServer) Handle(match func(GraphQLRequest) bool, respond func(http.ResponseWriter, GraphQLRequest)) {
	ms.handlers = append(ms.handlers, handler{match: match, respond: respond})
}

// HandleQuery registers a handler that responds with a static JSON body
// when the request query contains the given substring.
func (ms *MockServer) HandleQuery(querySubstring string, responseBody any) {
	data, err := json.Marshal(responseBody)
	if err != nil {
		panic("testutil: failed to marshal response: " + err.Error())
	}

	ms.Handle(
		func(req GraphQLRequest) bool {
			return containsSubstring(req.Query, querySubstring)
		},
		func(w http.ResponseWriter, _ GraphQLRequest) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(data)
		},
	)
}

// HandleREST registers a handler that responds to REST API calls where the
// request path contains the given substring.
func (ms *MockServer) HandleREST(pathSubstring string, statusCode int, responseBody any) {
	data, err := json.Marshal(responseBody)
	if err != nil {
		panic("testutil: failed to marshal response: " + err.Error())
	}

	ms.restHandlers = append(ms.restHandlers, restHandler{
		pathSubstring: pathSubstring,
		respond: func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(statusCode)
			_, _ = w.Write(data)
		},
	})
}

func containsSubstring(s, sub string) bool {
	return len(sub) > 0 && len(s) >= len(sub) && searchString(s, sub)
}

func searchString(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
