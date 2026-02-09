// Package resolve handles entity identifier resolution for ZenHub resources.
//
// It implements the resolution patterns described in the spec: exact ID match,
// exact name match (case-insensitive), unique substring match, and alias lookup.
// All resolvers use the cache with invalidate-on-miss semantics.
package resolve

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/dslh/zh/internal/api"
	"github.com/dslh/zh/internal/cache"
	"github.com/dslh/zh/internal/exitcode"
)

// CachedPipeline is the shape stored in the pipeline cache.
type CachedPipeline struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// PipelineResult is the resolved pipeline returned to callers.
type PipelineResult struct {
	ID   string
	Name string
}

const listPipelinesQuery = `query ListPipelines($workspaceId: ID!) {
  workspace(id: $workspaceId) {
    pipelinesConnection(first: 50) {
      nodes {
        id
        name
      }
    }
  }
}`

// FetchPipelines fetches all pipelines for a workspace from the API and
// updates the cache. Returns the cached pipeline entries.
func FetchPipelines(client *api.Client, workspaceID string) ([]CachedPipeline, error) {
	data, err := client.Execute(listPipelinesQuery, map[string]any{
		"workspaceId": workspaceID,
	})
	if err != nil {
		return nil, exitcode.General("fetching pipelines", err)
	}

	var resp struct {
		Workspace struct {
			PipelinesConnection struct {
				Nodes []CachedPipeline `json:"nodes"`
			} `json:"pipelinesConnection"`
		} `json:"workspace"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, exitcode.General("parsing pipelines response", err)
	}

	pipelines := resp.Workspace.PipelinesConnection.Nodes
	_ = cache.Set(PipelineCacheKey(workspaceID), pipelines)
	return pipelines, nil
}

// FetchPipelinesIntoCache stores pre-fetched pipeline entries in the cache.
// This is used by commands that already have pipeline data (e.g. pipeline list)
// to populate the cache without an extra API call.
func FetchPipelinesIntoCache(entries []CachedPipeline, workspaceID string) error {
	return cache.Set(PipelineCacheKey(workspaceID), entries)
}

// PipelineCacheKey returns the cache key for pipeline data scoped to a workspace.
func PipelineCacheKey(workspaceID string) cache.Key {
	return cache.NewScopedKey("pipelines", workspaceID)
}

// Pipeline resolves a pipeline identifier to a PipelineResult. It checks
// aliases first, then attempts resolution by exact ID, exact name
// (case-insensitive), and unique substring match, using the cache with
// invalidate-on-miss semantics.
func Pipeline(client *api.Client, workspaceID string, identifier string, aliases map[string]string) (*PipelineResult, error) {
	// Check aliases first — an alias maps to a pipeline name, which we
	// then resolve normally.
	if aliases != nil {
		if target, ok := aliases[identifier]; ok {
			identifier = target
		}
	}

	key := PipelineCacheKey(workspaceID)

	// Try cache first
	if entries, ok := cache.Get[[]CachedPipeline](key); ok {
		if p, found := matchPipeline(entries, identifier); found {
			return p, nil
		}
		// Check for ambiguity before refreshing — if it's ambiguous in cache,
		// it'll still be ambiguous after refresh.
		if err := checkPipelineAmbiguous(entries, identifier); err != nil {
			return nil, err
		}
	}

	// Cache miss — refresh from API
	entries, err := FetchPipelines(client, workspaceID)
	if err != nil {
		return nil, err
	}

	if p, found := matchPipeline(entries, identifier); found {
		return p, nil
	}

	if err := checkPipelineAmbiguous(entries, identifier); err != nil {
		return nil, err
	}

	return nil, exitcode.NotFoundError(fmt.Sprintf("pipeline %q not found — run 'zh pipeline list' to see available pipelines", identifier))
}

// matchPipeline attempts to match a pipeline by exact ID, exact name,
// or unique substring.
func matchPipeline(entries []CachedPipeline, identifier string) (*PipelineResult, bool) {
	idLower := strings.ToLower(identifier)

	// Exact ID match
	for _, p := range entries {
		if p.ID == identifier {
			return &PipelineResult{ID: p.ID, Name: p.Name}, true
		}
	}

	// Exact name match (case-insensitive)
	for _, p := range entries {
		if strings.ToLower(p.Name) == idLower {
			return &PipelineResult{ID: p.ID, Name: p.Name}, true
		}
	}

	// Unique substring match (case-insensitive)
	var matches []CachedPipeline
	for _, p := range entries {
		if strings.Contains(strings.ToLower(p.Name), idLower) {
			matches = append(matches, p)
		}
	}
	if len(matches) == 1 {
		return &PipelineResult{ID: matches[0].ID, Name: matches[0].Name}, true
	}

	return nil, false
}

// checkPipelineAmbiguous returns a descriptive error if the identifier matches
// multiple pipelines.
func checkPipelineAmbiguous(entries []CachedPipeline, identifier string) error {
	idLower := strings.ToLower(identifier)
	var matches []string
	for _, p := range entries {
		if strings.Contains(strings.ToLower(p.Name), idLower) {
			matches = append(matches, fmt.Sprintf("%s [%s]", p.Name, p.ID))
		}
	}
	if len(matches) > 1 {
		msg := fmt.Sprintf("pipeline %q is ambiguous — matches %d pipelines:\n", identifier, len(matches))
		for _, m := range matches {
			msg += "  - " + m + "\n"
		}
		msg += "\nUse a more specific name or the pipeline ID."
		return exitcode.Usage(msg)
	}
	return nil
}
