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

// --- epic list ---

func TestEpicList(t *testing.T) {
	resetEpicFlags()

	ms := testutil.NewMockServer(t)
	handleEpicListQueries(ms)

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
	rootCmd.SetArgs([]string{"epic", "list"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("epic list returned error: %v", err)
	}

	out := buf.String()

	// Check headers
	if !strings.Contains(out, "TYPE") {
		t.Error("output should contain TYPE header")
	}
	if !strings.Contains(out, "STATE") {
		t.Error("output should contain STATE header")
	}
	if !strings.Contains(out, "TITLE") {
		t.Error("output should contain TITLE header")
	}
	if !strings.Contains(out, "ISSUES") {
		t.Error("output should contain ISSUES header")
	}

	// Check epic entries
	if !strings.Contains(out, "Q1 Platform Improvements") {
		t.Error("output should contain ZenHub epic title")
	}
	if !strings.Contains(out, "zenhub") {
		t.Error("output should contain 'zenhub' type")
	}
	if !strings.Contains(out, "legacy") {
		t.Error("output should contain 'legacy' type")
	}
	if !strings.Contains(out, "task-tracker#1") {
		t.Error("output should contain legacy epic repo#number reference")
	}

	// Check footer
	if !strings.Contains(out, "epic(s)") {
		t.Errorf("output should show epic count, got: %s", out)
	}

	// Verify cache was populated
	cached, ok := cache.Get[[]resolve.CachedEpic](resolve.EpicCacheKey("ws-123"))
	if !ok {
		t.Error("epics should be cached after listing")
	}
	if len(cached) != 2 {
		t.Errorf("expected 2 cached epics, got %d", len(cached))
	}
}

func TestEpicListJSON(t *testing.T) {
	resetEpicFlags()

	ms := testutil.NewMockServer(t)
	handleEpicListQueries(ms)

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
	rootCmd.SetArgs([]string{"epic", "list", "--output=json"})
	outputFormat = "json"
	defer func() { outputFormat = "" }()

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("epic list --output=json returned error: %v", err)
	}

	var result []any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\nOutput: %s", err, buf.String())
	}

	if len(result) != 2 {
		t.Errorf("expected 2 epics in JSON output, got %d", len(result))
	}
}

func TestEpicListEmpty(t *testing.T) {
	resetEpicFlags()

	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListZenhubEpicsFull", map[string]any{
		"data": map[string]any{
			"workspace": map[string]any{
				"zenhubEpics": map[string]any{
					"totalCount": 0,
					"pageInfo": map[string]any{
						"hasNextPage": false,
						"endCursor":   "",
					},
					"nodes": []any{},
				},
			},
		},
	})
	ms.HandleQuery("ListRoadmapEpicsFull", map[string]any{
		"data": map[string]any{
			"workspace": map[string]any{
				"roadmap": map[string]any{
					"items": map[string]any{
						"totalCount": 0,
						"pageInfo": map[string]any{
							"hasNextPage": false,
							"endCursor":   "",
						},
						"nodes": []any{},
					},
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
	rootCmd.SetArgs([]string{"epic", "list"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("epic list returned error: %v", err)
	}

	if !strings.Contains(buf.String(), "No epics found") {
		t.Errorf("expected empty message, got: %s", buf.String())
	}
}

// --- epic show ---

func TestEpicShowZenhub(t *testing.T) {
	resetEpicFlags()

	ms := testutil.NewMockServer(t)
	handleEpicResolutionQueries(ms)
	ms.HandleQuery("GetZenhubEpic", epicShowZenhubResponse())

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
	rootCmd.SetArgs([]string{"epic", "show", "Q1 Platform"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("epic show returned error: %v", err)
	}

	out := buf.String()

	if !strings.Contains(out, "EPIC: Q1 Platform Improvements") {
		t.Error("output should contain epic title")
	}
	if !strings.Contains(out, "ZenHub Epic") {
		t.Error("output should show type as ZenHub Epic")
	}
	if !strings.Contains(out, "in_progress") {
		t.Error("output should show state")
	}
	if !strings.Contains(out, "PROGRESS") {
		t.Error("output should contain PROGRESS section")
	}
	if !strings.Contains(out, "CHILD ISSUES") {
		t.Error("output should contain CHILD ISSUES section")
	}
	if !strings.Contains(out, "Fix login") {
		t.Error("output should contain child issue title")
	}
	if !strings.Contains(out, "DESCRIPTION") {
		t.Error("output should contain DESCRIPTION section")
	}
}

func TestEpicShowLegacy(t *testing.T) {
	resetEpicFlags()

	ms := testutil.NewMockServer(t)
	// Pre-populate cache with legacy epic
	_ = cache.Set(resolve.EpicCacheKey("ws-123"), []resolve.CachedEpic{
		{ID: "legacy-epic-1", Title: "Bug Tracker Improvements", Type: "legacy", IssueNumber: 1, RepoName: "task-tracker", RepoOwner: "dlakehammond"},
	})
	ms.HandleQuery("GetLegacyEpic", epicShowLegacyResponse())

	cacheDir := t.TempDir()
	configDir := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", cacheDir)
	t.Setenv("XDG_CONFIG_HOME", configDir)
	t.Setenv("ZH_API_KEY", "test-key")
	t.Setenv("ZH_WORKSPACE", "ws-123")
	t.Setenv("ZH_GITHUB_TOKEN", "")

	// Pre-populate cache (must happen after setting env)
	_ = cache.Set(resolve.EpicCacheKey("ws-123"), []resolve.CachedEpic{
		{ID: "legacy-epic-1", Title: "Bug Tracker Improvements", Type: "legacy", IssueNumber: 1, RepoName: "task-tracker", RepoOwner: "dlakehammond"},
	})

	origNew := apiNewFunc
	apiNewFunc = func(apiKey string, opts ...api.Option) *api.Client {
		return api.New(apiKey, append(opts, api.WithEndpoint(ms.URL()))...)
	}
	defer func() { apiNewFunc = origNew }()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"epic", "show", "Bug Tracker"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("epic show returned error: %v", err)
	}

	out := buf.String()

	if !strings.Contains(out, "EPIC:") {
		t.Error("output should contain epic header")
	}
	if !strings.Contains(out, "Legacy Epic") {
		t.Error("output should show type as Legacy Epic")
	}
	if !strings.Contains(out, "task-tracker#1") {
		t.Error("output should contain issue reference")
	}
}

func TestEpicShowJSON(t *testing.T) {
	resetEpicFlags()

	ms := testutil.NewMockServer(t)
	handleEpicResolutionQueries(ms)
	ms.HandleQuery("GetZenhubEpic", epicShowZenhubResponse())

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
	rootCmd.SetArgs([]string{"epic", "show", "Q1 Platform", "--output=json"})
	outputFormat = "json"
	defer func() { outputFormat = "" }()

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("epic show --output=json returned error: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\nOutput: %s", err, buf.String())
	}

	if _, ok := result["title"]; !ok {
		t.Error("JSON output should contain 'title' key")
	}
	if _, ok := result["state"]; !ok {
		t.Error("JSON output should contain 'state' key")
	}
}

func TestEpicHelpText(t *testing.T) {
	resetEpicFlags()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"epic", "--help"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("epic --help returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "list") {
		t.Error("help should mention list subcommand")
	}
	if !strings.Contains(out, "show") {
		t.Error("help should mention show subcommand")
	}
	if !strings.Contains(out, "alias") {
		t.Error("help should mention alias subcommand")
	}
}

// --- epic progress ---

func TestEpicProgressZenhub(t *testing.T) {
	resetEpicFlags()

	ms := testutil.NewMockServer(t)
	handleEpicResolutionQueries(ms)
	ms.HandleQuery("GetZenhubEpicProgress", epicProgressZenhubResponse())

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
	rootCmd.SetArgs([]string{"epic", "progress", "Q1 Platform"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("epic progress returned error: %v", err)
	}

	out := buf.String()

	if !strings.Contains(out, "EPIC PROGRESS: Q1 Platform Improvements") {
		t.Error("output should contain epic progress title")
	}
	if !strings.Contains(out, "PROGRESS") {
		t.Error("output should contain PROGRESS section")
	}
	if !strings.Contains(out, "Issues:") {
		t.Error("output should contain Issues progress line")
	}
	if !strings.Contains(out, "12/20") {
		t.Errorf("output should show 12/20 completed, got: %s", out)
	}
	if !strings.Contains(out, "Estimates:") {
		t.Error("output should contain Estimates progress line")
	}
	if !strings.Contains(out, "34/55") {
		t.Errorf("output should show 34/55 completed, got: %s", out)
	}
}

func TestEpicProgressLegacy(t *testing.T) {
	resetEpicFlags()

	ms := testutil.NewMockServer(t)
	ms.HandleQuery("GetLegacyEpicProgress", epicProgressLegacyResponse())

	cacheDir := t.TempDir()
	configDir := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", cacheDir)
	t.Setenv("XDG_CONFIG_HOME", configDir)
	t.Setenv("ZH_API_KEY", "test-key")
	t.Setenv("ZH_WORKSPACE", "ws-123")
	t.Setenv("ZH_GITHUB_TOKEN", "")

	_ = cache.Set(resolve.EpicCacheKey("ws-123"), []resolve.CachedEpic{
		{ID: "legacy-epic-1", Title: "Bug Tracker Improvements", Type: "legacy", IssueNumber: 1, RepoName: "task-tracker", RepoOwner: "dlakehammond"},
	})

	origNew := apiNewFunc
	apiNewFunc = func(apiKey string, opts ...api.Option) *api.Client {
		return api.New(apiKey, append(opts, api.WithEndpoint(ms.URL()))...)
	}
	defer func() { apiNewFunc = origNew }()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"epic", "progress", "Bug Tracker"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("epic progress (legacy) returned error: %v", err)
	}

	out := buf.String()

	if !strings.Contains(out, "EPIC PROGRESS:") {
		t.Error("output should contain epic progress header")
	}
	if !strings.Contains(out, "task-tracker#1") {
		t.Error("output should contain issue reference")
	}
	if !strings.Contains(out, "1/2") {
		t.Errorf("output should show 1/2 completed, got: %s", out)
	}
}

func TestEpicProgressNoIssues(t *testing.T) {
	resetEpicFlags()

	ms := testutil.NewMockServer(t)
	handleEpicResolutionQueries(ms)
	ms.HandleQuery("GetZenhubEpicProgress", epicProgressEmptyResponse())

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
	rootCmd.SetArgs([]string{"epic", "progress", "Q1 Platform"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("epic progress (no issues) returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "No child issues") {
		t.Errorf("output should say no child issues, got: %s", out)
	}
}

func TestEpicProgressJSON(t *testing.T) {
	resetEpicFlags()

	ms := testutil.NewMockServer(t)
	handleEpicResolutionQueries(ms)
	ms.HandleQuery("GetZenhubEpicProgress", epicProgressZenhubResponse())

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
	rootCmd.SetArgs([]string{"epic", "progress", "Q1 Platform", "--output=json"})
	outputFormat = "json"
	defer func() { outputFormat = "" }()

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("epic progress --output=json returned error: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\nOutput: %s", err, buf.String())
	}

	if result["issues"] == nil {
		t.Error("JSON should contain issues field")
	}
	if result["estimates"] == nil {
		t.Error("JSON should contain estimates field")
	}
	issues := result["issues"].(map[string]any)
	if issues["closed"] != float64(12) {
		t.Errorf("JSON issues.closed should be 12, got: %v", issues["closed"])
	}
	if issues["total"] != float64(20) {
		t.Errorf("JSON issues.total should be 20, got: %v", issues["total"])
	}
}

// --- epic alias ---

func TestEpicAliasSet(t *testing.T) {
	resetEpicFlags()
	resetEpicMutationFlags()

	ms := testutil.NewMockServer(t)
	handleEpicResolutionQueries(ms)

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
	rootCmd.SetArgs([]string{"epic", "alias", "Q1 Platform", "q1"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("epic alias returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "q1") {
		t.Error("output should contain alias name")
	}
	if !strings.Contains(out, "Q1 Platform Improvements") {
		t.Error("output should contain epic title")
	}
}

func TestEpicAliasList(t *testing.T) {
	resetEpicFlags()
	resetEpicMutationFlags()
	epicAliasList = true

	cacheDir := t.TempDir()
	configDir := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", cacheDir)
	t.Setenv("XDG_CONFIG_HOME", configDir)
	t.Setenv("ZH_API_KEY", "test-key")
	t.Setenv("ZH_WORKSPACE", "ws-123")
	t.Setenv("ZH_GITHUB_TOKEN", "")

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"epic", "alias", "--list"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("epic alias --list returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "No epic aliases configured") {
		t.Errorf("expected empty aliases message, got: %s", out)
	}
}

func TestEpicAliasDelete(t *testing.T) {
	resetEpicFlags()
	resetEpicMutationFlags()
	epicAliasDelete = true

	cacheDir := t.TempDir()
	configDir := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", cacheDir)
	t.Setenv("XDG_CONFIG_HOME", configDir)
	t.Setenv("ZH_API_KEY", "test-key")
	t.Setenv("ZH_WORKSPACE", "ws-123")
	t.Setenv("ZH_GITHUB_TOKEN", "")

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"epic", "alias", "--delete", "nonexistent"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("epic alias --delete should error for missing alias")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error = %q, want mention of not found", err.Error())
	}
}

// --- helpers ---

// handleEpicListQueries registers mock handlers for both the zenhubEpics
// and roadmap queries used by fetchEpicList.
func handleEpicListQueries(ms *testutil.MockServer) {
	ms.HandleQuery("ListZenhubEpicsFull", map[string]any{
		"data": map[string]any{
			"workspace": map[string]any{
				"zenhubEpics": map[string]any{
					"totalCount": 1,
					"pageInfo": map[string]any{
						"hasNextPage": false,
						"endCursor":   "",
					},
					"nodes": []any{
						map[string]any{
							"id":       "epic-zen-1",
							"title":    "Q1 Platform Improvements",
							"state":    "IN_PROGRESS",
							"startOn":  "2026-01-01",
							"endOn":    "2026-03-31",
							"estimate": map[string]any{"value": 34},
							"zenhubIssueCountProgress": map[string]any{
								"open":   8,
								"closed": 12,
								"total":  20,
							},
							"zenhubIssueEstimateProgress": map[string]any{
								"open":   21,
								"closed": 34,
								"total":  55,
							},
						},
					},
				},
			},
		},
	})
	ms.HandleQuery("ListRoadmapEpicsFull", map[string]any{
		"data": map[string]any{
			"workspace": map[string]any{
				"roadmap": map[string]any{
					"items": map[string]any{
						"totalCount": 2,
						"pageInfo": map[string]any{
							"hasNextPage": false,
							"endCursor":   "",
						},
						"nodes": []any{
							map[string]any{
								"__typename": "ZenhubEpic",
								"id":         "epic-zen-1",
								"title":      "Q1 Platform Improvements",
								"state":      "IN_PROGRESS",
								"startOn":    "2026-01-01",
								"endOn":      "2026-03-31",
								"estimate":   map[string]any{"value": 34},
								"zenhubIssueCountProgress": map[string]any{
									"open":   8,
									"closed": 12,
									"total":  20,
								},
								"zenhubIssueEstimateProgress": map[string]any{
									"open":   21,
									"closed": 34,
									"total":  55,
								},
							},
							map[string]any{
								"__typename": "Epic",
								"id":         "epic-legacy-1",
								"startOn":    nil,
								"endOn":      nil,
								"issue": map[string]any{
									"title":  "Bug Tracker Improvements",
									"number": 1,
									"state":  "OPEN",
									"repository": map[string]any{
										"name":      "task-tracker",
										"ownerName": "dlakehammond",
									},
								},
								"childIssues": map[string]any{
									"totalCount": 5,
								},
								"issueCountProgress": map[string]any{
									"open":   3,
									"closed": 2,
									"total":  5,
								},
								"issueEstimateProgress": map[string]any{
									"open":   0,
									"closed": 0,
									"total":  0,
								},
							},
						},
					},
				},
			},
		},
	})
}

// handleEpicResolutionQueries registers mock handlers for both queries
// used by resolve.FetchEpics during epic resolution.
func handleEpicResolutionQueries(ms *testutil.MockServer) {
	ms.HandleQuery("ListZenhubEpics", map[string]any{
		"data": map[string]any{
			"workspace": map[string]any{
				"zenhubEpics": map[string]any{
					"pageInfo": map[string]any{
						"hasNextPage": false,
						"endCursor":   "",
					},
					"nodes": []any{
						map[string]any{
							"id":    "epic-zen-1",
							"title": "Q1 Platform Improvements",
						},
					},
				},
			},
		},
	})
	ms.HandleQuery("ListRoadmapEpics", map[string]any{
		"data": map[string]any{
			"workspace": map[string]any{
				"roadmap": map[string]any{
					"items": map[string]any{
						"pageInfo": map[string]any{
							"hasNextPage": false,
							"endCursor":   "",
						},
						"nodes": []any{
							map[string]any{
								"__typename": "Epic",
								"id":         "epic-legacy-1",
								"issue": map[string]any{
									"title":  "Bug Tracker Improvements",
									"number": 1,
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
		},
	})
}

func epicShowZenhubResponse() map[string]any {
	return map[string]any{
		"data": map[string]any{
			"node": map[string]any{
				"id":        "epic-zen-1",
				"title":     "Q1 Platform Improvements",
				"body":      "Platform improvements for Q1:\n- Auth system\n- Performance",
				"state":     "IN_PROGRESS",
				"startOn":   "2026-01-01",
				"endOn":     "2026-03-31",
				"createdAt": "2025-12-15T10:00:00Z",
				"updatedAt": "2026-02-01T15:30:00Z",
				"estimate":  map[string]any{"value": 34},
				"creator": map[string]any{
					"id":         "user-1",
					"name":       "John Doe",
					"githubUser": map[string]any{"login": "johndoe"},
				},
				"assignees": map[string]any{
					"nodes": []any{
						map[string]any{
							"id":         "user-1",
							"name":       "John Doe",
							"githubUser": map[string]any{"login": "johndoe"},
						},
						map[string]any{
							"id":         "user-2",
							"name":       "Jane Doe",
							"githubUser": map[string]any{"login": "janedoe"},
						},
					},
				},
				"labels": map[string]any{
					"nodes": []any{
						map[string]any{"id": "lbl-1", "name": "platform", "color": "0000ff"},
						map[string]any{"id": "lbl-2", "name": "priority:high", "color": "ff0000"},
					},
				},
				"childIssues": map[string]any{
					"totalCount": 3,
					"nodes": []any{
						map[string]any{
							"id":         "issue-1",
							"number":     1,
							"title":      "Fix login authentication flow",
							"state":      "CLOSED",
							"estimate":   map[string]any{"value": 5},
							"repository": map[string]any{"name": "task-tracker", "ownerName": "dlakehammond"},
							"pipelineIssue": map[string]any{
								"pipeline": map[string]any{"id": "p3", "name": "Done"},
							},
						},
						map[string]any{
							"id":         "issue-2",
							"number":     2,
							"title":      "Update user permissions model",
							"state":      "OPEN",
							"estimate":   map[string]any{"value": 3},
							"repository": map[string]any{"name": "task-tracker", "ownerName": "dlakehammond"},
							"pipelineIssue": map[string]any{
								"pipeline": map[string]any{"id": "p2", "name": "Doing"},
							},
						},
						map[string]any{
							"id":            "issue-3",
							"number":        1,
							"title":         "Add rate limiting to endpoints",
							"state":         "OPEN",
							"estimate":      nil,
							"repository":    map[string]any{"name": "recipe-book", "ownerName": "dlakehammond"},
							"pipelineIssue": nil,
						},
					},
				},
				"zenhubIssueCountProgress": map[string]any{
					"open":   8,
					"closed": 12,
					"total":  20,
				},
				"zenhubIssueEstimateProgress": map[string]any{
					"open":   21,
					"closed": 34,
					"total":  55,
				},
				"blockingItems": map[string]any{
					"totalCount": 0,
					"nodes":      []any{},
				},
				"blockedItems": map[string]any{
					"totalCount": 0,
					"nodes":      []any{},
				},
			},
		},
	}
}

func epicProgressZenhubResponse() map[string]any {
	return map[string]any{
		"data": map[string]any{
			"node": map[string]any{
				"id":       "epic-zen-1",
				"title":    "Q1 Platform Improvements",
				"state":    "IN_PROGRESS",
				"estimate": map[string]any{"value": 34},
				"zenhubIssueCountProgress": map[string]any{
					"open":   8,
					"closed": 12,
					"total":  20,
				},
				"zenhubIssueEstimateProgress": map[string]any{
					"open":   21,
					"closed": 34,
					"total":  55,
				},
			},
		},
	}
}

func epicProgressLegacyResponse() map[string]any {
	return map[string]any{
		"data": map[string]any{
			"node": map[string]any{
				"id": "legacy-epic-1",
				"issue": map[string]any{
					"title":    "Bug Tracker Improvements",
					"number":   1,
					"state":    "OPEN",
					"estimate": nil,
					"repository": map[string]any{
						"name":      "task-tracker",
						"ownerName": "dlakehammond",
					},
				},
				"issueCountProgress": map[string]any{
					"open":   1,
					"closed": 1,
					"total":  2,
				},
				"issueEstimateProgress": map[string]any{
					"open":   0,
					"closed": 3,
					"total":  3,
				},
			},
		},
	}
}

func epicProgressEmptyResponse() map[string]any {
	return map[string]any{
		"data": map[string]any{
			"node": map[string]any{
				"id":       "epic-zen-1",
				"title":    "Q1 Platform Improvements",
				"state":    "OPEN",
				"estimate": nil,
				"zenhubIssueCountProgress": map[string]any{
					"open":   0,
					"closed": 0,
					"total":  0,
				},
				"zenhubIssueEstimateProgress": map[string]any{
					"open":   0,
					"closed": 0,
					"total":  0,
				},
			},
		},
	}
}

func epicShowLegacyResponse() map[string]any {
	return map[string]any{
		"data": map[string]any{
			"node": map[string]any{
				"id":        "legacy-epic-1",
				"startOn":   nil,
				"endOn":     nil,
				"createdAt": "2025-11-01T10:00:00Z",
				"updatedAt": "2026-01-15T12:00:00Z",
				"issue": map[string]any{
					"id":      "issue-legacy-1",
					"number":  1,
					"title":   "Bug Tracker Improvements",
					"body":    "Improve the bug tracker with:\n- Better search\n- Filters",
					"state":   "OPEN",
					"htmlUrl": "https://github.com/dlakehammond/task-tracker/issues/1",
					"repository": map[string]any{
						"id":        "repo-1",
						"name":      "task-tracker",
						"ownerName": "dlakehammond",
					},
					"assignees": map[string]any{
						"nodes": []any{
							map[string]any{"login": "dlakehammond"},
						},
					},
					"labels": map[string]any{
						"nodes": []any{
							map[string]any{"id": "lbl-1", "name": "enhancement", "color": "a2eeef"},
						},
					},
					"estimate": nil,
				},
				"childIssues": map[string]any{
					"totalCount": 2,
					"nodes": []any{
						map[string]any{
							"id":         "child-1",
							"number":     2,
							"title":      "Better search",
							"state":      "OPEN",
							"estimate":   nil,
							"repository": map[string]any{"name": "task-tracker", "ownerName": "dlakehammond"},
						},
						map[string]any{
							"id":         "child-2",
							"number":     3,
							"title":      "Add filters",
							"state":      "CLOSED",
							"estimate":   map[string]any{"value": 3},
							"repository": map[string]any{"name": "task-tracker", "ownerName": "dlakehammond"},
						},
					},
				},
				"issueCountProgress": map[string]any{
					"open":   1,
					"closed": 1,
					"total":  2,
				},
				"issueEstimateProgress": map[string]any{
					"open":   0,
					"closed": 3,
					"total":  3,
				},
			},
		},
	}
}
