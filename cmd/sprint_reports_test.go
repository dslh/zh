package cmd

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/dslh/zh/internal/api"
	"github.com/dslh/zh/internal/testutil"
)

// ── sprint velocity ──────────────────────────────────────────────────────

func TestSprintVelocity(t *testing.T) {
	resetSprintFlags()
	resetSprintReportFlags()

	ms := testutil.NewMockServer(t)
	ms.HandleQuery("SprintVelocity", sprintVelocityResponse())

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
	defer func() { apiNewFunc = origNew }()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"sprint", "velocity"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("sprint velocity returned error: %v", err)
	}

	out := buf.String()

	if !strings.Contains(out, "VELOCITY") {
		t.Error("output should contain VELOCITY header")
	}
	if !strings.Contains(out, "Sprint cadence") {
		t.Error("output should show sprint cadence")
	}
	if !strings.Contains(out, "Avg velocity") {
		t.Error("output should show average velocity")
	}
	if !strings.Contains(out, "SPRINT") {
		t.Error("output should contain SPRINT column header")
	}
	if !strings.Contains(out, "Sprint 47") {
		t.Error("output should contain active sprint")
	}
	if !strings.Contains(out, "in progress") {
		t.Error("output should show active sprint as in progress")
	}
	if !strings.Contains(out, "Sprint 46") {
		t.Error("output should contain closed sprint")
	}
	if !strings.Contains(out, "Sprint 45") {
		t.Error("output should contain second closed sprint")
	}
}

func TestSprintVelocityJSON(t *testing.T) {
	resetSprintFlags()
	resetSprintReportFlags()

	ms := testutil.NewMockServer(t)
	ms.HandleQuery("SprintVelocity", sprintVelocityResponse())

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
	defer func() { apiNewFunc = origNew }()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"sprint", "velocity", "--output=json"})
	outputFormat = "json"
	defer func() { outputFormat = "" }()

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("sprint velocity --output=json returned error: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\nOutput: %s", err, buf.String())
	}

	if result["workspace"] != "Dev Test" {
		t.Errorf("JSON should contain workspace name, got: %v", result["workspace"])
	}
	if result["closedSprints"] == nil {
		t.Error("JSON should contain closedSprints field")
	}
}

func TestSprintVelocityNoSprints(t *testing.T) {
	resetSprintFlags()
	resetSprintReportFlags()

	ms := testutil.NewMockServer(t)
	ms.HandleQuery("SprintVelocity", map[string]any{
		"data": map[string]any{
			"workspace": map[string]any{
				"displayName":                  "Dev Test",
				"averageSprintVelocity":        nil,
				"averageSprintVelocityWithDiff": nil,
				"sprintConfig":                 nil,
				"activeSprint":                 nil,
				"sprints": map[string]any{
					"totalCount": 0,
					"nodes":      []any{},
				},
			},
		},
	})

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
	defer func() { apiNewFunc = origNew }()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"sprint", "velocity"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("sprint velocity returned error: %v", err)
	}

	if !strings.Contains(buf.String(), "not configured") {
		t.Errorf("expected not configured message, got: %s", buf.String())
	}
}

func TestSprintVelocityNoActive(t *testing.T) {
	resetSprintFlags()
	resetSprintReportFlags()

	ms := testutil.NewMockServer(t)
	ms.HandleQuery("SprintVelocity", sprintVelocityResponse())

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
	defer func() { apiNewFunc = origNew }()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"sprint", "velocity", "--no-active"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("sprint velocity --no-active returned error: %v", err)
	}

	out := buf.String()
	if strings.Contains(out, "in progress") {
		t.Error("output should not show active sprint when --no-active is set")
	}
	if !strings.Contains(out, "Sprint 46") {
		t.Error("output should still show closed sprints")
	}
}

// ── sprint scope ─────────────────────────────────────────────────────────

func TestSprintScope(t *testing.T) {
	resetSprintFlags()
	resetSprintReportFlags()

	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListSprints", sprintResolutionResponse())
	ms.HandleQuery("SprintScopeChange", sprintScopeResponse())

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
	defer func() { apiNewFunc = origNew }()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"sprint", "scope", "Sprint 47"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("sprint scope returned error: %v", err)
	}

	out := buf.String()

	if !strings.Contains(out, "SCOPE CHANGES") {
		t.Error("output should contain SCOPE CHANGES header")
	}
	if !strings.Contains(out, "EVENT LOG") {
		t.Error("output should contain EVENT LOG section")
	}
	if !strings.Contains(out, "SUMMARY") {
		t.Error("output should contain SUMMARY section")
	}
	if !strings.Contains(out, "added") {
		t.Error("output should show added events")
	}
	if !strings.Contains(out, "removed") {
		t.Error("output should show removed events")
	}
	if !strings.Contains(out, "Initial scope") {
		t.Error("output should show initial scope in summary")
	}
	if !strings.Contains(out, "task-tracker") {
		t.Error("output should contain repo name")
	}
}

func TestSprintScopeSummaryOnly(t *testing.T) {
	resetSprintFlags()
	resetSprintReportFlags()

	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListSprints", sprintResolutionResponse())
	ms.HandleQuery("SprintScopeChange", sprintScopeResponse())

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
	defer func() { apiNewFunc = origNew }()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"sprint", "scope", "Sprint 47", "--summary"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("sprint scope --summary returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "SUMMARY") {
		t.Error("output should contain SUMMARY section")
	}
	if strings.Contains(out, "EVENT LOG") {
		t.Error("output should NOT contain EVENT LOG when --summary is set")
	}
}

func TestSprintScopeNoChanges(t *testing.T) {
	resetSprintFlags()
	resetSprintReportFlags()

	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListSprints", sprintResolutionResponse())
	ms.HandleQuery("SprintScopeChange", sprintScopeEmptyResponse())

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
	defer func() { apiNewFunc = origNew }()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"sprint", "scope", "Sprint 47"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("sprint scope returned error: %v", err)
	}

	if !strings.Contains(buf.String(), "No scope changes") {
		t.Errorf("expected no scope changes message, got: %s", buf.String())
	}
}

func TestSprintScopeJSON(t *testing.T) {
	resetSprintFlags()
	resetSprintReportFlags()

	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListSprints", sprintResolutionResponse())
	ms.HandleQuery("SprintScopeChange", sprintScopeResponse())

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
	defer func() { apiNewFunc = origNew }()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"sprint", "scope", "Sprint 47", "--output=json"})
	outputFormat = "json"
	defer func() { outputFormat = "" }()

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("sprint scope --output=json returned error: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\nOutput: %s", err, buf.String())
	}

	if result["sprint"] == nil {
		t.Error("JSON should contain sprint field")
	}
	if result["events"] == nil {
		t.Error("JSON should contain events field")
	}
}

// ── sprint review ────────────────────────────────────────────────────────

func TestSprintReview(t *testing.T) {
	resetSprintFlags()
	resetSprintReportFlags()

	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListSprints", sprintResolutionResponse())
	ms.HandleQuery("SprintReview", sprintReviewResponse())

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
	defer func() { apiNewFunc = origNew }()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"sprint", "review", "Sprint 47"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("sprint review returned error: %v", err)
	}

	out := buf.String()

	if !strings.Contains(out, "SPRINT REVIEW") {
		t.Error("output should contain SPRINT REVIEW header")
	}
	if !strings.Contains(out, "COMPLETED") {
		t.Error("output should show COMPLETED state")
	}
	if !strings.Contains(out, "PROGRESS") {
		t.Error("output should contain PROGRESS section")
	}
	if !strings.Contains(out, "REVIEW") {
		t.Error("output should contain REVIEW section")
	}
	if !strings.Contains(out, "sprint focused on") {
		t.Error("output should render review body")
	}
	if !strings.Contains(out, "--features") {
		t.Error("output should hint about --features flag")
	}
}

func TestSprintReviewNoReview(t *testing.T) {
	resetSprintFlags()
	resetSprintReportFlags()

	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListSprints", sprintResolutionResponse())
	ms.HandleQuery("SprintReview", sprintReviewNoReviewResponse())

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
	defer func() { apiNewFunc = origNew }()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"sprint", "review", "Sprint 47"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("sprint review returned error: %v", err)
	}

	if !strings.Contains(buf.String(), "No review has been generated") {
		t.Errorf("expected no review message, got: %s", buf.String())
	}
}

func TestSprintReviewJSON(t *testing.T) {
	resetSprintFlags()
	resetSprintReportFlags()

	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListSprints", sprintResolutionResponse())
	ms.HandleQuery("SprintReview", sprintReviewResponse())

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
	defer func() { apiNewFunc = origNew }()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"sprint", "review", "Sprint 47", "--output=json"})
	outputFormat = "json"
	defer func() { outputFormat = "" }()

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("sprint review --output=json returned error: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\nOutput: %s", err, buf.String())
	}

	if result["sprintReview"] == nil {
		t.Error("JSON should contain sprintReview field")
	}
}

func TestSprintReviewWithFeatures(t *testing.T) {
	resetSprintFlags()
	resetSprintReportFlags()

	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListSprints", sprintResolutionResponse())
	ms.HandleQuery("SprintReview", sprintReviewResponse())

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
	defer func() { apiNewFunc = origNew }()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"sprint", "review", "Sprint 47", "--features"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("sprint review --features returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "FEATURES") {
		t.Error("output should contain FEATURES section")
	}
	if !strings.Contains(out, "API Performance") {
		t.Error("output should contain feature title")
	}
	if !strings.Contains(out, "task-tracker") {
		t.Error("output should contain issue repo in features")
	}
}

// ── test fixtures ────────────────────────────────────────────────────────

func sprintVelocityResponse() map[string]any {
	return map[string]any{
		"data": map[string]any{
			"workspace": map[string]any{
				"displayName":           "Dev Test",
				"averageSprintVelocity": float64(42),
				"averageSprintVelocityWithDiff": map[string]any{
					"velocity":     float64(42),
					"difference":   float64(5),
					"sprintsCount": 3,
				},
				"sprintConfig": map[string]any{
					"kind":         "WEEKS",
					"period":       2,
					"startDay":     "Sunday",
					"endDay":       "Sunday",
					"tzIdentifier": "America/New_York",
				},
				"activeSprint": map[string]any{
					"id":              "sprint-47",
					"name":            "",
					"generatedName":   "Sprint 47",
					"state":           "OPEN",
					"startAt":         "2026-01-20T00:00:00Z",
					"endAt":           "2026-02-03T00:00:00Z",
					"totalPoints":     float64(52),
					"completedPoints": float64(18),
					"closedIssuesCount": 5,
					"sprintIssues": map[string]any{
						"totalCount": 15,
					},
				},
				"sprints": map[string]any{
					"totalCount": 3,
					"nodes": []any{
						map[string]any{
							"id":              "sprint-46",
							"name":            "",
							"generatedName":   "Sprint 46",
							"startAt":         "2026-01-06T00:00:00Z",
							"endAt":           "2026-01-20T00:00:00Z",
							"totalPoints":     float64(48),
							"completedPoints": float64(48),
							"closedIssuesCount": 15,
							"sprintIssues": map[string]any{
								"totalCount": 15,
							},
						},
						map[string]any{
							"id":              "sprint-45",
							"name":            "",
							"generatedName":   "Sprint 45",
							"startAt":         "2025-12-23T00:00:00Z",
							"endAt":           "2026-01-06T00:00:00Z",
							"totalPoints":     float64(38),
							"completedPoints": float64(38),
							"closedIssuesCount": 12,
							"sprintIssues": map[string]any{
								"totalCount": 12,
							},
						},
						map[string]any{
							"id":              "sprint-44",
							"name":            "",
							"generatedName":   "Sprint 44",
							"startAt":         "2025-12-09T00:00:00Z",
							"endAt":           "2025-12-23T00:00:00Z",
							"totalPoints":     float64(40),
							"completedPoints": float64(40),
							"closedIssuesCount": 13,
							"sprintIssues": map[string]any{
								"totalCount": 13,
							},
						},
					},
				},
			},
		},
	}
}

func sprintScopeResponse() map[string]any {
	return map[string]any{
		"data": map[string]any{
			"node": map[string]any{
				"id":              "sprint-47",
				"name":            "",
				"generatedName":   "Sprint 47",
				"state":           "OPEN",
				"startAt":         "2026-01-20T00:00:00Z",
				"endAt":           "2026-02-03T00:00:00Z",
				"totalPoints":     float64(52),
				"completedPoints": float64(34),
				"closedIssuesCount": 8,
				"scopeChange": map[string]any{
					"totalCount": 4,
					"pageInfo": map[string]any{
						"hasNextPage": false,
						"endCursor":   "",
					},
					"nodes": []any{
						map[string]any{
							"action":        "ISSUE_ADDED",
							"effectiveAt":   "2026-01-20T00:00:00Z",
							"estimateValue": float64(5),
							"issue": map[string]any{
								"id":       "issue-1",
								"number":   1,
								"title":    "Add due dates to tasks",
								"state":    "CLOSED",
								"estimate": map[string]any{"value": float64(5)},
								"repository": map[string]any{
									"name":      "task-tracker",
									"ownerName": "dlakehammond",
								},
							},
						},
						map[string]any{
							"action":        "ISSUE_ADDED",
							"effectiveAt":   "2026-01-20T00:00:00Z",
							"estimateValue": float64(3),
							"issue": map[string]any{
								"id":       "issue-2",
								"number":   2,
								"title":    "Fix date parsing bug",
								"state":    "CLOSED",
								"estimate": map[string]any{"value": float64(3)},
								"repository": map[string]any{
									"name":      "task-tracker",
									"ownerName": "dlakehammond",
								},
							},
						},
						map[string]any{
							"action":        "ISSUE_ADDED",
							"effectiveAt":   "2026-01-25T10:00:00Z",
							"estimateValue": float64(8),
							"issue": map[string]any{
								"id":       "issue-3",
								"number":   3,
								"title":    "Add priority levels",
								"state":    "OPEN",
								"estimate": map[string]any{"value": float64(8)},
								"repository": map[string]any{
									"name":      "task-tracker",
									"ownerName": "dlakehammond",
								},
							},
						},
						map[string]any{
							"action":        "ISSUE_REMOVED",
							"effectiveAt":   "2026-01-28T14:00:00Z",
							"estimateValue": float64(8),
							"issue": map[string]any{
								"id":       "issue-3",
								"number":   3,
								"title":    "Add priority levels",
								"state":    "OPEN",
								"estimate": map[string]any{"value": float64(8)},
								"repository": map[string]any{
									"name":      "task-tracker",
									"ownerName": "dlakehammond",
								},
							},
						},
					},
				},
			},
		},
	}
}

func sprintScopeEmptyResponse() map[string]any {
	return map[string]any{
		"data": map[string]any{
			"node": map[string]any{
				"id":              "sprint-47",
				"name":            "",
				"generatedName":   "Sprint 47",
				"state":           "OPEN",
				"startAt":         "2026-01-20T00:00:00Z",
				"endAt":           "2026-02-03T00:00:00Z",
				"totalPoints":     float64(0),
				"completedPoints": float64(0),
				"closedIssuesCount": 0,
				"scopeChange": map[string]any{
					"totalCount": 0,
					"pageInfo": map[string]any{
						"hasNextPage": false,
						"endCursor":   "",
					},
					"nodes": []any{},
				},
			},
		},
	}
}

func sprintReviewResponse() map[string]any {
	return map[string]any{
		"data": map[string]any{
			"node": map[string]any{
				"id":              "sprint-47",
				"name":            "",
				"generatedName":   "Sprint 47",
				"state":           "OPEN",
				"startAt":         "2026-01-20T00:00:00Z",
				"endAt":           "2026-02-03T00:00:00Z",
				"totalPoints":     float64(48),
				"completedPoints": float64(48),
				"closedIssuesCount": 15,
				"sprintReview": map[string]any{
					"id":              "review-1",
					"title":           "Sprint 47 Review",
					"body":            "This sprint focused on performance improvements and bug fixes.",
					"state":           "COMPLETED",
					"language":        "en",
					"lastGeneratedAt": "2026-02-03T15:45:00Z",
					"manuallyEdited":  true,
					"createdAt":       "2026-02-03T15:00:00Z",
					"updatedAt":       "2026-02-03T16:00:00Z",
					"initiatedBy": map[string]any{
						"id":   "user-1",
						"name": "Doug Hammond",
						"githubUser": map[string]any{
							"login": "dlakehammond",
						},
					},
					"sprintReviewFeatures": map[string]any{
						"totalCount": 2,
						"nodes": []any{
							map[string]any{
								"id":    "feature-1",
								"title": "API Performance",
								"aiGeneratedIssues": map[string]any{
									"totalCount": 2,
									"nodes": []any{
										map[string]any{
											"id":       "issue-1",
											"number":   1,
											"title":    "Optimize query performance",
											"state":    "CLOSED",
											"estimate": map[string]any{"value": float64(5)},
											"repository": map[string]any{
												"name":      "task-tracker",
												"ownerName": "dlakehammond",
											},
										},
										map[string]any{
											"id":       "issue-2",
											"number":   2,
											"title":    "Add caching layer",
											"state":    "CLOSED",
											"estimate": map[string]any{"value": float64(3)},
											"repository": map[string]any{
												"name":      "task-tracker",
												"ownerName": "dlakehammond",
											},
										},
									},
								},
								"manuallyAddedIssues": map[string]any{
									"totalCount": 0,
									"nodes":      []any{},
								},
							},
							map[string]any{
								"id":    "feature-2",
								"title": "Bug Fixes",
								"aiGeneratedIssues": map[string]any{
									"totalCount": 1,
									"nodes": []any{
										map[string]any{
											"id":       "issue-3",
											"number":   3,
											"title":    "Fix login timeout",
											"state":    "CLOSED",
											"estimate": map[string]any{"value": float64(2)},
											"repository": map[string]any{
												"name":      "task-tracker",
												"ownerName": "dlakehammond",
											},
										},
									},
								},
								"manuallyAddedIssues": map[string]any{
									"totalCount": 1,
									"nodes": []any{
										map[string]any{
											"id":       "issue-4",
											"number":   4,
											"title":    "Fix session handling",
											"state":    "CLOSED",
											"estimate": map[string]any{"value": float64(1)},
											"repository": map[string]any{
												"name":      "task-tracker",
												"ownerName": "dlakehammond",
											},
										},
									},
								},
							},
						},
					},
					"sprintReviewSchedules": map[string]any{
						"totalCount": 1,
						"nodes": []any{
							map[string]any{
								"id":          "schedule-1",
								"title":       "Sprint Review Meeting",
								"startAt":     "2026-02-03T14:00:00Z",
								"completedAt": "2026-02-03T15:00:00Z",
							},
						},
					},
					"issuesClosedAfterSprintReview": map[string]any{
						"totalCount": 1,
						"nodes": []any{
							map[string]any{
								"id":       "issue-5",
								"number":   5,
								"title":    "Last-minute hotfix",
								"state":    "CLOSED",
								"estimate": map[string]any{"value": float64(1)},
								"repository": map[string]any{
									"name":      "task-tracker",
									"ownerName": "dlakehammond",
								},
							},
						},
					},
				},
			},
		},
	}
}

func sprintReviewNoReviewResponse() map[string]any {
	return map[string]any{
		"data": map[string]any{
			"node": map[string]any{
				"id":              "sprint-47",
				"name":            "",
				"generatedName":   "Sprint 47",
				"state":           "OPEN",
				"startAt":         "2026-01-20T00:00:00Z",
				"endAt":           "2026-02-03T00:00:00Z",
				"totalPoints":     float64(52),
				"completedPoints": float64(34),
				"closedIssuesCount": 8,
				"sprintReview":    nil,
			},
		},
	}
}
