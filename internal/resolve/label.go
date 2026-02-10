package resolve

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/dslh/zh/internal/api"
	"github.com/dslh/zh/internal/cache"
	"github.com/dslh/zh/internal/exitcode"
)

// CachedLabel is the shape stored in the label cache.
type CachedLabel struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
}

// LabelResult is the resolved label returned to callers.
type LabelResult struct {
	ID    string
	Name  string
	Color string
}

const listLabelsQuery = `query GetWorkspaceLabels($workspaceId: ID!) {
  workspace(id: $workspaceId) {
    repositoriesConnection(first: 100) {
      nodes {
        labels(first: 100) {
          nodes {
            id
            name
            color
          }
        }
      }
    }
  }
}`

// FetchLabels fetches all labels across all workspace repos from the API and
// updates the cache. Labels are deduplicated by name (keeping the first
// occurrence, since labels with the same name across repos are equivalent).
func FetchLabels(client *api.Client, workspaceID string) ([]CachedLabel, error) {
	data, err := client.Execute(listLabelsQuery, map[string]any{
		"workspaceId": workspaceID,
	})
	if err != nil {
		return nil, exitcode.General("fetching labels", err)
	}

	var resp struct {
		Workspace struct {
			RepositoriesConnection struct {
				Nodes []struct {
					Labels struct {
						Nodes []CachedLabel `json:"nodes"`
					} `json:"labels"`
				} `json:"nodes"`
			} `json:"repositoriesConnection"`
		} `json:"workspace"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, exitcode.General("parsing labels response", err)
	}

	// Deduplicate by name (case-insensitive)
	seen := make(map[string]bool)
	var labels []CachedLabel
	for _, repo := range resp.Workspace.RepositoriesConnection.Nodes {
		for _, l := range repo.Labels.Nodes {
			key := strings.ToLower(l.Name)
			if !seen[key] {
				seen[key] = true
				labels = append(labels, l)
			}
		}
	}

	_ = cache.Set(LabelCacheKey(workspaceID), labels)
	return labels, nil
}

// LabelCacheKey returns the cache key for label data scoped to a workspace.
func LabelCacheKey(workspaceID string) cache.Key {
	return cache.NewScopedKey("labels", workspaceID)
}

// Label resolves a label name to a LabelResult using case-insensitive exact
// match, with invalidate-on-miss caching.
func Label(client *api.Client, workspaceID string, name string) (*LabelResult, error) {
	key := LabelCacheKey(workspaceID)

	// Try cache first
	if entries, ok := cache.Get[[]CachedLabel](key); ok {
		if l, found := matchLabel(entries, name); found {
			return l, nil
		}
	}

	// Cache miss — refresh from API
	entries, err := FetchLabels(client, workspaceID)
	if err != nil {
		return nil, err
	}

	if l, found := matchLabel(entries, name); found {
		return l, nil
	}

	return nil, exitcode.NotFoundError(fmt.Sprintf("label %q not found — run 'zh label list' to see available labels", name))
}

// Labels resolves multiple label names, returning results and any errors.
func Labels(client *api.Client, workspaceID string, names []string) ([]*LabelResult, error) {
	key := LabelCacheKey(workspaceID)

	// Ensure cache is populated
	entries, ok := cache.Get[[]CachedLabel](key)
	if !ok {
		var err error
		entries, err = FetchLabels(client, workspaceID)
		if err != nil {
			return nil, err
		}
	}

	var results []*LabelResult
	var notFound []string
	for _, name := range names {
		if l, found := matchLabel(entries, name); found {
			results = append(results, l)
		} else {
			notFound = append(notFound, name)
		}
	}

	if len(notFound) > 0 {
		// Try refreshing cache and retry not-found ones
		var err error
		entries, err = FetchLabels(client, workspaceID)
		if err != nil {
			return nil, err
		}

		var stillNotFound []string
		for _, name := range notFound {
			if l, found := matchLabel(entries, name); found {
				results = append(results, l)
			} else {
				stillNotFound = append(stillNotFound, name)
			}
		}

		if len(stillNotFound) > 0 {
			return nil, exitcode.NotFoundError(fmt.Sprintf(
				"label(s) not found: %s — run 'zh label list' to see available labels",
				strings.Join(stillNotFound, ", "),
			))
		}
	}

	return results, nil
}

// matchLabel looks up a label by exact name (case-insensitive) or by ID.
func matchLabel(entries []CachedLabel, name string) (*LabelResult, bool) {
	nameLower := strings.ToLower(name)

	// Exact ID match
	for _, l := range entries {
		if l.ID == name {
			return &LabelResult{ID: l.ID, Name: l.Name, Color: l.Color}, true
		}
	}

	// Exact name match (case-insensitive)
	for _, l := range entries {
		if strings.ToLower(l.Name) == nameLower {
			return &LabelResult{ID: l.ID, Name: l.Name, Color: l.Color}, true
		}
	}

	return nil, false
}
