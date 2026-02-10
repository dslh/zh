# 014: Issue move command

## Scope
Implemented `zh issue move <issue>... <pipeline>` — the first issue mutation command.

## What was done
- New file `cmd/issue_move.go` with the full `issue move` implementation
- Resolves issue identifiers to ZenHub IDs, then fetches PipelineIssue IDs
- Uses `moveIssue` mutation for numeric position (single issue only)
- Uses `moveIssueRelativeTo` mutation for symbolic position (top/bottom) and default
- `--position=<top|bottom|n>` flag with validation
- `--dry-run` shows what would be moved, including current pipeline context
- Stop-on-first-error by default; `--continue-on-error` processes all items and reports partial failures
- `--repo` flag for bare issue numbers
- JSON output support
- Batch move support (multiple issues to same pipeline)
- Numeric position correctly restricted to single-issue moves (API limitation)

## Tests (14 total)
- Single move, batch move (with differentiated mock responses)
- Position: top, bottom, numeric, invalid
- Numeric position + batch = error
- Dry run: basic, with position, shows current pipeline
- Stop-on-error, continue-on-error with partial failure
- JSON output
- Help text

## Design decisions
- Used `moveIssueRelativeTo` for default (no position) moves rather than `movePipelineIssues`, since the latter requires PipelineIssue IDs which closed issues lack
- Batch moves are sequential per-issue rather than using `movePipelineIssues` bulk API — simpler error handling and more consistent position semantics
- Help test placed last in file to avoid Cobra `--help` flag state poisoning subsequent tests
