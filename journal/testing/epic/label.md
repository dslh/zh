# Testing: `zh epic label`

## Commands Tested
- `zh epic label` (parent command, help only)
- `zh epic label add <epic> <label>...`
- `zh epic label remove <epic> <label>...`

## Test Environment
- Workspace: Dev Test (`69866ab95c14bf002977146b`)
- Epics used: "Q1 Platform Improvements" (ZenHub), "Bug Bash Sprint" (ZenHub), "Legacy Epic Test" (legacy)
- ZenHub labels available: bug, documentation, duplicate, enhancement, help wanted, wont fix, support, design, product, marketing, sales, tech debt, feature, frontend, backend (15 total)

## Test Results

### Help Output
| Test | Result |
|------|--------|
| `zh epic label --help` | PASS — shows add/remove subcommands |
| `zh epic label add --help` | PASS — documents flags and usage |
| `zh epic label remove --help` | PASS — documents flags and usage |

### Basic Add Operations
| Test | Result |
|------|--------|
| Add single label by exact name | PASS — `bug` added, confirmed via `epic show` |
| Add multiple labels at once | PASS — `enhancement feature` added in one call |
| Add label with spaces in name | PASS — `"help wanted"` and `"wont fix"` resolved correctly |
| Add already-present label (idempotency) | PASS — no error, API handles gracefully |
| Add label to different epic ("Bug Bash Sprint") | PASS — works on any ZenHub epic |

### Basic Remove Operations
| Test | Result |
|------|--------|
| Remove single label | PASS — `bug` removed, confirmed via `epic show` |
| Remove multiple labels | PASS — `enhancement feature` removed in one call |
| Remove 4 labels at once | PASS — `tech debt`, `frontend`, `backend`, `design` all removed |
| Remove label not on epic (idempotency) | PASS — no error, API handles gracefully |

### Epic Identifier Types
| Identifier type | Input | Result |
|-----------------|-------|--------|
| Full title | `"Q1 Platform Improvements"` | PASS |
| Title substring | `"Platform"` | PASS — resolved to "Q1 Platform Improvements" |
| ZenHub epic ID | `Z2lkOi8vcmFwdG9yL1plbmh1YkVwaWMvMjMyNDIy` | PASS |
| Epic alias | `"q1 platform"` | PASS — resolved via config alias |

### Label Identifier Types
| Identifier type | Input | Result |
|-----------------|-------|--------|
| Exact name | `bug` | PASS |
| Case-insensitive name | `BUG` | PASS — resolved to `bug` |
| ZenHub label ID | `Z2lkOi8vcmFwdG9yL1plbmh1YkxhYmVsLzU0Nzk2ODI` | PASS — resolved to `design` |
| Name with spaces | `"tech debt"` | PASS |

### Flags
| Test | Result |
|------|--------|
| `--dry-run` on add (single) | PASS — shows "Would add" message, no mutation executed |
| `--dry-run` on add (multiple) | PASS — lists all labels that would be added |
| `--dry-run` on remove | PASS — shows "Would remove" message, no mutation executed |
| `--output json` on add | PASS — structured JSON with epic, labels, operation, failed fields |
| `--output json` on remove | PASS — structured JSON output with operation "remove" |
| `--verbose` | PASS — API request/response logged to stderr (shows mutation and variables) |
| `--continue-on-error` on add (mixed valid/invalid) | PASS — valid labels added, failure reported, exit code 1 |
| `--continue-on-error` on remove (mixed valid/invalid) | PASS — valid label removed, failure reported, exit code 1 |
| `--continue-on-error` with all invalid | PASS — "all labels failed to resolve", exit code 1 |
| `--dry-run` + `--continue-on-error` | PASS — shows would-add for valid, "Failed to resolve" for invalid |

### Error Handling
| Test | Result |
|------|--------|
| Missing all arguments | PASS — "requires at least 2 arg(s)", exit code 2 |
| Missing label argument | PASS — "requires at least 2 arg(s)", exit code 2 |
| Non-existent label | PASS — "label(s) not found: nonexistent-label", exit code 4 |
| Non-existent epic | PASS — "epic not found", exit code 4 |
| Ambiguous epic substring | PASS — lists matching candidates, exit code 2 |
| Legacy epic | PASS — "managing labels is only supported for ZenHub epics", exit code 2 |
| Stop on first error (default) | PASS — stops before processing remaining valid labels |

## Bugs Found

### Partial failure message was poorly formatted
**Location:** `cmd/epic_assignee_label.go:550`

**Before:** The partial failure header read:
```
Added label(s) bug, feature to 2 of 3 epic label(s).
```
This was confusing — it listed all label names in the header and used "epic label(s)" which was grammatically awkward.

**After:** Fixed to match the assignee command's format:
```
Added 2 of 3 label(s) to epic "Q1 Platform Improvements".
```
This is consistent with the assignee partial failure format and reads naturally.

**Fix:** Changed format string from `"%s label(s) %s %s %d of %d epic label(s)."` to `"%s %d of %d label(s) %s epic %q."`.

## Notes
- Labels are organization-scoped "ZenHub labels", distinct from GitHub's repository-scoped labels. The command correctly uses the ZenHub label resolution path.
- Label resolution is case-insensitive by exact match only (no substring matching), which is appropriate since label names are typically short and specific.
- The command correctly blocks operations on legacy epics with a clear error message.
- All exit codes are appropriate: 0 for success, 1 for partial failure with `--continue-on-error`, 2 for usage/validation errors, 4 for not-found.
- Tests and linter pass after the fix.
