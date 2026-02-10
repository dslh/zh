# zh cache clear — Manual Testing

## Summary

The `zh cache clear` command works correctly. One minor fix was applied.

## Tests Performed

| # | Test | Result |
|---|------|--------|
| 1 | `zh cache clear --help` | Pass — shows usage, flags (`--workspace`), and description |
| 2 | `zh cache --help` | Pass — shows parent command help with `clear` subcommand |
| 3 | `zh cache clear --workspace` | Pass — removes workspace-scoped files (e.g. `pipelines-{id}.json`, `epics-{id}.json`, `sprints-{id}.json`), preserves unscoped files (`workspaces.json`) |
| 4 | `zh cache clear` | Pass — removes all JSON cache files |
| 5 | `zh cache clear` (empty cache) | Pass — succeeds gracefully with confirmation message |
| 6 | `zh cache clear` (no cache directory) | Pass — succeeds gracefully when `~/.cache/zh/` doesn't exist |
| 7 | `zh cache clear --workspace` (no cache directory) | Pass — succeeds gracefully |
| 8 | `zh cache clear --workspace` (no workspace configured) | Pass — exit code 2, message: "no workspace configured — use 'zh workspace switch' to set one" |
| 9 | `zh cache clear --output=json` | Pass — prints plain text confirmation (no structured JSON, acceptable for this command) |
| 10 | `zh cache clear --verbose` | Pass — no crash, no extra output (expected since no API calls) |
| 11 | `zh cache clear extra-arg` | Pass (after fix) — rejects extra arguments with exit code 2 |
| 12 | Commands work after cache clear | Pass — `zh board` works after clearing cache, cache is repopulated from API |
| 13 | `--workspace` preserves other workspaces | Pass — files scoped to other workspace IDs are preserved |

## Fixes Applied

### Added `cobra.NoArgs` validation (`cmd/cache.go`)

The `cache clear` command was missing `Args` validation, causing it to silently accept and ignore extra positional arguments. Added `Args: cobra.NoArgs` to reject unexpected arguments with a proper error message and exit code 2.

## Notes

- The `--output=json` flag doesn't produce structured JSON for this command — it just prints the same plain text confirmation. This is acceptable since `cache clear` is a local operation with no meaningful structured data to return.
- The cache directory itself is preserved after clearing (only JSON files are removed), which is standard behavior.
