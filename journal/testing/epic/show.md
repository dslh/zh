# zh epic show — Manual Testing Report

## Summary

Tested `zh epic show <epic>` for both ZenHub epics and legacy (GitHub-issue-backed) epics across all supported identifier types, output formats, and flags.

## Test Environment

- Workspace: "Dev Test" (`69866ab95c14bf002977146b`)
- Epics available: 2 ZenHub epics ("Q1 Platform Improvements", "Bug Bash Sprint"), 1 legacy epic ("Recipe Book Improvements" backed by `recipe-book#5`)

## Tests Performed

### Identifier types

| Identifier type | Input | Result |
|---|---|---|
| Exact title | `"Q1 Platform Improvements"` | OK |
| Title substring (unique) | `"Q1 Platform"` | OK |
| Title substring (unique, legacy) | `"Recipe Book"` | OK |
| Title substring (unique, 2nd zenhub) | `"Bug Bash"` | OK |
| ZenHub ID (zenhub epic) | `Z2lkOi8vcmFwdG9yL1plbmh1YkVwaWMvMjMyNDIy` | OK |
| ZenHub ID (legacy epic) | `Z2lkOi8vcmFwdG9yL0VwaWMvMTIyMDYzOQ` | OK |
| repo#number (legacy) | `recipe-book#5` | OK |
| owner/repo#number (legacy) | `dlakehammond/recipe-book#5` | OK |
| Alias | `q1` (set via `zh epic alias`) | OK |
| Ambiguous substring | `"Improvements"` | Correctly errors with candidate list |
| Nonexistent | `"nonexistent-epic"` | Correctly errors with exit code 4 |

### Flags

| Flag | Result |
|---|---|
| `--output=json` (zenhub) | Valid JSON with all fields |
| `--output=json` (legacy) | Valid JSON with all fields |
| `--verbose` | Logs GraphQL query, variables, and response to stderr |
| `--help` | Shows correct usage, all flags documented |
| No argument | Correctly errors with usage hint (exit code 2) |

### Display sections (tested with child issues added to epic)

| Section | Result |
|---|---|
| Header (EPIC: title) | OK |
| Metadata (Type, ID, State, Estimate, Creator, Assignees, Labels, Created, Updated) | OK |
| PROGRESS (issue count + estimate progress bars) | OK (after fix) |
| CHILD ISSUES (table with ref, state, title, estimate, pipeline) | OK |
| DESCRIPTION (markdown rendered) | OK |
| LINKS (legacy epic GitHub URL) | OK |

## Bugs Found and Fixed

### 1. IssueEstimateProgress fields typed as `int` instead of `float64`

**Symptom:** `zh epic show` crashed with:
```
Error: parsing epic details: json: cannot unmarshal number 1.0 into Go struct field .node.zenhubIssueEstimateProgress.open of type int
```

**Cause:** The ZenHub API returns `zenhubIssueEstimateProgress` values as floats (e.g. `1.0`, `0.0`) since they represent story point estimates. Six structs in `cmd/epic.go` had these fields typed as `int`.

**Fix:** Changed all `IssueEstimateProgress` struct fields from `int` to `float64` across all six struct definitions. Added `int()` casts at `FormatProgress()` call sites (consistent with existing pattern in `sprint.go` and `workspace.go`).

**Affected structs:**
- `epicListEntry` (line 33)
- `epicDetailZenhub` (line 91)
- `epicDetailLegacy` (line 153)
- Anonymous struct in `fetchLegacyEpicList` (line 702)
- Anonymous struct in `runEpicProgressZenhub` (line 1214)
- Anonymous struct in `runEpicProgressLegacy` (line 1297)

### 2. Unused `$workspaceId` variable in `epicProgressZenhubQuery`

**Symptom:** `zh epic progress <epic>` crashed with:
```
Error: fetching epic progress: Variable $workspaceId is declared by GetZenhubEpicProgress but not used
```

**Cause:** The GraphQL query declared `$workspaceId: ID!` in its parameter list but never referenced it in the query body. The ZenHub API rejects queries with declared-but-unused variables.

**Fix:** Removed `$workspaceId` from the query declaration and from the variables map passed to `client.Execute()`.

## Test Suite

All existing tests pass after the fixes. Linter reports 0 issues.
