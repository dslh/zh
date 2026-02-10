# Manual Testing: `zh issue label`

## Summary

Tested `zh issue label add` and `zh issue label remove` subcommands. All functionality works correctly with no bugs found.

## Test Environment

- Workspace: Dev Test (`69866ab95c14bf002977146b`)
- Repos: `dlakehammond/task-tracker`, `dlakehammond/recipe-book`
- 10 labels available (bug, documentation, duplicate, enhancement, Epic, good first issue, help wanted, invalid, question, wontfix)

## Tests Performed

### `zh issue label add`

| # | Test | Command | Result |
|---|------|---------|--------|
| 1 | Single label, single issue (repo#number) | `zh issue label add task-tracker#3 -- documentation` | Pass - "Added label(s) documentation to task-tracker#3." |
| 2 | Multiple labels, multiple issues | `zh issue label add task-tracker#3 task-tracker#4 -- "help wanted" "good first issue"` | Pass - batch output with 2 issues listed |
| 3 | owner/repo#number format | `zh issue label add dlakehammond/recipe-book#1 -- documentation` | Pass |
| 4 | --repo flag with bare number | `zh issue label add --repo=task-tracker 2 -- documentation` | Pass |
| 5 | --repo flag with multiple bare numbers | `zh issue label add --repo=recipe-book 2 3 -- documentation` | Pass - batch output |
| 6 | ZenHub ID format | `zh issue label add Z2lkOi8v...833 -- wontfix` | Pass - resolved to task-tracker#4 |
| 7 | Case-insensitive label matching | `zh issue label add --dry-run task-tracker#1 -- BUG` | Pass - resolved to "bug" |
| 8 | Label on PR (not just issue) | `zh issue label add --dry-run task-tracker#5 -- documentation` | Pass - resolved PR title |
| 9 | Mixed repos in one command | `zh issue label add --dry-run task-tracker#1 recipe-book#1 -- documentation` | Pass |

### `zh issue label remove`

| # | Test | Command | Result |
|---|------|---------|--------|
| 10 | Single label, single issue | `zh issue label remove task-tracker#3 -- documentation` | Pass |
| 11 | Multiple labels, multiple issues | `zh issue label remove task-tracker#3 task-tracker#4 -- "help wanted" "good first issue"` | Pass |
| 12 | owner/repo#number format | `zh issue label remove dlakehammond/recipe-book#1 -- documentation` | Pass |
| 13 | --repo flag with multiple bare numbers | `zh issue label remove --repo=recipe-book 2 3 -- documentation` | Pass |
| 14 | ZenHub ID format | `zh issue label remove Z2lkOi8v...833 -- wontfix` | Pass |
| 15 | Remove label not present on issue | `zh issue label remove task-tracker#3 -- wontfix` | Pass - idempotent, returns success |

### Flags

| # | Test | Command | Result |
|---|------|---------|--------|
| 16 | --dry-run (add) | `zh issue label add --dry-run task-tracker#1 -- invalid` | Pass - "Would add..." output, no mutation |
| 17 | --dry-run (remove) | `zh issue label remove --dry-run task-tracker#1 -- invalid` | Pass - "Would remove..." output |
| 18 | --output=json (add) | `zh issue label add --output=json task-tracker#1 -- invalid` | Pass - JSON with operation, labels, succeeded, failed, successCount |
| 19 | --output=json (remove) | `zh issue label remove --output=json task-tracker#1 -- invalid` | Pass - JSON output |
| 20 | --continue-on-error (add) | `zh issue label add --continue-on-error task-tracker#1 task-tracker#999 -- duplicate` | Pass - partial success with failed section, exit code 1 |
| 21 | --continue-on-error (remove) | `zh issue label remove --continue-on-error task-tracker#1 task-tracker#999 -- duplicate` | Pass - partial success output |
| 22 | --verbose | `zh issue label add --dry-run --verbose task-tracker#1 -- documentation` | Pass - shows API requests/responses |

### Error Cases

| # | Test | Command | Result |
|---|------|---------|--------|
| 23 | No arguments | `zh issue label add` | Pass - exit code 2, "requires at least 1 arg(s)" |
| 24 | No -- separator | `zh issue label add task-tracker#1` | Pass - exit code 2, helpful error with example |
| 25 | No labels after -- | `zh issue label add task-tracker#1 --` | Pass - exit code 2, "at least one label name is required" |
| 26 | Nonexistent label | `zh issue label add task-tracker#1 -- nonexistent-label` | Pass - exit code 4, suggests `zh label list` |
| 27 | Nonexistent issue | `zh issue label add task-tracker#999 -- bug` | Pass - exit code 4, "not found" |
| 28 | Stop on first error (default) | `zh issue label add task-tracker#999 task-tracker#1 -- wontfix` | Pass - stops at first error, doesn't process remaining |

## Observations

- All label mutations were verified by checking `zh issue show` or `zh issue list` after each operation
- Labels added during testing were cleaned up, leaving issues in their original state
- `--dry-run` with `--output=json` shows human-readable dry-run output (same pattern as other commands like `issue move`)
- The `--` separator for distinguishing issues from labels is well-documented in help text and error messages

## Bugs Found

None.
