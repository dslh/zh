# zh issue reopen

Reopen closed issues into a pipeline.

## Usage

```
zh issue reopen <issue>... --pipeline=<name>
```

- `<issue>...` - One or more issue identifiers (ZenHub ID, owner/repo#number, repo#number)
- `--pipeline=<name>` - Target pipeline for the reopened issues (required)

## Feasibility

**Fully Feasible** - The ZenHub GraphQL API provides a `reopenIssues` mutation that supports reopening multiple issues and placing them in a specified pipeline. The mutation also reopens the corresponding GitHub issues.

## ZenHub API

### Reopening Issues

**Mutation:** `reopenIssues`

```graphql
mutation ReopenIssues($input: ReopenIssuesInput!) {
  reopenIssues(input: $input) {
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
    ],
    "pipelineId": "Z2lkOi8vcmFwdG9yL1BpcGVsaW5lLzE5MTY5NzY",
    "position": "START"
  }
}
```

### ReopenIssuesInput Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `issueIds` | [ID!]! | Yes | List of ZenHub Issue IDs to reopen |
| `pipelineId` | ID! | Yes | Target pipeline for reopened issues |
| `position` | PipelineIssuePosition! | Yes | Position in pipeline: `START` or `END` |
| `clientMutationId` | String | No | Optional client identifier for the mutation |

### PipelineIssuePosition Enum

| Value | Description |
|-------|-------------|
| `START` | Place at the top of the pipeline |
| `END` | Place at the bottom of the pipeline |

### Response Fields

| Field | Type | Description |
|-------|------|-------------|
| `successCount` | Int! | Number of issues successfully reopened |
| `failedIssues` | [Issue!]! | Issues that failed to reopen |
| `githubErrors` | JSON! | Any errors from GitHub when reopening the underlying issues |

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

### Resolving Pipeline Identifier

Pipelines should be resolved from cache. If needed, fetch from workspace:

```graphql
query GetWorkspacePipelines($workspaceId: ID!) {
  workspace(id: $workspaceId) {
    pipelinesConnection {
      nodes {
        id
        name
      }
    }
  }
}
```

## Implementation Flow

1. **Parse issue identifiers** - Resolve each issue identifier to a ZenHub Issue ID:
   - If ZenHub ID provided directly, validate it
   - If `owner/repo#number` or `repo#number` format, use `issueByInfo` query with cached `repositoryGhId`

2. **Validate state** - Optionally check that issues are currently closed. Warn if any are already open.

3. **Resolve pipeline** - Look up the target pipeline ID from cache using the provided name/identifier

4. **Execute mutation** - Call `reopenIssues` with the collected Issue IDs, pipeline ID, and position

5. **Report results** - Show success count, target pipeline, and any failures

## Cached Information

| Data | Purpose |
|------|---------|
| Repositories | Map repo name to `ghId` for `issueByInfo` lookups |
| Pipelines | Resolve pipeline name/substring to ID |
| Workspace ID | Required for pipeline lookup |

## Suggested Flags

| Flag | Description |
|------|-------------|
| `--pipeline=<name>` | Target pipeline (required) |
| `--position=<top\|bottom>` | Position in pipeline. Default: `bottom` |
| `--dry-run` | Show what would be reopened without executing |
| `--output=json` | Output in JSON format |

### Position Behavior

- `--position=top`: Uses `position: START`
- `--position=bottom` (default): Uses `position: END`

Note: Unlike `zh issue move`, numeric positions are not supported for reopen. The API only accepts `START` or `END`.

## Output

### Success (single issue)
```
Reopened mpt#1234 into "Backlog": Fix login button alignment
```

### Success (multiple issues)
```
Reopened 3 issues into "Backlog":

  mpt#1234 Fix login button alignment
  mpt#1235 Update error messages
  api#567  Add rate limiting headers
```

### Partial success
```
Reopened 2 of 3 issues into "Backlog":

  mpt#1234 Fix login button alignment
  mpt#1235 Update error messages

Failed to reopen:

  api#568  Permission denied
```

### Dry-run
```
Would reopen 2 issues into "Backlog" at bottom:

  mpt#1234 Fix login button alignment (closed)
  mpt#1235 Update error messages (closed)
```

### Already open warning
```
1 issue already open:

  mpt#1236 Current work item (in "In Progress")

Reopened 2 issues into "Backlog":

  mpt#1234 Fix login button alignment
  mpt#1235 Update error messages
```

### JSON output
```json
{
  "reopened": [
    {
      "id": "Z2lkOi8vcmFwdG9yL0lzc3VlLzEyMzQ1",
      "number": 1234,
      "repository": "gohiring/mpt",
      "title": "Fix login button alignment"
    }
  ],
  "pipeline": {
    "id": "Z2lkOi8vcmFwdG9yL1BpcGVsaW5lLzE5MTY5NzY",
    "name": "Backlog"
  },
  "position": "END",
  "failed": [],
  "alreadyOpen": [],
  "successCount": 1
}
```

## Error Cases

- Issue not found
- Issue already open (warn but continue with others)
- Pipeline not found
- No permission to reopen issue
- GitHub API errors (reported via `githubErrors` field)

## GitHub API

**Not strictly needed** - The ZenHub `reopenIssues` mutation handles reopening issues on both ZenHub and GitHub.

However, if GitHub CLI integration is enabled, could optionally use `gh issue reopen` for:
- Reopening with a comment: `gh issue reopen <number> --comment "Reopening via zh"`
- More detailed error messages from GitHub

The ZenHub mutation's `githubErrors` field should provide sufficient error information in most cases.

## Limitations

### Position Options Limited

Unlike `zh issue move`, which supports numeric positions and relative positioning (after/before another issue), `reopenIssues` only supports `START` and `END` positions. To place a reopened issue at a specific position, you would need to:
1. Reopen to the pipeline
2. Use `zh issue move` to reposition

### No Reopen Comment

The ZenHub `reopenIssues` mutation does not support adding a comment when reopening. To reopen with a comment, use the GitHub API separately.

### Pipeline Required

Unlike the ZenHub web UI which may have a default pipeline for reopened issues, the API requires explicitly specifying a target pipeline.

### Bulk Operation Atomicity

The `reopenIssues` mutation is not atomic - some issues may reopen successfully while others fail. The response includes both `successCount` and `failedIssues` to report partial success.

## Related Subcommands

- **`zh issue close`** - Close issues
- **`zh issue move`** - Move issues between pipelines (including after reopen for precise positioning)
- **`zh issue show`** - View issue details including current state
- **`zh issue list --pipeline=Closed`** - List closed issues to find candidates for reopening
- **`zh board`** - View all pipelines to choose target pipeline

## Adjacent API Capabilities

None beyond what's already covered. The reopen functionality is straightforward and complete.
