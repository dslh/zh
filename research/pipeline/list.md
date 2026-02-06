# zh pipeline list

List all pipelines in the workspace.

## Feasibility

**Fully Feasible** - All required data is available through the ZenHub GraphQL API.

## API Query

### Primary Query: List Pipelines

```graphql
query ListPipelines($workspaceId: ID!) {
  workspace(id: $workspaceId) {
    id
    displayName
    pipelinesConnection(first: 50) {
      totalCount
      nodes {
        id
        name
        description
        stage
        isDefaultPRPipeline
        createdAt
        updatedAt
        issues {
          totalCount
        }
      }
    }
  }
}
```

### Extended Query: With Configuration Details

For verbose output or when WIP limit / stale issue configuration is desired:

```graphql
query ListPipelinesWithConfig($workspaceId: ID!) {
  workspace(id: $workspaceId) {
    id
    displayName
    pipelinesConnection(first: 50) {
      totalCount
      nodes {
        id
        name
        description
        stage
        isDefaultPRPipeline
        createdAt
        updatedAt
        issues {
          totalCount
        }
        pipelineConfiguration {
          showAgeInPipeline
          staleInterval
          staleIssues
          wipLimits {
            totalCount
            nodes {
              blockPipeline
              limitValue
            }
          }
        }
      }
    }
  }
}
```

## Pipeline Fields

| Field | Type | Description |
|-------|------|-------------|
| `id` | ID! | ZenHub pipeline ID (base64 encoded) |
| `name` | String! | Pipeline display name |
| `description` | String | Optional description |
| `stage` | PipelineStage | Workflow stage classification |
| `isDefaultPRPipeline` | Boolean! | Whether new PRs are auto-assigned here |
| `createdAt` | DateTime! | When the pipeline was created |
| `updatedAt` | DateTime! | Last modification timestamp |
| `issues` | IssueConnection! | Issues in this pipeline |
| `itemBefore` | Pipeline | Previous pipeline in order (for determining position) |
| `pipelineConfiguration` | PipelineConfiguration! | Stale issue and WIP limit settings |

### PipelineStage Enum

Pipelines can be classified into workflow stages:

| Value | Description |
|-------|-------------|
| `BACKLOG` | Pre-sprint backlog |
| `SPRINT_BACKLOG` | Committed to current sprint |
| `DEVELOPMENT` | Active development work |
| `REVIEW` | Review/QA/deployment stages |
| `COMPLETED` | Done/closed |
| `null` | Unclassified (custom pipelines) |

### PipelineConfiguration Fields

| Field | Type | Description |
|-------|------|-------------|
| `showAgeInPipeline` | Boolean | Show age indicator on cards |
| `staleInterval` | Int | Days until issue is considered stale |
| `staleIssues` | Boolean | Whether stale highlighting is enabled |
| `wipLimits` | WipLimitConnection | WIP limit settings |

### WIP Limit Fields

| Field | Type | Description |
|-------|------|-------------|
| `blockPipeline` | Boolean! | Whether WIP limit blocks additions |
| `limitValue` | JSON! | Limit configuration (format: `{limitType: value}`) |

## Pipeline Ordering

Pipelines are returned in board order. The API uses a linked-list style ordering via the `itemBefore` field:
- The first pipeline has `itemBefore: null`
- Subsequent pipelines reference the previous pipeline

The nodes returned by `pipelinesConnection` are already sorted in display order.

## Caching Requirements

Pipeline data should be cached for:
- **Name-to-ID resolution** - Enable `--pipeline=<name>` on other commands
- **Substring matching** - Support partial name matches per spec
- **Alias resolution** - Support user-defined aliases

Cache structure (`~/.cache/zh/pipelines-{workspace_id}.json`):
```json
{
  "workspace_id": "5c5c2662a623f9724788f533",
  "fetched_at": "2024-01-15T10:30:00Z",
  "pipelines": [
    {
      "id": "Z2lkOi8vcmFwdG9yL1BpcGVsaW5lLzE5MTY1NDY",
      "name": "New Issues",
      "description": null,
      "stage": null
    }
  ]
}
```

## Suggested Flags

| Flag | Description |
|------|-------------|
| `--stage=<stage>` | Filter by workflow stage (backlog, sprint_backlog, development, review, completed) |
| `--with-counts` | Include issue counts (default: true, use `--no-counts` to disable) |
| `--verbose` | Include configuration details (stale settings, WIP limits) |
| `--output=json` | Output in JSON format |

## Default Output Format

Suggested markdown table format:

```
# Pipelines in Development

| # | Pipeline | Issues | Stage |
|---|----------|--------|-------|
| 1 | New Issues | 727 | - |
| 2 | Icebox | 157 | - |
| 3 | Backlog | 91 | Backlog |
| 4 | Next Up | 15 | Sprint Backlog |
| 5 | In Development | 441 | Development |
| 6 | Code Review | 754 | Development |
| 7 | Done | 3771 | Completed |
```

With `--verbose`:

```
# Pipelines in Development

| # | Pipeline | Issues | Stage | Stale (days) | Default PR |
|---|----------|--------|-------|--------------|------------|
| 1 | New Issues | 727 | - | - | No |
| 2 | Ready for Code Review | 1272 | Development | 5 | Yes |
| 3 | Done | 3771 | Completed | - | No |
```

## GitHub API

Not needed. All pipeline information is available from ZenHub's API.

## Limitations

None identified. The API provides complete access to pipeline metadata.

## Related Subcommands

Data from this query can support:

- **`zh pipeline show <name>`** - Same query with extended issue data for a specific pipeline
- **`zh board`** - Uses `pipelinesConnection` as the top-level structure
- **`zh issue move`** - Needs pipeline ID resolution from cached list

## Adjacent API Capabilities

### Pipeline Automations

The API exposes pipeline automations via `pipelineConfiguration.pipelineAutomations`. This could support a future `zh pipeline automations` subcommand to list configured automations.

### Pipeline-to-Pipeline Automations

The `pipelineToPipelineAutomationSources` and `pipelineToPipelineAutomationDestinations` fields show cross-pipeline automation rules.

### Closed Pipeline

The workspace has a special `closedPipeline` field (described as "Only for querying control chart") which represents the terminal state for closed issues. This is separate from user-visible pipelines and not shown in `pipelinesConnection`.
