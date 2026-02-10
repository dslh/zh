# Manual Testing: `zh epic remove`

## Summary

Tested the `zh epic remove` command for removing issues from ZenHub epics. Found and fixed one bug: dry-run mode did not respect `--output=json` across all four dry-run code paths.

## Test Environment

- Workspace: Dev Test (`69866ab95c14bf002977146b`)
- Repos: `dlakehammond/task-tracker`, `dlakehammond/recipe-book`
- ZenHub epic: "Q1 Platform Improvements"
- Legacy epic: "Recipe Book Improvements" (recipe-book#5)

## Tests Performed

### Help text
- `zh epic remove --help` — Displays correct usage, flags, and examples.

### Dry-run with various issue identifier formats
- `zh epic remove "Q1 Platform" task-tracker#1 --dry-run` — Single issue, repo#number format. Works.
- `zh epic remove "Q1 Platform" task-tracker#1 task-tracker#2 --dry-run` — Multiple issues. Works.
- `zh epic remove "Q1 Platform" dlakehammond/task-tracker#1 --dry-run` — owner/repo#number format. Works.
- `zh epic remove "Q1 Platform" --repo=task-tracker 1 2 --dry-run` — `--repo` with bare numbers. Works.
- `zh epic remove "Q1 Platform" --all --dry-run` — `--all` flag. Works; lists all 4 child issues.

### Actual removal operations
- `zh epic remove "Q1 Platform" task-tracker#3` — Single issue removal. Confirmed via `epic show`.
- `zh epic remove "Q1 Platform" task-tracker#1 task-tracker#2` — Multiple issue removal. Confirmed.
- `zh epic remove "Q1 Platform" --repo=task-tracker 1 2` — `--repo` flag with bare numbers. Works.
- `zh epic remove "Q1 Platform" dlakehammond/task-tracker#1` — owner/repo#number format. Works.
- `zh epic remove "Q1 Platform" --all` — Remove all issues. Confirmed epic shows no child issues afterward.

### JSON output
- `zh epic remove "Q1 Platform" recipe-book#1 --output=json` — Valid JSON with `epic` and `removed` fields.
- `zh epic remove "Q1 Platform" --all --output=json` — Valid JSON with all removed issues listed.

### Epic identifier types
- Full title: `"Q1 Platform Improvements"` — Works.
- Substring: `"Q1 Platform"`, `"Platform"` — Works.
- Alias: `"q1 platform"` (configured in config.yml) — Works.
- ZenHub ID: `Z2lkOi8vcmFwdG9yL1plbmh1YkVwaWMvMjMyNDIy` — Works.

### Error handling
- `zh epic remove "Q1 Platform"` (no issues, no --all) — Exit code 2, clear error message.
- `zh epic remove "Nonexistent Epic" task-tracker#1` — Exit code 4, "not found" message.
- `zh epic remove "Q1 Platform" nonexistent-repo#999` — Exit code 4, repo not found.
- `zh epic remove "Q1 Platform" --all` on empty epic — Graceful "no child issues" message.

### --continue-on-error
- `zh epic remove "Q1 Platform" task-tracker#1 nonexistent-repo#999 --continue-on-error` — Successfully removes the valid issue, reports the failed one separately. Exit code 1.

### Legacy epic
- `zh epic remove "Recipe Book Improvements" recipe-book#2` — Correctly reports that a REST API token is required. Clear error message with instructions on how to configure one.

### Verbose output
- `zh epic remove "Q1 Platform" task-tracker#1 --verbose` — Shows full API request/response cycle (3 GraphQL calls: IssueByInfo, GetIssueForEpic, RemoveIssuesFromZenhubEpics).

## Bug Found and Fixed

### Dry-run `--output=json` not producing JSON

**Symptom:** Running `zh epic remove ... --dry-run --output=json` produced plain text output instead of JSON.

**Root cause:** The dry-run code paths in `renderEpicRemoveDryRun`, `renderEpicRemoveLegacyDryRun`, `runEpicRemoveAll` (dry-run block), and `runEpicRemoveAllLegacy` (dry-run block) did not check `output.IsJSON(outputFormat)` before rendering. The dry-run early return bypassed the JSON output block that appears later in the non-dry-run flow.

**Fix:** Added `output.IsJSON(outputFormat)` checks at the top of each dry-run code path, producing JSON output with `dryRun: true` along with the epic and removed issue details. This matches the pattern used by `epic delete --dry-run --output=json`.

**Files changed:**
- `cmd/epic_mutations.go` — Added JSON output in all four dry-run paths
- `cmd/epic_mutations_test.go` — Added four new tests: `TestEpicRemoveDryRunJSON`, `TestEpicRemoveAllDryRunJSON`, `TestEpicRemoveLegacyDryRunJSON`, `TestEpicRemoveAllLegacyDryRunJSON`

## Notes

- Legacy epic add/remove requires a separate REST API token (`rest_api_key` / `ZH_REST_API_KEY`), which is distinct from the GraphQL API key. No REST API key was available in the test credentials, so legacy epic remove could only be tested for the error message path. The unit tests cover the legacy removal logic via mock server.
- The same dry-run JSON bug likely exists in `epic add` (`renderEpicAddDryRun`), which has identical code structure but was not fixed since it's outside the scope of this test.
