package resolve

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/dslh/zh/internal/api"
	"github.com/dslh/zh/internal/cache"
	"github.com/dslh/zh/internal/exitcode"
)

// CachedSprint is the shape stored in the sprint cache.
type CachedSprint struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	GeneratedName string `json:"generatedName"`
	State         string `json:"state"` // "OPEN" or "CLOSED"
	StartAt       string `json:"startAt"`
	EndAt         string `json:"endAt"`
}

// DisplayName returns the sprint's display name: the custom name if set,
// otherwise the generated name.
func (s *CachedSprint) DisplayName() string {
	if s.Name != "" {
		return s.Name
	}
	return s.GeneratedName
}

// SprintResult is the resolved sprint returned to callers.
type SprintResult struct {
	ID   string
	Name string // display name (custom or generated)
}

// SprintCacheKey returns the cache key for sprint data scoped to a workspace.
func SprintCacheKey(workspaceID string) cache.Key {
	return cache.NewScopedKey("sprints", workspaceID)
}

const listSprintsQuery = `query ListSprints($workspaceId: ID!, $first: Int!, $after: String) {
  workspace(id: $workspaceId) {
    sprints(first: $first, after: $after, orderBy: {field: START_AT, direction: DESC}) {
      totalCount
      pageInfo {
        hasNextPage
        endCursor
      }
      nodes {
        id
        name
        generatedName
        state
        startAt
        endAt
      }
    }
    activeSprint {
      id
    }
    upcomingSprint {
      id
    }
    previousSprint {
      id
    }
  }
}`

// sprintAccessors holds the IDs from the workspace's convenience sprint accessors.
type sprintAccessors struct {
	ActiveID   string
	UpcomingID string
	PreviousID string
}

// SprintAccessorsCacheKey returns the cache key for sprint accessor data.
func SprintAccessorsCacheKey(workspaceID string) cache.Key {
	return cache.NewScopedKey("sprint-accessors", workspaceID)
}

// FetchSprints fetches all sprints for a workspace from the API and updates
// the cache. Returns the cached sprint entries.
func FetchSprints(client *api.Client, workspaceID string) ([]CachedSprint, error) {
	var allSprints []CachedSprint
	var cursor *string
	var accessors sprintAccessors

	for page := 0; ; page++ {
		vars := map[string]any{
			"workspaceId": workspaceID,
			"first":       100,
		}
		if cursor != nil {
			vars["after"] = *cursor
		}

		data, err := client.Execute(listSprintsQuery, vars)
		if err != nil {
			return nil, exitcode.General("fetching sprints", err)
		}

		var resp struct {
			Workspace struct {
				Sprints struct {
					PageInfo struct {
						HasNextPage bool   `json:"hasNextPage"`
						EndCursor   string `json:"endCursor"`
					} `json:"pageInfo"`
					Nodes []CachedSprint `json:"nodes"`
				} `json:"sprints"`
				ActiveSprint   *struct{ ID string } `json:"activeSprint"`
				UpcomingSprint *struct{ ID string } `json:"upcomingSprint"`
				PreviousSprint *struct{ ID string } `json:"previousSprint"`
			} `json:"workspace"`
		}
		if err := json.Unmarshal(data, &resp); err != nil {
			return nil, exitcode.General("parsing sprints response", err)
		}

		allSprints = append(allSprints, resp.Workspace.Sprints.Nodes...)

		// Capture accessors from first page only
		if page == 0 {
			if resp.Workspace.ActiveSprint != nil {
				accessors.ActiveID = resp.Workspace.ActiveSprint.ID
			}
			if resp.Workspace.UpcomingSprint != nil {
				accessors.UpcomingID = resp.Workspace.UpcomingSprint.ID
			}
			if resp.Workspace.PreviousSprint != nil {
				accessors.PreviousID = resp.Workspace.PreviousSprint.ID
			}
		}

		if !resp.Workspace.Sprints.PageInfo.HasNextPage {
			break
		}
		cursor = &resp.Workspace.Sprints.PageInfo.EndCursor
	}

	_ = cache.Set(SprintCacheKey(workspaceID), allSprints)
	_ = cache.Set(SprintAccessorsCacheKey(workspaceID), accessors)
	return allSprints, nil
}

// FetchSprintsIntoCache stores pre-fetched sprint entries in the cache.
func FetchSprintsIntoCache(entries []CachedSprint, workspaceID string) error {
	return cache.Set(SprintCacheKey(workspaceID), entries)
}

// Sprint resolves a sprint identifier to a SprintResult. It supports:
//   - ZenHub ID: exact match
//   - Exact name match (case-insensitive, checking both custom and generated names)
//   - Unique substring match (case-insensitive)
//   - Relative references: "current", "next", "previous"
//
// Uses the cache with invalidate-on-miss semantics.
func Sprint(client *api.Client, workspaceID string, identifier string) (*SprintResult, error) {
	// Handle relative references
	idLower := strings.ToLower(identifier)
	if idLower == "current" || idLower == "next" || idLower == "previous" {
		return resolveRelativeSprint(client, workspaceID, idLower)
	}

	key := SprintCacheKey(workspaceID)

	// Try cache first
	if entries, ok := cache.Get[[]CachedSprint](key); ok {
		if s, found := matchSprint(entries, identifier); found {
			return s, nil
		}
		// Check for ambiguity before refreshing
		if err := checkSprintAmbiguous(entries, identifier); err != nil {
			return nil, err
		}
	}

	// Cache miss — refresh from API
	entries, err := FetchSprints(client, workspaceID)
	if err != nil {
		return nil, err
	}

	if s, found := matchSprint(entries, identifier); found {
		return s, nil
	}

	if err := checkSprintAmbiguous(entries, identifier); err != nil {
		return nil, err
	}

	return nil, exitcode.NotFoundError(fmt.Sprintf("sprint %q not found — run 'zh sprint list' to see available sprints", identifier))
}

// resolveRelativeSprint resolves "current", "next", or "previous" using
// cached sprint accessors (or fetching fresh data).
func resolveRelativeSprint(client *api.Client, workspaceID string, relative string) (*SprintResult, error) {
	accessorsKey := SprintAccessorsCacheKey(workspaceID)
	sprintsKey := SprintCacheKey(workspaceID)

	// Try resolving from cache first
	accessors, aOK := cache.Get[sprintAccessors](accessorsKey)
	sprints, sOK := cache.Get[[]CachedSprint](sprintsKey)

	if !aOK || !sOK {
		// Need to fetch fresh data
		var err error
		sprints, err = FetchSprints(client, workspaceID)
		if err != nil {
			return nil, err
		}
		accessors, _ = cache.Get[sprintAccessors](accessorsKey)
	}

	var targetID string
	switch relative {
	case "current":
		targetID = accessors.ActiveID
		if targetID == "" {
			// Fallback: find an open sprint that covers now
			targetID = findActiveByDate(sprints)
		}
		if targetID == "" {
			return nil, exitcode.NotFoundError("no active sprint — the workspace may not have sprints configured, or no sprint is currently in progress")
		}
	case "next":
		targetID = accessors.UpcomingID
		if targetID == "" {
			return nil, exitcode.NotFoundError("no upcoming sprint found")
		}
	case "previous":
		targetID = accessors.PreviousID
		if targetID == "" {
			return nil, exitcode.NotFoundError("no previous sprint found")
		}
	}

	for _, s := range sprints {
		if s.ID == targetID {
			return &SprintResult{ID: s.ID, Name: s.DisplayName()}, nil
		}
	}

	return nil, exitcode.NotFoundError(fmt.Sprintf("sprint %q not found in cached sprint list", relative))
}

// findActiveByDate returns the ID of the first open sprint whose date range
// covers the current time, or "" if none is found.
func findActiveByDate(sprints []CachedSprint) string {
	now := time.Now()
	for _, s := range sprints {
		if s.State != "OPEN" {
			continue
		}
		start, err1 := time.Parse(time.RFC3339, s.StartAt)
		end, err2 := time.Parse(time.RFC3339, s.EndAt)
		if err1 != nil || err2 != nil {
			continue
		}
		if !now.Before(start) && now.Before(end) {
			return s.ID
		}
	}
	return ""
}

// matchSprint attempts to match a sprint by exact ID, exact name,
// or unique substring.
func matchSprint(entries []CachedSprint, identifier string) (*SprintResult, bool) {
	idLower := strings.ToLower(identifier)

	// Exact ID match
	for _, s := range entries {
		if s.ID == identifier {
			return &SprintResult{ID: s.ID, Name: s.DisplayName()}, true
		}
	}

	// Exact name match (case-insensitive, check both names)
	for _, s := range entries {
		if strings.ToLower(s.DisplayName()) == idLower {
			return &SprintResult{ID: s.ID, Name: s.DisplayName()}, true
		}
		// Also check the other name (if custom name set, also match generated)
		if s.Name != "" && strings.ToLower(s.GeneratedName) == idLower {
			return &SprintResult{ID: s.ID, Name: s.DisplayName()}, true
		}
	}

	// Unique substring match (case-insensitive, across both name fields)
	var matches []CachedSprint
	for _, s := range entries {
		display := strings.ToLower(s.DisplayName())
		gen := strings.ToLower(s.GeneratedName)
		if strings.Contains(display, idLower) || (s.Name != "" && strings.Contains(gen, idLower)) {
			matches = append(matches, s)
		}
	}
	if len(matches) == 1 {
		return &SprintResult{ID: matches[0].ID, Name: matches[0].DisplayName()}, true
	}

	return nil, false
}

// checkSprintAmbiguous returns a descriptive error if the identifier matches
// multiple sprints.
func checkSprintAmbiguous(entries []CachedSprint, identifier string) error {
	idLower := strings.ToLower(identifier)
	var matches []string
	for _, s := range entries {
		display := strings.ToLower(s.DisplayName())
		gen := strings.ToLower(s.GeneratedName)
		if strings.Contains(display, idLower) || (s.Name != "" && strings.Contains(gen, idLower)) {
			matches = append(matches, fmt.Sprintf("%s [%s]", s.DisplayName(), s.ID))
		}
	}
	if len(matches) > 1 {
		msg := fmt.Sprintf("sprint %q is ambiguous — matches %d sprints:\n", identifier, len(matches))
		for _, m := range matches {
			msg += "  - " + m + "\n"
		}
		msg += "\nUse a more specific name or the sprint ID."
		return exitcode.Usage(msg)
	}
	return nil
}
