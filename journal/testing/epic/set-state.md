# `zh epic set-state` — Manual Testing

## Summary

All tests passed. No bugs were found. The command correctly handles ZenHub epics, legacy epics, all valid state values, state aliases, dry-run mode, JSON output, verbose mode, --apply-to-issues, and various epic identifier types.

## Test Environment

- Workspace: Dev Test (`69866ab95c14bf002977146b`)
- ZenHub epics tested: "Q1 Platform Improvements", "Bug Bash Sprint"
- Legacy epic tested: "Recipe Book Improvements" (`recipe-book#5`)
- gh CLI switched to `dlakehammond` for legacy epic operations

## Tests Performed

### Basic state changes on ZenHub epics

| Command | Result |
|---------|--------|
| `zh epic set-state 'Q1 Platform' todo` | Set state to todo |
| `zh epic set-state 'Q1 Platform' in_progress` | Set state to in_progress |
| `zh epic set-state 'Q1 Platform' in-progress` | Set state to in_progress (hyphen variant) |
| `zh epic set-state 'Q1 Platform' inprogress` | Set state to in_progress (no separator variant) |
| `zh epic set-state 'Q1 Platform' closed` | Set state to closed |
| `zh epic set-state 'Q1 Platform' open` | Set state to open |
| `zh epic set-state 'Bug Bash' CLOSED` | Uppercase accepted (case insensitive) |
| `zh epic set-state 'Bug Bash' Open` | Mixed case accepted |

### Epic identifier types

| Identifier | Result |
|------------|--------|
| Title substring (`'Q1 Platform'`) | Resolved to "Q1 Platform Improvements" |
| ZenHub ID (`Z2lkOi8v...`) | Resolved correctly |
| Alias (`bb`) | Resolved correctly after `zh epic alias` |
| `repo#number` (`recipe-book#5`) | Resolved legacy epic correctly |
| `owner/repo#number` (`dlakehammond/recipe-book#5`) | Resolved legacy epic correctly |

### Flags

| Flag | Result |
|------|--------|
| `--dry-run` | Shows "Would set state..." without executing |
| `--dry-run` on legacy epic | Shows legacy-specific message with GitHub note |
| `--output=json` | Returns structured JSON with id, title, state |
| `--output=json` on legacy epic | Returns JSON with id, issue ref, title, state |
| `--apply-to-issues` | Applies state change to child issues, shows confirmation |
| `--apply-to-issues --dry-run` | Shows note about child issues without executing |
| `--apply-to-issues` on legacy epic | Shows warning that flag is not supported |
| `--verbose` | Shows full GraphQL request/response cycle |

### Legacy epic state mapping

| Requested State | Mapped GitHub State | Note Shown |
|----------------|---------------------|------------|
| `closed` | CLOSED | No |
| `open` | OPEN | No |
| `todo` | OPEN | Yes — "GitHub issues only support open/closed" |
| `in_progress` | OPEN | Yes — "GitHub issues only support open/closed" |

### Error cases

| Scenario | Result |
|----------|--------|
| Invalid state (`invalid`) | Exit code 2, message lists valid states |
| Missing state argument | Exit code 2, "accepts 2 arg(s), received 1" |
| No arguments | Exit code 2, "accepts 2 arg(s), received 0" |
| Non-existent epic | Exit code 4, "not found" message |
| Legacy epic without GitHub access | Error with guidance to configure GitHub access |

## Bugs Found

None.
