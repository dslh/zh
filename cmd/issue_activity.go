package cmd

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/dslh/zh/internal/api"
	"github.com/dslh/zh/internal/exitcode"
	"github.com/dslh/zh/internal/gh"
	"github.com/dslh/zh/internal/output"
	"github.com/dslh/zh/internal/resolve"
	"github.com/spf13/cobra"
)

// ZenHub timeline item types

type timelineItemNode struct {
	ID        string          `json:"id"`
	Key       string          `json:"key"`
	Data      json.RawMessage `json:"data"`
	CreatedAt string          `json:"createdAt"`
}

// activityEvent is a unified event for display (ZenHub or GitHub).
type activityEvent struct {
	Time        time.Time `json:"time"`
	Source      string    `json:"source"` // "zenhub" or "github"
	Description string    `json:"description"`
	Actor       string    `json:"actor,omitempty"`
	Raw         any       `json:"raw,omitempty"` // original data for JSON output
}

// GraphQL queries

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

const issueTimelineByNodeQuery = `query GetIssueTimelineByNode($id: ID!, $first: Int!, $after: String) {
  node(id: $id) {
    ... on Issue {
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
  }
}`

const githubIssueTimelineQuery = `query GetGitHubTimeline($owner: String!, $repo: String!, $number: Int!, $first: Int!, $after: String) {
  repository(owner: $owner, name: $repo) {
    issueOrPullRequest(number: $number) {
      ... on Issue {
        timelineItems(first: $first, after: $after) {
          totalCount
          pageInfo {
            hasNextPage
            endCursor
          }
          nodes {
            __typename
            ... on LabeledEvent { createdAt actor { login } label { name } }
            ... on UnlabeledEvent { createdAt actor { login } label { name } }
            ... on AssignedEvent { createdAt actor { login } assignee { ... on User { login } } }
            ... on UnassignedEvent { createdAt actor { login } assignee { ... on User { login } } }
            ... on ClosedEvent { createdAt actor { login } }
            ... on ReopenedEvent { createdAt actor { login } }
            ... on CrossReferencedEvent { createdAt actor { login } source { ... on Issue { number title } ... on PullRequest { number title } } }
            ... on IssueComment { createdAt author { login } body }
            ... on RenamedTitleEvent { createdAt actor { login } previousTitle currentTitle }
            ... on MilestonedEvent { createdAt actor { login } milestoneTitle }
            ... on DemilestonedEvent { createdAt actor { login } milestoneTitle }
          }
        }
      }
      ... on PullRequest {
        timelineItems(first: $first, after: $after) {
          totalCount
          pageInfo {
            hasNextPage
            endCursor
          }
          nodes {
            __typename
            ... on LabeledEvent { createdAt actor { login } label { name } }
            ... on UnlabeledEvent { createdAt actor { login } label { name } }
            ... on AssignedEvent { createdAt actor { login } assignee { ... on User { login } } }
            ... on UnassignedEvent { createdAt actor { login } assignee { ... on User { login } } }
            ... on ClosedEvent { createdAt actor { login } }
            ... on ReopenedEvent { createdAt actor { login } }
            ... on CrossReferencedEvent { createdAt actor { login } source { ... on Issue { number title } ... on PullRequest { number title } } }
            ... on IssueComment { createdAt author { login } body }
            ... on RenamedTitleEvent { createdAt actor { login } previousTitle currentTitle }
            ... on MilestonedEvent { createdAt actor { login } milestoneTitle }
            ... on DemilestonedEvent { createdAt actor { login } milestoneTitle }
            ... on MergedEvent { createdAt actor { login } }
            ... on HeadRefDeletedEvent { createdAt actor { login } }
          }
        }
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
			ghEvents, err := fetchGitHubTimeline(ghClient, issueInfo.RepoOwner, issueInfo.RepoName, issueInfo.Number)
			if err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "%s\n", output.Yellow("Warning: failed to fetch GitHub timeline: "+err.Error()))
			} else {
				allEvents = append(allEvents, ghEvents...)
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

// fetchZenHubTimelineByNode fetches ZenHub timeline items using a ZenHub node ID.
func fetchZenHubTimelineByNode(client *api.Client, nodeID string) (struct {
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
			"id":    nodeID,
			"first": pageSize,
		}
		if cursor != nil {
			vars["after"] = *cursor
		}

		data, err := client.Execute(issueTimelineByNodeQuery, vars)
		if err != nil {
			return info, nil, exitcode.General("fetching issue timeline", err)
		}

		var resp struct {
			Node *struct {
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
			} `json:"node"`
		}
		if err := json.Unmarshal(data, &resp); err != nil {
			return info, nil, exitcode.General("parsing issue timeline", err)
		}
		if resp.Node == nil {
			return info, nil, exitcode.NotFoundError(fmt.Sprintf("issue %q not found", nodeID))
		}

		info.Number = resp.Node.Number
		info.Title = resp.Node.Title
		info.RepoName = resp.Node.Repository.Name
		info.RepoOwner = resp.Node.Repository.Owner.Login

		allItems = append(allItems, resp.Node.TimelineItems.Nodes...)

		if !resp.Node.TimelineItems.PageInfo.HasNextPage {
			break
		}
		cursor = &resp.Node.TimelineItems.PageInfo.EndCursor
	}

	events := parseZenHubTimelineItems(allItems)
	return info, events, nil
}

// parseZenHubTimelineItems converts raw timeline items to activity events.
func parseZenHubTimelineItems(items []timelineItemNode) []activityEvent {
	var events []activityEvent
	for _, item := range items {
		ev := parseZenHubTimelineItem(item)
		if ev != nil {
			events = append(events, *ev)
		}
	}
	return events
}

// parseZenHubTimelineItem converts a single timeline item to an activity event.
func parseZenHubTimelineItem(item timelineItemNode) *activityEvent {
	t, err := time.Parse(time.RFC3339, item.CreatedAt)
	if err != nil {
		return nil
	}

	var data map[string]any
	if item.Data != nil {
		_ = json.Unmarshal(item.Data, &data)
	}

	actor := extractActor(data)
	description := describeZenHubEvent(item.Key, data)
	if description == "" {
		// Unknown event type — show key as fallback
		description = item.Key
	}

	return &activityEvent{
		Time:        t,
		Source:      "ZenHub",
		Description: description,
		Actor:       actor,
		Raw: map[string]any{
			"key":  item.Key,
			"data": data,
		},
	}
}

// extractActor gets the actor login from timeline item data.
func extractActor(data map[string]any) string {
	if data == nil {
		return ""
	}
	if user, ok := data["github_user"].(map[string]any); ok {
		if login, ok := user["login"].(string); ok {
			return login
		}
	}
	return ""
}

// describeZenHubEvent returns a human-readable description of a ZenHub timeline event.
func describeZenHubEvent(key string, data map[string]any) string {
	switch key {
	case "issue.set_estimate":
		if v, ok := data["current_value"].(string); ok {
			return fmt.Sprintf("set estimate to %s", v)
		}
		if v, ok := data["previous_value"].(string); ok {
			return fmt.Sprintf("cleared estimate (was %s)", v)
		}
		return "changed estimate"

	case "issue.set_priority":
		if p, ok := data["priority"].(map[string]any); ok {
			if name, ok := p["name"].(string); ok {
				return fmt.Sprintf("set priority to %q", name)
			}
		}
		return "set priority"

	case "issue.remove_priority":
		if p, ok := data["previous_priority"].(map[string]any); ok {
			if name, ok := p["name"].(string); ok {
				return fmt.Sprintf("cleared priority (was %q)", name)
			}
		}
		return "cleared priority"

	case "issue.connect_issue_to_pr":
		if pr, ok := data["pull_request"].(map[string]any); ok {
			number := jsonInt(pr["number"])
			title := jsonString(pr["title"])
			repo := ""
			if prRepo, ok := data["pull_request_repository"].(map[string]any); ok {
				repo = jsonString(prRepo["name"])
			}
			if repo != "" && number > 0 {
				ref := fmt.Sprintf("%s#%d", repo, number)
				if title != "" {
					return fmt.Sprintf("connected PR %s %q", ref, truncateTitle(title))
				}
				return fmt.Sprintf("connected PR %s", ref)
			}
		}
		return "connected PR"

	case "issue.disconnect_issue_from_pr":
		if pr, ok := data["pull_request"].(map[string]any); ok {
			number := jsonInt(pr["number"])
			repo := ""
			if prRepo, ok := data["pull_request_repository"].(map[string]any); ok {
				repo = jsonString(prRepo["name"])
			}
			if repo != "" && number > 0 {
				return fmt.Sprintf("disconnected PR %s#%d", repo, number)
			}
		}
		return "disconnected PR"

	case "issue.transfer_pipeline":
		from := ""
		to := ""
		if fp, ok := data["from_pipeline"].(map[string]any); ok {
			from = jsonString(fp["name"])
		}
		if tp, ok := data["to_pipeline"].(map[string]any); ok {
			to = jsonString(tp["name"])
		}
		if from != "" && to != "" {
			return fmt.Sprintf("moved from %q to %q", from, to)
		}
		if to != "" {
			return fmt.Sprintf("moved to %q", to)
		}
		return "moved to another pipeline"

	case "issue.add_to_sprint":
		if s, ok := data["sprint"].(map[string]any); ok {
			if name, ok := s["name"].(string); ok {
				return fmt.Sprintf("added to sprint %q", name)
			}
		}
		return "added to sprint"

	case "issue.remove_from_sprint":
		if s, ok := data["sprint"].(map[string]any); ok {
			if name, ok := s["name"].(string); ok {
				return fmt.Sprintf("removed from sprint %q", name)
			}
		}
		return "removed from sprint"

	case "issue.add_to_epic":
		if e, ok := data["epic"].(map[string]any); ok {
			if title, ok := e["title"].(string); ok {
				return fmt.Sprintf("added to epic %q", truncateTitle(title))
			}
		}
		return "added to epic"

	case "issue.remove_from_epic":
		if e, ok := data["epic"].(map[string]any); ok {
			if title, ok := e["title"].(string); ok {
				return fmt.Sprintf("removed from epic %q", truncateTitle(title))
			}
		}
		return "removed from epic"

	default:
		// Return the key as-is for unrecognized events
		return formatEventKey(key)
	}
}

// formatEventKey converts a dot-notation key like "issue.set_estimate" to a readable form.
func formatEventKey(key string) string {
	// Strip "issue." prefix
	s := strings.TrimPrefix(key, "issue.")
	// Replace underscores with spaces
	s = strings.ReplaceAll(s, "_", " ")
	return s
}

// jsonString safely extracts a string from an any value.
func jsonString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

// jsonInt safely extracts an int from an any value (JSON numbers are float64).
func jsonInt(v any) int {
	if f, ok := v.(float64); ok {
		return int(f)
	}
	return 0
}

// fetchGitHubTimeline fetches timeline events from the GitHub API.
func fetchGitHubTimeline(ghClient *gh.Client, owner, repo string, number int) ([]activityEvent, error) {
	var allEvents []activityEvent
	var cursor *string
	pageSize := 100

	for {
		vars := map[string]any{
			"owner":  owner,
			"repo":   repo,
			"number": number,
			"first":  pageSize,
		}
		if cursor != nil {
			vars["after"] = *cursor
		}

		data, err := ghClient.Execute(githubIssueTimelineQuery, vars)
		if err != nil {
			return nil, err
		}

		var resp struct {
			Repository *struct {
				IssueOrPullRequest json.RawMessage `json:"issueOrPullRequest"`
			} `json:"repository"`
		}
		if err := json.Unmarshal(data, &resp); err != nil {
			return nil, fmt.Errorf("parsing GitHub timeline: %w", err)
		}
		if resp.Repository == nil {
			return nil, fmt.Errorf("repository not found")
		}

		var timeline struct {
			TimelineItems struct {
				TotalCount int `json:"totalCount"`
				PageInfo   struct {
					HasNextPage bool   `json:"hasNextPage"`
					EndCursor   string `json:"endCursor"`
				} `json:"pageInfo"`
				Nodes []json.RawMessage `json:"nodes"`
			} `json:"timelineItems"`
		}
		if err := json.Unmarshal(resp.Repository.IssueOrPullRequest, &timeline); err != nil {
			return nil, fmt.Errorf("parsing GitHub timeline items: %w", err)
		}

		for _, raw := range timeline.TimelineItems.Nodes {
			ev := parseGitHubTimelineItem(raw)
			if ev != nil {
				allEvents = append(allEvents, *ev)
			}
		}

		if !timeline.TimelineItems.PageInfo.HasNextPage {
			break
		}
		cursor = &timeline.TimelineItems.PageInfo.EndCursor
	}

	return allEvents, nil
}

// parseGitHubTimelineItem converts a GitHub timeline node to an activity event.
func parseGitHubTimelineItem(raw json.RawMessage) *activityEvent {
	var base struct {
		TypeName  string `json:"__typename"`
		CreatedAt string `json:"createdAt"`
		Actor     *struct {
			Login string `json:"login"`
		} `json:"actor"`
		Author *struct {
			Login string `json:"login"`
		} `json:"author"`
	}
	if err := json.Unmarshal(raw, &base); err != nil {
		return nil
	}
	if base.TypeName == "" {
		return nil
	}

	t, err := time.Parse(time.RFC3339, base.CreatedAt)
	if err != nil {
		return nil
	}

	actor := ""
	if base.Actor != nil {
		actor = base.Actor.Login
	} else if base.Author != nil {
		actor = base.Author.Login
	}

	description := describeGitHubEvent(base.TypeName, raw)
	if description == "" {
		return nil
	}

	return &activityEvent{
		Time:        t,
		Source:      "GitHub",
		Description: description,
		Actor:       actor,
	}
}

// describeGitHubEvent returns a human-readable description of a GitHub timeline event.
func describeGitHubEvent(typename string, raw json.RawMessage) string {
	switch typename {
	case "LabeledEvent":
		var ev struct {
			Label struct{ Name string } `json:"label"`
		}
		_ = json.Unmarshal(raw, &ev)
		return fmt.Sprintf("added label %q", ev.Label.Name)

	case "UnlabeledEvent":
		var ev struct {
			Label struct{ Name string } `json:"label"`
		}
		_ = json.Unmarshal(raw, &ev)
		return fmt.Sprintf("removed label %q", ev.Label.Name)

	case "AssignedEvent":
		var ev struct {
			Assignee struct {
				Login string `json:"login"`
			} `json:"assignee"`
		}
		_ = json.Unmarshal(raw, &ev)
		if ev.Assignee.Login != "" {
			return fmt.Sprintf("assigned @%s", ev.Assignee.Login)
		}
		return "assigned someone"

	case "UnassignedEvent":
		var ev struct {
			Assignee struct {
				Login string `json:"login"`
			} `json:"assignee"`
		}
		_ = json.Unmarshal(raw, &ev)
		if ev.Assignee.Login != "" {
			return fmt.Sprintf("unassigned @%s", ev.Assignee.Login)
		}
		return "unassigned someone"

	case "ClosedEvent":
		return "closed this issue"

	case "ReopenedEvent":
		return "reopened this issue"

	case "CrossReferencedEvent":
		var ev struct {
			Source struct {
				Number int    `json:"number"`
				Title  string `json:"title"`
			} `json:"source"`
		}
		_ = json.Unmarshal(raw, &ev)
		if ev.Source.Number > 0 {
			title := truncateTitle(ev.Source.Title)
			return fmt.Sprintf("referenced from #%d %q", ev.Source.Number, title)
		}
		return "cross-referenced"

	case "IssueComment":
		var ev struct {
			Body string `json:"body"`
		}
		_ = json.Unmarshal(raw, &ev)
		body := ev.Body
		if len(body) > 60 {
			body = body[:57] + "..."
		}
		body = strings.ReplaceAll(body, "\n", " ")
		return fmt.Sprintf("commented: %s", body)

	case "RenamedTitleEvent":
		var ev struct {
			PreviousTitle string `json:"previousTitle"`
			CurrentTitle  string `json:"currentTitle"`
		}
		_ = json.Unmarshal(raw, &ev)
		return fmt.Sprintf("renamed from %q to %q", truncateTitle(ev.PreviousTitle), truncateTitle(ev.CurrentTitle))

	case "MilestonedEvent":
		var ev struct {
			MilestoneTitle string `json:"milestoneTitle"`
		}
		_ = json.Unmarshal(raw, &ev)
		return fmt.Sprintf("added to milestone %q", ev.MilestoneTitle)

	case "DemilestonedEvent":
		var ev struct {
			MilestoneTitle string `json:"milestoneTitle"`
		}
		_ = json.Unmarshal(raw, &ev)
		return fmt.Sprintf("removed from milestone %q", ev.MilestoneTitle)

	case "MergedEvent":
		return "merged this pull request"

	case "HeadRefDeletedEvent":
		return "deleted the branch"

	default:
		return ""
	}
}
