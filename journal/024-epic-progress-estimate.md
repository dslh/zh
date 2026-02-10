# 024: Epic progress and estimate commands

## Scope

Phase 11 (partial): `zh epic progress` and `zh epic estimate` commands.

## Changes

### `zh epic progress <epic>`
- New read-only command showing epic completion status
- Displays issue count progress (closed/total) with progress bar
- Displays estimate progress (completed/total) with progress bar
- Supports both ZenHub and legacy epics
- JSON output includes structured issues/estimates breakdown
- Shows "No child issues" for empty epics
- Lightweight dedicated GraphQL queries (avoids fetching full epic detail)

### `zh epic estimate <epic> [value]`
- New mutation command to set or clear estimate on a ZenHub epic
- Uses `setMultipleEstimatesOnZenhubEpics` GraphQL mutation
- Fetches current estimate for dry-run context display
- `--dry-run` support showing current vs proposed value
- JSON output includes previous and current estimate values
- Legacy epics rejected with clear error message
- Validates value argument is numeric

### Tests
- `epic progress`: ZenHub epic, legacy epic, empty epic, JSON output (4 tests)
- `epic estimate`: set, clear, dry-run set, dry-run clear, JSON, invalid value, legacy error (7 tests)

### Files modified
- `cmd/epic.go` — progress command, queries, and registration
- `cmd/epic_mutations.go` — estimate command, mutation, query, and registration
- `cmd/epic_test.go` — progress tests and fixtures
- `cmd/epic_mutations_test.go` — estimate tests and fixtures
- `ROADMAP.md` — checked off completed items
