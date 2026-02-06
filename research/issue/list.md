# zh issue list

List issues in the workspace with filtering options.

## Feasibility

**Fully Feasible** - All required data is available through the ZenHub GraphQL API, though some filtering approaches require client-side processing.

## API Queries

There are multiple query strategies depending on the filters used:

### Strategy 1: List by Pipeline (Most Common)

When filtering by pipeline or listing all open issues, use `searchIssuesByPipeline`:

```graphql
query ListIssuesByPipeline($pipelineId: ID!, $filters: IssueSearchFiltersInput!, $first: Int!, $after: String) {
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
      pullRequest
      htmlUrl
      estimate { value }
      repository {
        id
        ghId
        name
        ownerName
      }
      assignees(first: 10) {
        nodes { login avatarUrl }
      }
      labels(first: 20) {
        nodes { name color }
      }
      blockingIssues(first: 5) {
        totalCount
      }
      blockedIssues(first: 5) {
        totalCount
      }
      sprints(first: 1) {
        nodes { id name state }
      }
      parentZenhubEpics(first: 3) {
        nodes { id title }
      }
      connectedPrs(first: 5) {
        totalCount
      }
      pipelineIssue(workspaceId: $workspaceId) {
        pipeline { id name }
        priority { id name color }
        relativePosition
        latestTransferTime
      }
    }
  }
}
```

To list all open issues across all pipelines, iterate through each pipeline. The pipelines can be fetched once and cached.

### Strategy 2: List Closed Issues

For closed issues, use `searchClosedIssues`:

```graphql
query ListClosedIssues($workspaceId: ID!, $filters: IssueSearchFiltersInput!, $first: Int!, $after: String) {
  searchClosedIssues(
    workspaceId: $workspaceId
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
      closedAt
      htmlUrl
      estimate { value }
      repository {
        id
        ghId
        name
        ownerName
      }
      assignees(first: 10) {
        nodes { login }
      }
      labels(first: 20) {
        nodes { name color }
      }
    }
  }
}
```

### Strategy 3: List by Epic

When filtering by epic, use `searchIssuesByZenhubEpics`:

```graphql
query ListIssuesByEpic($zenhubEpicIds: [ID!]!, $filters: ZenhubEpicIssueSearchFiltersInput!, $first: Int!, $after: String) {
  searchIssuesByZenhubEpics(
    zenhubEpicIds: $zenhubEpicIds
    filters: $filters
    first: $first
    after: $after
  ) {
    totalCount
    nodes {
      id
      number
      title
      state
      # ... same fields as above
    }
  }
}
```

### Strategy 4: List by Sprint

When filtering by sprint, get issues directly from the Sprint object:

```graphql
query ListIssuesBySprint($workspaceId: ID!) {
  workspace(id: $workspaceId) {
    activeSprint {
      id
      name
      issues(first: 100) {
        totalCount
        nodes {
          id
          number
          title
          state
          # ... same fields as above
        }
      }
    }
  }
}
```

Or for a specific sprint by ID, use `node`:

```graphql
query GetSprintIssues($sprintId: ID!) {
  node(id: $sprintId) {
    ... on Sprint {
      id
      name
      issues(first: 100) {
        nodes {
          id
          number
          title
          # ...
        }
      }
    }
  }
}
```

### Strategy 5: List Blocked Issues

For finding blocked issues, query workspace dependencies:

```graphql
query ListBlockedIssues($workspaceId: ID!, $first: Int!, $after: String) {
  workspace(id: $workspaceId) {
    issueDependencies(first: $first, after: $after) {
      totalCount
      nodes {
        id
        blockedIssue {
          id
          number
          title
          state
          repository { name ownerName }
        }
        blockingIssue {
          id
          number
          title
          state
          repository { name ownerName }
        }
      }
    }
  }
}
```

## Filter Input Reference

The `IssueSearchFiltersInput` supports:

| Filter | Type | Description |
|--------|------|-------------|
| `repositoryIds` | `[ID!]` | Filter by repository ZenHub IDs |
| `labels` | `StringInput` | Filter by label names (`in`, `nin`, `notInAny`) |
| `assignees` | `IssueUserLoginInput` | Filter by assignee login (`in`, `nin`, `notInAny`) |
| `sprints` | `SprintIdInput` | Filter by sprint IDs or `current_sprint` |
| `estimates` | `EstimateSearchFiltersInput` | Filter by estimate values or `not_estimated` |
| `zenhubEpics` | `ZenhubEpicSearchFiltersInput` | Filter by epic IDs or `not_in_epic` |
| `displayType` | `DisplayFilter` | `all`, `issues`, or `prs` |
| `matchType` | `MatchingFilter` | `all` or `any` (AND vs OR for filters) |
| `milestones` | `StringInput` | Filter by milestone title |
| `releases` | `IdInput` | Filter by release IDs |

### Special Filters

- **Estimates**: `{ specialFilters: not_estimated }` for issues without estimates
- **Sprints**: `{ specialFilters: current_sprint }` for current sprint
- **Epics**: `{ specialFilters: not_in_epic }` for issues not in any epic

### Sort Options

Use `IssueOrderInput` with `searchIssuesByPipeline`:

| Field | Description |
|-------|-------------|
| `assignees` | Sort by assignee login |
| `created_at` | Sort by creation date |
| `updated_at` | Sort by ZenHub update date |
| `gh_updated_at` | Sort by GitHub update date |
| `sprints` | Sort by sprint dates |
| `title` | Sort alphabetically by title |
| `estimate` | Sort by estimate value |
| `stale` | Sort by staleness |
| `time_in_pipeline` | Sort by time in current pipeline |

Direction: `ASC` or `DESC`

## Caching Requirements

To support the various filter options, the following should be cached:

| Data | Purpose |
|------|---------|
| Pipelines | Pipeline name-to-ID resolution, iterating all pipelines |
| Repositories | Repository name-to-ID resolution for `--repo` filter |
| Sprints | Sprint name-to-ID resolution for `--sprint` filter |
| Epic names/IDs | Epic title-to-ID resolution for `--epic` filter |

Cache structure additions to existing files:
- `sprints-{workspace_id}.json` - Sprint metadata
- `epics-{workspace_id}.json` - Epic ID/title mappings

## Suggested Flags

| Flag | Description | API Mapping |
|------|-------------|-------------|
| `--pipeline=<name>` | Filter to a specific pipeline | Query that pipeline only |
| `--sprint=<id\|name\|current>` | Filter by sprint | `filters.sprints` |
| `--epic=<id\|title>` | Filter by ZenHub epic | Use `searchIssuesByZenhubEpics` |
| `--assignee=<user>` | Filter by assignee login | `filters.assignees.in` |
| `--no-assignee` | Issues with no assignee | `filters.assignees.notInAny: true` |
| `--label=<label>` | Filter by label | `filters.labels.in` |
| `--repo=<name>` | Filter by repository | `filters.repositoryIds` |
| `--estimate=<value>` | Filter by estimate value | `filters.estimates.values.in` |
| `--no-estimate` | Issues without estimates | `filters.estimates.specialFilters: not_estimated` |
| `--blocked` | Only blocked issues | Query `issueDependencies`, filter client-side |
| `--blocking` | Only blocking issues | Query `issueDependencies`, filter client-side |
| `--type=<issues\|prs\|all>` | Filter by issue/PR | `filters.displayType` |
| `--closed` | Include/show closed issues | Use `searchClosedIssues` |
| `--state=<open\|closed>` | Filter by state | Determines which query to use |
| `--sort=<field>` | Sort field | `order.field` |
| `--order=<asc\|desc>` | Sort direction | `order.direction` |
| `--limit=<n>` | Maximum results | `first` parameter |
| `--view=<name>` | Apply saved view filters | Resolve view, apply its filters |

## Default Output Format

```
# Issues in Development (25 total)

| # | Issue | Est | Assignees | Labels | Sprint |
|---|-------|-----|-----------|--------|--------|
| 1 | posting#1392 Premium CSP expiry... | 3 | @GieRam | | |
| 2 | mpt#2451 Lock browser version... | - | @davebream, @carlos-motiro | daily hit, improvement | |
| 3 | mpt#2849 review & update pnp... | - | @dslh | | |
```

## GitHub API

**Not required for core functionality.**

GitHub's API could optionally supplement:
- Richer issue metadata (reactions, comments count, milestone details)
- Issue body preview
- More detailed PR status (checks, reviews)

But ZenHub provides all essential fields for listing.

## Limitations

### No Direct "All Issues" Query
There's no single query to fetch all issues across all pipelines with filters. The implementation must:
1. Fetch all pipeline IDs
2. Query each pipeline individually
3. Merge and deduplicate results

This is inefficient for workspaces with many pipelines. Consider parallel queries.

### Blocked Filter is Client-Side
There's no API filter for "only blocked issues". The `--blocked` flag requires:
1. Fetching all issue dependencies via `workspace.issueDependencies`
2. Filtering the result set client-side

### Priority Filter Not Available
While `pipelineIssue.priority` exposes the priority, there's no filter input for priority. Filtering by priority requires fetching all issues and filtering client-side.

### No Text Search
The `query` parameter on `searchIssuesByPipeline` appears to search issue titles, but full-text search of issue bodies is not available.

## Related/Adjacent API Capabilities

### Issue Type Filtering
The `issueIssueTypes` filter and `issueIssueTypeDisposition` enable filtering by custom issue types if the workspace uses them.

### Parent/Child Issue Filtering
`parentIssues` filter supports:
- `parents_with_children` - Show hierarchy
- `parents_only` - Only parent issues
- `not_in_parent` - Orphan issues
- `not_a_parent` - Leaf issues only

This could support future `--parent`, `--children`, `--orphan` flags.

### Milestone Filtering
Issues can be filtered by GitHub milestone title via `filters.milestones`.

### Release Filtering
Issues associated with ZenHub releases can be filtered via `filters.releases`.
