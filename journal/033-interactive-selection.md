# 033 Interactive Selection

Phase 15 (partial): Bubble Tea interactive selector for `show` commands.

## Changes

- Created `cmd/interactive.go` with a reusable Bubble Tea list selector component:
  - `selectItem` type implementing `list.Item` interface with title, description, and filter support
  - `selectModel` Bubble Tea model with keyboard navigation, filtering, and cancel/select handling
  - `runInteractiveSelect()` runs the TUI selector with TTY detection
  - `interactiveOrArg()` helper that dispatches between interactive mode and positional arguments

- Wired `--interactive` / `-i` flag to all five `show` commands:
  - `zh issue show -i` — fetches issues from all pipelines, shows ref + title + pipeline + estimate
  - `zh epic show -i` — fetches epics from workspace roadmap, shows title + state + issue count
  - `zh sprint show -i` — fetches all sprints, shows name + state + dates
  - `zh pipeline show -i` — fetches pipelines, shows name + issue count + stage
  - `zh workspace show -i` — fetches all workspaces, shows name + org + repo count

- Each show command now accepts `MaximumNArgs(1)` instead of `ExactArgs(1)`, with explicit validation when neither arg nor `--interactive` is provided

- Non-TTY detection: `--interactive` fails early with a clear error message in non-TTY environments, before making any API calls

- Added `cmd/interactive_test.go` with 26 tests:
  - selectItem interface tests (Title, Description, FilterValue)
  - selectModel unit tests (Init, Ctrl+C cancel, Esc cancel, Enter select, View states)
  - interactiveOrArg logic tests (with arg, no arg, non-TTY)
  - Flag registration audit for all 5 show commands
  - Help text audit for all 5 show commands
  - Non-TTY fallback tests for all 5 show commands
  - Missing argument tests for issue, epic, and pipeline show
