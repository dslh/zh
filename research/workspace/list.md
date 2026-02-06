# zh workspace list

List all workspaces accessible to the current user.

## API Query

### Primary Query: List All Workspaces

Fetch all workspaces across all organizations the user belongs to:

```graphql
query ListWorkspaces {
  viewer {
    zenhubOrganizations(first: 50) {
      totalCount
      pageInfo {
        hasNextPage
        endCursor
      }
      nodes {
        id
        name
        workspaces(first: 100) {
          totalCount
          pageInfo {
            hasNextPage
            endCursor
          }
          nodes {
            id
            name
            displayName
            description
            private
            isFavorite
            viewerPermission
            importState
            createdAt
            updatedAt
            repositoriesConnection(first: 1) {
              totalCount
            }
            pipelinesConnection(first: 1) {
              totalCount
            }
          }
        }
      }
    }
  }
}
```

### Alternative: Recently Viewed Workspaces

For a quick list of recently accessed workspaces (useful for `--recent` flag):

```graphql
query RecentWorkspaces($first: Int) {
  recentlyViewedWorkspaces(first: $first) {
    nodes {
      id
      name
      displayName
      description
      isFavorite
      viewerPermission
      private
      createdAt
      updatedAt
      zenhubOrganization {
        id
        name
      }
    }
  }
}
```

### Alternative: Favorite Workspaces

For listing only favorited workspaces:

```graphql
query FavoriteWorkspaces {
  viewer {
    workspaceFavorites(first: 50) {
      nodes {
        id
        workspace {
          id
          name
          displayName
          description
          viewerPermission
          zenhubOrganization {
            id
            name
          }
        }
      }
    }
  }
}
```

### Search Workspaces

For filtering by name (requires non-empty query string):

```graphql
query SearchWorkspaces($query: String!) {
  viewer {
    searchWorkspaces(query: $query, first: 50) {
      nodes {
        id
        name
        displayName
        description
        isFavorite
        viewerPermission
        zenhubOrganization {
          id
          name
        }
      }
    }
  }
}
```

Note: `searchWorkspaces` requires a non-empty query string. It cannot be used to list all workspaces.

## Available Fields

| Field | Type | Description |
|-------|------|-------------|
| `id` | ID | Workspace identifier (used for API calls) |
| `name` | String | Internal workspace name |
| `displayName` | String | Human-readable workspace name (use this for display) |
| `description` | String | Optional workspace description |
| `private` | Boolean | Whether the workspace is private |
| `isFavorite` | Boolean | Whether the current user has favorited this workspace |
| `viewerPermission` | PermissionLevel | User's permission: `NONE`, `READ`, `ZENHUB_WRITE`, `WRITE`, `ADMIN` |
| `importState` | WorkspaceImportState | Import status: `PENDING`, `IN_PROGRESS`, `USABLE`, `COMPLETED` |
| `createdAt` | DateTime | Workspace creation timestamp |
| `updatedAt` | DateTime | Last update timestamp |
| `repositoriesConnection.totalCount` | Int | Number of connected repositories |
| `pipelinesConnection.totalCount` | Int | Number of pipelines |

## Organization Context

Workspaces are nested under ZenHub Organizations. Each organization has:

| Field | Type | Description |
|-------|------|-------------|
| `id` | ID | Organization identifier |
| `name` | String | Organization name (typically matches GitHub org) |

## Ordering

The `workspaces` connection on `ZenhubOrganization` supports ordering via `WorkspaceOrderInput`:

| Field | Description |
|-------|-------------|
| `WORKSPACE_VIEWS` | Order by view count (popularity) |

Direction: `ASC` or `DESC`

Note: No alphabetical sort option is available in the API; client-side sorting is needed for name-based ordering.

## Caching Requirements

The following should be cached for efficient operation:

- **Organization list** - ID and name for each organization
- **Workspace list** - ID, name, displayName for each workspace (enables workspace name resolution for other commands)
- **Default workspace** - The user's configured default workspace ID (stored in config, not from API)

## Suggested Flags

Based on API capabilities:

| Flag | Description |
|------|-------------|
| `--org=<name>` | Filter workspaces by organization name |
| `--favorites` | Show only favorited workspaces |
| `--recent` | Show recently viewed workspaces |
| `--search=<query>` | Search workspaces by name |
| `--show-id` | Include workspace IDs in output |
| `--output=json` | Output in JSON format |

## Display Considerations

The output should indicate:
- Which workspace is the currently configured default (from local config)
- Whether a workspace is favorited
- The organization each workspace belongs to (useful if user has access to multiple orgs)
- Permission level (especially useful to distinguish read-only access)

Example output format:
```
ORGANIZATION  WORKSPACE                              REPOS  PIPELINES  PERMISSION
gohiring      Development *                          35     16         write
gohiring      Data Warehouse                         2      14         write
gohiring      Discovery and shaping board (draft)    0      4          write
```

(`*` indicates current default workspace)

## Limitations

### No Global Workspace List

There is no single API query that returns all workspaces. You must:
1. First fetch all organizations via `viewer.zenhubOrganizations`
2. Then fetch workspaces for each organization

This is generally not a problem since most users belong to 1-2 organizations.

### Search Requires Non-Empty Query

The `searchWorkspaces` query cannot be used to list all workspaces (empty string fails validation). It's only useful for filtering by name when the user provides a search term.

### No "Default Workspace" Concept in API

ZenHub's API doesn't have a concept of a "default" workspace. The CLI must maintain this setting locally in its config file.

## Related Subcommands

- `zh workspace show` - Uses the same workspace fields, plus additional details
- `zh workspace switch` - Sets the default workspace in local config (no API call needed for the switch itself, but may want to validate the workspace exists)
- `zh workspace repos` - Uses `workspace.repositoriesConnection` for the selected workspace
