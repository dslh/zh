# Manual Testing: `zh sprint remove`

## Summary

All tests passed. No bugs found. The command works correctly across all tested scenarios.

## Test Environment

- Workspace: Dev Test (`69866ab95c14bf002977146b`)
- Active sprint: Sprint: Feb 8 - Feb 22, 2026 (`Z2lkOi8vcmFwdG9yL1NwcmludC80NjMzMDg0`)
- Repositories: `dlakehammond/task-tracker`, `dlakehammond/recipe-book`

## Tests Performed

### Basic removal

| # | Test | Result |
|---|------|--------|
| 1 | Single issue removal (`task-tracker#3`) | Pass |
| 2 | Verified issue actually removed via `sprint show` | Pass |
| 3 | Owner/repo#number format (`dlakehammond/task-tracker#4`) | Pass |
| 4 | Multiple issues at once (`task-tracker#3 recipe-book#2`) | Pass |
| 5 | `--repo` flag with bare numbers (`--repo=task-tracker 3 4`) | Pass |
| 6 | ZenHub issue ID (`Z2lkOi8vcmFwdG9yL0lzc3VlLzM4NjI2MTgzMA`) | Pass |
| 7 | Cross-repo issues in single command (`task-tracker#2 recipe-book#2`) | Pass |
| 8 | PR removal (`task-tracker#5`) | Pass |

### Sprint targeting (`--sprint` flag)

| # | Test | Result |
|---|------|--------|
| 9 | `--sprint=next` | Pass |
| 10 | `--sprint=current` (equivalent to default) | Pass |
| 11 | `--sprint=previous` (closed sprint) | Pass |
| 12 | Sprint name substring (`--sprint='Feb 22 - Mar'`) | Pass |
| 13 | Sprint ZenHub ID (`--sprint=Z2lkOi8vcmFwdG9yL1NwcmludC80NjMzMDg1`) | Pass |
| 14 | Ambiguous sprint name substring — correctly errors with candidates | Pass |

### Output formats

| # | Test | Result |
|---|------|--------|
| 15 | Single issue output: `Removed task-tracker#3 from Sprint: ...` | Pass |
| 16 | Multi-issue output: `Removed 2 issue(s) from Sprint: ...` with list | Pass |
| 17 | `--output=json` — structured JSON with `sprint` and `removed` fields | Pass |
| 18 | `--verbose` — shows all GraphQL API requests and responses | Pass |

### Dry-run (`--dry-run`)

| # | Test | Result |
|---|------|--------|
| 19 | Single issue dry-run: `Would remove 1 issue(s) from Sprint: ...` | Pass |
| 20 | Multi-issue dry-run | Pass |
| 21 | Verified dry-run does not actually remove (sprint show confirms) | Pass |
| 22 | `--dry-run --continue-on-error` with mixed valid/invalid issues | Pass |

### Error handling

| # | Test | Result |
|---|------|--------|
| 23 | No arguments — exit code 2, usage error | Pass |
| 24 | Invalid repo reference — exit code 4, helpful error message | Pass |
| 25 | Stop on first error (default) — stops processing, does not execute remaining | Pass |
| 26 | `--continue-on-error` — processes valid issues, reports failures separately | Pass |
| 27 | All issues fail with `--continue-on-error` — `all issues failed to resolve` | Pass |

### Idempotency

| # | Test | Result |
|---|------|--------|
| 28 | Removing an issue not in the sprint — succeeds silently (API is idempotent) | Pass |

### Help

| # | Test | Result |
|---|------|--------|
| 29 | `--help` — shows usage, examples, all flags documented | Pass |

## Notes

- The ZenHub API treats remove as idempotent — removing an issue that isn't in the sprint succeeds silently. This is reasonable behavior.
- The `MutationDryRun` output does not align issue refs (unlike `MutationBatch` which uses `printItems`). This is a pre-existing cosmetic inconsistency across all dry-run outputs, not specific to `sprint remove`.
- Partial failure output correctly separates succeeded and failed items, with exit code 1.

## Bugs Found

None.
