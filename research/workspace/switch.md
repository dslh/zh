# zh workspace switch

Switch the default workspace used by the CLI.

## Feasibility

**Fully Feasible** - This command is primarily a local configuration operation. The CLI stores the default workspace ID in its config file (`~/.config/zh/config.yml`). No API mutation is required for the switch itself, though the API is used to validate that the workspace exists and the user has access.

## Implementation Approach

The `workspace switch` command:
1. Resolves the workspace name/substring to a workspace ID (using cache or API)
2. Validates the workspace exists and the user has access
3. Updates the local config file with the new workspace ID
4. Optionally refreshes the cache for the new workspace (pipelines, repos)

## API Queries

### Validate Workspace by ID

If the user provides a workspace ID directly:

```graphql
query ValidateWorkspace($workspaceId: ID!) {
  workspace(id: $workspaceId) {
    id
    name
    displayName
    viewerPermission
    zenhubOrganization {
      id
      name
    }
  }
}
```

### Search Workspace by Name

If the user provides a name or substring:

```graphql
query SearchWorkspace($query: String!) {
  viewer {
    searchWorkspaces(query: $query, first: 20) {
      nodes {
        id
        name
        displayName
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

Note: `searchWorkspaces` requires a non-empty query string. For exact name matching, prefer checking the local cache first.

### List All Workspaces (for interactive selection)

When using `--interactive` or when the cache is empty:

```graphql
query ListAllWorkspaces {
  viewer {
    zenhubOrganizations(first: 50) {
      nodes {
        id
        name
        workspaces(first: 100) {
          nodes {
            id
            name
            displayName
            viewerPermission
          }
        }
      }
    }
  }
}
```

## No Mutation Required

Switching the default workspace does not require any ZenHub API mutation. The "default workspace" is a CLI-local concept stored in the config file:

```yaml
# ~/.config/zh/config.yml
workspace: 5c5c2662a623f9724788f533
```

## Optional: Favorite Workspace Sync

The API provides a `setFavoriteWorkspace` mutation that marks a workspace as a "favorite" in ZenHub's web UI. The CLI could optionally call this when switching:

```graphql
mutation SetFavoriteWorkspace($workspaceId: ID!) {
  setFavoriteWorkspace(input: { workspaceId: $workspaceId }) {
    workspace {
      id
      isFavorite
    }
  }
}
```

This is **not required** for the switch to work but could be offered as a `--favorite` flag to sync the CLI default with ZenHub's favorites.

## Caching Requirements

| Data | Purpose |
|------|---------|
| Workspace list (ID, name, displayName, org) | Name/substring resolution without API calls |
| Pipeline list per workspace | Refresh after switch for pipeline name resolution |
| Repository list per workspace | Refresh after switch for repo name resolution |

When switching workspaces, consider:
- **Lazy refresh**: Only fetch new workspace's cache on first command that needs it
- **Eager refresh**: Immediately populate cache for the new workspace (better UX, small delay)

A `--no-cache-refresh` flag could skip eager refresh for faster switching.

## Suggested Flags

| Flag | Description |
|------|-------------|
| `--interactive` / `-i` | Show list of workspaces and let user select |
| `--favorite` | Also mark the workspace as a favorite in ZenHub |
| `--no-cache-refresh` | Skip refreshing the cache for the new workspace |
| `--show-id` | Display workspace IDs in interactive mode |

## Workspace Resolution

The `<name>` argument should support:
1. **Exact ID**: `zh workspace switch 5c5c2662a623f9724788f533`
2. **Exact name**: `zh workspace switch "Development"`
3. **Unique substring**: `zh workspace switch dev` (if "Development" is the only match)
4. **Organization-qualified**: `zh workspace switch gohiring/Development` (for disambiguation)

If multiple workspaces match a substring, the command should list matches and ask the user to be more specific.

## GitHub API Requirements

**None** - Workspace management is entirely within ZenHub's domain.

## Limitations

### No "Default Workspace" Concept in API

ZenHub's API has no concept of a user's "default" workspace. This is purely a CLI-local setting. The web UI uses "recently viewed" and "favorites" instead.

### Search Requires Non-Empty Query

The `searchWorkspaces` query cannot accept an empty string. When switching by name, the CLI should:
1. First check the local cache for matches
2. Fall back to API search only if not found in cache

### No Workspace Lookup by Exact Name

There's no `workspace(name: "...")` query. Workspace lookup requires either knowing the ID or searching with `searchWorkspaces`.

## Error Handling

| Scenario | Exit Code | Message |
|----------|-----------|---------|
| Workspace not found | 4 | `Workspace "foo" not found. Run 'zh workspace list' to see available workspaces.` |
| Multiple matches | 2 | `Multiple workspaces match "dev": Development, DevOps. Please be more specific.` |
| No access | 3 | `You don't have access to workspace "..." (permission: NONE)` |
| Already current | 0 | `Already using workspace "Development"` (with success, not error) |

## Output Examples

### Successful Switch

```
Switched to workspace "Development" (gohiring)
```

### Interactive Mode

```
? Select a workspace:
  > Development (gohiring) *
    Data Warehouse (gohiring)
    Discovery and shaping board (draft) (gohiring)

(* = current)
```

### Multiple Matches

```
Multiple workspaces match "dev":
  - Development (gohiring) [5c5c2662a623f9724788f533]
  - DevOps (gohiring) [5c617887e11cd566ed0ffe5b]

Use a more specific name or the workspace ID.
```

## Implementation Notes

1. **Validate before writing config**: Always verify the workspace exists and is accessible before updating the config file.

2. **Show confirmation**: After switching, display the workspace name and organization to confirm the switch was correct.

3. **Handle case sensitivity**: Workspace name matching should be case-insensitive for better UX.

4. **Preserve other config**: When updating the config file, preserve all other settings (API key, GitHub config, aliases, etc.).

5. **Update cache association**: If maintaining workspace-specific caches (pipelines, repos), ensure subsequent commands use the new workspace's cache.

## Related Commands

- `zh workspace list` - Lists available workspaces (useful before switching)
- `zh workspace show` - Shows details of current or specified workspace
- `zh workspace repos` - Lists repos in the current workspace (uses the switched workspace)
