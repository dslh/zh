package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestUpdateEpicIssues(t *testing.T) {
	var receivedBody map[string]any
	var receivedPath string
	var receivedAuth string

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		receivedAuth = r.Header.Get("X-Authentication-Token")

		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &receivedBody)

		w.WriteHeader(200)
	}))
	defer ts.Close()

	client := New("test-api-key", WithEndpoint(ts.URL))

	addIssues := []RESTIssueRef{
		{RepoID: 123, IssueNumber: 42},
	}
	removeIssues := []RESTIssueRef{
		{RepoID: 456, IssueNumber: 99},
	}

	err := client.UpdateEpicIssues(789, 10, addIssues, removeIssues)
	if err != nil {
		t.Fatalf("UpdateEpicIssues returned error: %v", err)
	}

	if receivedPath != "/p1/repositories/789/epics/10/update_issues" {
		t.Errorf("unexpected path: %s", receivedPath)
	}

	if receivedAuth != "test-api-key" {
		t.Errorf("unexpected auth header: %s", receivedAuth)
	}

	add := receivedBody["add_issues"].([]any)
	if len(add) != 1 {
		t.Fatalf("expected 1 add_issues, got %d", len(add))
	}
	addItem := add[0].(map[string]any)
	if addItem["repo_id"] != float64(123) {
		t.Errorf("add_issues[0].repo_id = %v, want 123", addItem["repo_id"])
	}
	if addItem["issue_number"] != float64(42) {
		t.Errorf("add_issues[0].issue_number = %v, want 42", addItem["issue_number"])
	}

	remove := receivedBody["remove_issues"].([]any)
	if len(remove) != 1 {
		t.Fatalf("expected 1 remove_issues, got %d", len(remove))
	}
	removeItem := remove[0].(map[string]any)
	if removeItem["repo_id"] != float64(456) {
		t.Errorf("remove_issues[0].repo_id = %v, want 456", removeItem["repo_id"])
	}
	if removeItem["issue_number"] != float64(99) {
		t.Errorf("remove_issues[0].issue_number = %v, want 99", removeItem["issue_number"])
	}
}

func TestUpdateEpicIssuesAuthFailure(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		_, _ = w.Write([]byte(`{"message":"Unauthorized"}`))
	}))
	defer ts.Close()

	client := New("bad-key", WithEndpoint(ts.URL))

	err := client.UpdateEpicIssues(789, 10, []RESTIssueRef{{RepoID: 123, IssueNumber: 42}}, nil)
	if err == nil {
		t.Fatal("expected error for 401 response")
	}
	if !strings.Contains(err.Error(), "authentication failed") {
		t.Errorf("error should mention auth failure, got: %v", err)
	}
}

func TestRESTEndpoint(t *testing.T) {
	// Default endpoint
	c := New("key")
	if ep := c.RESTEndpoint(); ep != "https://api.zenhub.com" {
		t.Errorf("default REST endpoint = %q, want https://api.zenhub.com", ep)
	}

	// Custom endpoint (test server)
	c2 := New("key", WithEndpoint("http://localhost:12345"))
	if ep := c2.RESTEndpoint(); ep != "http://localhost:12345" {
		t.Errorf("custom REST endpoint = %q, want http://localhost:12345", ep)
	}

	// Custom endpoint with GraphQL suffix
	c3 := New("key", WithEndpoint("http://localhost:12345/public/graphql"))
	if ep := c3.RESTEndpoint(); ep != "http://localhost:12345" {
		t.Errorf("custom REST endpoint with suffix = %q, want http://localhost:12345", ep)
	}
}

