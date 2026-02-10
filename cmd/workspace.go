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
	"github.com/dslh/zh/internal/gh"
	"github.com/dslh/zh/internal/output"
	"github.com/dslh/zh/internal/resolve"
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

// cachedRepo is an alias for resolve.CachedRepo for convenience within this file.
type cachedRepo = resolve.CachedRepo

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

// GitHub enrichment query for repos
const githubRepoQuery = `query RepoDetails($owner: String!, $name: String!) {
  repository(owner: $owner, name: $name) {
    description
    primaryLanguage { name }
    stargazerCount
  }
}`

// githubRepoInfo holds GitHub-enriched repo data.
type githubRepoInfo struct {
	Description string `json:"description"`
	Language    string `json:"language"`
	Stars       int    `json:"stars"`
}

// Commands

var workspaceCmd = &cobra.Command{
	Use:   "workspace",
	Short: "Workspace information and configuration",
	Long:  `List, view, and switch between ZenHub workspaces.`,
}

var (
	workspaceListFavorites   bool
	workspaceListRecent      bool
	workspaceReposGitHub     bool
	workspaceStatsSprints    int
	workspaceStatsDays       int
	workspaceShowInteractive bool
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
	Long: `Display details about a workspace. Defaults to the current workspace if no name is given.

Use --interactive to select a workspace from a list.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runWorkspaceShow,
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
	Long:  `List all GitHub repositories connected to the current workspace. Use --github to include description, language, and stars from GitHub.`,
	RunE:  runWorkspaceRepos,
}

var workspaceStatsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show workspace metrics",
	Long:  `Show workspace metrics including velocity trends, cycle time, and pipeline distribution.`,
	RunE:  runWorkspaceStats,
}

func init() {
	workspaceListCmd.Flags().BoolVar(&workspaceListFavorites, "favorites", false, "Show only favorited workspaces")
	workspaceListCmd.Flags().BoolVar(&workspaceListRecent, "recent", false, "Show recently viewed workspaces")
	workspaceListCmd.MarkFlagsMutuallyExclusive("favorites", "recent")

	workspaceShowCmd.Flags().BoolVarP(&workspaceShowInteractive, "interactive", "i", false, "Select a workspace from a list")

	workspaceReposCmd.Flags().BoolVar(&workspaceReposGitHub, "github", false, "Include description, language, and stars from GitHub")

	workspaceStatsCmd.Flags().IntVar(&workspaceStatsSprints, "sprints", 6, "Number of recent closed sprints for velocity trend")
	workspaceStatsCmd.Flags().IntVar(&workspaceStatsDays, "days", 30, "Cycle time window in days")

	workspaceCmd.AddCommand(workspaceListCmd)
	workspaceCmd.AddCommand(workspaceShowCmd)
	workspaceCmd.AddCommand(workspaceSwitchCmd)
	workspaceCmd.AddCommand(workspaceReposCmd)
	workspaceCmd.AddCommand(workspaceStatsCmd)
	rootCmd.AddCommand(workspaceCmd)
}

// apiNewFunc is the function used to create API clients. It can be replaced
// in tests to inject a mock server endpoint.
var apiNewFunc = api.New

// ghNewFunc is the function used to create GitHub clients. It can be replaced
// in tests to inject a mock GitHub endpoint.
var ghNewFunc = gh.New

// newClient creates an API client from config, wiring up verbose logging.
func newClient(cfg *config.Config, cmd *cobra.Command) *api.Client {
	var opts []api.Option
	if cfg.RESTAPIKey != "" {
		opts = append(opts, api.WithRESTAPIKey(cfg.RESTAPIKey))
	}
	if verbose {
		opts = append(opts, api.WithVerbose(func(format string, args ...any) {
			fmt.Fprintf(cmd.ErrOrStderr(), format, args...)
		}))
	}
	return apiNewFunc(cfg.APIKey, opts...)
}

// newGitHubClient creates a GitHub API client from config. Returns nil if
// GitHub access is not configured.
func newGitHubClient(cfg *config.Config, cmd *cobra.Command) *gh.Client {
	var opts []gh.Option
	if verbose {
		opts = append(opts, gh.WithVerbose(func(format string, args ...any) {
			fmt.Fprintf(cmd.ErrOrStderr(), format, args...)
		}))
	}
	return ghNewFunc(cfg.GitHub.Method, cfg.GitHub.Token, opts...)
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

	workspaces := make([]workspaceNode, 0)
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

	workspaces := make([]workspaceNode, 0, len(resp.Viewer.WorkspaceFavorites.Nodes))
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
	if workspaceShowInteractive {
		identifier, err := interactiveOrArg(cmd, nil, true, func() ([]selectItem, error) {
			return fetchWorkspaceSelectItems(client)
		}, "Select a workspace")
		if err != nil {
			return err
		}
		workspaceID = identifier
	} else if len(args) > 0 {
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

// fetchWorkspaceSelectItems fetches workspaces and converts them to selectItems for interactive mode.
func fetchWorkspaceSelectItems(client *api.Client) ([]selectItem, error) {
	workspaces, err := fetchAllWorkspaces(client)
	if err != nil {
		return nil, err
	}

	items := make([]selectItem, len(workspaces))
	for i, ws := range workspaces {
		desc := ""
		if ws.Organization != nil {
			desc = ws.Organization.Name
		}
		if ws.ReposConnection != nil {
			if desc != "" {
				desc += " · "
			}
			desc += fmt.Sprintf("%d repos", ws.ReposConnection.TotalCount)
		}
		items[i] = selectItem{
			id:          ws.ID,
			title:       ws.DisplayName,
			description: desc,
		}
	}
	return items, nil
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
		_ = resolve.FetchReposIntoCache(repos, ws.ID)
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
	_ = resolve.FetchReposIntoCache(cached, cfg.Workspace)

	// GitHub enrichment
	var ghInfo map[string]githubRepoInfo
	if workspaceReposGitHub {
		ghClient := newGitHubClient(cfg, cmd)
		if ghClient == nil {
			fmt.Fprintln(cmd.ErrOrStderr(), "Warning: GitHub access not configured — ignoring --github flag")
		} else {
			ghInfo = fetchGitHubRepoInfo(ghClient, repos)
		}
	}

	if output.IsJSON(outputFormat) {
		if ghInfo != nil {
			type enrichedRepo struct {
				repoNode
				GitHub *githubRepoInfo `json:"github,omitempty"`
			}
			enriched := make([]enrichedRepo, len(repos))
			for i, r := range repos {
				enriched[i] = enrichedRepo{repoNode: r}
				key := r.OwnerName + "/" + r.Name
				if info, ok := ghInfo[key]; ok {
					enriched[i].GitHub = &info
				}
			}
			return output.JSON(w, enriched)
		}
		return output.JSON(w, repos)
	}

	if len(repos) == 0 {
		fmt.Fprintln(w, "No repositories connected to this workspace.")
		return nil
	}

	if ghInfo != nil {
		lw := output.NewListWriter(w, "REPO", "DESCRIPTION", "LANGUAGE", "STARS", "PRIVATE")
		for _, r := range repos {
			name := r.OwnerName + "/" + r.Name
			private := "no"
			if r.IsPrivate {
				private = "yes"
			}
			info := ghInfo[name]
			desc := output.TableMissing
			if info.Description != "" {
				desc = info.Description
				if len(desc) > 40 {
					desc = desc[:37] + "..."
				}
			}
			lang := output.TableMissing
			if info.Language != "" {
				lang = info.Language
			}
			lw.Row(name, desc, lang, fmt.Sprintf("%d", info.Stars), private)
		}
		lw.FlushWithFooter(fmt.Sprintf("Total: %d repo(s)", len(repos)))
	} else {
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
	}
	return nil
}

// fetchGitHubRepoInfo fetches enriched repo info from GitHub for each repo.
func fetchGitHubRepoInfo(client *gh.Client, repos []repoNode) map[string]githubRepoInfo {
	info := make(map[string]githubRepoInfo)
	for _, r := range repos {
		data, err := client.Execute(githubRepoQuery, map[string]any{
			"owner": r.OwnerName,
			"name":  r.Name,
		})
		if err != nil {
			continue
		}

		var resp struct {
			Repository struct {
				Description     *string `json:"description"`
				PrimaryLanguage *struct {
					Name string `json:"name"`
				} `json:"primaryLanguage"`
				StargazerCount int `json:"stargazerCount"`
			} `json:"repository"`
		}
		if err := json.Unmarshal(data, &resp); err != nil {
			continue
		}

		ri := githubRepoInfo{
			Stars: resp.Repository.StargazerCount,
		}
		if resp.Repository.Description != nil {
			ri.Description = *resp.Repository.Description
		}
		if resp.Repository.PrimaryLanguage != nil {
			ri.Language = resp.Repository.PrimaryLanguage.Name
		}
		info[r.OwnerName+"/"+r.Name] = ri
	}
	return info
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

// --- Workspace stats ---

const workspaceStatsQuery = `query WorkspaceStats($workspaceId: ID!, $sprintCount: Int!, $daysInCycle: Int!) {
  workspace(id: $workspaceId) {
    displayName

    averageSprintVelocity
    averageSprintVelocityWithDiff(skipDiff: false) {
      velocity
      difference
      sprintsCount
    }

    assumeEstimates
    assumedEstimateValue
    hasEstimatedIssues

    issueFlowStats(daysInCycle: $daysInCycle) {
      avgCycleDays
      inDevelopmentDays
      inReviewDays
    }

    pipelinesConnection(first: 50) {
      totalCount
      nodes {
        name
        stage
        issues(first: 0) {
          totalCount
          pipelineCounts {
            issuesCount
            pullRequestsCount
            sumEstimates
          }
        }
      }
    }

    closedPipeline {
      issues(first: 0) {
        totalCount
        pipelineCounts {
          issuesCount
          pullRequestsCount
          sumEstimates
        }
      }
    }

    issues(first: 0) {
      totalCount
      pipelineCounts {
        issuesCount
        pullRequestsCount
        sumEstimates
      }
    }

    activeSprint {
      name
      generatedName
      state
      startAt
      endAt
      totalPoints
      completedPoints
      closedIssuesCount
      sprintIssues(first: 0) {
        totalCount
      }
    }

    sprints(
      first: $sprintCount
      filters: { state: { eq: CLOSED } }
      orderBy: { field: END_AT, direction: DESC }
    ) {
      totalCount
      nodes {
        name
        generatedName
        startAt
        endAt
        totalPoints
        completedPoints
        closedIssuesCount
        sprintIssues(first: 0) {
          totalCount
        }
      }
    }

    sprintConfig {
      kind
      period
    }

    repositoriesConnection(first: 0) { totalCount }
    zenhubEpics(first: 0) { totalCount }
    prioritiesConnection(first: 0) { totalCount }
    issueDependencies(first: 0) { totalCount }
    pipelineToPipelineAutomations(first: 0) { totalCount }
  }
}`

// Stats response types

type statsResponse struct {
	DisplayName string `json:"displayName"`

	AvgVelocity      *float64 `json:"averageSprintVelocity"`
	VelocityWithDiff *struct {
		Velocity     float64  `json:"velocity"`
		Difference   *float64 `json:"difference"`
		SprintsCount int      `json:"sprintsCount"`
	} `json:"averageSprintVelocityWithDiff"`

	AssumeEstimates      bool    `json:"assumeEstimates"`
	AssumedEstimateValue float64 `json:"assumedEstimateValue"`
	HasEstimatedIssues   bool    `json:"hasEstimatedIssues"`

	IssueFlowStats *struct {
		AvgCycleDays      *int `json:"avgCycleDays"`
		InDevelopmentDays *int `json:"inDevelopmentDays"`
		InReviewDays      *int `json:"inReviewDays"`
	} `json:"issueFlowStats"`

	PipelinesConn struct {
		TotalCount int             `json:"totalCount"`
		Nodes      []statsPipeline `json:"nodes"`
	} `json:"pipelinesConnection"`

	ClosedPipeline *struct {
		Issues pipelineIssues `json:"issues"`
	} `json:"closedPipeline"`

	Issues pipelineIssues `json:"issues"`

	ActiveSprint *statsSprint `json:"activeSprint"`

	Sprints struct {
		TotalCount int           `json:"totalCount"`
		Nodes      []statsSprint `json:"nodes"`
	} `json:"sprints"`

	SprintConfig *struct {
		Kind   string `json:"kind"`
		Period int    `json:"period"`
	} `json:"sprintConfig"`

	ReposConn       totalCountConn `json:"repositoriesConnection"`
	EpicsConn       totalCountConn `json:"zenhubEpics"`
	PrioritiesConn  totalCountConn `json:"prioritiesConnection"`
	DepsConn        totalCountConn `json:"issueDependencies"`
	AutomationsConn totalCountConn `json:"pipelineToPipelineAutomations"`
}

type totalCountConn struct {
	TotalCount int `json:"totalCount"`
}

type statsPipeline struct {
	Name   string         `json:"name"`
	Stage  *string        `json:"stage"`
	Issues pipelineIssues `json:"issues"`
}

type pipelineIssues struct {
	TotalCount     int `json:"totalCount"`
	PipelineCounts *struct {
		IssuesCount       int     `json:"issuesCount"`
		PullRequestsCount int     `json:"pullRequestsCount"`
		SumEstimates      float64 `json:"sumEstimates"`
	} `json:"pipelineCounts"`
}

type statsSprint struct {
	Name              string  `json:"name"`
	GeneratedName     string  `json:"generatedName"`
	State             string  `json:"state"`
	StartAt           string  `json:"startAt"`
	EndAt             string  `json:"endAt"`
	TotalPoints       float64 `json:"totalPoints"`
	CompletedPoints   float64 `json:"completedPoints"`
	ClosedIssuesCount int     `json:"closedIssuesCount"`
	SprintIssues      *struct {
		TotalCount int `json:"totalCount"`
	} `json:"sprintIssues"`
}

// runWorkspaceStats implements `zh workspace stats`.
func runWorkspaceStats(cmd *cobra.Command, args []string) error {
	cfg, err := requireWorkspace()
	if err != nil {
		return err
	}

	client := newClient(cfg, cmd)
	w := cmd.OutOrStdout()

	data, err := client.Execute(workspaceStatsQuery, map[string]any{
		"workspaceId": cfg.Workspace,
		"sprintCount": workspaceStatsSprints,
		"daysInCycle": workspaceStatsDays,
	})
	if err != nil {
		return exitcode.General("fetching workspace stats", err)
	}

	var resp struct {
		Workspace statsResponse `json:"workspace"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return exitcode.General("parsing workspace stats", err)
	}

	stats := resp.Workspace

	if output.IsJSON(outputFormat) {
		return output.JSON(w, stats)
	}

	d := output.NewDetailWriter(w, "WORKSPACE STATS", stats.DisplayName)

	// Summary section
	d.Section("SUMMARY")

	issueCount := 0
	prCount := 0
	totalEstimates := 0.0
	if stats.Issues.PipelineCounts != nil {
		issueCount = stats.Issues.PipelineCounts.IssuesCount
		prCount = stats.Issues.PipelineCounts.PullRequestsCount
		totalEstimates = stats.Issues.PipelineCounts.SumEstimates
	}

	fmt.Fprintf(w, "%-20s%-20s%s\n",
		fmt.Sprintf("Repositories: %d", stats.ReposConn.TotalCount),
		fmt.Sprintf("Epics: %d", stats.EpicsConn.TotalCount),
		fmt.Sprintf("Automations: %d", stats.AutomationsConn.TotalCount))
	fmt.Fprintf(w, "%-20s%-20s%s\n",
		fmt.Sprintf("Issues: %d", issueCount),
		fmt.Sprintf("PRs: %d", prCount),
		fmt.Sprintf("Dependencies: %d", stats.DepsConn.TotalCount))
	fmt.Fprintf(w, "%-20s%-20s%s\n",
		fmt.Sprintf("Estimates: %.0f pts", totalEstimates),
		fmt.Sprintf("Priorities: %d", stats.PrioritiesConn.TotalCount),
		fmt.Sprintf("Pipelines: %d", stats.PipelinesConn.TotalCount))

	// Velocity section
	hasSprints := stats.SprintConfig != nil
	if hasSprints {
		d.Section("VELOCITY")

		if stats.VelocityWithDiff != nil && stats.VelocityWithDiff.SprintsCount > 0 {
			vwd := stats.VelocityWithDiff
			velocityStr := fmt.Sprintf("%.0f pts/sprint", vwd.Velocity)
			if vwd.SprintsCount > 0 {
				velocityStr += fmt.Sprintf(" (last %d sprints", vwd.SprintsCount)
				if vwd.Difference != nil {
					diff := *vwd.Difference
					if diff > 0 {
						velocityStr += fmt.Sprintf(", %s", output.Green(fmt.Sprintf("+%.0f trend", diff)))
					} else if diff < 0 {
						velocityStr += fmt.Sprintf(", %s", output.Red(fmt.Sprintf("%.0f trend", diff)))
					}
				}
				velocityStr += ")"
			}
			fmt.Fprintf(w, "Average velocity: %s\n", velocityStr)
		} else if stats.AvgVelocity != nil && *stats.AvgVelocity > 0 {
			fmt.Fprintf(w, "Average velocity: %.0f pts/sprint\n", *stats.AvgVelocity)
		} else {
			fmt.Fprintln(w, "No velocity data available (no closed sprints yet).")
		}

		if stats.AssumeEstimates {
			fmt.Fprintf(w, "Assumed estimates: %.0f pt (unestimated issues counted)\n", stats.AssumedEstimateValue)
		}

		// Sprint table
		var sprintRows []statsSprint
		if stats.ActiveSprint != nil {
			sprintRows = append(sprintRows, *stats.ActiveSprint)
		}
		sprintRows = append(sprintRows, stats.Sprints.Nodes...)

		if len(sprintRows) > 0 {
			fmt.Fprintln(w)
			lw := output.NewListWriter(w, "SPRINT", "DATES", "DONE", "TOTAL", "ISSUES")
			for i, sp := range sprintRows {
				name := sp.Name
				if sp.Name == "" {
					name = sp.GeneratedName
				}

				prefix := "  "
				if i == 0 && stats.ActiveSprint != nil {
					prefix = output.Green("▶ ")
				}
				name = prefix + name

				dates := output.TableMissing
				if sp.StartAt != "" && sp.EndAt != "" {
					startAt, _ := time.Parse(time.RFC3339, sp.StartAt)
					endAt, _ := time.Parse(time.RFC3339, sp.EndAt)
					dates = output.FormatDateRange(startAt, endAt)
				}

				issueTotal := 0
				if sp.SprintIssues != nil {
					issueTotal = sp.SprintIssues.TotalCount
				}

				lw.Row(
					name,
					dates,
					fmt.Sprintf("%.0f", sp.CompletedPoints),
					fmt.Sprintf("%.0f", sp.TotalPoints),
					fmt.Sprintf("%d/%d", sp.ClosedIssuesCount, issueTotal),
				)
			}
			lw.Flush()
		}
	} else {
		d.Section("VELOCITY")
		fmt.Fprintln(w, "Sprints are not configured for this workspace.")
	}

	// Cycle time section
	d.Section(fmt.Sprintf("CYCLE TIME (last %d days)", workspaceStatsDays))
	if stats.IssueFlowStats != nil && stats.IssueFlowStats.AvgCycleDays != nil {
		flow := stats.IssueFlowStats
		fmt.Fprintf(w, "Average cycle:     %d days\n", *flow.AvgCycleDays)
		if flow.InDevelopmentDays != nil {
			fmt.Fprintf(w, "  In development:  %d days\n", *flow.InDevelopmentDays)
		}
		if flow.InReviewDays != nil {
			fmt.Fprintf(w, "  In review:       %d days\n", *flow.InReviewDays)
		}
	} else {
		fmt.Fprintln(w, "No cycle time data available.")
		fmt.Fprintln(w, output.Dim("Issues may not have completed a full cycle, or pipeline stages may not be configured."))
	}

	// Pipeline distribution section
	d.Section("PIPELINE DISTRIBUTION")
	lw := output.NewListWriter(w, "PIPELINE", "STAGE", "ISSUES", "PRS", "ESTIMATES")
	for _, p := range stats.PipelinesConn.Nodes {
		stage := output.TableMissing
		if p.Stage != nil && *p.Stage != "" {
			stage = *p.Stage
		}
		issues := "0"
		prs := "0"
		estimates := "0"
		if p.Issues.PipelineCounts != nil {
			issues = fmt.Sprintf("%d", p.Issues.PipelineCounts.IssuesCount)
			prs = fmt.Sprintf("%d", p.Issues.PipelineCounts.PullRequestsCount)
			estimates = formatEstimate(p.Issues.PipelineCounts.SumEstimates)
		}
		lw.Row(p.Name, stage, issues, prs, estimates)
	}
	// Closed pipeline
	if stats.ClosedPipeline != nil && stats.ClosedPipeline.Issues.PipelineCounts != nil {
		pc := stats.ClosedPipeline.Issues.PipelineCounts
		lw.Row(
			output.Green("Closed"),
			output.Green("DONE"),
			fmt.Sprintf("%d", pc.IssuesCount),
			fmt.Sprintf("%d", pc.PullRequestsCount),
			formatEstimate(pc.SumEstimates),
		)
	}
	lw.Flush()

	return nil
}

// formatEstimate formats an estimate value, omitting decimal for whole numbers.
func formatEstimate(v float64) string {
	if v == float64(int(v)) {
		return fmt.Sprintf("%.0f", v)
	}
	return fmt.Sprintf("%.1f", v)
}
