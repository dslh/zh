package cmd

// timeline.go contains shared timeline types, GraphQL queries, and parsing functions
// used by both `zh issue activity` and `zh activity`.

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/dslh/zh/internal/api"
	"github.com/dslh/zh/internal/exitcode"
	"github.com/dslh/zh/internal/gh"
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
        __typename
        createdAt
        author { login }
        userContentEdits(first: 1) { nodes { createdAt editor { login } } }
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
            ... on IssueTypeAddedEvent { createdAt actor { login } issueType { name } }
            ... on IssueTypeChangedEvent { createdAt actor { login } issueType { name } prevIssueType { name } }
            ... on IssueTypeRemovedEvent { createdAt actor { login } issueType { name } }
            ... on ParentIssueAddedEvent { createdAt actor { login } parent { number title repository { name } } }
            ... on ParentIssueRemovedEvent { createdAt actor { login } parent { number title repository { name } } }
            ... on SubIssueAddedEvent { createdAt actor { login } subIssue { number title repository { name } } }
            ... on SubIssueRemovedEvent { createdAt actor { login } subIssue { number title repository { name } } }
          }
        }
      }
      ... on PullRequest {
        __typename
        createdAt
        author { login }
        userContentEdits(first: 1) { nodes { createdAt editor { login } } }
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
            ... on PullRequestCommit { commit { committedDate message author { user { login } } } }
            ... on PullRequestReview { createdAt author { login } state }
            ... on ReviewRequestedEvent { createdAt actor { login } requestedReviewer { ... on User { login } } }
            ... on HeadRefForcePushedEvent { createdAt actor { login } }
            ... on ReadyForReviewEvent { createdAt actor { login } }
            ... on ConvertToDraftEvent { createdAt actor { login } }
          }
        }
      }
    }
  }
}`

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
	t = t.Local()

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

	case "issue.connect_pr_to_issue":
		if issue, ok := data["issue"].(map[string]any); ok {
			number := jsonInt(issue["number"])
			title := jsonString(issue["title"])
			repo := ""
			if issueRepo, ok := data["issue_repository"].(map[string]any); ok {
				repo = jsonString(issueRepo["name"])
			}
			if repo != "" && number > 0 {
				ref := fmt.Sprintf("%s#%d", repo, number)
				if title != "" {
					return fmt.Sprintf("connected to issue %s %q", ref, truncateTitle(title))
				}
				return fmt.Sprintf("connected to issue %s", ref)
			}
		}
		return "connected to issue"

	case "issue.disconnect_pr_from_issue":
		if issue, ok := data["issue"].(map[string]any); ok {
			number := jsonInt(issue["number"])
			repo := ""
			if issueRepo, ok := data["issue_repository"].(map[string]any); ok {
				repo = jsonString(issueRepo["name"])
			}
			if repo != "" && number > 0 {
				return fmt.Sprintf("disconnected from issue %s#%d", repo, number)
			}
		}
		return "disconnected from issue"

	case "issue.change_pipeline", "issue.transfer_pipeline":
		from := ""
		to := ""
		if fp, ok := data["from_pipeline"].(map[string]any); ok {
			from = jsonString(fp["name"])
		}
		if tp, ok := data["to_pipeline"].(map[string]any); ok {
			to = jsonString(tp["name"])
		}
		var desc string
		if from != "" && to != "" {
			desc = fmt.Sprintf("moved from %q to %q", from, to)
		} else if to != "" {
			desc = fmt.Sprintf("moved to %q", to)
		} else {
			desc = "moved to another pipeline"
		}
		// Check for "via PR" context
		if pr, ok := data["pull_request"].(map[string]any); ok {
			number := jsonInt(pr["number"])
			repo := ""
			if prRepo, ok := data["pull_request_repository"].(map[string]any); ok {
				repo = jsonString(prRepo["name"])
			}
			if repo != "" && number > 0 {
				desc += fmt.Sprintf(" via PR %s#%d", repo, number)
			}
		}
		return desc

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

	case "issue.add_blocking_issue":
		if bi, ok := data["blocking_issue"].(map[string]any); ok {
			number := jsonInt(bi["number"])
			title := jsonString(bi["title"])
			repo := ""
			if biRepo, ok := data["blocking_issue_repository"].(map[string]any); ok {
				repo = jsonString(biRepo["name"])
			}
			if repo != "" && number > 0 {
				ref := fmt.Sprintf("%s#%d", repo, number)
				if title != "" {
					return fmt.Sprintf("added blocking issue %s %q", ref, truncateTitle(title))
				}
				return fmt.Sprintf("added blocking issue %s", ref)
			}
		}
		return "added blocking issue"

	case "issue.remove_blocking_issue":
		if bi, ok := data["blocking_issue"].(map[string]any); ok {
			number := jsonInt(bi["number"])
			repo := ""
			if biRepo, ok := data["blocking_issue_repository"].(map[string]any); ok {
				repo = jsonString(biRepo["name"])
			}
			if repo != "" && number > 0 {
				return fmt.Sprintf("removed blocking issue %s#%d", repo, number)
			}
		}
		return "removed blocking issue"

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

// ghTimelineResult holds the results from a GitHub timeline fetch.
type ghTimelineResult struct {
	Events    []activityEvent
	IsPR      bool
	CreatedAt time.Time
	CreatedBy string
}

// fetchGitHubTimeline fetches timeline events from the GitHub API.
func fetchGitHubTimeline(ghClient *gh.Client, owner, repo string, number int) (*ghTimelineResult, error) {
	result := &ghTimelineResult{}
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
			TypeName  string `json:"__typename"`
			CreatedAt string `json:"createdAt"`
			Author    *struct {
				Login string `json:"login"`
			} `json:"author"`
			UserContentEdits *struct {
				Nodes []struct {
					CreatedAt string `json:"createdAt"`
					Editor    *struct {
						Login string `json:"login"`
					} `json:"editor"`
				} `json:"nodes"`
			} `json:"userContentEdits"`
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

		// On first page, extract metadata and description edits
		if cursor == nil {
			result.IsPR = timeline.TypeName == "PullRequest"
			if t, err := time.Parse(time.RFC3339, timeline.CreatedAt); err == nil {
				result.CreatedAt = t.Local()
			}
			if timeline.Author != nil {
				result.CreatedBy = timeline.Author.Login
			}

			if timeline.UserContentEdits != nil {
				for _, edit := range timeline.UserContentEdits.Nodes {
					t, err := time.Parse(time.RFC3339, edit.CreatedAt)
					if err != nil {
						continue
					}
					t = t.Local()
					actor := ""
					if edit.Editor != nil {
						actor = edit.Editor.Login
					}
					result.Events = append(result.Events, activityEvent{
						Time:        t,
						Source:      "GitHub",
						Description: "edited description",
						Actor:       actor,
					})
				}
			}
		}

		for _, raw := range timeline.TimelineItems.Nodes {
			ev := parseGitHubTimelineItem(raw)
			if ev != nil {
				result.Events = append(result.Events, *ev)
			}
		}

		if !timeline.TimelineItems.PageInfo.HasNextPage {
			break
		}
		cursor = &timeline.TimelineItems.PageInfo.EndCursor
	}

	return result, nil
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
		// PullRequestCommit has no createdAt; time is nested under commit
		Commit *struct {
			CommittedDate string `json:"committedDate"`
			Author        *struct {
				User *struct {
					Login string `json:"login"`
				} `json:"user"`
			} `json:"author"`
		} `json:"commit"`
	}
	if err := json.Unmarshal(raw, &base); err != nil {
		return nil
	}
	if base.TypeName == "" {
		return nil
	}

	// PullRequestCommit uses commit.committedDate instead of createdAt
	timeStr := base.CreatedAt
	if base.TypeName == "PullRequestCommit" && base.Commit != nil {
		timeStr = base.Commit.CommittedDate
	}

	t, err := time.Parse(time.RFC3339, timeStr)
	if err != nil {
		return nil
	}
	t = t.Local()

	actor := ""
	if base.Actor != nil {
		actor = base.Actor.Login
	} else if base.Author != nil {
		actor = base.Author.Login
	}
	// PullRequestCommit actor is nested under commit.author.user
	if actor == "" && base.Commit != nil && base.Commit.Author != nil && base.Commit.Author.User != nil {
		actor = base.Commit.Author.User.Login
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

	case "PullRequestCommit":
		var ev struct {
			Commit struct {
				Message string `json:"message"`
			} `json:"commit"`
		}
		_ = json.Unmarshal(raw, &ev)
		msg := ev.Commit.Message
		// Use first line only
		if idx := strings.IndexByte(msg, '\n'); idx >= 0 {
			msg = msg[:idx]
		}
		if len(msg) > 60 {
			msg = msg[:57] + "..."
		}
		if msg != "" {
			return fmt.Sprintf("pushed commit: %s", msg)
		}
		return "pushed a commit"

	case "PullRequestReview":
		var ev struct {
			State string `json:"state"`
		}
		_ = json.Unmarshal(raw, &ev)
		switch strings.ToUpper(ev.State) {
		case "APPROVED":
			return "approved this pull request"
		case "CHANGES_REQUESTED":
			return "requested changes"
		case "COMMENTED":
			return "reviewed (commented)"
		case "DISMISSED":
			return "review dismissed"
		default:
			return "reviewed this pull request"
		}

	case "ReviewRequestedEvent":
		var ev struct {
			RequestedReviewer struct {
				Login string `json:"login"`
			} `json:"requestedReviewer"`
		}
		_ = json.Unmarshal(raw, &ev)
		if ev.RequestedReviewer.Login != "" {
			return fmt.Sprintf("requested review from @%s", ev.RequestedReviewer.Login)
		}
		return "requested a review"

	case "HeadRefForcePushedEvent":
		return "force-pushed the branch"

	case "ReadyForReviewEvent":
		return "marked as ready for review"

	case "ConvertToDraftEvent":
		return "converted to draft"

	case "IssueTypeAddedEvent":
		var ev struct {
			IssueType *struct {
				Name string `json:"name"`
			} `json:"issueType"`
		}
		_ = json.Unmarshal(raw, &ev)
		if ev.IssueType != nil && ev.IssueType.Name != "" {
			return fmt.Sprintf("set issue type to %q", ev.IssueType.Name)
		}
		return "set issue type"

	case "IssueTypeChangedEvent":
		var ev struct {
			IssueType *struct {
				Name string `json:"name"`
			} `json:"issueType"`
			PrevIssueType *struct {
				Name string `json:"name"`
			} `json:"prevIssueType"`
		}
		_ = json.Unmarshal(raw, &ev)
		from := ""
		to := ""
		if ev.PrevIssueType != nil {
			from = ev.PrevIssueType.Name
		}
		if ev.IssueType != nil {
			to = ev.IssueType.Name
		}
		if from != "" && to != "" {
			return fmt.Sprintf("changed issue type from %q to %q", from, to)
		}
		return "changed issue type"

	case "IssueTypeRemovedEvent":
		var ev struct {
			IssueType *struct {
				Name string `json:"name"`
			} `json:"issueType"`
		}
		_ = json.Unmarshal(raw, &ev)
		if ev.IssueType != nil && ev.IssueType.Name != "" {
			return fmt.Sprintf("removed issue type %q", ev.IssueType.Name)
		}
		return "removed issue type"

	case "ParentIssueAddedEvent":
		var ev struct {
			Parent *struct {
				Number int    `json:"number"`
				Title  string `json:"title"`
			} `json:"parent"`
		}
		_ = json.Unmarshal(raw, &ev)
		if ev.Parent != nil && ev.Parent.Number > 0 {
			return fmt.Sprintf("added parent issue #%d %q", ev.Parent.Number, truncateTitle(ev.Parent.Title))
		}
		return "added parent issue"

	case "ParentIssueRemovedEvent":
		var ev struct {
			Parent *struct {
				Number int    `json:"number"`
				Title  string `json:"title"`
			} `json:"parent"`
		}
		_ = json.Unmarshal(raw, &ev)
		if ev.Parent != nil && ev.Parent.Number > 0 {
			return fmt.Sprintf("removed parent issue #%d %q", ev.Parent.Number, truncateTitle(ev.Parent.Title))
		}
		return "removed parent issue"

	case "SubIssueAddedEvent":
		var ev struct {
			SubIssue *struct {
				Number int    `json:"number"`
				Title  string `json:"title"`
			} `json:"subIssue"`
		}
		_ = json.Unmarshal(raw, &ev)
		if ev.SubIssue != nil && ev.SubIssue.Number > 0 {
			return fmt.Sprintf("added sub-issue #%d %q", ev.SubIssue.Number, truncateTitle(ev.SubIssue.Title))
		}
		return "added sub-issue"

	case "SubIssueRemovedEvent":
		var ev struct {
			SubIssue *struct {
				Number int    `json:"number"`
				Title  string `json:"title"`
			} `json:"subIssue"`
		}
		_ = json.Unmarshal(raw, &ev)
		if ev.SubIssue != nil && ev.SubIssue.Number > 0 {
			return fmt.Sprintf("removed sub-issue #%d %q", ev.SubIssue.Number, truncateTitle(ev.SubIssue.Title))
		}
		return "removed sub-issue"

	default:
		return ""
	}
}
