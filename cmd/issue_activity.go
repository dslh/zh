package cmd

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/dslh/zh/internal/api"
	"github.com/dslh/zh/internal/exitcode"
	"github.com/dslh/zh/internal/output"
	"github.com/dslh/zh/internal/resolve"
	"github.com/spf13/cobra"
)

// GraphQL query using repo GH ID + issue number (issue-specific path)

const issueTimelineQuery = `query GetIssueTimeline($repositoryGhId: Int!, $issueNumber: Int!, $first: Int!, $after: String) {
  issueByInfo(repositoryGhId: $repositoryGhId, issueNumber: $issueNumber) {
    id
    number
    title
    repository {
      name
      owner { login }
    }
    timelineItems(first: $first, after: $after) {
      totalCount
      pageInfo {
        hasNextPage
        endCursor
      }
      nodes {
        id
        key
        data
        createdAt
      }
    }
  }
}`

// Commands

var issueActivityCmd = &cobra.Command{
	Use:   "activity <issue>",
	Short: "Show ZenHub activity feed for an issue",
	Long: `Show the activity timeline for an issue, including pipeline moves,
estimate changes, priority changes, PR connections, and other ZenHub events.

Use --github to also include GitHub timeline events (labels, assignments,
comments, close/reopen). This requires GitHub access to be configured.

Examples:
  zh issue activity task-tracker#1
  zh issue activity --repo=task-tracker 1
  zh issue activity task-tracker#1 --github`,
	Args: cobra.ExactArgs(1),
	RunE: runIssueActivity,
}

var (
	issueActivityRepo   string
	issueActivityGitHub bool
)

func init() {
	issueActivityCmd.Flags().StringVar(&issueActivityRepo, "repo", "", "Repository context for bare issue numbers")
	issueActivityCmd.Flags().BoolVar(&issueActivityGitHub, "github", false, "Include GitHub timeline events (requires GitHub access)")

	issueCmd.AddCommand(issueActivityCmd)
}

func resetIssueActivityFlags() {
	issueActivityRepo = ""
	issueActivityGitHub = false
}

// runIssueActivity implements `zh issue activity <issue>`.
func runIssueActivity(cmd *cobra.Command, args []string) error {
	cfg, err := requireWorkspace()
	if err != nil {
		return err
	}

	client := newClient(cfg, cmd)
	ghClient := newGitHubClient(cfg, cmd)
	w := cmd.OutOrStdout()

	parsed, parseErr := resolve.ParseIssueRef(args[0])

	var issueInfo struct {
		Number    int
		Title     string
		RepoName  string
		RepoOwner string
	}
	var zhEvents []activityEvent

	if parseErr == nil && parsed.ZenHubID != "" {
		info, events, err := fetchZenHubTimelineByNode(client, parsed.ZenHubID)
		if err != nil {
			return err
		}
		issueInfo = info
		zhEvents = events
	} else {
		resolved, err := resolve.Issue(client, cfg.Workspace, args[0], &resolve.IssueOptions{
			RepoFlag:     issueActivityRepo,
			GitHubClient: ghClient,
		})
		if err != nil {
			return err
		}

		info, events, err := fetchZenHubTimeline(client, resolved.RepoGhID, resolved.Number)
		if err != nil {
			return err
		}
		issueInfo = info
		zhEvents = events
	}

	allEvents := zhEvents

	// Merge GitHub timeline if requested
	if issueActivityGitHub {
		if ghClient == nil {
			fmt.Fprintln(cmd.ErrOrStderr(), output.Yellow("Warning: --github flag ignored — GitHub access not configured"))
		} else {
			ghResult, err := fetchGitHubTimeline(ghClient, issueInfo.RepoOwner, issueInfo.RepoName, issueInfo.Number)
			if err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "%s\n", output.Yellow("Warning: failed to fetch GitHub timeline: "+err.Error()))
			} else {
				allEvents = append(allEvents, ghResult.Events...)
			}
		}
	}

	// Sort by time
	sort.Slice(allEvents, func(i, j int) bool {
		return allEvents[i].Time.Before(allEvents[j].Time)
	})

	issueRef := fmt.Sprintf("%s#%d", issueInfo.RepoName, issueInfo.Number)

	if output.IsJSON(outputFormat) {
		return output.JSON(w, map[string]any{
			"issue":  map[string]any{"ref": issueRef, "title": issueInfo.Title, "number": issueInfo.Number},
			"events": allEvents,
		})
	}

	if len(allEvents) == 0 {
		fmt.Fprintf(w, "No activity found for %s.\n", issueRef)
		return nil
	}

	title := fmt.Sprintf("%s: %s", issueRef, issueInfo.Title)
	d := output.NewDetailWriter(w, "ACTIVITY", title)

	showSource := issueActivityGitHub && ghClient != nil

	d.Section("EVENTS")
	for _, ev := range allEvents {
		dateStr := output.Dim(output.FormatDate(ev.Time))
		actor := ""
		if ev.Actor != "" {
			actor = "@" + ev.Actor + " "
		}
		line := fmt.Sprintf("  %s  %s%s", dateStr, actor, ev.Description)
		if showSource {
			tag := output.Dim("[" + ev.Source + "]")
			line += "  " + tag
		}
		fmt.Fprintln(w, line)
	}

	fmt.Fprintf(w, "\nTotal: %d event(s)\n", len(allEvents))

	return nil
}

// fetchZenHubTimeline fetches ZenHub timeline items using repo GH ID and issue number.
func fetchZenHubTimeline(client *api.Client, repoGhID, issueNumber int) (struct {
	Number    int
	Title     string
	RepoName  string
	RepoOwner string
}, []activityEvent, error) {
	var info struct {
		Number    int
		Title     string
		RepoName  string
		RepoOwner string
	}

	var allItems []timelineItemNode
	var cursor *string
	pageSize := 50

	for {
		vars := map[string]any{
			"repositoryGhId": repoGhID,
			"issueNumber":    issueNumber,
			"first":          pageSize,
		}
		if cursor != nil {
			vars["after"] = *cursor
		}

		data, err := client.Execute(issueTimelineQuery, vars)
		if err != nil {
			return info, nil, exitcode.General("fetching issue timeline", err)
		}

		var resp struct {
			IssueByInfo *struct {
				ID         string `json:"id"`
				Number     int    `json:"number"`
				Title      string `json:"title"`
				Repository struct {
					Name  string `json:"name"`
					Owner struct {
						Login string `json:"login"`
					} `json:"owner"`
				} `json:"repository"`
				TimelineItems struct {
					TotalCount int `json:"totalCount"`
					PageInfo   struct {
						HasNextPage bool   `json:"hasNextPage"`
						EndCursor   string `json:"endCursor"`
					} `json:"pageInfo"`
					Nodes []timelineItemNode `json:"nodes"`
				} `json:"timelineItems"`
			} `json:"issueByInfo"`
		}
		if err := json.Unmarshal(data, &resp); err != nil {
			return info, nil, exitcode.General("parsing issue timeline", err)
		}
		if resp.IssueByInfo == nil {
			return info, nil, exitcode.NotFoundError(fmt.Sprintf("issue #%d not found", issueNumber))
		}

		info.Number = resp.IssueByInfo.Number
		info.Title = resp.IssueByInfo.Title
		info.RepoName = resp.IssueByInfo.Repository.Name
		info.RepoOwner = resp.IssueByInfo.Repository.Owner.Login

		allItems = append(allItems, resp.IssueByInfo.TimelineItems.Nodes...)

		if !resp.IssueByInfo.TimelineItems.PageInfo.HasNextPage {
			break
		}
		cursor = &resp.IssueByInfo.TimelineItems.PageInfo.EndCursor
	}

	events := parseZenHubTimelineItems(allItems)
	return info, events, nil
}
