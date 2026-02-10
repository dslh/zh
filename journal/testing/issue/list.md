# Manual Testing: `zh issue list`

## Summary

Tested all flags and filter combinations for `zh issue list`. Found and fixed three bugs related to API filter construction.

## Bugs Found and Fixed

### 1. `--estimate` filter passed string instead of float (cmd/issue.go)

**Symptom:** `zh issue list --estimate 1` failed with:
```
Error: Variable $filters of type IssueSearchFiltersInput! was provided invalid value for estimates.values.in.0 (Could not coerce value "1" to Float)
```

**Root cause:** The `buildIssueListFilters` function passed the estimate value as a string (`[]string{issueListEstimate}`) but the ZenHub API `EstimateSearchFiltersInput.values.in` field expects `[Float!]`.

**Fix:** Parse the estimate string to `float64` using `strconv.ParseFloat` before passing it in the filter as `[]float64{val}`.

### 2. `--epic` filter missing required `filters` argument (cmd/issue.go)

**Symptom:** `zh issue list --epic "Q1 Platform"` failed with:
```
Error: fetching epic issues: Field 'searchIssuesByZenhubEpics' is missing required arguments: filters
```

**Root cause:** The `issueListByEpicQuery` GraphQL query and the `fetchIssuesByEpic` function did not include the required `filters` parameter for `searchIssuesByZenhubEpics`. The API requires a `ZenhubEpicIssueSearchFiltersInput!` argument (which only has an optional `workspaces` field).

**Fix:** Added `$filters: ZenhubEpicIssueSearchFiltersInput!` variable to the query and passed an empty `filters: {}` map in the variables.

### 3. `--sprint` filter used wrong field name (cmd/issue.go)

**Symptom:** `zh issue list --sprint "Feb 8"` failed with:
```
Error: Variable $filters of type IssueSearchFiltersInput! was provided invalid value for sprints.ids (Field is not defined on SprintIdInput)
```

**Root cause:** The sprint filter used `"ids"` as the key (`map[string]any{"ids": []string{resolved.ID}}`) but `SprintIdInput` expects `"in"` (matching the `IdInput` pattern).

**Fix:** Changed `"ids"` to `"in"` in the sprint filter construction.

## Test Results

### Basic listing
- `zh issue list` — Lists all open issues across all pipelines with correct columns (ISSUE, TITLE, EST, PIPELINE, ASSIGNEE, LABELS)
- Footer shows correct count: "Showing 8 issue(s)"

### Pipeline filter (`--pipeline`)
- `--pipeline Todo` — Shows only issues in Todo pipeline
- `--pipeline Doing` — Shows only issues in Doing pipeline
- `--pipeline oing` — Substring match correctly resolves to "Doing"
- `--pipeline Do` — Correctly errors as ambiguous (matches "Todo" and "Doing") with candidate list
- `--pipeline nonexistent` — Correctly errors with "not found" message and exit code 4

### Label filter (`--label`)
- `--label bug` — Shows only issues with "bug" label
- `--label enhancement` — Shows only issues with "enhancement" label

### Repo filter (`--repo`)
- `--repo task-tracker` — Shows only issues from task-tracker repo
- `--repo recipe-book` — Shows only issues from recipe-book repo
- `--repo dlakehammond/task-tracker` — Full owner/name format works

### Estimate filters (`--estimate`, `--no-estimate`)
- `--estimate 1` — Shows only issues with estimate of 1 (after fix)
- `--no-estimate` — Shows only unestimated issues

### Type filter (`--type`)
- `--type issues` — Shows only issues (no PRs)
- `--type prs` — Shows only PRs

### State filter (`--state`)
- `--state closed` — Shows closed issues
- Default (no flag) — Shows open issues

### Sprint filter (`--sprint`)
- `--sprint current` — Shows issues in the active sprint
- `--sprint "Feb 8"` — Substring match resolves sprint by name (after fix)

### Epic filter (`--epic`)
- `--epic "Q1 Platform"` — Shows issues in the matched epic (after fix)
- `--epic "Q1 Platform" --pipeline Doing` — Combined epic + pipeline filter works

### Assignee filter (`--assignee`, `--no-assignee`)
- `--assignee dlakehammond` — Returns empty (no assigned issues in workspace), correct behavior
- `--no-assignee` — Shows all unassigned issues

### Combined filters
- `--label bug --repo task-tracker` — Correctly combines filters
- `--type prs --repo recipe-book` — Shows only PRs from recipe-book

### Output format (`--output`)
- `--output json` — Produces valid JSON array with all issue fields
- `--state closed --output json` — JSON output works with closed state

### Limit and pagination (`--limit`, `--all`)
- `--limit 3` — Shows exactly 3 issues, footer shows "Showing 3 of 8 issue(s)"
- `--all` — Shows all issues

### Verbose mode (`--verbose`)
- `--verbose` — Logs API requests to stderr

### Error handling
- No workspace configured — Returns appropriate error message
- Nonexistent pipeline — Exit code 4 with helpful error
- Ambiguous pipeline — Exit code 2 with candidate list
