package resolve

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/dslh/zh/internal/api"
	"github.com/dslh/zh/internal/cache"
	"github.com/dslh/zh/internal/exitcode"
)

// CachedPriority is the shape stored in the priority cache.
type CachedPriority struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Color       string `json:"color"`
	Description string `json:"description"`
}

// PriorityResult is the resolved priority returned to callers.
type PriorityResult struct {
	ID    string
	Name  string
	Color string
}

const listPrioritiesQuery = `query GetWorkspacePriorities($workspaceId: ID!) {
  workspace(id: $workspaceId) {
    prioritiesConnection {
      nodes {
        id
        name
        color
        description
      }
    }
  }
}`

// FetchPriorities fetches all priorities for a workspace from the API and
// updates the cache. Returns the cached priority entries.
func FetchPriorities(client *api.Client, workspaceID string) ([]CachedPriority, error) {
	data, err := client.Execute(listPrioritiesQuery, map[string]any{
		"workspaceId": workspaceID,
	})
	if err != nil {
		return nil, exitcode.General("fetching priorities", err)
	}

	var resp struct {
		Workspace struct {
			PrioritiesConnection struct {
				Nodes []CachedPriority `json:"nodes"`
			} `json:"prioritiesConnection"`
		} `json:"workspace"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, exitcode.General("parsing priorities response", err)
	}

	priorities := resp.Workspace.PrioritiesConnection.Nodes
	_ = cache.Set(PriorityCacheKey(workspaceID), priorities)
	return priorities, nil
}

// PriorityCacheKey returns the cache key for priority data scoped to a workspace.
func PriorityCacheKey(workspaceID string) cache.Key {
	return cache.NewScopedKey("priorities", workspaceID)
}

// Priority resolves a priority identifier to a PriorityResult. It attempts
// resolution by exact ID, exact name (case-insensitive), and unique substring
// match, using the cache with invalidate-on-miss semantics.
func Priority(client *api.Client, workspaceID string, identifier string) (*PriorityResult, error) {
	key := PriorityCacheKey(workspaceID)

	// Try cache first
	if entries, ok := cache.Get[[]CachedPriority](key); ok {
		if p, found := matchPriority(entries, identifier); found {
			return p, nil
		}
		// Check for ambiguity before refreshing
		if err := checkPriorityAmbiguous(entries, identifier); err != nil {
			return nil, err
		}
	}

	// Cache miss — refresh from API
	entries, err := FetchPriorities(client, workspaceID)
	if err != nil {
		return nil, err
	}

	if p, found := matchPriority(entries, identifier); found {
		return p, nil
	}

	if err := checkPriorityAmbiguous(entries, identifier); err != nil {
		return nil, err
	}

	return nil, exitcode.NotFoundError(fmt.Sprintf("priority %q not found — run 'zh priority list' to see available priorities", identifier))
}

// matchPriority attempts to match a priority by exact ID, exact name,
// or unique substring.
func matchPriority(entries []CachedPriority, identifier string) (*PriorityResult, bool) {
	idLower := strings.ToLower(identifier)

	// Exact ID match
	for _, p := range entries {
		if p.ID == identifier {
			return &PriorityResult{ID: p.ID, Name: p.Name, Color: p.Color}, true
		}
	}

	// Exact name match (case-insensitive)
	for _, p := range entries {
		if strings.ToLower(p.Name) == idLower {
			return &PriorityResult{ID: p.ID, Name: p.Name, Color: p.Color}, true
		}
	}

	// Unique substring match (case-insensitive)
	var matches []CachedPriority
	for _, p := range entries {
		if strings.Contains(strings.ToLower(p.Name), idLower) {
			matches = append(matches, p)
		}
	}
	if len(matches) == 1 {
		return &PriorityResult{ID: matches[0].ID, Name: matches[0].Name, Color: matches[0].Color}, true
	}

	return nil, false
}

// checkPriorityAmbiguous returns a descriptive error if the identifier matches
// multiple priorities.
func checkPriorityAmbiguous(entries []CachedPriority, identifier string) error {
	idLower := strings.ToLower(identifier)
	var matches []string
	for _, p := range entries {
		if strings.Contains(strings.ToLower(p.Name), idLower) {
			matches = append(matches, fmt.Sprintf("%s [%s]", p.Name, p.ID))
		}
	}
	if len(matches) > 1 {
		msg := fmt.Sprintf("priority %q is ambiguous — matches %d priorities:\n", identifier, len(matches))
		for _, m := range matches {
			msg += "  - " + m + "\n"
		}
		msg += "\nUse a more specific name or the priority ID."
		return exitcode.Usage(msg)
	}
	return nil
}
