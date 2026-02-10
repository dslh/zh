# 012: Pipeline automations and board command

## Scope

Phase 6 completion (pipeline automations) and Phase 7 (board command).

## Changes

### `zh pipeline automations <name>` (Phase 6)

- Queries all pipelines with both automation types in a single request, filters client-side to the target pipeline
- Displays event automations (with raw `elementDetails` JSON) and pipeline-to-pipeline automations (as a direction/pipeline table)
- Shows "No automations configured" when the pipeline has none
- JSON output includes all automation data in structured form
- Tests: with P2P automations, with event automations, no automations, JSON output

### `zh board` (Phase 7)

- Fetches full board (all pipelines with issues) in a single GraphQL query
- Renders each pipeline as a section: bold name, separator, issue list with ref/title/estimate/assignee
- `--pipeline=<name>` flag filters to a single pipeline using `searchIssuesByPipeline` (reuses existing `fetchPipelineIssues`)
- Caches pipeline list from board query for subsequent resolution
- Handles repo name ambiguity (short vs long form issue references)
- JSON output: array of pipelines with nested issues
- Tests: full board, empty pipelines, no pipelines, JSON, filtered pipeline, filtered pipeline JSON

## Files

- `cmd/pipeline.go` -- Added automations types, query, command, and run function
- `cmd/pipeline_test.go` -- 4 new tests for automations
- `cmd/board.go` -- New file: board command with full and filtered views
- `cmd/board_test.go` -- New file: 6 tests for board command

## Verified

- `zh pipeline automations Todo` -- shows "No automations configured" (test workspace has none)
- `zh pipeline automations Todo --output=json` -- structured JSON output
- `zh board` -- displays both pipelines with all 10 issues
- `zh board --pipeline=Todo` -- filters to single pipeline
- `zh board --output=json` -- full structured JSON output
- All tests pass, linter clean
