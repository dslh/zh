# 003 — Cache Framework

Phase 2 complete. The cache framework provides file-backed JSON caching with XDG-compliant paths and an invalidate-on-miss pattern.

## What was done

- **Cache package** (`internal/cache/`):
  - XDG-compliant path resolution: `$XDG_CACHE_HOME/zh/` or `~/.cache/zh/`
  - `Key` type with `NewKey` (unscoped) and `NewScopedKey` (workspace-scoped) constructors
  - File naming per spec: `{resource}.json` or `{resource}-{workspace_id}.json`
  - Generic `Get[T]` / `Set[T]` using JSON serialization with 0600 permissions
  - `Clear(key)`, `ClearAll()`, `ClearWorkspace(id)` for targeted and bulk cache removal
  - `ClearAll` only removes `.json` files, preserving any non-cache files
  - `GetOrRefresh[T, R]`: invalidate-on-miss pattern — checks cache, calls lookup, refreshes from API on miss, retries lookup

- **Cache clear command** (`cmd/cache.go`):
  - `zh cache clear` — removes all cached data
  - `zh cache clear --workspace` — removes only cache files scoped to the current workspace
  - Errors with guidance when `--workspace` is used but no workspace is configured

## Tests

- 16 cache package tests: Dir resolution (XDG and fallback), key filenames, get/set round-trip, cache miss, corrupted cache, clear (single, all, workspace-scoped), nonexistent cache dir, GetOrRefresh (cache hit, cache miss, stale lookup miss, refresh error, not found after refresh)
- 6 command tests: clear all, clear workspace, no workspace configured, help text, empty dir, preserves non-JSON files
- All 37 tests pass, lint clean
