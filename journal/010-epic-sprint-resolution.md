# 010: Epic & Sprint Resolution

Completed the remaining Phase 5 identifier resolution items: epic resolution and sprint resolution.

## Epic Resolution (`internal/resolve/epic.go`)

- Resolves epics by: ZenHub ID, exact title, unique title substring, config alias, or `owner/repo#number` / `repo#number` for legacy epics
- Fetches both epic types (ZenhubEpic and legacy Epic) via the workspace roadmap query, filtering out Project nodes
- Paginated fetch with cursor-based navigation for large epic lists
- Cache key: `epics-{workspace_id}.json` storing ID, title, type, and legacy issue reference info
- `FetchEpicsIntoCache()` for commands that already have epic data
- 10 tests covering: ID match, exact title, case-insensitive title, unique substring, alias (to title and to ID), repo#number for legacy epics, ambiguous match error, cache refresh on miss, not found after refresh, fetch with mixed types

## Sprint Resolution (`internal/resolve/sprint.go`)

- Resolves sprints by: ZenHub ID, exact name (custom or generated), unique substring, or relative reference (`current`, `next`, `previous`)
- Matches against both custom name and generated name for sprints that have both
- `DisplayName()` helper: returns custom name if set, generated name otherwise
- Relative references resolved via cached sprint accessors (activeSprint, upcomingSprint, previousSprint from the API)
- Fallback date-based active sprint detection when accessor is missing
- Paginated fetch; accessors captured from first page only
- Two cache keys: `sprints-{workspace_id}.json` for sprint list, `sprint-accessors-{workspace_id}.json` for relative reference IDs
- 14 tests covering: display name logic, ID match, exact name (generated and custom), case-insensitive, generated name when custom set, unique substring, ambiguous match, relative references (current/next/previous), no active sprint error, cache refresh on miss, not found after refresh, fetch with accessor caching, relative ref with no prior cache

## Verified

- All 60 resolve package tests pass
- Full project test suite passes
- `go vet` clean
- API query structures validated against live ZenHub API via MCP
