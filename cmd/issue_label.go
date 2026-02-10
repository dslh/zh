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

// GraphQL mutations for issue label

const addLabelsToIssuesMutation = `mutation AddLabelsToIssues($input: AddLabelsToIssuesInput!) {
  addLabelsToIssues(input: $input) {
    successCount
    failedIssues {
      id
      number
      title
    }
    labels {
      id
      name
      color
    }
    githubErrors
  }
}`

const removeLabelsFromIssuesMutation = `mutation RemoveLabelsFromIssues($input: RemoveLabelsFromIssuesInput!) {
  removeLabelsFromIssues(input: $input) {
    successCount
    failedIssues {
      id
      number
      title
    }
    labels {
      id
      name
      color
    }
    githubErrors
  }
}`

// resolvedLabelIssue holds the info needed to add/remove labels from a single issue.
type resolvedLabelIssue struct {
	IssueID   string
	Number    int
	Title     string
	RepoName  string
	RepoOwner string
}

func (r *resolvedLabelIssue) Ref() string {
	return fmt.Sprintf("%s#%d", r.RepoName, r.Number)
}

// Commands

var issueLabelCmd = &cobra.Command{
	Use:   "label",
	Short: "Add or remove labels from issues",
	Long: `Add or remove labels from issues.

Use -- to separate issue identifiers from label names.

Examples:
  zh issue label add task-tracker#1 -- bug
  zh issue label add task-tracker#1 task-tracker#2 -- bug enhancement
  zh issue label remove task-tracker#1 -- bug`,
}

var issueLabelAddCmd = &cobra.Command{
	Use:   "add <issue>... -- <label>...",
	Short: "Add labels to issues",
	Long: `Add one or more labels to one or more issues.

Use -- to separate issue identifiers from label names. Arguments before --
are issue identifiers; arguments after -- are label names.

Issues can be specified as repo#number, owner/repo#number, ZenHub IDs,
or bare numbers with --repo.

Examples:
  zh issue label add task-tracker#1 -- bug
  zh issue label add task-tracker#1 task-tracker#2 -- bug enhancement
  zh issue label add --repo=task-tracker 1 2 -- bug "help wanted"`,
	Args:               cobra.MinimumNArgs(1),
	RunE:               runIssueLabelAdd,
	DisableFlagParsing: false,
}

var issueLabelRemoveCmd = &cobra.Command{
	Use:   "remove <issue>... -- <label>...",
	Short: "Remove labels from issues",
	Long: `Remove one or more labels from one or more issues.

Use -- to separate issue identifiers from label names. Arguments before --
are issue identifiers; arguments after -- are label names.

Issues can be specified as repo#number, owner/repo#number, ZenHub IDs,
or bare numbers with --repo.

Examples:
  zh issue label remove task-tracker#1 -- bug
  zh issue label remove task-tracker#1 task-tracker#2 -- bug enhancement
  zh issue label remove --repo=task-tracker 1 2 -- "help wanted"`,
	Args: cobra.MinimumNArgs(1),
	RunE: runIssueLabelRemove,
}

var (
	issueLabelAddDryRun          bool
	issueLabelAddRepo            string
	issueLabelAddContinueOnError bool

	issueLabelRemoveDryRun          bool
	issueLabelRemoveRepo            string
	issueLabelRemoveContinueOnError bool
)

func init() {
	issueLabelAddCmd.Flags().BoolVar(&issueLabelAddDryRun, "dry-run", false, "Show what would be changed without executing")
	issueLabelAddCmd.Flags().StringVar(&issueLabelAddRepo, "repo", "", "Repository context for bare issue numbers")
	issueLabelAddCmd.Flags().BoolVar(&issueLabelAddContinueOnError, "continue-on-error", false, "Continue processing remaining issues after a resolution error")

	issueLabelRemoveCmd.Flags().BoolVar(&issueLabelRemoveDryRun, "dry-run", false, "Show what would be changed without executing")
	issueLabelRemoveCmd.Flags().StringVar(&issueLabelRemoveRepo, "repo", "", "Repository context for bare issue numbers")
	issueLabelRemoveCmd.Flags().BoolVar(&issueLabelRemoveContinueOnError, "continue-on-error", false, "Continue processing remaining issues after a resolution error")

	issueLabelCmd.AddCommand(issueLabelAddCmd)
	issueLabelCmd.AddCommand(issueLabelRemoveCmd)
	issueCmd.AddCommand(issueLabelCmd)
}

func resetIssueLabelFlags() {
	issueLabelAddDryRun = false
	issueLabelAddRepo = ""
	issueLabelAddContinueOnError = false
	issueLabelRemoveDryRun = false
	issueLabelRemoveRepo = ""
	issueLabelRemoveContinueOnError = false
}

// splitIssuesAndLabels separates issue identifiers from label names
// using the "--" separator. Arguments before -- are issue identifiers;
// arguments after -- are label names.
func splitIssuesAndLabels(cmd *cobra.Command, args []string) (issueArgs, labelArgs []string, err error) {
	dash := cmd.ArgsLenAtDash()
	if dash == -1 {
		return nil, nil, exitcode.Usage("use -- to separate issue identifiers from label names\n\nExample: zh issue label add task-tracker#1 -- bug enhancement")
	}

	issueArgs = args[:dash]
	labelArgs = args[dash:]

	if len(issueArgs) == 0 {
		return nil, nil, exitcode.Usage("at least one issue identifier is required")
	}
	if len(labelArgs) == 0 {
		return nil, nil, exitcode.Usage("at least one label name is required")
	}

	return issueArgs, labelArgs, nil
}

func runIssueLabelAdd(cmd *cobra.Command, args []string) error {
	issueArgs, labelArgs, err := splitIssuesAndLabels(cmd, args)
	if err != nil {
		return err
	}

	return runIssueLabelOp(cmd, issueArgs, labelArgs, "add",
		issueLabelAddRepo, issueLabelAddDryRun, issueLabelAddContinueOnError)
}

func runIssueLabelRemove(cmd *cobra.Command, args []string) error {
	issueArgs, labelArgs, err := splitIssuesAndLabels(cmd, args)
	if err != nil {
		return err
	}

	return runIssueLabelOp(cmd, issueArgs, labelArgs, "remove",
		issueLabelRemoveRepo, issueLabelRemoveDryRun, issueLabelRemoveContinueOnError)
}

func runIssueLabelOp(cmd *cobra.Command, issueArgs, labelNames []string, op, repoFlag string, dryRun, continueOnError bool) error {
	cfg, err := requireWorkspace()
	if err != nil {
		return err
	}

	client := newClient(cfg, cmd)
	w := cmd.OutOrStdout()
	ghClient := newGitHubClient(cfg, cmd)

	// Resolve each issue
	var resolved []resolvedLabelIssue
	var resolveFailed []output.FailedItem

	for _, arg := range issueArgs {
		issue, err := resolveForLabel(client, cfg.Workspace, arg, repoFlag, ghClient)
		if err != nil {
			if continueOnError {
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

	// Resolve label names to IDs
	resolvedLabels, err := resolve.Labels(client, cfg.Workspace, labelNames)
	if err != nil {
		return err
	}

	// Build display names from resolved labels
	resolvedLabelNames := make([]string, len(resolvedLabels))
	for i, l := range resolvedLabels {
		resolvedLabelNames[i] = l.Name
	}
	labelDisplay := strings.Join(resolvedLabelNames, ", ")

	// Dry run
	if dryRun {
		return renderLabelDryRun(w, resolved, resolveFailed, resolvedLabelNames, op)
	}

	// Build issue IDs
	issueIDs := make([]string, len(resolved))
	for i, r := range resolved {
		issueIDs[i] = r.IssueID
	}

	// Build label IDs
	labelIDs := make([]string, len(resolvedLabels))
	for i, l := range resolvedLabels {
		labelIDs[i] = l.ID
	}

	// Execute mutation
	var mutation string
	var mutationKey string
	if op == "add" {
		mutation = addLabelsToIssuesMutation
		mutationKey = "addLabelsToIssues"
	} else {
		mutation = removeLabelsFromIssuesMutation
		mutationKey = "removeLabelsFromIssues"
	}

	data, err := client.Execute(mutation, map[string]any{
		"input": map[string]any{
			"issueIds": issueIDs,
			"labelIds": labelIDs,
		},
	})
	if err != nil {
		return exitcode.General(fmt.Sprintf("%sing labels", op), err)
	}

	var resp struct {
		SuccessCount int `json:"successCount"`
		FailedIssues []struct {
			ID     string `json:"id"`
			Number int    `json:"number"`
			Title  string `json:"title"`
		} `json:"failedIssues"`
		Labels []struct {
			ID    string `json:"id"`
			Name  string `json:"name"`
			Color string `json:"color"`
		} `json:"labels"`
		GithubErrors json.RawMessage `json:"githubErrors"`
	}

	// The response is nested under the mutation name
	var rawResp map[string]json.RawMessage
	if err := json.Unmarshal(data, &rawResp); err != nil {
		return exitcode.General("parsing label response", err)
	}
	if mutData, ok := rawResp[mutationKey]; ok {
		if err := json.Unmarshal(mutData, &resp); err != nil {
			return exitcode.General("parsing label response", err)
		}
	}

	// Build succeeded/failed lists from mutation response
	failedIDs := make(map[string]bool)
	var mutationFailed []output.FailedItem
	for _, f := range resp.FailedIssues {
		failedIDs[f.ID] = true
		mutationFailed = append(mutationFailed, output.FailedItem{
			Ref:    fmt.Sprintf("#%d", f.Number),
			Reason: fmt.Sprintf("failed to %s labels", op),
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
			"operation":    op,
			"labels":       labelNames,
			"succeeded":    formatMutationItemsJSON(succeeded),
			"failed":       allFailed,
			"successCount": resp.SuccessCount,
		})
	}

	// Render output
	verb := "Added"
	preposition := "to"
	if op == "remove" {
		verb = "Removed"
		preposition = "from"
	}

	totalAttempted := len(succeeded) + len(allFailed)
	if len(allFailed) > 0 {
		header := output.Green(fmt.Sprintf("%s label(s) %s %s %d of %d issue(s).", verb, labelDisplay, preposition, len(succeeded), totalAttempted))
		output.MutationPartialFailure(w, header, succeeded, allFailed)
	} else if len(succeeded) == 1 {
		output.MutationSingle(w, output.Green(fmt.Sprintf(
			"%s label(s) %s %s %s.", verb, labelDisplay, preposition, succeeded[0].Ref,
		)))
	} else {
		header := output.Green(fmt.Sprintf("%s label(s) %s %s %d issue(s).", verb, labelDisplay, preposition, len(succeeded)))
		output.MutationBatch(w, header, succeeded)
	}

	if len(allFailed) > 0 {
		return exitcode.Generalf("some issues failed")
	}

	return nil
}

// resolveForLabel resolves an issue identifier and fetches basic info.
func resolveForLabel(client *api.Client, workspaceID, identifier, repoFlag string, ghClient *gh.Client) (*resolvedLabelIssue, error) {
	result, err := resolve.Issue(client, workspaceID, identifier, &resolve.IssueOptions{
		RepoFlag:     repoFlag,
		GitHubClient: ghClient,
	})
	if err != nil {
		return nil, err
	}

	// Fetch issue details for display
	data, err := client.Execute(issueLabelResolveQuery, map[string]any{
		"issueId": result.ID,
	})
	if err != nil {
		return nil, exitcode.General("fetching issue details", err)
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
		} `json:"node"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, exitcode.General("parsing issue label response", err)
	}

	if resp.Node == nil {
		return nil, exitcode.NotFoundError(fmt.Sprintf("issue %q not found", identifier))
	}

	return &resolvedLabelIssue{
		IssueID:   resp.Node.ID,
		Number:    resp.Node.Number,
		Title:     resp.Node.Title,
		RepoName:  resp.Node.Repository.Name,
		RepoOwner: resp.Node.Repository.OwnerName,
	}, nil
}

const issueLabelResolveQuery = `query GetIssueForLabel($issueId: ID!) {
  node(id: $issueId) {
    ... on Issue {
      id
      number
      title
      repository {
        name
        ownerName
      }
    }
  }
}`

func renderLabelDryRun(w writerFlusher, resolved []resolvedLabelIssue, resolveFailed []output.FailedItem, labels []string, op string) error {
	items := make([]output.MutationItem, len(resolved))
	for i, r := range resolved {
		items[i] = output.MutationItem{
			Ref:   r.Ref(),
			Title: truncateTitle(r.Title),
		}
	}

	labelDisplay := strings.Join(labels, ", ")
	var header string
	if op == "add" {
		header = fmt.Sprintf("Would add label(s) %s to %d issue(s)", labelDisplay, len(resolved))
	} else {
		header = fmt.Sprintf("Would remove label(s) %s from %d issue(s)", labelDisplay, len(resolved))
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
