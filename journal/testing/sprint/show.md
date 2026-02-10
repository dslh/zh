# Manual Testing: `zh sprint show`

## Summary

All features of `zh sprint show` are working correctly. No bugs found.

## Test Environment

- Workspace: Dev Test (`69866ab95c14bf002977146b`)
- Repos: `dlakehammond/task-tracker`, `dlakehammond/recipe-book`
- Active sprint: Sprint: Feb 8 - Feb 22, 2026

## Tests Performed

### Sprint Identifier Types

| Identifier | Command | Result |
|---|---|---|
| Default (no arg) | `zh sprint show` | Shows active sprint |
| `current` keyword | `zh sprint show current` | Shows active sprint |
| `next` keyword | `zh sprint show next` | Shows next upcoming sprint |
| `previous` keyword | `zh sprint show previous` | Shows most recent closed sprint |
| Exact name | `zh sprint show "Sprint: Feb 8 - Feb 22, 2026"` | Resolves correctly |
| Unique substring | `zh sprint show "Feb 8"` | Resolves correctly |
| ZenHub ID | `zh sprint show Z2lkOi8vcmFwdG9yL1NwcmludC80NjMzMDg0` | Resolves correctly |
| Ambiguous substring | `zh sprint show "Feb"` | Exit 2, lists 3 matching candidates |
| Nonexistent name | `zh sprint show "nonexistent sprint"` | Exit 4, suggests `zh sprint list` |

### Flags

| Flag | Command | Result |
|---|---|---|
| `--limit` | `zh sprint show --limit 2` | Shows 2 of 5 issues with "Showing 2 of 5 issue(s)" footer |
| `--all` | `zh sprint show --all` | Shows all 5 issues |
| `--output json` | `zh sprint show --output json` | Full JSON output with sprint metadata and all issues |
| `--verbose` | `zh sprint show --verbose` | Logs GraphQL query and response to stderr |
| `--help` | `zh sprint show --help` | Displays usage, flags, and identifier documentation |

### Display Verification

- **Empty sprint**: Shows "No estimates" and "No issues in sprint" for progress section
- **Sprint with issues**: Shows issues table with ISSUE, STATE, TITLE, EST, PIPELINE, ASSIGNEE columns
- **Mixed pipelines**: Issues in different pipelines display correct pipeline names
- **Estimates**: Issues with and without estimates display correctly (value or `-`)
- **Closed issues**: Closed state displayed correctly in STATE column
- **Progress bars**: Points and Issues progress bars render correctly with filled/empty blocks
- **Closed sprint**: State shows as "closed", no issues section when empty
- **Date formatting**: Date ranges display correctly (e.g., "Feb 8 → 22, 2026 (14 days)")
- **Issue references**: Short form `repo#number` used consistently across both repos

### Edge Cases

| Scenario | Result |
|---|---|
| Too many arguments | Exit 2: "accepts at most 1 arg(s), received 2" |
| `--limit 0` | Treated as unlimited, shows all issues |

### Exit Codes

| Scenario | Exit Code |
|---|---|
| Success | 0 |
| Ambiguous sprint name | 2 |
| Sprint not found | 4 |
| Too many arguments | 2 |

## Bugs Found

None.
