package cmd

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"

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
  zh issue activity task-tracker#1 --github
  zh issue activity task-tracker#1 --from=7d
  zh issue activity task-tracker#1 --from=2026-01-01 --to=2026-02-01
  zh issue activity task-tracker#1 --prs`,
	Args: cobra.ExactArgs(1),
	RunE: runIssueActivity,
}

var (
	issueActivityRepo   string
	issueActivityGitHub bool
	issueActivityFrom   string
	issueActivityTo     string
	issueActivityPRs    bool
)

func init() {
	issueActivityCmd.Flags().StringVar(&issueActivityRepo, "repo", "", "Repository context for bare issue numbers")
	issueActivityCmd.Flags().BoolVar(&issueActivityGitHub, "github", false, "Include GitHub timeline events (requires GitHub access)")
	issueActivityCmd.Flags().StringVar(&issueActivityFrom, "from", "", "Start of time range (e.g. 1d, 7d, 2h, yesterday, 2026-02-01)")
	issueActivityCmd.Flags().StringVar(&issueActivityTo, "to", "", "End of time range (default: now)")
	issueActivityCmd.Flags().BoolVar(&issueActivityPRs, "prs", false, "Include activity for connected pull requests")

	issueCmd.AddCommand(issueActivityCmd)
}

func resetIssueActivityFlags() {
	issueActivityRepo = ""
	issueActivityGitHub = false
	issueActivityFrom = ""
	issueActivityTo = ""
	issueActivityPRs = false
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
		ID        string
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
		issueInfo.ID = parsed.ZenHubID
		issueInfo.Number = info.Number
		issueInfo.Title = info.Title
		issueInfo.RepoName = info.RepoName
		issueInfo.RepoOwner = info.RepoOwner
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
		issueInfo.ID = info.ID
		issueInfo.Number = info.Number
		issueInfo.Title = info.Title
		issueInfo.RepoName = info.RepoName
		issueInfo.RepoOwner = info.RepoOwner
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

	// Filter by time range if --from or --to is set
	if issueActivityFrom != "" || issueActivityTo != "" {
		now := time.Now()
		if issueActivityFrom != "" {
			fromTime, err := parseTimeFlag(issueActivityFrom, now)
			if err != nil {
				return exitcode.Usage(fmt.Sprintf("invalid --from value: %v", err))
			}
			filtered := allEvents[:0]
			for _, ev := range allEvents {
				if !ev.Time.Before(fromTime) {
					filtered = append(filtered, ev)
				}
			}
			allEvents = filtered
		}
		if issueActivityTo != "" {
			toTime, err := parseTimeFlag(issueActivityTo, now)
			if err != nil {
				return exitcode.Usage(fmt.Sprintf("invalid --to value: %v", err))
			}
			filtered := allEvents[:0]
			for _, ev := range allEvents {
				if !ev.Time.After(toTime) {
					filtered = append(filtered, ev)
				}
			}
			allEvents = filtered
		}
	}

	issueRef := fmt.Sprintf("%s#%d", issueInfo.RepoName, issueInfo.Number)

	// Fetch connected PRs if --prs is set
	type connectedPR struct {
		Ref        string          `json:"ref"`
		Title      string          `json:"title"`
		HeadBranch string          `json:"headBranch,omitempty"`
		Events     []activityEvent `json:"events"`
	}
	var connectedPRs []connectedPR

	if issueActivityPRs && issueInfo.ID != "" {
		prRefs := fetchIssueConnectedPRs(client, issueInfo.ID)
		for _, pr := range prRefs {
			repo, err := resolve.LookupRepoWithRefresh(client, cfg.Workspace, pr.RepoName)
			if err != nil {
				continue
			}
			_, prEvents, err := fetchZenHubTimeline(client, repo.GhID, pr.Number)
			if err != nil {
				continue
			}

			var headBranch string
			if issueActivityGitHub && ghClient != nil {
				ghResult, err := fetchGitHubTimeline(ghClient, pr.RepoOwner, pr.RepoName, pr.Number)
				if err == nil {
					prEvents = append(prEvents, ghResult.Events...)
					headBranch = ghResult.HeadBranch
				}
			}

			sort.Slice(prEvents, func(i, j int) bool {
				return prEvents[i].Time.Before(prEvents[j].Time)
			})

			// Apply same time filtering
			if issueActivityFrom != "" || issueActivityTo != "" {
				now := time.Now()
				if issueActivityFrom != "" {
					fromTime, _ := parseTimeFlag(issueActivityFrom, now)
					filtered := prEvents[:0]
					for _, ev := range prEvents {
						if !ev.Time.Before(fromTime) {
							filtered = append(filtered, ev)
						}
					}
					prEvents = filtered
				}
				if issueActivityTo != "" {
					toTime, _ := parseTimeFlag(issueActivityTo, now)
					filtered := prEvents[:0]
					for _, ev := range prEvents {
						if !ev.Time.After(toTime) {
							filtered = append(filtered, ev)
						}
					}
					prEvents = filtered
				}
			}

			connectedPRs = append(connectedPRs, connectedPR{
				Ref:        pr.Ref,
				Title:      pr.Title,
				HeadBranch: headBranch,
				Events:     prEvents,
			})
		}
	}

	if output.IsJSON(outputFormat) {
		result := map[string]any{
			"issue":  map[string]any{"ref": issueRef, "title": issueInfo.Title, "number": issueInfo.Number},
			"events": allEvents,
		}
		if issueActivityPRs {
			result["connectedPRs"] = connectedPRs
		}
		return output.JSON(w, result)
	}

	if len(allEvents) == 0 && len(connectedPRs) == 0 {
		fmt.Fprintf(w, "No activity found for %s.\n", issueRef)
		return nil
	}

	title := fmt.Sprintf("%s: %s", issueRef, issueInfo.Title)
	d := output.NewDetailWriter(w, "ACTIVITY", title)

	showSource := issueActivityGitHub && ghClient != nil

	d.Section("EVENTS")
	if len(allEvents) == 0 {
		fmt.Fprintln(w, output.Dim("  (no events in time range)"))
	}
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

	totalEvents := len(allEvents)

	// Render connected PRs
	for _, pr := range connectedPRs {
		prHeader := fmt.Sprintf("  %s %s: %s", output.Dim("└─"), output.Cyan(pr.Ref), pr.Title)
		if pr.HeadBranch != "" {
			prHeader += " " + output.Dim("("+pr.HeadBranch+")")
		}
		fmt.Fprintln(w, prHeader)
		if len(pr.Events) == 0 {
			fmt.Fprintln(w, output.Dim("     (no events in time range)"))
		} else {
			for _, ev := range pr.Events {
				dateStr := output.Dim(output.FormatDate(ev.Time))
				actor := ""
				if ev.Actor != "" {
					actor = "@" + ev.Actor + " "
				}
				line := fmt.Sprintf("     %s  %s%s", dateStr, actor, ev.Description)
				if showSource {
					line += "  " + output.Dim("["+ev.Source+"]")
				}
				fmt.Fprintln(w, line)
				totalEvents++
			}
		}
	}

	summary := fmt.Sprintf("\nTotal: %d event(s)", totalEvents)
	if len(connectedPRs) > 0 {
		summary += fmt.Sprintf(", %d connected PR(s)", len(connectedPRs))
	}
	fmt.Fprintln(w, summary)

	return nil
}

// fetchZenHubTimeline fetches ZenHub timeline items using repo GH ID and issue number.
func fetchZenHubTimeline(client *api.Client, repoGhID, issueNumber int) (struct {
	ID        string
	Number    int
	Title     string
	RepoName  string
	RepoOwner string
}, []activityEvent, error) {
	var info struct {
		ID        string
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

		info.ID = resp.IssueByInfo.ID
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
