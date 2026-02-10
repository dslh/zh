package resolve

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/dslh/zh/internal/api"
	"github.com/dslh/zh/internal/cache"
	"github.com/dslh/zh/internal/exitcode"
)

// CachedEpic is the shape stored in the epic cache.
type CachedEpic struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Type  string `json:"type"` // "zenhub" or "legacy"

	// Legacy epics (backed by a GitHub issue) include the issue reference.
	IssueNumber int    `json:"issueNumber,omitempty"`
	RepoName    string `json:"repoName,omitempty"`
	RepoOwner   string `json:"repoOwner,omitempty"`
}

// EpicResult is the resolved epic returned to callers.
type EpicResult struct {
	ID    string
	Title string
	Type  string // "zenhub" or "legacy"
}

// EpicCacheKey returns the cache key for epic data scoped to a workspace.
func EpicCacheKey(workspaceID string) cache.Key {
	return cache.NewScopedKey("epics", workspaceID)
}

const listZenhubEpicsQuery = `query ListZenhubEpics($workspaceId: ID!, $first: Int!, $after: String) {
  workspace(id: $workspaceId) {
    zenhubEpics(first: $first, after: $after) {
      pageInfo {
        hasNextPage
        endCursor
      }
      nodes {
        id
        title
      }
    }
  }
}`

const listRoadmapEpicsQuery = `query ListRoadmapEpics($workspaceId: ID!, $first: Int!, $after: String) {
  workspace(id: $workspaceId) {
    roadmap {
      items(first: $first, after: $after) {
        pageInfo {
          hasNextPage
          endCursor
        }
        nodes {
          __typename
          ... on ZenhubEpic {
            id
            title
          }
          ... on Epic {
            id
            issue {
              title
              number
              repository {
                name
                ownerName
              }
            }
          }
        }
      }
    }
  }
}`

// FetchEpics fetches all epics for a workspace from the API and updates
// the cache. It combines the zenhubEpics query (which returns all standalone
// epics, including those not on the roadmap) with the roadmap items query
// (which is the only way to discover legacy epics). Results are deduplicated
// by ID so epics appearing in both sources are not listed twice.
func FetchEpics(client *api.Client, workspaceID string) ([]CachedEpic, error) {
	seen := make(map[string]bool)
	var allEpics []CachedEpic

	// 1. Fetch all ZenHub epics via the dedicated query.
	zenhubEpics, err := fetchZenhubEpics(client, workspaceID)
	if err != nil {
		return nil, err
	}
	for _, e := range zenhubEpics {
		seen[e.ID] = true
		allEpics = append(allEpics, e)
	}

	// 2. Fetch roadmap items to pick up legacy epics (and any ZenHub epics
	//    already seen, which we skip).
	roadmapEpics, err := fetchRoadmapEpics(client, workspaceID)
	if err != nil {
		return nil, err
	}
	for _, e := range roadmapEpics {
		if !seen[e.ID] {
			seen[e.ID] = true
			allEpics = append(allEpics, e)
		}
	}

	_ = cache.Set(EpicCacheKey(workspaceID), allEpics)
	return allEpics, nil
}

// fetchZenhubEpics fetches all standalone ZenHub epics via workspace.zenhubEpics.
func fetchZenhubEpics(client *api.Client, workspaceID string) ([]CachedEpic, error) {
	var result []CachedEpic
	var cursor *string

	for {
		vars := map[string]any{
			"workspaceId": workspaceID,
			"first":       100,
		}
		if cursor != nil {
			vars["after"] = *cursor
		}

		data, err := client.Execute(listZenhubEpicsQuery, vars)
		if err != nil {
			return nil, exitcode.General("fetching zenhub epics", err)
		}

		var resp struct {
			Workspace struct {
				ZenhubEpics struct {
					PageInfo struct {
						HasNextPage bool   `json:"hasNextPage"`
						EndCursor   string `json:"endCursor"`
					} `json:"pageInfo"`
					Nodes []struct {
						ID    string `json:"id"`
						Title string `json:"title"`
					} `json:"nodes"`
				} `json:"zenhubEpics"`
			} `json:"workspace"`
		}
		if err := json.Unmarshal(data, &resp); err != nil {
			return nil, exitcode.General("parsing zenhub epics response", err)
		}

		for _, n := range resp.Workspace.ZenhubEpics.Nodes {
			result = append(result, CachedEpic{
				ID:    n.ID,
				Title: n.Title,
				Type:  "zenhub",
			})
		}

		if !resp.Workspace.ZenhubEpics.PageInfo.HasNextPage {
			break
		}
		cursor = &resp.Workspace.ZenhubEpics.PageInfo.EndCursor
	}

	return result, nil
}

// fetchRoadmapEpics fetches epics from the workspace roadmap. This is
// the only way to discover legacy (issue-backed) epics.
func fetchRoadmapEpics(client *api.Client, workspaceID string) ([]CachedEpic, error) {
	var result []CachedEpic
	var cursor *string

	for {
		vars := map[string]any{
			"workspaceId": workspaceID,
			"first":       100,
		}
		if cursor != nil {
			vars["after"] = *cursor
		}

		data, err := client.Execute(listRoadmapEpicsQuery, vars)
		if err != nil {
			return nil, exitcode.General("fetching roadmap epics", err)
		}

		var resp struct {
			Workspace struct {
				Roadmap struct {
					Items struct {
						PageInfo struct {
							HasNextPage bool   `json:"hasNextPage"`
							EndCursor   string `json:"endCursor"`
						} `json:"pageInfo"`
						Nodes []json.RawMessage `json:"nodes"`
					} `json:"items"`
				} `json:"roadmap"`
			} `json:"workspace"`
		}
		if err := json.Unmarshal(data, &resp); err != nil {
			return nil, exitcode.General("parsing roadmap epics response", err)
		}

		for _, raw := range resp.Workspace.Roadmap.Items.Nodes {
			if epic, ok := parseRoadmapItem(raw); ok {
				result = append(result, epic)
			}
		}

		if !resp.Workspace.Roadmap.Items.PageInfo.HasNextPage {
			break
		}
		cursor = &resp.Workspace.Roadmap.Items.PageInfo.EndCursor
	}

	return result, nil
}

// parseRoadmapItem parses a single roadmap item node into a CachedEpic.
// Returns false if the node is not an epic (e.g. a Project).
func parseRoadmapItem(raw json.RawMessage) (CachedEpic, bool) {
	var typed struct {
		TypeName string `json:"__typename"`
	}
	if err := json.Unmarshal(raw, &typed); err != nil {
		return CachedEpic{}, false
	}

	switch typed.TypeName {
	case "ZenhubEpic":
		var ze struct {
			ID    string `json:"id"`
			Title string `json:"title"`
		}
		if err := json.Unmarshal(raw, &ze); err != nil {
			return CachedEpic{}, false
		}
		return CachedEpic{
			ID:    ze.ID,
			Title: ze.Title,
			Type:  "zenhub",
		}, true

	case "Epic":
		var le struct {
			ID    string `json:"id"`
			Issue struct {
				Title      string `json:"title"`
				Number     int    `json:"number"`
				Repository struct {
					Name      string `json:"name"`
					OwnerName string `json:"ownerName"`
				} `json:"repository"`
			} `json:"issue"`
		}
		if err := json.Unmarshal(raw, &le); err != nil {
			return CachedEpic{}, false
		}
		return CachedEpic{
			ID:          le.ID,
			Title:       le.Issue.Title,
			Type:        "legacy",
			IssueNumber: le.Issue.Number,
			RepoName:    le.Issue.Repository.Name,
			RepoOwner:   le.Issue.Repository.OwnerName,
		}, true

	default:
		return CachedEpic{}, false
	}
}

// FetchEpicsIntoCache stores pre-fetched epic entries in the cache.
func FetchEpicsIntoCache(entries []CachedEpic, workspaceID string) error {
	return cache.Set(EpicCacheKey(workspaceID), entries)
}

// Epic resolves an epic identifier to an EpicResult. It checks aliases first,
// then attempts resolution by exact ID, exact title (case-insensitive), unique
// title substring, and owner/repo#number for legacy epics, using the cache
// with invalidate-on-miss semantics.
func Epic(client *api.Client, workspaceID string, identifier string, aliases map[string]string) (*EpicResult, error) {
	// Check aliases first — an alias maps to an epic ID or title, which we
	// then resolve normally.
	if aliases != nil {
		if target, ok := aliases[identifier]; ok {
			identifier = target
		}
	}

	key := EpicCacheKey(workspaceID)

	// Try cache first
	if entries, ok := cache.Get[[]CachedEpic](key); ok {
		if e, found := matchEpic(entries, identifier); found {
			return e, nil
		}
		// Check for ambiguity before refreshing
		if err := checkEpicAmbiguous(entries, identifier); err != nil {
			return nil, err
		}
	}

	// Cache miss — refresh from API
	entries, err := FetchEpics(client, workspaceID)
	if err != nil {
		return nil, err
	}

	if e, found := matchEpic(entries, identifier); found {
		return e, nil
	}

	if err := checkEpicAmbiguous(entries, identifier); err != nil {
		return nil, err
	}

	return nil, exitcode.NotFoundError(fmt.Sprintf("epic %q not found — run 'zh epic list' to see available epics", identifier))
}

// matchEpic attempts to match an epic by exact ID, exact title,
// unique title substring, or owner/repo#number for legacy epics.
func matchEpic(entries []CachedEpic, identifier string) (*EpicResult, bool) {
	idLower := strings.ToLower(identifier)

	// Exact ID match
	for _, e := range entries {
		if e.ID == identifier {
			return &EpicResult{ID: e.ID, Title: e.Title, Type: e.Type}, true
		}
	}

	// owner/repo#number match for legacy epics
	if ref := issueRefPattern.FindStringSubmatch(identifier); ref != nil {
		owner, repo, numStr := ref[1], ref[2], ref[3]
		for _, e := range entries {
			if e.Type != "legacy" {
				continue
			}
			numMatch := fmt.Sprintf("%d", e.IssueNumber) == numStr
			repoMatch := strings.EqualFold(e.RepoName, repo)
			ownerMatch := owner == "" || strings.EqualFold(e.RepoOwner, owner)
			if numMatch && repoMatch && ownerMatch {
				return &EpicResult{ID: e.ID, Title: e.Title, Type: e.Type}, true
			}
		}
	}

	// Exact title match (case-insensitive)
	for _, e := range entries {
		if strings.ToLower(e.Title) == idLower {
			return &EpicResult{ID: e.ID, Title: e.Title, Type: e.Type}, true
		}
	}

	// Unique substring match (case-insensitive)
	var matches []CachedEpic
	for _, e := range entries {
		if strings.Contains(strings.ToLower(e.Title), idLower) {
			matches = append(matches, e)
		}
	}
	if len(matches) == 1 {
		return &EpicResult{ID: matches[0].ID, Title: matches[0].Title, Type: matches[0].Type}, true
	}

	return nil, false
}

// checkEpicAmbiguous returns a descriptive error if the identifier matches
// multiple epics by title substring.
func checkEpicAmbiguous(entries []CachedEpic, identifier string) error {
	idLower := strings.ToLower(identifier)
	var matches []string
	for _, e := range entries {
		if strings.Contains(strings.ToLower(e.Title), idLower) {
			matches = append(matches, fmt.Sprintf("%s [%s]", e.Title, e.ID))
		}
	}
	if len(matches) > 1 {
		msg := fmt.Sprintf("epic %q is ambiguous — matches %d epics:\n", identifier, len(matches))
		for _, m := range matches {
			msg += "  - " + m + "\n"
		}
		msg += "\nUse a more specific name or the epic ID."
		return exitcode.Usage(msg)
	}
	return nil
}
