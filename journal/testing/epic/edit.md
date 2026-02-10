# Manual Testing: `zh epic edit`

## Summary

Tested `zh epic edit <epic>` with various identifier types, flags, and epic types (ZenHub and legacy). All functionality works correctly. No bugs found.

## Test Environment

- Workspace: Dev Test (`69866ab95c14bf002977146b`)
- ZenHub epics tested: "Bug Bash Sprint", "Q1 Platform Improvements"
- Legacy epics tested: "Recipe Book Improvements" (`recipe-book#5`)
- GitHub auth: `gh auth switch -u dlakehammond` (required for legacy epic edits)

## Tests Performed

### Validation

| Test | Command | Result |
|------|---------|--------|
| No flags provided | `zh epic edit "Q1 Platform Improvements"` | Exit 2: "at least one of --title or --body must be provided" |
| No epic argument | `zh epic edit` | Exit 1: "accepts 1 arg(s), received 0" |
| Nonexistent epic | `zh epic edit "Nonexistent" --title "x"` | Exit 4: "epic not found" |
| Ambiguous substring | `zh epic edit "Legacy" --title "x" --dry-run` | Exit 2: lists 3 matching epics |

### ZenHub Epic Edits

| Test | Command | Result |
|------|---------|--------|
| Edit title (dry-run) | `zh epic edit "Platform" --title "Q1 Platform Improvements" --dry-run` | Shows dry-run output with title |
| Edit body (dry-run) | `zh epic edit "Bug Bash" --body "..." --dry-run` | Shows dry-run output with body |
| Edit both (dry-run) | `zh epic edit "Bug Bash" --title "..." --body "..." --dry-run` | Shows both title and body |
| Edit title (real) | `zh epic edit "Bug Bash Sprint" --title "Bug Bash Sprint EDITED"` | Title updated, verified via `epic list` |
| Edit body (real) | `zh epic edit "Bug Bash Sprint" --body "..."` | Body updated, verified via `epic show --output=json` |
| JSON output | `zh epic edit "Bug Bash Sprint" --body "Reset body" --output=json` | Returns JSON with id, title, body, state, updatedAt |
| Verbose mode | `zh epic edit "Bug Bash Sprint" --body "..." --verbose` | Shows GraphQL request/response on stderr |

### Identifier Types

| Test | Identifier | Result |
|------|------------|--------|
| Exact title | `"Q1 Platform Improvements"` | Resolves correctly |
| Title substring | `"Platform"` | Resolves to "Q1 Platform Improvements" |
| ZenHub ID | `Z2lkOi8vcmFwdG9yL1plbmh1YkVwaWMvMjMyNDIz` | Resolves correctly |
| Epic alias | `bb` (alias for "Bug Bash Sprint") | Resolves correctly |
| Issue ref (legacy) | `recipe-book#5` | Resolves to legacy epic |
| Full ref (legacy) | `dlakehammond/recipe-book#5` | Resolves to legacy epic |

### Legacy Epic Edits

| Test | Command | Result |
|------|---------|--------|
| Edit title (dry-run) | `zh epic edit "Recipe Book Improvements" --title "..." --dry-run` | Shows legacy dry-run with GitHub note |
| Edit body (dry-run) | `zh epic edit "dlakehammond/recipe-book#5" --body "..." --dry-run` | Resolves via owner/repo#number |
| Edit title (real) | `zh epic edit "Recipe Book Improvements" --title "... EDITED"` | Updated, verified via `gh issue view` |
| JSON output (legacy) | `zh epic edit "recipe-book#5" --body "..." --output=json` | Returns JSON with id, issue ref, title, body, state |
| Wrong GitHub auth | Edit with `dslh` (not `dlakehammond`) active | Correct error: "does not have the correct permissions" |

## Notes

- Cobra's built-in argument count validation returns exit code 1 (general error) rather than 2 (usage error). This is a cross-cutting concern affecting all commands with `cobra.ExactArgs`, not specific to `epic edit`.
- After editing an epic, the cache entry for the old title becomes stale. The `epic edit` command correctly invalidates the epic cache after a real mutation. Dry-run mode does not invalidate the cache (expected behavior).
- Legacy epic edits require `gh auth switch -u dlakehammond` since the test repos are owned by that account.

## Bugs Found

None.
