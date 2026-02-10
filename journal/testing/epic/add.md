# Manual Testing: `zh epic add`

## Summary

Tested the `zh epic add <epic> <issue>...` command for adding issues to both ZenHub epics and legacy epics. Found and fixed a critical bug: the REST v1 API (used for legacy epic operations) requires a separate authentication token from the GraphQL API, but the code was using the same token for both.

## Bug Found

**REST v1 API authentication failure for legacy epics**

The ZenHub REST v1 API uses `X-Authentication-Token` with a different token than the GraphQL API's `Authorization: Bearer` token. The `zh_`-prefixed GraphQL API keys are not accepted by the REST v1 endpoint, causing a 401 Unauthorized error when attempting to add/remove issues from legacy epics.

### Fix

- Added `rest_api_key` config field and `ZH_REST_API_KEY` environment variable support
- Added `api.WithRESTAPIKey()` option to the API client
- REST `UpdateEpicIssues` now uses the separate REST API key and provides a clear error with setup instructions when unconfigured
- Updated tests to pass the REST API key where needed
- Added a test for the missing-key error case

### Files Changed

- `internal/api/client.go` — Added `restAPIKey` field and `WithRESTAPIKey` option
- `internal/api/rest.go` — Use separate REST key; check and error if missing
- `internal/api/rest_test.go` — Updated tests to pass REST key; added missing-key test
- `internal/config/config.go` — Added `RESTAPIKey` field, env var binding, persistence
- `cmd/workspace.go` — Pass REST API key from config to client
- `cmd/epic_mutations_test.go` — Set `ZH_REST_API_KEY` env var in test setup

## Tests Performed

### ZenHub Epic (standalone)

| # | Test | Result |
|---|------|--------|
| 1 | Single issue, `repo#number` format | Pass |
| 2 | Multiple issues, `repo#number` format | Pass |
| 3 | `owner/repo#number` format | Pass |
| 4 | `--repo` flag with bare numbers | Pass |
| 5 | `--dry-run` flag | Pass (no mutation executed) |
| 6 | `--output=json` flag | Pass (valid JSON output) |
| 7 | Epic substring identifier | Pass |
| 8 | Epic ZenHub ID identifier | Pass |
| 9 | Issues from different repos | Pass |
| 10 | ZenHub ID for issue identifier | Pass |
| 11 | `--repo=owner/repo` format | Pass |
| 12 | Epic alias identifier | Pass |
| 13 | Adding duplicate issue (already in epic) | Pass (idempotent) |
| 14 | Adding a PR (not just issues) | Pass |

### Legacy Epic

| # | Test | Result |
|---|------|--------|
| 15 | `--dry-run` on legacy epic | Pass |
| 16 | Actual add without REST API key | Pass (clear error with instructions) |

### Error Cases

| # | Test | Result |
|---|------|--------|
| 17 | No arguments | Pass (exit code 2, usage error) |
| 18 | Only epic, no issues | Pass (exit code 2) |
| 19 | Non-existent epic | Pass (exit code 4, helpful message) |
| 20 | Non-existent issue | Pass (exit code 1, error message) |
| 21 | Non-existent repo | Pass (exit code 4, helpful message) |
| 22 | `--continue-on-error` with mix of valid/invalid | Pass (partial success) |

### Other

| # | Test | Result |
|---|------|--------|
| 23 | `--help` flag | Pass (complete documentation) |

## Notes

- The `addIssuesToZenhubEpics` GraphQL mutation only works with standalone ZenHub epics. Legacy epics require the REST v1 API (`/p1/repositories/{repoId}/epics/{epicNumber}/update_issues`).
- The REST v1 API is deprecated. The ZenHub GraphQL API token (`zh_` prefix) is not accepted by the REST endpoint; a separate REST API token must be generated from the ZenHub dashboard.
- Adding an issue that is already a child of the epic is idempotent — the API accepts it silently.
- PRs can be added to epics just like issues.
