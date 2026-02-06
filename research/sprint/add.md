# zh sprint add

Add issues to a sprint.

## Usage

```
zh sprint add <issue>...
zh sprint add <issue>... --sprint=<sprint>
```

## API

### Mutation: `addIssuesToSprints`

```graphql
mutation AddIssuesToSprints($input: AddIssuesToSprintsInput!) {
  addIssuesToSprints(input: $input) {
    sprintIssues {
      id
      issue {
        id
        number
        title
        repository {
          name
          owner {
            login
          }
        }
      }
      sprint {
        id
        name
        state
      }
    }
  }
}
```

**Input:**
```json
{
  "input": {
    "issueIds": ["Z2lkOi8v...issue1", "Z2lkOi8v...issue2"],
    "sprintIds": ["Z2lkOi8v...sprint"]
  }
}
```

Both `issueIds` and `sprintIds` are required arrays of ZenHub IDs. Typically only one sprint is targeted, but the API supports adding issues to multiple sprints simultaneously.

### Query: Get Active Sprint (default target)

When `--sprint` is not specified, the active sprint is used:

```graphql
query GetActiveSprint($workspaceId: ID!) {
  workspace(id: $workspaceId) {
    activeSprint {
      id
      name
      state
      startAt
      endAt
    }
  }
}
```

### Query: Resolve Sprint by Name/ID

To support sprint identifiers like "Sprint 42" or substring matching:

```graphql
query FindSprints($workspaceId: ID!, $query: String) {
  workspace(id: $workspaceId) {
    sprints(first: 50, query: $query, filters: { state: { eq: OPEN } }) {
      nodes {
        id
        name
        generatedName
        state
        startAt
        endAt
      }
    }
  }
}
```

### Query: Resolve Issue IDs

To convert `owner/repo#number` format to ZenHub IDs:

```graphql
query GetIssueByInfo($repositoryGhId: Int!, $issueNumber: Int!) {
  issueByInfo(repositoryGhId: $repositoryGhId, issueNumber: $issueNumber) {
    id
    number
    title
    state
    sprints(first: 10) {
      nodes {
        id
        name
        state
      }
    }
  }
}
```

## Cached Data Requirements

The following data should be cached for efficient lookups:

| Data | Purpose |
|------|---------|
| Workspace ID | Required for all sprint queries |
| Repository name → `ghId` mapping | Convert `repo#123` to `repositoryGhId` for issue lookup |
| Sprint ID → name mapping | Enable substring matching for `--sprint` flag |

## Flags and Parameters

| Flag | Description |
|------|-------------|
| `<issue>...` | One or more issue identifiers (required). Supports ZenHub ID, `owner/repo#number`, `repo#number` |
| `--sprint=<sprint>` | Target sprint. Defaults to active sprint. Supports: sprint ID, sprint name, unique substring, `current`/`next`/`previous` |
| `--dry-run` | Show what would be added without making changes |

## Implementation Notes

1. **Default sprint behavior**: When no `--sprint` flag is provided, use `workspace.activeSprint`. If no active sprint exists, error with a helpful message.

2. **Relative sprint references**:
   - `current` → `workspace.activeSprint`
   - `next` → `workspace.upcomingSprint`
   - `previous` → `workspace.previousSprint`

3. **Idempotency**: Adding an issue that's already in the sprint succeeds silently (API handles this gracefully).

4. **Bulk operations**: The API accepts multiple issues in a single call, so batch all provided issues into one mutation.

5. **Validation**: Before calling the mutation, verify:
   - All issues exist and are accessible
   - The target sprint exists and is OPEN (closed sprints may reject additions)

## Not Available in ZenHub API

None identified. The `addIssuesToSprints` mutation provides full functionality for this command.

## Related Subcommands

The API also supports:

- **`removeIssuesFromSprints`** mutation - for `zh sprint remove`
- **Sprint issues query** via `sprint.sprintIssues` - for `zh sprint show`
- **Sprint filtering** via `workspace.sprints(filters: {...})` - for `zh sprint list`
