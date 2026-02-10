package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/dslh/zh/internal/api"
	"github.com/dslh/zh/internal/cache"
	"github.com/dslh/zh/internal/config"
	"github.com/dslh/zh/internal/exitcode"
	"github.com/dslh/zh/internal/output"
	"github.com/dslh/zh/internal/resolve"
	"github.com/spf13/cobra"
)

// GraphQL mutations and queries for epic commands

const getWorkspaceOrgQuery = `query GetWorkspaceOrg($workspaceId: ID!) {
  workspace(id: $workspaceId) {
    zenhubOrganization {
      id
      name
    }
  }
}`

const createZenhubEpicMutation = `mutation CreateZenhubEpic($input: CreateZenhubEpicInput!) {
  createZenhubEpic(input: $input) {
    zenhubEpic {
      id
      title
      body
      state
      createdAt
    }
  }
}`

const updateZenhubEpicMutation = `mutation UpdateZenhubEpic($input: UpdateZenhubEpicInput!) {
  updateZenhubEpic(input: $input) {
    zenhubEpic {
      id
      title
      body
      state
      updatedAt
    }
  }
}`

const deleteZenhubEpicMutation = `mutation DeleteZenhubEpic($input: DeleteZenhubEpicInput!) {
  deleteZenhubEpic(input: $input) {
    zenhubEpicId
  }
}`

const epicChildIssueCountQuery = `query GetEpicChildCount($id: ID!, $workspaceId: ID!) {
  node(id: $id) {
    ... on ZenhubEpic {
      id
      title
      state
      childIssues(first: 1, workspaceId: $workspaceId) {
        totalCount
      }
    }
  }
}`

const updateZenhubEpicStateMutation = `mutation UpdateZenhubEpicState($input: UpdateZenhubEpicStateInput!) {
  updateZenhubEpicState(input: $input) {
    zenhubEpic {
      id
      title
      state
    }
  }
}`

const createLegacyEpicMutation = `mutation CreateEpic($input: CreateEpicInput!) {
  createEpic(input: $input) {
    epic {
      id
      issue {
        id
        number
        title
        htmlUrl
        repository {
          name
          ownerName
        }
      }
    }
  }
}`

// Commands

var epicCreateCmd = &cobra.Command{
	Use:   "create <title>",
	Short: "Create a new epic",
	Long: `Create a new ZenHub epic in the current workspace.

By default, creates a standalone ZenHub epic. Use --repo to create a
legacy epic backed by a GitHub issue instead.`,
	Args: cobra.ExactArgs(1),
	RunE: runEpicCreate,
}

var epicEditCmd = &cobra.Command{
	Use:   "edit <epic>",
	Short: "Update an epic's title or body",
	Long: `Update the title and/or body of a ZenHub epic.

At least one of --title or --body must be provided.`,
	Args: cobra.ExactArgs(1),
	RunE: runEpicEdit,
}

var epicDeleteCmd = &cobra.Command{
	Use:   "delete <epic>",
	Short: "Delete an epic",
	Long: `Delete a ZenHub epic. Child issues will be removed from the epic
but will not be deleted.`,
	Args: cobra.ExactArgs(1),
	RunE: runEpicDelete,
}

var epicSetStateCmd = &cobra.Command{
	Use:   "set-state <epic> <state>",
	Short: "Set the state of an epic",
	Long: `Set the state of a ZenHub epic.

Valid states: open, todo, in_progress, closed.

Use --apply-to-issues to also update the state of all child issues.`,
	Args: cobra.ExactArgs(2),
	RunE: runEpicSetState,
}

var epicAliasCmd = &cobra.Command{
	Use:   "alias <epic> <alias>",
	Short: "Set a shorthand name for an epic",
	Long: `Set a shorthand alias that can be used to reference the epic in
future commands. Aliases are stored in the config file.

Use --delete to remove an existing alias. Use --list to show all
epic aliases.`,
	Args: cobra.RangeArgs(0, 2),
	RunE: runEpicAlias,
}

// Flag variables

var (
	epicCreateBody   string
	epicCreateRepo   string
	epicCreateDryRun bool

	epicEditTitle  string
	epicEditBody   string
	epicEditDryRun bool

	epicDeleteDryRun bool

	epicSetStateApplyToIssues bool
	epicSetStateDryRun        bool

	epicAliasDelete bool
	epicAliasList   bool
)

func init() {
	epicCreateCmd.Flags().StringVar(&epicCreateBody, "body", "", "Epic description (markdown)")
	epicCreateCmd.Flags().StringVar(&epicCreateRepo, "repo", "", "Repository for legacy epic (creates a GitHub issue-backed epic)")
	epicCreateCmd.Flags().BoolVar(&epicCreateDryRun, "dry-run", false, "Show what would be created without executing")

	epicEditCmd.Flags().StringVar(&epicEditTitle, "title", "", "New title for the epic")
	epicEditCmd.Flags().StringVar(&epicEditBody, "body", "", "New body/description for the epic")
	epicEditCmd.Flags().BoolVar(&epicEditDryRun, "dry-run", false, "Show what would be changed without executing")

	epicDeleteCmd.Flags().BoolVar(&epicDeleteDryRun, "dry-run", false, "Show what would be deleted without executing")

	epicSetStateCmd.Flags().BoolVar(&epicSetStateApplyToIssues, "apply-to-issues", false, "Also update the state of all child issues")
	epicSetStateCmd.Flags().BoolVar(&epicSetStateDryRun, "dry-run", false, "Show what would be changed without executing")

	epicAliasCmd.Flags().BoolVar(&epicAliasDelete, "delete", false, "Remove an existing alias")
	epicAliasCmd.Flags().BoolVar(&epicAliasList, "list", false, "List all epic aliases")

	epicCmd.AddCommand(epicCreateCmd)
	epicCmd.AddCommand(epicEditCmd)
	epicCmd.AddCommand(epicDeleteCmd)
	epicCmd.AddCommand(epicSetStateCmd)
	epicCmd.AddCommand(epicAliasCmd)
}

func resetEpicMutationFlags() {
	epicCreateBody = ""
	epicCreateRepo = ""
	epicCreateDryRun = false

	epicEditTitle = ""
	epicEditBody = ""
	epicEditDryRun = false

	epicDeleteDryRun = false

	epicSetStateApplyToIssues = false
	epicSetStateDryRun = false

	epicAliasDelete = false
	epicAliasList = false
}

// fetchWorkspaceOrgID retrieves the ZenHub organization ID for a workspace.
func fetchWorkspaceOrgID(client *api.Client, workspaceID string) (string, error) {
	data, err := client.Execute(getWorkspaceOrgQuery, map[string]any{
		"workspaceId": workspaceID,
	})
	if err != nil {
		return "", exitcode.General("fetching workspace organization", err)
	}

	var resp struct {
		Workspace struct {
			Organization *struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"zenhubOrganization"`
		} `json:"workspace"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return "", exitcode.General("parsing workspace organization response", err)
	}

	if resp.Workspace.Organization == nil {
		return "", exitcode.General("workspace has no organization", nil)
	}

	return resp.Workspace.Organization.ID, nil
}

// runEpicCreate implements `zh epic create <title>`.
func runEpicCreate(cmd *cobra.Command, args []string) error {
	cfg, err := requireWorkspace()
	if err != nil {
		return err
	}

	client := newClient(cfg, cmd)
	title := args[0]

	if epicCreateRepo != "" {
		return runEpicCreateLegacy(client, cfg, cmd, title)
	}

	return runEpicCreateZenhub(client, cfg, cmd, title)
}

// runEpicCreateZenhub creates a standalone ZenHub epic.
func runEpicCreateZenhub(client *api.Client, cfg *config.Config, cmd *cobra.Command, title string) error {
	w := cmd.OutOrStdout()

	if epicCreateDryRun {
		msg := fmt.Sprintf("Would create epic %q.", title)
		output.MutationSingle(w, output.Yellow(msg))
		fmt.Fprintln(w)
		fmt.Fprintln(w, output.Yellow("  Type: ZenHub Epic"))
		if epicCreateBody != "" {
			body := epicCreateBody
			if len(body) > 60 {
				body = body[:57] + "..."
			}
			fmt.Fprintln(w, output.Yellow(fmt.Sprintf("  Body: %s", body)))
		}
		return nil
	}

	// Fetch org ID
	orgID, err := fetchWorkspaceOrgID(client, cfg.Workspace)
	if err != nil {
		return err
	}

	epicInput := map[string]any{
		"title": title,
	}
	if epicCreateBody != "" {
		epicInput["body"] = epicCreateBody
	}

	input := map[string]any{
		"zenhubOrganizationId": orgID,
		"zenhubEpic":           epicInput,
	}

	data, err := client.Execute(createZenhubEpicMutation, map[string]any{
		"input": input,
	})
	if err != nil {
		return exitcode.General("creating epic", err)
	}

	var resp struct {
		CreateZenhubEpic struct {
			ZenhubEpic struct {
				ID        string `json:"id"`
				Title     string `json:"title"`
				Body      string `json:"body"`
				State     string `json:"state"`
				CreatedAt string `json:"createdAt"`
			} `json:"zenhubEpic"`
		} `json:"createZenhubEpic"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return exitcode.General("parsing create epic response", err)
	}

	created := resp.CreateZenhubEpic.ZenhubEpic

	// Invalidate epic cache
	_ = cache.Clear(resolve.EpicCacheKey(cfg.Workspace))

	if output.IsJSON(outputFormat) {
		return output.JSON(w, created)
	}

	output.MutationSingle(w, output.Green(fmt.Sprintf("Created epic %q.", created.Title)))
	fmt.Fprintln(w)
	fmt.Fprintf(w, "  ID:    %s\n", output.Cyan(created.ID))
	fmt.Fprintf(w, "  Type:  ZenHub Epic\n")
	fmt.Fprintf(w, "  State: %s\n", formatEpicState(created.State))

	return nil
}

// runEpicCreateLegacy creates a legacy epic backed by a GitHub issue.
func runEpicCreateLegacy(client *api.Client, cfg *config.Config, cmd *cobra.Command, title string) error {
	w := cmd.OutOrStdout()

	// Resolve the repository
	repo, err := resolve.LookupRepoWithRefresh(client, cfg.Workspace, epicCreateRepo)
	if err != nil {
		return err
	}

	if epicCreateDryRun {
		msg := fmt.Sprintf("Would create epic %q.", title)
		output.MutationSingle(w, output.Yellow(msg))
		fmt.Fprintln(w)
		fmt.Fprintln(w, output.Yellow("  Type: Legacy Epic (GitHub issue)"))
		fmt.Fprintln(w, output.Yellow(fmt.Sprintf("  Repo: %s/%s", repo.OwnerName, repo.Name)))
		if epicCreateBody != "" {
			body := epicCreateBody
			if len(body) > 60 {
				body = body[:57] + "..."
			}
			fmt.Fprintln(w, output.Yellow(fmt.Sprintf("  Body: %s", body)))
		}
		return nil
	}

	issueInput := map[string]any{
		"repositoryGhId": repo.GhID,
		"title":          title,
	}
	if epicCreateBody != "" {
		issueInput["body"] = epicCreateBody
	}

	data, err := client.Execute(createLegacyEpicMutation, map[string]any{
		"input": map[string]any{
			"issue": issueInput,
		},
	})
	if err != nil {
		return exitcode.General("creating legacy epic", err)
	}

	var resp struct {
		CreateEpic struct {
			Epic struct {
				ID    string `json:"id"`
				Issue struct {
					ID      string `json:"id"`
					Number  int    `json:"number"`
					Title   string `json:"title"`
					HtmlUrl string `json:"htmlUrl"`
					Repo    struct {
						Name      string `json:"name"`
						OwnerName string `json:"ownerName"`
					} `json:"repository"`
				} `json:"issue"`
			} `json:"epic"`
		} `json:"createEpic"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return exitcode.General("parsing create legacy epic response", err)
	}

	created := resp.CreateEpic.Epic

	// Invalidate epic cache
	_ = cache.Clear(resolve.EpicCacheKey(cfg.Workspace))

	if output.IsJSON(outputFormat) {
		return output.JSON(w, created)
	}

	ref := fmt.Sprintf("%s/%s#%d", created.Issue.Repo.OwnerName, created.Issue.Repo.Name, created.Issue.Number)
	output.MutationSingle(w, output.Green(fmt.Sprintf("Created legacy epic %q.", created.Issue.Title)))
	fmt.Fprintln(w)
	fmt.Fprintf(w, "  ID:    %s\n", output.Cyan(created.ID))
	fmt.Fprintf(w, "  Type:  Legacy Epic (GitHub issue)\n")
	fmt.Fprintf(w, "  Issue: %s\n", output.Cyan(ref))

	return nil
}

// validEpicStates maps user input to GraphQL enum values.
var validEpicStates = map[string]string{
	"open":        "OPEN",
	"todo":        "TODO",
	"in_progress": "IN_PROGRESS",
	"in-progress": "IN_PROGRESS",
	"inprogress":  "IN_PROGRESS",
	"closed":      "CLOSED",
}

// runEpicEdit implements `zh epic edit <epic>`.
func runEpicEdit(cmd *cobra.Command, args []string) error {
	if epicEditTitle == "" && epicEditBody == "" {
		return exitcode.Usage("at least one of --title or --body must be provided")
	}

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
		return exitcode.Usage(fmt.Sprintf("epic %q is a legacy epic (backed by a GitHub issue) — edit it via GitHub instead", resolved.Title))
	}

	if epicEditDryRun {
		msg := fmt.Sprintf("Would update epic %q.", resolved.Title)
		output.MutationSingle(w, output.Yellow(msg))
		fmt.Fprintln(w)
		if epicEditTitle != "" {
			fmt.Fprintln(w, output.Yellow(fmt.Sprintf("  Title: %s", epicEditTitle)))
		}
		if epicEditBody != "" {
			body := epicEditBody
			if len(body) > 60 {
				body = body[:57] + "..."
			}
			fmt.Fprintln(w, output.Yellow(fmt.Sprintf("  Body: %s", body)))
		}
		return nil
	}

	input := map[string]any{
		"zenhubEpicId": resolved.ID,
	}
	if epicEditTitle != "" {
		input["title"] = epicEditTitle
	}
	if epicEditBody != "" {
		input["body"] = epicEditBody
	}

	data, err := client.Execute(updateZenhubEpicMutation, map[string]any{
		"input": input,
	})
	if err != nil {
		return exitcode.General("updating epic", err)
	}

	var resp struct {
		UpdateZenhubEpic struct {
			ZenhubEpic struct {
				ID        string `json:"id"`
				Title     string `json:"title"`
				Body      string `json:"body"`
				State     string `json:"state"`
				UpdatedAt string `json:"updatedAt"`
			} `json:"zenhubEpic"`
		} `json:"updateZenhubEpic"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return exitcode.General("parsing update epic response", err)
	}

	updated := resp.UpdateZenhubEpic.ZenhubEpic

	// Invalidate epic cache
	_ = cache.Clear(resolve.EpicCacheKey(cfg.Workspace))

	if output.IsJSON(outputFormat) {
		return output.JSON(w, updated)
	}

	output.MutationSingle(w, output.Green(fmt.Sprintf("Updated epic %q.", updated.Title)))
	fmt.Fprintln(w)
	if epicEditTitle != "" {
		fmt.Fprintf(w, "  Title: %s\n", updated.Title)
	}
	if epicEditBody != "" {
		fmt.Fprintf(w, "  Body:  updated\n")
	}

	return nil
}

// runEpicDelete implements `zh epic delete <epic>`.
func runEpicDelete(cmd *cobra.Command, args []string) error {
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
		return exitcode.Usage(fmt.Sprintf("epic %q is a legacy epic (backed by a GitHub issue) — delete it via GitHub instead", resolved.Title))
	}

	// Fetch child issue count for informational output
	detailData, err := client.Execute(epicChildIssueCountQuery, map[string]any{
		"id":          resolved.ID,
		"workspaceId": cfg.Workspace,
	})
	if err != nil {
		return exitcode.General("fetching epic details", err)
	}

	var detailResp struct {
		Node *struct {
			ID          string `json:"id"`
			Title       string `json:"title"`
			State       string `json:"state"`
			ChildIssues struct {
				TotalCount int `json:"totalCount"`
			} `json:"childIssues"`
		} `json:"node"`
	}
	if err := json.Unmarshal(detailData, &detailResp); err != nil {
		return exitcode.General("parsing epic details", err)
	}

	childCount := 0
	state := ""
	if detailResp.Node != nil {
		childCount = detailResp.Node.ChildIssues.TotalCount
		state = detailResp.Node.State
	}

	if epicDeleteDryRun {
		msg := fmt.Sprintf("Would delete epic %q.", resolved.Title)
		output.MutationSingle(w, output.Yellow(msg))
		fmt.Fprintln(w)
		fmt.Fprintln(w, output.Yellow(fmt.Sprintf("  ID:           %s", resolved.ID)))
		if state != "" {
			fmt.Fprintln(w, output.Yellow(fmt.Sprintf("  State:        %s", strings.ToLower(state))))
		}
		fmt.Fprintln(w, output.Yellow(fmt.Sprintf("  Child issues: %d (will be removed from epic, not deleted)", childCount)))
		return nil
	}

	data, err := client.Execute(deleteZenhubEpicMutation, map[string]any{
		"input": map[string]any{
			"zenhubEpicId": resolved.ID,
		},
	})
	if err != nil {
		return exitcode.General("deleting epic", err)
	}

	var resp struct {
		DeleteZenhubEpic struct {
			ZenhubEpicID string `json:"zenhubEpicId"`
		} `json:"deleteZenhubEpic"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return exitcode.General("parsing delete epic response", err)
	}

	// Invalidate epic cache
	_ = cache.Clear(resolve.EpicCacheKey(cfg.Workspace))

	if output.IsJSON(outputFormat) {
		return output.JSON(w, map[string]any{
			"deleted":     resolved.Title,
			"id":          resp.DeleteZenhubEpic.ZenhubEpicID,
			"childIssues": childCount,
		})
	}

	output.MutationSingle(w, output.Green(fmt.Sprintf("Deleted epic %q.", resolved.Title)))
	if childCount > 0 {
		fmt.Fprintf(w, "%d child issue(s) removed from epic.\n", childCount)
	}

	return nil
}

// runEpicSetState implements `zh epic set-state <epic> <state>`.
func runEpicSetState(cmd *cobra.Command, args []string) error {
	cfg, err := requireWorkspace()
	if err != nil {
		return err
	}

	client := newClient(cfg, cmd)
	w := cmd.OutOrStdout()

	// Validate state
	stateInput := strings.ToLower(args[1])
	graphqlState, ok := validEpicStates[stateInput]
	if !ok {
		return exitcode.Usage(fmt.Sprintf("invalid state %q — valid states: open, todo, in_progress, closed", args[1]))
	}

	// Resolve the epic
	resolved, err := resolve.Epic(client, cfg.Workspace, args[0], cfg.Aliases.Epics)
	if err != nil {
		return err
	}

	if resolved.Type == "legacy" {
		return exitcode.Usage(fmt.Sprintf("epic %q is a legacy epic (backed by a GitHub issue) — change its state via GitHub instead", resolved.Title))
	}

	if epicSetStateDryRun {
		msg := fmt.Sprintf("Would set state of epic %q to %s.", resolved.Title, strings.ToLower(graphqlState))
		output.MutationSingle(w, output.Yellow(msg))
		if epicSetStateApplyToIssues {
			fmt.Fprintln(w)
			fmt.Fprintln(w, output.Yellow("  Also applying state change to child issues."))
		}
		return nil
	}

	input := map[string]any{
		"zenhubEpicId": resolved.ID,
		"state":        graphqlState,
	}
	if epicSetStateApplyToIssues {
		input["applyToIssues"] = true
	}

	data, err := client.Execute(updateZenhubEpicStateMutation, map[string]any{
		"input": input,
	})
	if err != nil {
		return exitcode.General("updating epic state", err)
	}

	var resp struct {
		UpdateZenhubEpicState struct {
			ZenhubEpic struct {
				ID    string `json:"id"`
				Title string `json:"title"`
				State string `json:"state"`
			} `json:"zenhubEpic"`
		} `json:"updateZenhubEpicState"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return exitcode.General("parsing update epic state response", err)
	}

	updated := resp.UpdateZenhubEpicState.ZenhubEpic

	// Invalidate epic cache
	_ = cache.Clear(resolve.EpicCacheKey(cfg.Workspace))

	if output.IsJSON(outputFormat) {
		return output.JSON(w, updated)
	}

	output.MutationSingle(w, output.Green(fmt.Sprintf("Set state of epic %q to %s.", updated.Title, formatEpicState(updated.State))))
	if epicSetStateApplyToIssues {
		fmt.Fprintln(w, "Also applied state change to child issues.")
	}

	return nil
}

// runEpicAlias implements `zh epic alias <epic> <alias>`.
func runEpicAlias(cmd *cobra.Command, args []string) error {
	cfg, err := requireWorkspace()
	if err != nil {
		return err
	}

	w := cmd.OutOrStdout()

	// --list: show all epic aliases
	if epicAliasList {
		if output.IsJSON(outputFormat) {
			return output.JSON(w, cfg.Aliases.Epics)
		}

		if len(cfg.Aliases.Epics) == 0 {
			fmt.Fprintln(w, "No epic aliases configured.")
			return nil
		}

		lw := output.NewListWriter(w, "ALIAS", "EPIC")
		for alias, name := range cfg.Aliases.Epics {
			lw.Row(alias, name)
		}
		lw.FlushWithFooter(fmt.Sprintf("Total: %d alias(es)", len(cfg.Aliases.Epics)))
		return nil
	}

	// --delete: remove an alias
	if epicAliasDelete {
		if len(args) != 1 {
			return exitcode.Usage("usage: zh epic alias --delete <alias>")
		}
		alias := args[0]

		if cfg.Aliases.Epics == nil {
			return exitcode.NotFoundError(fmt.Sprintf("alias %q not found", alias))
		}

		if _, ok := cfg.Aliases.Epics[alias]; !ok {
			return exitcode.NotFoundError(fmt.Sprintf("alias %q not found", alias))
		}

		delete(cfg.Aliases.Epics, alias)
		if err := config.Write(cfg); err != nil {
			return exitcode.General("saving config", err)
		}

		output.MutationSingle(w, fmt.Sprintf("Removed alias %q.", alias))
		return nil
	}

	// Set an alias: requires exactly 2 args
	if len(args) != 2 {
		return exitcode.Usage("usage: zh epic alias <epic> <alias>")
	}

	epicName := args[0]
	alias := args[1]

	// Validate the epic exists
	client := newClient(cfg, cmd)
	resolved, err := resolve.Epic(client, cfg.Workspace, epicName, cfg.Aliases.Epics)
	if err != nil {
		return err
	}

	// Initialize map if needed
	if cfg.Aliases.Epics == nil {
		cfg.Aliases.Epics = make(map[string]string)
	}

	// Check if alias already exists
	if existing, ok := cfg.Aliases.Epics[alias]; ok {
		if strings.EqualFold(existing, resolved.Title) || existing == resolved.ID {
			fmt.Fprintf(w, "Alias %q already points to %q.\n", alias, resolved.Title)
			return nil
		}
		return exitcode.Usage(fmt.Sprintf("alias %q already exists (points to %q) — use --delete first to remove it", alias, existing))
	}

	// Store alias mapping to epic title
	cfg.Aliases.Epics[alias] = resolved.Title
	if err := config.Write(cfg); err != nil {
		return exitcode.General("saving config", err)
	}

	output.MutationSingle(w, fmt.Sprintf("Alias %q -> %q.", alias, resolved.Title))
	return nil
}
