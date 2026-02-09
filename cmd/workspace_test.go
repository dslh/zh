package cmd

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/dslh/zh/internal/api"
	"github.com/dslh/zh/internal/cache"
	"github.com/dslh/zh/internal/testutil"
)

func resetWorkspaceFlags() {
	workspaceListFavorites = false
	workspaceListRecent = false
}

// --- workspace list ---

func TestWorkspaceList(t *testing.T) {
	resetWorkspaceFlags()

	ms := testutil.NewMockServer(t)
	ms.HandleQuery("zenhubOrganizations", map[string]any{
		"data": map[string]any{
			"viewer": map[string]any{
				"zenhubOrganizations": map[string]any{
					"nodes": []any{
						map[string]any{
							"id":   "org1",
							"name": "TestOrg",
							"workspaces": map[string]any{
								"nodes": []any{
									map[string]any{
										"id":               "ws1",
										"name":             "Development",
										"displayName":      "Development",
										"description":      nil,
										"isFavorite":       false,
										"viewerPermission": "ADMIN",
										"repositoriesConnection": map[string]any{
											"totalCount": 5,
										},
										"pipelinesConnection": map[string]any{
											"totalCount": 3,
										},
									},
									map[string]any{
										"id":               "ws2",
										"name":             "DevOps",
										"displayName":      "DevOps",
										"description":      nil,
										"isFavorite":       true,
										"viewerPermission": "WRITE",
										"repositoriesConnection": map[string]any{
											"totalCount": 2,
										},
										"pipelinesConnection": map[string]any{
											"totalCount": 4,
										},
									},
								},
							},
						},
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
	t.Setenv("ZH_WORKSPACE", "ws1")
	t.Setenv("ZH_GITHUB_TOKEN", "")

	// Inject the mock client endpoint
	origNew := apiNewFunc
	apiNewFunc = func(apiKey string, opts ...api.Option) *api.Client {
		return api.New(apiKey, append(opts, api.WithEndpoint(ms.URL()))...)
	}
	defer func() { apiNewFunc = origNew }()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"workspace", "list"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("workspace list returned error: %v", err)
	}

	out := buf.String()

	// Check headers
	if !strings.Contains(out, "ORGANIZATION") {
		t.Error("output should contain ORGANIZATION header")
	}
	if !strings.Contains(out, "WORKSPACE") {
		t.Error("output should contain WORKSPACE header")
	}

	// Check workspace names
	if !strings.Contains(out, "Development *") {
		t.Error("output should show Development with * for current workspace")
	}
	if !strings.Contains(out, "DevOps") {
		t.Error("output should show DevOps workspace")
	}

	// Check org name
	if !strings.Contains(out, "TestOrg") {
		t.Error("output should show organization name")
	}

	// Check footer
	if !strings.Contains(out, "Total: 2 workspace(s)") {
		t.Errorf("output should show total count, got: %s", out)
	}
}

func TestWorkspaceListJSON(t *testing.T) {
	resetWorkspaceFlags()

	ms := testutil.NewMockServer(t)
	ms.HandleQuery("zenhubOrganizations", map[string]any{
		"data": map[string]any{
			"viewer": map[string]any{
				"zenhubOrganizations": map[string]any{
					"nodes": []any{
						map[string]any{
							"id":   "org1",
							"name": "TestOrg",
							"workspaces": map[string]any{
								"nodes": []any{
									map[string]any{
										"id":               "ws1",
										"name":             "Dev",
										"displayName":      "Dev",
										"viewerPermission": "ADMIN",
										"repositoriesConnection": map[string]any{"totalCount": 1},
										"pipelinesConnection":    map[string]any{"totalCount": 2},
									},
								},
							},
						},
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
	t.Setenv("ZH_WORKSPACE", "ws1")
	t.Setenv("ZH_GITHUB_TOKEN", "")

	origNew := apiNewFunc
	apiNewFunc = func(apiKey string, opts ...api.Option) *api.Client {
		return api.New(apiKey, append(opts, api.WithEndpoint(ms.URL()))...)
	}
	defer func() { apiNewFunc = origNew }()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"workspace", "list", "--output=json"})
	outputFormat = "json"
	defer func() { outputFormat = "" }()

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("workspace list --output=json returned error: %v", err)
	}

	var result []any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\nOutput: %s", err, buf.String())
	}

	if len(result) != 1 {
		t.Errorf("expected 1 workspace in JSON output, got %d", len(result))
	}
}

func TestWorkspaceListRecent(t *testing.T) {
	resetWorkspaceFlags()

	ms := testutil.NewMockServer(t)
	ms.HandleQuery("recentlyViewedWorkspaces", map[string]any{
		"data": map[string]any{
			"recentlyViewedWorkspaces": map[string]any{
				"nodes": []any{
					map[string]any{
						"id":               "ws1",
						"name":             "Recent WS",
						"displayName":      "Recent WS",
						"viewerPermission": "WRITE",
						"zenhubOrganization": map[string]any{
							"id":   "org1",
							"name": "MyOrg",
						},
						"repositoriesConnection": map[string]any{"totalCount": 3},
						"pipelinesConnection":    map[string]any{"totalCount": 5},
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
	t.Setenv("ZH_WORKSPACE", "")
	t.Setenv("ZH_GITHUB_TOKEN", "")

	origNew := apiNewFunc
	apiNewFunc = func(apiKey string, opts ...api.Option) *api.Client {
		return api.New(apiKey, append(opts, api.WithEndpoint(ms.URL()))...)
	}
	defer func() { apiNewFunc = origNew }()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"workspace", "list", "--recent"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("workspace list --recent returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Recent WS") {
		t.Errorf("output should contain 'Recent WS', got: %s", out)
	}
}

func TestWorkspaceListEmpty(t *testing.T) {
	resetWorkspaceFlags()

	ms := testutil.NewMockServer(t)
	ms.HandleQuery("zenhubOrganizations", map[string]any{
		"data": map[string]any{
			"viewer": map[string]any{
				"zenhubOrganizations": map[string]any{
					"nodes": []any{},
				},
			},
		},
	})

	cacheDir := t.TempDir()
	configDir := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", cacheDir)
	t.Setenv("XDG_CONFIG_HOME", configDir)
	t.Setenv("ZH_API_KEY", "test-key")
	t.Setenv("ZH_WORKSPACE", "")
	t.Setenv("ZH_GITHUB_TOKEN", "")

	origNew := apiNewFunc
	apiNewFunc = func(apiKey string, opts ...api.Option) *api.Client {
		return api.New(apiKey, append(opts, api.WithEndpoint(ms.URL()))...)
	}
	defer func() { apiNewFunc = origNew }()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"workspace", "list"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("workspace list returned error: %v", err)
	}

	if !strings.Contains(buf.String(), "No workspaces found") {
		t.Errorf("expected empty message, got: %s", buf.String())
	}
}

// --- workspace show ---

func TestWorkspaceShowDefault(t *testing.T) {
	resetWorkspaceFlags()

	ms := testutil.NewMockServer(t)
	ms.HandleQuery("GetWorkspace", workspaceDetailResponse())

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
	rootCmd.SetArgs([]string{"workspace", "show"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("workspace show returned error: %v", err)
	}

	out := buf.String()

	if !strings.Contains(out, "WORKSPACE: Dev Test") {
		t.Error("output should contain workspace title")
	}
	if !strings.Contains(out, "TestOrg") {
		t.Error("output should contain organization name")
	}
	if !strings.Contains(out, "SPRINT CONFIGURATION") {
		t.Error("output should contain sprint configuration section")
	}
	if !strings.Contains(out, "SUMMARY") {
		t.Error("output should contain summary section")
	}
	if !strings.Contains(out, "Repositories:") {
		t.Error("output should contain repositories count")
	}
	if !strings.Contains(out, "Pipelines:") {
		t.Error("output should contain pipelines count")
	}
}

func TestWorkspaceShowNoWorkspace(t *testing.T) {
	resetWorkspaceFlags()

	cacheDir := t.TempDir()
	configDir := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", cacheDir)
	t.Setenv("XDG_CONFIG_HOME", configDir)
	t.Setenv("ZH_API_KEY", "test-key")
	t.Setenv("ZH_WORKSPACE", "")
	t.Setenv("ZH_GITHUB_TOKEN", "")

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"workspace", "show"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("workspace show should error when no workspace configured")
	}
	if !strings.Contains(err.Error(), "no workspace") {
		t.Errorf("error = %q, want mention of no workspace", err.Error())
	}
}

func TestWorkspaceShowNamed(t *testing.T) {
	resetWorkspaceFlags()

	ms := testutil.NewMockServer(t)
	// Handle workspace list for resolution
	ms.HandleQuery("zenhubOrganizations", map[string]any{
		"data": map[string]any{
			"viewer": map[string]any{
				"zenhubOrganizations": map[string]any{
					"nodes": []any{
						map[string]any{
							"id":   "org1",
							"name": "TestOrg",
							"workspaces": map[string]any{
								"nodes": []any{
									map[string]any{
										"id":               "ws-target",
										"name":             "Target WS",
										"displayName":      "Target WS",
										"viewerPermission": "ADMIN",
										"repositoriesConnection": map[string]any{"totalCount": 1},
										"pipelinesConnection":    map[string]any{"totalCount": 2},
									},
								},
							},
						},
					},
				},
			},
		},
	})
	ms.HandleQuery("GetWorkspace", workspaceDetailResponseWithName("ws-target", "Target WS"))

	cacheDir := t.TempDir()
	configDir := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", cacheDir)
	t.Setenv("XDG_CONFIG_HOME", configDir)
	t.Setenv("ZH_API_KEY", "test-key")
	t.Setenv("ZH_WORKSPACE", "ws-other")
	t.Setenv("ZH_GITHUB_TOKEN", "")

	origNew := apiNewFunc
	apiNewFunc = func(apiKey string, opts ...api.Option) *api.Client {
		return api.New(apiKey, append(opts, api.WithEndpoint(ms.URL()))...)
	}
	defer func() { apiNewFunc = origNew }()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"workspace", "show", "Target"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("workspace show Target returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "WORKSPACE: Target WS") {
		t.Errorf("output should contain workspace title, got: %s", out)
	}
}

// --- workspace switch ---

func TestWorkspaceSwitch(t *testing.T) {
	resetWorkspaceFlags()

	ms := testutil.NewMockServer(t)
	ms.HandleQuery("zenhubOrganizations", map[string]any{
		"data": map[string]any{
			"viewer": map[string]any{
				"zenhubOrganizations": map[string]any{
					"nodes": []any{
						map[string]any{
							"id":   "org1",
							"name": "TestOrg",
							"workspaces": map[string]any{
								"nodes": []any{
									map[string]any{
										"id":               "ws-new",
										"name":             "New WS",
										"displayName":      "New WS",
										"viewerPermission": "ADMIN",
										"repositoriesConnection": map[string]any{"totalCount": 1},
										"pipelinesConnection":    map[string]any{"totalCount": 2},
									},
								},
							},
						},
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
	t.Setenv("ZH_WORKSPACE", "ws-old")
	t.Setenv("ZH_GITHUB_TOKEN", "")

	// Create old workspace cache to verify it gets cleared
	oldKey := cache.NewScopedKey("pipelines", "ws-old")
	if err := cache.Set(oldKey, "old-data"); err != nil {
		t.Fatal(err)
	}

	origNew := apiNewFunc
	apiNewFunc = func(apiKey string, opts ...api.Option) *api.Client {
		return api.New(apiKey, append(opts, api.WithEndpoint(ms.URL()))...)
	}
	defer func() { apiNewFunc = origNew }()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"workspace", "switch", "New WS"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("workspace switch returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, `Switched to workspace "New WS"`) {
		t.Errorf("output should confirm switch, got: %s", out)
	}

	// Old workspace cache should be cleared
	if _, ok := cache.Get[string](oldKey); ok {
		t.Error("old workspace cache should be cleared after switch")
	}
}

func TestWorkspaceSwitchAlreadyCurrent(t *testing.T) {
	resetWorkspaceFlags()

	cacheDir := t.TempDir()
	configDir := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", cacheDir)
	t.Setenv("XDG_CONFIG_HOME", configDir)
	t.Setenv("ZH_API_KEY", "test-key")
	t.Setenv("ZH_WORKSPACE", "ws-current")
	t.Setenv("ZH_GITHUB_TOKEN", "")

	// Pre-populate cache with the workspace
	entries := []cachedWorkspace{
		{ID: "ws-current", Name: "Current", DisplayName: "Current", OrgName: "TestOrg"},
	}
	_ = cache.Set(cache.NewKey("workspaces"), entries)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"workspace", "switch", "Current"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("workspace switch returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Already using workspace") {
		t.Errorf("output should say already using, got: %s", out)
	}
}

func TestWorkspaceSwitchNotFound(t *testing.T) {
	resetWorkspaceFlags()

	ms := testutil.NewMockServer(t)
	ms.HandleQuery("zenhubOrganizations", map[string]any{
		"data": map[string]any{
			"viewer": map[string]any{
				"zenhubOrganizations": map[string]any{
					"nodes": []any{
						map[string]any{
							"id":   "org1",
							"name": "TestOrg",
							"workspaces": map[string]any{
								"nodes": []any{
									map[string]any{
										"id":               "ws1",
										"name":             "Development",
										"displayName":      "Development",
										"viewerPermission": "ADMIN",
										"repositoriesConnection": map[string]any{"totalCount": 1},
										"pipelinesConnection":    map[string]any{"totalCount": 2},
									},
								},
							},
						},
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
	t.Setenv("ZH_WORKSPACE", "ws1")
	t.Setenv("ZH_GITHUB_TOKEN", "")

	origNew := apiNewFunc
	apiNewFunc = func(apiKey string, opts ...api.Option) *api.Client {
		return api.New(apiKey, append(opts, api.WithEndpoint(ms.URL()))...)
	}
	defer func() { apiNewFunc = origNew }()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"workspace", "switch", "nonexistent"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("workspace switch should error for nonexistent workspace")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error = %q, want mention of not found", err.Error())
	}
}

// --- workspace repos ---

func TestWorkspaceRepos(t *testing.T) {
	resetWorkspaceFlags()

	ms := testutil.NewMockServer(t)
	ms.HandleQuery("WorkspaceRepos", map[string]any{
		"data": map[string]any{
			"workspace": map[string]any{
				"repositoriesConnection": map[string]any{
					"totalCount": 2,
					"pageInfo": map[string]any{
						"hasNextPage": false,
						"endCursor":   "",
					},
					"nodes": []any{
						map[string]any{
							"id":         "repo1",
							"ghId":       12345,
							"name":       "task-tracker",
							"ownerName":  "dlakehammond",
							"isPrivate":  false,
							"isArchived": false,
						},
						map[string]any{
							"id":         "repo2",
							"ghId":       67890,
							"name":       "recipe-book",
							"ownerName":  "dlakehammond",
							"isPrivate":  true,
							"isArchived": false,
						},
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
	rootCmd.SetArgs([]string{"workspace", "repos"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("workspace repos returned error: %v", err)
	}

	out := buf.String()

	if !strings.Contains(out, "dlakehammond/task-tracker") {
		t.Error("output should contain task-tracker repo")
	}
	if !strings.Contains(out, "dlakehammond/recipe-book") {
		t.Error("output should contain recipe-book repo")
	}
	if !strings.Contains(out, "Total: 2 repo(s)") {
		t.Errorf("output should show repo count, got: %s", out)
	}

	// Verify cache was populated
	key := cache.NewScopedKey("repos", "ws-123")
	repos, ok := cache.Get[[]cachedRepo](key)
	if !ok {
		t.Error("repos should be cached after listing")
	}
	if len(repos) != 2 {
		t.Errorf("expected 2 cached repos, got %d", len(repos))
	}
}

func TestWorkspaceReposNoWorkspace(t *testing.T) {
	resetWorkspaceFlags()

	cacheDir := t.TempDir()
	configDir := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", cacheDir)
	t.Setenv("XDG_CONFIG_HOME", configDir)
	t.Setenv("ZH_API_KEY", "test-key")
	t.Setenv("ZH_WORKSPACE", "")
	t.Setenv("ZH_GITHUB_TOKEN", "")

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"workspace", "repos"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("workspace repos should error when no workspace configured")
	}
	if !strings.Contains(err.Error(), "no workspace") {
		t.Errorf("error = %q, want mention of no workspace", err.Error())
	}
}

func TestWorkspaceHelpText(t *testing.T) {
	resetWorkspaceFlags()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"workspace", "--help"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("workspace --help returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "list") {
		t.Error("help should mention list subcommand")
	}
	if !strings.Contains(out, "show") {
		t.Error("help should mention show subcommand")
	}
	if !strings.Contains(out, "switch") {
		t.Error("help should mention switch subcommand")
	}
	if !strings.Contains(out, "repos") {
		t.Error("help should mention repos subcommand")
	}
}

// --- helpers ---

func workspaceDetailResponse() map[string]any {
	return workspaceDetailResponseWithName("ws-123", "Dev Test")
}

func workspaceDetailResponseWithName(id, name string) map[string]any {
	return map[string]any{
		"data": map[string]any{
			"workspace": map[string]any{
				"id":               id,
				"name":             name,
				"displayName":      name,
				"description":      nil,
				"private":          false,
				"createdAt":        "2026-02-06T22:27:05Z",
				"updatedAt":        "2026-02-06T22:27:05Z",
				"viewerPermission": "ADMIN",
				"isFavorite":       false,
				"zenhubOrganization": map[string]any{
					"id":   "org1",
					"name": "TestOrg",
				},
				"defaultRepository": map[string]any{
					"id":        "repo1",
					"name":      "task-tracker",
					"ownerName": "dlakehammond",
					"ghId":      12345,
				},
				"sprintConfig": map[string]any{
					"id":           "sc1",
					"name":         "Sprint",
					"kind":         "weekly",
					"period":       2,
					"startDay":     "SUNDAY",
					"endDay":       "SUNDAY",
					"tzIdentifier": "America/New_York",
				},
				"activeSprint": map[string]any{
					"id":              "sp1",
					"name":            "Sprint: Feb 8 - Feb 22, 2026",
					"generatedName":   "Sprint: Feb 8 - Feb 22, 2026",
					"state":           "OPEN",
					"startAt":         "2026-02-08T14:00:00Z",
					"endAt":           "2026-02-22T12:59:59Z",
					"totalPoints":     10,
					"completedPoints": 3,
				},
				"averageSprintVelocity": 42,
				"pipelinesConnection": map[string]any{
					"totalCount": 2,
					"nodes": []any{
						map[string]any{"id": "p1", "name": "Todo", "description": "Ready to work on"},
						map[string]any{"id": "p2", "name": "Doing", "description": "In progress"},
					},
				},
				"repositoriesConnection": map[string]any{
					"totalCount": 2,
					"nodes": []any{
						map[string]any{"id": "r1", "name": "task-tracker", "ownerName": "dlakehammond", "ghId": 12345, "isPrivate": false, "isArchived": false},
						map[string]any{"id": "r2", "name": "recipe-book", "ownerName": "dlakehammond", "ghId": 67890, "isPrivate": false, "isArchived": false},
					},
				},
				"prioritiesConnection": map[string]any{
					"nodes": []any{
						map[string]any{"id": "pri1", "name": "High priority", "color": "red"},
					},
				},
			},
		},
	}
}
