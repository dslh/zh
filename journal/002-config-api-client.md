# 002 — Configuration & API Client

Phase 1 complete. The configuration layer, GraphQL API client, and exit code system are in place.

## What was done

- **Exit codes** (`internal/exitcode/`):
  - Constants: Success (0), GeneralError (1), UsageError (2), AuthFailure (3), NotFound (4)
  - `Error` type carrying an exit code, with `Unwrap` support
  - Constructor helpers: `General`, `Generalf`, `Usage`, `Auth`, `NotFoundError`
  - `ExitCode(err)` extracts the code from any error (defaults to 1 for plain errors)
  - Wired into `main.go` — prints to stderr and exits with the correct code

- **Config management** (`internal/config/`):
  - XDG-compliant path resolution (`$XDG_CONFIG_HOME/zh/config.yml` or `~/.config/zh/config.yml`)
  - Viper setup: reads YAML config, binds `ZH_API_KEY`, `ZH_WORKSPACE`, `ZH_GITHUB_TOKEN` env vars
  - Typed `Config` struct with `GitHubConfig` and `AliasConfig` sub-structs
  - `Write()` for config persistence (cold start wizard, workspace switch)
  - Env vars override config file values (verified by tests)

- **GraphQL API client** (`internal/api/`):
  - HTTP client with Bearer auth header, User-Agent, 30s timeout
  - `Execute(query, variables) -> json.RawMessage` method
  - GraphQL error parsing — surfaces ZenHub error messages as `*GraphQLError`
  - HTTP error handling: 401/403 → exit code 3, 429 → retry-after message, 5xx → exit code 1
  - `--verbose` logging via functional option (`WithVerbose`) — dumps request/response to stderr
  - Rate limit awareness: parses `Retry-After` header on 429
  - Configurable endpoint via `WithEndpoint` (for testing with mock server)
  - Integration test (gated behind `ZH_INTEGRATION=1`) — verified against real ZenHub API

- **Dependencies added**: `github.com/spf13/viper` (and transitive deps)

## Tests

- 4 config tests: load from file, env overrides file, missing config returns zero values, write and read back
- 9 API client tests: auth header, user-agent, data response, variables, GraphQL errors, HTTP auth failure, rate limiting, verbose logging, HTTP errors
- 3 exit code tests: typed error extraction, error messages, constant values
- 1 integration test (skipped by default): real API call to verify full stack
- All 21 tests pass, lint clean
