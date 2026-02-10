package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/dslh/zh/internal/api"
	"github.com/dslh/zh/internal/exitcode"
	"github.com/dslh/zh/internal/gh"
	"github.com/dslh/zh/internal/output"
	"github.com/dslh/zh/internal/resolve"
	"github.com/spf13/cobra"
)

// GraphQL queries and mutations for issue connect/disconnect

const issueConnectResolveQuery = `query GetIssueForConnect($repositoryGhId: Int!, $issueNumber: Int!) {
  issueByInfo(repositoryGhId: $repositoryGhId, issueNumber: $issueNumber) {
    id
    number
    title
    pullRequest
    repository {
      name
      ownerName
    }
  }
}`

const issueConnectResolveByNodeQuery = `query GetIssueForConnectByNode($id: ID!) {
  node(id: $id) {
    ... on Issue {
      id
      number
      title
      pullRequest
      repository {
        name
        ownerName
      }
    }
  }
}`

const createIssuePrConnectionMutation = `mutation CreateIssuePrConnection($input: CreateIssuePrConnectionInput!) {
  createIssuePrConnection(input: $input) {
    issue {
      id
      number
      title
    }
    pullRequest {
      id
      number
      title
    }
  }
}`

const deleteIssuePrConnectionMutation = `mutation DeleteIssuePrConnection($input: DeleteIssuePrConnectionInput!) {
  deleteIssuePrConnection(input: $input) {
    issue {
      id
      number
      title
    }
    pullRequest {
      id
      number
      title
    }
  }
}`

// resolvedConnectItem holds the info for a resolved issue or PR in a connect/disconnect operation.
type resolvedConnectItem struct {
	ID          string
	Number      int
	Title       string
	RepoName    string
	RepoOwner   string
	PullRequest bool
}

func (r *resolvedConnectItem) Ref() string {
	return fmt.Sprintf("%s#%d", r.RepoName, r.Number)
}

// Commands

var issueConnectCmd = &cobra.Command{
	Use:   "connect <issue> <pr>",
	Short: "Connect a PR to an issue",
	Long: `Connect a pull request to an issue.

The first argument is the issue, the second is the PR.
Both can be specified as repo#number, owner/repo#number, or ZenHub IDs.

Examples:
  zh issue connect task-tracker#1 task-tracker#5
  zh issue connect --repo=task-tracker 1 5`,
	Args: cobra.ExactArgs(2),
	RunE: runIssueConnect,
}

var issueDisconnectCmd = &cobra.Command{
	Use:   "disconnect <issue> <pr>",
	Short: "Disconnect a PR from an issue",
	Long: `Disconnect a pull request from an issue.

The first argument is the issue, the second is the PR.
Both can be specified as repo#number, owner/repo#number, or ZenHub IDs.

Examples:
  zh issue disconnect task-tracker#1 task-tracker#5
  zh issue disconnect --repo=task-tracker 1 5`,
	Args: cobra.ExactArgs(2),
	RunE: runIssueDisconnect,
}

var (
	issueConnectDryRun    bool
	issueConnectRepo      string
	issueDisconnectDryRun bool
	issueDisconnectRepo   string
)

func init() {
	issueConnectCmd.Flags().BoolVar(&issueConnectDryRun, "dry-run", false, "Show what would be connected without executing")
	issueConnectCmd.Flags().StringVar(&issueConnectRepo, "repo", "", "Repository context for bare issue numbers")

	issueDisconnectCmd.Flags().BoolVar(&issueDisconnectDryRun, "dry-run", false, "Show what would be disconnected without executing")
	issueDisconnectCmd.Flags().StringVar(&issueDisconnectRepo, "repo", "", "Repository context for bare issue numbers")

	issueCmd.AddCommand(issueConnectCmd)
	issueCmd.AddCommand(issueDisconnectCmd)
}

func resetIssueConnectFlags() {
	issueConnectDryRun = false
	issueConnectRepo = ""
}

func resetIssueDisconnectFlags() {
	issueDisconnectDryRun = false
	issueDisconnectRepo = ""
}

func runIssueConnect(cmd *cobra.Command, args []string) error {
	cfg, err := requireWorkspace()
	if err != nil {
		return err
	}

	client := newClient(cfg, cmd)
	ghClient := newGitHubClient(cfg, cmd)
	w := cmd.OutOrStdout()

	// Resolve both identifiers
	issue, err := resolveForConnect(client, cfg.Workspace, args[0], issueConnectRepo, ghClient)
	if err != nil {
		return fmt.Errorf("resolving issue: %w", err)
	}

	pr, err := resolveForConnect(client, cfg.Workspace, args[1], issueConnectRepo, ghClient)
	if err != nil {
		return fmt.Errorf("resolving PR: %w", err)
	}

	// Validate: first arg should be an issue, second should be a PR
	if issue.PullRequest {
		return exitcode.Usage(fmt.Sprintf("%s is a pull request, not an issue — the first argument should be an issue", issue.Ref()))
	}
	if !pr.PullRequest {
		return exitcode.Usage(fmt.Sprintf("%s is an issue, not a pull request — the second argument should be a PR", pr.Ref()))
	}

	// Dry run
	if issueConnectDryRun {
		if output.IsJSON(outputFormat) {
			return output.JSON(w, map[string]any{
				"dryRun": true,
				"issue":  formatConnectItemJSON(issue),
				"pr":     formatConnectItemJSON(pr),
			})
		}
		msg := fmt.Sprintf("Would connect %s to %s", pr.Ref(), issue.Ref())
		output.MutationDryRun(w, msg, []output.MutationItem{
			{Ref: issue.Ref(), Title: truncateTitle(issue.Title), Context: "(issue)"},
			{Ref: pr.Ref(), Title: truncateTitle(pr.Title), Context: "(PR)"},
		})
		return nil
	}

	// Execute mutation
	data, err := client.Execute(createIssuePrConnectionMutation, map[string]any{
		"input": map[string]any{
			"issueId":       issue.ID,
			"pullRequestId": pr.ID,
		},
	})
	if err != nil {
		return exitcode.General("connecting PR to issue", err)
	}

	var resp struct {
		CreateIssuePrConnection struct {
			Issue struct {
				ID     string `json:"id"`
				Number int    `json:"number"`
				Title  string `json:"title"`
			} `json:"issue"`
			PullRequest struct {
				ID     string `json:"id"`
				Number int    `json:"number"`
				Title  string `json:"title"`
			} `json:"pullRequest"`
		} `json:"createIssuePrConnection"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return exitcode.General("parsing connect response", err)
	}

	// JSON output
	if output.IsJSON(outputFormat) {
		return output.JSON(w, map[string]any{
			"issue": formatConnectItemJSON(issue),
			"pr":    formatConnectItemJSON(pr),
		})
	}

	output.MutationSingle(w, output.Green(fmt.Sprintf(
		"Connected %s to %s.",
		pr.Ref(), issue.Ref(),
	)))
	return nil
}

func runIssueDisconnect(cmd *cobra.Command, args []string) error {
	cfg, err := requireWorkspace()
	if err != nil {
		return err
	}

	client := newClient(cfg, cmd)
	ghClient := newGitHubClient(cfg, cmd)
	w := cmd.OutOrStdout()

	// Resolve both identifiers
	issue, err := resolveForConnect(client, cfg.Workspace, args[0], issueDisconnectRepo, ghClient)
	if err != nil {
		return fmt.Errorf("resolving issue: %w", err)
	}

	pr, err := resolveForConnect(client, cfg.Workspace, args[1], issueDisconnectRepo, ghClient)
	if err != nil {
		return fmt.Errorf("resolving PR: %w", err)
	}

	// Validate: first arg should be an issue, second should be a PR
	if issue.PullRequest {
		return exitcode.Usage(fmt.Sprintf("%s is a pull request, not an issue — the first argument should be an issue", issue.Ref()))
	}
	if !pr.PullRequest {
		return exitcode.Usage(fmt.Sprintf("%s is an issue, not a pull request — the second argument should be a PR", pr.Ref()))
	}

	// Dry run
	if issueDisconnectDryRun {
		if output.IsJSON(outputFormat) {
			return output.JSON(w, map[string]any{
				"dryRun": true,
				"issue":  formatConnectItemJSON(issue),
				"pr":     formatConnectItemJSON(pr),
			})
		}
		msg := fmt.Sprintf("Would disconnect %s from %s", pr.Ref(), issue.Ref())
		output.MutationDryRun(w, msg, []output.MutationItem{
			{Ref: issue.Ref(), Title: truncateTitle(issue.Title), Context: "(issue)"},
			{Ref: pr.Ref(), Title: truncateTitle(pr.Title), Context: "(PR)"},
		})
		return nil
	}

	// Execute mutation
	data, err := client.Execute(deleteIssuePrConnectionMutation, map[string]any{
		"input": map[string]any{
			"issueId":       issue.ID,
			"pullRequestId": pr.ID,
		},
	})
	if err != nil {
		return exitcode.General("disconnecting PR from issue", err)
	}

	var resp struct {
		DeleteIssuePrConnection struct {
			Issue struct {
				ID     string `json:"id"`
				Number int    `json:"number"`
				Title  string `json:"title"`
			} `json:"issue"`
			PullRequest struct {
				ID     string `json:"id"`
				Number int    `json:"number"`
				Title  string `json:"title"`
			} `json:"pullRequest"`
		} `json:"deleteIssuePrConnection"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return exitcode.General("parsing disconnect response", err)
	}

	// JSON output
	if output.IsJSON(outputFormat) {
		return output.JSON(w, map[string]any{
			"issue": formatConnectItemJSON(issue),
			"pr":    formatConnectItemJSON(pr),
		})
	}

	output.MutationSingle(w, output.Green(fmt.Sprintf(
		"Disconnected %s from %s.",
		pr.Ref(), issue.Ref(),
	)))
	return nil
}

// resolveForConnect resolves an issue/PR identifier and fetches its type (issue vs PR).
func resolveForConnect(client *api.Client, workspaceID, identifier, repoFlag string, ghClient *gh.Client) (*resolvedConnectItem, error) {
	parsed, parseErr := resolve.ParseIssueRef(identifier)

	// If it's a ZenHub ID, use the node query directly
	if parseErr == nil && parsed.ZenHubID != "" {
		return resolveConnectByNode(client, parsed.ZenHubID)
	}

	// Resolve to get repo GH ID and issue number
	result, err := resolve.Issue(client, workspaceID, identifier, &resolve.IssueOptions{
		RepoFlag:     repoFlag,
		GitHubClient: ghClient,
	})
	if err != nil {
		return nil, err
	}

	return resolveConnectByInfo(client, result.RepoGhID, result.Number)
}

func resolveConnectByInfo(client *api.Client, repoGhID, issueNumber int) (*resolvedConnectItem, error) {
	data, err := client.Execute(issueConnectResolveQuery, map[string]any{
		"repositoryGhId": repoGhID,
		"issueNumber":    issueNumber,
	})
	if err != nil {
		return nil, exitcode.General("fetching issue for connect", err)
	}

	var resp struct {
		IssueByInfo *struct {
			ID          string `json:"id"`
			Number      int    `json:"number"`
			Title       string `json:"title"`
			PullRequest bool   `json:"pullRequest"`
			Repository  struct {
				Name      string `json:"name"`
				OwnerName string `json:"ownerName"`
			} `json:"repository"`
		} `json:"issueByInfo"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, exitcode.General("parsing connect resolve response", err)
	}

	if resp.IssueByInfo == nil {
		return nil, exitcode.NotFoundError(fmt.Sprintf("issue #%d not found", issueNumber))
	}

	return &resolvedConnectItem{
		ID:          resp.IssueByInfo.ID,
		Number:      resp.IssueByInfo.Number,
		Title:       resp.IssueByInfo.Title,
		RepoName:    resp.IssueByInfo.Repository.Name,
		RepoOwner:   resp.IssueByInfo.Repository.OwnerName,
		PullRequest: resp.IssueByInfo.PullRequest,
	}, nil
}

func resolveConnectByNode(client *api.Client, nodeID string) (*resolvedConnectItem, error) {
	data, err := client.Execute(issueConnectResolveByNodeQuery, map[string]any{
		"id": nodeID,
	})
	if err != nil {
		return nil, exitcode.General("fetching issue for connect", err)
	}

	var resp struct {
		Node *struct {
			ID          string `json:"id"`
			Number      int    `json:"number"`
			Title       string `json:"title"`
			PullRequest bool   `json:"pullRequest"`
			Repository  struct {
				Name      string `json:"name"`
				OwnerName string `json:"ownerName"`
			} `json:"repository"`
		} `json:"node"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, exitcode.General("parsing connect resolve response", err)
	}

	if resp.Node == nil {
		return nil, exitcode.NotFoundError(fmt.Sprintf("issue %q not found", nodeID))
	}

	return &resolvedConnectItem{
		ID:          resp.Node.ID,
		Number:      resp.Node.Number,
		Title:       resp.Node.Title,
		RepoName:    resp.Node.Repository.Name,
		RepoOwner:   resp.Node.Repository.OwnerName,
		PullRequest: resp.Node.PullRequest,
	}, nil
}

func formatConnectItemJSON(item *resolvedConnectItem) map[string]any {
	return map[string]any{
		"id":          item.ID,
		"number":      item.Number,
		"repository":  fmt.Sprintf("%s/%s", item.RepoOwner, item.RepoName),
		"title":       item.Title,
		"pullRequest": item.PullRequest,
	}
}
