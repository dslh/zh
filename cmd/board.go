package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/dslh/zh/internal/api"
	"github.com/dslh/zh/internal/config"
	"github.com/dslh/zh/internal/exitcode"
	"github.com/dslh/zh/internal/output"
	"github.com/dslh/zh/internal/resolve"
	"github.com/spf13/cobra"
)

// Board types

type boardPipeline struct {
	ID     string         `json:"id"`
	Name   string         `json:"name"`
	Issues boardIssueConn `json:"issues"`
}

type boardIssueConn struct {
	TotalCount int              `json:"totalCount"`
	Nodes      []boardIssueNode `json:"nodes"`
}

type boardIssueNode struct {
	ID          string `json:"id"`
	Number      int    `json:"number"`
	Title       string `json:"title"`
	State       string `json:"state"`
	PullRequest bool   `json:"pullRequest"`
	Estimate    *struct {
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
	Labels struct {
		Nodes []struct {
			Name string `json:"name"`
		} `json:"nodes"`
	} `json:"labels"`
	PipelineIssue *struct {
		Priority *struct {
			Name string `json:"name"`
		} `json:"priority"`
	} `json:"pipelineIssue"`
}

// GraphQL query for the full board

const boardQuery = `query GetBoard($workspaceId: ID!) {
  workspace(id: $workspaceId) {
    id
    displayName
    pipelinesConnection(first: 50) {
      nodes {
        id
        name
        issues(first: 100) {
          totalCount
          nodes {
            id
            number
            title
            state
            pullRequest
            estimate {
              value
            }
            repository {
              name
              ownerName
            }
            assignees(first: 5) {
              nodes {
                login
              }
            }
            labels(first: 10) {
              nodes {
                name
              }
            }
            pipelineIssue(workspaceId: $workspaceId) {
              priority {
                name
              }
            }
          }
        }
      }
    }
  }
}`

// Commands

var boardCmd = &cobra.Command{
	Use:   "board",
	Short: "Display all pipelines with their issues",
	Long: `Display the workspace board showing all pipelines and their issues.

Use --pipeline to filter to a single pipeline.`,
	RunE: runBoard,
}

var (
	boardPipelineFilter string
)

func init() {
	boardCmd.Flags().StringVar(&boardPipelineFilter, "pipeline", "", "Show only the specified pipeline")

	rootCmd.AddCommand(boardCmd)
}

func resetBoardFlags() {
	boardPipelineFilter = ""
}

// runBoard implements `zh board`.
func runBoard(cmd *cobra.Command, args []string) error {
	cfg, err := requireWorkspace()
	if err != nil {
		return err
	}

	client := newClient(cfg, cmd)
	w := cmd.OutOrStdout()

	// If --pipeline is specified, use the single pipeline path
	if boardPipelineFilter != "" {
		return runBoardSinglePipeline(cmd, cfg, client)
	}

	// Fetch full board
	data, err := client.Execute(boardQuery, map[string]any{
		"workspaceId": cfg.Workspace,
	})
	if err != nil {
		return exitcode.General("fetching board", err)
	}

	var resp struct {
		Workspace struct {
			ID                  string `json:"id"`
			DisplayName         string `json:"displayName"`
			PipelinesConnection struct {
				Nodes []boardPipeline `json:"nodes"`
			} `json:"pipelinesConnection"`
		} `json:"workspace"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return exitcode.General("parsing board response", err)
	}

	pipelines := resp.Workspace.PipelinesConnection.Nodes

	// Cache pipeline list for resolution
	cachePipelinesFromBoard(pipelines, cfg.Workspace)

	if output.IsJSON(outputFormat) {
		return output.JSON(w, pipelines)
	}

	if len(pipelines) == 0 {
		fmt.Fprintln(w, "No pipelines found.")
		return nil
	}

	// Determine if long-form references are needed
	needLongRef := boardRepoNamesAmbiguous(pipelines)

	// Render each pipeline as a section
	totalIssues := 0
	for i, p := range pipelines {
		if i > 0 {
			fmt.Fprintln(w)
		}

		issueCountStr := fmt.Sprintf("%d", p.Issues.TotalCount)
		if len(p.Issues.Nodes) < p.Issues.TotalCount {
			issueCountStr = fmt.Sprintf("%d of %d", len(p.Issues.Nodes), p.Issues.TotalCount)
		}

		fmt.Fprintf(w, "%s  %s\n", output.Bold(p.Name), output.Dim(fmt.Sprintf("(%s issues)", issueCountStr)))
		fmt.Fprintln(w, strings.Repeat("─", 80))

		if len(p.Issues.Nodes) == 0 {
			fmt.Fprintln(w, output.Dim("  No issues"))
		} else {
			for _, issue := range p.Issues.Nodes {
				renderBoardIssue(w, issue, needLongRef)
			}
		}

		totalIssues += p.Issues.TotalCount
	}

	fmt.Fprintln(w)
	fmt.Fprintf(w, "%d pipeline(s), %d issue(s)\n", len(pipelines), totalIssues)

	return nil
}

// runBoardSinglePipeline fetches and displays a single pipeline when --pipeline is used.
func runBoardSinglePipeline(cmd *cobra.Command, cfg *config.Config, client *api.Client) error {
	w := cmd.OutOrStdout()

	// Resolve the pipeline
	resolved, err := resolve.Pipeline(client, cfg.Workspace, boardPipelineFilter, cfg.Aliases.Pipelines)
	if err != nil {
		return err
	}

	// Fetch issues using the existing pipeline issues query
	issues, totalCount, err := fetchPipelineIssues(client, resolved.ID, cfg.Workspace, 100)
	if err != nil {
		return err
	}

	if output.IsJSON(outputFormat) {
		jsonOut := []struct {
			ID     string         `json:"id"`
			Name   string         `json:"name"`
			Issues map[string]any `json:"issues"`
		}{
			{
				ID:   resolved.ID,
				Name: resolved.Name,
				Issues: map[string]any{
					"totalCount": totalCount,
					"nodes":      issues,
				},
			},
		}
		return output.JSON(w, jsonOut)
	}

	needLongRef := repoNamesAmbiguous(issues)

	issueCountStr := fmt.Sprintf("%d", totalCount)
	if len(issues) < totalCount {
		issueCountStr = fmt.Sprintf("%d of %d", len(issues), totalCount)
	}

	fmt.Fprintf(w, "%s  %s\n", output.Bold(resolved.Name), output.Dim(fmt.Sprintf("(%s issues)", issueCountStr)))
	fmt.Fprintln(w, strings.Repeat("─", 80))

	if len(issues) == 0 {
		fmt.Fprintln(w, output.Dim("  No issues"))
	} else {
		for _, issue := range issues {
			renderBoardIssueFromPipelineNode(w, issue, needLongRef)
		}
	}

	fmt.Fprintln(w)
	fmt.Fprintf(w, "1 pipeline, %d issue(s)\n", totalCount)

	return nil
}

// renderBoardIssue renders a single issue line for the board view.
func renderBoardIssue(w interface{ Write([]byte) (int, error) }, issue boardIssueNode, longRef bool) {
	ref := boardFormatIssueRef(issue, longRef)
	title := issue.Title
	if len(title) > 50 {
		title = title[:47] + "..."
	}

	est := ""
	if issue.Estimate != nil {
		est = fmt.Sprintf(" [%s]", formatEstimate(issue.Estimate.Value))
	}

	assignee := ""
	if len(issue.Assignees.Nodes) > 0 {
		logins := make([]string, len(issue.Assignees.Nodes))
		for i, a := range issue.Assignees.Nodes {
			logins[i] = a.Login
		}
		assignee = " @" + strings.Join(logins, ", @")
	}

	fmt.Fprintf(w, "  %s  %s%s%s\n", output.Cyan(ref), title, output.Dim(est), output.Dim(assignee))
}

// renderBoardIssueFromPipelineNode renders a pipelineIssueNode for --pipeline board view.
func renderBoardIssueFromPipelineNode(w interface{ Write([]byte) (int, error) }, issue pipelineIssueNode, longRef bool) {
	ref := formatIssueRef(issue, longRef)
	title := issue.Title
	if len(title) > 50 {
		title = title[:47] + "..."
	}

	est := ""
	if issue.Estimate != nil {
		est = fmt.Sprintf(" [%s]", formatEstimate(issue.Estimate.Value))
	}

	assignee := ""
	if len(issue.Assignees.Nodes) > 0 {
		logins := make([]string, len(issue.Assignees.Nodes))
		for i, a := range issue.Assignees.Nodes {
			logins[i] = a.Login
		}
		assignee = " @" + strings.Join(logins, ", @")
	}

	fmt.Fprintf(w, "  %s  %s%s%s\n", output.Cyan(ref), title, output.Dim(est), output.Dim(assignee))
}

// boardFormatIssueRef formats an issue reference for board display.
func boardFormatIssueRef(issue boardIssueNode, longForm bool) string {
	if longForm {
		return fmt.Sprintf("%s/%s#%d", issue.Repository.OwnerName, issue.Repository.Name, issue.Number)
	}
	return fmt.Sprintf("%s#%d", issue.Repository.Name, issue.Number)
}

// boardRepoNamesAmbiguous checks if any repo name appears with different owners.
func boardRepoNamesAmbiguous(pipelines []boardPipeline) bool {
	seen := make(map[string]string) // name -> owner
	for _, p := range pipelines {
		for _, issue := range p.Issues.Nodes {
			name := issue.Repository.Name
			owner := issue.Repository.OwnerName
			if prev, ok := seen[name]; ok && prev != owner {
				return true
			}
			seen[name] = owner
		}
	}
	return false
}

// cachePipelinesFromBoard stores pipeline entries in cache for resolution.
func cachePipelinesFromBoard(pipelines []boardPipeline, workspaceID string) {
	var entries []resolve.CachedPipeline
	for _, p := range pipelines {
		entries = append(entries, resolve.CachedPipeline{
			ID:   p.ID,
			Name: p.Name,
		})
	}
	_ = resolve.FetchPipelinesIntoCache(entries, workspaceID)
}
