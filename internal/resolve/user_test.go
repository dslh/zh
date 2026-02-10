package resolve

import (
	"testing"

	"github.com/dslh/zh/internal/api"
	"github.com/dslh/zh/internal/cache"
	"github.com/dslh/zh/internal/exitcode"
	"github.com/dslh/zh/internal/testutil"
)

func setupUserCache(t *testing.T, workspaceID string, users []CachedUser) {
	t.Helper()
	cacheDir := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", cacheDir)
	_ = cache.Set(UserCacheKey(workspaceID), users)
}

func testUsers() []CachedUser {
	return []CachedUser{
		{ID: "u1", Name: "John Doe", GithubUser: &struct {
			Login string `json:"login"`
		}{Login: "johndoe"}},
		{ID: "u2", Name: "Jane Doe", GithubUser: &struct {
			Login string `json:"login"`
		}{Login: "janedoe"}},
		{ID: "u3", Name: "Bob Smith"},
	}
}

func TestUserResolveByID(t *testing.T) {
	setupUserCache(t, "ws1", testUsers())

	result, err := User(nil, "ws1", "u2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "u2" || result.Name != "Jane Doe" {
		t.Errorf("got %+v, want ID=u2 Name=Jane Doe", result)
	}
	if result.Login != "janedoe" {
		t.Errorf("got Login=%s, want janedoe", result.Login)
	}
}

func TestUserResolveByGitHubLogin(t *testing.T) {
	setupUserCache(t, "ws1", testUsers())

	result, err := User(nil, "ws1", "johndoe")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "u1" {
		t.Errorf("got ID=%s, want u1", result.ID)
	}
}

func TestUserResolveByGitHubLoginCaseInsensitive(t *testing.T) {
	setupUserCache(t, "ws1", testUsers())

	result, err := User(nil, "ws1", "JohnDoe")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "u1" {
		t.Errorf("got ID=%s, want u1", result.ID)
	}
}

func TestUserResolveByName(t *testing.T) {
	setupUserCache(t, "ws1", testUsers())

	result, err := User(nil, "ws1", "Bob Smith")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "u3" {
		t.Errorf("got ID=%s, want u3", result.ID)
	}
	if result.Login != "" {
		t.Errorf("got Login=%s, want empty (no GitHub user)", result.Login)
	}
}

func TestUserResolveWithAtPrefix(t *testing.T) {
	setupUserCache(t, "ws1", testUsers())

	result, err := User(nil, "ws1", "@janedoe")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "u2" {
		t.Errorf("got ID=%s, want u2", result.ID)
	}
}

func usersAPIResponse() map[string]any {
	return map[string]any{
		"data": map[string]any{
			"workspace": map[string]any{
				"zenhubUsers": map[string]any{
					"totalCount": 3,
					"pageInfo": map[string]any{
						"hasNextPage": false,
						"endCursor":   "",
					},
					"nodes": []any{
						map[string]any{"id": "u1", "name": "John Doe", "githubUser": map[string]any{"login": "johndoe"}},
						map[string]any{"id": "u2", "name": "Jane Doe", "githubUser": map[string]any{"login": "janedoe"}},
						map[string]any{"id": "u3", "name": "Bob Smith"},
					},
				},
			},
		},
	}
}

func TestUserNotFound(t *testing.T) {
	cacheDir := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", cacheDir)
	_ = cache.Set(UserCacheKey("ws1"), testUsers())

	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListZenhubUsers", usersAPIResponse())
	client := api.New("test-key", api.WithEndpoint(ms.URL()))

	_, err := User(client, "ws1", "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent user")
	}
	if ec := exitcode.ExitCode(err); ec != exitcode.NotFound {
		t.Errorf("exit code = %d, want %d (NotFound)", ec, exitcode.NotFound)
	}
}

func TestUserDisplayName(t *testing.T) {
	u := &UserResult{ID: "u1", Name: "John Doe", Login: "johndoe"}
	if got := u.DisplayName(); got != "@johndoe" {
		t.Errorf("DisplayName() = %q, want @johndoe", got)
	}

	u2 := &UserResult{ID: "u3", Name: "Bob Smith"}
	if got := u2.DisplayName(); got != "Bob Smith" {
		t.Errorf("DisplayName() = %q, want Bob Smith", got)
	}

	u3 := &UserResult{ID: "u4"}
	if got := u3.DisplayName(); got != "u4" {
		t.Errorf("DisplayName() = %q, want u4", got)
	}
}

func TestUsersResolveMultiple(t *testing.T) {
	setupUserCache(t, "ws1", testUsers())

	results, err := Users(nil, "ws1", []string{"johndoe", "janedoe"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("got %d results, want 2", len(results))
	}
}

func TestUsersResolveNotFound(t *testing.T) {
	cacheDir := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", cacheDir)
	_ = cache.Set(UserCacheKey("ws1"), testUsers())

	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListZenhubUsers", usersAPIResponse())
	client := api.New("test-key", api.WithEndpoint(ms.URL()))

	_, err := Users(client, "ws1", []string{"johndoe", "nonexistent"})
	if err == nil {
		t.Fatal("expected error for nonexistent user")
	}
	if ec := exitcode.ExitCode(err); ec != exitcode.NotFound {
		t.Errorf("exit code = %d, want %d (NotFound)", ec, exitcode.NotFound)
	}
}

func TestUserResolveWithAPIRefresh(t *testing.T) {
	cacheDir := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", cacheDir)

	// Start with empty cache
	_ = cache.Set(UserCacheKey("ws1"), []CachedUser{})

	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListZenhubUsers", map[string]any{
		"data": map[string]any{
			"workspace": map[string]any{
				"zenhubUsers": map[string]any{
					"totalCount": 1,
					"pageInfo": map[string]any{
						"hasNextPage": false,
						"endCursor":   "",
					},
					"nodes": []any{
						map[string]any{
							"id":   "u-new",
							"name": "New User",
							"githubUser": map[string]any{
								"login": "newuser",
							},
						},
					},
				},
			},
		},
	})

	client := api.New("test-key", api.WithEndpoint(ms.URL()))

	result, err := User(client, "ws1", "newuser")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "u-new" {
		t.Errorf("got ID=%s, want u-new", result.ID)
	}
}
