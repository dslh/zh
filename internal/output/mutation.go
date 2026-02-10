package output

import (
	"fmt"
	"io"
)

// MutationItem represents an item affected by a mutation.
type MutationItem struct {
	Ref     string // e.g. "mpt#1234"
	Title   string // e.g. "Fix login button alignment"
	Context string // e.g. "(currently in \"Backlog\")" for dry-run
}

// FailedItem represents an item that failed during a mutation.
type FailedItem struct {
	Ref    string // e.g. "api#568"
	Reason string // e.g. "Permission denied"
}

// MutationSingle prints a single-item confirmation.
// Example: "Set estimate on mpt#1234 to 5"
func MutationSingle(w io.Writer, message string) {
	fmt.Fprintln(w, message)
}

// MutationBatch prints a multi-item confirmation with an indented list.
//
// Example:
//
//	Moved 3 issues to "In Development":
//
//	  mpt#1234 Fix login button alignment
//	  mpt#1235 Update error messages
//	  api#567  Add rate limiting headers
func MutationBatch(w io.Writer, header string, items []MutationItem) {
	fmt.Fprintln(w, header)
	fmt.Fprintln(w)
	printItems(w, items)
}

// MutationPartialFailure prints successes followed by failures.
func MutationPartialFailure(w io.Writer, header string, succeeded []MutationItem, failed []FailedItem) {
	fmt.Fprintln(w, header)
	fmt.Fprintln(w)
	printItems(w, succeeded)

	fmt.Fprintln(w)
	fmt.Fprintln(w, Red("Failed:"))
	fmt.Fprintln(w)
	for _, f := range failed {
		fmt.Fprintf(w, "  %s  %s\n", f.Ref, Red(f.Reason))
	}
}

// MutationDryRun prints a dry-run confirmation using "Would" prefix.
// Items may include context showing before/after state.
func MutationDryRun(w io.Writer, header string, items []MutationItem) {
	fmt.Fprintln(w, Yellow(header))
	fmt.Fprintln(w)
	for _, item := range items {
		line := fmt.Sprintf("  %s", item.Ref)
		if item.Title != "" {
			line += " " + item.Title
		}
		if item.Context != "" {
			line += " " + Dim(item.Context)
		}
		fmt.Fprintln(w, Yellow(line))
	}
}

// DetailLine represents a key-value pair for single-entity dry-run output.
type DetailLine struct {
	Key   string // e.g. "Name", "Position", "Description"
	Value string // e.g. "Active Work", "3", "QA verification"
}

// MutationDryRunDetail prints a dry-run confirmation for a single-entity
// operation with key-value detail lines. Use this for create, edit, delete,
// and state-change operations on a single entity.
//
// Example:
//
//	Would create pipeline "QA Review" at position 3.
//
//	  Description: QA verification
func MutationDryRunDetail(w io.Writer, header string, details []DetailLine) {
	fmt.Fprintln(w, Yellow(header))
	if len(details) > 0 {
		// Compute max key width for alignment.
		maxKey := 0
		for _, d := range details {
			if len(d.Key) > maxKey {
				maxKey = len(d.Key)
			}
		}
		fmt.Fprintln(w)
		for _, d := range details {
			fmt.Fprintln(w, Yellow(fmt.Sprintf("  %-*s %s", maxKey+1, d.Key+":", d.Value)))
		}
	}
}

func printItems(w io.Writer, items []MutationItem) {
	// Compute max ref width for alignment.
	maxRef := 0
	for _, item := range items {
		if len(item.Ref) > maxRef {
			maxRef = len(item.Ref)
		}
	}
	for _, item := range items {
		fmt.Fprintf(w, "  %-*s %s\n", maxRef, item.Ref, item.Title)
	}
}
