# Phase 16 (partial): Shell completions

## Summary

Implemented shell completion support for bash, zsh, and fish, including dynamic completions for entity names.

## Changes

### `zh completion` command (`cmd/completion.go`)
- Added `zh completion bash`, `zh completion zsh`, `zh completion fish` subcommands
- Each generates the appropriate shell completion script to stdout
- Parent `zh completion` shows help with installation instructions for all three shells
- Command is skipped by the setup wizard (already in the skip list)

### Dynamic completions (`cmd/completion_dynamic.go`)
- Cache-based completion functions for all entity types:
  - **Pipelines**: names + aliases from config
  - **Sprints**: names + relative references (current, next, previous)
  - **Epics**: titles + aliases from config
  - **Workspaces**: display names from workspace cache
  - **Repos**: repository names from workspace cache
  - **Labels**: label names from workspace cache
  - **Priorities**: priority names from workspace cache
- Static completions for epic states, position values, and output formats
- All completion functions are best-effort: return empty results on cache miss or config error (no API calls during completion)

### Completion registration (`cmd/completion_register.go`)
- `ValidArgsFunction` on 28 commands that accept entity identifiers as positional args:
  - Pipeline: show, edit, delete, alias, automations
  - Sprint: show, scope, review
  - Epic: show, edit, delete, set-state, set-dates, progress, estimate, alias, add, remove, assignee add/remove, label add/remove, key-date list/add/remove
  - Workspace: show, switch
- `RegisterFlagCompletionFunc` on flags across all commands:
  - `--pipeline` flags (board, issue list, issue reopen, pipeline delete --into)
  - `--sprint` flags (issue list, sprint add, sprint remove)
  - `--epic` flag (issue list)
  - `--repo` flags (20 commands across issue, sprint, and epic groups)
  - `--position` flags (issue move, issue reopen)
  - `--output` global flag

### Tests (`cmd/completion_test.go`)
- Script generation tests: bash, zsh, fish output contains expected content
- Help text test
- Dynamic completion tests for each entity type (pipeline, sprint, epic, workspace, repo, label, priority)
- Edge case tests: no cache, no workspace configured, empty state
- Verification that completion doesn't trigger setup wizard
