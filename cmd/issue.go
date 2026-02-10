package cmd

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/dslh/zh/internal/api"
	"github.com/dslh/zh/internal/config"
	"github.com/dslh/zh/internal/exitcode"
	"github.com/dslh/zh/internal/gh"
	"github.com/dslh/zh/internal/output"
	"github.com/dslh/zh/internal/resolve"
	"github.com/spf13/cobra"
)

// Issue list types

type issueListNode struct {
	ID          string `json:"id"`
	Number      int    `json:"number"`
	Title       string `json:"title"`
	State       string `json:"state"`
	HtmlUrl     string `json:"htmlUrl"`
	PullRequest bool   `json:"pullRequest"`
	Estimate    *struct {
		Value float64 `json:"value"`
	} `json:"estimate"`
	Repository struct {
		ID        string `json:"id"`
		GhID      int    `json:"ghId"`
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
			Name  string `json:"name"`
			Color string `json:"color"`
		} `json:"nodes"`
	} `json:"labels"`
	Sprints struct {
		Nodes []struct {
			ID    string `json:"id"`
			Name  string `json:"name"`
			State string `json:"state"`
		} `json:"nodes"`
	} `json:"sprints"`
	PipelineIssue *struct {
		Pipeline struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"pipeline"`
		Priority *struct {
			Name  string `json:"name"`
			Color string `json:"color"`
		} `json:"priority"`
	} `json:"pipelineIssue"`
}

// Issue detail types

type issueDetailNode struct {
	ID          string  `json:"id"`
	Number      int     `json:"number"`
	Title       string  `json:"title"`
	Body        string  `json:"body"`
	State       string  `json:"state"`
	PullRequest bool    `json:"pullRequest"`
	HtmlUrl     string  `json:"htmlUrl"`
	ZenhubUrl   string  `json:"zenhubUrl"`
	CreatedAt   string  `json:"createdAt"`
	ClosedAt    *string `json:"closedAt"`

	Estimate *struct {
		Value float64 `json:"value"`
	} `json:"estimate"`

	PipelineIssue *struct {
		Pipeline struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"pipeline"`
		Priority *struct {
			Name  string `json:"name"`
			Color string `json:"color"`
		} `json:"priority"`
		LatestTransferTime string `json:"latestTransferTime"`
	} `json:"pipelineIssue"`

	Assignees struct {
		Nodes []struct {
			Login string `json:"login"`
			Name  string `json:"name"`
		} `json:"nodes"`
	} `json:"assignees"`

	Labels struct {
		Nodes []struct {
			ID    string `json:"id"`
			Name  string `json:"name"`
			Color string `json:"color"`
		} `json:"nodes"`
	} `json:"labels"`

	ConnectedPrs struct {
		Nodes []struct {
			ID          string `json:"id"`
			Number      int    `json:"number"`
			Title       string `json:"title"`
			State       string `json:"state"`
			HtmlUrl     string `json:"htmlUrl"`
			PullRequest bool   `json:"pullRequest"`
			Repository  struct {
				Name  string `json:"name"`
				Owner struct {
					Login string `json:"login"`
				} `json:"owner"`
			} `json:"repository"`
		} `json:"nodes"`
	} `json:"connectedPrs"`

	BlockingIssues struct {
		Nodes []issueRefNode `json:"nodes"`
	} `json:"blockingIssues"`

	BlockedIssues struct {
		Nodes []issueRefNode `json:"nodes"`
	} `json:"blockedIssues"`

	ParentZenhubEpics struct {
		Nodes []struct {
			ID    string `json:"id"`
			Title string `json:"title"`
			State string `json:"state"`
		} `json:"nodes"`
	} `json:"parentZenhubEpics"`

	Sprints struct {
		Nodes []struct {
			ID      string `json:"id"`
			Name    string `json:"name"`
			State   string `json:"state"`
			StartAt string `json:"startAt"`
			EndAt   string `json:"endAt"`
		} `json:"nodes"`
	} `json:"sprints"`

	Repository struct {
		ID    string `json:"id"`
		GhID  int    `json:"ghId"`
		Name  string `json:"name"`
		Owner struct {
			Login string `json:"login"`
		} `json:"owner"`
	} `json:"repository"`

	Milestone *struct {
		Title string `json:"title"`
		State string `json:"state"`
		DueOn string `json:"dueOn"`
	} `json:"milestone"`
}

type issueRefNode struct {
	ID         string `json:"id"`
	Number     int    `json:"number"`
	Title      string `json:"title"`
	State      string `json:"state"`
	Repository struct {
		Name  string `json:"name"`
		Owner struct {
			Login string `json:"login"`
		} `json:"owner"`
	} `json:"repository"`
}

// issueGitHubData holds supplementary data from the GitHub API.
type issueGitHubData struct {
	Author    string
	Reactions []issueReaction
	Reviews   []issueReview
	CIStatus  string // "success", "failure", "pending", ""
	IsMerged  bool
	IsDraft   bool
}

type issueReaction struct {
	Content string
	Count   int
}

type issueReview struct {
	Author string
	State  string // "APPROVED", "CHANGES_REQUESTED", "COMMENTED"
}

// GraphQL queries

const issueListByPipelineQuery = `query ListIssuesByPipeline(
  $pipelineId: ID!
  $workspaceId: ID!
  $filters: IssueSearchFiltersInput!
  $first: Int!
  $after: String
) {
  searchIssuesByPipeline(
    pipelineId: $pipelineId
    filters: $filters
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
      htmlUrl
      pullRequest
      estimate { value }
      repository {
        id
        ghId
        name
        ownerName
      }
      assignees(first: 10) {
        nodes { login }
      }
      labels(first: 20) {
        nodes { name color }
      }
      sprints(first: 1) {
        nodes { id name state }
      }
      pipelineIssue(workspaceId: $workspaceId) {
        pipeline { id name }
        priority { name color }
      }
    }
  }
}`

const issueListClosedQuery = `query ListClosedIssues(
  $workspaceId: ID!
  $filters: IssueSearchFiltersInput!
  $first: Int!
  $after: String
) {
  searchClosedIssues(
    workspaceId: $workspaceId
    filters: $filters
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
      htmlUrl
      pullRequest
      estimate { value }
      repository {
        id
        ghId
        name
        ownerName
      }
      assignees(first: 10) {
        nodes { login }
      }
      labels(first: 20) {
        nodes { name color }
      }
      sprints(first: 1) {
        nodes { id name state }
      }
      pipelineIssue(workspaceId: $workspaceId) {
        pipeline { id name }
        priority { name color }
      }
    }
  }
}`

const issueShowQuery = `query GetIssueDetails($repositoryGhId: Int!, $issueNumber: Int!, $workspaceId: ID!) {
  issueByInfo(repositoryGhId: $repositoryGhId, issueNumber: $issueNumber) {
    id
    number
    title
    body
    state
    pullRequest
    htmlUrl
    zenhubUrl(workspaceId: $workspaceId)
    createdAt
    closedAt

    estimate { value }

    pipelineIssue(workspaceId: $workspaceId) {
      pipeline { id name }
      priority { name color }
      latestTransferTime
    }

    assignees(first: 20) {
      nodes { login name }
    }

    labels(first: 50) {
      nodes { id name color }
    }

    connectedPrs(first: 20) {
      nodes {
        id
        number
        title
        state
        htmlUrl
        pullRequest
        repository {
          name
          owner { login }
        }
      }
    }

    blockingIssues(first: 20) {
      nodes {
        id number title state
        repository { name owner { login } }
      }
    }
    blockedIssues(first: 20) {
      nodes {
        id number title state
        repository { name owner { login } }
      }
    }

    parentZenhubEpics(first: 10) {
      nodes { id title state }
    }

    sprints(first: 5) {
      nodes { id name state startAt endAt }
    }

    repository {
      id ghId name
      owner { login }
    }

    milestone { title state dueOn }
  }
}`

const issueShowByNodeQuery = `query GetIssueByNode($id: ID!, $workspaceId: ID!) {
  node(id: $id) {
    ... on Issue {
      id
      number
      title
      body
      state
      pullRequest
      htmlUrl
      zenhubUrl(workspaceId: $workspaceId)
      createdAt
      closedAt

      estimate { value }

      pipelineIssue(workspaceId: $workspaceId) {
        pipeline { id name }
        priority { name color }
        latestTransferTime
      }

      assignees(first: 20) {
        nodes { login name }
      }

      labels(first: 50) {
        nodes { id name color }
      }

      connectedPrs(first: 20) {
        nodes {
          id
          number
          title
          state
          htmlUrl
          pullRequest
          repository {
            name
            owner { login }
          }
        }
      }

      blockingIssues(first: 20) {
        nodes {
          id number title state
          repository { name owner { login } }
        }
      }
      blockedIssues(first: 20) {
        nodes {
          id number title state
          repository { name owner { login } }
        }
      }

      parentZenhubEpics(first: 10) {
        nodes { id title state }
      }

      sprints(first: 5) {
        nodes { id name state startAt endAt }
      }

      repository {
        id ghId name
        owner { login }
      }

      milestone { title state dueOn }
    }
  }
}`

// GitHub GraphQL query for supplementary issue data
const issueShowGitHubQuery = `query GetIssueGitHub($owner: String!, $repo: String!, $number: Int!) {
  repository(owner: $owner, name: $repo) {
    issueOrPullRequest(number: $number) {
      ... on Issue {
        author { login }
        reactionGroups {
          content
          reactors { totalCount }
        }
      }
      ... on PullRequest {
        author { login }
        isDraft
        merged
        reactionGroups {
          content
          reactors { totalCount }
        }
        reviews(last: 20) {
          nodes {
            author { login }
            state
          }
        }
        commits(last: 1) {
          nodes {
            commit {
              statusCheckRollup {
                state
              }
            }
          }
        }
      }
    }
  }
}`

// Commands

var issueCmd = &cobra.Command{
	Use:   "issue",
	Short: "View and manage issues",
	Long:  `List, view, and manage issues and pull requests in the current ZenHub workspace.`,
}

var issueListCmd = &cobra.Command{
	Use:   "list",
	Short: "List issues in the workspace",
	Long: `List issues across all pipelines in the current workspace.

By default, lists open issues across all pipelines. Use filters to narrow results.
Issues are fetched from each pipeline in parallel.`,
	RunE: runIssueList,
}

var issueShowCmd = &cobra.Command{
	Use:   "show [issue]",
	Short: "View issue details",
	Long: `Display detailed information about a single issue or PR.

The issue can be specified as:
  - repo#number (e.g. mpt#1234)
  - owner/repo#number (e.g. gohiring/mpt#1234)
  - ZenHub ID
  - bare number with --repo flag

Use --interactive to select an issue from a list.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runIssueShow,
}

var (
	issueListPipeline   string
	issueListSprint     string
	issueListEpic       string
	issueListAssignee   string
	issueListNoAssigne  bool
	issueListLabel      string
	issueListRepo       string
	issueListEstimate   string
	issueListNoEstimate bool
	issueListType       string
	issueListState      string
	issueListLimit      int
	issueListAll        bool

	issueShowRepo        string
	issueShowInteractive bool
)

func init() {
	issueListCmd.Flags().StringVar(&issueListPipeline, "pipeline", "", "Filter to a specific pipeline")
	issueListCmd.Flags().StringVar(&issueListSprint, "sprint", "", "Filter by sprint (name, ID, or 'current')")
	issueListCmd.Flags().StringVar(&issueListEpic, "epic", "", "Filter by epic (title, ID, or alias)")
	issueListCmd.Flags().StringVar(&issueListAssignee, "assignee", "", "Filter by assignee login")
	issueListCmd.Flags().BoolVar(&issueListNoAssigne, "no-assignee", false, "Show only unassigned issues")
	issueListCmd.Flags().StringVar(&issueListLabel, "label", "", "Filter by label name")
	issueListCmd.Flags().StringVar(&issueListRepo, "repo", "", "Filter by repository name")
	issueListCmd.Flags().StringVar(&issueListEstimate, "estimate", "", "Filter by estimate value")
	issueListCmd.Flags().BoolVar(&issueListNoEstimate, "no-estimate", false, "Show only unestimated issues")
	issueListCmd.Flags().StringVar(&issueListType, "type", "", "Filter by type: issues, prs, or all (default: all)")
	issueListCmd.Flags().StringVar(&issueListState, "state", "", "Filter by state: open or closed (default: open)")
	issueListCmd.Flags().IntVar(&issueListLimit, "limit", 100, "Maximum number of results")
	issueListCmd.Flags().BoolVar(&issueListAll, "all", false, "Fetch all results (ignore --limit)")

	issueShowCmd.Flags().StringVar(&issueShowRepo, "repo", "", "Repository context for bare issue numbers")
	issueShowCmd.Flags().BoolVarP(&issueShowInteractive, "interactive", "i", false, "Select an issue from a list")

	issueCmd.AddCommand(issueListCmd)
	issueCmd.AddCommand(issueShowCmd)
	rootCmd.AddCommand(issueCmd)
}

func resetIssueFlags() {
	issueListPipeline = ""
	issueListSprint = ""
	issueListEpic = ""
	issueListAssignee = ""
	issueListNoAssigne = false
	issueListLabel = ""
	issueListRepo = ""
	issueListEstimate = ""
	issueListNoEstimate = false
	issueListType = ""
	issueListState = ""
	issueListLimit = 100
	issueListAll = false
	issueShowRepo = ""
	issueShowInteractive = false
}

// runIssueList implements `zh issue list`.
func runIssueList(cmd *cobra.Command, args []string) error {
	cfg, err := requireWorkspace()
	if err != nil {
		return err
	}

	client := newClient(cfg, cmd)
	w := cmd.OutOrStdout()

	limit := issueListLimit
	if issueListAll {
		limit = 0
	}

	// Build API filters
	filters := buildIssueListFilters(client, cfg.Workspace)

	// Determine query strategy
	if issueListState == "closed" {
		return runIssueListClosed(client, cfg.Workspace, filters, limit, w)
	}

	// If --epic is set, use epic query strategy
	if issueListEpic != "" {
		return runIssueListByEpic(client, cfg, filters, limit, w)
	}

	// Default: fetch from pipelines
	return runIssueListByPipelines(client, cfg, filters, limit, w)
}

// buildIssueListFilters builds the IssueSearchFiltersInput from flags.
func buildIssueListFilters(client *api.Client, workspaceID string) map[string]any {
	filters := map[string]any{}

	if issueListAssignee != "" {
		filters["assignees"] = map[string]any{"in": []string{issueListAssignee}}
	}
	if issueListNoAssigne {
		filters["assignees"] = map[string]any{"notInAny": true}
	}
	if issueListLabel != "" {
		filters["labels"] = map[string]any{"in": []string{issueListLabel}}
	}
	if issueListEstimate != "" {
		val, parseErr := strconv.ParseFloat(issueListEstimate, 64)
		if parseErr != nil {
			return filters // invalid estimate value, skip filter
		}
		filters["estimates"] = map[string]any{"values": map[string]any{"in": []float64{val}}}
	}
	if issueListNoEstimate {
		filters["estimates"] = map[string]any{"specialFilters": "not_estimated"}
	}
	if issueListType != "" {
		filters["displayType"] = issueListType
	}
	if issueListSprint == "current" {
		filters["sprints"] = map[string]any{"specialFilters": "current_sprint"}
	} else if issueListSprint != "" {
		// Resolve sprint identifier
		resolved, err := resolve.Sprint(client, workspaceID, issueListSprint)
		if err == nil {
			filters["sprints"] = map[string]any{"in": []string{resolved.ID}}
		}
	}
	if issueListRepo != "" {
		// Resolve repo to get its ZenHub ID for filtering
		repo, err := resolve.LookupRepoWithRefresh(client, workspaceID, issueListRepo)
		if err == nil {
			filters["repositoryIds"] = []string{repo.ID}
		}
	}

	return filters
}

// runIssueListByPipelines fetches issues across all (or a filtered) pipeline(s).
func runIssueListByPipelines(client *api.Client, cfg *config.Config, filters map[string]any, limit int, w writerFlusher) error {
	workspaceID := cfg.Workspace

	// Resolve pipelines
	var pipelineIDs []string

	if issueListPipeline != "" {
		resolved, err := resolve.Pipeline(client, workspaceID, issueListPipeline, cfg.Aliases.Pipelines)
		if err != nil {
			return err
		}
		pipelineIDs = []string{resolved.ID}
	} else {
		// Fetch all pipeline IDs
		pipelines, err := fetchPipelineIDsForList(client, workspaceID)
		if err != nil {
			return err
		}
		for _, p := range pipelines {
			pipelineIDs = append(pipelineIDs, p.ID)
		}
	}

	// Fetch issues from each pipeline in parallel
	type pipelineResult struct {
		issues     []issueListNode
		totalCount int
		err        error
	}

	results := make([]pipelineResult, len(pipelineIDs))
	var wg sync.WaitGroup

	for i, pID := range pipelineIDs {
		wg.Add(1)
		go func(idx int, pipelineID string) {
			defer wg.Done()
			issues, total, err := fetchIssuesByPipeline(client, pipelineID, workspaceID, filters, limit)
			results[idx] = pipelineResult{issues: issues, totalCount: total, err: err}
		}(i, pID)
	}
	wg.Wait()

	// Collect results
	var allIssues []issueListNode
	totalCount := 0
	for _, r := range results {
		if r.err != nil {
			return r.err
		}
		allIssues = append(allIssues, r.issues...)
		totalCount += r.totalCount
	}

	// Apply limit after merging
	if limit > 0 && len(allIssues) > limit {
		allIssues = allIssues[:limit]
	}

	if output.IsJSON(outputFormat) {
		return output.JSON(w, allIssues)
	}

	if len(allIssues) == 0 {
		fmt.Fprintln(w, "No issues found.")
		return nil
	}

	renderIssueList(w, allIssues, totalCount)
	return nil
}

// runIssueListClosed fetches closed issues.
func runIssueListClosed(client *api.Client, workspaceID string, filters map[string]any, limit int, w writerFlusher) error {
	issues, totalCount, err := fetchClosedIssues(client, workspaceID, filters, limit)
	if err != nil {
		return err
	}

	if output.IsJSON(outputFormat) {
		return output.JSON(w, issues)
	}

	if len(issues) == 0 {
		fmt.Fprintln(w, "No closed issues found.")
		return nil
	}

	renderIssueList(w, issues, totalCount)
	return nil
}

// runIssueListByEpic fetches issues belonging to an epic.
func runIssueListByEpic(client *api.Client, cfg *config.Config, filters map[string]any, limit int, w writerFlusher) error {
	workspaceID := cfg.Workspace

	resolved, err := resolve.Epic(client, workspaceID, issueListEpic, cfg.Aliases.Epics)
	if err != nil {
		return err
	}

	issues, totalCount, err := fetchIssuesByEpic(client, workspaceID, resolved.ID, filters, limit)
	if err != nil {
		return err
	}

	// Client-side pipeline filter for epic queries
	if issueListPipeline != "" {
		pResolved, err := resolve.Pipeline(client, workspaceID, issueListPipeline, cfg.Aliases.Pipelines)
		if err != nil {
			return err
		}
		var filtered []issueListNode
		for _, issue := range issues {
			if issue.PipelineIssue != nil && issue.PipelineIssue.Pipeline.ID == pResolved.ID {
				filtered = append(filtered, issue)
			}
		}
		issues = filtered
	}

	if output.IsJSON(outputFormat) {
		return output.JSON(w, issues)
	}

	if len(issues) == 0 {
		fmt.Fprintln(w, "No issues found.")
		return nil
	}

	renderIssueList(w, issues, totalCount)
	return nil
}

type writerFlusher = interface {
	Write([]byte) (int, error)
}

// fetchPipelineIDsForList fetches all pipeline IDs for the workspace.
func fetchPipelineIDsForList(client *api.Client, workspaceID string) ([]resolve.CachedPipeline, error) {
	// Try cache first
	key := resolve.PipelineCacheKey(workspaceID)
	if entries, ok := resolve.GetCachedPipelines(key); ok {
		return entries, nil
	}

	// Fetch from API (reuses resolve package's fetch)
	return resolve.FetchPipelines(client, workspaceID)
}

// fetchIssuesByPipeline fetches issues from a single pipeline with pagination.
func fetchIssuesByPipeline(client *api.Client, pipelineID, workspaceID string, filters map[string]any, limit int) ([]issueListNode, int, error) {
	var allIssues []issueListNode
	var cursor *string
	totalCount := 0
	pageSize := 50

	for {
		if limit > 0 {
			remaining := limit - len(allIssues)
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
			"filters":     filters,
			"first":       pageSize,
		}
		if cursor != nil {
			vars["after"] = *cursor
		}

		data, err := client.Execute(issueListByPipelineQuery, vars)
		if err != nil {
			return nil, 0, exitcode.General("fetching issues", err)
		}

		var resp struct {
			SearchIssuesByPipeline struct {
				TotalCount int `json:"totalCount"`
				PageInfo   struct {
					HasNextPage bool   `json:"hasNextPage"`
					EndCursor   string `json:"endCursor"`
				} `json:"pageInfo"`
				Nodes []issueListNode `json:"nodes"`
			} `json:"searchIssuesByPipeline"`
		}
		if err := json.Unmarshal(data, &resp); err != nil {
			return nil, 0, exitcode.General("parsing issue list", err)
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

// fetchClosedIssues fetches closed issues with pagination.
func fetchClosedIssues(client *api.Client, workspaceID string, filters map[string]any, limit int) ([]issueListNode, int, error) {
	var allIssues []issueListNode
	var cursor *string
	totalCount := 0
	pageSize := 50

	for {
		if limit > 0 {
			remaining := limit - len(allIssues)
			if remaining <= 0 {
				break
			}
			if remaining < pageSize {
				pageSize = remaining
			}
		}

		vars := map[string]any{
			"workspaceId": workspaceID,
			"filters":     filters,
			"first":       pageSize,
		}
		if cursor != nil {
			vars["after"] = *cursor
		}

		data, err := client.Execute(issueListClosedQuery, vars)
		if err != nil {
			return nil, 0, exitcode.General("fetching closed issues", err)
		}

		var resp struct {
			SearchClosedIssues struct {
				TotalCount int `json:"totalCount"`
				PageInfo   struct {
					HasNextPage bool   `json:"hasNextPage"`
					EndCursor   string `json:"endCursor"`
				} `json:"pageInfo"`
				Nodes []issueListNode `json:"nodes"`
			} `json:"searchClosedIssues"`
		}
		if err := json.Unmarshal(data, &resp); err != nil {
			return nil, 0, exitcode.General("parsing closed issues", err)
		}

		totalCount = resp.SearchClosedIssues.TotalCount
		allIssues = append(allIssues, resp.SearchClosedIssues.Nodes...)

		if !resp.SearchClosedIssues.PageInfo.HasNextPage {
			break
		}
		if limit > 0 && len(allIssues) >= limit {
			break
		}

		cursor = &resp.SearchClosedIssues.PageInfo.EndCursor
	}

	return allIssues, totalCount, nil
}

const issueListByEpicQuery = `query ListIssuesByEpic(
  $zenhubEpicIds: [ID!]!
  $workspaceId: ID!
  $filters: ZenhubEpicIssueSearchFiltersInput!
  $first: Int!
  $after: String
) {
  searchIssuesByZenhubEpics(
    zenhubEpicIds: $zenhubEpicIds
    filters: $filters
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
      htmlUrl
      pullRequest
      estimate { value }
      repository {
        id
        ghId
        name
        ownerName
      }
      assignees(first: 10) {
        nodes { login }
      }
      labels(first: 20) {
        nodes { name color }
      }
      sprints(first: 1) {
        nodes { id name state }
      }
      pipelineIssue(workspaceId: $workspaceId) {
        pipeline { id name }
        priority { name color }
      }
    }
  }
}`

// fetchIssuesByEpic fetches issues belonging to an epic.
func fetchIssuesByEpic(client *api.Client, workspaceID, epicID string, filters map[string]any, limit int) ([]issueListNode, int, error) {
	var allIssues []issueListNode
	var cursor *string
	totalCount := 0
	pageSize := 50

	// Build epic-specific filters (ZenhubEpicIssueSearchFiltersInput only supports workspaces)
	epicFilters := map[string]any{}

	for {
		if limit > 0 {
			remaining := limit - len(allIssues)
			if remaining <= 0 {
				break
			}
			if remaining < pageSize {
				pageSize = remaining
			}
		}

		vars := map[string]any{
			"zenhubEpicIds": []string{epicID},
			"workspaceId":   workspaceID,
			"filters":       epicFilters,
			"first":         pageSize,
		}
		if cursor != nil {
			vars["after"] = *cursor
		}

		data, err := client.Execute(issueListByEpicQuery, vars)
		if err != nil {
			return nil, 0, exitcode.General("fetching epic issues", err)
		}

		var resp struct {
			SearchIssuesByZenhubEpics struct {
				TotalCount int `json:"totalCount"`
				PageInfo   struct {
					HasNextPage bool   `json:"hasNextPage"`
					EndCursor   string `json:"endCursor"`
				} `json:"pageInfo"`
				Nodes []issueListNode `json:"nodes"`
			} `json:"searchIssuesByZenhubEpics"`
		}
		if err := json.Unmarshal(data, &resp); err != nil {
			return nil, 0, exitcode.General("parsing epic issues", err)
		}

		totalCount = resp.SearchIssuesByZenhubEpics.TotalCount
		allIssues = append(allIssues, resp.SearchIssuesByZenhubEpics.Nodes...)

		if !resp.SearchIssuesByZenhubEpics.PageInfo.HasNextPage {
			break
		}
		if limit > 0 && len(allIssues) >= limit {
			break
		}

		cursor = &resp.SearchIssuesByZenhubEpics.PageInfo.EndCursor
	}

	return allIssues, totalCount, nil
}

// renderIssueList renders the issue list in tabular format.
func renderIssueList(w writerFlusher, issues []issueListNode, totalCount int) {
	needLongRef := issueListRepoNamesAmbiguous(issues)

	lw := output.NewListWriter(w, "ISSUE", "TITLE", "EST", "PIPELINE", "ASSIGNEE", "LABELS")
	for _, issue := range issues {
		ref := issueListFormatRef(issue, needLongRef)
		title := issue.Title
		if len(title) > 50 {
			title = title[:47] + "..."
		}

		est := output.TableMissing
		if issue.Estimate != nil {
			est = formatEstimate(issue.Estimate.Value)
		}

		pipeline := output.TableMissing
		if issue.PipelineIssue != nil {
			pipeline = issue.PipelineIssue.Pipeline.Name
		}

		assignee := output.TableMissing
		if len(issue.Assignees.Nodes) > 0 {
			logins := make([]string, len(issue.Assignees.Nodes))
			for i, a := range issue.Assignees.Nodes {
				logins[i] = a.Login
			}
			assignee = strings.Join(logins, ", ")
		}

		labels := output.TableMissing
		if len(issue.Labels.Nodes) > 0 {
			names := make([]string, len(issue.Labels.Nodes))
			for i, l := range issue.Labels.Nodes {
				names[i] = l.Name
			}
			labels = strings.Join(names, ", ")
		}

		lw.Row(output.Cyan(ref), title, est, pipeline, assignee, labels)
	}

	footer := fmt.Sprintf("Showing %d", len(issues))
	if totalCount > len(issues) {
		footer += fmt.Sprintf(" of %d", totalCount)
	}
	footer += " issue(s)"
	lw.FlushWithFooter(footer)
}

// issueListFormatRef formats an issue reference for the list.
func issueListFormatRef(issue issueListNode, longForm bool) string {
	if longForm {
		return fmt.Sprintf("%s/%s#%d", issue.Repository.OwnerName, issue.Repository.Name, issue.Number)
	}
	return fmt.Sprintf("%s#%d", issue.Repository.Name, issue.Number)
}

// issueListRepoNamesAmbiguous checks if any repo name appears with different owners.
func issueListRepoNamesAmbiguous(issues []issueListNode) bool {
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

// runIssueShow implements `zh issue show <issue>`.
func runIssueShow(cmd *cobra.Command, args []string) error {
	cfg, err := requireWorkspace()
	if err != nil {
		return err
	}

	client := newClient(cfg, cmd)
	ghClient := newGitHubClient(cfg, cmd)
	w := cmd.OutOrStdout()

	// Interactive mode: fetch issue list and let the user pick one
	if issueShowInteractive {
		identifier, err := interactiveOrArg(cmd, nil, true, func() ([]selectItem, error) {
			return fetchIssueSelectItems(client, cfg)
		}, "Select an issue")
		if err != nil {
			return err
		}
		return runIssueShowByNode(client, ghClient, cfg.Workspace, identifier, w)
	}

	if len(args) < 1 {
		return exitcode.Usage("requires an issue argument or --interactive flag")
	}

	// Parse the identifier to determine query strategy
	parsed, parseErr := resolve.ParseIssueRef(args[0])

	// If it's a ZenHub ID, use the node query directly
	if parseErr == nil && parsed.ZenHubID != "" {
		return runIssueShowByNode(client, ghClient, cfg.Workspace, parsed.ZenHubID, w)
	}

	// Resolve to get repo GH ID and issue number
	resolved, err := resolve.Issue(client, cfg.Workspace, args[0], &resolve.IssueOptions{
		RepoFlag:     issueShowRepo,
		GitHubClient: ghClient,
	})
	if err != nil {
		return err
	}

	return runIssueShowByInfo(client, ghClient, cfg.Workspace, resolved.RepoGhID, resolved.Number, w)
}

// fetchIssueSelectItems fetches issues and converts them to selectItems for interactive mode.
func fetchIssueSelectItems(client *api.Client, cfg *config.Config) ([]selectItem, error) {
	// Fetch all pipeline IDs
	pipelines, err := fetchPipelineIDsForList(client, cfg.Workspace)
	if err != nil {
		return nil, err
	}

	// Fetch issues from all pipelines in parallel
	type pipelineResult struct {
		issues []issueListNode
		err    error
	}
	results := make([]pipelineResult, len(pipelines))
	var wg sync.WaitGroup
	for i, p := range pipelines {
		wg.Add(1)
		go func(idx int, pipelineID string) {
			defer wg.Done()
			issues, _, err := fetchIssuesByPipeline(client, pipelineID, cfg.Workspace, nil, 100)
			results[idx] = pipelineResult{issues: issues, err: err}
		}(i, p.ID)
	}
	wg.Wait()

	var allIssues []issueListNode
	for _, r := range results {
		if r.err != nil {
			return nil, r.err
		}
		allIssues = append(allIssues, r.issues...)
	}

	needLongRef := issueListRepoNamesAmbiguous(allIssues)
	items := make([]selectItem, len(allIssues))
	for i, issue := range allIssues {
		ref := issueListFormatRef(issue, needLongRef)
		desc := ""
		if issue.PipelineIssue != nil {
			desc = issue.PipelineIssue.Pipeline.Name
		}
		if issue.Estimate != nil {
			if desc != "" {
				desc += " · "
			}
			desc += formatEstimate(issue.Estimate.Value) + "pts"
		}
		items[i] = selectItem{
			id:          issue.ID,
			title:       fmt.Sprintf("%s %s", ref, issue.Title),
			description: desc,
		}
	}
	return items, nil
}

// runIssueShowByInfo fetches issue details using repo GH ID and issue number.
func runIssueShowByInfo(client *api.Client, ghClient *gh.Client, workspaceID string, repoGhID, issueNumber int, w writerFlusher) error {
	data, err := client.Execute(issueShowQuery, map[string]any{
		"repositoryGhId": repoGhID,
		"issueNumber":    issueNumber,
		"workspaceId":    workspaceID,
	})
	if err != nil {
		return exitcode.General("fetching issue details", err)
	}

	var resp struct {
		IssueByInfo *issueDetailNode `json:"issueByInfo"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return exitcode.General("parsing issue details", err)
	}

	if resp.IssueByInfo == nil {
		return exitcode.NotFoundError(fmt.Sprintf("issue #%d not found", issueNumber))
	}

	ghData := fetchGitHubIssueData(ghClient, resp.IssueByInfo)
	return renderIssueDetail(w, resp.IssueByInfo, ghData)
}

// runIssueShowByNode fetches issue details using ZenHub node ID.
func runIssueShowByNode(client *api.Client, ghClient *gh.Client, workspaceID, nodeID string, w writerFlusher) error {
	data, err := client.Execute(issueShowByNodeQuery, map[string]any{
		"id":          nodeID,
		"workspaceId": workspaceID,
	})
	if err != nil {
		return exitcode.General("fetching issue details", err)
	}

	var resp struct {
		Node *issueDetailNode `json:"node"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return exitcode.General("parsing issue details", err)
	}

	if resp.Node == nil {
		return exitcode.NotFoundError(fmt.Sprintf("issue %q not found", nodeID))
	}

	ghData := fetchGitHubIssueData(ghClient, resp.Node)
	return renderIssueDetail(w, resp.Node, ghData)
}

// renderIssueDetail renders the full issue detail view.
func renderIssueDetail(w writerFlusher, issue *issueDetailNode, ghData *issueGitHubData) error {
	if output.IsJSON(outputFormat) {
		jsonData := map[string]any{}
		raw, _ := json.Marshal(issue)
		_ = json.Unmarshal(raw, &jsonData)
		if ghData != nil {
			if ghData.Author != "" {
				jsonData["author"] = ghData.Author
			}
			if len(ghData.Reactions) > 0 {
				jsonData["reactions"] = ghData.Reactions
			}
			if len(ghData.Reviews) > 0 {
				jsonData["reviews"] = ghData.Reviews
			}
			if ghData.CIStatus != "" {
				jsonData["ciStatus"] = ghData.CIStatus
			}
		}
		return output.JSON(w, jsonData)
	}

	// Build the title: "ISSUE: repo#number: Title" or "PR: repo#number: Title"
	entityType := "ISSUE"
	if issue.PullRequest {
		entityType = "PR"
	}
	ref := fmt.Sprintf("%s#%d", issue.Repository.Name, issue.Number)
	title := fmt.Sprintf("%s: %s", ref, issue.Title)

	d := output.NewDetailWriter(w, entityType, title)

	// State (enhanced with PR merge/draft status from GitHub)
	state := formatIssueShowState(issue.State, issue.PullRequest, ghData)

	// Pipeline
	pipeline := output.DetailMissing
	if issue.PipelineIssue != nil {
		pipeline = issue.PipelineIssue.Pipeline.Name
	}

	// Estimate
	estimate := output.DetailMissing
	if issue.Estimate != nil {
		estimate = formatEstimate(issue.Estimate.Value)
	}

	// Priority
	priority := output.DetailMissing
	if issue.PipelineIssue != nil && issue.PipelineIssue.Priority != nil {
		priority = issue.PipelineIssue.Priority.Name
	}

	// Author (from GitHub)
	author := output.DetailMissing
	if ghData != nil && ghData.Author != "" {
		author = "@" + ghData.Author
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

	// Sprint
	sprint := output.DetailMissing
	if len(issue.Sprints.Nodes) > 0 {
		sprint = issue.Sprints.Nodes[0].Name
	}

	// Epic
	epic := output.DetailMissing
	if len(issue.ParentZenhubEpics.Nodes) > 0 {
		names := make([]string, len(issue.ParentZenhubEpics.Nodes))
		for i, e := range issue.ParentZenhubEpics.Nodes {
			names[i] = e.Title
		}
		epic = strings.Join(names, ", ")
	}

	// Milestone
	milestone := output.DetailMissing
	if issue.Milestone != nil {
		milestone = issue.Milestone.Title
	}

	fields := []output.KeyValue{
		output.KV("State", state),
		output.KV("Pipeline", pipeline),
		output.KV("Estimate", estimate),
		output.KV("Priority", priority),
	}

	if ghData != nil && ghData.Author != "" {
		fields = append(fields, output.KV("Author", author))
	}

	fields = append(fields,
		output.KV("Assignees", assignees),
		output.KV("Labels", labels),
		output.KV("Sprint", sprint),
		output.KV("Epic", epic),
	)

	if issue.Milestone != nil {
		fields = append(fields, output.KV("Milestone", milestone))
	}

	// CI status for PRs (from GitHub)
	if ghData != nil && ghData.CIStatus != "" {
		ciStr := formatCIStatus(ghData.CIStatus)
		fields = append(fields, output.KV("CI", ciStr))
	}

	d.Fields(fields)

	// Description section
	if issue.Body != "" {
		d.Section("DESCRIPTION")
		_ = output.RenderMarkdown(w, issue.Body, 80)
	}

	// Connected PRs section (for issues, not PRs themselves)
	if !issue.PullRequest && len(issue.ConnectedPrs.Nodes) > 0 {
		d.Section("CONNECTED PRS")
		lw := output.NewListWriter(w, "PR", "STATE", "TITLE")
		for _, pr := range issue.ConnectedPrs.Nodes {
			prRef := fmt.Sprintf("%s#%d", pr.Repository.Name, pr.Number)
			prState := strings.ToLower(pr.State)
			prTitle := pr.Title
			if len(prTitle) > 50 {
				prTitle = prTitle[:47] + "..."
			}
			lw.Row(output.Cyan(prRef), prState, prTitle)
		}
		lw.Flush()
	}

	// Reviews section (PRs with GitHub access)
	if ghData != nil && len(ghData.Reviews) > 0 {
		d.Section("REVIEWS")
		lw := output.NewListWriter(w, "REVIEWER", "STATUS")
		for _, r := range ghData.Reviews {
			status := formatReviewState(r.State)
			lw.Row("@"+r.Author, status)
		}
		lw.Flush()
	}

	// Reactions (from GitHub)
	if ghData != nil && len(ghData.Reactions) > 0 {
		d.Section("REACTIONS")
		var parts []string
		for _, r := range ghData.Reactions {
			emoji := reactionEmoji(r.Content)
			parts = append(parts, fmt.Sprintf("%s %d", emoji, r.Count))
		}
		fmt.Fprintf(w, "  %s\n", strings.Join(parts, "  "))
	}

	// Blocking section
	if len(issue.BlockedIssues.Nodes) > 0 {
		d.Section("BLOCKING")
		fmt.Fprintln(w, "This issue is blocking:")
		for _, blocked := range issue.BlockedIssues.Nodes {
			ref := fmt.Sprintf("%s#%d", blocked.Repository.Name, blocked.Number)
			state := strings.ToLower(blocked.State)
			fmt.Fprintf(w, "  %s  %s  %s\n", output.Cyan(ref), blocked.Title, output.Dim("("+state+")"))
		}
	}

	// Blocked by section
	if len(issue.BlockingIssues.Nodes) > 0 {
		d.Section("BLOCKED BY")
		fmt.Fprintln(w, "This issue is blocked by:")
		for _, blocker := range issue.BlockingIssues.Nodes {
			ref := fmt.Sprintf("%s#%d", blocker.Repository.Name, blocker.Number)
			state := strings.ToLower(blocker.State)
			fmt.Fprintf(w, "  %s  %s  %s\n", output.Cyan(ref), blocker.Title, output.Dim("("+state+")"))
		}
	}

	// Links section
	d.Section("LINKS")
	if issue.HtmlUrl != "" {
		fmt.Fprintf(w, "  GitHub:  %s\n", output.Cyan(issue.HtmlUrl))
	}
	if issue.ZenhubUrl != "" {
		fmt.Fprintf(w, "  ZenHub:  %s\n", output.Cyan(issue.ZenhubUrl))
	}

	// Timeline section
	d.Section("TIMELINE")
	if issue.CreatedAt != "" {
		if t, err := time.Parse(time.RFC3339, issue.CreatedAt); err == nil {
			fmt.Fprintf(w, "  Created:  %s\n", output.FormatDate(t))
		}
	}
	if issue.ClosedAt != nil && *issue.ClosedAt != "" {
		if t, err := time.Parse(time.RFC3339, *issue.ClosedAt); err == nil {
			fmt.Fprintf(w, "  Closed:   %s\n", output.FormatDate(t))
		}
	}
	if issue.PipelineIssue != nil && issue.PipelineIssue.LatestTransferTime != "" {
		if t, err := time.Parse(time.RFC3339, issue.PipelineIssue.LatestTransferTime); err == nil {
			fmt.Fprintf(w, "  In pipeline since:  %s\n", output.FormatDate(t))
		}
	}

	return nil
}

// formatIssueState formats the issue state for display (used by list view).
func formatIssueState(state string, isPR bool) string {
	lower := strings.ToLower(state)
	switch lower {
	case "open":
		return output.Green("Open")
	case "closed":
		if isPR {
			return output.Red("Closed")
		}
		return output.Green("Closed")
	case "merged":
		return output.Green("Merged")
	default:
		return lower
	}
}

// formatIssueShowState formats state for the detail view, enhanced with GitHub data.
func formatIssueShowState(state string, isPR bool, ghData *issueGitHubData) string {
	if ghData != nil && isPR {
		if ghData.IsMerged {
			return output.Green("Merged")
		}
		if ghData.IsDraft {
			base := formatIssueState(state, isPR)
			return base + " " + output.Dim("(draft)")
		}
	}
	return formatIssueState(state, isPR)
}

// fetchGitHubIssueData fetches supplementary data from GitHub.
// Returns nil if GitHub client is not configured.
func fetchGitHubIssueData(ghClient *gh.Client, issue *issueDetailNode) *issueGitHubData {
	if ghClient == nil {
		return nil
	}

	owner := issue.Repository.Owner.Login
	repo := issue.Repository.Name

	data, err := ghClient.Execute(issueShowGitHubQuery, map[string]any{
		"owner":  owner,
		"repo":   repo,
		"number": issue.Number,
	})
	if err != nil {
		// GitHub data is optional — don't fail the command
		return nil
	}

	var resp struct {
		Repository *struct {
			IssueOrPullRequest json.RawMessage `json:"issueOrPullRequest"`
		} `json:"repository"`
	}
	if err := json.Unmarshal(data, &resp); err != nil || resp.Repository == nil {
		return nil
	}

	// Parse into a generic structure that works for both issues and PRs
	var node struct {
		Author *struct {
			Login string `json:"login"`
		} `json:"author"`
		ReactionGroups []struct {
			Content  string `json:"content"`
			Reactors struct {
				TotalCount int `json:"totalCount"`
			} `json:"reactors"`
		} `json:"reactionGroups"`
		IsDraft bool `json:"isDraft"`
		Merged  bool `json:"merged"`
		Reviews *struct {
			Nodes []struct {
				Author *struct {
					Login string `json:"login"`
				} `json:"author"`
				State string `json:"state"`
			} `json:"nodes"`
		} `json:"reviews"`
		Commits *struct {
			Nodes []struct {
				Commit struct {
					StatusCheckRollup *struct {
						State string `json:"state"`
					} `json:"statusCheckRollup"`
				} `json:"commit"`
			} `json:"nodes"`
		} `json:"commits"`
	}
	if err := json.Unmarshal(resp.Repository.IssueOrPullRequest, &node); err != nil {
		return nil
	}

	result := &issueGitHubData{
		IsMerged: node.Merged,
		IsDraft:  node.IsDraft,
	}

	if node.Author != nil {
		result.Author = node.Author.Login
	}

	// Reactions (only include those with count > 0)
	for _, rg := range node.ReactionGroups {
		if rg.Reactors.TotalCount > 0 {
			result.Reactions = append(result.Reactions, issueReaction{
				Content: rg.Content,
				Count:   rg.Reactors.TotalCount,
			})
		}
	}

	// Reviews (deduplicate per author, keep latest state)
	if node.Reviews != nil {
		seen := make(map[string]string) // author -> latest state
		var order []string
		for _, r := range node.Reviews.Nodes {
			if r.Author == nil {
				continue
			}
			if _, ok := seen[r.Author.Login]; !ok {
				order = append(order, r.Author.Login)
			}
			seen[r.Author.Login] = r.State
		}
		for _, login := range order {
			result.Reviews = append(result.Reviews, issueReview{
				Author: login,
				State:  seen[login],
			})
		}
	}

	// CI status
	if node.Commits != nil && len(node.Commits.Nodes) > 0 {
		commit := node.Commits.Nodes[0]
		if commit.Commit.StatusCheckRollup != nil {
			result.CIStatus = strings.ToLower(commit.Commit.StatusCheckRollup.State)
		}
	}

	return result
}

// formatCIStatus formats the CI status for display.
func formatCIStatus(status string) string {
	switch status {
	case "success":
		return output.Green("Passing")
	case "failure", "error":
		return output.Red("Failing")
	case "pending", "expected":
		return output.Yellow("Pending")
	default:
		return status
	}
}

// formatReviewState formats a PR review state for display.
func formatReviewState(state string) string {
	switch state {
	case "APPROVED":
		return output.Green("Approved")
	case "CHANGES_REQUESTED":
		return output.Red("Changes requested")
	case "COMMENTED":
		return "Commented"
	case "DISMISSED":
		return output.Dim("Dismissed")
	case "PENDING":
		return output.Yellow("Pending")
	default:
		return state
	}
}

// reactionEmoji converts a GitHub reaction content to an emoji.
func reactionEmoji(content string) string {
	switch content {
	case "THUMBS_UP":
		return "+1"
	case "THUMBS_DOWN":
		return "-1"
	case "LAUGH":
		return "laugh"
	case "HOORAY":
		return "hooray"
	case "CONFUSED":
		return "confused"
	case "HEART":
		return "heart"
	case "ROCKET":
		return "rocket"
	case "EYES":
		return "eyes"
	default:
		return strings.ToLower(content)
	}
}
