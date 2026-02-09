# 008: Pipeline resolution and commands

## Scope

Phase 5 (pipeline resolution) and Phase 6 (`pipeline list`, `pipeline show`).

## Work done

### Pipeline resolution (`internal/resolve/pipeline.go`)
- Created the `resolve` package with pipeline identifier resolution
- Supports resolution by: ZenHub ID, exact name (case-insensitive), unique substring, and config alias
- Uses the cache with invalidate-on-miss: tries cached pipelines first, refreshes from API on miss, retries
- Ambiguous substring matches produce a helpful error listing all candidates with their IDs
- Not-found errors use exit code 4, ambiguity errors use exit code 2
- `FetchPipelines()` fetches pipelines from the API and populates the cache
- `FetchPipelinesIntoCache()` allows commands that already have pipeline data to populate the cache without an extra API call

### `zh pipeline list` (`cmd/pipeline.go`)
- Lists all pipelines in the workspace in board order
- Columns: position number, name, issue count, stage, default PR pipeline
- Stage enum values formatted for display (e.g. `SPRINT_BACKLOG` → `Sprint Backlog`)
- Caches pipeline list for use by resolution
- JSON output support
- Empty workspace handled

### `zh pipeline show <name>` (`cmd/pipeline.go`)
- Resolves pipeline by name/substring/alias/ID using the resolve package
- Fetches full pipeline details (description, stage, configuration, dates)
- Fetches issues in the pipeline using `searchIssuesByPipeline` with pagination
- `--limit` (default 100) and `--all` flags for controlling issue count
- Issue references use short form (`repo#number`) unless repo names are ambiguous across owners
- Issues displayed with estimate, assignees, and priority
- JSON output includes both pipeline details and issues
- Stale issue configuration shown when present

### Tests
- 9 resolution tests: ID, exact name, case-insensitive name, unique substring, alias, ambiguous, not-found with cache refresh, not-found after refresh, fetch
- 9 command tests: list (normal, JSON, empty, no workspace), show (normal, substring, not found, JSON), help text

## Verified manually
- `zh pipeline list` — shows 2 pipelines in Dev Test workspace
- `zh pipeline show Todo` — shows pipeline details and 7 issues
- `zh pipeline show Doing` — shows empty pipeline
- `zh pipeline show Do` — correctly reports ambiguity between Todo and Doing
- `zh pipeline list --output=json` — valid JSON output
