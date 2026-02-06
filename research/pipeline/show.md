# zh pipeline show

View details about a pipeline and the issues in it.

## Feasibility

**Fully Feasible** - All required data is available through the ZenHub GraphQL API.

## API Queries

### Step 1: Resolve Pipeline ID

If the pipeline is specified by name (or substring), first resolve to an ID using the cached pipeline list or fetch via:

```graphql
query GetPipelines($workspaceId: ID!) {
  workspace(id: $workspaceId) {
    pipelinesConnection(first: 50) {
      nodes {
        id
        name
      }
    }
  }
}
```

### Step 2: Fetch Pipeline Details

Once the pipeline ID is known, fetch full details:

```graphql
query GetPipelineDetails($pipelineId: ID!) {
  node(id: $pipelineId) {
    ... on Pipeline {
      id
      name
      description
      stage
      isDefaultPRPipeline
      createdAt
      updatedAt
      pipelineConfiguration {
        showAgeInPipeline
        staleIssues
        staleInterval
        wipLimits {
          nodes {
            blockPipeline
            limitValue
          }
        }
      }
      issues {
        totalCount
      }
    }
  }
}
```

### Step 3: Fetch Pipeline Issues

Use `searchIssuesByPipeline` for rich issue data with filtering and ordering:

```graphql
query GetPipelineIssues(
  $pipelineId: ID!
  $workspaceId: ID!
  $first: Int
  $after: String
  $filters: IssueSearchFiltersInput!
  $order: IssueOrderInput
) {
  searchIssuesByPipeline(
    pipelineId: $pipelineId
    filters: $filters
    first: $first
    after: $after
    order: $order
  ) {
    totalCount
    pageInfo {
      hasNextPage
      endCursor
    }
    nodes {
      id
      number
      title
      state
      pullRequest
      htmlUrl
      estimate {
        value
      }
      assignees {
        nodes {
          login
        }
      }
      labels {
        nodes {
          name
          color
        }
      }
      repository {
        name
        ownerName
        ghId
      }
      blockingIssues {
        totalCount
      }
      blockedIssues {
        totalCount
      }
      connectedPrs {
        totalCount
      }
      sprints {
        nodes {
          name
        }
      }
      pipelineIssue(workspaceId: $workspaceId) {
        priority {
          name
          color
        }
        latestTransferTime
      }
      createdAt
      updatedAt
    }
  }
}
```

**Note:** The `pipelineIssue` field requires `workspaceId` as an argument to get workspace-specific data like priority and transfer time.

## Alternative: Direct Pipeline Issues Query

For simpler use cases without filtering, issues can be fetched directly from the Pipeline node:

```graphql
query GetPipelineWithIssues($pipelineId: ID!, $first: Int, $after: String) {
  node(id: $pipelineId) {
    ... on Pipeline {
      id
      name
      description
      stage
      issues(first: $first, after: $after) {
        totalCount
        pageInfo {
          hasNextPage
          endCursor
        }
        nodes {
          id
          number
          title
          state
          # ... other fields
        }
      }
    }
  }
}
```

The `issues` field on Pipeline accepts:
- `first`, `last`, `before`, `after` - Pagination
- `repositoryId` - Filter to a specific repository
- `state` - Filter by issue state (OPEN, CLOSED)

However, `searchIssuesByPipeline` is preferred because it supports richer filtering and ordering.

## Issue Fields Available

| Field | Type | Description |
|-------|------|-------------|
| `id` | ID! | ZenHub issue ID |
| `number` | Int! | GitHub issue number |
| `title` | String! | Issue title |
| `state` | IssueState! | OPEN or CLOSED |
| `pullRequest` | Boolean! | Whether this is a PR |
| `htmlUrl` | String! | GitHub URL |
| `estimate.value` | Float | Story point estimate |
| `assignees` | UserConnection! | Assigned users |
| `labels` | LabelConnection! | GitHub labels |
| `repository` | Repository! | Parent repository |
| `blockingIssues` | IssueConnection! | Issues this blocks |
| `blockedIssues` | IssueConnection! | Issues blocking this |
| `connectedPrs` | IssueConnection! | Connected pull requests |
| `sprints` | SprintConnection! | Sprints containing this issue |
| `pipelineIssue.priority` | Priority | ZenHub priority (requires workspaceId) |
| `pipelineIssue.latestTransferTime` | DateTime | When issue entered current pipeline |
| `createdAt` | DateTime! | Issue creation time |
| `updatedAt` | DateTime! | Last update time |

## Filter Options (IssueSearchFiltersInput)

| Filter | Type | Description |
|--------|------|-------------|
| `repositoryIds` | [ID!] | Filter to specific repositories |
| `labels` | StringInput | Filter by label names |
| `assignees` | IssueUserLoginInput | Filter by assignee login |
| `sprints` | SprintIdInput | Filter by sprint |
| `releases` | IdInput | Filter by release |
| `milestones` | StringInput | Filter by milestone |
| `estimates` | EstimateSearchFiltersInput | Filter by estimate |
| `zenhubEpics` | ZenhubEpicSearchFiltersInput | Filter by epic |
| `displayType` | DisplayFilter | Show/hide PRs |
| `matchType` | MatchingFilter | Match all or any filters |

## Ordering Options (IssueOrderInput)

| Field | Description |
|-------|-------------|
| `assignees` | Order by assignee logins |
| `created_at` | Order by creation date |
| `updated_at` | Order by last update |
| `gh_updated_at` | Order by GitHub update time |
| `sprints` | Order by sprint dates |
| `title` | Order alphabetically by title |
| `estimate` | Order by story point estimate |
| `stale` | Order stale issues first/last |
| `time_in_pipeline` | Order by time spent in pipeline |

Direction: `ASC` or `DESC`

## Caching Requirements

The following data should be cached for pipeline resolution:

- **Pipeline ID by name** - From `pipelines-{workspace_id}.json`
- **Repository GH IDs** - From `repos-{workspace_id}.json` for `--repo` filter

No additional caching needed beyond what `zh pipeline list` already provides.

## Suggested Flags

| Flag | Description |
|------|-------------|
| `--limit=<n>` | Limit number of issues shown (default: all) |
| `--order=<field>` | Sort by: position (default), estimate, created, updated, title, time_in_pipeline |
| `--desc` / `--asc` | Sort direction (default depends on field) |
| `--assignee=<user>` | Filter by assignee |
| `--label=<label>` | Filter by label |
| `--repo=<repo>` | Filter to a specific repository |
| `--sprint=<sprint>` | Filter by sprint |
| `--epic=<epic>` | Filter by epic |
| `--no-prs` | Hide pull requests |
| `--prs-only` | Show only pull requests |
| `--blocked` | Show only blocked issues |
| `--no-estimate` | Show only issues without estimates |
| `--output=json` | Output in JSON format |
| `--verbose` | Include additional details (pipeline config, PR review status) |

## Default Output Format

```
# Next Up

Sprint backlog pipeline with 15 issues.

**Configuration:**
- Stale after: 15 days
- Default for PRs: No

| # | Issue | Est | Assignee | Labels | Blocked |
|---|-------|-----|----------|--------|---------|
| 1 | api#3225 Expose supplier attribute... | 1 | - | - | - |
| 2 | posting#1408 Publish Indeed posting... | 2 | - | - | Yes |
| 3 | job_application#145 Handle job applications... | 2 | - | - | - |
```

With `--verbose`, include additional columns like sprint, time in pipeline, and connected PRs.

## GitHub API

Not strictly needed. However, GitHub's API could supplement:

- **PR review status** - ZenHub has `pullRequestReviews` but GitHub provides more detail
- **PR merge status** - Whether a PR is mergeable, has conflicts
- **Recent activity** - Comments, commits since last review

These could be fetched on-demand for `--verbose` output or when showing a single PR.

## Limitations

1. **Issue body not searchable** - The `query` parameter in `searchIssuesByPipeline` searches titles only, not issue bodies
2. **No "time in pipeline" filter** - Can order by `time_in_pipeline` but cannot filter (e.g., "issues in pipeline > 7 days")
3. **Position not directly exposed** - Issue position within the pipeline is implicit (via `itemBefore`/`itemAfter` on PipelineIssue), not a numeric value

## Related Subcommands

- **`zh pipeline list`** - Lists pipelines; provides ID resolution
- **`zh board --pipeline=<name>`** - Similar but in context of full board
- **`zh issue list --pipeline=<name>`** - Alternative interface for the same data
- **`zh issue show <issue>`** - Detailed view of a single issue

## Adjacent API Capabilities

### Pipeline Automations

The pipeline's `pipelineConfiguration.pipelineAutomations` field exposes automation rules. A future `zh pipeline automations <name>` subcommand could display:
- Auto-assign labels when issues enter the pipeline
- Auto-assign to sprints
- Auto-move to other pipelines

### Pipeline-to-Pipeline Automations

The `pipelineToPipelineAutomationSources` and `pipelineToPipelineAutomationDestinations` fields reveal cross-pipeline automation rules (e.g., "when issue enters Pipeline A, also move linked issues to Pipeline B").

### Issue Age / Staleness

The `enteredStageAt(pipelineStage: PipelineStage)` field on Issue can return when an issue entered a specific workflow stage. Combined with `staleInterval` from pipeline configuration, this could power a `--stale` flag to highlight stale issues.
