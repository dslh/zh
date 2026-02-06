# zh workspace show

Show detailed information about a workspace. Defaults to the current (configured default) workspace if no name is provided.

## Feasibility

**Fully Feasible** - The ZenHub GraphQL API provides comprehensive workspace data via the `workspace(id:)` query. All information mentioned in the spec (name, repos, pipelines, sprint config) is available.

## Primary Query: Workspace Details

Fetch complete workspace details by ID:

```graphql
query GetWorkspace($workspaceId: ID!) {
  workspace(id: $workspaceId) {
    id
    name
    displayName
    description
    private
    createdAt
    updatedAt
    viewerPermission
    isFavorite
    isEditable
    isDeletable
    importState

    # Organization
    zenhubOrganization {
      id
      name
    }

    # Creator
    creator {
      id
      name
      email
      githubUser {
        login
        name
      }
    }

    # Default repository
    defaultRepository {
      id
      name
      ownerName
      ghId
    }

    # Sprint configuration
    sprintConfig {
      id
      name
      kind
      period
      startDay
      endDay
      tzIdentifier
    }

    # Current sprint state
    activeSprint {
      id
      name
      generatedName
      state
      startAt
      endAt
      totalPoints
      completedPoints
    }
    upcomingSprint {
      id
      name
      generatedName
      startAt
      endAt
    }
    previousSprint {
      id
      name
      generatedName
      state
      startAt
      endAt
      totalPoints
      completedPoints
    }

    # Velocity metrics
    averageSprintVelocity

    # Estimate settings
    assumeEstimates
    assumedEstimateValue
    hasEstimatedIssues

    # Pipelines
    pipelinesConnection(first: 50) {
      totalCount
      nodes {
        id
        name
        description
        stage
        isDefaultPRPipeline
      }
    }

    # Repositories
    repositoriesConnection(first: 100) {
      totalCount
      nodes {
        id
        name
        ownerName
        ghId
        isPrivate
        isArchived
      }
    }

    # Priorities defined for this workspace
    prioritiesConnection {
      nodes {
        id
        name
        color
      }
    }

    # Related workspaces (share repos)
    relatedWorkspaces {
      totalCount
      nodes {
        id
        name
        displayName
      }
    }
  }
}
```

## Finding Workspace by Name

When the user specifies a workspace by name or substring (not ID), search via the viewer's organizations:

```graphql
query FindWorkspaceByName($query: String!) {
  viewer {
    searchWorkspaces(query: $query, first: 20) {
      nodes {
        id
        name
        displayName
        description
        zenhubOrganization {
          id
          name
        }
      }
    }
  }
}
```

If multiple matches are found, prompt the user to be more specific or show available options.

Note: `searchWorkspaces` requires a non-empty query string. For an exact name match, the CLI should first check the cached workspace list before falling back to the search query.

## Available Fields

### Core Workspace Fields

| Field | Type | Description |
|-------|------|-------------|
| `id` | ID | Workspace identifier |
| `name` | String | Internal workspace name |
| `displayName` | String | Human-readable name (use this for display) |
| `description` | String | Optional workspace description |
| `private` | Boolean | Whether the workspace is private |
| `createdAt` | DateTime | When the workspace was created |
| `updatedAt` | DateTime | Last modification time |
| `viewerPermission` | PermissionLevel | User's permission level |
| `isFavorite` | Boolean | Whether user has favorited this workspace |
| `isEditable` | Boolean | Whether user can edit this workspace |
| `isDeletable` | Boolean | Whether user can delete this workspace |
| `importState` | WorkspaceImportState | Repository import status |

### Permission Levels

| Value | Description |
|-------|-------------|
| `NONE` | No access |
| `READ` | Read-only access |
| `ZENHUB_WRITE` | Can modify ZenHub-specific data |
| `WRITE` | Full write access |
| `ADMIN` | Administrative access |

### Import States

| Value | Description |
|-------|-------------|
| `PENDING` | Import not started |
| `IN_PROGRESS` | Currently importing |
| `USABLE` | Partially imported, usable |
| `COMPLETED` | Fully imported |

### Sprint Configuration Fields

| Field | Type | Description |
|-------|------|-------------|
| `kind` | SprintConfigKind | `weekly` or `monthly` cadence |
| `period` | Int | Number of weeks (for weekly) or months |
| `startDay` | DayOfWeek | Day sprints start (SUNDAY-SATURDAY) |
| `endDay` | DayOfWeek | Day sprints end (SUNDAY-SATURDAY) |
| `tzIdentifier` | String | Timezone (e.g., "America/New_York") |

### Pipeline Fields

| Field | Type | Description |
|-------|------|-------------|
| `id` | ID | Pipeline identifier |
| `name` | String | Pipeline name |
| `description` | String | Optional description |
| `stage` | PipelineStage | Workflow stage classification |
| `isDefaultPRPipeline` | Boolean | Default pipeline for new PRs |

### Pipeline Stages

| Value | Description |
|-------|-------------|
| `BACKLOG` | Work not yet started |
| `SPRINT_BACKLOG` | Queued for current sprint |
| `DEVELOPMENT` | Actively being worked on |
| `REVIEW` | In review or testing |
| `COMPLETED` | Finished work |
| `null` | No stage assigned |

### Repository Fields

| Field | Type | Description |
|-------|------|-------------|
| `id` | ID | ZenHub repository ID |
| `name` | String | Repository name |
| `ownerName` | String | GitHub owner (org or user) |
| `ghId` | Int | GitHub repository ID |
| `isPrivate` | Boolean | Whether repo is private |
| `isArchived` | Boolean | Whether repo is archived |

## Suggested Flags

| Flag | Description |
|------|-------------|
| `--repos` | Show full repository list (default: count only) |
| `--pipelines` | Show full pipeline list (default: count only) |
| `--sprints` | Include sprint information (active, upcoming, previous) |
| `--all` | Show all details (repos, pipelines, sprints) |
| `--show-id` | Include IDs in output |
| `--output=json` | Output as JSON |

## Caching Requirements

| Data | Purpose |
|------|---------|
| Workspace name → ID mappings | For name/substring lookup |
| Organization ID → name mappings | For display |
| Default workspace ID | From local config (not API) |

The workspace list cache should include `id`, `name`, `displayName`, and `zenhubOrganization.name` for each workspace.

## GitHub API Requirements

**None** - All workspace metadata is available from ZenHub's API. GitHub has no workspace concept.

Additional GitHub repository details (like description, language, stars) could be fetched if desired, but are not essential for this command.

## Limitations

### No Workspace Lookup by Name in API

There's no `workspace(name: "...")` query. You must either:
1. Know the workspace ID (from cache or config)
2. Search via `viewer.searchWorkspaces(query:)` which requires a non-empty string
3. List all workspaces via `viewer.zenhubOrganizations.workspaces` and filter client-side

### No Issue/Epic Counts

The Workspace type doesn't provide total issue or epic counts. Getting these would require additional queries with pagination, which is expensive. Consider omitting or making optional via a `--counts` flag.

### Creator May Be Null

The `creator` field may be null if the creating user's account no longer exists or has been deactivated.

### Sprint Config May Be Null

The `sprintConfig` field is null if sprints haven't been configured for the workspace.

### Related Workspaces

The `relatedWorkspaces` connection shows workspaces that share at least one repository. This may be empty for workspaces with unique repositories.

## Related/Adjacent Capabilities

### Workspace Statistics

The workspace provides some aggregate metrics that could be shown:

```graphql
{
  workspace(id: $id) {
    hasEstimatedIssues
    averageSprintVelocity
    averageSprintVelocityWithDiff {
      value
      diff
      direction
    }
  }
}
```

### Saved Views

Show count of saved views for the workspace:

```graphql
{
  workspace(id: $id) {
    savedViews(first: 1) {
      totalCount
    }
    defaultSavedView {
      id
      name
    }
  }
}
```

### Pipeline Automations

The workspace has pipeline-to-pipeline automations:

```graphql
{
  workspace(id: $id) {
    pipelineToPipelineAutomations {
      totalCount
      nodes {
        id
        sourcePipeline { name }
        destinationPipeline { name }
      }
    }
  }
}
```

### Potential Related Command

- `zh workspace stats` - Detailed velocity trends, issue counts, activity metrics
- `zh workspace automations` - List configured pipeline automations

## Output Example

### Default Output (Summary)

```
WORKSPACE: Development
══════════════════════════════════════════════════════════════════════════════

Organization:   gohiring
ID:             5c5c2662a623f9724788f533
Created:        Feb 7, 2019
Last updated:   Nov 4, 2024
Permission:     write
Visibility:     Public

SPRINT CONFIGURATION
────────────────────────────────────────────────────────────────────────────────
Sprints are not configured for this workspace.

SUMMARY
────────────────────────────────────────────────────────────────────────────────
Repositories:   35 (2 archived)
Pipelines:      16
Priorities:     1 defined

Default repo:   gohiring/api

Use --repos or --pipelines for full lists.
```

### With Sprint Config

```
SPRINT CONFIGURATION
────────────────────────────────────────────────────────────────────────────────
Cadence:        2-week sprints (weekly)
Schedule:       Monday → Sunday
Timezone:       Europe/Berlin

Active sprint:  Sprint 47 (Jan 20 - Feb 2, 2025)
                34/52 points completed (65%)
Upcoming:       Sprint 48 (Feb 3 - Feb 16, 2025)
Velocity:       42 pts/sprint (avg last 3)
```

### With --repos Flag

```
REPOSITORIES (35)
────────────────────────────────────────────────────────────────────────────────
REPO                        GITHUB ID    PRIVATE  ARCHIVED
────────────────────────────────────────────────────────────────────────────────
gohiring/api *              4925400      yes      no
gohiring/mpt                38994263     yes      no
gohiring/dashboard          32623610     yes      no
gohiring/landing-pages      144871792    yes      no
...

* = default repository
```

### With --pipelines Flag

```
PIPELINES (16)
────────────────────────────────────────────────────────────────────────────────
NAME                      STAGE          DEFAULT PR  DESCRIPTION
────────────────────────────────────────────────────────────────────────────────
New Issues                -              no          -
Icebox                    -              no          -
Improvements Backlog      -              no          Backlog of issues aimed at improving...
Backlog                   BACKLOG        no          -
Next Up                   SPRINT_BACKLOG no          -
In Research               -              no          -
In Development            DEVELOPMENT    no          -
Ready for Code Review     DEVELOPMENT    yes         Issues become stale after 4 days...
Code Review               DEVELOPMENT    no          -
On Staging                REVIEW         no          -
Ready for Production      REVIEW         no          -
On Production             REVIEW         no          -
Documentation             REVIEW         no          Any internal and external documentation...
Follow Up                 REVIEW         no          -
Done                      COMPLETED      no          -
Refused                   -              no          -
```

## Implementation Notes

1. **Default to current workspace**: If no name argument provided, use the workspace ID from local config. If no default is configured, prompt user to run `zh workspace switch` first.

2. **Name resolution priority**:
   - Check if argument is a valid workspace ID (starts with expected prefix or is hex string)
   - Check cached workspace list for exact name match
   - Check cached list for substring match (must be unique)
   - Fall back to `searchWorkspaces` API query

3. **Indicate current default**: When showing a workspace, note if it's the configured default workspace.

4. **Handle null sprint config**: Display a helpful message suggesting how to configure sprints if `sprintConfig` is null.

5. **Repository ordering**: Consider sorting by name alphabetically. Show archived repos at the end or with a visual indicator.

6. **Pipeline ordering**: Pipelines are returned in board order (left to right). Maintain this order in display.
