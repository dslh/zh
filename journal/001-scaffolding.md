# 001 — Project Scaffolding

Phase 0 complete. The `zh` CLI project is bootstrapped and ready for feature development.

## What was done

- Initialized Go module (`github.com/dslh/zh`) with Cobra dependency
- Created directory structure: `cmd/`, `internal/{api,config,cache,resolve,output,gh}/`, `test/{config,cache,fixtures,snapshots}/`
- `main.go` — entry point, delegates to `cmd.Execute()`
- Root Cobra command with `--verbose` and `--output` global flags, `SilenceUsage` and `SilenceErrors` enabled
- `zh version` subcommand — prints version, commit, and build date; build vars wired via ldflags
- Makefile with `build`, `test`, `lint`, `run`, and `clean` targets; `run` sets `XDG_CONFIG_HOME` and `XDG_CACHE_HOME` to project-local `test/` directories
- Pre-populated `test/config/zh/config.yml` with Dev Test workspace credentials
- Installed golangci-lint (v2), configured `.golangci.yml` with misspell linter and gofmt formatter
- Updated `.gitignore` to exclude `zh` binary and `test/config/`, `test/cache/`
- Tests: root help output, version subcommand output, unknown command error
- Test infrastructure in `internal/testutil/`:
  - `MockServer` — mock HTTP server for GraphQL with substring-based query matching
  - `LoadFixture` / `LoadFixtureString` — reads files from `test/fixtures/`
  - `AssertSnapshot` — golden file comparison with `-update-snapshots` flag
- All tests pass, lint clean, build produces working binary
