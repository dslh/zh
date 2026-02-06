# zh issue estimate

Set or clear the estimate on an issue.

## Usage

```
zh issue estimate <issue> [value]
```

- `<issue>` - Issue identifier (ZenHub ID, owner/repo#number, repo#number)
- `[value]` - Estimate value to set. Omit to clear the estimate.

## ZenHub API

### Setting an Estimate

**Mutation:** `setEstimate`

```graphql
mutation SetEstimate($input: SetEstimateInput!) {
  setEstimate(input: $input) {
    issue {
      id
      number
      title
      estimate {
        id
        value
      }
      repository {
        name
        ownerName
      }
    }
  }
}
```

**Variables:**
```json
{
  "input": {
    "issueId": "Z2lkOi8vcmFwdG9yL0lzc3VlLzEyMzQ1",
    "value": 5
  }
}
```

### Clearing an Estimate

To clear an estimate, pass `null` for the `value` field:

```json
{
  "input": {
    "issueId": "Z2lkOi8vcmFwdG9yL0lzc3VlLzEyMzQ1",
    "value": null
  }
}
```

### Setting Estimates on Multiple Issues

There's also a bulk mutation available:

**Mutation:** `setMultipleEstimates`

```graphql
mutation SetMultipleEstimates($input: SetMultipleEstimatesInput!) {
  setMultipleEstimates(input: $input) {
    issues {
      id
      number
      estimate {
        value
      }
    }
  }
}
```

**Variables:**
```json
{
  "input": {
    "issueIds": ["Z2lkOi8vcmFwdG9yL0lzc3VlLzEyMzQ1", "Z2lkOi8vcmFwdG9yL0lzc3VlLzY3ODkw"],
    "value": 3
  }
}
```

### Resolving Issue Identifier to ZenHub ID

When the user provides a GitHub-style identifier (e.g., `mpt#1234`), resolve it to a ZenHub ID:

```graphql
query GetIssueByInfo($repoGhId: Int!, $issueNumber: Int!) {
  issueByInfo(repositoryGhId: $repoGhId, issueNumber: $issueNumber) {
    id
    number
    title
    estimate {
      value
    }
  }
}
```

## Cached Information

The following should be cached to support issue identifier resolution:

- **Repository mappings** - Map repo name (and owner/repo) to GitHub ID (`ghId`). Required for `issueByInfo` query.
- **Estimate sets per repository** - The valid estimate values for each repository. Useful for validation and tab completion.

### Fetching Estimate Set

Estimate values are configured per-repository:

```graphql
query GetEstimateSet($repoGhId: Int!, $issueNumber: Int!) {
  issueByInfo(repositoryGhId: $repoGhId, issueNumber: $issueNumber) {
    repository {
      estimateSet {
        values
      }
    }
  }
}
```

Typical values: `[1, 2, 3, 5, 8, 13, 21, 40]` (Fibonacci-like sequence)

## Suggested Flags

| Flag | Description |
|------|-------------|
| `--dry-run` | Show what would be changed without making the change |
| `--json` | Output result as JSON |

## Validation

- If `value` is provided, validate it against the repository's `estimateSet.values`. If the value is not in the allowed set, show an error with the valid options.
- The API accepts `Float` values, but in practice ZenHub uses the predefined estimate set values.

## Output

### Success (setting estimate)
```
Set estimate on mpt#1234 to 5
```

### Success (clearing estimate)
```
Cleared estimate from mpt#1234
```

### Dry-run
```
Would set estimate on mpt#1234 to 5 (currently: 3)
```

### JSON output
```json
{
  "issue": {
    "id": "Z2lkOi8vcmFwdG9yL0lzc3VlLzEyMzQ1",
    "number": 1234,
    "repository": "gohiring/mpt",
    "title": "Issue title here",
    "estimate": {
      "previous": 3,
      "current": 5
    }
  }
}
```

## Error Cases

- Issue not found
- Invalid estimate value (not in repository's estimate set)
- No permission to modify the issue

## Not Available in ZenHub API

Nothing significant missing - the estimate functionality is fully supported.

## GitHub API

Not needed for this subcommand. Estimates are a ZenHub-only concept.

## Related Functionality

The API also supports:

- **Estimate voting** (`inviteToEstimate`, `setEstimationVote`, `removeEstimationVote`) - For team estimation sessions. Could be a future `zh estimate vote` or `zh planning-poker` command.
- **Estimate set management** (`addEstimateSetValue`, `removeEstimateSetValue`) - Admin functionality to customize the available estimate values per repository.
