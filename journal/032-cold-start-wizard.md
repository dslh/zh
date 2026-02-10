# 032 — Cold start wizard

Phase 15 (partial): Interactive cold start setup wizard.

## Changes

- Added Bubble Tea and Bubbles dependencies for terminal UI
- Created `cmd/setup.go` with the cold start wizard:
  - Detects first run (missing API key in config or env)
  - Prompts for ZenHub API key via text input
  - Validates the key by fetching the workspace list
  - Presents workspace selection via Bubble Tea list component
  - Offers GitHub access method selection: `gh` CLI, PAT, or none
  - Validates `gh auth status` when gh CLI is selected
  - Prompts for PAT when token method is selected
  - Shows list of unavailable features when "none" is selected
  - Writes config file with all selections
- Exposed `zh setup` subcommand for manual re-configuration
- Wired wizard into root command:
  - `PersistentPreRunE` triggers setup for unconfigured subcommands
  - `RunE` on root command triggers setup when running bare `zh`
  - Commands that don't need config are skipped: version, help, setup, completion, cache
  - Non-TTY environments get a clear error message directing to env vars
- Created `cmd/setup_test.go` with 18 tests:
  - `needsSetup()` detection (with/without API key)
  - API key validation (success and auth failure via mock server)
  - Model initialization and GitHub choices
  - GitHub method labels and descriptions
  - PersistentPreRunE skip behavior (version, help, cache clear)
  - Config write round-trip
  - `gh auth status` check (success and failure)
  - Non-interactive error handling (setup command and pre-run hook)
  - Workspace choice filter values
  - Setup help text

## Design decisions

- Used an injectable `isInteractive` function variable rather than checking `os.Stdin`/`os.Stdout` directly, enabling reliable non-TTY testing
- Made `ghAuthCheckFunc` injectable for testing without requiring the actual `gh` CLI
- Reused existing `fetchAllWorkspaces()` from workspace.go for API key validation
- Wizard renders via stderr (Bubble Tea output) while final confirmation goes to stdout
- PersistentPreRunE skips the root command path since RunE handles it with different behavior (shows help vs error in non-TTY)

## Files changed

- `cmd/setup.go` — new: cold start wizard implementation
- `cmd/setup_test.go` — new: 18 tests for wizard logic
- `cmd/root.go` — added `PersistentPreRunE` and `RunE`
- `go.mod`, `go.sum` — added Bubble Tea + Bubbles dependencies
- `ROADMAP.md` — checked off cold start wizard items
