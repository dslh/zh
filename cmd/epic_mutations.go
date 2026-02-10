package cmd

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/dslh/zh/internal/api"
	"github.com/dslh/zh/internal/cache"
	"github.com/dslh/zh/internal/config"
	"github.com/dslh/zh/internal/exitcode"
	"github.com/dslh/zh/internal/gh"
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

const updateZenhubEpicDatesMutation = `mutation UpdateZenhubEpicDates($input: UpdateZenhubEpicDatesInput!) {
  updateZenhubEpicDates(input: $input) {
    zenhubEpic {
      id
      title
      startOn
      endOn
    }
  }
}`

const updateLegacyEpicDatesMutation = `mutation UpdateEpicDates($input: UpdateEpicDatesInput!) {
  updateEpicDates(input: $input) {
    epic {
      id
      startOn
      endOn
      issue {
        title
        number
      }
    }
  }
}`

const addIssuesToZenhubEpicsMutation = `mutation AddIssuesToZenhubEpics($input: AddIssuesToZenhubEpicsInput!) {
  addIssuesToZenhubEpics(input: $input) {
    zenhubEpics {
      id
      title
    }
  }
}`

const removeIssuesFromZenhubEpicsMutation = `mutation RemoveIssuesFromZenhubEpics($input: RemoveIssuesFromZenhubEpicsInput!) {
  removeIssuesFromZenhubEpics(input: $input) {
    zenhubEpics {
      id
      title
    }
  }
}`

const epicChildIssueIDsQuery = `query GetEpicChildIssueIDs($id: ID!, $workspaceId: ID!, $first: Int!, $after: String) {
  node(id: $id) {
    ... on ZenhubEpic {
      id
      title
      childIssues(workspaceId: $workspaceId, first: $first, after: $after) {
        totalCount
        pageInfo {
          hasNextPage
          endCursor
        }
        nodes {
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
  }
}`

const setMultipleEstimatesOnZenhubEpicsMutation = `mutation SetEstimateOnZenhubEpics($input: SetMultipleEstimatesOnZenhubEpicsInput!) {
  setMultipleEstimatesOnZenhubEpics(input: $input) {
    zenhubEpics {
      id
      title
      estimate {
        value
      }
    }
  }
}`

const epicEstimateQuery = `query GetZenhubEpicEstimate($id: ID!) {
  node(id: $id) {
    ... on ZenhubEpic {
      id
      title
      estimate { value }
    }
  }
}`

// GitHub GraphQL queries/mutations for legacy epic operations.
//
// Legacy epics are backed by a GitHub issue — editing their title/body and
// changing their open/closed state requires GitHub API access.

const legacyEpicGitHubIssueQuery = `query GetGitHubIssue($owner: String!, $repo: String!, $number: Int!) {
  repository(owner: $owner, name: $repo) {
    issue(number: $number) {
      id
      title
      body
      state
    }
  }
}`

const legacyEpicUpdateIssueMutation = `mutation UpdateIssue($input: UpdateIssueInput!) {
  updateIssue(input: $input) {
    issue {
      id
      title
      body
      state
    }
  }
}`

// requireLegacyEpicGitHubID fetches the GitHub node ID for a legacy epic's
// backing issue, which is needed for GitHub GraphQL mutations.
func requireLegacyEpicGitHubID(ghClient *gh.Client, resolved *resolve.EpicResult) (string, error) {
	data, err := ghClient.Execute(legacyEpicGitHubIssueQuery, map[string]any{
		"owner":  resolved.RepoOwner,
		"repo":   resolved.RepoName,
		"number": resolved.IssueNumber,
	})
	if err != nil {
		return "", exitcode.General("fetching GitHub issue for legacy epic", err)
	}

	var resp struct {
		Repository struct {
			Issue *struct {
				ID string `json:"id"`
			} `json:"issue"`
		} `json:"repository"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return "", exitcode.General("parsing GitHub issue response", err)
	}

	if resp.Repository.Issue == nil {
		return "", exitcode.NotFoundError(fmt.Sprintf("GitHub issue %s/%s#%d not found",
			resolved.RepoOwner, resolved.RepoName, resolved.IssueNumber))
	}

	return resp.Repository.Issue.ID, nil
}

// legacyEpicRef returns the short issue reference for a legacy epic.
func legacyEpicRef(resolved *resolve.EpicResult) string {
	return fmt.Sprintf("%s/%s#%d", resolved.RepoOwner, resolved.RepoName, resolved.IssueNumber)
}

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

var epicSetDatesCmd = &cobra.Command{
	Use:   "set-dates <epic>",
	Short: "Set start and/or end dates on an epic",
	Long: `Set start and/or end dates on an epic.

At least one of --start, --end, --clear-start, or --clear-end must be provided.
Dates use YYYY-MM-DD format.

Examples:
  zh epic set-dates "Q1 Roadmap" --start=2025-03-01 --end=2025-03-31
  zh epic set-dates "Q1 Roadmap" --clear-end
  zh epic set-dates "Q1 Roadmap" --start=2025-04-01 --clear-end`,
	Args: cobra.ExactArgs(1),
	RunE: runEpicSetDates,
}

var epicAddCmd = &cobra.Command{
	Use:   "add <epic> <issue>...",
	Short: "Add issues to an epic",
	Long: `Add one or more issues to an epic (ZenHub or legacy).

Issues can be specified as repo#number, owner/repo#number, ZenHub IDs,
or bare numbers with --repo.

Examples:
  zh epic add "Q1 Roadmap" task-tracker#1 task-tracker#2
  zh epic add "Q1 Roadmap" --repo=task-tracker 1 2 3`,
	Args: cobra.MinimumNArgs(2),
	RunE: runEpicAdd,
}

var epicRemoveCmd = &cobra.Command{
	Use:   "remove <epic> <issue>...",
	Short: "Remove issues from an epic",
	Long: `Remove one or more issues from an epic (ZenHub or legacy).

Issues can be specified as repo#number, owner/repo#number, ZenHub IDs,
or bare numbers with --repo. Use --all to remove all issues.

Examples:
  zh epic remove "Q1 Roadmap" task-tracker#1 task-tracker#2
  zh epic remove "Q1 Roadmap" --all`,
	Args: cobra.MinimumNArgs(1),
	RunE: runEpicRemove,
}

var epicEstimateCmd = &cobra.Command{
	Use:   "estimate <epic> [value]",
	Short: "Set or clear the estimate on an epic",
	Long: `Set or clear the estimate on a ZenHub epic.

Provide a value to set the estimate. Omit the value to clear it.

Examples:
  zh epic estimate "Q1 Roadmap" 13
  zh epic estimate "Q1 Roadmap"      # clears the estimate`,
	Args: cobra.RangeArgs(1, 2),
	RunE: runEpicEstimate,
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

	epicSetDatesStart      string
	epicSetDatesEnd        string
	epicSetDatesClearStart bool
	epicSetDatesClearEnd   bool
	epicSetDatesDryRun     bool

	epicAddDryRun          bool
	epicAddRepo            string
	epicAddContinueOnError bool

	epicRemoveDryRun          bool
	epicRemoveRepo            string
	epicRemoveAll             bool
	epicRemoveContinueOnError bool

	epicEstimateDryRun bool
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

	epicSetDatesCmd.Flags().StringVar(&epicSetDatesStart, "start", "", "Start date (YYYY-MM-DD)")
	epicSetDatesCmd.Flags().StringVar(&epicSetDatesEnd, "end", "", "End date (YYYY-MM-DD)")
	epicSetDatesCmd.Flags().BoolVar(&epicSetDatesClearStart, "clear-start", false, "Clear the start date")
	epicSetDatesCmd.Flags().BoolVar(&epicSetDatesClearEnd, "clear-end", false, "Clear the end date")
	epicSetDatesCmd.Flags().BoolVar(&epicSetDatesDryRun, "dry-run", false, "Show what would be changed without executing")

	epicAddCmd.Flags().BoolVar(&epicAddDryRun, "dry-run", false, "Show what would be added without executing")
	epicAddCmd.Flags().StringVar(&epicAddRepo, "repo", "", "Repository context for bare issue numbers")
	epicAddCmd.Flags().BoolVar(&epicAddContinueOnError, "continue-on-error", false, "Continue processing remaining issues after a resolution error")

	epicRemoveCmd.Flags().BoolVar(&epicRemoveDryRun, "dry-run", false, "Show what would be removed without executing")
	epicRemoveCmd.Flags().StringVar(&epicRemoveRepo, "repo", "", "Repository context for bare issue numbers")
	epicRemoveCmd.Flags().BoolVar(&epicRemoveAll, "all", false, "Remove all issues from the epic")
	epicRemoveCmd.Flags().BoolVar(&epicRemoveContinueOnError, "continue-on-error", false, "Continue processing remaining issues after a resolution error")

	epicEstimateCmd.Flags().BoolVar(&epicEstimateDryRun, "dry-run", false, "Show what would be changed without executing")

	epicCmd.AddCommand(epicCreateCmd)
	epicCmd.AddCommand(epicEditCmd)
	epicCmd.AddCommand(epicDeleteCmd)
	epicCmd.AddCommand(epicSetStateCmd)
	epicCmd.AddCommand(epicAliasCmd)
	epicCmd.AddCommand(epicSetDatesCmd)
	epicCmd.AddCommand(epicAddCmd)
	epicCmd.AddCommand(epicRemoveCmd)
	epicCmd.AddCommand(epicEstimateCmd)
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

	epicSetDatesStart = ""
	epicSetDatesEnd = ""
	epicSetDatesClearStart = false
	epicSetDatesClearEnd = false
	epicSetDatesDryRun = false

	epicAddDryRun = false
	epicAddRepo = ""
	epicAddContinueOnError = false

	epicRemoveDryRun = false
	epicRemoveRepo = ""
	epicRemoveAll = false
	epicRemoveContinueOnError = false

	epicEstimateDryRun = false
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
		details := []output.DetailLine{
			{Key: "Type", Value: "ZenHub Epic"},
		}
		if epicCreateBody != "" {
			body := epicCreateBody
			if len(body) > 60 {
				body = body[:57] + "..."
			}
			details = append(details, output.DetailLine{Key: "Body", Value: body})
		}
		output.MutationDryRunDetail(w, msg, details)
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
		details := []output.DetailLine{
			{Key: "Type", Value: "Legacy Epic (GitHub issue)"},
			{Key: "Repo", Value: fmt.Sprintf("%s/%s", repo.OwnerName, repo.Name)},
		}
		if epicCreateBody != "" {
			body := epicCreateBody
			if len(body) > 60 {
				body = body[:57] + "..."
			}
			details = append(details, output.DetailLine{Key: "Body", Value: body})
		}
		output.MutationDryRunDetail(w, msg, details)
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
		ghClient := newGitHubClient(cfg, cmd)
		if ghClient == nil {
			return exitcode.Generalf("epic %q is a legacy epic (backed by GitHub issue %s/%s#%d) — GitHub access is required to edit it\n\nConfigure GitHub access with: zh",
				resolved.Title, resolved.RepoOwner, resolved.RepoName, resolved.IssueNumber)
		}
		return runEpicEditLegacy(ghClient, w, resolved)
	}

	if epicEditDryRun {
		msg := fmt.Sprintf("Would update epic %q.", resolved.Title)
		var details []output.DetailLine
		if epicEditTitle != "" {
			details = append(details, output.DetailLine{Key: "Title", Value: epicEditTitle})
		}
		if epicEditBody != "" {
			body := epicEditBody
			if len(body) > 60 {
				body = body[:57] + "..."
			}
			details = append(details, output.DetailLine{Key: "Body", Value: body})
		}
		output.MutationDryRunDetail(w, msg, details)
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

// runEpicEditLegacy edits a legacy epic's title/body via the GitHub API.
func runEpicEditLegacy(ghClient *gh.Client, w writerFlusher, resolved *resolve.EpicResult) error {
	ref := legacyEpicRef(resolved)

	if epicEditDryRun {
		msg := fmt.Sprintf("Would update legacy epic %q (%s) via GitHub.", resolved.Title, ref)
		var details []output.DetailLine
		if epicEditTitle != "" {
			details = append(details, output.DetailLine{Key: "Title", Value: epicEditTitle})
		}
		if epicEditBody != "" {
			body := epicEditBody
			if len(body) > 60 {
				body = body[:57] + "..."
			}
			details = append(details, output.DetailLine{Key: "Body", Value: body})
		}
		output.MutationDryRunDetail(w, msg, details)
		return nil
	}

	// Get the GitHub node ID
	ghNodeID, err := requireLegacyEpicGitHubID(ghClient, resolved)
	if err != nil {
		return err
	}

	input := map[string]any{
		"id": ghNodeID,
	}
	if epicEditTitle != "" {
		input["title"] = epicEditTitle
	}
	if epicEditBody != "" {
		input["body"] = epicEditBody
	}

	data, err := ghClient.Execute(legacyEpicUpdateIssueMutation, map[string]any{
		"input": input,
	})
	if err != nil {
		return exitcode.General("updating legacy epic via GitHub", err)
	}

	var resp struct {
		UpdateIssue struct {
			Issue struct {
				ID    string `json:"id"`
				Title string `json:"title"`
				Body  string `json:"body"`
				State string `json:"state"`
			} `json:"issue"`
		} `json:"updateIssue"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return exitcode.General("parsing GitHub update issue response", err)
	}

	updated := resp.UpdateIssue.Issue

	if output.IsJSON(outputFormat) {
		return output.JSON(w, map[string]any{
			"id":    resolved.ID,
			"issue": ref,
			"title": updated.Title,
			"body":  updated.Body,
			"state": updated.State,
		})
	}

	output.MutationSingle(w, output.Green(fmt.Sprintf("Updated legacy epic %q (%s).", updated.Title, ref)))
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
		return exitcode.Usage(fmt.Sprintf("epic %q is a legacy epic (backed by GitHub issue %s/%s#%d) — delete it via GitHub instead",
			resolved.Title, resolved.RepoOwner, resolved.RepoName, resolved.IssueNumber))
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
		if output.IsJSON(outputFormat) {
			return output.JSON(w, map[string]any{
				"dryRun":      true,
				"deleted":     resolved.Title,
				"id":          resolved.ID,
				"state":       strings.ToLower(state),
				"childIssues": childCount,
			})
		}
		msg := fmt.Sprintf("Would delete epic %q.", resolved.Title)
		details := []output.DetailLine{
			{Key: "ID", Value: resolved.ID},
		}
		if state != "" {
			details = append(details, output.DetailLine{Key: "State", Value: strings.ToLower(state)})
		}
		details = append(details, output.DetailLine{Key: "Child issues", Value: fmt.Sprintf("%d (will be removed from epic, not deleted)", childCount)})
		output.MutationDryRunDetail(w, msg, details)
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
		ghClient := newGitHubClient(cfg, cmd)
		if ghClient == nil {
			return exitcode.Generalf("epic %q is a legacy epic (backed by GitHub issue %s/%s#%d) — GitHub access is required to change its state\n\nConfigure GitHub access with: zh",
				resolved.Title, resolved.RepoOwner, resolved.RepoName, resolved.IssueNumber)
		}
		return runEpicSetStateLegacy(ghClient, w, resolved, graphqlState)
	}

	if epicSetStateDryRun {
		msg := fmt.Sprintf("Would set state of epic %q to %s.", resolved.Title, strings.ToLower(graphqlState))
		var details []output.DetailLine
		if epicSetStateApplyToIssues {
			details = append(details, output.DetailLine{Key: "Note", Value: "Also applying state change to child issues"})
		}
		output.MutationDryRunDetail(w, msg, details)
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

// runEpicSetStateLegacy changes the state of a legacy epic via the GitHub API.
// Legacy epic state maps to GitHub issue state: CLOSED means closed, anything
// else means open. The --apply-to-issues flag is not supported for legacy epics.
func runEpicSetStateLegacy(ghClient *gh.Client, w writerFlusher, resolved *resolve.EpicResult, graphqlState string) error {
	ref := legacyEpicRef(resolved)

	// Map ZenHub epic states to GitHub issue states
	ghState := "OPEN"
	if graphqlState == "CLOSED" {
		ghState = "CLOSED"
	}

	if epicSetStateApplyToIssues {
		fmt.Fprintln(w, output.Yellow("Warning: --apply-to-issues is not supported for legacy epics."))
	}

	if epicSetStateDryRun {
		msg := fmt.Sprintf("Would set state of legacy epic %q (%s) to %s via GitHub.", resolved.Title, ref, strings.ToLower(ghState))
		if ghState == "OPEN" && graphqlState != "OPEN" {
			msg += fmt.Sprintf("\nNote: GitHub issues only support open/closed — %q maps to open.", strings.ToLower(graphqlState))
		}
		output.MutationDryRunDetail(w, msg, nil)
		return nil
	}

	// Get the GitHub node ID
	ghNodeID, err := requireLegacyEpicGitHubID(ghClient, resolved)
	if err != nil {
		return err
	}

	data, err := ghClient.Execute(legacyEpicUpdateIssueMutation, map[string]any{
		"input": map[string]any{
			"id":    ghNodeID,
			"state": ghState,
		},
	})
	if err != nil {
		return exitcode.General("updating legacy epic state via GitHub", err)
	}

	var resp struct {
		UpdateIssue struct {
			Issue struct {
				ID    string `json:"id"`
				Title string `json:"title"`
				State string `json:"state"`
			} `json:"issue"`
		} `json:"updateIssue"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return exitcode.General("parsing GitHub update issue response", err)
	}

	updated := resp.UpdateIssue.Issue

	if output.IsJSON(outputFormat) {
		return output.JSON(w, map[string]any{
			"id":    resolved.ID,
			"issue": ref,
			"title": updated.Title,
			"state": strings.ToLower(updated.State),
		})
	}

	output.MutationSingle(w, output.Green(fmt.Sprintf("Set state of legacy epic %q (%s) to %s.", updated.Title, ref, strings.ToLower(updated.State))))
	if ghState == "OPEN" && graphqlState != "OPEN" {
		fmt.Fprintf(w, "Note: GitHub issues only support open/closed — %q maps to open.\n", strings.ToLower(graphqlState))
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

// parseDate validates a date string is in YYYY-MM-DD format.
func parseDate(s string) (string, error) {
	_, err := time.Parse("2006-01-02", s)
	if err != nil {
		return "", exitcode.Usage(fmt.Sprintf("invalid date %q — expected YYYY-MM-DD format", s))
	}
	return s, nil
}

// runEpicSetDates implements `zh epic set-dates <epic>`.
func runEpicSetDates(cmd *cobra.Command, args []string) error {
	if epicSetDatesStart == "" && epicSetDatesEnd == "" && !epicSetDatesClearStart && !epicSetDatesClearEnd {
		return exitcode.Usage("at least one of --start, --end, --clear-start, or --clear-end must be provided")
	}

	if epicSetDatesStart != "" && epicSetDatesClearStart {
		return exitcode.Usage("cannot set --start and --clear-start at the same time")
	}
	if epicSetDatesEnd != "" && epicSetDatesClearEnd {
		return exitcode.Usage("cannot set --end and --clear-end at the same time")
	}

	// Validate dates
	if epicSetDatesStart != "" {
		if _, err := parseDate(epicSetDatesStart); err != nil {
			return err
		}
	}
	if epicSetDatesEnd != "" {
		if _, err := parseDate(epicSetDatesEnd); err != nil {
			return err
		}
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

	// Build input
	var startOn any
	var endOn any
	if epicSetDatesStart != "" {
		startOn = epicSetDatesStart
	} else if epicSetDatesClearStart {
		startOn = nil
	}
	if epicSetDatesEnd != "" {
		endOn = epicSetDatesEnd
	} else if epicSetDatesClearEnd {
		endOn = nil
	}

	if epicSetDatesDryRun {
		msg := fmt.Sprintf("Would update dates on epic %q.", resolved.Title)
		var details []output.DetailLine
		if epicSetDatesStart != "" {
			details = append(details, output.DetailLine{Key: "Start", Value: epicSetDatesStart})
		} else if epicSetDatesClearStart {
			details = append(details, output.DetailLine{Key: "Start", Value: "(clear)"})
		}
		if epicSetDatesEnd != "" {
			details = append(details, output.DetailLine{Key: "End", Value: epicSetDatesEnd})
		} else if epicSetDatesClearEnd {
			details = append(details, output.DetailLine{Key: "End", Value: "(clear)"})
		}
		output.MutationDryRunDetail(w, msg, details)
		return nil
	}

	if resolved.Type == "zenhub" {
		return runEpicSetDatesZenhub(client, cfg, w, resolved, startOn, endOn)
	}
	return runEpicSetDatesLegacy(client, w, resolved, startOn, endOn)
}

// runEpicSetDatesZenhub sets dates on a ZenHub epic.
func runEpicSetDatesZenhub(client *api.Client, cfg *config.Config, w writerFlusher, resolved *resolve.EpicResult, startOn, endOn any) error {
	input := map[string]any{
		"zenhubEpicId": resolved.ID,
	}
	if epicSetDatesStart != "" || epicSetDatesClearStart {
		input["startOn"] = startOn
	}
	if epicSetDatesEnd != "" || epicSetDatesClearEnd {
		input["endOn"] = endOn
	}

	data, err := client.Execute(updateZenhubEpicDatesMutation, map[string]any{
		"input": input,
	})
	if err != nil {
		return exitcode.General("updating epic dates", err)
	}

	var resp struct {
		UpdateZenhubEpicDates struct {
			ZenhubEpic struct {
				ID      string  `json:"id"`
				Title   string  `json:"title"`
				StartOn *string `json:"startOn"`
				EndOn   *string `json:"endOn"`
			} `json:"zenhubEpic"`
		} `json:"updateZenhubEpicDates"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return exitcode.General("parsing update dates response", err)
	}

	updated := resp.UpdateZenhubEpicDates.ZenhubEpic

	if output.IsJSON(outputFormat) {
		return output.JSON(w, updated)
	}

	output.MutationSingle(w, output.Green(fmt.Sprintf("Updated dates on epic %q.", updated.Title)))
	fmt.Fprintln(w)
	fmt.Fprintf(w, "  Start: %s\n", formatDatePointer(updated.StartOn))
	fmt.Fprintf(w, "  End:   %s\n", formatDatePointer(updated.EndOn))

	return nil
}

// runEpicSetDatesLegacy sets dates on a legacy epic.
func runEpicSetDatesLegacy(client *api.Client, w writerFlusher, resolved *resolve.EpicResult, startOn, endOn any) error {
	input := map[string]any{
		"epicId": resolved.ID,
	}
	if epicSetDatesStart != "" || epicSetDatesClearStart {
		input["startOn"] = startOn
	}
	if epicSetDatesEnd != "" || epicSetDatesClearEnd {
		input["endOn"] = endOn
	}

	data, err := client.Execute(updateLegacyEpicDatesMutation, map[string]any{
		"input": input,
	})
	if err != nil {
		return exitcode.General("updating epic dates", err)
	}

	var resp struct {
		UpdateEpicDates struct {
			Epic struct {
				ID      string  `json:"id"`
				StartOn *string `json:"startOn"`
				EndOn   *string `json:"endOn"`
				Issue   struct {
					Title  string `json:"title"`
					Number int    `json:"number"`
				} `json:"issue"`
			} `json:"epic"`
		} `json:"updateEpicDates"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return exitcode.General("parsing update dates response", err)
	}

	updated := resp.UpdateEpicDates.Epic

	if output.IsJSON(outputFormat) {
		return output.JSON(w, updated)
	}

	output.MutationSingle(w, output.Green(fmt.Sprintf("Updated dates on epic %q.", resolved.Title)))
	fmt.Fprintln(w)
	fmt.Fprintf(w, "  Start: %s\n", formatDatePointer(updated.StartOn))
	fmt.Fprintf(w, "  End:   %s\n", formatDatePointer(updated.EndOn))

	return nil
}

// formatDatePointer returns a date string or "None" for nil.
func formatDatePointer(d *string) string {
	if d == nil || *d == "" {
		return "None"
	}
	return *d
}

// resolvedEpicIssue holds minimal info about an issue resolved for epic add/remove.
type resolvedEpicIssue struct {
	ID        string
	Number    int
	RepoGhID  int
	RepoName  string
	RepoOwner string
	Title     string
}

func (r *resolvedEpicIssue) Ref() string {
	return fmt.Sprintf("%s#%d", r.RepoName, r.Number)
}

// issueResolveForEpicQuery fetches title and repo info for an issue by ID.
const issueResolveForEpicQuery = `query GetIssueForEpic($issueId: ID!) {
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

// resolveIssueForEpic resolves an issue identifier and fetches its title.
func resolveIssueForEpic(client *api.Client, workspaceID, identifier, repoFlag string, ghClient *gh.Client) (*resolvedEpicIssue, error) {
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

	return &resolvedEpicIssue{
		ID:        resp.Node.ID,
		Number:    resp.Node.Number,
		RepoGhID:  result.RepoGhID,
		Title:     resp.Node.Title,
		RepoName:  resp.Node.Repository.Name,
		RepoOwner: resp.Node.Repository.OwnerName,
	}, nil
}

// runEpicAdd implements `zh epic add <epic> <issue>...`.
func runEpicAdd(cmd *cobra.Command, args []string) error {
	cfg, err := requireWorkspace()
	if err != nil {
		return err
	}

	client := newClient(cfg, cmd)
	w := cmd.OutOrStdout()
	ghClient := newGitHubClient(cfg, cmd)

	// Resolve the epic
	resolved, err := resolve.Epic(client, cfg.Workspace, args[0], cfg.Aliases.Epics)
	if err != nil {
		return err
	}

	// Resolve issue identifiers
	issueArgs := args[1:]
	var issues []resolvedEpicIssue
	var failed []output.FailedItem

	for _, arg := range issueArgs {
		issue, err := resolveIssueForEpic(client, cfg.Workspace, arg, epicAddRepo, ghClient)
		if err != nil {
			if epicAddContinueOnError {
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

	if resolved.Type == "legacy" {
		return runEpicAddLegacy(client, cfg, w, resolved, issues, failed)
	}

	// Dry run
	if epicAddDryRun {
		return renderEpicAddDryRun(w, resolved, issues, failed)
	}

	// Execute the mutation
	issueIDs := make([]string, len(issues))
	for i, iss := range issues {
		issueIDs[i] = iss.ID
	}

	data, err := client.Execute(addIssuesToZenhubEpicsMutation, map[string]any{
		"input": map[string]any{
			"zenhubEpicIds": []string{resolved.ID},
			"issueIds":      issueIDs,
		},
	})
	if err != nil {
		return exitcode.General("adding issues to epic", err)
	}

	// Parse response (just confirm mutation succeeded)
	var resp struct {
		AddIssuesToZenhubEpics struct {
			ZenhubEpics []struct {
				ID    string `json:"id"`
				Title string `json:"title"`
			} `json:"zenhubEpics"`
		} `json:"addIssuesToZenhubEpics"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return exitcode.General("parsing add issues response", err)
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
			"epic":  map[string]any{"id": resolved.ID, "title": resolved.Title},
			"added": formatEpicIssueItemsJSON(issues),
		})
	}

	// Render output
	if len(failed) > 0 {
		totalAttempted := len(succeeded) + len(failed)
		header := output.Green(fmt.Sprintf("Added %d of %d issue(s) to epic %q.", len(succeeded), totalAttempted, resolved.Title))
		output.MutationPartialFailure(w, header, succeeded, failed)
		return exitcode.Generalf("some issues failed to resolve")
	} else if len(succeeded) == 1 {
		output.MutationSingle(w, output.Green(fmt.Sprintf("Added %s to epic %q.", succeeded[0].Ref, resolved.Title)))
	} else {
		header := output.Green(fmt.Sprintf("Added %d issue(s) to epic %q.", len(succeeded), resolved.Title))
		output.MutationBatch(w, header, succeeded)
	}

	return nil
}

// runEpicAddLegacy adds issues to a legacy epic via the ZenHub REST API v1.
func runEpicAddLegacy(client *api.Client, cfg *config.Config, w writerFlusher, resolved *resolve.EpicResult, issues []resolvedEpicIssue, failed []output.FailedItem) error {
	ref := legacyEpicRef(resolved)

	// Look up the epic's repo GhID
	epicRepo, err := resolve.LookupRepoWithRefresh(client, cfg.Workspace, resolved.RepoOwner+"/"+resolved.RepoName)
	if err != nil {
		return exitcode.General(fmt.Sprintf("resolving repository for legacy epic %s", ref), err)
	}

	// Dry run
	if epicAddDryRun {
		return renderEpicAddLegacyDryRun(w, resolved, ref, issues, failed)
	}

	// Build REST API issue refs
	addIssues := make([]api.RESTIssueRef, len(issues))
	for i, iss := range issues {
		addIssues[i] = api.RESTIssueRef{
			RepoID:      iss.RepoGhID,
			IssueNumber: iss.Number,
		}
	}

	if err := client.UpdateEpicIssues(epicRepo.GhID, resolved.IssueNumber, addIssues, nil); err != nil {
		return exitcode.General("adding issues to legacy epic", err)
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
			"epic":  map[string]any{"id": resolved.ID, "title": resolved.Title, "issue": ref},
			"added": formatEpicIssueItemsJSON(issues),
		})
	}

	// Render output
	if len(failed) > 0 {
		totalAttempted := len(succeeded) + len(failed)
		header := output.Green(fmt.Sprintf("Added %d of %d issue(s) to legacy epic %q (%s).", len(succeeded), totalAttempted, resolved.Title, ref))
		output.MutationPartialFailure(w, header, succeeded, failed)
		return exitcode.Generalf("some issues failed to resolve")
	} else if len(succeeded) == 1 {
		output.MutationSingle(w, output.Green(fmt.Sprintf("Added %s to legacy epic %q (%s).", succeeded[0].Ref, resolved.Title, ref)))
	} else {
		header := output.Green(fmt.Sprintf("Added %d issue(s) to legacy epic %q (%s).", len(succeeded), resolved.Title, ref))
		output.MutationBatch(w, header, succeeded)
	}

	return nil
}

func renderEpicAddLegacyDryRun(w writerFlusher, epic *resolve.EpicResult, ref string, issues []resolvedEpicIssue, failed []output.FailedItem) error {
	if len(issues) > 0 {
		items := make([]output.MutationItem, len(issues))
		for i, iss := range issues {
			items[i] = output.MutationItem{
				Ref:   iss.Ref(),
				Title: truncateTitle(iss.Title),
			}
		}
		header := fmt.Sprintf("Would add %d issue(s) to legacy epic %q (%s)", len(issues), epic.Title, ref)
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

func renderEpicAddDryRun(w writerFlusher, epic *resolve.EpicResult, issues []resolvedEpicIssue, failed []output.FailedItem) error {
	if len(issues) > 0 {
		items := make([]output.MutationItem, len(issues))
		for i, iss := range issues {
			items[i] = output.MutationItem{
				Ref:   iss.Ref(),
				Title: truncateTitle(iss.Title),
			}
		}
		header := fmt.Sprintf("Would add %d issue(s) to epic %q", len(issues), epic.Title)
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

// runEpicRemove implements `zh epic remove <epic> <issue>...`.
func runEpicRemove(cmd *cobra.Command, args []string) error {
	cfg, err := requireWorkspace()
	if err != nil {
		return err
	}

	client := newClient(cfg, cmd)
	w := cmd.OutOrStdout()
	ghClient := newGitHubClient(cfg, cmd)

	// Resolve the epic
	resolved, err := resolve.Epic(client, cfg.Workspace, args[0], cfg.Aliases.Epics)
	if err != nil {
		return err
	}

	// Handle --all flag
	if epicRemoveAll {
		if resolved.Type == "legacy" {
			return runEpicRemoveAllLegacy(client, cfg, w, resolved)
		}
		return runEpicRemoveAll(client, cfg, w, resolved)
	}

	if len(args) < 2 {
		return exitcode.Usage("at least one issue identifier is required (or use --all)")
	}

	// Resolve issue identifiers
	issueArgs := args[1:]
	var issues []resolvedEpicIssue
	var failed []output.FailedItem

	for _, arg := range issueArgs {
		issue, err := resolveIssueForEpic(client, cfg.Workspace, arg, epicRemoveRepo, ghClient)
		if err != nil {
			if epicRemoveContinueOnError {
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

	if resolved.Type == "legacy" {
		return runEpicRemoveLegacy(client, cfg, w, resolved, issues, failed)
	}

	// Dry run
	if epicRemoveDryRun {
		return renderEpicRemoveDryRun(w, resolved, issues, failed)
	}

	// Execute the mutation
	issueIDs := make([]string, len(issues))
	for i, iss := range issues {
		issueIDs[i] = iss.ID
	}

	data, err := client.Execute(removeIssuesFromZenhubEpicsMutation, map[string]any{
		"input": map[string]any{
			"zenhubEpicIds": []string{resolved.ID},
			"issueIds":      issueIDs,
		},
	})
	if err != nil {
		return exitcode.General("removing issues from epic", err)
	}

	var resp struct {
		RemoveIssuesFromZenhubEpics struct {
			ZenhubEpics []struct {
				ID    string `json:"id"`
				Title string `json:"title"`
			} `json:"zenhubEpics"`
		} `json:"removeIssuesFromZenhubEpics"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return exitcode.General("parsing remove issues response", err)
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
			"epic":    map[string]any{"id": resolved.ID, "title": resolved.Title},
			"removed": formatEpicIssueItemsJSON(issues),
		})
	}

	// Render output
	if len(failed) > 0 {
		totalAttempted := len(succeeded) + len(failed)
		header := output.Green(fmt.Sprintf("Removed %d of %d issue(s) from epic %q.", len(succeeded), totalAttempted, resolved.Title))
		output.MutationPartialFailure(w, header, succeeded, failed)
		return exitcode.Generalf("some issues failed to resolve")
	} else if len(succeeded) == 1 {
		output.MutationSingle(w, output.Green(fmt.Sprintf("Removed %s from epic %q.", succeeded[0].Ref, resolved.Title)))
	} else {
		header := output.Green(fmt.Sprintf("Removed %d issue(s) from epic %q.", len(succeeded), resolved.Title))
		output.MutationBatch(w, header, succeeded)
	}

	return nil
}

// runEpicRemoveLegacy removes issues from a legacy epic via the ZenHub REST API v1.
func runEpicRemoveLegacy(client *api.Client, cfg *config.Config, w writerFlusher, resolved *resolve.EpicResult, issues []resolvedEpicIssue, failed []output.FailedItem) error {
	ref := legacyEpicRef(resolved)

	// Look up the epic's repo GhID
	epicRepo, err := resolve.LookupRepoWithRefresh(client, cfg.Workspace, resolved.RepoOwner+"/"+resolved.RepoName)
	if err != nil {
		return exitcode.General(fmt.Sprintf("resolving repository for legacy epic %s", ref), err)
	}

	// Dry run
	if epicRemoveDryRun {
		return renderEpicRemoveLegacyDryRun(w, resolved, ref, issues, failed)
	}

	// Build REST API issue refs
	removeIssues := make([]api.RESTIssueRef, len(issues))
	for i, iss := range issues {
		removeIssues[i] = api.RESTIssueRef{
			RepoID:      iss.RepoGhID,
			IssueNumber: iss.Number,
		}
	}

	if err := client.UpdateEpicIssues(epicRepo.GhID, resolved.IssueNumber, nil, removeIssues); err != nil {
		return exitcode.General("removing issues from legacy epic", err)
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
			"epic":    map[string]any{"id": resolved.ID, "title": resolved.Title, "issue": ref},
			"removed": formatEpicIssueItemsJSON(issues),
		})
	}

	// Render output
	if len(failed) > 0 {
		totalAttempted := len(succeeded) + len(failed)
		header := output.Green(fmt.Sprintf("Removed %d of %d issue(s) from legacy epic %q (%s).", len(succeeded), totalAttempted, resolved.Title, ref))
		output.MutationPartialFailure(w, header, succeeded, failed)
		return exitcode.Generalf("some issues failed to resolve")
	} else if len(succeeded) == 1 {
		output.MutationSingle(w, output.Green(fmt.Sprintf("Removed %s from legacy epic %q (%s).", succeeded[0].Ref, resolved.Title, ref)))
	} else {
		header := output.Green(fmt.Sprintf("Removed %d issue(s) from legacy epic %q (%s).", len(succeeded), resolved.Title, ref))
		output.MutationBatch(w, header, succeeded)
	}

	return nil
}

func renderEpicRemoveLegacyDryRun(w writerFlusher, epic *resolve.EpicResult, ref string, issues []resolvedEpicIssue, failed []output.FailedItem) error {
	if len(issues) > 0 {
		items := make([]output.MutationItem, len(issues))
		for i, iss := range issues {
			items[i] = output.MutationItem{
				Ref:   iss.Ref(),
				Title: truncateTitle(iss.Title),
			}
		}
		header := fmt.Sprintf("Would remove %d issue(s) from legacy epic %q (%s)", len(issues), epic.Title, ref)
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

// runEpicRemoveAllLegacy removes all child issues from a legacy epic via the ZenHub REST API v1.
func runEpicRemoveAllLegacy(client *api.Client, cfg *config.Config, w writerFlusher, resolved *resolve.EpicResult) error {
	ref := legacyEpicRef(resolved)

	// Fetch all child issues via GraphQL
	issues, err := fetchAllEpicChildIssues(client, cfg.Workspace, resolved.ID)
	if err != nil {
		return err
	}

	if len(issues) == 0 {
		if output.IsJSON(outputFormat) {
			return output.JSON(w, map[string]any{
				"epic":    map[string]any{"id": resolved.ID, "title": resolved.Title, "issue": ref},
				"removed": []any{},
			})
		}
		fmt.Fprintf(w, "Legacy epic %q (%s) has no child issues.\n", resolved.Title, ref)
		return nil
	}

	if epicRemoveDryRun {
		items := make([]output.MutationItem, len(issues))
		for i, iss := range issues {
			items[i] = output.MutationItem{
				Ref:   iss.Ref(),
				Title: truncateTitle(iss.Title),
			}
		}
		header := fmt.Sprintf("Would remove all %d issue(s) from legacy epic %q (%s)", len(issues), resolved.Title, ref)
		output.MutationDryRun(w, header, items)
		return nil
	}

	// Look up the epic's repo GhID
	epicRepo, err := resolve.LookupRepoWithRefresh(client, cfg.Workspace, resolved.RepoOwner+"/"+resolved.RepoName)
	if err != nil {
		return exitcode.General(fmt.Sprintf("resolving repository for legacy epic %s", ref), err)
	}

	// We need RepoGhID for each issue. The fetchAllEpicChildIssues query only
	// returns RepoName/RepoOwner. Resolve GhIDs from the repo cache.
	removeIssues := make([]api.RESTIssueRef, len(issues))
	for i, iss := range issues {
		repo, err := resolve.LookupRepoWithRefresh(client, cfg.Workspace, iss.RepoOwner+"/"+iss.RepoName)
		if err != nil {
			return exitcode.General(fmt.Sprintf("resolving repository for %s#%d", iss.RepoName, iss.Number), err)
		}
		removeIssues[i] = api.RESTIssueRef{
			RepoID:      repo.GhID,
			IssueNumber: iss.Number,
		}
	}

	if err := client.UpdateEpicIssues(epicRepo.GhID, resolved.IssueNumber, nil, removeIssues); err != nil {
		return exitcode.General("removing issues from legacy epic", err)
	}

	if output.IsJSON(outputFormat) {
		return output.JSON(w, map[string]any{
			"epic":    map[string]any{"id": resolved.ID, "title": resolved.Title, "issue": ref},
			"removed": formatEpicIssueItemsJSON(issues),
		})
	}

	output.MutationSingle(w, output.Green(fmt.Sprintf("Removed all %d issue(s) from legacy epic %q (%s).", len(issues), resolved.Title, ref)))

	return nil
}

// runEpicRemoveAll removes all child issues from a ZenHub epic.
func runEpicRemoveAll(client *api.Client, cfg *config.Config, w writerFlusher, resolved *resolve.EpicResult) error {
	// Fetch all child issues
	issues, err := fetchAllEpicChildIssues(client, cfg.Workspace, resolved.ID)
	if err != nil {
		return err
	}

	if len(issues) == 0 {
		if output.IsJSON(outputFormat) {
			return output.JSON(w, map[string]any{
				"epic":    map[string]any{"id": resolved.ID, "title": resolved.Title},
				"removed": []any{},
			})
		}
		fmt.Fprintf(w, "Epic %q has no child issues.\n", resolved.Title)
		return nil
	}

	if epicRemoveDryRun {
		items := make([]output.MutationItem, len(issues))
		for i, iss := range issues {
			items[i] = output.MutationItem{
				Ref:   iss.Ref(),
				Title: truncateTitle(iss.Title),
			}
		}
		header := fmt.Sprintf("Would remove all %d issue(s) from epic %q", len(issues), resolved.Title)
		output.MutationDryRun(w, header, items)
		return nil
	}

	// Execute the mutation
	issueIDs := make([]string, len(issues))
	for i, iss := range issues {
		issueIDs[i] = iss.ID
	}

	data, err := client.Execute(removeIssuesFromZenhubEpicsMutation, map[string]any{
		"input": map[string]any{
			"zenhubEpicIds": []string{resolved.ID},
			"issueIds":      issueIDs,
		},
	})
	if err != nil {
		return exitcode.General("removing issues from epic", err)
	}

	var resp struct {
		RemoveIssuesFromZenhubEpics struct {
			ZenhubEpics []struct {
				ID    string `json:"id"`
				Title string `json:"title"`
			} `json:"zenhubEpics"`
		} `json:"removeIssuesFromZenhubEpics"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return exitcode.General("parsing remove issues response", err)
	}

	if output.IsJSON(outputFormat) {
		return output.JSON(w, map[string]any{
			"epic":    map[string]any{"id": resolved.ID, "title": resolved.Title},
			"removed": formatEpicIssueItemsJSON(issues),
		})
	}

	output.MutationSingle(w, output.Green(fmt.Sprintf("Removed all %d issue(s) from epic %q.", len(issues), resolved.Title)))

	return nil
}

// fetchAllEpicChildIssues fetches all child issues of a ZenHub epic, paginating as needed.
func fetchAllEpicChildIssues(client *api.Client, workspaceID, epicID string) ([]resolvedEpicIssue, error) {
	var all []resolvedEpicIssue
	var cursor *string
	pageSize := 100

	for {
		vars := map[string]any{
			"id":          epicID,
			"workspaceId": workspaceID,
			"first":       pageSize,
		}
		if cursor != nil {
			vars["after"] = *cursor
		}

		data, err := client.Execute(epicChildIssueIDsQuery, vars)
		if err != nil {
			return nil, exitcode.General("fetching epic child issues", err)
		}

		var resp struct {
			Node *struct {
				ID          string `json:"id"`
				Title       string `json:"title"`
				ChildIssues struct {
					TotalCount int `json:"totalCount"`
					PageInfo   struct {
						HasNextPage bool   `json:"hasNextPage"`
						EndCursor   string `json:"endCursor"`
					} `json:"pageInfo"`
					Nodes []struct {
						ID         string `json:"id"`
						Number     int    `json:"number"`
						Title      string `json:"title"`
						Repository struct {
							Name      string `json:"name"`
							OwnerName string `json:"ownerName"`
						} `json:"repository"`
					} `json:"nodes"`
				} `json:"childIssues"`
			} `json:"node"`
		}
		if err := json.Unmarshal(data, &resp); err != nil {
			return nil, exitcode.General("parsing epic child issues", err)
		}

		if resp.Node == nil {
			return nil, exitcode.NotFoundError(fmt.Sprintf("epic %q not found", epicID))
		}

		for _, n := range resp.Node.ChildIssues.Nodes {
			all = append(all, resolvedEpicIssue{
				ID:        n.ID,
				Number:    n.Number,
				Title:     n.Title,
				RepoName:  n.Repository.Name,
				RepoOwner: n.Repository.OwnerName,
			})
		}

		if !resp.Node.ChildIssues.PageInfo.HasNextPage {
			break
		}
		c := resp.Node.ChildIssues.PageInfo.EndCursor
		cursor = &c
	}

	return all, nil
}

// runEpicEstimate implements `zh epic estimate <epic> [value]`.
func runEpicEstimate(cmd *cobra.Command, args []string) error {
	cfg, err := requireWorkspace()
	if err != nil {
		return err
	}

	client := newClient(cfg, cmd)
	w := cmd.OutOrStdout()

	// Parse value argument (if present)
	var newValue *float64
	if len(args) == 2 {
		v, err := strconv.ParseFloat(args[1], 64)
		if err != nil {
			return exitcode.Usage(fmt.Sprintf("invalid estimate value %q — must be a number", args[1]))
		}
		newValue = &v
	}

	// Resolve the epic
	resolved, err := resolve.Epic(client, cfg.Workspace, args[0], cfg.Aliases.Epics)
	if err != nil {
		return err
	}

	if resolved.Type == "legacy" {
		return exitcode.Usage(fmt.Sprintf("epic %q is a legacy epic (backed by GitHub issue %s/%s#%d) — setting estimates is only supported for ZenHub epics",
			resolved.Title, resolved.RepoOwner, resolved.RepoName, resolved.IssueNumber))
	}

	// Fetch current estimate for dry-run context
	var currentEstimate *float64
	data, err := client.Execute(epicEstimateQuery, map[string]any{
		"id": resolved.ID,
	})
	if err != nil {
		return exitcode.General("fetching epic estimate", err)
	}

	var fetchResp struct {
		Node *struct {
			ID       string `json:"id"`
			Title    string `json:"title"`
			Estimate *struct {
				Value float64 `json:"value"`
			} `json:"estimate"`
		} `json:"node"`
	}
	if err := json.Unmarshal(data, &fetchResp); err != nil {
		return exitcode.General("parsing epic estimate", err)
	}
	if fetchResp.Node != nil && fetchResp.Node.Estimate != nil {
		v := fetchResp.Node.Estimate.Value
		currentEstimate = &v
	}

	// Dry run
	if epicEstimateDryRun {
		var header string
		var ctx string

		if currentEstimate != nil {
			ctx = fmt.Sprintf("(currently: %s)", formatEstimate(*currentEstimate))
		} else {
			ctx = "(currently: none)"
		}

		if newValue != nil {
			header = fmt.Sprintf("Would set estimate on epic %q to %s", resolved.Title, formatEstimate(*newValue))
		} else {
			header = fmt.Sprintf("Would clear estimate from epic %q", resolved.Title)
		}

		items := []output.MutationItem{
			{
				Ref:   resolved.Title,
				Title: ctx,
			},
		}

		output.MutationDryRun(w, header, items)
		return nil
	}

	// Execute mutation
	input := map[string]any{
		"zenhubEpicIds": []string{resolved.ID},
	}
	if newValue != nil {
		input["value"] = *newValue
	} else {
		input["value"] = nil
	}

	data, err = client.Execute(setMultipleEstimatesOnZenhubEpicsMutation, map[string]any{
		"input": input,
	})
	if err != nil {
		return exitcode.General(fmt.Sprintf("setting estimate on epic %q", resolved.Title), err)
	}

	// Parse response
	var resp struct {
		SetMultipleEstimatesOnZenhubEpics struct {
			ZenhubEpics []struct {
				ID       string `json:"id"`
				Title    string `json:"title"`
				Estimate *struct {
					Value float64 `json:"value"`
				} `json:"estimate"`
			} `json:"zenhubEpics"`
		} `json:"setMultipleEstimatesOnZenhubEpics"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return exitcode.General("parsing estimate response", err)
	}

	// JSON output
	if output.IsJSON(outputFormat) {
		jsonResp := map[string]any{
			"epic": map[string]any{
				"id":    resolved.ID,
				"title": resolved.Title,
			},
			"estimate": map[string]any{
				"previous": formatEstimateJSON(currentEstimate),
				"current":  formatEstimateJSON(newValue),
			},
		}
		return output.JSON(w, jsonResp)
	}

	// Render confirmation
	if newValue != nil {
		output.MutationSingle(w, output.Green(fmt.Sprintf(
			"Set estimate on epic %q to %s.",
			resolved.Title, formatEstimate(*newValue),
		)))
	} else {
		output.MutationSingle(w, output.Green(fmt.Sprintf(
			"Cleared estimate from epic %q.",
			resolved.Title,
		)))
	}

	return nil
}

func renderEpicRemoveDryRun(w writerFlusher, epic *resolve.EpicResult, issues []resolvedEpicIssue, failed []output.FailedItem) error {
	if len(issues) > 0 {
		items := make([]output.MutationItem, len(issues))
		for i, iss := range issues {
			items[i] = output.MutationItem{
				Ref:   iss.Ref(),
				Title: truncateTitle(iss.Title),
			}
		}
		header := fmt.Sprintf("Would remove %d issue(s) from epic %q", len(issues), epic.Title)
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

func formatEpicIssueItemsJSON(issues []resolvedEpicIssue) []map[string]any {
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
