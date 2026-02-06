# zh workspace repos

List repositories connected to the current workspace.

## ZenHub API Query

```graphql
query WorkspaceRepos($workspaceId: ID!, $first: Int!, $after: String) {
  workspace(id: $workspaceId) {
    repositoriesConnection(first: $first, after: $after) {
      totalCount
      pageInfo {
        hasNextPage
        endCursor
      }
      nodes {
        id
        ghId
        name
        ownerName
        description
        isPrivate
        isArchived
        isFavorite
        ghCreatedAt
        ghUpdatedAt
      }
    }
  }
}
```

### Variables

```json
{
  "workspaceId": "5c5c2662a623f9724788f533",
  "first": 100,
  "after": null
}
```

### Notes

- `repositoriesConnection` supports standard cursor-based pagination with `first`, `after`, `last`, `before`
- No filter arguments are available on `repositoriesConnection` - all repos are returned
- `totalCount` is available for showing "X repositories" summary
- There's also a `workspaceRepositories` connection that returns `WorkspaceRepository` objects with `readModeEnabled` and nested `repository` data, but `repositoriesConnection` is more direct

## Caching

The following should be cached per-workspace:

| Field | Purpose |
|-------|---------|
| `id` | ZenHub repository ID |
| `ghId` | GitHub repository ID (needed for issue lookups) |
| `name` | Repository name |
| `ownerName` | GitHub organization/user |

This cache is essential for resolving `repo#123` and `owner/repo#123` issue references to the `repositoryGhId` required by most ZenHub API calls.

## Suggested Flags

| Flag | Description |
|------|-------------|
| `--archived` | Include archived repositories (excluded by default) |
| `--favorites` | Show only favorited repositories |
| `--json` | Output as JSON (standard flag) |

## GitHub API Enrichment

ZenHub provides basic repository metadata, but GitHub's API can provide additional useful fields:

```graphql
query RepoDetails($owner: String!, $name: String!) {
  repository(owner: $owner, name: $name) {
    primaryLanguage { name }
    defaultBranchRef { name }
    pushedAt
    openIssues: issues(states: OPEN) { totalCount }
    closedIssues: issues(states: CLOSED) { totalCount }
    openPullRequests: pullRequests(states: OPEN) { totalCount }
  }
}
```

### Enrichment Flag

| Flag | Description |
|------|-------------|
| `--github` | Enrich output with GitHub data (language, issue counts, last push) |

This would require iterating over repos and making individual GitHub API calls, so it should be opt-in. Could be useful for a quick health check across all repos.

## Limitations

- No server-side filtering (can't filter by name, owner, or archived status via API)
- No sorting options in the API
- ZenHub doesn't track open/closed issue counts per repo - must use GitHub API

## Related Subcommands

The repository data from this command naturally suggests:

| Potential Command | Description |
|-------------------|-------------|
| `zh repo show <repo>` | Show detailed info about a single repo including labels, milestones, issue counts |
| `zh repo issues <repo>` | List issues for a specific repository |
| `zh repo add <repo>` | Add a GitHub repository to the workspace |
| `zh repo remove <repo>` | Remove a repository from the workspace |

These are not currently in the spec but the API appears to support them via mutations (not investigated in detail).
