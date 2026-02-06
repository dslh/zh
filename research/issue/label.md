# zh issue label

Add or remove labels from issues.

## Usage

```
zh issue label add <issue>... <label>...
zh issue label remove <issue>... <label>...
```

## API

### Adding labels

**Mutation:** `addLabelsToIssues`

```graphql
mutation AddLabelsToIssues($input: AddLabelsToIssuesInput!) {
  addLabelsToIssues(input: $input) {
    successCount
    failedIssues {
      id
      number
      title
    }
    labels {
      id
      name
      color
    }
    githubErrors
  }
}
```

**Variables (using label IDs):**

```json
{
  "input": {
    "issueIds": [
      "Z2lkOi8vcmFwdG9yL0lzc3VlLzI4ODQ3OTU",
      "Z2lkOi8vcmFwdG9yL0lzc3VlLzI5Mjg0ODA"
    ],
    "labelIds": [
      "Z2lkOi8vcmFwdG9yL0xhYmVsLzQ2NzgwOA",
      "Z2lkOi8vcmFwdG9yL0xhYmVsLzQ3OTYyMw"
    ]
  }
}
```

**Variables (using label names):**

```json
{
  "input": {
    "issueIds": [
      "Z2lkOi8vcmFwdG9yL0lzc3VlLzI4ODQ3OTU"
    ],
    "labelInfos": [
      { "name": "bug", "color": "d73a4a" },
      { "name": "enhancement" }
    ]
  }
}
```

The `labelInfos` alternative accepts:
- `name` (String) - Label name
- `color` (String) - Label color (optional, hex without #)

Note: When using `labelInfos`, if the label doesn't exist on the issue's repository, ZenHub may create it (behavior to verify during implementation).

### Removing labels

**Mutation:** `removeLabelsFromIssues`

```graphql
mutation RemoveLabelsFromIssues($input: RemoveLabelsFromIssuesInput!) {
  removeLabelsFromIssues(input: $input) {
    successCount
    failedIssues {
      id
      number
      title
    }
    labels {
      id
      name
      color
    }
    githubErrors
  }
}
```

**Variables:**

```json
{
  "input": {
    "issueIds": [
      "Z2lkOi8vcmFwdG9yL0lzc3VlLzI4ODQ3OTU"
    ],
    "labelIds": [
      "Z2lkOi8vcmFwdG9yL0xhYmVsLzQ2NzgwOA"
    ]
  }
}
```

Both `labelIds` and `labelInfos` are accepted (one is required).

### Querying available labels

Labels are repository-scoped (GitHub labels). To list labels for a repository:

```graphql
query GetRepositoryLabels($workspaceId: ID!) {
  workspace(id: $workspaceId) {
    repositoriesConnection(first: 100) {
      nodes {
        id
        ghId
        name
        ownerName
        labels(first: 100) {
          nodes {
            id
            ghId
            name
            color
            description
          }
        }
      }
    }
  }
}
```

### Querying workspace label options (aggregated)

For a combined view of all labels across workspace repositories:

```graphql
query GetWorkspaceLabelOptions($workspaceId: ID!) {
  workspace(id: $workspaceId) {
    issueLabelOptions(first: 100) {
      nodes {
        name
        color
      }
      totalCount
    }
  }
}
```

Note: `issueLabelOptions` returns only `name` and `color`, not the label ID. To get label IDs, query repository labels directly.

### Querying an issue's current labels

```graphql
query GetIssueLabels($repoGhId: Int!, $issueNumber: Int!) {
  issueByInfo(repositoryGhId: $repoGhId, issueNumber: $issueNumber) {
    id
    number
    title
    repository {
      name
      ownerName
    }
    labels {
      nodes {
        id
        ghId
        name
        color
        description
      }
    }
  }
}
```

### Resolving issue ID from repo/number

The mutations require ZenHub issue IDs. To resolve from repo/number:

```graphql
query ResolveIssueId($repoGhId: Int!, $issueNumber: Int!) {
  issueByInfo(repositoryGhId: $repoGhId, issueNumber: $issueNumber) {
    id
  }
}
```

## Cache requirements

| Data | Purpose |
|------|---------|
| Repository name → ghId mapping | Resolve `repo#123` format |
| Repository labels (per repo) | Map label names to IDs, avoid repeated lookups |

Labels are repository-scoped, so the cache should store labels per repository. Since labels can be created/modified in GitHub, consider a shorter cache TTL or invalidation strategy for labels.

## Flags and parameters

| Flag | Description |
|------|-------------|
| `--workspace` | Target workspace (if not using default) |
| `--dry-run` | Show what would be changed without executing |
| `--output=json` | Output in JSON format |
| `--create` | Create labels that don't exist (requires color for new labels) |

### Label identifier

The `<label>` argument should accept:
- Exact label name (e.g., "bug")
- Case-insensitive match (labels are case-insensitive on GitHub)

### Issue identifier

Standard issue identifier formats as per SPEC.md:
- ZenHub ID: `Z2lkOi8vcmFwdG9yL0lzc3VlLzU2Nzg5`
- GitHub format: `owner/repo#123` or `repo#123`

## Not available in ZenHub API

The ZenHub API delegates label operations to GitHub. The `githubErrors` field in the response indicates any GitHub-side failures.

**Label creation**: The `createGithubLabel` mutation exists in ZenHub's API for creating new labels:

```graphql
mutation CreateGithubLabel($input: CreateGithubLabelInput!) {
  createGithubLabel(input: $input) {
    label {
      id
      name
      color
    }
  }
}
```

This could support a `--create` flag to auto-create missing labels.

## GitHub API alternative

If ZenHub's API proves unreliable or limited, GitHub's GraphQL API provides direct label management:

**Add labels:**
```graphql
mutation AddLabels($labelableId: ID!, $labelIds: [ID!]!) {
  addLabelsToLabelable(input: { labelableId: $labelableId, labelIds: $labelIds }) {
    labelable {
      labels(first: 10) {
        nodes {
          name
        }
      }
    }
  }
}
```

**Remove labels:**
```graphql
mutation RemoveLabels($labelableId: ID!, $labelIds: [ID!]!) {
  removeLabelsFromLabelable(input: { labelableId: $labelableId, labelIds: $labelIds }) {
    labelable {
      labels(first: 10) {
        nodes {
          name
        }
      }
    }
  }
}
```

GitHub's API requires:
- `labelableId`: The GitHub node ID of the issue (available as `ghNodeId` from ZenHub)
- `labelIds`: GitHub label node IDs

The ZenHub approach is preferred since it:
1. Uses the same authentication as other `zh` commands
2. Returns ZenHub-specific error handling
3. Keeps the label cache in sync with ZenHub's view

## ZenHub Labels vs GitHub Labels

ZenHub has two label systems:

1. **GitHub Labels** (`Label` type) - Synced from GitHub, repository-scoped
   - Mutations: `addLabelsToIssues`, `removeLabelsFromIssues`
   - What users typically mean by "labels"

2. **ZenHub Labels** (`ZenhubLabel` type) - ZenHub-native, organization-scoped
   - Mutations: `addZenhubLabelsToIssues`, `removeZenhubLabelsFromIssues`
   - Less commonly used, typically for ZenHub-specific workflows

The `zh issue label` command should operate on GitHub labels by default, as these are what users see in both GitHub and ZenHub UIs.

## Related subcommands

Based on the API, these related commands could be useful:

| Command | Description |
|---------|-------------|
| `zh label list` | List all labels in the workspace (across all repos) |
| `zh label create` | Create a new label in a repository |
| `zh label delete` | Delete a label from a repository |

The workspace's `issueLabelOptions` field provides a convenient aggregated view of all labels across repositories, useful for a `zh label list` command.
