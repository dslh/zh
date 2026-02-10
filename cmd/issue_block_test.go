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

// --- issue block ---

func TestIssueBlockIssueToIssue(t *testing.T) {
	resetIssueFlags()
	resetIssueBlockFlags()

	ms := setupIssueBlockServer(t)
	setupIssueTestEnv(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"issue", "block", "task-tracker#1", "task-tracker#2"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("issue block returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Marked") {
		t.Errorf("output should confirm block, got: %s", out)
	}
	if !strings.Contains(out, "blocking") {
		t.Errorf("output should mention blocking, got: %s", out)
	}
	if !strings.Contains(out, "cannot be removed") {
		t.Errorf("output should contain API limitation note, got: %s", out)
	}
}

func TestIssueBlockDryRun(t *testing.T) {
	resetIssueFlags()
	resetIssueBlockFlags()

	ms := setupIssueBlockServer(t)
	setupIssueTestEnv(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"issue", "block", "task-tracker#1", "task-tracker#2", "--dry-run"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("issue block --dry-run returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Would mark") {
		t.Errorf("dry run should say 'Would mark', got: %s", out)
	}
	if !strings.Contains(out, "blocking") {
		t.Errorf("dry run should mention blocking, got: %s", out)
	}
}

func TestIssueBlockJSON(t *testing.T) {
	resetIssueFlags()
	resetIssueBlockFlags()

	ms := setupIssueBlockServer(t)
	setupIssueTestEnv(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"issue", "block", "task-tracker#1", "task-tracker#2", "--output=json"})
	outputFormat = "json"
	defer func() { outputFormat = "" }()

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("issue block --output=json returned error: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\nOutput: %s", err, buf.String())
	}

	blocking, ok := result["blocking"].(map[string]any)
	if !ok {
		t.Fatal("JSON should contain blocking object")
	}
	if blocking["type"] != "ISSUE" {
		t.Errorf("blocking type should be ISSUE, got: %v", blocking["type"])
	}
}

func TestIssueBlockInvalidType(t *testing.T) {
	resetIssueFlags()
	resetIssueBlockFlags()

	ms := setupIssueBlockServer(t)
	setupIssueTestEnv(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"issue", "block", "task-tracker#1", "task-tracker#2", "--blocker-type=invalid"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("invalid type should return error")
	}
	if !strings.Contains(err.Error(), "invalid --blocker-type") {
		t.Errorf("error should mention invalid type, got: %v", err)
	}
}

func TestIssueBlockHelp(t *testing.T) {
	resetIssueFlags()
	resetIssueBlockFlags()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"issue", "block", "--help"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("issue block --help returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "blocking") {
		t.Errorf("help should mention blocking, got: %s", out)
	}
	if !strings.Contains(out, "--dry-run") {
		t.Error("help should mention --dry-run flag")
	}
	if !strings.Contains(out, "--blocker-type") {
		t.Error("help should mention --blocker-type flag")
	}
	if !strings.Contains(out, "--blocked-type") {
		t.Error("help should mention --blocked-type flag")
	}
	if !strings.Contains(out, "cannot be removed") {
		t.Error("help should mention API limitation")
	}
}

// --- issue blockers ---

func TestIssueBlockersWithBlockers(t *testing.T) {
	resetIssueFlags()
	resetIssueBlockersFlags()

	ms := setupIssueBlockersServer(t, true)
	setupIssueTestEnv(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"issue", "blockers", "task-tracker#1"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("issue blockers returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "is blocked by") {
		t.Errorf("output should say 'is blocked by', got: %s", out)
	}
	if !strings.Contains(out, "task-tracker#5") {
		t.Errorf("output should contain blocking issue ref, got: %s", out)
	}
	if !strings.Contains(out, "Database migration") {
		t.Errorf("output should contain blocking issue title, got: %s", out)
	}
}

func TestIssueBlockersNoBlockers(t *testing.T) {
	resetIssueFlags()
	resetIssueBlockersFlags()

	ms := setupIssueBlockersServer(t, false)
	setupIssueTestEnv(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"issue", "blockers", "task-tracker#1"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("issue blockers returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "no blockers") {
		t.Errorf("output should say no blockers, got: %s", out)
	}
}

func TestIssueBlockersJSON(t *testing.T) {
	resetIssueFlags()
	resetIssueBlockersFlags()

	ms := setupIssueBlockersServer(t, true)
	setupIssueTestEnv(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"issue", "blockers", "task-tracker#1", "--output=json"})
	outputFormat = "json"
	defer func() { outputFormat = "" }()

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("issue blockers --output=json returned error: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\nOutput: %s", err, buf.String())
	}

	blockers, ok := result["blockers"].([]any)
	if !ok {
		t.Fatal("JSON should contain blockers array")
	}
	if len(blockers) != 1 {
		t.Errorf("expected 1 blocker, got %d", len(blockers))
	}
}

// --- issue blocking ---

func TestIssueBlockingWithItems(t *testing.T) {
	resetIssueFlags()
	resetIssueBlockingFlags()

	ms := setupIssueBlockingServer(t, true)
	setupIssueTestEnv(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"issue", "blocking", "task-tracker#1"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("issue blocking returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "is blocking") {
		t.Errorf("output should say 'is blocking', got: %s", out)
	}
	if !strings.Contains(out, "task-tracker#3") {
		t.Errorf("output should contain blocked issue ref, got: %s", out)
	}
}

func TestIssueBlockingNothing(t *testing.T) {
	resetIssueFlags()
	resetIssueBlockingFlags()

	ms := setupIssueBlockingServer(t, false)
	setupIssueTestEnv(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"issue", "blocking", "task-tracker#1"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("issue blocking returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "not blocking anything") {
		t.Errorf("output should say not blocking anything, got: %s", out)
	}
}

func TestIssueBlockingJSON(t *testing.T) {
	resetIssueFlags()
	resetIssueBlockingFlags()

	ms := setupIssueBlockingServer(t, true)
	setupIssueTestEnv(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"issue", "blocking", "task-tracker#1", "--output=json"})
	outputFormat = "json"
	defer func() { outputFormat = "" }()

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("issue blocking --output=json returned error: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\nOutput: %s", err, buf.String())
	}

	blocking, ok := result["blocking"].([]any)
	if !ok {
		t.Fatal("JSON should contain blocking array")
	}
	if len(blocking) != 1 {
		t.Errorf("expected 1 blocked item, got %d", len(blocking))
	}
}

// --- test helpers ---

func setupIssueBlockServer(t *testing.T) *testutil.MockServer {
	t.Helper()

	ms := testutil.NewMockServer(t)

	ms.HandleQuery("ListRepos", repoResolutionResponse())
	ms.HandleQuery("IssueByInfo", issueByInfoResolutionResponse())
	ms.HandleQuery("CreateBlockage", map[string]any{
		"data": map[string]any{
			"createBlockage": map[string]any{
				"blockage": map[string]any{
					"id":        "b1",
					"createdAt": "2026-01-20T10:00:00Z",
					"blocking": map[string]any{
						"__typename": "Issue",
						"id":         "i1",
						"number":     1,
						"title":      "Fix login button alignment",
						"repository": map[string]any{
							"name":      "task-tracker",
							"ownerName": "dlakehammond",
						},
					},
					"blocked": map[string]any{
						"__typename": "Issue",
						"id":         "i2",
						"number":     2,
						"title":      "Add error handling",
						"repository": map[string]any{
							"name":      "task-tracker",
							"ownerName": "dlakehammond",
						},
					},
				},
			},
		},
	})

	_ = cache.Set(resolve.RepoCacheKey("ws-123"), []resolve.CachedRepo{
		{ID: "r1", GhID: 12345, Name: "task-tracker", OwnerName: "dlakehammond"},
	})

	return ms
}

func setupIssueBlockersServer(t *testing.T, hasBlockers bool) *testutil.MockServer {
	t.Helper()

	ms := testutil.NewMockServer(t)

	ms.HandleQuery("ListRepos", repoResolutionResponse())
	ms.HandleQuery("IssueByInfo", issueByInfoResolutionResponse())

	var blockerNodes []any
	if hasBlockers {
		blockerNodes = []any{
			map[string]any{
				"__typename": "Issue",
				"id":         "i5",
				"number":     5,
				"title":      "Database migration prerequisite",
				"state":      "OPEN",
				"repository": map[string]any{
					"name":      "task-tracker",
					"ownerName": "dlakehammond",
				},
			},
		}
	}

	ms.HandleQuery("GetIssueBlockers", map[string]any{
		"data": map[string]any{
			"issueByInfo": map[string]any{
				"id":     "i1",
				"number": 1,
				"title":  "Fix login button alignment",
				"repository": map[string]any{
					"name":      "task-tracker",
					"ownerName": "dlakehammond",
				},
				"blockingItems": map[string]any{
					"nodes": blockerNodes,
				},
			},
		},
	})

	_ = cache.Set(resolve.RepoCacheKey("ws-123"), []resolve.CachedRepo{
		{ID: "r1", GhID: 12345, Name: "task-tracker", OwnerName: "dlakehammond"},
	})

	return ms
}

func setupIssueBlockingServer(t *testing.T, hasBlocking bool) *testutil.MockServer {
	t.Helper()

	ms := testutil.NewMockServer(t)

	ms.HandleQuery("ListRepos", repoResolutionResponse())
	ms.HandleQuery("IssueByInfo", issueByInfoResolutionResponse())

	var blockedNodes []any
	if hasBlocking {
		blockedNodes = []any{
			map[string]any{
				"__typename": "Issue",
				"id":         "i3",
				"number":     3,
				"title":      "Implement search feature",
				"state":      "OPEN",
				"repository": map[string]any{
					"name":      "task-tracker",
					"ownerName": "dlakehammond",
				},
			},
		}
	}

	ms.HandleQuery("GetIssueBlocking", map[string]any{
		"data": map[string]any{
			"issueByInfo": map[string]any{
				"id":     "i1",
				"number": 1,
				"title":  "Fix login button alignment",
				"repository": map[string]any{
					"name":      "task-tracker",
					"ownerName": "dlakehammond",
				},
				"blockedItems": map[string]any{
					"nodes": blockedNodes,
				},
			},
		},
	})

	_ = cache.Set(resolve.RepoCacheKey("ws-123"), []resolve.CachedRepo{
		{ID: "r1", GhID: 12345, Name: "task-tracker", OwnerName: "dlakehammond"},
	})

	return ms
}
