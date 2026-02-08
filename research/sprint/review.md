# zh sprint review

Show details of the sprint review associated with a sprint. Defaults to the active sprint if no sprint identifier is provided.

## Feasibility

**Fully Feasible** — The ZenHub GraphQL API exposes a `sprintReview` field on the `Sprint` type that returns the full review content, metadata, associated features (work themes grouped with issues), review schedules, and issues closed after the review was generated. The `generateSprintReview` mutation is available if we ever want to support triggering review generation from the CLI.

Sprint reviews are an AI-generated feature in ZenHub. They summarize what was accomplished in a sprint, group work into thematic "features", and can be manually edited after generation. Not all sprints will have a review — the `sprintReview` field is nullable and will be `null` if no review has been generated.

## Primary Query

Fetch a sprint review by sprint ID:

```graphql
query SprintReview($sprintId: ID!) {
  node(id: $sprintId) {
    ... on Sprint {
      id
      name
      generatedName
      state
      startAt
      endAt
      totalPoints
      completedPoints
      closedIssuesCount
      sprintReview {
        id
        title
        body
        htmlBody
        state
        language
        lastGeneratedAt
        manuallyEdited
        createdAt
        updatedAt
        initiatedBy {
          id
          name
          githubUser {
            login
          }
        }
        sprintReviewFeatures(first: 50) {
          totalCount
          nodes {
            id
            title
            createdAt
            updatedAt
            aiGeneratedIssues(first: 50) {
              totalCount
              nodes {
                id
                number
                title
                state
                estimate {
                  value
                }
                repository {
                  name
                  ownerName
                }
              }
            }
            manuallyAddedIssues(first: 50) {
              totalCount
              nodes {
                id
                number
                title
                state
                estimate {
                  value
                }
                repository {
                  name
                  ownerName
                }
              }
            }
          }
        }
        sprintReviewSchedules(first: 20) {
          totalCount
          nodes {
            id
            title
            startAt
            completedAt
            createdAt
          }
        }
        issuesClosedAfterSprintReview(first: 50) {
          totalCount
          nodes {
            id
            number
            title
            state
            estimate {
              value
            }
            repository {
              name
              ownerName
            }
          }
        }
      }
    }
  }
}
```

### Default to active sprint

When no sprint identifier is provided, resolve via the workspace accessor:

```graphql
query ActiveSprintReview($workspaceId: ID!) {
  workspace(id: $workspaceId) {
    activeSprint {
      id
      name
      generatedName
      state
      startAt
      endAt
      totalPoints
      completedPoints
      closedIssuesCount
      sprintReview {
        id
        title
        body
        htmlBody
        state
        language
        lastGeneratedAt
        manuallyEdited
        createdAt
        updatedAt
        initiatedBy {
          id
          name
          githubUser {
            login
          }
        }
        sprintReviewFeatures(first: 50) {
          totalCount
          nodes {
            id
            title
            aiGeneratedIssues(first: 50) {
              totalCount
              nodes {
                id
                number
                title
                state
                estimate { value }
                repository { name ownerName }
              }
            }
            manuallyAddedIssues(first: 50) {
              totalCount
              nodes {
                id
                number
                title
                state
                estimate { value }
                repository { name ownerName }
              }
            }
          }
        }
        sprintReviewSchedules(first: 20) {
          totalCount
          nodes {
            id
            title
            startAt
            completedAt
          }
        }
        issuesClosedAfterSprintReview(first: 50) {
          totalCount
          nodes {
            id
            number
            title
            state
            estimate { value }
            repository { name ownerName }
          }
        }
      }
    }
  }
}
```

The same pattern works for `previousSprint` and `upcomingSprint`.

## Relevant Mutation: generateSprintReview

The API exposes a `generateSprintReview` mutation that triggers AI generation of a sprint review:

```graphql
mutation GenerateSprintReview($input: GenerateSprintReviewInput!) {
  generateSprintReview(input: $input) {
    sprintReview {
      id
      title
      body
      state
    }
  }
}
```

Input:
```json
{
  "input": {
    "sprintId": "<sprint_id>",
    "callAsync": true
  }
}
```

| Field | Type | Description |
|---|---|---|
| `sprintId` | `ID!` | The sprint to generate a review for |
| `callAsync` | `Boolean` | Whether to run generation asynchronously (recommended — generation involves AI and can be slow) |

The spec calls for `zh sprint review` as read-only, but a `--generate` flag or a separate `zh sprint review generate` subcommand could invoke this mutation if desired.

## SprintReview Fields

| Field | Type | Description |
|---|---|---|
| `id` | `ID!` | Review ID |
| `title` | `String` | Review title (nullable) |
| `body` | `String` | Review body as plain text / markdown (nullable) |
| `htmlBody` | `String` | Review body rendered as HTML (nullable) |
| `state` | `SprintReviewState!` | `INITIAL`, `IN_PROGRESS`, or `COMPLETED` |
| `language` | `String` | Language the review was generated in (nullable) |
| `lastGeneratedAt` | `ISO8601DateTime` | When the review was last (re)generated (nullable — null if never generated) |
| `manuallyEdited` | `Boolean!` | Whether the review has been manually edited after generation |
| `initiatedBy` | `ZenhubUser` | The user who triggered review generation (nullable) |
| `createdAt` | `ISO8601DateTime!` | When the review record was created |
| `updatedAt` | `ISO8601DateTime!` | Last update timestamp |

### SprintReviewState enum

| Value | Description |
|---|---|
| `INITIAL` | Review record exists but content has not been generated |
| `IN_PROGRESS` | Review generation is currently running |
| `COMPLETED` | Review has been generated and is ready to view |

### SprintReviewFeature

Features are thematic groupings of issues — the AI groups related completed work into named categories. Each feature has two issue connections: AI-grouped issues and manually added issues.

| Field | Type | Description |
|---|---|---|
| `id` | `ID!` | Feature ID |
| `title` | `String!` | Feature/theme name (e.g. "Authentication improvements") |
| `aiGeneratedIssues` | `IssueConnection!` | Issues the AI grouped into this feature |
| `manuallyAddedIssues` | `IssueConnection!` | Issues manually added to this feature by users |
| `createdAt` | `ISO8601DateTime!` | When the feature was created |
| `updatedAt` | `ISO8601DateTime!` | Last update timestamp |

### SprintReviewSchedule

Scheduled review meetings or checkpoints associated with the review.

| Field | Type | Description |
|---|---|---|
| `id` | `ID!` | Schedule ID |
| `title` | `String!` | Schedule title/name |
| `startAt` | `ISO8601DateTime!` | Scheduled start time |
| `completedAt` | `ISO8601DateTime` | When it was marked complete (nullable) |
| `createdAt` | `ISO8601DateTime!` | When the schedule was created |

### issuesClosedAfterSprintReview

An `IssueConnection` containing issues that were closed after the sprint review was generated. Useful for showing work that happened after the review snapshot.

## Suggested CLI Flags

| Flag | Description |
|---|---|
| `--features` | Show the feature breakdown with grouped issues (default: show summary body only) |
| `--schedules` | Show associated review schedules |
| `--late-closes` | Show issues closed after the review was generated |
| `--raw` | Output the review body as raw text without terminal markdown rendering |
| `--html` | Output the `htmlBody` instead of the plain text `body` |

### Sprint identifier

Supports all standard sprint identifiers:
- No argument or `current` — active sprint
- `next` — upcoming sprint
- `previous` or `last` — previous sprint
- Sprint name or substring
- Sprint ZenHub ID

## Caching Requirements

| Data | Cache file | Purpose |
|---|---|---|
| Workspace ID | config | Required for active/previous/upcoming sprint resolution |
| Sprint name/ID mappings | `sprints-{workspace_id}.json` | For resolving sprint by name or substring |

Sprint review data itself should never be cached — it can change if a review is regenerated or manually edited.

## GitHub API Requirements

**None.** All sprint review data is ZenHub-native. Issue references within the review include `repository.name` and `repository.ownerName` from ZenHub, which is sufficient for displaying `owner/repo#number` format.

## Limitations

### Review may not exist

The `sprintReview` field is nullable. Most sprints won't have a review unless the team has explicitly generated one. The CLI should handle this gracefully with a message like "No review has been generated for this sprint."

### AI-generated content

The review body is generated by AI and may be verbose or structured differently between sprints. The CLI should render it as markdown (using Glamour), but the content quality is outside `zh`'s control.

### No API to edit review content

There is no mutation to update the review body or title. Reviews can only be generated (or regenerated) via `generateSprintReview`. Manual editing is only available in ZenHub's web UI. This means `zh sprint review` is read-only.

### Async generation

The `generateSprintReview` mutation supports a `callAsync` flag. When `true`, the mutation returns immediately with a review in `IN_PROGRESS` state. The CLI would need to poll for completion if we want to show the result, or simply inform the user that generation has started.

### No review history

There's no way to access previous versions of a review. Regenerating a review overwrites the previous content. The `lastGeneratedAt` field shows when it was last regenerated, and `manuallyEdited` indicates if it was modified after that, but old versions are not accessible.

## Adjacent API Capabilities

### Sprint summary data alongside review

When displaying a sprint review, including sprint-level summary data provides useful context:

```graphql
{
  sprint {
    totalPoints
    completedPoints
    closedIssuesCount
    sprintIssues(first: 0) { totalCount }
  }
}
```

This allows the CLI to show a progress header above the review content.

### Potential related subcommand: `zh sprint review generate`

The `generateSprintReview` mutation could power a `zh sprint review generate [sprint]` subcommand (or `zh sprint review --generate`). This would:
1. Call `generateSprintReview` with `callAsync: true`
2. Print "Generating sprint review..."
3. Optionally poll until `state` transitions from `IN_PROGRESS` to `COMPLETED`
4. Display the generated review

This is not in the current spec but the API fully supports it.

## Output Example

```
SPRINT REVIEW — Sprint: Jan 22 - Feb 5, 2026
══════════════════════════════════════════════════════════════════════════════

State:      COMPLETED
Generated:  Feb 5, 2026 at 3:45 PM (manually edited)
Initiated:  @dlakehammond

PROGRESS
──────────────────────────────────────────────────────────────────────────────
Points:   48/48 completed (100%)  ████████████████████
Issues:   15 closed

REVIEW
──────────────────────────────────────────────────────────────────────────────

## Sprint Summary

This sprint focused on performance improvements and bug fixes across the
API and dashboard. The team completed all planned work and delivered three
key features ahead of schedule.

## Key Accomplishments

- Reduced API response times by 40% through query optimization
- Resolved 5 customer-reported bugs in the authentication flow
- Shipped the new dashboard loading states for improved UX

──────────────────────────────────────────────────────────────────────────────
Use --features to see the feature breakdown with grouped issues.
Use --schedules to see associated review meeting schedules.
```

### With `--features`

```
FEATURES (3)
──────────────────────────────────────────────────────────────────────────────

API Performance Optimization
  task-tracker  #1  Add due dates to tasks               5 pts  closed
  task-tracker  #3  Add priority levels                   3 pts  closed
  recipe-book   #1  Support ingredient quantities         8 pts  closed

Authentication Bug Fixes
  recipe-book   #2  Search by tag partial match fix       3 pts  closed
  task-tracker  #2  Fix date parsing bug                  3 pts  closed
  + 1 manually added issue

Dashboard UX Improvements
  recipe-book   #3  Add import/export to JSON             8 pts  closed
  recipe-book   #4  Fix partial tag matching in search    3 pts  closed
```

## Implementation Notes

1. **Render `body` as markdown**: Use Glamour to render the review body. The `body` field contains markdown-formatted text. Fall back to `htmlBody` (stripped of tags) if `body` is null.

2. **State handling**: If `state` is `INITIAL`, display "Review has not been generated yet." If `IN_PROGRESS`, display "Review is currently being generated..." with a suggestion to check back shortly.

3. **manuallyEdited indicator**: When `manuallyEdited` is true, note it alongside the generation timestamp so the reader knows the content may differ from the AI output.

4. **Feature issue deduplication**: An issue could theoretically appear in both `aiGeneratedIssues` and `manuallyAddedIssues` for the same feature. Deduplicate by issue ID when displaying.

5. **Empty features**: A feature with zero issues in both connections is unusual but possible after manual editing. Show the feature title with a "(no issues)" note.

6. **Sprint identifier resolution**: Reuse the same sprint resolution logic from `zh sprint show` — the review is just a different view of the same sprint entity.
