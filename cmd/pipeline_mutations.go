package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/dslh/zh/internal/cache"
	"github.com/dslh/zh/internal/config"
	"github.com/dslh/zh/internal/exitcode"
	"github.com/dslh/zh/internal/output"
	"github.com/dslh/zh/internal/resolve"
	"github.com/spf13/cobra"
)

// GraphQL mutations

const createPipelineMutation = `mutation CreatePipeline($input: CreatePipelineInput!) {
  createPipeline(input: $input) {
    pipeline {
      id
      name
      description
      stage
      createdAt
    }
  }
}`

const updatePipelineMutation = `mutation UpdatePipeline($input: UpdatePipelineInput!) {
  updatePipeline(input: $input) {
    pipeline {
      id
      name
      description
      stage
      isDefaultPRPipeline
      updatedAt
    }
  }
}`

const deletePipelineMutation = `mutation DeletePipeline($input: DeletePipelineInput!) {
  deletePipeline(input: $input) {
    clientMutationId
    destinationPipeline {
      id
      name
      issues {
        totalCount
      }
    }
  }
}`

// Commands

var pipelineCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new pipeline in the workspace",
	Long: `Create a new pipeline with optional position and description.

The pipeline is added to the current workspace. If --position is not
specified, the pipeline is appended to the end of the board.`,
	Args: cobra.ExactArgs(1),
	RunE: runPipelineCreate,
}

var pipelineEditCmd = &cobra.Command{
	Use:   "edit <name>",
	Short: "Update a pipeline's name, position, or description",
	Long: `Update properties of an existing pipeline. Resolve the pipeline by name,
substring, alias, or ID.

Only the properties specified via flags are changed; other properties
remain unchanged.`,
	Args: cobra.ExactArgs(1),
	RunE: runPipelineEdit,
}

var pipelineDeleteCmd = &cobra.Command{
	Use:   "delete <name> --into=<name>",
	Short: "Delete a pipeline, moving its issues into the target pipeline",
	Long: `Delete a pipeline and move all its issues into the specified target pipeline.
Both the pipeline to delete and the target pipeline are resolved by name,
substring, alias, or ID.

The --into flag is required to ensure issues are not lost.`,
	Args: cobra.ExactArgs(1),
	RunE: runPipelineDelete,
}

var pipelineAliasCmd = &cobra.Command{
	Use:   "alias <name> <alias>",
	Short: "Set a shorthand name for a pipeline",
	Long: `Set a shorthand alias that can be used to reference the pipeline in
future commands. Aliases are stored in the config file.

Use --delete to remove an existing alias. Use --list to show all
pipeline aliases.`,
	Args: cobra.RangeArgs(0, 2),
	RunE: runPipelineAlias,
}

// Flag variables

var (
	pipelineCreatePosition    int
	pipelineCreateDescription string
	pipelineCreateDryRun      bool

	pipelineEditName        string
	pipelineEditPosition    int
	pipelineEditDescription string
	pipelineEditDryRun      bool

	pipelineDeleteInto   string
	pipelineDeleteDryRun bool

	pipelineAliasDelete bool
	pipelineAliasList   bool
)

func init() {
	pipelineCreateCmd.Flags().IntVar(&pipelineCreatePosition, "position", -1, "Zero-indexed position from the left")
	pipelineCreateCmd.Flags().StringVar(&pipelineCreateDescription, "description", "", "Pipeline description")
	pipelineCreateCmd.Flags().BoolVar(&pipelineCreateDryRun, "dry-run", false, "Show what would be created without executing")

	pipelineEditCmd.Flags().StringVar(&pipelineEditName, "name", "", "New pipeline name")
	pipelineEditCmd.Flags().IntVar(&pipelineEditPosition, "position", -1, "New zero-indexed position from the left")
	pipelineEditCmd.Flags().StringVar(&pipelineEditDescription, "description", "", "New description (use empty string to clear)")
	pipelineEditCmd.Flags().BoolVar(&pipelineEditDryRun, "dry-run", false, "Show what would change without executing")

	pipelineDeleteCmd.Flags().StringVar(&pipelineDeleteInto, "into", "", "Target pipeline for issues (required)")
	_ = pipelineDeleteCmd.MarkFlagRequired("into")
	pipelineDeleteCmd.Flags().BoolVar(&pipelineDeleteDryRun, "dry-run", false, "Show what would be deleted without executing")

	pipelineAliasCmd.Flags().BoolVar(&pipelineAliasDelete, "delete", false, "Remove an existing alias")
	pipelineAliasCmd.Flags().BoolVar(&pipelineAliasList, "list", false, "List all pipeline aliases")

	pipelineCmd.AddCommand(pipelineCreateCmd)
	pipelineCmd.AddCommand(pipelineEditCmd)
	pipelineCmd.AddCommand(pipelineDeleteCmd)
	pipelineCmd.AddCommand(pipelineAliasCmd)
}

// resetPipelineMutationFlags resets flag variables between test runs.
func resetPipelineMutationFlags() {
	pipelineCreatePosition = -1
	pipelineCreateDescription = ""
	pipelineCreateDryRun = false

	pipelineEditName = ""
	pipelineEditPosition = -1
	pipelineEditDescription = ""
	pipelineEditDryRun = false

	pipelineDeleteInto = ""
	pipelineDeleteDryRun = false

	pipelineAliasDelete = false
	pipelineAliasList = false
}

// runPipelineCreate implements `zh pipeline create <name>`.
func runPipelineCreate(cmd *cobra.Command, args []string) error {
	cfg, err := requireWorkspace()
	if err != nil {
		return err
	}

	client := newClient(cfg, cmd)
	w := cmd.OutOrStdout()
	name := args[0]

	if pipelineCreateDryRun {
		msg := fmt.Sprintf("Would create pipeline %q", name)
		if pipelineCreatePosition >= 0 {
			msg += fmt.Sprintf(" at position %d", pipelineCreatePosition)
		}
		msg += "."
		output.MutationSingle(w, output.Yellow(msg))

		if pipelineCreateDescription != "" {
			fmt.Fprintf(w, "\n%s\n", output.Yellow(fmt.Sprintf("  Description: %s", pipelineCreateDescription)))
		}
		return nil
	}

	input := map[string]any{
		"workspaceId": cfg.Workspace,
		"name":        name,
	}
	if pipelineCreatePosition >= 0 {
		input["position"] = pipelineCreatePosition
	}
	if pipelineCreateDescription != "" {
		input["description"] = pipelineCreateDescription
	}

	data, err := client.Execute(createPipelineMutation, map[string]any{
		"input": input,
	})
	if err != nil {
		return exitcode.General("creating pipeline", err)
	}

	var resp struct {
		CreatePipeline struct {
			Pipeline struct {
				ID          string  `json:"id"`
				Name        string  `json:"name"`
				Description *string `json:"description"`
				Stage       *string `json:"stage"`
				CreatedAt   string  `json:"createdAt"`
			} `json:"pipeline"`
		} `json:"createPipeline"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return exitcode.General("parsing create pipeline response", err)
	}

	created := resp.CreatePipeline.Pipeline

	// Invalidate pipeline cache
	_ = cache.Clear(resolve.PipelineCacheKey(cfg.Workspace))

	if output.IsJSON(outputFormat) {
		return output.JSON(w, created)
	}

	msg := fmt.Sprintf("Created pipeline %q.", created.Name)
	if pipelineCreatePosition >= 0 {
		msg = fmt.Sprintf("Created pipeline %q at position %d.", created.Name, pipelineCreatePosition)
	}
	output.MutationSingle(w, output.Green(msg))
	fmt.Fprintln(w)
	fmt.Fprintf(w, "  ID: %s\n", output.Cyan(created.ID))
	if created.Description != nil && *created.Description != "" {
		fmt.Fprintf(w, "  Description: %s\n", *created.Description)
	}

	return nil
}

// runPipelineEdit implements `zh pipeline edit <name>`.
func runPipelineEdit(cmd *cobra.Command, args []string) error {
	cfg, err := requireWorkspace()
	if err != nil {
		return err
	}

	client := newClient(cfg, cmd)
	w := cmd.OutOrStdout()

	// Check that at least one flag was provided
	hasName := pipelineEditName != ""
	hasPosition := pipelineEditPosition >= 0
	hasDescription := pipelineEditDescription != ""

	if !hasName && !hasPosition && !hasDescription {
		return exitcode.Usage("no changes specified — use --name, --position, or --description")
	}

	// Resolve the pipeline
	resolved, err := resolve.Pipeline(client, cfg.Workspace, args[0], cfg.Aliases.Pipelines)
	if err != nil {
		return err
	}

	if pipelineEditDryRun {
		msg := fmt.Sprintf("Would update pipeline %q:", resolved.Name)
		output.MutationSingle(w, output.Yellow(msg))
		fmt.Fprintln(w)
		if hasName {
			fmt.Fprintln(w, output.Yellow(fmt.Sprintf("  Name: %s -> %s", resolved.Name, pipelineEditName)))
		}
		if hasPosition {
			fmt.Fprintln(w, output.Yellow(fmt.Sprintf("  Position: -> %d", pipelineEditPosition)))
		}
		if hasDescription {
			fmt.Fprintln(w, output.Yellow(fmt.Sprintf("  Description: -> %s", pipelineEditDescription)))
		}
		return nil
	}

	input := map[string]any{
		"pipelineId": resolved.ID,
	}
	if hasName {
		input["name"] = pipelineEditName
	}
	if hasPosition {
		input["position"] = pipelineEditPosition
	}
	if hasDescription {
		input["description"] = pipelineEditDescription
	}

	data, err := client.Execute(updatePipelineMutation, map[string]any{
		"input": input,
	})
	if err != nil {
		return exitcode.General("updating pipeline", err)
	}

	var resp struct {
		UpdatePipeline struct {
			Pipeline struct {
				ID          string  `json:"id"`
				Name        string  `json:"name"`
				Description *string `json:"description"`
				Stage       *string `json:"stage"`
				UpdatedAt   string  `json:"updatedAt"`
			} `json:"pipeline"`
		} `json:"updatePipeline"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return exitcode.General("parsing update pipeline response", err)
	}

	updated := resp.UpdatePipeline.Pipeline

	// Invalidate pipeline cache
	_ = cache.Clear(resolve.PipelineCacheKey(cfg.Workspace))

	if output.IsJSON(outputFormat) {
		return output.JSON(w, updated)
	}

	output.MutationSingle(w, output.Green(fmt.Sprintf("Updated pipeline %q.", updated.Name)))

	return nil
}

// runPipelineDelete implements `zh pipeline delete <name> --into=<name>`.
func runPipelineDelete(cmd *cobra.Command, args []string) error {
	cfg, err := requireWorkspace()
	if err != nil {
		return err
	}

	client := newClient(cfg, cmd)
	w := cmd.OutOrStdout()

	// Resolve the pipeline to delete
	source, err := resolve.Pipeline(client, cfg.Workspace, args[0], cfg.Aliases.Pipelines)
	if err != nil {
		return err
	}

	// Resolve the destination pipeline
	dest, err := resolve.Pipeline(client, cfg.Workspace, pipelineDeleteInto, cfg.Aliases.Pipelines)
	if err != nil {
		return err
	}

	// Cannot delete into itself
	if source.ID == dest.ID {
		return exitcode.Usage("cannot delete pipeline into itself")
	}

	// Get issue count for the source pipeline
	detailData, err := client.Execute(pipelineDetailQuery, map[string]any{
		"pipelineId": source.ID,
	})
	if err != nil {
		return exitcode.General("fetching pipeline details", err)
	}

	var detailResp struct {
		Node struct {
			Issues issueCountConn `json:"issues"`
		} `json:"node"`
	}
	if err := json.Unmarshal(detailData, &detailResp); err != nil {
		return exitcode.General("parsing pipeline details", err)
	}

	issueCount := detailResp.Node.Issues.TotalCount

	if pipelineDeleteDryRun {
		msg := fmt.Sprintf("Would delete pipeline %q.", source.Name)
		output.MutationSingle(w, output.Yellow(msg))
		fmt.Fprintln(w)
		fmt.Fprintln(w, output.Yellow(fmt.Sprintf("  Pipeline ID: %s", source.ID)))
		fmt.Fprintln(w, output.Yellow(fmt.Sprintf("  Issues to move: %d", issueCount)))
		fmt.Fprintln(w, output.Yellow(fmt.Sprintf("  Destination: %s (%s)", dest.Name, dest.ID)))
		return nil
	}

	data, err := client.Execute(deletePipelineMutation, map[string]any{
		"input": map[string]any{
			"pipelineId":            source.ID,
			"destinationPipelineId": dest.ID,
		},
	})
	if err != nil {
		return exitcode.General("deleting pipeline", err)
	}

	var resp struct {
		DeletePipeline struct {
			DestinationPipeline struct {
				ID     string `json:"id"`
				Name   string `json:"name"`
				Issues struct {
					TotalCount int `json:"totalCount"`
				} `json:"issues"`
			} `json:"destinationPipeline"`
		} `json:"deletePipeline"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return exitcode.General("parsing delete pipeline response", err)
	}

	// Invalidate pipeline cache
	_ = cache.Clear(resolve.PipelineCacheKey(cfg.Workspace))

	if output.IsJSON(outputFormat) {
		return output.JSON(w, map[string]any{
			"deleted":     source.Name,
			"destination": resp.DeletePipeline.DestinationPipeline.Name,
			"issuesMoved": issueCount,
		})
	}

	output.MutationSingle(w, output.Green(fmt.Sprintf("Deleted pipeline %q.", source.Name)))
	if issueCount > 0 {
		fmt.Fprintf(w, "Moved %d issue(s) to %q.\n", issueCount, dest.Name)
	}

	return nil
}

// runPipelineAlias implements `zh pipeline alias <name> <alias>`.
func runPipelineAlias(cmd *cobra.Command, args []string) error {
	cfg, err := requireWorkspace()
	if err != nil {
		return err
	}

	w := cmd.OutOrStdout()

	// --list: show all pipeline aliases
	if pipelineAliasList {
		if output.IsJSON(outputFormat) {
			return output.JSON(w, cfg.Aliases.Pipelines)
		}

		if len(cfg.Aliases.Pipelines) == 0 {
			fmt.Fprintln(w, "No pipeline aliases configured.")
			return nil
		}

		lw := output.NewListWriter(w, "ALIAS", "PIPELINE")
		for alias, name := range cfg.Aliases.Pipelines {
			lw.Row(alias, name)
		}
		lw.FlushWithFooter(fmt.Sprintf("Total: %d alias(es)", len(cfg.Aliases.Pipelines)))
		return nil
	}

	// --delete: remove an alias
	if pipelineAliasDelete {
		if len(args) != 1 {
			return exitcode.Usage("usage: zh pipeline alias --delete <alias>")
		}
		alias := args[0]

		if cfg.Aliases.Pipelines == nil {
			return exitcode.NotFoundError(fmt.Sprintf("alias %q not found", alias))
		}

		if _, ok := cfg.Aliases.Pipelines[alias]; !ok {
			return exitcode.NotFoundError(fmt.Sprintf("alias %q not found", alias))
		}

		delete(cfg.Aliases.Pipelines, alias)
		if err := config.Write(cfg); err != nil {
			return exitcode.General("saving config", err)
		}

		output.MutationSingle(w, fmt.Sprintf("Removed alias %q.", alias))
		return nil
	}

	// Set an alias: requires exactly 2 args
	if len(args) != 2 {
		return exitcode.Usage("usage: zh pipeline alias <pipeline> <alias>")
	}

	pipelineName := args[0]
	alias := args[1]

	// Validate the pipeline exists
	client := newClient(cfg, cmd)
	resolved, err := resolve.Pipeline(client, cfg.Workspace, pipelineName, cfg.Aliases.Pipelines)
	if err != nil {
		return err
	}

	// Initialize map if needed
	if cfg.Aliases.Pipelines == nil {
		cfg.Aliases.Pipelines = make(map[string]string)
	}

	// Check if alias already exists
	if existing, ok := cfg.Aliases.Pipelines[alias]; ok {
		if strings.EqualFold(existing, resolved.Name) {
			fmt.Fprintf(w, "Alias %q already points to %q.\n", alias, resolved.Name)
			return nil
		}
		return exitcode.Usage(fmt.Sprintf("alias %q already exists (points to %q) — use --delete first to remove it", alias, existing))
	}

	cfg.Aliases.Pipelines[alias] = resolved.Name
	if err := config.Write(cfg); err != nil {
		return exitcode.General("saving config", err)
	}

	output.MutationSingle(w, fmt.Sprintf("Alias %q -> %q.", alias, resolved.Name))
	return nil
}
