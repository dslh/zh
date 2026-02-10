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

// --- issue connect ---

func TestIssueConnectSingle(t *testing.T) {
	resetIssueFlags()
	resetIssueConnectFlags()

	ms := setupIssueConnectServer(t)
	setupIssueTestEnv(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"issue", "connect", "task-tracker#1", "task-tracker#5"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("issue connect returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Connected") {
		t.Errorf("output should confirm connect, got: %s", out)
	}
	if !strings.Contains(out, "task-tracker#5") {
		t.Errorf("output should contain PR ref, got: %s", out)
	}
	if !strings.Contains(out, "task-tracker#1") {
		t.Errorf("output should contain issue ref, got: %s", out)
	}
}

func TestIssueConnectDryRun(t *testing.T) {
	resetIssueFlags()
	resetIssueConnectFlags()

	ms := setupIssueConnectServer(t)
	setupIssueTestEnv(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"issue", "connect", "task-tracker#1", "task-tracker#5", "--dry-run"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("issue connect --dry-run returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Would connect") {
		t.Errorf("dry run should say 'Would connect', got: %s", out)
	}
	if !strings.Contains(out, "(issue)") {
		t.Errorf("dry run should show issue context, got: %s", out)
	}
	if !strings.Contains(out, "(PR)") {
		t.Errorf("dry run should show PR context, got: %s", out)
	}
}

func TestIssueConnectWrongTypes(t *testing.T) {
	resetIssueFlags()
	resetIssueConnectFlags()

	// Both resolve as issues (not PRs) — second arg should be a PR
	ms := setupIssueConnectServerBothIssues(t)
	setupIssueTestEnv(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"issue", "connect", "task-tracker#1", "task-tracker#2"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("connect should error when second arg is not a PR")
	}
	if !strings.Contains(err.Error(), "not a pull request") {
		t.Errorf("error should mention not a PR, got: %v", err)
	}
}

func TestIssueConnectJSON(t *testing.T) {
	resetIssueFlags()
	resetIssueConnectFlags()

	ms := setupIssueConnectServer(t)
	setupIssueTestEnv(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"issue", "connect", "task-tracker#1", "task-tracker#5", "--output=json"})
	outputFormat = "json"
	defer func() { outputFormat = "" }()

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("issue connect --output=json returned error: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\nOutput: %s", err, buf.String())
	}

	issue, ok := result["issue"].(map[string]any)
	if !ok {
		t.Fatal("JSON should contain issue object")
	}
	if issue["number"] != float64(1) {
		t.Errorf("JSON issue number should be 1, got: %v", issue["number"])
	}
	pr, ok := result["pr"].(map[string]any)
	if !ok {
		t.Fatal("JSON should contain pr object")
	}
	if pr["pullRequest"] != true {
		t.Errorf("JSON pr should have pullRequest=true, got: %v", pr["pullRequest"])
	}
}

func TestIssueConnectHelp(t *testing.T) {
	resetIssueFlags()
	resetIssueConnectFlags()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"issue", "connect", "--help"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("issue connect --help returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Connect") {
		t.Errorf("help should contain Connect, got: %s", out)
	}
	if !strings.Contains(out, "--dry-run") {
		t.Error("help should mention --dry-run flag")
	}
}

// --- issue disconnect ---

func TestIssueDisconnectSingle(t *testing.T) {
	resetIssueFlags()
	resetIssueDisconnectFlags()

	ms := setupIssueDisconnectServer(t)
	setupIssueTestEnv(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"issue", "disconnect", "task-tracker#1", "task-tracker#5"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("issue disconnect returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Disconnected") {
		t.Errorf("output should confirm disconnect, got: %s", out)
	}
	if !strings.Contains(out, "task-tracker#5") {
		t.Errorf("output should contain PR ref, got: %s", out)
	}
}

func TestIssueDisconnectDryRun(t *testing.T) {
	resetIssueFlags()
	resetIssueDisconnectFlags()

	ms := setupIssueDisconnectServer(t)
	setupIssueTestEnv(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"issue", "disconnect", "task-tracker#1", "task-tracker#5", "--dry-run"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("issue disconnect --dry-run returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Would disconnect") {
		t.Errorf("dry run should say 'Would disconnect', got: %s", out)
	}
}

func TestIssueDisconnectJSON(t *testing.T) {
	resetIssueFlags()
	resetIssueDisconnectFlags()

	ms := setupIssueDisconnectServer(t)
	setupIssueTestEnv(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"issue", "disconnect", "task-tracker#1", "task-tracker#5", "--output=json"})
	outputFormat = "json"
	defer func() { outputFormat = "" }()

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("issue disconnect --output=json returned error: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\nOutput: %s", err, buf.String())
	}
	if result["issue"] == nil {
		t.Fatal("JSON should contain issue object")
	}
	if result["pr"] == nil {
		t.Fatal("JSON should contain pr object")
	}
}

func TestIssueDisconnectHelp(t *testing.T) {
	resetIssueFlags()
	resetIssueDisconnectFlags()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"issue", "disconnect", "--help"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("issue disconnect --help returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Disconnect") {
		t.Errorf("help should contain Disconnect, got: %s", out)
	}
	if !strings.Contains(out, "--dry-run") {
		t.Error("help should mention --dry-run flag")
	}
}

// --- test helpers ---

func issueConnectResolveResponse(id string, number int, title string, isPR bool) map[string]any {
	return map[string]any{
		"data": map[string]any{
			"issueByInfo": map[string]any{
				"id":          id,
				"number":      number,
				"title":       title,
				"pullRequest": isPR,
				"repository": map[string]any{
					"name":      "task-tracker",
					"ownerName": "dlakehammond",
				},
			},
		},
	}
}

func connectMutationResponse() map[string]any {
	return map[string]any{
		"data": map[string]any{
			"createIssuePrConnection": map[string]any{
				"issue": map[string]any{
					"id":     "i1",
					"number": 1,
					"title":  "Fix login button alignment",
				},
				"pullRequest": map[string]any{
					"id":     "pr5",
					"number": 5,
					"title":  "Fix button CSS",
				},
			},
		},
	}
}

func disconnectMutationResponse() map[string]any {
	return map[string]any{
		"data": map[string]any{
			"deleteIssuePrConnection": map[string]any{
				"issue": map[string]any{
					"id":     "i1",
					"number": 1,
					"title":  "Fix login button alignment",
				},
				"pullRequest": map[string]any{
					"id":     "pr5",
					"number": 5,
					"title":  "Fix button CSS",
				},
			},
		},
	}
}

func setupIssueConnectServer(t *testing.T) *testutil.MockServer {
	t.Helper()

	ms := testutil.NewMockServer(t)

	ms.HandleQuery("ListRepos", repoResolutionResponse())

	// First call resolves the issue, second call resolves the PR
	resolveCallCount := 0
	ms.Handle(
		func(req testutil.GraphQLRequest) bool {
			return strings.Contains(req.Query, "GetIssueForConnect") && !strings.Contains(req.Query, "ByNode")
		},
		func(w http.ResponseWriter, req testutil.GraphQLRequest) {
			resolveCallCount++
			var resp map[string]any
			if resolveCallCount%2 == 1 {
				resp = issueConnectResolveResponse("i1", 1, "Fix login button alignment", false)
			} else {
				resp = issueConnectResolveResponse("pr5", 5, "Fix button CSS", true)
			}
			data, _ := json.Marshal(resp)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(data)
		},
	)

	// IssueByInfo for resolve.Issue()
	issueByInfoCallCount := 0
	ms.Handle(
		func(req testutil.GraphQLRequest) bool {
			return strings.Contains(req.Query, "IssueByInfo") && !strings.Contains(req.Query, "GetIssueForConnect")
		},
		func(w http.ResponseWriter, req testutil.GraphQLRequest) {
			issueByInfoCallCount++
			var resp map[string]any
			if issueByInfoCallCount%2 == 1 {
				resp = issueByInfoResolutionResponse()
			} else {
				resp = issueByInfoPRResolutionResponse()
			}
			data, _ := json.Marshal(resp)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(data)
		},
	)

	ms.HandleQuery("CreateIssuePrConnection", connectMutationResponse())

	_ = cache.Set(resolve.RepoCacheKey("ws-123"), []resolve.CachedRepo{
		{ID: "r1", GhID: 12345, Name: "task-tracker", OwnerName: "dlakehammond"},
	})

	return ms
}

func setupIssueConnectServerBothIssues(t *testing.T) *testutil.MockServer {
	t.Helper()

	ms := testutil.NewMockServer(t)

	ms.HandleQuery("ListRepos", repoResolutionResponse())
	ms.HandleQuery("IssueByInfo", issueByInfoResolutionResponse())
	ms.HandleQuery("GetIssueForConnect", issueConnectResolveResponse("i1", 1, "Fix login button alignment", false))

	_ = cache.Set(resolve.RepoCacheKey("ws-123"), []resolve.CachedRepo{
		{ID: "r1", GhID: 12345, Name: "task-tracker", OwnerName: "dlakehammond"},
	})

	return ms
}

func setupIssueDisconnectServer(t *testing.T) *testutil.MockServer {
	t.Helper()

	ms := testutil.NewMockServer(t)

	ms.HandleQuery("ListRepos", repoResolutionResponse())

	resolveCallCount := 0
	ms.Handle(
		func(req testutil.GraphQLRequest) bool {
			return strings.Contains(req.Query, "GetIssueForConnect") && !strings.Contains(req.Query, "ByNode")
		},
		func(w http.ResponseWriter, req testutil.GraphQLRequest) {
			resolveCallCount++
			var resp map[string]any
			if resolveCallCount%2 == 1 {
				resp = issueConnectResolveResponse("i1", 1, "Fix login button alignment", false)
			} else {
				resp = issueConnectResolveResponse("pr5", 5, "Fix button CSS", true)
			}
			data, _ := json.Marshal(resp)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(data)
		},
	)

	issueByInfoCallCount := 0
	ms.Handle(
		func(req testutil.GraphQLRequest) bool {
			return strings.Contains(req.Query, "IssueByInfo") && !strings.Contains(req.Query, "GetIssueForConnect")
		},
		func(w http.ResponseWriter, req testutil.GraphQLRequest) {
			issueByInfoCallCount++
			var resp map[string]any
			if issueByInfoCallCount%2 == 1 {
				resp = issueByInfoResolutionResponse()
			} else {
				resp = issueByInfoPRResolutionResponse()
			}
			data, _ := json.Marshal(resp)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(data)
		},
	)

	ms.HandleQuery("DeleteIssuePrConnection", disconnectMutationResponse())

	_ = cache.Set(resolve.RepoCacheKey("ws-123"), []resolve.CachedRepo{
		{ID: "r1", GhID: 12345, Name: "task-tracker", OwnerName: "dlakehammond"},
	})

	return ms
}

func issueByInfoPRResolutionResponse() map[string]any {
	return map[string]any{
		"data": map[string]any{
			"issueByInfo": map[string]any{
				"id":     "pr5",
				"number": 5,
				"repository": map[string]any{
					"ghId":      12345,
					"name":      "task-tracker",
					"ownerName": "dlakehammond",
				},
			},
		},
	}
}
