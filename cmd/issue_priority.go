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

// GraphQL queries and mutations for issue priority

const issuePriorityResolveQuery = `query GetIssueForPriority($issueId: ID!, $workspaceId: ID!) {
  node(id: $issueId) {
    ... on Issue {
      id
      number
      title
      repository {
        name
        ownerName
        ghId
      }
      pipelineIssue(workspaceId: $workspaceId) {
        priority {
          id
          name
        }
      }
    }
  }
}`

const setIssuePrioritiesMutation = `mutation SetIssuePriority($input: SetIssueInfoPrioritiesInput!) {
  setIssueInfoPriorities(input: $input) {
    pipelineIssues {
      id
      priority {
        id
        name
        color
      }
      issue {
        id
        number
        title
        repository {
          name
          ownerName
        }
      }
    }
  }
}`

const removeIssuePrioritiesMutation = `mutation RemoveIssuePriority($input: RemoveIssueInfoPrioritiesInput!) {
  removeIssueInfoPriorities(input: $input) {
    pipelineIssues {
      id
      priority {
        id
        name
      }
      issue {
        id
        number
        title
        repository {
          name
          ownerName
        }
      }
    }
  }
}`

// resolvedPriorityIssue holds the info needed to set/clear priority on a single issue.
type resolvedPriorityIssue struct {
	IssueID         string
	Number          int
	Title           string
	RepoName        string
	RepoOwner       string
	RepoGhID        int
	CurrentPriority string // empty if none
}

func (r *resolvedPriorityIssue) Ref() string {
	return fmt.Sprintf("%s#%d", r.RepoName, r.Number)
}

// Commands

var issuePriorityCmd = &cobra.Command{
	Use:   "priority <issue>... [priority]",
	Short: "Set or clear the priority on issues",
	Long: `Set or clear the priority on one or more issues.

Provide a priority name as the last argument to set it. Omit the priority
to clear it from all specified issues.

Issues can be specified as repo#number, owner/repo#number, ZenHub IDs,
or bare numbers with --repo.

Examples:
  zh issue priority task-tracker#1 "High priority"
  zh issue priority task-tracker#1 high
  zh issue priority task-tracker#1 task-tracker#2 urgent
  zh issue priority --repo=task-tracker 1 2 3 high
  zh issue priority task-tracker#1                    # clears priority`,
	Args: cobra.MinimumNArgs(1),
	RunE: runIssuePriority,
}

var (
	issuePriorityDryRun          bool
	issuePriorityRepo            string
	issuePriorityContinueOnError bool
	issuePriorityClear           bool
)

func init() {
	issuePriorityCmd.Flags().BoolVar(&issuePriorityDryRun, "dry-run", false, "Show what would be changed without executing")
	issuePriorityCmd.Flags().StringVar(&issuePriorityRepo, "repo", "", "Repository context for bare issue numbers")
	issuePriorityCmd.Flags().BoolVar(&issuePriorityContinueOnError, "continue-on-error", false, "Continue processing remaining issues after a resolution error")
	issuePriorityCmd.Flags().BoolVar(&issuePriorityClear, "clear", false, "Clear priority from specified issues")

	issueCmd.AddCommand(issuePriorityCmd)
}

func resetIssuePriorityFlags() {
	issuePriorityDryRun = false
	issuePriorityRepo = ""
	issuePriorityContinueOnError = false
	issuePriorityClear = false
}

func runIssuePriority(cmd *cobra.Command, args []string) error {
	cfg, err := requireWorkspace()
	if err != nil {
		return err
	}

	client := newClient(cfg, cmd)
	w := cmd.OutOrStdout()
	ghClient := newGitHubClient(cfg, cmd)

	// Determine if we're setting or clearing priority.
	// If --clear is set, all args are issue identifiers.
	// Otherwise, try to resolve the last argument as a priority.
	// If it resolves, the rest are issue identifiers.
	// If it doesn't and there's only 1 arg, we clear.
	var priority *resolve.PriorityResult
	var issueArgs []string

	if issuePriorityClear {
		issueArgs = args
	} else {
		// Try the last arg as a priority
		lastArg := args[len(args)-1]
		p, err := resolve.Priority(client, cfg.Workspace, lastArg)
		if err == nil {
			priority = p
			issueArgs = args[:len(args)-1]
			if len(issueArgs) == 0 {
				return exitcode.Usage("at least one issue identifier is required")
			}
		} else {
			// Last arg is not a priority — treat all args as issues (clear mode)
			issueArgs = args
		}
	}

	// Resolve each issue identifier
	var resolved []resolvedPriorityIssue
	var resolveFailed []output.FailedItem

	for _, arg := range issueArgs {
		issue, err := resolveForPriority(client, cfg.Workspace, arg, ghClient)
		if err != nil {
			if issuePriorityContinueOnError {
				resolveFailed = append(resolveFailed, output.FailedItem{
					Ref:    arg,
					Reason: err.Error(),
				})
				continue
			}
			return err
		}
		resolved = append(resolved, *issue)
	}

	if len(resolved) == 0 && len(resolveFailed) > 0 {
		return exitcode.Generalf("all issues failed to resolve")
	}

	// Dry run
	if issuePriorityDryRun {
		return renderPriorityDryRun(w, resolved, resolveFailed, priority)
	}

	// Execute mutation
	if priority != nil {
		err = executeSetPriority(client, resolved, priority.ID)
	} else {
		err = executeClearPriority(client, cfg.Workspace, resolved)
	}
	if err != nil {
		return err
	}

	// JSON output
	if output.IsJSON(outputFormat) {
		jsonResp := map[string]any{
			"issues": formatPriorityItemsJSON(resolved),
			"failed": resolveFailed,
		}
		if priority != nil {
			jsonResp["priority"] = map[string]any{
				"id":   priority.ID,
				"name": priority.Name,
			}
		} else {
			jsonResp["priority"] = nil
		}
		return output.JSON(w, jsonResp)
	}

	// Render output
	succeeded := make([]output.MutationItem, len(resolved))
	for i, r := range resolved {
		succeeded[i] = output.MutationItem{
			Ref:   r.Ref(),
			Title: truncateTitle(r.Title),
		}
	}

	totalAttempted := len(succeeded) + len(resolveFailed)
	if priority != nil {
		if len(resolveFailed) > 0 {
			header := output.Green(fmt.Sprintf("Set priority %q on %d of %d issue(s).", priority.Name, len(succeeded), totalAttempted))
			output.MutationPartialFailure(w, header, succeeded, resolveFailed)
		} else if len(succeeded) == 1 {
			output.MutationSingle(w, output.Green(fmt.Sprintf(
				"Set priority %q on %s.", priority.Name, succeeded[0].Ref,
			)))
		} else {
			header := output.Green(fmt.Sprintf("Set priority %q on %d issue(s).", priority.Name, len(succeeded)))
			output.MutationBatch(w, header, succeeded)
		}
	} else {
		if len(resolveFailed) > 0 {
			header := output.Green(fmt.Sprintf("Cleared priority from %d of %d issue(s).", len(succeeded), totalAttempted))
			output.MutationPartialFailure(w, header, succeeded, resolveFailed)
		} else if len(succeeded) == 1 {
			output.MutationSingle(w, output.Green(fmt.Sprintf(
				"Cleared priority from %s.", succeeded[0].Ref,
			)))
		} else {
			header := output.Green(fmt.Sprintf("Cleared priority from %d issue(s).", len(succeeded)))
			output.MutationBatch(w, header, succeeded)
		}
	}

	if len(resolveFailed) > 0 {
		return exitcode.Generalf("some issues failed to resolve")
	}

	return nil
}

// resolveForPriority resolves an issue identifier and fetches its current priority.
func resolveForPriority(client *api.Client, workspaceID, identifier string, ghClient *gh.Client) (*resolvedPriorityIssue, error) {
	result, err := resolve.Issue(client, workspaceID, identifier, &resolve.IssueOptions{
		RepoFlag:     issuePriorityRepo,
		GitHubClient: ghClient,
	})
	if err != nil {
		return nil, err
	}

	// Fetch current priority
	data, err := client.Execute(issuePriorityResolveQuery, map[string]any{
		"issueId":     result.ID,
		"workspaceId": workspaceID,
	})
	if err != nil {
		return nil, exitcode.General("fetching issue priority", err)
	}

	var resp struct {
		Node *struct {
			ID         string `json:"id"`
			Number     int    `json:"number"`
			Title      string `json:"title"`
			Repository struct {
				Name      string `json:"name"`
				OwnerName string `json:"ownerName"`
				GhID      int    `json:"ghId"`
			} `json:"repository"`
			PipelineIssue *struct {
				Priority *struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				} `json:"priority"`
			} `json:"pipelineIssue"`
		} `json:"node"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, exitcode.General("parsing issue priority response", err)
	}

	if resp.Node == nil {
		return nil, exitcode.NotFoundError(fmt.Sprintf("issue %q not found", identifier))
	}

	resolved := &resolvedPriorityIssue{
		IssueID:   resp.Node.ID,
		Number:    resp.Node.Number,
		Title:     resp.Node.Title,
		RepoName:  resp.Node.Repository.Name,
		RepoOwner: resp.Node.Repository.OwnerName,
		RepoGhID:  resp.Node.Repository.GhID,
	}

	if resp.Node.PipelineIssue != nil && resp.Node.PipelineIssue.Priority != nil {
		resolved.CurrentPriority = resp.Node.PipelineIssue.Priority.Name
	}

	return resolved, nil
}

func executeSetPriority(client *api.Client, issues []resolvedPriorityIssue, priorityID string) error {
	issueInfos := make([]map[string]any, len(issues))
	for i, iss := range issues {
		issueInfos[i] = map[string]any{
			"repositoryGhId": iss.RepoGhID,
			"issueNumber":    iss.Number,
		}
	}

	_, err := client.Execute(setIssuePrioritiesMutation, map[string]any{
		"input": map[string]any{
			"priorityId": priorityID,
			"issues":     issueInfos,
		},
	})
	if err != nil {
		return exitcode.General("setting priority", err)
	}

	return nil
}

func executeClearPriority(client *api.Client, workspaceID string, issues []resolvedPriorityIssue) error {
	issueInfos := make([]map[string]any, len(issues))
	for i, iss := range issues {
		issueInfos[i] = map[string]any{
			"repositoryGhId": iss.RepoGhID,
			"issueNumber":    iss.Number,
		}
	}

	_, err := client.Execute(removeIssuePrioritiesMutation, map[string]any{
		"input": map[string]any{
			"workspaceId": workspaceID,
			"issues":      issueInfos,
		},
	})
	if err != nil {
		return exitcode.General("clearing priority", err)
	}

	return nil
}

func renderPriorityDryRun(w writerFlusher, resolved []resolvedPriorityIssue, resolveFailed []output.FailedItem, priority *resolve.PriorityResult) error {
	items := make([]output.MutationItem, len(resolved))
	for i, r := range resolved {
		ctx := "(no priority)"
		if r.CurrentPriority != "" {
			ctx = fmt.Sprintf("(currently: %s)", r.CurrentPriority)
		}
		items[i] = output.MutationItem{
			Ref:     r.Ref(),
			Title:   truncateTitle(r.Title),
			Context: ctx,
		}
	}

	var header string
	if priority != nil {
		header = fmt.Sprintf("Would set priority %q on %d issue(s)", priority.Name, len(resolved))
	} else {
		header = fmt.Sprintf("Would clear priority from %d issue(s)", len(resolved))
	}

	output.MutationDryRun(w, header, items)

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

func formatPriorityItemsJSON(items []resolvedPriorityIssue) []map[string]any {
	result := make([]map[string]any, len(items))
	for i, item := range items {
		result[i] = map[string]any{
			"id":               item.IssueID,
			"number":           item.Number,
			"repository":       fmt.Sprintf("%s/%s", item.RepoOwner, item.RepoName),
			"title":            item.Title,
			"previousPriority": priorityOrNil(item.CurrentPriority),
		}
	}
	return result
}

func priorityOrNil(p string) any {
	if p == "" {
		return nil
	}
	return p
}
