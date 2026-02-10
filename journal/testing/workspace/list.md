# Manual Testing: `zh workspace list`

## Summary

Tested the `zh workspace list` command and its flags (`--recent`, `--favorites`, `--output=json`, `--verbose`). Found and fixed one bug related to JSON output of empty lists.

## Test Environment

- Workspace: Dev Test (`69866ab95c14bf002977146b`)
- Organization: `hambend@gmail.com`
- Repos: 2 (task-tracker, recipe-book)
- Pipelines: 3

## Tests Performed

### Basic list (default)
```
$ zh workspace list
ORGANIZATION         WORKSPACE     REPOS    PIPELINES    PERMISSION
────────────────────────────────────────────────────────────────────────────────
hambend@gmail.com    Dev Test *    2        3            admin

Total: 1 workspace(s)
```
- Organization name displayed correctly
- `*` indicator marks current workspace
- Repo and pipeline counts are accurate (verified against API)
- Permission shown in lowercase
- Footer shows total count

### JSON output
```
$ zh workspace list --output=json
```
- Returns valid JSON array with one workspace object
- All fields present: id, name, displayName, description, isFavorite, viewerPermission, repositoriesConnection, pipelinesConnection, zenhubOrganization
- Data matches ZenHub API response

### Recent workspaces
```
$ zh workspace list --recent
```
- Returns the same workspace (only one in the account)
- Output format matches default list
- `*` indicator present for current workspace

### Favorite workspaces
```
$ zh workspace list --favorites
No favorite workspaces.
```
- Correctly reports empty when no favorites exist
- JSON output returns `[]` (after fix, was `null`)

### Mutual exclusivity of --favorites and --recent
```
$ zh workspace list --favorites --recent
Error: if any flags in the group [favorites recent] are set none of the others can be; [favorites recent] were all set
```
- Correct exit code 2 (usage error)
- Clear error message

### Verbose mode
```
$ zh workspace list --verbose
```
- Logs the GraphQL query to stderr
- Logs the API response
- Regular output still goes to stdout

### Help text
```
$ zh workspace list --help
```
- Describes the command and all flags
- Shows global flags (--output, --verbose)

### JSON with --recent
```
$ zh workspace list --recent --output=json
```
- Returns valid JSON array

### JSON with --favorites (empty)
```
$ zh workspace list --favorites --output=json
[]
```
- Returns empty JSON array (after fix)

## Bug Found and Fixed

### Empty JSON output returns `null` instead of `[]`

**Problem:** When `--favorites --output=json` was used and there were no favorites, the command output `null` instead of `[]`. This is because Go's `json.Marshal` encodes a nil slice as `null`, not as `[]`.

**Root cause:** In `runWorkspaceListFavorites`, the `workspaces` variable was declared as `var workspaces []workspaceNode` (nil slice). When no favorites existed, the loop never appended to it, so it remained nil. The same pattern existed in `fetchAllWorkspaces`.

**Fix:** Changed both locations to use `make([]workspaceNode, 0)` which initializes an empty (non-nil) slice that serializes to `[]` in JSON.

**Files changed:** `cmd/workspace.go` (lines 484 and 770)

## All Tests Pass

- `go test ./...` - all pass
- `golangci-lint run ./...` - 0 issues
