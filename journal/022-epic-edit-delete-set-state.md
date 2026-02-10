# 022: Epic edit, delete, and set-state commands

Phase 11 continuation — three new epic mutation commands.

## Commands added

### `zh epic edit <epic>`
- Accepts `--title` and `--body` flags (at least one required)
- Updates ZenHub epic via `updateZenhubEpic` mutation
- Returns error for legacy epics (must edit via GitHub)
- `--dry-run` shows what would change
- JSON output returns full updated epic object
- Invalidates epic cache on success

### `zh epic delete <epic>`
- Fetches child issue count before deletion for informational output
- Uses `deleteZenhubEpic` mutation
- `--dry-run` shows epic ID, state, and child issue count
- Returns error for legacy epics
- Invalidates epic cache on success

### `zh epic set-state <epic> <state>`
- Valid states: `open`, `todo`, `in_progress` (also `in-progress`, `inprogress`), `closed`
- `--apply-to-issues` flag to cascade state change to child issues
- Uses `updateZenhubEpicState` mutation
- `--dry-run` shows target state and whether child issues would be affected
- Returns error for legacy epics (state controlled by GitHub issue)
- Invalidates epic cache on success

## Tests
- 18 new tests covering: normal operation, dry-run, JSON output, legacy epic errors, missing flags, invalid state values
- All existing tests continue to pass

## Files changed
- `cmd/epic_mutations.go` — added 4 GraphQL mutations/queries, 3 commands, flag variables, and run functions
- `cmd/epic_mutations_test.go` — added tests and mock response helpers
- `ROADMAP.md` — checked off completed items
