package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var completionCmd = &cobra.Command{
	Use:   "completion <shell>",
	Short: "Generate shell completion scripts",
	Long: `Generate shell completion scripts for zh.

To load completions:

Bash:
  # Linux:
  $ zh completion bash > /etc/bash_completion.d/zh

  # macOS (requires bash-completion):
  $ zh completion bash > $(brew --prefix)/etc/bash_completion.d/zh

Zsh:
  # If shell completion is not already enabled in your zsh, add:
  #   autoload -U compinit; compinit
  $ zh completion zsh > "${fpath[1]}/_zh"

  # Or for Oh My Zsh:
  $ zh completion zsh > ~/.oh-my-zsh/completions/_zh

Fish:
  $ zh completion fish > ~/.config/fish/completions/zh.fish

After installing, restart your shell or source the completion file.`,
}

var completionBashCmd = &cobra.Command{
	Use:   "bash",
	Short: "Generate bash completion script",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return rootCmd.GenBashCompletionV2(cmd.OutOrStdout(), true)
	},
}

var completionZshCmd = &cobra.Command{
	Use:   "zsh",
	Short: "Generate zsh completion script",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return rootCmd.GenZshCompletion(cmd.OutOrStdout())
	},
}

var completionFishCmd = &cobra.Command{
	Use:   "fish",
	Short: "Generate fish completion script",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return rootCmd.GenFishCompletion(cmd.OutOrStdout(), true)
	},
}

var completionInstallHelp bool

func init() {
	completionCmd.Flags().BoolVar(&completionInstallHelp, "help-install", false, "Show detailed installation instructions")
	completionCmd.RunE = func(cmd *cobra.Command, args []string) error {
		if completionInstallHelp {
			fmt.Fprintln(cmd.OutOrStdout(), cmd.Long)
			return nil
		}
		return cmd.Help()
	}

	completionCmd.AddCommand(completionBashCmd)
	completionCmd.AddCommand(completionZshCmd)
	completionCmd.AddCommand(completionFishCmd)
	rootCmd.AddCommand(completionCmd)
}
