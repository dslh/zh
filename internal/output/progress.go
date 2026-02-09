package output

import (
	"fmt"
	"strings"
)

const barWidth = 20

// FormatProgress renders a progress bar in the format:
//
//	34/52 completed (65%)  █████████████░░░░░░░
//
// The bar is fixed at 20 characters using █ for filled and ░ for remaining.
func FormatProgress(completed, total int) string {
	pct := 0
	filled := 0
	if total > 0 {
		pct = completed * 100 / total
		filled = completed * barWidth / total
		if filled > barWidth {
			filled = barWidth
		}
	}

	bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)
	return fmt.Sprintf("%d/%d completed (%d%%)  %s", completed, total, pct, bar)
}
