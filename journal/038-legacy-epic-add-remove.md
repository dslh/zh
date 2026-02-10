# 038: Legacy epic add/remove via ZenHub REST API v1

Phase 17 (complete): Add and remove child issues on legacy epics.

## Changes

- **Added ZenHub REST API v1 client** (`internal/api/rest.go`) with `UpdateEpicIssues()` method that calls `POST /p1/repositories/{repo_id}/epics/{issue_number}/update_issues`
- **Implemented `zh epic add` for legacy epics** — resolves the epic's backing repo GhID, builds REST-format issue references (`repo_id` + `issue_number`), and calls the REST API
- **Implemented `zh epic remove` for legacy epics** — same pattern, sends `remove_issues` instead of `add_issues`
- **Implemented `zh epic remove --all` for legacy epics** — fetches child issues via GraphQL, resolves repo GhIDs from cache, then removes all via REST API
- **Extended `resolvedEpicIssue`** to carry `RepoGhID` from the initial issue resolution, needed for constructing REST API requests
- **Extended `MockServer`** with `HandleREST(pathSubstring, statusCode, responseBody)` for path-based REST mock routing alongside existing GraphQL handlers
- **Updated help text** for `epic add` and `epic remove` to indicate they work with both ZenHub and legacy epics
- **Dry-run, JSON, and partial-failure output** all work correctly for legacy epics, with messages indicating the epic is legacy

## New functions

- `api.Client.UpdateEpicIssues()` — REST API v1 call for legacy epic child issue management
- `api.Client.RESTEndpoint()` — derives REST base URL from the GraphQL endpoint
- `runEpicAddLegacy()` — legacy epic add via REST API
- `renderEpicAddLegacyDryRun()` — dry-run output for legacy add
- `runEpicRemoveLegacy()` — legacy epic remove via REST API
- `renderEpicRemoveLegacyDryRun()` — dry-run output for legacy remove
- `runEpicRemoveAllLegacy()` — legacy epic remove-all via REST API
- `testutil.MockServer.HandleREST()` — REST path-based mock handler

## Tests added

- `TestEpicAddLegacy` — success path via REST API
- `TestEpicAddLegacyDryRun` — dry-run output
- `TestEpicAddLegacyJSON` — JSON output with legacy epic ref
- `TestEpicRemoveLegacy` — success path via REST API
- `TestEpicRemoveLegacyDryRun` — dry-run output
- `TestEpicRemoveLegacyJSON` — JSON output with legacy epic ref
- `TestEpicRemoveAllLegacy` — remove all child issues from legacy epic
- `TestEpicRemoveAllLegacyDryRun` — dry-run for remove all
- `TestEpicRemoveAllLegacyEmpty` — empty legacy epic reports no child issues
- `TestUpdateEpicIssues` — REST API client unit test (verifies path, auth, body)
- `TestUpdateEpicIssuesAuthFailure` — REST API 401 handling
- `TestRESTEndpoint` — endpoint derivation logic
