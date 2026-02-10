# Manual Testing: `zh issue move`

## Summary

`zh issue move` works correctly across all tested scenarios. One bug was found and fixed in the shared `MutationItem` and `FailedItem` structs (missing JSON tags causing capitalized keys in JSON output).

## Bug Found and Fixed

**JSON output had capitalized field names**: `MutationItem` and `FailedItem` structs in `internal/output/mutation.go` lacked `json:"..."` tags, causing Go's default behavior of exporting field names with capital letters (e.g., `"Ref"` instead of `"ref"`). Added proper JSON tags to both structs. Also added `omitempty` to `MutationItem.Context` since it's empty in non-dry-run output.

## Tests Performed

### Basic Move Operations

| # | Test | Command | Result |
|---|------|---------|--------|
| 1 | Single issue move (repo#number) | `zh issue move task-tracker#2 Doing` | OK - Moved successfully |
| 2 | Move with owner/repo#number | `zh issue move dlakehammond/task-tracker#2 Todo` | OK |
| 3 | Move with --repo and bare number | `zh issue move --repo=task-tracker 3 Doing` | OK |
| 4 | Batch move (multiple issues) | `zh issue move task-tracker#3 recipe-book#1 Todo` | OK - Both moved, batch output shown |
| 5 | Move using ZenHub ID | `zh issue move Z2lkOi8v... Doing --dry-run` | OK - Resolved correctly |
| 6 | Move a PR | `zh issue move task-tracker#5 Doing --dry-run` | OK - PRs movable like issues |
| 7 | Cross-repo batch move | `zh issue move task-tracker#2 recipe-book#2 Doing --dry-run` | OK |
| 8 | Move to empty pipeline | `zh issue move task-tracker#2 Test` | OK |
| 9 | Move to current pipeline (no-op) | `zh issue move task-tracker#2 Todo` | OK - Silent success |
| 10 | Move PR by branch name | `zh issue move --repo=task-tracker fix/empty-tasks-file Doing --dry-run` | OK - Branch resolved to PR #5 |

### Position Flag

| # | Test | Command | Result |
|---|------|---------|--------|
| 11 | --position=top | `zh issue move task-tracker#4 Doing --position=top` | OK |
| 12 | --position=bottom | `zh issue move task-tracker#4 Todo --position=bottom` | OK |
| 13 | --position=0 (numeric) | `zh issue move task-tracker#4 Doing --position=0` | OK |
| 14 | Numeric position with batch | `zh issue move task-tracker#2 task-tracker#3 Doing --position=1` | OK - Exit 2, usage error |
| 15 | Invalid position | `zh issue move task-tracker#2 Doing --position=abc` | OK - Exit 2 |
| 16 | Negative position | `zh issue move task-tracker#2 Doing --position=-1` | OK - Exit 2 |

### Dry Run

| # | Test | Command | Result |
|---|------|---------|--------|
| 17 | Basic dry-run | `zh issue move task-tracker#2 task-tracker#3 Doing --dry-run` | OK - Shows "Would move", current pipeline |
| 18 | Dry-run with position | `zh issue move task-tracker#2 Doing --dry-run --position=top` | OK - Shows "at top" |
| 19 | Dry-run doesn't execute | Verified board unchanged after dry-run | OK |

### Pipeline Identifiers

| # | Test | Command | Result |
|---|------|---------|--------|
| 20 | Exact pipeline name | `zh issue move task-tracker#2 Doing` | OK |
| 21 | Pipeline substring (unique) | `zh issue move task-tracker#2 oing --dry-run` | OK - Matched "Doing" |
| 22 | Pipeline substring (ambiguous) | `zh issue move task-tracker#2 Do --dry-run` | OK - Exit 2, lists candidates |
| 23 | Pipeline alias | `zh issue move task-tracker#2 todo --dry-run` | OK - Resolved alias |
| 24 | Pipeline ID | `zh issue move task-tracker#2 Z2lkOi8v... --dry-run` | OK |
| 25 | Non-existent pipeline | `zh issue move task-tracker#2 Nonexistent` | OK - Exit 4, helpful message |

### Error Handling

| # | Test | Command | Result |
|---|------|---------|--------|
| 26 | Non-existent issue | `zh issue move task-tracker#9999 Doing` | OK - Exit 4 |
| 27 | Not enough args | `zh issue move Doing` | OK - Exit 2 |
| 28 | Stop on error (default) | `zh issue move task-tracker#9999 task-tracker#2 Doing` | OK - Stops at first failure, #2 not moved |
| 29 | --continue-on-error | `zh issue move task-tracker#2 task-tracker#9999 Doing --continue-on-error` | OK - Partial success output |

### Output Formats

| # | Test | Command | Result |
|---|------|---------|--------|
| 30 | Default output (single) | Single move | OK - "Moved task-tracker#2 to \"Doing\"." |
| 31 | Default output (batch) | Multi move | OK - Header + indented list |
| 32 | JSON output | `--output=json` | OK (after fix) - lowercase keys |
| 33 | Verbose output | `--verbose` | OK - Shows API requests/responses |

## Board State

Board was restored to its original state after testing. All moves were either dry-runs or reversed.
