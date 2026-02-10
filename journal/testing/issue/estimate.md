# Manual Testing: `zh issue estimate`

## Summary

`zh issue estimate` works correctly across all tested scenarios. No bugs were found. The command correctly sets, clears, and validates estimates on issues and PRs, with proper support for all identifier types, output formats, dry-run, and error handling.

## Tests Performed

### Set Estimate

| # | Test | Command | Result |
|---|------|---------|--------|
| 1 | Set estimate (repo#number) | `zh issue estimate task-tracker#2 5` | OK - "Set estimate on task-tracker#2 to 5." |
| 2 | Verify via `issue show` | `zh issue show task-tracker#2` | OK - Shows "Estimate: 5" |
| 3 | Set estimate on different repo | `zh issue estimate recipe-book#1 5` | OK |
| 4 | Set estimate on a PR | `zh issue estimate task-tracker#5 3` | OK - PRs support estimates |
| 5 | Set same value already set | `zh issue estimate task-tracker#2 13 --output=json` | OK - previous=13, current=13 |

### Clear Estimate

| # | Test | Command | Result |
|---|------|---------|--------|
| 6 | Clear estimate (omit value) | `zh issue estimate task-tracker#2` | OK - "Cleared estimate from task-tracker#2." |
| 7 | Clear estimate JSON output | `zh issue estimate recipe-book#1 --output=json` | OK - previous=8, current=null |

### Identifier Types

| # | Test | Command | Result |
|---|------|---------|--------|
| 8 | repo#number | `zh issue estimate task-tracker#2 5` | OK |
| 9 | owner/repo#number | `zh issue estimate dlakehammond/task-tracker#2 --dry-run` | OK - Resolved correctly |
| 10 | --repo with bare number | `zh issue estimate --repo=task-tracker 2 3` | OK |
| 11 | --repo with owner/repo | `zh issue estimate --repo=dlakehammond/task-tracker 2 13` | OK |
| 12 | ZenHub node ID | `zh issue estimate Z2lkOi8vcmFwdG9yL0lzc3VlLzM4NjI2MTg1NA 8` | OK - Resolved to recipe-book#1 |

### Dry Run

| # | Test | Command | Result |
|---|------|---------|--------|
| 13 | Dry-run set (with current estimate) | `zh issue estimate task-tracker#2 8 --dry-run` | OK - "Would set estimate on task-tracker#2 to 8" with "(currently: 3)" |
| 14 | Dry-run clear (with current estimate) | `zh issue estimate dlakehammond/task-tracker#2 --dry-run` | OK - "Would clear estimate from task-tracker#2" with "(currently: 3)" |
| 15 | Dry-run set (no current estimate) | `zh issue estimate task-tracker#3 1 --dry-run` | OK - Shows "(currently: none)" |

### Validation and Error Handling

| # | Test | Command | Result |
|---|------|---------|--------|
| 16 | Invalid estimate value | `zh issue estimate task-tracker#2 7` | OK - Exit 2, lists valid values (1, 2, 3, 5, 8, 13, 21, 40) |
| 17 | Non-numeric value | `zh issue estimate task-tracker#2 abc` | OK - Exit 2, "must be a number" |
| 18 | Non-existent issue | `zh issue estimate task-tracker#999 5` | OK - Exit 4, "not found" |
| 19 | No arguments | `zh issue estimate` | OK - Exit 2, "accepts between 1 and 2 arg(s)" |
| 20 | Too many arguments | `zh issue estimate a b c` | OK - Exit 2, "accepts between 1 and 2 arg(s)" |

### Output Formats

| # | Test | Command | Result |
|---|------|---------|--------|
| 21 | Default output (set) | `zh issue estimate task-tracker#2 5` | OK - "Set estimate on task-tracker#2 to 5." |
| 22 | Default output (clear) | `zh issue estimate task-tracker#2` | OK - "Cleared estimate from task-tracker#2." |
| 23 | JSON output (set) | `zh issue estimate task-tracker#2 3 --output=json` | OK - Proper JSON with previous/current |
| 24 | JSON output (clear) | `zh issue estimate recipe-book#1 --output=json` | OK - current: null |
| 25 | Verbose output | `zh issue estimate task-tracker#5 3 --verbose` | OK - Shows all API requests/responses |
| 26 | Help flag | `zh issue estimate --help` | OK - Shows usage, examples, flags |

## Test Data Cleanup

All modified estimates were restored to their original values after testing:
- task-tracker#2: cleared (was originally unset)
- task-tracker#5: cleared (was originally unset)
- recipe-book#1: cleared (was originally unset)
- task-tracker#1: unchanged (estimate of 1 throughout)
