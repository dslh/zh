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

// setupEpicMutationTest configures env, mock server, and returns a cleanup function.
func setupEpicMutationTest(t *testing.T, ms *testutil.MockServer) {
	t.Helper()

	resetEpicFlags()
	resetEpicMutationFlags()

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
}

// --- epic create (ZenHub epic) ---

func TestEpicCreate(t *testing.T) {
	ms := testutil.NewMockServer(t)
	ms.HandleQuery("GetWorkspaceOrg", workspaceOrgResponse())
	ms.HandleQuery("CreateZenhubEpic", createZenhubEpicResponse())
	setupEpicMutationTest(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"epic", "create", "Q2 Platform Improvements"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("epic create returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Created epic") {
		t.Errorf("output should confirm creation, got: %s", out)
	}
	if !strings.Contains(out, "Q2 Platform Improvements") {
		t.Errorf("output should contain epic title, got: %s", out)
	}
	if !strings.Contains(out, "ZenHub Epic") {
		t.Errorf("output should show type, got: %s", out)
	}

	// Verify cache was invalidated
	_, ok := cache.Get[[]resolve.CachedEpic](resolve.EpicCacheKey("ws-123"))
	if ok {
		t.Error("epic cache should be cleared after create")
	}
}

func TestEpicCreateWithBody(t *testing.T) {
	ms := testutil.NewMockServer(t)
	ms.HandleQuery("GetWorkspaceOrg", workspaceOrgResponse())
	ms.HandleQuery("CreateZenhubEpic", createZenhubEpicResponse())
	setupEpicMutationTest(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"epic", "create", "Q2 Platform Improvements", "--body=Covers all Q2 improvements."})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("epic create with body returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Created epic") {
		t.Errorf("output should confirm creation, got: %s", out)
	}
}

func TestEpicCreateDryRun(t *testing.T) {
	ms := testutil.NewMockServer(t)
	setupEpicMutationTest(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"epic", "create", "Q2 Platform Improvements", "--body=Covers all Q2 improvements.", "--dry-run"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("epic create --dry-run returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Would create") {
		t.Errorf("dry-run should use 'Would' prefix, got: %s", out)
	}
	if !strings.Contains(out, "Q2 Platform Improvements") {
		t.Errorf("dry-run should contain epic title, got: %s", out)
	}
	if !strings.Contains(out, "ZenHub Epic") {
		t.Errorf("dry-run should show type, got: %s", out)
	}
	if !strings.Contains(out, "Covers all Q2") {
		t.Errorf("dry-run should show body, got: %s", out)
	}
}

func TestEpicCreateJSON(t *testing.T) {
	ms := testutil.NewMockServer(t)
	ms.HandleQuery("GetWorkspaceOrg", workspaceOrgResponse())
	ms.HandleQuery("CreateZenhubEpic", createZenhubEpicResponse())
	setupEpicMutationTest(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"epic", "create", "Q2 Platform Improvements", "--output=json"})
	outputFormat = "json"
	defer func() { outputFormat = "" }()

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("epic create --output=json returned error: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\nOutput: %s", err, buf.String())
	}

	if result["title"] != "Q2 Platform Improvements" {
		t.Errorf("JSON should contain title, got: %v", result)
	}
}

// --- epic create (legacy epic) ---

func TestEpicCreateLegacy(t *testing.T) {
	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListRepos", repoListForEpicResponse())
	ms.HandleQuery("CreateEpic", createLegacyEpicResponse())
	setupEpicMutationTest(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"epic", "create", "Bug Tracker Overhaul", "--repo=task-tracker"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("epic create --repo returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Created legacy epic") {
		t.Errorf("output should confirm legacy epic creation, got: %s", out)
	}
	if !strings.Contains(out, "Bug Tracker Overhaul") {
		t.Errorf("output should contain epic title, got: %s", out)
	}
	if !strings.Contains(out, "Legacy Epic") {
		t.Errorf("output should show legacy type, got: %s", out)
	}

	// Verify cache was invalidated
	_, ok := cache.Get[[]resolve.CachedEpic](resolve.EpicCacheKey("ws-123"))
	if ok {
		t.Error("epic cache should be cleared after create")
	}
}

func TestEpicCreateLegacyDryRun(t *testing.T) {
	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListRepos", repoListForEpicResponse())
	setupEpicMutationTest(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"epic", "create", "Bug Tracker Overhaul", "--repo=task-tracker", "--dry-run"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("epic create --repo --dry-run returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Would create") {
		t.Errorf("dry-run should use 'Would' prefix, got: %s", out)
	}
	if !strings.Contains(out, "Legacy Epic") {
		t.Errorf("dry-run should show legacy type, got: %s", out)
	}
	if !strings.Contains(out, "task-tracker") {
		t.Errorf("dry-run should show repo, got: %s", out)
	}
}

func TestEpicCreateLegacyJSON(t *testing.T) {
	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListRepos", repoListForEpicResponse())
	ms.HandleQuery("CreateEpic", createLegacyEpicResponse())
	setupEpicMutationTest(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"epic", "create", "Bug Tracker Overhaul", "--repo=task-tracker", "--output=json"})
	outputFormat = "json"
	defer func() { outputFormat = "" }()

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("epic create --repo --output=json returned error: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\nOutput: %s", err, buf.String())
	}
}

func TestEpicCreateHelpText(t *testing.T) {
	resetEpicFlags()
	resetEpicMutationFlags()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"epic", "create", "--help"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("epic create --help returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "body") {
		t.Error("help should mention --body flag")
	}
	if !strings.Contains(out, "repo") {
		t.Error("help should mention --repo flag")
	}
	if !strings.Contains(out, "dry-run") {
		t.Error("help should mention --dry-run flag")
	}
}

// --- helpers ---

func workspaceOrgResponse() map[string]any {
	return map[string]any{
		"data": map[string]any{
			"workspace": map[string]any{
				"zenhubOrganization": map[string]any{
					"id":   "org-123",
					"name": "Test Org",
				},
			},
		},
	}
}

func createZenhubEpicResponse() map[string]any {
	return map[string]any{
		"data": map[string]any{
			"createZenhubEpic": map[string]any{
				"zenhubEpic": map[string]any{
					"id":        "epic-new-1",
					"title":     "Q2 Platform Improvements",
					"body":      "Covers all Q2 improvements.",
					"state":     "OPEN",
					"createdAt": "2026-02-10T12:00:00Z",
				},
			},
		},
	}
}

func repoListForEpicResponse() map[string]any {
	return map[string]any{
		"data": map[string]any{
			"workspace": map[string]any{
				"repositoriesConnection": map[string]any{
					"totalCount": 1,
					"pageInfo": map[string]any{
						"hasNextPage": false,
						"endCursor":   "",
					},
					"nodes": []any{
						map[string]any{
							"id":        "repo-1",
							"ghId":      1152464818,
							"name":      "task-tracker",
							"ownerName": "dlakehammond",
						},
					},
				},
			},
		},
	}
}

func createLegacyEpicResponse() map[string]any {
	return map[string]any{
		"data": map[string]any{
			"createEpic": map[string]any{
				"epic": map[string]any{
					"id": "epic-legacy-new-1",
					"issue": map[string]any{
						"id":      "issue-new-1",
						"number":  7,
						"title":   "Bug Tracker Overhaul",
						"htmlUrl": "https://github.com/dlakehammond/task-tracker/issues/7",
						"repository": map[string]any{
							"name":      "task-tracker",
							"ownerName": "dlakehammond",
						},
					},
				},
			},
		},
	}
}
