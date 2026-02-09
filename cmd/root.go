package cmd

import (
	"github.com/spf13/cobra"
)

var (
	verbose      bool
	outputFormat string
)

var rootCmd = &cobra.Command{
	Use:           "zh",
	Short:         "ZenHub CLI — like gh, but for ZenHub",
	Long:          `zh is a command-line tool for interacting with ZenHub. Manage your board, issues, epics, sprints, and more from the terminal.`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output (log API requests/responses to stderr)")
	rootCmd.PersistentFlags().StringVarP(&outputFormat, "output", "o", "", "Output format: json")
}

func Execute() error {
	return rootCmd.Execute()
}
