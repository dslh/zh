# zh pipeline alias

Set a shorthand name that can be used to reference the pipeline in future calls to `zh`.

## Feasibility

**Fully Feasible** - This is primarily a local configuration operation. The only API interaction required is verifying the pipeline exists.

## API Query

### Pipeline Existence Verification

Before setting an alias, verify the pipeline exists:

```graphql
query VerifyPipeline($workspaceId: ID!, $pipelineName: String!) {
  workspace(id: $workspaceId) {
    pipelinesConnection(first: 50) {
      nodes {
        id
        name
      }
    }
  }
}
```

Alternatively, if pipeline data is already cached, no API call is required - just validate against the cache.

## Implementation

This command modifies the local config file (`~/.config/zh/config.yml`):

```yaml
aliases:
  pipelines:
    ip: "In Progress"
    review: "Code Review"
    backlog: "Product Backlog"
```

The alias can then be used anywhere a pipeline identifier is accepted:

```bash
zh issue move mpt#123 ip          # Moves to "In Progress"
zh board --pipeline=review         # Shows "Code Review" pipeline
zh pipeline show backlog           # Shows details for "Product Backlog"
```

## Suggested Flags

| Flag | Description |
|------|-------------|
| `--delete` | Remove an existing alias |
| `--list` | List all pipeline aliases (alternative to inspecting config) |
| `--force` | Overwrite an existing alias without prompting |

## Validation Rules

1. **Pipeline must exist** - Verify the target pipeline exists in the workspace (from cache or API)
2. **Alias must be unique** - Cannot conflict with another alias
3. **Alias cannot shadow pipeline names** - Warn if alias matches an existing pipeline's exact name
4. **Reserved words** - Reject aliases that match command names or flags

## Error Cases

- Pipeline not found: Exit code 4
- Alias already exists (without --force): Prompt user or exit code 2
- Alias shadows a real pipeline name: Warning (not an error)

## Caching Requirements

Uses the existing pipeline cache (`~/.cache/zh/pipelines-{workspace_id}.json`) for validation. If the cache is empty or stale, fetches fresh pipeline data before validating.

## GitHub API

Not needed.

## Limitations

None. This is a straightforward local configuration operation.
