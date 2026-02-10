package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/dslh/zh/internal/api"
	"github.com/dslh/zh/internal/config"
	"github.com/dslh/zh/internal/exitcode"
	"github.com/dslh/zh/internal/gh"
	"github.com/dslh/zh/internal/output"
	"github.com/dslh/zh/internal/resolve"
	"github.com/spf13/cobra"
)

// GraphQL mutations and queries for issue blocking

const createBlockageMutation = `mutation CreateBlockage($input: CreateBlockageInput!) {
  createBlockage(input: $input) {
    blockage {
      id
      createdAt
      blocking {
        ... on Issue {
          __typename
          id
          number
          title
          repository {
            name
            ownerName
          }
        }
        ... on ZenhubEpic {
          __typename
          id
          title
        }
      }
      blocked {
        ... on Issue {
          __typename
          id
          number
          title
          repository {
            name
            ownerName
          }
        }
        ... on ZenhubEpic {
          __typename
          id
          title
        }
      }
    }
  }
}`

const issueBlockersQuery = `query GetIssueBlockers($repositoryGhId: Int!, $issueNumber: Int!) {
  issueByInfo(repositoryGhId: $repositoryGhId, issueNumber: $issueNumber) {
    id
    number
    title
    repository {
      name
      ownerName
    }
    blockingItems(first: 50) {
      nodes {
        ... on Issue {
          __typename
          id
          number
          title
          state
          repository {
            name
            ownerName
          }
        }
        ... on ZenhubEpic {
          __typename
          id
          title
          state
        }
      }
    }
  }
}`

const issueBlockersByNodeQuery = `query GetIssueBlockersByNode($id: ID!) {
  node(id: $id) {
    ... on Issue {
      id
      number
      title
      repository {
        name
        ownerName
      }
      blockingItems(first: 50) {
        nodes {
          ... on Issue {
            __typename
            id
            number
            title
            state
            repository {
              name
              ownerName
            }
          }
          ... on ZenhubEpic {
            __typename
            id
            title
            state
          }
        }
      }
    }
  }
}`

const issueBlockingQuery = `query GetIssueBlocking($repositoryGhId: Int!, $issueNumber: Int!) {
  issueByInfo(repositoryGhId: $repositoryGhId, issueNumber: $issueNumber) {
    id
    number
    title
    repository {
      name
      ownerName
    }
    blockedItems(first: 50) {
      nodes {
        ... on Issue {
          __typename
          id
          number
          title
          state
          repository {
            name
            ownerName
          }
        }
        ... on ZenhubEpic {
          __typename
          id
          title
          state
        }
      }
    }
  }
}`

const issueBlockingByNodeQuery = `query GetIssueBlockingByNode($id: ID!) {
  node(id: $id) {
    ... on Issue {
      id
      number
      title
      repository {
        name
        ownerName
      }
      blockedItems(first: 50) {
        nodes {
          ... on Issue {
            __typename
            id
            number
            title
            state
            repository {
              name
              ownerName
            }
          }
          ... on ZenhubEpic {
            __typename
            id
            title
            state
          }
        }
      }
    }
  }
}`

// blockItem represents a resolved item (issue or epic) in a blocking relationship.
type blockItem struct {
	ID        string
	Type      string // "ISSUE" or "ZENHUB_EPIC"
	Ref       string // display reference (e.g. "repo#1" or epic title)
	Title     string
	RepoName  string
	RepoOwner string
}

// blockDependencyNode represents a blocking/blocked item from the API response.
type blockDependencyNode struct {
	TypeName   string `json:"__typename"`
	ID         string `json:"id"`
	Number     int    `json:"number"`
	Title      string `json:"title"`
	State      string `json:"state"`
	Repository *struct {
		Name      string `json:"name"`
		OwnerName string `json:"ownerName"`
	} `json:"repository"`
}

func (n *blockDependencyNode) Ref() string {
	if n.Repository != nil {
		return fmt.Sprintf("%s#%d", n.Repository.Name, n.Number)
	}
	return n.Title
}

// Commands

var issueBlockCmd = &cobra.Command{
	Use:   "block <blocker> <blocked>",
	Short: "Mark an issue/epic as blocking another",
	Long: `Mark the first argument as blocking the second argument.

Both arguments default to issues. Use --blocker-type=epic or --blocked-type=epic
to specify that either side is a ZenHub epic.

Note: Blocks cannot be removed via the API. Use ZenHub's web UI to remove
blocking relationships.

Examples:
  zh issue block task-tracker#1 task-tracker#2
  zh issue block task-tracker#1 "Auth Epic" --blocked-type=epic
  zh issue block --repo=task-tracker 1 2`,
	Args: cobra.ExactArgs(2),
	RunE: runIssueBlock,
}

var issueBlockersCmd = &cobra.Command{
	Use:   "blockers <issue>",
	Short: "List issues and epics blocking this issue",
	Long: `List all issues and epics that are blocking the specified issue.

Examples:
  zh issue blockers task-tracker#1
  zh issue blockers --repo=task-tracker 1`,
	Args: cobra.ExactArgs(1),
	RunE: runIssueBlockers,
}

var issueBlockingCmd = &cobra.Command{
	Use:   "blocking <issue>",
	Short: "List issues and epics this issue is blocking",
	Long: `List all issues and epics that the specified issue is blocking.

Examples:
  zh issue blocking task-tracker#1
  zh issue blocking --repo=task-tracker 1`,
	Args: cobra.ExactArgs(1),
	RunE: runIssueBlocking,
}

var (
	issueBlockDryRun      bool
	issueBlockRepo        string
	issueBlockBlockerType string
	issueBlockBlockedType string
	issueBlockersRepo     string
	issueBlockingRepo     string
)

func init() {
	issueBlockCmd.Flags().BoolVar(&issueBlockDryRun, "dry-run", false, "Show what would be blocked without executing")
	issueBlockCmd.Flags().StringVar(&issueBlockRepo, "repo", "", "Repository context for bare issue numbers")
	issueBlockCmd.Flags().StringVar(&issueBlockBlockerType, "blocker-type", "issue", "Type of the blocker: issue or epic")
	issueBlockCmd.Flags().StringVar(&issueBlockBlockedType, "blocked-type", "issue", "Type of the blocked item: issue or epic")

	issueBlockersCmd.Flags().StringVar(&issueBlockersRepo, "repo", "", "Repository context for bare issue numbers")

	issueBlockingCmd.Flags().StringVar(&issueBlockingRepo, "repo", "", "Repository context for bare issue numbers")

	issueCmd.AddCommand(issueBlockCmd)
	issueCmd.AddCommand(issueBlockersCmd)
	issueCmd.AddCommand(issueBlockingCmd)
}

func resetIssueBlockFlags() {
	issueBlockDryRun = false
	issueBlockRepo = ""
	issueBlockBlockerType = "issue"
	issueBlockBlockedType = "issue"
}

func resetIssueBlockersFlags() {
	issueBlockersRepo = ""
}

func resetIssueBlockingFlags() {
	issueBlockingRepo = ""
}

func runIssueBlock(cmd *cobra.Command, args []string) error {
	cfg, err := requireWorkspace()
	if err != nil {
		return err
	}

	client := newClient(cfg, cmd)
	ghClient := newGitHubClient(cfg, cmd)
	w := cmd.OutOrStdout()

	// Validate type flags
	blockerType := strings.ToLower(issueBlockBlockerType)
	blockedType := strings.ToLower(issueBlockBlockedType)
	if blockerType != "issue" && blockerType != "epic" {
		return exitcode.Usage(fmt.Sprintf("invalid --blocker-type %q — must be 'issue' or 'epic'", issueBlockBlockerType))
	}
	if blockedType != "issue" && blockedType != "epic" {
		return exitcode.Usage(fmt.Sprintf("invalid --blocked-type %q — must be 'issue' or 'epic'", issueBlockBlockedType))
	}

	// Resolve blocker
	blocker, err := resolveBlockItem(client, cfg, args[0], blockerType, issueBlockRepo, ghClient)
	if err != nil {
		return fmt.Errorf("resolving blocker: %w", err)
	}

	// Resolve blocked
	blocked, err := resolveBlockItem(client, cfg, args[1], blockedType, issueBlockRepo, ghClient)
	if err != nil {
		return fmt.Errorf("resolving blocked: %w", err)
	}

	// Dry run
	if issueBlockDryRun {
		if output.IsJSON(outputFormat) {
			return output.JSON(w, map[string]any{
				"dryRun":   true,
				"blocking": formatBlockItemJSON(blocker),
				"blocked":  formatBlockItemJSON(blocked),
			})
		}
		msg := fmt.Sprintf("Would mark %s as blocking %s", blocker.Ref, blocked.Ref)
		output.MutationDryRun(w, msg, []output.MutationItem{
			{Ref: blocker.Ref, Title: truncateTitle(blocker.Title), Context: fmt.Sprintf("(%s, blocking)", blocker.Type)},
			{Ref: blocked.Ref, Title: truncateTitle(blocked.Title), Context: fmt.Sprintf("(%s, blocked)", blocked.Type)},
		})
		return nil
	}

	// Execute mutation
	data, err := client.Execute(createBlockageMutation, map[string]any{
		"input": map[string]any{
			"blocking": map[string]any{
				"id":   blocker.ID,
				"type": blocker.Type,
			},
			"blocked": map[string]any{
				"id":   blocked.ID,
				"type": blocked.Type,
			},
		},
	})
	if err != nil {
		return exitcode.General("creating blockage", err)
	}

	// Parse response
	var resp struct {
		CreateBlockage struct {
			Blockage struct {
				ID        string `json:"id"`
				CreatedAt string `json:"createdAt"`
			} `json:"blockage"`
		} `json:"createBlockage"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return exitcode.General("parsing blockage response", err)
	}

	// JSON output
	if output.IsJSON(outputFormat) {
		return output.JSON(w, map[string]any{
			"blocking": formatBlockItemJSON(blocker),
			"blocked":  formatBlockItemJSON(blocked),
		})
	}

	output.MutationSingle(w, output.Green(fmt.Sprintf(
		"Marked %s as blocking %s.",
		blocker.Ref, blocked.Ref,
	)))
	fmt.Fprintln(w)
	fmt.Fprintln(w, output.Dim("Note: Blocks cannot be removed via the API. Use ZenHub's web UI to remove blocking relationships."))

	return nil
}

func runIssueBlockers(cmd *cobra.Command, args []string) error {
	cfg, err := requireWorkspace()
	if err != nil {
		return err
	}

	client := newClient(cfg, cmd)
	ghClient := newGitHubClient(cfg, cmd)
	w := cmd.OutOrStdout()

	parsed, parseErr := resolve.ParseIssueRef(args[0])

	var issueData struct {
		ID        string
		Number    int
		Title     string
		RepoName  string
		RepoOwner string
		Blockers  []blockDependencyNode
	}

	if parseErr == nil && parsed.ZenHubID != "" {
		data, err := client.Execute(issueBlockersByNodeQuery, map[string]any{"id": parsed.ZenHubID})
		if err != nil {
			return exitcode.General("fetching blockers", err)
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
				BlockingItems struct {
					Nodes []blockDependencyNode `json:"nodes"`
				} `json:"blockingItems"`
			} `json:"node"`
		}
		if err := json.Unmarshal(data, &resp); err != nil {
			return exitcode.General("parsing blockers response", err)
		}
		if resp.Node == nil {
			return exitcode.NotFoundError(fmt.Sprintf("issue %q not found", args[0]))
		}
		issueData.ID = resp.Node.ID
		issueData.Number = resp.Node.Number
		issueData.Title = resp.Node.Title
		issueData.RepoName = resp.Node.Repository.Name
		issueData.RepoOwner = resp.Node.Repository.OwnerName
		issueData.Blockers = resp.Node.BlockingItems.Nodes
	} else {
		result, err := resolve.Issue(client, cfg.Workspace, args[0], &resolve.IssueOptions{
			RepoFlag:     issueBlockersRepo,
			GitHubClient: ghClient,
		})
		if err != nil {
			return err
		}

		data, err := client.Execute(issueBlockersQuery, map[string]any{
			"repositoryGhId": result.RepoGhID,
			"issueNumber":    result.Number,
		})
		if err != nil {
			return exitcode.General("fetching blockers", err)
		}
		var resp struct {
			IssueByInfo *struct {
				ID         string `json:"id"`
				Number     int    `json:"number"`
				Title      string `json:"title"`
				Repository struct {
					Name      string `json:"name"`
					OwnerName string `json:"ownerName"`
				} `json:"repository"`
				BlockingItems struct {
					Nodes []blockDependencyNode `json:"nodes"`
				} `json:"blockingItems"`
			} `json:"issueByInfo"`
		}
		if err := json.Unmarshal(data, &resp); err != nil {
			return exitcode.General("parsing blockers response", err)
		}
		if resp.IssueByInfo == nil {
			return exitcode.NotFoundError(fmt.Sprintf("issue %q not found", args[0]))
		}
		issueData.ID = resp.IssueByInfo.ID
		issueData.Number = resp.IssueByInfo.Number
		issueData.Title = resp.IssueByInfo.Title
		issueData.RepoName = resp.IssueByInfo.Repository.Name
		issueData.RepoOwner = resp.IssueByInfo.Repository.OwnerName
		issueData.Blockers = resp.IssueByInfo.BlockingItems.Nodes
	}

	issueRef := fmt.Sprintf("%s#%d", issueData.RepoName, issueData.Number)

	if output.IsJSON(outputFormat) {
		return output.JSON(w, map[string]any{
			"issue":    map[string]any{"id": issueData.ID, "number": issueData.Number, "ref": issueRef, "title": issueData.Title},
			"blockers": formatBlockDependencyNodesJSON(issueData.Blockers),
		})
	}

	if len(issueData.Blockers) == 0 {
		fmt.Fprintf(w, "%s has no blockers.\n", issueRef)
		return nil
	}

	fmt.Fprintf(w, "%s is blocked by:\n\n", issueRef)
	for _, node := range issueData.Blockers {
		renderBlockDependencyNode(w, &node)
	}

	return nil
}

func runIssueBlocking(cmd *cobra.Command, args []string) error {
	cfg, err := requireWorkspace()
	if err != nil {
		return err
	}

	client := newClient(cfg, cmd)
	ghClient := newGitHubClient(cfg, cmd)
	w := cmd.OutOrStdout()

	parsed, parseErr := resolve.ParseIssueRef(args[0])

	var issueData struct {
		ID        string
		Number    int
		Title     string
		RepoName  string
		RepoOwner string
		Blocking  []blockDependencyNode
	}

	if parseErr == nil && parsed.ZenHubID != "" {
		data, err := client.Execute(issueBlockingByNodeQuery, map[string]any{"id": parsed.ZenHubID})
		if err != nil {
			return exitcode.General("fetching blocking items", err)
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
				BlockedItems struct {
					Nodes []blockDependencyNode `json:"nodes"`
				} `json:"blockedItems"`
			} `json:"node"`
		}
		if err := json.Unmarshal(data, &resp); err != nil {
			return exitcode.General("parsing blocking response", err)
		}
		if resp.Node == nil {
			return exitcode.NotFoundError(fmt.Sprintf("issue %q not found", args[0]))
		}
		issueData.ID = resp.Node.ID
		issueData.Number = resp.Node.Number
		issueData.Title = resp.Node.Title
		issueData.RepoName = resp.Node.Repository.Name
		issueData.RepoOwner = resp.Node.Repository.OwnerName
		issueData.Blocking = resp.Node.BlockedItems.Nodes
	} else {
		result, err := resolve.Issue(client, cfg.Workspace, args[0], &resolve.IssueOptions{
			RepoFlag:     issueBlockingRepo,
			GitHubClient: ghClient,
		})
		if err != nil {
			return err
		}

		data, err := client.Execute(issueBlockingQuery, map[string]any{
			"repositoryGhId": result.RepoGhID,
			"issueNumber":    result.Number,
		})
		if err != nil {
			return exitcode.General("fetching blocking items", err)
		}
		var resp struct {
			IssueByInfo *struct {
				ID         string `json:"id"`
				Number     int    `json:"number"`
				Title      string `json:"title"`
				Repository struct {
					Name      string `json:"name"`
					OwnerName string `json:"ownerName"`
				} `json:"repository"`
				BlockedItems struct {
					Nodes []blockDependencyNode `json:"nodes"`
				} `json:"blockedItems"`
			} `json:"issueByInfo"`
		}
		if err := json.Unmarshal(data, &resp); err != nil {
			return exitcode.General("parsing blocking response", err)
		}
		if resp.IssueByInfo == nil {
			return exitcode.NotFoundError(fmt.Sprintf("issue %q not found", args[0]))
		}
		issueData.ID = resp.IssueByInfo.ID
		issueData.Number = resp.IssueByInfo.Number
		issueData.Title = resp.IssueByInfo.Title
		issueData.RepoName = resp.IssueByInfo.Repository.Name
		issueData.RepoOwner = resp.IssueByInfo.Repository.OwnerName
		issueData.Blocking = resp.IssueByInfo.BlockedItems.Nodes
	}

	issueRef := fmt.Sprintf("%s#%d", issueData.RepoName, issueData.Number)

	if output.IsJSON(outputFormat) {
		return output.JSON(w, map[string]any{
			"issue":    map[string]any{"id": issueData.ID, "number": issueData.Number, "ref": issueRef, "title": issueData.Title},
			"blocking": formatBlockDependencyNodesJSON(issueData.Blocking),
		})
	}

	if len(issueData.Blocking) == 0 {
		fmt.Fprintf(w, "%s is not blocking anything.\n", issueRef)
		return nil
	}

	fmt.Fprintf(w, "%s is blocking:\n\n", issueRef)
	for _, node := range issueData.Blocking {
		renderBlockDependencyNode(w, &node)
	}

	return nil
}

// resolveBlockItem resolves a blocker/blocked item by type (issue or epic).
func resolveBlockItem(client *api.Client, cfg *config.Config, identifier, itemType, repoFlag string, ghClient *gh.Client) (*blockItem, error) {
	if itemType == "epic" {
		epic, err := resolve.Epic(client, cfg.Workspace, identifier, cfg.Aliases.Epics)
		if err != nil {
			return nil, err
		}
		return &blockItem{
			ID:    epic.ID,
			Type:  "ZENHUB_EPIC",
			Ref:   epic.Title,
			Title: epic.Title,
		}, nil
	}

	// Default: issue
	result, err := resolve.Issue(client, cfg.Workspace, identifier, &resolve.IssueOptions{
		RepoFlag:     repoFlag,
		GitHubClient: ghClient,
	})
	if err != nil {
		return nil, err
	}

	return &blockItem{
		ID:        result.ID,
		Type:      "ISSUE",
		Ref:       result.Ref(),
		Title:     "", // We'll fill this in from the resolve query if needed
		RepoName:  result.RepoName,
		RepoOwner: result.RepoOwner,
	}, nil
}

func renderBlockDependencyNode(w writerFlusher, node *blockDependencyNode) {
	state := strings.ToLower(node.State)
	if node.TypeName == "Issue" && node.Repository != nil {
		ref := fmt.Sprintf("%s#%d", node.Repository.Name, node.Number)
		fmt.Fprintf(w, "  %s  %s  %s\n", output.Cyan(ref), truncateTitle(node.Title), output.Dim("("+state+")"))
	} else {
		// Epic
		label := fmt.Sprintf("[epic] %s", node.Title)
		fmt.Fprintf(w, "  %s  %s\n", label, output.Dim("("+state+")"))
	}
}

func formatBlockItemJSON(item *blockItem) map[string]any {
	result := map[string]any{
		"id":   item.ID,
		"type": item.Type,
		"ref":  item.Ref,
	}
	if item.Title != "" {
		result["title"] = item.Title
	}
	return result
}

func formatBlockDependencyNodesJSON(nodes []blockDependencyNode) []map[string]any {
	result := make([]map[string]any, len(nodes))
	for i, node := range nodes {
		item := map[string]any{
			"id":    node.ID,
			"type":  node.TypeName,
			"title": node.Title,
			"state": node.State,
		}
		if node.Repository != nil {
			item["ref"] = fmt.Sprintf("%s#%d", node.Repository.Name, node.Number)
			item["number"] = node.Number
			item["repository"] = fmt.Sprintf("%s/%s", node.Repository.OwnerName, node.Repository.Name)
		}
		result[i] = item
	}
	return result
}
