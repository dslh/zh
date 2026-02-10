# Epic Key Dates

Implemented the `zh epic key-date` command group for managing key dates (milestones) within ZenHub epics.

## Changes

- **New file `cmd/epic_key_date.go`**: `key-date` subcommand group with `list`, `add`, and `remove` commands
  - `list <epic>`: fetches and displays key dates in table format (DATE, NAME columns)
  - `add <epic> <name> <date>`: creates a key date via `createZenhubEpicKeyDate` mutation
  - `remove <epic> <name>`: resolves key date by name (case-insensitive), deletes via `deleteZenhubEpicKeyDate` mutation
  - All commands support `--output=json`; `add` and `remove` support `--dry-run`
  - Legacy epics return a clear error (key dates are ZenHub-epic only)
- **Updated `cmd/epic.go`**: added `keyDates` field to `epicShowZenhubQuery` and `epicDetailZenhub` struct, rendered in the KEY DATES section of `epic show`
- **New file `cmd/epic_key_date_test.go`**: 14 tests covering list/add/remove with dry-run, JSON output, legacy epic errors, invalid date, and name-not-found cases
- **New file `research/epic/key-date.md`**: documents the GraphQL API for key dates (query, create, update, delete mutations, field types)

## API Notes

- The `KeyDate` type has fields: `id`, `date` (ISO8601Date), `description` (String!), `color` (String nullable)
- The SPEC uses "name" as the user-facing term; the API field is `description`
- Color cannot be set when creating key dates on ZenHub epics (only on issue key dates)
- The `remove` command fetches existing key dates first, then matches by description to find the key date ID for deletion
