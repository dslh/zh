# zh pipeline edit

Update a pipeline's name, position, description, or other settings.

## Feasibility

**Fully Feasible** - The ZenHub GraphQL API provides an `updatePipeline` mutation for basic properties (name, position, description). Additional mutations handle stage assignment (`setPipelineStages`), default PR pipeline (`setPullRequestPipeline`), and stale issue configuration (`updatePipelineConfiguration`).

## API Mutations

### Update Pipeline (Basic Properties)

```graphql
mutation UpdatePipeline($input: UpdatePipelineInput!) {
  updatePipeline(input: $input) {
    pipeline {
      id
      name
      description
      stage
      isDefaultPRPipeline
      updatedAt
    }
  }
}
```

**Variables:**

```json
{
  "input": {
    "pipelineId": "Z2lkOi8vcmFwdG9yL1BpcGVsaW5lLzE5MTY1NDY",
    "name": "Triage",
    "position": 0,
    "description": "New issues to be triaged"
  }
}
```

### UpdatePipelineInput Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `pipelineId` | ID! | Yes | The pipeline to update |
| `name` | String | No | New display name |
| `position` | Int | No | New zero-indexed position from the left |
| `description` | String | No | New description text |

All fields except `pipelineId` are optional. Only provide the fields you want to change.

### Set Pipeline Stage

Pipeline stages must be updated via a separate mutation:

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

**Variables (to set a single pipeline's stage):**

```json
{
  "input": {
    "workspaceId": "5c5c2662a623f9724788f533",
    "inDevelopmentPipelineIds": [
      "Z2lkOi8vcmFwdG9yL1BpcGVsaW5lLzE5MTY1NDY"
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

**Note:** To clear a pipeline's stage, you would need to know which list it's currently in and remove it. The API doesn't provide a direct "clear stage" operation.

### Set Default PR Pipeline

```graphql
mutation SetPullRequestPipeline($input: SetPullRequestPipelineInput!) {
  setPullRequestPipeline(input: $input) {
    workspace {
      id
      pipelinesConnection(first: 50) {
        nodes {
          id
          name
          isDefaultPRPipeline
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
    "pipelineId": "Z2lkOi8vcmFwdG9yL1BpcGVsaW5lLzE5MTcwMjk"
  }
}
```

**Note:** Only one pipeline can be the default PR pipeline at a time. Setting a new default automatically clears the previous one.

### Update Pipeline Configuration (Stale Issues)

```graphql
mutation UpdatePipelineConfiguration($input: UpdatePipelineConfigurationInput!) {
  updatePipelineConfiguration(input: $input) {
    pipelineConfiguration {
      id
      staleIssues
      staleInterval
      showAgeInPipeline
    }
  }
}
```

**Variables:**

```json
{
  "input": {
    "pipelineId": "Z2lkOi8vcmFwdG9yL1BpcGVsaW5lLzE5MTcwMjk",
    "staleIssues": true,
    "staleInterval": 4,
    "showAgeInPipeline": true
  }
}
```

### UpdatePipelineConfigurationInput Fields

| Field | Type | Description |
|-------|------|-------------|
| `pipelineId` | ID! | The pipeline to configure |
| `staleIssues` | Boolean | Enable/disable stale issue highlighting |
| `staleInterval` | Int | Number of days before an issue is considered stale |
| `showAgeInPipeline` | Boolean | Show issue age in the pipeline view |

## Implementation Flow

1. Resolve the pipeline name/identifier to a pipeline ID (from cache or API)
2. Determine which mutations are needed based on provided flags
3. Execute `updatePipeline` if name, position, or description changed
4. Execute `setPipelineStages` if stage changed
5. Execute `setPullRequestPipeline` if `--default-pr-pipeline` flag provided
6. Execute `updatePipelineConfiguration` if stale settings changed
7. Invalidate/update the local pipeline cache
8. Return the updated pipeline details

## Caching Requirements

- **Pipeline ID lookup** - Resolve pipeline name to ID from `pipelines-{workspace_id}.json` cache
- **Workspace ID** - Required for `setPipelineStages` and `setPullRequestPipeline`
- **Cache invalidation** - Update cache after successful mutation, especially if name or position changed

## Suggested Flags

| Flag | Description |
|------|-------------|
| `--name=<text>` | New pipeline name |
| `--position=<n>` | New position (0-indexed). Special values: `start`, `end` |
| `--after=<pipeline>` | Move to position after a named pipeline |
| `--before=<pipeline>` | Move to position before a named pipeline |
| `--description=<text>` | New description (use `--description=""` to clear) |
| `--stage=<stage>` | Workflow stage: `backlog`, `sprint_backlog`, `development`, `review`, `completed`, `none` |
| `--default-pr-pipeline` | Set as the default pipeline for new PRs |
| `--stale-after=<days>` | Enable stale issue highlighting after N days (0 to disable) |
| `--show-age` | Show issue age in pipeline view |
| `--no-show-age` | Hide issue age in pipeline view |
| `--output=json` | Output in JSON format |
| `--dry-run` | Show what would change without executing |

### Position Resolution

When `--after` or `--before` is specified:
1. Resolve the reference pipeline's current position from cache
2. Calculate target position (`after` = reference + 1, `before` = reference)
3. Use the calculated position in the mutation

## Default Output Format

```
Updated pipeline "Code Review".

Name: Code Review
Position: 8
Stage: Development
Description: Issues under active code review
Stale after: 4 days
Default PR pipeline: No
```

With `--dry-run`:

```
Would update pipeline "Code Review":

  Name: Code Review (unchanged)
  Position: 8 -> 6 (moving before "In Development")
  Stage: Development (unchanged)
  Description: (none) -> "Issues under active code review"
  Stale after: (disabled) -> 4 days
```

## GitHub API

Not needed. Pipeline management is entirely within ZenHub's domain.

## Limitations

1. **Multiple mutations required** - Changing name and stage requires two separate API calls. There's no single mutation that handles all pipeline properties.

2. **No stage clearing** - The `setPipelineStages` mutation sets stages but doesn't have a direct way to clear a stage from a pipeline. To remove a stage, you'd need to call the mutation without the pipeline in any list, which may require knowing all current stage assignments.

3. **Position behavior** - The API doesn't document what happens when the target position is invalid (negative, or beyond the end). Testing would be needed to confirm behavior.

4. **WIP limits not editable** - While WIP limits appear in the `PipelineConfiguration` type, there's no visible mutation to set them via the GraphQL API. This may be a UI-only or Enterprise feature.

5. **No atomic updates** - If the command updates multiple properties (name + stage + stale config), they happen as separate mutations. A failure partway through could leave the pipeline in a partially-updated state.

## Related Subcommands

- **`zh pipeline show <name>`** - View current pipeline settings before editing
- **`zh pipeline list`** - Verify changes and see new position
- **`zh pipeline create`** - Create a new pipeline with similar options
- **`zh pipeline delete`** - Remove a pipeline instead of editing

## Adjacent API Capabilities

### Pipeline Automations

The API supports pipeline automations (`createPipelineAutomation`, `updatePipelineAutomation`, `deletePipelineAutomation`) that trigger actions when issues enter or leave pipelines. This could support a future `zh pipeline automate` subcommand or `--automation` flags on edit.

### Pipeline-to-Pipeline Automations

The `createPipelineToPipelineAutomation` and related mutations allow setting up automatic issue movement between pipelines based on events. For example: "When a PR is merged, move the connected issue to 'Done'".

### Move All Issues

The `moveAllPipelineIssues` mutation allows moving all issues from one pipeline to another. While this is covered by `zh pipeline delete --into=<name>`, it could also be a standalone feature: `zh pipeline move-all <from> <to>`.
