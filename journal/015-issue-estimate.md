# 015: Issue estimate command

## Scope

- `zh issue estimate <issue> [value]` — set or clear estimate on an issue

## Work done

- Implemented `cmd/issue_estimate.go` with full command logic:
  - Resolves issue via ZenHub ID or GitHub-style `repo#number` identifiers
  - Supports `--repo` flag for bare issue numbers
  - Parses optional numeric value argument (omit to clear estimate)
  - Fetches current estimate and valid estimate set from the repository
  - Validates value against the repository's configured estimate set
  - Executes `setEstimate` mutation (passes `nil` to clear)
  - Renders confirmation message (set/clear) or JSON output
  - `--dry-run` shows current value and what would change
- Implemented `cmd/issue_estimate_test.go` with 9 test cases:
  - Set estimate to valid value
  - Clear estimate (omit value)
  - Invalid estimate value (not in valid set)
  - Non-numeric value
  - Dry-run set (shows current value)
  - Dry-run clear (shows "currently: none")
  - JSON output format
  - Issue not found
  - Help text
- Manual verification against Dev Test workspace confirmed:
  - Setting estimate on `task-tracker#1`
  - Clearing estimate
  - Invalid value rejection with valid values listed
  - Dry-run with current value display
  - JSON output format
  - `--repo` flag resolution

## Notes

- Estimate validation uses the repository's `estimateSet.values` fetched at resolution time rather than a separate cache — the values come back with the issue query for free
- Single-issue operation only (no batch) — the spec doesn't call for `<issue>...` syntax on this command
