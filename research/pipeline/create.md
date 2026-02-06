# zh pipeline create

Create a new pipeline in the workspace.

## Feasibility

**Fully Feasible** - The ZenHub GraphQL API provides a `createPipeline` mutation with all necessary inputs.

## API Mutation

### Create Pipeline

```graphql
mutation CreatePipeline($input: CreatePipelineInput!) {
  createPipeline(input: $input) {
    pipeline {
      id
      name
      description
      stage
      createdAt
    }
  }
}
```

**Variables:**

```json
{
  "input": {
    "workspaceId": "5c5c2662a623f9724788f533",
    "name": "QA Review",
    "position": 7,
    "description": "Issues waiting for QA verification"
  }
}
```

### CreatePipelineInput Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `workspaceId` | ID! | Yes | The workspace to create the pipeline in |
| `name` | String! | Yes | Pipeline display name |
| `position` | Int | No | Zero-indexed position from the left. If omitted, likely appends to end |
| `description` | String | No | Optional description text |

### Response

The mutation returns the created `Pipeline` object with all standard fields available.

## Setting Pipeline Stage

The `createPipeline` mutation does **not** accept a stage parameter. Pipeline stages must be set separately using the `setPipelineStages` mutation after creation:

```graphql
mutation SetPipelineStages($input: SetPipelineStagesInput!) {
  setPipelineStages(input: $input) {
    workspace {
      id
      pipelinesConnection(first: 50) {
        nodes {
          id
          name
          stage
        }
      }
    }
  }
}
```

**Variables:**

```json
{
  "input": {
    "workspaceId": "5c5c2662a623f9724788f533",
    "inReviewPipelineIds": [
      "Z2lkOi8vcmFwdG9yL1BpcGVsaW5lLzM0NTM1MDA"
    ]
  }
}
```

### SetPipelineStagesInput Fields

| Field | Type | Description |
|-------|------|-------------|
| `workspaceId` | ID! | The workspace ID |
| `backlogPipelineIds` | [ID!] | Pipelines to assign BACKLOG stage |
| `sprintBacklogPipelineIds` | [ID!] | Pipelines to assign SPRINT_BACKLOG stage |
| `inDevelopmentPipelineIds` | [ID!] | Pipelines to assign DEVELOPMENT stage |
| `inReviewPipelineIds` | [ID!] | Pipelines to assign REVIEW stage |
| `completedPipelineIds` | [ID!] | Pipelines to assign COMPLETED stage |

**Important:** This mutation sets stages for the specified pipelines. If you only provide `inReviewPipelineIds`, it will only update those pipelines. Existing stage assignments for other pipelines are preserved.

## Implementation Flow

1. Execute `createPipeline` mutation with name, optional position, and optional description
2. If `--stage` flag is provided, execute `setPipelineStages` mutation to assign the stage
3. Invalidate the local pipeline cache
4. Return the created pipeline details

## Caching Requirements

After creating a pipeline:

- **Invalidate pipeline cache** - The local `pipelines-{workspace_id}.json` cache should be invalidated or updated to include the new pipeline
- **Workspace ID required** - Must be retrieved from config or cache

## Suggested Flags

| Flag | Description |
|------|-------------|
| `--position=<n>` | Position from the left (0-indexed). Special values: `start` (0), `end` (append) |
| `--description=<text>` | Pipeline description |
| `--stage=<stage>` | Workflow stage: `backlog`, `sprint_backlog`, `development`, `review`, `completed` |
| `--after=<pipeline>` | Position after a named pipeline (alternative to numeric position) |
| `--before=<pipeline>` | Position before a named pipeline |
| `--output=json` | Output in JSON format |
| `--dry-run` | Show what would be created without executing |

### Position Resolution

When `--after` or `--before` is specified:
1. Resolve the reference pipeline's current position from the cache
2. Calculate the target position (`after` = reference position + 1, `before` = reference position)
3. Use the calculated position in the mutation

## Default Output Format

```
Created pipeline "QA Review" at position 7.

ID: Z2lkOi8vcmFwdG9yL1BpcGVsaW5lLzM0NTM1MDA
Stage: Review
Description: Issues waiting for QA verification
```

With `--dry-run`:

```
Would create pipeline "QA Review" at position 7.

Workspace: Development (5c5c2662a623f9724788f533)
Stage: Review
Description: Issues waiting for QA verification
```

## GitHub API

Not needed. Pipeline management is entirely within ZenHub's domain.

## Limitations

1. **Stage requires separate mutation** - Cannot set the workflow stage in a single API call; requires a follow-up `setPipelineStages` mutation
2. **No duplicate name validation** - The API may allow pipelines with identical names, which could cause issues with name-based resolution
3. **Position behavior unclear** - The API documentation doesn't specify what happens when position is omitted or when the specified position exceeds the current pipeline count

## Related Subcommands

- **`zh pipeline list`** - Verify the pipeline was created and see its position
- **`zh pipeline edit <name>`** - Modify the pipeline after creation
- **`zh pipeline delete <name>`** - Remove the pipeline
- **`zh pipeline alias <name> <alias>`** - Set a shorthand for the new pipeline

## Adjacent API Capabilities

### Pipeline Configuration

After creating a pipeline, additional configuration can be set via `updatePipelineConfiguration`:

- **Stale issue highlighting** - `staleIssues`, `staleInterval`, `showAgeInPipeline`
- **WIP limits** - Limit the number of issues in the pipeline

This could support a future `--wip-limit=<n>` or `--stale-after=<days>` flag, or a dedicated `zh pipeline configure` subcommand.

### Pipeline Automations

The API supports `createPipelineAutomation` and `createPipelineToPipelineAutomation` for setting up automatic actions when issues enter or leave pipelines. This could support a future `zh pipeline automate` subcommand.

### Default PR Pipeline

The `setPullRequestPipeline` mutation allows designating a pipeline as the default for new PRs. This could be exposed as a `--default-pr-pipeline` flag or via `zh pipeline edit`.
