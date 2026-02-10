# 017: Issue connections & dependencies (Phase 9)

## Scope
Implemented all five commands in Phase 9: issue connections (connect/disconnect PRs) and dependencies (block/blockers/blocking).

## Commands added

### `zh issue connect <issue> <pr>`
- Resolves both identifiers and validates that the first is an issue and the second is a PR (using the `pullRequest` boolean field from the API)
- Executes `createIssuePrConnection` mutation
- Supports `--dry-run`, `--repo`, `--output=json`

### `zh issue disconnect <issue> <pr>`
- Mirror of connect; resolves and validates both identifiers
- Executes `deleteIssuePrConnection` mutation
- Supports `--dry-run`, `--repo`, `--output=json`

### `zh issue block <blocker> <blocked>`
- Uses `createBlockage` mutation (newer API, supports both issues and epics)
- `--blocker-type` and `--blocked-type` flags (default: `issue`, can be `epic`)
- Epic resolution leverages the existing `resolve.Epic()` infrastructure
- Displays a warning that blocks cannot be removed via the API
- Supports `--dry-run`, `--repo`, `--output=json`

### `zh issue blockers <issue>`
- Queries `blockingItems` field on the issue (returns both issues and epics)
- Supports both `issueByInfo` and node ID query paths
- Displays formatted list or "no blockers" message
- Supports `--repo`, `--output=json`

### `zh issue blocking <issue>`
- Queries `blockedItems` field on the issue (returns both issues and epics)
- Same dual query path as blockers
- Displays formatted list or "not blocking anything" message
- Supports `--repo`, `--output=json`

## Tests added
- 19 new tests across two test files
- Connect: success, dry-run, wrong types validation, JSON output, help
- Disconnect: success, dry-run, JSON output, help
- Block: issue-to-issue, dry-run, JSON output, invalid type, help
- Blockers: with blockers, no blockers, JSON output
- Blocking: with items, nothing, JSON output

## Files changed
- `cmd/issue_connect.go` — connect and disconnect commands
- `cmd/issue_block.go` — block, blockers, and blocking commands
- `cmd/issue_connect_test.go` — connect/disconnect tests
- `cmd/issue_block_test.go` — block/blockers/blocking tests
- `ROADMAP.md` — checked off Phase 9 items
