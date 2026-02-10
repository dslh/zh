package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/dslh/zh/internal/exitcode"
	"github.com/dslh/zh/internal/output"
	"github.com/dslh/zh/internal/resolve"
	"github.com/spf13/cobra"
)

// GraphQL mutations for issue reopen

const reopenIssuesMutation = `mutation ReopenIssues($input: ReopenIssuesInput!) {
  reopenIssues(input: $input) {
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

// Commands

var issueReopenCmd = &cobra.Command{
	Use:   "reopen <issue>... --pipeline=<name>",
	Short: "Reopen closed issues into a pipeline",
	Long: `Reopen one or more closed issues and place them into a pipeline.

The --pipeline flag is required — it specifies which pipeline the
reopened issues will be placed in. Issues are also reopened on GitHub.

Issues can be specified as repo#number, owner/repo#number, ZenHub IDs,
or bare numbers with --repo.

Examples:
  zh issue reopen task-tracker#1 --pipeline=Todo
  zh issue reopen task-tracker#1 task-tracker#2 --pipeline=Backlog
  zh issue reopen --repo=task-tracker 1 2 --pipeline=Todo --position=top`,
	Args: cobra.MinimumNArgs(1),
	RunE: runIssueReopen,
}

var (
	issueReopenPipeline        string
	issueReopenPosition        string
	issueReopenDryRun          bool
	issueReopenRepo            string
	issueReopenContinueOnError bool
)

func init() {
	issueReopenCmd.Flags().StringVar(&issueReopenPipeline, "pipeline", "", "Target pipeline for reopened issues (required)")
	_ = issueReopenCmd.MarkFlagRequired("pipeline")
	issueReopenCmd.Flags().StringVar(&issueReopenPosition, "position", "", "Position in pipeline: top or bottom (default: bottom)")
	issueReopenCmd.Flags().BoolVar(&issueReopenDryRun, "dry-run", false, "Show what would be reopened without executing")
	issueReopenCmd.Flags().StringVar(&issueReopenRepo, "repo", "", "Repository context for bare issue numbers")
	issueReopenCmd.Flags().BoolVar(&issueReopenContinueOnError, "continue-on-error", false, "Continue processing remaining issues after a resolution error")

	issueCmd.AddCommand(issueReopenCmd)
}

func resetIssueReopenFlags() {
	issueReopenPipeline = ""
	issueReopenPosition = ""
	issueReopenDryRun = false
	issueReopenRepo = ""
	issueReopenContinueOnError = false
}

func runIssueReopen(cmd *cobra.Command, args []string) error {
	cfg, err := requireWorkspace()
	if err != nil {
		return err
	}

	client := newClient(cfg, cmd)
	w := cmd.OutOrStdout()
	ghClient := newGitHubClient(cfg, cmd)

	// Resolve target pipeline
	targetPipeline, err := resolve.Pipeline(client, cfg.Workspace, issueReopenPipeline, cfg.Aliases.Pipelines)
	if err != nil {
		return err
	}

	// Parse position flag (only top/bottom allowed for reopen)
	position := "END" // default: bottom
	if issueReopenPosition != "" {
		switch strings.ToLower(issueReopenPosition) {
		case "top":
			position = "START"
		case "bottom":
			position = "END"
		default:
			return exitcode.Usage(fmt.Sprintf("invalid position %q — expected top or bottom", issueReopenPosition))
		}
	}

	// Resolve each issue identifier
	var resolved []resolvedCloseIssue // reuse the same struct
	var resolveFailed []output.FailedItem
	var alreadyOpen []resolvedCloseIssue

	for _, arg := range args {
		issue, err := resolveForClose(client, cfg.Workspace, arg, issueReopenRepo, ghClient)
		if err != nil {
			if issueReopenContinueOnError {
				resolveFailed = append(resolveFailed, output.FailedItem{
					Ref:    arg,
					Reason: err.Error(),
				})
				continue
			}
			return err
		}

		if strings.EqualFold(issue.State, "OPEN") {
			alreadyOpen = append(alreadyOpen, *issue)
			continue
		}

		resolved = append(resolved, *issue)
	}

	if len(resolved) == 0 && len(alreadyOpen) == 0 && len(resolveFailed) > 0 {
		return exitcode.Generalf("all issues failed to resolve")
	}

	// Dry run
	if issueReopenDryRun {
		if output.IsJSON(outputFormat) {
			return renderReopenDryRunJSON(w, resolved, alreadyOpen, resolveFailed, targetPipeline, position)
		}
		return renderReopenDryRun(w, resolved, alreadyOpen, resolveFailed, targetPipeline.Name, position)
	}

	if len(resolved) == 0 {
		if output.IsJSON(outputFormat) {
			return output.JSON(w, map[string]any{
				"reopened":     []any{},
				"pipeline":     map[string]any{"id": targetPipeline.ID, "name": targetPipeline.Name},
				"position":     position,
				"failed":       resolveFailed,
				"alreadyOpen":  formatCloseItemsJSON(alreadyOpen),
				"successCount": 0,
			})
		}
		if len(alreadyOpen) > 0 {
			renderAlreadyOpen(w, alreadyOpen)
		}
		return nil
	}

	// Execute the mutation
	issueIDs := make([]string, len(resolved))
	for i, r := range resolved {
		issueIDs[i] = r.IssueID
	}

	data, err := client.Execute(reopenIssuesMutation, map[string]any{
		"input": map[string]any{
			"issueIds":   issueIDs,
			"pipelineId": targetPipeline.ID,
			"position":   position,
		},
	})
	if err != nil {
		return exitcode.General("reopening issues", err)
	}

	var resp struct {
		ReopenIssues struct {
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
		} `json:"reopenIssues"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return exitcode.General("parsing reopen response", err)
	}

	// Build succeeded/failed lists from mutation response
	failedIDs := make(map[string]bool)
	var mutationFailed []output.FailedItem
	for _, f := range resp.ReopenIssues.FailedIssues {
		failedIDs[f.ID] = true
		ref := fmt.Sprintf("%s#%d", f.Repository.Name, f.Number)
		mutationFailed = append(mutationFailed, output.FailedItem{
			Ref:    ref,
			Reason: "failed to reopen",
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
			"reopened":     formatMutationItemsJSON(succeeded),
			"pipeline":     map[string]any{"id": targetPipeline.ID, "name": targetPipeline.Name},
			"position":     position,
			"failed":       allFailed,
			"alreadyOpen":  formatCloseItemsJSON(alreadyOpen),
			"successCount": resp.ReopenIssues.SuccessCount,
		})
	}

	// Render output
	if len(alreadyOpen) > 0 {
		renderAlreadyOpen(w, alreadyOpen)
		fmt.Fprintln(w)
	}

	totalAttempted := len(succeeded) + len(allFailed)
	if len(allFailed) > 0 {
		header := output.Green(fmt.Sprintf("Reopened %d of %d issue(s) into %q.", len(succeeded), totalAttempted, targetPipeline.Name))
		output.MutationPartialFailure(w, header, succeeded, allFailed)
	} else if len(succeeded) == 1 {
		output.MutationSingle(w, output.Green(fmt.Sprintf("Reopened %s into %q: %s", succeeded[0].Ref, targetPipeline.Name, succeeded[0].Title)))
	} else {
		header := output.Green(fmt.Sprintf("Reopened %d issue(s) into %q.", len(succeeded), targetPipeline.Name))
		output.MutationBatch(w, header, succeeded)
	}

	if len(allFailed) > 0 {
		return exitcode.Generalf("some issues failed to reopen")
	}

	return nil
}

func renderReopenDryRunJSON(w writerFlusher, resolved []resolvedCloseIssue, alreadyOpen []resolvedCloseIssue, resolveFailed []output.FailedItem, pipeline *resolve.PipelineResult, position string) error {
	wouldReopen := make([]map[string]any, len(resolved))
	for i, r := range resolved {
		wouldReopen[i] = map[string]any{
			"ref":   r.Ref(),
			"title": r.Title,
			"state": strings.ToLower(r.State),
		}
	}
	return output.JSON(w, map[string]any{
		"dryRun":      true,
		"wouldReopen": wouldReopen,
		"pipeline":    map[string]any{"id": pipeline.ID, "name": pipeline.Name},
		"position":    position,
		"alreadyOpen": formatCloseItemsJSON(alreadyOpen),
		"failed":      resolveFailed,
	})
}

func renderReopenDryRun(w writerFlusher, resolved []resolvedCloseIssue, alreadyOpen []resolvedCloseIssue, resolveFailed []output.FailedItem, pipelineName, position string) error {
	if len(alreadyOpen) > 0 {
		renderAlreadyOpen(w, alreadyOpen)
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
				Context: "(closed)",
			}
		}

		posLabel := "bottom"
		if position == "START" {
			posLabel = "top"
		}
		header := fmt.Sprintf("Would reopen %d issue(s) into %q at %s", len(resolved), pipelineName, posLabel)
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

func renderAlreadyOpen(w writerFlusher, alreadyOpen []resolvedCloseIssue) {
	noun := "issue"
	if len(alreadyOpen) != 1 {
		noun = "issues"
	}
	fmt.Fprintf(w, "%d %s already open:\n\n", len(alreadyOpen), noun)
	for _, r := range alreadyOpen {
		fmt.Fprintf(w, "  %s %s\n", r.Ref(), truncateTitle(r.Title))
	}
}
