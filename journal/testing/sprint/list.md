# Manual testing: zh sprint list

## Summary

`zh sprint list` works correctly. No bugs found. All flags and output formats function as expected.

## Test environment

- Workspace: Dev Test (`69866ab95c14bf002977146b`)
- 20 sprints total: 8 OPEN, 12 CLOSED
- Active sprint: "Sprint: Feb 8 - Feb 22, 2026"
- Upcoming sprint: "Sprint: Feb 22 - Mar 8, 2026"
- Previous sprint: "Sprint: Jan 22 - Feb 5, 2026"

## Tests performed

### Basic output (`zh sprint list`)

- Displays all 20 sprints in a tabular format with columns: STATE, NAME, DATES, POINTS, CLOSED
- Active sprint correctly marked with `▶ active` indicator
- Open sprints show as "open", closed sprints show as "closed"
- Date ranges formatted correctly (e.g., "May 17 → 31, 2026", "Dec 26, 2025 → Jan 9, 2026")
- Points column shows `-` when no estimates (all sprints in test workspace have 0 points)
- Footer shows "Showing 20 sprint(s)"

### `--state=open`

- Shows only the 8 OPEN sprints
- Footer: "Showing 8 sprint(s)"
- Active sprint still correctly marked

### `--state=closed`

- Shows only the 12 CLOSED sprints
- Footer: "Showing 12 sprint(s)"
- No active marker shown (correct, since active sprint is OPEN)

### `--state=all`

- Shows all 20 sprints (same as default in this case since total < 100)

### `--state=invalid`

- Correctly errors: `Error: invalid --state value "invalid": must be open, closed, or all`
- Exit code: 2 (usage error)

### `--limit=3`

- Shows only 3 sprints (the 3 most recent by start date)
- Footer: "Showing 3 of 20 sprint(s)"

### `--state=open --limit=2`

- Correctly combines filters: shows 2 open sprints
- Footer: "Showing 2 of 8 sprint(s)"

### `--all`

- Shows all 20 sprints
- Footer: "Showing 20 sprint(s)"

### `--output=json`

- Valid JSON array of sprint objects
- Each object contains: id, name, generatedName, state, startAt, endAt, totalPoints, completedPoints, closedIssuesCount, createdAt, updatedAt
- Data matches ZenHub API responses exactly

### `--output=json --limit=2`

- Returns a JSON array with exactly 2 sprint objects

### `--verbose`

- Logs full GraphQL query, variables, and API response to stderr
- Main table output still appears on stdout

### `--help`

- Displays usage information with all available flags
- Includes state filter documentation

### Extra arguments

- `zh sprint list extra-arg` correctly errors: "unknown command"
- Exit code: 2

## Data verification

Compared output against direct ZenHub GraphQL API queries. All sprint IDs, names, states, dates, and counts match the API data exactly.

## Bugs found

None.
