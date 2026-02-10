# Phase 13: Utility commands — label list and priority list

## Commands implemented

### `zh label list`
- Lists all labels aggregated across all repos in the current workspace
- Labels are deduplicated by name (case-insensitive), matching existing `resolve.FetchLabels` behavior
- Sorted alphabetically by name for consistent output
- Table columns: LABEL, COLOR (hex with `#` prefix)
- JSON output mode supported
- Populates label cache for use by resolution

### `zh priority list`
- Lists all priorities configured for the current workspace
- Preserves API ordering (typically: Urgent, High, Medium, Low)
- Table columns: PRIORITY, COLOR (hex with `#` prefix)
- JSON output mode supported
- Populates priority cache for use by resolution

## Implementation notes
- Both commands reuse existing `resolve.FetchLabels` and `resolve.FetchPriorities` functions, which handle API fetching and caching
- No new GraphQL queries needed — the resolve layer already had the queries defined
- Follows the same pattern as `zh pipeline list`: fetch, cache, render table or JSON

## Tests
- `TestLabelList`: verifies headers, label names, colors, alphabetical sort, footer, cache population
- `TestLabelListJSON`: verifies valid JSON array output
- `TestLabelListEmpty`: verifies "No labels found" message
- `TestLabelListDeduplicates`: verifies labels with same name across repos are deduplicated
- `TestLabelListNoWorkspace`: verifies error when no workspace configured
- `TestPriorityList`: verifies headers, priority names, colors, footer, cache population
- `TestPriorityListJSON`: verifies valid JSON array output
- `TestPriorityListEmpty`: verifies "No priorities configured" message
- `TestPriorityListNoWorkspace`: verifies error when no workspace configured

## Files added
- `cmd/label.go` — label command group and list subcommand
- `cmd/label_test.go` — label list tests
- `cmd/priority.go` — priority command group and list subcommand
- `cmd/priority_test.go` — priority list tests
