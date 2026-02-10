# 025 — Epic assignee and label commands

Phase 11 (partial): `zh epic assignee add/remove` and `zh epic label add/remove`.

## New resolvers

- **`internal/resolve/user.go`** — Resolves user identifiers (ZenHub ID, GitHub login, display name) to `UserResult` via the `workspace.zenhubUsers` API. Supports `@`-prefix stripping, case-insensitive matching, and invalidate-on-miss caching.
- **`internal/resolve/zenhub_label.go`** — Resolves ZenHub label identifiers (ID or name) to `ZenhubLabelResult` via the `workspace.zenhubLabels` API. Same caching pattern as the user resolver.
- Unit tests for both resolvers covering: ID match, name match, case-insensitive match, `@`-prefix, not found (with API refresh), batch resolve, and API refresh on cache miss.

## New commands

- **`zh epic assignee add <epic> <user>...`** — Adds assignees to an epic via `addAssigneesToZenhubEpics` mutation.
- **`zh epic assignee remove <epic> <user>...`** — Removes assignees via `removeAssigneesFromZenhubEpics` mutation.
- **`zh epic label add <epic> <label>...`** — Adds ZenHub labels via `addZenhubLabelsToZenhubEpics` mutation.
- **`zh epic label remove <epic> <label>...`** — Removes labels via `removeZenhubLabelsFromZenhubEpics` mutation.

All commands support `--dry-run`, `--json`, and `--continue-on-error` flags.

## Key decisions

- Epic labels use **ZenHub-scoped labels** (`ZenhubLabel` type), not GitHub-scoped labels. Created a separate resolver rather than reusing `resolve/label.go`.
- Shared `runEpicAssigneeOp()` and `runEpicLabelOp()` functions handle both add and remove via parameterized mutation name and query string.
- `--continue-on-error` resolves users/labels individually and reports failures without aborting.

## Tests

- 21 command tests in `cmd/epic_assignee_label_test.go`: add, remove, dry-run, JSON output, user/label not found, continue-on-error, legacy epic error, `@`-prefix handling.
- 17 resolver tests across `internal/resolve/user_test.go` and `internal/resolve/zenhub_label_test.go`.

## Files changed

- `cmd/epic_assignee_label.go` (new)
- `cmd/epic_assignee_label_test.go` (new)
- `internal/resolve/user.go` (new)
- `internal/resolve/user_test.go` (new)
- `internal/resolve/zenhub_label.go` (new)
- `internal/resolve/zenhub_label_test.go` (new)
- `ROADMAP.md` (checked off 16 items)
