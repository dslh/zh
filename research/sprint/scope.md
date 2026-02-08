# zh sprint scope

Show scope change history for a sprint — issues added and removed over the sprint's lifetime.

## Feasibility

**Fully Feasible** — The `Sprint.scopeChange` connection provides a complete, timestamped log of every issue added to or removed from a sprint. Each event includes the action (added/removed), the timestamp, the estimate at the time of the change, and the full issue object. This is everything needed to reconstruct a scope change timeline.

## Primary Query

Fetch scope change history for a sprint by its ID:

```graphql
query SprintScopeChange($sprintId: ID!, $first: Int!, $after: String) {
  node(id: $sprintId) {
    ... on Sprint {
      id
      name
      generatedName
      state
      startAt
      endAt
      totalPoints
      completedPoints
      closedIssuesCount
      scopeChange(first: $first, after: $after) {
        totalCount
        pageInfo {
          hasNextPage
          endCursor
        }
        nodes {
          action
          effectiveAt
          estimateValue
          issue {
            id
            number
            title
            state
            estimate {
              value
            }
            repository {
              name
              ownerName
            }
          }
        }
      }
    }
  }
}
```

Variables:
```json
{
  "sprintId": "<sprint_id>",
  "first": 100
}
```

### Default to active sprint

When no sprint identifier is provided, resolve the active sprint first, then fetch its scope changes:

```graphql
query ActiveSprintScopeChange($workspaceId: ID!, $first: Int!) {
  workspace(id: $workspaceId) {
    activeSprint {
      id
      name
      generatedName
      state
      startAt
      endAt
      totalPoints
      completedPoints
      closedIssuesCount
      scopeChange(first: $first) {
        totalCount
        pageInfo {
          hasNextPage
          endCursor
        }
        nodes {
          action
          effectiveAt
          estimateValue
          issue {
            id
            number
            title
            state
            estimate {
              value
            }
            repository {
              name
              ownerName
            }
          }
        }
      }
    }
  }
}
```

The same pattern applies for `previous`/`next` relative references using `previousSprint`/`upcomingSprint`.

## ScopeChange Fields

| Field | Type | Description |
|---|---|---|
| `action` | `BucketIssueHistoryAction!` | `ISSUE_ADDED` or `ISSUE_REMOVED` |
| `effectiveAt` | `ISO8601DateTime!` | When the change occurred |
| `estimateValue` | `Float` | The issue's estimate at the time of the change (nullable — unestimated issues will be null) |
| `issue` | `Issue!` | The full issue object |

The `ScopeChangeConnection` is a standard Relay connection with `nodes`, `edges`, `pageInfo`, and `totalCount`. It supports pagination via `first`/`after`/`last`/`before` but has no filtering or ordering arguments — events are returned in chronological order.

Note: The API description for this field says "batching is disabled", suggesting each `scopeChange` query is resolved independently and cannot be batched with other sprint queries. This shouldn't affect the CLI but is worth knowing for performance.

## Suggested CLI Flags

| Flag | Description |
|---|---|
| `--summary` | Show only the net summary (issues added, removed, net change in points) without the full event log |
| `--limit=<n>` | Limit number of scope change events shown |

### Sprint identifier

Supports all standard sprint identifiers:
- No argument or `current` — active sprint
- `next` — upcoming sprint
- `previous` or `last` — previous sprint
- Sprint name or substring
- Sprint ZenHub ID

## Caching Requirements

| Data | Cache file | Purpose |
|---|---|---|
| Workspace ID | config | Required for active/previous/upcoming sprint resolution |
| Sprint name/ID mappings | `sprints-{workspace_id}.json` | For resolving sprint by name or substring |

Scope change data itself should never be cached — it changes every time an issue is added to or removed from the sprint.

## GitHub API Requirements

**None.** All scope change data is ZenHub-native. The issue's `repository.name` and `repository.ownerName` fields from ZenHub are sufficient for displaying `owner/repo#number` format.

## Limitations

### No filtering or ordering on scopeChange

The `scopeChange` connection only accepts standard pagination arguments. There is no way to filter by action type (e.g., only additions) or by date range at the API level. Any such filtering must be done client-side.

### No actor information

Scope change events do not include who performed the action. If a user wants to know who added or removed an issue from the sprint, this information is not available from the ZenHub API. The issue's `activityFeed` might contain related events, but correlating them would be unreliable.

### estimateValue is a snapshot, not current

The `estimateValue` on a scope change event reflects the issue's estimate at the time of the change. The issue's current estimate (via `issue.estimate.value`) may differ if it was re-estimated after being added to the sprint. This is actually a feature for scope tracking — it shows the point impact at the time of the change — but it's worth noting because the two values may not match.

### No distinction between initial scope and mid-sprint changes

All additions show as `ISSUE_ADDED` regardless of whether the issue was part of the sprint's initial planning or was added mid-sprint. To distinguish initial scope from mid-sprint additions, the CLI would need to compare each event's `effectiveAt` against the sprint's `startAt`. Issues added before or at sprint start are initial scope; those added after are mid-sprint additions.

## Adjacent API Capabilities

### SprintIssue.createdAt for initial scope detection

The `sprintIssues` connection provides a `createdAt` field on each `SprintIssue` association record. Cross-referencing this with the sprint's `startAt` could help distinguish initial scope from later additions, complementing the `scopeChange` data.

### Sprint-level summary fields

The sprint itself provides `totalPoints`, `completedPoints`, and `closedIssuesCount` which can serve as a current-state summary alongside the scope change history. A scope change view could show these as a header, with the event log below.

### Potential related subcommand: `zh sprint burndown`

While the API doesn't provide daily burndown data points, the `scopeChange` events could be used to construct a rudimentary scope change chart (text-based). Each ISSUE_ADDED increases total scope by `estimateValue`; each ISSUE_REMOVED decreases it. Overlaying this with issue close events (from `sprintIssues` with `issue.state` and `issue.closedAt`) could approximate a burndown, though it would require additional queries and would not match ZenHub's own burndown chart exactly.

## Output Example

```
SCOPE CHANGES — Sprint: Feb 8 - Feb 22, 2026
══════════════════════════════════════════════════════════════════════════════

Dates:    Feb 8, 2026 → Feb 22, 2026
Points:   34/52 completed (65%)
Changes:  12 events (8 added, 4 removed)

EVENT LOG
──────────────────────────────────────────────────────────────────────────────
DATE          ACTION    PTS  REPO                  #     TITLE
──────────────────────────────────────────────────────────────────────────────
Feb 8         + added    5   task-tracker          #1    Add due dates to tasks
Feb 8         + added    3   task-tracker          #2    Fix date parsing bug
Feb 8         + added    8   recipe-book           #1    Support ingredient quantities
Feb 8         + added    3   recipe-book           #2    Search by tag doesn't match partial tags
Feb 10        + added    5   task-tracker          #3    Add priority levels
Feb 11        - removed  3   recipe-book           #2    Search by tag doesn't match partial tags
Feb 13        + added    2   task-tracker          #4    Questions about task format
Feb 14        + added    8   recipe-book           #3    Add import/export to JSON
Feb 15        - removed  5   task-tracker          #3    Add priority levels
Feb 18        + added    3   recipe-book           #4    Fix partial tag matching in search
Feb 19        - removed  8   recipe-book           #3    Add import/export to JSON
Feb 20        - removed  2   task-tracker          #4    Questions about task format

SUMMARY
──────────────────────────────────────────────────────────────────────────────
Initial scope (at sprint start):     4 issues, 19 pts
Added mid-sprint:                    4 issues, 18 pts
Removed mid-sprint:                  4 issues, 18 pts
Net scope change:                    0 issues,  0 pts
Current scope:                       4 issues, 19 pts
```

### Summary-only output (`--summary`)

```
SCOPE CHANGES — Sprint: Feb 8 - Feb 22, 2026

Initial scope:        4 issues, 19 pts
Added mid-sprint:     4 issues, 18 pts
Removed mid-sprint:   4 issues, 18 pts
Net change:           0 issues,  0 pts
Current scope:        4 issues, 19 pts
```

## Implementation Notes

1. **Chronological display**: Events appear to be returned in chronological order. Display them as-is; no client-side sorting should be needed.

2. **Initial vs mid-sprint**: Compare each event's `effectiveAt` to the sprint's `startAt`. Events where `effectiveAt <= startAt` (or within a small tolerance, e.g. same day) are initial scope; the rest are mid-sprint changes.

3. **Net points calculation**: Sum `estimateValue` for ISSUE_ADDED events and subtract the sum for ISSUE_REMOVED events. Use `estimateValue` (the snapshot value) rather than the issue's current estimate, since the snapshot reflects the actual point impact at the time.

4. **Null estimates**: When `estimateValue` is null, the issue was unestimated at the time of the change. Display as `-` or `0` and note that unestimated issues affect issue counts but not point totals.

5. **Pagination**: The `scopeChange` connection supports standard Relay pagination. For most sprints, 100 events should be sufficient in a single page, but pagination should be implemented for teams with very active scope changes.

6. **Empty state**: If `scopeChange.totalCount` is 0, the sprint has had no scope changes. This likely means issues have not been added to the sprint yet — display a helpful message.
