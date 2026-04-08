package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/dslh/zh/internal/api"
	"github.com/dslh/zh/internal/cache"
	"github.com/dslh/zh/internal/gh"
	"github.com/dslh/zh/internal/resolve"
	"github.com/dslh/zh/internal/testutil"
)

func setupWorkspaceRepoMutationTest(t *testing.T, ms *testutil.MockServer) {
	t.Helper()

	resetWorkspaceRepoMutationFlags()

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

func setupWorkspaceRepoMutationTestWithGitHub(t *testing.T, ms, ghMs *testutil.MockServer) {
	t.Helper()
	setupWorkspaceRepoMutationTest(t, ms)

	origGh := ghNewFunc
	ghNewFunc = func(method, token string, opts ...gh.Option) *gh.Client {
		return gh.New("pat", "test-token", append(opts, gh.WithEndpoint(ghMs.URL()))...)
	}
	t.Cleanup(func() { ghNewFunc = origGh })
}

func emptyMutationResponse(field string) map[string]any {
	return map[string]any{
		"data": map[string]any{
			field: map[string]any{
				"clientMutationId": nil,
			},
		},
	}
}

// --- workspace repo add ---

func TestWorkspaceRepoAddByGhID(t *testing.T) {
	ms := testutil.NewMockServer(t)
	ms.HandleQuery("AddRepositoryToWorkspace", emptyMutationResponse("addRepositoryToWorkspace"))
	setupWorkspaceRepoMutationTest(t, ms)

	// Pre-seed the repo cache so we can verify it gets cleared.
	_ = cache.Set(resolve.RepoCacheKey("ws-123"), []resolve.CachedRepo{
		{ID: "r1", GhID: 111, Name: "existing", OwnerName: "myorg"},
	})

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"workspace", "repo", "add", "myorg/newrepo", "--gh-id", "999"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("workspace repo add returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Added") || !strings.Contains(out, "myorg/newrepo") {
		t.Errorf("output should confirm add, got: %s", out)
	}

	if _, ok := cache.Get[[]resolve.CachedRepo](resolve.RepoCacheKey("ws-123")); ok {
		t.Error("repo cache should be cleared after add")
	}
}

func TestWorkspaceRepoAddByOwnerRepo(t *testing.T) {
	ms := testutil.NewMockServer(t)
	ms.HandleQuery("AddRepositoryToWorkspace", emptyMutationResponse("addRepositoryToWorkspace"))

	ghMs := testutil.NewMockServer(t)
	ghMs.HandleQuery("LookupRepository", map[string]any{
		"data": map[string]any{
			"repository": map[string]any{
				"databaseId":    424242,
				"nameWithOwner": "myorg/newrepo",
			},
		},
	})

	setupWorkspaceRepoMutationTestWithGitHub(t, ms, ghMs)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"workspace", "repo", "add", "myorg/newrepo"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("workspace repo add returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Added") || !strings.Contains(out, "myorg/newrepo") {
		t.Errorf("output should confirm add, got: %s", out)
	}
}

func TestWorkspaceRepoAddDryRun(t *testing.T) {
	ms := testutil.NewMockServer(t)
	// No mutation handler — dry-run must not call the API.
	setupWorkspaceRepoMutationTest(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"workspace", "repo", "add", "myorg/newrepo", "--gh-id", "555", "--dry-run"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("workspace repo add --dry-run returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Would add") {
		t.Errorf("dry-run should use 'Would add' prefix, got: %s", out)
	}
	if !strings.Contains(out, "555") {
		t.Errorf("dry-run should show GitHub ID, got: %s", out)
	}
}

func TestWorkspaceRepoAddRequiresOwnerRepoFormat(t *testing.T) {
	ms := testutil.NewMockServer(t)
	setupWorkspaceRepoMutationTest(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"workspace", "repo", "add", "justrepo"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for non-owner/repo arg without --gh-id")
	}
	if !strings.Contains(err.Error(), "owner/repo") {
		t.Errorf("error should mention owner/repo format, got: %v", err)
	}
}

// --- workspace repo remove ---

func TestWorkspaceRepoRemove(t *testing.T) {
	ms := testutil.NewMockServer(t)
	ms.HandleQuery("DisconnectWorkspaceRepository", emptyMutationResponse("disconnectWorkspaceRepository"))
	setupWorkspaceRepoMutationTest(t, ms)

	// Pre-seed cache so resolve.LookupRepoWithRefresh finds the repo without an API call.
	_ = cache.Set(resolve.RepoCacheKey("ws-123"), []resolve.CachedRepo{
		{ID: "r1", GhID: 111, Name: "myrepo", OwnerName: "myorg"},
	})

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"workspace", "repo", "remove", "myorg/myrepo"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("workspace repo remove returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Removed") || !strings.Contains(out, "myorg/myrepo") {
		t.Errorf("output should confirm removal, got: %s", out)
	}

	if _, ok := cache.Get[[]resolve.CachedRepo](resolve.RepoCacheKey("ws-123")); ok {
		t.Error("repo cache should be cleared after remove")
	}
}

func TestWorkspaceRepoRemoveDryRun(t *testing.T) {
	ms := testutil.NewMockServer(t)
	// No mutation handler — dry-run must not call the API.
	setupWorkspaceRepoMutationTest(t, ms)

	_ = cache.Set(resolve.RepoCacheKey("ws-123"), []resolve.CachedRepo{
		{ID: "r1", GhID: 222, Name: "myrepo", OwnerName: "myorg"},
	})

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"workspace", "repo", "remove", "myorg/myrepo", "--dry-run"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("workspace repo remove --dry-run returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Would remove") {
		t.Errorf("dry-run should use 'Would remove' prefix, got: %s", out)
	}
	if !strings.Contains(out, "222") {
		t.Errorf("dry-run should show GitHub ID, got: %s", out)
	}

	// Cache should still be present (no mutation happened).
	if _, ok := cache.Get[[]resolve.CachedRepo](resolve.RepoCacheKey("ws-123")); !ok {
		t.Error("repo cache should not be cleared on dry-run")
	}
}

func TestWorkspaceRepoRemoveByGhID(t *testing.T) {
	ms := testutil.NewMockServer(t)
	ms.HandleQuery("DisconnectWorkspaceRepository", emptyMutationResponse("disconnectWorkspaceRepository"))
	setupWorkspaceRepoMutationTest(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"workspace", "repo", "remove", "--gh-id", "777"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("workspace repo remove --gh-id returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Removed") {
		t.Errorf("output should confirm removal, got: %s", out)
	}
}
