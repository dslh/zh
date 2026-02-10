package cmd

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/dslh/zh/internal/api"
	"github.com/dslh/zh/internal/exitcode"
	"github.com/dslh/zh/internal/output"
	"github.com/dslh/zh/internal/resolve"
	"github.com/spf13/cobra"
)

// Epic list types — unified across both ZenhubEpic and legacy Epic types.

type epicListEntry struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Type     string `json:"type"` // "zenhub" or "legacy"
	State    string `json:"state"`
	StartOn  string `json:"startOn,omitempty"`
	EndOn    string `json:"endOn,omitempty"`
	Estimate *struct {
		Value float64 `json:"value"`
	} `json:"estimate,omitempty"`
	IssueCountProgress struct {
		Open   int `json:"open"`
		Closed int `json:"closed"`
		Total  int `json:"total"`
	} `json:"issueCountProgress"`
	IssueEstimateProgress struct {
		Open   int `json:"open"`
		Closed int `json:"closed"`
		Total  int `json:"total"`
	} `json:"issueEstimateProgress"`

	// Legacy epic fields
	IssueNumber int    `json:"issueNumber,omitempty"`
	RepoName    string `json:"repoName,omitempty"`
	RepoOwner   string `json:"repoOwner,omitempty"`
}

// Epic detail types for show command.

type epicDetailZenhub struct {
	ID        string  `json:"id"`
	Title     string  `json:"title"`
	Body      string  `json:"body"`
	State     string  `json:"state"`
	StartOn   *string `json:"startOn"`
	EndOn     *string `json:"endOn"`
	CreatedAt string  `json:"createdAt"`
	UpdatedAt string  `json:"updatedAt"`
	Estimate  *struct {
		Value float64 `json:"value"`
	} `json:"estimate"`
	Creator *struct {
		ID         string `json:"id"`
		Name       string `json:"name"`
		GithubUser *struct {
			Login string `json:"login"`
		} `json:"githubUser"`
	} `json:"creator"`
	Assignees struct {
		Nodes []struct {
			ID         string `json:"id"`
			Name       string `json:"name"`
			GithubUser *struct {
				Login string `json:"login"`
			} `json:"githubUser"`
		} `json:"nodes"`
	} `json:"assignees"`
	Labels struct {
		Nodes []struct {
			ID    string `json:"id"`
			Name  string `json:"name"`
			Color string `json:"color"`
		} `json:"nodes"`
	} `json:"labels"`
	ChildIssues struct {
		TotalCount int                  `json:"totalCount"`
		Nodes      []epicChildIssueNode `json:"nodes"`
	} `json:"childIssues"`
	IssueCountProgress struct {
		Open   int `json:"open"`
		Closed int `json:"closed"`
		Total  int `json:"total"`
	} `json:"zenhubIssueCountProgress"`
	IssueEstimateProgress struct {
		Open   int `json:"open"`
		Closed int `json:"closed"`
		Total  int `json:"total"`
	} `json:"zenhubIssueEstimateProgress"`
	BlockingItems struct {
		TotalCount int               `json:"totalCount"`
		Nodes      []json.RawMessage `json:"nodes"`
	} `json:"blockingItems"`
	BlockedItems struct {
		TotalCount int               `json:"totalCount"`
		Nodes      []json.RawMessage `json:"nodes"`
	} `json:"blockedItems"`
	KeyDates struct {
		TotalCount int           `json:"totalCount"`
		Nodes      []keyDateNode `json:"nodes"`
	} `json:"keyDates"`
}

type epicDetailLegacy struct {
	ID        string  `json:"id"`
	StartOn   *string `json:"startOn"`
	EndOn     *string `json:"endOn"`
	CreatedAt string  `json:"createdAt"`
	UpdatedAt string  `json:"updatedAt"`
	Issue     struct {
		ID         string `json:"id"`
		Number     int    `json:"number"`
		Title      string `json:"title"`
		Body       string `json:"body"`
		State      string `json:"state"`
		HtmlUrl    string `json:"htmlUrl"`
		Repository struct {
			ID        string `json:"id"`
			Name      string `json:"name"`
			OwnerName string `json:"ownerName"`
		} `json:"repository"`
		Assignees struct {
			Nodes []struct {
				Login string `json:"login"`
			} `json:"nodes"`
		} `json:"assignees"`
		Labels struct {
			Nodes []struct {
				ID    string `json:"id"`
				Name  string `json:"name"`
				Color string `json:"color"`
			} `json:"nodes"`
		} `json:"labels"`
		Estimate *struct {
			Value float64 `json:"value"`
		} `json:"estimate"`
	} `json:"issue"`
	ChildIssues struct {
		TotalCount int                  `json:"totalCount"`
		Nodes      []epicChildIssueNode `json:"nodes"`
	} `json:"childIssues"`
	IssueCountProgress struct {
		Open   int `json:"open"`
		Closed int `json:"closed"`
		Total  int `json:"total"`
	} `json:"issueCountProgress"`
	IssueEstimateProgress struct {
		Open   int `json:"open"`
		Closed int `json:"closed"`
		Total  int `json:"total"`
	} `json:"issueEstimateProgress"`
}

type epicChildIssueNode struct {
	ID       string `json:"id"`
	Number   int    `json:"number"`
	Title    string `json:"title"`
	State    string `json:"state"`
	Estimate *struct {
		Value float64 `json:"value"`
	} `json:"estimate"`
	Repository struct {
		Name      string `json:"name"`
		OwnerName string `json:"ownerName"`
	} `json:"repository"`
	PipelineIssue *struct {
		Pipeline struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"pipeline"`
	} `json:"pipelineIssue"`
}

// blockItemIssue and blockItemEpic are used for parsing blocking/blocked items.
type blockItemIssue struct {
	ID         string `json:"id"`
	Number     int    `json:"number"`
	Title      string `json:"title"`
	Repository struct {
		Name      string `json:"name"`
		OwnerName string `json:"ownerName"`
	} `json:"repository"`
}

type blockItemEpic struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

// GraphQL queries

const listEpicsFullQuery = `query ListEpics($workspaceId: ID!, $first: Int!, $after: String) {
  workspace(id: $workspaceId) {
    roadmap {
      items(first: $first, after: $after) {
        totalCount
        pageInfo {
          hasNextPage
          endCursor
        }
        nodes {
          __typename
          ... on ZenhubEpic {
            id
            title
            state
            startOn
            endOn
            estimate { value }
            zenhubIssueCountProgress { open closed total }
            zenhubIssueEstimateProgress { open closed total }
          }
          ... on Epic {
            id
            startOn
            endOn
            issue {
              title
              number
              state
              repository { name ownerName }
            }
            childIssues(first: 1) { totalCount }
            issueCountProgress { open closed total }
            issueEstimateProgress { open closed total }
          }
        }
      }
    }
  }
}`

const epicShowZenhubQuery = `query GetZenhubEpic($id: ID!, $workspaceId: ID!) {
  node(id: $id) {
    ... on ZenhubEpic {
      id
      title
      body
      state
      startOn
      endOn
      createdAt
      updatedAt
      estimate { value }
      creator {
        id
        name
        githubUser { login }
      }
      assignees(first: 50) {
        nodes {
          id
          name
          githubUser { login }
        }
      }
      labels(first: 50) {
        nodes { id name color }
      }
      childIssues(first: 100, workspaceId: $workspaceId) {
        totalCount
        nodes {
          id
          number
          title
          state
          estimate { value }
          repository { name ownerName }
          pipelineIssue(workspaceId: $workspaceId) {
            pipeline { id name }
          }
        }
      }
      zenhubIssueCountProgress { open closed total }
      zenhubIssueEstimateProgress { open closed total }
      blockingItems(first: 20) {
        totalCount
        nodes {
          ... on Issue {
            id
            number
            title
            repository { name ownerName }
          }
          ... on ZenhubEpic {
            id
            title
          }
        }
      }
      blockedItems(first: 20) {
        totalCount
        nodes {
          ... on Issue {
            id
            number
            title
            repository { name ownerName }
          }
          ... on ZenhubEpic {
            id
            title
          }
        }
      }
      keyDates(first: 50) {
        totalCount
        nodes {
          id
          date
          description
          color
        }
      }
    }
  }
}`

const epicShowLegacyQuery = `query GetLegacyEpic($id: ID!) {
  node(id: $id) {
    ... on Epic {
      id
      startOn
      endOn
      createdAt
      updatedAt
      issue {
        id
        number
        title
        body
        state
        htmlUrl
        repository { id name ownerName }
        assignees(first: 50) {
          nodes { login }
        }
        labels(first: 50) {
          nodes { id name color }
        }
        estimate { value }
      }
      childIssues(first: 100) {
        totalCount
        nodes {
          id
          number
          title
          state
          estimate { value }
          repository { name ownerName }
        }
      }
      issueCountProgress { open closed total }
      issueEstimateProgress { open closed total }
    }
  }
}`

// Commands

var epicCmd = &cobra.Command{
	Use:   "epic",
	Short: "Manage ZenHub Epics",
	Long:  `List, view, and manage epics in the current ZenHub workspace.`,
}

var epicListCmd = &cobra.Command{
	Use:   "list",
	Short: "List epics in the workspace",
	Long: `List all epics in the current workspace, including both ZenHub epics
and legacy (issue-backed) epics.`,
	RunE: runEpicList,
}

var epicProgressCmd = &cobra.Command{
	Use:   "progress <epic>",
	Short: "Show epic completion status",
	Long: `Show completion status for an epic: issue count (closed/total) and
estimate progress (completed/total).

The epic can be specified as:
  - ZenHub ID
  - exact title or unique title substring
  - owner/repo#number (for legacy epics)
  - an alias set with 'zh epic alias'`,
	Args: cobra.ExactArgs(1),
	RunE: runEpicProgress,
}

var epicShowCmd = &cobra.Command{
	Use:   "show [epic]",
	Short: "View epic details",
	Long: `Display detailed information about a single epic.

The epic can be specified as:
  - ZenHub ID
  - exact title or unique title substring
  - owner/repo#number (for legacy epics)
  - an alias set with 'zh epic alias'

Use --interactive to select an epic from a list.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runEpicShow,
}

var (
	epicListLimit int
	epicListAll   bool

	epicShowLimit       int
	epicShowAll         bool
	epicShowInteractive bool
)

func init() {
	epicListCmd.Flags().IntVar(&epicListLimit, "limit", 100, "Maximum number of epics to show")
	epicListCmd.Flags().BoolVar(&epicListAll, "all", false, "Show all epics (ignore --limit)")

	epicShowCmd.Flags().IntVar(&epicShowLimit, "limit", 100, "Maximum number of child issues to show")
	epicShowCmd.Flags().BoolVar(&epicShowAll, "all", false, "Show all child issues (ignore --limit)")
	epicShowCmd.Flags().BoolVarP(&epicShowInteractive, "interactive", "i", false, "Select an epic from a list")

	epicCmd.AddCommand(epicListCmd)
	epicCmd.AddCommand(epicShowCmd)
	epicCmd.AddCommand(epicProgressCmd)
	rootCmd.AddCommand(epicCmd)
}

func resetEpicFlags() {
	epicListLimit = 100
	epicListAll = false
	epicShowLimit = 100
	epicShowAll = false
	epicShowInteractive = false
}

// runEpicList implements `zh epic list`.
func runEpicList(cmd *cobra.Command, args []string) error {
	cfg, err := requireWorkspace()
	if err != nil {
		return err
	}

	client := newClient(cfg, cmd)
	w := cmd.OutOrStdout()

	limit := epicListLimit
	if epicListAll {
		limit = 0
	}

	epics, totalCount, err := fetchEpicList(client, cfg.Workspace, limit)
	if err != nil {
		return err
	}

	// Cache epic entries for resolution
	cacheEpicsFromList(epics, cfg.Workspace)

	if output.IsJSON(outputFormat) {
		return output.JSON(w, epics)
	}

	if len(epics) == 0 {
		fmt.Fprintln(w, "No epics found.")
		return nil
	}

	lw := output.NewListWriter(w, "TYPE", "STATE", "TITLE", "ISSUES", "ESTIMATE", "DATES")
	for _, e := range epics {
		epicType := e.Type
		state := formatEpicState(e.State)

		title := e.Title
		if e.Type == "legacy" && e.RepoName != "" {
			title = fmt.Sprintf("%s (%s#%d)", e.Title, e.RepoName, e.IssueNumber)
		}
		if len(title) > 50 {
			title = title[:47] + "..."
		}

		issues := output.TableMissing
		if e.IssueCountProgress.Total > 0 {
			issues = fmt.Sprintf("%d/%d", e.IssueCountProgress.Closed, e.IssueCountProgress.Total)
		}

		est := output.TableMissing
		if e.Estimate != nil {
			est = formatEstimate(e.Estimate.Value)
		}

		dates := output.TableMissing
		if e.StartOn != "" || e.EndOn != "" {
			dates = formatEpicDates(e.StartOn, e.EndOn)
		}

		lw.Row(epicType, state, title, issues, est, dates)
	}

	footer := fmt.Sprintf("Showing %d", len(epics))
	if totalCount > len(epics) {
		footer += fmt.Sprintf(" of %d", totalCount)
	}
	footer += " epic(s)"
	lw.FlushWithFooter(footer)
	return nil
}

// fetchEpicList fetches epics from the workspace roadmap with pagination.
func fetchEpicList(client *api.Client, workspaceID string, limit int) ([]epicListEntry, int, error) {
	var allEpics []epicListEntry
	var cursor *string
	totalCount := 0
	pageSize := 50

	for {
		if limit > 0 {
			remaining := limit - len(allEpics)
			if remaining <= 0 {
				break
			}
			if remaining < pageSize {
				pageSize = remaining
			}
		}

		vars := map[string]any{
			"workspaceId": workspaceID,
			"first":       pageSize,
		}
		if cursor != nil {
			vars["after"] = *cursor
		}

		data, err := client.Execute(listEpicsFullQuery, vars)
		if err != nil {
			return nil, 0, exitcode.General("fetching epics", err)
		}

		var resp struct {
			Workspace struct {
				Roadmap struct {
					Items struct {
						TotalCount int `json:"totalCount"`
						PageInfo   struct {
							HasNextPage bool   `json:"hasNextPage"`
							EndCursor   string `json:"endCursor"`
						} `json:"pageInfo"`
						Nodes []json.RawMessage `json:"nodes"`
					} `json:"items"`
				} `json:"roadmap"`
			} `json:"workspace"`
		}
		if err := json.Unmarshal(data, &resp); err != nil {
			return nil, 0, exitcode.General("parsing epics response", err)
		}

		totalCount = resp.Workspace.Roadmap.Items.TotalCount

		for _, raw := range resp.Workspace.Roadmap.Items.Nodes {
			if entry, ok := parseEpicListItem(raw); ok {
				allEpics = append(allEpics, entry)
			}
		}

		if !resp.Workspace.Roadmap.Items.PageInfo.HasNextPage {
			break
		}
		if limit > 0 && len(allEpics) >= limit {
			break
		}

		cursor = &resp.Workspace.Roadmap.Items.PageInfo.EndCursor
	}

	return allEpics, totalCount, nil
}

// parseEpicListItem parses a single roadmap item into an epicListEntry.
// Returns false if the item is not an epic (e.g. a Project).
func parseEpicListItem(raw json.RawMessage) (epicListEntry, bool) {
	var typed struct {
		TypeName string `json:"__typename"`
	}
	if err := json.Unmarshal(raw, &typed); err != nil {
		return epicListEntry{}, false
	}

	switch typed.TypeName {
	case "ZenhubEpic":
		var ze struct {
			ID       string `json:"id"`
			Title    string `json:"title"`
			State    string `json:"state"`
			StartOn  string `json:"startOn"`
			EndOn    string `json:"endOn"`
			Estimate *struct {
				Value float64 `json:"value"`
			} `json:"estimate"`
			IssueCountProgress struct {
				Open   int `json:"open"`
				Closed int `json:"closed"`
				Total  int `json:"total"`
			} `json:"zenhubIssueCountProgress"`
			IssueEstimateProgress struct {
				Open   int `json:"open"`
				Closed int `json:"closed"`
				Total  int `json:"total"`
			} `json:"zenhubIssueEstimateProgress"`
		}
		if err := json.Unmarshal(raw, &ze); err != nil {
			return epicListEntry{}, false
		}
		return epicListEntry{
			ID:                    ze.ID,
			Title:                 ze.Title,
			Type:                  "zenhub",
			State:                 ze.State,
			StartOn:               ze.StartOn,
			EndOn:                 ze.EndOn,
			Estimate:              ze.Estimate,
			IssueCountProgress:    ze.IssueCountProgress,
			IssueEstimateProgress: ze.IssueEstimateProgress,
		}, true

	case "Epic":
		var le struct {
			ID      string `json:"id"`
			StartOn string `json:"startOn"`
			EndOn   string `json:"endOn"`
			Issue   struct {
				Title      string `json:"title"`
				Number     int    `json:"number"`
				State      string `json:"state"`
				Repository struct {
					Name      string `json:"name"`
					OwnerName string `json:"ownerName"`
				} `json:"repository"`
			} `json:"issue"`
			ChildIssues struct {
				TotalCount int `json:"totalCount"`
			} `json:"childIssues"`
			IssueCountProgress struct {
				Open   int `json:"open"`
				Closed int `json:"closed"`
				Total  int `json:"total"`
			} `json:"issueCountProgress"`
			IssueEstimateProgress struct {
				Open   int `json:"open"`
				Closed int `json:"closed"`
				Total  int `json:"total"`
			} `json:"issueEstimateProgress"`
		}
		if err := json.Unmarshal(raw, &le); err != nil {
			return epicListEntry{}, false
		}
		return epicListEntry{
			ID:                    le.ID,
			Title:                 le.Issue.Title,
			Type:                  "legacy",
			State:                 le.Issue.State,
			StartOn:               le.StartOn,
			EndOn:                 le.EndOn,
			IssueCountProgress:    le.IssueCountProgress,
			IssueEstimateProgress: le.IssueEstimateProgress,
			IssueNumber:           le.Issue.Number,
			RepoName:              le.Issue.Repository.Name,
			RepoOwner:             le.Issue.Repository.OwnerName,
		}, true

	default:
		return epicListEntry{}, false
	}
}

// cacheEpicsFromList stores epic entries in the cache for resolution.
func cacheEpicsFromList(epics []epicListEntry, workspaceID string) {
	var entries []resolve.CachedEpic
	for _, e := range epics {
		entries = append(entries, resolve.CachedEpic{
			ID:          e.ID,
			Title:       e.Title,
			Type:        e.Type,
			IssueNumber: e.IssueNumber,
			RepoName:    e.RepoName,
			RepoOwner:   e.RepoOwner,
		})
	}
	_ = resolve.FetchEpicsIntoCache(entries, workspaceID)
}

// runEpicShow implements `zh epic show [epic]`.
func runEpicShow(cmd *cobra.Command, args []string) error {
	cfg, err := requireWorkspace()
	if err != nil {
		return err
	}

	client := newClient(cfg, cmd)
	w := cmd.OutOrStdout()

	var identifier string
	if epicShowInteractive {
		identifier, err = interactiveOrArg(cmd, nil, true, func() ([]selectItem, error) {
			return fetchEpicSelectItems(client, cfg.Workspace)
		}, "Select an epic")
		if err != nil {
			return err
		}
	} else {
		if len(args) < 1 {
			return exitcode.Usage("requires an epic argument or --interactive flag")
		}
		identifier = args[0]
	}

	// Resolve the epic
	resolved, err := resolve.Epic(client, cfg.Workspace, identifier, cfg.Aliases.Epics)
	if err != nil {
		return err
	}

	// Fetch full details based on type
	switch resolved.Type {
	case "zenhub":
		return runEpicShowZenhub(client, cfg.Workspace, resolved.ID, w)
	case "legacy":
		return runEpicShowLegacy(client, resolved.ID, w)
	default:
		// Unknown type — try zenhub first, fall back to legacy
		return runEpicShowZenhub(client, cfg.Workspace, resolved.ID, w)
	}
}

// fetchEpicSelectItems fetches epics and converts them to selectItems for interactive mode.
func fetchEpicSelectItems(client *api.Client, workspaceID string) ([]selectItem, error) {
	epics, _, err := fetchEpicList(client, workspaceID, 0)
	if err != nil {
		return nil, err
	}

	items := make([]selectItem, len(epics))
	for i, e := range epics {
		desc := e.State
		if e.IssueCountProgress.Total > 0 {
			desc += fmt.Sprintf(" · %d/%d issues", e.IssueCountProgress.Closed, e.IssueCountProgress.Total)
		}
		items[i] = selectItem{
			id:          e.ID,
			title:       e.Title,
			description: desc,
		}
	}
	return items, nil
}

// runEpicShowZenhub renders a ZenHub epic detail view.
func runEpicShowZenhub(client *api.Client, workspaceID, epicID string, w writerFlusher) error {
	data, err := client.Execute(epicShowZenhubQuery, map[string]any{
		"id":          epicID,
		"workspaceId": workspaceID,
	})
	if err != nil {
		return exitcode.General("fetching epic details", err)
	}

	var resp struct {
		Node *epicDetailZenhub `json:"node"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return exitcode.General("parsing epic details", err)
	}

	if resp.Node == nil {
		return exitcode.NotFoundError(fmt.Sprintf("epic %q not found", epicID))
	}

	epic := resp.Node

	if output.IsJSON(outputFormat) {
		return output.JSON(w, epic)
	}

	d := output.NewDetailWriter(w, "EPIC", epic.Title)

	// State
	state := formatEpicState(epic.State)

	// Estimate
	estimate := output.DetailMissing
	if epic.Estimate != nil {
		estimate = formatEstimate(epic.Estimate.Value)
	}

	// Creator
	creator := output.DetailMissing
	if epic.Creator != nil {
		if epic.Creator.GithubUser != nil {
			creator = "@" + epic.Creator.GithubUser.Login
		} else if epic.Creator.Name != "" {
			creator = epic.Creator.Name
		}
	}

	// Assignees
	assignees := output.DetailMissing
	if len(epic.Assignees.Nodes) > 0 {
		names := make([]string, 0, len(epic.Assignees.Nodes))
		for _, a := range epic.Assignees.Nodes {
			if a.GithubUser != nil {
				names = append(names, "@"+a.GithubUser.Login)
			} else if a.Name != "" {
				names = append(names, a.Name)
			}
		}
		if len(names) > 0 {
			assignees = strings.Join(names, ", ")
		}
	}

	// Labels
	labels := output.DetailMissing
	if len(epic.Labels.Nodes) > 0 {
		names := make([]string, len(epic.Labels.Nodes))
		for i, l := range epic.Labels.Nodes {
			names[i] = l.Name
		}
		labels = strings.Join(names, ", ")
	}

	fields := []output.KeyValue{
		output.KV("Type", "ZenHub Epic"),
		output.KV("ID", output.Cyan(epic.ID)),
		output.KV("State", state),
		output.KV("Estimate", estimate),
	}

	// Dates
	if epic.StartOn != nil || epic.EndOn != nil {
		dates := formatEpicDatePtrs(epic.StartOn, epic.EndOn)
		fields = append(fields, output.KV("Dates", dates))
	}

	fields = append(fields,
		output.KV("Creator", creator),
		output.KV("Assignees", assignees),
		output.KV("Labels", labels),
	)

	// Created/Updated
	if epic.CreatedAt != "" {
		if t, err := time.Parse(time.RFC3339, epic.CreatedAt); err == nil {
			fields = append(fields, output.KV("Created", output.FormatDate(t)))
		}
	}
	if epic.UpdatedAt != "" {
		if t, err := time.Parse(time.RFC3339, epic.UpdatedAt); err == nil {
			fields = append(fields, output.KV("Updated", output.FormatDate(t)))
		}
	}

	d.Fields(fields)

	// Progress section
	if epic.IssueCountProgress.Total > 0 {
		d.Section("PROGRESS")
		fmt.Fprintf(w, "Issues:     %s\n", output.FormatProgress(epic.IssueCountProgress.Closed, epic.IssueCountProgress.Total))
		if epic.IssueEstimateProgress.Total > 0 {
			fmt.Fprintf(w, "Estimates:  %s\n", output.FormatProgress(epic.IssueEstimateProgress.Closed, epic.IssueEstimateProgress.Total))
		}
	}

	// Child issues section
	renderEpicChildIssues(w, d, epic.ChildIssues.Nodes, epic.ChildIssues.TotalCount)

	// Blocking section
	if epic.BlockingItems.TotalCount > 0 {
		d.Section("BLOCKING")
		renderBlockItems(w, epic.BlockingItems.Nodes)
	}

	// Blocked by section
	if epic.BlockedItems.TotalCount > 0 {
		d.Section("BLOCKED BY")
		renderBlockItems(w, epic.BlockedItems.Nodes)
	}

	// Key dates section
	if epic.KeyDates.TotalCount > 0 {
		d.Section(fmt.Sprintf("KEY DATES (%d)", epic.KeyDates.TotalCount))
		for _, kd := range epic.KeyDates.Nodes {
			fmt.Fprintf(w, "  %s  %s\n", kd.Date, kd.Description)
		}
	}

	// Description section
	if epic.Body != "" {
		d.Section("DESCRIPTION")
		_ = output.RenderMarkdown(w, epic.Body, 80)
	}

	return nil
}

// runEpicShowLegacy renders a legacy epic detail view.
func runEpicShowLegacy(client *api.Client, epicID string, w writerFlusher) error {
	data, err := client.Execute(epicShowLegacyQuery, map[string]any{
		"id": epicID,
	})
	if err != nil {
		return exitcode.General("fetching epic details", err)
	}

	var resp struct {
		Node *epicDetailLegacy `json:"node"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return exitcode.General("parsing epic details", err)
	}

	if resp.Node == nil {
		return exitcode.NotFoundError(fmt.Sprintf("epic %q not found", epicID))
	}

	epic := resp.Node
	issue := epic.Issue

	if output.IsJSON(outputFormat) {
		return output.JSON(w, epic)
	}

	ref := fmt.Sprintf("%s#%d", issue.Repository.Name, issue.Number)
	title := fmt.Sprintf("%s: %s", ref, issue.Title)
	d := output.NewDetailWriter(w, "EPIC", title)

	// State
	state := formatEpicState(issue.State)

	// Estimate
	estimate := output.DetailMissing
	if issue.Estimate != nil {
		estimate = formatEstimate(issue.Estimate.Value)
	}

	// Assignees
	assignees := output.DetailMissing
	if len(issue.Assignees.Nodes) > 0 {
		logins := make([]string, len(issue.Assignees.Nodes))
		for i, a := range issue.Assignees.Nodes {
			logins[i] = "@" + a.Login
		}
		assignees = strings.Join(logins, ", ")
	}

	// Labels
	labels := output.DetailMissing
	if len(issue.Labels.Nodes) > 0 {
		names := make([]string, len(issue.Labels.Nodes))
		for i, l := range issue.Labels.Nodes {
			names[i] = l.Name
		}
		labels = strings.Join(names, ", ")
	}

	fields := []output.KeyValue{
		output.KV("Type", "Legacy Epic (GitHub issue)"),
		output.KV("ID", output.Cyan(epic.ID)),
		output.KV("State", state),
		output.KV("Estimate", estimate),
	}

	// Dates
	if epic.StartOn != nil || epic.EndOn != nil {
		dates := formatEpicDatePtrs(epic.StartOn, epic.EndOn)
		fields = append(fields, output.KV("Dates", dates))
	}

	fields = append(fields,
		output.KV("Assignees", assignees),
		output.KV("Labels", labels),
	)

	// Created/Updated
	if epic.CreatedAt != "" {
		if t, err := time.Parse(time.RFC3339, epic.CreatedAt); err == nil {
			fields = append(fields, output.KV("Created", output.FormatDate(t)))
		}
	}

	d.Fields(fields)

	// Progress section
	if epic.IssueCountProgress.Total > 0 {
		d.Section("PROGRESS")
		fmt.Fprintf(w, "Issues:     %s\n", output.FormatProgress(epic.IssueCountProgress.Closed, epic.IssueCountProgress.Total))
		if epic.IssueEstimateProgress.Total > 0 {
			fmt.Fprintf(w, "Estimates:  %s\n", output.FormatProgress(epic.IssueEstimateProgress.Closed, epic.IssueEstimateProgress.Total))
		}
	}

	// Child issues section
	renderEpicChildIssues(w, d, epic.ChildIssues.Nodes, epic.ChildIssues.TotalCount)

	// Description section
	if issue.Body != "" {
		d.Section("DESCRIPTION")
		_ = output.RenderMarkdown(w, issue.Body, 80)
	}

	// Links section
	if issue.HtmlUrl != "" {
		d.Section("LINKS")
		fmt.Fprintf(w, "  GitHub:  %s\n", output.Cyan(issue.HtmlUrl))
	}

	return nil
}

// GraphQL queries for epic progress
const epicProgressZenhubQuery = `query GetZenhubEpicProgress($id: ID!, $workspaceId: ID!) {
  node(id: $id) {
    ... on ZenhubEpic {
      id
      title
      state
      estimate { value }
      zenhubIssueCountProgress { open closed total }
      zenhubIssueEstimateProgress { open closed total }
    }
  }
}`

const epicProgressLegacyQuery = `query GetLegacyEpicProgress($id: ID!) {
  node(id: $id) {
    ... on Epic {
      id
      issue {
        title
        number
        state
        estimate { value }
        repository { name ownerName }
      }
      issueCountProgress { open closed total }
      issueEstimateProgress { open closed total }
    }
  }
}`

// runEpicProgress implements `zh epic progress <epic>`.
func runEpicProgress(cmd *cobra.Command, args []string) error {
	cfg, err := requireWorkspace()
	if err != nil {
		return err
	}

	client := newClient(cfg, cmd)
	w := cmd.OutOrStdout()

	// Resolve the epic
	resolved, err := resolve.Epic(client, cfg.Workspace, args[0], cfg.Aliases.Epics)
	if err != nil {
		return err
	}

	switch resolved.Type {
	case "legacy":
		return runEpicProgressLegacy(client, resolved.ID, w)
	default:
		return runEpicProgressZenhub(client, cfg.Workspace, resolved.ID, w)
	}
}

// runEpicProgressZenhub shows progress for a ZenHub epic.
func runEpicProgressZenhub(client *api.Client, workspaceID, epicID string, w writerFlusher) error {
	data, err := client.Execute(epicProgressZenhubQuery, map[string]any{
		"id":          epicID,
		"workspaceId": workspaceID,
	})
	if err != nil {
		return exitcode.General("fetching epic progress", err)
	}

	var resp struct {
		Node *struct {
			ID       string `json:"id"`
			Title    string `json:"title"`
			State    string `json:"state"`
			Estimate *struct {
				Value float64 `json:"value"`
			} `json:"estimate"`
			IssueCountProgress struct {
				Open   int `json:"open"`
				Closed int `json:"closed"`
				Total  int `json:"total"`
			} `json:"zenhubIssueCountProgress"`
			IssueEstimateProgress struct {
				Open   int `json:"open"`
				Closed int `json:"closed"`
				Total  int `json:"total"`
			} `json:"zenhubIssueEstimateProgress"`
		} `json:"node"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return exitcode.General("parsing epic progress", err)
	}

	if resp.Node == nil {
		return exitcode.NotFoundError(fmt.Sprintf("epic %q not found", epicID))
	}

	epic := resp.Node

	if output.IsJSON(outputFormat) {
		return output.JSON(w, map[string]any{
			"id":    epic.ID,
			"title": epic.Title,
			"state": epic.State,
			"issues": map[string]any{
				"open":   epic.IssueCountProgress.Open,
				"closed": epic.IssueCountProgress.Closed,
				"total":  epic.IssueCountProgress.Total,
			},
			"estimates": map[string]any{
				"open":   epic.IssueEstimateProgress.Open,
				"closed": epic.IssueEstimateProgress.Closed,
				"total":  epic.IssueEstimateProgress.Total,
			},
		})
	}

	d := output.NewDetailWriter(w, "EPIC PROGRESS", epic.Title)
	d.Fields([]output.KeyValue{
		output.KV("State", formatEpicState(epic.State)),
	})

	if epic.IssueCountProgress.Total > 0 {
		d.Section("PROGRESS")
		fmt.Fprintf(w, "Issues:     %s\n", output.FormatProgress(epic.IssueCountProgress.Closed, epic.IssueCountProgress.Total))
		if epic.IssueEstimateProgress.Total > 0 {
			fmt.Fprintf(w, "Estimates:  %s\n", output.FormatProgress(epic.IssueEstimateProgress.Closed, epic.IssueEstimateProgress.Total))
		}
	} else {
		d.Section("PROGRESS")
		fmt.Fprintln(w, "No child issues.")
	}

	return nil
}

// runEpicProgressLegacy shows progress for a legacy epic.
func runEpicProgressLegacy(client *api.Client, epicID string, w writerFlusher) error {
	data, err := client.Execute(epicProgressLegacyQuery, map[string]any{
		"id": epicID,
	})
	if err != nil {
		return exitcode.General("fetching epic progress", err)
	}

	var resp struct {
		Node *struct {
			ID    string `json:"id"`
			Issue struct {
				Title    string `json:"title"`
				Number   int    `json:"number"`
				State    string `json:"state"`
				Estimate *struct {
					Value float64 `json:"value"`
				} `json:"estimate"`
				Repository struct {
					Name      string `json:"name"`
					OwnerName string `json:"ownerName"`
				} `json:"repository"`
			} `json:"issue"`
			IssueCountProgress struct {
				Open   int `json:"open"`
				Closed int `json:"closed"`
				Total  int `json:"total"`
			} `json:"issueCountProgress"`
			IssueEstimateProgress struct {
				Open   int `json:"open"`
				Closed int `json:"closed"`
				Total  int `json:"total"`
			} `json:"issueEstimateProgress"`
		} `json:"node"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return exitcode.General("parsing epic progress", err)
	}

	if resp.Node == nil {
		return exitcode.NotFoundError(fmt.Sprintf("epic %q not found", epicID))
	}

	epic := resp.Node

	if output.IsJSON(outputFormat) {
		return output.JSON(w, map[string]any{
			"id":    epic.ID,
			"title": epic.Issue.Title,
			"state": epic.Issue.State,
			"issues": map[string]any{
				"open":   epic.IssueCountProgress.Open,
				"closed": epic.IssueCountProgress.Closed,
				"total":  epic.IssueCountProgress.Total,
			},
			"estimates": map[string]any{
				"open":   epic.IssueEstimateProgress.Open,
				"closed": epic.IssueEstimateProgress.Closed,
				"total":  epic.IssueEstimateProgress.Total,
			},
		})
	}

	title := fmt.Sprintf("%s#%d: %s", epic.Issue.Repository.Name, epic.Issue.Number, epic.Issue.Title)
	d := output.NewDetailWriter(w, "EPIC PROGRESS", title)
	d.Fields([]output.KeyValue{
		output.KV("State", formatEpicState(epic.Issue.State)),
	})

	if epic.IssueCountProgress.Total > 0 {
		d.Section("PROGRESS")
		fmt.Fprintf(w, "Issues:     %s\n", output.FormatProgress(epic.IssueCountProgress.Closed, epic.IssueCountProgress.Total))
		if epic.IssueEstimateProgress.Total > 0 {
			fmt.Fprintf(w, "Estimates:  %s\n", output.FormatProgress(epic.IssueEstimateProgress.Closed, epic.IssueEstimateProgress.Total))
		}
	} else {
		d.Section("PROGRESS")
		fmt.Fprintln(w, "No child issues.")
	}

	return nil
}

// renderEpicChildIssues renders the child issues section for an epic.
func renderEpicChildIssues(w writerFlusher, d *output.DetailWriter, issues []epicChildIssueNode, totalCount int) {
	if totalCount == 0 {
		return
	}

	d.Section(fmt.Sprintf("CHILD ISSUES (%d)", totalCount))

	if len(issues) == 0 {
		fmt.Fprintf(w, "%d child issue(s).\n", totalCount)
		return
	}

	// Check if we need long-form refs
	needLongRef := epicChildRepoNamesAmbiguous(issues)

	lw := output.NewListWriter(w, "ISSUE", "STATE", "TITLE", "EST", "PIPELINE")
	for _, issue := range issues {
		ref := epicChildFormatRef(issue, needLongRef)

		state := strings.ToLower(issue.State)

		title := issue.Title
		if len(title) > 40 {
			title = title[:37] + "..."
		}

		est := output.TableMissing
		if issue.Estimate != nil {
			est = formatEstimate(issue.Estimate.Value)
		}

		pipeline := output.TableMissing
		if issue.PipelineIssue != nil {
			pipeline = issue.PipelineIssue.Pipeline.Name
		}

		lw.Row(output.Cyan(ref), state, title, est, pipeline)
	}

	footer := fmt.Sprintf("Showing %d of %d child issue(s)", len(issues), totalCount)
	lw.FlushWithFooter(footer)
}

// renderBlockItems renders a list of blocking or blocked items.
func renderBlockItems(w writerFlusher, items []json.RawMessage) {
	for _, raw := range items {
		// Try as issue first
		var issue blockItemIssue
		if err := json.Unmarshal(raw, &issue); err == nil && issue.Number > 0 {
			ref := fmt.Sprintf("%s#%d", issue.Repository.Name, issue.Number)
			fmt.Fprintf(w, "  %s  %s\n", output.Cyan(ref), issue.Title)
			continue
		}
		// Try as epic
		var epic blockItemEpic
		if err := json.Unmarshal(raw, &epic); err == nil && epic.Title != "" {
			fmt.Fprintf(w, "  Epic: %s\n", epic.Title)
			continue
		}
	}
}

// epicChildFormatRef formats a child issue reference.
func epicChildFormatRef(issue epicChildIssueNode, longForm bool) string {
	if longForm {
		return fmt.Sprintf("%s/%s#%d", issue.Repository.OwnerName, issue.Repository.Name, issue.Number)
	}
	return fmt.Sprintf("%s#%d", issue.Repository.Name, issue.Number)
}

// epicChildRepoNamesAmbiguous checks if repo names are ambiguous.
func epicChildRepoNamesAmbiguous(issues []epicChildIssueNode) bool {
	seen := make(map[string]string) // name -> owner
	for _, issue := range issues {
		name := issue.Repository.Name
		owner := issue.Repository.OwnerName
		if prev, ok := seen[name]; ok && prev != owner {
			return true
		}
		seen[name] = owner
	}
	return false
}

// formatEpicState formats an epic state for display.
func formatEpicState(state string) string {
	lower := strings.ToLower(state)
	switch lower {
	case "open":
		return output.Green("open")
	case "todo":
		return "todo"
	case "in_progress":
		return output.Yellow("in_progress")
	case "closed":
		return output.Green("closed")
	default:
		return lower
	}
}

// formatEpicDates formats start/end date strings for the list view.
func formatEpicDates(startOn, endOn string) string {
	start, startErr := time.Parse("2006-01-02", startOn)
	end, endErr := time.Parse("2006-01-02", endOn)

	if startErr == nil && endErr == nil {
		return output.FormatDateRange(start, end)
	}
	if startErr == nil {
		return output.FormatDate(start) + " →"
	}
	if endErr == nil {
		return "→ " + output.FormatDate(end)
	}
	return output.TableMissing
}

// formatEpicDatePtrs formats optional start/end date pointers for detail view.
func formatEpicDatePtrs(startOn, endOn *string) string {
	var start, end time.Time
	var hasStart, hasEnd bool

	if startOn != nil && *startOn != "" {
		if t, err := time.Parse("2006-01-02", *startOn); err == nil {
			start = t
			hasStart = true
		}
	}
	if endOn != nil && *endOn != "" {
		if t, err := time.Parse("2006-01-02", *endOn); err == nil {
			end = t
			hasEnd = true
		}
	}

	if hasStart && hasEnd {
		return output.FormatDateRange(start, end)
	}
	if hasStart {
		return output.FormatDate(start) + " →"
	}
	if hasEnd {
		return "→ " + output.FormatDate(end)
	}
	return output.DetailMissing
}
