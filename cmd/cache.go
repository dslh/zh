package cmd

import (
	"fmt"

	"github.com/dslh/zh/internal/cache"
	"github.com/dslh/zh/internal/config"
	"github.com/dslh/zh/internal/exitcode"
	"github.com/spf13/cobra"
)

var cacheCmd = &cobra.Command{
	Use:   "cache",
	Short: "Manage the local cache",
}

var cacheClearWorkspace bool

var cacheClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear cached data",
	Long:  `Clear all cached data, or use --workspace to clear only the current workspace's cache.`,
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if cacheClearWorkspace {
			cfg, err := config.Load()
			if err != nil {
				return exitcode.General("loading config", err)
			}
			if cfg.Workspace == "" {
				return exitcode.Usage("no workspace configured — use 'zh workspace switch' to set one")
			}
			if err := cache.ClearWorkspace(cfg.Workspace); err != nil {
				return exitcode.General("clearing workspace cache", err)
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Cleared cache for current workspace.")
			return nil
		}

		if err := cache.ClearAll(); err != nil {
			return exitcode.General("clearing cache", err)
		}
		fmt.Fprintln(cmd.OutOrStdout(), "Cleared all cached data.")
		return nil
	},
}

func init() {
	cacheClearCmd.Flags().BoolVar(&cacheClearWorkspace, "workspace", false, "Clear only the current workspace's cache")
	cacheCmd.AddCommand(cacheClearCmd)
	rootCmd.AddCommand(cacheCmd)
}
