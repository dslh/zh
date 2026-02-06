# zh issue priority

Set or clear the priority on one or more issues.

## Usage

```
zh issue priority <issue>... <priority>
zh issue priority <issue>...              # Clear priority (omit priority argument)
```

## API

### Setting priority

**Mutation:** `setIssueInfoPriorities`

```graphql
mutation SetIssuePriority($input: SetIssueInfoPrioritiesInput!) {
  setIssueInfoPriorities(input: $input) {
    pipelineIssues {
      id
      priority {
        id
        name
        color
      }
      issue {
        id
        number
        title
        repository {
          name
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
    "priorityId": "Z2lkOi8vcmFwdG9yL1ByaW9yaXR5LzMzMDcyMQ",
    "issues": [
      { "repositoryGhId": 4925400, "issueNumber": 1234 },
      { "repositoryGhId": 4925400, "issueNumber": 5678 }
    ]
  }
}
```

The `issues` parameter uses `IssueInfoInput`, which accepts:
- `repositoryGhId` (Int) - GitHub repository ID
- `repositoryId` (ID) - ZenHub repository ID (alternative to `repositoryGhId`)
- `issueNumber` (Int!) - Required issue number

### Clearing priority

**Mutation:** `removeIssueInfoPriorities`

```graphql
mutation RemoveIssuePriority($input: RemoveIssueInfoPrioritiesInput!) {
  removeIssueInfoPriorities(input: $input) {
    pipelineIssues {
      id
      priority {
        id
        name
      }
      issue {
        id
        number
        title
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
    "issues": [
      { "repositoryGhId": 4925400, "issueNumber": 1234 }
    ]
  }
}
```

### Querying available priorities

Priorities are workspace-specific. To list available priorities:

```graphql
query GetWorkspacePriorities($workspaceId: ID!) {
  workspace(id: $workspaceId) {
    prioritiesConnection {
      nodes {
        id
        name
        color
        description
      }
    }
  }
}
```

### Querying an issue's current priority

Priority lives on `PipelineIssue`, not `Issue` directly (since it's workspace-specific):

```graphql
query GetIssuePriority($repoGhId: Int!, $issueNumber: Int!, $workspaceId: ID!) {
  issueByInfo(repositoryGhId: $repoGhId, issueNumber: $issueNumber) {
    id
    number
    title
    pipelineIssue(workspaceId: $workspaceId) {
      id
      priority {
        id
        name
        color
      }
    }
  }
}
```

## Cache requirements

The following should be cached for efficient command execution:

| Data | Purpose |
|------|---------|
| Workspace priorities | Map priority names/substrings to IDs |
| Repository name → ghId mapping | Resolve `repo#123` format to API parameters |

Priorities should be cached per-workspace since they're workspace-specific.

## Flags and parameters

| Flag | Description |
|------|-------------|
| `--workspace` | Target workspace (if not using default) |
| `--dry-run` | Show what would be changed without executing |
| `--output=json` | Output in JSON format |

### Priority identifier

The `<priority>` argument should accept:
- Exact priority name (e.g., "High priority")
- Case-insensitive substring match (e.g., "high")
- Priority ID (e.g., `Z2lkOi8vcmFwdG9yL1ByaW9yaXR5LzMzMDcyMQ`)

## Not available in ZenHub API

- **Creating/deleting priorities**: There are no mutations to create, update, or delete priority definitions. Priorities must be managed through the ZenHub web UI. The CLI can only set/clear priorities on issues using existing priority definitions.

## Not available without GitHub API

Nothing critical. The ZenHub API provides all necessary functionality for this command.

## Alternative mutation

There's also `setPriorityOnPipelineIssues` which takes `pipelineIssueIds` instead of `IssueInfoInput`:

```graphql
mutation SetPriorityOnPipelineIssues($input: SetPriorityOnPipelineIssuesInput!) {
  setPriorityOnPipelineIssues(input: $input) {
    pipelineIssues {
      id
      priority {
        id
        name
      }
    }
  }
}
```

**Variables:**

```json
{
  "input": {
    "priorityId": "Z2lkOi8vcmFwdG9yL1ByaW9yaXR5LzMzMDcyMQ",
    "pipelineIssueIds": ["Z2lkOi8vcmFwdG9yL1BpcGVsaW5lSXNzdWUvOTA3MzQ3MQ"]
  }
}
```

To clear priority with this mutation, pass `null` for `priorityId`.

This mutation is useful when you already have the `PipelineIssue` ID (e.g., from a previous query). However, `setIssueInfoPriorities` is more convenient for CLI usage since it accepts repo/issue number directly.

## Related subcommands

Based on the API, these related commands could be useful:

| Command | Description |
|---------|-------------|
| `zh priority list` | List available priorities in the workspace |
| `zh priority create` | Not possible via API (manage in web UI) |

The `zh priority list` command would help users discover available priority values before setting them.
