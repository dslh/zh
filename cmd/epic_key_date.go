package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/dslh/zh/internal/api"
	"github.com/dslh/zh/internal/exitcode"
	"github.com/dslh/zh/internal/output"
	"github.com/dslh/zh/internal/resolve"
	"github.com/spf13/cobra"
)

// GraphQL queries and mutations for epic key-date operations

const epicKeyDatesQuery = `query GetEpicKeyDates($id: ID!) {
  node(id: $id) {
    ... on ZenhubEpic {
      id
      title
      keyDates(first: 50) {
        totalCount
        nodes {
          id
          date
          description
          color
        }
      }
    }
  }
}`

const createZenhubEpicKeyDateMutation = `mutation CreateZenhubEpicKeyDate($input: CreateZenhubEpicKeyDateInput!) {
  createZenhubEpicKeyDate(input: $input) {
    keyDate {
      id
      date
      description
      color
    }
    zenhubEpic {
      id
      title
    }
  }
}`

const deleteZenhubEpicKeyDateMutation = `mutation DeleteZenhubEpicKeyDate($input: DeleteZenhubEpicKeyDateInput!) {
  deleteZenhubEpicKeyDate(input: $input) {
    keyDate {
      id
      date
      description
    }
    zenhubEpic {
      id
      title
    }
  }
}`

// Key date data types

type keyDateNode struct {
	ID          string  `json:"id"`
	Date        string  `json:"date"`
	Description string  `json:"description"`
	Color       *string `json:"color"`
}

// Commands

var epicKeyDateCmd = &cobra.Command{
	Use:   "key-date",
	Short: "List, add, or remove key dates from an epic",
	Long: `Manage key dates (milestones) within a ZenHub epic.

Examples:
  zh epic key-date list "Q1 Roadmap"
  zh epic key-date add "Q1 Roadmap" "Beta Release" 2026-04-01
  zh epic key-date remove "Q1 Roadmap" "Beta Release"`,
}

var epicKeyDateListCmd = &cobra.Command{
	Use:   "list <epic>",
	Short: "List key dates within an epic",
	Long: `List all key dates (milestones) within a ZenHub epic.

The epic can be specified as:
  - ZenHub ID
  - exact title or unique title substring
  - an alias set with 'zh epic alias'

Examples:
  zh epic key-date list "Q1 Roadmap"`,
	Args: cobra.ExactArgs(1),
	RunE: runEpicKeyDateList,
}

var epicKeyDateAddCmd = &cobra.Command{
	Use:   "add <epic> <name> <date>",
	Short: "Add a key date to an epic",
	Long: `Add a key date (milestone) to a ZenHub epic.

The date must be in YYYY-MM-DD format.

Examples:
  zh epic key-date add "Q1 Roadmap" "Beta Release" 2026-04-01
  zh epic key-date add "Q1 Roadmap" "Code Freeze" 2026-03-15 --dry-run`,
	Args: cobra.ExactArgs(3),
	RunE: runEpicKeyDateAdd,
}

var epicKeyDateRemoveCmd = &cobra.Command{
	Use:   "remove <epic> <name>",
	Short: "Remove a key date from an epic",
	Long: `Remove a key date (milestone) from a ZenHub epic.

The key date is matched by name (case-insensitive).

Examples:
  zh epic key-date remove "Q1 Roadmap" "Beta Release"
  zh epic key-date remove "Q1 Roadmap" "Beta Release" --dry-run`,
	Args: cobra.ExactArgs(2),
	RunE: runEpicKeyDateRemove,
}

// Flag variables

var (
	epicKeyDateAddDryRun    bool
	epicKeyDateRemoveDryRun bool
)

func init() {
	epicKeyDateAddCmd.Flags().BoolVar(&epicKeyDateAddDryRun, "dry-run", false, "Show what would be changed without executing")
	epicKeyDateRemoveCmd.Flags().BoolVar(&epicKeyDateRemoveDryRun, "dry-run", false, "Show what would be changed without executing")

	epicKeyDateCmd.AddCommand(epicKeyDateListCmd)
	epicKeyDateCmd.AddCommand(epicKeyDateAddCmd)
	epicKeyDateCmd.AddCommand(epicKeyDateRemoveCmd)
	epicCmd.AddCommand(epicKeyDateCmd)
}

func resetEpicKeyDateFlags() {
	epicKeyDateAddDryRun = false
	epicKeyDateRemoveDryRun = false
}

// --- epic key-date list ---

func runEpicKeyDateList(cmd *cobra.Command, args []string) error {
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
		return exitcode.Usage(fmt.Sprintf("epic %q is a legacy epic (backed by GitHub issue %s/%s#%d) — key dates are only supported for ZenHub epics",
			resolved.Title, resolved.RepoOwner, resolved.RepoName, resolved.IssueNumber))
	}

	// Fetch key dates
	keyDates, err := fetchEpicKeyDates(client, resolved.ID)
	if err != nil {
		return err
	}

	if output.IsJSON(outputFormat) {
		return output.JSON(w, map[string]any{
			"epic":     map[string]any{"id": resolved.ID, "title": resolved.Title},
			"keyDates": keyDates,
		})
	}

	if len(keyDates) == 0 {
		fmt.Fprintf(w, "No key dates on epic %q.\n", resolved.Title)
		return nil
	}

	lw := output.NewListWriter(w, "DATE", "NAME")
	for _, kd := range keyDates {
		lw.Row(kd.Date, kd.Description)
	}

	footer := fmt.Sprintf("%d key date(s) on epic %q", len(keyDates), resolved.Title)
	lw.FlushWithFooter(footer)
	return nil
}

func fetchEpicKeyDates(client *api.Client, epicID string) ([]keyDateNode, error) {
	data, err := client.Execute(epicKeyDatesQuery, map[string]any{
		"id": epicID,
	})
	if err != nil {
		return nil, exitcode.General("fetching key dates", err)
	}

	var resp struct {
		Node *struct {
			ID       string `json:"id"`
			Title    string `json:"title"`
			KeyDates struct {
				TotalCount int           `json:"totalCount"`
				Nodes      []keyDateNode `json:"nodes"`
			} `json:"keyDates"`
		} `json:"node"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, exitcode.General("parsing key dates response", err)
	}

	if resp.Node == nil {
		return nil, exitcode.NotFoundError(fmt.Sprintf("epic %q not found", epicID))
	}

	return resp.Node.KeyDates.Nodes, nil
}

// --- epic key-date add ---

func runEpicKeyDateAdd(cmd *cobra.Command, args []string) error {
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
		return exitcode.Usage(fmt.Sprintf("epic %q is a legacy epic (backed by GitHub issue %s/%s#%d) — key dates are only supported for ZenHub epics",
			resolved.Title, resolved.RepoOwner, resolved.RepoName, resolved.IssueNumber))
	}

	name := args[1]
	date, err := parseDate(args[2])
	if err != nil {
		return err
	}

	// Dry run
	if epicKeyDateAddDryRun {
		if output.IsJSON(outputFormat) {
			return output.JSON(w, map[string]any{
				"dryRun":    true,
				"operation": "add",
				"epic":      map[string]any{"id": resolved.ID, "title": resolved.Title},
				"keyDate":   map[string]any{"description": name, "date": date},
			})
		}

		header := fmt.Sprintf("Would add key date %q (%s) to epic %q", name, date, resolved.Title)
		output.MutationDryRun(w, header, nil)
		return nil
	}

	// Execute mutation
	data, err := client.Execute(createZenhubEpicKeyDateMutation, map[string]any{
		"input": map[string]any{
			"zenhubEpicId": resolved.ID,
			"date":         date,
			"description":  name,
		},
	})
	if err != nil {
		return exitcode.General("creating key date", err)
	}

	// Parse response
	var resp struct {
		CreateZenhubEpicKeyDate struct {
			KeyDate struct {
				ID          string `json:"id"`
				Date        string `json:"date"`
				Description string `json:"description"`
			} `json:"keyDate"`
			ZenhubEpic struct {
				ID    string `json:"id"`
				Title string `json:"title"`
			} `json:"zenhubEpic"`
		} `json:"createZenhubEpicKeyDate"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return exitcode.General("parsing key date response", err)
	}

	if output.IsJSON(outputFormat) {
		return output.JSON(w, map[string]any{
			"operation": "add",
			"epic":      map[string]any{"id": resolved.ID, "title": resolved.Title},
			"keyDate": map[string]any{
				"id":          resp.CreateZenhubEpicKeyDate.KeyDate.ID,
				"description": resp.CreateZenhubEpicKeyDate.KeyDate.Description,
				"date":        resp.CreateZenhubEpicKeyDate.KeyDate.Date,
			},
		})
	}

	output.MutationSingle(w, output.Green(fmt.Sprintf("Added key date %q (%s) to epic %q.", name, date, resolved.Title)))
	return nil
}

// --- epic key-date remove ---

func runEpicKeyDateRemove(cmd *cobra.Command, args []string) error {
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
		return exitcode.Usage(fmt.Sprintf("epic %q is a legacy epic (backed by GitHub issue %s/%s#%d) — key dates are only supported for ZenHub epics",
			resolved.Title, resolved.RepoOwner, resolved.RepoName, resolved.IssueNumber))
	}

	name := args[1]

	// Fetch existing key dates to find the one matching by name
	keyDates, err := fetchEpicKeyDates(client, resolved.ID)
	if err != nil {
		return err
	}

	// Find the key date by name (case-insensitive)
	var match *keyDateNode
	for i := range keyDates {
		if strings.EqualFold(keyDates[i].Description, name) {
			match = &keyDates[i]
			break
		}
	}

	if match == nil {
		return exitcode.NotFoundError(fmt.Sprintf("key date %q not found on epic %q", name, resolved.Title))
	}

	// Dry run
	if epicKeyDateRemoveDryRun {
		if output.IsJSON(outputFormat) {
			return output.JSON(w, map[string]any{
				"dryRun":    true,
				"operation": "remove",
				"epic":      map[string]any{"id": resolved.ID, "title": resolved.Title},
				"keyDate":   map[string]any{"id": match.ID, "description": match.Description, "date": match.Date},
			})
		}

		header := fmt.Sprintf("Would remove key date %q (%s) from epic %q", match.Description, match.Date, resolved.Title)
		output.MutationDryRun(w, header, nil)
		return nil
	}

	// Execute mutation
	data, err := client.Execute(deleteZenhubEpicKeyDateMutation, map[string]any{
		"input": map[string]any{
			"keyDateId": match.ID,
		},
	})
	if err != nil {
		return exitcode.General("deleting key date", err)
	}

	// Parse response
	var resp struct {
		DeleteZenhubEpicKeyDate struct {
			KeyDate struct {
				ID          string `json:"id"`
				Date        string `json:"date"`
				Description string `json:"description"`
			} `json:"keyDate"`
		} `json:"deleteZenhubEpicKeyDate"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return exitcode.General("parsing key date response", err)
	}

	if output.IsJSON(outputFormat) {
		return output.JSON(w, map[string]any{
			"operation": "remove",
			"epic":      map[string]any{"id": resolved.ID, "title": resolved.Title},
			"keyDate": map[string]any{
				"id":          resp.DeleteZenhubEpicKeyDate.KeyDate.ID,
				"description": resp.DeleteZenhubEpicKeyDate.KeyDate.Description,
				"date":        resp.DeleteZenhubEpicKeyDate.KeyDate.Date,
			},
		})
	}

	output.MutationSingle(w, output.Green(fmt.Sprintf("Removed key date %q (%s) from epic %q.", match.Description, match.Date, resolved.Title)))
	return nil
}
