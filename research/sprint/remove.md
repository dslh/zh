# zh sprint remove

Remove issues from a sprint.

## Usage

```
zh sprint remove <issue>...
zh sprint remove <issue>... --sprint=<sprint>
```

## Feasibility

**Fully Feasible** - The ZenHub GraphQL API provides the `removeIssuesFromSprints` mutation which directly supports this functionality. The mutation accepts arrays of issue IDs and sprint IDs, and returns the updated sprint objects.

## API

### Mutation: `removeIssuesFromSprints`

```graphql
mutation RemoveIssuesFromSprints($input: RemoveIssuesFromSprintsInput!) {
  removeIssuesFromSprints(input: $input) {
    sprints {
      id
      name
      generatedName
      state
      totalPoints
      completedPoints
      closedIssuesCount
    }
  }
}
```

**Input:**
```json
{
  "input": {
    "issueIds": ["Z2lkOi8v...issue1", "Z2lkOi8v...issue2"],
    "sprintIds": ["Z2lkOi8v...sprint"]
  }
}
```

Both `issueIds` and `sprintIds` are required arrays of ZenHub IDs. Unlike `addIssuesToSprints` which returns `sprintIssues` (the association records), this mutation returns the `sprints` array directly.

### Query: Determine Sprint from Issue

When `--sprint` is not specified, we need to determine which sprint(s) the issue belongs to. Each issue has a `sprints` connection:

```graphql
query GetIssueWithSprints($repositoryGhId: Int!, $issueNumber: Int!) {
  issueByInfo(repositoryGhId: $repositoryGhId, issueNumber: $issueNumber) {
    id
    number
    title
    state
    repository {
      name
      owner {
        login
      }
    }
    sprints(first: 10) {
      nodes {
        id
        name
        generatedName
        state
      }
    }
  }
}
```

### Query: Get Active Sprint (default target)

When `--sprint` is not specified and the issue belongs to multiple sprints, or for validation:

```graphql
query GetActiveSprint($workspaceId: ID!) {
  workspace(id: $workspaceId) {
    activeSprint {
      id
      name
      generatedName
      state
      startAt
      endAt
    }
    upcomingSprint {
      id
      name
    }
    previousSprint {
      id
      name
    }
  }
}
```

### Query: Resolve Sprint by Name/ID

To support sprint identifiers like "Sprint 42" or substring matching:

```graphql
query FindSprints($workspaceId: ID!, $query: String) {
  workspace(id: $workspaceId) {
    sprints(first: 50, query: $query, filters: { state: { eq: OPEN } }) {
      nodes {
        id
        name
        generatedName
        state
        startAt
        endAt
      }
    }
  }
}
```

## Cached Data Requirements

| Data | Purpose |
|------|---------|
| Workspace ID | Required for sprint queries |
| Repository name → `ghId` mapping | Convert `repo#123` to `repositoryGhId` for issue lookup |
| Sprint ID → name mapping | Enable substring matching for `--sprint` flag |

## Flags and Parameters

| Flag | Description |
|------|-------------|
| `<issue>...` | One or more issue identifiers (required). Supports ZenHub ID, `owner/repo#number`, `repo#number` |
| `--sprint=<sprint>` | Target sprint to remove from. Supports: sprint ID, sprint name, unique substring, `current`/`next`/`previous`. If not specified, see behavior below |
| `--all` | Remove from all sprints the issue belongs to |
| `--dry-run` | Show what would be removed without making changes |

## Implementation Notes

1. **Default sprint behavior**: When no `--sprint` flag is provided:
   - If the issue belongs to exactly one sprint, remove from that sprint
   - If the issue belongs to multiple sprints, error and ask user to specify `--sprint` or use `--all`
   - If the issue is not in any sprint, report this (not an error, but informational)

2. **Relative sprint references**:
   - `current` → `workspace.activeSprint`
   - `next` → `workspace.upcomingSprint`
   - `previous` → `workspace.previousSprint`

3. **Idempotency**: Removing an issue that's not in the sprint should succeed silently or report "not in sprint" (verify actual API behavior).

4. **Bulk operations**: The API accepts multiple issues in a single call, so batch all provided issues into one mutation.

5. **Validation**: Before calling the mutation, verify:
   - All issues exist and are accessible
   - The target sprint exists
   - Issues are actually in the target sprint (optional - API may handle gracefully)

6. **Return type difference**: Note that `removeIssuesFromSprints` returns `sprints` (the Sprint objects), while `addIssuesToSprints` returns `sprintIssues` (the association records). Adjust response handling accordingly.

## Not Available in ZenHub API

None identified. The `removeIssuesFromSprints` mutation provides full functionality for this command.

## GitHub API Requirements

**None** - Sprints are a ZenHub-only concept. GitHub API is only needed for issue identifier resolution (converting `owner/repo#number` to ZenHub issue IDs), which is already handled by the `issueByInfo` query.

## Output Example

### Success (single issue)

```
Removed mpt#1234 from Sprint 47

Sprint 47 now has 14 issues (33 points)
```

### Success (multiple issues)

```
Removed 3 issues from Sprint 47:
  - mpt#1234: Fix auth timeout
  - mpt#1235: Update error messages
  - api#567: Optimize query performance

Sprint 47 now has 12 issues (25 points)
```

### Issue not in sprint

```
Issue mpt#1234 is not in Sprint 47 (no changes made)
```

### Issue in multiple sprints (no --sprint specified)

```
Error: Issue mpt#1234 belongs to multiple sprints:
  - Sprint 47 (active)
  - Sprint 48 (upcoming)

Specify which sprint with --sprint=<name>, or use --all to remove from all sprints.
```

### Dry run

```
Would remove from Sprint 47:
  - mpt#1234: Fix auth timeout (5 pts)
  - mpt#1235: Update error messages (3 pts)

Sprint would have 12 issues (44 points) after removal.

Use without --dry-run to apply changes.
```

## Related Subcommands

The API also supports:

- **`addIssuesToSprints`** mutation - for `zh sprint add`
- **Sprint issues query** via `sprint.sprintIssues` - for `zh sprint show`
- **Issue sprint membership** via `issue.sprints` - useful for showing which sprints an issue belongs to in `zh issue show`
