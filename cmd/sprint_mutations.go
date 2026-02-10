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

// GraphQL mutations for sprint commands

const addIssuesToSprintsMutation = `mutation AddIssuesToSprints($input: AddIssuesToSprintsInput!) {
  addIssuesToSprints(input: $input) {
    sprintIssues {
      id
      issue {
        id
        number
        title
        repository {
          name
          owner {
            login
          }
        }
      }
      sprint {
        id
        name
        state
      }
    }
  }
}`

const removeIssuesFromSprintsMutation = `mutation RemoveIssuesFromSprints($input: RemoveIssuesFromSprintsInput!) {
  removeIssuesFromSprints(input: $input) {
    sprints {
      id
      name
      generatedName
      state
      totalPoints
      completedPoints
      closedIssuesCount
    }
  }
}`

// Commands

var sprintAddCmd = &cobra.Command{
	Use:   "add <issue>...",
	Short: "Add issues to a sprint",
	Long: `Add one or more issues to a sprint. Defaults to the active sprint.

Issues can be specified as repo#number, owner/repo#number, ZenHub IDs,
or bare numbers with --repo.

Examples:
  zh sprint add task-tracker#1 task-tracker#2
  zh sprint add --sprint=next task-tracker#5
  zh sprint add --repo=task-tracker 1 2 3`,
	Args: cobra.MinimumNArgs(1),
	RunE: runSprintAdd,
}

var sprintRemoveCmd = &cobra.Command{
	Use:   "remove <issue>...",
	Short: "Remove issues from a sprint",
	Long: `Remove one or more issues from a sprint. Defaults to the active sprint.

Issues can be specified as repo#number, owner/repo#number, ZenHub IDs,
or bare numbers with --repo.

Examples:
  zh sprint remove task-tracker#1
  zh sprint remove --sprint="Sprint 47" task-tracker#5
  zh sprint remove --repo=task-tracker 1 2 3`,
	Args: cobra.MinimumNArgs(1),
	RunE: runSprintRemove,
}

// Flag variables

var (
	sprintAddSprint          string
	sprintAddRepo            string
	sprintAddDryRun          bool
	sprintAddContinueOnError bool

	sprintRemoveSprint          string
	sprintRemoveRepo            string
	sprintRemoveDryRun          bool
	sprintRemoveContinueOnError bool
)

func init() {
	sprintAddCmd.Flags().StringVar(&sprintAddSprint, "sprint", "", "Target sprint (default: active). Supports name, ID, or current/next/previous")
	sprintAddCmd.Flags().StringVar(&sprintAddRepo, "repo", "", "Repository context for bare issue numbers")
	sprintAddCmd.Flags().BoolVar(&sprintAddDryRun, "dry-run", false, "Show what would be added without executing")
	sprintAddCmd.Flags().BoolVar(&sprintAddContinueOnError, "continue-on-error", false, "Continue processing remaining issues after a resolution error")

	sprintRemoveCmd.Flags().StringVar(&sprintRemoveSprint, "sprint", "", "Target sprint (default: active). Supports name, ID, or current/next/previous")
	sprintRemoveCmd.Flags().StringVar(&sprintRemoveRepo, "repo", "", "Repository context for bare issue numbers")
	sprintRemoveCmd.Flags().BoolVar(&sprintRemoveDryRun, "dry-run", false, "Show what would be removed without executing")
	sprintRemoveCmd.Flags().BoolVar(&sprintRemoveContinueOnError, "continue-on-error", false, "Continue processing remaining issues after a resolution error")

	sprintCmd.AddCommand(sprintAddCmd)
	sprintCmd.AddCommand(sprintRemoveCmd)
}

func resetSprintMutationFlags() {
	sprintAddSprint = ""
	sprintAddRepo = ""
	sprintAddDryRun = false
	sprintAddContinueOnError = false

	sprintRemoveSprint = ""
	sprintRemoveRepo = ""
	sprintRemoveDryRun = false
	sprintRemoveContinueOnError = false
}

// resolvedSprintIssue holds minimal info about an issue resolved for sprint add/remove.
type resolvedSprintIssue struct {
	ID        string
	Number    int
	RepoName  string
	RepoOwner string
	Title     string
}

func (r *resolvedSprintIssue) Ref() string {
	return fmt.Sprintf("%s#%d", r.RepoName, r.Number)
}

// resolveIssueForSprint resolves an issue identifier and fetches its title.
func resolveIssueForSprint(client *api.Client, workspaceID, identifier, repoFlag string, ghClient *gh.Client) (*resolvedSprintIssue, error) {
	result, err := resolve.Issue(client, workspaceID, identifier, &resolve.IssueOptions{
		RepoFlag:     repoFlag,
		GitHubClient: ghClient,
	})
	if err != nil {
		return nil, err
	}

	// Fetch title
	data, err := client.Execute(issueResolveForEpicQuery, map[string]any{
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
		return nil, exitcode.General("parsing issue details", err)
	}

	if resp.Node == nil {
		return nil, exitcode.NotFoundError(fmt.Sprintf("issue %q not found", identifier))
	}

	return &resolvedSprintIssue{
		ID:        resp.Node.ID,
		Number:    resp.Node.Number,
		Title:     resp.Node.Title,
		RepoName:  resp.Node.Repository.Name,
		RepoOwner: resp.Node.Repository.OwnerName,
	}, nil
}

// resolveTargetSprint resolves the sprint to target, defaulting to "current".
func resolveTargetSprint(client *api.Client, workspaceID, sprintFlag string) (*resolve.SprintResult, error) {
	identifier := "current"
	if sprintFlag != "" {
		identifier = sprintFlag
	}
	return resolve.Sprint(client, workspaceID, identifier)
}

// runSprintAdd implements `zh sprint add <issue>...`.
func runSprintAdd(cmd *cobra.Command, args []string) error {
	cfg, err := requireWorkspace()
	if err != nil {
		return err
	}

	client := newClient(cfg, cmd)
	w := cmd.OutOrStdout()
	ghClient := newGitHubClient(cfg, cmd)

	// Resolve target sprint
	sprint, err := resolveTargetSprint(client, cfg.Workspace, sprintAddSprint)
	if err != nil {
		return err
	}

	// Resolve issue identifiers
	var issues []resolvedSprintIssue
	var failed []output.FailedItem

	for _, arg := range args {
		issue, err := resolveIssueForSprint(client, cfg.Workspace, arg, sprintAddRepo, ghClient)
		if err != nil {
			if sprintAddContinueOnError {
				failed = append(failed, output.FailedItem{
					Ref:    arg,
					Reason: err.Error(),
				})
				continue
			}
			return err
		}
		issues = append(issues, *issue)
	}

	if len(issues) == 0 && len(failed) > 0 {
		return exitcode.Generalf("all issues failed to resolve")
	}

	// Dry run
	if sprintAddDryRun {
		return renderSprintAddDryRun(w, sprint, issues, failed)
	}

	// Execute the mutation
	issueIDs := make([]string, len(issues))
	for i, iss := range issues {
		issueIDs[i] = iss.ID
	}

	data, err := client.Execute(addIssuesToSprintsMutation, map[string]any{
		"input": map[string]any{
			"issueIds":  issueIDs,
			"sprintIds": []string{sprint.ID},
		},
	})
	if err != nil {
		return exitcode.General("adding issues to sprint", err)
	}

	// Parse response
	var resp struct {
		AddIssuesToSprints struct {
			SprintIssues []struct {
				ID    string `json:"id"`
				Issue struct {
					ID    string `json:"id"`
					Title string `json:"title"`
				} `json:"issue"`
			} `json:"sprintIssues"`
		} `json:"addIssuesToSprints"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return exitcode.General("parsing add issues to sprint response", err)
	}

	// Build succeeded list
	succeeded := make([]output.MutationItem, len(issues))
	for i, iss := range issues {
		succeeded[i] = output.MutationItem{
			Ref:   iss.Ref(),
			Title: truncateTitle(iss.Title),
		}
	}

	// JSON output
	if output.IsJSON(outputFormat) {
		return output.JSON(w, map[string]any{
			"sprint": map[string]any{"id": sprint.ID, "name": sprint.Name},
			"added":  formatSprintIssueItemsJSON(issues),
		})
	}

	// Render output
	if len(failed) > 0 {
		totalAttempted := len(succeeded) + len(failed)
		header := output.Green(fmt.Sprintf("Added %d of %d issue(s) to %s.", len(succeeded), totalAttempted, sprint.Name))
		output.MutationPartialFailure(w, header, succeeded, failed)
		return exitcode.Generalf("some issues failed to resolve")
	} else if len(succeeded) == 1 {
		output.MutationSingle(w, output.Green(fmt.Sprintf("Added %s to %s.", succeeded[0].Ref, sprint.Name)))
	} else {
		header := output.Green(fmt.Sprintf("Added %d issue(s) to %s.", len(succeeded), sprint.Name))
		output.MutationBatch(w, header, succeeded)
	}

	return nil
}

func renderSprintAddDryRun(w writerFlusher, sprint *resolve.SprintResult, issues []resolvedSprintIssue, failed []output.FailedItem) error {
	if len(issues) > 0 {
		items := make([]output.MutationItem, len(issues))
		for i, iss := range issues {
			items[i] = output.MutationItem{
				Ref:   iss.Ref(),
				Title: truncateTitle(iss.Title),
			}
		}
		header := fmt.Sprintf("Would add %d issue(s) to %s", len(issues), sprint.Name)
		output.MutationDryRun(w, header, items)
	}

	if len(failed) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, output.Red("Failed to resolve:"))
		fmt.Fprintln(w)
		for _, f := range failed {
			fmt.Fprintf(w, "  %s  %s\n", f.Ref, output.Red(f.Reason))
		}
	}

	return nil
}

// runSprintRemove implements `zh sprint remove <issue>...`.
func runSprintRemove(cmd *cobra.Command, args []string) error {
	cfg, err := requireWorkspace()
	if err != nil {
		return err
	}

	client := newClient(cfg, cmd)
	w := cmd.OutOrStdout()
	ghClient := newGitHubClient(cfg, cmd)

	// Resolve target sprint
	sprint, err := resolveTargetSprint(client, cfg.Workspace, sprintRemoveSprint)
	if err != nil {
		return err
	}

	// Resolve issue identifiers
	var issues []resolvedSprintIssue
	var failed []output.FailedItem

	for _, arg := range args {
		issue, err := resolveIssueForSprint(client, cfg.Workspace, arg, sprintRemoveRepo, ghClient)
		if err != nil {
			if sprintRemoveContinueOnError {
				failed = append(failed, output.FailedItem{
					Ref:    arg,
					Reason: err.Error(),
				})
				continue
			}
			return err
		}
		issues = append(issues, *issue)
	}

	if len(issues) == 0 && len(failed) > 0 {
		return exitcode.Generalf("all issues failed to resolve")
	}

	// Dry run
	if sprintRemoveDryRun {
		return renderSprintRemoveDryRun(w, sprint, issues, failed)
	}

	// Execute the mutation
	issueIDs := make([]string, len(issues))
	for i, iss := range issues {
		issueIDs[i] = iss.ID
	}

	data, err := client.Execute(removeIssuesFromSprintsMutation, map[string]any{
		"input": map[string]any{
			"issueIds":  issueIDs,
			"sprintIds": []string{sprint.ID},
		},
	})
	if err != nil {
		return exitcode.General("removing issues from sprint", err)
	}

	// Parse response
	var resp struct {
		RemoveIssuesFromSprints struct {
			Sprints []struct {
				ID              string  `json:"id"`
				Name            string  `json:"name"`
				GeneratedName   string  `json:"generatedName"`
				State           string  `json:"state"`
				TotalPoints     float64 `json:"totalPoints"`
				CompletedPoints float64 `json:"completedPoints"`
				ClosedIssues    int     `json:"closedIssuesCount"`
			} `json:"sprints"`
		} `json:"removeIssuesFromSprints"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return exitcode.General("parsing remove issues from sprint response", err)
	}

	// Build succeeded list
	succeeded := make([]output.MutationItem, len(issues))
	for i, iss := range issues {
		succeeded[i] = output.MutationItem{
			Ref:   iss.Ref(),
			Title: truncateTitle(iss.Title),
		}
	}

	// JSON output
	if output.IsJSON(outputFormat) {
		return output.JSON(w, map[string]any{
			"sprint":  map[string]any{"id": sprint.ID, "name": sprint.Name},
			"removed": formatSprintIssueItemsJSON(issues),
		})
	}

	// Render output
	if len(failed) > 0 {
		totalAttempted := len(succeeded) + len(failed)
		header := output.Green(fmt.Sprintf("Removed %d of %d issue(s) from %s.", len(succeeded), totalAttempted, sprint.Name))
		output.MutationPartialFailure(w, header, succeeded, failed)
		return exitcode.Generalf("some issues failed to resolve")
	} else if len(succeeded) == 1 {
		output.MutationSingle(w, output.Green(fmt.Sprintf("Removed %s from %s.", succeeded[0].Ref, sprint.Name)))
	} else {
		header := output.Green(fmt.Sprintf("Removed %d issue(s) from %s.", len(succeeded), sprint.Name))
		output.MutationBatch(w, header, succeeded)
	}

	return nil
}

func renderSprintRemoveDryRun(w writerFlusher, sprint *resolve.SprintResult, issues []resolvedSprintIssue, failed []output.FailedItem) error {
	if len(issues) > 0 {
		items := make([]output.MutationItem, len(issues))
		for i, iss := range issues {
			items[i] = output.MutationItem{
				Ref:   iss.Ref(),
				Title: truncateTitle(iss.Title),
			}
		}
		header := fmt.Sprintf("Would remove %d issue(s) from %s", len(issues), sprint.Name)
		output.MutationDryRun(w, header, items)
	}

	if len(failed) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, output.Red("Failed to resolve:"))
		fmt.Fprintln(w)
		for _, f := range failed {
			fmt.Fprintf(w, "  %s  %s\n", f.Ref, output.Red(f.Reason))
		}
	}

	return nil
}

func formatSprintIssueItemsJSON(issues []resolvedSprintIssue) []map[string]any {
	result := make([]map[string]any, len(issues))
	for i, iss := range issues {
		result[i] = map[string]any{
			"id":         iss.ID,
			"number":     iss.Number,
			"repository": fmt.Sprintf("%s/%s", iss.RepoOwner, iss.RepoName),
			"title":      iss.Title,
		}
	}
	return result
}
