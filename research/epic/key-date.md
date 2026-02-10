# zh epic key-date

Manage key dates (milestones) within a ZenHub epic.

## API

### KeyDate Type

The `KeyDate` object has the following fields:

| Field | Type | Description |
|-------|------|-------------|
| `id` | `ID!` | Unique identifier |
| `date` | `ISO8601Date!` | Date in YYYY-MM-DD format |
| `description` | `String!` | Name/description of the key date |
| `color` | `String` | Optional color (nullable) |

### Querying Key Dates

Key dates are available as a connection field on `ZenhubEpic`:

```graphql
query GetEpicKeyDates($id: ID!) {
  node(id: $id) {
    ... on ZenhubEpic {
      id
      title
      keyDates(first: 50) {
        totalCount
        nodes {
          id
          date
          description
          color
        }
      }
    }
  }
}
```

### Creating a Key Date

**Mutation:** `createZenhubEpicKeyDate`

```graphql
mutation CreateZenhubEpicKeyDate($input: CreateZenhubEpicKeyDateInput!) {
  createZenhubEpicKeyDate(input: $input) {
    keyDate {
      id
      date
      description
      color
    }
    zenhubEpic {
      id
      title
    }
  }
}
```

**Input:**

| Field | Type | Required |
|-------|------|----------|
| `zenhubEpicId` | `ID!` | yes |
| `date` | `ISO8601Date!` | yes |
| `description` | `String!` | yes |
| `clientMutationId` | `String` | no |

### Updating a Key Date

**Mutation:** `updateZenhubEpicKeyDate`

**Input:**

| Field | Type | Required |
|-------|------|----------|
| `keyDateId` | `ID!` | yes |
| `date` | `ISO8601Date!` | yes |
| `description` | `String!` | yes |
| `clientMutationId` | `String` | no |

Note: Both `date` and `description` are required for update — partial updates are not supported.

### Deleting a Key Date

**Mutation:** `deleteZenhubEpicKeyDate`

```graphql
mutation DeleteZenhubEpicKeyDate($input: DeleteZenhubEpicKeyDateInput!) {
  deleteZenhubEpicKeyDate(input: $input) {
    keyDate {
      id
      date
      description
    }
    zenhubEpic {
      id
      title
    }
  }
}
```

**Input:**

| Field | Type | Required |
|-------|------|----------|
| `keyDateId` | `ID!` | yes |
| `clientMutationId` | `String` | no |

## Legacy Epic Key Dates

The API also exposes `CreateIssueKeyDateInput` and `DeleteIssueKeyDateInput` for issue-level key dates (which would apply to legacy epics). However, the `IssueKeyDate` type does not exist in the schema, suggesting this feature may be incomplete or deprecated for legacy epics.

The current implementation only supports ZenHub epics; legacy epics return an error.

## Notes

- Color is exposed on the `KeyDate` type but cannot be set via `CreateZenhubEpicKeyDateInput` (only via `CreateIssueKeyDateInput`).
- The `remove` command resolves key dates by name (case-insensitive match on `description`), not by ID.
- The SPEC uses "name" for the user-facing concept; the API field is `description`.
