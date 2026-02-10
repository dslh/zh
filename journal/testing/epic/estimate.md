# Manual Testing: `zh epic estimate`

## Summary

Tested `zh epic estimate <epic> [value]` for setting, clearing, and dry-running estimates on ZenHub epics. All functionality works as expected with no bugs found.

## Test Environment

- Workspace: Dev Test (`69866ab95c14bf002977146b`)
- ZenHub epics tested: "Q1 Platform Improvements", "Bug Bash Sprint"
- Legacy epics tested: "Legacy Epic Test", "Recipe Book Improvements"

## Tests Performed

### Help text
- `zh epic estimate --help` — displays correct usage, examples, and flags.

### Setting estimates
- `zh epic estimate "Q1 Platform Improvements" 13` — set estimate to 13. Confirmed via `epic show`.
- `zh epic estimate "Q1 Platform Improvements" 5` — changed estimate to 5. Confirmed.
- `zh epic estimate "Q1 Platform Improvements" 0` — set estimate to 0. Accepted.
- `zh epic estimate "Q1 Platform Improvements" 2.5` — decimal values accepted.

### Clearing estimates
- `zh epic estimate "Q1 Platform Improvements"` — cleared the estimate. Verified via `epic show` (shows "None").

### Dry-run mode
- `zh epic estimate "Q1 Platform" 8 --dry-run` — shows "Would set estimate... (currently: 5)".
- `zh epic estimate "Q1 Platform" --dry-run` — shows "Would clear estimate... (currently: 5)".
- `zh epic estimate "Q1 Platform" 5 --dry-run` (no current estimate) — shows "(currently: none)".
- `zh epic estimate "Q1 Platform" --dry-run` (no current estimate) — shows "(currently: none)".

### JSON output
- `zh epic estimate "Q1 Platform" --output=json` — valid JSON with `epic.id`, `epic.title`, `estimate.previous`, `estimate.current` (null for clear).
- `zh epic estimate "Q1 Platform" 21 --output=json` — shows `"current": 21` and `"previous": null`.

### Epic identifier types
- Full title: `"Q1 Platform Improvements"` — works.
- Substring: `"Bug Bash"` — resolves to "Bug Bash Sprint".
- ZenHub ID: `Z2lkOi8vcmFwdG9yL1plbmh1YkVwaWMvMjMyNDIy` — resolves correctly.

### Legacy epic handling
- `zh epic estimate "Legacy Epic Test" 5` — correctly rejects with exit code 2 and message explaining it's a legacy epic.
- `zh epic estimate "Recipe Book" 5` — same proper rejection for recipe-book legacy epic.

### Error handling
- No arguments: `zh epic estimate` — "accepts between 1 and 2 arg(s), received 0".
- Invalid value: `zh epic estimate "Q1 Platform" abc` — "invalid estimate value "abc" — must be a number" (exit 2).
- Too many args: `zh epic estimate "Q1 Platform" 5 extra` — "accepts between 1 and 2 arg(s), received 3".
- Nonexistent epic: `zh epic estimate "NonExistent" 5` — "epic not found" (exit 4).
- Negative value: `zh epic estimate "Q1 Platform" -- -5` — API returns HTTP 500 (expected; API rejects negative estimates).

### Verbose mode
- `zh epic estimate "Bug Bash Sprint" 3 --verbose` — shows API requests/responses to stderr with proper query and variable details.

## Bugs Found

None.

## Cleanup

Reset estimates on both "Q1 Platform Improvements" and "Bug Bash Sprint" to cleared state after testing.
