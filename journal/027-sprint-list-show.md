# 027: Sprint list and show commands

Phase 12 (partial): `zh sprint list` and `zh sprint show`.

## What was done

- **`zh sprint list`** — Lists sprints in the workspace with state, name, dates, points progress, and closed issue count.
  - `--state=open|closed|all` filter flag
  - `--limit` and `--all` pagination flags
  - Active sprint highlighted with `▶ active` indicator
  - Caches sprint entries for resolution by subsequent commands
  - JSON output mode

- **`zh sprint show [sprint]`** — Detail view of a single sprint with progress bars and issue table.
  - Defaults to `current` (active sprint) when no argument given
  - Resolves sprint by name, substring, ZenHub ID, or relative reference (`current`, `next`, `previous`)
  - Progress section: points bar and issues bar
  - Issue table: ref, state, title, estimate, pipeline, assignee
  - Pagination for sprints with more than 100 issues
  - `--limit` and `--all` flags for issue display
  - JSON output mode

## Files changed

- `cmd/sprint.go` — New file: sprint list and show commands, GraphQL queries, types, rendering
- `cmd/sprint_test.go` — New file: 10 tests covering list (default, JSON, empty, state filter), show (by name, default current, JSON, not found, no issues), and help text

## Tests

10 tests, all passing. Full suite green. Linter clean.

## Design decisions

- Reused existing sprint resolution infrastructure from `internal/resolve/sprint.go` (built in Phase 5)
- Followed the epic list/show pattern closely — same pagination, caching, and output structure
- Sprint state display uses `▶ active` for the active sprint per spec, `open` for other open sprints, and dimmed `closed` for closed sprints
- Sprint duration shown in parentheses in detail view (e.g., "14 days")
- Points rendered with `formatEstimate` (shared with issue/epic commands) for consistency
