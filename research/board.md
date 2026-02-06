# zh board

Display the workspace board - pipelines and their issues.

## API Queries

### Primary Query: Get Full Board

Fetch all pipelines and their issues in a single query:

```graphql
query GetBoard($workspaceId: ID!) {
  workspace(id: $workspaceId) {
    id
    displayName
    pipelinesConnection(first: 50) {
      nodes {
        id
        name
        description
        issues(first: 100) {
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
            estimate { value }
            repository {
              name
              ownerName
            }
            assignees(first: 5) {
              nodes { login }
            }
            labels(first: 10) {
              nodes {
                name
                color
              }
            }
            pipelineIssue(workspaceId: $workspaceId) {
              priority {
                name
                color
              }
              relativePosition
            }
            blockingIssues(first: 1) {
              totalCount
            }
          }
        }
      }
    }
  }
}
```

### Alternative: Query Single Pipeline with Filters

For `--pipeline` flag or filtered views, use `searchIssuesByPipeline`:

```graphql
query SearchPipelineIssues(
  $pipelineId: ID!
  $filters: IssueSearchFiltersInput!
  $first: Int
  $after: String
) {
  searchIssuesByPipeline(
    pipelineId: $pipelineId
    filters: $filters
    first: $first
    after: $after
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
      estimate { value }
      repository {
        name
        ownerName
      }
      assignees(first: 5) {
        nodes { login }
      }
      labels(first: 10) {
        nodes { name color }
      }
    }
  }
}
```

## Filter Options (IssueSearchFiltersInput)

The following filters are available for `searchIssuesByPipeline`:

| Filter | Type | Description |
|--------|------|-------------|
| `repositoryIds` | `[ID!]` | Filter by repository |
| `labels` | `StringInput` | Filter by label names (`in`, `notIn`) |
| `assignees` | `IssueUserLoginInput` | Filter by assignee login |
| `assigneeIds` | `IssueUserIdInput` | Filter by assignee ID |
| `users` | `IssueUserLoginInput` | Filter by issue creator login |
| `sprints` | `SprintIdInput` | Filter by sprint |
| `releases` | `IdInput` | Filter by release |
| `milestones` | `StringInput` | Filter by milestone title |
| `estimates` | `EstimateSearchFiltersInput` | Filter by estimate value |
| `zenhubEpics` | `ZenhubEpicSearchFiltersInput` | Filter by epic |
| `displayType` | `DisplayFilter` | `all`, `issues`, or `prs` |
| `matchType` | `MatchingFilter` | `all` (AND) or `any` (OR) |

## Ordering Options (IssueOrderInput)

Issues can be ordered by:

| Field | Description |
|-------|-------------|
| `assignees` | Order by assignee logins |
| `created_at` | Order by issue creation date |
| `updated_at` | Order by last update |
| `gh_updated_at` | Order by GitHub update timestamp |
| `sprints` | Order by sprint dates |
| `title` | Order by issue title |
| `estimate` | Order by estimate value |
| `stale` | Order by staleness |
| `time_in_pipeline` | Order by time spent in current pipeline |

Direction: `ASC` or `DESC`

## Caching Requirements

The following should be cached for efficient operation:

- **Workspace ID** - Required for all queries
- **Pipeline list** - ID, name, description for each pipeline (for `--pipeline` name resolution)
- **Repository mappings** - Repository ID to `owner/name` (for display and filtering)

## Suggested Flags

Based on API capabilities:

| Flag | Description |
|------|-------------|
| `--pipeline=<name>` | Show only the specified pipeline |
| `--assignee=<login>` | Filter issues by assignee |
| `--label=<name>` | Filter issues by label (can be repeated) |
| `--repo=<name>` | Filter issues by repository |
| `--sprint=<id>` | Filter issues by sprint |
| `--epic=<id>` | Filter issues by epic |
| `--estimate=<value>` | Filter by estimate value |
| `--no-estimate` | Show only issues without estimates |
| `--type=<issues\|prs\|all>` | Filter by issue type (default: all) |
| `--sort=<field>` | Sort by: created, updated, estimate, title, stale, time-in-pipeline |
| `--order=<asc\|desc>` | Sort direction (default: varies by field) |
| `--limit=<n>` | Maximum issues per pipeline |

## Limitations

### SavedView API Gap

**The `--view=<name>` flag cannot be fully implemented via the API.**

The `SavedView` type only exposes an `id` field - no name, no filter definitions. This means:
- Cannot list saved views with their names
- Cannot retrieve filter settings from a saved view
- Cannot apply saved view filters programmatically

**Workaround options:**
1. Store view definitions locally in the CLI config file
2. Use the saved view IDs if known, but this provides poor UX
3. Feature request to ZenHub for expanded SavedView API

### Position Ordering

Issues have a `relativePosition` field in `PipelineIssue` that indicates their order within the pipeline. However, the API does not provide a sort option to order by position - issues must be sorted client-side after fetching.

## Related Subcommands

The board query naturally supports these related commands:

- `zh pipeline list` - Can use the same `pipelinesConnection` query
- `zh pipeline show <name>` - Can use `searchIssuesByPipeline` for a single pipeline
- `zh issue list` - The filtering capabilities map directly to this command

## Additional Fields of Interest

The Issue type includes several fields not used in the basic board view that could enhance display:

| Field | Description |
|-------|-------------|
| `pullRequest` | Boolean indicating if this is a PR |
| `connectedPrs` | PRs connected to this issue |
| `blockingIssues` / `blockedIssues` | Dependency relationships |
| `sprints` | Sprints the issue belongs to |
| `parentZenhubEpics` | Parent epics |
| `enteredStageAt` | When issue entered current pipeline stage |
| `htmlUrl` | Link to GitHub |
| `zenhubUrl` | Link to ZenHub |
