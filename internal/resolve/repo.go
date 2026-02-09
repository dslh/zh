package resolve

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/dslh/zh/internal/api"
	"github.com/dslh/zh/internal/cache"
	"github.com/dslh/zh/internal/exitcode"
)

// CachedRepo is the shape stored in the repo cache.
type CachedRepo struct {
	ID        string `json:"id"`
	GhID      int    `json:"ghId"`
	Name      string `json:"name"`
	OwnerName string `json:"ownerName"`
}

// RepoCacheKey returns the cache key for repo data scoped to a workspace.
func RepoCacheKey(workspaceID string) cache.Key {
	return cache.NewScopedKey("repos", workspaceID)
}

const listReposQuery = `query ListRepos($workspaceId: ID!, $first: Int!, $after: String) {
  workspace(id: $workspaceId) {
    repositoriesConnection(first: $first, after: $after) {
      totalCount
      pageInfo {
        hasNextPage
        endCursor
      }
      nodes {
        id
        ghId
        name
        ownerName
      }
    }
  }
}`

// FetchRepos fetches all repos for a workspace from the API and updates
// the cache. Returns the cached repo entries.
func FetchRepos(client *api.Client, workspaceID string) ([]CachedRepo, error) {
	var allRepos []CachedRepo
	var cursor *string

	for {
		vars := map[string]any{
			"workspaceId": workspaceID,
			"first":       100,
		}
		if cursor != nil {
			vars["after"] = *cursor
		}

		data, err := client.Execute(listReposQuery, vars)
		if err != nil {
			return nil, exitcode.General("fetching repos", err)
		}

		var resp struct {
			Workspace struct {
				ReposConn struct {
					PageInfo struct {
						HasNextPage bool   `json:"hasNextPage"`
						EndCursor   string `json:"endCursor"`
					} `json:"pageInfo"`
					Nodes []CachedRepo `json:"nodes"`
				} `json:"repositoriesConnection"`
			} `json:"workspace"`
		}
		if err := json.Unmarshal(data, &resp); err != nil {
			return nil, exitcode.General("parsing repos response", err)
		}

		allRepos = append(allRepos, resp.Workspace.ReposConn.Nodes...)

		if !resp.Workspace.ReposConn.PageInfo.HasNextPage {
			break
		}
		cursor = &resp.Workspace.ReposConn.PageInfo.EndCursor
	}

	_ = cache.Set(RepoCacheKey(workspaceID), allRepos)
	return allRepos, nil
}

// FetchReposIntoCache stores pre-fetched repo entries in the cache.
// This is used by commands that already have repo data (e.g. workspace show/repos)
// to populate the cache without an extra API call.
func FetchReposIntoCache(entries []CachedRepo, workspaceID string) error {
	return cache.Set(RepoCacheKey(workspaceID), entries)
}

// LookupRepo finds a repo in a list by name (with optional owner prefix).
// Accepts "repo" or "owner/repo" format. If the short form is ambiguous
// (multiple repos with the same name but different owners), returns an error.
func LookupRepo(repos []CachedRepo, identifier string) (*CachedRepo, error) {
	// Check for owner/repo format
	if parts := strings.SplitN(identifier, "/", 2); len(parts) == 2 {
		owner, name := parts[0], parts[1]
		for i, r := range repos {
			if strings.EqualFold(r.OwnerName, owner) && strings.EqualFold(r.Name, name) {
				return &repos[i], nil
			}
		}
		return nil, exitcode.NotFoundError(fmt.Sprintf("repository %q not found in workspace — run 'zh workspace repos' to see connected repos", identifier))
	}

	// Short form: just repo name
	var matches []int
	for i, r := range repos {
		if strings.EqualFold(r.Name, identifier) {
			matches = append(matches, i)
		}
	}

	if len(matches) == 1 {
		return &repos[matches[0]], nil
	}

	if len(matches) > 1 {
		msg := fmt.Sprintf("repository %q is ambiguous — matches %d repos:\n", identifier, len(matches))
		for _, i := range matches {
			msg += fmt.Sprintf("  - %s/%s\n", repos[i].OwnerName, repos[i].Name)
		}
		msg += "\nUse the full owner/repo format."
		return nil, exitcode.Usage(msg)
	}

	return nil, exitcode.NotFoundError(fmt.Sprintf("repository %q not found in workspace — run 'zh workspace repos' to see connected repos", identifier))
}

// RepoNamesAmbiguous checks whether any repo name appears under different
// owners in the given repo list, requiring long-form issue references.
func RepoNamesAmbiguous(repos []CachedRepo) bool {
	seen := make(map[string]string) // name -> owner
	for _, r := range repos {
		if prev, ok := seen[r.Name]; ok && prev != r.OwnerName {
			return true
		}
		seen[r.Name] = r.OwnerName
	}
	return false
}
