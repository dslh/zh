# zh sprint velocity

Show velocity trends for recent sprints (points completed per sprint).

## Feasibility

**Fully Feasible** — ZenHub provides both a workspace-level average velocity field and per-sprint `completedPoints`/`totalPoints` data on every closed sprint. Together these are sufficient to build a complete velocity report.

## Primary Query

A single query fetches everything needed: the workspace average velocity (with trend diff), sprint cadence config, and per-sprint completion data for recent closed sprints.

```graphql
query SprintVelocity($workspaceId: ID!, $sprintCount: Int!) {
  workspace(id: $workspaceId) {
    displayName
    averageSprintVelocity
    averageSprintVelocityWithDiff(skipDiff: false) {
      velocity
      difference
      sprintsCount
    }
    sprintConfig {
      kind
      period
      startDay
      endDay
      tzIdentifier
    }
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
      sprintIssues(first: 0) {
        totalCount
      }
    }
    sprints(
      first: $sprintCount
      filters: { state: { eq: CLOSED } }
      orderBy: { field: END_AT, direction: DESC }
    ) {
      totalCount
      nodes {
        id
        name
        generatedName
        startAt
        endAt
        totalPoints
        completedPoints
        closedIssuesCount
        sprintIssues(first: 0) {
          totalCount
        }
      }
    }
  }
}
```

Variables:
```json
{
  "workspaceId": "<workspace_id>",
  "sprintCount": 6
}
```

### Key fields

| Field | Source | Description |
|---|---|---|
| `averageSprintVelocity` | `Workspace` | Average completed points across last 3 closed sprints (Float, nullable) |
| `averageSprintVelocityWithDiff` | `Workspace` | Richer velocity object with trend comparison |
| `.velocity` | `VelocityDiff` | Average velocity (Float!) |
| `.difference` | `VelocityDiff` | Change vs previous period (Float, null when `skipDiff: true`) |
| `.sprintsCount` | `VelocityDiff` | Number of sprints included in the calculation (Int!) |
| `completedPoints` | `Sprint` | Points completed in that sprint (Float!) |
| `totalPoints` | `Sprint` | Total points assigned to that sprint (Float!) |
| `closedIssuesCount` | `Sprint` | Number of issues closed (Int!) |
| `sprintIssues.totalCount` | `Sprint` | Total issues in the sprint (requires subquery with `first: 0`) |

### VelocityDiff behavior

- `skipDiff: false` — `difference` is a Float representing change from the previous averaging window. `sprintsCount` reflects the number of sprints in the primary window.
- `skipDiff: true` — `difference` is null. `sprintsCount` appears to include one additional sprint (the comparison sprint is counted but not diffed).

The `difference` field represents the change in velocity compared to the previous averaging window. A positive value means velocity is trending up, negative means it's trending down.

## Suggested CLI Flags

| Flag | Description | Default |
|---|---|---|
| `--sprints=<n>` | Number of recent closed sprints to include | 6 |
| `--include-active` | Include the current active sprint (in-progress) in the table | true |

The active sprint is included in the output by default (shown separately from closed sprints) since it provides useful context about work in progress vs the historical trend. It is excluded from average calculations.

## Caching Requirements

| Data | Cache file | Purpose |
|---|---|---|
| Workspace ID | config | Required for all queries |
| Sprint config | `sprints-{workspace_id}.json` | For displaying cadence info in the header |

Velocity data itself should not be cached — it changes every time an issue is closed or estimated.

## GitHub API Requirements

**None.** All velocity and sprint data is ZenHub-native.

## Limitations

### Average window is not configurable
`averageSprintVelocity` and `averageSprintVelocityWithDiff` calculate over ZenHub's fixed window (last 3 closed sprints). There's no parameter to change the window size. If a different window is desired, the CLI must compute it client-side from per-sprint `completedPoints`.

### No total issue count on the Sprint object itself
The Sprint type has `closedIssuesCount` but not a `totalIssuesCount` field. To get the total, you need `sprintIssues(first: 0) { totalCount }`, which adds a subquery per sprint. This is fine for the small number of sprints in a velocity report but worth noting.

### No burndown/burnup data
The API does not expose daily burndown or burnup chart data points. Velocity is limited to completed points per sprint — there's no day-by-day breakdown available.

### Assumed estimates
Workspaces can be configured with `assumeEstimates: true` and an `assumedEstimateValue` (e.g. 1). When enabled, unestimated issues count toward velocity with the assumed value. This is a workspace setting — the CLI should note it in the output when active, since it affects how velocity numbers should be interpreted.

## Adjacent API Capabilities

### Issue flow stats
The `Workspace.issueFlowStats` field provides cycle time metrics that complement velocity:

```graphql
{
  workspace(id: $workspaceId) {
    issueFlowStats(daysInCycle: 30) {
      avgCycleDays
      inDevelopmentDays
      inReviewDays
      anomalies {
        duration
        issue {
          id
          number
          title
        }
      }
    }
  }
}
```

| Field | Type | Description |
|---|---|---|
| `avgCycleDays` | `Int` | Average days from start to close |
| `inDevelopmentDays` | `Int` | Average days in development pipelines |
| `inReviewDays` | `Int` | Average days in review pipelines |
| `anomalies` | `[AnomalousIssue!]` | Issues with unusually long cycle times |

The `daysInCycle` parameter filters to issues closed within the last N days.

This data could naturally extend `zh sprint velocity` (e.g. `--cycle-time`) or power a separate `zh workspace stats` subcommand. It provides a complementary view: velocity measures throughput while cycle time measures speed.

## Output Example

```
VELOCITY — "Development"

Sprint cadence: 2-week (Sunday - Sunday)
Average velocity: 42.0 pts (last 3 sprints, trending +5.0)

Sprint assumed estimate: 1 pt (unestimated issues counted)

SPRINT                          DATES                   PTS DONE   PTS TOTAL   ISSUES   VELOCITY
────────────────────────────────────────────────────────────────────────────────────────────────────
▶ Sprint: Feb 8 - Feb 22        Feb 8 → Feb 22, 2026        18.0       52.0    5/15   (in progress)
  Sprint: Jan 22 - Feb 5        Jan 22 → Feb 5, 2026        48.0       48.0   15/15        48.0
  Sprint: Jan 8 - Jan 22        Jan 8 → Jan 22, 2026        38.0       42.0   12/14        38.0
  Sprint: Dec 25 - Jan 8        Dec 25 → Jan 8, 2026        40.0       40.0   13/13        40.0
  Sprint: Dec 11 - Dec 25       Dec 11 → Dec 25, 2025       45.0       45.0   14/14        45.0
  Sprint: Nov 27 - Dec 11       Nov 27 → Dec 11, 2025       35.0       38.0   11/13        35.0
  Sprint: Nov 13 - Nov 27       Nov 13 → Nov 27, 2025       42.0       42.0   14/14        42.0
                                                                                       ─────────
                                                                            avg (last 3):  42.0
```

## Implementation Notes

1. **Velocity per sprint** is simply `completedPoints`. The `totalPoints` and issue counts provide additional context about scope and completion rate.

2. **Completion rate** can be derived as `closedIssuesCount / sprintIssues.totalCount` (issues) or `completedPoints / totalPoints` (points).

3. **Trend arrow**: Use the `difference` from `averageSprintVelocityWithDiff` to show direction. Display as `+N` or `-N` next to the average.

4. **Active sprint**: Show it at the top of the table with a marker (e.g. `▶`) and "(in progress)" instead of a velocity number, since it's incomplete.

5. **Empty state**: If `sprintConfig` is null, display a message that sprints are not configured. If sprints exist but have zero points, the data is still valid — just show zeros.

6. **Assumed estimates notice**: When `workspace.assumeEstimates` is true, include a note in the header so users understand that velocity numbers may include assumed values.
