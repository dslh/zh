# Manual Testing: `zh epic delete`

## Summary

Tested the `zh epic delete` command with various identifier types, flags, and edge cases. Found and fixed one bug: `--dry-run --output=json` was not emitting JSON output.

## Test Cases

### Help flag
- `zh epic delete --help` — displays usage, flags (`--dry-run`, `--help`), and global flags (`--output`, `--verbose`).

### Dry-run with title substring
- `zh epic delete "Deletable" --dry-run` — correctly resolves the epic by substring, displays "Would delete" message with ID, state, and child issue count.

### Dry-run with ZenHub ID
- `zh epic delete Z2lkOi8vcmFwdG9yL1plbmh1YkVwaWMvMjMyNDI4 --dry-run` — correctly resolves by ZenHub ID and displays the same dry-run output.

### Dry-run with alias
- `zh epic alias "Deletable" del-test` then `zh epic delete del-test --dry-run` — alias resolution works correctly.

### Dry-run with child issues
- Added `task-tracker#1` to the epic, then ran dry-run. Output correctly shows "Child issues: 1 (will be removed from epic, not deleted)".

### Ambiguous substring
- `zh epic delete "Epic" --dry-run` — correctly reports ambiguity, lists 4 matching epics with their IDs, and exits with code 2.

### Nonexistent epic
- `zh epic delete "nonexistent-epic-xyz"` — correctly reports "not found" and exits with code 4.

### Legacy epic rejection
- `zh epic delete "Legacy Epic Test" --dry-run` — correctly rejects with "legacy epic (backed by GitHub issue dlakehammond/task-tracker#7) — delete it via GitHub instead" and exits with code 2.

### Missing argument
- `zh epic delete` — correctly rejects with "accepts 1 arg(s), received 0" (exit code 1; see note below).

### Too many arguments
- `zh epic delete "one" "two"` — correctly rejects with "accepts 1 arg(s), received 2" (exit code 2).

### Actual deletion with JSON output
- `zh epic delete del-test --output=json` — successfully deletes the epic and outputs structured JSON with `deleted`, `id`, and `childIssues` fields. Verified the epic was removed from `zh epic list` and the child issue (`task-tracker#1`) still exists with "Epic: None".

### Actual deletion with human-readable output (no children)
- `zh epic delete "Delete Me Plain"` — outputs "Deleted epic \"Delete Me Plain\"." with no child issue count line (correctly omitted when count is 0).

### Actual deletion with human-readable output (with children)
- Created "Delete With Children", added `task-tracker#2` and `recipe-book#1`, then deleted. Output: "Deleted epic \"Delete With Children\"." followed by "2 child issue(s) removed from epic."

### Verbose mode
- `zh epic delete "Verbose Delete" --verbose` — correctly logs all API requests (ListZenhubEpics, ListLegacyEpics, GetEpicChildCount, DeleteZenhubEpic) to stderr with request/response details.

### Dry-run with JSON output (fixed)
- `zh epic delete "DryRunJsonTest" --dry-run --output=json` — after fix, correctly emits structured JSON with `dryRun: true`, `deleted`, `id`, `state`, and `childIssues` fields.

## Bug Found and Fixed

### `--dry-run --output=json` ignores JSON format

**Symptom:** Running `zh epic delete <epic> --dry-run --output=json` outputs the same human-readable dry-run format instead of JSON.

**Root cause:** The dry-run code path in `runEpicDelete` did not check `output.IsJSON(outputFormat)` before rendering. It always called `output.MutationDryRunDetail()` which only produces human-readable text.

**Fix:** Added a JSON check at the top of the dry-run block in `cmd/epic_mutations.go`. When `--output=json` is specified, the dry-run now emits a JSON object with `dryRun: true` plus the same fields as the actual deletion response (`deleted`, `id`, `state`, `childIssues`).

**Test added:** `TestEpicDeleteDryRunJSON` in `cmd/epic_mutations_test.go` validates that dry-run with `--output=json` produces valid JSON with the expected fields.

## Notes

- **Systemic: dry-run + JSON across all commands.** This same issue (dry-run ignoring `--output=json`) exists in all other mutation commands. Only the `epic delete` instance was fixed in this pass.
- **Systemic: Cobra arg validation exit codes.** Commands using `cobra.ExactArgs(N)` return exit code 1 instead of 2 for usage errors. This is because Cobra returns a plain `error`, not an `exitcode.Error`, so `ExitCode()` falls through to `GeneralError` (1). This affects all commands with Cobra arg validation.
