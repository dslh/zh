package cmd

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/dslh/zh/internal/api"
	"github.com/dslh/zh/internal/exitcode"
	"github.com/dslh/zh/internal/output"
	"github.com/dslh/zh/internal/resolve"
	"github.com/spf13/cobra"
)

// Sprint list types

type sprintListEntry struct {
	ID              string  `json:"id"`
	Name            string  `json:"name"`
	GeneratedName   string  `json:"generatedName"`
	Description     string  `json:"description,omitempty"`
	State           string  `json:"state"`
	StartAt         string  `json:"startAt"`
	EndAt           string  `json:"endAt"`
	TotalPoints     float64 `json:"totalPoints"`
	CompletedPoints float64 `json:"completedPoints"`
	ClosedIssues    int     `json:"closedIssuesCount"`
	CreatedAt       string  `json:"createdAt,omitempty"`
	UpdatedAt       string  `json:"updatedAt,omitempty"`
}

func (s *sprintListEntry) DisplayName() string {
	if s.Name != "" {
		return s.Name
	}
	return s.GeneratedName
}

// Sprint show types

type sprintDetail struct {
	ID              string  `json:"id"`
	Name            string  `json:"name"`
	GeneratedName   string  `json:"generatedName"`
	Description     string  `json:"description"`
	State           string  `json:"state"`
	StartAt         string  `json:"startAt"`
	EndAt           string  `json:"endAt"`
	TotalPoints     float64 `json:"totalPoints"`
	CompletedPoints float64 `json:"completedPoints"`
	ClosedIssues    int     `json:"closedIssuesCount"`
	CreatedAt       string  `json:"createdAt"`
	UpdatedAt       string  `json:"updatedAt"`
	SprintIssues    struct {
		TotalCount int               `json:"totalCount"`
		PageInfo   pageInfoNode      `json:"pageInfo"`
		Nodes      []sprintIssueNode `json:"nodes"`
	} `json:"sprintIssues"`
}

type pageInfoNode struct {
	HasNextPage bool   `json:"hasNextPage"`
	EndCursor   string `json:"endCursor"`
}

func (s *sprintDetail) DisplayName() string {
	if s.Name != "" {
		return s.Name
	}
	return s.GeneratedName
}

type sprintIssueNode struct {
	ID    string `json:"id"`
	Issue struct {
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
		Assignees struct {
			Nodes []struct {
				Login string `json:"login"`
			} `json:"nodes"`
		} `json:"assignees"`
		PipelineIssues struct {
			Nodes []struct {
				Pipeline struct {
					Name string `json:"name"`
				} `json:"pipeline"`
			} `json:"nodes"`
		} `json:"pipelineIssues"`
	} `json:"issue"`
}

// GraphQL queries

const sprintListQuery = `query ListSprints($workspaceId: ID!, $first: Int!, $after: String, $filters: SprintFiltersInput, $orderBy: SprintOrderInput) {
  workspace(id: $workspaceId) {
    sprints(first: $first, after: $after, filters: $filters, orderBy: $orderBy) {
      totalCount
      pageInfo {
        hasNextPage
        endCursor
      }
      nodes {
        id
        name
        generatedName
        description
        state
        startAt
        endAt
        totalPoints
        completedPoints
        closedIssuesCount
        createdAt
        updatedAt
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

const sprintShowQuery = `query GetSprint($sprintId: ID!) {
  node(id: $sprintId) {
    ... on Sprint {
      id
      name
      generatedName
      description
      state
      startAt
      endAt
      totalPoints
      completedPoints
      closedIssuesCount
      createdAt
      updatedAt
      sprintIssues(first: 100) {
        totalCount
        pageInfo {
          hasNextPage
          endCursor
        }
        nodes {
          id
          issue {
            id
            number
            title
            state
            estimate { value }
            repository { name ownerName }
            assignees(first: 10) {
              nodes { login }
            }
            pipelineIssues(first: 1) {
              nodes {
                pipeline { name }
              }
            }
          }
        }
      }
    }
  }
}`

const sprintShowIssuesPageQuery = `query GetSprintIssues($sprintId: ID!, $first: Int!, $after: String) {
  node(id: $sprintId) {
    ... on Sprint {
      sprintIssues(first: $first, after: $after) {
        totalCount
        pageInfo {
          hasNextPage
          endCursor
        }
        nodes {
          id
          issue {
            id
            number
            title
            state
            estimate { value }
            repository { name ownerName }
            assignees(first: 10) {
              nodes { login }
            }
            pipelineIssues(first: 1) {
              nodes {
                pipeline { name }
              }
            }
          }
        }
      }
    }
  }
}`

// Commands

var sprintCmd = &cobra.Command{
	Use:   "sprint",
	Short: "View and manage sprints",
	Long:  `List, view, and manage sprints in the current ZenHub workspace.`,
}

var sprintListCmd = &cobra.Command{
	Use:   "list",
	Short: "List sprints in the workspace",
	Long: `List sprints in the current workspace. By default, shows the active sprint,
upcoming sprint, and a few recent closed sprints.

Use --state to filter by sprint state:
  --state=open     Show only open (active/upcoming) sprints
  --state=closed   Show only closed sprints
  --state=all      Show all sprints`,
	Args: cobra.NoArgs,
	RunE: runSprintList,
}

var sprintShowCmd = &cobra.Command{
	Use:   "show [sprint]",
	Short: "View sprint details and issues",
	Long: `Display detailed information about a sprint, including progress and issues.

Defaults to the active sprint if no sprint identifier is provided.

The sprint can be specified as:
  - ZenHub ID
  - sprint name or unique name substring
  - relative reference: current, next, previous`,
	Args: cobra.MaximumNArgs(1),
	RunE: runSprintShow,
}

var (
	sprintListLimit int
	sprintListAll   bool
	sprintListState string

	sprintShowLimit int
	sprintShowAll   bool
)

func init() {
	output.AddPaginationFlags(sprintListCmd, &sprintListLimit, &sprintListAll)
	sprintListCmd.Flags().StringVar(&sprintListState, "state", "", "Filter by state: open, closed, all (default: recent)")

	output.AddPaginationFlags(sprintShowCmd, &sprintShowLimit, &sprintShowAll)

	sprintCmd.AddCommand(sprintListCmd)
	sprintCmd.AddCommand(sprintShowCmd)
	rootCmd.AddCommand(sprintCmd)
}

func resetSprintFlags() {
	sprintListLimit = 100
	sprintListAll = false
	sprintListState = ""
	sprintShowLimit = 100
	sprintShowAll = false
}

// runSprintList implements `zh sprint list`.
func runSprintList(cmd *cobra.Command, args []string) error {
	cfg, err := requireWorkspace()
	if err != nil {
		return err
	}

	client := newClient(cfg, cmd)
	w := cmd.OutOrStdout()

	limit := output.EffectiveLimit(sprintListLimit, sprintListAll)

	sprints, activeID, totalCount, err := fetchSprintList(client, cfg.Workspace, limit, sprintListState)
	if err != nil {
		return err
	}

	// Cache sprints for resolution
	cacheSprintsFromList(sprints, cfg.Workspace)

	if output.IsJSON(outputFormat) {
		return output.JSON(w, sprints)
	}

	if len(sprints) == 0 {
		fmt.Fprintln(w, "No sprints found.")
		return nil
	}

	lw := output.NewListWriter(w, "STATE", "NAME", "DATES", "POINTS", "CLOSED")
	for _, s := range sprints {
		state := formatSprintState(s.State, s.ID, activeID)
		name := s.DisplayName()
		dates := formatSprintDates(s.StartAt, s.EndAt)

		points := output.TableMissing
		if s.TotalPoints > 0 {
			points = fmt.Sprintf("%s/%s", formatEstimate(s.CompletedPoints), formatEstimate(s.TotalPoints))
		}

		closed := fmt.Sprintf("%d", s.ClosedIssues)

		lw.Row(state, name, dates, points, closed)
	}

	footer := fmt.Sprintf("Showing %d", len(sprints))
	if totalCount > len(sprints) {
		footer += fmt.Sprintf(" of %d", totalCount)
	}
	footer += " sprint(s)"
	lw.FlushWithFooter(footer)
	return nil
}

// fetchSprintList fetches sprints from the workspace with pagination.
// Returns the sprints, the active sprint ID, and the total count.
func fetchSprintList(client *api.Client, workspaceID string, limit int, stateFilter string) ([]sprintListEntry, string, int, error) {
	var allSprints []sprintListEntry
	var cursor *string
	var activeID string
	totalCount := 0
	pageSize := 50

	for {
		if limit > 0 {
			remaining := limit - len(allSprints)
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
			"orderBy": map[string]any{
				"field":     "START_AT",
				"direction": "DESC",
			},
		}
		if cursor != nil {
			vars["after"] = *cursor
		}

		// Apply state filter
		switch strings.ToLower(stateFilter) {
		case "open":
			vars["filters"] = map[string]any{"state": map[string]any{"eq": "OPEN"}}
		case "closed":
			vars["filters"] = map[string]any{"state": map[string]any{"eq": "CLOSED"}}
		case "all", "":
			// no filter
		default:
			return nil, "", 0, exitcode.Usage(fmt.Sprintf("invalid --state value %q: must be open, closed, or all", stateFilter))
		}

		data, err := client.Execute(sprintListQuery, vars)
		if err != nil {
			return nil, "", 0, exitcode.General("fetching sprints", err)
		}

		var resp struct {
			Workspace struct {
				Sprints struct {
					TotalCount int `json:"totalCount"`
					PageInfo   struct {
						HasNextPage bool   `json:"hasNextPage"`
						EndCursor   string `json:"endCursor"`
					} `json:"pageInfo"`
					Nodes []sprintListEntry `json:"nodes"`
				} `json:"sprints"`
				ActiveSprint   *struct{ ID string } `json:"activeSprint"`
				UpcomingSprint *struct{ ID string } `json:"upcomingSprint"`
				PreviousSprint *struct{ ID string } `json:"previousSprint"`
			} `json:"workspace"`
		}
		if err := json.Unmarshal(data, &resp); err != nil {
			return nil, "", 0, exitcode.General("parsing sprints response", err)
		}

		totalCount = resp.Workspace.Sprints.TotalCount

		// Capture active sprint ID from first page
		if cursor == nil && resp.Workspace.ActiveSprint != nil {
			activeID = resp.Workspace.ActiveSprint.ID
		}

		allSprints = append(allSprints, resp.Workspace.Sprints.Nodes...)

		if !resp.Workspace.Sprints.PageInfo.HasNextPage {
			break
		}
		if limit > 0 && len(allSprints) >= limit {
			break
		}

		cursor = &resp.Workspace.Sprints.PageInfo.EndCursor
	}

	return allSprints, activeID, totalCount, nil
}

// cacheSprintsFromList stores sprint entries in the cache for resolution.
func cacheSprintsFromList(sprints []sprintListEntry, workspaceID string) {
	var entries []resolve.CachedSprint
	for _, s := range sprints {
		entries = append(entries, resolve.CachedSprint{
			ID:            s.ID,
			Name:          s.Name,
			GeneratedName: s.GeneratedName,
			State:         s.State,
			StartAt:       s.StartAt,
			EndAt:         s.EndAt,
		})
	}
	_ = resolve.FetchSprintsIntoCache(entries, workspaceID)
}

// runSprintShow implements `zh sprint show [sprint]`.
func runSprintShow(cmd *cobra.Command, args []string) error {
	cfg, err := requireWorkspace()
	if err != nil {
		return err
	}

	client := newClient(cfg, cmd)
	w := cmd.OutOrStdout()

	// Resolve sprint — default to "current" if no argument provided
	identifier := "current"
	if len(args) > 0 {
		identifier = args[0]
	}

	resolved, err := resolve.Sprint(client, cfg.Workspace, identifier)
	if err != nil {
		return err
	}

	// Fetch sprint detail
	data, err := client.Execute(sprintShowQuery, map[string]any{
		"sprintId": resolved.ID,
	})
	if err != nil {
		return exitcode.General("fetching sprint details", err)
	}

	var resp struct {
		Node *sprintDetail `json:"node"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return exitcode.General("parsing sprint details", err)
	}

	if resp.Node == nil {
		return exitcode.NotFoundError(fmt.Sprintf("sprint %q not found", identifier))
	}

	sprint := resp.Node

	// Paginate remaining issues if needed
	limit := output.EffectiveLimit(sprintShowLimit, sprintShowAll)
	if sprint.SprintIssues.PageInfo.HasNextPage && (limit == 0 || len(sprint.SprintIssues.Nodes) < limit) {
		cursor := sprint.SprintIssues.PageInfo.EndCursor
		for limit == 0 || len(sprint.SprintIssues.Nodes) < limit {
			pageVars := map[string]any{
				"sprintId": resolved.ID,
				"first":    100,
				"after":    cursor,
			}

			pageData, err := client.Execute(sprintShowIssuesPageQuery, pageVars)
			if err != nil {
				break // partial data is better than no data
			}

			var pageResp struct {
				Node *struct {
					SprintIssues struct {
						TotalCount int               `json:"totalCount"`
						PageInfo   pageInfoNode      `json:"pageInfo"`
						Nodes      []sprintIssueNode `json:"nodes"`
					} `json:"sprintIssues"`
				} `json:"node"`
			}
			if err := json.Unmarshal(pageData, &pageResp); err != nil || pageResp.Node == nil {
				break
			}

			sprint.SprintIssues.Nodes = append(sprint.SprintIssues.Nodes, pageResp.Node.SprintIssues.Nodes...)

			if !pageResp.Node.SprintIssues.PageInfo.HasNextPage {
				break
			}
			cursor = pageResp.Node.SprintIssues.PageInfo.EndCursor
		}
	}

	// Truncate to limit
	if limit > 0 {
		sprint.SprintIssues.Nodes, _ = output.Truncate(sprint.SprintIssues.Nodes, limit)
	}

	if output.IsJSON(outputFormat) {
		return output.JSON(w, sprint)
	}

	return renderSprintDetail(w, sprint)
}

// renderSprintDetail renders the sprint detail view.
func renderSprintDetail(w writerFlusher, sprint *sprintDetail) error {
	title := sprint.DisplayName()
	d := output.NewDetailWriter(w, "SPRINT", title)

	state := strings.ToLower(sprint.State)
	if state == "open" {
		state = output.Green("open")
	} else {
		state = output.Dim("closed")
	}

	fields := []output.KeyValue{
		output.KV("ID", output.Cyan(sprint.ID)),
		output.KV("State", state),
	}

	// Dates
	dates := formatSprintDates(sprint.StartAt, sprint.EndAt)
	if dates != output.TableMissing {
		duration := formatSprintDuration(sprint.StartAt, sprint.EndAt)
		if duration != "" {
			dates += fmt.Sprintf(" (%s)", duration)
		}
		fields = append(fields, output.KV("Dates", dates))
	}

	if sprint.Description != "" {
		fields = append(fields, output.KV("Description", sprint.Description))
	}

	d.Fields(fields)

	// Progress section
	d.Section("PROGRESS")
	if sprint.TotalPoints > 0 {
		fmt.Fprintf(w, "Points:  %s\n", output.FormatProgress(int(sprint.CompletedPoints), int(sprint.TotalPoints)))
	} else {
		fmt.Fprintln(w, "Points:  No estimates")
	}

	totalIssues := sprint.SprintIssues.TotalCount
	if totalIssues > 0 {
		fmt.Fprintf(w, "Issues:  %s\n", output.FormatProgress(sprint.ClosedIssues, totalIssues))
	} else {
		fmt.Fprintln(w, "Issues:  No issues in sprint")
	}

	// Issues section
	if sprint.SprintIssues.TotalCount > 0 {
		renderSprintIssues(w, d, sprint.SprintIssues.Nodes, sprint.SprintIssues.TotalCount)
	}

	return nil
}

// renderSprintIssues renders the issues section of a sprint detail view.
func renderSprintIssues(w writerFlusher, d *output.DetailWriter, issues []sprintIssueNode, totalCount int) {
	d.Section(fmt.Sprintf("ISSUES (%d)", totalCount))

	if len(issues) == 0 {
		fmt.Fprintf(w, "%d issue(s).\n", totalCount)
		return
	}

	needLongRef := sprintIssueRepoNamesAmbiguous(issues)

	lw := output.NewListWriter(w, "ISSUE", "STATE", "TITLE", "EST", "PIPELINE", "ASSIGNEE")
	for _, si := range issues {
		issue := si.Issue

		ref := sprintIssueFormatRef(si, needLongRef)

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
		if len(issue.PipelineIssues.Nodes) > 0 {
			pipeline = issue.PipelineIssues.Nodes[0].Pipeline.Name
		}

		assignee := output.TableMissing
		if len(issue.Assignees.Nodes) > 0 {
			logins := make([]string, 0, len(issue.Assignees.Nodes))
			for _, a := range issue.Assignees.Nodes {
				logins = append(logins, "@"+a.Login)
			}
			assignee = strings.Join(logins, ", ")
		}

		lw.Row(output.Cyan(ref), state, title, est, pipeline, assignee)
	}

	footer := fmt.Sprintf("Showing %d of %d issue(s)", len(issues), totalCount)
	lw.FlushWithFooter(footer)
}

// formatSprintState formats a sprint state for list display.
func formatSprintState(state, id, activeID string) string {
	if id == activeID && activeID != "" {
		return output.Green("▶ active")
	}

	lower := strings.ToLower(state)
	switch lower {
	case "open":
		return output.Green("open")
	case "closed":
		return output.Dim("closed")
	default:
		return lower
	}
}

// formatSprintDates formats sprint start/end timestamps for display.
func formatSprintDates(startAt, endAt string) string {
	start, startErr := time.Parse(time.RFC3339, startAt)
	end, endErr := time.Parse(time.RFC3339, endAt)

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

// formatSprintDuration returns a human-readable duration between two RFC3339 timestamps.
func formatSprintDuration(startAt, endAt string) string {
	start, err1 := time.Parse(time.RFC3339, startAt)
	end, err2 := time.Parse(time.RFC3339, endAt)
	if err1 != nil || err2 != nil {
		return ""
	}

	days := int(math.Round(end.Sub(start).Hours() / 24))
	if days == 1 {
		return "1 day"
	}
	return fmt.Sprintf("%d days", days)
}

// sprintIssueFormatRef formats a sprint issue reference.
func sprintIssueFormatRef(si sprintIssueNode, longForm bool) string {
	issue := si.Issue
	if longForm {
		return fmt.Sprintf("%s/%s#%d", issue.Repository.OwnerName, issue.Repository.Name, issue.Number)
	}
	return fmt.Sprintf("%s#%d", issue.Repository.Name, issue.Number)
}

// sprintIssueRepoNamesAmbiguous checks if repo names are ambiguous across sprint issues.
func sprintIssueRepoNamesAmbiguous(issues []sprintIssueNode) bool {
	seen := make(map[string]string) // name -> owner
	for _, si := range issues {
		name := si.Issue.Repository.Name
		owner := si.Issue.Repository.OwnerName
		if prev, ok := seen[name]; ok && prev != owner {
			return true
		}
		seen[name] = owner
	}
	return false
}
