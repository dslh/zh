package cmd

import (
	"fmt"
	"strings"

	"github.com/dslh/zh/internal/config"
	"github.com/dslh/zh/internal/exitcode"
	"github.com/dslh/zh/internal/output"
	"github.com/dslh/zh/internal/resolve"
	"github.com/spf13/cobra"
)

// Commands

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
	epicAliasDelete bool
	epicAliasList   bool
)

func init() {
	epicAliasCmd.Flags().BoolVar(&epicAliasDelete, "delete", false, "Remove an existing alias")
	epicAliasCmd.Flags().BoolVar(&epicAliasList, "list", false, "List all epic aliases")

	epicCmd.AddCommand(epicAliasCmd)
}

func resetEpicMutationFlags() {
	epicAliasDelete = false
	epicAliasList = false
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
