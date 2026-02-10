# zh issue close - Manual Testing

## Summary

Tested the `zh issue close` command across all supported identifier types, flags, and edge cases. One bug was found and fixed: dry-run mode did not respect `--output=json`.

## Test Environment

- Workspace: Dev Test (`69866ab95c14bf002977146b`)
- Repos: `dlakehammond/task-tracker`, `dlakehammond/recipe-book`
- Created test issues #9-#18 in task-tracker and #7-#8 in recipe-book for testing

## Tests Performed

### Identifier Types

| Test | Command | Result |
|------|---------|--------|
| repo#number | `zh issue close task-tracker#9` | Pass - closed successfully |
| owner/repo#number | `zh issue close dlakehammond/recipe-book#7` | Pass - closed successfully |
| --repo with bare number | `zh issue close --repo=task-tracker 10` | Pass - closed successfully |
| ZenHub ID | `zh issue close Z2lkOi8vcmFwdG9yL0lzc3VlLzM4NjQ0OTA2Nw` | Pass - closed successfully |
| Invalid repo name | `zh issue close nonexistent-repo#1` | Pass - exit code 4, clear error message |
| Non-existent issue | `zh issue close task-tracker#99999` | Pass - exit code 4, "not found" error |
| No arguments | `zh issue close` | Pass - exit code 2, usage error |

### Flags

| Test | Command | Result |
|------|---------|--------|
| --dry-run single | `zh issue close task-tracker#4 --dry-run` | Pass - shows "Would close 1 issue(s)" with "(open)" context |
| --dry-run batch | `zh issue close --repo=task-tracker 3 4 --dry-run` | Pass - shows both issues |
| --dry-run mixed repos | `zh issue close task-tracker#3 recipe-book#2 --dry-run` | Pass |
| --output=json | `zh issue close task-tracker#15 --output=json` | Pass - valid JSON with successCount |
| --output=json already closed | `zh issue close task-tracker#15 --output=json` | Pass - shows alreadyClosed array |
| --dry-run --output=json | `zh issue close task-tracker#4 --dry-run --output=json` | **Fixed** - was outputting human-readable text, now outputs JSON with dryRun: true |
| --verbose | `zh issue close task-tracker#4 --dry-run --verbose` | Pass - shows API request/response details on stderr |
| --continue-on-error | `zh issue close task-tracker#16 task-tracker#99999 --continue-on-error` | Pass - closes valid issue, reports failure for invalid |
| stop-on-error (default) | `zh issue close task-tracker#99999 task-tracker#17` | Pass - stops at first error, second issue not processed |

### Batch Operations

| Test | Command | Result |
|------|---------|--------|
| Multiple same-repo | `zh issue close --repo=task-tracker 13 14` | Pass - "Closed 2 issue(s)." with both listed |
| Multiple cross-repo | `zh issue close task-tracker#11 task-tracker#12 recipe-book#8` | Pass - "Closed 3 issue(s)." with aligned refs |
| Mix open + already-closed | `zh issue close task-tracker#17 task-tracker#18` | Pass - reports already-closed separately, closes open one |

### Edge Cases

| Test | Result |
|------|--------|
| Already-closed issue | Shows "1 issue already closed:" with ref, no error |
| Verified on GitHub | All closed issues confirmed CLOSED via `gh issue view` |
| Stop-on-error preserves state | Second issue confirmed still OPEN after first-issue error |

## Bug Found and Fixed

### Dry-run ignores --output=json

**Symptom:** `zh issue close task-tracker#4 --dry-run --output=json` outputs human-readable text instead of JSON.

**Root cause:** The `renderCloseDryRun` function did not check `outputFormat` before rendering. The dry-run code path returned early without checking if JSON output was requested.

**Fix:** Added a JSON check before the dry-run render call in `runIssueClose`, and added a new `renderCloseDryRunJSON` function that outputs structured JSON with `dryRun: true`, `wouldClose`, `alreadyClosed`, and `failed` fields. This follows the same pattern used by `epic delete` and `epic remove` dry-run JSON.

**Files changed:**
- `cmd/issue_close.go` - Added `renderCloseDryRunJSON` function and JSON check in dry-run path
- `cmd/issue_close_test.go` - Added `TestIssueCloseDryRunJSON` test

**Test added:** `TestIssueCloseDryRunJSON` verifies that dry-run with `--output=json` returns valid JSON containing `dryRun: true` and a `wouldClose` array.
