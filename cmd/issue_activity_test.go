package cmd

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/dslh/zh/internal/api"
	"github.com/dslh/zh/internal/cache"
	"github.com/dslh/zh/internal/gh"
	"github.com/dslh/zh/internal/resolve"
	"github.com/dslh/zh/internal/testutil"
)

// --- issue activity (ZenHub only) ---

func TestIssueActivityWithEvents(t *testing.T) {
	resetIssueFlags()
	resetIssueActivityFlags()

	ms := setupIssueActivityServer(t, true)
	setupIssueTestEnv(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"issue", "activity", "task-tracker#1"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("issue activity returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "ACTIVITY") {
		t.Errorf("output should contain ACTIVITY header, got: %s", out)
	}
	if !strings.Contains(out, "task-tracker#1") {
		t.Errorf("output should contain issue ref, got: %s", out)
	}
	if !strings.Contains(out, "set estimate to 5.0") {
		t.Errorf("output should contain estimate event, got: %s", out)
	}
	if !strings.Contains(out, "set priority") {
		t.Errorf("output should contain priority event, got: %s", out)
	}
	if !strings.Contains(out, "connected PR") {
		t.Errorf("output should contain PR connection event, got: %s", out)
	}
	if !strings.Contains(out, "Total: 3 event(s)") {
		t.Errorf("output should contain event count, got: %s", out)
	}
	// Should NOT show source tags without --github
	if strings.Contains(out, "[ZenHub]") {
		t.Errorf("output should not show source tags without --github, got: %s", out)
	}
}

func TestIssueActivityNoEvents(t *testing.T) {
	resetIssueFlags()
	resetIssueActivityFlags()

	ms := setupIssueActivityServer(t, false)
	setupIssueTestEnv(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"issue", "activity", "task-tracker#1"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("issue activity returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "No activity found") {
		t.Errorf("output should say no activity found, got: %s", out)
	}
}

func TestIssueActivityJSON(t *testing.T) {
	resetIssueFlags()
	resetIssueActivityFlags()

	ms := setupIssueActivityServer(t, true)
	setupIssueTestEnv(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"issue", "activity", "task-tracker#1", "--output=json"})
	outputFormat = "json"
	defer func() { outputFormat = "" }()

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("issue activity --output=json returned error: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\nOutput: %s", err, buf.String())
	}

	events, ok := result["events"].([]any)
	if !ok {
		t.Fatal("JSON should contain events array")
	}
	if len(events) != 3 {
		t.Errorf("expected 3 events, got %d", len(events))
	}

	issue, ok := result["issue"].(map[string]any)
	if !ok {
		t.Fatal("JSON should contain issue object")
	}
	if issue["ref"] != "task-tracker#1" {
		t.Errorf("issue ref should be task-tracker#1, got: %v", issue["ref"])
	}
}

// --- issue activity with --github ---

func TestIssueActivityWithGitHub(t *testing.T) {
	resetIssueFlags()
	resetIssueActivityFlags()

	ms := setupIssueActivityServer(t, true)
	ghMs := setupGitHubTimelineServer(t)
	setupIssueActivityTestEnvWithGitHub(t, ms, ghMs)

	issueActivityGitHub = true

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"issue", "activity", "task-tracker#1", "--github"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("issue activity --github returned error: %v", err)
	}

	out := buf.String()
	// Should contain both ZenHub and GitHub events
	if !strings.Contains(out, "set estimate") {
		t.Errorf("output should contain ZenHub estimate event, got: %s", out)
	}
	if !strings.Contains(out, "added label") {
		t.Errorf("output should contain GitHub label event, got: %s", out)
	}
	// Should show source tags when --github is used
	if !strings.Contains(out, "[ZenHub]") {
		t.Errorf("output should show [ZenHub] source tags, got: %s", out)
	}
	if !strings.Contains(out, "[GitHub]") {
		t.Errorf("output should show [GitHub] source tags, got: %s", out)
	}
}

func TestIssueActivityGitHubNoAccess(t *testing.T) {
	resetIssueFlags()
	resetIssueActivityFlags()

	ms := setupIssueActivityServer(t, true)
	setupIssueTestEnv(t, ms)

	issueActivityGitHub = true

	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(errBuf)
	rootCmd.SetArgs([]string{"issue", "activity", "task-tracker#1", "--github"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("issue activity --github without access returned error: %v", err)
	}

	// Should still show ZenHub events
	out := buf.String()
	if !strings.Contains(out, "set estimate") {
		t.Errorf("output should still contain ZenHub events, got: %s", out)
	}

	// Should warn about GitHub not configured
	errOut := errBuf.String()
	if !strings.Contains(errOut, "GitHub access not configured") {
		t.Errorf("stderr should warn about GitHub access, got: %s", errOut)
	}
}

func TestIssueActivityHelp(t *testing.T) {
	resetIssueFlags()
	resetIssueActivityFlags()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"issue", "activity", "--help"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("issue activity --help returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "activity") {
		t.Errorf("help should mention activity, got: %s", out)
	}
	if !strings.Contains(out, "--github") {
		t.Error("help should mention --github flag")
	}
	if !strings.Contains(out, "--repo") {
		t.Error("help should mention --repo flag")
	}
}

// --- event description parsing ---

func TestDescribeZenHubEventSetEstimate(t *testing.T) {
	data := map[string]any{"current_value": "5.0"}
	desc := describeZenHubEvent("issue.set_estimate", data)
	if desc != "set estimate to 5.0" {
		t.Errorf("expected 'set estimate to 5.0', got: %s", desc)
	}
}

func TestDescribeZenHubEventClearEstimate(t *testing.T) {
	data := map[string]any{"previous_value": "3.0"}
	desc := describeZenHubEvent("issue.set_estimate", data)
	if desc != "cleared estimate (was 3.0)" {
		t.Errorf("expected 'cleared estimate (was 3.0)', got: %s", desc)
	}
}

func TestDescribeZenHubEventSetPriority(t *testing.T) {
	data := map[string]any{
		"priority": map[string]any{"name": "High priority"},
	}
	desc := describeZenHubEvent("issue.set_priority", data)
	if !strings.Contains(desc, "High priority") {
		t.Errorf("expected priority name in description, got: %s", desc)
	}
}

func TestDescribeZenHubEventRemovePriority(t *testing.T) {
	data := map[string]any{
		"previous_priority": map[string]any{"name": "High priority"},
	}
	desc := describeZenHubEvent("issue.remove_priority", data)
	if !strings.Contains(desc, "cleared priority") {
		t.Errorf("expected 'cleared priority' in description, got: %s", desc)
	}
}

func TestDescribeZenHubEventConnectPR(t *testing.T) {
	data := map[string]any{
		"pull_request":            map[string]any{"number": float64(5), "title": "Fix bug"},
		"pull_request_repository": map[string]any{"name": "task-tracker"},
	}
	desc := describeZenHubEvent("issue.connect_issue_to_pr", data)
	if !strings.Contains(desc, "task-tracker#5") {
		t.Errorf("expected PR ref in description, got: %s", desc)
	}
}

func TestDescribeZenHubEventTransferPipeline(t *testing.T) {
	data := map[string]any{
		"from_pipeline": map[string]any{"name": "Backlog"},
		"to_pipeline":   map[string]any{"name": "In Progress"},
	}
	desc := describeZenHubEvent("issue.transfer_pipeline", data)
	if !strings.Contains(desc, "Backlog") || !strings.Contains(desc, "In Progress") {
		t.Errorf("expected pipeline names in description, got: %s", desc)
	}
}

func TestDescribeZenHubEventChangePipeline(t *testing.T) {
	data := map[string]any{
		"from_pipeline": map[string]any{"name": "Backlog"},
		"to_pipeline":   map[string]any{"name": "In Progress"},
	}
	desc := describeZenHubEvent("issue.change_pipeline", data)
	if !strings.Contains(desc, "Backlog") || !strings.Contains(desc, "In Progress") {
		t.Errorf("expected pipeline names in description, got: %s", desc)
	}
}

func TestDescribeZenHubEventAddBlockingIssue(t *testing.T) {
	data := map[string]any{
		"blocking_issue":            map[string]any{"number": float64(2), "title": "Task list crashes"},
		"blocking_issue_repository": map[string]any{"name": "task-tracker"},
	}
	desc := describeZenHubEvent("issue.add_blocking_issue", data)
	if !strings.Contains(desc, "task-tracker#2") {
		t.Errorf("expected blocking issue ref in description, got: %s", desc)
	}
	if !strings.Contains(desc, "Task list crashes") {
		t.Errorf("expected blocking issue title in description, got: %s", desc)
	}
}

func TestDescribeZenHubEventRemoveBlockingIssue(t *testing.T) {
	data := map[string]any{
		"blocking_issue":            map[string]any{"number": float64(2)},
		"blocking_issue_repository": map[string]any{"name": "task-tracker"},
	}
	desc := describeZenHubEvent("issue.remove_blocking_issue", data)
	if !strings.Contains(desc, "task-tracker#2") {
		t.Errorf("expected blocking issue ref in description, got: %s", desc)
	}
}

func TestDescribeZenHubEventConnectPRToIssue(t *testing.T) {
	data := map[string]any{
		"issue":            map[string]any{"number": float64(3), "title": "Add color output"},
		"issue_repository": map[string]any{"name": "task-tracker"},
	}
	desc := describeZenHubEvent("issue.connect_pr_to_issue", data)
	if !strings.Contains(desc, "task-tracker#3") {
		t.Errorf("expected issue ref in description, got: %s", desc)
	}
	if !strings.Contains(desc, "connected to issue") {
		t.Errorf("expected 'connected to issue' in description, got: %s", desc)
	}
}

func TestDescribeZenHubEventDisconnectPRFromIssue(t *testing.T) {
	data := map[string]any{
		"issue":            map[string]any{"number": float64(3)},
		"issue_repository": map[string]any{"name": "task-tracker"},
	}
	desc := describeZenHubEvent("issue.disconnect_pr_from_issue", data)
	if !strings.Contains(desc, "task-tracker#3") {
		t.Errorf("expected issue ref in description, got: %s", desc)
	}
}

func TestDescribeZenHubEventUnknown(t *testing.T) {
	desc := describeZenHubEvent("issue.some_unknown_event", nil)
	if desc != "some unknown event" {
		t.Errorf("expected formatted key, got: %s", desc)
	}
}

func TestDescribeGitHubEventLabeled(t *testing.T) {
	raw, _ := json.Marshal(map[string]any{
		"__typename": "LabeledEvent",
		"label":      map[string]any{"name": "bug"},
	})
	desc := describeGitHubEvent("LabeledEvent", raw)
	if !strings.Contains(desc, "bug") {
		t.Errorf("expected label name, got: %s", desc)
	}
}

func TestDescribeGitHubEventClosed(t *testing.T) {
	raw, _ := json.Marshal(map[string]any{"__typename": "ClosedEvent"})
	desc := describeGitHubEvent("ClosedEvent", raw)
	if !strings.Contains(desc, "closed") {
		t.Errorf("expected 'closed' in description, got: %s", desc)
	}
}

func TestDescribeGitHubEventComment(t *testing.T) {
	raw, _ := json.Marshal(map[string]any{
		"__typename": "IssueComment",
		"body":       "This is a test comment",
	})
	desc := describeGitHubEvent("IssueComment", raw)
	if !strings.Contains(desc, "commented") || !strings.Contains(desc, "test comment") {
		t.Errorf("expected comment in description, got: %s", desc)
	}
}

func TestDescribeGitHubEventUnknown(t *testing.T) {
	raw, _ := json.Marshal(map[string]any{"__typename": "SomeUnknownEvent"})
	desc := describeGitHubEvent("SomeUnknownEvent", raw)
	if desc != "" {
		t.Errorf("expected empty string for unknown event, got: %s", desc)
	}
}

// --- test helpers ---

func setupIssueActivityServer(t *testing.T, hasEvents bool) *testutil.MockServer {
	t.Helper()

	ms := testutil.NewMockServer(t)

	ms.HandleQuery("ListRepos", repoResolutionResponse())
	ms.HandleQuery("IssueByInfo", issueByInfoResolutionResponse())

	var timelineNodes []any
	if hasEvents {
		timelineNodes = []any{
			map[string]any{
				"id":        "t1",
				"key":       "issue.set_estimate",
				"createdAt": "2026-02-10T00:47:58Z",
				"data": map[string]any{
					"github_user":   map[string]any{"login": "dlakehammond"},
					"current_value": "5.0",
				},
			},
			map[string]any{
				"id":        "t2",
				"key":       "issue.set_priority",
				"createdAt": "2026-02-10T01:28:59Z",
				"data": map[string]any{
					"github_user": map[string]any{"login": "dlakehammond"},
					"priority":    map[string]any{"name": "High priority"},
				},
			},
			map[string]any{
				"id":        "t3",
				"key":       "issue.connect_issue_to_pr",
				"createdAt": "2026-02-07T23:06:36Z",
				"data": map[string]any{
					"pull_request":            map[string]any{"number": 6, "title": "Add due date support"},
					"pull_request_repository": map[string]any{"name": "task-tracker"},
				},
			},
		}
	}

	ms.HandleQuery("GetIssueTimeline", map[string]any{
		"data": map[string]any{
			"issueByInfo": map[string]any{
				"id":     "i1",
				"number": 1,
				"title":  "Fix login button alignment",
				"repository": map[string]any{
					"name":  "task-tracker",
					"owner": map[string]any{"login": "dlakehammond"},
				},
				"timelineItems": map[string]any{
					"totalCount": len(timelineNodes),
					"pageInfo": map[string]any{
						"hasNextPage": false,
						"endCursor":   "",
					},
					"nodes": timelineNodes,
				},
			},
		},
	})

	_ = cache.Set(resolve.RepoCacheKey("ws-123"), []resolve.CachedRepo{
		{ID: "r1", GhID: 12345, Name: "task-tracker", OwnerName: "dlakehammond"},
	})

	return ms
}

func setupGitHubTimelineServer(t *testing.T) *testutil.MockServer {
	t.Helper()

	ghMs := testutil.NewMockServer(t)

	ghMs.HandleQuery("GetGitHubTimeline", map[string]any{
		"data": map[string]any{
			"repository": map[string]any{
				"issueOrPullRequest": map[string]any{
					"timelineItems": map[string]any{
						"totalCount": 2,
						"pageInfo": map[string]any{
							"hasNextPage": false,
							"endCursor":   "",
						},
						"nodes": []any{
							map[string]any{
								"__typename": "LabeledEvent",
								"createdAt":  "2026-02-07T23:03:18Z",
								"actor":      map[string]any{"login": "dlakehammond"},
								"label":      map[string]any{"name": "enhancement"},
							},
							map[string]any{
								"__typename": "ClosedEvent",
								"createdAt":  "2026-02-11T10:00:00Z",
								"actor":      map[string]any{"login": "dlakehammond"},
							},
						},
					},
				},
			},
		},
	})

	return ghMs
}

func setupIssueActivityTestEnvWithGitHub(t *testing.T, ms *testutil.MockServer, ghMs *testutil.MockServer) {
	t.Helper()

	cacheDir := t.TempDir()
	configDir := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", cacheDir)
	t.Setenv("XDG_CONFIG_HOME", configDir)
	t.Setenv("ZH_API_KEY", "test-key")
	t.Setenv("ZH_WORKSPACE", "ws-123")
	t.Setenv("ZH_GITHUB_TOKEN", "")

	origNew := apiNewFunc
	apiNewFunc = func(apiKey string, opts ...api.Option) *api.Client {
		return api.New(apiKey, append(opts, api.WithEndpoint(ms.URL()))...)
	}
	t.Cleanup(func() { apiNewFunc = origNew })

	origGh := ghNewFunc
	ghNewFunc = func(method, token string, opts ...gh.Option) *gh.Client {
		return gh.New("pat", "test-token", append(opts, gh.WithEndpoint(ghMs.URL()))...)
	}
	t.Cleanup(func() { ghNewFunc = origGh })

	_ = cache.Set(resolve.RepoCacheKey("ws-123"), []resolve.CachedRepo{
		{ID: "r1", GhID: 12345, Name: "task-tracker", OwnerName: "dlakehammond"},
	})
}
