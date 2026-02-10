# Manual Testing: `zh pipeline show`

## Summary

Tested the `zh pipeline show` command across all supported identifier types, flags, and error cases. Found and fixed one bug related to inconsistent issue counts.

## Test Environment

- Workspace: Dev Test (`69866ab95c14bf002977146b`)
- Pipelines: Todo (10 issues), Doing (1 issue), Test (0 issues)

## Tests Performed

### Identifier types

| Input | Result |
|---|---|
| Exact name (`Todo`) | Resolved correctly |
| Case-insensitive (`todo`) | Resolved correctly |
| Substring (`odo` -> Todo, `oin` -> Doing) | Resolved correctly |
| ZenHub ID (`Z2lkOi8v...`) | Resolved correctly |
| Alias (`td` after `pipeline alias Todo td`) | Resolved correctly |

### Flags

| Flag | Result |
|---|---|
| `--limit 3` | Showed 3 of 10 issues |
| `--limit 0` | Fetched all issues (same as `--all`) |
| `--all` | Fetched all issues |
| `--output json` | Valid JSON with pipeline details, issues array, and totalIssues |
| `--verbose` | Logged API requests/responses to stderr |

### Error cases

| Input | Result | Exit Code |
|---|---|---|
| No argument | `Error: requires a pipeline name or --interactive flag` | 2 |
| Ambiguous substring (`o`) | Listed candidates: Todo, Doing | 2 |
| Non-existent (`nonexistent`) | `Error: pipeline "nonexistent" not found` | 4 |

### Display

- Pipeline with issues: Shows detail header, metadata fields (ID, Description, Stage, Issues, Created), and issue table with ISSUE, TITLE, EST, ASSIGNEE, PRIORITY columns
- Pipeline with stale config: Shows "Stale after: N days" field
- Pipeline marked as default PR pipeline: Shows "Default PR pipeline: yes"
- Empty pipeline (Test): Shows metadata only, no ISSUES section
- Issue references use short form (`repo#number`) since no repo name conflicts exist

## Bug Found and Fixed

### Issue count inconsistency between metadata and footer

**Symptom:** The "Issues" field in the metadata section showed a different count than the "Showing X of Y" footer. For example, the Todo pipeline showed "Issues: 24" in metadata but "Showing 10 of 10 issue(s)" in the footer. The Doing pipeline showed "Issues: 2" but "Showing 1 of 1 issue(s)".

**Root cause:** The metadata field used `detail.Issues.TotalCount` from the `GetPipelineDetails` GraphQL query, which counts all issues in the pipeline (possibly including closed issues or PRs not returned by search). The footer used `totalCount` from the `searchIssuesByPipeline` query, which only returns currently searchable open issues.

**Fix:** Changed the metadata field to use `totalCount` from the search query instead of the pipeline detail query. This ensures the "Issues" count in the metadata is consistent with the actual issues that can be listed and shown to the user.

**File changed:** `cmd/pipeline.go` line 494: `detail.Issues.TotalCount` -> `totalCount`

## Test Suite

All existing tests pass after the fix. The mock data in tests had both counts set to the same value, so no test changes were needed.
