# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What is this?

`zh` is a CLI tool for ZenHub (like GitHub's `gh`, but for ZenHub). Built with Go and Cobra, it manages boards, issues, epics, sprints, pipelines, and workspaces from the terminal. It talks to ZenHub's GraphQL API, ZenHub's REST v1 API (legacy epic operations), and optionally GitHub's GraphQL API.

## Build & Test Commands

```bash
make build          # Build binary with version/commit/date ldflags → ./zh
make test           # go test ./...
make lint           # golangci-lint run ./...
make run ARGS="..." # Build and run with test config/cache dirs
go test -run TestBoardDefault ./cmd  # Run a single test
```

## Architecture

**Command layer** (`cmd/`): Cobra commands organized by domain (board, issue, epic, sprint, pipeline, workspace). Each command file has a corresponding `_test.go`. Commands delegate to internal packages for all business logic.

**API layer** (`internal/api/`): `client.go` wraps ZenHub's GraphQL API. `rest.go` wraps ZenHub's REST v1 API (used only for legacy GitHub-issue-backed epic add/remove). Both accept `api.Option` functional options; tests inject `api.WithEndpoint(mockURL)`.

**Resolution layer** (`internal/resolve/`): Parses flexible entity references (e.g. `repo#123`, `owner/repo#123`, ZenHub IDs, aliases) into API-ready identifiers. Cache-backed for performance.

**Output layer** (`internal/output/`): Formatters for detail views, tabular lists, JSON, markdown (via Glamour), progress bars, colors. Respects `--output=json` and `NO_COLOR`.

**Config** (`internal/config/`): Viper-based, XDG-compliant (`~/.config/zh/config.yml`). Supports aliases for pipelines and epics. Env vars: `ZH_API_KEY`, `ZH_WORKSPACE`, `ZH_GITHUB_TOKEN`.

**Cache** (`internal/cache/`): File-backed JSON cache at `~/.cache/zh/`, workspace-scoped, using invalidate-on-miss pattern.

**GitHub integration** (`internal/gh/`): Optional layer for features needing GitHub data (close/reopen issues, activity timelines). Authenticates via `gh` CLI or a personal access token.

**Exit codes** (`internal/exitcode/`): 0=success, 1=general error, 2=usage error, 3=auth failure, 4=not found.

## Testing Patterns

Tests use a mock HTTP server (`internal/testutil/server.go`) to intercept GraphQL and REST calls. The standard test setup:

1. Create mock server: `ms := testutil.NewMockServer(t)`
2. Register handlers: `ms.HandleQuery("OperationName", responseStruct)` or `ms.HandleREST("/path", statusCode, responseStruct)`
3. Override the API client factory: swap `apiNewFunc` to inject `api.WithEndpoint(ms.URL())`
4. Set env vars: `t.Setenv("ZH_API_KEY", "test-key")`, `t.Setenv("ZH_WORKSPACE", "ws-id")`
5. Use `t.TempDir()` for `XDG_CONFIG_HOME` and `XDG_CACHE_HOME` to isolate config/cache

Commands are tested by setting `rootCmd.SetArgs(...)` and `rootCmd.SetOut(buf)`, then asserting on the captured output.

## Key Design Decisions

- `apiNewFunc` (defined in `cmd/workspace.go`) is a replaceable factory `var apiNewFunc = api.New` — tests swap it to redirect API calls to the mock server
- `isInteractive` (in `cmd/setup.go`) is a replaceable `var` so tests can control TTY detection
- GraphQL queries are matched by operation name substring in tests, so keep operation names unique and descriptive
- The `--dry-run` flag is supported on all mutation commands
- Batch operations support `--continue-on-error` for partial failure tolerance

## Linting

golangci-lint with `misspell` and `gofmt` enabled (see `.golangci.yml`).
