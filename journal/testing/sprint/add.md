# Manual Testing: `zh sprint add`

## Summary

All tests passed. No bugs found. The command works correctly across all tested scenarios.

## Test Environment

- Active sprint: "Sprint: Feb 8 - Feb 22, 2026" (`Z2lkOi8vcmFwdG9yL1NwcmludC80NjMzMDg0`)
- Repositories: `dlakehammond/task-tracker`, `dlakehammond/recipe-book`
- Sprint started with 0 issues

## Tests Performed

### Issue Identifier Formats

| # | Test | Result |
|---|------|--------|
| 1 | Single issue, `repo#number` format (`task-tracker#1`) | Pass |
| 2 | Multiple issues, `repo#number` format (`task-tracker#2 recipe-book#1`) | Pass |
| 3 | `owner/repo#number` format (`dlakehammond/task-tracker#3`) | Pass |
| 4 | `--repo` flag with bare number (`--repo=task-tracker 4`) | Pass |
| 5 | `--repo` flag with multiple bare numbers (`--repo=recipe-book 2 3`) | Pass |
| 6 | ZenHub ID (`Z2lkOi8vcmFwdG9yL0lzc3VlLzM4NjQ0NDUwNg`) | Pass |
| 7 | PR reference (`task-tracker#5`) | Pass |

### Sprint Targeting

| # | Test | Result |
|---|------|--------|
| 8 | Default (active sprint, no `--sprint` flag) | Pass |
| 9 | `--sprint=next` | Pass — added to "Sprint: Feb 22 - Mar 8, 2026" |
| 10 | `--sprint=previous` (dry-run) | Pass — resolved to "Sprint: Jan 22 - Feb 5, 2026" |
| 11 | Sprint name substring (`--sprint="Mar 8 - Mar 22"`) | Pass |
| 12 | Sprint ZenHub ID (`--sprint=Z2lkOi8v...`) | Pass |

### Flags and Options

| # | Test | Result |
|---|------|--------|
| 13 | `--dry-run` with multiple issues | Pass — shows "Would add 2 issue(s)" with list |
| 14 | `--dry-run` with `--continue-on-error` and mixed valid/invalid | Pass — shows resolved issues and failed list |
| 15 | `--output=json` | Pass — valid JSON with `sprint` and `added` fields |
| 16 | `--help` | Pass — shows usage, examples, and all flags |
| 17 | `--continue-on-error` with partial failure | Pass — adds valid issues, reports failed ones, exits with error |

### Error Handling

| # | Test | Result |
|---|------|--------|
| 18 | No arguments | Pass — "requires at least 1 arg(s)", exit code 2 |
| 19 | Invalid repo reference (`nonexistent-repo#999`) | Pass — "repository not found", exit code 4 |
| 20 | All issues fail with `--continue-on-error` | Pass — "all issues failed to resolve", exit code 1 |

### Idempotency

| # | Test | Result |
|---|------|--------|
| 21 | Adding an already-added issue | Pass — succeeds silently (API is idempotent) |

### Output Format

- Single issue: `Added task-tracker#1 to Sprint: Feb 8 - Feb 22, 2026.`
- Multiple issues: Header line with count, followed by indented list of `ref title` pairs
- Partial failure: Success count header, success list, then "Failed:" section with reasons
- Dry run: "Would add N issue(s)" prefix with indented list
- JSON: `{"sprint": {"id": "...", "name": "..."}, "added": [...]}`

## Verification

After all add operations, `zh sprint show current` confirmed 12 issues in the sprint matching all additions. Issues were cleaned up after testing.

## Bugs Found

None.
