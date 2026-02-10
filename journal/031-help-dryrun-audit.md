# Phase 14: --help and --dry-run audit

## Summary

Audited all commands for --help text accuracy, --dry-run flag completeness, and dry-run output format consistency.

## What was done

- **Help text audit**: Reviewed --help output for all 90+ commands and subcommands against SPEC.md. All commands have accurate, well-structured help text with appropriate Use, Short, Long descriptions and flag documentation. No missing commands or contradictions found.

- **Dry-run flag audit**: Verified all 28 commands listed in SPEC.md as requiring --dry-run have the flag properly registered and checked. No commands are missing --dry-run, and no read-only commands have it erroneously.

- **Dry-run output consistency**: Found 9 dry-run implementations in `pipeline_mutations.go` and `epic_mutations.go` using ad-hoc `MutationSingle(Yellow(...))` + manual `fmt.Fprintln` calls instead of the standard `MutationDryRun()` helper. These were single-entity operations (create, edit, delete, set-state, set-dates) that display key-value metadata rather than item lists. Added a new `MutationDryRunDetail` helper to `output/mutation.go` that handles this pattern with aligned key-value detail lines. Migrated all 9 commands to use it.

- **Tests**: Added `cmd/audit_test.go` with three comprehensive test suites:
  - `TestAllHelpText`: Verifies --help produces valid output for all 90+ commands
  - `TestDryRunFlagRegistered`: Verifies --dry-run flag exists on all 28 mutation commands
  - `TestNoDryRunOnReadOnlyCommands`: Verifies 24 read-only commands do NOT have --dry-run

## Files changed

- `internal/output/mutation.go` — Added `DetailLine` type and `MutationDryRunDetail` function
- `internal/output/mutation_test.go` — Added tests for `MutationDryRunDetail` (with and without details)
- `test/snapshots/mutation-dry-run-detail.txt` — New snapshot for single-detail dry-run output
- `test/snapshots/mutation-dry-run-detail-multi.txt` — New snapshot for multi-detail dry-run output
- `cmd/pipeline_mutations.go` — Refactored 3 dry-run blocks to use `MutationDryRunDetail`
- `cmd/epic_mutations.go` — Refactored 6 dry-run blocks to use `MutationDryRunDetail`
- `cmd/audit_test.go` — New test file with help text and dry-run audit tests
