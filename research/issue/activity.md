# zh issue activity

## Overview

Show ZenHub activity feed for an issue (pipeline moves, estimate changes, priority changes, PR connections, etc.). Optionally merge in GitHub timeline events (labels, assignments, comments, close/reopen events).

## ZenHub API

### `timelineItems` field on Issue

The `Issue` type has a `timelineItems` connection that returns ZenHub-specific activity:

```graphql
query GetIssueTimeline($repositoryGhId: Int!, $issueNumber: Int!, $first: Int!, $after: String) {
  issueByInfo(repositoryGhId: $repositoryGhId, issueNumber: $issueNumber) {
    id
    number
    title
    repository {
      name
      owner { login }
    }
    timelineItems(first: $first, after: $after) {
      totalCount
      pageInfo {
        hasNextPage
        endCursor
      }
      nodes {
        id
        key
        data
        createdAt
      }
    }
  }
}
```

Each `TimelineItem` has:
- `id`: Node ID
- `key`: Event type string (e.g. `issue.set_estimate`, `issue.connect_issue_to_pr`)
- `data`: JSON blob with event-specific details
- `createdAt`: ISO 8601 timestamp

### Observed event keys and their `data` shapes

#### `issue.set_estimate`
```json
{
  "github_user": { "id": 123, "gh_id": 456, "login": "user", "avatar_url": "..." },
  "current_value": "5.0"
}
```
When clearing: has `previous_value` instead of `current_value`.

#### `issue.set_priority`
```json
{
  "priority": { "id": 123, "name": "High priority", "color": "...", "description": null },
  "workspace": { "id": 123, "name": "Dev Test", "mongo_id": "..." },
  "repository": { "id": 123, "name": "task-tracker", "gh_id": 123 },
  "github_user": { "id": 123, "gh_id": 456, "login": "user", "avatar_url": "..." },
  "organization": { "id": 123, "login": "org", "avatar_url": "..." }
}
```

#### `issue.remove_priority`
```json
{
  "workspace": { ... },
  "repository": { ... },
  "github_user": { ... },
  "organization": { ... },
  "previous_priority": { "id": 123, "name": "High priority", "color": "...", "description": null }
}
```

#### `issue.connect_issue_to_pr`
```json
{
  "automated": true,
  "repository": { "id": 123, "name": "task-tracker", "gh_id": 123 },
  "github_user": { "deleted": true },
  "organization": { "id": 123, "login": "org", "avatar_url": "..." },
  "pull_request": { "id": 123, "type": "GithubIssue", "state": "open", "title": "PR title", "number": 6 },
  "pull_request_repository": { "id": 123, "name": "task-tracker", "gh_id": 123 },
  "pull_request_organization": { "id": 123, "login": "org", "avatar_url": "..." }
}
```

#### Other expected keys (from ZenHub web UI patterns)
- `issue.transfer_pipeline` — pipeline moves
- `issue.add_to_sprint` / `issue.remove_from_sprint`
- `issue.add_to_epic` / `issue.remove_from_epic`
- `issue.assign` / `issue.unassign`

### `activityFeed` field on Issue

A UNION of `Comment | TimelineItem`. Includes both ZenHub comments and timeline items interleaved. Supports `skipTimelineItems` arg to filter out timeline items.

For `zh issue activity`, we use `timelineItems` directly since we want the ZenHub activity feed, not comments (which are GitHub-side).

## GitHub API (for `--github` flag)

The GitHub GraphQL `Issue.timelineItems` connection returns GitHub-side events:

```graphql
query GetGitHubTimeline($owner: String!, $repo: String!, $number: Int!, $first: Int!) {
  repository(owner: $owner, name: $repo) {
    issueOrPullRequest(number: $number) {
      ... on Issue {
        timelineItems(first: $first) {
          nodes {
            __typename
            ... on LabeledEvent { createdAt, label { name }, actor { login } }
            ... on UnlabeledEvent { createdAt, label { name }, actor { login } }
            ... on AssignedEvent { createdAt, assignee { ... on User { login } }, actor { login } }
            ... on UnassignedEvent { createdAt, assignee { ... on User { login } }, actor { login } }
            ... on ClosedEvent { createdAt, actor { login } }
            ... on ReopenedEvent { createdAt, actor { login } }
            ... on CrossReferencedEvent { createdAt, actor { login }, source { ... on Issue { number title } ... on PullRequest { number title } } }
            ... on IssueComment { createdAt, author { login }, body }
            ... on RenamedTitleEvent { createdAt, actor { login }, previousTitle, currentTitle }
            ... on MilestonedEvent { createdAt, actor { login }, milestoneTitle }
            ... on DemilestonedEvent { createdAt, actor { login }, milestoneTitle }
          }
        }
      }
    }
  }
}
```

## Implementation strategy

1. Fetch ZenHub `timelineItems` with pagination
2. Parse each item's `key` and `data` JSON to produce a human-readable description
3. If `--github` flag is set and GitHub client is available, also fetch GitHub timeline items
4. Merge both timelines chronologically by `createdAt`
5. Render as a chronological list with timestamps, actors, and descriptions

## Display format

```
ACTIVITY: task-tracker#1: Add task creation feature
════════════════════════════════════════════════════════════════════════════════

Feb 7, 2026   Connected PR task-tracker#6 "Add due date support"
Feb 10, 2026  @dlakehammond set estimate to 5
Feb 10, 2026  @dlakehammond cleared estimate
Feb 10, 2026  @dlakehammond set priority to "High priority"
Feb 10, 2026  @dlakehammond cleared priority

Total: 5 events
```

With `--github`:
```
Feb 7, 2026   @dlakehammond added label "enhancement"        [GitHub]
Feb 7, 2026   Connected PR task-tracker#6 "Add due date..."  [ZenHub]
Feb 10, 2026  @dlakehammond set estimate to 5                [ZenHub]
...
```
