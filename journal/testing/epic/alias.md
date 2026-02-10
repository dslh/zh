# Manual Testing: `zh epic alias`

## Summary

All operations of `zh epic alias` work correctly. No bugs found.

## Tests Performed

### Set Alias (`zh epic alias <epic> <alias>`)

| Test | Command | Result |
|------|---------|--------|
| Title substring | `zh epic alias "Bug Bash" bugbash` | Alias "bugbash" -> "Bug Bash Sprint" |
| Existing alias as epic ref | `zh epic alias bugbash bb2` | Alias "bb2" -> "Bug Bash Sprint" |
| repo#id format (legacy) | `zh epic alias recipe-book#5 rbi` | Alias "rbi" -> "Recipe Book Improvements" |
| owner/repo#id format | `zh epic alias dlakehammond/task-tracker#7 ltest` | Alias "ltest" -> "Legacy Epic Test" |
| ZenHub ID | `zh epic alias Z2lkOi8v...MjMyNDIz zid-test` | Alias "zid-test" -> "Bug Bash Sprint" |
| Duplicate (same epic) | `zh epic alias "Bug Bash" bugbash` | "Alias already points to..." |
| Duplicate (different epic) | `zh epic alias "Q1 Platform" bugbash` | Error: alias already exists, exit 2 |
| Non-existent epic | `zh epic alias "No Such Epic" myalias` | Error: epic not found, exit 4 |
| Ambiguous epic | `zh epic alias "Legacy" legtest` | Error: ambiguous, lists 3 candidates, exit 2 |

### List Aliases (`zh epic alias --list`)

| Test | Result |
|------|--------|
| List with aliases | Table with ALIAS and EPIC columns, footer count |
| List with no aliases | "No epic aliases configured." |
| JSON output | Valid JSON object mapping alias -> epic title |
| JSON with no aliases | `{}` |

### Delete Alias (`zh epic alias --delete <alias>`)

| Test | Result |
|------|--------|
| Delete existing alias | "Removed alias..." |
| Delete non-existent alias | Error: not found, exit 4 |

### Error Handling

| Test | Result |
|------|--------|
| No arguments | Error: usage message, exit 2 |
| One argument (no flags) | Error: usage message, exit 2 |
| Three arguments | Error: accepts 0-2 args, exit 2 |
| --delete with 2 args | Error: usage message, exit 2 |

### Other

| Test | Result |
|------|--------|
| Alias usable in `epic show` | `zh epic show bugbash` correctly displayed the aliased epic |
| --verbose flag | Shows API calls for epic resolution when cache is cold |
| --list + --delete combined | --list takes priority (acceptable) |
| Config persistence | Aliases correctly written to and removed from config.yml |

## Bugs Found

None.

## Notes

- Aliases map to epic **titles** (not IDs), making them human-readable in the config file.
- Aliases with spaces in the name (e.g., "q1 platform") work correctly.
- Aliases for deleted epics remain in config but fail with "not found" when used; `--delete` can remove them.
