# 006: Workspace Commands

Phase 4 of the roadmap — workspace commands that establish the foundation for all other commands.

## What was done

- **`zh workspace list`**: Queries all workspaces across all ZenHub organizations. Supports `--favorites` and `--recent` filters (mutually exclusive). Highlights current workspace with `*`. JSON output mode supported. Caches workspace list for name resolution.

- **`zh workspace show [name]`**: Displays detailed workspace information — organization, ID, permission, visibility, dates, sprint configuration (cadence, schedule, timezone, active sprint with progress bar), and summary (repos, pipelines, priorities, default repo). Defaults to current workspace; accepts optional name/substring argument with case-insensitive resolution.

- **`zh workspace switch <name>`**: Resolves workspace by exact ID, exact name (case-insensitive), or unique substring. Updates config file. Clears workspace-scoped caches on switch. Handles "already current" case gracefully. Provides helpful error for ambiguous matches listing candidates.

- **`zh workspace repos`**: Lists all repositories connected to the current workspace with pagination support. Caches repo name-to-GitHub-ID mappings (critical for issue resolution in later phases).

## Shared infrastructure added

- `newClient()` / `apiNewFunc` — centralized API client creation with verbose logging, injectable for tests
- `requireConfig()` / `requireWorkspace()` — config loading with validation helpers
- Workspace resolution: exact ID → exact name → unique substring, with ambiguity detection and helpful error messages
- Workspace cache (`workspaces.json`) and repo cache (`repos-{workspace_id}.json`)

## Tests

13 tests covering:
- List: full list, JSON output, `--recent` filter, empty result
- Show: default workspace, no workspace configured, named workspace resolution
- Switch: successful switch (with cache clearing), already current, not found
- Repos: full list with cache verification, no workspace configured
- Help text verification

## Not done

- `zh workspace stats` — deferred to a later session
- GitHub API enrichment for `zh workspace repos` — requires GitHub integration layer (Phase 4 roadmap item left unchecked)
