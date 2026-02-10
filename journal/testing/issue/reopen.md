# Manual Testing: `zh issue reopen`

## Summary

Tested the `zh issue reopen` command end-to-end against the Dev Test workspace. Found and fixed 2 bugs.

## Test Results

| # | Test | Result |
|---|------|--------|
| 1 | Basic single issue reopen (`task-tracker#9 --pipeline=Todo`) | Pass |
| 2 | Batch reopen (2 issues, `task-tracker#9 task-tracker#10 --pipeline=Todo`) | Pass |
| 3 | Cross-repo batch (`task-tracker#9 recipe-book#7 --pipeline=Doing`) | Pass |
| 4 | `--position=top` (verified issue at top of pipeline) | Pass |
| 5 | `--position=bottom` (verified issue at bottom of pipeline) | Pass |
| 6 | Invalid position (`--position=3`) | Pass (exit code 2, clear error) |
| 7 | Missing required `--pipeline` flag | Pass (error message) |
| 8 | `--dry-run` with batch | Pass (shows "Would reopen", no state change) |
| 9 | `--dry-run --position=top` | Pass |
| 10 | `owner/repo#number` identifier format | Pass |
| 11 | `--repo` flag with bare number | **Fail** (fixed) |
| 12 | Already-open issue | Pass ("1 issue already open") |
| 13 | Mix of open and closed issues | Pass (reopens closed, reports open) |
| 14 | Pipeline substring matching (ambiguous) | Pass (lists candidates) |
| 15 | Pipeline substring matching (unique) | Pass |
| 16 | `--continue-on-error` with bad identifier | Pass (partial success) |
| 17 | Stop-on-first-error (default) | Pass (exit code 4) |
| 18 | `--output=json` | Pass |
| 19 | `--output=json --dry-run` | **Fail** (fixed) |
| 20 | ZenHub ID identifier | Pass |
| 21 | No arguments | Pass (exit code 2) |
| 22 | `--verbose` flag | Pass (shows API requests/responses) |
| 23 | `--repo=dlakehammond/task-tracker` (owner/repo format) | Pass |
| 24 | `--repo` with multiple bare numbers | Pass |

## Bugs Found and Fixed

### Bug 1: `--repo` flag not passed to issue resolver

**Symptom:** `zh issue reopen --repo=task-tracker 9 --pipeline=Todo` failed with "bare issue number 9 requires --repo flag" even though `--repo` was provided.

**Root cause:** The shared `resolveForClose()` function hardcoded `issueCloseRepo` (the close command's global flag variable) instead of accepting the repo flag as a parameter. When called from the reopen command, the close command's repo variable was always empty.

**Fix:** Added a `repoFlag` parameter to `resolveForClose()` and updated both call sites (close and reopen) to pass their respective repo flag variables.

**Files changed:** `cmd/issue_close.go`, `cmd/issue_reopen.go`

### Bug 2: `--dry-run` ignored `--output=json`

**Symptom:** `zh issue reopen task-tracker#9 --pipeline=Todo --output=json --dry-run` produced human-readable output instead of JSON.

**Root cause:** The dry-run code path in `runIssueReopen` did not check `output.IsJSON(outputFormat)` before rendering. The close command had this check, but it was missed when implementing reopen.

**Fix:** Added a JSON output check in the dry-run branch and a new `renderReopenDryRunJSON()` function that produces structured JSON output matching the style of the close command's dry-run JSON.

**Files changed:** `cmd/issue_reopen.go`

### Test added

Added `TestIssueReopenDryRunJSON` to `cmd/issue_close_test.go` to cover the new JSON dry-run code path.

## Environment State

All test issues were restored to their original open state after testing.
