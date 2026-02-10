# 018 — Issue Priority & Label Commands

Phase 10 (partial): `zh issue priority`, `zh issue label add`, `zh issue label remove`.

## What was done

- **Priority resolution** (`internal/resolve/priority.go`): Resolver with cache for workspace priorities. Matches by exact ID, exact name (case-insensitive), or unique substring. Reports ambiguous matches with helpful error messages.
- **`zh issue priority`** (`cmd/issue_priority.go`): Set or clear priority on one or more issues. Auto-detects set vs clear based on whether the last argument resolves as a priority. Flags: `--dry-run`, `--clear`, `--repo`, `--continue-on-error`. Uses `setIssueInfoPriorities`/`removeIssueInfoPriorities` mutations with `repositoryGhId` + `issueNumber`.
- **Label resolution** (`internal/resolve/label.go`): Resolver with cache for workspace labels. Fetches labels across all repos and deduplicates by name (case-insensitive). Batch `Labels()` function with invalidate-on-miss retry for not-found entries.
- **`zh issue label add/remove`** (`cmd/issue_label.go`): Add or remove labels from issues. Uses `--` separator to split issue args from label names. Resolves label names to IDs via the label resolver. Flags: `--dry-run`, `--repo`, `--continue-on-error`. Uses `addLabelsToIssues`/`removeLabelsFromIssues` mutations with `labelIds`.
- **Tests**: 10 priority tests, 11 label tests covering set/clear, batch, dry-run, JSON output, continue-on-error, and help.

## Design decisions

- **`--` separator for labels**: The label commands use `--` to separate issue identifiers from label names, rather than auto-detecting. This avoids ambiguity when label names could look like issue identifiers or vice versa. Cobra's `ArgsLenAtDash()` provides reliable detection.
- **`labelIds` over `labelInfos`**: The ZenHub API's `labelInfos` parameter (name-based) returns HTTP 500 errors. Switched to resolving label names to IDs locally and using the `labelIds` parameter, which works correctly.
- **Priority auto-detection**: The priority command tries the last argument as a priority name; if it resolves, it's a "set" operation, otherwise all args are issues and priority is cleared. The `--clear` flag provides explicit control.

## Remaining in Phase 10

- `zh issue activity` — no research file exists yet, deferred.
