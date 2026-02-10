# Manual Testing: `zh pipeline list`

## Summary

All tests passed. No bugs found. The command correctly lists all pipelines in the workspace with their position, name, issue count, stage, and default PR status.

## Test Environment

- Workspace: "Dev Test" (`69866ab95c14bf002977146b`)
- Pipelines: Todo, Doing, Test

## Tests Performed

### Basic table output
```
$ zh pipeline list
#    PIPELINE    ISSUES    STAGE          DEFAULT PR
────────────────────────────────────────────────────────────────────────────────
1    Todo        24        Backlog        no
2    Doing       2         Development    no
3    Test        0         -              no

Total: 3 pipeline(s)
```
Output matches the ZenHub API response exactly. All 3 pipelines shown in correct position order. The `Test` pipeline (with null stage) correctly displays `-`.

### JSON output (`--output=json` and `--output json`)
Both forms produce valid JSON with all fields: `id`, `name`, `description`, `stage`, `isDefaultPRPipeline`, and `issues.totalCount`. Null values for `description` and `stage` are preserved in JSON output.

### Verbose mode (`--verbose`)
Correctly logs the GraphQL request and response to stderr, including the query, variables, and response body. The table output still renders to stdout.

### Piped output
When piped through `cat`, output contains no ANSI color escape codes — confirming proper TTY detection.

### NO_COLOR environment variable
Output with `NO_COLOR=1` is identical to piped output, with no color codes present.

### Invalid output format (`--output=yaml`)
Falls through to default table output without error. This is consistent with how the `--output` flag works across other commands.

### Help output (`--help`)
Displays usage, flags, and global flags correctly.

### Parent command (`zh pipeline`)
Displays the full list of pipeline subcommands as expected.

### Cache verification
After running `pipeline list`, the cache file `pipelines-69866ab95c14bf002977146b.json` was populated with all 3 pipelines (id and name), confirming the cache-on-list behavior works correctly.

### API verification
The ZenHub GraphQL API was queried directly to confirm the CLI output matches:
- Pipeline count: 3 (matches)
- Pipeline names and order: Todo, Doing, Test (matches)
- Issue counts: 24, 2, 0 (matches)
- Stages: BACKLOG, DEVELOPMENT, null (matches, with null displayed as `-`)
- Default PR pipeline: all false (matches, displayed as `no`)

## Bugs Found

None.
