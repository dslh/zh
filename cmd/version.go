package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Build variables — set via ldflags at build time.
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Display version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Fprintf(cmd.OutOrStdout(), "zh version %s\n", Version)
		fmt.Fprintf(cmd.OutOrStdout(), "commit: %s\n", Commit)
		fmt.Fprintf(cmd.OutOrStdout(), "built:  %s\n", Date)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
