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

	epicAliasDelete bool
	epicAliasList   bool
)

func init() {
	epicCreateCmd.Flags().StringVar(&epicCreateBody, "body", "", "Epic description (markdown)")
	epicCreateCmd.Flags().StringVar(&epicCreateRepo, "repo", "", "Repository for legacy epic (creates a GitHub issue-backed epic)")
	epicCreateCmd.Flags().BoolVar(&epicCreateDryRun, "dry-run", false, "Show what would be created without executing")

	epicAliasCmd.Flags().BoolVar(&epicAliasDelete, "delete", false, "Remove an existing alias")
	epicAliasCmd.Flags().BoolVar(&epicAliasList, "list", false, "List all epic aliases")

	epicCmd.AddCommand(epicCreateCmd)
	epicCmd.AddCommand(epicAliasCmd)
}

func resetEpicMutationFlags() {
	epicCreateBody = ""
	epicCreateRepo = ""
	epicCreateDryRun = false

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
