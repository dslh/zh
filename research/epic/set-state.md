# zh epic set-state

Set the state of a ZenHub epic.

## Command

```
zh epic set-state <epic> <state>
```

Where `<state>` is one of: `open`, `todo`, `in_progress`, `closed`

## ZenHub API

### Mutation

```graphql
mutation UpdateZenhubEpicState($input: UpdateZenhubEpicStateInput!) {
  updateZenhubEpicState(input: $input) {
    zenhubEpic {
      id
      title
      state
    }
  }
}
```

**Variables:**
```json
{
  "input": {
    "zenhubEpicId": "Z2lkOi8vcmFwdG9yL1plbmh1YkVwaWMvMTIzNDU",
    "state": "IN_PROGRESS",
    "applyToIssues": false
  }
}
```

### Input Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `zenhubEpicId` | ID | Yes | The ZenHub epic ID |
| `state` | ZenhubEpicState | Yes | One of: `OPEN`, `TODO`, `IN_PROGRESS`, `CLOSED` |
| `applyToIssues` | Boolean | No | When true, also updates the state of child issues |

### Response

Returns the updated `ZenhubEpic` object with its new state.

## Epic Lookup

To resolve an epic identifier to a ZenHub ID, the CLI needs to query epics from the workspace:

```graphql
query GetWorkspaceEpics($workspaceId: ID!, $query: String) {
  workspace(id: $workspaceId) {
    zenhubEpics(first: 100, query: $query) {
      nodes {
        id
        title
        state
      }
    }
  }
}
```

Or look up a specific epic by ID:

```graphql
query GetEpicById($id: ID!) {
  node(id: $id) {
    ... on ZenhubEpic {
      id
      title
      state
    }
  }
}
```

## Flags

| Flag | Description |
|------|-------------|
| `--apply-to-issues` | Also update the state of all child issues in the epic |
| `--dry-run` | Show what would be changed without making changes |
| `--output=json` | Output result as JSON |

## Caching

**Required cached data:**
- Workspace ID (from config)
- Epic ID cache (optional, for faster lookups by title/substring)

The CLI should cache a mapping of epic titles to IDs (`epics-{workspace_id}.json`) to enable fast lookups by title or substring without querying the API each time.

## State Mapping

User input should be case-insensitive and mapped to the GraphQL enum:

| CLI Input | GraphQL Enum |
|-----------|--------------|
| `open` | `OPEN` |
| `todo` | `TODO` |
| `in_progress`, `in-progress`, `inprogress` | `IN_PROGRESS` |
| `closed` | `CLOSED` |

## Limitations

### Legacy Epics Not Supported

This mutation only works with **ZenHub Epics** (standalone epics). Legacy epics that are backed by GitHub issues do not have a `state` field in ZenHub - their state is controlled by the underlying GitHub issue.

For legacy epics, the CLI should:
1. Detect that the epic is a legacy epic (the `Epic` type has an `issue` field pointing to the backing GitHub issue)
2. Inform the user that state must be changed via GitHub
3. Optionally offer to close/reopen the backing GitHub issue using the GitHub API/CLI

### No Batch State Update

The API only supports updating one epic's state at a time. To update multiple epics, the CLI would need to make multiple API calls.

## Related Functionality

The `ZenhubEpic` type exposes additional state-related information that could be useful for related subcommands:

- `zenhubIssueCountProgress` - Progress based on issue count (closed vs total)
- `zenhubIssueEstimateProgress` - Progress based on estimates (completed vs total points)

These could support a `zh epic progress <epic>` subcommand showing completion status.

## Example Usage

```bash
# Set epic to in progress
zh epic set-state "Authentication Overhaul" in_progress

# Set epic to closed and also close all child issues
zh epic set-state auth-epic closed --apply-to-issues

# Preview changes without applying
zh epic set-state "Q1 Roadmap" closed --dry-run
```
