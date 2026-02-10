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

// cssVarColors maps ZenHub CSS theme variables to hex color codes.
var cssVarColors = map[string]string{
	"var(--zh-theme-color-red-primary)":    "#ff5630",
	"var(--zh-theme-color-orange-primary)": "#ff7452",
	"var(--zh-theme-color-yellow-primary)": "#ffab00",
	"var(--zh-theme-color-green-primary)":  "#36b37e",
	"var(--zh-theme-color-teal-primary)":   "#00b8d9",
	"var(--zh-theme-color-blue-primary)":   "#0065ff",
	"var(--zh-theme-color-purple-primary)": "#6554c0",
}

// formatPriorityColor formats a color value for display.
// Handles hex codes (with or without '#' prefix) and ZenHub CSS variable references.
func formatPriorityColor(color string) string {
	if strings.HasPrefix(color, "#") {
		return color
	}

	if strings.HasPrefix(color, "var(") {
		if hex, ok := cssVarColors[color]; ok {
			return hex
		}
		// Extract color name from unknown CSS variable
		// e.g. "var(--zh-theme-color-red-primary)" → "red"
		name := color
		name = strings.TrimPrefix(name, "var(")
		name = strings.TrimSuffix(name, ")")
		name = strings.TrimPrefix(name, "--zh-theme-color-")
		if idx := strings.LastIndex(name, "-"); idx >= 0 {
			name = name[:idx]
		}
		return name
	}

	// Bare hex code
	return "#" + color
}
