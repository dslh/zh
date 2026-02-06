# zh sprint list

List sprints in the workspace (active, upcoming, recent).

## Feasibility

**Fully Feasible** - The ZenHub GraphQL API provides comprehensive sprint data through the `workspace.sprints` connection, plus convenient accessors for `activeSprint`, `previousSprint`, and `upcomingSprint`.

## Primary Query

List all sprints with filtering and ordering:

```graphql
query ListSprints($workspaceId: ID!, $first: Int!, $after: String, $filters: SprintFiltersInput, $orderBy: SprintOrderInput) {
  workspace(id: $workspaceId) {
    sprints(first: $first, after: $after, filters: $filters, orderBy: $orderBy) {
      totalCount
      pageInfo {
        hasNextPage
        endCursor
      }
      nodes {
        id
        name
        generatedName
        description
        state
        startAt
        endAt
        totalPoints
        completedPoints
        closedIssuesCount
        createdAt
        updatedAt
      }
    }
    # Also fetch convenience accessors for quick reference
    activeSprint {
      id
      name
    }
    upcomingSprint {
      id
      name
    }
    previousSprint {
      id
      name
    }
    # Sprint configuration provides context about cadence
    sprintConfig {
      id
      name
      kind
      period
      startDay
      endDay
      tzIdentifier
    }
    # Velocity data for additional context
    averageSprintVelocity
  }
}
```

## Quick Sprint Access Query

For commands that just need the current/next/previous sprint without full listing:

```graphql
query QuickSprintAccess($workspaceId: ID!) {
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
    }
    upcomingSprint {
      id
      name
      generatedName
      state
      startAt
      endAt
    }
    previousSprint {
      id
      name
      generatedName
      state
      startAt
      endAt
      totalPoints
      completedPoints
      closedIssuesCount
    }
  }
}
```

## Filtering Options

### SprintFiltersInput

| Filter | Type | Description |
|--------|------|-------------|
| `state` | `SprintStateInput` | Filter by state: `{eq: OPEN}` or `{eq: CLOSED}` |
| `id` | `SprintIdInput` | Filter by sprint ID |

### SprintStateInput

The `eq` field accepts:
- `OPEN` - Active or upcoming sprints
- `CLOSED` - Completed sprints

### SprintOrderInput

| Field | Type | Description |
|-------|------|-------------|
| `field` | `SprintOrderField` | `START_AT` or `END_AT` |
| `direction` | `OrderDirection` | `ASC` or `DESC` |

## Sprint Fields Reference

| Field | Type | Description |
|-------|------|-------------|
| `id` | `ID!` | ZenHub sprint ID |
| `name` | `String` | Custom sprint name (nullable) |
| `generatedName` | `String` | Auto-generated name like "Sprint 42" |
| `description` | `String` | Sprint description/goal |
| `state` | `SprintState!` | `OPEN` or `CLOSED` |
| `startAt` | `ISO8601DateTime!` | Sprint start timestamp |
| `endAt` | `ISO8601DateTime!` | Sprint end timestamp |
| `totalPoints` | `Float!` | Sum of estimates for all issues in sprint |
| `completedPoints` | `Float!` | Sum of estimates for closed issues |
| `closedIssuesCount` | `Int!` | Number of closed issues |
| `createdAt` | `ISO8601DateTime!` | When the sprint was created |
| `updatedAt` | `ISO8601DateTime!` | Last update timestamp |
| `persisted` | `Boolean!` | Whether the sprint is persisted (vs preview) |

## Sprint Configuration

The `sprintConfig` provides context about the workspace's sprint cadence:

| Field | Type | Description |
|-------|------|-------------|
| `kind` | `SprintConfigKind!` | `weekly` or `monthly` |
| `period` | `Int!` | Sprint duration (e.g., 2 for 2-week sprints) |
| `startDay` | `SprintConfigDayOfTheWeek!` | Day sprints start |
| `endDay` | `SprintConfigDayOfTheWeek!` | Day sprints end |
| `tzIdentifier` | `String!` | Timezone (e.g., "America/New_York") |
| `name` | `String!` | Config name pattern |

## Suggested CLI Flags

| Flag | Description | API Mapping |
|------|-------------|-------------|
| `--state=<open\|closed\|all>` | Filter by state (default: all recent) | `filters.state.eq` |
| `--active` | Show only the active sprint | Use `activeSprint` accessor |
| `--upcoming` | Include upcoming/future sprints | `filters.state.eq: OPEN` + date filter |
| `--recent=<n>` | Show last N closed sprints (default: 3) | `filters.state.eq: CLOSED`, `first: n` |
| `--sort=<start\|end>` | Sort by start or end date | `orderBy.field` |
| `--order=<asc\|desc>` | Sort direction (default: desc) | `orderBy.direction` |
| `--limit=<n>` | Maximum sprints to return | `first` parameter |

### Suggested Default Behavior

Without flags, `zh sprint list` should show:
1. The active sprint (if any)
2. The upcoming sprint (if any)
3. The 3 most recent closed sprints

This provides a useful overview without overwhelming output.

## Caching Requirements

| Data | Purpose |
|------|---------|
| Workspace ID | Required for query |
| Sprint config | Optional - for displaying cadence info |

Sprint data itself should **not** be cached long-term as sprints change state frequently. However, a short TTL cache (e.g., 5 minutes) could improve performance for repeated commands.

For sprint name-to-ID resolution (needed by other commands like `zh sprint show <name>`):
- `sprints-{workspace_id}.json` - Sprint ID, name, generatedName, state mappings

## GitHub API Requirements

**None** - All sprint data is ZenHub-native. GitHub has no concept of sprints.

## Limitations

### No Text Search
The `sprints` connection supports a `query` parameter, but it's unclear what it searches. Sprint names are typically auto-generated (e.g., "Sprint 42"), so search is less useful.

### Limited State Values
Only `OPEN` and `CLOSED` states exist. There's no distinction between:
- Active (currently in progress)
- Upcoming (scheduled but not started)
- Planned (future but not scheduled)

The distinction between active and upcoming must be inferred from dates and the `activeSprint`/`upcomingSprint` accessors.

### No Sprint Creation via API
While `createSprintConfig` exists to set up recurring sprints, individual sprints cannot be manually created. Sprints are auto-generated based on the configuration.

### No Issue Count (Total)
The API provides `closedIssuesCount` but not total issue count. To get total issues, you'd need to query `sprint.issues.totalCount` for each sprint, which adds N+1 queries.

## Related/Adjacent API Capabilities

### Sprint Issues
Each sprint has an `issues` connection and a `sprintIssues` connection that can be queried for sprint contents:

```graphql
query SprintIssues($sprintId: ID!, $first: Int!) {
  node(id: $sprintId) {
    ... on Sprint {
      issues(first: $first) {
        totalCount
        nodes {
          id
          number
          title
          state
          estimate { value }
        }
      }
    }
  }
}
```

This would support `zh sprint show <sprint>`.

### Sprint Review
Sprints can have an associated `sprintReview` for retrospective data:

```graphql
{
  sprint {
    sprintReview {
      # Review data if configured
    }
  }
}
```

### Scope Change Tracking
The `scopeChange` field tracks how sprint scope changed over time:

```graphql
{
  sprint {
    scopeChange(first: 100) {
      nodes {
        # Scope change events
      }
    }
  }
}
```

### Velocity Tracking
The workspace provides velocity data:
- `averageSprintVelocity` - Average velocity of last 3 closed sprints
- `averageSprintVelocityWithDiff` - Velocity with trend comparison

This could support a `zh sprint velocity` or `zh sprint stats` subcommand.

## Output Example

```
SPRINTS IN WORKSPACE "Development"

Sprint configuration: 2-week sprints (Monday - Friday)
Average velocity: 42 points

STATE    NAME                  DATES                      POINTS     CLOSED
─────────────────────────────────────────────────────────────────────────────
active   Sprint 47             Jan 20, 2025 - Feb 2       34/52      8 issues
upcoming Sprint 48             Feb 3, 2025 - Feb 16       12/12      0 issues
closed   Sprint 46             Jan 6, 2025 - Jan 19       48/48      15 issues
closed   Sprint 45             Dec 23, 2024 - Jan 5       38/42      12 issues
closed   Sprint 44             Dec 9, 2024 - Dec 22       45/45      14 issues

Total: 5 sprints shown (1 active, 1 upcoming, 3 closed)
```

## Implementation Notes

1. **Display name logic**: Use `name` if set, otherwise fall back to `generatedName`

2. **Active sprint detection**: Use the `activeSprint` accessor rather than filtering by state and dates

3. **Progress calculation**: `completedPoints / totalPoints` gives completion percentage

4. **Date formatting**: Convert ISO8601 timestamps to user-friendly format based on locale

5. **Empty state**: If `sprintConfig` is null, the workspace has no sprints configured. Display a helpful message suggesting sprint setup in the ZenHub UI.
