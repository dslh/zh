# Manual testing: zh pipeline alias

## Summary

Tested `zh pipeline alias` command for creating, listing, deleting, and using pipeline aliases. All functionality works correctly with no bugs found.

## Test environment

- Workspace: Dev Test (`69866ab95c14bf002977146b`)
- Pipelines: Todo, Doing, Test

## Tests performed

### Help output
- `zh pipeline alias --help` — displays correct usage, flags (`--delete`, `--list`), and description.

### Creating aliases

| Test | Command | Result |
|------|---------|--------|
| Exact pipeline name | `zh pipeline alias Doing dev` | `Alias "dev" -> "Doing".` |
| Substring match | `zh pipeline alias Tes t` | `Alias "t" -> "Test".` |
| Pipeline ZenHub ID | `zh pipeline alias Z2lkOi8v...MzcyMjk doing2` | `Alias "doing2" -> "Doing".` |
| Existing alias as reference | `zh pipeline alias todo td` | `Alias "td" -> "Todo".` |

### Listing aliases

| Test | Command | Result |
|------|---------|--------|
| Table format | `zh pipeline alias --list` | Shows ALIAS/PIPELINE table with footer |
| JSON format | `zh pipeline alias --list --output json` | `{"dev": "Doing", "t": "Test", ...}` |
| Empty list (table) | `zh pipeline alias --list` (after deleting all) | `No pipeline aliases configured.` |
| Empty list (JSON) | `zh pipeline alias --list --output json` | `{}` |

### Deleting aliases

| Test | Command | Result |
|------|---------|--------|
| Delete existing | `zh pipeline alias --delete doing2` | `Removed alias "doing2".` (exit 0) |
| Delete non-existent | `zh pipeline alias --delete nonexistent` | `Error: alias "nonexistent" not found` (exit 4) |

### Alias resolution in other commands

| Test | Command | Result |
|------|---------|--------|
| pipeline show | `zh pipeline show dev` | Correctly shows "Doing" pipeline details |
| pipeline show | `zh pipeline show t` | Correctly shows "Test" pipeline details |
| issue move (dry-run) | `zh issue move task-tracker#4 dev --dry-run` | Resolves "dev" to "Doing" |
| board --pipeline | `zh board --pipeline dev` | Shows Doing pipeline board |

### Error handling

| Test | Command | Exit code | Error message |
|------|---------|-----------|---------------|
| No arguments | `zh pipeline alias` | 2 | `usage: zh pipeline alias <pipeline> <alias>` |
| Too many arguments | `zh pipeline alias one two three` | 2 | `accepts between 0 and 2 arg(s), received 3` |
| Non-existent pipeline | `zh pipeline alias NonExistent ne` | 4 | `pipeline "NonExistent" not found` |
| Duplicate alias (different pipeline) | `zh pipeline alias Test dev` | 2 | `alias "dev" already exists (points to "Doing")` |
| Duplicate alias (same pipeline) | `zh pipeline alias Doing dev` | 0 | `Alias "dev" already points to "Doing".` (idempotent) |
| Ambiguous substring | `zh pipeline alias o amb` | 2 | Lists matching pipelines (Todo, Doing) |
| Delete with no args | `zh pipeline alias --delete` | 2 | Usage error |
| Delete with too many args | `zh pipeline alias --delete a b` | 2 | Usage error |

### Config persistence
- Verified that aliases are correctly written to and removed from `config.yml`
- Verified that the config file structure is preserved after alias operations

## Bugs found

None.

## Test suite
- All unit tests pass (`make test`)
- Linter clean (`make lint`)
