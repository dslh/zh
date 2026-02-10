# 016: Issue close, reopen, and GitHub-enhanced show

## Scope
Phase 8 completion: `zh issue close`, `zh issue reopen`, and GitHub data enhancement for `zh issue show`.

## What was done

### `zh issue close`
- New file `cmd/issue_close.go` with `closeIssues` batch mutation
- Resolves issues via `resolveForClose` helper, checks state, skips already-closed
- Shared `resolvedCloseIssue` struct and helpers reused by reopen command
- Flags: `--dry-run`, `--repo`, `--continue-on-error`
- Output follows established mutation patterns (single, batch, partial failure)

### `zh issue reopen`
- New file `cmd/issue_reopen.go` with `reopenIssues` batch mutation
- Requires `--pipeline` flag (API constraint), optional `--position` (top/bottom only, no numeric)
- Reuses `resolvedCloseIssue` and `resolveForClose` from close command
- Flags: `--pipeline` (required), `--position`, `--dry-run`, `--repo`, `--continue-on-error`

### GitHub-enhanced `issue show`
- Added `issueGitHubData` struct with Author, Reactions, Reviews, CIStatus, IsMerged, IsDraft
- GitHub GraphQL query using `issueOrPullRequest` with fragments for Issue and PullRequest types
- New fields in detail view: Author, Reactions section, Reviews section, CI status
- Enhanced state display for PRs (merged/draft indicators)
- JSON output includes GitHub data when available
- Graceful fallback: `fetchGitHubIssueData` returns nil on any error, command still works

### Tests
- `cmd/issue_close_test.go`: 8 close tests + 10 reopen tests
- `cmd/issue_test.go`: 3 new tests for GitHub-enhanced show (issue, PR, JSON output)
- All 27+ new tests pass, all existing tests pass, lint clean

### Bug fix
- Fixed Makefile `run` target: added `GH_CONFIG_DIR=$(HOME)/.config/gh` so the `gh` CLI subprocess can find its auth tokens when `XDG_CONFIG_HOME` is overridden for zh's test config

## Design decisions
- Reopen reuses close's resolution infrastructure rather than duplicating it
- GitHub data is always best-effort; nil ghClient or any error silently falls back
- `issueOrPullRequest` GitHub query handles both Issues and PRs with inline fragments
- Review deduplication: keeps latest review per author (matching GitHub's PR review summary behavior)

## Verified against test workspace
- `issue show task-tracker#1` displays author from GitHub
- `issue show task-tracker#6` (PR) displays author
- `issue close task-tracker#4` closes successfully, "already closed" on re-run
- `issue reopen task-tracker#4 --pipeline=Todo` reopens successfully, "already open" on re-run
- Dry-run output correct for both close and reopen
- JSON output includes GitHub data
