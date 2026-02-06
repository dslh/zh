# zh epic list

List epics in the workspace.

## Overview

ZenHub has two types of epics:
1. **ZenhubEpic** (standalone) - Native ZenHub epics with title, body, state, dates, assignees, labels, and child issues
2. **Epic** (legacy) - GitHub issues that have been marked as epics, with child issues attached

Both types appear as `RoadmapItem` on the workspace roadmap, which is the most practical way to query all epics in a workspace.

## Primary Query

Query the workspace roadmap to get all epics (both types):

```graphql
query ListEpics($workspaceId: ID!, $first: Int!, $after: String, $state: RoadmapItemStateFilterInput, $query: String) {
  workspace(id: $workspaceId) {
    roadmap {
      items(first: $first, after: $after, state: $state, query: $query) {
        totalCount
        pageInfo {
          hasNextPage
          endCursor
        }
        nodes {
          __typename
          ... on ZenhubEpic {
            id
            title
            state
            body
            startOn
            endOn
            createdAt
            updatedAt
            estimate {
              value
            }
            assignees(first: 10) {
              nodes {
                id
                name
                githubUser {
                  login
                }
              }
            }
            labels(first: 10) {
              nodes {
                id
                name
                color
              }
            }
            zenhubIssueCountProgress {
              open
              closed
              total
            }
            zenhubIssueEstimateProgress {
              open
              closed
              total
            }
            project {
              id
              name
            }
          }
          ... on Epic {
            id
            startOn
            endOn
            createdAt
            updatedAt
            issue {
              id
              title
              number
              state
              body
              htmlUrl
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
              labels(first: 10) {
                nodes {
                  id
                  name
                  color
                }
              }
            }
            childIssues(first: 1) {
              totalCount
            }
            issueCountProgress {
              open
              closed
              total
            }
            issueEstimateProgress {
              open
              closed
              total
            }
            project {
              id
              name
            }
          }
          ... on Project {
            id
            name
          }
        }
      }
    }
  }
}
```

## Alternative: ZenhubEpics Only

If only standalone ZenHub epics are needed (excludes legacy issue-based epics):

```graphql
query ListZenhubEpics($workspaceId: ID!, $first: Int!, $after: String, $filters: ZenhubEpicFiltersInput, $orderBy: ZenhubEpicOrderInput, $query: String) {
  workspace(id: $workspaceId) {
    zenhubEpics(first: $first, after: $after, filters: $filters, orderBy: $orderBy, query: $query) {
      totalCount
      pageInfo {
        hasNextPage
        endCursor
      }
      nodes {
        id
        title
        state
        body
        startOn
        endOn
        createdAt
        updatedAt
        estimate {
          value
        }
        assignees(first: 10) {
          nodes {
            id
            name
            githubUser {
              login
            }
          }
        }
        labels(first: 10) {
          nodes {
            id
            name
            color
          }
        }
        zenhubIssueCountProgress {
          open
          closed
          total
        }
        zenhubIssueEstimateProgress {
          open
          closed
          total
        }
        project {
          id
          name
        }
      }
    }
  }
}
```

## Filtering and Ordering

### Roadmap Items (both epic types)

| Parameter | Type | Description |
|-----------|------|-------------|
| `state` | `RoadmapItemStateFilterInput` | Filter by state: `{in: [OPEN, TODO, IN_PROGRESS, CLOSED]}` or `{nin: [...]}` |
| `query` | `String` | Text search on title/description |
| `startOn` | `ISO8601Date` | Filter by start date |
| `endOn` | `ISO8601Date` | Filter by end date |
| `order` | `RoadmapItemOrderInput` | Order by `start_on` or `end_on`, direction `ASC` or `DESC` |

### ZenhubEpics Only

| Parameter | Type | Description |
|-----------|------|-------------|
| `filters.state` | `ZenhubEpicStateFilterInput` | Filter by state: `{in: [OPEN, TODO, IN_PROGRESS, CLOSED]}` |
| `filters.labelIds` | `IdInput` | Filter by label IDs: `{in: [...]}` or `{nin: [...]}` |
| `filters.projectIds` | `IdInput` | Filter by project IDs |
| `filters.estimateValues` | `FloatInput` | Filter by estimate values |
| `filters.assigneeIds` | `IdInput` | Filter by assignee IDs |
| `filters.matchType` | `MatchingFilter` | `any` or `all` for multiple filter conditions |
| `orderBy.field` | `ZenhubEpicOrderField` | `CREATED_AT`, `UPDATED_AT`, `TITLE`, `START_ON`, `END_ON`, `STATE`, `ASSIGNEES` |
| `orderBy.direction` | `OrderDirection` | `ASC` or `DESC` |
| `query` | `String` | Text search |

## Epic States

Both epic types use the same state values:
- `OPEN` - Default state for new epics
- `TODO` - Planned but not started
- `IN_PROGRESS` - Currently being worked on
- `CLOSED` - Completed

## Suggested CLI Flags

| Flag | Description |
|------|-------------|
| `--state=<state>` | Filter by state (open, todo, in_progress, closed). Multiple allowed. |
| `--assignee=<user>` | Filter by assignee (GitHub login or ZenHub user ID) |
| `--label=<label>` | Filter by label name |
| `--project=<name>` | Filter by project name |
| `--search=<query>` | Text search in title/description |
| `--sort=<field>` | Sort by: created, updated, title, start, end, state (default: created) |
| `--order=<dir>` | Sort direction: asc, desc (default: desc) |
| `--limit=<n>` | Limit number of results |
| `--type=<type>` | Filter by epic type: zenhub, legacy, all (default: all) |
| `--include-closed` | Include closed epics (by default, may want to exclude) |

## Caching Requirements

To support filtering by human-readable names:

| Data | Purpose |
|------|---------|
| Workspace ID | Required for all queries |
| Project names → IDs | For `--project` flag |
| User logins → ZenHub user IDs | For `--assignee` flag (ZenHub epics use ZenhubUser IDs) |
| Label names → IDs | For `--label` flag |

## GitHub API Requirements

None strictly required. Legacy epics include the GitHub issue data inline.

However, GitHub API could supplement:
- More detailed issue metadata for legacy epics
- User avatar URLs (included in ZenHub response)

## Limitations

1. **No unified epic query**: Must use roadmap items to get both types, or zenhubEpics for just standalone epics. There's no single query that returns both types with the same filtering options.

2. **Roadmap filtering is limited**: The roadmap items query only supports state, date range, and text search. More advanced filtering (by assignee, label, project) is only available for ZenhubEpics.

3. **Legacy epic filtering**: Cannot filter legacy epics by assignee or labels at the ZenHub API level - would need client-side filtering after fetching.

4. **No estimate filter for roadmap**: Estimate filtering is only available for ZenhubEpics query.

## Related Subcommands

The roadmap query reveals related entities that could support additional subcommands:

- **Projects**: `RoadmapItem` includes `Project` type - could support `zh project list` for viewing ZenHub projects
- **Key Dates**: Roadmap has `keyDates` field - could support `zh roadmap key-dates` for milestone markers
- **Releases**: Workspace has `releases` field - could support `zh release list`

## Output Example

```
EPICS IN WORKSPACE "Development"

TYPE       STATE        TITLE                                    ISSUES    ESTIMATE   START        END
───────────────────────────────────────────────────────────────────────────────────────────────────────────
zenhub     in_progress  Q1 Platform Improvements                 12/20     34         2024-01-01   2024-03-31
legacy     closed       LinkedIn Onsite Apply (api#2846)         28/28     -          2025-02-06   2025-04-18
legacy     open         Typescript migration (mpt#2469)          0/13      -          -            -
zenhub     todo         Mobile App Phase 2                       0/8       21         2024-04-01   2024-06-30

Total: 4 epics (2 open, 1 in progress, 1 closed)
```
