package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/dslh/zh/internal/api"
	"github.com/dslh/zh/internal/exitcode"
	"github.com/dslh/zh/internal/gh"
	"github.com/dslh/zh/internal/output"
	"github.com/dslh/zh/internal/resolve"
	"github.com/spf13/cobra"
)

// GraphQL queries and mutations for issue close

const issueCloseResolveQuery = `query GetIssueForClose($issueId: ID!) {
  node(id: $issueId) {
    ... on Issue {
      id
      number
      title
      state
      repository {
        name
        ownerName
      }
    }
  }
}`

const closeIssuesMutation = `mutation CloseIssues($input: CloseIssuesInput!) {
  closeIssues(input: $input) {
    successCount
    failedIssues {
      id
      number
      title
      repository {
        name
        ownerName
      }
    }
    githubErrors
  }
}`

// resolvedCloseIssue holds the info needed to close a single issue.
type resolvedCloseIssue struct {
	IssueID   string
	Number    int
	Title     string
	RepoName  string
	RepoOwner string
	State     string
}

func (r *resolvedCloseIssue) Ref() string {
	return fmt.Sprintf("%s#%d", r.RepoName, r.Number)
}

// Commands

var issueCloseCmd = &cobra.Command{
	Use:   "close <issue>...",
	Short: "Close one or more issues",
	Long: `Close one or more issues. The issues are closed on both ZenHub and GitHub.

Issues can be specified as repo#number, owner/repo#number, ZenHub IDs,
or bare numbers with --repo.

Examples:
  zh issue close task-tracker#1
  zh issue close task-tracker#1 task-tracker#2
  zh issue close --repo=task-tracker 1 2 3`,
	Args: cobra.MinimumNArgs(1),
	RunE: runIssueClose,
}

var (
	issueCloseDryRun          bool
	issueCloseRepo            string
	issueCloseContinueOnError bool
)

func init() {
	issueCloseCmd.Flags().BoolVar(&issueCloseDryRun, "dry-run", false, "Show what would be closed without executing")
	issueCloseCmd.Flags().StringVar(&issueCloseRepo, "repo", "", "Repository context for bare issue numbers")
	issueCloseCmd.Flags().BoolVar(&issueCloseContinueOnError, "continue-on-error", false, "Continue processing remaining issues after a resolution error")

	issueCmd.AddCommand(issueCloseCmd)
}

func resetIssueCloseFlags() {
	issueCloseDryRun = false
	issueCloseRepo = ""
	issueCloseContinueOnError = false
}

func runIssueClose(cmd *cobra.Command, args []string) error {
	cfg, err := requireWorkspace()
	if err != nil {
		return err
	}

	client := newClient(cfg, cmd)
	w := cmd.OutOrStdout()
	ghClient := newGitHubClient(cfg, cmd)

	// Resolve each issue identifier
	var resolved []resolvedCloseIssue
	var resolveFailed []output.FailedItem
	var alreadyClosed []resolvedCloseIssue

	for _, arg := range args {
		issue, err := resolveForClose(client, cfg.Workspace, arg, issueCloseRepo, ghClient)
		if err != nil {
			if issueCloseContinueOnError {
				resolveFailed = append(resolveFailed, output.FailedItem{
					Ref:    arg,
					Reason: err.Error(),
				})
				continue
			}
			return err
		}

		if strings.EqualFold(issue.State, "CLOSED") {
			alreadyClosed = append(alreadyClosed, *issue)
			continue
		}

		resolved = append(resolved, *issue)
	}

	if len(resolved) == 0 && len(alreadyClosed) == 0 && len(resolveFailed) > 0 {
		return exitcode.Generalf("all issues failed to resolve")
	}

	// Dry run
	if issueCloseDryRun {
		if output.IsJSON(outputFormat) {
			return renderCloseDryRunJSON(w, resolved, alreadyClosed, resolveFailed)
		}
		return renderCloseDryRun(w, resolved, alreadyClosed, resolveFailed)
	}

	if len(resolved) == 0 {
		// Only already-closed issues
		if output.IsJSON(outputFormat) {
			return output.JSON(w, map[string]any{
				"closed":        []any{},
				"failed":        resolveFailed,
				"alreadyClosed": formatCloseItemsJSON(alreadyClosed),
				"successCount":  0,
			})
		}
		if len(alreadyClosed) > 0 {
			renderAlreadyClosed(w, alreadyClosed)
		}
		return nil
	}

	// Execute the mutation
	issueIDs := make([]string, len(resolved))
	for i, r := range resolved {
		issueIDs[i] = r.IssueID
	}

	data, err := client.Execute(closeIssuesMutation, map[string]any{
		"input": map[string]any{
			"issueIds": issueIDs,
		},
	})
	if err != nil {
		return exitcode.General("closing issues", err)
	}

	var resp struct {
		CloseIssues struct {
			SuccessCount int `json:"successCount"`
			FailedIssues []struct {
				ID         string `json:"id"`
				Number     int    `json:"number"`
				Title      string `json:"title"`
				Repository struct {
					Name      string `json:"name"`
					OwnerName string `json:"ownerName"`
				} `json:"repository"`
			} `json:"failedIssues"`
			GithubErrors json.RawMessage `json:"githubErrors"`
		} `json:"closeIssues"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return exitcode.General("parsing close response", err)
	}

	// Build succeeded/failed lists from mutation response
	failedIDs := make(map[string]bool)
	var mutationFailed []output.FailedItem
	for _, f := range resp.CloseIssues.FailedIssues {
		failedIDs[f.ID] = true
		ref := fmt.Sprintf("%s#%d", f.Repository.Name, f.Number)
		mutationFailed = append(mutationFailed, output.FailedItem{
			Ref:    ref,
			Reason: "failed to close",
		})
	}

	var succeeded []output.MutationItem
	for _, r := range resolved {
		if failedIDs[r.IssueID] {
			continue
		}
		succeeded = append(succeeded, output.MutationItem{
			Ref:   r.Ref(),
			Title: truncateTitle(r.Title),
		})
	}

	allFailed := append(resolveFailed, mutationFailed...)

	// JSON output
	if output.IsJSON(outputFormat) {
		return output.JSON(w, map[string]any{
			"closed":        formatMutationItemsJSON(succeeded),
			"failed":        allFailed,
			"alreadyClosed": formatCloseItemsJSON(alreadyClosed),
			"successCount":  resp.CloseIssues.SuccessCount,
		})
	}

	// Render output
	if len(alreadyClosed) > 0 {
		renderAlreadyClosed(w, alreadyClosed)
		fmt.Fprintln(w)
	}

	totalAttempted := len(succeeded) + len(allFailed)
	if len(allFailed) > 0 {
		header := output.Green(fmt.Sprintf("Closed %d of %d issue(s).", len(succeeded), totalAttempted))
		output.MutationPartialFailure(w, header, succeeded, allFailed)
	} else if len(succeeded) == 1 {
		output.MutationSingle(w, output.Green(fmt.Sprintf("Closed %s: %s", succeeded[0].Ref, succeeded[0].Title)))
	} else {
		header := output.Green(fmt.Sprintf("Closed %d issue(s).", len(succeeded)))
		output.MutationBatch(w, header, succeeded)
	}

	if len(allFailed) > 0 {
		return exitcode.Generalf("some issues failed to close")
	}

	return nil
}

// resolveForClose resolves an issue identifier and fetches its state.
func resolveForClose(client *api.Client, workspaceID, identifier, repoFlag string, ghClient *gh.Client) (*resolvedCloseIssue, error) {
	result, err := resolve.Issue(client, workspaceID, identifier, &resolve.IssueOptions{
		RepoFlag:     repoFlag,
		GitHubClient: ghClient,
	})
	if err != nil {
		return nil, err
	}

	// Fetch current state
	data, err := client.Execute(issueCloseResolveQuery, map[string]any{
		"issueId": result.ID,
	})
	if err != nil {
		return nil, exitcode.General("fetching issue state", err)
	}

	var resp struct {
		Node *struct {
			ID         string `json:"id"`
			Number     int    `json:"number"`
			Title      string `json:"title"`
			State      string `json:"state"`
			Repository struct {
				Name      string `json:"name"`
				OwnerName string `json:"ownerName"`
			} `json:"repository"`
		} `json:"node"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, exitcode.General("parsing issue state", err)
	}

	if resp.Node == nil {
		return nil, exitcode.NotFoundError(fmt.Sprintf("issue %q not found", identifier))
	}

	return &resolvedCloseIssue{
		IssueID:   resp.Node.ID,
		Number:    resp.Node.Number,
		Title:     resp.Node.Title,
		RepoName:  resp.Node.Repository.Name,
		RepoOwner: resp.Node.Repository.OwnerName,
		State:     resp.Node.State,
	}, nil
}

func renderCloseDryRunJSON(w writerFlusher, resolved []resolvedCloseIssue, alreadyClosed []resolvedCloseIssue, resolveFailed []output.FailedItem) error {
	wouldClose := make([]map[string]any, len(resolved))
	for i, r := range resolved {
		wouldClose[i] = map[string]any{
			"ref":   r.Ref(),
			"title": r.Title,
			"state": strings.ToLower(r.State),
		}
	}
	return output.JSON(w, map[string]any{
		"dryRun":        true,
		"wouldClose":    wouldClose,
		"alreadyClosed": formatCloseItemsJSON(alreadyClosed),
		"failed":        resolveFailed,
	})
}

func renderCloseDryRun(w writerFlusher, resolved []resolvedCloseIssue, alreadyClosed []resolvedCloseIssue, resolveFailed []output.FailedItem) error {
	if len(alreadyClosed) > 0 {
		renderAlreadyClosed(w, alreadyClosed)
		if len(resolved) > 0 {
			fmt.Fprintln(w)
		}
	}

	if len(resolved) > 0 {
		items := make([]output.MutationItem, len(resolved))
		for i, r := range resolved {
			items[i] = output.MutationItem{
				Ref:     r.Ref(),
				Title:   truncateTitle(r.Title),
				Context: "(open)",
			}
		}
		header := fmt.Sprintf("Would close %d issue(s)", len(resolved))
		output.MutationDryRun(w, header, items)
	}

	if len(resolveFailed) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, output.Red("Failed to resolve:"))
		fmt.Fprintln(w)
		for _, f := range resolveFailed {
			fmt.Fprintf(w, "  %s  %s\n", f.Ref, output.Red(f.Reason))
		}
	}

	return nil
}

func renderAlreadyClosed(w writerFlusher, alreadyClosed []resolvedCloseIssue) {
	noun := "issue"
	if len(alreadyClosed) != 1 {
		noun = "issues"
	}
	fmt.Fprintf(w, "%d %s already closed:\n\n", len(alreadyClosed), noun)
	for _, r := range alreadyClosed {
		fmt.Fprintf(w, "  %s %s\n", r.Ref(), truncateTitle(r.Title))
	}
}

func formatCloseItemsJSON(items []resolvedCloseIssue) []map[string]any {
	result := make([]map[string]any, len(items))
	for i, item := range items {
		result[i] = map[string]any{
			"id":         item.IssueID,
			"number":     item.Number,
			"repository": fmt.Sprintf("%s/%s", item.RepoOwner, item.RepoName),
			"title":      item.Title,
		}
	}
	return result
}

func formatMutationItemsJSON(items []output.MutationItem) []map[string]any {
	result := make([]map[string]any, len(items))
	for i, item := range items {
		result[i] = map[string]any{
			"ref":   item.Ref,
			"title": item.Title,
		}
	}
	return result
}
