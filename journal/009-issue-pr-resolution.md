# 009 — Issue/PR Resolution

## Summary

Implemented the issue/PR identifier resolution layer in the `resolve` package, completing the Phase 5 "Issue/PR resolution" roadmap items.

## Changes

- **`internal/resolve/repo.go`** — New file. Defines `CachedRepo` type and repo-related resolution functions:
  - `FetchRepos()` — fetches repos from API with pagination, caches results
  - `FetchReposIntoCache()` — stores pre-fetched repo data (used by workspace commands)
  - `LookupRepo()` — finds a repo by "repo" or "owner/repo" format, with ambiguity detection
  - `RepoNamesAmbiguous()` — checks if long-form references are needed
  - `RepoCacheKey()` — consistent cache key generation

- **`internal/resolve/issue.go`** — New file. Core issue/PR resolution:
  - `ParseIssueRef()` — parses identifiers into structured components: ZenHub ID, owner/repo#number, repo#number, bare number
  - `Issue()` — main resolver entry point supporting all identifier formats:
    - ZenHub ID → `node()` query
    - `owner/repo#number` or `repo#number` → repo cache lookup + `issueByInfo` query
    - Bare number with `--repo` flag → resolves repo context first
    - Branch name with `--repo` + GitHub access → GitHub PR lookup by `headRefName`
  - `IssueResult.Ref()` / `FullRef()` — convenience methods for formatting references
  - Repo cache uses invalidate-on-miss: cache miss triggers API refresh before returning not-found

- **`cmd/workspace.go`** — Migrated from local `cachedRepo` type to `resolve.CachedRepo`. All repo caching now routes through `resolve.FetchReposIntoCache()` for consistency.

## Tests

27 new tests in `internal/resolve/issue_test.go` covering:
- Parsing: owner/repo#number, repo#number, bare number, ZenHub ID, invalid input, edge cases (zero, negative)
- Repo lookup: by owner/name, by name, case-insensitive, not found, ambiguous, disambiguated by owner
- Repo name ambiguity detection
- Full integration: resolve by repo#number, owner/repo#number, ZenHub ID, bare number with --repo, branch name
- Error paths: bare number without --repo, repo not found, issue not found, ambiguous repo
- Cache invalidate-on-miss: repo refresh when not in cache
- Branch resolution: PR found, no PR found

## Verification

- API queries (`issueByInfo`, `node`) verified against real ZenHub API via MCP
- `workspace repos` and `pipeline list` commands still work correctly after caching migration
- All tests pass, linter clean
