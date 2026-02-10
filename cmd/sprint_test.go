package cmd

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/dslh/zh/internal/api"
	"github.com/dslh/zh/internal/cache"
	"github.com/dslh/zh/internal/resolve"
	"github.com/dslh/zh/internal/testutil"
)

// --- sprint list ---

func TestSprintList(t *testing.T) {
	resetSprintFlags()

	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListSprints", sprintListResponse())

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
	rootCmd.SetArgs([]string{"sprint", "list"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("sprint list returned error: %v", err)
	}

	out := buf.String()

	// Check headers
	if !strings.Contains(out, "STATE") {
		t.Error("output should contain STATE header")
	}
	if !strings.Contains(out, "NAME") {
		t.Error("output should contain NAME header")
	}
	if !strings.Contains(out, "DATES") {
		t.Error("output should contain DATES header")
	}
	if !strings.Contains(out, "POINTS") {
		t.Error("output should contain POINTS header")
	}
	if !strings.Contains(out, "CLOSED") {
		t.Error("output should contain CLOSED header")
	}

	// Check sprint entries
	if !strings.Contains(out, "Sprint 47") {
		t.Error("output should contain active sprint name")
	}
	if !strings.Contains(out, "Sprint 46") {
		t.Error("output should contain closed sprint name")
	}
	if !strings.Contains(out, "active") {
		t.Error("output should show active state")
	}

	// Check footer
	if !strings.Contains(out, "sprint(s)") {
		t.Errorf("output should show sprint count, got: %s", out)
	}

	// Verify cache was populated
	cached, ok := cache.Get[[]resolve.CachedSprint](resolve.SprintCacheKey("ws-123"))
	if !ok {
		t.Error("sprints should be cached after listing")
	}
	if len(cached) != 3 {
		t.Errorf("expected 3 cached sprints, got %d", len(cached))
	}
}

func TestSprintListJSON(t *testing.T) {
	resetSprintFlags()

	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListSprints", sprintListResponse())

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
	rootCmd.SetArgs([]string{"sprint", "list", "--output=json"})
	outputFormat = "json"
	defer func() { outputFormat = "" }()

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("sprint list --output=json returned error: %v", err)
	}

	var result []any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\nOutput: %s", err, buf.String())
	}

	if len(result) != 3 {
		t.Errorf("expected 3 sprints in JSON output, got %d", len(result))
	}
}

func TestSprintListEmpty(t *testing.T) {
	resetSprintFlags()

	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListSprints", map[string]any{
		"data": map[string]any{
			"workspace": map[string]any{
				"sprints": map[string]any{
					"totalCount": 0,
					"pageInfo": map[string]any{
						"hasNextPage": false,
						"endCursor":   "",
					},
					"nodes": []any{},
				},
				"activeSprint":   nil,
				"upcomingSprint":  nil,
				"previousSprint": nil,
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
	rootCmd.SetArgs([]string{"sprint", "list"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("sprint list returned error: %v", err)
	}

	if !strings.Contains(buf.String(), "No sprints found") {
		t.Errorf("expected empty message, got: %s", buf.String())
	}
}

func TestSprintListStateFilter(t *testing.T) {
	resetSprintFlags()

	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListSprints", sprintListClosedResponse())

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
	rootCmd.SetArgs([]string{"sprint", "list", "--state=closed"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("sprint list --state=closed returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Sprint 46") {
		t.Error("output should contain closed sprint")
	}
	if !strings.Contains(out, "closed") {
		t.Error("output should show closed state")
	}
}

// --- sprint show ---

func TestSprintShow(t *testing.T) {
	resetSprintFlags()

	ms := testutil.NewMockServer(t)
	// Sprint resolution needs the sprints list
	ms.HandleQuery("ListSprints", sprintResolutionResponse())
	ms.HandleQuery("GetSprint", sprintShowResponse())

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
	rootCmd.SetArgs([]string{"sprint", "show", "Sprint 47"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("sprint show returned error: %v", err)
	}

	out := buf.String()

	if !strings.Contains(out, "SPRINT: Sprint 47") {
		t.Error("output should contain sprint title")
	}
	if !strings.Contains(out, "PROGRESS") {
		t.Error("output should contain PROGRESS section")
	}
	if !strings.Contains(out, "Points:") {
		t.Error("output should contain Points progress line")
	}
	if !strings.Contains(out, "Issues:") {
		t.Error("output should contain Issues progress line")
	}
	if !strings.Contains(out, "ISSUES") {
		t.Error("output should contain ISSUES section")
	}
	if !strings.Contains(out, "Fix login") {
		t.Error("output should contain issue title")
	}
	if !strings.Contains(out, "task-tracker#1") {
		t.Error("output should contain issue reference")
	}
}

func TestSprintShowDefaultCurrent(t *testing.T) {
	resetSprintFlags()

	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListSprints", sprintResolutionResponse())
	ms.HandleQuery("GetSprint", sprintShowResponse())

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
	rootCmd.SetArgs([]string{"sprint", "show"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("sprint show (no arg) returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "SPRINT: Sprint 47") {
		t.Error("should default to active sprint")
	}
}

func TestSprintShowJSON(t *testing.T) {
	resetSprintFlags()

	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListSprints", sprintResolutionResponse())
	ms.HandleQuery("GetSprint", sprintShowResponse())

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
	rootCmd.SetArgs([]string{"sprint", "show", "Sprint 47", "--output=json"})
	outputFormat = "json"
	defer func() { outputFormat = "" }()

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("sprint show --output=json returned error: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\nOutput: %s", err, buf.String())
	}

	if result["state"] != "OPEN" {
		t.Errorf("JSON should contain state=OPEN, got: %v", result["state"])
	}
	if result["sprintIssues"] == nil {
		t.Error("JSON should contain sprintIssues field")
	}
}

func TestSprintShowNotFound(t *testing.T) {
	resetSprintFlags()

	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListSprints", sprintResolutionResponse())

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
	rootCmd.SetArgs([]string{"sprint", "show", "nonexistent-sprint"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("sprint show should error for unknown sprint")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error = %q, want mention of not found", err.Error())
	}
}

func TestSprintShowNoIssues(t *testing.T) {
	resetSprintFlags()

	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListSprints", sprintResolutionResponse())
	ms.HandleQuery("GetSprint", sprintShowEmptyResponse())

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
	rootCmd.SetArgs([]string{"sprint", "show", "Sprint 47"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("sprint show returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "No issues in sprint") {
		t.Errorf("output should say no issues, got: %s", out)
	}
}

func TestSprintHelpText(t *testing.T) {
	resetSprintFlags()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"sprint", "--help"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("sprint --help returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "list") {
		t.Error("help should mention list subcommand")
	}
	if !strings.Contains(out, "show") {
		t.Error("help should mention show subcommand")
	}
}

// --- helpers ---

func sprintListResponse() map[string]any {
	return map[string]any{
		"data": map[string]any{
			"workspace": map[string]any{
				"sprints": map[string]any{
					"totalCount": 3,
					"pageInfo": map[string]any{
						"hasNextPage": false,
						"endCursor":   "",
					},
					"nodes": []any{
						map[string]any{
							"id":              "sprint-47",
							"name":            "",
							"generatedName":   "Sprint 47",
							"description":     "Focus on performance",
							"state":           "OPEN",
							"startAt":         "2026-01-20T00:00:00Z",
							"endAt":           "2026-02-03T00:00:00Z",
							"totalPoints":     float64(52),
							"completedPoints": float64(34),
							"closedIssuesCount": 8,
							"createdAt":       "2026-01-20T00:00:00Z",
							"updatedAt":       "2026-02-01T15:00:00Z",
						},
						map[string]any{
							"id":              "sprint-48",
							"name":            "",
							"generatedName":   "Sprint 48",
							"description":     "",
							"state":           "OPEN",
							"startAt":         "2026-02-03T00:00:00Z",
							"endAt":           "2026-02-17T00:00:00Z",
							"totalPoints":     float64(12),
							"completedPoints": float64(0),
							"closedIssuesCount": 0,
							"createdAt":       "2026-01-20T00:00:00Z",
							"updatedAt":       "2026-01-20T00:00:00Z",
						},
						map[string]any{
							"id":              "sprint-46",
							"name":            "",
							"generatedName":   "Sprint 46",
							"description":     "",
							"state":           "CLOSED",
							"startAt":         "2026-01-06T00:00:00Z",
							"endAt":           "2026-01-20T00:00:00Z",
							"totalPoints":     float64(48),
							"completedPoints": float64(48),
							"closedIssuesCount": 15,
							"createdAt":       "2026-01-06T00:00:00Z",
							"updatedAt":       "2026-01-20T12:00:00Z",
						},
					},
				},
				"activeSprint":   map[string]any{"id": "sprint-47"},
				"upcomingSprint":  map[string]any{"id": "sprint-48"},
				"previousSprint": map[string]any{"id": "sprint-46"},
			},
		},
	}
}

func sprintListClosedResponse() map[string]any {
	return map[string]any{
		"data": map[string]any{
			"workspace": map[string]any{
				"sprints": map[string]any{
					"totalCount": 1,
					"pageInfo": map[string]any{
						"hasNextPage": false,
						"endCursor":   "",
					},
					"nodes": []any{
						map[string]any{
							"id":              "sprint-46",
							"name":            "",
							"generatedName":   "Sprint 46",
							"description":     "",
							"state":           "CLOSED",
							"startAt":         "2026-01-06T00:00:00Z",
							"endAt":           "2026-01-20T00:00:00Z",
							"totalPoints":     float64(48),
							"completedPoints": float64(48),
							"closedIssuesCount": 15,
							"createdAt":       "2026-01-06T00:00:00Z",
							"updatedAt":       "2026-01-20T12:00:00Z",
						},
					},
				},
				"activeSprint":   nil,
				"upcomingSprint":  nil,
				"previousSprint": nil,
			},
		},
	}
}

func sprintResolutionResponse() map[string]any {
	return map[string]any{
		"data": map[string]any{
			"workspace": map[string]any{
				"sprints": map[string]any{
					"totalCount": 3,
					"pageInfo": map[string]any{
						"hasNextPage": false,
						"endCursor":   "",
					},
					"nodes": []any{
						map[string]any{
							"id":            "sprint-47",
							"name":          "",
							"generatedName": "Sprint 47",
							"state":         "OPEN",
							"startAt":       "2026-01-20T00:00:00Z",
							"endAt":         "2026-02-03T00:00:00Z",
						},
						map[string]any{
							"id":            "sprint-48",
							"name":          "",
							"generatedName": "Sprint 48",
							"state":         "OPEN",
							"startAt":       "2026-02-03T00:00:00Z",
							"endAt":         "2026-02-17T00:00:00Z",
						},
						map[string]any{
							"id":            "sprint-46",
							"name":          "",
							"generatedName": "Sprint 46",
							"state":         "CLOSED",
							"startAt":       "2026-01-06T00:00:00Z",
							"endAt":         "2026-01-20T00:00:00Z",
						},
					},
				},
				"activeSprint":   map[string]any{"id": "sprint-47"},
				"upcomingSprint":  map[string]any{"id": "sprint-48"},
				"previousSprint": map[string]any{"id": "sprint-46"},
			},
		},
	}
}

func sprintShowResponse() map[string]any {
	return map[string]any{
		"data": map[string]any{
			"node": map[string]any{
				"id":              "sprint-47",
				"name":            "",
				"generatedName":   "Sprint 47",
				"description":     "Focus on performance improvements",
				"state":           "OPEN",
				"startAt":         "2026-01-20T00:00:00Z",
				"endAt":           "2026-02-03T00:00:00Z",
				"totalPoints":     float64(52),
				"completedPoints": float64(34),
				"closedIssuesCount": 8,
				"createdAt":       "2026-01-20T00:00:00Z",
				"updatedAt":       "2026-02-01T15:00:00Z",
				"sprintIssues": map[string]any{
					"totalCount": 3,
					"pageInfo": map[string]any{
						"hasNextPage": false,
						"endCursor":   "",
					},
					"nodes": []any{
						map[string]any{
							"id": "si-1",
							"issue": map[string]any{
								"id":       "issue-1",
								"number":   1,
								"title":    "Fix login authentication flow",
								"state":    "CLOSED",
								"estimate": map[string]any{"value": 5},
								"repository": map[string]any{
									"name":      "task-tracker",
									"ownerName": "dlakehammond",
								},
								"assignees": map[string]any{
									"nodes": []any{
										map[string]any{"login": "johndoe"},
									},
								},
								"pipelineIssues": map[string]any{
									"nodes": []any{
										map[string]any{
											"pipeline": map[string]any{"name": "Done"},
										},
									},
								},
							},
						},
						map[string]any{
							"id": "si-2",
							"issue": map[string]any{
								"id":       "issue-2",
								"number":   2,
								"title":    "Update user permissions model",
								"state":    "OPEN",
								"estimate": map[string]any{"value": 3},
								"repository": map[string]any{
									"name":      "task-tracker",
									"ownerName": "dlakehammond",
								},
								"assignees": map[string]any{
									"nodes": []any{
										map[string]any{"login": "janedoe"},
									},
								},
								"pipelineIssues": map[string]any{
									"nodes": []any{
										map[string]any{
											"pipeline": map[string]any{"name": "In Progress"},
										},
									},
								},
							},
						},
						map[string]any{
							"id": "si-3",
							"issue": map[string]any{
								"id":       "issue-3",
								"number":   3,
								"title":    "Add rate limiting to API endpoints",
								"state":    "OPEN",
								"estimate": nil,
								"repository": map[string]any{
									"name":      "task-tracker",
									"ownerName": "dlakehammond",
								},
								"assignees": map[string]any{
									"nodes": []any{},
								},
								"pipelineIssues": map[string]any{
									"nodes": []any{
										map[string]any{
											"pipeline": map[string]any{"name": "To Do"},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func sprintShowEmptyResponse() map[string]any {
	return map[string]any{
		"data": map[string]any{
			"node": map[string]any{
				"id":              "sprint-47",
				"name":            "",
				"generatedName":   "Sprint 47",
				"description":     "",
				"state":           "OPEN",
				"startAt":         "2026-01-20T00:00:00Z",
				"endAt":           "2026-02-03T00:00:00Z",
				"totalPoints":     float64(0),
				"completedPoints": float64(0),
				"closedIssuesCount": 0,
				"createdAt":       "2026-01-20T00:00:00Z",
				"updatedAt":       "2026-01-20T00:00:00Z",
				"sprintIssues": map[string]any{
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
