package cmd

import (
	"fmt"
	"strings"

	"github.com/dslh/zh/internal/output"
	"github.com/dslh/zh/internal/resolve"
	"github.com/spf13/cobra"
)

// Commands

var priorityCmd = &cobra.Command{
	Use:   "priority",
	Short: "View priorities configured for the workspace",
	Long:  `List priorities configured for the current ZenHub workspace.`,
}

var priorityListCmd = &cobra.Command{
	Use:   "list",
	Short: "List workspace priorities with their colors",
	Long:  `List all priorities configured for the current workspace, including their colors.`,
	RunE:  runPriorityList,
}

func init() {
	priorityCmd.AddCommand(priorityListCmd)
	rootCmd.AddCommand(priorityCmd)
}

// runPriorityList implements `zh priority list`.
func runPriorityList(cmd *cobra.Command, args []string) error {
	cfg, err := requireWorkspace()
	if err != nil {
		return err
	}

	client := newClient(cfg, cmd)
	w := cmd.OutOrStdout()

	priorities, err := resolve.FetchPriorities(client, cfg.Workspace)
	if err != nil {
		return err
	}

	if output.IsJSON(outputFormat) {
		return output.JSON(w, priorities)
	}

	if len(priorities) == 0 {
		fmt.Fprintln(w, "No priorities configured.")
		return nil
	}

	lw := output.NewListWriter(w, "PRIORITY", "COLOR")
	for _, p := range priorities {
		color := output.TableMissing
		if p.Color != "" {
			color = formatPriorityColor(p.Color)
		}
		lw.Row(p.Name, color)
	}

	lw.FlushWithFooter(fmt.Sprintf("Total: %d priority(s)", len(priorities)))
	return nil
}

// formatPriorityColor formats a hex color code for display.
// Ensures a '#' prefix if the color looks like a hex code.
func formatPriorityColor(color string) string {
	if strings.HasPrefix(color, "#") {
		return color
	}
	return "#" + color
}
