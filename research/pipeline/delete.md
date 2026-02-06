# zh pipeline delete

Delete a pipeline from the workspace, moving its issues to another pipeline.

## Feasibility

**Fully Feasible** - The ZenHub GraphQL API provides a `deletePipeline` mutation that handles both the deletion and the issue migration in a single atomic operation. The `destinationPipelineId` is a required parameter, which aligns with the `--into=<name>` flag requirement in the spec.

## API Mutation

### Delete Pipeline

```graphql
mutation DeletePipeline($input: DeletePipelineInput!) {
  deletePipeline(input: $input) {
    clientMutationId
    destinationPipeline {
      id
      name
      issues {
        totalCount
      }
    }
  }
}
```

**Variables:**

```json
{
  "input": {
    "pipelineId": "Z2lkOi8vcmFwdG9yL1BpcGVsaW5lLzE5MTY1NDY",
    "destinationPipelineId": "Z2lkOi8vcmFwdG9yL1BpcGVsaW5lLzE5MTY4MTI"
  }
}
```

### DeletePipelineInput Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `pipelineId` | ID! | Yes | The pipeline to delete |
| `destinationPipelineId` | ID! | Yes | The pipeline to move issues into |
| `clientMutationId` | String | No | Optional client identifier for the mutation |

### Response

The mutation returns a `DeletePipelinePayload` containing:

| Field | Type | Description |
|-------|------|-------------|
| `destinationPipeline` | Pipeline! | The pipeline that received the moved issues |
| `clientMutationId` | String | Echo of the client mutation ID if provided |

The destination pipeline object can be queried for updated issue counts to confirm the migration.

## Pre-Delete Query

Before deleting, the CLI should fetch information about the pipeline being deleted to show in confirmation/dry-run output:

```graphql
query GetPipelineForDelete($workspaceId: ID!) {
  workspace(id: $workspaceId) {
    pipelinesConnection(first: 50) {
      nodes {
        id
        name
        issues {
          totalCount
        }
      }
    }
  }
}
```

This allows:
1. Resolving pipeline names to IDs
2. Showing how many issues will be moved
3. Validating both source and destination pipelines exist

## Implementation Flow

1. Resolve the pipeline name to delete to its ID (from cache or API)
2. Resolve the `--into` pipeline name to its ID
3. Fetch the issue count for the pipeline being deleted (for confirmation message)
4. If `--dry-run`, display what would happen and exit
5. Execute the `deletePipeline` mutation
6. Invalidate/update the local pipeline cache
7. Display success message with issue migration details

## Caching Requirements

**Before deletion:**
- **Pipeline ID lookup** - Resolve both pipeline names to IDs from `pipelines-{workspace_id}.json` cache

**After deletion:**
- **Invalidate pipeline cache** - Remove the deleted pipeline from `pipelines-{workspace_id}.json`
- **Consider alias cleanup** - If the deleted pipeline had an alias, it should be removed from config

## Suggested Flags

| Flag | Description |
|------|-------------|
| `--into=<name>` | **Required.** Target pipeline for issues (by name, ID, or alias) |
| `--force` | Skip confirmation prompt |
| `--output=json` | Output in JSON format |
| `--dry-run` | Show what would be deleted without executing |

### Pipeline Identifier Resolution

Both `<name>` (the pipeline to delete) and `--into=<name>` should support:
- Exact pipeline name
- Pipeline ID (base64 ZenHub ID)
- Unique substring of the name
- Alias (from `zh pipeline alias`)

## Default Output Format

```
Deleting pipeline "In Research" will move 7 issues to "Backlog".

Proceed? [y/N] y

Deleted pipeline "In Research".
Moved 7 issues to "Backlog" (now has 98 issues total).
```

With `--dry-run`:

```
Would delete pipeline "In Research".

Pipeline ID: Z2lkOi8vcmFwdG9yL1BpcGVsaW5lLzIyNDk5NTI
Issues to move: 7
Destination: Backlog (Z2lkOi8vcmFwdG9yL1BpcGVsaW5lLzE5MTY4MTI)
Current destination count: 91
```

With `--force` (no prompt):

```
Deleted pipeline "In Research".
Moved 7 issues to "Backlog".
```

## Error Cases

| Error | Exit Code | Message |
|-------|-----------|---------|
| Pipeline not found | 4 | `Error: Pipeline "Xyz" not found in workspace` |
| Destination not found | 4 | `Error: Destination pipeline "Xyz" not found` |
| Same source and destination | 2 | `Error: Cannot delete pipeline into itself` |
| Only one pipeline remaining | 1 | `Error: Cannot delete the last pipeline in workspace` |
| Permission denied | 3 | `Error: You don't have permission to delete pipelines` |

## GitHub API

Not needed. Pipeline management is entirely within ZenHub's domain.

## Limitations

1. **No undo** - Once deleted, the pipeline cannot be recovered. Issues are moved but the pipeline's configuration (stage, description, automations) is lost.

2. **No position control for moved issues** - The API doesn't expose where in the destination pipeline the issues are placed. They likely go to the bottom.

3. **Automation cleanup unknown** - It's unclear what happens to pipeline automations that reference the deleted pipeline (either as source or destination). They may be automatically removed or may cause errors.

4. **No partial migration** - All issues must go to a single destination. If you want to distribute issues to multiple pipelines, you'd need to move them first using `zh issue move`.

## Related Subcommands

- **`zh pipeline list`** - Verify the pipeline exists and see its issue count before deleting
- **`zh pipeline show <name>`** - View pipeline details including issues before deciding to delete
- **`zh issue move`** - Manually move specific issues before deleting if you don't want all issues going to the same destination
- **`zh pipeline create`** - Create a new pipeline if the deletion was a mistake

## Adjacent API Capabilities

### Move All Pipeline Issues

The API also provides a separate `moveAllPipelineIssues` mutation that can move issues from multiple source pipelines to a single destination without deleting the source pipelines:

```graphql
mutation MoveAllPipelineIssues($input: MoveAllPipelineIssuesInput!) {
  moveAllPipelineIssues(input: $input) {
    workspace {
      id
    }
  }
}
```

**Variables:**

```json
{
  "input": {
    "pipelineIds": [
      "Z2lkOi8vcmFwdG9yL1BpcGVsaW5lLzE5MTY1NDY",
      "Z2lkOi8vcmFwdG9yL1BpcGVsaW5lLzE5MTY2ODc"
    ],
    "destinationPipelineId": "Z2lkOi8vcmFwdG9yL1BpcGVsaW5lLzE5MTY4MTI"
  }
}
```

This could support a future `zh pipeline move-all <source>... <destination>` command that empties pipelines without deleting them.

### Workspace Pipeline Limits

It's unclear if there's a minimum number of pipelines required in a workspace. The API may reject deletion of the last pipeline. Testing would be needed to confirm this behavior.
