package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/dslh/zh/internal/output"
	"github.com/dslh/zh/internal/resolve"
	"github.com/spf13/cobra"
)

// Commands

var labelCmd = &cobra.Command{
	Use:   "label",
	Short: "View labels available in the workspace",
	Long:  `List labels available across all repositories connected to the current workspace.`,
}

var labelListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all labels in the workspace",
	Long:  `List all labels aggregated across all repositories in the current workspace. Labels with the same name across repos are deduplicated.`,
	RunE:  runLabelList,
}

func init() {
	labelCmd.AddCommand(labelListCmd)
	rootCmd.AddCommand(labelCmd)
}

// runLabelList implements `zh label list`.
func runLabelList(cmd *cobra.Command, args []string) error {
	cfg, err := requireWorkspace()
	if err != nil {
		return err
	}

	client := newClient(cfg, cmd)
	w := cmd.OutOrStdout()

	labels, err := resolve.FetchLabels(client, cfg.Workspace)
	if err != nil {
		return err
	}

	// Sort labels by name (case-insensitive)
	sort.Slice(labels, func(i, j int) bool {
		return strings.ToLower(labels[i].Name) < strings.ToLower(labels[j].Name)
	})

	if output.IsJSON(outputFormat) {
		return output.JSON(w, labels)
	}

	if len(labels) == 0 {
		fmt.Fprintln(w, "No labels found.")
		return nil
	}

	lw := output.NewListWriter(w, "LABEL", "COLOR")
	for _, l := range labels {
		color := output.TableMissing
		if l.Color != "" {
			color = "#" + l.Color
		}
		lw.Row(l.Name, color)
	}

	lw.FlushWithFooter(fmt.Sprintf("Total: %d label(s)", len(labels)))
	return nil
}
