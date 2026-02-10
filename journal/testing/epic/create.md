# zh epic create

## Summary

All features of `zh epic create` are working correctly. No bugs found.

## Test Environment

- Workspace: Dev Test (`69866ab95c14bf002977146b`)
- Repos: `dlakehammond/task-tracker`, `dlakehammond/recipe-book`

## Tests Performed

### Help text
- `zh epic create --help` — displays correct usage, flags (`--body`, `--repo`, `--dry-run`), and description.

### Standalone ZenHub epic creation
- `zh epic create "Test Epic from CLI"` — created successfully. Confirmed via `zh epic show` and `zh epic list`.
- `zh epic create "Epic with Body" --body "..."` — body saved and displayed in `zh epic show`.
- `zh epic create "JSON Output Epic" -o json` — returns well-formed JSON with `id`, `title`, `body`, `state`, `createdAt`.

### Legacy epic creation (GitHub issue-backed)
- `zh epic create "Legacy Epic Test" --repo task-tracker` — created successfully, shows issue reference (e.g. `task-tracker#7`).
- `zh epic create "Legacy Epic Recipe" --repo recipe-book --body "..."` — works with body and alternate repo.
- `zh epic create "Legacy JSON Epic" --repo task-tracker -o json` — JSON includes `id`, `issue.number`, `issue.title`, `issue.htmlUrl`, `issue.repository`.
- `zh epic create "Owner Repo Format" --repo dlakehammond/task-tracker --dry-run` — owner/repo format resolves correctly.

### Dry-run mode
- `zh epic create "Test" --dry-run` — shows "Would create epic" without making API calls.
- `zh epic create "Test" --body "..." --dry-run` — includes body in preview.
- `zh epic create "Test" --repo task-tracker --dry-run` — shows legacy type and resolved repo.
- `zh epic create "Test" --repo task-tracker --body "..." --dry-run` — shows both repo and body.

### Verbose mode
- `zh epic create "Verbose Create Test" -v` — logs both GraphQL requests (`GetWorkspaceOrg` and `CreateZenhubEpic`) with variables and responses to stderr.
- `zh epic create "Test" --dry-run -v` — no API output (expected, since dry-run skips API calls).

### Error handling
- `zh epic create` (no title) — exit code 2 (usage error), message: "accepts 1 arg(s), received 0".
- `zh epic create "Test" --repo nonexistent` — exit code 4 (not found), message: "repository 'nonexistent' not found in workspace".

### Cache invalidation
- After creating an epic, `zh epic list` and `zh epic show` immediately reflect the new epic, confirming the epic cache is properly invalidated on creation.

## Bugs Found

None.

## Cleanup

- Deleted all standalone ZenHub test epics via `zh epic delete`.
- Closed legacy epic GitHub issues (#7, #8 in task-tracker; #6 in recipe-book) via `gh issue close`.
