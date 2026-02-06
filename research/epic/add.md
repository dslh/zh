# zh epic add

Add issues to a ZenHub epic.

## Command

```
zh epic add <epic> <issue>...
```

Add one or more issues to the specified epic.

## ZenHub API

### Mutation

```graphql
mutation AddIssuesToZenhubEpics($input: AddIssuesToZenhubEpicsInput!) {
  addIssuesToZenhubEpics(input: $input) {
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
| `zenhubEpicIds` | [ID!]! | Yes | Array of ZenHub epic IDs to add issues to |
| `issueIds` | [ID!]! | Yes | Array of issue IDs to add to the epics |
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

To resolve an epic identifier to a ZenHub ID:

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

Or look up by ID directly:

```graphql
query GetEpicById($id: ID!) {
  node(id: $id) {
    ... on ZenhubEpic {
      id
      title
      state
      childIssues(workspaceId: $workspaceId, first: 100) {
        totalCount
      }
    }
  }
}
```

## Flags

| Flag | Description |
|------|-------------|
| `--repo=<repo>` | Specify repository for issue numbers (e.g., `--repo=mpt 123 456`) |
| `--dry-run` | Show what would be added without making changes |
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

The `addIssuesToZenhubEpics` mutation only works with **ZenHub Epics** (standalone epics). Legacy epics that are backed by GitHub issues cannot have child issues added via this mutation.

For legacy epics, the CLI should:
1. Detect that the epic is a legacy epic (query `workspace.epics` which returns `Epic` type with an `issue` field)
2. Inform the user that this epic type doesn't support adding child issues via the API
3. Suggest using the ZenHub web UI or converting to a ZenHub Epic

### Batch Operation

The mutation supports adding multiple issues to multiple epics in a single call. The CLI could support this with syntax like:

```bash
zh epic add epic1 epic2 --issues issue1 issue2 issue3
```

However, the spec only calls for adding issues to a single epic, so the typical usage will pass one epic ID in the array.

### No Duplicate Detection

The API doesn't appear to error when adding an issue that's already a child of the epic. The CLI may want to:
1. Query existing child issues first
2. Filter out issues already in the epic
3. Warn the user about duplicates

Or simply let the API handle it silently.

### Issue Must Exist in Workspace

Issues must be from repositories connected to the same ZenHub organization. The API will error if trying to add issues from unconnected repositories.

## Related Functionality

### Parent Epic Information

When adding issues, it may be useful to show if issues are already part of other epics. The `Issue` type has a `parentZenhubEpics` field:

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

### Sub-Issues (GitHub Native)

Note: ZenHub also supports GitHub's native sub-issues feature via `addSubIssues` / `removeSubIssues` mutations. This is separate from the epic-child relationship and represents GitHub's built-in parent-child issue tracking.

## Example Usage

```bash
# Add a single issue to an epic
zh epic add "Authentication Overhaul" mpt#1234

# Add multiple issues
zh epic add "Q1 Roadmap" mpt#1234 mpt#1235 mpt#1236

# Add issues from different repos
zh epic add "Cross-Team Initiative" gohiring/mpt#1234 gohiring/api#567

# Using --repo flag for multiple issues from same repo
zh epic add "Sprint 42" --repo=mpt 1234 1235 1236 1237

# Preview without making changes
zh epic add "My Epic" mpt#1234 --dry-run

# Using ZenHub IDs directly
zh epic add Z2lkOi8vcmFwdG9yL1plbmh1YkVwaWMvMTIzNDU Z2lkOi8vcmFwdG9yL0lzc3VlLzU2Nzg5
```

## Output

Default output should confirm the operation:

```
Added 3 issues to epic "Q1 Roadmap"
  - mpt#1234: Fix login bug
  - mpt#1235: Update user settings
  - mpt#1236: Add password reset flow

Epic now has 15 child issues
```

With `--output=json`:

```json
{
  "epic": {
    "id": "Z2lkOi8vcmFwdG9yL1plbmh1YkVwaWMvMTIzNDU",
    "title": "Q1 Roadmap"
  },
  "added": [
    {"id": "Z2lkOi8vcmFwdG9yL0lzc3VlLzU2Nzg5", "number": 1234, "repo": "gohiring/mpt", "title": "Fix login bug"},
    {"id": "Z2lkOi8vcmFwdG9yL0lzc3VlLzY3ODkw", "number": 1235, "repo": "gohiring/mpt", "title": "Update user settings"},
    {"id": "Z2lkOi8vcmFwdG9yL0lzc3VlLzc4OTAx", "number": 1236, "repo": "gohiring/mpt", "title": "Add password reset flow"}
  ],
  "totalChildIssues": 15
}
```
