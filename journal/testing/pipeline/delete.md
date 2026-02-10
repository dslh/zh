# Manual Testing: `zh pipeline delete`

## Summary

All tests passed. No bugs found. The command correctly deletes a pipeline and moves its issues to the specified destination pipeline.

## Test Environment

- Workspace: "Dev Test" (`69866ab95c14bf002977146b`)
- Existing pipelines: Todo, Doing, Test
- Temporary pipelines created for testing: DeleteMe, DeleteMe2, DeleteMe3

## Tests Performed

### Help
- `zh pipeline delete --help` — displays usage, flags, and description correctly.

### Dry Run
- `zh pipeline delete DeleteMe --into=Todo --dry-run` — shows pipeline ID, issue count (0), and destination. No mutation executed.
- After moving `task-tracker#1` into DeleteMe, dry-run correctly showed 2 issues to move.
- Dry-run with `--output=json` still produces human-readable output (consistent with other commands).

### Actual Deletion
- `zh pipeline delete DeleteMe --into=Todo` — deleted pipeline, printed "Deleted pipeline \"DeleteMe\"." and "Moved 2 issue(s) to \"Todo\"."
- Verified via `zh pipeline list` that DeleteMe was removed and Todo gained the issues.

### JSON Output
- `zh pipeline delete DeleteMe2 --into=Todo --output=json` — produced valid JSON: `{"deleted":"DeleteMe2","destination":"Todo","issuesMoved":0}`.

### Verbose Mode
- `zh pipeline delete DeleteMe3 --into=Todo --verbose` — logged all three API calls (ListPipelines for resolution, GetPipelineDetails for issue count, DeletePipeline mutation) with request/response details.
- When 0 issues moved, the "Moved N issue(s)" line is correctly suppressed.

### Error Cases
- Missing `--into` flag: exit code 2, "required flag(s) \"into\" not set".
- Missing pipeline argument: exit code 2, "accepts 1 arg(s), received 0".
- Delete into itself (`--into=DeleteMe`): exit code 2, "cannot delete pipeline into itself".
- Non-existent pipeline: exit code 4, "pipeline \"NonExistent\" not found".
- Non-existent destination: exit code 4, "pipeline \"NonExistent\" not found".

### Identifier Resolution
- Exact name match: `DeleteMe` resolved correctly.
- Substring match: `Delete` and `Tod` both resolved correctly.
- ZenHub ID: `Z2lkOi8vcmFwdG9yL1BpcGVsaW5lLzM1MzgzNzI` resolved correctly.
- Alias: set `todo-alias` -> `Todo`, then `--into=todo-alias` resolved correctly.
- Ambiguous substring: `D` matched Todo, Doing, DeleteMe — listed candidates with IDs.

## Bugs Found

None.

## Cleanup

- Deleted temporary pipelines (DeleteMe, DeleteMe2, DeleteMe3) during testing.
- Removed test alias `todo-alias`.
- Final pipeline state matches pre-test state (Todo, Doing, Test).
