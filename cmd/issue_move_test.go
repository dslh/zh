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

// --- issue move ---

func TestIssueMoveSingle(t *testing.T) {
	resetIssueFlags()
	resetIssueMoveFlags()

	ms := setupIssueMoveServer(t)
	setupIssueTestEnv(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"issue", "move", "task-tracker#1", "Done"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("issue move returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Moved") {
		t.Errorf("output should confirm move, got: %s", out)
	}
	if !strings.Contains(out, "task-tracker#1") {
		t.Errorf("output should contain issue ref, got: %s", out)
	}
	if !strings.Contains(out, "Done") {
		t.Errorf("output should contain pipeline name, got: %s", out)
	}
}

func TestIssueMoveBatch(t *testing.T) {
	resetIssueFlags()
	resetIssueMoveFlags()

	ms := setupIssueMoveServerBatch(t)
	setupIssueTestEnv(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"issue", "move", "task-tracker#1", "task-tracker#2", "Done"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("issue move batch returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Moved 2 issue(s)") {
		t.Errorf("output should show batch count, got: %s", out)
	}
	if !strings.Contains(out, "task-tracker#1") {
		t.Errorf("output should contain first issue ref, got: %s", out)
	}
	if !strings.Contains(out, "task-tracker#2") {
		t.Errorf("output should contain second issue ref, got: %s", out)
	}
}

func TestIssueMovePositionTop(t *testing.T) {
	resetIssueFlags()
	resetIssueMoveFlags()

	ms := setupIssueMoveServer(t)
	setupIssueTestEnv(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"issue", "move", "task-tracker#1", "Done", "--position=top"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("issue move --position=top returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Moved") {
		t.Errorf("output should confirm move, got: %s", out)
	}
}

func TestIssueMovePositionBottom(t *testing.T) {
	resetIssueFlags()
	resetIssueMoveFlags()

	ms := setupIssueMoveServer(t)
	setupIssueTestEnv(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"issue", "move", "task-tracker#1", "Done", "--position=bottom"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("issue move --position=bottom returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Moved") {
		t.Errorf("output should confirm move, got: %s", out)
	}
}

func TestIssueMovePositionNumeric(t *testing.T) {
	resetIssueFlags()
	resetIssueMoveFlags()

	ms := setupIssueMoveServer(t)
	setupIssueTestEnv(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"issue", "move", "task-tracker#1", "Done", "--position=3"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("issue move --position=3 returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Moved") {
		t.Errorf("output should confirm move, got: %s", out)
	}
}

func TestIssueMoveNumericPositionBatchError(t *testing.T) {
	resetIssueFlags()
	resetIssueMoveFlags()

	ms := setupIssueMoveServerBatch(t)
	setupIssueTestEnv(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"issue", "move", "task-tracker#1", "task-tracker#2", "Done", "--position=3"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("numeric position with batch should return error")
	}
	if !strings.Contains(err.Error(), "numeric") {
		t.Errorf("error should mention numeric position limitation, got: %v", err)
	}
}

func TestIssueMoveDryRun(t *testing.T) {
	resetIssueFlags()
	resetIssueMoveFlags()

	ms := setupIssueMoveServer(t)
	setupIssueTestEnv(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"issue", "move", "task-tracker#1", "Done", "--dry-run"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("issue move --dry-run returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Would move") {
		t.Errorf("dry run should say 'Would move', got: %s", out)
	}
	if !strings.Contains(out, "task-tracker#1") {
		t.Errorf("dry run should contain issue ref, got: %s", out)
	}
	if !strings.Contains(out, "Done") {
		t.Errorf("dry run should contain target pipeline, got: %s", out)
	}
}

func TestIssueMoveDryRunWithPosition(t *testing.T) {
	resetIssueFlags()
	resetIssueMoveFlags()

	ms := setupIssueMoveServer(t)
	setupIssueTestEnv(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"issue", "move", "task-tracker#1", "Done", "--dry-run", "--position=top"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("issue move --dry-run --position=top returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "at top") {
		t.Errorf("dry run should mention position, got: %s", out)
	}
}

func TestIssueMoveDryRunShowsCurrentPipeline(t *testing.T) {
	resetIssueFlags()
	resetIssueMoveFlags()

	ms := setupIssueMoveServer(t)
	setupIssueTestEnv(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"issue", "move", "task-tracker#1", "Done", "--dry-run"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("issue move --dry-run returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "In Development") {
		t.Errorf("dry run should show current pipeline, got: %s", out)
	}
}

func TestIssueMoveStopOnError(t *testing.T) {
	resetIssueFlags()
	resetIssueMoveFlags()

	ms := setupIssueMoveServerWithNotFound(t)
	setupIssueTestEnv(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"issue", "move", "task-tracker#999", "Done"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("issue move should error for nonexistent issue")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should mention not found, got: %v", err)
	}
}

func TestIssueMoveContinueOnError(t *testing.T) {
	resetIssueFlags()
	resetIssueMoveFlags()

	ms := setupIssueMoveServerMixed(t)
	setupIssueTestEnv(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"issue", "move", "task-tracker#1", "task-tracker#999", "Done", "--continue-on-error"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("continue-on-error should not return error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "task-tracker#1") {
		t.Errorf("output should contain successful move, got: %s", out)
	}
	if !strings.Contains(out, "Failed") {
		t.Errorf("output should show failures, got: %s", out)
	}
}

func TestIssueMoveJSON(t *testing.T) {
	resetIssueFlags()
	resetIssueMoveFlags()

	ms := setupIssueMoveServer(t)
	setupIssueTestEnv(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"issue", "move", "task-tracker#1", "Done", "--output=json"})
	outputFormat = "json"
	defer func() { outputFormat = "" }()

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("issue move --output=json returned error: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\nOutput: %s", err, buf.String())
	}
	if result["pipeline"] != "Done" {
		t.Errorf("JSON should contain pipeline name, got: %v", result["pipeline"])
	}
}

func TestIssueMovePositionInvalid(t *testing.T) {
	resetIssueFlags()
	resetIssueMoveFlags()

	ms := setupIssueMoveServer(t)
	setupIssueTestEnv(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"issue", "move", "task-tracker#1", "Done", "--position=invalid"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("invalid position should return error")
	}
	if !strings.Contains(err.Error(), "invalid position") {
		t.Errorf("error should mention invalid position, got: %v", err)
	}
}

// TestIssueMoveHelp should be the last test since --help sets persistent
// Cobra state that's hard to reset in a shared command tree.
func TestIssueMoveHelp(t *testing.T) {
	resetIssueFlags()
	resetIssueMoveFlags()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"issue", "move", "--help"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("issue move --help returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Move") {
		t.Errorf("help should contain Move, got: %s", out)
	}
	if !strings.Contains(out, "--position") {
		t.Error("help should mention --position flag")
	}
	if !strings.Contains(out, "--dry-run") {
		t.Error("help should mention --dry-run flag")
	}
	if !strings.Contains(out, "--continue-on-error") {
		t.Error("help should mention --continue-on-error flag")
	}
}

// --- test helpers ---

func setupIssueMoveServer(t *testing.T) *testutil.MockServer {
	t.Helper()

	ms := testutil.NewMockServer(t)

	ms.HandleQuery("ListPipelines", pipelineResolutionResponse())
	ms.HandleQuery("IssueByInfo", issueByInfoResolutionResponse())
	ms.HandleQuery("GetPipelineIssueId", pipelineIssueIDResponse("i1", 1, "Fix login button alignment", "pi1", "p2", "In Development"))
	ms.HandleQuery("MoveIssue", moveIssueResponse())
	ms.HandleQuery("MoveIssueRelativeTo", moveIssueRelativeResponse())
	ms.HandleQuery("ListRepos", repoResolutionResponse())

	_ = cache.Set(resolve.RepoCacheKey("ws-123"), []resolve.CachedRepo{
		{ID: "r1", GhID: 12345, Name: "task-tracker", OwnerName: "dlakehammond"},
	})

	return ms
}

func setupIssueMoveServerBatch(t *testing.T) *testutil.MockServer {
	t.Helper()

	ms := testutil.NewMockServer(t)

	ms.HandleQuery("ListPipelines", pipelineResolutionResponse())
	ms.HandleQuery("ListRepos", repoResolutionResponse())

	// Return different issues based on call count
	issueCallCount := 0
	ms.Handle(
		func(req testutil.GraphQLRequest) bool {
			return strings.Contains(req.Query, "IssueByInfo")
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

	// Return different pipeline issue data based on call count
	piCallCount := 0
	ms.Handle(
		func(req testutil.GraphQLRequest) bool {
			return strings.Contains(req.Query, "GetPipelineIssueId")
		},
		func(w http.ResponseWriter, req testutil.GraphQLRequest) {
			piCallCount++
			var resp map[string]any
			if piCallCount%2 == 1 {
				resp = pipelineIssueIDResponse("i1", 1, "Fix login button alignment", "pi1", "p2", "In Development")
			} else {
				resp = pipelineIssueIDResponse("i2", 2, "Add error handling", "pi2", "p2", "In Development")
			}
			data, _ := json.Marshal(resp)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(data)
		},
	)

	ms.HandleQuery("MoveIssue", moveIssueResponse())
	ms.HandleQuery("MoveIssueRelativeTo", moveIssueRelativeResponse())

	_ = cache.Set(resolve.RepoCacheKey("ws-123"), []resolve.CachedRepo{
		{ID: "r1", GhID: 12345, Name: "task-tracker", OwnerName: "dlakehammond"},
	})

	return ms
}

func setupIssueMoveServerWithNotFound(t *testing.T) *testutil.MockServer {
	t.Helper()

	ms := testutil.NewMockServer(t)

	ms.HandleQuery("ListPipelines", pipelineResolutionResponse())
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

func setupIssueMoveServerMixed(t *testing.T) *testutil.MockServer {
	t.Helper()

	ms := testutil.NewMockServer(t)

	ms.HandleQuery("ListPipelines", pipelineResolutionResponse())
	ms.HandleQuery("ListRepos", repoResolutionResponse())

	// First call succeeds, second returns not found
	callCount := 0
	ms.Handle(
		func(req testutil.GraphQLRequest) bool {
			return strings.Contains(req.Query, "IssueByInfo")
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

	ms.HandleQuery("GetPipelineIssueId", pipelineIssueIDResponse("i1", 1, "Fix login button alignment", "pi1", "p2", "In Development"))
	ms.HandleQuery("MoveIssue", moveIssueResponse())
	ms.HandleQuery("MoveIssueRelativeTo", moveIssueRelativeResponse())

	_ = cache.Set(resolve.RepoCacheKey("ws-123"), []resolve.CachedRepo{
		{ID: "r1", GhID: 12345, Name: "task-tracker", OwnerName: "dlakehammond"},
	})

	return ms
}

func issueByInfoResolutionResponse2() map[string]any {
	return map[string]any{
		"data": map[string]any{
			"issueByInfo": map[string]any{
				"id":     "i2",
				"number": 2,
				"repository": map[string]any{
					"ghId":      12345,
					"name":      "task-tracker",
					"ownerName": "dlakehammond",
				},
			},
		},
	}
}

func pipelineIssueIDResponse(issueID string, number int, title, piID, pipelineID, pipelineName string) map[string]any {
	return map[string]any{
		"data": map[string]any{
			"node": map[string]any{
				"id":     issueID,
				"number": number,
				"title":  title,
				"repository": map[string]any{
					"name":      "task-tracker",
					"ownerName": "dlakehammond",
				},
				"pipelineIssue": map[string]any{
					"id": piID,
					"pipeline": map[string]any{
						"id":   pipelineID,
						"name": pipelineName,
					},
				},
			},
		},
	}
}

func moveIssueResponse() map[string]any {
	return map[string]any{
		"data": map[string]any{
			"moveIssue": map[string]any{
				"issue": map[string]any{
					"id":     "i1",
					"number": 1,
					"title":  "Fix login button alignment",
					"repository": map[string]any{
						"name":      "task-tracker",
						"ownerName": "dlakehammond",
					},
				},
				"pipeline": map[string]any{
					"id":   "p3",
					"name": "Done",
				},
			},
		},
	}
}

func moveIssueRelativeResponse() map[string]any {
	return map[string]any{
		"data": map[string]any{
			"moveIssueRelativeTo": map[string]any{
				"issue": map[string]any{
					"id":     "i1",
					"number": 1,
					"title":  "Fix login button alignment",
					"repository": map[string]any{
						"name":      "task-tracker",
						"ownerName": "dlakehammond",
					},
				},
				"pipeline": map[string]any{
					"id":   "p3",
					"name": "Done",
				},
			},
		},
	}
}
