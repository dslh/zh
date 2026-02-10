package cmd

import (
	"strings"

	"github.com/dslh/zh/internal/exitcode"
	"github.com/spf13/cobra"
)

var (
	verbose      bool
	outputFormat string
)

var rootCmd = &cobra.Command{
	Use:               "zh",
	Short:             "ZenHub CLI — like gh, but for ZenHub",
	Long:              `zh is a command-line tool for interacting with ZenHub. Manage your board, issues, epics, sprints, and more from the terminal.`,
	SilenceUsage:      true,
	SilenceErrors:     true,
	PersistentPreRunE: setupPersistentPreRun,
	RunE:              runRoot,
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output (log API requests/responses to stderr)")
	rootCmd.PersistentFlags().StringVarP(&outputFormat, "output", "o", "", "Output format: json")
}

func Execute() error {
	err := rootCmd.Execute()
	if err != nil {
		// Cobra's built-in validators (ExactArgs, MinimumNArgs, etc.) and
		// flag parsing errors return plain errors. Wrap them as usage errors
		// so they exit with code 2.
		if _, ok := err.(*exitcode.Error); !ok && isCobraUsageError(err) {
			return exitcode.Usage(err.Error())
		}
	}
	return err
}

// isCobraUsageError returns true if the error looks like a Cobra argument
// validation or flag parsing error.
func isCobraUsageError(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "arg(s)") ||
		strings.HasPrefix(msg, "unknown command") ||
		strings.HasPrefix(msg, "unknown flag") ||
		strings.HasPrefix(msg, "unknown shorthand flag")
}
