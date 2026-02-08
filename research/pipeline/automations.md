# zh pipeline automations <name>

Display configured automations for a pipeline.

## Feasibility

**Fully feasible** — ZenHub's GraphQL API exposes two distinct automation systems on pipelines, both of which are queryable.

## Automation Types

ZenHub pipelines support two kinds of automations:

### 1. Pipeline Automations (`PipelineAutomation`)

Event-driven automations configured per-pipeline. These are accessed via `pipeline.pipelineConfiguration.pipelineAutomations`. Each automation has an opaque `elementDetails` field (JSON) that encodes the trigger and action. The exact structure of `elementDetails` is not documented in the schema — it is a freeform `JSON` scalar.

**Fields:**

| Field | Type | Description |
|-------|------|-------------|
| `id` | ID! | Automation ID |
| `elementDetails` | JSON! | Opaque JSON describing the trigger/action configuration |
| `createdAt` | DateTime! | When the automation was created |
| `updatedAt` | DateTime! | Last modification timestamp |

### 2. Pipeline-to-Pipeline Automations (`PipelineToPipelineAutomation`)

Rules that automatically move issues from one pipeline to another when certain conditions are met. These can be accessed either per-pipeline (via `pipelineToPipelineAutomationSources` / `pipelineToPipelineAutomationDestinations`) or at the workspace level (via `workspace.pipelineToPipelineAutomations`).

**Fields:**

| Field | Type | Description |
|-------|------|-------------|
| `id` | ID! | Automation ID |
| `sourcePipeline` | Pipeline! | The pipeline issues move from |
| `destinationPipeline` | Pipeline! | The pipeline issues move to |
| `createdAt` | DateTime! | When the automation was created |
| `updatedAt` | DateTime! | Last modification timestamp |

## Queries

### Primary Query: Automations for a Specific Pipeline

Fetches both automation types for a given pipeline. Requires knowing the pipeline ID (resolved from name/alias via cache).

```graphql
query PipelineAutomations($workspaceId: ID!) {
  workspace(id: $workspaceId) {
    pipelinesConnection(first: 50) {
      nodes {
        id
        name
        pipelineConfiguration {
          pipelineAutomations(first: 50) {
            totalCount
            nodes {
              id
              elementDetails
              createdAt
              updatedAt
            }
          }
        }
        pipelineToPipelineAutomationSources(first: 50) {
          totalCount
          nodes {
            id
            destinationPipeline {
              id
              name
            }
            createdAt
          }
        }
        pipelineToPipelineAutomationDestinations(first: 50) {
          totalCount
          nodes {
            id
            sourcePipeline {
              id
              name
            }
            createdAt
          }
        }
      }
    }
  }
}
```

Since the API doesn't allow querying a single pipeline directly by ID (pipelines are accessed through `workspace.pipelinesConnection`), the implementation should:
1. Query all pipelines with their automations in one request.
2. Filter client-side to the target pipeline.
3. Cache the full result for use by other commands.

Alternatively, if the pipeline ID is already known, the `node` query could be used, but that requires testing to verify it works for Pipeline types.

### Alternative: Workspace-Level Pipeline-to-Pipeline Automations

For a broader view or when listing all automations across the board:

```graphql
query WorkspaceP2PAutomations($workspaceId: ID!) {
  workspace(id: $workspaceId) {
    pipelineToPipelineAutomations(first: 50) {
      totalCount
      nodes {
        id
        sourcePipeline {
          id
          name
        }
        destinationPipeline {
          id
          name
        }
        createdAt
      }
    }
  }
}
```

## Caching

**Useful to have cached:**
- Workspace ID (from config)
- Pipeline names and IDs (from `pipelines-{workspace_id}.json` cache) for name/alias resolution

Automation data itself should not be cached — it is infrequently accessed and should always reflect current state.

## Suggested Flags

| Flag | Description |
|------|-------------|
| `--output=json` | Output in JSON format |
| `--verbose` | Include raw `elementDetails` JSON for pipeline automations |

No other flags seem necessary given this is a read-only display command.

## Output Format

### Default

```
# Automations for "In Progress"

## Event Automations

No event automations configured.

## Pipeline-to-Pipeline Automations

| Direction | Pipeline | Created |
|-----------|----------|---------|
| Moves from | Code Review | 2024-03-15 |
| Moves into | QA | 2024-03-15 |
```

When no automations exist at all:

```
# Automations for "In Progress"

No automations configured.
```

## Limitations

### Opaque `elementDetails`

The `PipelineAutomation.elementDetails` field is a freeform `JSON` scalar. The schema provides no enumeration of possible automation types, triggers, or actions. The CLI will need to parse this JSON and present it in a human-readable way, but without documentation of the possible structures, this will require reverse-engineering from real data. A `--verbose` flag that dumps the raw JSON would be a useful escape hatch.

If the `elementDetails` format proves too opaque or unstable to parse reliably, the command could:
- Display only pipeline-to-pipeline automations (which have a clean, structured schema).
- Show a count of event automations with a note to view details in the web UI.
- Always show the raw JSON for event automations.

### No Automation Type Metadata

The `PipelineToPipelineAutomation` type does not include a field describing *what condition* triggers the move (e.g., "when PR is merged", "when issue is closed"). It only records the source and destination pipelines. The trigger condition may be implicit (ZenHub's standard pipeline-to-pipeline automations are typically "when an issue enters pipeline A, move it to pipeline B") or may be encoded somewhere not visible in the public API.

### Write Operations Exist But Are Out of Scope

The API exposes full CRUD for both automation types:
- `createPipelineAutomation` / `updatePipelineAutomation` / `deletePipelineAutomation` / `duplicatePipelineAutomation`
- `createPipelineToPipelineAutomation` / `deletePipelineToPipelineAutomation`

The `createPipelineToPipelineAutomation` mutation also supports `applyRetroactively: Boolean` to move existing issues when the rule is created. These are not needed for `zh pipeline automations` (which is read-only), but could support future subcommands.

## GitHub API

Not needed. All automation information is available from ZenHub's API.

## Adjacent API Capabilities

### WIP Limits and Stale Issue Config

The `PipelineConfiguration` type also exposes `wipLimits` and stale issue settings (`staleIssues`, `staleInterval`, `showAgeInPipeline`). These are not automations per se, but are pipeline-level configuration that could be included in a broader `zh pipeline config <name>` subcommand or folded into `zh pipeline show` with a `--verbose` flag. The existing `zh pipeline show` research already covers these.

### Workspace-Wide Automation Overview

The `workspace.pipelineToPipelineAutomations` field could support a `zh board automations` or `zh automations` command that shows all pipeline-to-pipeline automations across the entire board in one view, which might be more useful than viewing them one pipeline at a time.
