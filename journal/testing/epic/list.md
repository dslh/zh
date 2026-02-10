# Testing: `zh epic list`

## Summary

`zh epic list` lists all epics in the current workspace, including both ZenHub epics and legacy (issue-backed) epics.

## Test Environment

- Created 2 ZenHub epics ("Q1 Platform Improvements", "Bug Bash Sprint") and 1 legacy epic ("Recipe Book Improvements" backed by recipe-book#5) in the Dev Test workspace.

## Tests Performed

| Test | Result | Notes |
|------|--------|-------|
| `zh epic list` (basic) | PASS | Shows all 3 epics with correct TYPE, STATE, TITLE columns |
| `zh epic list --output=json` | PASS | Valid JSON array with all 3 epics, includes all fields |
| `zh epic list --limit=1` | PASS | Shows 1 epic with "Showing 1 of 3 epic(s)" footer |
| `zh epic list --limit=2` | PASS | Shows 2 epics with "Showing 2 of 3 epic(s)" footer |
| `zh epic list --all` | PASS | Shows all 3 epics, footer says "Showing 3 epic(s)" |
| `zh epic list --verbose` | PASS | Logs both API requests/responses to stderr |
| `zh epic list --help` | PASS | Shows usage, flags (--all, --limit, --output, --verbose) |
| Empty workspace | PASS (unit test) | Prints "No epics found." |

## Bugs Found and Fixed

### 1. Legacy epics not appearing in list (critical)

**Problem:** `zh epic list` only showed ZenHub epics. Legacy (issue-backed) epics were completely invisible. The code used the `workspace.roadmap.items` query to discover legacy epics, but legacy epics are not automatically added to the roadmap.

**Root cause:** The original implementation assumed that legacy epics would appear in the workspace roadmap. In practice, the ZenHub API has a dedicated `workspace.epics` field that returns legacy epics directly, but this was not being used.

**Fix:** Replaced the `workspace.roadmap.items` query with `workspace.epics` in both:
- `internal/resolve/epic.go` (epic resolution/cache layer)
- `cmd/epic.go` (list command)

This affects the following GraphQL queries:
- Resolution: `ListRoadmapEpics` -> `ListLegacyEpics` (uses `workspace.epics`)
- List: `ListRoadmapEpicsFull` -> `ListLegacyEpicsFull` (uses `workspace.epics`)

The `parseEpicListItem` and `parseRoadmapItem` functions (which parsed `__typename`-tagged union types from the roadmap) were replaced with simpler, typed response parsing since `workspace.epics` returns only legacy epics.

### 2. Inaccurate total count with --limit (minor)

**Problem:** When `--limit` was small enough that all results came from ZenHub epics only, the total count in the footer only reflected ZenHub epics (e.g., "1 of 2" instead of "1 of 3").

**Root cause:** When the limit was satisfied entirely by ZenHub epics, the legacy epic query was skipped, so the legacy `totalCount` was never retrieved.

**Fix:** Changed `fetchEpicList` to always make the legacy query (at least one item) so the `totalCount` is always available. The final count is `zenhubTotal + legacyTotal`.

## Files Changed

- `internal/resolve/epic.go` - Replaced roadmap-based legacy epic fetching with `workspace.epics` query
- `internal/resolve/epic_test.go` - Updated mock responses for new query structure
- `cmd/epic.go` - Replaced roadmap query with `workspace.epics`, fixed total count, removed unused `parseEpicListItem`
- `cmd/epic_test.go` - Updated mock responses for new query structure
- `cmd/epic_mutations_test.go` - Updated mock response for epic resolution query
