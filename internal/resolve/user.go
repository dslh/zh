package resolve

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/dslh/zh/internal/api"
	"github.com/dslh/zh/internal/cache"
	"github.com/dslh/zh/internal/exitcode"
)

// CachedUser is the shape stored in the user cache.
type CachedUser struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	// GithubUser holds the linked GitHub account info, if any.
	GithubUser *struct {
		Login string `json:"login"`
	} `json:"githubUser"`
}

// UserResult is the resolved user returned to callers.
type UserResult struct {
	ID    string
	Name  string
	Login string // GitHub login, if available
}

// DisplayName returns the best human-readable name for the user.
func (u *UserResult) DisplayName() string {
	if u.Login != "" {
		return "@" + u.Login
	}
	if u.Name != "" {
		return u.Name
	}
	return u.ID
}

const listZenhubUsersQuery = `query ListZenhubUsers($workspaceId: ID!, $first: Int!, $after: String) {
  workspace(id: $workspaceId) {
    zenhubUsers(first: $first, after: $after) {
      totalCount
      pageInfo {
        hasNextPage
        endCursor
      }
      nodes {
        id
        name
        githubUser { login }
      }
    }
  }
}`

// FetchUsers fetches all ZenHub users in a workspace and updates the cache.
func FetchUsers(client *api.Client, workspaceID string) ([]CachedUser, error) {
	var allUsers []CachedUser
	var cursor *string
	pageSize := 100

	for {
		vars := map[string]any{
			"workspaceId": workspaceID,
			"first":       pageSize,
		}
		if cursor != nil {
			vars["after"] = *cursor
		}

		data, err := client.Execute(listZenhubUsersQuery, vars)
		if err != nil {
			return nil, exitcode.General("fetching workspace users", err)
		}

		var resp struct {
			Workspace struct {
				ZenhubUsers struct {
					TotalCount int `json:"totalCount"`
					PageInfo   struct {
						HasNextPage bool   `json:"hasNextPage"`
						EndCursor   string `json:"endCursor"`
					} `json:"pageInfo"`
					Nodes []CachedUser `json:"nodes"`
				} `json:"zenhubUsers"`
			} `json:"workspace"`
		}
		if err := json.Unmarshal(data, &resp); err != nil {
			return nil, exitcode.General("parsing users response", err)
		}

		allUsers = append(allUsers, resp.Workspace.ZenhubUsers.Nodes...)

		if !resp.Workspace.ZenhubUsers.PageInfo.HasNextPage {
			break
		}
		c := resp.Workspace.ZenhubUsers.PageInfo.EndCursor
		cursor = &c
	}

	_ = cache.Set(UserCacheKey(workspaceID), allUsers)
	return allUsers, nil
}

// UserCacheKey returns the cache key for user data scoped to a workspace.
func UserCacheKey(workspaceID string) cache.Key {
	return cache.NewScopedKey("users", workspaceID)
}

// User resolves a single user identifier to a UserResult using
// case-insensitive match on GitHub login, name, or ZenHub ID,
// with invalidate-on-miss caching.
func User(client *api.Client, workspaceID string, identifier string) (*UserResult, error) {
	key := UserCacheKey(workspaceID)

	// Try cache first
	if entries, ok := cache.Get[[]CachedUser](key); ok {
		if u, found := matchUser(entries, identifier); found {
			return u, nil
		}
	}

	// Cache miss — refresh from API
	entries, err := FetchUsers(client, workspaceID)
	if err != nil {
		return nil, err
	}

	if u, found := matchUser(entries, identifier); found {
		return u, nil
	}

	return nil, exitcode.NotFoundError(fmt.Sprintf("user %q not found in workspace", identifier))
}

// Users resolves multiple user identifiers, returning results and any errors.
func Users(client *api.Client, workspaceID string, identifiers []string) ([]*UserResult, error) {
	key := UserCacheKey(workspaceID)

	// Ensure cache is populated
	entries, ok := cache.Get[[]CachedUser](key)
	if !ok {
		var err error
		entries, err = FetchUsers(client, workspaceID)
		if err != nil {
			return nil, err
		}
	}

	var results []*UserResult
	var notFound []string
	for _, ident := range identifiers {
		if u, found := matchUser(entries, ident); found {
			results = append(results, u)
		} else {
			notFound = append(notFound, ident)
		}
	}

	if len(notFound) > 0 {
		// Try refreshing cache and retry
		var err error
		entries, err = FetchUsers(client, workspaceID)
		if err != nil {
			return nil, err
		}

		var stillNotFound []string
		for _, ident := range notFound {
			if u, found := matchUser(entries, ident); found {
				results = append(results, u)
			} else {
				stillNotFound = append(stillNotFound, ident)
			}
		}

		if len(stillNotFound) > 0 {
			return nil, exitcode.NotFoundError(fmt.Sprintf(
				"user(s) not found: %s",
				strings.Join(stillNotFound, ", "),
			))
		}
	}

	return results, nil
}

// matchUser looks up a user by ZenHub ID, GitHub login (case-insensitive),
// or display name (case-insensitive).
func matchUser(entries []CachedUser, identifier string) (*UserResult, bool) {
	identLower := strings.ToLower(identifier)
	// Strip leading @ if present (common convention)
	if strings.HasPrefix(identLower, "@") {
		identLower = identLower[1:]
		identifier = identifier[1:]
	}

	// Exact ID match
	for _, u := range entries {
		if u.ID == identifier {
			return buildUserResult(&u), true
		}
	}

	// GitHub login match (case-insensitive)
	for _, u := range entries {
		if u.GithubUser != nil && strings.ToLower(u.GithubUser.Login) == identLower {
			return buildUserResult(&u), true
		}
	}

	// Name match (case-insensitive)
	for _, u := range entries {
		if strings.ToLower(u.Name) == identLower {
			return buildUserResult(&u), true
		}
	}

	return nil, false
}

func buildUserResult(u *CachedUser) *UserResult {
	login := ""
	if u.GithubUser != nil {
		login = u.GithubUser.Login
	}

	return &UserResult{
		ID:    u.ID,
		Name:  u.Name,
		Login: login,
	}
}
