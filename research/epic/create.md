# zh epic create

Create a new epic in the workspace.

## Overview

ZenHub supports two types of epics:
1. **ZenhubEpic** (standalone) - Native ZenHub epics created via `createZenhubEpic`
2. **Epic** (legacy) - GitHub issues promoted to epic status via `createEpic` or `createEpicFromIssue`

This command creates **standalone ZenHub epics** by default, which is the modern approach. Legacy epics require a GitHub issue to be created first.

## Feasibility

**Fully Feasible** - The ZenHub GraphQL API provides all necessary mutations.

## Primary Mutation: Create Standalone Epic

```graphql
mutation CreateZenhubEpic($input: CreateZenhubEpicInput!) {
  createZenhubEpic(input: $input) {
    zenhubEpic {
      id
      title
      body
      state
      createdAt
    }
  }
}
```

**Variables:**

```json
{
  "input": {
    "zenhubOrganizationId": "Z2lkOi8vcmFwdG9yL1plbmh1Yk9yZ2FuaXphdGlvbi8xMjkw",
    "zenhubEpic": {
      "title": "Q2 Platform Improvements",
      "body": "This epic covers all platform improvements for Q2 2024."
    }
  }
}
```

### CreateZenhubEpicInput Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `zenhubOrganizationId` | ID! | Yes | The ZenHub organization ID (obtained from workspace) |
| `zenhubEpic` | ZenhubEpicInput! | Yes | The epic title and body |
| `zenhubRepositoryId` | ID | No | Associate the epic with a specific repository |

### ZenhubEpicInput Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `title` | String! | Yes | Epic title |
| `body` | String | No | Epic description (markdown) |

### Response

The mutation returns the created `ZenhubEpic` object with all standard fields available.

## Alternative: Create Legacy Epic

For creating an issue-backed legacy epic (creates a GitHub issue):

```graphql
mutation CreateEpic($input: CreateEpicInput!) {
  createEpic(input: $input) {
    epic {
      id
      issue {
        id
        number
        title
        htmlUrl
        repository {
          name
          ownerName
        }
      }
    }
  }
}
```

**Variables:**

```json
{
  "input": {
    "issue": {
      "repositoryGhId": 38994263,
      "title": "Q2 Platform Improvements",
      "body": "This epic covers all platform improvements for Q2 2024.",
      "labels": ["epic"],
      "assignees": ["username"]
    }
  }
}
```

### IssueInput Fields (for legacy epics)

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `repositoryGhId` | Int | Yes* | GitHub repository ID (*or `repositoryId`) |
| `repositoryId` | ID | Yes* | ZenHub repository ID (*or `repositoryGhId`) |
| `title` | String! | Yes | Issue/epic title |
| `body` | String | No | Issue body (markdown) |
| `labels` | [String!] | No | Label names to apply |
| `assignees` | [String!] | No | GitHub usernames to assign |
| `milestone` | Int | No | GitHub milestone number |

## Setting Additional Properties

The `createZenhubEpic` mutation only sets title and body. Additional properties require follow-up mutations:

### Set Start/End Dates

```graphql
mutation UpdateZenhubEpicDates($input: UpdateZenhubEpicDatesInput!) {
  updateZenhubEpicDates(input: $input) {
    zenhubEpic {
      id
      startOn
      endOn
    }
  }
}
```

**Variables:**

```json
{
  "input": {
    "zenhubEpicId": "Z2lkOi8vcmFwdG9yL1plbmh1YkVwaWMvMTIzNDU",
    "startOn": "2024-04-01",
    "endOn": "2024-06-30"
  }
}
```

### Set State

```graphql
mutation UpdateZenhubEpicState($input: UpdateZenhubEpicStateInput!) {
  updateZenhubEpicState(input: $input) {
    zenhubEpic {
      id
      state
    }
  }
}
```

**Variables:**

```json
{
  "input": {
    "zenhubEpicId": "Z2lkOi8vcmFwdG9yL1plbmh1YkVwaWMvMTIzNDU",
    "state": "TODO"
  }
}
```

### Add Assignees

```graphql
mutation AddAssigneesToZenhubEpics($input: AddAssigneesToZenhubEpicsInput!) {
  addAssigneesToZenhubEpics(input: $input) {
    zenhubEpics {
      id
      assignees(first: 10) {
        nodes {
          id
          name
        }
      }
    }
  }
}
```

**Variables:**

```json
{
  "input": {
    "zenhubEpicIds": ["Z2lkOi8vcmFwdG9yL1plbmh1YkVwaWMvMTIzNDU"],
    "assigneeIds": ["Z2lkOi8vcmFwdG9yL1plbmh1YlVzZXIvMTIzNDU"]
  }
}
```

### Add Labels

```graphql
mutation AddZenhubLabelsToZenhubEpics($input: AddZenhubLabelsToZenhubEpicsInput!) {
  addZenhubLabelsToZenhubEpics(input: $input) {
    zenhubEpics {
      id
      labels(first: 10) {
        nodes {
          id
          name
        }
      }
    }
  }
}
```

### Set Estimate

```graphql
mutation SetEstimateOnZenhubEpics($input: SetMultipleEstimatesOnZenhubEpicsInput!) {
  setMultipleEstimatesOnZenhubEpics(input: $input) {
    zenhubEpics {
      id
      estimate {
        value
      }
    }
  }
}
```

### Add Child Issues

```graphql
mutation AddIssuesToZenhubEpics($input: AddIssuesToZenhubEpicsInput!) {
  addIssuesToZenhubEpics(input: $input) {
    zenhubEpics {
      id
      childIssues(first: 10) {
        totalCount
      }
    }
  }
}
```

## Implementation Flow

1. Execute `createZenhubEpic` mutation with title and optional body
2. If `--start` or `--end` flags provided, execute `updateZenhubEpicDates`
3. If `--state` flag provided, execute `updateZenhubEpicState`
4. If `--assignee` flags provided, execute `addAssigneesToZenhubEpics`
5. If `--label` flags provided, execute `addZenhubLabelsToZenhubEpics`
6. If `--estimate` flag provided, execute `setMultipleEstimatesOnZenhubEpics`
7. If `--issue` flags provided, execute `addIssuesToZenhubEpics`
8. Return the created epic details

For legacy epics with `--legacy` flag:
1. Execute `createEpic` mutation (creates GitHub issue + epic)
2. Dates can be set via `updateEpicDates` mutation if needed

## Caching Requirements

| Data | Purpose |
|------|---------|
| Workspace ID | To retrieve the zenhubOrganizationId |
| ZenhubOrganization ID | Required for `createZenhubEpic` |
| Repository ghId mappings | For `--repo` flag with legacy epics |
| ZenhubUser IDs | For `--assignee` flag (need to map GitHub logins to ZenHub user IDs) |
| ZenhubLabel IDs | For `--label` flag |

### Getting ZenhubOrganization ID

The organization ID is retrieved from the workspace:

```graphql
query GetWorkspaceOrg($workspaceId: ID!) {
  workspace(id: $workspaceId) {
    zenhubOrganization {
      id
      name
    }
  }
}
```

This should be cached alongside workspace data.

## Suggested Flags

| Flag | Description |
|------|-------------|
| `--body=<text>` | Epic description (markdown) |
| `--start=<date>` | Start date (ISO8601 format: YYYY-MM-DD) |
| `--end=<date>` | End date (ISO8601 format: YYYY-MM-DD) |
| `--state=<state>` | Initial state: `open`, `todo`, `in_progress` (default: `open`) |
| `--assignee=<user>` | Assign user(s) - repeatable flag |
| `--label=<label>` | Add label(s) - repeatable flag |
| `--estimate=<value>` | Set estimate value |
| `--issue=<ref>` | Add child issue(s) - repeatable flag |
| `--repo=<repo>` | Repository for legacy epic (implies `--legacy`) |
| `--legacy` | Create a legacy issue-backed epic instead of standalone |
| `--output=json` | Output in JSON format |
| `--dry-run` | Show what would be created without executing |

## Default Output Format

```
Created epic "Q2 Platform Improvements"

ID:     Z2lkOi8vcmFwdG9yL1plbmh1YkVwaWMvMTIzNDU
Type:   ZenHub Epic
State:  open
Dates:  2024-04-01 → 2024-06-30
```

With `--dry-run`:

```
Would create epic "Q2 Platform Improvements"

Type:        ZenHub Epic
Body:        This epic covers all platform improvements for Q2 2024.
Start date:  2024-04-01
End date:    2024-06-30
State:       todo
Assignees:   @johndoe, @janedoe
Labels:      platform, priority:high
```

## GitHub API Requirements

**Not required for standalone ZenHub epics.**

For legacy epics, the ZenHub API creates the GitHub issue directly via `createEpic`, so no separate GitHub API call is needed.

However, GitHub API could be useful for:
- Validating that assignee usernames exist before attempting to assign
- Looking up repository by `owner/repo` format if only that is provided

## Limitations

1. **Multiple API calls required** - Unlike the web UI, setting dates, assignees, labels, and state requires separate mutations after initial creation. This makes the operation non-atomic.

2. **No rollback on partial failure** - If the epic is created but a follow-up mutation fails (e.g., invalid assignee), the epic will exist with incomplete data.

3. **ZenhubUser ID resolution** - Assignees must be specified by ZenHub user ID, not GitHub username. The CLI must resolve GitHub logins to ZenHub user IDs using workspace user data.

4. **No project assignment at creation** - Epics cannot be added to a Project during creation. This requires a separate `addZenhubEpicsToProject` mutation.

5. **State default is OPEN** - New epics default to OPEN state; cannot set state in the create mutation.

## Related Subcommands

- **`zh epic show <epic>`** - View the created epic's details
- **`zh epic edit <epic>`** - Modify epic title/body after creation
- **`zh epic set-dates <epic>`** - Update start/end dates
- **`zh epic set-state <epic>`** - Change epic state
- **`zh epic add <epic> <issue>...`** - Add child issues
- **`zh epic alias <epic> <alias>`** - Set a shorthand for the new epic

## Adjacent API Capabilities

### Create Epic on Roadmap

The API provides `createZenhubEpicOnRoadmap` which creates an epic and adds it to the roadmap in one operation. This could support a `--roadmap` flag.

### Create Epic on Project

The API provides `createZenhubEpicOnProject` which creates an epic within a specific project. This could support a `--project=<name>` flag.

### Key Dates

Epics support key dates (milestones within the epic timeline) via `createZenhubEpicKeyDate`. This could support a future `zh epic key-date add` command.

### Convert Issue to Epic

The `createEpicFromIssue` mutation allows promoting an existing GitHub issue to a legacy epic. This could support a `zh epic promote <issue>` command.
