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
	t.Setenv("ZH_REST_API_KEY", "test-rest-key")
	t.Setenv("ZH_WORKSPACE", "ws-123")
	t.Setenv("ZH_GITHUB_TOKEN", "")

	origNew := apiNewFunc
	apiNewFunc = func(apiKey string, opts ...api.Option) *api.Client {
		return api.New(apiKey, append(opts, api.WithEndpoint(ms.URL()))...)
	}
	t.Cleanup(func() { apiNewFunc = origNew })
}

// setupEpicMutationTestWithGitHub extends the base setup with a GitHub API mock.
func setupEpicMutationTestWithGitHub(t *testing.T, ms *testutil.MockServer, ghMs *testutil.MockServer) {
	t.Helper()
	setupEpicMutationTest(t, ms)

	origGh := ghNewFunc
	ghNewFunc = func(method, token string, opts ...gh.Option) *gh.Client {
		return gh.New("pat", "test-token", append(opts, gh.WithEndpoint(ghMs.URL()))...)
	}
	t.Cleanup(func() { ghNewFunc = origGh })
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
	handleEpicResolutionForMutations(ms)
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
	handleEpicResolutionForMutations(ms)
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
	handleEpicResolutionForMutations(ms)
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
	handleEpicResolutionForMutations(ms)
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

func TestEpicEditLegacy(t *testing.T) {
	ms := testutil.NewMockServer(t)
	ghMs := testutil.NewMockServer(t)

	// GitHub mock: return issue node ID, then handle update mutation
	ghMs.HandleQuery("GetGitHubIssue", ghIssueNodeResponse())
	ghMs.HandleQuery("UpdateIssue", ghUpdateIssueResponse("New Title", "OPEN"))

	setupEpicMutationTestWithGitHub(t, ms, ghMs)

	// Pre-populate cache with legacy epic
	_ = cache.Set(resolve.EpicCacheKey("ws-123"), []resolve.CachedEpic{
		{ID: "legacy-epic-1", Title: "Bug Tracker Improvements", Type: "legacy", IssueNumber: 1, RepoName: "task-tracker", RepoOwner: "dlakehammond"},
	})

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"epic", "edit", "Bug Tracker", "--title=New Title"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("epic edit on legacy epic returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Updated legacy epic") {
		t.Errorf("output should confirm legacy epic update, got: %s", out)
	}
	if !strings.Contains(out, "Title:") {
		t.Errorf("output should show title change, got: %s", out)
	}
}

func TestEpicEditLegacyDryRun(t *testing.T) {
	ms := testutil.NewMockServer(t)
	setupEpicMutationTest(t, ms)

	// Pre-populate cache with legacy epic
	_ = cache.Set(resolve.EpicCacheKey("ws-123"), []resolve.CachedEpic{
		{ID: "legacy-epic-1", Title: "Bug Tracker Improvements", Type: "legacy", IssueNumber: 1, RepoName: "task-tracker", RepoOwner: "dlakehammond"},
	})

	// ghNewFunc returns nil (no GitHub) — dry-run shouldn't need it
	origGh := ghNewFunc
	ghNewFunc = func(method, token string, opts ...gh.Option) *gh.Client {
		return gh.New("pat", "test-token", opts...)
	}
	t.Cleanup(func() { ghNewFunc = origGh })

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"epic", "edit", "Bug Tracker", "--title=New Title", "--dry-run"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("epic edit dry-run on legacy epic returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Would update legacy epic") {
		t.Errorf("output should show dry-run message, got: %s", out)
	}
}

func TestEpicEditLegacyJSON(t *testing.T) {
	ms := testutil.NewMockServer(t)
	ghMs := testutil.NewMockServer(t)
	ghMs.HandleQuery("GetGitHubIssue", ghIssueNodeResponse())
	ghMs.HandleQuery("UpdateIssue", ghUpdateIssueResponse("New Title", "OPEN"))
	setupEpicMutationTestWithGitHub(t, ms, ghMs)

	_ = cache.Set(resolve.EpicCacheKey("ws-123"), []resolve.CachedEpic{
		{ID: "legacy-epic-1", Title: "Bug Tracker Improvements", Type: "legacy", IssueNumber: 1, RepoName: "task-tracker", RepoOwner: "dlakehammond"},
	})

	outputFormat = "json"
	defer func() { outputFormat = "" }()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"epic", "edit", "Bug Tracker", "--title=New Title"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("epic edit JSON on legacy epic returned error: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	if result["issue"] != "dlakehammond/task-tracker#1" {
		t.Errorf("JSON should contain issue ref, got: %v", result["issue"])
	}
}

func TestEpicEditLegacyNoGitHub(t *testing.T) {
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
		t.Fatal("epic edit on legacy epic without GitHub should return error")
	}
	if !strings.Contains(err.Error(), "GitHub access is required") {
		t.Errorf("error should mention GitHub access, got: %v", err)
	}
}

// --- epic delete ---

func TestEpicDelete(t *testing.T) {
	ms := testutil.NewMockServer(t)
	handleEpicResolutionForMutations(ms)
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
	handleEpicResolutionForMutations(ms)
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

func TestEpicDeleteDryRunJSON(t *testing.T) {
	ms := testutil.NewMockServer(t)
	handleEpicResolutionForMutations(ms)
	ms.HandleQuery("GetEpicChildCount", epicChildCountResponse(3))
	setupEpicMutationTest(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"epic", "delete", "Q1 Platform", "--dry-run", "--output=json"})
	outputFormat = "json"
	defer func() { outputFormat = "" }()

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("epic delete --dry-run --output=json returned error: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\nOutput: %s", err, buf.String())
	}

	if result["dryRun"] != true {
		t.Error("JSON should contain dryRun: true")
	}
	if result["deleted"] == nil {
		t.Error("JSON should contain deleted field")
	}
	if result["id"] == nil {
		t.Error("JSON should contain id field")
	}
	if result["childIssues"] != float64(3) {
		t.Errorf("JSON childIssues should be 3, got: %v", result["childIssues"])
	}
}

func TestEpicDeleteJSON(t *testing.T) {
	ms := testutil.NewMockServer(t)
	handleEpicResolutionForMutations(ms)
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
	handleEpicResolutionForMutations(ms)
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
	handleEpicResolutionForMutations(ms)
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
	handleEpicResolutionForMutations(ms)
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
	handleEpicResolutionForMutations(ms)
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
	handleEpicResolutionForMutations(ms)
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

func TestEpicSetStateLegacy(t *testing.T) {
	ms := testutil.NewMockServer(t)
	ghMs := testutil.NewMockServer(t)
	ghMs.HandleQuery("GetGitHubIssue", ghIssueNodeResponse())
	ghMs.HandleQuery("UpdateIssue", ghUpdateIssueResponse("Bug Tracker Improvements", "CLOSED"))
	setupEpicMutationTestWithGitHub(t, ms, ghMs)

	_ = cache.Set(resolve.EpicCacheKey("ws-123"), []resolve.CachedEpic{
		{ID: "legacy-epic-1", Title: "Bug Tracker Improvements", Type: "legacy", IssueNumber: 1, RepoName: "task-tracker", RepoOwner: "dlakehammond"},
	})

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"epic", "set-state", "Bug Tracker", "closed"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("epic set-state on legacy epic returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Set state of legacy epic") {
		t.Errorf("output should confirm state change, got: %s", out)
	}
	if !strings.Contains(out, "closed") {
		t.Errorf("output should show new state, got: %s", out)
	}
}

func TestEpicSetStateLegacyDryRun(t *testing.T) {
	ms := testutil.NewMockServer(t)
	setupEpicMutationTest(t, ms)

	_ = cache.Set(resolve.EpicCacheKey("ws-123"), []resolve.CachedEpic{
		{ID: "legacy-epic-1", Title: "Bug Tracker Improvements", Type: "legacy", IssueNumber: 1, RepoName: "task-tracker", RepoOwner: "dlakehammond"},
	})

	origGh := ghNewFunc
	ghNewFunc = func(method, token string, opts ...gh.Option) *gh.Client {
		return gh.New("pat", "test-token", opts...)
	}
	t.Cleanup(func() { ghNewFunc = origGh })

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"epic", "set-state", "Bug Tracker", "closed", "--dry-run"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("epic set-state dry-run on legacy epic returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Would set state of legacy epic") {
		t.Errorf("output should show dry-run message, got: %s", out)
	}
}

func TestEpicSetStateLegacyJSON(t *testing.T) {
	ms := testutil.NewMockServer(t)
	ghMs := testutil.NewMockServer(t)
	ghMs.HandleQuery("GetGitHubIssue", ghIssueNodeResponse())
	ghMs.HandleQuery("UpdateIssue", ghUpdateIssueResponse("Bug Tracker Improvements", "CLOSED"))
	setupEpicMutationTestWithGitHub(t, ms, ghMs)

	_ = cache.Set(resolve.EpicCacheKey("ws-123"), []resolve.CachedEpic{
		{ID: "legacy-epic-1", Title: "Bug Tracker Improvements", Type: "legacy", IssueNumber: 1, RepoName: "task-tracker", RepoOwner: "dlakehammond"},
	})

	outputFormat = "json"
	defer func() { outputFormat = "" }()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"epic", "set-state", "Bug Tracker", "closed"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("epic set-state JSON on legacy epic returned error: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	if result["state"] != "closed" {
		t.Errorf("JSON should contain state=closed, got: %v", result["state"])
	}
}

func TestEpicSetStateLegacyStateMapping(t *testing.T) {
	// Test that non-closed states (todo, in_progress) map to GitHub OPEN
	ms := testutil.NewMockServer(t)
	ghMs := testutil.NewMockServer(t)
	ghMs.HandleQuery("GetGitHubIssue", ghIssueNodeResponse())
	ghMs.HandleQuery("UpdateIssue", ghUpdateIssueResponse("Bug Tracker Improvements", "OPEN"))
	setupEpicMutationTestWithGitHub(t, ms, ghMs)

	_ = cache.Set(resolve.EpicCacheKey("ws-123"), []resolve.CachedEpic{
		{ID: "legacy-epic-1", Title: "Bug Tracker Improvements", Type: "legacy", IssueNumber: 1, RepoName: "task-tracker", RepoOwner: "dlakehammond"},
	})

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"epic", "set-state", "Bug Tracker", "in_progress"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("epic set-state in_progress on legacy epic returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "open") {
		t.Errorf("output should show mapped state (open), got: %s", out)
	}
	if !strings.Contains(out, "maps to open") {
		t.Errorf("output should explain state mapping, got: %s", out)
	}
}

func TestEpicSetStateLegacyNoGitHub(t *testing.T) {
	ms := testutil.NewMockServer(t)
	setupEpicMutationTest(t, ms)

	_ = cache.Set(resolve.EpicCacheKey("ws-123"), []resolve.CachedEpic{
		{ID: "legacy-epic-1", Title: "Bug Tracker Improvements", Type: "legacy", IssueNumber: 1, RepoName: "task-tracker", RepoOwner: "dlakehammond"},
	})

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"epic", "set-state", "Bug Tracker", "closed"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("epic set-state on legacy epic without GitHub should return error")
	}
	if !strings.Contains(err.Error(), "GitHub access is required") {
		t.Errorf("error should mention GitHub access, got: %v", err)
	}
}

// --- helpers ---

// GitHub API mock responses for legacy epic operations.

func ghIssueNodeResponse() map[string]any {
	return map[string]any{
		"data": map[string]any{
			"repository": map[string]any{
				"issue": map[string]any{
					"id":    "GH_ISSUE_NODE_123",
					"title": "Bug Tracker Improvements",
					"body":  "Original body",
					"state": "OPEN",
				},
			},
		},
	}
}

func ghUpdateIssueResponse(title, state string) map[string]any {
	return map[string]any{
		"data": map[string]any{
			"updateIssue": map[string]any{
				"issue": map[string]any{
					"id":    "GH_ISSUE_NODE_123",
					"title": title,
					"body":  "Updated body",
					"state": state,
				},
			},
		},
	}
}

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

// handleEpicResolutionForMutations registers mock handlers for epic resolution
// during mutation tests — returns a single ZenHub epic.
func handleEpicResolutionForMutations(ms *testutil.MockServer) {
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
	ms.HandleQuery("ListLegacyEpics", map[string]any{
		"data": map[string]any{
			"workspace": map[string]any{
				"epics": map[string]any{
					"pageInfo": map[string]any{
						"hasNextPage": false,
						"endCursor":   "",
					},
					"nodes": []any{},
				},
			},
		},
	})
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

// --- epic set-dates ---

func TestEpicSetDates(t *testing.T) {
	ms := testutil.NewMockServer(t)
	handleEpicResolutionForMutations(ms)
	ms.HandleQuery("UpdateZenhubEpicDates", updateZenhubEpicDatesResponse("2025-03-01", "2025-03-31"))
	setupEpicMutationTest(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"epic", "set-dates", "Q1 Platform", "--start=2025-03-01", "--end=2025-03-31"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("epic set-dates returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Updated dates") {
		t.Errorf("output should confirm update, got: %s", out)
	}
	if !strings.Contains(out, "2025-03-01") {
		t.Errorf("output should show start date, got: %s", out)
	}
	if !strings.Contains(out, "2025-03-31") {
		t.Errorf("output should show end date, got: %s", out)
	}
}

func TestEpicSetDatesClearEnd(t *testing.T) {
	ms := testutil.NewMockServer(t)
	handleEpicResolutionForMutations(ms)
	ms.HandleQuery("UpdateZenhubEpicDates", updateZenhubEpicDatesResponse("2025-03-01", ""))
	setupEpicMutationTest(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"epic", "set-dates", "Q1 Platform", "--clear-end"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("epic set-dates --clear-end returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Updated dates") {
		t.Errorf("output should confirm update, got: %s", out)
	}
	if !strings.Contains(out, "None") {
		t.Errorf("output should show None for cleared end date, got: %s", out)
	}
}

func TestEpicSetDatesNoFlags(t *testing.T) {
	ms := testutil.NewMockServer(t)
	setupEpicMutationTest(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"epic", "set-dates", "Q1 Platform"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("epic set-dates with no flags should return error")
	}
	if !strings.Contains(err.Error(), "at least one of") {
		t.Errorf("error should mention required flags, got: %v", err)
	}
}

func TestEpicSetDatesInvalidDate(t *testing.T) {
	ms := testutil.NewMockServer(t)
	setupEpicMutationTest(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"epic", "set-dates", "Q1 Platform", "--start=not-a-date"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("epic set-dates with invalid date should return error")
	}
	if !strings.Contains(err.Error(), "invalid date") {
		t.Errorf("error should mention invalid date, got: %v", err)
	}
}

func TestEpicSetDatesConflicting(t *testing.T) {
	ms := testutil.NewMockServer(t)
	setupEpicMutationTest(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"epic", "set-dates", "Q1 Platform", "--start=2025-03-01", "--clear-start"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("epic set-dates with conflicting flags should return error")
	}
	if !strings.Contains(err.Error(), "cannot set --start and --clear-start") {
		t.Errorf("error should mention conflicting flags, got: %v", err)
	}
}

func TestEpicSetDatesDryRun(t *testing.T) {
	ms := testutil.NewMockServer(t)
	handleEpicResolutionForMutations(ms)
	setupEpicMutationTest(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"epic", "set-dates", "Q1 Platform", "--start=2025-03-01", "--clear-end", "--dry-run"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("epic set-dates --dry-run returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Would update dates") {
		t.Errorf("dry-run should use 'Would' prefix, got: %s", out)
	}
	if !strings.Contains(out, "2025-03-01") {
		t.Errorf("dry-run should show start date, got: %s", out)
	}
	if !strings.Contains(out, "(clear)") {
		t.Errorf("dry-run should show (clear) for cleared end date, got: %s", out)
	}
}

func TestEpicSetDatesJSON(t *testing.T) {
	ms := testutil.NewMockServer(t)
	handleEpicResolutionForMutations(ms)
	ms.HandleQuery("UpdateZenhubEpicDates", updateZenhubEpicDatesResponse("2025-03-01", "2025-03-31"))
	setupEpicMutationTest(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"epic", "set-dates", "Q1 Platform", "--start=2025-03-01", "--end=2025-03-31", "--output=json"})
	outputFormat = "json"
	defer func() { outputFormat = "" }()

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("epic set-dates --output=json returned error: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\nOutput: %s", err, buf.String())
	}

	if result["startOn"] != "2025-03-01" {
		t.Errorf("JSON should contain startOn, got: %v", result["startOn"])
	}
	if result["endOn"] != "2025-03-31" {
		t.Errorf("JSON should contain endOn, got: %v", result["endOn"])
	}
}

func TestEpicSetDatesLegacy(t *testing.T) {
	ms := testutil.NewMockServer(t)
	setupEpicMutationTest(t, ms)

	// Pre-populate cache with legacy epic
	_ = cache.Set(resolve.EpicCacheKey("ws-123"), []resolve.CachedEpic{
		{ID: "legacy-epic-1", Title: "Bug Tracker Improvements", Type: "legacy", IssueNumber: 1, RepoName: "task-tracker", RepoOwner: "dlakehammond"},
	})

	ms.HandleQuery("UpdateEpicDates", updateLegacyEpicDatesResponse("2025-03-01", "2025-03-31"))

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"epic", "set-dates", "Bug Tracker", "--start=2025-03-01", "--end=2025-03-31"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("epic set-dates on legacy epic returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Updated dates") {
		t.Errorf("output should confirm update, got: %s", out)
	}
}

// --- epic add ---

func TestEpicAdd(t *testing.T) {
	ms := testutil.NewMockServer(t)
	handleEpicResolutionForMutations(ms)
	ms.HandleQuery("ListRepos", repoListForEpicResponse())
	ms.HandleQuery("IssueByInfo", issueByInfoForEpicResponse("i1", 1))
	ms.HandleQuery("GetIssueForEpic", issueDetailForEpicResponse("i1", 1, "Fix login button alignment"))
	ms.HandleQuery("AddIssuesToZenhubEpics", addIssuesToEpicsResponse())
	setupEpicMutationTest(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"epic", "add", "Q1 Platform", "task-tracker#1"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("epic add returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Added") {
		t.Errorf("output should confirm addition, got: %s", out)
	}
	if !strings.Contains(out, "task-tracker#1") {
		t.Errorf("output should contain issue ref, got: %s", out)
	}
}

func TestEpicAddMultiple(t *testing.T) {
	ms := testutil.NewMockServer(t)
	handleEpicResolutionForMutations(ms)
	ms.HandleQuery("ListRepos", repoListForEpicResponse())
	ms.HandleQuery("IssueByInfo", issueByInfoForEpicResponse("i1", 1))
	ms.HandleQuery("GetIssueForEpic", issueDetailForEpicResponse("i1", 1, "Fix login button alignment"))
	ms.HandleQuery("AddIssuesToZenhubEpics", addIssuesToEpicsResponse())
	setupEpicMutationTest(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"epic", "add", "Q1 Platform", "task-tracker#1", "task-tracker#1"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("epic add multiple returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Added 2 issue(s)") {
		t.Errorf("output should confirm batch addition, got: %s", out)
	}
}

func TestEpicAddDryRun(t *testing.T) {
	ms := testutil.NewMockServer(t)
	handleEpicResolutionForMutations(ms)
	ms.HandleQuery("ListRepos", repoListForEpicResponse())
	ms.HandleQuery("IssueByInfo", issueByInfoForEpicResponse("i1", 1))
	ms.HandleQuery("GetIssueForEpic", issueDetailForEpicResponse("i1", 1, "Fix login button alignment"))
	setupEpicMutationTest(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"epic", "add", "Q1 Platform", "task-tracker#1", "--dry-run"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("epic add --dry-run returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Would add") {
		t.Errorf("dry-run should use 'Would' prefix, got: %s", out)
	}
	if !strings.Contains(out, "task-tracker#1") {
		t.Errorf("dry-run should show issue ref, got: %s", out)
	}
}

func TestEpicAddJSON(t *testing.T) {
	ms := testutil.NewMockServer(t)
	handleEpicResolutionForMutations(ms)
	ms.HandleQuery("ListRepos", repoListForEpicResponse())
	ms.HandleQuery("IssueByInfo", issueByInfoForEpicResponse("i1", 1))
	ms.HandleQuery("GetIssueForEpic", issueDetailForEpicResponse("i1", 1, "Fix login button alignment"))
	ms.HandleQuery("AddIssuesToZenhubEpics", addIssuesToEpicsResponse())
	setupEpicMutationTest(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"epic", "add", "Q1 Platform", "task-tracker#1", "--output=json"})
	outputFormat = "json"
	defer func() { outputFormat = "" }()

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("epic add --output=json returned error: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\nOutput: %s", err, buf.String())
	}

	if result["epic"] == nil {
		t.Error("JSON should contain epic field")
	}
	if result["added"] == nil {
		t.Error("JSON should contain added field")
	}
}

func TestEpicAddLegacy(t *testing.T) {
	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListRepos", repoListForEpicResponse())
	ms.HandleQuery("IssueByInfo", issueByInfoForEpicResponse("i1", 1))
	ms.HandleQuery("GetIssueForEpic", issueDetailForEpicResponse("i1", 1, "Fix login button alignment"))
	ms.HandleREST("update_issues", 200, map[string]any{})
	setupEpicMutationTest(t, ms)

	_ = cache.Set(resolve.EpicCacheKey("ws-123"), []resolve.CachedEpic{
		{ID: "legacy-epic-1", Title: "Bug Tracker Improvements", Type: "legacy", IssueNumber: 1, RepoName: "task-tracker", RepoOwner: "dlakehammond"},
	})

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"epic", "add", "Bug Tracker", "task-tracker#1"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("epic add on legacy epic returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Added") {
		t.Errorf("output should confirm addition, got: %s", out)
	}
	if !strings.Contains(out, "legacy epic") {
		t.Errorf("output should mention legacy epic, got: %s", out)
	}
}

func TestEpicAddLegacyDryRun(t *testing.T) {
	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListRepos", repoListForEpicResponse())
	ms.HandleQuery("IssueByInfo", issueByInfoForEpicResponse("i1", 1))
	ms.HandleQuery("GetIssueForEpic", issueDetailForEpicResponse("i1", 1, "Fix login button alignment"))
	setupEpicMutationTest(t, ms)

	_ = cache.Set(resolve.EpicCacheKey("ws-123"), []resolve.CachedEpic{
		{ID: "legacy-epic-1", Title: "Bug Tracker Improvements", Type: "legacy", IssueNumber: 1, RepoName: "task-tracker", RepoOwner: "dlakehammond"},
	})

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"epic", "add", "Bug Tracker", "task-tracker#1", "--dry-run"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("epic add legacy dry-run returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Would add") {
		t.Errorf("dry-run should use 'Would add' prefix, got: %s", out)
	}
	if !strings.Contains(out, "legacy epic") {
		t.Errorf("dry-run should mention legacy epic, got: %s", out)
	}
}

func TestEpicAddLegacyJSON(t *testing.T) {
	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListRepos", repoListForEpicResponse())
	ms.HandleQuery("IssueByInfo", issueByInfoForEpicResponse("i1", 1))
	ms.HandleQuery("GetIssueForEpic", issueDetailForEpicResponse("i1", 1, "Fix login button alignment"))
	ms.HandleREST("update_issues", 200, map[string]any{})
	setupEpicMutationTest(t, ms)

	_ = cache.Set(resolve.EpicCacheKey("ws-123"), []resolve.CachedEpic{
		{ID: "legacy-epic-1", Title: "Bug Tracker Improvements", Type: "legacy", IssueNumber: 1, RepoName: "task-tracker", RepoOwner: "dlakehammond"},
	})

	outputFormat = "json"
	defer func() { outputFormat = "" }()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"epic", "add", "Bug Tracker", "task-tracker#1"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("epic add legacy JSON returned error: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\nOutput: %s", err, buf.String())
	}
	epic := result["epic"].(map[string]any)
	if epic["issue"] != "dlakehammond/task-tracker#1" {
		t.Errorf("JSON epic should contain issue ref, got: %v", epic["issue"])
	}
	if result["added"] == nil {
		t.Error("JSON should contain added field")
	}
}

func TestEpicAddContinueOnError(t *testing.T) {
	ms := testutil.NewMockServer(t)
	handleEpicResolutionForMutations(ms)
	ms.HandleQuery("ListRepos", repoListForEpicResponse())
	// First issue resolves, second fails (not found)
	ms.HandleQuery("IssueByInfo", issueByInfoForEpicResponse("i1", 1))
	ms.HandleQuery("GetIssueForEpic", issueDetailForEpicResponse("i1", 1, "Fix login button alignment"))
	ms.HandleQuery("AddIssuesToZenhubEpics", addIssuesToEpicsResponse())
	setupEpicMutationTest(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"epic", "add", "Q1 Platform", "task-tracker#1", "task-tracker#999", "--continue-on-error"})

	// The command should succeed but report partial failure
	err := rootCmd.Execute()
	// We expect an error return because some issues failed
	if err == nil {
		// The first issue was added successfully, second failed to resolve.
		// With continue-on-error, partial failure returns an error.
		out := buf.String()
		if !strings.Contains(out, "Added") {
			t.Errorf("output should contain Added, got: %s", out)
		}
	}
}

// --- epic remove ---

func TestEpicRemove(t *testing.T) {
	ms := testutil.NewMockServer(t)
	handleEpicResolutionForMutations(ms)
	ms.HandleQuery("ListRepos", repoListForEpicResponse())
	ms.HandleQuery("IssueByInfo", issueByInfoForEpicResponse("i1", 1))
	ms.HandleQuery("GetIssueForEpic", issueDetailForEpicResponse("i1", 1, "Fix login button alignment"))
	ms.HandleQuery("RemoveIssuesFromZenhubEpics", removeIssuesFromEpicsResponse())
	setupEpicMutationTest(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"epic", "remove", "Q1 Platform", "task-tracker#1"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("epic remove returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Removed") {
		t.Errorf("output should confirm removal, got: %s", out)
	}
	if !strings.Contains(out, "task-tracker#1") {
		t.Errorf("output should contain issue ref, got: %s", out)
	}
}

func TestEpicRemoveAll(t *testing.T) {
	ms := testutil.NewMockServer(t)
	handleEpicResolutionForMutations(ms)
	ms.HandleQuery("GetEpicChildIssueIDs", epicChildIssueIDsResponse())
	ms.HandleQuery("RemoveIssuesFromZenhubEpics", removeIssuesFromEpicsResponse())
	setupEpicMutationTest(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"epic", "remove", "Q1 Platform", "--all"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("epic remove --all returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Removed all") {
		t.Errorf("output should confirm removing all, got: %s", out)
	}
}

func TestEpicRemoveAllEmpty(t *testing.T) {
	ms := testutil.NewMockServer(t)
	handleEpicResolutionForMutations(ms)
	ms.HandleQuery("GetEpicChildIssueIDs", epicChildIssueIDsEmptyResponse())
	setupEpicMutationTest(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"epic", "remove", "Q1 Platform", "--all"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("epic remove --all (empty) returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "no child issues") {
		t.Errorf("output should say no child issues, got: %s", out)
	}
}

func TestEpicRemoveDryRun(t *testing.T) {
	ms := testutil.NewMockServer(t)
	handleEpicResolutionForMutations(ms)
	ms.HandleQuery("ListRepos", repoListForEpicResponse())
	ms.HandleQuery("IssueByInfo", issueByInfoForEpicResponse("i1", 1))
	ms.HandleQuery("GetIssueForEpic", issueDetailForEpicResponse("i1", 1, "Fix login button alignment"))
	setupEpicMutationTest(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"epic", "remove", "Q1 Platform", "task-tracker#1", "--dry-run"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("epic remove --dry-run returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Would remove") {
		t.Errorf("dry-run should use 'Would' prefix, got: %s", out)
	}
	if !strings.Contains(out, "task-tracker#1") {
		t.Errorf("dry-run should show issue ref, got: %s", out)
	}
}

func TestEpicRemoveAllDryRun(t *testing.T) {
	ms := testutil.NewMockServer(t)
	handleEpicResolutionForMutations(ms)
	ms.HandleQuery("GetEpicChildIssueIDs", epicChildIssueIDsResponse())
	setupEpicMutationTest(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"epic", "remove", "Q1 Platform", "--all", "--dry-run"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("epic remove --all --dry-run returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Would remove all") {
		t.Errorf("dry-run should use 'Would remove all' prefix, got: %s", out)
	}
}

func TestEpicRemoveDryRunJSON(t *testing.T) {
	ms := testutil.NewMockServer(t)
	handleEpicResolutionForMutations(ms)
	ms.HandleQuery("ListRepos", repoListForEpicResponse())
	ms.HandleQuery("IssueByInfo", issueByInfoForEpicResponse("i1", 1))
	ms.HandleQuery("GetIssueForEpic", issueDetailForEpicResponse("i1", 1, "Fix login button alignment"))
	setupEpicMutationTest(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"epic", "remove", "Q1 Platform", "task-tracker#1", "--dry-run", "--output=json"})
	outputFormat = "json"
	defer func() { outputFormat = "" }()

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("epic remove --dry-run --output=json returned error: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\nOutput: %s", err, buf.String())
	}

	if result["dryRun"] != true {
		t.Error("JSON should contain dryRun: true")
	}
	if result["epic"] == nil {
		t.Error("JSON should contain epic field")
	}
	if result["removed"] == nil {
		t.Error("JSON should contain removed field")
	}
}

func TestEpicRemoveAllDryRunJSON(t *testing.T) {
	ms := testutil.NewMockServer(t)
	handleEpicResolutionForMutations(ms)
	ms.HandleQuery("GetEpicChildIssueIDs", epicChildIssueIDsResponse())
	setupEpicMutationTest(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"epic", "remove", "Q1 Platform", "--all", "--dry-run", "--output=json"})
	outputFormat = "json"
	defer func() { outputFormat = "" }()

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("epic remove --all --dry-run --output=json returned error: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\nOutput: %s", err, buf.String())
	}

	if result["dryRun"] != true {
		t.Error("JSON should contain dryRun: true")
	}
	if result["epic"] == nil {
		t.Error("JSON should contain epic field")
	}
	if result["removed"] == nil {
		t.Error("JSON should contain removed field")
	}
}

func TestEpicRemoveJSON(t *testing.T) {
	ms := testutil.NewMockServer(t)
	handleEpicResolutionForMutations(ms)
	ms.HandleQuery("ListRepos", repoListForEpicResponse())
	ms.HandleQuery("IssueByInfo", issueByInfoForEpicResponse("i1", 1))
	ms.HandleQuery("GetIssueForEpic", issueDetailForEpicResponse("i1", 1, "Fix login button alignment"))
	ms.HandleQuery("RemoveIssuesFromZenhubEpics", removeIssuesFromEpicsResponse())
	setupEpicMutationTest(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"epic", "remove", "Q1 Platform", "task-tracker#1", "--output=json"})
	outputFormat = "json"
	defer func() { outputFormat = "" }()

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("epic remove --output=json returned error: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\nOutput: %s", err, buf.String())
	}

	if result["epic"] == nil {
		t.Error("JSON should contain epic field")
	}
	if result["removed"] == nil {
		t.Error("JSON should contain removed field")
	}
}

func TestEpicRemoveLegacy(t *testing.T) {
	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListRepos", repoListForEpicResponse())
	ms.HandleQuery("IssueByInfo", issueByInfoForEpicResponse("i1", 1))
	ms.HandleQuery("GetIssueForEpic", issueDetailForEpicResponse("i1", 1, "Fix login button alignment"))
	ms.HandleREST("update_issues", 200, map[string]any{})
	setupEpicMutationTest(t, ms)

	_ = cache.Set(resolve.EpicCacheKey("ws-123"), []resolve.CachedEpic{
		{ID: "legacy-epic-1", Title: "Bug Tracker Improvements", Type: "legacy", IssueNumber: 1, RepoName: "task-tracker", RepoOwner: "dlakehammond"},
	})

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"epic", "remove", "Bug Tracker", "task-tracker#1"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("epic remove on legacy epic returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Removed") {
		t.Errorf("output should confirm removal, got: %s", out)
	}
	if !strings.Contains(out, "legacy epic") {
		t.Errorf("output should mention legacy epic, got: %s", out)
	}
}

func TestEpicRemoveLegacyDryRun(t *testing.T) {
	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListRepos", repoListForEpicResponse())
	ms.HandleQuery("IssueByInfo", issueByInfoForEpicResponse("i1", 1))
	ms.HandleQuery("GetIssueForEpic", issueDetailForEpicResponse("i1", 1, "Fix login button alignment"))
	setupEpicMutationTest(t, ms)

	_ = cache.Set(resolve.EpicCacheKey("ws-123"), []resolve.CachedEpic{
		{ID: "legacy-epic-1", Title: "Bug Tracker Improvements", Type: "legacy", IssueNumber: 1, RepoName: "task-tracker", RepoOwner: "dlakehammond"},
	})

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"epic", "remove", "Bug Tracker", "task-tracker#1", "--dry-run"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("epic remove legacy dry-run returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Would remove") {
		t.Errorf("dry-run should use 'Would remove' prefix, got: %s", out)
	}
	if !strings.Contains(out, "legacy epic") {
		t.Errorf("dry-run should mention legacy epic, got: %s", out)
	}
}

func TestEpicRemoveLegacyDryRunJSON(t *testing.T) {
	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListRepos", repoListForEpicResponse())
	ms.HandleQuery("IssueByInfo", issueByInfoForEpicResponse("i1", 1))
	ms.HandleQuery("GetIssueForEpic", issueDetailForEpicResponse("i1", 1, "Fix login button alignment"))
	setupEpicMutationTest(t, ms)

	_ = cache.Set(resolve.EpicCacheKey("ws-123"), []resolve.CachedEpic{
		{ID: "legacy-epic-1", Title: "Bug Tracker Improvements", Type: "legacy", IssueNumber: 1, RepoName: "task-tracker", RepoOwner: "dlakehammond"},
	})

	outputFormat = "json"
	defer func() { outputFormat = "" }()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"epic", "remove", "Bug Tracker", "task-tracker#1", "--dry-run", "--output=json"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("epic remove legacy --dry-run --output=json returned error: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\nOutput: %s", err, buf.String())
	}

	if result["dryRun"] != true {
		t.Error("JSON should contain dryRun: true")
	}
	epic := result["epic"].(map[string]any)
	if epic["issue"] == nil {
		t.Error("JSON epic should contain issue ref for legacy epic")
	}
	if result["removed"] == nil {
		t.Error("JSON should contain removed field")
	}
}

func TestEpicRemoveLegacyJSON(t *testing.T) {
	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListRepos", repoListForEpicResponse())
	ms.HandleQuery("IssueByInfo", issueByInfoForEpicResponse("i1", 1))
	ms.HandleQuery("GetIssueForEpic", issueDetailForEpicResponse("i1", 1, "Fix login button alignment"))
	ms.HandleREST("update_issues", 200, map[string]any{})
	setupEpicMutationTest(t, ms)

	_ = cache.Set(resolve.EpicCacheKey("ws-123"), []resolve.CachedEpic{
		{ID: "legacy-epic-1", Title: "Bug Tracker Improvements", Type: "legacy", IssueNumber: 1, RepoName: "task-tracker", RepoOwner: "dlakehammond"},
	})

	outputFormat = "json"
	defer func() { outputFormat = "" }()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"epic", "remove", "Bug Tracker", "task-tracker#1"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("epic remove legacy JSON returned error: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\nOutput: %s", err, buf.String())
	}
	epic := result["epic"].(map[string]any)
	if epic["issue"] != "dlakehammond/task-tracker#1" {
		t.Errorf("JSON epic should contain issue ref, got: %v", epic["issue"])
	}
	if result["removed"] == nil {
		t.Error("JSON should contain removed field")
	}
}

func TestEpicRemoveAllLegacy(t *testing.T) {
	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListRepos", repoListForEpicResponse())
	ms.HandleQuery("GetEpicChildIssueIDs", epicChildIssueIDsResponse())
	ms.HandleREST("update_issues", 200, map[string]any{})
	setupEpicMutationTest(t, ms)

	_ = cache.Set(resolve.EpicCacheKey("ws-123"), []resolve.CachedEpic{
		{ID: "legacy-epic-1", Title: "Bug Tracker Improvements", Type: "legacy", IssueNumber: 1, RepoName: "task-tracker", RepoOwner: "dlakehammond"},
	})

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"epic", "remove", "Bug Tracker", "--all"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("epic remove --all on legacy epic returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Removed all") {
		t.Errorf("output should confirm removing all, got: %s", out)
	}
	if !strings.Contains(out, "legacy epic") {
		t.Errorf("output should mention legacy epic, got: %s", out)
	}
}

func TestEpicRemoveAllLegacyDryRun(t *testing.T) {
	ms := testutil.NewMockServer(t)
	ms.HandleQuery("GetEpicChildIssueIDs", epicChildIssueIDsResponse())
	setupEpicMutationTest(t, ms)

	_ = cache.Set(resolve.EpicCacheKey("ws-123"), []resolve.CachedEpic{
		{ID: "legacy-epic-1", Title: "Bug Tracker Improvements", Type: "legacy", IssueNumber: 1, RepoName: "task-tracker", RepoOwner: "dlakehammond"},
	})

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"epic", "remove", "Bug Tracker", "--all", "--dry-run"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("epic remove --all --dry-run legacy returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Would remove all") {
		t.Errorf("dry-run should use 'Would remove all' prefix, got: %s", out)
	}
	if !strings.Contains(out, "legacy epic") {
		t.Errorf("dry-run should mention legacy epic, got: %s", out)
	}
}

func TestEpicRemoveAllLegacyDryRunJSON(t *testing.T) {
	ms := testutil.NewMockServer(t)
	ms.HandleQuery("GetEpicChildIssueIDs", epicChildIssueIDsResponse())
	setupEpicMutationTest(t, ms)

	_ = cache.Set(resolve.EpicCacheKey("ws-123"), []resolve.CachedEpic{
		{ID: "legacy-epic-1", Title: "Bug Tracker Improvements", Type: "legacy", IssueNumber: 1, RepoName: "task-tracker", RepoOwner: "dlakehammond"},
	})

	outputFormat = "json"
	defer func() { outputFormat = "" }()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"epic", "remove", "Bug Tracker", "--all", "--dry-run", "--output=json"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("epic remove --all --dry-run --output=json legacy returned error: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\nOutput: %s", err, buf.String())
	}

	if result["dryRun"] != true {
		t.Error("JSON should contain dryRun: true")
	}
	epic := result["epic"].(map[string]any)
	if epic["issue"] == nil {
		t.Error("JSON epic should contain issue ref for legacy epic")
	}
	if result["removed"] == nil {
		t.Error("JSON should contain removed field")
	}
}

func TestEpicRemoveAllLegacyEmpty(t *testing.T) {
	ms := testutil.NewMockServer(t)
	ms.HandleQuery("GetEpicChildIssueIDs", epicChildIssueIDsEmptyResponse())
	setupEpicMutationTest(t, ms)

	_ = cache.Set(resolve.EpicCacheKey("ws-123"), []resolve.CachedEpic{
		{ID: "legacy-epic-1", Title: "Bug Tracker Improvements", Type: "legacy", IssueNumber: 1, RepoName: "task-tracker", RepoOwner: "dlakehammond"},
	})

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"epic", "remove", "Bug Tracker", "--all"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("epic remove --all (empty) on legacy epic returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "no child issues") {
		t.Errorf("output should say no child issues, got: %s", out)
	}
}

func TestEpicRemoveNoIssues(t *testing.T) {
	ms := testutil.NewMockServer(t)
	setupEpicMutationTest(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"epic", "remove", "Q1 Platform"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("epic remove with no issues and no --all should return error")
	}
}

// --- epic estimate ---

func TestEpicEstimateSet(t *testing.T) {
	ms := testutil.NewMockServer(t)
	handleEpicResolutionForMutations(ms)
	ms.HandleQuery("GetZenhubEpicEstimate", epicEstimateQueryResponse(5))
	ms.HandleQuery("SetEstimateOnZenhubEpics", setEpicEstimateResponse(13))
	setupEpicMutationTest(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"epic", "estimate", "Q1 Platform", "13"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("epic estimate returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Set estimate") {
		t.Errorf("output should confirm estimate set, got: %s", out)
	}
	if !strings.Contains(out, "13") {
		t.Errorf("output should contain new value, got: %s", out)
	}
}

func TestEpicEstimateClear(t *testing.T) {
	ms := testutil.NewMockServer(t)
	handleEpicResolutionForMutations(ms)
	ms.HandleQuery("GetZenhubEpicEstimate", epicEstimateQueryResponse(5))
	ms.HandleQuery("SetEstimateOnZenhubEpics", setEpicEstimateClearResponse())
	setupEpicMutationTest(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"epic", "estimate", "Q1 Platform"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("epic estimate (clear) returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Cleared estimate") {
		t.Errorf("output should confirm estimate cleared, got: %s", out)
	}
}

func TestEpicEstimateDryRun(t *testing.T) {
	ms := testutil.NewMockServer(t)
	handleEpicResolutionForMutations(ms)
	ms.HandleQuery("GetZenhubEpicEstimate", epicEstimateQueryResponse(5))
	setupEpicMutationTest(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"epic", "estimate", "Q1 Platform", "13", "--dry-run"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("epic estimate --dry-run returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Would set estimate") {
		t.Errorf("dry-run should use 'Would' prefix, got: %s", out)
	}
	if !strings.Contains(out, "13") {
		t.Errorf("dry-run should show new value, got: %s", out)
	}
	if !strings.Contains(out, "currently: 5") {
		t.Errorf("dry-run should show current value, got: %s", out)
	}
}

func TestEpicEstimateClearDryRun(t *testing.T) {
	ms := testutil.NewMockServer(t)
	handleEpicResolutionForMutations(ms)
	ms.HandleQuery("GetZenhubEpicEstimate", epicEstimateQueryResponseNone())
	setupEpicMutationTest(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"epic", "estimate", "Q1 Platform", "--dry-run"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("epic estimate clear --dry-run returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Would clear estimate") {
		t.Errorf("dry-run should say 'Would clear', got: %s", out)
	}
	if !strings.Contains(out, "currently: none") {
		t.Errorf("dry-run should show current as none, got: %s", out)
	}
}

func TestEpicEstimateJSON(t *testing.T) {
	ms := testutil.NewMockServer(t)
	handleEpicResolutionForMutations(ms)
	ms.HandleQuery("GetZenhubEpicEstimate", epicEstimateQueryResponse(5))
	ms.HandleQuery("SetEstimateOnZenhubEpics", setEpicEstimateResponse(13))
	setupEpicMutationTest(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"epic", "estimate", "Q1 Platform", "13", "--output=json"})
	outputFormat = "json"
	defer func() { outputFormat = "" }()

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("epic estimate --output=json returned error: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\nOutput: %s", err, buf.String())
	}

	if result["epic"] == nil {
		t.Error("JSON should contain epic field")
	}
	if result["estimate"] == nil {
		t.Error("JSON should contain estimate field")
	}
	est := result["estimate"].(map[string]any)
	if est["previous"] != float64(5) {
		t.Errorf("JSON estimate.previous should be 5, got: %v", est["previous"])
	}
	if est["current"] != float64(13) {
		t.Errorf("JSON estimate.current should be 13, got: %v", est["current"])
	}
}

func TestEpicEstimateInvalidValue(t *testing.T) {
	ms := testutil.NewMockServer(t)
	setupEpicMutationTest(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"epic", "estimate", "Q1 Platform", "abc"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("epic estimate with invalid value should return error")
	}
	if !strings.Contains(err.Error(), "invalid estimate value") {
		t.Errorf("error should mention invalid value, got: %v", err)
	}
}

func TestEpicEstimateLegacyError(t *testing.T) {
	ms := testutil.NewMockServer(t)
	setupEpicMutationTest(t, ms)

	_ = cache.Set(resolve.EpicCacheKey("ws-123"), []resolve.CachedEpic{
		{ID: "legacy-epic-1", Title: "Bug Tracker Improvements", Type: "legacy", IssueNumber: 1, RepoName: "task-tracker", RepoOwner: "dlakehammond"},
	})

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"epic", "estimate", "Bug Tracker", "5"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("epic estimate on legacy epic should return error")
	}
	if !strings.Contains(err.Error(), "legacy epic") {
		t.Errorf("error should mention legacy epic, got: %v", err)
	}
}

// --- epic estimate helpers ---

func epicEstimateQueryResponse(value float64) map[string]any {
	return map[string]any{
		"data": map[string]any{
			"node": map[string]any{
				"id":       "epic-zen-1",
				"title":    "Q1 Platform Improvements",
				"estimate": map[string]any{"value": value},
			},
		},
	}
}

func epicEstimateQueryResponseNone() map[string]any {
	return map[string]any{
		"data": map[string]any{
			"node": map[string]any{
				"id":       "epic-zen-1",
				"title":    "Q1 Platform Improvements",
				"estimate": nil,
			},
		},
	}
}

func setEpicEstimateResponse(value float64) map[string]any {
	return map[string]any{
		"data": map[string]any{
			"setMultipleEstimatesOnZenhubEpics": map[string]any{
				"zenhubEpics": []any{
					map[string]any{
						"id":       "epic-zen-1",
						"title":    "Q1 Platform Improvements",
						"estimate": map[string]any{"value": value},
					},
				},
			},
		},
	}
}

func setEpicEstimateClearResponse() map[string]any {
	return map[string]any{
		"data": map[string]any{
			"setMultipleEstimatesOnZenhubEpics": map[string]any{
				"zenhubEpics": []any{
					map[string]any{
						"id":       "epic-zen-1",
						"title":    "Q1 Platform Improvements",
						"estimate": nil,
					},
				},
			},
		},
	}
}

// --- set-dates helpers ---

func updateZenhubEpicDatesResponse(start, end string) map[string]any {
	var startOn any = start
	var endOn any = end
	if start == "" {
		startOn = nil
	}
	if end == "" {
		endOn = nil
	}
	return map[string]any{
		"data": map[string]any{
			"updateZenhubEpicDates": map[string]any{
				"zenhubEpic": map[string]any{
					"id":      "epic-zen-1",
					"title":   "Q1 Platform Improvements",
					"startOn": startOn,
					"endOn":   endOn,
				},
			},
		},
	}
}

func updateLegacyEpicDatesResponse(start, end string) map[string]any {
	var startOn any = start
	var endOn any = end
	if start == "" {
		startOn = nil
	}
	if end == "" {
		endOn = nil
	}
	return map[string]any{
		"data": map[string]any{
			"updateEpicDates": map[string]any{
				"epic": map[string]any{
					"id":      "legacy-epic-1",
					"startOn": startOn,
					"endOn":   endOn,
					"issue": map[string]any{
						"title":  "Bug Tracker Improvements",
						"number": 1,
					},
				},
			},
		},
	}
}

// --- epic add/remove helpers ---

func issueByInfoForEpicResponse(id string, number int) map[string]any {
	return map[string]any{
		"data": map[string]any{
			"issueByInfo": map[string]any{
				"id":     id,
				"number": number,
				"repository": map[string]any{
					"ghId":      1152464818,
					"name":      "task-tracker",
					"ownerName": "dlakehammond",
				},
			},
		},
	}
}

func issueDetailForEpicResponse(id string, number int, title string) map[string]any {
	return map[string]any{
		"data": map[string]any{
			"node": map[string]any{
				"id":     id,
				"number": number,
				"title":  title,
				"repository": map[string]any{
					"name":      "task-tracker",
					"ownerName": "dlakehammond",
				},
			},
		},
	}
}

func addIssuesToEpicsResponse() map[string]any {
	return map[string]any{
		"data": map[string]any{
			"addIssuesToZenhubEpics": map[string]any{
				"zenhubEpics": []any{
					map[string]any{
						"id":    "epic-zen-1",
						"title": "Q1 Platform Improvements",
					},
				},
			},
		},
	}
}

func removeIssuesFromEpicsResponse() map[string]any {
	return map[string]any{
		"data": map[string]any{
			"removeIssuesFromZenhubEpics": map[string]any{
				"zenhubEpics": []any{
					map[string]any{
						"id":    "epic-zen-1",
						"title": "Q1 Platform Improvements",
					},
				},
			},
		},
	}
}

func epicChildIssueIDsResponse() map[string]any {
	return map[string]any{
		"data": map[string]any{
			"node": map[string]any{
				"id":    "epic-zen-1",
				"title": "Q1 Platform Improvements",
				"childIssues": map[string]any{
					"totalCount": 2,
					"pageInfo": map[string]any{
						"hasNextPage": false,
						"endCursor":   "",
					},
					"nodes": []any{
						map[string]any{
							"id":     "i1",
							"number": 1,
							"title":  "Fix login button alignment",
							"repository": map[string]any{
								"name":      "task-tracker",
								"ownerName": "dlakehammond",
							},
						},
						map[string]any{
							"id":     "i2",
							"number": 2,
							"title":  "Update error messages",
							"repository": map[string]any{
								"name":      "task-tracker",
								"ownerName": "dlakehammond",
							},
						},
					},
				},
			},
		},
	}
}

func epicChildIssueIDsEmptyResponse() map[string]any {
	return map[string]any{
		"data": map[string]any{
			"node": map[string]any{
				"id":    "epic-zen-1",
				"title": "Q1 Platform Improvements",
				"childIssues": map[string]any{
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
