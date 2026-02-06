# zh view - Saved Views API Investigation

## Summary

**Feasibility: Limited / Partially Blocked**

ZenHub's GraphQL API has significant limitations around SavedView objects. While the API supports creating, updating, and deleting saved views via mutations, **the SavedView type only exposes the `id` field** - the name, filters, sharing status, and other properties are not readable through the API.

This severely limits the usefulness of the `zh view` subcommands.

## API Findings

### SavedView Type

The `SavedView` type is described as "A set of filters saved by a user for later use" but only exposes:

```graphql
type SavedView {
  id: ID!
  # No other fields are exposed
}
```

### Available Queries

**List saved views for a workspace:**
```graphql
query ListSavedViews($workspaceId: ID!) {
  workspace(id: $workspaceId) {
    savedViews(first: 100, query: $searchQuery) {
      totalCount
      nodes {
        id
        # Only ID is available - no name, filters, etc.
      }
    }
    defaultSavedView {
      id
    }
  }
}
```

The `savedViews` connection supports:
- Pagination (`first`, `last`, `after`, `before`)
- Search via `query` parameter (but we can't see what it searches against since names aren't exposed)

### Available Mutations

**Create a saved view:**
```graphql
mutation CreateSavedView($input: CreateSavedViewInput!) {
  createSavedView(input: $input) {
    savedView {
      id
    }
  }
}

# Input type:
input CreateSavedViewInput {
  workspaceId: ID!          # Required
  name: String!             # Required
  filters: IssueSearchFiltersInput!  # Required
  isShared: Boolean         # Optional - whether view is shared with team
}
```

**Update a saved view:**
```graphql
mutation UpdateSavedView($input: UpdateSavedViewInput!) {
  updateSavedView(input: $input) {
    savedView {
      id
    }
  }
}

# Input type:
input UpdateSavedViewInput {
  savedViewId: ID!          # Required
  name: String              # Optional
  filters: IssueSearchFiltersInput  # Optional - replaces all filters
  isShared: Boolean         # Optional
}
```

**Delete a saved view:**
```graphql
mutation DeleteSavedView($input: DeleteSavedViewInput!) {
  deleteSavedView(input: $input) {
    savedView {
      id
    }
  }
}

# Input type:
input DeleteSavedViewInput {
  savedViewId: ID!          # Required
}
```

### Filter Structure (IssueSearchFiltersInput)

When creating or updating saved views, filters are specified using:

```graphql
input IssueSearchFiltersInput {
  repositoryIds: [ID!]                    # Filter by repos
  matchType: MatchingFilter               # "any" or "all"
  displayType: DisplayFilter              # "all", "issues", or "prs"
  labels: StringInput                     # { in: [...], nin: [...], notInAny: bool }
  assignees: IssueUserLoginInput          # { in: [...], nin: [...], notInAny: bool }
  assigneeIds: IssueUserIdInput           # Alternative to assignees
  users: IssueUserLoginInput              # Filter by issue creator
  userIds: IssueUserIdInput               # Alternative to users
  sprints: SprintIdInput                  # { in: [...], nin: [...], specialFilters: "current_sprint" }
  releases: IdInput
  milestones: StringInput
  estimates: EstimateSearchFiltersInput   # { values: FloatInput, specialFilters: "not_estimated" }
  zenhubEpics: ZenhubEpicSearchFiltersInput
  parentIssues: ParentIssuesInput
  issueIssueTypes: StringInput
}
```

**Special filter values:**
- `SprintSpecialFilter`: `current_sprint`
- `EstimateSpecialFilter`: `assigned_for_voting`, `assigned_to_user_for_voting`, `not_estimated`
- `ZenhubEpicSpecialFilter`: `not_in_epic`

## Subcommand Feasibility

### `zh view list` - **NOT FEASIBLE**

Cannot list saved views with their names because the API only returns IDs.

**Workaround:** None available via ZenHub API. Would require storing view metadata locally when views are created via `zh`.

### `zh view show <name>` - **NOT FEASIBLE**

Cannot look up a view by name or display its filters because neither is exposed by the API.

**Workaround:** Would require local storage of view metadata created via `zh`.

### `zh view create <name>` - **PARTIALLY FEASIBLE**

The mutation works, but:
- Cannot verify the view was created with correct name (only ID returned)
- Must store the ID-to-name mapping locally to enable future operations

```graphql
mutation CreateSavedView {
  createSavedView(input: {
    workspaceId: "5c5c2662a623f9724788f533"
    name: "My Issues"
    filters: {
      assignees: { in: ["myusername"] }
      displayType: issues
    }
    isShared: false
  }) {
    savedView {
      id
    }
  }
}
```

**Supported flags based on API:**
- `--assignee=<user>` - Filter by assignee login
- `--label=<label>` - Filter by label name
- `--repo=<repo>` - Filter by repository (requires repo ID lookup)
- `--sprint=<id>` or `--sprint=current` - Filter by sprint
- `--epic=<epic>` - Filter by ZenHub epic
- `--milestone=<name>` - Filter by milestone
- `--estimate=<value>` or `--no-estimate` - Filter by estimate
- `--author=<user>` - Filter by issue creator
- `--type=issues|prs|all` - Show issues, PRs, or both
- `--match=any|all` - Match any or all filters
- `--shared` - Share view with workspace members

### `zh view delete <name>` - **PARTIALLY FEASIBLE**

Requires knowing the view ID. Without local storage, would need the user to provide the ID directly.

```graphql
mutation DeleteSavedView {
  deleteSavedView(input: {
    savedViewId: "Z2lkOi8vcmFwdG9yL1NhdmVkVmlldy8xNDEwNw"
  }) {
    savedView {
      id
    }
  }
}
```

## Information Useful to Cache

For views created via `zh`, local storage would need:
- View ID (from API)
- View name (user-provided, not retrievable from API)
- Workspace ID
- Possibly the filter configuration (for `zh view show`)

Cache location: `~/.cache/zh/views-{workspace_id}.json`

## GitHub API Alternatives

GitHub's API does not have saved views or filter presets. This is a ZenHub-specific feature.

## Critical Limitations

1. **No read access to view metadata** - The most significant limitation. View names, filters, and sharing status cannot be retrieved after creation.

2. **No way to match existing views** - Cannot determine if a view with a given name already exists before creating.

3. **Views created in ZenHub UI are inaccessible** - Only views created via `zh` (with local metadata storage) would be usable.

## Recommendations

Given the API limitations, consider one of these approaches:

### Option A: Implement with Local Storage (Degraded Experience)
- `zh view create` works but stores metadata locally
- `zh view list` only shows views created via `zh`
- `zh view show` only works for views created via `zh`
- `zh view delete` only works for views created via `zh`
- Clear documentation that existing ZenHub views are not visible

### Option B: Skip View Management, Support View Application Only
- Skip `zh view list`, `zh view show`, `zh view create`, `zh view delete`
- Keep `zh board --view=<id>` where users provide the view ID directly
- Users would need to get view IDs from ZenHub's UI or browser dev tools

### Option C: Defer Until API Improves
- Track this as a known limitation
- Revisit if ZenHub expands their GraphQL API

## Related Observations

The Workspace type has a `defaultSavedView` field which returns a SavedView (with only ID). This suggests ZenHub uses saved views internally but hasn't fully exposed them via their public API.

The `labelFilters` field on Workspace (`WorkspaceLabelFilterConnection`) is a separate concept - these are default label filters applied to the workspace, not user-saved views.
