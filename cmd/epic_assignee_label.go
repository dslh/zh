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

// GraphQL mutations for epic assignee and label operations

const addAssigneesToZenhubEpicsMutation = `mutation AddAssigneesToZenhubEpics($input: AddAssigneesToZenhubEpicsInput!) {
  addAssigneesToZenhubEpics(input: $input) {
    zenhubEpics {
      id
      assignees(first: 50) {
        nodes {
          id
          name
          githubUser { login }
        }
      }
    }
  }
}`

const removeAssigneesFromZenhubEpicsMutation = `mutation RemoveAssigneesFromZenhubEpics($input: RemoveAssigneesFromZenhubEpicsInput!) {
  removeAssigneesFromZenhubEpics(input: $input) {
    zenhubEpics {
      id
      assignees(first: 50) {
        nodes {
          id
          name
          githubUser { login }
        }
      }
    }
  }
}`

const addZenhubLabelsToZenhubEpicsMutation = `mutation AddZenhubLabelsToZenhubEpics($input: AddZenhubLabelsToZenhubEpicsInput!) {
  addZenhubLabelsToZenhubEpics(input: $input) {
    zenhubEpics {
      id
      labels(first: 50) {
        nodes { id name color }
      }
    }
  }
}`

const removeZenhubLabelsFromZenhubEpicsMutation = `mutation RemoveZenhubLabelsFromZenhubEpics($input: RemoveZenhubLabelsFromZenhubEpicsInput!) {
  removeZenhubLabelsFromZenhubEpics(input: $input) {
    zenhubEpics {
      id
      labels(first: 50) {
        nodes { id name color }
      }
    }
  }
}`

// Commands

var epicAssigneeCmd = &cobra.Command{
	Use:   "assignee",
	Short: "Add or remove assignees from an epic",
	Long: `Add or remove assignees from a ZenHub epic.

Examples:
  zh epic assignee add "Q1 Roadmap" johndoe janedoe
  zh epic assignee remove "Q1 Roadmap" johndoe`,
}

var epicAssigneeAddCmd = &cobra.Command{
	Use:   "add <epic> <user>...",
	Short: "Add assignees to an epic",
	Long: `Add one or more assignees to a ZenHub epic.

Users can be specified by GitHub login, display name, or ZenHub user ID.
Prefix with @ is optional (e.g., @johndoe and johndoe are equivalent).

Examples:
  zh epic assignee add "Q1 Roadmap" johndoe
  zh epic assignee add "Q1 Roadmap" @johndoe @janedoe`,
	Args: cobra.MinimumNArgs(2),
	RunE: runEpicAssigneeAdd,
}

var epicAssigneeRemoveCmd = &cobra.Command{
	Use:   "remove <epic> <user>...",
	Short: "Remove assignees from an epic",
	Long: `Remove one or more assignees from a ZenHub epic.

Users can be specified by GitHub login, display name, or ZenHub user ID.

Examples:
  zh epic assignee remove "Q1 Roadmap" johndoe
  zh epic assignee remove "Q1 Roadmap" @johndoe @janedoe`,
	Args: cobra.MinimumNArgs(2),
	RunE: runEpicAssigneeRemove,
}

var epicLabelCmd = &cobra.Command{
	Use:   "label",
	Short: "Add or remove labels from an epic",
	Long: `Add or remove labels from a ZenHub epic.

Examples:
  zh epic label add "Q1 Roadmap" platform priority:high
  zh epic label remove "Q1 Roadmap" platform`,
}

var epicLabelAddCmd = &cobra.Command{
	Use:   "add <epic> <label>...",
	Short: "Add labels to an epic",
	Long: `Add one or more labels to a ZenHub epic.

Labels are resolved by name (case-insensitive) from the workspace's
ZenHub labels.

Examples:
  zh epic label add "Q1 Roadmap" platform
  zh epic label add "Q1 Roadmap" platform "priority:high"`,
	Args: cobra.MinimumNArgs(2),
	RunE: runEpicLabelAdd,
}

var epicLabelRemoveCmd = &cobra.Command{
	Use:   "remove <epic> <label>...",
	Short: "Remove labels from an epic",
	Long: `Remove one or more labels from a ZenHub epic.

Labels are resolved by name (case-insensitive) from the workspace's
ZenHub labels.

Examples:
  zh epic label remove "Q1 Roadmap" platform
  zh epic label remove "Q1 Roadmap" platform "priority:high"`,
	Args: cobra.MinimumNArgs(2),
	RunE: runEpicLabelRemove,
}

// Flag variables

var (
	epicAssigneeAddDryRun          bool
	epicAssigneeAddContinueOnError bool

	epicAssigneeRemoveDryRun          bool
	epicAssigneeRemoveContinueOnError bool

	epicLabelAddDryRun          bool
	epicLabelAddContinueOnError bool

	epicLabelRemoveDryRun          bool
	epicLabelRemoveContinueOnError bool
)

func init() {
	epicAssigneeAddCmd.Flags().BoolVar(&epicAssigneeAddDryRun, "dry-run", false, "Show what would be changed without executing")
	epicAssigneeAddCmd.Flags().BoolVar(&epicAssigneeAddContinueOnError, "continue-on-error", false, "Continue processing remaining users after a resolution error")

	epicAssigneeRemoveCmd.Flags().BoolVar(&epicAssigneeRemoveDryRun, "dry-run", false, "Show what would be changed without executing")
	epicAssigneeRemoveCmd.Flags().BoolVar(&epicAssigneeRemoveContinueOnError, "continue-on-error", false, "Continue processing remaining users after a resolution error")

	epicLabelAddCmd.Flags().BoolVar(&epicLabelAddDryRun, "dry-run", false, "Show what would be changed without executing")
	epicLabelAddCmd.Flags().BoolVar(&epicLabelAddContinueOnError, "continue-on-error", false, "Continue processing remaining labels after a resolution error")

	epicLabelRemoveCmd.Flags().BoolVar(&epicLabelRemoveDryRun, "dry-run", false, "Show what would be changed without executing")
	epicLabelRemoveCmd.Flags().BoolVar(&epicLabelRemoveContinueOnError, "continue-on-error", false, "Continue processing remaining labels after a resolution error")

	epicAssigneeCmd.AddCommand(epicAssigneeAddCmd)
	epicAssigneeCmd.AddCommand(epicAssigneeRemoveCmd)
	epicCmd.AddCommand(epicAssigneeCmd)

	epicLabelCmd.AddCommand(epicLabelAddCmd)
	epicLabelCmd.AddCommand(epicLabelRemoveCmd)
	epicCmd.AddCommand(epicLabelCmd)
}

func resetEpicAssigneeLabelFlags() {
	epicAssigneeAddDryRun = false
	epicAssigneeAddContinueOnError = false
	epicAssigneeRemoveDryRun = false
	epicAssigneeRemoveContinueOnError = false
	epicLabelAddDryRun = false
	epicLabelAddContinueOnError = false
	epicLabelRemoveDryRun = false
	epicLabelRemoveContinueOnError = false
}

// --- Epic assignee commands ---

func runEpicAssigneeAdd(cmd *cobra.Command, args []string) error {
	return runEpicAssigneeOp(cmd, args, "add", epicAssigneeAddDryRun, epicAssigneeAddContinueOnError)
}

func runEpicAssigneeRemove(cmd *cobra.Command, args []string) error {
	return runEpicAssigneeOp(cmd, args, "remove", epicAssigneeRemoveDryRun, epicAssigneeRemoveContinueOnError)
}

func runEpicAssigneeOp(cmd *cobra.Command, args []string, op string, dryRun, continueOnError bool) error {
	cfg, err := requireWorkspace()
	if err != nil {
		return err
	}

	client := newClient(cfg, cmd)
	w := cmd.OutOrStdout()

	// Resolve the epic
	resolved, err := resolve.Epic(client, cfg.Workspace, args[0], cfg.Aliases.Epics)
	if err != nil {
		return err
	}

	if resolved.Type == "legacy" {
		return exitcode.Usage(fmt.Sprintf("epic %q is a legacy epic (backed by GitHub issue %s/%s#%d) — managing assignees is only supported for ZenHub epics",
			resolved.Title, resolved.RepoOwner, resolved.RepoName, resolved.IssueNumber))
	}

	// Resolve users
	userArgs := args[1:]
	var users []*resolve.UserResult
	var failed []output.FailedItem

	for _, arg := range userArgs {
		user, err := resolve.User(client, cfg.Workspace, arg)
		if err != nil {
			if continueOnError {
				failed = append(failed, output.FailedItem{
					Ref:    arg,
					Reason: err.Error(),
				})
				continue
			}
			return err
		}
		users = append(users, user)
	}

	if len(users) == 0 && len(failed) > 0 {
		return exitcode.Generalf("all users failed to resolve")
	}

	// Build display names
	userNames := make([]string, len(users))
	for i, u := range users {
		userNames[i] = u.DisplayName()
	}
	userDisplay := strings.Join(userNames, ", ")

	// Dry run
	if dryRun {
		return renderEpicAssigneeDryRun(w, resolved, users, failed, op)
	}

	// Build mutation input
	userIDs := make([]string, len(users))
	for i, u := range users {
		userIDs[i] = u.ID
	}

	var mutation string
	var mutationKey string
	if op == "add" {
		mutation = addAssigneesToZenhubEpicsMutation
		mutationKey = "addAssigneesToZenhubEpics"
	} else {
		mutation = removeAssigneesFromZenhubEpicsMutation
		mutationKey = "removeAssigneesFromZenhubEpics"
	}

	data, err := client.Execute(mutation, map[string]any{
		"input": map[string]any{
			"zenhubEpicIds": []string{resolved.ID},
			"assigneeIds":   userIDs,
		},
	})
	if err != nil {
		return exitcode.General(fmt.Sprintf("%sing assignees", op), err)
	}

	// Parse response
	var resp struct {
		ZenhubEpics []struct {
			ID        string `json:"id"`
			Assignees struct {
				Nodes []struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				} `json:"nodes"`
			} `json:"assignees"`
		} `json:"zenhubEpics"`
	}

	var rawResp map[string]json.RawMessage
	if err := json.Unmarshal(data, &rawResp); err != nil {
		return exitcode.General("parsing assignee response", err)
	}
	if mutData, ok := rawResp[mutationKey]; ok {
		if err := json.Unmarshal(mutData, &resp); err != nil {
			return exitcode.General("parsing assignee response", err)
		}
	}

	// Build succeeded list
	succeeded := make([]output.MutationItem, len(users))
	for i, u := range users {
		succeeded[i] = output.MutationItem{
			Ref:   u.DisplayName(),
			Title: u.Name,
		}
	}

	// JSON output
	if output.IsJSON(outputFormat) {
		return output.JSON(w, map[string]any{
			"operation": op,
			"epic":      map[string]any{"id": resolved.ID, "title": resolved.Title},
			"users":     formatUserItemsJSON(users),
			"failed":    failed,
		})
	}

	// Render output
	verb := "Added"
	preposition := "to"
	if op == "remove" {
		verb = "Removed"
		preposition = "from"
	}

	totalAttempted := len(succeeded) + len(failed)
	if len(failed) > 0 {
		header := output.Green(fmt.Sprintf("%s %d of %d assignee(s) %s epic %q.", verb, len(succeeded), totalAttempted, preposition, resolved.Title))
		output.MutationPartialFailure(w, header, succeeded, failed)
		return exitcode.Generalf("some users failed to resolve")
	} else if len(succeeded) == 1 {
		output.MutationSingle(w, output.Green(fmt.Sprintf("%s %s %s epic %q.", verb, userDisplay, preposition, resolved.Title)))
	} else {
		header := output.Green(fmt.Sprintf("%s %d assignee(s) %s epic %q.", verb, len(succeeded), preposition, resolved.Title))
		output.MutationBatch(w, header, succeeded)
	}

	return nil
}

func renderEpicAssigneeDryRun(w writerFlusher, epic *resolve.EpicResult, users []*resolve.UserResult, failed []output.FailedItem, op string) error {
	if len(users) > 0 {
		items := make([]output.MutationItem, len(users))
		for i, u := range users {
			items[i] = output.MutationItem{
				Ref:   u.DisplayName(),
				Title: u.Name,
			}
		}

		var header string
		if op == "add" {
			header = fmt.Sprintf("Would add %d assignee(s) to epic %q", len(users), epic.Title)
		} else {
			header = fmt.Sprintf("Would remove %d assignee(s) from epic %q", len(users), epic.Title)
		}

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

func formatUserItemsJSON(users []*resolve.UserResult) []map[string]any {
	result := make([]map[string]any, len(users))
	for i, u := range users {
		result[i] = map[string]any{
			"id":    u.ID,
			"name":  u.Name,
			"login": u.Login,
		}
	}
	return result
}

// --- Epic label commands ---

func runEpicLabelAdd(cmd *cobra.Command, args []string) error {
	return runEpicLabelOp(cmd, args, "add", epicLabelAddDryRun, epicLabelAddContinueOnError)
}

func runEpicLabelRemove(cmd *cobra.Command, args []string) error {
	return runEpicLabelOp(cmd, args, "remove", epicLabelRemoveDryRun, epicLabelRemoveContinueOnError)
}

func runEpicLabelOp(cmd *cobra.Command, args []string, op string, dryRun, continueOnError bool) error {
	cfg, err := requireWorkspace()
	if err != nil {
		return err
	}

	client := newClient(cfg, cmd)
	w := cmd.OutOrStdout()

	// Resolve the epic
	resolved, err := resolve.Epic(client, cfg.Workspace, args[0], cfg.Aliases.Epics)
	if err != nil {
		return err
	}

	if resolved.Type == "legacy" {
		return exitcode.Usage(fmt.Sprintf("epic %q is a legacy epic (backed by GitHub issue %s/%s#%d) — managing labels is only supported for ZenHub epics",
			resolved.Title, resolved.RepoOwner, resolved.RepoName, resolved.IssueNumber))
	}

	// Resolve labels
	labelArgs := args[1:]
	var labels []*resolve.ZenhubLabelResult
	var failed []output.FailedItem

	if continueOnError {
		// Resolve one at a time for granular error reporting
		for _, arg := range labelArgs {
			label, err := resolve.ZenhubLabel(client, cfg.Workspace, arg)
			if err != nil {
				failed = append(failed, output.FailedItem{
					Ref:    arg,
					Reason: err.Error(),
				})
				continue
			}
			labels = append(labels, label)
		}
	} else {
		// Resolve all at once (stops on first error)
		labels, err = resolve.ZenhubLabels(client, cfg.Workspace, labelArgs)
		if err != nil {
			return err
		}
	}

	if len(labels) == 0 && len(failed) > 0 {
		return exitcode.Generalf("all labels failed to resolve")
	}

	// Build display names
	labelNames := make([]string, len(labels))
	for i, l := range labels {
		labelNames[i] = l.Name
	}
	labelDisplay := strings.Join(labelNames, ", ")

	// Dry run
	if dryRun {
		return renderEpicLabelDryRun(w, resolved, labels, failed, op)
	}

	// Build mutation input
	labelIDs := make([]string, len(labels))
	for i, l := range labels {
		labelIDs[i] = l.ID
	}

	var mutation string
	var mutationKey string
	if op == "add" {
		mutation = addZenhubLabelsToZenhubEpicsMutation
		mutationKey = "addZenhubLabelsToZenhubEpics"
	} else {
		mutation = removeZenhubLabelsFromZenhubEpicsMutation
		mutationKey = "removeZenhubLabelsFromZenhubEpics"
	}

	data, err := client.Execute(mutation, map[string]any{
		"input": map[string]any{
			"zenhubEpicIds":  []string{resolved.ID},
			"zenhubLabelIds": labelIDs,
		},
	})
	if err != nil {
		return exitcode.General(fmt.Sprintf("%sing labels on epic", op), err)
	}

	// Parse response
	var resp struct {
		ZenhubEpics []struct {
			ID     string `json:"id"`
			Labels struct {
				Nodes []struct {
					ID    string `json:"id"`
					Name  string `json:"name"`
					Color string `json:"color"`
				} `json:"nodes"`
			} `json:"labels"`
		} `json:"zenhubEpics"`
	}

	var rawResp map[string]json.RawMessage
	if err := json.Unmarshal(data, &rawResp); err != nil {
		return exitcode.General("parsing label response", err)
	}
	if mutData, ok := rawResp[mutationKey]; ok {
		if err := json.Unmarshal(mutData, &resp); err != nil {
			return exitcode.General("parsing label response", err)
		}
	}

	// Build succeeded list
	succeeded := make([]output.MutationItem, len(labels))
	for i, l := range labels {
		succeeded[i] = output.MutationItem{
			Ref:   l.Name,
			Title: "",
		}
	}

	// JSON output
	if output.IsJSON(outputFormat) {
		return output.JSON(w, map[string]any{
			"operation": op,
			"epic":      map[string]any{"id": resolved.ID, "title": resolved.Title},
			"labels":    formatZenhubLabelItemsJSON(labels),
			"failed":    failed,
		})
	}

	// Render output
	verb := "Added"
	preposition := "to"
	if op == "remove" {
		verb = "Removed"
		preposition = "from"
	}

	totalAttempted := len(succeeded) + len(failed)
	if len(failed) > 0 {
		header := output.Green(fmt.Sprintf("%s label(s) %s %s %d of %d epic label(s).", verb, labelDisplay, preposition, len(succeeded), totalAttempted))
		output.MutationPartialFailure(w, header, succeeded, failed)
		return exitcode.Generalf("some labels failed to resolve")
	} else if len(succeeded) == 1 {
		output.MutationSingle(w, output.Green(fmt.Sprintf("%s label %s %s epic %q.", verb, labelDisplay, preposition, resolved.Title)))
	} else {
		header := output.Green(fmt.Sprintf("%s %d label(s) %s epic %q.", verb, len(succeeded), preposition, resolved.Title))
		output.MutationBatch(w, header, succeeded)
	}

	return nil
}

func renderEpicLabelDryRun(w writerFlusher, epic *resolve.EpicResult, labels []*resolve.ZenhubLabelResult, failed []output.FailedItem, op string) error {
	if len(labels) > 0 {
		items := make([]output.MutationItem, len(labels))
		for i, l := range labels {
			items[i] = output.MutationItem{
				Ref:   l.Name,
				Title: "",
			}
		}

		labelNames := make([]string, len(labels))
		for i, l := range labels {
			labelNames[i] = l.Name
		}
		labelDisplay := strings.Join(labelNames, ", ")

		var header string
		if op == "add" {
			header = fmt.Sprintf("Would add label(s) %s to epic %q", labelDisplay, epic.Title)
		} else {
			header = fmt.Sprintf("Would remove label(s) %s from epic %q", labelDisplay, epic.Title)
		}

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

func formatZenhubLabelItemsJSON(labels []*resolve.ZenhubLabelResult) []map[string]any {
	result := make([]map[string]any, len(labels))
	for i, l := range labels {
		result[i] = map[string]any{
			"id":    l.ID,
			"name":  l.Name,
			"color": l.Color,
		}
	}
	return result
}
