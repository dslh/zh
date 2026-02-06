# zh epic delete

Delete a ZenHub epic.

## API

### Mutation

```graphql
mutation DeleteZenhubEpic($input: DeleteZenhubEpicInput!) {
  deleteZenhubEpic(input: $input) {
    zenhubEpicId
  }
}
```

**Variables:**
```json
{
  "input": {
    "zenhubEpicId": "Z2lkOi8vcmFwdG9yL1plbmh1YkVwaWMvMTIzNDU"
  }
}
```

### Input Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `zenhubEpicId` | ID | Yes | The ZenHub epic ID to delete |

### Lookup Query (for resolving epic by title/substring)

```graphql
query FindEpic($workspaceId: ID!, $query: String) {
  workspace(id: $workspaceId) {
    zenhubEpics(first: 100, query: $query) {
      nodes {
        id
        title
        state
        childIssues(first: 1, workspaceId: $workspaceId) {
          totalCount
        }
      }
    }
  }
}
```

The `query` parameter performs a text search on epic titles. Fetching `childIssues.totalCount` is useful for displaying a warning to the user about issues that will be orphaned.

### Direct Lookup by ID

```graphql
query GetEpicById($id: ID!) {
  node(id: $id) {
    ... on ZenhubEpic {
      id
      title
      state
      childIssues(first: 1, workspaceId: "WORKSPACE_ID") {
        totalCount
      }
    }
  }
}
```

## Flags

| Flag | Description |
|------|-------------|
| `--force` / `-f` | Skip confirmation prompt |

## Behavior

1. Resolve the epic identifier to a ZenHub epic ID (by direct ID, title match, or substring)
2. Fetch the epic's details including child issue count for the confirmation prompt
3. Display confirmation prompt showing:
   - Epic title
   - Current state
   - Number of child issues that will be orphaned (removed from the epic)
4. If confirmed (or `--force` flag used), execute the delete mutation
5. Display success message with the deleted epic's title

### Confirmation Prompt Example

```
Delete epic "Q1 Feature Work"?
  State: IN_PROGRESS
  Child issues: 12 (will be removed from epic, not deleted)

This action cannot be undone. Continue? [y/N]
```

## Caching

- **Workspace ID** - Required to resolve epic by title/substring
- **Epic ID cache** - Could cache epic title -> ID mappings for faster subsequent lookups

## GitHub API

Not needed for deleting standalone ZenHub epics.

For legacy epics (backed by a GitHub issue), the behavior depends on implementation choice:
- The `oldIssue` field on `ZenhubEpic` indicates if an epic is backed by a GitHub issue
- Deleting a legacy epic via `deleteZenhubEpic` likely removes the epic designation but does NOT delete the underlying GitHub issue
- If the intent is to also close/delete the GitHub issue, that would require a separate GitHub API call

## Limitations

- **No undo** - Deleted epics cannot be recovered
- **Child issues are orphaned, not deleted** - Issues that belong to a deleted epic will simply no longer have that parent epic; they are not deleted themselves
- **Legacy epics** - The mutation deletes the ZenHub epic entity but may not affect the underlying GitHub issue for legacy epics. Need to verify exact behavior.
- **No cascade options** - There's no API option to close child issues when deleting an epic

## Related

The ZenhubEpic type has several relationships that may be of interest before deletion:

| Field | Description |
|-------|-------------|
| `childIssues` | Issues belonging to this epic |
| `blockedItems` | Dependencies blocked by this epic |
| `blockingItems` | Dependencies blocking this epic |
| `project` | Project this epic belongs to |
| `oldIssue` | For legacy epics: the backing GitHub issue |

A more sophisticated implementation could:
- Warn if the epic is blocking other items
- Warn if the epic belongs to a project
- Offer to reassign child issues to another epic before deletion
