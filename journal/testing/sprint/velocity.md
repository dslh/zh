# zh sprint velocity

## Summary

The `zh sprint velocity` command displays velocity trends for recent sprints, including the workspace average velocity with trend direction and a table of sprint data.

## Test Results

All flags and output modes work correctly. One bug was found and fixed.

### Basic command (`zh sprint velocity`)

Displays the velocity detail view with:
- Header: "VELOCITY: Dev Test"
- Sprint cadence metadata (e.g., "2-week (SUNDAY - SUNDAY)")
- Average velocity with sprint count
- Table with columns: SPRINT, DATES, PTS DONE, PTS TOTAL, ISSUES, VELOCITY
- Active sprint marked with green `▶` and "(in progress)" label
- Closed sprints listed in reverse chronological order
- Footer with average velocity

### Flags tested

| Flag | Result | Notes |
|------|--------|-------|
| `--sprints=3` | Pass | Limits closed sprints to 3 |
| `--sprints=0` | Pass | Shows only active sprint (no closed sprints) |
| `--no-active` | Pass | Excludes active sprint row from table |
| `--sprints=3 --no-active` | Pass | Combined correctly |
| `--sprints=0 --no-active` | Pass | Shows "No closed sprints found." message |
| `--output=json` | Pass | Valid JSON with all fields |
| `--no-active --output=json` | Pass (after fix) | `activeSprint` is `null` |
| `--verbose` | Pass | Logs API request/response to stderr |
| `--help` | Pass | Shows usage with examples |

### Edge cases tested

| Scenario | Result | Notes |
|----------|--------|-------|
| Unexpected argument | Pass | Returns usage error (exit code 2) |
| Active sprint with data | Pass | Added issues with estimates to sprint, verified PTS TOTAL and ISSUES columns updated correctly |

### Data verification

Added issues with estimates to the active sprint to verify non-zero data displays correctly:
- PTS TOTAL showed the sum of estimates (8)
- ISSUES column showed "0/2" (0 closed out of 2 total)
- Cleaned up after testing (removed issues, restored estimates)

## Bug Found and Fixed

### `--no-active` flag not respected in JSON output

**File:** `cmd/sprint_reports.go`

**Problem:** When `--output=json` was combined with `--no-active`, the JSON output still included the active sprint data. The JSON serialization happened before the `--no-active` flag was checked, and the flag was only applied to the human-readable table rendering.

**Fix:** Added a check before JSON serialization: when `velocityNoActive` is true, set the `activeSprint` field to `nil` in the JSON output. Added a corresponding unit test (`TestSprintVelocityNoActiveJSON`).

## Files Modified

- `cmd/sprint_reports.go` - Fixed `--no-active` to apply to JSON output
- `cmd/sprint_reports_test.go` - Added `TestSprintVelocityNoActiveJSON` test
