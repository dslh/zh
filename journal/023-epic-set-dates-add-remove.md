# 023: Epic set-dates, add, and remove commands

## Summary

Implemented three new epic mutation commands: `set-dates`, `add`, and `remove`.

## Changes

### `zh epic set-dates <epic>`
- Set start/end dates with `--start` and `--end` flags (YYYY-MM-DD format)
- Clear dates with `--clear-start` and `--clear-end` flags
- Validates date format and conflicting flags (e.g. `--start` + `--clear-start`)
- Supports both ZenHub epics (`updateZenhubEpicDates`) and legacy epics (`updateEpicDates`)
- `--dry-run` and `--output=json` support

### `zh epic add <epic> <issue>...`
- Add one or more issues to a ZenHub epic
- Issue resolution via `repo#number`, `owner/repo#number`, ZenHub IDs, or bare numbers with `--repo`
- `--continue-on-error` for batch operations with partial failure reporting
- `--dry-run` and `--output=json` support
- Rejects legacy epics with informative error message

### `zh epic remove <epic> <issue>...`
- Remove one or more issues from a ZenHub epic
- Same issue resolution as `add`
- `--all` flag fetches and removes all child issues (with pagination)
- `--continue-on-error`, `--dry-run`, `--output=json` support
- Rejects legacy epics with informative error message

## Tests

- 22 new tests covering all three commands
- Coverage: success paths, dry-run, JSON output, validation errors, legacy epic rejection, batch operations, --all flag (with issues and empty), edge cases
- All existing tests continue to pass
- Linter clean

## Files changed

- `cmd/epic_mutations.go` — new mutations, commands, and implementation functions
- `cmd/epic_mutations_test.go` — 22 new tests and helper response functions
- `ROADMAP.md` — checked off completed items
