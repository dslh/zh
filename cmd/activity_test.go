package cmd

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/dslh/zh/internal/api"
	"github.com/dslh/zh/internal/cache"
	"github.com/dslh/zh/internal/gh"
	"github.com/dslh/zh/internal/resolve"
	"github.com/dslh/zh/internal/testutil"
)

// --- parseTimeFlag ---

func TestParseTimeFlagRelative(t *testing.T) {
	now := time.Date(2026, 2, 13, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		input    string
		expected time.Time
	}{
		{"1d", now.AddDate(0, 0, -1)},
		{"7d", now.AddDate(0, 0, -7)},
		{"2h", now.Add(-2 * time.Hour)},
		{"30m", now.Add(-30 * time.Minute)},
		{"2w", now.AddDate(0, 0, -14)},
	}

	for _, tt := range tests {
		result, err := parseTimeFlag(tt.input, now)
		if err != nil {
			t.Errorf("parseTimeFlag(%q): unexpected error: %v", tt.input, err)
			continue
		}
		if !result.Equal(tt.expected) {
			t.Errorf("parseTimeFlag(%q) = %v, want %v", tt.input, result, tt.expected)
		}
	}
}

func TestParseTimeFlagKeywords(t *testing.T) {
	now := time.Date(2026, 2, 13, 14, 30, 0, 0, time.UTC)

	tests := []struct {
		input    string
		expected time.Time
	}{
		{"now", now},
		{"yesterday", time.Date(2026, 2, 12, 0, 0, 0, 0, time.UTC)},
		{"last week", time.Date(2026, 2, 6, 0, 0, 0, 0, time.UTC)},
	}

	for _, tt := range tests {
		result, err := parseTimeFlag(tt.input, now)
		if err != nil {
			t.Errorf("parseTimeFlag(%q): unexpected error: %v", tt.input, err)
			continue
		}
		if !result.Equal(tt.expected) {
			t.Errorf("parseTimeFlag(%q) = %v, want %v", tt.input, result, tt.expected)
		}
	}
}

func TestParseTimeFlagAbsolute(t *testing.T) {
	now := time.Date(2026, 2, 13, 12, 0, 0, 0, time.UTC)

	// ISO date
	result, err := parseTimeFlag("2026-02-01", now)
	if err != nil {
		t.Fatalf("parseTimeFlag ISO date: %v", err)
	}
	expected := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	if !result.Equal(expected) {
		t.Errorf("ISO date: got %v, want %v", result, expected)
	}

	// RFC3339
	result, err = parseTimeFlag("2026-02-01T10:00:00Z", now)
	if err != nil {
		t.Fatalf("parseTimeFlag RFC3339: %v", err)
	}
	expected = time.Date(2026, 2, 1, 10, 0, 0, 0, time.UTC)
	if !result.Equal(expected) {
		t.Errorf("RFC3339: got %v, want %v", result, expected)
	}
}

func TestParseTimeFlagEmpty(t *testing.T) {
	now := time.Date(2026, 2, 13, 12, 0, 0, 0, time.UTC)
	result, err := parseTimeFlag("", now)
	if err != nil {
		t.Fatalf("parseTimeFlag empty: %v", err)
	}
	if !result.Equal(now) {
		t.Errorf("empty should return now, got %v", result)
	}
}

func TestParseTimeFlagInvalid(t *testing.T) {
	now := time.Now()
	_, err := parseTimeFlag("invalid-time", now)
	if err == nil {
		t.Error("expected error for invalid time format")
	}
}

// --- activity command tests ---

func setupActivityTestEnv(t *testing.T, ms *testutil.MockServer) {
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

	// Pre-populate pipeline cache
	_ = cache.Set(resolve.PipelineCacheKey("ws-123"), []resolve.CachedPipeline{
		{ID: "p1", Name: "Backlog"},
		{ID: "p2", Name: "In Progress"},
	})

	// Pre-populate repo cache
	_ = cache.Set(resolve.RepoCacheKey("ws-123"), []resolve.CachedRepo{
		{ID: "r1", GhID: 12345, Name: "task-tracker", OwnerName: "dlakehammond"},
	})
}

func setupActivityTestEnvWithGitHub(t *testing.T, ms *testutil.MockServer, ghMs *testutil.MockServer) {
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

	_ = cache.Set(resolve.PipelineCacheKey("ws-123"), []resolve.CachedPipeline{
		{ID: "p1", Name: "Backlog"},
		{ID: "p2", Name: "In Progress"},
	})
	_ = cache.Set(resolve.RepoCacheKey("ws-123"), []resolve.CachedRepo{
		{ID: "r1", GhID: 12345, Name: "task-tracker", OwnerName: "dlakehammond"},
	})
}

func activityPipelineSearchResponse(pipelineName string, issues []map[string]any) map[string]any {
	return map[string]any{
		"data": map[string]any{
			"searchIssuesByPipeline": map[string]any{
				"totalCount": len(issues),
				"pageInfo": map[string]any{
					"hasNextPage": false,
					"endCursor":   "",
				},
				"nodes": issues,
			},
		},
	}
}

func activityClosedResponse(issues []map[string]any) map[string]any {
	return map[string]any{
		"data": map[string]any{
			"searchClosedIssues": map[string]any{
				"nodes": issues,
			},
		},
	}
}

func makeActivityIssueNode(id string, number int, title, repo, owner, updatedAt, pipeline string) map[string]any {
	node := map[string]any{
		"id":          id,
		"number":      number,
		"title":       title,
		"state":       "OPEN",
		"updatedAt":   updatedAt,
		"ghUpdatedAt": "",
		"repository": map[string]any{
			"name":      repo,
			"ownerName": owner,
		},
		"assignees": map[string]any{
			"nodes": []any{
				map[string]any{"login": "alice"},
			},
		},
	}
	if pipeline != "" {
		node["pipelineIssue"] = map[string]any{
			"pipeline": map[string]any{"name": pipeline},
		}
	}
	return node
}

func setupActivityServer(t *testing.T) *testutil.MockServer {
	t.Helper()

	ms := testutil.NewMockServer(t)

	now := time.Now().UTC()
	recent := now.Add(-2 * time.Hour).Format(time.RFC3339)
	old := now.Add(-48 * time.Hour).Format(time.RFC3339)

	// Pipeline search responses — mock handles both calls (one per pipeline)
	ms.HandleQuery("ActivitySearch", activityPipelineSearchResponse("In Progress", []map[string]any{
		makeActivityIssueNode("i1", 42, "Fix login page", "task-tracker", "dlakehammond", recent, "In Progress"),
		makeActivityIssueNode("i2", 38, "Add dark mode", "task-tracker", "dlakehammond", recent, "In Progress"),
		makeActivityIssueNode("i3", 15, "Old issue", "task-tracker", "dlakehammond", old, "Backlog"),
	}))

	// Closed issues
	ms.HandleQuery("ActivityClosed", activityClosedResponse([]map[string]any{
		makeActivityIssueNode("i4", 40, "Fix CORS headers", "task-tracker", "dlakehammond", recent, ""),
	}))

	return ms
}

func TestActivitySummaryDefault(t *testing.T) {
	resetActivityFlags()

	ms := setupActivityServer(t)
	setupActivityTestEnv(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"activity"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("activity returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Activity since") {
		t.Errorf("output should contain 'Activity since', got: %s", out)
	}
	if !strings.Contains(out, "task-tracker#42") {
		t.Errorf("output should contain issue ref task-tracker#42, got: %s", out)
	}
	if !strings.Contains(out, "Fix login page") {
		t.Errorf("output should contain issue title, got: %s", out)
	}
	// Old issue should be filtered out (older than 24h default)
	if strings.Contains(out, "Old issue") {
		t.Errorf("output should not contain old issue, got: %s", out)
	}
	if !strings.Contains(out, "issue(s) updated") {
		t.Errorf("output should contain summary count, got: %s", out)
	}
}

func TestActivityNoResults(t *testing.T) {
	resetActivityFlags()

	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ActivitySearch", activityPipelineSearchResponse("", nil))
	ms.HandleQuery("ActivityClosed", activityClosedResponse(nil))
	setupActivityTestEnv(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"activity"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("activity returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "No activity found") {
		t.Errorf("output should say no activity found, got: %s", out)
	}
}

func TestActivityJSON(t *testing.T) {
	resetActivityFlags()

	ms := setupActivityServer(t)
	setupActivityTestEnv(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"activity", "--output=json"})
	outputFormat = "json"
	defer func() { outputFormat = "" }()

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("activity --output=json returned error: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\nOutput: %s", err, buf.String())
	}

	if _, ok := result["from"]; !ok {
		t.Error("JSON should contain 'from' field")
	}
	if _, ok := result["to"]; !ok {
		t.Error("JSON should contain 'to' field")
	}

	issues, ok := result["issues"].([]any)
	if !ok {
		t.Fatal("JSON should contain issues array")
	}
	if len(issues) == 0 {
		t.Error("expected at least one issue in JSON output")
	}

	summary, ok := result["summary"].(map[string]any)
	if !ok {
		t.Fatal("JSON should contain summary object")
	}
	if summary["issueCount"] == nil {
		t.Error("summary should contain issueCount")
	}
}

func TestActivityEarlyTermination(t *testing.T) {
	resetActivityFlags()

	ms := testutil.NewMockServer(t)

	now := time.Now().UTC()
	old := now.Add(-48 * time.Hour).Format(time.RFC3339)

	// All issues are old — should get filtered out
	ms.HandleQuery("ActivitySearch", activityPipelineSearchResponse("Backlog", []map[string]any{
		makeActivityIssueNode("i1", 1, "Old issue 1", "task-tracker", "dlakehammond", old, "Backlog"),
		makeActivityIssueNode("i2", 2, "Old issue 2", "task-tracker", "dlakehammond", old, "Backlog"),
	}))
	ms.HandleQuery("ActivityClosed", activityClosedResponse(nil))
	setupActivityTestEnv(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"activity"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("activity returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "No activity found") {
		t.Errorf("should show no activity when all issues are old, got: %s", out)
	}
}

func TestActivityWithFromFlag(t *testing.T) {
	resetActivityFlags()

	ms := testutil.NewMockServer(t)

	now := time.Now().UTC()
	recent := now.Add(-2 * time.Hour).Format(time.RFC3339)
	threeDaysAgo := now.Add(-72 * time.Hour).Format(time.RFC3339)

	// Both issues are within 7 days but one is outside 24h
	ms.HandleQuery("ActivitySearch", activityPipelineSearchResponse("In Progress", []map[string]any{
		makeActivityIssueNode("i1", 42, "Recent issue", "task-tracker", "dlakehammond", recent, "In Progress"),
		makeActivityIssueNode("i2", 43, "Three day old issue", "task-tracker", "dlakehammond", threeDaysAgo, "In Progress"),
	}))
	ms.HandleQuery("ActivityClosed", activityClosedResponse(nil))
	setupActivityTestEnv(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"activity", "--from=7d"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("activity --from=7d returned error: %v", err)
	}

	out := buf.String()
	// Both issues should appear with 7d range
	if !strings.Contains(out, "Recent issue") {
		t.Errorf("output should contain recent issue, got: %s", out)
	}
	if !strings.Contains(out, "Three day old issue") {
		t.Errorf("output should contain 3-day-old issue with --from=7d, got: %s", out)
	}
}

func TestActivityRepoFilter(t *testing.T) {
	resetActivityFlags()

	ms := testutil.NewMockServer(t)

	now := time.Now().UTC()
	recent := now.Add(-2 * time.Hour).Format(time.RFC3339)

	ms.HandleQuery("ActivitySearch", activityPipelineSearchResponse("In Progress", []map[string]any{
		makeActivityIssueNode("i1", 42, "In right repo", "task-tracker", "dlakehammond", recent, "In Progress"),
		makeActivityIssueNode("i2", 43, "In wrong repo", "other-repo", "dlakehammond", recent, "In Progress"),
	}))
	ms.HandleQuery("ActivityClosed", activityClosedResponse(nil))
	ms.HandleQuery("ListRepos", map[string]any{
		"data": map[string]any{
			"workspace": map[string]any{
				"repositoriesConnection": map[string]any{
					"nodes": []any{
						map[string]any{"id": "r1", "ghId": 12345, "name": "task-tracker", "ownerName": "dlakehammond"},
						map[string]any{"id": "r2", "ghId": 12346, "name": "other-repo", "ownerName": "dlakehammond"},
					},
				},
			},
		},
	})
	setupActivityTestEnv(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"activity", "--repo=task-tracker"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("activity --repo returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "In right repo") {
		t.Errorf("output should contain matching repo issue, got: %s", out)
	}
	if strings.Contains(out, "In wrong repo") {
		t.Errorf("output should not contain non-matching repo issue, got: %s", out)
	}
}

func TestActivityPipelineFilter(t *testing.T) {
	resetActivityFlags()

	ms := testutil.NewMockServer(t)

	now := time.Now().UTC()
	recent := now.Add(-2 * time.Hour).Format(time.RFC3339)

	// Only one pipeline searched when filtering
	ms.HandleQuery("ActivitySearch", activityPipelineSearchResponse("In Progress", []map[string]any{
		makeActivityIssueNode("i1", 42, "In progress issue", "task-tracker", "dlakehammond", recent, "In Progress"),
	}))
	ms.HandleQuery("ActivityClosed", activityClosedResponse(nil))
	setupActivityTestEnv(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"activity", "--pipeline=In Progress"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("activity --pipeline returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "In progress issue") {
		t.Errorf("output should contain filtered pipeline issue, got: %s", out)
	}
}

func TestActivityWithDetail(t *testing.T) {
	resetActivityFlags()

	ms := testutil.NewMockServer(t)

	now := time.Now().UTC()
	recent := now.Add(-2 * time.Hour).Format(time.RFC3339)

	ms.HandleQuery("ActivitySearch", activityPipelineSearchResponse("In Progress", []map[string]any{
		makeActivityIssueNode("i1", 42, "Fix login page", "task-tracker", "dlakehammond", recent, "In Progress"),
	}))
	ms.HandleQuery("ActivityClosed", activityClosedResponse(nil))

	// Timeline response for the detail fetch
	ms.HandleQuery("GetIssueTimelineByNode", map[string]any{
		"data": map[string]any{
			"node": map[string]any{
				"__typename": "Issue",
				"id":         "i1",
				"number":     42,
				"title":      "Fix login page",
				"repository": map[string]any{
					"name":  "task-tracker",
					"owner": map[string]any{"login": "dlakehammond"},
				},
				"timelineItems": map[string]any{
					"totalCount": 1,
					"pageInfo": map[string]any{
						"hasNextPage": false,
						"endCursor":   "",
					},
					"nodes": []any{
						map[string]any{
							"id":        "t1",
							"key":       "issue.change_pipeline",
							"createdAt": recent,
							"data": map[string]any{
								"from_pipeline": map[string]any{"name": "Backlog"},
								"to_pipeline":   map[string]any{"name": "In Progress"},
							},
						},
					},
				},
			},
		},
	})
	setupActivityTestEnv(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"activity", "--detail"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("activity --detail returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "task-tracker#42") {
		t.Errorf("detail output should contain issue ref, got: %s", out)
	}
	if !strings.Contains(out, "Backlog") || !strings.Contains(out, "In Progress") {
		t.Errorf("detail output should contain pipeline move event, got: %s", out)
	}
	if !strings.Contains(out, "event(s)") {
		t.Errorf("detail output should contain event count, got: %s", out)
	}
}

func TestActivityGitHubNoAccess(t *testing.T) {
	resetActivityFlags()

	ms := setupActivityServer(t)
	setupActivityTestEnv(t, ms)

	activityGitHub = true

	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(errBuf)
	rootCmd.SetArgs([]string{"activity", "--github"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("activity --github without access returned error: %v", err)
	}

	// Should still show ZenHub results
	out := buf.String()
	if !strings.Contains(out, "task-tracker#42") {
		t.Errorf("output should still contain ZenHub issues, got: %s", out)
	}

	// Should warn about GitHub not configured
	errOut := errBuf.String()
	if !strings.Contains(errOut, "GitHub access not configured") {
		t.Errorf("stderr should warn about GitHub access, got: %s", errOut)
	}
}

func activityIssueByInfoResponse(id string, pipelineName string) map[string]any {
	result := map[string]any{
		"id": id,
	}
	if pipelineName != "" {
		result["pipelineIssue"] = map[string]any{
			"pipeline": map[string]any{"name": pipelineName},
		}
	} else {
		result["pipelineIssue"] = nil
	}
	return map[string]any{
		"data": map[string]any{
			"issueByInfo": result,
		},
	}
}

func activityDefaultPRPipelineResponse(pipelines []map[string]any) map[string]any {
	return map[string]any{
		"data": map[string]any{
			"workspace": map[string]any{
				"pipelinesConnection": map[string]any{
					"nodes": pipelines,
				},
			},
		},
	}
}

func TestActivityWithGitHubSearch(t *testing.T) {
	resetActivityFlags()

	ms := testutil.NewMockServer(t)
	ghMs := testutil.NewMockServer(t)

	now := time.Now().UTC()
	recent := now.Add(-2 * time.Hour).Format(time.RFC3339)

	ms.HandleQuery("ActivitySearch", activityPipelineSearchResponse("In Progress", []map[string]any{
		makeActivityIssueNode("i1", 42, "ZenHub issue", "task-tracker", "dlakehammond", recent, "In Progress"),
	}))
	ms.HandleQuery("ActivityClosed", activityClosedResponse(nil))

	// Pipeline resolution for GitHub-discovered issue
	ms.HandleQuery("ActivityIssueByInfo", activityIssueByInfoResponse("zh-99", "Review"))

	// GitHub search returns an additional issue
	ghMs.HandleQuery("ActivityGitHubSearch", map[string]any{
		"data": map[string]any{
			"search": map[string]any{
				"issueCount": 1,
				"pageInfo": map[string]any{
					"hasNextPage": false,
					"endCursor":   "",
				},
				"nodes": []any{
					map[string]any{
						"number":    99,
						"title":     "GitHub only issue",
						"updatedAt": recent,
						"repository": map[string]any{
							"name":  "task-tracker",
							"owner": map[string]any{"login": "dlakehammond"},
						},
					},
				},
			},
		},
	})

	setupActivityTestEnvWithGitHub(t, ms, ghMs)
	activityGitHub = true

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"activity", "--github"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("activity --github returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "task-tracker#42") {
		t.Errorf("output should contain ZenHub issue, got: %s", out)
	}
	if !strings.Contains(out, "task-tracker#99") {
		t.Errorf("output should contain GitHub-discovered issue, got: %s", out)
	}
	// GitHub-discovered issue should be resolved to "Review" pipeline, not "Unknown"
	if strings.Contains(out, "Unknown") {
		t.Errorf("output should not contain 'Unknown' pipeline — GitHub issues should be resolved, got: %s", out)
	}
	if !strings.Contains(out, "Review") {
		t.Errorf("output should contain 'Review' pipeline for resolved GitHub issue, got: %s", out)
	}
}

func TestActivityGitHubPipelineResolutionDefaultPR(t *testing.T) {
	resetActivityFlags()

	ms := testutil.NewMockServer(t)
	ghMs := testutil.NewMockServer(t)

	now := time.Now().UTC()
	recent := now.Add(-2 * time.Hour).Format(time.RFC3339)

	ms.HandleQuery("ActivitySearch", activityPipelineSearchResponse("", nil))
	ms.HandleQuery("ActivityClosed", activityClosedResponse(nil))

	// Issue has null pipelineIssue → falls back to default PR pipeline
	ms.HandleQuery("ActivityIssueByInfo", activityIssueByInfoResponse("zh-pr-10", ""))
	ms.HandleQuery("ActivityDefaultPRPipeline", activityDefaultPRPipelineResponse([]map[string]any{
		{"name": "Backlog", "isDefaultPRPipeline": false},
		{"name": "In Review", "isDefaultPRPipeline": true},
		{"name": "Done", "isDefaultPRPipeline": false},
	}))

	ghMs.HandleQuery("ActivityGitHubSearch", map[string]any{
		"data": map[string]any{
			"search": map[string]any{
				"issueCount": 1,
				"pageInfo": map[string]any{
					"hasNextPage": false,
					"endCursor":   "",
				},
				"nodes": []any{
					map[string]any{
						"number":    10,
						"title":     "Add feature X",
						"updatedAt": recent,
						"repository": map[string]any{
							"name":  "task-tracker",
							"owner": map[string]any{"login": "dlakehammond"},
						},
					},
				},
			},
		},
	})

	setupActivityTestEnvWithGitHub(t, ms, ghMs)
	activityGitHub = true

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"activity", "--github"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("activity --github returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "task-tracker#10") {
		t.Errorf("output should contain GitHub-discovered PR, got: %s", out)
	}
	// Should use default PR pipeline, not "Unknown"
	if strings.Contains(out, "Unknown") {
		t.Errorf("output should not contain 'Unknown' — null pipelineIssue should use default PR pipeline, got: %s", out)
	}
	if !strings.Contains(out, "In Review") {
		t.Errorf("output should contain 'In Review' (default PR pipeline), got: %s", out)
	}
}

func TestActivityClosedIssues(t *testing.T) {
	resetActivityFlags()

	ms := testutil.NewMockServer(t)

	now := time.Now().UTC()
	recent := now.Add(-2 * time.Hour).Format(time.RFC3339)

	// No pipeline issues
	ms.HandleQuery("ActivitySearch", activityPipelineSearchResponse("", nil))

	// One recently closed issue
	ms.HandleQuery("ActivityClosed", activityClosedResponse([]map[string]any{
		{
			"id":          "i5",
			"number":      50,
			"title":       "Recently closed issue",
			"state":       "CLOSED",
			"updatedAt":   recent,
			"ghUpdatedAt": "",
			"repository": map[string]any{
				"name":      "task-tracker",
				"ownerName": "dlakehammond",
			},
			"assignees": map[string]any{
				"nodes": []any{},
			},
		},
	}))
	setupActivityTestEnv(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"activity"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("activity returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "task-tracker#50") {
		t.Errorf("output should contain closed issue ref, got: %s", out)
	}
	if !strings.Contains(out, "Closed") {
		t.Errorf("output should show Closed pipeline, got: %s", out)
	}
}

// --- formatTimeAgo ---

func TestFormatTimeAgo(t *testing.T) {
	now := time.Now()

	tests := []struct {
		t        time.Time
		contains string
	}{
		{now.Add(-30 * time.Second), "just now"},
		{now.Add(-5 * time.Minute), "5m ago"},
		{now.Add(-1 * time.Minute), "1m ago"},
		{now.Add(-3 * time.Hour), "3h ago"},
		{now.Add(-1 * time.Hour), "1h ago"},
		{now.Add(-2 * 24 * time.Hour), "2d ago"},
		{now.Add(-1 * 24 * time.Hour), "1d ago"},
	}

	for _, tt := range tests {
		result := formatTimeAgo(tt.t)
		if result != tt.contains {
			t.Errorf("formatTimeAgo(%v) = %q, want %q", time.Since(tt.t), result, tt.contains)
		}
	}
}

// TestActivityHelp must be last — Cobra's --help flag leaks across Execute() calls.
func TestActivityHelp(t *testing.T) {
	resetActivityFlags()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"activity", "--help"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("activity --help returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "activity") {
		t.Errorf("help should mention activity, got: %s", out)
	}
	if !strings.Contains(out, "--from") {
		t.Error("help should mention --from flag")
	}
	if !strings.Contains(out, "--github") {
		t.Error("help should mention --github flag")
	}
	if !strings.Contains(out, "--detail") {
		t.Error("help should mention --detail flag")
	}
	if !strings.Contains(out, "--pipeline") {
		t.Error("help should mention --pipeline flag")
	}
	if !strings.Contains(out, "--repo") {
		t.Error("help should mention --repo flag")
	}
}
