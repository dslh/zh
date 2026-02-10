# Phase 12 (complete): Sprint report commands

Implemented the three remaining sprint report commands, completing Phase 12.

## `zh sprint velocity`
- Queries workspace average velocity with trend diff, sprint cadence config, active sprint, and recent closed sprints
- Displays header with cadence info and average velocity (with +/- trend indicator)
- Table shows each sprint's points done, points total, issue counts, and velocity
- Active sprint shown with `▶` marker and "(in progress)" instead of velocity
- Footer shows ZenHub's calculated average over last N sprints
- `--sprints=<n>` flag controls how many closed sprints to include (default 6)
- `--no-active` flag excludes the active sprint from output
- Handles edge cases: no sprint config, no closed sprints

## `zh sprint scope [sprint]`
- Fetches `scopeChange` connection on a sprint with full pagination
- Defaults to active sprint; supports all standard sprint identifiers
- Chronological event log table: date, action (+added/-removed), points, repo, issue number, title
- Summary section: initial scope (at/before sprint start), mid-sprint adds/removes, net change, current scope
- `--summary` flag shows only the summary without the event log
- Standard `--limit` and `--all` pagination flags for events
- Handles empty state (no scope changes recorded)

## `zh sprint review [sprint]`
- Fetches AI-generated sprint review with features, schedules, and late-closed issues
- Handles review states: null (no review), INITIAL (not generated), IN_PROGRESS (generating), COMPLETED
- Completed reviews show: state, generation timestamp, manually-edited indicator, initiator
- Progress section with points and issue counts
- Review body rendered as terminal markdown via Glamour (or raw with `--raw`)
- `--features` flag shows feature breakdown with deduplicated AI + manual issue groupings
- `--schedules` flag shows review meeting schedules with status
- `--late-closes` flag shows issues closed after review generation
- Hint text shown for available optional sections when flags aren't used

## Tests (12 new)
- Velocity: standard output, JSON, no sprints (not configured), --no-active
- Scope: standard output, --summary only, no changes, JSON
- Review: standard output, no review, JSON, --features
