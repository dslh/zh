# Manual Testing: `zh priority list`

## Summary

The `zh priority list` command lists priorities configured for the current ZenHub workspace. Testing revealed one bug related to color formatting which was fixed.

## Tests Performed

### Basic output
```
$ zh priority list
PRIORITY         COLOR
────────────────────────────────────────────────────────────────────────────────
High priority    #ff5630

Total: 1 priority(s)
```
Working correctly after fix. Displays priority name and color in a two-column table with a footer count.

### JSON output (`--output=json`)
```
$ zh priority list --output=json
[
  {
    "id": "Z2lkOi8vcmFwdG9yL1ByaW9yaXR5LzU1Mzg4NA",
    "name": "High priority",
    "color": "var(--zh-theme-color-red-primary)",
    "description": ""
  }
]
```
Working correctly. Outputs raw priority objects as JSON array. Note that JSON output preserves the raw API color value (CSS variable reference) rather than the resolved hex code — this is acceptable since JSON consumers may want the raw value.

### Help text (`--help`)
```
$ zh priority list --help
List all priorities configured for the current workspace, including their colors.
```
Help text is clear and accurate. Global flags (`--output`, `--verbose`) are shown.

### Parent command (`zh priority`)
Shows usage information with available subcommands. Exits cleanly with code 0.

### Verbose mode (`--verbose`)
Correctly logs the GraphQL request and response to stderr while displaying normal output to stdout. Shows the `GetWorkspacePriorities` query and raw API response.

### NO_COLOR
Output renders correctly without color when `NO_COLOR=1` is set.

### Caching
After first run, priority data is cached to `priorities-{workspace_id}.json`. Subsequent runs use the cached data.

## Bugs Found and Fixed

### Bug: CSS variable color values displayed incorrectly

**Symptom:** The COLOR column showed `#var(--zh-theme-color-red-primary)` — a `#` was prepended to a CSS variable reference.

**Cause:** The ZenHub API returns CSS variable references (e.g. `var(--zh-theme-color-red-primary)`) instead of hex color codes for priority colors. The `formatPriorityColor` function in `cmd/priority.go` assumed all non-`#`-prefixed values were bare hex codes and blindly prepended `#`.

**Fix:** Updated `formatPriorityColor` to:
1. Return hex codes with `#` prefix (existing behavior for actual hex values)
2. Map known ZenHub CSS variables to their corresponding hex colors (e.g. `var(--zh-theme-color-red-primary)` → `#ff5630`)
3. Extract a readable color name from unknown CSS variables as a fallback

Added a `cssVarColors` map covering the known ZenHub theme color palette (red, orange, yellow, green, teal, blue, purple).

**Tests added:**
- `TestFormatPriorityColor` — unit test for the formatting function covering hex codes, CSS variables (known and unknown)
- `TestPriorityListCSSVarColors` — integration test verifying CSS variable colors are resolved correctly in table output

**Files changed:**
- `cmd/priority.go` — Updated `formatPriorityColor`, added `cssVarColors` map
- `cmd/priority_test.go` — Added two new test functions
