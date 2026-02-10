# Manual Testing: `zh pipeline automations`

## Summary

The `zh pipeline automations <pipeline>` command displays event automations and pipeline-to-pipeline automations configured for a specified pipeline. Tested against the "Dev Test" workspace.

## Test Results

All tests passed. No bugs found.

### Pipeline resolution

| Input | Result |
|-------|--------|
| `Todo` (exact name) | Resolved correctly, showed automations for Todo |
| `Doi` (unique substring) | Resolved to "Doing" correctly |
| `Do` (ambiguous substring) | Exit code 2, listed both "Todo" and "Doing" as candidates |
| `Z2lkOi8v...` (ZenHub ID) | Resolved correctly to "Todo" |
| `todo` (alias) | Resolved via config alias to "Todo" |
| `nonexistent` | Exit code 4, "not found" error with suggestion to run `zh pipeline list` |

### Output formats

| Format | Result |
|--------|--------|
| Default (text) | Detail view with header "AUTOMATIONS: <name>" and "No automations configured." |
| `--output=json` | Valid JSON with `pipeline`, `pipelineId`, `eventAutomations`, `p2pSources`, `p2pDestinations` (all empty arrays) |

### Flags

| Flag | Result |
|------|--------|
| `--help` | Displayed usage, description, and available flags |
| `--verbose` | Logged the GraphQL query, variables, and raw API response to stderr |
| `--output=json` | Produced structured JSON output |

### Edge cases

| Scenario | Result |
|----------|--------|
| No arguments | Exit code 2, "accepts 1 arg(s), received 0" |
| No automations on any pipeline | Correctly showed "No automations configured." for all 3 pipelines |

### Unit tests

All 4 unit tests pass:
- `TestPipelineAutomations` — P2P automations display
- `TestPipelineAutomationsNoAutomations` — empty state
- `TestPipelineAutomationsJSON` — JSON output structure
- `TestPipelineAutomationsWithEventAutomations` — event automations display

## Notes

The test workspace has no automations configured on any pipeline, so the "no automations" path was the only one exercised against the live API. The code paths for displaying event automations and P2P automations are covered by the unit tests with mock data. Automations are a workspace-level configuration typically set up via the ZenHub web UI and cannot be created via the API.

## Bugs

None found.
