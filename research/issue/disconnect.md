# zh issue disconnect

Disconnect a PR from an issue.

## API Feasibility

Fully supported via ZenHub's GraphQL API.

## Mutation

```graphql
mutation DeleteIssuePrConnection($input: DeleteIssuePrConnectionInput!) {
  deleteIssuePrConnection(input: $input) {
    issue {
      id
      number
      title
      connectedPrs {
        nodes {
          id
          number
          title
          repository {
            name
            ownerName
          }
        }
      }
    }
    pullRequest {
      id
      number
      title
    }
  }
}
```

Variables:
```json
{
  "input": {
    "issueId": "Z2lkOi8vcmFwdG9yL0lzc3VlLzM3NjQ3MDIxNw",
    "pullRequestId": "Z2lkOi8vcmFwdG9yL0lzc3VlLzM4NjEzNzczMQ"
  }
}
```

## Resolving Issue and PR Identifiers

Both `issueId` and `pullRequestId` require ZenHub's internal ID format. To resolve from GitHub identifiers (e.g., `mpt#123`), use `issueByInfo`:

```graphql
query GetIssueId($repoGhId: Int!, $issueNumber: Int!) {
  issueByInfo(repositoryGhId: $repoGhId, issueNumber: $issueNumber) {
    id
    pullRequest
  }
}
```

The `pullRequest` boolean field confirms whether the resolved item is a PR (should be `false` for the issue, `true` for the PR).

## Required Cached Data

- **Repository mappings**: `owner/name` -> `ghId` for all repos in the workspace
  - Fetch via `workspace.repositoriesConnection` and cache in `repos-{workspace_id}.json`
  - Required to translate `owner/repo#number` to `repositoryGhId` for the `issueByInfo` query

## Suggested Flags and Parameters

| Parameter | Description |
|-----------|-------------|
| `<issue>` | The issue to disconnect from (required). Accepts ZenHub ID or `owner/repo#number` or `repo#number` format |
| `<pr>` | The PR to disconnect (required). Same identifier formats as issue |

No additional flags appear necessary based on the API.

## Validation

Before calling the mutation, the CLI should verify:
1. The issue identifier resolves to an actual issue (`pullRequest: false`)
2. The PR identifier resolves to an actual PR (`pullRequest: true`)
3. Both exist in repos connected to the current workspace
4. The PR is currently connected to the issue (optional - the API may return an error if not connected)

## Verifying Connection Exists

To confirm a connection exists before attempting to disconnect, query the issue's connected PRs:

```graphql
query GetConnectedPrs($repoGhId: Int!, $issueNumber: Int!) {
  issueByInfo(repositoryGhId: $repoGhId, issueNumber: $issueNumber) {
    id
    connectedPrs {
      nodes {
        id
        number
        repository {
          ghId
        }
      }
    }
  }
}
```

The CLI can then verify the specified PR ID appears in `connectedPrs.nodes` before calling the mutation.

## GitHub API Fallback

Not required. ZenHub's `issueByInfo` query can resolve both issues and PRs by repository GitHub ID and number.

However, if the user specifies a PR by branch name (as mentioned in SPEC.md), the GitHub API would be needed to resolve the branch name to a PR number:

```bash
gh pr view <branch> --repo <owner/repo> --json number
```

## Limitations

None identified. The API fully supports deleting PR-to-issue connections.

## Related Functionality

This is the inverse of `zh issue connect`. The Issue type exposes:
- `connectedPrs` - PRs connected to this issue (useful for listing what can be disconnected)
- `connections` - Issues connected to this PR (inverse relationship, available on PR objects)

A potential enhancement would be `zh issue disconnect --all <issue>` to disconnect all PRs from an issue at once (would require multiple mutation calls).
