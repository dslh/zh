# Manual testing: zh workspace show

## Summary

`zh workspace show` displays detailed information about a workspace including organization, permissions, sprint configuration, and a summary of repositories, pipelines, and priorities. All tests passed with no bugs found.

## Test results

### Default workspace (no arguments)
```
$ zh workspace show
```
Correctly displays the current default workspace ("Dev Test") with all sections:
- Header with workspace name
- Metadata fields (Organization, ID, Permission, Visibility, Created, Last updated)
- Sprint configuration (Cadence, Schedule, Timezone, Active sprint)
- Summary (Repositories, Pipelines, Priorities, Default repo)

### Named workspace (exact name)
```
$ zh workspace show 'Dev Test'
```
Correctly resolves by exact name and displays the same output.

### Substring match
```
$ zh workspace show Dev
$ zh workspace show test
```
Both correctly resolve to "Dev Test" via case-insensitive substring matching.

### Workspace ID
```
$ zh workspace show 69866ab95c14bf002977146b
```
Correctly resolves by workspace ID.

### JSON output
```
$ zh workspace show --output=json
```
Outputs well-formed JSON with all workspace fields. Verified against direct ZenHub API response — all fields match exactly.

### Verbose mode
```
$ zh workspace show --verbose
```
Logs the full GraphQL query, variables, and response to stderr before displaying formatted output.

### Error cases
- **Nonexistent workspace:** `zh workspace show Nonexistent` → "workspace \"Nonexistent\" not found" with exit code 4
- **Extra arguments:** `zh workspace show 'Dev Test' extra-arg` → "accepts at most 1 arg(s)" with exit code 2
- **No API key:** Exit code 3 with helpful message about configuring ZH_API_KEY
- **No default workspace set:** Exit code 2 with message about using `zh workspace switch`

### NO_COLOR / piped output
Output renders cleanly without ANSI escape codes when piped or when `NO_COLOR=1` is set.

### Help text
```
$ zh workspace show --help
```
Displays usage, description, and available flags (--interactive, --output, --verbose).

## Data verification

Compared `zh workspace show --output=json` against a direct ZenHub GraphQL API query. All fields match:
- Organization, ID, permissions, visibility
- Sprint config (cadence, period, start/end day, timezone)
- Active sprint (name, state, dates, points)
- Repositories (2: task-tracker, recipe-book)
- Pipelines (3: Todo, Doing, Test)
- Priorities (1: High priority)
- Default repository (dlakehammond/task-tracker)

## Bugs found

None.

## Unit tests

All 4 existing unit tests pass:
- `TestWorkspaceShowDefault` — Shows default workspace without arguments
- `TestWorkspaceShowNoWorkspace` — Errors when no default workspace configured
- `TestWorkspaceShowNamed` — Shows a workspace by name argument
- `TestWorkspaceShowInteractiveNonTTY` — Errors when --interactive used in non-TTY
