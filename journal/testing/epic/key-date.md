# Testing: `zh epic key-date`

## Summary

Tested all three subcommands (`list`, `add`, `remove`) of `zh epic key-date` against the Dev Test workspace. All commands function correctly with no bugs found.

## Commands Tested

### `zh epic key-date list <epic>`

| Test | Result |
|------|--------|
| List key dates on epic with no key dates | Pass - shows "No key dates on epic" message |
| List key dates after adding entries | Pass - table format with DATE and NAME columns |
| `--output=json` with key dates present | Pass - correct JSON with epic metadata and keyDates array |
| `--output=json` with no key dates | Pass - returns empty `keyDates: []` array |
| `--verbose` flag | Pass - shows GraphQL query, variables, and response |
| ZenHub ID as epic identifier | Pass |
| Exact title as epic identifier | Pass |
| Title substring as epic identifier | Pass |
| GitHub `repo#number` for legacy epic | Pass - returns exit code 2 with clear error message |
| Missing epic argument | Pass - exit code 2 with usage error |

### `zh epic key-date add <epic> <name> <date>`

| Test | Result |
|------|--------|
| Add key date with valid args | Pass - confirmation message shown |
| `--dry-run` flag | Pass - shows "Would add" prefix, no mutation executed |
| `--output=json` | Pass - returns JSON with epic, keyDate, and operation fields |
| Invalid date format | Pass - exit code 2, clear error about YYYY-MM-DD format |
| Legacy epic | Pass - exit code 2, explains key dates only for ZenHub epics |
| Nonexistent epic | Pass - exit code 4, suggests `zh epic list` |
| Missing arguments | Pass - exit code 2 |

### `zh epic key-date remove <epic> <name>`

| Test | Result |
|------|--------|
| Remove existing key date | Pass - confirmation message shown |
| `--dry-run` flag | Pass - shows "Would remove" prefix, no mutation |
| `--output=json` | Pass - returns JSON with operation "remove" |
| Case-insensitive name matching | Pass - "code freeze" matches "Code Freeze" |
| Nonexistent key date name | Pass - exit code 4 |
| Legacy epic | Pass - exit code 2 |
| Missing arguments | Pass - exit code 2 |

### Help text

| Test | Result |
|------|--------|
| `zh epic key-date --help` | Pass - shows subcommand overview |
| `zh epic key-date list --help` | Pass - shows usage and flags |
| `zh epic key-date add --help` | Pass - shows usage, flags, and examples |
| `zh epic key-date remove --help` | Pass - shows usage, flags, and examples |

## Identifier Types Tested

- Exact epic title: `"Q1 Platform Improvements"`
- Title substring: `"Q1 Platform"`, `"Bug Bash"`
- ZenHub ID: `Z2lkOi8vcmFwdG9yL1plbmh1YkVwaWMvMjMyNDIy`
- GitHub issue ref for legacy epics: `task-tracker#7`, `recipe-book#5`

## Bugs Found

None.

## Test Suite

All unit tests pass. Linter reports 0 issues.
