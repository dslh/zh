# Testing: `zh board`

**Date:** 2026-02-10
**Binary:** dev (commit 149963d)

## Summary

Both `zh board` and `zh board --pipeline <name>` are working correctly. No code bugs were found. One API-level behavioral note was identified regarding PRs in filtered pipeline views.

## Tests Performed

### `zh board` (default view)

| Test | Result | Notes |
|------|--------|-------|
| Default board display | Pass | Shows all 3 pipelines (Todo, Doing, Test) with correct issue counts |
| Issue rendering | Pass | Shows repo#number, title, estimate (when present), assignees |
| Empty pipeline rendering | Pass | "Test" pipeline shows "No issues" |
| Footer summary | Pass | Shows "3 pipeline(s), 10 issue(s)" |
| `--output=json` | Pass | Returns array of pipeline objects with nested issues |
| `--verbose` | Pass | Logs full GraphQL request/response to stderr |
| `--help` | Pass | Shows usage, flags, and description |

### `zh board --pipeline <name>`

| Test | Result | Notes |
|------|--------|-------|
| Exact name (`--pipeline=Todo`) | Pass | Shows single pipeline with issues |
| Exact name (`--pipeline=Doing`) | Pass | Shows single pipeline with 1 issue |
| Empty pipeline (`--pipeline=Test`) | Pass | Shows "No issues" |
| Substring match (`--pipeline=od`) | Pass | Resolves to "Todo" |
| Ambiguous substring (`--pipeline=Do`) | Pass | Exit code 2, lists both "Todo" and "Doing" as candidates |
| Nonexistent (`--pipeline=nonexistent`) | Pass | Exit code 4, suggests `zh pipeline list` |
| Pipeline ID | Pass | Resolves correctly using raw ZenHub ID |
| Pipeline alias (`--pipeline=todo`) | Pass | Alias set via `zh pipeline alias` resolves correctly |
| `--pipeline` with `--output=json` | Pass | Returns single-element array with pipeline and issues |

## Known Issues / Notes

### PR count discrepancy between full board and `--pipeline` views

The full board view shows PRs alongside issues within each pipeline (e.g., Todo shows 8 items), while the `--pipeline` filtered view excludes PRs (e.g., Todo shows 6 items). This is caused by a ZenHub API difference:

- **Full board** uses `pipelinesConnection.nodes.issues` which includes both issues and PRs
- **Filtered pipeline** uses `searchIssuesByPipeline` which returns only issues, not PRs

This was verified directly against the ZenHub GraphQL API. Both endpoints return correct data for their respective semantics — `searchIssuesByPipeline` simply does not include PRs.

**Impact:** Users may see different issue counts depending on whether they view the full board or filter to a pipeline. This is a ZenHub API behavior, not a `zh` bug.

### JSON output field differences

The full board JSON includes fewer fields per issue (no `blockingIssues`) compared to the filtered pipeline JSON (which includes `blockingIssues`). This is because the two code paths use different GraphQL queries. The difference is cosmetic and does not affect functionality.

## Test Suite

- `make test`: All tests pass
- `make lint`: 0 issues
