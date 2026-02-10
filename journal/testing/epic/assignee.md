# Testing: `zh epic assignee`

## Commands Tested
- `zh epic assignee` (parent command, help only)
- `zh epic assignee add <epic> <user>...`
- `zh epic assignee remove <epic> <user>...`

## Test Environment
- Workspace: Dev Test (`69866ab95c14bf002977146b`)
- Epics used: "Q1 Platform Improvements" (ZenHub), "Bug Bash Sprint" (ZenHub), "Legacy Epic Test" (legacy)
- User: dlakehammond (display name "Hambend", ID `Z2lkOi8vcmFwdG9yL1plbmh1YlVzZXIvMjAxMzEyNw`)

## Test Results

### Help Output
| Test | Result |
|------|--------|
| `zh epic assignee --help` | PASS — shows add/remove subcommands |
| `zh epic assignee add --help` | PASS — documents flags and usage |
| `zh epic assignee remove --help` | PASS — documents flags and usage |

### Basic Operations
| Test | Result |
|------|--------|
| Add assignee by GitHub login | PASS — `dlakehammond` resolved and added |
| Remove assignee by GitHub login | PASS — assignee removed, verified via `epic show` |
| Add assignee with `@` prefix | PASS — `@dlakehammond` treated same as `dlakehammond` |
| Add assignee by display name | PASS — `Hambend` resolved to `@dlakehammond` |
| Add assignee by ZenHub user ID | PASS — full ZenHub ID resolved correctly |
| Add same user twice (idempotency) | PASS — no error, API handles gracefully |
| Remove user not currently assigned | PASS — no error, API handles gracefully |

### Epic Identifier Types
| Test | Result |
|------|--------|
| Full epic title | PASS — exact match works |
| Epic title substring | PASS — `"Platform"` resolved to "Q1 Platform Improvements" |
| ZenHub epic ID | PASS — `Z2lkOi8vcmFwdG9yL1plbmh1YkVwaWMvMjMyNDIy` works |
| Different epic ("Bug Bash Sprint") | PASS — works on any ZenHub epic |

### Case Sensitivity
| Test | Result |
|------|--------|
| Mixed-case GitHub login (`DlakeHammond`) | PASS — case-insensitive match |
| Lowercase display name (`hambend`) | PASS — case-insensitive match |

### Flags
| Test | Result |
|------|--------|
| `--dry-run` on add | PASS — shows "Would add" message, no mutation executed |
| `--dry-run` on remove | PASS — shows "Would remove" message, no mutation executed |
| `--output json` on add | PASS — structured JSON with epic, users, operation, failed fields |
| `--output json` on remove | PASS — structured JSON output |
| `--verbose` | PASS — API request/response logged to stderr |
| `--continue-on-error` with mixed valid/invalid users | PASS — valid user added, failure reported, exit code 1 |

### Error Handling
| Test | Result |
|------|--------|
| Missing all arguments | PASS — "requires at least 2 arg(s)", exit code 2 |
| Missing user argument | PASS — "requires at least 2 arg(s)", exit code 2 |
| Non-existent user | PASS — "user not found in workspace", exit code 4 |
| Non-existent epic | PASS — "epic not found", exit code 4 |
| Ambiguous epic substring | PASS — lists matching candidates, exit code 2 |
| Legacy epic | PASS — "managing assignees is only supported for ZenHub epics", exit code 2 |
| Stop on first error (default) | PASS — stops before processing remaining valid users |

## Bugs Found
None.

## Notes
- Only one user exists in the test workspace (`dlakehammond`), so multi-user add was only testable with one valid + one invalid user via `--continue-on-error`.
- The command correctly validates that assignee operations are only supported on ZenHub epics, not legacy GitHub-issue-backed epics.
- All exit codes are appropriate: 0 for success, 1 for partial failure with `--continue-on-error`, 2 for usage/validation errors, 4 for not-found.
