package cmd

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/dslh/zh/internal/api"
	"github.com/dslh/zh/internal/cache"
	"github.com/dslh/zh/internal/exitcode"
	"github.com/dslh/zh/internal/gh"
	"github.com/dslh/zh/internal/output"
	"github.com/dslh/zh/internal/resolve"
	"github.com/spf13/cobra"
)

// activityIssue holds an issue found during the activity scan.
type activityIssue struct {
	ID             string          `json:"id"`
	Number         int             `json:"number"`
	Title          string          `json:"title"`
	Ref            string          `json:"ref"`
	Pipeline       string          `json:"pipeline"`
	UpdatedAt      time.Time       `json:"updatedAt"`
	GhUpdatedAt    time.Time       `json:"ghUpdatedAt,omitempty"`
	Assignees      []string        `json:"assignees,omitempty"`
	RepoName       string          `json:"repoName"`
	RepoOwner      string          `json:"repoOwner"`
	IsPR           bool            `json:"isPR,omitempty"`
	ConnectedIssue string          `json:"connectedIssue,omitempty"`
	Events         []activityEvent `json:"events,omitempty"`
}

// GraphQL query for scanning pipeline issues ordered by updated_at

const activitySearchQuery = `query ActivitySearch(
  $pipelineId: ID!
  $workspaceId: ID!
  $first: Int!
  $after: String
) {
  searchIssuesByPipeline(
    pipelineId: $pipelineId
    filters: {}
    order: { field: updated_at, direction: DESC }
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
      updatedAt
      ghUpdatedAt
      repository {
        name
        ownerName
      }
      assignees(first: 5) {
        nodes { login }
      }
      pipelineIssue(workspaceId: $workspaceId) {
        pipeline { name }
      }
    }
  }
}`

// activityClosedQuery fetches recently closed issues.
const activityClosedQuery = `query ActivityClosed(
  $workspaceId: ID!
  $first: Int!
) {
  searchClosedIssues(
    workspaceId: $workspaceId
    filters: {}
    first: $first
  ) {
    nodes {
      id
      number
      title
      state
      updatedAt
      ghUpdatedAt
      repository {
        name
        ownerName
      }
      assignees(first: 5) {
        nodes { login }
      }
    }
  }
}`

// GitHub search query for finding recently updated issues/PRs
const activityGitHubSearchQuery = `query ActivityGitHubSearch($query: String!, $first: Int!, $after: String) {
  search(query: $query, type: ISSUE, first: $first, after: $after) {
    issueCount
    pageInfo {
      hasNextPage
      endCursor
    }
    nodes {
      ... on Issue {
        number
        title
        updatedAt
        repository { name owner { login } }
      }
      ... on PullRequest {
        number
        title
        updatedAt
        repository { name owner { login } }
      }
    }
  }
}`

// Commands

var activityCmd = &cobra.Command{
	Use:   "activity",
	Short: "Show recent workspace activity",
	Long: `Show recently updated issues across the workspace.

Scans all pipelines for issues updated within the time range, using
ZenHub's updated_at ordering for efficient early termination.

Use --github to also discover issues with GitHub-only activity
(comments, reviews, merges) that may not appear in ZenHub's
updated_at ordering.

Use --detail to fetch per-issue event timelines showing exactly
what changed (pipeline moves, estimate changes, comments, etc.).

Examples:
  zh activity                          # last 24 hours
  zh activity --from=7d               # last 7 days
  zh activity --from=yesterday        # since yesterday
  zh activity --from=2026-02-01       # since a specific date
  zh activity --github                # include GitHub activity
  zh activity --detail                # show per-issue events
  zh activity --pipeline="In Progress" # filter to one pipeline`,
	RunE: runActivity,
}

var (
	activityFrom     string
	activityTo       string
	activityGitHub   bool
	activityDetail   bool
	activityPipeline string
	activityRepo     string
)

func init() {
	activityCmd.Flags().StringVar(&activityFrom, "from", "1d", "Start of time range (e.g. 1d, 7d, 2h, yesterday, 2026-02-01)")
	activityCmd.Flags().StringVar(&activityTo, "to", "", "End of time range (default: now)")
	activityCmd.Flags().BoolVar(&activityGitHub, "github", false, "Also fetch GitHub activity (requires GitHub access)")
	activityCmd.Flags().BoolVar(&activityDetail, "detail", false, "Fetch per-issue event timelines (slower)")
	activityCmd.Flags().StringVar(&activityPipeline, "pipeline", "", "Filter to a specific pipeline")
	activityCmd.Flags().StringVar(&activityRepo, "repo", "", "Filter to a specific repository")

	rootCmd.AddCommand(activityCmd)
}

func resetActivityFlags() {
	activityFrom = "1d"
	activityTo = ""
	activityGitHub = false
	activityDetail = false
	activityPipeline = ""
	activityRepo = ""
}

// parseTimeFlag parses a time flag value into a time.Time.
// Supports: relative durations (1d, 7d, 2h, 30m), keywords (yesterday, last week),
// ISO dates (2026-02-01), and RFC3339 timestamps.
func parseTimeFlag(value string, now time.Time) (time.Time, error) {
	if value == "" {
		return now, nil
	}

	lower := strings.ToLower(strings.TrimSpace(value))

	// Keywords
	switch lower {
	case "now":
		return now, nil
	case "yesterday":
		y := now.AddDate(0, 0, -1)
		return time.Date(y.Year(), y.Month(), y.Day(), 0, 0, 0, 0, now.Location()), nil
	case "last week":
		w := now.AddDate(0, 0, -7)
		return time.Date(w.Year(), w.Month(), w.Day(), 0, 0, 0, 0, now.Location()), nil
	}

	// Relative durations: 1d, 7d, 2h, 30m
	if len(lower) >= 2 {
		suffix := lower[len(lower)-1]
		numStr := lower[:len(lower)-1]
		var n int
		if _, err := fmt.Sscanf(numStr, "%d", &n); err == nil && n > 0 {
			switch suffix {
			case 'd':
				return now.AddDate(0, 0, -n), nil
			case 'h':
				return now.Add(-time.Duration(n) * time.Hour), nil
			case 'm':
				return now.Add(-time.Duration(n) * time.Minute), nil
			case 'w':
				return now.AddDate(0, 0, -n*7), nil
			}
		}
	}

	// RFC3339
	if t, err := time.Parse(time.RFC3339, value); err == nil {
		return t, nil
	}

	// ISO date (YYYY-MM-DD)
	if t, err := time.Parse("2006-01-02", value); err == nil {
		return t, nil
	}

	// ISO date + time without timezone
	if t, err := time.Parse("2006-01-02T15:04:05", value); err == nil {
		return t, nil
	}

	return time.Time{}, fmt.Errorf("unrecognized time format: %q (try 1d, 7d, 2h, yesterday, 2026-02-01, or RFC3339)", value)
}

// runActivity implements `zh activity`.
func runActivity(cmd *cobra.Command, args []string) error {
	cfg, err := requireWorkspace()
	if err != nil {
		return err
	}

	client := newClient(cfg, cmd)
	ghClient := newGitHubClient(cfg, cmd)

	now := time.Now()
	fromTime, err := parseTimeFlag(activityFrom, now)
	if err != nil {
		return exitcode.Usage(fmt.Sprintf("invalid --from value: %v", err))
	}
	toTime, err := parseTimeFlag(activityTo, now)
	if err != nil {
		return exitcode.Usage(fmt.Sprintf("invalid --to value: %v", err))
	}

	// Resolve repo filter
	var repoFilter string
	if activityRepo != "" {
		repo, err := resolve.LookupRepoWithRefresh(client, cfg.Workspace, activityRepo)
		if err != nil {
			return err
		}
		repoFilter = repo.Name
	}

	// Step 1: Scan pipelines for recently updated issues
	var pipelineIDs []struct{ ID, Name string }

	if activityPipeline != "" {
		resolved, err := resolve.Pipeline(client, cfg.Workspace, activityPipeline, cfg.Aliases.Pipelines)
		if err != nil {
			return err
		}
		pipelineIDs = []struct{ ID, Name string }{{resolved.ID, resolved.Name}}
	} else {
		pipelines, err := fetchPipelineIDsForList(client, cfg.Workspace)
		if err != nil {
			return err
		}
		for _, p := range pipelines {
			pipelineIDs = append(pipelineIDs, struct{ ID, Name string }{p.ID, p.Name})
		}
	}

	// Scan each pipeline in parallel
	type pipelineResult struct {
		issues []activityIssue
		err    error
	}
	results := make([]pipelineResult, len(pipelineIDs))
	var wg sync.WaitGroup

	for i, p := range pipelineIDs {
		wg.Add(1)
		go func(idx int, pipelineID, pipelineName string) {
			defer wg.Done()
			issues, err := scanPipelineActivity(client, cfg.Workspace, pipelineID, pipelineName, fromTime, toTime)
			results[idx] = pipelineResult{issues: issues, err: err}
		}(i, p.ID, p.Name)
	}
	wg.Wait()

	// Collect results
	issueMap := make(map[string]*activityIssue) // dedup by ID
	for _, r := range results {
		if r.err != nil {
			return r.err
		}
		for i := range r.issues {
			issue := &r.issues[i]
			if _, exists := issueMap[issue.ID]; !exists {
				issueMap[issue.ID] = issue
			}
		}
	}

	// Scan closed issues
	closedIssues, err := scanClosedActivity(client, cfg.Workspace, fromTime, toTime)
	if err != nil {
		return err
	}
	for i := range closedIssues {
		issue := &closedIssues[i]
		if _, exists := issueMap[issue.ID]; !exists {
			issueMap[issue.ID] = issue
		}
	}

	// Step 2: GitHub search (optional)
	if activityGitHub {
		if ghClient == nil {
			fmt.Fprintln(cmd.ErrOrStderr(), output.Yellow("Warning: --github flag ignored — GitHub access not configured"))
		} else {
			ghIssues, repos, err := searchGitHubActivity(client, ghClient, cfg.Workspace, fromTime)
			if err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "%s\n", output.Yellow("Warning: GitHub search failed: "+err.Error()))
			} else {
				// Build a ref set from ZenHub results to dedup against GitHub results
				// (ZenHub items have richer data: pipeline, assignees, etc.)
				refSeen := make(map[string]bool)
				for _, issue := range issueMap {
					refSeen[issue.Ref] = true
				}
				var ghOnly []*activityIssue
				for i := range ghIssues {
					issue := &ghIssues[i]
					if !refSeen[issue.Ref] {
						issueMap[issue.ID] = issue
						refSeen[issue.Ref] = true
						ghOnly = append(ghOnly, issue)
					}
				}

				// Resolve pipelines for GitHub-sourced items
				if len(ghOnly) > 0 {
					resolveGitHubIssuePipelines(client, cfg.Workspace, ghOnly, repos, issueMap)
				}
			}
		}
	}

	// Collect and filter
	var issues []activityIssue
	for _, issue := range issueMap {
		if repoFilter != "" && !strings.EqualFold(issue.RepoName, repoFilter) {
			continue
		}
		issues = append(issues, *issue)
	}

	// Sort by updated time descending
	sort.Slice(issues, func(i, j int) bool {
		return issues[i].UpdatedAt.After(issues[j].UpdatedAt)
	})

	// Step 3: Detail mode — fetch per-issue timelines
	if activityDetail && len(issues) > 0 {
		fetchActivityTimelines(client, ghClient, activityGitHub, &issues, fromTime, toTime)
	}

	w := cmd.OutOrStdout()

	// JSON output
	if output.IsJSON(outputFormat) {
		pipelineCount := countPipelines(issues)
		return output.JSON(w, map[string]any{
			"from":   fromTime.Format(time.RFC3339),
			"to":     toTime.Format(time.RFC3339),
			"issues": issues,
			"summary": map[string]any{
				"issueCount":    len(issues),
				"pipelineCount": pipelineCount,
			},
		})
	}

	// Render output
	if len(issues) == 0 {
		fmt.Fprintf(w, "No activity found since %s.\n", output.FormatDate(fromTime))
		return nil
	}

	// Build canonical pipeline order from the workspace pipeline list
	pipelineNameOrder := make([]string, len(pipelineIDs))
	for i, p := range pipelineIDs {
		pipelineNameOrder[i] = p.Name
	}

	if activityDetail {
		renderActivityDetail(w, issues, fromTime, toTime, activityGitHub && ghClient != nil, pipelineNameOrder)
	} else {
		renderActivitySummary(w, issues, fromTime, toTime, pipelineNameOrder)
	}

	return nil
}

// scanPipelineActivity scans a single pipeline for issues updated within the time range.
// Uses early termination: stops paginating when updatedAt falls before fromTime.
func scanPipelineActivity(client *api.Client, workspaceID, pipelineID, pipelineName string, fromTime, toTime time.Time) ([]activityIssue, error) {
	var issues []activityIssue
	var cursor *string
	pageSize := 100

	for {
		vars := map[string]any{
			"pipelineId":  pipelineID,
			"workspaceId": workspaceID,
			"first":       pageSize,
		}
		if cursor != nil {
			vars["after"] = *cursor
		}

		data, err := client.Execute(activitySearchQuery, vars)
		if err != nil {
			return nil, exitcode.General("scanning pipeline activity", err)
		}

		var resp struct {
			SearchIssuesByPipeline struct {
				TotalCount int `json:"totalCount"`
				PageInfo   struct {
					HasNextPage bool   `json:"hasNextPage"`
					EndCursor   string `json:"endCursor"`
				} `json:"pageInfo"`
				Nodes []struct {
					ID          string `json:"id"`
					Number      int    `json:"number"`
					Title       string `json:"title"`
					State       string `json:"state"`
					UpdatedAt   string `json:"updatedAt"`
					GhUpdatedAt string `json:"ghUpdatedAt"`
					Repository  struct {
						Name      string `json:"name"`
						OwnerName string `json:"ownerName"`
					} `json:"repository"`
					Assignees struct {
						Nodes []struct {
							Login string `json:"login"`
						} `json:"nodes"`
					} `json:"assignees"`
					PipelineIssue *struct {
						Pipeline struct {
							Name string `json:"name"`
						} `json:"pipeline"`
					} `json:"pipelineIssue"`
				} `json:"nodes"`
			} `json:"searchIssuesByPipeline"`
		}
		if err := json.Unmarshal(data, &resp); err != nil {
			return nil, exitcode.General("parsing pipeline activity", err)
		}

		pastCutoff := false
		for _, node := range resp.SearchIssuesByPipeline.Nodes {
			updatedAt, _ := time.Parse(time.RFC3339, node.UpdatedAt)
			ghUpdatedAt, _ := time.Parse(time.RFC3339, node.GhUpdatedAt)

			// Early termination: if ZenHub updatedAt is before our range
			// and ghUpdatedAt is also before our range, we can stop
			if !updatedAt.IsZero() && updatedAt.Before(fromTime) {
				if ghUpdatedAt.IsZero() || ghUpdatedAt.Before(fromTime) {
					pastCutoff = true
					continue
				}
			}

			// Check if either timestamp is in range
			inRange := false
			if !updatedAt.IsZero() && !updatedAt.Before(fromTime) && !updatedAt.After(toTime) {
				inRange = true
			}
			if !ghUpdatedAt.IsZero() && !ghUpdatedAt.Before(fromTime) && !ghUpdatedAt.After(toTime) {
				inRange = true
			}

			if !inRange {
				continue
			}

			pipeline := pipelineName
			if node.PipelineIssue != nil {
				pipeline = node.PipelineIssue.Pipeline.Name
			}

			var assignees []string
			for _, a := range node.Assignees.Nodes {
				assignees = append(assignees, a.Login)
			}

			// Use the most recent timestamp
			displayTime := updatedAt
			if !ghUpdatedAt.IsZero() && ghUpdatedAt.After(displayTime) {
				displayTime = ghUpdatedAt
			}

			issues = append(issues, activityIssue{
				ID:          node.ID,
				Number:      node.Number,
				Title:       node.Title,
				Ref:         fmt.Sprintf("%s#%d", node.Repository.Name, node.Number),
				Pipeline:    pipeline,
				UpdatedAt:   displayTime,
				GhUpdatedAt: ghUpdatedAt,
				Assignees:   assignees,
				RepoName:    node.Repository.Name,
				RepoOwner:   node.Repository.OwnerName,
			})
		}

		// Stop if we've gone past the cutoff or no more pages
		if pastCutoff || !resp.SearchIssuesByPipeline.PageInfo.HasNextPage {
			break
		}
		cursor = &resp.SearchIssuesByPipeline.PageInfo.EndCursor
	}

	return issues, nil
}

// scanClosedActivity fetches recently closed issues and filters by time range.
func scanClosedActivity(client *api.Client, workspaceID string, fromTime, toTime time.Time) ([]activityIssue, error) {
	vars := map[string]any{
		"workspaceId": workspaceID,
		"first":       100,
	}

	data, err := client.Execute(activityClosedQuery, vars)
	if err != nil {
		return nil, exitcode.General("scanning closed issues", err)
	}

	var resp struct {
		SearchClosedIssues struct {
			Nodes []struct {
				ID          string `json:"id"`
				Number      int    `json:"number"`
				Title       string `json:"title"`
				State       string `json:"state"`
				UpdatedAt   string `json:"updatedAt"`
				GhUpdatedAt string `json:"ghUpdatedAt"`
				Repository  struct {
					Name      string `json:"name"`
					OwnerName string `json:"ownerName"`
				} `json:"repository"`
				Assignees struct {
					Nodes []struct {
						Login string `json:"login"`
					} `json:"nodes"`
				} `json:"assignees"`
			} `json:"nodes"`
		} `json:"searchClosedIssues"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, exitcode.General("parsing closed issues activity", err)
	}

	var issues []activityIssue
	for _, node := range resp.SearchClosedIssues.Nodes {
		updatedAt, _ := time.Parse(time.RFC3339, node.UpdatedAt)
		ghUpdatedAt, _ := time.Parse(time.RFC3339, node.GhUpdatedAt)

		inRange := false
		if !updatedAt.IsZero() && !updatedAt.Before(fromTime) && !updatedAt.After(toTime) {
			inRange = true
		}
		if !ghUpdatedAt.IsZero() && !ghUpdatedAt.Before(fromTime) && !ghUpdatedAt.After(toTime) {
			inRange = true
		}
		if !inRange {
			continue
		}

		var assignees []string
		for _, a := range node.Assignees.Nodes {
			assignees = append(assignees, a.Login)
		}

		displayTime := updatedAt
		if !ghUpdatedAt.IsZero() && ghUpdatedAt.After(displayTime) {
			displayTime = ghUpdatedAt
		}

		issues = append(issues, activityIssue{
			ID:          node.ID,
			Number:      node.Number,
			Title:       node.Title,
			Ref:         fmt.Sprintf("%s#%d", node.Repository.Name, node.Number),
			Pipeline:    "Closed",
			UpdatedAt:   displayTime,
			GhUpdatedAt: ghUpdatedAt,
			Assignees:   assignees,
			RepoName:    node.Repository.Name,
			RepoOwner:   node.Repository.OwnerName,
		})
	}

	return issues, nil
}

// searchGitHubActivity searches GitHub for recently updated issues/PRs
// across all workspace repos. Returns the discovered issues and the repo list.
func searchGitHubActivity(client *api.Client, ghClient *gh.Client, workspaceID string, fromTime time.Time) ([]activityIssue, []resolve.CachedRepo, error) {
	// Get workspace repos
	repos, err := fetchWorkspaceReposForActivity(client, workspaceID)
	if err != nil {
		return nil, nil, err
	}
	if len(repos) == 0 {
		return nil, repos, nil
	}

	// Build GitHub search query: "repo:owner/name repo:owner/name2 updated:>YYYY-MM-DD"
	// GitHub search has length limits, so batch if needed
	dateStr := fromTime.Format("2006-01-02T15:04:05")
	var allIssues []activityIssue

	// Build repo parts
	var repoParts []string
	for _, r := range repos {
		repoParts = append(repoParts, fmt.Sprintf("repo:%s/%s", r.OwnerName, r.Name))
	}

	// Batch into queries that fit within ~200 chars for repo portion
	batch := make([]string, 0)
	batchLen := 0
	for _, part := range repoParts {
		if batchLen+len(part)+1 > 200 && len(batch) > 0 {
			issues, err := runGitHubSearchBatch(ghClient, batch, dateStr)
			if err != nil {
				return nil, repos, err
			}
			allIssues = append(allIssues, issues...)
			batch = batch[:0]
			batchLen = 0
		}
		batch = append(batch, part)
		batchLen += len(part) + 1
	}
	if len(batch) > 0 {
		issues, err := runGitHubSearchBatch(ghClient, batch, dateStr)
		if err != nil {
			return nil, repos, err
		}
		allIssues = append(allIssues, issues...)
	}

	return allIssues, repos, nil
}

func runGitHubSearchBatch(ghClient *gh.Client, repoParts []string, dateStr string) ([]activityIssue, error) {
	repoClause := strings.Join(repoParts, " ")

	// GitHub's search API requires an explicit is:issue or is:pr qualifier
	// to return results; without one, repo-scoped searches return 0 hits.
	// Run both searches and merge.
	var allIssues []activityIssue
	for _, typeFilter := range []string{"is:issue", "is:pr"} {
		query := repoClause + " " + typeFilter + " updated:>" + dateStr
		var cursor *string

		for {
			vars := map[string]any{
				"query": query,
				"first": 100,
			}
			if cursor != nil {
				vars["after"] = *cursor
			}

			data, err := ghClient.Execute(activityGitHubSearchQuery, vars)
			if err != nil {
				return nil, err
			}

			var resp struct {
				Search struct {
					IssueCount int `json:"issueCount"`
					PageInfo   struct {
						HasNextPage bool   `json:"hasNextPage"`
						EndCursor   string `json:"endCursor"`
					} `json:"pageInfo"`
					Nodes []struct {
						Number     int    `json:"number"`
						Title      string `json:"title"`
						UpdatedAt  string `json:"updatedAt"`
						Repository struct {
							Name  string `json:"name"`
							Owner struct {
								Login string `json:"login"`
							} `json:"owner"`
						} `json:"repository"`
					} `json:"nodes"`
				} `json:"search"`
			}
			if err := json.Unmarshal(data, &resp); err != nil {
				return nil, fmt.Errorf("parsing GitHub search: %w", err)
			}

			for _, node := range resp.Search.Nodes {
				if node.Number == 0 {
					continue
				}
				updatedAt, _ := time.Parse(time.RFC3339, node.UpdatedAt)
				allIssues = append(allIssues, activityIssue{
					ID:        fmt.Sprintf("gh:%s/%s#%d", node.Repository.Owner.Login, node.Repository.Name, node.Number),
					Number:    node.Number,
					Title:     node.Title,
					Ref:       fmt.Sprintf("%s#%d", node.Repository.Name, node.Number),
					UpdatedAt: updatedAt,
					RepoName:  node.Repository.Name,
					RepoOwner: node.Repository.Owner.Login,
				})
			}

			if !resp.Search.PageInfo.HasNextPage {
				break
			}
			cursor = &resp.Search.PageInfo.EndCursor
		}
	}

	return allIssues, nil
}

// fetchWorkspaceReposForActivity gets workspace repos from cache,
// falling back to the API if the cache is empty.
func fetchWorkspaceReposForActivity(client *api.Client, workspaceID string) ([]resolve.CachedRepo, error) {
	key := resolve.RepoCacheKey(workspaceID)
	repos, ok := cache.Get[[]resolve.CachedRepo](key)
	if !ok {
		var err error
		repos, err = resolve.FetchRepos(client, workspaceID)
		if err != nil {
			return nil, err
		}
	}
	return repos, nil
}

// activityIssueByInfoQuery fetches a single issue by repo+number, including its pipeline.
const activityIssueByInfoQuery = `query ActivityIssueByInfo($repositoryGhId: Int!, $issueNumber: Int!, $workspaceId: ID!) {
  issueByInfo(repositoryGhId: $repositoryGhId, issueNumber: $issueNumber) {
    id
    pipelineIssue(workspaceId: $workspaceId) {
      pipeline { name }
    }
  }
}`

// activityDefaultPRPipelineQuery fetches pipelines to find the default PR pipeline.
const activityDefaultPRPipelineQuery = `query ActivityDefaultPRPipeline($workspaceId: ID!) {
  workspace(id: $workspaceId) {
    pipelinesConnection(first: 50) {
      nodes {
        name
        isDefaultPRPipeline
      }
    }
  }
}`

// resolveGitHubIssuePipelines resolves pipeline names for GitHub-sourced activity items.
// For each item it queries ZenHub's issueByInfo to get the real node ID and pipeline.
// Items where pipelineIssue is null (e.g. PRs never moved) get the workspace's default
// PR pipeline name. The issueMap is updated so the old synthetic ID is replaced.
func resolveGitHubIssuePipelines(client *api.Client, workspaceID string, items []*activityIssue, repos []resolve.CachedRepo, issueMap map[string]*activityIssue) {
	// Build repo GhID lookup by owner/name
	repoGhIDs := make(map[string]int) // "owner/name" -> GhID
	for _, r := range repos {
		key := strings.ToLower(r.OwnerName + "/" + r.Name)
		repoGhIDs[key] = r.GhID
	}

	// Lazy-fetch default PR pipeline name
	var defaultPRPipeline string
	var defaultPROnce sync.Once

	fetchDefaultPRPipeline := func() string {
		defaultPROnce.Do(func() {
			data, err := client.Execute(activityDefaultPRPipelineQuery, map[string]any{
				"workspaceId": workspaceID,
			})
			if err != nil {
				return
			}
			var resp struct {
				Workspace struct {
					PipelinesConnection struct {
						Nodes []struct {
							Name                string `json:"name"`
							IsDefaultPRPipeline bool   `json:"isDefaultPRPipeline"`
						} `json:"nodes"`
					} `json:"pipelinesConnection"`
				} `json:"workspace"`
			}
			if err := json.Unmarshal(data, &resp); err != nil {
				return
			}
			for _, p := range resp.Workspace.PipelinesConnection.Nodes {
				if p.IsDefaultPRPipeline {
					defaultPRPipeline = p.Name
					return
				}
			}
		})
		return defaultPRPipeline
	}

	const concurrency = 5
	sem := make(chan struct{}, concurrency)
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, item := range items {
		ghID, ok := repoGhIDs[strings.ToLower(item.RepoOwner+"/"+item.RepoName)]
		if !ok {
			continue
		}

		wg.Add(1)
		go func(issue *activityIssue, repoGhID int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			data, err := client.Execute(activityIssueByInfoQuery, map[string]any{
				"repositoryGhId": repoGhID,
				"issueNumber":    issue.Number,
				"workspaceId":    workspaceID,
			})
			if err != nil {
				return
			}

			var resp struct {
				IssueByInfo *struct {
					ID            string `json:"id"`
					PipelineIssue *struct {
						Pipeline struct {
							Name string `json:"name"`
						} `json:"pipeline"`
					} `json:"pipelineIssue"`
				} `json:"issueByInfo"`
			}
			if err := json.Unmarshal(data, &resp); err != nil || resp.IssueByInfo == nil {
				return
			}

			mu.Lock()
			defer mu.Unlock()

			oldID := issue.ID
			issue.ID = resp.IssueByInfo.ID

			if resp.IssueByInfo.PipelineIssue != nil {
				issue.Pipeline = resp.IssueByInfo.PipelineIssue.Pipeline.Name
			} else if name := fetchDefaultPRPipeline(); name != "" {
				issue.Pipeline = name
			}

			// Update issueMap: remove old synthetic key, add real ID
			if oldID != issue.ID {
				delete(issueMap, oldID)
				issueMap[issue.ID] = issue
			}
		}(item, ghID)
	}
	wg.Wait()
}

// fetchActivityTimelines fetches per-issue event timelines with bounded concurrency.
func fetchActivityTimelines(client *api.Client, ghClient *gh.Client, includeGitHub bool, issues *[]activityIssue, fromTime, toTime time.Time) {
	const concurrency = 5
	sem := make(chan struct{}, concurrency)
	var mu sync.Mutex
	var wg sync.WaitGroup

	for i := range *issues {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			issue := &(*issues)[idx]
			var events []activityEvent
			ghSourced := strings.HasPrefix(issue.ID, "gh:")

			// Fetch ZenHub timeline (skip for GitHub-sourced items with synthetic IDs)
			if !ghSourced {
				_, zhEvents, err := fetchZenHubTimelineByNode(client, issue.ID)
				if err == nil {
					for _, ev := range zhEvents {
						if !ev.Time.Before(fromTime) && !ev.Time.After(toTime) {
							events = append(events, ev)
						}
					}
				}
			}

			// Fetch GitHub timeline if requested
			var isPR bool
			var createdAt time.Time
			var createdBy string
			if (includeGitHub || ghSourced) && ghClient != nil && issue.RepoOwner != "" {
				ghResult, err := fetchGitHubTimeline(ghClient, issue.RepoOwner, issue.RepoName, issue.Number)
				if err == nil {
					isPR = ghResult.IsPR
					createdAt = ghResult.CreatedAt
					createdBy = ghResult.CreatedBy
					for _, ev := range ghResult.Events {
						if !ev.Time.Before(fromTime) && !ev.Time.After(toTime) {
							events = append(events, ev)
						}
					}
				}
			}

			// For PRs, fetch connected issues via ZenHub connections field
			var connectedIssue string
			if isPR && !ghSourced {
				connectedIssue = fetchPRConnection(client, issue.ID)
			}

			// Synthesize "created" event if creation is within the time range
			if !createdAt.IsZero() && !createdAt.Before(fromTime) && !createdAt.After(toTime) {
				desc := "created this issue"
				if isPR {
					desc = "opened this pull request"
				}
				events = append(events, activityEvent{
					Time:        createdAt,
					Source:      "GitHub",
					Description: desc,
					Actor:       createdBy,
				})
			}

			// Sort events chronologically
			sort.Slice(events, func(i, j int) bool {
				return events[i].Time.Before(events[j].Time)
			})

			mu.Lock()
			issue.Events = events
			issue.IsPR = isPR
			issue.ConnectedIssue = connectedIssue
			mu.Unlock()
		}(i)
	}
	wg.Wait()
}

// Rendering functions

const prConnectionQuery = `query PRConnections($id: ID!) {
  node(id: $id) {
    ... on Issue {
      connections(first: 1) {
        nodes {
          number
          repository { name }
        }
      }
    }
  }
}`

// fetchPRConnection fetches the connected issue for a PR via ZenHub's connections field.
func fetchPRConnection(client *api.Client, nodeID string) string {
	data, err := client.Execute(prConnectionQuery, map[string]any{"id": nodeID})
	if err != nil {
		return ""
	}
	var resp struct {
		Node *struct {
			Connections struct {
				Nodes []struct {
					Number     int `json:"number"`
					Repository struct {
						Name string `json:"name"`
					} `json:"repository"`
				} `json:"nodes"`
			} `json:"connections"`
		} `json:"node"`
	}
	if err := json.Unmarshal(data, &resp); err != nil || resp.Node == nil {
		return ""
	}
	if len(resp.Node.Connections.Nodes) > 0 {
		conn := resp.Node.Connections.Nodes[0]
		if conn.Repository.Name != "" && conn.Number > 0 {
			return fmt.Sprintf("%s#%d", conn.Repository.Name, conn.Number)
		}
	}
	return ""
}

// groupByPipeline groups issues by pipeline, returning pipeline names ordered
// to match the canonical workspace pipeline order. Pipelines not in the
// canonical list (e.g. "Closed", "Unknown") are appended at the end.
func groupByPipeline(issues []activityIssue, canonicalOrder []string) (map[string][]activityIssue, []string) {
	groups := make(map[string][]activityIssue)
	seen := make(map[string]bool)
	for _, issue := range issues {
		pipeline := issue.Pipeline
		if pipeline == "" {
			pipeline = "Unknown"
		}
		groups[pipeline] = append(groups[pipeline], issue)
		seen[pipeline] = true
	}

	// Build ordered list: canonical pipelines first (preserving UI order),
	// then any extras (Closed, Unknown, etc.) in the order encountered.
	var pipelineOrder []string
	for _, name := range canonicalOrder {
		if seen[name] {
			pipelineOrder = append(pipelineOrder, name)
			delete(seen, name)
		}
	}
	// Append remaining pipelines in stable order
	var extras []string
	for name := range seen {
		extras = append(extras, name)
	}
	sort.Strings(extras)
	pipelineOrder = append(pipelineOrder, extras...)

	return groups, pipelineOrder
}

// formatActivityPrefix returns a dim "PR " prefix for pull requests, or empty string for issues.
func formatActivityPrefix(issue activityIssue) string {
	if issue.IsPR {
		return output.Dim("PR ")
	}
	return ""
}

func renderActivitySummary(w interface{ Write([]byte) (int, error) }, issues []activityIssue, fromTime, toTime time.Time, pipelineNameOrder []string) {
	fmt.Fprintf(w, "Activity since %s\n\n", output.FormatDate(fromTime))

	groups, pipelineOrder := groupByPipeline(issues, pipelineNameOrder)

	for i, pipeline := range pipelineOrder {
		if i > 0 {
			fmt.Fprintln(w)
		}
		pipelineIssues := groups[pipeline]
		fmt.Fprintf(w, "%s  %s\n", output.Bold(pipeline), output.Dim(fmt.Sprintf("(%d updated)", len(pipelineIssues))))
		for _, issue := range pipelineIssues {
			title := issue.Title
			if len(title) > 40 {
				title = title[:37] + "..."
			}
			assignee := ""
			if len(issue.Assignees) > 0 {
				assignee = "@" + strings.Join(issue.Assignees, ", @")
			}
			ago := formatTimeAgo(issue.UpdatedAt)
			prefix := formatActivityPrefix(issue)
			line := fmt.Sprintf("  %s%s  %s", prefix, output.Cyan(fmt.Sprintf("%-24s", issue.Ref)), title)
			if assignee != "" {
				line += "  " + output.Dim(assignee)
			}
			line += "  " + output.Dim("updated "+ago)
			fmt.Fprintln(w, line)
		}
	}

	fmt.Fprintf(w, "\n%d issue(s) updated across %d pipeline(s)\n", len(issues), len(pipelineOrder))
}

func renderActivityDetail(w interface{ Write([]byte) (int, error) }, issues []activityIssue, fromTime, toTime time.Time, showSource bool, pipelineNameOrder []string) {
	fmt.Fprintf(w, "Activity since %s\n", output.FormatDate(fromTime))

	groups, pipelineOrder := groupByPipeline(issues, pipelineNameOrder)

	totalEvents := 0
	first := true
	for _, pipeline := range pipelineOrder {
		if !first {
			fmt.Fprintln(w)
		}
		first = false
		pipelineIssues := groups[pipeline]
		fmt.Fprintf(w, "\n%s\n", output.Bold(pipeline))

		for _, issue := range pipelineIssues {
			prefix := formatActivityPrefix(issue)
			header := fmt.Sprintf("\n%s%s: %s", prefix, output.Cyan(issue.Ref), issue.Title)
			if issue.ConnectedIssue != "" {
				header += "  " + output.Dim("→ "+issue.ConnectedIssue)
			}
			fmt.Fprintln(w, header)

			if len(issue.Events) == 0 {
				fmt.Fprintln(w, output.Dim("  (no events in time range)"))
				continue
			}

			for _, ev := range issue.Events {
				dateStr := ev.Time.Format("Jan 2 15:04")
				actor := ""
				if ev.Actor != "" {
					actor = "@" + ev.Actor + " "
				}
				line := fmt.Sprintf("  %s  %s%s", output.Dim(dateStr), actor, ev.Description)
				if showSource {
					line += "  " + output.Dim("["+ev.Source+"]")
				}
				fmt.Fprintln(w, line)
				totalEvents++
			}
		}
	}

	fmt.Fprintf(w, "\n%d issue(s), %d event(s) across %d pipeline(s)\n", len(issues), totalEvents, countPipelines(issues))
}

// formatTimeAgo formats a time as a human-readable relative duration.
func formatTimeAgo(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		m := int(d.Minutes())
		if m == 1 {
			return "1m ago"
		}
		return fmt.Sprintf("%dm ago", m)
	case d < 24*time.Hour:
		h := int(d.Hours())
		if h == 1 {
			return "1h ago"
		}
		return fmt.Sprintf("%dh ago", h)
	default:
		days := int(d.Hours() / 24)
		if days == 1 {
			return "1d ago"
		}
		return fmt.Sprintf("%dd ago", days)
	}
}

func countPipelines(issues []activityIssue) int {
	seen := make(map[string]bool)
	for _, issue := range issues {
		if issue.Pipeline != "" {
			seen[issue.Pipeline] = true
		}
	}
	return len(seen)
}
