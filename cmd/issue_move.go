package cmd

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/dslh/zh/internal/api"
	"github.com/dslh/zh/internal/exitcode"
	"github.com/dslh/zh/internal/gh"
	"github.com/dslh/zh/internal/output"
	"github.com/dslh/zh/internal/resolve"
	"github.com/spf13/cobra"
)

// GraphQL queries and mutations for issue move

const pipelineIssueIDQuery = `query GetPipelineIssueId($issueId: ID!, $workspaceId: ID!) {
  node(id: $issueId) {
    ... on Issue {
      id
      number
      title
      repository {
        name
        ownerName
      }
      pipelineIssue(workspaceId: $workspaceId) {
        id
        pipeline {
          id
          name
        }
      }
    }
  }
}`

const moveIssueMutation = `mutation MoveIssue($input: MoveIssueInput!) {
  moveIssue(input: $input) {
    issue {
      id
      number
      title
      repository {
        name
        ownerName
      }
    }
    pipeline {
      id
      name
    }
  }
}`

const movePipelineIssuesMutation = `mutation MovePipelineIssues($input: MovePipelineIssuesInput!) {
  movePipelineIssues(input: $input) {
    pipeline {
      id
      name
    }
  }
}`

// resolvedMoveIssue holds the info needed to move a single issue.
type resolvedMoveIssue struct {
	IssueID         string
	PipelineIssueID string
	Number          int
	Title           string
	RepoName        string
	RepoOwner       string
	CurrentPipeline string
}

func (r *resolvedMoveIssue) Ref() string {
	return fmt.Sprintf("%s#%d", r.RepoName, r.Number)
}

// Commands

var issueMoveCmd = &cobra.Command{
	Use:   "move <issue>... <pipeline>",
	Short: "Move issues to a pipeline",
	Long: `Move one or more issues to a target pipeline. The last argument is the
pipeline name; all preceding arguments are issue identifiers.

Issues can be specified as repo#number, owner/repo#number, ZenHub IDs,
or bare numbers with --repo.

Examples:
  zh issue move task-tracker#1 "In Development"
  zh issue move task-tracker#1 task-tracker#2 Done
  zh issue move --repo=task-tracker 1 2 3 "In Development"
  zh issue move task-tracker#1 Done --position=top`,
	Args: cobra.MinimumNArgs(2),
	RunE: runIssueMove,
}

var (
	issueMovePosition        string
	issueMoveDryRun          bool
	issueMoveRepo            string
	issueMoveContinueOnError bool
)

func init() {
	issueMoveCmd.Flags().StringVar(&issueMovePosition, "position", "", "Position in target pipeline: top, bottom, or a number")
	issueMoveCmd.Flags().BoolVar(&issueMoveDryRun, "dry-run", false, "Show what would be moved without executing")
	issueMoveCmd.Flags().StringVar(&issueMoveRepo, "repo", "", "Repository context for bare issue numbers")
	issueMoveCmd.Flags().BoolVar(&issueMoveContinueOnError, "continue-on-error", false, "Continue processing remaining issues after an error")

	issueCmd.AddCommand(issueMoveCmd)
}

func resetIssueMoveFlags() {
	issueMovePosition = ""
	issueMoveDryRun = false
	issueMoveRepo = ""
	issueMoveContinueOnError = false
}

func runIssueMove(cmd *cobra.Command, args []string) error {
	cfg, err := requireWorkspace()
	if err != nil {
		return err
	}

	client := newClient(cfg, cmd)
	w := cmd.OutOrStdout()

	// Last arg is the pipeline, everything before is issue identifiers
	pipelineName := args[len(args)-1]
	issueArgs := args[:len(args)-1]

	// Resolve target pipeline
	targetPipeline, err := resolve.Pipeline(client, cfg.Workspace, pipelineName, cfg.Aliases.Pipelines)
	if err != nil {
		return err
	}

	// Resolve each issue and fetch its PipelineIssue ID
	ghClient := newGitHubClient(cfg, cmd)
	var resolved []resolvedMoveIssue
	var failed []output.FailedItem

	for _, arg := range issueArgs {
		issue, err := resolveForMove(client, cfg.Workspace, arg, ghClient)
		if err != nil {
			if issueMoveContinueOnError {
				failed = append(failed, output.FailedItem{
					Ref:    arg,
					Reason: err.Error(),
				})
				continue
			}
			return err
		}
		resolved = append(resolved, *issue)
	}

	if len(resolved) == 0 && len(failed) > 0 {
		return exitcode.Generalf("all issues failed to resolve")
	}

	// Parse position flag
	posType, posNum, err := parsePosition(issueMovePosition)
	if err != nil {
		return err
	}

	// Numeric position only works for single issues
	if posType == posNumeric && len(resolved) > 1 {
		return exitcode.Usage("numeric --position only works for a single issue")
	}

	// Dry run
	if issueMoveDryRun {
		items := make([]output.MutationItem, len(resolved))
		for i, r := range resolved {
			ctx := ""
			if r.CurrentPipeline != "" {
				ctx = fmt.Sprintf("(currently in %q)", r.CurrentPipeline)
			}
			items[i] = output.MutationItem{
				Ref:     r.Ref(),
				Title:   truncateTitle(r.Title),
				Context: ctx,
			}
		}

		header := fmt.Sprintf("Would move %d issue(s) to %q", len(resolved), targetPipeline.Name)
		switch posType {
		case posTop:
			header += " at top"
		case posBottom:
			header += " at bottom"
		case posNumeric:
			header += fmt.Sprintf(" at position %d", posNum)
		}

		output.MutationDryRun(w, header, items)

		if len(failed) > 0 {
			fmt.Fprintln(w)
			for _, f := range failed {
				fmt.Fprintf(w, "  %s  %s\n", f.Ref, output.Red(f.Reason))
			}
		}
		return nil
	}

	// Execute moves
	var succeeded []output.MutationItem
	for _, r := range resolved {
		err := executeMoveIssue(client, r, targetPipeline.ID, posType, posNum)
		if err != nil {
			if issueMoveContinueOnError {
				failed = append(failed, output.FailedItem{
					Ref:    r.Ref(),
					Reason: err.Error(),
				})
				continue
			}
			return err
		}
		succeeded = append(succeeded, output.MutationItem{
			Ref:   r.Ref(),
			Title: truncateTitle(r.Title),
		})
	}

	if output.IsJSON(outputFormat) {
		return output.JSON(w, map[string]any{
			"moved":    succeeded,
			"failed":   failed,
			"pipeline": targetPipeline.Name,
		})
	}

	// Render output
	header := output.Green(fmt.Sprintf("Moved %d issue(s) to %q.", len(succeeded), targetPipeline.Name))
	if len(failed) > 0 {
		output.MutationPartialFailure(w, header, succeeded, failed)
	} else if len(succeeded) == 1 {
		output.MutationSingle(w, output.Green(fmt.Sprintf("Moved %s to %q.", succeeded[0].Ref, targetPipeline.Name)))
	} else {
		output.MutationBatch(w, header, succeeded)
	}

	return nil
}

// resolveForMove resolves an issue identifier and fetches its PipelineIssue ID.
func resolveForMove(client *api.Client, workspaceID, identifier string, ghClient *gh.Client) (*resolvedMoveIssue, error) {
	// Resolve the issue
	result, err := resolve.Issue(client, workspaceID, identifier, &resolve.IssueOptions{
		RepoFlag:     issueMoveRepo,
		GitHubClient: ghClient,
	})
	if err != nil {
		return nil, err
	}

	// Fetch the PipelineIssue ID for this issue
	data, err := client.Execute(pipelineIssueIDQuery, map[string]any{
		"issueId":     result.ID,
		"workspaceId": workspaceID,
	})
	if err != nil {
		return nil, exitcode.General("fetching pipeline issue ID", err)
	}

	var resp struct {
		Node *struct {
			ID         string `json:"id"`
			Number     int    `json:"number"`
			Title      string `json:"title"`
			Repository struct {
				Name      string `json:"name"`
				OwnerName string `json:"ownerName"`
			} `json:"repository"`
			PipelineIssue *struct {
				ID       string `json:"id"`
				Pipeline struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				} `json:"pipeline"`
			} `json:"pipelineIssue"`
		} `json:"node"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, exitcode.General("parsing pipeline issue response", err)
	}

	if resp.Node == nil {
		return nil, exitcode.NotFoundError(fmt.Sprintf("issue %q not found", identifier))
	}

	resolved := &resolvedMoveIssue{
		IssueID:   result.ID,
		Number:    resp.Node.Number,
		Title:     resp.Node.Title,
		RepoName:  resp.Node.Repository.Name,
		RepoOwner: resp.Node.Repository.OwnerName,
	}

	if resp.Node.PipelineIssue != nil {
		resolved.PipelineIssueID = resp.Node.PipelineIssue.ID
		resolved.CurrentPipeline = resp.Node.PipelineIssue.Pipeline.Name
	}

	return resolved, nil
}

type positionType int

const (
	posDefault positionType = iota
	posTop
	posBottom
	posNumeric
)

func parsePosition(s string) (positionType, int, error) {
	if s == "" {
		return posDefault, 0, nil
	}

	lower := strings.ToLower(s)
	switch lower {
	case "top":
		return posTop, 0, nil
	case "bottom":
		return posBottom, 0, nil
	}

	n, err := strconv.Atoi(s)
	if err != nil || n < 0 {
		return posDefault, 0, exitcode.Usage(fmt.Sprintf("invalid position %q — expected top, bottom, or a non-negative number", s))
	}
	return posNumeric, n, nil
}

// executeMoveIssue performs the actual move API call for a single issue.
func executeMoveIssue(client *api.Client, issue resolvedMoveIssue, targetPipelineID string, posType positionType, posNum int) error {
	// Use moveIssue for numeric position, moveIssueRelativeTo for symbolic
	if posType == posNumeric {
		input := map[string]any{
			"issueId":    issue.IssueID,
			"pipelineId": targetPipelineID,
			"position":   posNum,
		}

		_, err := client.Execute(moveIssueMutation, map[string]any{"input": input})
		if err != nil {
			return exitcode.General(fmt.Sprintf("moving %s", issue.Ref()), err)
		}
		return nil
	}

	// Use movePipelineIssues for top/bottom/default
	input := map[string]any{
		"pipelineId":       targetPipelineID,
		"pipelineIssueIds": []string{issue.PipelineIssueID},
	}

	switch posType {
	case posTop:
		input["position"] = "START"
	default:
		input["position"] = "END"
	}

	_, err := client.Execute(movePipelineIssuesMutation, map[string]any{"input": input})
	if err != nil {
		return exitcode.General(fmt.Sprintf("moving %s", issue.Ref()), err)
	}
	return nil
}

func truncateTitle(title string) string {
	if len(title) > 50 {
		return title[:47] + "..."
	}
	return title
}
