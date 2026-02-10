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

// --- epic edit ---

func TestEpicEdit(t *testing.T) {
	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListEpics", epicResolutionResponseForMutations())
	ms.HandleQuery("UpdateZenhubEpic", updateZenhubEpicResponse())
	setupEpicMutationTest(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"epic", "edit", "Q1 Platform", "--title=Q1 Platform Improvements v2"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("epic edit returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Updated epic") {
		t.Errorf("output should confirm update, got: %s", out)
	}
	if !strings.Contains(out, "Title:") {
		t.Errorf("output should show updated title, got: %s", out)
	}

	// Verify cache was invalidated
	_, ok := cache.Get[[]resolve.CachedEpic](resolve.EpicCacheKey("ws-123"))
	if ok {
		t.Error("epic cache should be cleared after edit")
	}
}

func TestEpicEditBody(t *testing.T) {
	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListEpics", epicResolutionResponseForMutations())
	ms.HandleQuery("UpdateZenhubEpic", updateZenhubEpicResponse())
	setupEpicMutationTest(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"epic", "edit", "Q1 Platform", "--body=Updated description"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("epic edit --body returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Updated epic") {
		t.Errorf("output should confirm update, got: %s", out)
	}
	if !strings.Contains(out, "Body:") {
		t.Errorf("output should mention body update, got: %s", out)
	}
}

func TestEpicEditNoFlags(t *testing.T) {
	ms := testutil.NewMockServer(t)
	setupEpicMutationTest(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"epic", "edit", "Q1 Platform"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("epic edit with no flags should return error")
	}
	if !strings.Contains(err.Error(), "at least one of --title or --body") {
		t.Errorf("error should mention required flags, got: %v", err)
	}
}

func TestEpicEditDryRun(t *testing.T) {
	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListEpics", epicResolutionResponseForMutations())
	setupEpicMutationTest(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"epic", "edit", "Q1 Platform", "--title=New Title", "--body=New body", "--dry-run"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("epic edit --dry-run returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Would update") {
		t.Errorf("dry-run should use 'Would' prefix, got: %s", out)
	}
	if !strings.Contains(out, "Title:") {
		t.Errorf("dry-run should show title change, got: %s", out)
	}
	if !strings.Contains(out, "Body:") {
		t.Errorf("dry-run should show body change, got: %s", out)
	}
}

func TestEpicEditJSON(t *testing.T) {
	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListEpics", epicResolutionResponseForMutations())
	ms.HandleQuery("UpdateZenhubEpic", updateZenhubEpicResponse())
	setupEpicMutationTest(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"epic", "edit", "Q1 Platform", "--title=New Title", "--output=json"})
	outputFormat = "json"
	defer func() { outputFormat = "" }()

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("epic edit --output=json returned error: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\nOutput: %s", err, buf.String())
	}

	if result["title"] == nil {
		t.Error("JSON should contain title")
	}
}

func TestEpicEditLegacyError(t *testing.T) {
	ms := testutil.NewMockServer(t)
	setupEpicMutationTest(t, ms)

	// Pre-populate cache with legacy epic
	_ = cache.Set(resolve.EpicCacheKey("ws-123"), []resolve.CachedEpic{
		{ID: "legacy-epic-1", Title: "Bug Tracker Improvements", Type: "legacy", IssueNumber: 1, RepoName: "task-tracker", RepoOwner: "dlakehammond"},
	})

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"epic", "edit", "Bug Tracker", "--title=New Title"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("epic edit on legacy epic should return error")
	}
	if !strings.Contains(err.Error(), "legacy epic") {
		t.Errorf("error should mention legacy epic, got: %v", err)
	}
}

// --- epic delete ---

func TestEpicDelete(t *testing.T) {
	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListEpics", epicResolutionResponseForMutations())
	ms.HandleQuery("GetEpicChildCount", epicChildCountResponse(5))
	ms.HandleQuery("DeleteZenhubEpic", deleteZenhubEpicResponse())
	setupEpicMutationTest(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"epic", "delete", "Q1 Platform"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("epic delete returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Deleted epic") {
		t.Errorf("output should confirm deletion, got: %s", out)
	}
	if !strings.Contains(out, "Q1 Platform") {
		t.Errorf("output should contain epic title, got: %s", out)
	}
	if !strings.Contains(out, "5 child issue(s)") {
		t.Errorf("output should mention child issue count, got: %s", out)
	}

	// Verify cache was invalidated
	_, ok := cache.Get[[]resolve.CachedEpic](resolve.EpicCacheKey("ws-123"))
	if ok {
		t.Error("epic cache should be cleared after delete")
	}
}

func TestEpicDeleteDryRun(t *testing.T) {
	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListEpics", epicResolutionResponseForMutations())
	ms.HandleQuery("GetEpicChildCount", epicChildCountResponse(3))
	setupEpicMutationTest(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"epic", "delete", "Q1 Platform", "--dry-run"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("epic delete --dry-run returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Would delete") {
		t.Errorf("dry-run should use 'Would' prefix, got: %s", out)
	}
	if !strings.Contains(out, "Q1 Platform") {
		t.Errorf("dry-run should contain epic title, got: %s", out)
	}
	if !strings.Contains(out, "Child issues: 3") {
		t.Errorf("dry-run should show child issue count, got: %s", out)
	}
}

func TestEpicDeleteJSON(t *testing.T) {
	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListEpics", epicResolutionResponseForMutations())
	ms.HandleQuery("GetEpicChildCount", epicChildCountResponse(2))
	ms.HandleQuery("DeleteZenhubEpic", deleteZenhubEpicResponse())
	setupEpicMutationTest(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"epic", "delete", "Q1 Platform", "--output=json"})
	outputFormat = "json"
	defer func() { outputFormat = "" }()

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("epic delete --output=json returned error: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\nOutput: %s", err, buf.String())
	}

	if result["deleted"] == nil {
		t.Error("JSON should contain deleted field")
	}
}

func TestEpicDeleteLegacyError(t *testing.T) {
	ms := testutil.NewMockServer(t)
	setupEpicMutationTest(t, ms)

	// Pre-populate cache with legacy epic
	_ = cache.Set(resolve.EpicCacheKey("ws-123"), []resolve.CachedEpic{
		{ID: "legacy-epic-1", Title: "Bug Tracker Improvements", Type: "legacy", IssueNumber: 1, RepoName: "task-tracker", RepoOwner: "dlakehammond"},
	})

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"epic", "delete", "Bug Tracker"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("epic delete on legacy epic should return error")
	}
	if !strings.Contains(err.Error(), "legacy epic") {
		t.Errorf("error should mention legacy epic, got: %v", err)
	}
}

// --- epic set-state ---

func TestEpicSetState(t *testing.T) {
	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListEpics", epicResolutionResponseForMutations())
	ms.HandleQuery("UpdateZenhubEpicState", updateZenhubEpicStateResponse("CLOSED"))
	setupEpicMutationTest(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"epic", "set-state", "Q1 Platform", "closed"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("epic set-state returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Set state") {
		t.Errorf("output should confirm state change, got: %s", out)
	}
	if !strings.Contains(out, "closed") {
		t.Errorf("output should contain new state, got: %s", out)
	}

	// Verify cache was invalidated
	_, ok := cache.Get[[]resolve.CachedEpic](resolve.EpicCacheKey("ws-123"))
	if ok {
		t.Error("epic cache should be cleared after set-state")
	}
}

func TestEpicSetStateInProgress(t *testing.T) {
	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListEpics", epicResolutionResponseForMutations())
	ms.HandleQuery("UpdateZenhubEpicState", updateZenhubEpicStateResponse("IN_PROGRESS"))
	setupEpicMutationTest(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"epic", "set-state", "Q1 Platform", "in_progress"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("epic set-state in_progress returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Set state") {
		t.Errorf("output should confirm state change, got: %s", out)
	}
}

func TestEpicSetStateInvalidState(t *testing.T) {
	ms := testutil.NewMockServer(t)
	setupEpicMutationTest(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"epic", "set-state", "Q1 Platform", "invalid"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("epic set-state with invalid state should return error")
	}
	if !strings.Contains(err.Error(), "invalid state") {
		t.Errorf("error should mention invalid state, got: %v", err)
	}
}

func TestEpicSetStateDryRun(t *testing.T) {
	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListEpics", epicResolutionResponseForMutations())
	setupEpicMutationTest(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"epic", "set-state", "Q1 Platform", "todo", "--dry-run"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("epic set-state --dry-run returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Would set state") {
		t.Errorf("dry-run should use 'Would' prefix, got: %s", out)
	}
	if !strings.Contains(out, "todo") {
		t.Errorf("dry-run should contain target state, got: %s", out)
	}
}

func TestEpicSetStateApplyToIssues(t *testing.T) {
	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListEpics", epicResolutionResponseForMutations())
	ms.HandleQuery("UpdateZenhubEpicState", updateZenhubEpicStateResponse("CLOSED"))
	setupEpicMutationTest(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"epic", "set-state", "Q1 Platform", "closed", "--apply-to-issues"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("epic set-state --apply-to-issues returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Set state") {
		t.Errorf("output should confirm state change, got: %s", out)
	}
	if !strings.Contains(out, "child issues") {
		t.Errorf("output should mention child issues, got: %s", out)
	}
}

func TestEpicSetStateJSON(t *testing.T) {
	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListEpics", epicResolutionResponseForMutations())
	ms.HandleQuery("UpdateZenhubEpicState", updateZenhubEpicStateResponse("CLOSED"))
	setupEpicMutationTest(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"epic", "set-state", "Q1 Platform", "closed", "--output=json"})
	outputFormat = "json"
	defer func() { outputFormat = "" }()

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("epic set-state --output=json returned error: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\nOutput: %s", err, buf.String())
	}

	if result["state"] != "CLOSED" {
		t.Errorf("JSON should contain state CLOSED, got: %v", result["state"])
	}
}

func TestEpicSetStateLegacyError(t *testing.T) {
	ms := testutil.NewMockServer(t)
	setupEpicMutationTest(t, ms)

	// Pre-populate cache with legacy epic
	_ = cache.Set(resolve.EpicCacheKey("ws-123"), []resolve.CachedEpic{
		{ID: "legacy-epic-1", Title: "Bug Tracker Improvements", Type: "legacy", IssueNumber: 1, RepoName: "task-tracker", RepoOwner: "dlakehammond"},
	})

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"epic", "set-state", "Bug Tracker", "closed"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("epic set-state on legacy epic should return error")
	}
	if !strings.Contains(err.Error(), "legacy epic") {
		t.Errorf("error should mention legacy epic, got: %v", err)
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

func epicResolutionResponseForMutations() map[string]any {
	return map[string]any{
		"data": map[string]any{
			"workspace": map[string]any{
				"roadmap": map[string]any{
					"items": map[string]any{
						"totalCount": 1,
						"pageInfo": map[string]any{
							"hasNextPage": false,
							"endCursor":   "",
						},
						"nodes": []any{
							map[string]any{
								"__typename": "ZenhubEpic",
								"id":         "epic-zen-1",
								"title":      "Q1 Platform Improvements",
							},
						},
					},
				},
			},
		},
	}
}

func updateZenhubEpicResponse() map[string]any {
	return map[string]any{
		"data": map[string]any{
			"updateZenhubEpic": map[string]any{
				"zenhubEpic": map[string]any{
					"id":        "epic-zen-1",
					"title":     "Q1 Platform Improvements v2",
					"body":      "Updated description",
					"state":     "OPEN",
					"updatedAt": "2026-02-10T14:00:00Z",
				},
			},
		},
	}
}

func deleteZenhubEpicResponse() map[string]any {
	return map[string]any{
		"data": map[string]any{
			"deleteZenhubEpic": map[string]any{
				"zenhubEpicId": "epic-zen-1",
			},
		},
	}
}

func epicChildCountResponse(count int) map[string]any {
	return map[string]any{
		"data": map[string]any{
			"node": map[string]any{
				"id":    "epic-zen-1",
				"title": "Q1 Platform Improvements",
				"state": "OPEN",
				"childIssues": map[string]any{
					"totalCount": count,
				},
			},
		},
	}
}

func updateZenhubEpicStateResponse(state string) map[string]any {
	return map[string]any{
		"data": map[string]any{
			"updateZenhubEpicState": map[string]any{
				"zenhubEpic": map[string]any{
					"id":    "epic-zen-1",
					"title": "Q1 Platform Improvements",
					"state": state,
				},
			},
		},
	}
}
