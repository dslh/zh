# zh epic alias

Set a shorthand name that can be used to reference the epic in future calls to `zh`.

## Feasibility

**Fully Feasible** - This is primarily a local configuration operation. The only API interaction required is verifying the epic exists.

## API Query

### Epic Existence Verification

Before setting an alias, verify the epic exists. The approach depends on how the user specifies the epic:

#### By ZenHub ID

Use the `node` interface to verify the epic exists:

```graphql
query VerifyEpicById($id: ID!) {
  node(id: $id) {
    __typename
    ... on ZenhubEpic {
      id
      title
    }
    ... on Epic {
      id
      issue {
        title
        number
        repository {
          name
          ownerName
        }
      }
    }
  }
}
```

#### By Title or Substring

Search the workspace roadmap for matching epics:

```graphql
query FindEpicByTitle($workspaceId: ID!, $query: String!) {
  workspace(id: $workspaceId) {
    roadmap {
      items(first: 50, query: $query) {
        nodes {
          __typename
          ... on ZenhubEpic {
            id
            title
          }
          ... on Epic {
            id
            issue {
              title
              number
              repository {
                name
                ownerName
              }
            }
          }
        }
      }
    }
  }
}
```

If exactly one match is found, use that epic's ID. If multiple matches are found, require the user to be more specific.

#### By GitHub Issue Reference (Legacy Epics)

For legacy epics specified as `owner/repo#number`, first resolve the issue, then check if it's an epic:

```graphql
query GetIssueForEpic($repositoryGhId: Int!, $issueNumber: Int!) {
  issueByInfo(repositoryGhId: $repositoryGhId, issueNumber: $issueNumber) {
    id
    parentZenhubEpics(first: 1) {
      nodes {
        id
      }
    }
  }
}
```

Note: The issue itself may be the epic. To verify, check if the roadmap contains an Epic with this issue, or query the issue and see if it has child issues via the Epic type.

## Implementation

This command modifies the local config file (`~/.config/zh/config.yml`):

```yaml
aliases:
  epics:
    auth: "Z2lkOi8vcmFwdG9yL1plbmh1YkVwaWMvMTIzNDU"
    stripe: "Z2lkOi8vcmFwdG9yL0VwaWMvMTE4NzcxMw"
    ts: "Z2lkOi8vcmFwdG9yL0VwaWMvMTE4NjA5Ng"
```

The alias can then be used anywhere an epic identifier is accepted:

```bash
zh epic show auth                  # Shows the "auth" epic
zh epic add stripe mpt#123         # Adds issue to the Stripe integration epic
zh issue list --epic=ts            # Lists issues in the TypeScript migration epic
zh epic set-state auth closed      # Closes the auth epic
```

## Suggested Flags

| Flag | Description |
|------|-------------|
| `--delete` | Remove an existing alias |
| `--list` | List all epic aliases (alternative to inspecting config) |
| `--force` | Overwrite an existing alias without prompting |

## Validation Rules

1. **Epic must exist** - Verify the target epic exists in the workspace (via API query)
2. **Alias must be unique** - Cannot conflict with another epic alias
3. **Alias cannot shadow epic titles** - Warn if alias matches an existing epic's exact title
4. **Reserved words** - Reject aliases that match command names or flags (e.g., `list`, `show`, `--help`)
5. **Valid identifier** - Alias should be a simple alphanumeric string (with hyphens/underscores allowed)

## Error Cases

| Scenario | Exit Code | Message |
|----------|-----------|---------|
| Epic not found | 4 | `Error: Epic "xyz" not found in workspace` |
| Multiple epics match | 2 | `Error: "typescript" matches multiple epics. Be more specific or use an ID.` |
| Alias already exists (without --force) | 2 | `Error: Alias "ts" already exists. Use --force to overwrite.` |
| Alias shadows epic title | 0 (warning) | `Warning: Alias "Q1 Roadmap" matches an existing epic title` |
| Invalid alias format | 2 | `Error: Alias must contain only letters, numbers, hyphens, and underscores` |

## Caching Requirements

| Data | Purpose |
|------|---------|
| Workspace ID | Required for roadmap search query |
| Repository ghId mappings | To resolve `owner/repo#number` format for legacy epics |

If epics are cached locally (which they could be for faster title/substring lookup), the alias validation could potentially use the cache rather than querying the API. However, the cache should be verified against the API at least once to ensure the epic still exists.

## GitHub API Requirements

None required. All epic data is available from ZenHub's API.

For legacy epics specified by GitHub issue reference, the `repositoryGhId` must be resolved from the cached repo mappings or fetched via:

```graphql
query GetRepoId($workspaceId: ID!) {
  workspace(id: $workspaceId) {
    repositories(first: 100) {
      nodes {
        id
        ghId
        name
        ownerName
      }
    }
  }
}
```

## Limitations

None. This is a straightforward local configuration operation with a simple existence check.

## Related Features

### Alias Portability

Aliases are stored in the local config file and are not synced across machines. Users who work on multiple machines would need to manually copy their config or set up aliases on each machine.

### Alias Suggestions

When a user references an epic frequently (e.g., uses `zh epic show "Typescript migration 32.07%"` multiple times), the CLI could suggest:

```
Tip: Create an alias for faster access: zh epic alias "Typescript migration 32.07%" ts
```

### Alias Tab Completion

Shell completions should include epic aliases alongside other epic identifiers for commands that accept epic arguments.
