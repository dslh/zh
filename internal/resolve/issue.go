package resolve

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"

	"github.com/dslh/zh/internal/api"
	"github.com/dslh/zh/internal/cache"
	"github.com/dslh/zh/internal/exitcode"
	"github.com/dslh/zh/internal/gh"
)

// IssueResult is the resolved issue returned to callers.
type IssueResult struct {
	ID        string // ZenHub node ID
	Number    int    // GitHub issue/PR number
	RepoGhID  int    // GitHub repository ID
	RepoOwner string // Repository owner
	RepoName  string // Repository name
}

// issueRefPattern matches "repo#123" or "owner/repo#123".
var issueRefPattern = regexp.MustCompile(`^(?:([^/#]+)/)?([^/#]+)#(\d+)$`)

// ParsedIssueRef is the result of parsing an issue identifier string.
type ParsedIssueRef struct {
	ZenHubID string // non-empty if the identifier is a ZenHub ID
	Owner    string // optional, from owner/repo#number
	Repo     string // from repo#number
	Number   int    // issue/PR number
}

// ParseIssueRef parses an issue identifier string into its components.
// Accepted formats:
//   - ZenHub ID (base64-encoded string, typically starting with Z2lk)
//   - owner/repo#number
//   - repo#number
//   - bare number (only valid when repoFlag is set)
func ParseIssueRef(identifier string) (*ParsedIssueRef, error) {
	// Try GitHub ref format: owner/repo#number or repo#number
	if m := issueRefPattern.FindStringSubmatch(identifier); m != nil {
		num, _ := strconv.Atoi(m[3])
		return &ParsedIssueRef{
			Owner:  m[1],
			Repo:   m[2],
			Number: num,
		}, nil
	}

	// Try bare number (requires --repo context, handled by caller)
	if num, err := strconv.Atoi(identifier); err == nil && num > 0 {
		return &ParsedIssueRef{Number: num}, nil
	}

	// Assume ZenHub ID (base64 string, typically long)
	if looksLikeZenHubID(identifier) {
		return &ParsedIssueRef{ZenHubID: identifier}, nil
	}

	return nil, exitcode.Usage(fmt.Sprintf("invalid issue identifier %q — expected repo#number, owner/repo#number, or ZenHub ID", identifier))
}

// looksLikeZenHubID returns true if the string looks like a base64-encoded
// ZenHub node ID. These are typically long alphanumeric strings.
func looksLikeZenHubID(s string) bool {
	if len(s) < 10 {
		return false
	}
	for _, c := range s {
		isAlpha := (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
		isDigit := c >= '0' && c <= '9'
		isBase64 := c == '/' || c == '+' || c == '='
		if !isAlpha && !isDigit && !isBase64 {
			return false
		}
	}
	return true
}

// IssueOptions configures issue resolution behavior.
type IssueOptions struct {
	// RepoFlag is the value of the --repo flag, providing default repo context
	// for bare issue numbers. Format: "repo" or "owner/repo".
	RepoFlag string

	// GitHubClient is used for branch name resolution. May be nil.
	GitHubClient *gh.Client
}

// Issue resolves an issue identifier to an IssueResult. It supports:
//   - ZenHub ID: queries the node directly
//   - owner/repo#number: looks up repo in cache, queries issueByInfo
//   - repo#number: same, using cached repos for owner resolution
//   - bare number (with --repo): resolves against the repo flag
//   - branch name (with --repo and GitHub access): resolves PR by branch
//
// The client parameter is for the ZenHub API. Options provide additional
// context like --repo flag and GitHub client.
func Issue(client *api.Client, workspaceID string, identifier string, opts *IssueOptions) (*IssueResult, error) {
	if opts == nil {
		opts = &IssueOptions{}
	}

	parsed, err := ParseIssueRef(identifier)
	if err != nil {
		// If parsing fails and we have a repo flag + GitHub client, try branch name
		if opts.RepoFlag != "" && opts.GitHubClient != nil {
			return resolveByBranch(client, workspaceID, identifier, opts)
		}
		return nil, err
	}

	// ZenHub ID — query directly
	if parsed.ZenHubID != "" {
		return resolveByZenHubID(client, workspaceID, parsed.ZenHubID)
	}

	// Bare number — requires --repo
	if parsed.Repo == "" && parsed.Number > 0 {
		if opts.RepoFlag == "" {
			return nil, exitcode.Usage(fmt.Sprintf("bare issue number %d requires --repo flag", parsed.Number))
		}
		// Resolve the repo flag to get owner/name
		repo, err := LookupRepoWithRefresh(client, workspaceID, opts.RepoFlag)
		if err != nil {
			return nil, err
		}
		return resolveByRepoAndNumber(client, workspaceID, repo, parsed.Number)
	}

	// repo#number or owner/repo#number
	repoID := parsed.Repo
	if parsed.Owner != "" {
		repoID = parsed.Owner + "/" + parsed.Repo
	}
	repo, err := LookupRepoWithRefresh(client, workspaceID, repoID)
	if err != nil {
		return nil, err
	}
	return resolveByRepoAndNumber(client, workspaceID, repo, parsed.Number)
}

// LookupRepoWithRefresh resolves a repo identifier using cache with
// invalidate-on-miss. The identifier can be "repo" or "owner/repo".
func LookupRepoWithRefresh(client *api.Client, workspaceID string, identifier string) (*CachedRepo, error) {
	key := RepoCacheKey(workspaceID)

	// Try cache first
	if entries, ok := cache.Get[[]CachedRepo](key); ok {
		repo, err := LookupRepo(entries, identifier)
		if err == nil {
			return repo, nil
		}
		// If ambiguous, return that error immediately (won't change after refresh)
		if exitcode.ExitCode(err) == exitcode.UsageError {
			return nil, err
		}
	}

	// Cache miss — refresh from API
	entries, err := FetchRepos(client, workspaceID)
	if err != nil {
		return nil, err
	}

	return LookupRepo(entries, identifier)
}

const issueByInfoQuery = `query IssueByInfo($repositoryGhId: Int!, $issueNumber: Int!) {
  issueByInfo(repositoryGhId: $repositoryGhId, issueNumber: $issueNumber) {
    id
    number
    repository {
      ghId
      name
      ownerName
    }
  }
}`

// resolveByRepoAndNumber resolves an issue by its repo and number using
// the issueByInfo query.
func resolveByRepoAndNumber(client *api.Client, workspaceID string, repo *CachedRepo, number int) (*IssueResult, error) {
	data, err := client.Execute(issueByInfoQuery, map[string]any{
		"repositoryGhId": repo.GhID,
		"issueNumber":    number,
	})
	if err != nil {
		return nil, exitcode.General("fetching issue details", err)
	}

	var resp struct {
		IssueByInfo *struct {
			ID         string `json:"id"`
			Number     int    `json:"number"`
			Repository struct {
				GhID      int    `json:"ghId"`
				Name      string `json:"name"`
				OwnerName string `json:"ownerName"`
			} `json:"repository"`
		} `json:"issueByInfo"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, exitcode.General("parsing issue response", err)
	}

	if resp.IssueByInfo == nil {
		return nil, exitcode.NotFoundError(fmt.Sprintf("issue %s/%s#%d not found", repo.OwnerName, repo.Name, number))
	}

	issue := resp.IssueByInfo
	return &IssueResult{
		ID:        issue.ID,
		Number:    issue.Number,
		RepoGhID:  issue.Repository.GhID,
		RepoOwner: issue.Repository.OwnerName,
		RepoName:  issue.Repository.Name,
	}, nil
}

const issueByNodeQuery = `query IssueByNode($id: ID!) {
  node(id: $id) {
    ... on Issue {
      id
      number
      repository {
        ghId
        name
        ownerName
      }
    }
  }
}`

// resolveByZenHubID resolves an issue by its ZenHub node ID.
func resolveByZenHubID(client *api.Client, workspaceID string, zenHubID string) (*IssueResult, error) {
	data, err := client.Execute(issueByNodeQuery, map[string]any{
		"id": zenHubID,
	})
	if err != nil {
		return nil, exitcode.General("fetching issue by ID", err)
	}

	var resp struct {
		Node *struct {
			ID         string `json:"id"`
			Number     int    `json:"number"`
			Repository *struct {
				GhID      int    `json:"ghId"`
				Name      string `json:"name"`
				OwnerName string `json:"ownerName"`
			} `json:"repository"`
		} `json:"node"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, exitcode.General("parsing issue response", err)
	}

	if resp.Node == nil || resp.Node.Repository == nil {
		return nil, exitcode.NotFoundError(fmt.Sprintf("issue %q not found", zenHubID))
	}

	node := resp.Node
	return &IssueResult{
		ID:        node.ID,
		Number:    node.Number,
		RepoGhID:  node.Repository.GhID,
		RepoOwner: node.Repository.OwnerName,
		RepoName:  node.Repository.Name,
	}, nil
}

// GitHub REST query for finding PR by branch name
const githubPRByBranchQuery = `query PRByBranch($owner: String!, $repo: String!, $head: String!) {
  repository(owner: $owner, name: $repo) {
    pullRequests(headRefName: $head, first: 1, states: [OPEN, CLOSED, MERGED]) {
      nodes {
        number
      }
    }
  }
}`

// resolveByBranch resolves an issue by branch name using GitHub API.
// The branch name is looked up as a PR head ref in the given repo.
func resolveByBranch(client *api.Client, workspaceID string, branchName string, opts *IssueOptions) (*IssueResult, error) {
	repo, err := LookupRepoWithRefresh(client, workspaceID, opts.RepoFlag)
	if err != nil {
		return nil, err
	}

	data, err := opts.GitHubClient.Execute(githubPRByBranchQuery, map[string]any{
		"owner": repo.OwnerName,
		"repo":  repo.Name,
		"head":  branchName,
	})
	if err != nil {
		return nil, exitcode.General("looking up PR by branch name", err)
	}

	var resp struct {
		Repository struct {
			PullRequests struct {
				Nodes []struct {
					Number int `json:"number"`
				} `json:"nodes"`
			} `json:"pullRequests"`
		} `json:"repository"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, exitcode.General("parsing GitHub PR response", err)
	}

	prs := resp.Repository.PullRequests.Nodes
	if len(prs) == 0 {
		return nil, exitcode.NotFoundError(fmt.Sprintf("no PR found for branch %q in %s/%s", branchName, repo.OwnerName, repo.Name))
	}

	return resolveByRepoAndNumber(client, workspaceID, repo, prs[0].Number)
}

// Ref returns the short reference string (repo#number) for this resolved issue.
func (r *IssueResult) Ref() string {
	return fmt.Sprintf("%s#%d", r.RepoName, r.Number)
}

// FullRef returns the long reference string (owner/repo#number).
func (r *IssueResult) FullRef() string {
	return fmt.Sprintf("%s/%s#%d", r.RepoOwner, r.RepoName, r.Number)
}
