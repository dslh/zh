# 013: Issue list and show commands

Phase 8 (partial): core issue read commands.

## What was done

- **`zh issue list`**: lists issues across all workspace pipelines with parallel API calls
  - Filters: `--pipeline`, `--sprint`, `--epic`, `--assignee`, `--no-assignee`, `--label`, `--repo`, `--estimate`, `--no-estimate`, `--type`, `--state`
  - `--limit` (default 100) and `--all` flags
  - Three query strategies: by pipeline (default), by epic (`searchIssuesByZenhubEpics`), and closed issues (`searchClosedIssues`)
  - Client-side pipeline filtering for epic queries
  - Tabular output with ISSUE, TITLE, EST, PIPELINE, ASSIGNEE, LABELS columns
  - JSON output mode

- **`zh issue show <issue>`**: displays full issue/PR detail view
  - Resolves identifiers via `resolve.Issue` (repo#number, owner/repo#number, ZenHub ID, bare number with --repo)
  - Two query paths: `issueByInfo` for repo+number, `node` query for ZenHub IDs
  - Detail view with: state, pipeline, estimate, priority, assignees, labels, sprint, epic, milestone
  - Sections: DESCRIPTION (rendered markdown), CONNECTED PRS, BLOCKING, BLOCKED BY, LINKS, TIMELINE
  - JSON output mode

- **Resolve package changes**:
  - Exported `LookupRepoWithRefresh` (was lowercase) for use by issue list's `--repo` filter
  - Added `GetCachedPipelines` helper for reading pipeline cache without API call

- **Tests**: 12 new tests covering:
  - List: default, JSON, empty, no workspace, pipeline filter, closed state
  - Show: default, JSON, with blockers, with connected PRs, not found, help text

## Not in scope (remaining Phase 8 items)

- `--interactive` mode for issue show (deferred to Phase 15)
- GitHub enrichment for issue show (author, reactions, PR review/CI status)
- `zh issue move`, `estimate`, `close`, `reopen` (mutation commands)
