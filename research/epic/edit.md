# zh epic edit

Update the title and/or body of a ZenHub epic.

## API

### Mutation

```graphql
mutation UpdateZenhubEpic($input: UpdateZenhubEpicInput!) {
  updateZenhubEpic(input: $input) {
    zenhubEpic {
      id
      title
      body
      state
      updatedAt
    }
  }
}
```

**Variables:**
```json
{
  "input": {
    "zenhubEpicId": "Z2lkOi8vcmFwdG9yL1plbmh1YkVwaWMvMTIzNDU",
    "title": "New Epic Title",
    "body": "Updated description for the epic"
  }
}
```

### Input Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `zenhubEpicId` | ID | Yes | The ZenHub epic ID |
| `title` | String | No | New title for the epic |
| `body` | String | No | New body/description for the epic |

At least one of `title` or `body` should be provided.

### Lookup Query (for resolving epic by title/substring)

```graphql
query FindEpic($workspaceId: ID!, $query: String) {
  workspace(id: $workspaceId) {
    zenhubEpics(first: 100, query: $query) {
      nodes {
        id
        title
        body
        state
      }
    }
  }
}
```

The `query` parameter performs a text search on epic titles.

## Flags

| Flag | Description |
|------|-------------|
| `--title=<text>` | New title for the epic |
| `--body=<text>` | New body/description for the epic |
| `--body-file=<path>` | Read body content from a file (useful for longer descriptions) |

## Caching

- **Workspace ID** - Required to resolve epic by title/substring
- **Epic ID cache** - Could cache epic title -> ID mappings for faster subsequent lookups

## GitHub API

Not needed. ZenHub epics (standalone epics, not legacy issue-backed epics) are entirely managed through ZenHub's API.

## Limitations

- The `updateZenhubEpic` mutation only supports updating `title` and `body`. Other epic properties like state, dates, and assignees have dedicated mutations:
  - State: `updateZenhubEpicState`
  - Dates: `updateZenhubEpicDates`
  - Assignees: `addAssigneesToZenhubEpics` / `removeAssigneesFromZenhubEpics`
  - Labels: `addZenhubLabelsToZenhubEpics` / `removeZenhubLabelsFromZenhubEpics`

- Legacy epics (backed by a GitHub issue) may need different handling - their title/body would need to be updated through GitHub's API instead.

## Related Subcommands

Based on available mutations, these related epic operations are available:

- `zh epic set-state` - Uses `updateZenhubEpicState` mutation
- `zh epic set-dates` - Uses `updateZenhubEpicDates` mutation
- `zh epic assignee add/remove` - Uses `addAssigneesToZenhubEpics` / `removeAssigneesFromZenhubEpics`
- `zh epic label add/remove` - Uses `addZenhubLabelsToZenhubEpics` / `removeZenhubLabelsFromZenhubEpics`
- `zh epic estimate` - Uses `setMultipleEstimatesOnZenhubEpics` mutation
