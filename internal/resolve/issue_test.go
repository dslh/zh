package resolve

import (
	"testing"

	"github.com/dslh/zh/internal/api"
	"github.com/dslh/zh/internal/cache"
	"github.com/dslh/zh/internal/exitcode"
	"github.com/dslh/zh/internal/gh"
	"github.com/dslh/zh/internal/testutil"
)

// --- ParseIssueRef tests ---

func TestParseIssueRefOwnerRepoNumber(t *testing.T) {
	ref, err := ParseIssueRef("dlakehammond/task-tracker#3")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ref.Owner != "dlakehammond" || ref.Repo != "task-tracker" || ref.Number != 3 {
		t.Errorf("got owner=%q repo=%q number=%d, want dlakehammond/task-tracker#3", ref.Owner, ref.Repo, ref.Number)
	}
	if ref.ZenHubID != "" {
		t.Errorf("ZenHubID should be empty, got %q", ref.ZenHubID)
	}
}

func TestParseIssueRefRepoNumber(t *testing.T) {
	ref, err := ParseIssueRef("task-tracker#3")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ref.Owner != "" || ref.Repo != "task-tracker" || ref.Number != 3 {
		t.Errorf("got owner=%q repo=%q number=%d, want task-tracker#3", ref.Owner, ref.Repo, ref.Number)
	}
}

func TestParseIssueRefBareNumber(t *testing.T) {
	ref, err := ParseIssueRef("42")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ref.Number != 42 || ref.Repo != "" || ref.ZenHubID != "" {
		t.Errorf("got number=%d repo=%q zenHubID=%q, want bare number 42", ref.Number, ref.Repo, ref.ZenHubID)
	}
}

func TestParseIssueRefZenHubID(t *testing.T) {
	id := "Z2lkOi8vcmFwdG9yL0lzc3VlLzU2Nzg5"
	ref, err := ParseIssueRef(id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ref.ZenHubID != id {
		t.Errorf("got ZenHubID=%q, want %q", ref.ZenHubID, id)
	}
	if ref.Number != 0 || ref.Repo != "" {
		t.Errorf("expected only ZenHubID to be set, got number=%d repo=%q", ref.Number, ref.Repo)
	}
}

func TestParseIssueRefInvalid(t *testing.T) {
	_, err := ParseIssueRef("not-valid!")
	if err == nil {
		t.Fatal("expected error for invalid identifier, got nil")
	}
}

func TestParseIssueRefZeroNumber(t *testing.T) {
	_, err := ParseIssueRef("0")
	if err == nil {
		t.Fatal("expected error for zero number, got nil")
	}
}

func TestParseIssueRefNegativeNumber(t *testing.T) {
	_, err := ParseIssueRef("-1")
	if err == nil {
		t.Fatal("expected error for negative number, got nil")
	}
}

// --- LookupRepo tests ---

func testRepos() []CachedRepo {
	return []CachedRepo{
		{ID: "r1", GhID: 100, Name: "task-tracker", OwnerName: "dlakehammond"},
		{ID: "r2", GhID: 200, Name: "recipe-book", OwnerName: "dlakehammond"},
	}
}

func TestLookupRepoByOwnerAndName(t *testing.T) {
	repo, err := LookupRepo(testRepos(), "dlakehammond/task-tracker")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.GhID != 100 {
		t.Errorf("got GhID=%d, want 100", repo.GhID)
	}
}

func TestLookupRepoByName(t *testing.T) {
	repo, err := LookupRepo(testRepos(), "recipe-book")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.GhID != 200 {
		t.Errorf("got GhID=%d, want 200", repo.GhID)
	}
}

func TestLookupRepoCaseInsensitive(t *testing.T) {
	repo, err := LookupRepo(testRepos(), "Task-Tracker")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.GhID != 100 {
		t.Errorf("got GhID=%d, want 100", repo.GhID)
	}
}

func TestLookupRepoNotFound(t *testing.T) {
	_, err := LookupRepo(testRepos(), "nonexistent")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if exitcode.ExitCode(err) != exitcode.NotFound {
		t.Errorf("exit code = %d, want %d (NotFound)", exitcode.ExitCode(err), exitcode.NotFound)
	}
}

func TestLookupRepoAmbiguous(t *testing.T) {
	repos := []CachedRepo{
		{ID: "r1", GhID: 100, Name: "my-app", OwnerName: "alice"},
		{ID: "r2", GhID: 200, Name: "my-app", OwnerName: "bob"},
	}
	_, err := LookupRepo(repos, "my-app")
	if err == nil {
		t.Fatal("expected ambiguous error, got nil")
	}
	if exitcode.ExitCode(err) != exitcode.UsageError {
		t.Errorf("exit code = %d, want %d (UsageError)", exitcode.ExitCode(err), exitcode.UsageError)
	}
	if !containsStr(err.Error(), "ambiguous") {
		t.Errorf("error should mention 'ambiguous', got: %s", err.Error())
	}
}

func TestLookupRepoAmbiguousResolvedByOwner(t *testing.T) {
	repos := []CachedRepo{
		{ID: "r1", GhID: 100, Name: "my-app", OwnerName: "alice"},
		{ID: "r2", GhID: 200, Name: "my-app", OwnerName: "bob"},
	}
	repo, err := LookupRepo(repos, "bob/my-app")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.GhID != 200 {
		t.Errorf("got GhID=%d, want 200", repo.GhID)
	}
}

// --- RepoNamesAmbiguous tests ---

func TestRepoNamesAmbiguousFalse(t *testing.T) {
	if RepoNamesAmbiguous(testRepos()) {
		t.Error("expected no ambiguity for repos with different names")
	}
}

func TestRepoNamesAmbiguousTrue(t *testing.T) {
	repos := []CachedRepo{
		{Name: "app", OwnerName: "alice"},
		{Name: "app", OwnerName: "bob"},
	}
	if !RepoNamesAmbiguous(repos) {
		t.Error("expected ambiguity for repos with same name, different owners")
	}
}

// --- Issue resolution integration tests ---

func setupRepoCache(t *testing.T, workspaceID string, repos []CachedRepo) {
	t.Helper()
	cacheDir := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", cacheDir)
	_ = cache.Set(RepoCacheKey(workspaceID), repos)
}

func issueByInfoResponse(id string, number int, ghID int, name, owner string) map[string]any {
	return map[string]any{
		"data": map[string]any{
			"issueByInfo": map[string]any{
				"id":     id,
				"number": number,
				"repository": map[string]any{
					"ghId":      ghID,
					"name":      name,
					"ownerName": owner,
				},
			},
		},
	}
}

func issueByNodeResponse(id string, number int, ghID int, name, owner string) map[string]any {
	return map[string]any{
		"data": map[string]any{
			"node": map[string]any{
				"id":     id,
				"number": number,
				"repository": map[string]any{
					"ghId":      ghID,
					"name":      name,
					"ownerName": owner,
				},
			},
		},
	}
}

func TestIssueResolveByRepoNumber(t *testing.T) {
	setupRepoCache(t, "ws1", testRepos())

	ms := testutil.NewMockServer(t)
	ms.HandleQuery("IssueByInfo", issueByInfoResponse("issue1", 3, 100, "task-tracker", "dlakehammond"))

	client := api.New("test-key", api.WithEndpoint(ms.URL()))

	result, err := Issue(client, "ws1", "task-tracker#3", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "issue1" {
		t.Errorf("got ID=%q, want issue1", result.ID)
	}
	if result.Number != 3 {
		t.Errorf("got Number=%d, want 3", result.Number)
	}
	if result.RepoName != "task-tracker" {
		t.Errorf("got RepoName=%q, want task-tracker", result.RepoName)
	}
}

func TestIssueResolveByOwnerRepoNumber(t *testing.T) {
	setupRepoCache(t, "ws1", testRepos())

	ms := testutil.NewMockServer(t)
	ms.HandleQuery("IssueByInfo", issueByInfoResponse("issue1", 3, 100, "task-tracker", "dlakehammond"))

	client := api.New("test-key", api.WithEndpoint(ms.URL()))

	result, err := Issue(client, "ws1", "dlakehammond/task-tracker#3", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "issue1" || result.Number != 3 {
		t.Errorf("got ID=%q Number=%d, want issue1 #3", result.ID, result.Number)
	}
}

func TestIssueResolveByZenHubID(t *testing.T) {
	cacheDir := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", cacheDir)

	ms := testutil.NewMockServer(t)
	ms.HandleQuery("IssueByNode", issueByNodeResponse("Z2lkOi8vcmFwdG9yL0lzc3VlLzEyMzQ1", 5, 100, "task-tracker", "dlakehammond"))

	client := api.New("test-key", api.WithEndpoint(ms.URL()))

	result, err := Issue(client, "ws1", "Z2lkOi8vcmFwdG9yL0lzc3VlLzEyMzQ1", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "Z2lkOi8vcmFwdG9yL0lzc3VlLzEyMzQ1" {
		t.Errorf("got ID=%q, want Z2lkOi8vcmFwdG9yL0lzc3VlLzEyMzQ1", result.ID)
	}
	if result.Number != 5 {
		t.Errorf("got Number=%d, want 5", result.Number)
	}
}

func TestIssueResolveByBareNumberWithRepoFlag(t *testing.T) {
	setupRepoCache(t, "ws1", testRepos())

	ms := testutil.NewMockServer(t)
	ms.HandleQuery("IssueByInfo", issueByInfoResponse("issue42", 42, 100, "task-tracker", "dlakehammond"))

	client := api.New("test-key", api.WithEndpoint(ms.URL()))

	result, err := Issue(client, "ws1", "42", &IssueOptions{RepoFlag: "task-tracker"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "issue42" || result.Number != 42 {
		t.Errorf("got ID=%q Number=%d, want issue42 #42", result.ID, result.Number)
	}
}

func TestIssueResolveBareNumberWithoutRepoFlag(t *testing.T) {
	cacheDir := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", cacheDir)

	ms := testutil.NewMockServer(t)
	client := api.New("test-key", api.WithEndpoint(ms.URL()))

	_, err := Issue(client, "ws1", "42", nil)
	if err == nil {
		t.Fatal("expected error for bare number without --repo, got nil")
	}
	if exitcode.ExitCode(err) != exitcode.UsageError {
		t.Errorf("exit code = %d, want %d (UsageError)", exitcode.ExitCode(err), exitcode.UsageError)
	}
}

func TestIssueResolveRepoNotFound(t *testing.T) {
	setupRepoCache(t, "ws1", testRepos())

	// Mock server returns repos without the target
	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListRepos", map[string]any{
		"data": map[string]any{
			"workspace": map[string]any{
				"repositoriesConnection": map[string]any{
					"pageInfo": map[string]any{
						"hasNextPage": false,
						"endCursor":   "",
					},
					"nodes": []any{
						map[string]any{"id": "r1", "ghId": 100, "name": "task-tracker", "ownerName": "dlakehammond"},
						map[string]any{"id": "r2", "ghId": 200, "name": "recipe-book", "ownerName": "dlakehammond"},
					},
				},
			},
		},
	})

	client := api.New("test-key", api.WithEndpoint(ms.URL()))

	_, err := Issue(client, "ws1", "nonexistent#1", nil)
	if err == nil {
		t.Fatal("expected error for repo not found, got nil")
	}
	if exitcode.ExitCode(err) != exitcode.NotFound {
		t.Errorf("exit code = %d, want %d (NotFound)", exitcode.ExitCode(err), exitcode.NotFound)
	}
}

func TestIssueResolveIssueNotFound(t *testing.T) {
	setupRepoCache(t, "ws1", testRepos())

	ms := testutil.NewMockServer(t)
	ms.HandleQuery("IssueByInfo", map[string]any{
		"data": map[string]any{
			"issueByInfo": nil,
		},
	})

	client := api.New("test-key", api.WithEndpoint(ms.URL()))

	_, err := Issue(client, "ws1", "task-tracker#9999", nil)
	if err == nil {
		t.Fatal("expected error for issue not found, got nil")
	}
	if exitcode.ExitCode(err) != exitcode.NotFound {
		t.Errorf("exit code = %d, want %d (NotFound)", exitcode.ExitCode(err), exitcode.NotFound)
	}
}

func TestIssueRefAndFullRef(t *testing.T) {
	r := &IssueResult{
		ID:        "id1",
		Number:    42,
		RepoGhID:  100,
		RepoOwner: "dlakehammond",
		RepoName:  "task-tracker",
	}
	if got := r.Ref(); got != "task-tracker#42" {
		t.Errorf("Ref() = %q, want task-tracker#42", got)
	}
	if got := r.FullRef(); got != "dlakehammond/task-tracker#42" {
		t.Errorf("FullRef() = %q, want dlakehammond/task-tracker#42", got)
	}
}

func TestIssueResolveRepoRefreshesOnMiss(t *testing.T) {
	cacheDir := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", cacheDir)

	// Cache with only one repo
	_ = cache.Set(RepoCacheKey("ws1"), []CachedRepo{
		{ID: "r1", GhID: 100, Name: "task-tracker", OwnerName: "dlakehammond"},
	})

	ms := testutil.NewMockServer(t)
	// API returns both repos
	ms.HandleQuery("ListRepos", map[string]any{
		"data": map[string]any{
			"workspace": map[string]any{
				"repositoriesConnection": map[string]any{
					"pageInfo": map[string]any{
						"hasNextPage": false,
						"endCursor":   "",
					},
					"nodes": []any{
						map[string]any{"id": "r1", "ghId": 100, "name": "task-tracker", "ownerName": "dlakehammond"},
						map[string]any{"id": "r2", "ghId": 200, "name": "recipe-book", "ownerName": "dlakehammond"},
					},
				},
			},
		},
	})
	ms.HandleQuery("IssueByInfo", issueByInfoResponse("issue1", 1, 200, "recipe-book", "dlakehammond"))

	client := api.New("test-key", api.WithEndpoint(ms.URL()))

	// recipe-book is not in cache, should trigger refresh
	result, err := Issue(client, "ws1", "recipe-book#1", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "issue1" {
		t.Errorf("got ID=%q, want issue1", result.ID)
	}
}

func TestIssueResolveByBranchName(t *testing.T) {
	setupRepoCache(t, "ws1", testRepos())

	// Mock ZenHub API
	zhMs := testutil.NewMockServer(t)
	zhMs.HandleQuery("IssueByInfo", issueByInfoResponse("pr5", 5, 100, "task-tracker", "dlakehammond"))

	// Mock GitHub API
	ghMs := testutil.NewMockServer(t)
	ghMs.HandleQuery("PRByBranch", map[string]any{
		"data": map[string]any{
			"repository": map[string]any{
				"pullRequests": map[string]any{
					"nodes": []any{
						map[string]any{"number": 5},
					},
				},
			},
		},
	})

	zhClient := api.New("test-key", api.WithEndpoint(zhMs.URL()))
	ghClient := gh.New("pat", "test-token", gh.WithEndpoint(ghMs.URL()))

	result, err := Issue(zhClient, "ws1", "fix-login-bug", &IssueOptions{
		RepoFlag:     "task-tracker",
		GitHubClient: ghClient,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "pr5" || result.Number != 5 {
		t.Errorf("got ID=%q Number=%d, want pr5 #5", result.ID, result.Number)
	}
}

func TestIssueResolveByBranchNameNoPRFound(t *testing.T) {
	setupRepoCache(t, "ws1", testRepos())

	zhMs := testutil.NewMockServer(t)
	ghMs := testutil.NewMockServer(t)
	ghMs.HandleQuery("PRByBranch", map[string]any{
		"data": map[string]any{
			"repository": map[string]any{
				"pullRequests": map[string]any{
					"nodes": []any{},
				},
			},
		},
	})

	zhClient := api.New("test-key", api.WithEndpoint(zhMs.URL()))
	ghClient := gh.New("pat", "test-token", gh.WithEndpoint(ghMs.URL()))

	_, err := Issue(zhClient, "ws1", "nonexistent-branch", &IssueOptions{
		RepoFlag:     "task-tracker",
		GitHubClient: ghClient,
	})
	if err == nil {
		t.Fatal("expected error for branch not found, got nil")
	}
	if exitcode.ExitCode(err) != exitcode.NotFound {
		t.Errorf("exit code = %d, want %d (NotFound)", exitcode.ExitCode(err), exitcode.NotFound)
	}
}

func TestIssueResolveAmbiguousRepo(t *testing.T) {
	repos := []CachedRepo{
		{ID: "r1", GhID: 100, Name: "my-app", OwnerName: "alice"},
		{ID: "r2", GhID: 200, Name: "my-app", OwnerName: "bob"},
	}
	setupRepoCache(t, "ws1", repos)

	ms := testutil.NewMockServer(t)
	client := api.New("test-key", api.WithEndpoint(ms.URL()))

	_, err := Issue(client, "ws1", "my-app#1", nil)
	if err == nil {
		t.Fatal("expected ambiguous error, got nil")
	}
	if exitcode.ExitCode(err) != exitcode.UsageError {
		t.Errorf("exit code = %d, want %d (UsageError)", exitcode.ExitCode(err), exitcode.UsageError)
	}
	if !containsStr(err.Error(), "ambiguous") {
		t.Errorf("error should mention 'ambiguous', got: %s", err.Error())
	}
}
