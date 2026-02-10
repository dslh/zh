# Manual testing: zh issue activity

## Summary

Tested `zh issue activity <issue>` and its flags. Found and fixed three bugs related to unrecognized ZenHub timeline event keys.

## Bugs found and fixed

### Bug 1: Pipeline move events not parsed (issue.change_pipeline)

The code handled `issue.transfer_pipeline` but the actual ZenHub API returns `issue.change_pipeline`. Events fell through to the `formatEventKey` fallback, displaying as "change pipeline" without from/to pipeline names.

**Fix:** Added `issue.change_pipeline` as an alternate key in the switch case alongside `issue.transfer_pipeline`.

**Before:** `@dlakehammond change pipeline`
**After:** `@dlakehammond moved from "Todo" to "Doing"`

### Bug 2: Blocking issue events not parsed (issue.add_blocking_issue)

The `issue.add_blocking_issue` event type was not handled. Events fell through to the `formatEventKey` fallback, displaying as "add blocking issue" without identifying the blocked issue.

**Fix:** Added cases for `issue.add_blocking_issue` and `issue.remove_blocking_issue`, extracting `blocking_issue.number`, `blocking_issue.title`, and `blocking_issue_repository.name` from the event data.

**Before:** `@dlakehammond add blocking issue`
**After:** `@dlakehammond added blocking issue task-tracker#2 "Task list crashes when no tasks exist"`

### Bug 3: PR-side connect/disconnect events not parsed

When viewing activity for a PR (not an issue), connect/disconnect events use different keys: `issue.connect_pr_to_issue` and `issue.disconnect_pr_from_issue` (vs the issue-side `issue.connect_issue_to_pr` / `issue.disconnect_issue_from_pr`). The data structure also differs, using `issue` and `issue_repository` instead of `pull_request` and `pull_request_repository`.

**Fix:** Added cases for `issue.connect_pr_to_issue` and `issue.disconnect_pr_from_issue` with appropriate data extraction.

**Before:** `@dlakehammond connect pr to issue`
**After:** `@dlakehammond connected to issue task-tracker#3 "Add color output for task list"`

## Tests performed

### Identifier types
- `task-tracker#1` (repo#num) - works
- `dlakehammond/task-tracker#1` (owner/repo#num) - works
- `--repo=task-tracker 1` (--repo flag) - works
- `Z2lkOi8vcmFwdG9yL0lzc3VlLzM4NjI2MTgzMA` (ZenHub ID) - works
- `recipe-book#1` (different repo) - works
- `task-tracker#5` (PR) - works

### Flags
- `--github`: Merges GitHub timeline events (labels, comments, close/reopen) with ZenHub events, shows `[ZenHub]`/`[GitHub]` source tags. Tested with both issues and PRs.
- `--output=json`: Produces valid JSON with `issue` object and `events` array. Each event includes `time`, `source`, `description`, `actor`, and `raw` (for ZenHub events). Works correctly with and without `--github`.
- `--verbose`: Logs API requests/responses to stderr.
- `--help`: Shows usage with all flags documented.

### Error handling
- Non-existent repo (`nonexistent#999`): exit code 4, error message suggests `zh workspace repos`
- Non-existent issue (`task-tracker#999`): exit code 4, "not found" error
- No arguments: exit code 2, usage error

### Event types verified in live output
- `issue.set_estimate` / clearing estimates
- `issue.set_priority` / `issue.remove_priority`
- `issue.connect_issue_to_pr` (from issue side)
- `issue.connect_pr_to_issue` / `issue.disconnect_pr_from_issue` (from PR side)
- `issue.change_pipeline` (with from/to pipeline names)
- `issue.add_to_sprint` / `issue.remove_from_sprint`
- `issue.add_blocking_issue` (with blocked issue ref and title)
- GitHub: `LabeledEvent`, `UnlabeledEvent`, `CrossReferencedEvent`

## Test updates

Added unit tests for all new event types:
- `TestDescribeZenHubEventChangePipeline`
- `TestDescribeZenHubEventAddBlockingIssue`
- `TestDescribeZenHubEventRemoveBlockingIssue`
- `TestDescribeZenHubEventConnectPRToIssue`
- `TestDescribeZenHubEventDisconnectPRFromIssue`

All existing tests continue to pass. Full test suite and linter pass.
