# zh issue connect / disconnect — Manual Testing

## Commands Tested

- `zh issue connect <issue> <pr>`
- `zh issue disconnect <issue> <pr>`

## Test Environment

- Repos: `dlakehammond/task-tracker` (issues #1–#4, PRs #5–#6), `dlakehammond/recipe-book` (issues #1–#3, PR #4)
- Workspace: Dev Test

## Test Results

### zh issue connect

| # | Test | Command | Result |
|---|------|---------|--------|
| 1 | Dry run with repo#number | `zh issue connect --dry-run task-tracker#3 task-tracker#5` | PASS — Shows "Would connect" with both items listed |
| 2 | Dry run with owner/repo#number | `zh issue connect --dry-run dlakehammond/task-tracker#3 dlakehammond/task-tracker#5` | PASS |
| 3 | Dry run with --repo flag | `zh issue connect --dry-run --repo=task-tracker 3 5` | PASS |
| 4 | Dry run with --repo=owner/repo | `zh issue connect --dry-run --repo=dlakehammond/task-tracker 3 5` | PASS |
| 5 | Swapped arguments (PR first) | `zh issue connect --dry-run task-tracker#5 task-tracker#3` | PASS — Error: "is a pull request, not an issue", exit code 2 |
| 6 | Too few arguments | `zh issue connect task-tracker#3` | PASS — Error: "accepts 2 arg(s), received 1", exit code 2 |
| 7 | Nonexistent issue | `zh issue connect task-tracker#999 task-tracker#5` | PASS — Error: "not found", exit code 4 (after bugfix) |
| 8 | JSON dry-run output | `zh issue connect --dry-run --output=json task-tracker#3 task-tracker#5` | PASS — Valid JSON with dryRun, issue, pr fields |
| 9 | Actual connect | `zh issue connect task-tracker#3 task-tracker#5` | PASS — "Connected task-tracker#5 to task-tracker#3.", verified via `issue show` |
| 10 | JSON output (actual) | `zh issue connect --output=json task-tracker#3 task-tracker#5` | PASS — Valid JSON with issue and pr objects |
| 11 | ZenHub ID format | `zh issue connect --dry-run <zenHubID1> <zenHubID2>` | PASS — Resolves correctly by node query |
| 12 | Cross-repo connect | `zh issue connect --dry-run recipe-book#1 task-tracker#5` | PASS — Works across repos in same workspace |
| 13 | Verbose flag | `zh issue connect --dry-run --verbose task-tracker#3 task-tracker#5` | PASS — Logs API requests/responses to stderr |
| 14 | Both args are issues | `zh issue connect task-tracker#1 task-tracker#3` | PASS — Error: "is an issue, not a pull request", exit code 2 |
| 15 | Nonexistent repo | `zh issue connect nonexistent#1 task-tracker#5` | PASS — Error: "not found in workspace", exit code 4 |
| 16 | Branch name resolution | `zh issue connect --dry-run --repo=task-tracker 3 fix/empty-tasks-file` | PASS — Resolves branch to PR #5 via GitHub CLI |

### zh issue disconnect

| # | Test | Command | Result |
|---|------|---------|--------|
| 1 | Dry run with repo#number | `zh issue disconnect --dry-run task-tracker#2 task-tracker#5` | PASS |
| 2 | Dry run with owner/repo#number | `zh issue disconnect --dry-run dlakehammond/task-tracker#2 dlakehammond/task-tracker#5` | PASS |
| 3 | Dry run with --repo flag | `zh issue disconnect --dry-run --repo=task-tracker 2 5` | PASS |
| 4 | JSON dry-run output | `zh issue disconnect --dry-run --output=json task-tracker#2 task-tracker#5` | PASS |
| 5 | Actual disconnect | `zh issue disconnect task-tracker#3 task-tracker#5` | PASS — Verified connection removed via `issue show` |
| 6 | JSON output (actual) | `zh issue disconnect --output=json task-tracker#3 task-tracker#5` | PASS |
| 7 | Disconnect non-connected pair | `zh issue disconnect task-tracker#3 task-tracker#5` | PASS — Error: "Not found", exit code 1 |
| 8 | Swapped arguments | `zh issue disconnect task-tracker#5 task-tracker#2` | PASS — Error: "is a pull request, not an issue", exit code 2 |
| 9 | ZenHub ID format | `zh issue disconnect --dry-run <zenHubID1> <zenHubID2>` | PASS |

## Bugs Found and Fixed

### Bug: Exit codes lost when errors are wrapped with fmt.Errorf

**Severity:** Medium — affects all commands that wrap resolver errors

**Description:** The `exitcode.ExitCode()` function used a direct type assertion `err.(*Error)` to extract exit codes. When errors were wrapped via `fmt.Errorf("resolving issue: %w", err)`, the `*exitcode.Error` type was nested inside a standard `fmt.wrapError`, causing the type assertion to fail. This meant NotFound errors (exit code 4) from the resolver were reported as GeneralError (exit code 1).

**Affected commands:** `issue connect`, `issue disconnect`, and any other command that wraps exitcode errors with `fmt.Errorf`.

**Fix:** Changed `exitcode.ExitCode()` to use `errors.As()` instead of direct type assertion, which properly unwraps nested errors to find the `*exitcode.Error`. Added tests for wrapped error scenarios.

**Files changed:**
- `internal/exitcode/errors.go` — Use `errors.As` in `ExitCode()`
- `internal/exitcode/exitcode_test.go` — Add wrapped error test cases

## Notes

- ZenHub API rejects connecting a PR to an issue when a "chained relation" already exists (e.g., the PR is already connected to another issue). Error: "Connection cannot be set because of chained relation".
- Disconnecting a non-existent connection returns a GraphQL "Not found" error with exit code 1, which is reasonable behavior.
- Branch name resolution requires `--repo` flag and GitHub CLI access.
