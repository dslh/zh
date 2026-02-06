# zh epic remove

Remove issues from a ZenHub epic.

## Command

```
zh epic remove <epic> <issue>...
```

Remove one or more issues from the specified epic.

## ZenHub API

### Mutation

```graphql
mutation RemoveIssuesFromZenhubEpics($input: RemoveIssuesFromZenhubEpicsInput!) {
  removeIssuesFromZenhubEpics(input: $input) {
    zenhubEpics {
      id
      title
      childIssues(workspaceId: $workspaceId, first: 100) {
        totalCount
        nodes {
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
      }
    }
  }
}
```

**Variables:**
```json
{
  "input": {
    "zenhubEpicIds": ["Z2lkOi8vcmFwdG9yL1plbmh1YkVwaWMvMTIzNDU"],
    "issueIds": [
      "Z2lkOi8vcmFwdG9yL0lzc3VlLzU2Nzg5",
      "Z2lkOi8vcmFwdG9yL0lzc3VlLzY3ODkw"
    ]
  },
  "workspaceId": "5c5c2662a623f9724788f533"
}
```

### Input Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `zenhubEpicIds` | [ID!]! | Yes | Array of ZenHub epic IDs to remove issues from |
| `issueIds` | [ID!]! | Yes | Array of issue IDs to remove from the epics |
| `clientMutationId` | String | No | Optional client identifier for the mutation |

### Response

Returns an array of updated `ZenhubEpic` objects. The `childIssues` field requires a `workspaceId` argument.

## Issue Lookup

To resolve issue identifiers (e.g., `owner/repo#123` or `repo#123`) to ZenHub IDs:

### By Repository GitHub ID and Issue Number

```graphql
query GetIssueByInfo($repositoryGhId: Int!, $issueNumber: Int!) {
  issueByInfo(repositoryGhId: $repositoryGhId, issueNumber: $issueNumber) {
    id
    number
    title
    state
    repository {
      ghId
      name
      owner {
        login
      }
    }
    parentZenhubEpics {
      nodes {
        id
        title
      }
    }
  }
}
```

### Batch Lookup by IDs

```graphql
query GetIssues($ids: [ID!]!) {
  issues(ids: $ids) {
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
}
```

## Epic Lookup

To resolve an epic identifier to a ZenHub ID and list its current child issues:

```graphql
query GetWorkspaceEpic($workspaceId: ID!, $query: String) {
  workspace(id: $workspaceId) {
    zenhubEpics(first: 100, query: $query) {
      nodes {
        id
        title
        state
        childIssues(workspaceId: $workspaceId, first: 100) {
          nodes {
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
        }
      }
    }
  }
}
```

## Flags

| Flag | Description |
|------|-------------|
| `--repo=<repo>` | Specify repository for issue numbers (e.g., `--repo=mpt 123 456`) |
| `--all` | Remove all issues from the epic |
| `--dry-run` | Show what would be removed without making changes |
| `--output=json` | Output result as JSON |

## Caching

**Required cached data:**
- Workspace ID (from config)
- Repository name to GitHub ID mapping (`repos-{workspace_id}.json`) - required for resolving `repo#123` format
- Epic title to ID mapping (optional, for faster lookups by title/substring)

**Cache lookup flow for issues:**
1. If ZenHub ID provided (starts with `Z2lk`), use directly
2. If `owner/repo#123` format, look up repo GitHub ID from cache, then query `issueByInfo`
3. If `repo#123` format, look up repo GitHub ID from cache (error if ambiguous), then query `issueByInfo`
4. If using `--repo` flag, look up repo once and use for all bare issue numbers

## Limitations

### Legacy Epics Not Supported

The `removeIssuesFromZenhubEpics` mutation only works with **ZenHub Epics** (standalone epics). Legacy epics that are backed by GitHub issues cannot have child issues removed via this mutation.

For legacy epics, the CLI should:
1. Detect that the epic is a legacy epic (query `workspace.epics` which returns `Epic` type with an `issue` field)
2. Inform the user that this epic type doesn't support removing child issues via the API
3. Suggest using the ZenHub web UI or converting to a ZenHub Epic

### No Error on Non-Member Issues

The API doesn't appear to error when removing an issue that isn't a child of the epic. The CLI may want to:
1. Query existing child issues first
2. Validate that the specified issues are actually in the epic
3. Warn the user about issues that aren't in the epic

Or simply let the API handle it silently.

### Batch Operation

The mutation supports removing issues from multiple epics in a single call. The CLI could theoretically support this, but the spec only calls for removing issues from a single epic.

## Related Functionality

### Verifying Membership

Before removing issues, it may be useful to verify they're actually in the epic. Use the `parentZenhubEpics` field on Issue:

```graphql
query CheckIssueEpics($id: ID!) {
  node(id: $id) {
    ... on Issue {
      id
      number
      title
      parentZenhubEpics {
        nodes {
          id
          title
        }
      }
    }
  }
}
```

### Remove All Issues

The `--all` flag should:
1. Query all child issues of the epic
2. Extract their IDs
3. Call the mutation with all issue IDs

```graphql
query GetEpicChildIssues($id: ID!, $workspaceId: ID!) {
  node(id: $id) {
    ... on ZenhubEpic {
      id
      title
      childIssues(workspaceId: $workspaceId, first: 500) {
        nodes {
          id
          number
          title
        }
      }
    }
  }
}
```

## Example Usage

```bash
# Remove a single issue from an epic
zh epic remove "Authentication Overhaul" mpt#1234

# Remove multiple issues
zh epic remove "Q1 Roadmap" mpt#1234 mpt#1235 mpt#1236

# Remove issues from different repos
zh epic remove "Cross-Team Initiative" gohiring/mpt#1234 gohiring/api#567

# Using --repo flag for multiple issues from same repo
zh epic remove "Sprint 42" --repo=mpt 1234 1235 1236 1237

# Remove all issues from an epic
zh epic remove "Abandoned Epic" --all

# Preview without making changes
zh epic remove "My Epic" mpt#1234 --dry-run

# Using ZenHub IDs directly
zh epic remove Z2lkOi8vcmFwdG9yL1plbmh1YkVwaWMvMTIzNDU Z2lkOi8vcmFwdG9yL0lzc3VlLzU2Nzg5
```

## Output

Default output should confirm the operation:

```
Removed 3 issues from epic "Q1 Roadmap"
  - mpt#1234: Fix login bug
  - mpt#1235: Update user settings
  - mpt#1236: Add password reset flow

Epic now has 12 child issues
```

With `--all`:

```
Removed all 15 issues from epic "Abandoned Epic"

Epic now has 0 child issues
```

With `--output=json`:

```json
{
  "epic": {
    "id": "Z2lkOi8vcmFwdG9yL1plbmh1YkVwaWMvMTIzNDU",
    "title": "Q1 Roadmap"
  },
  "removed": [
    {"id": "Z2lkOi8vcmFwdG9yL0lzc3VlLzU2Nzg5", "number": 1234, "repo": "gohiring/mpt", "title": "Fix login bug"},
    {"id": "Z2lkOi8vcmFwdG9yL0lzc3VlLzY3ODkw", "number": 1235, "repo": "gohiring/mpt", "title": "Update user settings"},
    {"id": "Z2lkOi8vcmFwdG9yL0lzc3VlLzc4OTAx", "number": 1236, "repo": "gohiring/mpt", "title": "Add password reset flow"}
  ],
  "totalChildIssues": 12
}
```
