package cmd

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/dslh/zh/internal/cache"
	"github.com/dslh/zh/internal/resolve"
	"github.com/dslh/zh/internal/testutil"
)

// --- issue estimate ---

func TestIssueEstimateSet(t *testing.T) {
	resetIssueFlags()
	resetIssueEstimateFlags()

	ms := setupIssueEstimateServer(t, 3)
	setupIssueTestEnv(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"issue", "estimate", "task-tracker#1", "5"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("issue estimate returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Set estimate") {
		t.Errorf("output should confirm estimate set, got: %s", out)
	}
	if !strings.Contains(out, "task-tracker#1") {
		t.Errorf("output should contain issue ref, got: %s", out)
	}
	if !strings.Contains(out, "5") {
		t.Errorf("output should contain estimate value, got: %s", out)
	}
}

func TestIssueEstimateClear(t *testing.T) {
	resetIssueFlags()
	resetIssueEstimateFlags()

	ms := setupIssueEstimateServer(t, 3)
	setupIssueTestEnv(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"issue", "estimate", "task-tracker#1"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("issue estimate clear returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Cleared estimate") {
		t.Errorf("output should confirm estimate cleared, got: %s", out)
	}
	if !strings.Contains(out, "task-tracker#1") {
		t.Errorf("output should contain issue ref, got: %s", out)
	}
}

func TestIssueEstimateInvalidValue(t *testing.T) {
	resetIssueFlags()
	resetIssueEstimateFlags()

	ms := setupIssueEstimateServer(t, 3)
	setupIssueTestEnv(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"issue", "estimate", "task-tracker#1", "7"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("issue estimate should error for invalid value")
	}
	if !strings.Contains(err.Error(), "invalid estimate value") {
		t.Errorf("error should mention invalid value, got: %v", err)
	}
	if !strings.Contains(err.Error(), "1, 2, 3, 5, 8, 13, 21, 40") {
		t.Errorf("error should list valid values, got: %v", err)
	}
}

func TestIssueEstimateNonNumericValue(t *testing.T) {
	resetIssueFlags()
	resetIssueEstimateFlags()

	// No server needed — parse fails before API call
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"issue", "estimate", "task-tracker#1", "abc"})

	// Need a minimal environment to get past requireWorkspace
	ms := setupIssueEstimateServer(t, 0)
	setupIssueTestEnv(t, ms)

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("issue estimate should error for non-numeric value")
	}
	if !strings.Contains(err.Error(), "must be a number") {
		t.Errorf("error should mention must be a number, got: %v", err)
	}
}

func TestIssueEstimateDryRunSet(t *testing.T) {
	resetIssueFlags()
	resetIssueEstimateFlags()

	ms := setupIssueEstimateServer(t, 3)
	setupIssueTestEnv(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"issue", "estimate", "task-tracker#1", "5", "--dry-run"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("issue estimate --dry-run returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Would set estimate") {
		t.Errorf("dry run should say 'Would set estimate', got: %s", out)
	}
	if !strings.Contains(out, "task-tracker#1") {
		t.Errorf("dry run should contain issue ref, got: %s", out)
	}
	if !strings.Contains(out, "currently: 3") {
		t.Errorf("dry run should show current estimate, got: %s", out)
	}
}

func TestIssueEstimateDryRunClear(t *testing.T) {
	resetIssueFlags()
	resetIssueEstimateFlags()

	ms := setupIssueEstimateServerNoEstimate(t)
	setupIssueTestEnv(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"issue", "estimate", "task-tracker#1", "--dry-run"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("issue estimate --dry-run clear returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Would clear estimate") {
		t.Errorf("dry run should say 'Would clear estimate', got: %s", out)
	}
	if !strings.Contains(out, "currently: none") {
		t.Errorf("dry run should show current estimate as none, got: %s", out)
	}
}

func TestIssueEstimateJSON(t *testing.T) {
	resetIssueFlags()
	resetIssueEstimateFlags()

	ms := setupIssueEstimateServer(t, 3)
	setupIssueTestEnv(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"issue", "estimate", "task-tracker#1", "5", "--output=json"})
	outputFormat = "json"
	defer func() { outputFormat = "" }()

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("issue estimate --output=json returned error: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\nOutput: %s", err, buf.String())
	}

	issue, ok := result["issue"].(map[string]any)
	if !ok {
		t.Fatalf("JSON should contain issue object, got: %v", result)
	}
	if issue["number"] != float64(1) {
		t.Errorf("JSON should contain number, got: %v", issue["number"])
	}

	estimate, ok := issue["estimate"].(map[string]any)
	if !ok {
		t.Fatalf("JSON should contain estimate object, got: %v", issue["estimate"])
	}
	if estimate["previous"] != float64(3) {
		t.Errorf("JSON estimate.previous should be 3, got: %v", estimate["previous"])
	}
	if estimate["current"] != float64(5) {
		t.Errorf("JSON estimate.current should be 5, got: %v", estimate["current"])
	}
}

func TestIssueEstimateNotFound(t *testing.T) {
	resetIssueFlags()
	resetIssueEstimateFlags()

	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListRepos", repoResolutionResponse())
	ms.HandleQuery("IssueByInfo", map[string]any{
		"data": map[string]any{
			"issueByInfo": nil,
		},
	})

	setupIssueTestEnv(t, ms)

	_ = cache.Set(resolve.RepoCacheKey("ws-123"), []resolve.CachedRepo{
		{ID: "r1", GhID: 12345, Name: "task-tracker", OwnerName: "dlakehammond"},
	})

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"issue", "estimate", "task-tracker#999", "5"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("issue estimate should error for nonexistent issue")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should mention not found, got: %v", err)
	}
}

func TestIssueEstimateHelp(t *testing.T) {
	resetIssueFlags()
	resetIssueEstimateFlags()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"issue", "estimate", "--help"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("issue estimate --help returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "estimate") {
		t.Error("help should mention estimate")
	}
	if !strings.Contains(out, "--dry-run") {
		t.Error("help should mention --dry-run flag")
	}
	if !strings.Contains(out, "--repo") {
		t.Error("help should mention --repo flag")
	}
}

// --- test helpers ---

func setupIssueEstimateServer(t *testing.T, currentEstimate float64) *testutil.MockServer {
	t.Helper()

	ms := testutil.NewMockServer(t)

	ms.HandleQuery("ListRepos", repoResolutionResponse())
	ms.HandleQuery("IssueByInfo", issueByInfoResolutionResponse())
	ms.HandleQuery("GetIssueForEstimate", issueEstimateResponse(currentEstimate))
	ms.HandleQuery("SetEstimate", setEstimateSuccessResponse())

	_ = cache.Set(resolve.RepoCacheKey("ws-123"), []resolve.CachedRepo{
		{ID: "r1", GhID: 12345, Name: "task-tracker", OwnerName: "dlakehammond"},
	})

	return ms
}

func setupIssueEstimateServerNoEstimate(t *testing.T) *testutil.MockServer {
	t.Helper()

	ms := testutil.NewMockServer(t)

	ms.HandleQuery("ListRepos", repoResolutionResponse())
	ms.HandleQuery("IssueByInfo", issueByInfoResolutionResponse())
	ms.HandleQuery("GetIssueForEstimate", issueEstimateResponseNoEstimate())
	ms.HandleQuery("SetEstimate", setEstimateSuccessResponse())

	_ = cache.Set(resolve.RepoCacheKey("ws-123"), []resolve.CachedRepo{
		{ID: "r1", GhID: 12345, Name: "task-tracker", OwnerName: "dlakehammond"},
	})

	return ms
}

func issueEstimateResponse(currentEstimate float64) map[string]any {
	return map[string]any{
		"data": map[string]any{
			"issueByInfo": map[string]any{
				"id":     "i1",
				"number": 1,
				"title":  "Fix login button alignment",
				"estimate": map[string]any{
					"value": currentEstimate,
				},
				"repository": map[string]any{
					"name":      "task-tracker",
					"ownerName": "dlakehammond",
					"estimateSet": map[string]any{
						"values": []any{1, 2, 3, 5, 8, 13, 21, 40},
					},
				},
			},
		},
	}
}

func issueEstimateResponseNoEstimate() map[string]any {
	return map[string]any{
		"data": map[string]any{
			"issueByInfo": map[string]any{
				"id":       "i1",
				"number":   1,
				"title":    "Fix login button alignment",
				"estimate": nil,
				"repository": map[string]any{
					"name":      "task-tracker",
					"ownerName": "dlakehammond",
					"estimateSet": map[string]any{
						"values": []any{1, 2, 3, 5, 8, 13, 21, 40},
					},
				},
			},
		},
	}
}

func setEstimateSuccessResponse() map[string]any {
	return map[string]any{
		"data": map[string]any{
			"setEstimate": map[string]any{
				"issue": map[string]any{
					"id":     "i1",
					"number": 1,
					"title":  "Fix login button alignment",
					"estimate": map[string]any{
						"value": 5,
					},
					"repository": map[string]any{
						"name":      "task-tracker",
						"ownerName": "dlakehammond",
					},
				},
			},
		},
	}
}
