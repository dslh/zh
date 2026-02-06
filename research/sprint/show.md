# zh sprint show

View sprint details and issues. Defaults to the active sprint if no sprint identifier is provided.

## Feasibility

**Fully Feasible** - The ZenHub GraphQL API provides comprehensive sprint data. Sprints implement the `Node` interface, allowing direct lookup by ID. The `activeSprint`, `upcomingSprint`, and `previousSprint` accessors provide convenient access to relative sprints. Issues in a sprint can be fetched via the `sprintIssues` or `issues` connections.

## Primary Query: By Sprint ID

Fetch a sprint directly by its ZenHub ID using the `node` interface:

```graphql
query GetSprint($sprintId: ID!) {
  node(id: $sprintId) {
    ... on Sprint {
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
      persisted
      workspace {
        id
        displayName
      }
      sprintIssues(first: 100) {
        totalCount
        pageInfo {
          hasNextPage
          endCursor
        }
        nodes {
          id
          createdAt
          issue {
            id
            number
            title
            state
            htmlUrl
            estimate {
              value
            }
            repository {
              id
              name
              ownerName
              ghId
            }
            assignees(first: 10) {
              nodes {
                login
                avatarUrl
              }
            }
            labels(first: 20) {
              nodes {
                id
                name
                color
              }
            }
            pipelineIssues(first: 10) {
              nodes {
                pipeline {
                  id
                  name
                }
              }
            }
            blockingIssues(first: 5) {
              totalCount
            }
            blockedIssues(first: 5) {
              totalCount
            }
          }
        }
      }
      sprintReview {
        id
        title
        body
        state
        lastGeneratedAt
        manuallyEdited
      }
      scopeChange(first: 50) {
        totalCount
        nodes {
          action
          effectiveAt
          estimateValue
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
    }
  }
}
```

## Quick Access Query: Active/Previous/Upcoming Sprint

For `zh sprint show` without arguments (defaults to active), or with relative references like `current`, `next`, `previous`:

```graphql
query GetRelativeSprint($workspaceId: ID!) {
  workspace(id: $workspaceId) {
    activeSprint {
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
      sprintIssues(first: 100) {
        totalCount
        nodes {
          id
          createdAt
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
            assignees(first: 10) {
              nodes {
                login
              }
            }
            pipelineIssues(first: 1) {
              nodes {
                pipeline {
                  name
                }
              }
            }
          }
        }
      }
    }
    upcomingSprint {
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
      sprintIssues(first: 100) {
        totalCount
        nodes {
          id
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
            assignees(first: 10) {
              nodes {
                login
              }
            }
          }
        }
      }
    }
    previousSprint {
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
      sprintIssues(first: 100) {
        totalCount
        nodes {
          id
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
    averageSprintVelocity
  }
}
```

## Finding Sprint by Name or Substring

When the user specifies a sprint by name (e.g., "Sprint 42") or substring, search the sprints list:

```graphql
query FindSprintByName($workspaceId: ID!, $query: String!) {
  workspace(id: $workspaceId) {
    sprints(first: 50, query: $query) {
      nodes {
        id
        name
        generatedName
        state
        startAt
        endAt
      }
    }
  }
}
```

The `query` parameter performs text search. If multiple matches are found, prompt the user to be more specific or show available options.

## Sprint Issue Details

The `sprintIssues` field is preferred over `issues` for sprint contents. It provides:
- `SprintIssue.id` - The association record ID
- `SprintIssue.createdAt` - When the issue was added to the sprint
- `SprintIssue.issue` - The full issue object

This allows filtering by labels and provides information about when issues were added:

```graphql
query SprintIssuesWithLabels($sprintId: ID!, $labelIds: [ID!]) {
  node(id: $sprintId) {
    ... on Sprint {
      sprintIssues(first: 100, labelIds: $labelIds) {
        totalCount
        nodes {
          id
          createdAt
          issue {
            # ... issue fields
          }
        }
      }
    }
  }
}
```

## Sprint Review Data

Sprints may have an associated sprint review containing retrospective data:

```graphql
{
  sprint {
    sprintReview {
      id
      title
      body
      htmlBody
      state           # INITIAL, IN_PROGRESS, COMPLETED
      language
      lastGeneratedAt
      manuallyEdited
      initiatedBy {
        name
        githubUser {
          login
        }
      }
    }
  }
}
```

The `SprintReviewState` enum values are:
- `INITIAL` - Not yet started
- `IN_PROGRESS` - Being worked on
- `COMPLETED` - Finalized

## Scope Change Tracking

The `scopeChange` field tracks how sprint scope evolved over time:

```graphql
{
  sprint {
    scopeChange(first: 50) {
      totalCount
      nodes {
        action         # ISSUE_ADDED or ISSUE_REMOVED
        effectiveAt    # When the change happened
        estimateValue  # Estimate at time of change
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
  }
}
```

The `BucketIssueHistoryAction` enum values are:
- `ISSUE_ADDED` - Issue was added to the sprint
- `ISSUE_REMOVED` - Issue was removed from the sprint

## Suggested CLI Flags

| Flag | Description | Notes |
|------|-------------|-------|
| `--issues` | Show detailed issue list (default: summary) | Full issue table vs counts only |
| `--scope-changes` | Show scope change history | For sprint analysis |
| `--review` | Include sprint review content | If available |
| `--label=<label>` | Filter issues by label | Uses `sprintIssues(labelIds:)` |
| `--pipeline=<name>` | Filter issues by pipeline | Client-side filter |
| `--assignee=<user>` | Filter issues by assignee | Client-side filter |
| `--state=<open\|closed>` | Filter issues by state | Client-side filter |
| `--limit=<n>` | Limit number of issues shown | Pagination |
| `--json` | Output as JSON | For scripting |

### Relative Sprint References

The sprint identifier supports these special values:
- (no argument) or `current` - Active sprint
- `next` - Upcoming sprint
- `previous` or `last` - Previous sprint

## Caching Requirements

| Data | Purpose |
|------|---------|
| Workspace ID | Required for all queries |
| Sprint name → ID mappings | For name/substring lookup |
| Pipeline names → IDs | For `--pipeline` filtering |
| Repository ghId mappings | For displaying `owner/repo#number` format |

A `sprints-{workspace_id}.json` cache file should map sprint names/generatedNames to IDs for fast resolution.

## GitHub API Requirements

**None** - All sprint and issue data is available from ZenHub's API. GitHub has no sprint concept.

## Limitations

### No Total Issue Count on Sprint Object
The Sprint type provides `closedIssuesCount` but not total open issue count. To get totals, query `sprintIssues.totalCount` (requires a subquery).

### Limited Filtering on sprintIssues
The `sprintIssues` connection only supports `labelIds` filtering. Other filters (assignee, pipeline, state) must be applied client-side.

### No Burndown Data
The API does not expose daily burndown/burnup chart data. This would need to be calculated from `scopeChange` events, which is complex and potentially incomplete.

### Sprint Review Not Always Present
The `sprintReview` field may be null if the team hasn't enabled or used sprint reviews.

### Pipeline Context Requires Workspace
To show which pipeline an issue is in, you need to use `pipelineIssues(first: 1)` or `pipelineIssue(workspaceId: $workspaceId)`. The latter requires passing the workspace ID.

## Related/Adjacent Capabilities

### Sprint Velocity Comparison
The workspace provides velocity metrics that could enhance sprint show:

```graphql
{
  workspace(id: $workspaceId) {
    averageSprintVelocity
    averageSprintVelocityWithDiff(skipDiff: false) {
      # Velocity with trend
    }
  }
}
```

### Issue Activity Within Sprint
Each issue has an `activityFeed` showing its history during the sprint:

```graphql
{
  issue {
    activityFeed(first: 50) {
      nodes {
        # Activity events (pipeline moves, estimate changes, etc.)
      }
    }
  }
}
```

### Sprint Configuration Context
Show sprint cadence info alongside sprint details:

```graphql
{
  workspace(id: $workspaceId) {
    sprintConfig {
      kind           # weekly or monthly
      period         # e.g., 2 for 2-week sprints
      startDay
      endDay
      tzIdentifier
    }
  }
}
```

### Potential Related Commands
- `zh sprint stats` - Velocity trends, burn rate analysis
- `zh sprint scope` - Detailed scope change history
- `zh sprint review` - Sprint review/retrospective management

## Output Example

```
SPRINT: Sprint 47
══════════════════════════════════════════════════════════════════════════════

ID:          Z2lkOi8vcmFwdG9yL1NwcmludC8xMjM0NQ
State:       OPEN (active)
Dates:       Jan 20, 2025 → Feb 2, 2025 (14 days)

PROGRESS
────────────────────────────────────────────────────────────────────────────────
Points:      34/52 completed (65%)  █████████████░░░░░░░
Issues:      8 closed

Workspace velocity: 42 pts (avg last 3 sprints)

DESCRIPTION
────────────────────────────────────────────────────────────────────────────────
Focus on API performance improvements and bug fixes from customer feedback.

ISSUES (15)
────────────────────────────────────────────────────────────────────────────────
STATE   REPO       #      PIPELINE        EST  ASSIGNEE     TITLE
────────────────────────────────────────────────────────────────────────────────
closed  mpt        #1234  Done            5    @johndoe     Fix auth timeout
closed  mpt        #1235  Done            3    @janedoe     Update error messages
closed  api        #567   Done            8    @johndoe     Optimize query performance
open    api        #568   Code Review     5    @bobsmith    Add caching layer
open    dashboard  #89    In Progress     5    @janedoe     Dashboard loading states
open    mpt        #1240  In Progress     3    @johndoe     User preference sync
open    api        #570   To Do           8    (unassigned) Rate limiting implementation
...

────────────────────────────────────────────────────────────────────────────────
Total: 15 issues (8 closed, 7 open) | 52 points (34 completed)
```

### Compact Output (without --issues)

```
SPRINT: Sprint 47 (active)
══════════════════════════════════════════════════════════════════════════════

Dates:    Jan 20, 2025 → Feb 2, 2025 (14 days)
Points:   34/52 completed (65%)  █████████████░░░░░░░
Issues:   8/15 closed (53%)

Use --issues to see the full issue list.
```

## Implementation Notes

1. **Display name logic**: Use `name` if set, otherwise fall back to `generatedName`

2. **Progress bar**: Calculate `completedPoints / totalPoints` for visual progress indicator

3. **Issue ordering**: Consider ordering issues by:
   - Pipeline position (default) - shows workflow status
   - State (closed first or open first)
   - Estimate (highest first) - shows effort distribution

4. **Duration calculation**: `endAt - startAt` gives sprint duration. Display in days.

5. **Active sprint indicator**: When showing the active sprint, display "(active)" in state

6. **Empty sprint**: If a sprint has no issues, display a message suggesting adding issues via `zh sprint add`

7. **No active sprint**: If `activeSprint` is null and no identifier provided, suggest checking sprint configuration or using `zh sprint list`
