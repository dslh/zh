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

// Pipeline detail types (for show command)

type pipelineDetail struct {
	ID                  string                `json:"id"`
	Name                string                `json:"name"`
	Description         *string               `json:"description"`
	Stage               *string               `json:"stage"`
	IsDefaultPRPipeline bool                  `json:"isDefaultPRPipeline"`
	CreatedAt           string                `json:"createdAt"`
	UpdatedAt           string                `json:"updatedAt"`
	PipelineConfig      *pipelineConfigDetail `json:"pipelineConfiguration"`
	Issues              issueCountConn        `json:"issues"`
}

type pipelineConfigDetail struct {
	ShowAgeInPipeline *bool `json:"showAgeInPipeline"`
	StaleIssues       *bool `json:"staleIssues"`
	StaleInterval     *int  `json:"staleInterval"`
}

type issueCountConn struct {
	TotalCount int `json:"totalCount"`
}

// Pipeline list types (richer than cached pipeline, includes counts)

type pipelineListEntry struct {
	ID                  string  `json:"id"`
	Name                string  `json:"name"`
	Description         *string `json:"description"`
	Stage               *string `json:"stage"`
	IsDefaultPRPipeline bool    `json:"isDefaultPRPipeline"`
	Issues              struct {
		TotalCount int `json:"totalCount"`
	} `json:"issues"`
}

// Issue types for pipeline show

type pipelineIssueNode struct {
	ID          string `json:"id"`
	Number      int    `json:"number"`
	Title       string `json:"title"`
	State       string `json:"state"`
	PullRequest bool   `json:"pullRequest"`
	Estimate    *struct {
		Value float64 `json:"value"`
	} `json:"estimate"`
	Assignees struct {
		Nodes []struct {
			Login string `json:"login"`
		} `json:"nodes"`
	} `json:"assignees"`
	Labels struct {
		Nodes []struct {
			Name string `json:"name"`
		} `json:"nodes"`
	} `json:"labels"`
	Repository struct {
		Name      string `json:"name"`
		OwnerName string `json:"ownerName"`
	} `json:"repository"`
	BlockingIssues struct {
		TotalCount int `json:"totalCount"`
	} `json:"blockingIssues"`
	PipelineIssue *struct {
		Priority *struct {
			Name string `json:"name"`
		} `json:"priority"`
	} `json:"pipelineIssue"`
}

// GraphQL queries

const listPipelinesFullQuery = `query ListPipelinesFull($workspaceId: ID!) {
  workspace(id: $workspaceId) {
    pipelinesConnection(first: 50) {
      totalCount
      nodes {
        id
        name
        description
        stage
        isDefaultPRPipeline
        issues {
          totalCount
        }
      }
    }
  }
}`

const pipelineDetailQuery = `query GetPipelineDetails($pipelineId: ID!) {
  node(id: $pipelineId) {
    ... on Pipeline {
      id
      name
      description
      stage
      isDefaultPRPipeline
      createdAt
      updatedAt
      pipelineConfiguration {
        showAgeInPipeline
        staleIssues
        staleInterval
      }
      issues {
        totalCount
      }
    }
  }
}`

const pipelineIssuesQuery = `query GetPipelineIssues(
  $pipelineId: ID!
  $workspaceId: ID!
  $first: Int
  $after: String
) {
  searchIssuesByPipeline(
    pipelineId: $pipelineId
    filters: {}
    first: $first
    after: $after
  ) {
    totalCount
    pageInfo {
      hasNextPage
      endCursor
    }
    nodes {
      id
      number
      title
      state
      pullRequest
      estimate {
        value
      }
      assignees {
        nodes {
          login
        }
      }
      labels {
        nodes {
          name
        }
      }
      repository {
        name
        ownerName
      }
      blockingIssues {
        totalCount
      }
      pipelineIssue(workspaceId: $workspaceId) {
        priority {
          name
        }
      }
    }
  }
}`

// Commands

var pipelineCmd = &cobra.Command{
	Use:   "pipeline",
	Short: "Manage pipelines (board columns)",
	Long:  `List, view, and manage pipelines in the current ZenHub workspace.`,
}

var pipelineListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all pipelines in the workspace",
	Long:  `List all pipelines in the current workspace with position order.`,
	RunE:  runPipelineList,
}

var pipelineShowCmd = &cobra.Command{
	Use:   "show <name>",
	Short: "View details about a pipeline and the issues in it",
	Long:  `Display pipeline details including configuration and issues. Resolve pipeline by name, substring, alias, or ID.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runPipelineShow,
}

var (
	pipelineShowLimit int
	pipelineShowAll   bool
)

func init() {
	pipelineShowCmd.Flags().IntVar(&pipelineShowLimit, "limit", 100, "Maximum number of issues to show")
	pipelineShowCmd.Flags().BoolVar(&pipelineShowAll, "all", false, "Show all issues (ignore --limit)")

	pipelineCmd.AddCommand(pipelineListCmd)
	pipelineCmd.AddCommand(pipelineShowCmd)
	rootCmd.AddCommand(pipelineCmd)
}

// runPipelineList implements `zh pipeline list`.
func runPipelineList(cmd *cobra.Command, args []string) error {
	cfg, err := requireWorkspace()
	if err != nil {
		return err
	}

	client := newClient(cfg, cmd)
	w := cmd.OutOrStdout()

	data, err := client.Execute(listPipelinesFullQuery, map[string]any{
		"workspaceId": cfg.Workspace,
	})
	if err != nil {
		return exitcode.General("fetching pipelines", err)
	}

	var resp struct {
		Workspace struct {
			PipelinesConnection struct {
				TotalCount int                 `json:"totalCount"`
				Nodes      []pipelineListEntry `json:"nodes"`
			} `json:"pipelinesConnection"`
		} `json:"workspace"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return exitcode.General("parsing pipelines response", err)
	}

	pipelines := resp.Workspace.PipelinesConnection.Nodes

	// Cache pipeline list for resolution
	cachePipelinesFromList(pipelines, cfg.Workspace)

	if output.IsJSON(outputFormat) {
		return output.JSON(w, pipelines)
	}

	if len(pipelines) == 0 {
		fmt.Fprintln(w, "No pipelines found.")
		return nil
	}

	lw := output.NewListWriter(w, "#", "PIPELINE", "ISSUES", "STAGE", "DEFAULT PR")
	for i, p := range pipelines {
		stage := output.TableMissing
		if p.Stage != nil && *p.Stage != "" {
			stage = formatStage(*p.Stage)
		}

		defaultPR := "no"
		if p.IsDefaultPRPipeline {
			defaultPR = "yes"
		}

		lw.Row(
			fmt.Sprintf("%d", i+1),
			p.Name,
			fmt.Sprintf("%d", p.Issues.TotalCount),
			stage,
			defaultPR,
		)
	}

	lw.FlushWithFooter(fmt.Sprintf("Total: %d pipeline(s)", len(pipelines)))
	return nil
}

// cachePipelinesFromList stores pipeline entries in the cache for resolution.
func cachePipelinesFromList(pipelines []pipelineListEntry, workspaceID string) {
	var entries []resolve.CachedPipeline
	for _, p := range pipelines {
		entries = append(entries, resolve.CachedPipeline{
			ID:   p.ID,
			Name: p.Name,
		})
	}
	_ = resolve.FetchPipelinesIntoCache(entries, workspaceID)
}

// runPipelineShow implements `zh pipeline show <name>`.
func runPipelineShow(cmd *cobra.Command, args []string) error {
	cfg, err := requireWorkspace()
	if err != nil {
		return err
	}

	client := newClient(cfg, cmd)
	w := cmd.OutOrStdout()

	// Resolve the pipeline
	resolved, err := resolve.Pipeline(client, cfg.Workspace, args[0], cfg.Aliases.Pipelines)
	if err != nil {
		return err
	}

	// Fetch full pipeline details
	data, err := client.Execute(pipelineDetailQuery, map[string]any{
		"pipelineId": resolved.ID,
	})
	if err != nil {
		return exitcode.General("fetching pipeline details", err)
	}

	var resp struct {
		Node pipelineDetail `json:"node"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return exitcode.General("parsing pipeline details", err)
	}

	detail := resp.Node

	// Fetch issues
	limit := pipelineShowLimit
	if pipelineShowAll {
		limit = 0 // fetch all
	}
	issues, totalCount, err := fetchPipelineIssues(client, resolved.ID, cfg.Workspace, limit)
	if err != nil {
		return err
	}

	if output.IsJSON(outputFormat) {
		jsonOut := struct {
			Pipeline pipelineDetail      `json:"pipeline"`
			Issues   []pipelineIssueNode `json:"issues"`
			Total    int                 `json:"totalIssues"`
		}{
			Pipeline: detail,
			Issues:   issues,
			Total:    totalCount,
		}
		return output.JSON(w, jsonOut)
	}

	// Determine repo context for short references
	needLongRef := repoNamesAmbiguous(issues)

	// Detail view
	d := output.NewDetailWriter(w, "PIPELINE", detail.Name)

	fields := []output.KeyValue{
		output.KV("ID", output.Cyan(detail.ID)),
	}

	if detail.Description != nil && *detail.Description != "" {
		fields = append(fields, output.KV("Description", *detail.Description))
	}

	stage := output.DetailMissing
	if detail.Stage != nil && *detail.Stage != "" {
		stage = formatStage(*detail.Stage)
	}
	fields = append(fields, output.KV("Stage", stage))
	fields = append(fields, output.KV("Issues", fmt.Sprintf("%d", detail.Issues.TotalCount)))

	if detail.IsDefaultPRPipeline {
		fields = append(fields, output.KV("Default PR pipeline", output.Green("yes")))
	}

	if detail.PipelineConfig != nil {
		pc := detail.PipelineConfig
		if pc.StaleIssues != nil && *pc.StaleIssues && pc.StaleInterval != nil {
			fields = append(fields, output.KV("Stale after", fmt.Sprintf("%d days", *pc.StaleInterval)))
		}
	}

	if detail.CreatedAt != "" {
		if t, err := time.Parse(time.RFC3339, detail.CreatedAt); err == nil {
			fields = append(fields, output.KV("Created", output.FormatDate(t)))
		}
	}

	d.Fields(fields)

	// Issues section
	if len(issues) > 0 {
		d.Section("ISSUES")

		lw := output.NewListWriter(w, "ISSUE", "TITLE", "EST", "ASSIGNEE", "PRIORITY")
		for _, issue := range issues {
			ref := formatIssueRef(issue, needLongRef)
			title := issue.Title
			if len(title) > 50 {
				title = title[:47] + "..."
			}

			est := output.TableMissing
			if issue.Estimate != nil {
				est = formatEstimate(issue.Estimate.Value)
			}

			assignee := output.TableMissing
			if len(issue.Assignees.Nodes) > 0 {
				logins := make([]string, len(issue.Assignees.Nodes))
				for i, a := range issue.Assignees.Nodes {
					logins[i] = a.Login
				}
				assignee = strings.Join(logins, ", ")
			}

			priority := output.TableMissing
			if issue.PipelineIssue != nil && issue.PipelineIssue.Priority != nil {
				priority = issue.PipelineIssue.Priority.Name
			}

			lw.Row(ref, title, est, assignee, priority)
		}

		footer := fmt.Sprintf("Showing %d of %d issue(s)", len(issues), totalCount)
		lw.FlushWithFooter(footer)
	} else if totalCount > 0 {
		d.Section("ISSUES")
		fmt.Fprintf(w, "%d issue(s) in this pipeline.\n", totalCount)
	}

	return nil
}

// fetchPipelineIssues fetches issues in a pipeline with pagination.
// If limit is 0, fetches all issues.
func fetchPipelineIssues(client *api.Client, pipelineID, workspaceID string, limit int) ([]pipelineIssueNode, int, error) {
	var allIssues []pipelineIssueNode
	var cursor *string
	totalCount := 0
	pageSize := 50

	for {
		remaining := 0
		if limit > 0 {
			remaining = limit - len(allIssues)
			if remaining <= 0 {
				break
			}
			if remaining < pageSize {
				pageSize = remaining
			}
		}

		vars := map[string]any{
			"pipelineId":  pipelineID,
			"workspaceId": workspaceID,
			"first":       pageSize,
		}
		if cursor != nil {
			vars["after"] = *cursor
		}

		data, err := client.Execute(pipelineIssuesQuery, vars)
		if err != nil {
			return nil, 0, exitcode.General("fetching pipeline issues", err)
		}

		var resp struct {
			SearchIssuesByPipeline struct {
				TotalCount int `json:"totalCount"`
				PageInfo   struct {
					HasNextPage bool   `json:"hasNextPage"`
					EndCursor   string `json:"endCursor"`
				} `json:"pageInfo"`
				Nodes []pipelineIssueNode `json:"nodes"`
			} `json:"searchIssuesByPipeline"`
		}
		if err := json.Unmarshal(data, &resp); err != nil {
			return nil, 0, exitcode.General("parsing pipeline issues", err)
		}

		totalCount = resp.SearchIssuesByPipeline.TotalCount
		allIssues = append(allIssues, resp.SearchIssuesByPipeline.Nodes...)

		if !resp.SearchIssuesByPipeline.PageInfo.HasNextPage {
			break
		}

		if limit > 0 && len(allIssues) >= limit {
			break
		}

		cursor = &resp.SearchIssuesByPipeline.PageInfo.EndCursor
	}

	return allIssues, totalCount, nil
}

// formatIssueRef formats an issue reference (repo#number or owner/repo#number).
func formatIssueRef(issue pipelineIssueNode, longForm bool) string {
	if longForm {
		return fmt.Sprintf("%s/%s#%d", issue.Repository.OwnerName, issue.Repository.Name, issue.Number)
	}
	return fmt.Sprintf("%s#%d", issue.Repository.Name, issue.Number)
}

// repoNamesAmbiguous checks if any repo name appears with different owners
// in the issue set, requiring long-form references.
func repoNamesAmbiguous(issues []pipelineIssueNode) bool {
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

// formatStage formats a pipeline stage enum value for display.
func formatStage(stage string) string {
	switch stage {
	case "BACKLOG":
		return "Backlog"
	case "SPRINT_BACKLOG":
		return "Sprint Backlog"
	case "DEVELOPMENT":
		return "Development"
	case "REVIEW":
		return "Review"
	case "COMPLETED":
		return "Completed"
	default:
		return stage
	}
}
