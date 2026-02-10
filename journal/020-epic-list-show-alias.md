# 020: Epic list, show, and alias commands

Phase 11 (partial): Implements the read-only epic commands and alias management.

## Changes

### `zh epic list` (`cmd/epic.go`)
- Queries workspace roadmap for both ZenHub epics and legacy (issue-backed) epics
- Displays type, state, title, issue count progress, estimate, and date range
- Legacy epics show repo#number reference alongside their title
- Pagination support with `--limit` and `--all` flags (default 100)
- Caches epic entries for resolver (invalidate-on-miss pattern)
- JSON output mode via `--output=json`

### `zh epic show <epic>` (`cmd/epic.go`)
- Resolves epic by ID, title, substring, repo#number (legacy), or alias
- Separate queries for ZenHub epics vs legacy epics (different GraphQL types)
- ZenHub epic detail: title, state, estimate, dates, creator, assignees, labels, progress bars, child issues, blocking/blocked items, description (rendered via Glamour)
- Legacy epic detail: title, state, estimate, dates, assignees, labels, progress bars, child issues, description, GitHub link
- Child issues shown in tabular format with issue ref, state, title, estimate, pipeline
- JSON output mode via `--output=json`

### `zh epic alias <epic> <alias>` (`cmd/epic_mutations.go`)
- Set, list, and delete epic aliases (stored in config file)
- Validates epic exists before creating alias
- `--list` shows all configured epic aliases
- `--delete` removes an existing alias
- Follows same pattern as `zh pipeline alias`

### Tests (`cmd/epic_test.go`)
- Epic list: tabular output, JSON output, empty workspace
- Epic show: ZenHub epic detail, legacy epic detail, JSON output
- Epic alias: set, list, delete (not found)
- Help text verification

## Notes
- The roadmap query only returns epics that have been explicitly added to the workspace roadmap. Newly created epics may not appear until added to the roadmap. This matches ZenHub's UI behavior.
- `--interactive` mode for epic show is deferred to Phase 15 (Bubble Tea integration).
- Legacy epics don't have blocking/blocked items at the ZenHub API level — this data is only available on the linked GitHub issue.
