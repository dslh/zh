package cmd

import (
	"github.com/dslh/zh/internal/cache"
	"github.com/dslh/zh/internal/config"
	"github.com/dslh/zh/internal/resolve"
	"github.com/spf13/cobra"
)

// completePipelineNames returns cached pipeline names for shell completion.
func completePipelineNames(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	cfg, wsID := completionConfig()
	if wsID == "" {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	entries, ok := cache.Get[[]resolve.CachedPipeline](resolve.PipelineCacheKey(wsID))
	if !ok {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	var names []string
	for _, p := range entries {
		names = append(names, p.Name)
	}

	// Include pipeline aliases
	if cfg != nil {
		for alias := range cfg.Aliases.Pipelines {
			names = append(names, alias)
		}
	}

	return names, cobra.ShellCompDirectiveNoFileComp
}

// completeSprintNames returns cached sprint names for shell completion.
func completeSprintNames(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	_, wsID := completionConfig()
	if wsID == "" {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	entries, ok := cache.Get[[]resolve.CachedSprint](resolve.SprintCacheKey(wsID))
	if !ok {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	names := []string{"current", "next", "previous"}
	for _, s := range entries {
		names = append(names, s.DisplayName())
	}

	return names, cobra.ShellCompDirectiveNoFileComp
}

// completeEpicNames returns cached epic titles for shell completion.
func completeEpicNames(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	cfg, wsID := completionConfig()
	if wsID == "" {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	entries, ok := cache.Get[[]resolve.CachedEpic](resolve.EpicCacheKey(wsID))
	if !ok {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	var names []string
	for _, e := range entries {
		names = append(names, e.Title)
	}

	// Include epic aliases
	if cfg != nil {
		for alias := range cfg.Aliases.Epics {
			names = append(names, alias)
		}
	}

	return names, cobra.ShellCompDirectiveNoFileComp
}

// completeWorkspaceNames returns cached workspace names for shell completion.
func completeWorkspaceNames(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	type cachedWS struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		DisplayName string `json:"displayName"`
		OrgName     string `json:"orgName"`
	}

	entries, ok := cache.Get[[]cachedWS](cache.NewKey("workspaces"))
	if !ok {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	var names []string
	for _, ws := range entries {
		name := ws.DisplayName
		if name == "" {
			name = ws.Name
		}
		names = append(names, name)
	}

	return names, cobra.ShellCompDirectiveNoFileComp
}

// completeRepoNames returns cached repo names for shell completion.
func completeRepoNames(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	_, wsID := completionConfig()
	if wsID == "" {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	entries, ok := cache.Get[[]resolve.CachedRepo](resolve.RepoCacheKey(wsID))
	if !ok {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	var names []string
	for _, r := range entries {
		names = append(names, r.Name)
	}

	return names, cobra.ShellCompDirectiveNoFileComp
}

// completeLabelNames returns cached label names for shell completion.
func completeLabelNames(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	_, wsID := completionConfig()
	if wsID == "" {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	entries, ok := cache.Get[[]resolve.CachedLabel](resolve.LabelCacheKey(wsID))
	if !ok {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	var names []string
	for _, l := range entries {
		names = append(names, l.Name)
	}

	return names, cobra.ShellCompDirectiveNoFileComp
}

// completePriorityNames returns cached priority names for shell completion.
func completePriorityNames(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	_, wsID := completionConfig()
	if wsID == "" {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	entries, ok := cache.Get[[]resolve.CachedPriority](resolve.PriorityCacheKey(wsID))
	if !ok {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	var names []string
	for _, p := range entries {
		names = append(names, p.Name)
	}

	return names, cobra.ShellCompDirectiveNoFileComp
}

// completeEpicStates returns valid epic states for shell completion.
func completeEpicStates(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return []string{"open", "todo", "in_progress", "closed"}, cobra.ShellCompDirectiveNoFileComp
}

// completePositionValues returns valid position values for shell completion.
func completePositionValues(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return []string{"top", "bottom"}, cobra.ShellCompDirectiveNoFileComp
}

// completeOutputFormats returns valid output format values for shell completion.
func completeOutputFormats(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return []string{"json"}, cobra.ShellCompDirectiveNoFileComp
}

// completionConfig loads config and returns the workspace ID for use in
// completion functions. Returns zero values on failure — completions are
// best-effort and should never error.
func completionConfig() (*config.Config, string) {
	cfg, err := config.Load()
	if err != nil {
		return nil, ""
	}
	return cfg, cfg.Workspace
}
