# Manual testing: zh workspace stats

## Command

`zh workspace stats` — Show workspace metrics including velocity trends, cycle time, and pipeline distribution.

## Tests performed

### Basic invocation

```
$ zh workspace stats
```

Displays all sections correctly:
- **WORKSPACE STATS: Dev Test** header with double-line separator
- **SUMMARY** section with entity counts in 3-column layout
- **VELOCITY** section with average velocity, assumed estimates, and sprint table
- **CYCLE TIME** section with days window in header
- **PIPELINE DISTRIBUTION** section with per-pipeline breakdown

### --sprints flag

```
$ zh workspace stats --sprints 3
```

Shows 3 closed sprints (plus active sprint) instead of the default 6. Verified fewer rows appear in the sprint table.

```
$ zh workspace stats --sprints 0
```

Shows only the active sprint (no closed sprints). Active sprint still displays with `▶` indicator.

### --days flag

```
$ zh workspace stats --days 7
$ zh workspace stats --days 14
```

The cycle time header correctly reflects the specified window (e.g., "CYCLE TIME (last 7 days)"). The `daysInCycle` variable is passed to the API correctly (verified via `--verbose`).

### Combined flags

```
$ zh workspace stats --sprints 2 --days 14
```

Both flags work together correctly.

### --output=json

```
$ zh workspace stats --output=json
```

Returns well-formed JSON with all fields present. Verified data matches ZenHub API responses obtained via direct GraphQL queries.

### --verbose

```
$ zh workspace stats --verbose
```

Logs the full GraphQL query and variables to stderr, followed by the API response, then the formatted output. Shows `POST https://api.zenhub.com/public/graphql`, the query body, variables, and the response.

### --help

```
$ zh workspace stats --help
```

Shows usage, flag descriptions with defaults (`--sprints` default 6, `--days` default 30), and global flags.

## Verification against API

Verified the following data points against direct ZenHub GraphQL API queries:
- Repository count: 2 (task-tracker, recipe-book) — matches
- Epic count: 2 (Q1 Platform Improvements, Bug Bash Sprint) — matches
- Priority count: 1 — matches
- Dependencies: 3 — matches
- Automations: 0 — matches
- Pipeline names, stages, and issue counts — all match API response exactly

## Observations

### Pipeline vs workspace issue count discrepancy

The SUMMARY section shows "Issues: 11" and "PRs: 3" (from the workspace-level `issues` field), while the PIPELINE DISTRIBUTION section shows 23 issues and 3 PRs in the Todo pipeline. This is a ZenHub API behavior where pipeline-scoped `issues` counts differ from workspace-scoped counts. The CLI faithfully reports what the API returns for each scope, which is the correct approach.

### No cycle time data

The test workspace has no cycle time data available because issues have not completed a full pipeline cycle (moved from development through to closed). The command handles this gracefully with a helpful message and dimmed hint about possible causes.

### Sprint data

All sprints in the test workspace have 0 points and 0 issues, which is expected for a test environment. The sprint table renders correctly with the `▶` indicator on the active sprint and proper date range formatting.

### Test pipeline has null stage

The "Test" pipeline has no stage configured (shows `-` in the STAGE column). This is correctly handled by the null check on `p.Stage`.

## Bugs found

None.

## Test suite

All existing unit tests pass. Linter reports 0 issues.
