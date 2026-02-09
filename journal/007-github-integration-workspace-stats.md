# 007 — GitHub integration & workspace stats

Completed the remaining Phase 4 items: GitHub-enriched `workspace repos` and `workspace stats`.

## What was done

### `internal/gh` package
- Built a GitHub GraphQL API client supporting both `gh` CLI and PAT authentication methods
- `gh.New(method, token, opts...)` returns nil for method "none", a functioning client otherwise
- `gh` CLI execution passes the full JSON request body via `--input -` to avoid shell escaping issues with `$` in GraphQL variables
- PAT execution uses direct HTTP requests to `api.github.com/graphql`
- Supports `WithEndpoint` for test injection and `WithVerbose` for debug logging

### `zh workspace repos --github`
- Added `--github` flag to enrich repo output with data from GitHub's API
- When enabled, fetches description, primary language, and star count per repo
- Enriched table shows: REPO, DESCRIPTION, LANGUAGE, STARS, PRIVATE
- Graceful degradation: if GitHub access isn't configured, prints a warning and falls back to the standard table
- JSON output includes a `github` object per repo when enriched
- Long descriptions truncated to 40 chars in table view

### `zh workspace stats`
- New command showing workspace metrics in four sections:
  - **Summary**: repo/epic/automation/issue/PR/dependency/estimate/priority/pipeline counts
  - **Velocity**: average velocity with trend, sprint completion table with active sprint indicator
  - **Cycle time**: average cycle days, development days, review days breakdown
  - **Pipeline distribution**: per-pipeline issue/PR/estimate counts including closed pipeline
- Flags: `--sprints` (default 6) and `--days` (default 30) for configuring velocity and cycle time windows
- Handles empty states: no sprints configured, no velocity data, no cycle time data
- JSON output via `--output=json`

### Testing
- 6 new tests added (19 total workspace tests):
  - `TestWorkspaceReposWithGitHub` — enriched output with mocked GitHub API
  - `TestWorkspaceReposGitHubNotConfigured` — graceful fallback with warning
  - `TestWorkspaceStats` — full stats output with all sections
  - `TestWorkspaceStatsJSON` — JSON output mode
  - `TestWorkspaceStatsNoSprints` — empty workspace handling
  - `TestWorkspaceStatsNoWorkspace` — missing workspace config error
- All tests pass, lint clean

### Manual verification
- `workspace repos` and `workspace repos --github` verified against Dev Test workspace
- `workspace stats` verified against Dev Test workspace (shows real pipeline distribution, sprint history, handles missing cycle time data)
- `workspace stats --output=json` produces valid JSON
- Help text for all new commands verified

## Files changed
- `internal/gh/gh.go` — new GitHub API client package
- `cmd/workspace.go` — added `--github` flag for repos, `workspace stats` command with query/types/rendering
- `cmd/workspace_test.go` — 6 new tests
