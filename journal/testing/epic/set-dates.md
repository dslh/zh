# Manual testing: zh epic set-dates

## Summary

Tested `zh epic set-dates <epic>` across both ZenHub and legacy epics, with all supported flags and identifier types. No bugs found.

## Test environment

- Workspace: Dev Test (`69866ab95c14bf002977146b`)
- ZenHub epics tested: "Q1 Platform Improvements", "Bug Bash Sprint"
- Legacy epics tested: `recipe-book#5`, `task-tracker#7`

## Tests performed

### Validation

| Test | Result |
|------|--------|
| No flags provided | Correct error: "at least one of --start, --end, --clear-start, or --clear-end must be provided" (exit 2) |
| `--start` with `--clear-start` | Correct error: "cannot set --start and --clear-start at the same time" (exit 2) |
| `--end` with `--clear-end` | Correct error: "cannot set --end and --clear-end at the same time" (exit 2) |
| Invalid date format (`03/01/2025`) | Correct error: "invalid date ... expected YYYY-MM-DD format" (exit 2) |
| Missing epic argument | Correct error: "accepts 1 arg(s), received 0" (exit 2) |
| Too many arguments | Correct error: "accepts 1 arg(s), received 2" (exit 2) |
| Non-existent epic | Correct error: "epic ... not found" (exit 4) |
| Ambiguous epic substring ("Legacy") | Lists 3 matching epics with IDs (exit 2) |

### ZenHub epic mutations

| Test | Result |
|------|--------|
| Set both `--start` and `--end` | Dates set correctly, confirmed via `epic show --output=json` |
| Set only `--start` (with end already set) | Start changed, end preserved |
| Clear both dates (`--clear-start --clear-end`) | Both dates cleared to None |
| Individual clear (`--clear-start` with end date set) | ZenHub API rejects: "Start on can't be blank" — correct passthrough of API error |
| JSON output (`--output=json`) | Returns `{id, title, startOn, endOn}` |

### Legacy epic mutations

| Test | Result |
|------|--------|
| Set dates on legacy epic (`recipe-book#5`) | Dates set correctly |
| Set dates on legacy epic (`task-tracker#7`) | Dates set correctly |
| Clear dates on legacy epic | Both dates cleared to None |
| JSON output for legacy epic | Returns `{id, startOn, endOn, issue: {title, number}}` |

### Identifier types

| Identifier type | Example | Result |
|----------------|---------|--------|
| Title substring | "Q1 Platform" | Resolved to "Q1 Platform Improvements" |
| Exact title | "Bug Bash Sprint" | Resolved correctly |
| ZenHub ID | `Z2lkOi8vcmFwdG9yL1plbmh1YkVwaWMvMjMyNDIy` | Resolved correctly |
| `repo#number` | `recipe-book#5` | Resolved to legacy epic |
| `owner/repo#number` | `dlakehammond/recipe-book#5` | Resolved to legacy epic |
| Alias | `q1` (after `epic alias "Q1 Platform" q1`) | Resolved correctly |

### Flags

| Flag | Result |
|------|--------|
| `--start=YYYY-MM-DD` | Sets start date |
| `--end=YYYY-MM-DD` | Sets end date |
| `--clear-start` | Clears start date (null) |
| `--clear-end` | Clears end date (null) |
| `--dry-run` | Prints "Would update dates..." without executing mutation |
| `--dry-run` with clear | Shows "(clear)" for cleared fields |
| `--output=json` | Structured JSON output |
| `--verbose` | Shows API request/response details on stderr |

### Integration verification

- After setting dates, `zh epic list` correctly displays date ranges (e.g. "Jan 1 → Mar 31, 2025")
- After clearing dates, `zh epic list` shows "-" for dates column
- `zh epic show` correctly displays dates in detail view

## Bugs found

None.

## Cleanup

All test dates were cleared after testing. Alias "q1" was deleted.
