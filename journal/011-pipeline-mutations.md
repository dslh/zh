# 011 - Pipeline Mutation Commands

Implemented the four remaining Phase 6 pipeline commands: `create`, `edit`, `delete`, and `alias`.

## Changes

- **`zh pipeline create <name>`**: Creates a new pipeline in the workspace via `createPipeline` mutation. Supports `--position` (0-indexed), `--description`, and `--dry-run` flags. Invalidates pipeline cache after creation. JSON output supported.

- **`zh pipeline edit <name>`**: Updates an existing pipeline's name, position, or description via `updatePipeline` mutation. Resolves the target pipeline by name/substring/alias/ID. Requires at least one change flag (`--name`, `--position`, `--description`). Supports `--dry-run` showing before/after state. Invalidates pipeline cache after edit.

- **`zh pipeline delete <name> --into=<name>`**: Deletes a pipeline and moves all its issues to the destination pipeline via `deletePipeline` mutation. Both source and destination are resolved via the standard pipeline resolver. Validates source and destination are different. Fetches issue count for confirmation output. Supports `--dry-run` showing pipeline ID, issue count, and destination details.

- **`zh pipeline alias <name> <alias>`**: Local config-only operation to set shorthand aliases for pipelines. Validates the target pipeline exists via the resolver before saving. Supports `--list` to show all aliases in table format, `--delete` to remove an alias. Aliases are stored in `config.yml` under `aliases.pipelines` and work anywhere a pipeline identifier is accepted.

## Files

- `cmd/pipeline_mutations.go` — New file with all four command implementations, GraphQL mutations, flag definitions, and init registration
- `cmd/pipeline_mutations_test.go` — Tests covering: create (basic, with flags, dry-run, JSON), edit (name, position, no-flags error, dry-run, JSON), delete (basic, dry-run, same-target error, JSON), alias (set, list, delete, not-found, already-exists, resolution integration)

## Testing

- All unit tests pass
- All commands verified manually against the Dev Test workspace:
  - Created "Testing" pipeline at position 1 with description
  - Edited pipeline name from "Testing" to "QA"
  - Created alias "qa" pointing to "QA" pipeline, verified alias resolves in `show` command
  - Deleted "QA" pipeline with issues moved to "Todo"
  - Cleaned up alias
