# zh epic show

View epic details: title, state, dates, child issues, assignees.

## Overview

This command displays detailed information about a single epic. It supports both:
1. **ZenhubEpic** (standalone) - Native ZenHub epics
2. **Epic** (legacy) - GitHub issues marked as epics

The primary query mechanism is the `node` interface, which can fetch any entity by its ZenHub ID.

## Primary Query: By ZenHub ID

Fetch a ZenhubEpic (standalone) by its ID:

```graphql
query GetZenhubEpic($id: ID!, $workspaceId: ID!) {
  node(id: $id) {
    ... on ZenhubEpic {
      id
      title
      body
      htmlBody
      state
      startOn
      endOn
      createdAt
      updatedAt
      estimate {
        value
      }
      creator {
        id
        name
        githubUser {
          login
          avatarUrl
        }
      }
      assignees(first: 50) {
        nodes {
          id
          name
          githubUser {
            login
            avatarUrl
          }
        }
      }
      labels(first: 50) {
        nodes {
          id
          name
          color
        }
      }
      childIssues(first: 100, workspaceId: $workspaceId) {
        totalCount
        nodes {
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
            ghId
          }
          pipelineIssue(workspaceId: $workspaceId) {
            pipeline {
              id
              name
            }
          }
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
      blockingItems(first: 20) {
        totalCount
        nodes {
          ... on Issue {
            id
            number
            title
            repository {
              name
              ownerName
            }
          }
          ... on ZenhubEpic {
            id
            title
          }
        }
      }
      blockedItems(first: 20) {
        totalCount
        nodes {
          ... on Issue {
            id
            number
            title
            repository {
              name
              ownerName
            }
          }
          ... on ZenhubEpic {
            id
            title
          }
        }
      }
      project {
        id
        name
      }
      comments(first: 50) {
        totalCount
        nodes {
          id
          body
          createdAt
          author {
            name
            githubUser {
              login
            }
          }
        }
      }
    }
  }
}
```

## Alternative Query: Legacy Epic by ID

Fetch a legacy Epic (issue-backed) by its ID:

```graphql
query GetLegacyEpic($id: ID!) {
  node(id: $id) {
    ... on Epic {
      id
      startOn
      endOn
      createdAt
      updatedAt
      issue {
        id
        number
        title
        body
        htmlBody
        state
        htmlUrl
        createdAt
        closedAt
        repository {
          id
          name
          ownerName
          ghId
        }
        user {
          login
          avatarUrl
        }
        assignees(first: 50) {
          nodes {
            login
            avatarUrl
          }
        }
        labels(first: 50) {
          nodes {
            id
            name
            color
          }
        }
        estimate {
          value
        }
      }
      childIssues(first: 100) {
        totalCount
        nodes {
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
            ghId
          }
        }
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
  }
}
```

## Finding Epic by Title or Substring

When the user specifies an epic by title or substring (rather than ID), we need to search across both epic types. Use the roadmap items query with text search:

```graphql
query FindEpicByTitle($workspaceId: ID!, $query: String!) {
  workspace(id: $workspaceId) {
    roadmap {
      items(first: 50, query: $query) {
        nodes {
          __typename
          ... on ZenhubEpic {
            id
            title
          }
          ... on Epic {
            id
            issue {
              title
              number
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
}
```

If exactly one match is found, proceed with the full show query. If multiple matches are found, prompt the user to be more specific or use `--interactive` mode.

## Finding Legacy Epic by GitHub Issue Reference

When the user specifies `owner/repo#number` format for a legacy epic:

```graphql
query GetIssueForEpic($repositoryGhId: Int!, $issueNumber: Int!) {
  issueByInfo(repositoryGhId: $repositoryGhId, issueNumber: $issueNumber) {
    id
    parentEpics(first: 1) {
      nodes {
        id
      }
    }
  }
}
```

Note: `parentEpics` is deprecated but may still work. Alternatively, check if the issue itself is an epic by looking for child issues.

## Suggested CLI Flags

| Flag | Description |
|------|-------------|
| `--comments` | Include comments in output (default: false, to reduce noise) |
| `--children` | Show full child issue details (default: summary only) |
| `--limit=<n>` | Limit number of child issues shown (default: all) |
| `--json` | Output as JSON |

## Caching Requirements

| Data | Purpose |
|------|---------|
| Workspace ID | Required for `childIssues` query (ZenhubEpic requires workspaceId for this field) |
| Repository ghId mappings | To resolve `owner/repo#number` format for legacy epics |
| Epic aliases | For user-defined shorthand names |

## GitHub API Requirements

None strictly required. All data is available from ZenHub's API.

However, GitHub API could supplement:
- Full issue body/comments for legacy epics (if ZenHub cache is stale)
- Additional issue metadata not exposed by ZenHub
- Issue timeline/events

## Limitations

1. **Comments on legacy epics**: The `Epic` type does not have a `comments` field. Comments on legacy epics live on the GitHub issue, accessible via `issue.comments` on the Issue type, but this is GitHub issue comments, not ZenHub-specific comments.

2. **Dependencies for legacy epics**: The `Epic` type does not have `blockingItems` or `blockedItems` fields. Dependencies would need to be queried through the linked issue's fields.

3. **Activity feed**: Both `ZenhubEpic` and `Issue` have an `activityFeed` field that could show pipeline movements, estimate changes, etc. This could be valuable but adds complexity.

4. **workspaceId required for childIssues**: The `ZenhubEpic.childIssues` field requires a `workspaceId` parameter, which must be cached or passed.

5. **No direct lookup by title**: Must search and filter client-side for title/substring matching.

## Related Capabilities

The ZenhubEpic type exposes additional fields that could enhance the show command or support related features:

- **`activityFeed`**: History of changes to the epic (pipeline moves, estimate changes, etc.)
- **`keyDates`**: Milestone dates associated with the epic
- **`relatedItems`**: Dependencies marked as "related" (not blocking)
- **`oldIssue`**: For epics migrated from legacy format, links to the original GitHub issue
- **`zenhubOrganization`**: The organization owning this epic

## Output Example

```
EPIC: Q1 Platform Improvements
══════════════════════════════════════════════════════════════════════════════

Type:        ZenHub Epic
ID:          Z2lkOi8vcmFwdG9yL1plbmh1YkVwaWMvMTIzNDU
State:       in_progress
Estimate:    34 points

Dates:       2024-01-01 → 2024-03-31 (90 days)
Created:     2023-12-15 by @johndoe
Updated:     2024-02-10

Project:     Platform Team Q1

Assignees:   @johndoe, @janedoe, @bobsmith

Labels:      platform (blue), priority:high (red)

PROGRESS
────────────────────────────────────────────────────────────────────────────────
Issues:      12/20 closed (60%)  ████████████░░░░░░░░
Estimates:   34/55 points (62%)  ████████████░░░░░░░░

CHILD ISSUES (20)
────────────────────────────────────────────────────────────────────────────────
STATE     REPO           #      TITLE                                    EST
closed    mpt            #1234  Implement new auth flow                  5
closed    mpt            #1235  Update user permissions model            3
open      api            #567   Add rate limiting to endpoints           8
open      dashboard      #89    Dashboard performance improvements       5
...

BLOCKING (2)
────────────────────────────────────────────────────────────────────────────────
- mpt#1456: Database migration for user schema
- Epic: Q4 Infrastructure Cleanup

BLOCKED BY (1)
────────────────────────────────────────────────────────────────────────────────
- api#234: API versioning strategy

DESCRIPTION
────────────────────────────────────────────────────────────────────────────────
This epic covers all platform improvements planned for Q1 2024, including:
- Authentication system overhaul
- Performance optimizations
- API rate limiting implementation
...
```
