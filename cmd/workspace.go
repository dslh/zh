package cmd

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/dslh/zh/internal/api"
	"github.com/dslh/zh/internal/cache"
	"github.com/dslh/zh/internal/config"
	"github.com/dslh/zh/internal/exitcode"
	"github.com/dslh/zh/internal/output"
	"github.com/spf13/cobra"
)

// API response types

type workspaceNode struct {
	ID               string  `json:"id"`
	Name             string  `json:"name"`
	DisplayName      string  `json:"displayName"`
	Description      *string `json:"description"`
	IsFavorite       bool    `json:"isFavorite"`
	Private          bool    `json:"private"`
	ViewerPermission string  `json:"viewerPermission"`
	CreatedAt        string  `json:"createdAt"`
	UpdatedAt        string  `json:"updatedAt"`
	ReposConnection  *struct {
		TotalCount int `json:"totalCount"`
	} `json:"repositoriesConnection"`
	PipelinesConnection *struct {
		TotalCount int `json:"totalCount"`
	} `json:"pipelinesConnection"`
	Organization *orgNode `json:"zenhubOrganization"`
}

type orgNode struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type orgWithWorkspaces struct {
	ID         string         `json:"id"`
	Name       string         `json:"name"`
	Workspaces workspaceNodes `json:"workspaces"`
}

type workspaceNodes struct {
	Nodes []workspaceNode `json:"nodes"`
}

// cachedWorkspace is what we store in the cache for workspace resolution.
type cachedWorkspace struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	OrgName     string `json:"orgName"`
}

// Workspace detail types (for show command)

type workspaceDetail struct {
	ID               string            `json:"id"`
	Name             string            `json:"name"`
	DisplayName      string            `json:"displayName"`
	Description      *string           `json:"description"`
	Private          bool              `json:"private"`
	CreatedAt        string            `json:"createdAt"`
	UpdatedAt        string            `json:"updatedAt"`
	ViewerPermission string            `json:"viewerPermission"`
	IsFavorite       bool              `json:"isFavorite"`
	Organization     *orgNode          `json:"zenhubOrganization"`
	DefaultRepo      *repoRef          `json:"defaultRepository"`
	SprintConfig     *sprintConfigNode `json:"sprintConfig"`
	ActiveSprint     *sprintNode       `json:"activeSprint"`
	AvgVelocity      float64           `json:"averageSprintVelocity"`
	PipelinesConn    struct {
		TotalCount int            `json:"totalCount"`
		Nodes      []pipelineNode `json:"nodes"`
	} `json:"pipelinesConnection"`
	ReposConn struct {
		TotalCount int        `json:"totalCount"`
		Nodes      []repoNode `json:"nodes"`
	} `json:"repositoriesConnection"`
	PrioritiesConn struct {
		Nodes []priorityNode `json:"nodes"`
	} `json:"prioritiesConnection"`
}

type repoRef struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	OwnerName string `json:"ownerName"`
	GhID      int    `json:"ghId"`
}

type sprintConfigNode struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Kind         string `json:"kind"`
	Period       int    `json:"period"`
	StartDay     string `json:"startDay"`
	EndDay       string `json:"endDay"`
	TzIdentifier string `json:"tzIdentifier"`
}

type sprintNode struct {
	ID              string  `json:"id"`
	Name            string  `json:"name"`
	GeneratedName   string  `json:"generatedName"`
	State           string  `json:"state"`
	StartAt         string  `json:"startAt"`
	EndAt           string  `json:"endAt"`
	TotalPoints     float64 `json:"totalPoints"`
	CompletedPoints float64 `json:"completedPoints"`
}

type pipelineNode struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description *string `json:"description"`
}

type repoNode struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	OwnerName  string `json:"ownerName"`
	GhID       int    `json:"ghId"`
	IsPrivate  bool   `json:"isPrivate"`
	IsArchived bool   `json:"isArchived"`
}

type priorityNode struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
}

// Cached repo entry for repo name → GitHub ID resolution.
type cachedRepo struct {
	ID        string `json:"id"`
	GhID      int    `json:"ghId"`
	Name      string `json:"name"`
	OwnerName string `json:"ownerName"`
}

// GraphQL queries

const listWorkspacesQuery = `query ListWorkspaces {
  viewer {
    zenhubOrganizations(first: 50) {
      nodes {
        id
        name
        workspaces(first: 100) {
          nodes {
            id
            name
            displayName
            description
            isFavorite
            viewerPermission
            repositoriesConnection(first: 1) {
              totalCount
            }
            pipelinesConnection(first: 1) {
              totalCount
            }
          }
        }
      }
    }
  }
}`

const recentWorkspacesQuery = `query RecentWorkspaces {
  recentlyViewedWorkspaces(first: 50) {
    nodes {
      id
      name
      displayName
      description
      isFavorite
      viewerPermission
      zenhubOrganization {
        id
        name
      }
      repositoriesConnection(first: 1) {
        totalCount
      }
      pipelinesConnection(first: 1) {
        totalCount
      }
    }
  }
}`

const favoriteWorkspacesQuery = `query FavoriteWorkspaces {
  viewer {
    workspaceFavorites(first: 50) {
      nodes {
        id
        workspace {
          id
          name
          displayName
          description
          viewerPermission
          isFavorite
          zenhubOrganization {
            id
            name
          }
          repositoriesConnection(first: 1) {
            totalCount
          }
          pipelinesConnection(first: 1) {
            totalCount
          }
        }
      }
    }
  }
}`

const workspaceDetailQuery = `query GetWorkspace($workspaceId: ID!) {
  workspace(id: $workspaceId) {
    id
    name
    displayName
    description
    private
    createdAt
    updatedAt
    viewerPermission
    isFavorite
    zenhubOrganization {
      id
      name
    }
    defaultRepository {
      id
      name
      ownerName
      ghId
    }
    sprintConfig {
      id
      name
      kind
      period
      startDay
      endDay
      tzIdentifier
    }
    activeSprint {
      id
      name
      generatedName
      state
      startAt
      endAt
      totalPoints
      completedPoints
    }
    averageSprintVelocity
    pipelinesConnection(first: 50) {
      totalCount
      nodes {
        id
        name
        description
      }
    }
    repositoriesConnection(first: 100) {
      totalCount
      nodes {
        id
        name
        ownerName
        ghId
        isPrivate
        isArchived
      }
    }
    prioritiesConnection {
      nodes {
        id
        name
        color
      }
    }
  }
}`

const workspaceReposQuery = `query WorkspaceRepos($workspaceId: ID!, $first: Int!, $after: String) {
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
        isPrivate
        isArchived
      }
    }
  }
}`

// Commands

var workspaceCmd = &cobra.Command{
	Use:   "workspace",
	Short: "Workspace information and configuration",
	Long:  `List, view, and switch between ZenHub workspaces.`,
}

var (
	workspaceListFavorites bool
	workspaceListRecent    bool
)

var workspaceListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available workspaces",
	Long:  `List all ZenHub workspaces you have access to. Use --favorites or --recent to filter.`,
	RunE:  runWorkspaceList,
}

var workspaceShowCmd = &cobra.Command{
	Use:   "show [name]",
	Short: "Show workspace details",
	Long:  `Display details about a workspace. Defaults to the current workspace if no name is given.`,
	Args:  cobra.MaximumNArgs(1),
	RunE:  runWorkspaceShow,
}

var workspaceSwitchCmd = &cobra.Command{
	Use:   "switch <name>",
	Short: "Switch the default workspace",
	Long:  `Set a different workspace as the default for subsequent commands.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runWorkspaceSwitch,
}

var workspaceReposCmd = &cobra.Command{
	Use:   "repos",
	Short: "List repos connected to the workspace",
	Long:  `List all GitHub repositories connected to the current workspace.`,
	RunE:  runWorkspaceRepos,
}

func init() {
	workspaceListCmd.Flags().BoolVar(&workspaceListFavorites, "favorites", false, "Show only favorited workspaces")
	workspaceListCmd.Flags().BoolVar(&workspaceListRecent, "recent", false, "Show recently viewed workspaces")
	workspaceListCmd.MarkFlagsMutuallyExclusive("favorites", "recent")

	workspaceCmd.AddCommand(workspaceListCmd)
	workspaceCmd.AddCommand(workspaceShowCmd)
	workspaceCmd.AddCommand(workspaceSwitchCmd)
	workspaceCmd.AddCommand(workspaceReposCmd)
	rootCmd.AddCommand(workspaceCmd)
}

// apiNewFunc is the function used to create API clients. It can be replaced
// in tests to inject a mock server endpoint.
var apiNewFunc = api.New

// newClient creates an API client from config, wiring up verbose logging.
func newClient(cfg *config.Config, cmd *cobra.Command) *api.Client {
	var opts []api.Option
	if verbose {
		opts = append(opts, api.WithVerbose(func(format string, args ...any) {
			fmt.Fprintf(cmd.ErrOrStderr(), format, args...)
		}))
	}
	return apiNewFunc(cfg.APIKey, opts...)
}

// requireConfig loads config and validates that an API key is present.
func requireConfig() (*config.Config, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, exitcode.General("loading config", err)
	}
	if cfg.APIKey == "" {
		return nil, exitcode.Auth("no API key configured — set ZH_API_KEY or run zh to configure", nil)
	}
	return cfg, nil
}

// requireWorkspace loads config and validates that a workspace is set.
func requireWorkspace() (*config.Config, error) {
	cfg, err := requireConfig()
	if err != nil {
		return nil, err
	}
	if cfg.Workspace == "" {
		return nil, exitcode.Usage("no workspace configured — use 'zh workspace switch' to set one")
	}
	return cfg, nil
}

// fetchAllWorkspaces fetches workspaces from all orgs and returns a flat list
// with org name attached.
func fetchAllWorkspaces(client *api.Client) ([]workspaceNode, error) {
	data, err := client.Execute(listWorkspacesQuery, nil)
	if err != nil {
		return nil, exitcode.General("fetching workspaces", err)
	}

	var resp struct {
		Viewer struct {
			ZenhubOrganizations struct {
				Nodes []orgWithWorkspaces `json:"nodes"`
			} `json:"zenhubOrganizations"`
		} `json:"viewer"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, exitcode.General("parsing workspaces response", err)
	}

	var workspaces []workspaceNode
	for _, org := range resp.Viewer.ZenhubOrganizations.Nodes {
		for _, ws := range org.Workspaces.Nodes {
			ws.Organization = &orgNode{ID: org.ID, Name: org.Name}
			workspaces = append(workspaces, ws)
		}
	}
	return workspaces, nil
}

// cacheWorkspaces stores the workspace list in the cache.
func cacheWorkspaces(workspaces []workspaceNode) {
	var entries []cachedWorkspace
	for _, ws := range workspaces {
		orgName := ""
		if ws.Organization != nil {
			orgName = ws.Organization.Name
		}
		entries = append(entries, cachedWorkspace{
			ID:          ws.ID,
			Name:        ws.Name,
			DisplayName: ws.DisplayName,
			OrgName:     orgName,
		})
	}
	_ = cache.Set(cache.NewKey("workspaces"), entries)
}

// matchWorkspace finds a workspace by exact ID, exact name, or unique substring.
func matchWorkspace(entries []cachedWorkspace, name string) (*cachedWorkspace, bool) {
	nameLower := strings.ToLower(name)

	// Exact ID match
	for i, ws := range entries {
		if ws.ID == name {
			return &entries[i], true
		}
	}

	// Exact name match (case-insensitive)
	for i, ws := range entries {
		if strings.ToLower(ws.DisplayName) == nameLower || strings.ToLower(ws.Name) == nameLower {
			return &entries[i], true
		}
	}

	// Unique substring match (case-insensitive)
	var matches []cachedWorkspace
	for _, ws := range entries {
		if strings.Contains(strings.ToLower(ws.DisplayName), nameLower) ||
			strings.Contains(strings.ToLower(ws.Name), nameLower) {
			matches = append(matches, ws)
		}
	}

	if len(matches) == 1 {
		return &matches[0], true
	}
	if len(matches) > 1 {
		// Return false so the caller can provide a helpful error.
		// The ambiguity will be handled at the command level.
		return nil, false
	}

	return nil, false
}

// resolveWorkspaceOrAmbiguous resolves a workspace, providing a helpful error
// for ambiguous matches.
func resolveWorkspaceOrAmbiguous(client *api.Client, name string) (*cachedWorkspace, error) {
	key := cache.NewKey("workspaces")

	// First try from cache
	if entries, ok := cache.Get[[]cachedWorkspace](key); ok {
		if ws, found := matchWorkspace(entries, name); found {
			return ws, nil
		}
		// Check for ambiguity
		if err := checkAmbiguous(entries, name); err != nil {
			return nil, err
		}
	}

	// Refresh and retry
	workspaces, err := fetchAllWorkspaces(client)
	if err != nil {
		return nil, err
	}

	var entries []cachedWorkspace
	for _, ws := range workspaces {
		orgName := ""
		if ws.Organization != nil {
			orgName = ws.Organization.Name
		}
		entries = append(entries, cachedWorkspace{
			ID:          ws.ID,
			Name:        ws.Name,
			DisplayName: ws.DisplayName,
			OrgName:     orgName,
		})
	}
	_ = cache.Set(key, entries)

	if ws, found := matchWorkspace(entries, name); found {
		return ws, nil
	}

	// Check for ambiguity after refresh
	if err := checkAmbiguous(entries, name); err != nil {
		return nil, err
	}

	return nil, exitcode.NotFoundError(fmt.Sprintf("workspace %q not found — run 'zh workspace list' to see available workspaces", name))
}

func checkAmbiguous(entries []cachedWorkspace, name string) error {
	nameLower := strings.ToLower(name)
	var matches []string
	for _, ws := range entries {
		if strings.Contains(strings.ToLower(ws.DisplayName), nameLower) ||
			strings.Contains(strings.ToLower(ws.Name), nameLower) {
			entry := ws.DisplayName
			if ws.OrgName != "" {
				entry += " (" + ws.OrgName + ")"
			}
			matches = append(matches, entry)
		}
	}
	if len(matches) > 1 {
		msg := fmt.Sprintf("workspace %q is ambiguous — matches %d workspaces:\n", name, len(matches))
		for _, m := range matches {
			msg += "  - " + m + "\n"
		}
		msg += "\nUse a more specific name or the workspace ID."
		return exitcode.Usage(msg)
	}
	return nil
}

// runWorkspaceList implements `zh workspace list`.
func runWorkspaceList(cmd *cobra.Command, args []string) error {
	cfg, err := requireConfig()
	if err != nil {
		return err
	}

	client := newClient(cfg, cmd)
	w := cmd.OutOrStdout()

	if workspaceListRecent {
		return runWorkspaceListRecent(client, cfg, cmd)
	}

	if workspaceListFavorites {
		return runWorkspaceListFavorites(client, cfg, cmd)
	}

	workspaces, err := fetchAllWorkspaces(client)
	if err != nil {
		return err
	}

	// Cache the workspace list
	cacheWorkspaces(workspaces)

	if output.IsJSON(outputFormat) {
		return output.JSON(w, workspaces)
	}

	if len(workspaces) == 0 {
		fmt.Fprintln(w, "No workspaces found.")
		return nil
	}

	lw := output.NewListWriter(w, "ORGANIZATION", "WORKSPACE", "REPOS", "PIPELINES", "PERMISSION")
	for _, ws := range workspaces {
		org := output.TableMissing
		if ws.Organization != nil {
			org = ws.Organization.Name
		}

		name := ws.DisplayName
		if ws.ID == cfg.Workspace {
			name += " *"
		}

		repos := output.TableMissing
		if ws.ReposConnection != nil {
			repos = fmt.Sprintf("%d", ws.ReposConnection.TotalCount)
		}

		pipelines := output.TableMissing
		if ws.PipelinesConnection != nil {
			pipelines = fmt.Sprintf("%d", ws.PipelinesConnection.TotalCount)
		}

		perm := strings.ToLower(ws.ViewerPermission)

		lw.Row(org, name, repos, pipelines, perm)
	}

	lw.FlushWithFooter(fmt.Sprintf("Total: %d workspace(s)", len(workspaces)))
	return nil
}

func runWorkspaceListRecent(client *api.Client, cfg *config.Config, cmd *cobra.Command) error {
	data, err := client.Execute(recentWorkspacesQuery, nil)
	if err != nil {
		return exitcode.General("fetching recent workspaces", err)
	}

	var resp struct {
		RecentlyViewedWorkspaces struct {
			Nodes []workspaceNode `json:"nodes"`
		} `json:"recentlyViewedWorkspaces"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return exitcode.General("parsing recent workspaces", err)
	}

	workspaces := resp.RecentlyViewedWorkspaces.Nodes
	w := cmd.OutOrStdout()

	if output.IsJSON(outputFormat) {
		return output.JSON(w, workspaces)
	}

	if len(workspaces) == 0 {
		fmt.Fprintln(w, "No recently viewed workspaces.")
		return nil
	}

	lw := output.NewListWriter(w, "ORGANIZATION", "WORKSPACE", "REPOS", "PIPELINES", "PERMISSION")
	for _, ws := range workspaces {
		org := output.TableMissing
		if ws.Organization != nil {
			org = ws.Organization.Name
		}

		name := ws.DisplayName
		if ws.ID == cfg.Workspace {
			name += " *"
		}

		repos := output.TableMissing
		if ws.ReposConnection != nil {
			repos = fmt.Sprintf("%d", ws.ReposConnection.TotalCount)
		}

		pipelines := output.TableMissing
		if ws.PipelinesConnection != nil {
			pipelines = fmt.Sprintf("%d", ws.PipelinesConnection.TotalCount)
		}

		perm := strings.ToLower(ws.ViewerPermission)

		lw.Row(org, name, repos, pipelines, perm)
	}

	lw.FlushWithFooter(fmt.Sprintf("Total: %d workspace(s)", len(workspaces)))
	return nil
}

func runWorkspaceListFavorites(client *api.Client, cfg *config.Config, cmd *cobra.Command) error {
	data, err := client.Execute(favoriteWorkspacesQuery, nil)
	if err != nil {
		return exitcode.General("fetching favorite workspaces", err)
	}

	var resp struct {
		Viewer struct {
			WorkspaceFavorites struct {
				Nodes []struct {
					ID        string        `json:"id"`
					Workspace workspaceNode `json:"workspace"`
				} `json:"nodes"`
			} `json:"workspaceFavorites"`
		} `json:"viewer"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return exitcode.General("parsing favorite workspaces", err)
	}

	w := cmd.OutOrStdout()

	var workspaces []workspaceNode
	for _, fav := range resp.Viewer.WorkspaceFavorites.Nodes {
		workspaces = append(workspaces, fav.Workspace)
	}

	if output.IsJSON(outputFormat) {
		return output.JSON(w, workspaces)
	}

	if len(workspaces) == 0 {
		fmt.Fprintln(w, "No favorite workspaces.")
		return nil
	}

	lw := output.NewListWriter(w, "ORGANIZATION", "WORKSPACE", "REPOS", "PIPELINES", "PERMISSION")
	for _, ws := range workspaces {
		org := output.TableMissing
		if ws.Organization != nil {
			org = ws.Organization.Name
		}

		name := ws.DisplayName
		if ws.ID == cfg.Workspace {
			name += " *"
		}

		repos := output.TableMissing
		if ws.ReposConnection != nil {
			repos = fmt.Sprintf("%d", ws.ReposConnection.TotalCount)
		}

		pipelines := output.TableMissing
		if ws.PipelinesConnection != nil {
			pipelines = fmt.Sprintf("%d", ws.PipelinesConnection.TotalCount)
		}

		perm := strings.ToLower(ws.ViewerPermission)

		lw.Row(org, name, repos, pipelines, perm)
	}

	lw.FlushWithFooter(fmt.Sprintf("Total: %d workspace(s)", len(workspaces)))
	return nil
}

// runWorkspaceShow implements `zh workspace show [name]`.
func runWorkspaceShow(cmd *cobra.Command, args []string) error {
	cfg, err := requireConfig()
	if err != nil {
		return err
	}

	client := newClient(cfg, cmd)

	// Determine workspace ID
	workspaceID := cfg.Workspace
	if len(args) > 0 {
		ws, err := resolveWorkspaceOrAmbiguous(client, args[0])
		if err != nil {
			return err
		}
		workspaceID = ws.ID
	}

	if workspaceID == "" {
		return exitcode.Usage("no workspace specified and no default configured — use 'zh workspace switch' to set one")
	}

	data, err := client.Execute(workspaceDetailQuery, map[string]any{
		"workspaceId": workspaceID,
	})
	if err != nil {
		return exitcode.General("fetching workspace details", err)
	}

	var resp struct {
		Workspace workspaceDetail `json:"workspace"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return exitcode.General("parsing workspace details", err)
	}

	ws := resp.Workspace
	w := cmd.OutOrStdout()

	if output.IsJSON(outputFormat) {
		return output.JSON(w, ws)
	}

	// Cache repos for later use by other commands
	cacheReposFromDetail(&ws)

	d := output.NewDetailWriter(w, "WORKSPACE", ws.DisplayName)

	org := output.DetailMissing
	if ws.Organization != nil {
		org = ws.Organization.Name
	}

	visibility := "Public"
	if ws.Private {
		visibility = "Private"
	}

	fields := []output.KeyValue{
		output.KV("Organization", org),
		output.KV("ID", output.Cyan(ws.ID)),
		output.KV("Permission", strings.ToLower(ws.ViewerPermission)),
		output.KV("Visibility", visibility),
	}

	if ws.CreatedAt != "" {
		if t, err := time.Parse(time.RFC3339, ws.CreatedAt); err == nil {
			fields = append(fields, output.KV("Created", output.FormatDate(t)))
		}
	}

	if ws.UpdatedAt != "" {
		if t, err := time.Parse(time.RFC3339, ws.UpdatedAt); err == nil {
			fields = append(fields, output.KV("Last updated", output.FormatDate(t)))
		}
	}

	d.Fields(fields)

	// Sprint configuration section
	d.Section("SPRINT CONFIGURATION")
	if ws.SprintConfig != nil {
		sc := ws.SprintConfig
		cadence := fmt.Sprintf("%d-week sprints (%s)", sc.Period, sc.Kind)
		schedule := fmt.Sprintf("%s → %s", formatDay(sc.StartDay), formatDay(sc.EndDay))
		fmt.Fprintf(w, "%-16s%s\n", "Cadence:", cadence)
		fmt.Fprintf(w, "%-16s%s\n", "Schedule:", schedule)
		fmt.Fprintf(w, "%-16s%s\n", "Timezone:", sc.TzIdentifier)

		if ws.ActiveSprint != nil {
			sp := ws.ActiveSprint
			startAt, _ := time.Parse(time.RFC3339, sp.StartAt)
			endAt, _ := time.Parse(time.RFC3339, sp.EndAt)
			fmt.Fprintln(w)
			fmt.Fprintf(w, "%-16s%s (%s)\n", "Active sprint:", sp.Name, output.FormatDateRange(startAt, endAt))
			if sp.TotalPoints > 0 {
				fmt.Fprintf(w, "%-16s%s\n", "", output.FormatProgress(int(sp.CompletedPoints), int(sp.TotalPoints)))
			}
		}

		if ws.AvgVelocity > 0 {
			fmt.Fprintf(w, "%-16s%.0f pts/sprint (avg)\n", "Velocity:", ws.AvgVelocity)
		}
	} else {
		fmt.Fprintln(w, "Sprints are not configured for this workspace.")
	}

	// Summary section
	d.Section("SUMMARY")

	repoCount := ws.ReposConn.TotalCount
	archivedCount := 0
	for _, r := range ws.ReposConn.Nodes {
		if r.IsArchived {
			archivedCount++
		}
	}
	repoSummary := fmt.Sprintf("%d", repoCount)
	if archivedCount > 0 {
		repoSummary += fmt.Sprintf(" (%d archived)", archivedCount)
	}

	fmt.Fprintf(w, "%-16s%s\n", "Repositories:", repoSummary)
	fmt.Fprintf(w, "%-16s%d\n", "Pipelines:", ws.PipelinesConn.TotalCount)

	priCount := len(ws.PrioritiesConn.Nodes)
	if priCount > 0 {
		fmt.Fprintf(w, "%-16s%d defined\n", "Priorities:", priCount)
	}

	if ws.DefaultRepo != nil {
		fmt.Fprintln(w)
		fmt.Fprintf(w, "%-16s%s/%s\n", "Default repo:", ws.DefaultRepo.OwnerName, ws.DefaultRepo.Name)
	}

	return nil
}

// cacheReposFromDetail stores repo data from workspace detail in the cache.
func cacheReposFromDetail(ws *workspaceDetail) {
	var repos []cachedRepo
	for _, r := range ws.ReposConn.Nodes {
		repos = append(repos, cachedRepo{
			ID:        r.ID,
			GhID:      r.GhID,
			Name:      r.Name,
			OwnerName: r.OwnerName,
		})
	}
	if len(repos) > 0 {
		_ = cache.Set(cache.NewScopedKey("repos", ws.ID), repos)
	}
}

// runWorkspaceSwitch implements `zh workspace switch <name>`.
func runWorkspaceSwitch(cmd *cobra.Command, args []string) error {
	cfg, err := requireConfig()
	if err != nil {
		return err
	}

	client := newClient(cfg, cmd)
	name := args[0]

	ws, err := resolveWorkspaceOrAmbiguous(client, name)
	if err != nil {
		return err
	}

	// Already current?
	if ws.ID == cfg.Workspace {
		fmt.Fprintf(cmd.OutOrStdout(), "Already using workspace %q", ws.DisplayName)
		if ws.OrgName != "" {
			fmt.Fprintf(cmd.OutOrStdout(), " (%s)", ws.OrgName)
		}
		fmt.Fprintln(cmd.OutOrStdout())
		return nil
	}

	// Clear workspace-scoped caches for the old workspace
	if cfg.Workspace != "" {
		_ = cache.ClearWorkspace(cfg.Workspace)
	}

	// Update config
	cfg.Workspace = ws.ID
	if err := config.Write(cfg); err != nil {
		return exitcode.General("saving config", err)
	}

	out := fmt.Sprintf("Switched to workspace %q", ws.DisplayName)
	if ws.OrgName != "" {
		out += fmt.Sprintf(" (%s)", ws.OrgName)
	}
	fmt.Fprintln(cmd.OutOrStdout(), out)
	return nil
}

// runWorkspaceRepos implements `zh workspace repos`.
func runWorkspaceRepos(cmd *cobra.Command, args []string) error {
	cfg, err := requireWorkspace()
	if err != nil {
		return err
	}

	client := newClient(cfg, cmd)
	w := cmd.OutOrStdout()

	repos, err := fetchWorkspaceRepos(client, cfg.Workspace)
	if err != nil {
		return err
	}

	// Cache repos for issue resolution
	var cached []cachedRepo
	for _, r := range repos {
		cached = append(cached, cachedRepo{
			ID:        r.ID,
			GhID:      r.GhID,
			Name:      r.Name,
			OwnerName: r.OwnerName,
		})
	}
	_ = cache.Set(cache.NewScopedKey("repos", cfg.Workspace), cached)

	if output.IsJSON(outputFormat) {
		return output.JSON(w, repos)
	}

	if len(repos) == 0 {
		fmt.Fprintln(w, "No repositories connected to this workspace.")
		return nil
	}

	lw := output.NewListWriter(w, "REPO", "GITHUB ID", "PRIVATE", "ARCHIVED")
	for _, r := range repos {
		name := r.OwnerName + "/" + r.Name
		private := "no"
		if r.IsPrivate {
			private = "yes"
		}
		archived := "no"
		if r.IsArchived {
			archived = "yes"
		}
		lw.Row(name, fmt.Sprintf("%d", r.GhID), private, archived)
	}
	lw.FlushWithFooter(fmt.Sprintf("Total: %d repo(s)", len(repos)))
	return nil
}

// fetchWorkspaceRepos fetches all repos for a workspace, handling pagination.
func fetchWorkspaceRepos(client *api.Client, workspaceID string) ([]repoNode, error) {
	var allRepos []repoNode
	var cursor *string

	for {
		vars := map[string]any{
			"workspaceId": workspaceID,
			"first":       100,
		}
		if cursor != nil {
			vars["after"] = *cursor
		}

		data, err := client.Execute(workspaceReposQuery, vars)
		if err != nil {
			return nil, exitcode.General("fetching workspace repos", err)
		}

		var resp struct {
			Workspace struct {
				ReposConn struct {
					TotalCount int `json:"totalCount"`
					PageInfo   struct {
						HasNextPage bool   `json:"hasNextPage"`
						EndCursor   string `json:"endCursor"`
					} `json:"pageInfo"`
					Nodes []repoNode `json:"nodes"`
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

	return allRepos, nil
}

// formatDay formats a day-of-week string to title case.
func formatDay(day string) string {
	if len(day) == 0 {
		return day
	}
	return strings.ToUpper(day[:1]) + strings.ToLower(day[1:])
}
