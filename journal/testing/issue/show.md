# Manual Testing: `zh issue show`

## Summary

`zh issue show` displays detailed information about a single issue or PR, including metadata, description, connected PRs, blockers, and links. All identifier types, flags, and output formats were tested and work correctly after one bug fix.

## Test Environment

- Workspace: Dev Test (`69866ab95c14bf002977146b`)
- Repos: `dlakehammond/task-tracker`, `dlakehammond/recipe-book`
- GitHub integration: configured via `gh` CLI

## Tests Performed

### Identifier types

| Format | Command | Result |
|--------|---------|--------|
| `repo#number` | `zh issue show task-tracker#1` | Pass |
| `owner/repo#number` | `zh issue show dlakehammond/task-tracker#1` | Pass |
| `--repo` with bare number | `zh issue show --repo=task-tracker 1` | Pass |
| `--repo` with `owner/repo` | `zh issue show --repo=dlakehammond/task-tracker 1` | Pass |
| ZenHub ID | `zh issue show Z2lkOi8vcmFwdG9yL0lzc3VlLzM4NjI2MTgzMA` | Pass |
| Branch name (with `--repo`) | `zh issue show --repo=task-tracker fix/empty-tasks-file` | Pass |

### Issue types

| Type | Command | Result |
|------|---------|--------|
| Issue (task-tracker#1) | Shows ISSUE header, description, connected PRs, labels | Pass |
| Issue (task-tracker#2) | Shows bug with connected PR | Pass |
| Issue (recipe-book#1) | Cross-repo issue display | Pass |
| PR (task-tracker#5) | Shows PR header, correct state | Pass |
| PR (task-tracker#6) | Shows PR with description | Pass |
| PR (recipe-book#4) | Cross-repo PR display | Pass |

### Output sections verified

- Title line with entity type (ISSUE/PR) and reference
- Metadata fields: State, Pipeline, Estimate, Priority, Author, Assignees, Labels, Sprint, Epic
- DESCRIPTION section with rendered markdown
- CONNECTED PRS section (when issue has connected PRs)
- LINKS section with GitHub and ZenHub URLs
- TIMELINE section with created date and pipeline transfer time

### Flags and options

| Flag | Result |
|------|--------|
| `--output=json` | Valid JSON with all fields including GitHub-enriched data (author, reactions) |
| `--verbose` | Logs API requests/responses to stderr |
| `--help` | Shows complete usage with all flags |
| `NO_COLOR=1` | No ANSI escape sequences in output |

### Error handling

| Scenario | Command | Result |
|----------|---------|--------|
| No arguments | `zh issue show` | `Error: requires an issue argument or --interactive flag` (exit 2) |
| Nonexistent issue | `zh issue show task-tracker#999` | `Error: issue dlakehammond/task-tracker#999 not found` (exit 4) |
| Nonexistent repo | `zh issue show nonexistent-repo#1` | `Error: repository "nonexistent-repo" not found...` (exit 4) |
| Bad branch name | `zh issue show --repo=task-tracker nonexistent-branch` | `Error: no PR found for branch...` (exit 4) |

## Bugs Found and Fixed

### Bug: NOT_FOUND GraphQL errors returned exit code 1 instead of 4

**Symptom:** When querying a nonexistent issue (e.g. `task-tracker#999`), ZenHub's GraphQL API returns an error response with `extensions.code: "NOT_FOUND"` rather than returning `null` for `issueByInfo`. The API client treated this as a general error (exit code 1) rather than a not-found error (exit code 4).

**Root cause:** The `graphQLError` struct in `internal/api/client.go` did not parse the `extensions` field from GraphQL error responses. The `resolveByRepoAndNumber` function in `internal/resolve/issue.go` wrapped all API execution errors as `exitcode.General`, not distinguishing NOT_FOUND errors.

**Fix:**
1. Added `Extensions` struct (with `Code` field) to `graphQLError` in `internal/api/client.go`
2. Added `IsNotFound()` method on `GraphQLError` and `IsGraphQLNotFound()` helper function
3. Updated `resolveByRepoAndNumber` in `internal/resolve/issue.go` to check for NOT_FOUND and return `exitcode.NotFoundError`
4. Updated `runIssueShowByInfo` and `runIssueShowByNode` in `cmd/issue.go` with the same NOT_FOUND check

**Files changed:**
- `internal/api/client.go` — added Extensions parsing and IsNotFound helpers
- `internal/resolve/issue.go` — check for NOT_FOUND before wrapping as general error
- `cmd/issue.go` — check for NOT_FOUND in show-by-info and show-by-node paths
