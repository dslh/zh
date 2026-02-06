# zh issue show

View detailed information about a single issue or PR.

## Feasibility

**Fully Feasible** - All required data is available through the ZenHub GraphQL API via the `issueByInfo` query. Some supplementary data (comments, reactions) requires GitHub API.

## API Query

### Primary Query: Get Issue Details

```graphql
query GetIssueDetails($repositoryGhId: Int!, $issueNumber: Int!, $workspaceId: ID!) {
  issueByInfo(repositoryGhId: $repositoryGhId, issueNumber: $issueNumber) {
    id
    number
    title
    body
    state
    pullRequest
    htmlUrl
    zenhubUrl(workspaceId: $workspaceId)
    createdAt
    closedAt
    ghCreatedAt

    # ZenHub-specific fields
    estimate {
      value
    }
    pipelineIssue(workspaceId: $workspaceId) {
      pipeline {
        id
        name
      }
      priority {
        id
        name
        color
      }
      latestTransferTime
    }

    # Assignees
    assignees(first: 20) {
      nodes {
        login
        name
        avatarUrl
      }
    }

    # Labels
    labels(first: 50) {
      nodes {
        id
        name
        color
      }
    }

    # Connected PRs (for issues)
    connectedPrs(first: 20) {
      nodes {
        id
        number
        title
        state
        htmlUrl
        pullRequest
        pullRequestObject {
          state
          draft
        }
        repository {
          name
          owner {
            login
          }
        }
      }
    }

    # Blocking relationships
    blockingIssues(first: 20) {
      nodes {
        id
        number
        title
        state
        repository {
          name
          owner {
            login
          }
        }
      }
    }
    blockedIssues(first: 20) {
      nodes {
        id
        number
        title
        state
        repository {
          name
          owner {
            login
          }
        }
      }
    }

    # Epic membership
    parentZenhubEpics(first: 10) {
      nodes {
        id
        title
        state
      }
    }

    # Sprint membership
    sprints(first: 5) {
      nodes {
        id
        name
        state
        startAt
        endAt
      }
    }

    # Repository info
    repository {
      id
      ghId
      name
      owner {
        login
      }
    }

    # Milestone (from GitHub)
    milestone {
      id
      title
      state
      dueOn
    }
  }
}
```

### Alternative: Fetch by ZenHub ID

If the user provides a ZenHub ID directly, use the `node` query:

```graphql
query GetIssueById($id: ID!, $workspaceId: ID!) {
  node(id: $id) {
    ... on Issue {
      id
      number
      title
      body
      state
      # ... same fields as above
    }
  }
}
```

## Caching Requirements

| Data | Purpose |
|------|---------|
| Repositories | Resolve `owner/repo` or `repo` to `repositoryGhId` |
| Workspace ID | Required for `pipelineIssue` and `zenhubUrl` fields |

No additional caching beyond what's already specified for other commands.

## Suggested Flags

| Flag | Description | Notes |
|------|-------------|-------|
| `--json` | Output raw JSON | Standard across all commands |
| `--web` | Open in browser | Opens `htmlUrl` (GitHub) or `zenhubUrl` (ZenHub) |
| `--zenhub` | Open ZenHub URL instead of GitHub | Use with `--web` |
| `--comments` | Include comments | Requires GitHub API |
| `--activity` | Show ZenHub activity feed | Uses `activityFeed` field |

### Issue Identifier Support

Per the spec, the command accepts:
- ZenHub ID: `Z2lkOi8vcmFwdG9yL0lzc3VlLzU2Nzg5`
- Full GitHub reference: `gohiring/mpt#1234`
- Short GitHub reference: `mpt#1234` (if repo name is unique in workspace)
- With `--repo` flag: `--repo=mpt 1234`

## Default Output Format

```markdown
# mpt#951: rename folders to satisfy Zeitwerk

**State:** Closed (PR merged)
**Pipeline:** On Staging
**Estimate:** -
**Priority:** -

**Assignees:** None
**Labels:** `daily hit`
**Sprint:** -
**Epic:** -

## Description

Unfortunately, Zeitwerk failed on the staging and would fail on the production
after eagerloading :( It didn't happen in the test and dev environments though.
To satisfy zeitwerk a few folders had to be renamed. The rule of thumb is to run:

    script/run bundle exec rake zeitwerk:check

to be sure it is happy.

## Links

- GitHub: https://github.com/gohiring/mpt/pull/951
- ZenHub: https://app.zenhub.com/workspaces/development-5c5c2662a623f9724788f533/issues/gh/gohiring/mpt/951

## Timeline

- Created: 2020-12-11 11:11:21
- Closed: 2020-12-15 18:36:45
- Time in pipeline: since 2020-12-15 18:06:30
```

### With Connected PRs (for issues)

```markdown
## Connected PRs

| PR | Status | Title |
|----|--------|-------|
| mpt#952 | Merged | Zeitwerk folder renames |
```

### With Blockers

```markdown
## Blocking

This issue is blocking:
- api#1234: Some dependent issue (Open)
- mpt#567: Another dependent issue (Closed)

## Blocked By

This issue is blocked by:
- api#999: Prerequisite work (Open)
```

## GitHub API Supplementation

The following data is not available in ZenHub's API and requires GitHub:

| Data | GitHub Field | Use Case |
|------|--------------|----------|
| Comments | `issue.comments` | `--comments` flag |
| Reactions | `issue.reactions` | Display reaction counts |
| Comment count | `issue.comments.totalCount` | Show in summary |
| PR review status | `pullRequest.reviews` | Show review state |
| PR checks status | `pullRequest.commits.nodes[0].commit.statusCheckRollup` | Show CI status |
| Author | `issue.author` | Show who created the issue |
| Participants | `issue.participants` | Show who's involved |

### GitHub Query for Comments

```graphql
query GetIssueComments($owner: String!, $repo: String!, $number: Int!) {
  repository(owner: $owner, name: $repo) {
    issue(number: $number) {
      author {
        login
      }
      comments(first: 50) {
        totalCount
        nodes {
          author {
            login
          }
          body
          createdAt
        }
      }
    }
  }
}
```

## Limitations

### No Comment Access via ZenHub
ZenHub's `Issue.comments` field exists but appears to be for ZenHub-specific comments, not GitHub issue comments. For GitHub comments, the GitHub API is required.

### Activity Feed is Separate
The `activityFeed` field provides ZenHub-specific activity (pipeline moves, estimate changes, etc.) but not GitHub activity. Consider a `--activity` flag to show this.

### PR-Specific Details Limited
For PRs, ZenHub provides basic state (`OPEN`, `CLOSED`, `MERGED`) and `draft` status via `pullRequestObject`, but not:
- Files changed
- Review requests
- Check status
- Merge conflicts

These require GitHub API.

## Related API Capabilities

### Activity Feed
The `Issue.activityFeed` field provides rich history:

```graphql
activityFeed(first: 50) {
  nodes {
    ... on PipelineMovedActivity {
      createdAt
      fromPipeline { name }
      toPipeline { name }
      user { login }
    }
    ... on EstimateChangedActivity {
      createdAt
      fromEstimate { value }
      toEstimate { value }
      user { login }
    }
    # ... other activity types
  }
}
```

This could support a future `zh issue activity <issue>` command or `--activity` flag.

### Estimation Votes
The `Issue.estimationVotes` field shows planning poker votes, which could be interesting for a `zh estimate` command group.

### Parent/Child Issues
The `Issue.parentIssue` and `Issue.zenhubChildIssues` / `Issue.githubChildIssues` fields support GitHub's sub-issue feature and ZenHub's hierarchy. Could support `--children` flag or tree view.

### Connected Issues
The `Issue.connections` field (separate from `connectedPrs`) shows related issues that aren't PRs. This could be exposed with a `--connections` flag.

### Review Requests
For PRs, `Issue.reviewRequests` shows requested reviewers. This could be shown in PR output.
