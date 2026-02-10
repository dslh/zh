# 028: Sprint add and remove commands

## Summary

Implemented `zh sprint add` and `zh sprint remove` commands for adding/removing issues to/from sprints.

## Changes

- **`cmd/sprint_mutations.go`** (new): Sprint mutation commands
  - `zh sprint add <issue>...` — adds issues to a sprint (defaults to active sprint)
  - `zh sprint remove <issue>...` — removes issues from a sprint (defaults to active sprint)
  - `--sprint` flag to target a specific sprint (supports name, ID, `current`/`next`/`previous`)
  - `--repo` flag for bare issue numbers
  - `--dry-run` support on both commands
  - `--continue-on-error` for batch operations with partial failure reporting
  - JSON output mode support
  - Uses `addIssuesToSprints` and `removeIssuesFromSprints` GraphQL mutations
  - Follows existing batch mutation patterns from epic add/remove

- **`cmd/sprint_mutations_test.go`** (new): 14 tests covering:
  - Single and batch add/remove
  - Targeting specific sprints via `--sprint`
  - `--dry-run` output
  - JSON output
  - No active sprint error
  - `--continue-on-error` with partial failure
  - Help text includes add/remove subcommands

## Design decisions

- Followed the same pattern as `epic add`/`epic remove` for consistency: resolve issues individually, batch the mutation, render results with standard mutation output helpers.
- Reused `issueResolveForEpicQuery` from epic mutations for fetching issue title/repo details after resolution, since the query is identical.
- Both commands default to the active sprint when `--sprint` is not provided, matching the spec and research docs.
