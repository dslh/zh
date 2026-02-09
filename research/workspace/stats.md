# zh workspace stats

Show workspace metrics: velocity trends, issue counts, cycle time, and activity metrics.

## Feasibility

**Feasible with caveats** — ZenHub's API provides velocity data, cycle time metrics (`issueFlowStats`), per-pipeline issue/PR counts, and per-sprint completion data. Combined, these cover the "velocity trends, issue counts, activity metrics" described in the spec. However, there is no single "stats" endpoint — the data must be assembled from several workspace fields. Cycle time data (`issueFlowStats`) returns null for workspaces with no recently closed issues that have moved through pipeline stages, so it may be empty more often than expected.

## Primary Query

A single query fetches all stats data. This is a wide query but avoids round-trips.

```graphql
query WorkspaceStats($workspaceId: ID!, $sprintCount: Int!, $daysInCycle: Int!) {
  workspace(id: $workspaceId) {
    displayName

    # Velocity
    averageSprintVelocity
    averageSprintVelocityWithDiff(skipDiff: false) {
      velocity
      difference
      sprintsCount
    }

    # Estimate settings (affects velocity interpretation)
    assumeEstimates
    assumedEstimateValue
    hasEstimatedIssues

    # Cycle time / flow stats
    issueFlowStats(daysInCycle: $daysInCycle) {
      avgCycleDays
      inDevelopmentDays
      inReviewDays
      anomalies {
        duration
        issue {
          id
          number
          title
          repository {
            name
            ownerName
          }
        }
      }
    }

    # Board issue distribution (per-pipeline counts and estimates)
    pipelinesConnection(first: 50) {
      totalCount
      nodes {
        name
        stage
        issues(first: 0) {
          totalCount
          pipelineCounts {
            issuesCount
            pullRequestsCount
            sumEstimates
          }
        }
      }
    }

    # Closed pipeline (not included in pipelinesConnection)
    closedPipeline {
      issues(first: 0) {
        totalCount
        pipelineCounts {
          issuesCount
          pullRequestsCount
          sumEstimates
        }
      }
    }

    # Workspace-wide totals
    issues(first: 0) {
      totalCount
      pipelineCounts {
        issuesCount
        pullRequestsCount
        sumEstimates
      }
    }

    # Active sprint progress
    activeSprint {
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

    # Recent closed sprints for velocity trend
    sprints(
      first: $sprintCount
      filters: { state: { eq: CLOSED } }
      orderBy: { field: END_AT, direction: DESC }
    ) {
      totalCount
      nodes {
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

    # Sprint config (for cadence display)
    sprintConfig {
      kind
      period
      startDay
      endDay
    }

    # Counts for summary
    repositoriesConnection(first: 0) { totalCount }
    zenhubEpics(first: 0) { totalCount }
    releases(first: 0) { totalCount }
    prioritiesConnection(first: 0) { totalCount }
    issueDependencies(first: 0) { totalCount }
    pipelineToPipelineAutomations(first: 0) { totalCount }
  }
}
```

### Variables

```json
{
  "workspaceId": "<workspace_id>",
  "sprintCount": 6,
  "daysInCycle": 30
}
```

## Key API Fields

### Velocity (`Workspace`)

| Field | Type | Description |
|---|---|---|
| `averageSprintVelocity` | `Float?` | Average completed points across last 3 closed sprints |
| `averageSprintVelocityWithDiff.velocity` | `Float!` | Same average, in richer object |
| `averageSprintVelocityWithDiff.difference` | `Float?` | Change vs previous averaging window (positive = trending up) |
| `averageSprintVelocityWithDiff.sprintsCount` | `Int!` | Number of sprints in the calculation |

### Cycle Time (`IssueFlowStats`)

| Field | Type | Description |
|---|---|---|
| `avgCycleDays` | `Int?` | Average days from first non-backlog pipeline to close |
| `inDevelopmentDays` | `Int?` | Average days spent in DEVELOPMENT-stage pipelines |
| `inReviewDays` | `Int?` | Average days spent in REVIEW-stage pipelines |
| `anomalies` | `[AnomalousIssue!]?` | Issues with unusually long cycle times |

`anomalies[].duration` is an `Int!` representing days. All fields are nullable — they return null when no issues have completed a full cycle within the `daysInCycle` window.

### Issue Distribution (`PipelineCounts`)

Available on `IssueConnection` (per-pipeline or workspace-wide):

| Field | Type | Description |
|---|---|---|
| `issuesCount` | `Int!` | Number of issues (not PRs) |
| `pullRequestsCount` | `Int!` | Number of pull requests |
| `sumEstimates` | `Float!` | Total estimate points |
| `unfilteredIssueCount` | `Int?` | Count ignoring filters (available but may be null) |
| `unfilteredSumEstimates` | `Float?` | Estimates ignoring filters |

### Sprint Completion (`Sprint`)

| Field | Type | Description |
|---|---|---|
| `totalPoints` | `Float!` | Total points assigned to sprint |
| `completedPoints` | `Float!` | Points completed |
| `closedIssuesCount` | `Int!` | Issues closed during the sprint |
| `sprintIssues.totalCount` | `Int!` | Total issues in the sprint (requires subquery) |

### Entity Counts

Available via `totalCount` on connections with `first: 0`:

| Connection | What it counts |
|---|---|
| `repositoriesConnection` | Connected repositories |
| `zenhubEpics` | ZenHub epics |
| `releases` | Releases (OPEN + CLOSED) |
| `prioritiesConnection` | Defined priority levels |
| `issueDependencies` | Blocking relationships |
| `pipelineToPipelineAutomations` | Pipeline automations |

## Suggested Flags

| Flag | Description | Default |
|---|---|---|
| `--sprints=<n>` | Number of recent closed sprints for velocity trend | 6 |
| `--days=<n>` | Cycle time window in days (`daysInCycle` parameter) | 30 |
| `--no-velocity` | Skip velocity section | false |
| `--no-cycle-time` | Skip cycle time section | false |
| `--no-distribution` | Skip pipeline distribution section | false |
| `--anomalies` | Show cycle time anomalies (issues with unusually long cycles) | false |
| `--output=json` | Output as JSON | — |

## Caching Requirements

| Data | Purpose |
|---|---|
| Workspace ID | Required for the query (from config) |
| Pipeline names/IDs | For display in distribution table (likely already cached) |

Stats data itself should **not** be cached — it is inherently point-in-time.

## GitHub API Enrichment

GitHub can provide per-repo open/closed issue and PR counts, which ZenHub does not track:

```graphql
query RepoStats($owner: String!, $name: String!) {
  repository(owner: $owner, name: $name) {
    openIssues: issues(states: OPEN) { totalCount }
    closedIssues: issues(states: CLOSED) { totalCount }
    openPRs: pullRequests(states: OPEN) { totalCount }
    mergedPRs: pullRequests(states: MERGED) { totalCount }
  }
}
```

This requires one query per repo, so it should be opt-in via a `--github` flag. It would enable a "GitHub summary" section showing aggregate open/closed issue and PR counts across all workspace repos.

## Limitations

### Cycle time requires pipeline stage classification

`issueFlowStats` computes cycle time based on pipeline stages (DEVELOPMENT, REVIEW, etc.). If pipelines have no stage assigned, the data will be null even if issues are being moved between pipelines and closed. This is a workspace configuration issue, not an API limitation — but the CLI should note when cycle time data is unavailable and suggest checking pipeline stage configuration.

### No daily burndown data

The API does not expose daily burndown or burnup chart data points. Sprint progress is limited to current `completedPoints` / `totalPoints` snapshots — no historical progression.

### No activity feed at workspace level

There is no workspace-level activity stream or event log. "Activity metrics" in the spec's description cannot include a feed of recent actions. The closest proxy is per-issue timeline events, but there's no way to aggregate these workspace-wide.

### Velocity averaging window is fixed

`averageSprintVelocity` / `averageSprintVelocityWithDiff` uses a fixed 3-sprint window. For a different window, the CLI must compute the average client-side from per-sprint `completedPoints`.

### Issue counts are board-scoped

`pipelineCounts` on the workspace `issues` connection reflects issues on the board. Issues not on any pipeline (possible in some edge cases) may not be counted. The `closedPipeline` must be queried separately to include closed issues.

### No estimate distribution

There's no API field to get a histogram of estimates (e.g., "how many issues have estimate 3 vs 5 vs 8"). This would require fetching all issues and computing client-side, which is expensive for large workspaces.

## Adjacent API Capabilities

### Releases summary

The `releases` connection can be filtered by state and includes `issuesCount` per release. A stats command could show active releases and their progress:

```graphql
{
  workspace(id: $workspaceId) {
    releases(first: 10, state: { eq: OPEN }) {
      totalCount
      nodes {
        title
        state
        startOn
        endOn
        issuesCount
      }
    }
  }
}
```

### Dependency graph stats

`issueDependencies` returns all blocking relationships. While the primary query gets `totalCount`, the full connection includes `blockedIssue` and `blockingIssue` with their pipeline info — useful for a "bottleneck analysis" showing which pipelines have the most blocked issues.

### Potential related subcommands

| Command | Description |
|---|---|
| `zh workspace cycle-time` | Dedicated cycle time report with anomaly details and per-pipeline breakdown |
| `zh workspace releases` | List/manage releases in the workspace |
| `zh workspace dependencies` | Visualize or summarize the dependency graph |

## Output Example

```
WORKSPACE STATS — "Development"
══════════════════════════════════════════════════════════════════════════════════

SUMMARY
────────────────────────────────────────────────────────────────────────────────
Repositories:    2          Epics:          0          Automations:   0
Issues on board: 7          PRs on board:   3          Dependencies:  0
Total estimates: 0 pts      Priorities:     1 defined  Releases:      0

VELOCITY (2-week sprints)
────────────────────────────────────────────────────────────────────────────────
Average velocity: 42.0 pts/sprint (last 3 sprints, +5.0 trend)
Assumed estimates: 1 pt (unestimated issues counted as 1)

SPRINT                          DATES                  DONE    TOTAL   ISSUES
────────────────────────────────────────────────────────────────────────────────
▶ Sprint: Feb 8 - Feb 22       Feb 8 → Feb 22          18.0    52.0    5/15
  Sprint: Jan 22 - Feb 5       Jan 22 → Feb 5          48.0    48.0   15/15
  Sprint: Jan 8 - Jan 22       Jan 8 → Jan 22          38.0    42.0   12/14
  Sprint: Dec 25 - Jan 8       Dec 25 → Jan 8          40.0    40.0   13/13

CYCLE TIME (last 30 days)
────────────────────────────────────────────────────────────────────────────────
Average cycle:     12 days
  In development:   8 days
  In review:        4 days

PIPELINE DISTRIBUTION
────────────────────────────────────────────────────────────────────────────────
PIPELINE              STAGE          ISSUES   PRS   ESTIMATES
────────────────────────────────────────────────────────────────────────────────
New Issues            -                  24     2       12.0
Backlog               BACKLOG           45     0       89.0
In Development        DEVELOPMENT       12     3       28.0
Code Review           DEVELOPMENT        5     6       15.0
On Staging            REVIEW             3     2        8.0
Done                  COMPLETED        120     0      245.0
                                     ─────  ────    ──────
                                       209    13      397.0
```

## Implementation Notes

1. **Section ordering**: Summary first (quick overview), then velocity (most commonly wanted metric), then cycle time, then pipeline distribution. Each section can be independently toggled off.

2. **Empty states**: When sprints aren't configured, skip the velocity section and show "Sprints not configured" in the summary. When `issueFlowStats` returns all nulls, skip cycle time and note "No cycle time data available (issues may not have completed a full cycle recently, or pipeline stages may not be configured)."

3. **Closed pipeline**: The `closedPipeline` is a virtual pipeline with a fixed ID (`Z2lkOi8vcmFwdG9yL1BpcGVsaW5lLzA`) and is not included in `pipelinesConnection`. It must be queried separately. Include it in the distribution table if the user wants a complete picture.

4. **Velocity overlap with `zh sprint velocity`**: This command shows a condensed velocity summary. `zh sprint velocity` is the dedicated deep-dive. The stats command should focus on the headline number and trend, not the full sprint-by-sprint table. Consider showing only the last 3 sprints inline and directing users to `zh sprint velocity` for the full history.

5. **JSON output**: When `--output=json`, structure the output with top-level keys matching the sections: `summary`, `velocity`, `cycleTime`, `pipelineDistribution`. This makes it easy to pipe into `jq` for specific metrics.
