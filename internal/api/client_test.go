package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dslh/zh/internal/exitcode"
)

func TestExecuteSendsAuthHeader(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"data":{"viewer":{"login":"test"}}}`))
	}))
	defer srv.Close()

	client := New("my-api-key", WithEndpoint(srv.URL))
	_, err := client.Execute("{ viewer { login } }", nil)
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	if gotAuth != "Bearer my-api-key" {
		t.Errorf("Authorization = %q, want %q", gotAuth, "Bearer my-api-key")
	}
}

func TestExecuteSendsUserAgent(t *testing.T) {
	var gotUA string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.Header.Get("User-Agent")
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"data":null}`))
	}))
	defer srv.Close()

	client := New("key", WithEndpoint(srv.URL))
	client.Execute("{ test }", nil)

	if gotUA != userAgent {
		t.Errorf("User-Agent = %q, want %q", gotUA, userAgent)
	}
}

func TestExecuteReturnsData(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"data":{"viewer":{"login":"testuser","id":"abc123"}}}`))
	}))
	defer srv.Close()

	client := New("key", WithEndpoint(srv.URL))
	data, err := client.Execute("{ viewer { login id } }", nil)
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	var result struct {
		Viewer struct {
			Login string `json:"login"`
			ID    string `json:"id"`
		} `json:"viewer"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal data error: %v", err)
	}
	if result.Viewer.Login != "testuser" {
		t.Errorf("login = %q, want %q", result.Viewer.Login, "testuser")
	}
}

func TestExecuteSendsVariables(t *testing.T) {
	var gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"data":null}`))
	}))
	defer srv.Close()

	client := New("key", WithEndpoint(srv.URL))
	client.Execute("query($id: ID!) { node(id: $id) { id } }", map[string]any{"id": "abc"})

	if !strings.Contains(gotBody, `"id":"abc"`) {
		t.Errorf("request body should contain variable, got: %s", gotBody)
	}
}

func TestExecuteGraphQLErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"data":null,"errors":[{"message":"Not authorized","path":["viewer"]}]}`))
	}))
	defer srv.Close()

	client := New("key", WithEndpoint(srv.URL))
	_, err := client.Execute("{ viewer { login } }", nil)

	if err == nil {
		t.Fatal("expected error for GraphQL errors response")
	}
	gqlErr, ok := err.(*GraphQLError)
	if !ok {
		t.Fatalf("expected *GraphQLError, got %T: %v", err, err)
	}
	if len(gqlErr.Errors) != 1 {
		t.Errorf("expected 1 error, got %d", len(gqlErr.Errors))
	}
	if !strings.Contains(gqlErr.Error(), "Not authorized") {
		t.Errorf("error message should contain 'Not authorized', got: %s", gqlErr.Error())
	}
}

func TestExecuteHTTPAuthFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	}))
	defer srv.Close()

	client := New("bad-key", WithEndpoint(srv.URL))
	_, err := client.Execute("{ viewer { login } }", nil)

	if err == nil {
		t.Fatal("expected error for 401 response")
	}
	if code := exitcode.ExitCode(err); code != exitcode.AuthFailure {
		t.Errorf("exit code = %d, want %d (AuthFailure)", code, exitcode.AuthFailure)
	}
}

func TestExecuteRateLimited(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "30")
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte("rate limited"))
	}))
	defer srv.Close()

	client := New("key", WithEndpoint(srv.URL))
	_, err := client.Execute("{ viewer { login } }", nil)

	if err == nil {
		t.Fatal("expected error for 429 response")
	}
	if !strings.Contains(err.Error(), "30 seconds") {
		t.Errorf("error should mention retry-after, got: %s", err.Error())
	}
}

func TestExecuteVerboseLogging(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"data":{"test":true}}`))
	}))
	defer srv.Close()

	var logs []string
	logFunc := func(format string, args ...any) {
		logs = append(logs, format)
	}

	client := New("key", WithEndpoint(srv.URL), WithVerbose(logFunc))
	_, err := client.Execute("{ test }", nil)
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	if len(logs) == 0 {
		t.Fatal("expected verbose log output, got none")
	}

	hasRequest := false
	hasResponse := false
	for _, log := range logs {
		if strings.Contains(log, "POST") {
			hasRequest = true
		}
		if strings.Contains(log, "←") {
			hasResponse = true
		}
	}
	if !hasRequest {
		t.Error("verbose output should include request info")
	}
	if !hasResponse {
		t.Error("verbose output should include response info")
	}
}

func TestExecuteHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := New("key", WithEndpoint(srv.URL))
	_, err := client.Execute("{ test }", nil)

	if err == nil {
		t.Fatal("expected error for 500 response")
	}
	if code := exitcode.ExitCode(err); code != exitcode.GeneralError {
		t.Errorf("exit code = %d, want %d (GeneralError)", code, exitcode.GeneralError)
	}
}
