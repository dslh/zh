# zh epic set-dates

Set start and/or end dates on an epic.

## API

ZenHub has two types of epics requiring different mutations:

### Standalone ZenHub Epics

**Mutation:** `updateZenhubEpicDates`

```graphql
mutation UpdateZenhubEpicDates($input: UpdateZenhubEpicDatesInput!) {
  updateZenhubEpicDates(input: $input) {
    zenhubEpic {
      id
      title
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
    "startOn": "2025-03-01",
    "endOn": "2025-03-31"
  }
}
```

### Legacy Epics (backed by GitHub issues)

**Mutation:** `updateEpicDates`

```graphql
mutation UpdateEpicDates($input: UpdateEpicDatesInput!) {
  updateEpicDates(input: $input) {
    epic {
      id
      startOn
      endOn
      issue {
        title
        number
      }
    }
  }
}
```

**Variables:**
```json
{
  "input": {
    "epicId": "Z2lkOi8vcmFwdG9yL0VwaWMvMTE4NjQ1Nw",
    "startOn": "2025-03-01",
    "endOn": "2025-03-31"
  }
}
```

The legacy mutation also accepts an optional `roadmapId` parameter, though its purpose is unclear—it may control which roadmap view the dates appear on.

### Querying Current Dates

To show the user what changed, fetch the epic's current dates before or after the mutation:

```graphql
query GetEpic($id: ID!) {
  node(id: $id) {
    ... on Epic {
      id
      startOn
      endOn
      issue {
        title
        number
        repository { name }
      }
    }
    ... on ZenhubEpic {
      id
      title
      startOn
      endOn
    }
  }
}
```

### Date Format

Dates use `ISO8601Date` format: `YYYY-MM-DD` (e.g., `2025-03-15`). Both `startOn` and `endOn` are nullable—pass `null` to clear a date.

## Flags and Parameters

| Flag | Description |
|------|-------------|
| `<epic>` | Required. Epic identifier (ZenHub ID, title substring, GitHub `repo#number` for legacy epics, or alias) |
| `--start=<date>` | Set start date (YYYY-MM-DD format) |
| `--end=<date>` | Set end date (YYYY-MM-DD format) |
| `--clear-start` | Clear the start date |
| `--clear-end` | Clear the end date |

At least one of `--start`, `--end`, `--clear-start`, or `--clear-end` must be provided.

### Usage Examples

```bash
# Set both dates
zh epic set-dates "LinkedIn MVP" --start=2025-03-01 --end=2025-03-31

# Set only start date
zh epic set-dates api#2969 --start=2025-03-01

# Clear end date
zh epic set-dates my-epic-alias --clear-end

# Set start and clear end in one command
zh epic set-dates "Feature X" --start=2025-04-01 --clear-end
```

## Caching

**Required cached data:**
- Workspace ID (to resolve epic by title/substring)
- Epic list with titles and IDs (for title/substring matching)
- Repository GH IDs (for resolving legacy epics by `repo#number` format)

The epic list can be fetched on-demand and cached. When an epic is specified by exact ZenHub ID, no cache lookup is needed.

## Limitations

- No validation that `endOn` is after `startOn`—the API accepts any dates
- Date-only granularity (no time component)
- Cannot set dates on multiple epics in a single command (unlike some other `zh` commands that accept multiple issues)

## Related Functionality

The API also supports **key dates** for epics via:
- `createZenhubEpicKeyDate` / `createIssueKeyDate`
- `updateZenhubEpicKeyDate` / `updateIssueKeyDate`
- `deleteZenhubEpicKeyDate` / `deleteIssueKeyDate`

Key dates are additional milestone markers within an epic's timeline. This could support a future `zh epic key-date` subcommand group:
- `zh epic key-date add <epic> <date> --name=<name>`
- `zh epic key-date list <epic>`
- `zh epic key-date remove <epic> <key-date-id>`
