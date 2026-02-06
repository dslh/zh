# zh issue close

Close one or more issues.

## Usage

```
zh issue close <issue>...
```

- `<issue>...` - One or more issue identifiers (ZenHub ID, owner/repo#number, repo#number)

## Feasibility

**Fully Feasible** - The ZenHub GraphQL API provides a `closeIssues` mutation that supports closing multiple issues in a single call. The mutation also closes the corresponding GitHub issues.

## ZenHub API

### Closing Issues

**Mutation:** `closeIssues`

```graphql
mutation CloseIssues($input: CloseIssuesInput!) {
  closeIssues(input: $input) {
    successCount
    failedIssues {
      id
      number
      title
      repository {
        name
        ownerName
      }
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
      "Z2lkOi8vcmFwdG9yL0lzc3VlLzEyMzQ1",
      "Z2lkOi8vcmFwdG9yL0lzc3VlLzY3ODkw"
    ]
  }
}
```

### CloseIssuesInput Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `issueIds` | [ID!]! | Yes | List of ZenHub Issue IDs to close |
| `clientMutationId` | String | No | Optional client identifier for the mutation |

### Response Fields

| Field | Type | Description |
|-------|------|-------------|
| `successCount` | Int! | Number of issues successfully closed |
| `failedIssues` | [Issue!]! | Issues that failed to close |
| `githubErrors` | JSON! | Any errors from GitHub when closing the underlying issues |

### Resolving Issue Identifiers

When the user provides a GitHub-style identifier (e.g., `mpt#1234`), resolve it to a ZenHub ID:

```graphql
query GetIssueByInfo($repoGhId: Int!, $issueNumber: Int!) {
  issueByInfo(repositoryGhId: $repoGhId, issueNumber: $issueNumber) {
    id
    number
    title
    state
    repository {
      name
      ownerName
    }
  }
}
```

For multiple issues, batch them where possible:

```graphql
query GetIssues($ids: [ID!]!) {
  issues(ids: $ids) {
    id
    number
    title
    state
    repository {
      name
      ownerName
    }
  }
}
```

## Implementation Flow

1. **Parse issue identifiers** - Resolve each issue identifier to a ZenHub Issue ID:
   - If ZenHub ID provided directly, validate it
   - If `owner/repo#number` or `repo#number` format, use `issueByInfo` query with cached `repositoryGhId`

2. **Validate state** - Optionally check that issues are currently open. Warn if any are already closed.

3. **Execute mutation** - Call `closeIssues` with the collected Issue IDs

4. **Report results** - Show success count and any failures

## Cached Information

| Data | Purpose |
|------|---------|
| Repositories | Map repo name to `ghId` for `issueByInfo` lookups |

## Suggested Flags

| Flag | Description |
|------|-------------|
| `--dry-run` | Show what would be closed without executing |
| `--output=json` | Output in JSON format |

## Output

### Success (single issue)
```
Closed mpt#1234: Fix login button alignment
```

### Success (multiple issues)
```
Closed 3 issues:

  mpt#1234 Fix login button alignment
  mpt#1235 Update error messages
  api#567  Add rate limiting headers
```

### Partial success
```
Closed 2 of 3 issues:

  mpt#1234 Fix login button alignment
  mpt#1235 Update error messages

Failed to close:

  api#568  Permission denied
```

### Dry-run
```
Would close 2 issues:

  mpt#1234 Fix login button alignment (open)
  mpt#1235 Update error messages (open)
```

### Already closed warning
```
1 issue already closed:

  mpt#1236 Old bug fix

Closed 2 issues:

  mpt#1234 Fix login button alignment
  mpt#1235 Update error messages
```

### JSON output
```json
{
  "closed": [
    {
      "id": "Z2lkOi8vcmFwdG9yL0lzc3VlLzEyMzQ1",
      "number": 1234,
      "repository": "gohiring/mpt",
      "title": "Fix login button alignment"
    }
  ],
  "failed": [],
  "alreadyClosed": [],
  "successCount": 1
}
```

## Error Cases

- Issue not found
- Issue already closed (warn but continue with others)
- No permission to close issue
- GitHub API errors (reported via `githubErrors` field)

## GitHub API

**Not strictly needed** - The ZenHub `closeIssues` mutation handles closing issues on both ZenHub and GitHub.

However, if GitHub CLI integration is enabled, could optionally use `gh issue close` for:
- Closing with a comment: `gh issue close <number> --comment "Closing via zh"`
- More detailed error messages from GitHub

The ZenHub mutation's `githubErrors` field should provide sufficient error information in most cases.

## Limitations

### No Close Reason

GitHub supports closing issues with a reason (`completed` or `not_planned`). The ZenHub API does not expose this option - issues are closed without a specific reason.

If close reason support is needed, the GitHub API would need to be used directly:
```bash
gh issue close <number> --reason "not planned"
```

### No Close Comment

The ZenHub `closeIssues` mutation does not support adding a comment when closing. To close with a comment, use the GitHub API separately.

### Bulk Operation Atomicity

The `closeIssues` mutation is not atomic - some issues may close successfully while others fail. The response includes both `successCount` and `failedIssues` to report partial success.

## Related Subcommands

- **`zh issue reopen`** - Reopen closed issues into a pipeline
- **`zh issue show`** - View issue details including current state
- **`zh issue list --pipeline=Closed`** - List closed issues

## Adjacent API Capabilities

### Delete Issue

For ZenHub-native issues (not backed by GitHub), there's a `deleteZenhubIssue` mutation. This is different from closing and completely removes the issue. This could support a future `zh issue delete` command for ZenHub-only issues.
