# Manual testing: zh sprint scope

## Summary

Tested `zh sprint scope` command with all supported identifier types, flags, and output modes. Found and fixed one bug related to `--limit` flag affecting summary computation.

## Test environment

- Active sprint: "Sprint: Feb 8 - Feb 22, 2026" (82 scope change events from prior testing)
- Previous sprint: "Sprint: Jan 22 - Feb 5, 2026" (0 scope change events)
- Added task-tracker#1 and task-tracker#2 to active sprint to generate scope change events, then removed them after testing.

## Tests performed

### Sprint identifiers

| Identifier | Command | Result |
|---|---|---|
| Default (current) | `zh sprint scope` | Showed active sprint scope changes |
| `current` keyword | `zh sprint scope current` | Same as default |
| `previous` keyword | `zh sprint scope previous` | Showed previous sprint (0 events) |
| `next` keyword | `zh sprint scope next` | Showed next sprint (0 events) |
| Name substring | `zh sprint scope "Feb 8"` | Matched active sprint correctly |
| Ambiguous substring | `zh sprint scope "Feb"` | Error: ambiguous, listed 3 candidates (exit 2) |
| ZenHub ID | `zh sprint scope Z2lkOi8v...` | Resolved correctly |
| Nonexistent | `zh sprint scope "nonexistent"` | Error: not found (exit 4) |

### Flags

| Flag | Command | Result |
|---|---|---|
| `--summary` | `zh sprint scope --summary` | Showed only header + summary, no event log |
| `--limit 5` | `zh sprint scope --limit 5` | Displayed 5 events, footer "Showing 5 of 82 event(s)" |
| `--all` | `zh sprint scope --all` | Showed all 82 events |
| `--output json` | `zh sprint scope --output json` | Correct JSON with sprint metadata, totalEvents, and events array |
| `--output json --limit 5` | `zh sprint scope --output json --limit 5` | JSON with totalEvents:82, 5 events returned |
| `--help` | `zh sprint scope --help` | Correct help text with examples |

### Output verification

- Header shows sprint name, dates, points progress bar (when applicable), and scope change summary
- Event log table columns: DATE, ACTION, PTS, REPO, #, TITLE
- Actions colored: green for `+ added`, red for `- removed`
- Missing estimates shown as `-`
- Summary section shows initial scope, mid-sprint adds/removes, net change, current scope
- Empty sprint shows "No scope changes recorded for this sprint."

## Bug found and fixed

### --limit flag corrupts summary computation

**Symptom:** When `--limit` is set (e.g., `--limit 5`), the SUMMARY section and the header "Changes" line were computed from only the limited events rather than all events.

With `--limit 5`, the output showed:
```
Changes:  82 events (2 added, 3 removed)   <-- mixed total count with limited counts
...
SUMMARY
Initial scope (at sprint start):  0 issues, 0 pts
Added mid-sprint:                 2 issues, 1 pts     <-- wrong, should be 42
Removed mid-sprint:               3 issues, 0 pts     <-- wrong, should be 40
Current scope:                    -1 issues, 1 pts     <-- wrong, should be 2
```

**Root cause:** `fetchScopeChanges()` was called with the display limit, so only 5 events were fetched. Both `formatScopeChangeSummaryLine()` and `computeScopeSummary()` then operated on these 5 events instead of all 82.

**Fix:** In `runSprintScope()`, for human-readable output, always fetch all events (limit=0) and compute the summary from the full set. The `--limit` flag now only limits which events are displayed in the event log table. JSON output continues to respect the limit directly (since it includes `totalEvents` for the caller to see the full count).

**File changed:** `cmd/sprint_reports.go`
