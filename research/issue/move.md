# zh issue move

Move one or more issues to a pipeline with optional position control.

## Feasibility

**Fully Feasible** - The ZenHub GraphQL API provides multiple mutations for moving issues between pipelines. The `movePipelineIssues` mutation is the most versatile, supporting bulk moves with position control.

## API Mutations

### Primary: Move Multiple Issues (Recommended)

For moving one or more issues to a pipeline with position control, use `movePipelineIssues`:

```graphql
mutation MovePipelineIssues($input: MovePipelineIssuesInput!) {
  movePipelineIssues(input: $input) {
    pipeline {
      id
      name
    }
    pipelineIssues {
      id
      issue {
        id
        number
        title
        repository {
          name
          ownerName
        }
      }
      pipeline {
        id
        name
      }
    }
  }
}
```

**Variables (move to top of pipeline):**

```json
{
  "input": {
    "pipelineId": "Z2lkOi8vcmFwdG9yL1BpcGVsaW5lLzE5MTY5NzY",
    "pipelineIssueIds": [
      "Z2lkOi8vcmFwdG9yL1BpcGVsaW5lSXNzdWUvOTA3MzQ3MQ",
      "Z2lkOi8vcmFwdG9yL1BpcGVsaW5lSXNzdWUvOTA3MzQ5MA"
    ],
    "position": "START"
  }
}
```

**Variables (move to bottom of pipeline):**

```json
{
  "input": {
    "pipelineId": "Z2lkOi8vcmFwdG9yL1BpcGVsaW5lLzE5MTY5NzY",
    "pipelineIssueIds": [
      "Z2lkOi8vcmFwdG9yL1BpcGVsaW5lSXNzdWUvOTA3MzQ3MQ"
    ],
    "position": "END"
  }
}
```

**Variables (move after a specific issue):**

```json
{
  "input": {
    "pipelineId": "Z2lkOi8vcmFwdG9yL1BpcGVsaW5lLzE5MTY5NzY",
    "pipelineIssueIds": [
      "Z2lkOi8vcmFwdG9yL1BpcGVsaW5lSXNzdWUvOTA3MzQ3MQ"
    ],
    "afterPipelineIssueId": "Z2lkOi8vcmFwdG9yL1BpcGVsaW5lSXNzdWUvODk3NjU0Mw"
  }
}
```

### MovePipelineIssuesInput Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `pipelineId` | ID! | Yes | Target pipeline ID |
| `pipelineIssueIds` | [ID!]! | Yes | List of **PipelineIssue** IDs (not Issue IDs) |
| `position` | PipelineIssuePosition | No | `START` or `END` |
| `afterPipelineIssueId` | ID | No | Place after this PipelineIssue |
| `beforePipelineIssueId` | ID | No | Place before this PipelineIssue |

**Note:** Use `position` OR `afterPipelineIssueId`/`beforePipelineIssueId`, not both.

### Alternative: Move Single Issue with Numeric Position

For moving a single issue to a specific numeric position:

```graphql
mutation MoveIssue($input: MoveIssueInput!) {
  moveIssue(input: $input) {
    issue {
      id
      number
      title
    }
    pipeline {
      id
      name
    }
  }
}
```

**Variables:**

```json
{
  "input": {
    "issueId": "Z2lkOi8vcmFwdG9yL0lzc3VlLzI4ODQ3OTU",
    "pipelineId": "Z2lkOi8vcmFwdG9yL1BpcGVsaW5lLzE5MTY5NzY",
    "position": 0
  }
}
```

### MoveIssueInput Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `issueId` | ID! | Yes | The Issue ID (not PipelineIssue ID) |
| `pipelineId` | ID! | Yes | Target pipeline ID |
| `position` | Int | No | Zero-indexed position in the pipeline |

### Alternative: Move Single Issue Relative

For moving a single issue with symbolic positioning:

```graphql
mutation MoveIssueRelativeTo($input: MoveIssueRelativeToInput!) {
  moveIssueRelativeTo(input: $input) {
    issue {
      id
      number
      title
    }
    pipeline {
      id
      name
    }
  }
}
```

**Variables:**

```json
{
  "input": {
    "issueId": "Z2lkOi8vcmFwdG9yL0lzc3VlLzI4ODQ3OTU",
    "pipelineId": "Z2lkOi8vcmFwdG9yL1BpcGVsaW5lLzE5MTY5NzY",
    "position": "START"
  }
}
```

### MoveIssueRelativeToInput Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `issueId` | ID! | Yes | The Issue ID |
| `pipelineId` | ID | No | Target pipeline (optional if moving within same pipeline) |
| `position` | PipelineIssuePosition | No | `START` or `END` |
| `afterPipelineIssueId` | ID | No | Place after this PipelineIssue |
| `beforePipelineIssueId` | ID | No | Place before this PipelineIssue |

## Implementation Flow

1. **Parse issue identifiers** - Resolve each issue identifier to a ZenHub Issue ID:
   - If ZenHub ID provided directly, use it
   - If `owner/repo#number` or `repo#number` format, use `issueByInfo` query with `repositoryGhId` and `issueNumber`

2. **Resolve pipeline** - Resolve the target pipeline name/identifier to a pipeline ID from cache

3. **Get PipelineIssue IDs** - For each Issue, query its `pipelineIssue(workspaceId)` to get the PipelineIssue ID:

```graphql
query GetPipelineIssueId($issueId: ID!, $workspaceId: ID!) {
  node(id: $issueId) {
    ... on Issue {
      id
      number
      title
      pipelineIssue(workspaceId: $workspaceId) {
        id
        pipeline {
          id
          name
        }
      }
    }
  }
}
```

4. **Execute mutation** - Call `movePipelineIssues` with the collected PipelineIssue IDs

5. **Report results** - Show which issues were moved and to which pipeline

### Batch Query for Multiple Issues

When moving multiple issues, batch the Issue ID lookups:

```graphql
query GetIssuesByInfo($repoGhId: Int!, $numbers: [Int!]!) {
  # Unfortunately there's no batch issueByInfo, so use individual queries
  # or use the issues(ids: [...]) query if you have ZenHub IDs
}
```

Alternatively, if you have the ZenHub Issue IDs:

```graphql
query GetPipelineIssueIds($ids: [ID!]!, $workspaceId: ID!) {
  issues(ids: $ids) {
    id
    number
    title
    repository {
      name
      ownerName
    }
    pipelineIssue(workspaceId: $workspaceId) {
      id
      pipeline {
        id
        name
      }
    }
  }
}
```

## Caching Requirements

| Data | Purpose |
|------|---------|
| Workspace ID | Required for `pipelineIssue(workspaceId)` lookup |
| Pipelines | Resolve pipeline name to ID |
| Repositories | Resolve `repo#number` format to `repositoryGhId` for `issueByInfo` |

## Suggested Flags

| Flag | Description |
|------|-------------|
| `--position=<top\|bottom\|n>` | Position in target pipeline. `top`/`bottom` use `START`/`END`, numeric uses `moveIssue` |
| `--after=<issue>` | Place after a specific issue (issue identifier) |
| `--before=<issue>` | Place before a specific issue (issue identifier) |
| `--output=json` | Output in JSON format |
| `--dry-run` | Show what would be moved without executing |

### Position Behavior

- No position specified: Issues are moved to the default position (typically bottom/end)
- `--position=top`: Uses `position: START`
- `--position=bottom`: Uses `position: END`
- `--position=N`: Uses `moveIssue` with numeric position (only works for single issue)
- `--after=<issue>`: Requires resolving the reference issue's PipelineIssue ID in the target pipeline
- `--before=<issue>`: Same as `--after` but uses `beforePipelineIssueId`

## Default Output Format

```
Moved 2 issues to "In Development":

  mpt#2451 Lock browser version...
  api#1662 Synchronize posting ids...
```

With `--dry-run`:

```
Would move 2 issues to "In Development":

  mpt#2451 Lock browser version... (currently in "Backlog")
  api#1662 Synchronize posting ids... (currently in "New Issues")
```

## GitHub API

**Not needed** - Issue movement is entirely a ZenHub concept. GitHub issues don't have pipelines.

However, if the `--repo` flag with branch names is used to identify PRs (as per the spec's identifier section), GitHub's API would be needed to resolve branch name to PR number.

## Limitations

### PipelineIssue ID Required for Bulk Moves

The `movePipelineIssues` mutation requires PipelineIssue IDs, not Issue IDs. This means:
1. An extra query is needed to resolve Issue IDs to PipelineIssue IDs
2. Closed issues don't have PipelineIssue records and cannot be moved (they must be reopened first)

### Numeric Position Only for Single Issue

The `moveIssue` mutation supports numeric position but only works for a single issue. To move multiple issues to specific positions, you'd need to call it multiple times, which could cause race conditions as each move shifts other issues.

### No Validation of Target Pipeline

The API doesn't prevent moving issues to any pipeline, including the Closed pipeline. Moving an issue to Closed via the API may not properly close the issue on GitHub.

### Cross-Workspace Moves

Issues can only be moved within the same workspace. Moving an issue to a pipeline in a different workspace requires `moveZenhubIssueToWorkspace`, which is a separate operation with different semantics.

## Related Subcommands

- **`zh issue show <issue>`** - View current pipeline before moving
- **`zh issue list --pipeline=<name>`** - List issues in source/target pipeline
- **`zh issue reopen <issue> --pipeline=<name>`** - Reopen closed issues into a pipeline
- **`zh pipeline show <name>`** - View issues currently in a pipeline

## Adjacent API Capabilities

### Move All Pipeline Issues

The `moveAllPipelineIssues` mutation moves all issues from source pipelines to a destination:

```graphql
mutation MoveAllPipelineIssues($input: MoveAllPipelineIssuesInput!) {
  moveAllPipelineIssues(input: $input) {
    destinationPipeline {
      id
      name
    }
  }
}
```

This could support a bulk operation like `zh pipeline move-all <from> <to>` or be used internally by `zh pipeline delete --into=<name>`.

### Priority Setting on Move

The `setPriorityOnPipelineIssues` mutation can set priority on multiple PipelineIssues. This could be combined with move to support a `--priority=<high|medium|low>` flag on move operations.
