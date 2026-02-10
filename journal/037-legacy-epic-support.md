# 037: Legacy epic support via GitHub API

Phase 17 (partial): Legacy epic edit and set-state operations.

## Changes

- **Extended `EpicResult`** to carry legacy epic issue reference fields (`IssueNumber`, `RepoName`, `RepoOwner`) so commands can identify the backing GitHub issue
- **Extracted `epicResultFromCache` helper** in `resolve/epic.go` to consistently populate legacy fields during epic resolution
- **Implemented `zh epic edit` for legacy epics** via GitHub GraphQL API (`updateIssue` mutation) — edits the title/body of the backing GitHub issue
- **Implemented `zh epic set-state` for legacy epics** via GitHub GraphQL API — maps ZenHub states to GitHub open/closed (with a note when `todo`/`in_progress` map to `open`)
- **Added graceful error messages** when GitHub access is not configured but required for legacy epic operations
- **Improved all legacy epic error messages** across `edit`, `delete`, `set-state`, `add`, `remove`, `estimate`, `assignee`, `label`, and `key-date` commands to include the backing GitHub issue reference (e.g. `owner/repo#number`)
- **Legacy epic `add`/`remove`** remain unsupported — the ZenHub GraphQL API has no mutation for managing child issues on legacy epics; error messages now direct users to the ZenHub web UI

## New helpers

- `requireLegacyEpicGitHubID()` — fetches the GitHub node ID for a legacy epic's backing issue
- `legacyEpicRef()` — formats the `owner/repo#number` reference string
- `runEpicEditLegacy()` — edits title/body via GitHub API
- `runEpicSetStateLegacy()` — changes open/closed state via GitHub API

## Tests added

- `TestEpicEditLegacy` — success path via GitHub API
- `TestEpicEditLegacyDryRun` — dry-run output for legacy edit
- `TestEpicEditLegacyJSON` — JSON output for legacy edit
- `TestEpicEditLegacyNoGitHub` — error when GitHub not configured
- `TestEpicSetStateLegacy` — success path via GitHub API
- `TestEpicSetStateLegacyDryRun` — dry-run output for legacy state change
- `TestEpicSetStateLegacyJSON` — JSON output for legacy state change
- `TestEpicSetStateLegacyStateMapping` — verifies `in_progress` maps to `open` with explanation
- `TestEpicSetStateLegacyNoGitHub` — error when GitHub not configured

## Not implemented (API limitation)

- `zh epic add` / `zh epic remove` for legacy epics — the ZenHub GraphQL API does not expose mutations for adding/removing child issues from legacy epics. Would require ZenHub REST API v1 support.
