package output

import "github.com/spf13/cobra"

// DefaultLimit is the default maximum number of results for list commands.
const DefaultLimit = 100

// AddPaginationFlags registers --limit and --all flags on a Cobra command.
// The limit and all pointers will be populated when the command runs.
func AddPaginationFlags(cmd *cobra.Command, limit *int, all *bool) {
	cmd.Flags().IntVar(limit, "limit", DefaultLimit, "Maximum number of results to display")
	cmd.Flags().BoolVar(all, "all", false, "Fetch all results (ignore --limit)")
	cmd.MarkFlagsMutuallyExclusive("limit", "all")
}

// EffectiveLimit returns the resolved result limit: 0 (unlimited) when --all
// is set, otherwise the --limit value. A limit of 0 means no cap.
func EffectiveLimit(limit int, all bool) int {
	if all {
		return 0
	}
	return limit
}

// Truncate returns at most limit items from the slice. If limit is 0 or
// negative, it returns the full slice (no truncation). The second return
// value is true if the slice was truncated.
func Truncate[T any](items []T, limit int) ([]T, bool) {
	if limit <= 0 || len(items) <= limit {
		return items, false
	}
	return items[:limit], true
}
