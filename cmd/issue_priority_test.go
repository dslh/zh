package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/dslh/zh/internal/cache"
	"github.com/dslh/zh/internal/resolve"
	"github.com/dslh/zh/internal/testutil"
)

// --- issue priority set ---

func TestIssuePrioritySet(t *testing.T) {
	resetIssueFlags()
	resetIssuePriorityFlags()

	ms := setupIssuePriorityServer(t, "OPEN", "")
	setupIssueTestEnv(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"issue", "priority", "task-tracker#1", "high"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("issue priority set returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Set priority") {
		t.Errorf("output should confirm set, got: %s", out)
	}
	if !strings.Contains(out, "High priority") {
		t.Errorf("output should contain priority name, got: %s", out)
	}
	if !strings.Contains(out, "task-tracker#1") {
		t.Errorf("output should contain issue ref, got: %s", out)
	}
}

func TestIssuePrioritySetBatch(t *testing.T) {
	resetIssueFlags()
	resetIssuePriorityFlags()

	ms := setupIssuePriorityServerBatch(t)
	setupIssueTestEnv(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"issue", "priority", "task-tracker#1", "task-tracker#2", "high"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("issue priority batch returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "2 issue(s)") {
		t.Errorf("output should show batch count, got: %s", out)
	}
	if !strings.Contains(out, "task-tracker#1") {
		t.Errorf("output should contain first issue ref, got: %s", out)
	}
	if !strings.Contains(out, "task-tracker#2") {
		t.Errorf("output should contain second issue ref, got: %s", out)
	}
}

func TestIssuePriorityClear(t *testing.T) {
	resetIssueFlags()
	resetIssuePriorityFlags()

	ms := setupIssuePriorityServer(t, "OPEN", "High priority")
	setupIssueTestEnv(t, ms)

	// Only one arg and it's an issue — should clear priority
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"issue", "priority", "task-tracker#1"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("issue priority clear returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Cleared priority") {
		t.Errorf("output should confirm clear, got: %s", out)
	}
	if !strings.Contains(out, "task-tracker#1") {
		t.Errorf("output should contain issue ref, got: %s", out)
	}
}

func TestIssuePriorityClearFlag(t *testing.T) {
	resetIssueFlags()
	resetIssuePriorityFlags()

	ms := setupIssuePriorityServer(t, "OPEN", "High priority")
	setupIssueTestEnv(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"issue", "priority", "task-tracker#1", "--clear"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("issue priority --clear returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Cleared priority") {
		t.Errorf("output should confirm clear, got: %s", out)
	}
}

func TestIssuePriorityDryRun(t *testing.T) {
	resetIssueFlags()
	resetIssuePriorityFlags()

	ms := setupIssuePriorityServer(t, "OPEN", "Low priority")
	setupIssueTestEnv(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"issue", "priority", "task-tracker#1", "high", "--dry-run"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("issue priority --dry-run returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Would set priority") {
		t.Errorf("dry run should say 'Would set priority', got: %s", out)
	}
	if !strings.Contains(out, "task-tracker#1") {
		t.Errorf("dry run should contain issue ref, got: %s", out)
	}
	if !strings.Contains(out, "Low priority") {
		t.Errorf("dry run should show current priority, got: %s", out)
	}
}

func TestIssuePriorityDryRunClear(t *testing.T) {
	resetIssueFlags()
	resetIssuePriorityFlags()

	ms := setupIssuePriorityServer(t, "OPEN", "High priority")
	setupIssueTestEnv(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"issue", "priority", "task-tracker#1", "--dry-run"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("issue priority --dry-run clear returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Would clear priority") {
		t.Errorf("dry run should say 'Would clear priority', got: %s", out)
	}
}

func TestIssuePriorityInvalid(t *testing.T) {
	resetIssueFlags()
	resetIssuePriorityFlags()

	ms := setupIssuePriorityServer(t, "OPEN", "")
	setupIssueTestEnv(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"issue", "priority", "task-tracker#1", "nonexistent"})

	// When there are 2+ args and the last arg doesn't match a priority,
	// the command returns the priority resolution error
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("issue priority should error for nonexistent priority used as issue identifier")
	}
}

func TestIssuePriorityContinueOnError(t *testing.T) {
	resetIssueFlags()
	resetIssuePriorityFlags()

	ms := setupIssuePriorityServerMixed(t)
	setupIssueTestEnv(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"issue", "priority", "task-tracker#1", "task-tracker#999", "high", "--continue-on-error"})

	if err := rootCmd.Execute(); err != nil {
		if !strings.Contains(err.Error(), "some issues failed") {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	out := buf.String()
	if !strings.Contains(out, "task-tracker#1") {
		t.Errorf("output should contain successful set, got: %s", out)
	}
	if !strings.Contains(out, "Failed") {
		t.Errorf("output should show failures, got: %s", out)
	}
}

func TestIssuePriorityJSON(t *testing.T) {
	resetIssueFlags()
	resetIssuePriorityFlags()

	ms := setupIssuePriorityServer(t, "OPEN", "")
	setupIssueTestEnv(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"issue", "priority", "task-tracker#1", "high", "--output=json"})
	outputFormat = "json"
	defer func() { outputFormat = "" }()

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("issue priority --output=json returned error: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\nOutput: %s", err, buf.String())
	}
	priority, ok := result["priority"].(map[string]any)
	if !ok {
		t.Fatal("JSON should contain priority object")
	}
	if priority["name"] != "High priority" {
		t.Errorf("JSON priority name should be 'High priority', got: %v", priority["name"])
	}
}

func TestIssuePriorityHelp(t *testing.T) {
	resetIssueFlags()
	resetIssuePriorityFlags()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"issue", "priority", "--help"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("issue priority --help returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "priority") {
		t.Errorf("help should contain 'priority', got: %s", out)
	}
	if !strings.Contains(out, "--dry-run") {
		t.Error("help should mention --dry-run flag")
	}
	if !strings.Contains(out, "--continue-on-error") {
		t.Error("help should mention --continue-on-error flag")
	}
}

// --- test helpers ---

func setupIssuePriorityServer(t *testing.T, state, currentPriority string) *testutil.MockServer {
	t.Helper()

	ms := testutil.NewMockServer(t)

	ms.HandleQuery("ListRepos", repoResolutionResponse())
	ms.HandleQuery("IssueByInfo", issueByInfoResolutionResponse())
	ms.HandleQuery("GetWorkspacePriorities", prioritiesResponse())
	ms.HandleQuery("GetIssueForPriority", issuePriorityResolveResponse("i1", 1, "Fix login button alignment", 12345, currentPriority))
	ms.HandleQuery("SetIssuePriority", setPriorityResponse())
	ms.HandleQuery("RemoveIssuePriority", removePriorityResponse())

	_ = cache.Set(resolve.RepoCacheKey("ws-123"), []resolve.CachedRepo{
		{ID: "r1", GhID: 12345, Name: "task-tracker", OwnerName: "dlakehammond"},
	})

	return ms
}

func setupIssuePriorityServerBatch(t *testing.T) *testutil.MockServer {
	t.Helper()

	ms := testutil.NewMockServer(t)

	ms.HandleQuery("ListRepos", repoResolutionResponse())
	ms.HandleQuery("GetWorkspacePriorities", prioritiesResponse())

	issueCallCount := 0
	ms.Handle(
		func(req testutil.GraphQLRequest) bool {
			return strings.Contains(req.Query, "IssueByInfo") && !strings.Contains(req.Query, "GetIssueForPriority")
		},
		func(w http.ResponseWriter, req testutil.GraphQLRequest) {
			issueCallCount++
			var resp map[string]any
			if issueCallCount%2 == 1 {
				resp = issueByInfoResolutionResponse()
			} else {
				resp = issueByInfoResolutionResponse2()
			}
			data, _ := json.Marshal(resp)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(data)
		},
	)

	priorityResolveCount := 0
	ms.Handle(
		func(req testutil.GraphQLRequest) bool {
			return strings.Contains(req.Query, "GetIssueForPriority")
		},
		func(w http.ResponseWriter, req testutil.GraphQLRequest) {
			priorityResolveCount++
			var resp map[string]any
			if priorityResolveCount%2 == 1 {
				resp = issuePriorityResolveResponse("i1", 1, "Fix login button alignment", 12345, "")
			} else {
				resp = issuePriorityResolveResponse("i2", 2, "Add error handling", 12345, "Low priority")
			}
			data, _ := json.Marshal(resp)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(data)
		},
	)

	ms.HandleQuery("SetIssuePriority", setPriorityResponse())

	_ = cache.Set(resolve.RepoCacheKey("ws-123"), []resolve.CachedRepo{
		{ID: "r1", GhID: 12345, Name: "task-tracker", OwnerName: "dlakehammond"},
	})

	return ms
}

func setupIssuePriorityServerMixed(t *testing.T) *testutil.MockServer {
	t.Helper()

	ms := testutil.NewMockServer(t)

	ms.HandleQuery("ListRepos", repoResolutionResponse())
	ms.HandleQuery("GetWorkspacePriorities", prioritiesResponse())

	callCount := 0
	ms.Handle(
		func(req testutil.GraphQLRequest) bool {
			return strings.Contains(req.Query, "IssueByInfo") && !strings.Contains(req.Query, "GetIssueForPriority")
		},
		func(w http.ResponseWriter, req testutil.GraphQLRequest) {
			callCount++
			var resp map[string]any
			if callCount == 1 {
				resp = issueByInfoResolutionResponse()
			} else {
				resp = map[string]any{
					"data": map[string]any{
						"issueByInfo": nil,
					},
				}
			}
			data, _ := json.Marshal(resp)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(data)
		},
	)

	ms.HandleQuery("GetIssueForPriority", issuePriorityResolveResponse("i1", 1, "Fix login button alignment", 12345, ""))
	ms.HandleQuery("SetIssuePriority", setPriorityResponse())

	_ = cache.Set(resolve.RepoCacheKey("ws-123"), []resolve.CachedRepo{
		{ID: "r1", GhID: 12345, Name: "task-tracker", OwnerName: "dlakehammond"},
	})

	return ms
}

// --- response helpers ---

func prioritiesResponse() map[string]any {
	return map[string]any{
		"data": map[string]any{
			"workspace": map[string]any{
				"prioritiesConnection": map[string]any{
					"nodes": []any{
						map[string]any{"id": "pri1", "name": "Urgent", "color": "FF0000", "description": ""},
						map[string]any{"id": "pri2", "name": "High priority", "color": "FF8800", "description": ""},
						map[string]any{"id": "pri3", "name": "Medium priority", "color": "FFCC00", "description": ""},
						map[string]any{"id": "pri4", "name": "Low priority", "color": "00CC00", "description": ""},
					},
				},
			},
		},
	}
}

func issuePriorityResolveResponse(id string, number int, title string, repoGhID int, currentPriority string) map[string]any {
	var priorityData any
	if currentPriority != "" {
		priorityData = map[string]any{
			"id":   "pri-current",
			"name": currentPriority,
		}
	}

	return map[string]any{
		"data": map[string]any{
			"node": map[string]any{
				"id":     id,
				"number": number,
				"title":  title,
				"repository": map[string]any{
					"name":      "task-tracker",
					"ownerName": "dlakehammond",
					"ghId":      repoGhID,
				},
				"pipelineIssue": map[string]any{
					"priority": priorityData,
				},
			},
		},
	}
}

func setPriorityResponse() map[string]any {
	return map[string]any{
		"data": map[string]any{
			"setIssueInfoPriorities": map[string]any{
				"pipelineIssues": []any{
					map[string]any{
						"id": "pi1",
						"priority": map[string]any{
							"id":    "pri2",
							"name":  "High priority",
							"color": "FF8800",
						},
						"issue": map[string]any{
							"id":     "i1",
							"number": 1,
							"title":  "Fix login button alignment",
							"repository": map[string]any{
								"name":      "task-tracker",
								"ownerName": "dlakehammond",
							},
						},
					},
				},
			},
		},
	}
}

func removePriorityResponse() map[string]any {
	return map[string]any{
		"data": map[string]any{
			"removeIssueInfoPriorities": map[string]any{
				"pipelineIssues": []any{
					map[string]any{
						"id":       "pi1",
						"priority": nil,
						"issue": map[string]any{
							"id":     "i1",
							"number": 1,
							"title":  "Fix login button alignment",
							"repository": map[string]any{
								"name":      "task-tracker",
								"ownerName": "dlakehammond",
							},
						},
					},
				},
			},
		},
	}
}
