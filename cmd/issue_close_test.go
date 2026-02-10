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

// --- issue close ---

func TestIssueCloseSingle(t *testing.T) {
	resetIssueFlags()
	resetIssueCloseFlags()

	ms := setupIssueCloseServer(t)
	setupIssueTestEnv(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"issue", "close", "task-tracker#1"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("issue close returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Closed") {
		t.Errorf("output should confirm close, got: %s", out)
	}
	if !strings.Contains(out, "task-tracker#1") {
		t.Errorf("output should contain issue ref, got: %s", out)
	}
}

func TestIssueCloseBatch(t *testing.T) {
	resetIssueFlags()
	resetIssueCloseFlags()

	ms := setupIssueCloseServerBatch(t)
	setupIssueTestEnv(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"issue", "close", "task-tracker#1", "task-tracker#2"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("issue close batch returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Closed 2 issue(s)") {
		t.Errorf("output should show batch count, got: %s", out)
	}
	if !strings.Contains(out, "task-tracker#1") {
		t.Errorf("output should contain first issue ref, got: %s", out)
	}
	if !strings.Contains(out, "task-tracker#2") {
		t.Errorf("output should contain second issue ref, got: %s", out)
	}
}

func TestIssueCloseDryRun(t *testing.T) {
	resetIssueFlags()
	resetIssueCloseFlags()

	ms := setupIssueCloseServer(t)
	setupIssueTestEnv(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"issue", "close", "task-tracker#1", "--dry-run"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("issue close --dry-run returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Would close") {
		t.Errorf("dry run should say 'Would close', got: %s", out)
	}
	if !strings.Contains(out, "task-tracker#1") {
		t.Errorf("dry run should contain issue ref, got: %s", out)
	}
	if !strings.Contains(out, "(open)") {
		t.Errorf("dry run should show state context, got: %s", out)
	}
}

func TestIssueCloseDryRunJSON(t *testing.T) {
	resetIssueFlags()
	resetIssueCloseFlags()

	ms := setupIssueCloseServer(t)
	setupIssueTestEnv(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"issue", "close", "task-tracker#1", "--dry-run", "--output=json"})
	outputFormat = "json"
	defer func() { outputFormat = "" }()

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("issue close --dry-run --output=json returned error: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\nOutput: %s", err, buf.String())
	}
	if result["dryRun"] != true {
		t.Error("JSON should contain dryRun: true")
	}
	wouldClose, ok := result["wouldClose"].([]any)
	if !ok || len(wouldClose) != 1 {
		t.Errorf("JSON should contain wouldClose array with 1 item, got: %v", result["wouldClose"])
	}
}

func TestIssueCloseAlreadyClosed(t *testing.T) {
	resetIssueFlags()
	resetIssueCloseFlags()

	ms := setupIssueCloseServerAlreadyClosed(t)
	setupIssueTestEnv(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"issue", "close", "task-tracker#10"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("issue close already-closed returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "already closed") {
		t.Errorf("output should mention already closed, got: %s", out)
	}
}

func TestIssueCloseStopOnError(t *testing.T) {
	resetIssueFlags()
	resetIssueCloseFlags()

	ms := setupIssueCloseServerNotFound(t)
	setupIssueTestEnv(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"issue", "close", "task-tracker#999"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("issue close should error for nonexistent issue")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should mention not found, got: %v", err)
	}
}

func TestIssueCloseContinueOnError(t *testing.T) {
	resetIssueFlags()
	resetIssueCloseFlags()

	ms := setupIssueCloseServerMixed(t)
	setupIssueTestEnv(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"issue", "close", "task-tracker#1", "task-tracker#999", "--continue-on-error"})

	if err := rootCmd.Execute(); err != nil {
		// May return error due to partial failure
		if !strings.Contains(err.Error(), "some issues failed") {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	out := buf.String()
	if !strings.Contains(out, "task-tracker#1") {
		t.Errorf("output should contain successful close, got: %s", out)
	}
	if !strings.Contains(out, "Failed") {
		t.Errorf("output should show failures, got: %s", out)
	}
}

func TestIssueCloseJSON(t *testing.T) {
	resetIssueFlags()
	resetIssueCloseFlags()

	ms := setupIssueCloseServer(t)
	setupIssueTestEnv(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"issue", "close", "task-tracker#1", "--output=json"})
	outputFormat = "json"
	defer func() { outputFormat = "" }()

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("issue close --output=json returned error: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\nOutput: %s", err, buf.String())
	}
	if result["successCount"] != float64(1) {
		t.Errorf("JSON should contain successCount of 1, got: %v", result["successCount"])
	}
}

func TestIssueCloseHelp(t *testing.T) {
	resetIssueFlags()
	resetIssueCloseFlags()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"issue", "close", "--help"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("issue close --help returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Close") {
		t.Errorf("help should contain Close, got: %s", out)
	}
	if !strings.Contains(out, "--dry-run") {
		t.Error("help should mention --dry-run flag")
	}
	if !strings.Contains(out, "--continue-on-error") {
		t.Error("help should mention --continue-on-error flag")
	}
}

// --- issue reopen ---

func TestIssueReopenSingle(t *testing.T) {
	resetIssueFlags()
	resetIssueReopenFlags()

	ms := setupIssueReopenServer(t)
	setupIssueTestEnv(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"issue", "reopen", "task-tracker#10", "--pipeline=New Issues"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("issue reopen returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Reopened") {
		t.Errorf("output should confirm reopen, got: %s", out)
	}
	if !strings.Contains(out, "task-tracker#10") {
		t.Errorf("output should contain issue ref, got: %s", out)
	}
	if !strings.Contains(out, "New Issues") {
		t.Errorf("output should contain pipeline name, got: %s", out)
	}
}

func TestIssueReopenBatch(t *testing.T) {
	resetIssueFlags()
	resetIssueReopenFlags()

	ms := setupIssueReopenServerBatch(t)
	setupIssueTestEnv(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"issue", "reopen", "task-tracker#10", "task-tracker#11", "--pipeline=New Issues"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("issue reopen batch returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Reopened 2 issue(s)") {
		t.Errorf("output should show batch count, got: %s", out)
	}
}

func TestIssueReopenDryRun(t *testing.T) {
	resetIssueFlags()
	resetIssueReopenFlags()

	ms := setupIssueReopenServer(t)
	setupIssueTestEnv(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"issue", "reopen", "task-tracker#10", "--pipeline=New Issues", "--dry-run"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("issue reopen --dry-run returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Would reopen") {
		t.Errorf("dry run should say 'Would reopen', got: %s", out)
	}
	if !strings.Contains(out, "New Issues") {
		t.Errorf("dry run should contain pipeline name, got: %s", out)
	}
	if !strings.Contains(out, "(closed)") {
		t.Errorf("dry run should show state context, got: %s", out)
	}
}

func TestIssueReopenPositionTop(t *testing.T) {
	resetIssueFlags()
	resetIssueReopenFlags()

	ms := setupIssueReopenServer(t)
	setupIssueTestEnv(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"issue", "reopen", "task-tracker#10", "--pipeline=New Issues", "--position=top"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("issue reopen --position=top returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Reopened") {
		t.Errorf("output should confirm reopen, got: %s", out)
	}
}

func TestIssueReopenPositionInvalid(t *testing.T) {
	resetIssueFlags()
	resetIssueReopenFlags()

	ms := setupIssueReopenServer(t)
	setupIssueTestEnv(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"issue", "reopen", "task-tracker#10", "--pipeline=New Issues", "--position=3"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("numeric position should return error for reopen")
	}
	if !strings.Contains(err.Error(), "invalid position") {
		t.Errorf("error should mention invalid position, got: %v", err)
	}
}

func TestIssueReopenMissingPipeline(t *testing.T) {
	resetIssueFlags()
	resetIssueReopenFlags()

	ms := setupIssueReopenServer(t)
	setupIssueTestEnv(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"issue", "reopen", "task-tracker#10"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("reopen without --pipeline should return error")
	}
	if !strings.Contains(err.Error(), "pipeline") {
		t.Errorf("error should mention pipeline, got: %v", err)
	}
}

func TestIssueReopenAlreadyOpen(t *testing.T) {
	resetIssueFlags()
	resetIssueReopenFlags()

	ms := setupIssueReopenServerAlreadyOpen(t)
	setupIssueTestEnv(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"issue", "reopen", "task-tracker#1", "--pipeline=New Issues"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("issue reopen already-open returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "already open") {
		t.Errorf("output should mention already open, got: %s", out)
	}
}

func TestIssueReopenContinueOnError(t *testing.T) {
	resetIssueFlags()
	resetIssueReopenFlags()

	ms := setupIssueReopenServerMixed(t)
	setupIssueTestEnv(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"issue", "reopen", "task-tracker#10", "task-tracker#999", "--pipeline=New Issues", "--continue-on-error"})

	if err := rootCmd.Execute(); err != nil {
		if !strings.Contains(err.Error(), "some issues failed") {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	out := buf.String()
	if !strings.Contains(out, "task-tracker#10") {
		t.Errorf("output should contain successful reopen, got: %s", out)
	}
	if !strings.Contains(out, "Failed") {
		t.Errorf("output should show failures, got: %s", out)
	}
}

func TestIssueReopenJSON(t *testing.T) {
	resetIssueFlags()
	resetIssueReopenFlags()

	ms := setupIssueReopenServer(t)
	setupIssueTestEnv(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"issue", "reopen", "task-tracker#10", "--pipeline=New Issues", "--output=json"})
	outputFormat = "json"
	defer func() { outputFormat = "" }()

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("issue reopen --output=json returned error: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\nOutput: %s", err, buf.String())
	}
	pipeline, ok := result["pipeline"].(map[string]any)
	if !ok {
		t.Fatal("JSON should contain pipeline object")
	}
	if pipeline["name"] != "New Issues" {
		t.Errorf("JSON pipeline name should be 'New Issues', got: %v", pipeline["name"])
	}
}

func TestIssueReopenDryRunJSON(t *testing.T) {
	resetIssueFlags()
	resetIssueReopenFlags()

	ms := setupIssueReopenServer(t)
	setupIssueTestEnv(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"issue", "reopen", "task-tracker#10", "--pipeline=New Issues", "--dry-run", "--output=json"})
	outputFormat = "json"
	defer func() { outputFormat = "" }()

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("issue reopen --dry-run --output=json returned error: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\nOutput: %s", err, buf.String())
	}
	if result["dryRun"] != true {
		t.Errorf("JSON should have dryRun=true, got: %v", result["dryRun"])
	}
	pipeline, ok := result["pipeline"].(map[string]any)
	if !ok {
		t.Fatal("JSON should contain pipeline object")
	}
	if pipeline["name"] != "New Issues" {
		t.Errorf("JSON pipeline name should be 'New Issues', got: %v", pipeline["name"])
	}
	wouldReopen, ok := result["wouldReopen"].([]any)
	if !ok || len(wouldReopen) == 0 {
		t.Error("JSON should contain non-empty wouldReopen array")
	}
}

func TestIssueReopenHelp(t *testing.T) {
	resetIssueFlags()
	resetIssueReopenFlags()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"issue", "reopen", "--help"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("issue reopen --help returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Reopen") {
		t.Errorf("help should contain Reopen, got: %s", out)
	}
	if !strings.Contains(out, "--pipeline") {
		t.Error("help should mention --pipeline flag")
	}
	if !strings.Contains(out, "--position") {
		t.Error("help should mention --position flag")
	}
	if !strings.Contains(out, "--dry-run") {
		t.Error("help should mention --dry-run flag")
	}
}

// --- test helpers ---

func setupIssueCloseServer(t *testing.T) *testutil.MockServer {
	t.Helper()

	ms := testutil.NewMockServer(t)

	ms.HandleQuery("ListRepos", repoResolutionResponse())
	ms.HandleQuery("IssueByInfo", issueByInfoResolutionResponse())
	ms.HandleQuery("GetIssueForClose", issueCloseResolveResponse("i1", 1, "Fix login button alignment", "OPEN"))
	ms.HandleQuery("CloseIssues", closeIssuesResponse(1))

	_ = cache.Set(resolve.RepoCacheKey("ws-123"), []resolve.CachedRepo{
		{ID: "r1", GhID: 12345, Name: "task-tracker", OwnerName: "dlakehammond"},
	})

	return ms
}

func setupIssueCloseServerBatch(t *testing.T) *testutil.MockServer {
	t.Helper()

	ms := testutil.NewMockServer(t)

	ms.HandleQuery("ListRepos", repoResolutionResponse())

	issueCallCount := 0
	ms.Handle(
		func(req testutil.GraphQLRequest) bool {
			return strings.Contains(req.Query, "IssueByInfo") && !strings.Contains(req.Query, "GetIssueForClose")
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

	closeResolveCount := 0
	ms.Handle(
		func(req testutil.GraphQLRequest) bool {
			return strings.Contains(req.Query, "GetIssueForClose")
		},
		func(w http.ResponseWriter, req testutil.GraphQLRequest) {
			closeResolveCount++
			var resp map[string]any
			if closeResolveCount%2 == 1 {
				resp = issueCloseResolveResponse("i1", 1, "Fix login button alignment", "OPEN")
			} else {
				resp = issueCloseResolveResponse("i2", 2, "Add error handling", "OPEN")
			}
			data, _ := json.Marshal(resp)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(data)
		},
	)

	ms.HandleQuery("CloseIssues", closeIssuesResponse(2))

	_ = cache.Set(resolve.RepoCacheKey("ws-123"), []resolve.CachedRepo{
		{ID: "r1", GhID: 12345, Name: "task-tracker", OwnerName: "dlakehammond"},
	})

	return ms
}

func setupIssueCloseServerAlreadyClosed(t *testing.T) *testutil.MockServer {
	t.Helper()

	ms := testutil.NewMockServer(t)

	ms.HandleQuery("ListRepos", repoResolutionResponse())
	ms.HandleQuery("IssueByInfo", issueByInfoResolutionResponseClosed())
	ms.HandleQuery("GetIssueForClose", issueCloseResolveResponse("i10", 10, "Old closed issue", "CLOSED"))

	_ = cache.Set(resolve.RepoCacheKey("ws-123"), []resolve.CachedRepo{
		{ID: "r1", GhID: 12345, Name: "task-tracker", OwnerName: "dlakehammond"},
	})

	return ms
}

func setupIssueCloseServerNotFound(t *testing.T) *testutil.MockServer {
	t.Helper()

	ms := testutil.NewMockServer(t)

	ms.HandleQuery("ListRepos", repoResolutionResponse())
	ms.HandleQuery("IssueByInfo", map[string]any{
		"data": map[string]any{
			"issueByInfo": nil,
		},
	})

	_ = cache.Set(resolve.RepoCacheKey("ws-123"), []resolve.CachedRepo{
		{ID: "r1", GhID: 12345, Name: "task-tracker", OwnerName: "dlakehammond"},
	})

	return ms
}

func setupIssueCloseServerMixed(t *testing.T) *testutil.MockServer {
	t.Helper()

	ms := testutil.NewMockServer(t)

	ms.HandleQuery("ListRepos", repoResolutionResponse())

	callCount := 0
	ms.Handle(
		func(req testutil.GraphQLRequest) bool {
			return strings.Contains(req.Query, "IssueByInfo") && !strings.Contains(req.Query, "GetIssueForClose")
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

	ms.HandleQuery("GetIssueForClose", issueCloseResolveResponse("i1", 1, "Fix login button alignment", "OPEN"))
	ms.HandleQuery("CloseIssues", closeIssuesResponse(1))

	_ = cache.Set(resolve.RepoCacheKey("ws-123"), []resolve.CachedRepo{
		{ID: "r1", GhID: 12345, Name: "task-tracker", OwnerName: "dlakehammond"},
	})

	return ms
}

func setupIssueReopenServer(t *testing.T) *testutil.MockServer {
	t.Helper()

	ms := testutil.NewMockServer(t)

	ms.HandleQuery("ListPipelines", pipelineResolutionResponse())
	ms.HandleQuery("ListRepos", repoResolutionResponse())
	ms.HandleQuery("IssueByInfo", issueByInfoResolutionResponseClosed())
	ms.HandleQuery("GetIssueForClose", issueCloseResolveResponse("i10", 10, "Old closed issue", "CLOSED"))
	ms.HandleQuery("ReopenIssues", reopenIssuesResponse(1))

	_ = cache.Set(resolve.RepoCacheKey("ws-123"), []resolve.CachedRepo{
		{ID: "r1", GhID: 12345, Name: "task-tracker", OwnerName: "dlakehammond"},
	})

	return ms
}

func setupIssueReopenServerBatch(t *testing.T) *testutil.MockServer {
	t.Helper()

	ms := testutil.NewMockServer(t)

	ms.HandleQuery("ListPipelines", pipelineResolutionResponse())
	ms.HandleQuery("ListRepos", repoResolutionResponse())

	issueCallCount := 0
	ms.Handle(
		func(req testutil.GraphQLRequest) bool {
			return strings.Contains(req.Query, "IssueByInfo") && !strings.Contains(req.Query, "GetIssueForClose")
		},
		func(w http.ResponseWriter, req testutil.GraphQLRequest) {
			issueCallCount++
			var resp map[string]any
			if issueCallCount%2 == 1 {
				resp = issueByInfoResolutionResponseClosed()
			} else {
				resp = issueByInfoResolutionResponseClosed2()
			}
			data, _ := json.Marshal(resp)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(data)
		},
	)

	closeResolveCount := 0
	ms.Handle(
		func(req testutil.GraphQLRequest) bool {
			return strings.Contains(req.Query, "GetIssueForClose")
		},
		func(w http.ResponseWriter, req testutil.GraphQLRequest) {
			closeResolveCount++
			var resp map[string]any
			if closeResolveCount%2 == 1 {
				resp = issueCloseResolveResponse("i10", 10, "Old closed issue", "CLOSED")
			} else {
				resp = issueCloseResolveResponse("i11", 11, "Another closed issue", "CLOSED")
			}
			data, _ := json.Marshal(resp)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(data)
		},
	)

	ms.HandleQuery("ReopenIssues", reopenIssuesResponse(2))

	_ = cache.Set(resolve.RepoCacheKey("ws-123"), []resolve.CachedRepo{
		{ID: "r1", GhID: 12345, Name: "task-tracker", OwnerName: "dlakehammond"},
	})

	return ms
}

func setupIssueReopenServerAlreadyOpen(t *testing.T) *testutil.MockServer {
	t.Helper()

	ms := testutil.NewMockServer(t)

	ms.HandleQuery("ListPipelines", pipelineResolutionResponse())
	ms.HandleQuery("ListRepos", repoResolutionResponse())
	ms.HandleQuery("IssueByInfo", issueByInfoResolutionResponse())
	ms.HandleQuery("GetIssueForClose", issueCloseResolveResponse("i1", 1, "Fix login button alignment", "OPEN"))

	_ = cache.Set(resolve.RepoCacheKey("ws-123"), []resolve.CachedRepo{
		{ID: "r1", GhID: 12345, Name: "task-tracker", OwnerName: "dlakehammond"},
	})

	return ms
}

func setupIssueReopenServerMixed(t *testing.T) *testutil.MockServer {
	t.Helper()

	ms := testutil.NewMockServer(t)

	ms.HandleQuery("ListPipelines", pipelineResolutionResponse())
	ms.HandleQuery("ListRepos", repoResolutionResponse())

	callCount := 0
	ms.Handle(
		func(req testutil.GraphQLRequest) bool {
			return strings.Contains(req.Query, "IssueByInfo") && !strings.Contains(req.Query, "GetIssueForClose")
		},
		func(w http.ResponseWriter, req testutil.GraphQLRequest) {
			callCount++
			var resp map[string]any
			if callCount == 1 {
				resp = issueByInfoResolutionResponseClosed()
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

	ms.HandleQuery("GetIssueForClose", issueCloseResolveResponse("i10", 10, "Old closed issue", "CLOSED"))
	ms.HandleQuery("ReopenIssues", reopenIssuesResponse(1))

	_ = cache.Set(resolve.RepoCacheKey("ws-123"), []resolve.CachedRepo{
		{ID: "r1", GhID: 12345, Name: "task-tracker", OwnerName: "dlakehammond"},
	})

	return ms
}

// --- response helpers ---

func issueByInfoResolutionResponseClosed() map[string]any {
	return map[string]any{
		"data": map[string]any{
			"issueByInfo": map[string]any{
				"id":     "i10",
				"number": 10,
				"repository": map[string]any{
					"ghId":      12345,
					"name":      "task-tracker",
					"ownerName": "dlakehammond",
				},
			},
		},
	}
}

func issueByInfoResolutionResponseClosed2() map[string]any {
	return map[string]any{
		"data": map[string]any{
			"issueByInfo": map[string]any{
				"id":     "i11",
				"number": 11,
				"repository": map[string]any{
					"ghId":      12345,
					"name":      "task-tracker",
					"ownerName": "dlakehammond",
				},
			},
		},
	}
}

func issueCloseResolveResponse(id string, number int, title, state string) map[string]any {
	return map[string]any{
		"data": map[string]any{
			"node": map[string]any{
				"id":     id,
				"number": number,
				"title":  title,
				"state":  state,
				"repository": map[string]any{
					"name":      "task-tracker",
					"ownerName": "dlakehammond",
				},
			},
		},
	}
}

func closeIssuesResponse(successCount int) map[string]any {
	return map[string]any{
		"data": map[string]any{
			"closeIssues": map[string]any{
				"successCount": successCount,
				"failedIssues": []any{},
				"githubErrors": "[]",
			},
		},
	}
}

func reopenIssuesResponse(successCount int) map[string]any {
	return map[string]any{
		"data": map[string]any{
			"reopenIssues": map[string]any{
				"successCount": successCount,
				"failedIssues": []any{},
				"githubErrors": "[]",
			},
		},
	}
}
