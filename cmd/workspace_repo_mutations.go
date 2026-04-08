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

const addRepositoryToWorkspaceMutation = `mutation AddRepositoryToWorkspace($input: AddRepositoryToWorkspaceInput!) {
  addRepositoryToWorkspace(input: $input) {
    clientMutationId
  }
}`

const disconnectWorkspaceRepositoryMutation = `mutation DisconnectWorkspaceRepository($input: DisconnectWorkspaceRepositoryInput!) {
  disconnectWorkspaceRepository(input: $input) {
    clientMutationId
  }
}`

const githubRepoLookupQuery = `query LookupRepository($owner: String!, $name: String!) {
  repository(owner: $owner, name: $name) {
    databaseId
    nameWithOwner
  }
}`

var (
	workspaceRepoAddGhID      int
	workspaceRepoAddDryRun    bool
	workspaceRepoRemoveGhID   int
	workspaceRepoRemoveDryRun bool
)

func resetWorkspaceRepoMutationFlags() {
	workspaceRepoAddGhID = 0
	workspaceRepoAddDryRun = false
	workspaceRepoRemoveGhID = 0
	workspaceRepoRemoveDryRun = false
}

var workspaceRepoCmd = &cobra.Command{
	Use:   "repo",
	Short: "Add or remove repositories in the workspace",
	Long:  `Manage which GitHub repositories are connected to the current workspace.`,
}

var workspaceRepoAddCmd = &cobra.Command{
	Use:   "add [owner/repo]",
	Short: "Add a repository to the workspace",
	Long: `Add a GitHub repository to the current workspace.

Provide the repository as owner/repo (e.g. "myorg/myrepo"). The command looks
up the GitHub repository database ID via GitHub's API, which requires GitHub
authentication (gh CLI or ZH_GITHUB_TOKEN). Alternatively, pass --gh-id to skip
the lookup entirely.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runWorkspaceRepoAdd,
}

var workspaceRepoRemoveCmd = &cobra.Command{
	Use:   "remove [owner/repo]",
	Short: "Remove a repository from the workspace",
	Long: `Remove a GitHub repository from the current workspace.

Provide the repository as owner/repo (or just repo if unambiguous). The
repository must currently be connected to the workspace. Alternatively, pass
--gh-id to skip resolution.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runWorkspaceRepoRemove,
}

func runWorkspaceRepoAdd(cmd *cobra.Command, args []string) error {
	cfg, err := requireWorkspace()
	if err != nil {
		return err
	}
	w := cmd.OutOrStdout()
	client := newClient(cfg, cmd)

	ghID, label, err := resolveRepoForAdd(cfg, cmd, args)
	if err != nil {
		return err
	}

	if workspaceRepoAddDryRun {
		output.MutationDryRunDetail(w, fmt.Sprintf("Would add %s to workspace.", label), []output.DetailLine{
			{Key: "GitHub ID", Value: fmt.Sprintf("%d", ghID)},
		})
		return nil
	}

	input := map[string]any{
		"workspaceId":    cfg.Workspace,
		"repositoryGhId": ghID,
	}
	if _, err := client.Execute(addRepositoryToWorkspaceMutation, map[string]any{"input": input}); err != nil {
		return exitcode.General("adding repository to workspace", err)
	}

	_ = cache.Clear(resolve.RepoCacheKey(cfg.Workspace))

	output.MutationSingle(w, output.Green(fmt.Sprintf("Added %s to workspace.", label)))
	return nil
}

func runWorkspaceRepoRemove(cmd *cobra.Command, args []string) error {
	cfg, err := requireWorkspace()
	if err != nil {
		return err
	}
	w := cmd.OutOrStdout()
	client := newClient(cfg, cmd)

	ghID, label, err := resolveRepoForRemove(client, cfg, args)
	if err != nil {
		return err
	}

	if workspaceRepoRemoveDryRun {
		output.MutationDryRunDetail(w, fmt.Sprintf("Would remove %s from workspace.", label), []output.DetailLine{
			{Key: "GitHub ID", Value: fmt.Sprintf("%d", ghID)},
		})
		return nil
	}

	input := map[string]any{
		"workspaceId":    cfg.Workspace,
		"repositoryGhId": ghID,
	}
	if _, err := client.Execute(disconnectWorkspaceRepositoryMutation, map[string]any{"input": input}); err != nil {
		return exitcode.General("removing repository from workspace", err)
	}

	_ = cache.Clear(resolve.RepoCacheKey(cfg.Workspace))

	output.MutationSingle(w, output.Green(fmt.Sprintf("Removed %s from workspace.", label)))
	return nil
}

// resolveRepoForAdd resolves a repo identifier to a GitHub database ID for the
// add command. If --gh-id is set, it's used directly; otherwise the positional
// arg is parsed as owner/repo and looked up via GitHub's GraphQL API.
func resolveRepoForAdd(cfg *config.Config, cmd *cobra.Command, args []string) (int, string, error) {
	if workspaceRepoAddGhID != 0 {
		label := fmt.Sprintf("repository (gh-id %d)", workspaceRepoAddGhID)
		if len(args) > 0 {
			label = args[0]
		}
		return workspaceRepoAddGhID, label, nil
	}
	if len(args) == 0 {
		return 0, "", exitcode.Usage("must provide owner/repo or --gh-id")
	}
	identifier := args[0]
	owner, name, ok := splitOwnerRepo(identifier)
	if !ok {
		return 0, "", exitcode.Usage(fmt.Sprintf("repository %q must be in owner/repo format (or pass --gh-id)", identifier))
	}

	ghClient := newGitHubClient(cfg, cmd)
	if ghClient == nil {
		return 0, "", exitcode.Auth("GitHub authentication required to look up repository by name — configure ZH_GITHUB_TOKEN or run 'gh auth login', or pass --gh-id <id>", nil)
	}

	data, err := ghClient.Execute(githubRepoLookupQuery, map[string]any{
		"owner": owner,
		"name":  name,
	})
	if err != nil {
		return 0, "", exitcode.General(fmt.Sprintf("looking up GitHub repository %q", identifier), err)
	}
	var resp struct {
		Repository *struct {
			DatabaseID    int    `json:"databaseId"`
			NameWithOwner string `json:"nameWithOwner"`
		} `json:"repository"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return 0, "", exitcode.General("parsing GitHub repository response", err)
	}
	if resp.Repository == nil {
		return 0, "", exitcode.NotFoundError(fmt.Sprintf("GitHub repository %q not found", identifier))
	}
	return resp.Repository.DatabaseID, resp.Repository.NameWithOwner, nil
}

// resolveRepoForRemove resolves a repo identifier to a GitHub database ID for
// the remove command. The repo must already be connected to the workspace, so
// we use the cache-backed resolver.
func resolveRepoForRemove(client *api.Client, cfg *config.Config, args []string) (int, string, error) {
	if workspaceRepoRemoveGhID != 0 {
		label := fmt.Sprintf("repository (gh-id %d)", workspaceRepoRemoveGhID)
		if len(args) > 0 {
			label = args[0]
		}
		return workspaceRepoRemoveGhID, label, nil
	}
	if len(args) == 0 {
		return 0, "", exitcode.Usage("must provide owner/repo or --gh-id")
	}
	repo, err := resolve.LookupRepoWithRefresh(client, cfg.Workspace, args[0])
	if err != nil {
		return 0, "", err
	}
	return repo.GhID, fmt.Sprintf("%s/%s", repo.OwnerName, repo.Name), nil
}

func splitOwnerRepo(s string) (owner, name string, ok bool) {
	parts := strings.SplitN(s, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", false
	}
	return parts[0], parts[1], true
}
