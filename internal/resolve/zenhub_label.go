package resolve

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/dslh/zh/internal/api"
	"github.com/dslh/zh/internal/cache"
	"github.com/dslh/zh/internal/exitcode"
)

// CachedZenhubLabel is the shape stored in the ZenHub label cache.
// These are organization-scoped labels used on ZenHub epics, distinct
// from the repository-scoped GitHub labels used on issues.
type CachedZenhubLabel struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
}

// ZenhubLabelResult is the resolved ZenHub label returned to callers.
type ZenhubLabelResult struct {
	ID    string
	Name  string
	Color string
}

const listZenhubLabelsQuery = `query ListZenhubLabels($workspaceId: ID!, $first: Int!, $after: String) {
  workspace(id: $workspaceId) {
    zenhubLabels(first: $first, after: $after) {
      totalCount
      pageInfo {
        hasNextPage
        endCursor
      }
      nodes {
        id
        name
        color
      }
    }
  }
}`

// FetchZenhubLabels fetches all ZenHub labels in a workspace and updates the cache.
func FetchZenhubLabels(client *api.Client, workspaceID string) ([]CachedZenhubLabel, error) {
	var allLabels []CachedZenhubLabel
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

		data, err := client.Execute(listZenhubLabelsQuery, vars)
		if err != nil {
			return nil, exitcode.General("fetching ZenHub labels", err)
		}

		var resp struct {
			Workspace struct {
				ZenhubLabels struct {
					TotalCount int `json:"totalCount"`
					PageInfo   struct {
						HasNextPage bool   `json:"hasNextPage"`
						EndCursor   string `json:"endCursor"`
					} `json:"pageInfo"`
					Nodes []CachedZenhubLabel `json:"nodes"`
				} `json:"zenhubLabels"`
			} `json:"workspace"`
		}
		if err := json.Unmarshal(data, &resp); err != nil {
			return nil, exitcode.General("parsing ZenHub labels response", err)
		}

		allLabels = append(allLabels, resp.Workspace.ZenhubLabels.Nodes...)

		if !resp.Workspace.ZenhubLabels.PageInfo.HasNextPage {
			break
		}
		c := resp.Workspace.ZenhubLabels.PageInfo.EndCursor
		cursor = &c
	}

	_ = cache.Set(ZenhubLabelCacheKey(workspaceID), allLabels)
	return allLabels, nil
}

// ZenhubLabelCacheKey returns the cache key for ZenHub label data scoped to a workspace.
func ZenhubLabelCacheKey(workspaceID string) cache.Key {
	return cache.NewScopedKey("zenhub-labels", workspaceID)
}

// ZenhubLabel resolves a label name to a ZenhubLabelResult using case-insensitive
// exact match, with invalidate-on-miss caching.
func ZenhubLabel(client *api.Client, workspaceID string, name string) (*ZenhubLabelResult, error) {
	key := ZenhubLabelCacheKey(workspaceID)

	// Try cache first
	if entries, ok := cache.Get[[]CachedZenhubLabel](key); ok {
		if l, found := matchZenhubLabel(entries, name); found {
			return l, nil
		}
	}

	// Cache miss — refresh from API
	entries, err := FetchZenhubLabels(client, workspaceID)
	if err != nil {
		return nil, err
	}

	if l, found := matchZenhubLabel(entries, name); found {
		return l, nil
	}

	return nil, exitcode.NotFoundError(fmt.Sprintf("label %q not found in workspace", name))
}

// ZenhubLabels resolves multiple label names, returning results and any errors.
func ZenhubLabels(client *api.Client, workspaceID string, names []string) ([]*ZenhubLabelResult, error) {
	key := ZenhubLabelCacheKey(workspaceID)

	// Ensure cache is populated
	entries, ok := cache.Get[[]CachedZenhubLabel](key)
	if !ok {
		var err error
		entries, err = FetchZenhubLabels(client, workspaceID)
		if err != nil {
			return nil, err
		}
	}

	var results []*ZenhubLabelResult
	var notFound []string
	for _, name := range names {
		if l, found := matchZenhubLabel(entries, name); found {
			results = append(results, l)
		} else {
			notFound = append(notFound, name)
		}
	}

	if len(notFound) > 0 {
		// Try refreshing cache and retry
		var err error
		entries, err = FetchZenhubLabels(client, workspaceID)
		if err != nil {
			return nil, err
		}

		var stillNotFound []string
		for _, name := range notFound {
			if l, found := matchZenhubLabel(entries, name); found {
				results = append(results, l)
			} else {
				stillNotFound = append(stillNotFound, name)
			}
		}

		if len(stillNotFound) > 0 {
			return nil, exitcode.NotFoundError(fmt.Sprintf(
				"label(s) not found: %s",
				strings.Join(stillNotFound, ", "),
			))
		}
	}

	return results, nil
}

// matchZenhubLabel looks up a ZenHub label by exact name (case-insensitive) or by ID.
func matchZenhubLabel(entries []CachedZenhubLabel, name string) (*ZenhubLabelResult, bool) {
	nameLower := strings.ToLower(name)

	// Exact ID match
	for _, l := range entries {
		if l.ID == name {
			return &ZenhubLabelResult{ID: l.ID, Name: l.Name, Color: l.Color}, true
		}
	}

	// Exact name match (case-insensitive)
	for _, l := range entries {
		if strings.ToLower(l.Name) == nameLower {
			return &ZenhubLabelResult{ID: l.ID, Name: l.Name, Color: l.Color}, true
		}
	}

	return nil, false
}
