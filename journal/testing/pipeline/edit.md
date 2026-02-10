# Manual Testing: `zh pipeline edit`

## Summary

Tested `zh pipeline edit <pipeline>` with various flags, identifier types, and edge cases. Found and fixed one bug related to clearing descriptions with an empty string.

## Test Environment

- Workspace: Dev Test (`69866ab95c14bf002977146b`)
- Pipelines at start: Todo, Doing, Test

## Tests Performed

### Flag Combinations

| Test | Command | Result |
|------|---------|--------|
| No flags | `zh pipeline edit Todo` | Error: "no changes specified" (exit 2) |
| --name only | `zh pipeline edit Test --name=Testing` | Renamed successfully, verified with `pipeline list` |
| --position only | `zh pipeline edit Test --position=1` | Moved successfully, verified with `pipeline list` |
| --description only | `zh pipeline edit Test --description='QA and testing pipeline'` | Set description, verified with `pipeline show` |
| All flags | `zh pipeline edit Todo --name=Backlog --position=2 --description='Items to be done' --dry-run` | Dry-run showed all three changes |
| Clear description | `zh pipeline edit Test --description=` | **BUG FOUND** (see below); fixed and retested successfully |

### Identifier Resolution

| Test | Command | Result |
|------|---------|--------|
| Exact name | `zh pipeline edit Todo --name=Backlog --dry-run` | Resolved correctly |
| Substring | `zh pipeline edit Tod --name=Todo2 --dry-run` | Resolved to "Todo" |
| Ambiguous substring | `zh pipeline edit Do --name=Test2 --dry-run` | Error listing 2 candidates: Todo, Doing (exit 2) |
| ZenHub ID | `zh pipeline edit Z2lkOi8vcmFwdG9yL1BpcGVsaW5lLzM1MzcyMjg --name=Todo2 --dry-run` | Resolved correctly |
| Alias | `zh pipeline edit tst --description='Using alias' --dry-run` | Resolved "tst" alias to "Test" |
| Nonexistent | `zh pipeline edit NonexistentPipeline --name=Test2 --dry-run` | Error: not found (exit 4) |

### Output Modes

| Test | Command | Result |
|------|---------|--------|
| Default output | `zh pipeline edit Test --name=Testing` | `Updated pipeline "Testing".` |
| JSON output | `zh pipeline edit Test --description='test desc' --output=json` | Valid JSON with id, name, description, stage, updatedAt |
| Dry-run | `zh pipeline edit Todo --name=Backlog --dry-run` | `Would update pipeline "Todo":` with detail lines |
| Verbose | `zh pipeline edit Test --description='verbose test' --verbose` | Logged full GraphQL request/response to stderr |

### Error Handling

| Test | Command | Result |
|------|---------|--------|
| Missing argument | `zh pipeline edit` | Error: "accepts 1 arg(s), received 0" (exit 2) |
| Extra arguments | `zh pipeline edit Test extra --name=New` | Error: "accepts 1 arg(s), received 2" (exit 2) |

## Bug Found and Fixed

### `--description=''` fails with "no changes specified"

**Problem:** The `--description` flag help text says "use empty string to clear", but passing `--description=` (empty string) caused the command to error with "no changes specified". This was because the code checked `pipelineEditDescription != ""` to determine if the flag was set, which returns false for an empty string.

**Fix:** Changed `hasDescription` check from `pipelineEditDescription != ""` to `cmd.Flags().Changed("description")`, which correctly detects whether the flag was explicitly provided regardless of its value. Also improved the dry-run output to show `(clear)` when the description is being set to empty.

**Files changed:**
- `cmd/pipeline_mutations.go`: Updated `hasDescription` detection and dry-run display
- `cmd/pipeline_mutations_test.go`: Added `TestPipelineEditClearDescription` and `TestPipelineEditClearDescriptionDryRun`

## Verification

- All existing tests pass
- Two new tests added and passing
- Linter clean (0 issues)
- Pipeline state restored to original after testing
