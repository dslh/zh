# Manual Testing: `zh pipeline create`

## Summary

All tests passed. No bugs found.

## Test Environment

- Workspace: Dev Test (`69866ab95c14bf002977146b`)
- Starting pipelines: Todo, Doing, Test

## Tests Performed

### Basic creation
```
$ zh pipeline create 'Review'
Created pipeline "Review".

  ID: Z2lkOi8vcmFwdG9yL1BpcGVsaW5lLzM1MzgzNjU
```
Pipeline appeared at the end of the board (position 4). Verified via `zh pipeline list` and `zh pipeline show Review`.

### --position flag
```
$ zh pipeline create 'QA' --position=2
Created pipeline "QA" at position 2.
```
Verified QA appeared at position 3 in the board (0-indexed position 2, after Todo and Doing).

### --position=0 (edge case)
```
$ zh pipeline create 'First' --position=0
Created pipeline "First" at position 0.
```
Verified "First" appeared at the very start of the board.

### --description flag
```
$ zh pipeline create 'Staging' --description='Pre-production staging area'
Created pipeline "Staging".

  ID: Z2lkOi8vcmFwdG9yL1BpcGVsaW5lLzM1MzgzNjc
  Description: Pre-production staging area
```
Description is displayed in creation output.

### --position + --description combined
```
$ zh pipeline create 'Deploy' --position=5 --description='Ready for deployment'
Created pipeline "Deploy" at position 5.

  ID: Z2lkOi8vcmFwdG9yL1BpcGVsaW5lLzM1MzgzNjg
  Description: Ready for deployment
```

### --dry-run
```
$ zh pipeline create 'Fake Pipeline' --dry-run
Would create pipeline "Fake Pipeline".
```
No pipeline was created. Confirmed via `zh pipeline list`.

### --dry-run with all flags
```
$ zh pipeline create 'Fake Pipeline' --dry-run --position=3 --description='A test description'
Would create pipeline "Fake Pipeline" at position 3.

  Description: A test description
```

### --output=json
```
$ zh pipeline create 'JSONTest' --output=json
{
  "id": "Z2lkOi8vcmFwdG9yL1BpcGVsaW5lLzM1MzgzNjk",
  "name": "JSONTest",
  "description": null,
  "stage": null,
  "createdAt": "2026-02-10T18:49:32Z"
}
```
Valid JSON with all expected fields.

### Missing arguments (error case)
```
$ zh pipeline create
Error: accepts 1 arg(s), received 0
```
Exit code 2 (usage error).

### Duplicate name (error case)
```
$ zh pipeline create 'First'
Error: creating pipeline: Not unique
```
Exit code 1 (general error from API).

### --help
```
$ zh pipeline create --help
```
Help text is complete with all flags documented.

## Cosmetic Notes

- The `--position` flag displays `(default -1)` in help text. The `-1` is an internal sentinel meaning "not set" and has no user-facing meaning. Not a bug, but slightly confusing.

## Cleanup

All test pipelines were deleted via `zh pipeline delete --into=Todo` after testing. Workspace was restored to its original state (Todo, Doing, Test).
