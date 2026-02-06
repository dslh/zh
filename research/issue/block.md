# zh issue block

Mark one issue/epic as blocking another issue/epic.

## API Feasibility

Fully supported via ZenHub's GraphQL API. There are two related systems in the API:

1. **`IssueDependency`** - Legacy system supporting only Issue-to-Issue blocking
2. **`Blockage`** - Newer system supporting Issue and ZenhubEpic as both blocker and blocked

The spec proposes `--type=issue|epic` for either side, which aligns with the `Blockage`/`createBlockage` approach.

## Mutations

### Creating a Blockage (Issue or Epic)

Use `createBlockage` for full flexibility (supports issues and epics on both sides):

```graphql
mutation CreateBlockage($input: CreateBlockageInput!) {
  createBlockage(input: $input) {
    blockage {
      id
      createdAt
      blocking {
        ... on Issue {
          __typename
          id
          number
          title
          repository {
            name
            ownerName
          }
        }
        ... on ZenhubEpic {
          __typename
          id
          title
        }
      }
      blocked {
        ... on Issue {
          __typename
          id
          number
          title
          repository {
            name
            ownerName
          }
        }
        ... on ZenhubEpic {
          __typename
          id
          title
        }
      }
    }
  }
}
```

Variables:
```json
{
  "input": {
    "blocking": {
      "id": "Z2lkOi8vcmFwdG9yL0lzc3VlLzEyMzQ1",
      "type": "ISSUE"
    },
    "blocked": {
      "id": "Z2lkOi8vcmFwdG9yL0lzc3VlLzY3ODkw",
      "type": "ISSUE"
    }
  }
}
```

The `type` field uses the `IssueDependencyField` enum:
- `ISSUE` - For GitHub or ZenHub issues
- `ZENHUB_EPIC` - For ZenHub epics

### Alternative: Issue-to-Issue Only

For issue-to-issue blocking specifically, there's also `createIssueDependency` which uses repository/issue number references directly:

```graphql
mutation CreateIssueDependency($input: CreateIssueDependencyInput!) {
  createIssueDependency(input: $input) {
    issueDependency {
      id
      createdAt
      blockingIssue {
        id
        number
        title
      }
      blockedIssue {
        id
        number
        title
      }
    }
  }
}
```

Variables:
```json
{
  "input": {
    "blockingIssue": {
      "repositoryGhId": 38994263,
      "issueNumber": 100
    },
    "blockedIssue": {
      "repositoryGhId": 38994263,
      "issueNumber": 101
    }
  }
}
```

This is simpler when both are issues since it accepts `repositoryGhId` + `issueNumber` directly without needing to resolve ZenHub IDs first.

## Resolving Identifiers

### For Issues

Use `issueByInfo` to resolve GitHub identifiers to ZenHub IDs:

```graphql
query GetIssueId($repoGhId: Int!, $issueNumber: Int!) {
  issueByInfo(repositoryGhId: $repoGhId, issueNumber: $issueNumber) {
    id
    number
    title
  }
}
```

### For Epics

Epics must be resolved by ZenHub ID or by searching. To find an epic by title substring:

```graphql
query SearchEpics($workspaceId: ID!, $query: String!) {
  workspace(id: $workspaceId) {
    zenhubEpics(first: 100, filters: { }) {
      nodes {
        id
        title
        state
      }
    }
  }
}
```

Then filter client-side by title match.

## Required Cached Data

- **Repository mappings**: `owner/name` -> `ghId` for all repos in the workspace
  - Required to translate `owner/repo#number` to `repositoryGhId`
- **Workspace ID**: Required for epic lookups

## Suggested Flags and Parameters

| Parameter | Description |
|-----------|-------------|
| `<blocker>` | The blocking issue/epic (required). Accepts ZenHub ID, `owner/repo#number`, `repo#number`, or epic title/substring |
| `<blocked>` | The blocked issue/epic (required). Same identifier formats |
| `--blocker-type` | Type of the blocker: `issue` (default) or `epic` |
| `--blocked-type` | Type of the blocked item: `issue` (default) or `epic` |

The original spec suggests `--type=issue|epic` for "either side", but having separate flags for each side provides clarity. Alternatively, a combined syntax could work:

| Alternative | Description |
|-------------|-------------|
| `--type` | Apply to both if both are the same type |
| Positional detection | Detect epics by ZenHub ID format or lack of `#` in identifier |

## Validation

Before calling the mutation, the CLI should verify:
1. The blocker identifier resolves to a valid issue or epic
2. The blocked identifier resolves to a valid issue or epic
3. The types specified match the actual resolved entities

## GitHub API Fallback

Not required for the core functionality. ZenHub's API can resolve issues by `repositoryGhId` and `issueNumber`.

However, if the CLI wants to validate that the GitHub issue exists or fetch additional metadata not in ZenHub, the GitHub API could supplement.

## Limitations

1. **No `deleteBlockage` mutation**: The API has `createBlockage` but no corresponding delete. For issue-to-issue dependencies, `deleteIssueDependency` exists. For blockages involving epics, there may be no API support for removal.

2. **No blocked filter in search**: The `IssueSearchFiltersInput` does not include a "blocked" or "blocking" filter, so `zh issue list --blocked` would need to:
   - Fetch all issues and filter client-side based on `blockingItems`/`blockedItems` fields, or
   - Not be supported

3. **Blockage ID not exposed on issues**: The `blockingItems`/`blockedItems` fields return the items themselves, not the Blockage object with its ID. This makes it harder to delete a specific blockage if that mutation existed.

## Related Functionality

### Querying Blockers on an Issue

The Issue type has these fields for viewing dependencies:

```graphql
query GetIssueBlockers($repoGhId: Int!, $issueNumber: Int!) {
  issueByInfo(repositoryGhId: $repoGhId, issueNumber: $issueNumber) {
    id
    number
    title
    # Issues blocking this one
    blockingItems(first: 50) {
      nodes {
        ... on Issue {
          __typename
          id
          number
          title
          state
          repository { name ownerName }
        }
        ... on ZenhubEpic {
          __typename
          id
          title
          state
        }
      }
    }
    # Issues this one is blocking
    blockedItems(first: 50) {
      nodes {
        ... on Issue {
          __typename
          id
          number
          title
          state
          repository { name ownerName }
        }
        ... on ZenhubEpic {
          __typename
          id
          title
          state
        }
      }
    }
    # Legacy fields (issues only)
    blockingIssues(first: 50) {
      nodes { id number title }
    }
    blockedIssues(first: 50) {
      nodes { id number title }
    }
  }
}
```

### ZenhubEpic Blocking Fields

ZenhubEpic also has `blockingItems` and `blockedItems`:

```graphql
query GetEpicBlockers($epicId: ID!) {
  node(id: $epicId) {
    ... on ZenhubEpic {
      id
      title
      blockingItems(first: 50) {
        nodes {
          ... on Issue { id number title }
          ... on ZenhubEpic { id title }
        }
      }
      blockedItems(first: 50) {
        nodes {
          ... on Issue { id number title }
          ... on ZenhubEpic { id title }
        }
      }
    }
  }
}
```

## Potential Related Subcommands

Based on the API, these related commands could be useful:

| Command | Description |
|---------|-------------|
| `zh issue unblock <blocker> <blocked>` | Remove a blocking relationship (if API supports) |
| `zh issue blockers <issue>` | List what's blocking an issue |
| `zh issue blocking <issue>` | List what an issue is blocking |
| `zh epic block` / `zh epic unblock` | Epic-specific blocking commands |

The `zh issue show` command should display blocking/blocked items as noted in the existing spec.
