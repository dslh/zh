# 021: Epic Create

Implemented `zh epic create <title>` command.

## What was done

- Added `zh epic create <title>` with `--body`, `--repo`, and `--dry-run` flags
- Standalone ZenHub epic creation via `createZenhubEpic` mutation (default)
- Legacy epic creation via `createEpic` mutation when `--repo` is specified
- Added `fetchWorkspaceOrgID` helper to retrieve the ZenHub organization ID required by the create mutation
- Cache invalidation after epic creation
- JSON output support
- Tests: create, create with body, dry-run, JSON output, legacy create, legacy dry-run, legacy JSON, help text

## Implementation notes

- The `createZenhubEpic` mutation requires the `zenhubOrganizationId`, which is obtained via a lightweight workspace query. This is a separate API call per creation; could be cached in a future optimization.
- Legacy epics (via `--repo`) use the `createEpic` mutation which creates a GitHub issue and promotes it to epic status in a single API call.
- The research doc suggested many additional flags (`--start`, `--end`, `--state`, `--assignee`, `--label`, `--estimate`, `--issue`), each requiring a follow-up mutation after creation. These are deferred — the roadmap only specifies `--body` and `--repo` for this phase.

## Issues discovered

- `epic list` and epic resolution both use the workspace roadmap query. Newly created epics are not automatically added to the roadmap, so they won't appear in `zh epic list` or be resolvable by title until added to the roadmap in ZenHub. Added a roadmap item to investigate using a separate epic-specific query.
