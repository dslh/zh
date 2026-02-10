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

// --- issue label add ---

func TestIssueLabelAdd(t *testing.T) {
	resetIssueFlags()
	resetIssueLabelFlags()

	ms := setupIssueLabelServer(t)
	setupIssueTestEnv(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"issue", "label", "add", "task-tracker#1", "--", "bug"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("issue label add returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Added") {
		t.Errorf("output should confirm add, got: %s", out)
	}
	if !strings.Contains(out, "bug") {
		t.Errorf("output should contain label name, got: %s", out)
	}
	if !strings.Contains(out, "task-tracker#1") {
		t.Errorf("output should contain issue ref, got: %s", out)
	}
}

func TestIssueLabelAddMultipleLabels(t *testing.T) {
	resetIssueFlags()
	resetIssueLabelFlags()

	ms := setupIssueLabelServer(t)
	setupIssueTestEnv(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"issue", "label", "add", "task-tracker#1", "--", "bug", "enhancement"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("issue label add multiple returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Added") {
		t.Errorf("output should confirm add, got: %s", out)
	}
	if !strings.Contains(out, "bug") {
		t.Errorf("output should contain first label, got: %s", out)
	}
	if !strings.Contains(out, "enhancement") {
		t.Errorf("output should contain second label, got: %s", out)
	}
}

func TestIssueLabelAddBatch(t *testing.T) {
	resetIssueFlags()
	resetIssueLabelFlags()

	ms := setupIssueLabelServerBatch(t)
	setupIssueTestEnv(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"issue", "label", "add", "task-tracker#1", "task-tracker#2", "--", "bug"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("issue label add batch returned error: %v", err)
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

func TestIssueLabelAddDryRun(t *testing.T) {
	resetIssueFlags()
	resetIssueLabelFlags()

	ms := setupIssueLabelServer(t)
	setupIssueTestEnv(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"issue", "label", "add", "task-tracker#1", "--dry-run", "--", "bug"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("issue label add --dry-run returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Would add") {
		t.Errorf("dry run should say 'Would add', got: %s", out)
	}
	if !strings.Contains(out, "bug") {
		t.Errorf("dry run should contain label name, got: %s", out)
	}
	if !strings.Contains(out, "task-tracker#1") {
		t.Errorf("dry run should contain issue ref, got: %s", out)
	}
}

func TestIssueLabelAddContinueOnError(t *testing.T) {
	resetIssueFlags()
	resetIssueLabelFlags()

	ms := setupIssueLabelServerMixed(t)
	setupIssueTestEnv(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"issue", "label", "add", "task-tracker#1", "task-tracker#999", "--continue-on-error", "--", "bug"})

	if err := rootCmd.Execute(); err != nil {
		if !strings.Contains(err.Error(), "some issues failed") {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	out := buf.String()
	if !strings.Contains(out, "task-tracker#1") {
		t.Errorf("output should contain successful add, got: %s", out)
	}
	if !strings.Contains(out, "Failed") {
		t.Errorf("output should show failures, got: %s", out)
	}
}

func TestIssueLabelAddJSON(t *testing.T) {
	resetIssueFlags()
	resetIssueLabelFlags()

	ms := setupIssueLabelServer(t)
	setupIssueTestEnv(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"issue", "label", "add", "task-tracker#1", "--output=json", "--", "bug"})
	outputFormat = "json"
	defer func() { outputFormat = "" }()

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("issue label add --output=json returned error: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\nOutput: %s", err, buf.String())
	}
	if result["operation"] != "add" {
		t.Errorf("JSON should contain operation 'add', got: %v", result["operation"])
	}
	if result["successCount"] != float64(1) {
		t.Errorf("JSON should contain successCount of 1, got: %v", result["successCount"])
	}
}

// --- issue label remove ---

func TestIssueLabelRemove(t *testing.T) {
	resetIssueFlags()
	resetIssueLabelFlags()

	ms := setupIssueLabelRemoveServer(t)
	setupIssueTestEnv(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"issue", "label", "remove", "task-tracker#1", "--", "bug"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("issue label remove returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Removed") {
		t.Errorf("output should confirm remove, got: %s", out)
	}
	if !strings.Contains(out, "bug") {
		t.Errorf("output should contain label name, got: %s", out)
	}
	if !strings.Contains(out, "task-tracker#1") {
		t.Errorf("output should contain issue ref, got: %s", out)
	}
}

func TestIssueLabelRemoveDryRun(t *testing.T) {
	resetIssueFlags()
	resetIssueLabelFlags()

	ms := setupIssueLabelRemoveServer(t)
	setupIssueTestEnv(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"issue", "label", "remove", "task-tracker#1", "--dry-run", "--", "bug"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("issue label remove --dry-run returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Would remove") {
		t.Errorf("dry run should say 'Would remove', got: %s", out)
	}
	if !strings.Contains(out, "bug") {
		t.Errorf("dry run should contain label name, got: %s", out)
	}
}

func TestIssueLabelRemoveContinueOnError(t *testing.T) {
	resetIssueFlags()
	resetIssueLabelFlags()

	ms := setupIssueLabelRemoveServerMixed(t)
	setupIssueTestEnv(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"issue", "label", "remove", "task-tracker#1", "task-tracker#999", "--continue-on-error", "--", "bug"})

	if err := rootCmd.Execute(); err != nil {
		if !strings.Contains(err.Error(), "some issues failed") {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	out := buf.String()
	if !strings.Contains(out, "task-tracker#1") {
		t.Errorf("output should contain successful remove, got: %s", out)
	}
	if !strings.Contains(out, "Failed") {
		t.Errorf("output should show failures, got: %s", out)
	}
}

// --- help ---

func TestIssueLabelAddHelp(t *testing.T) {
	resetIssueFlags()
	resetIssueLabelFlags()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"issue", "label", "add", "--help"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("issue label add --help returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Add") {
		t.Errorf("help should contain 'Add', got: %s", out)
	}
	if !strings.Contains(out, "--dry-run") {
		t.Error("help should mention --dry-run flag")
	}
	if !strings.Contains(out, "--continue-on-error") {
		t.Error("help should mention --continue-on-error flag")
	}
}

func TestIssueLabelRemoveHelp(t *testing.T) {
	resetIssueFlags()
	resetIssueLabelFlags()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"issue", "label", "remove", "--help"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("issue label remove --help returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Remove") {
		t.Errorf("help should contain 'Remove', got: %s", out)
	}
	if !strings.Contains(out, "--dry-run") {
		t.Error("help should mention --dry-run flag")
	}
}

// --- test helpers ---

func setupIssueLabelServer(t *testing.T) *testutil.MockServer {
	t.Helper()

	ms := testutil.NewMockServer(t)

	ms.HandleQuery("ListRepos", repoResolutionResponse())
	ms.HandleQuery("IssueByInfo", issueByInfoResolutionResponse())
	ms.HandleQuery("GetIssueForLabel", issueLabelResolveResponseHelper("i1", 1, "Fix login button alignment"))
	ms.HandleQuery("GetWorkspaceLabels", workspaceLabelsResponse())
	ms.HandleQuery("AddLabelsToIssues", labelMutationResponse(1))

	_ = cache.Set(resolve.RepoCacheKey("ws-123"), []resolve.CachedRepo{
		{ID: "r1", GhID: 12345, Name: "task-tracker", OwnerName: "dlakehammond"},
	})

	return ms
}

func setupIssueLabelServerBatch(t *testing.T) *testutil.MockServer {
	t.Helper()

	ms := testutil.NewMockServer(t)

	ms.HandleQuery("ListRepos", repoResolutionResponse())

	issueCallCount := 0
	ms.Handle(
		func(req testutil.GraphQLRequest) bool {
			return strings.Contains(req.Query, "IssueByInfo") && !strings.Contains(req.Query, "GetIssueForLabel")
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

	labelResolveCount := 0
	ms.Handle(
		func(req testutil.GraphQLRequest) bool {
			return strings.Contains(req.Query, "GetIssueForLabel")
		},
		func(w http.ResponseWriter, req testutil.GraphQLRequest) {
			labelResolveCount++
			var resp map[string]any
			if labelResolveCount%2 == 1 {
				resp = issueLabelResolveResponseHelper("i1", 1, "Fix login button alignment")
			} else {
				resp = issueLabelResolveResponseHelper("i2", 2, "Add error handling")
			}
			data, _ := json.Marshal(resp)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(data)
		},
	)

	ms.HandleQuery("GetWorkspaceLabels", workspaceLabelsResponse())
	ms.HandleQuery("AddLabelsToIssues", labelMutationResponse(2))

	_ = cache.Set(resolve.RepoCacheKey("ws-123"), []resolve.CachedRepo{
		{ID: "r1", GhID: 12345, Name: "task-tracker", OwnerName: "dlakehammond"},
	})

	return ms
}

func setupIssueLabelServerMixed(t *testing.T) *testutil.MockServer {
	t.Helper()

	ms := testutil.NewMockServer(t)

	ms.HandleQuery("ListRepos", repoResolutionResponse())

	callCount := 0
	ms.Handle(
		func(req testutil.GraphQLRequest) bool {
			return strings.Contains(req.Query, "IssueByInfo") && !strings.Contains(req.Query, "GetIssueForLabel")
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

	ms.HandleQuery("GetIssueForLabel", issueLabelResolveResponseHelper("i1", 1, "Fix login button alignment"))
	ms.HandleQuery("GetWorkspaceLabels", workspaceLabelsResponse())
	ms.HandleQuery("AddLabelsToIssues", labelMutationResponse(1))

	_ = cache.Set(resolve.RepoCacheKey("ws-123"), []resolve.CachedRepo{
		{ID: "r1", GhID: 12345, Name: "task-tracker", OwnerName: "dlakehammond"},
	})

	return ms
}

func setupIssueLabelRemoveServer(t *testing.T) *testutil.MockServer {
	t.Helper()

	ms := testutil.NewMockServer(t)

	ms.HandleQuery("ListRepos", repoResolutionResponse())
	ms.HandleQuery("IssueByInfo", issueByInfoResolutionResponse())
	ms.HandleQuery("GetIssueForLabel", issueLabelResolveResponseHelper("i1", 1, "Fix login button alignment"))
	ms.HandleQuery("GetWorkspaceLabels", workspaceLabelsResponse())
	ms.HandleQuery("RemoveLabelsFromIssues", labelMutationResponse(1))

	_ = cache.Set(resolve.RepoCacheKey("ws-123"), []resolve.CachedRepo{
		{ID: "r1", GhID: 12345, Name: "task-tracker", OwnerName: "dlakehammond"},
	})

	return ms
}

func setupIssueLabelRemoveServerMixed(t *testing.T) *testutil.MockServer {
	t.Helper()

	ms := testutil.NewMockServer(t)

	ms.HandleQuery("ListRepos", repoResolutionResponse())

	callCount := 0
	ms.Handle(
		func(req testutil.GraphQLRequest) bool {
			return strings.Contains(req.Query, "IssueByInfo") && !strings.Contains(req.Query, "GetIssueForLabel")
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

	ms.HandleQuery("GetIssueForLabel", issueLabelResolveResponseHelper("i1", 1, "Fix login button alignment"))
	ms.HandleQuery("GetWorkspaceLabels", workspaceLabelsResponse())
	ms.HandleQuery("RemoveLabelsFromIssues", labelMutationResponse(1))

	_ = cache.Set(resolve.RepoCacheKey("ws-123"), []resolve.CachedRepo{
		{ID: "r1", GhID: 12345, Name: "task-tracker", OwnerName: "dlakehammond"},
	})

	return ms
}

// --- response helpers ---

func issueLabelResolveResponseHelper(id string, number int, title string) map[string]any {
	return map[string]any{
		"data": map[string]any{
			"node": map[string]any{
				"id":     id,
				"number": number,
				"title":  title,
				"repository": map[string]any{
					"name":      "task-tracker",
					"ownerName": "dlakehammond",
				},
			},
		},
	}
}

func workspaceLabelsResponse() map[string]any {
	return map[string]any{
		"data": map[string]any{
			"workspace": map[string]any{
				"repositoriesConnection": map[string]any{
					"nodes": []any{
						map[string]any{
							"labels": map[string]any{
								"nodes": []any{
									map[string]any{"id": "l1", "name": "bug", "color": "d73a4a"},
									map[string]any{"id": "l2", "name": "enhancement", "color": "a2eeef"},
								},
							},
						},
					},
				},
			},
		},
	}
}

func labelMutationResponse(successCount int) map[string]any {
	return map[string]any{
		"data": map[string]any{
			"addLabelsToIssues": map[string]any{
				"successCount": successCount,
				"failedIssues": []any{},
				"labels": []any{
					map[string]any{"id": "l1", "name": "bug", "color": "d73a4a"},
				},
				"githubErrors": "[]",
			},
			"removeLabelsFromIssues": map[string]any{
				"successCount": successCount,
				"failedIssues": []any{},
				"labels": []any{
					map[string]any{"id": "l1", "name": "bug", "color": "d73a4a"},
				},
				"githubErrors": "[]",
			},
		},
	}
}
