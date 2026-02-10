# Manual Testing: `zh workspace switch`

## Summary

All tests passed. No bugs found. The command correctly switches the default workspace, updates the config file, clears workspace-scoped caches, and handles all identifier types and error cases.

## Test Environment

A second workspace ("Test Workspace 2") was created via the ZenHub API to enable switching tests. It was deleted after testing.

## Tests Performed

### Identifier types

| Test | Command | Result |
|------|---------|--------|
| Exact name | `zh workspace switch "Test Workspace 2"` | Switched successfully |
| Substring | `zh workspace switch Dev` | Switched to "Dev Test" |
| Workspace ID | `zh workspace switch 698b98cda226b2001c6b8f38` | Switched successfully |
| Case-insensitive | `zh workspace switch "dev test"` | Switched to "Dev Test" |

### Already current workspace

| Test | Command | Result |
|------|---------|--------|
| Exact name | `zh workspace switch "Dev Test"` | `Already using workspace "Dev Test" (hambend@gmail.com)` |
| Substring | `zh workspace switch Dev` | `Already using workspace "Dev Test" (hambend@gmail.com)` |
| ID | `zh workspace switch 69866ab95c14bf002977146b` | `Already using workspace "Dev Test" (hambend@gmail.com)` |

### Error cases

| Test | Command | Exit Code | Result |
|------|---------|-----------|--------|
| Not found | `zh workspace switch nonexistent` | 4 | `workspace "nonexistent" not found — run 'zh workspace list' to see available workspaces` |
| Ambiguous | `zh workspace switch Test` | 2 | Lists both "Dev Test" and "Test Workspace 2" as candidates |
| No argument | `zh workspace switch` | 2 | `accepts 1 arg(s), received 0` |
| Extra argument | `zh workspace switch "Dev Test" extra` | 2 | `accepts 1 arg(s), received 2` |

### Side effects

| Test | Verified |
|------|----------|
| Config updated | Workspace ID in `config.yml` changed to new workspace ID |
| Old cache cleared | All workspace-scoped cache files removed after switch |
| Workspaces cache preserved | `workspaces.json` retained after switch |
| Commands work after switch | `zh workspace show` showed new workspace details |

### Flags

| Flag | Result |
|------|--------|
| `--help` | Correct help text displayed |
| `--verbose` | API requests/responses logged to stderr when cache miss triggers API call |
| `--output=json` | Accepted but no effect (mutation confirmation, not data display) |

## Bugs Found

None.
