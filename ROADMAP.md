# zh Development Roadmap

## Phase 0: Project scaffolding

- [x] `go mod init github.com/dslh/zh`
- [x] Set up directory structure:
  ```
  cmd/         # Cobra command definitions, one file per command group
  internal/
    api/       # GraphQL client, queries, mutations
    config/    # Viper config management, XDG paths
    cache/     # Cache read/write/invalidation
    resolve/   # Entity identifier resolution (issues, pipelines, epics, sprints)
    output/    # Markdown and JSON formatters
    gh/        # GitHub API integration (gh CLI / PAT)
  main.go
  ```
- [x] Root Cobra command with `--verbose`, `--output=json` global flags
- [x] `zh version` subcommand (hardcoded for now, wired to build vars later)
- [x] Makefile with `build`, `test`, `lint` targets
- [x] Makefile `run` target that sets `XDG_CONFIG_HOME=test/config` and `XDG_CACHE_HOME=test/cache` for development
- [x] Pre-populate `test/config/zh/config.yml` with test account credentials (from credentials.md) and Dev Test workspace
- [x] Install linter (golangci-lint config)
- [x] First passing test: root command prints help without error
- [x] Test infrastructure: mock HTTP server helpers, test fixtures directory, snapshot test utilities

## Phase 1: Configuration & API client

### Config management
- [x] XDG-compliant config path resolution (`~/.config/zh/config.yml`)
- [x] Viper setup: read config file, bind environment variables (`ZH_API_KEY`, `ZH_WORKSPACE`, `ZH_GITHUB_TOKEN`)
- [x] Config struct with typed access (api key, workspace ID, GitHub method/token, aliases)
- [x] Config write-back for cold start and `workspace switch`
- [x] Tests: env vars override config file, missing config returns zero values

### GraphQL API client
- [x] HTTP client with auth header, user-agent, configurable base URL
- [x] Generic `Execute(query, variables) -> json.RawMessage` method
- [x] Error response parsing: surface ZenHub error messages cleanly
- [x] `--verbose` logging: dump request/response to stderr
- [x] Rate limit awareness (respect retry-after if present)
- [x] Tests: mock HTTP server, verify auth header, test error parsing

### Exit codes
- [x] Define exit code constants (0–4 per spec)
- [x] Wire error types to exit codes throughout Cobra's error handling
- [x] Tests: verify specific exit codes for auth failure, not found, usage error

## Phase 2: Cache framework

- [x] XDG-compliant cache path resolution (`~/.cache/zh/`)
- [x] Generic cache: `Get[T](key) -> (T, bool)`, `Set[T](key, T)`, `Clear(key)`
- [x] Cache file naming per spec (e.g. `pipelines-{workspace_id}.json`)
- [x] Invalidate-on-miss pattern: when a lookup fails, refresh that resource type from API, then retry
- [x] `zh cache clear` command, with `--workspace` flag
- [x] Tests: cache hit, cache miss triggers refresh, clear removes files

## Phase 3: Output framework

- [x] Detail view renderer: entity title, double-line separator, key-value metadata, section headers with single-line separators
- [x] List view renderer: column-aligned tabular output with ALL CAPS headers and separator
- [x] Mutation confirmation renderer: single-item, multi-item, partial failure, dry-run formats
- [x] Progress bar renderer: `fraction unit (percentage)  bar` format with fixed 20-char bar
- [x] Markdown renderer using Glamour for user-authored content (issue descriptions, epic bodies)
- [x] JSON output mode: structured output when `--output=json`
- [x] Color support: palette per spec, respect `NO_COLOR` and non-TTY detection
- [x] Date/time formatting: standalone dates, date ranges, ISO 8601 for JSON
- [x] Missing value rendering: `-` in tables, `None` in detail views
- [x] Issue reference formatting: short form `repo#number`, long form when repos share names
- [x] `--limit` and `--all` flag support for list commands (default 100)
- [x] Tests: snapshot tests for each output format, color vs no-color, JSON mode

## Phase 4: Workspace commands

These establish the foundation — workspace context is required for every other command.

### `zh workspace list`
- [x] Query `viewer.zenhubOrganizations` for all workspaces
- [x] `--favorites` and `--recent` filters
- [x] Highlight current workspace in output
- [x] Cache workspace list
- [x] Tests: list formatting, filter behavior

### `zh workspace show [name]`
- [x] Default to current workspace if no name given
- [x] Display: name, ID, connected repos, pipelines, sprint config
- [x] Resolve workspace by name or substring if argument given
- [x] Tests: default workspace, named workspace, not found

### `zh workspace switch <name>`
- [x] Resolve workspace by name/substring from cached list
- [x] Update config file with new workspace ID
- [x] Clear workspace-scoped caches on switch
- [x] Tests: switch updates config, clears caches

### `zh workspace repos`
- [x] List repos connected to current workspace
- [x] Cache repo name → GitHub ID mappings (critical for issue resolution later)
- [x] With GitHub access: include description, language, stars
- [x] Tests: with and without GitHub access

### `zh workspace stats`
- [x] Show workspace metrics (velocity, automations)
- [x] Tests: with data, empty workspace

## Phase 5: Identifier resolution

Build the resolution layer now — almost every subsequent command depends on it.

### Pipeline resolution
- [x] Resolve by: ZenHub ID, exact name, unique substring, alias
- [x] Ambiguous substring → error with list of candidates
- [x] Cache pipeline list, invalidate-on-miss
- [x] Alias lookup from config
- [x] Tests: each resolution method, ambiguous match error

### Issue/PR resolution
- [x] Parse identifiers: ZenHub ID, `owner/repo#number`, `repo#number`
- [x] `repo#number` → look up repo in cache, resolve `repositoryGhId`, query `issueByInfo`
- [x] `--repo` flag: allow bare issue numbers, resolve repo once
- [x] Branch name resolution when `--repo` is used (requires GitHub access)
- [x] Tests: each identifier format, repo not found, ambiguous repo name

### Epic resolution
- [x] Resolve by: ZenHub ID, exact title, unique substring, alias, `owner/repo#number` (legacy)
- [x] Cache epic list (ID, title, type), invalidate-on-miss
- [x] Alias lookup from config
- [x] Tests: each resolution method, ambiguous match error

### Sprint resolution
- [x] Resolve by: ZenHub ID, name, unique substring, relative reference (`current`, `next`, `previous`)
- [x] Cache sprint list, invalidate-on-miss
- [x] Tests: each resolution method, no active sprint error

## Phase 6: Pipeline commands

### `zh pipeline list`
- [x] List all pipelines in workspace with position order
- [x] Tests: ordering, empty workspace

### `zh pipeline show <name>`
- [x] Resolve pipeline by name/substring/alias
- [x] Display pipeline details and issues within it
- [x] Tests: resolution, output format

### `zh pipeline create <name>`
- [x] `--position`, `--description` flags
- [x] `--dry-run` support
- [x] Invalidate pipeline cache after creation
- [x] Tests: create with flags, dry run output

### `zh pipeline edit <name>`
- [x] Resolve pipeline, update name/position/description
- [x] `--dry-run` support
- [x] Invalidate pipeline cache after edit
- [x] Tests: edit each field, dry run

### `zh pipeline delete <name> --into=<name>`
- [x] Resolve both pipelines
- [x] `--dry-run`: show issue count that would be moved
- [x] Invalidate pipeline cache after deletion
- [x] Tests: delete with target, dry run, missing --into error

### `zh pipeline alias <name> <alias>`
- [x] Resolve pipeline, write alias to config
- [x] Tests: alias creation, alias used in resolution

### `zh pipeline automations <name>`
- [x] Display configured automations for the pipeline
- [x] Tests: with and without automations

## Phase 7: Board

### `zh board`
- [x] Fetch all pipelines with their issues
- [x] Render as columnar markdown view (pipeline name as header, issues listed underneath)
- [x] `--pipeline=<name>` filter to single pipeline
- [x] JSON output: structured pipeline/issue data
- [x] Tests: full board, filtered board, empty pipelines

## Phase 8: Issue commands (core)

### `zh issue list`
- [x] Query issues across all pipelines (parallel API calls)
- [x] Filters: `--pipeline`, `--sprint`, `--epic`, `--assignee`, `--label`, `--estimate`, `--no-estimate`
- [x] `--limit` and `--all` flags (default 100 results)
- [x] Client-side filtering where API doesn't support it
- [x] Tests: no filters, each filter individually, combined filters, limit and all behavior

### `zh issue show <issue>`
- [x] Resolve issue identifier
- [x] Display: title, state, body, estimate, pipeline, assignees, labels, connected PRs, blockers, priority
- [ ] `--interactive` mode: list issues, select one
- [x] With GitHub access: include author, reactions, PR review/merge/CI status
- [x] Tests: full detail output, without GitHub access, interactive mode

### `zh issue move <issue>... <pipeline>`
- [x] Resolve issue(s) and target pipeline
- [x] `--position=<top|bottom|n>` flag
- [x] `--dry-run` support
- [x] Stop-on-first-error by default, `--continue-on-error` to process all items
- [x] Tests: single move, batch move, position flag, dry run, stop-on-error, continue-on-error with partial failure

### `zh issue estimate <issue> <value>`
- [x] Resolve issue, set estimate (omit value to clear)
- [x] Validate estimate against cached valid values
- [x] `--dry-run` support
- [x] Tests: set, clear, invalid estimate

### `zh issue close <issue>...`
- [x] Resolve issue(s), close via API
- [x] `--dry-run` support
- [x] Stop-on-first-error by default, `--continue-on-error` to process all items
- [x] Tests: single close, batch close, already closed, continue-on-error with partial failure

### `zh issue reopen <issue>... --pipeline=<name>`
- [x] Resolve issue(s) and target pipeline
- [x] `--position=<top|bottom>`
- [x] `--dry-run` support
- [x] `--continue-on-error` for batch operations
- [x] Tests: reopen into pipeline, missing pipeline error, continue-on-error with partial failure

## Phase 9: Issue commands (connections & dependencies)

### `zh issue connect <issue> <pr>`
- [ ] Resolve both issue and PR identifiers
- [ ] `--dry-run` support
- [ ] Tests: connect, already connected

### `zh issue disconnect <issue> <pr>`
- [ ] Resolve both identifiers
- [ ] `--dry-run` support
- [ ] Tests: disconnect, not connected

### `zh issue block <blocker> <blocked>`
- [ ] `--type=issue|epic` for either side
- [ ] `--dry-run` support
- [ ] Note: blocks cannot be removed via API (display warning)
- [ ] Tests: issue blocks issue, epic blocks issue, dry run

### `zh issue blockers <issue>`
- [ ] List issues and epics blocking this issue
- [ ] Tests: with blockers, no blockers

### `zh issue blocking <issue>`
- [ ] List issues and epics this issue is blocking
- [ ] Tests: blocking something, blocking nothing

## Phase 10: Issue commands (metadata)

### `zh issue priority <issue>... <priority>`
- [ ] Resolve issue(s) and priority name
- [ ] Omit priority to clear
- [ ] `--dry-run` support
- [ ] `--continue-on-error` for batch operations
- [ ] Tests: set, clear, invalid priority, continue-on-error with partial failure

### `zh issue label add <issue>... <label>...`
- [ ] Resolve issue(s) and label(s)
- [ ] `--dry-run` support
- [ ] `--continue-on-error` for batch operations
- [ ] Tests: add single label, multiple labels, label not found, continue-on-error with partial failure

### `zh issue label remove <issue>... <label>...`
- [ ] Resolve issue(s) and label(s)
- [ ] `--dry-run` support
- [ ] `--continue-on-error` for batch operations
- [ ] Tests: remove, label not on issue, continue-on-error with partial failure

### `zh issue activity <issue>`
- [ ] Fetch ZenHub activity feed (pipeline moves, estimate changes, etc.)
- [ ] `--github` flag: merge in GitHub timeline events (requires GitHub access)
- [ ] Tests: ZenHub-only activity, merged timeline

## Phase 11: Epic commands (ZenHub epics only — legacy deferred)

### `zh epic list`
- [ ] List epics in workspace (ID, title, state, issue count)
- [ ] `--limit` and `--all` flags (default 100 results)
- [ ] Cache epic list
- [ ] Tests: list output, empty workspace

### `zh epic show <epic>`
- [ ] Resolve epic, display: title, state, body, dates, child issues, assignees, labels, estimate
- [ ] `--interactive` mode
- [ ] Tests: full output, interactive

### `zh epic create <title>`
- [ ] `--body`, `--repo` flags
- [ ] `--dry-run` support
- [ ] Invalidate epic cache
- [ ] Tests: create with flags, dry run

### `zh epic edit <epic>`
- [ ] `--title`, `--body` flags
- [ ] `--dry-run` support
- [ ] Tests: edit each field, dry run

### `zh epic delete <epic>`
- [ ] `--dry-run`: show child issue count
- [ ] Invalidate epic cache
- [ ] Tests: delete, dry run

### `zh epic set-state <epic> <state>`
- [ ] States: `open`, `todo`, `in_progress`, `closed`
- [ ] `--apply-to-issues` flag
- [ ] `--dry-run` support
- [ ] Tests: each state transition, apply-to-issues

### `zh epic set-dates <epic>`
- [ ] `--start=<date>`, `--end=<date>` flags
- [ ] `--dry-run` support
- [ ] Tests: set both, set one, clear

### `zh epic add <epic> <issue>...`
- [ ] Resolve epic and issue(s)
- [ ] `--dry-run` support
- [ ] `--continue-on-error` for batch operations
- [ ] Tests: add single, add multiple, issue already in epic, continue-on-error with partial failure

### `zh epic remove <epic> <issue>...`
- [ ] Resolve epic and issue(s)
- [ ] `--dry-run` support
- [ ] `--continue-on-error` for batch operations
- [ ] Tests: remove, issue not in epic, continue-on-error with partial failure

### `zh epic alias <epic> <alias>`
- [ ] Write alias to config
- [ ] Tests: alias creation, alias used in resolution

### `zh epic progress <epic>`
- [ ] Show completion: issue count (closed/total), estimate progress (completed/total)
- [ ] Tests: partial progress, all done, no estimates

### `zh epic estimate <epic> <value>`
- [ ] Set estimate on epic (omit value to clear)
- [ ] `--dry-run` support
- [ ] Tests: set, clear

### `zh epic assignee add <epic> <user>...`
- [ ] Resolve user(s), add to epic
- [ ] `--dry-run` support
- [ ] `--continue-on-error` for batch operations
- [ ] Tests: add, user not found, continue-on-error with partial failure

### `zh epic assignee remove <epic> <user>...`
- [ ] Resolve user(s), remove from epic
- [ ] `--dry-run` support
- [ ] `--continue-on-error` for batch operations
- [ ] Tests: remove, user not assigned, continue-on-error with partial failure

### `zh epic label add <epic> <label>...`
- [ ] Resolve label(s), add to epic
- [ ] `--dry-run` support
- [ ] `--continue-on-error` for batch operations
- [ ] Tests: add, label not found, continue-on-error with partial failure

### `zh epic label remove <epic> <label>...`
- [ ] Resolve label(s), remove from epic
- [ ] `--dry-run` support
- [ ] `--continue-on-error` for batch operations
- [ ] Tests: remove, label not on epic, continue-on-error with partial failure

### `zh epic key-date list <epic>`
- [ ] List key dates (milestones) within an epic
- [ ] Tests: with key dates, none

### `zh epic key-date add <epic> <name> <date>`
- [ ] `--dry-run` support
- [ ] Tests: add, duplicate name

### `zh epic key-date remove <epic> <name>`
- [ ] `--dry-run` support
- [ ] Tests: remove, name not found

## Phase 12: Sprint commands

### `zh sprint list`
- [ ] List sprints: active, upcoming, recent
- [ ] Cache sprint list
- [ ] Tests: with sprints, sprints not configured

### `zh sprint show [sprint]`
- [ ] Default to active sprint
- [ ] Display: name, dates, issues with estimates and pipeline
- [ ] `--interactive` mode
- [ ] Tests: active sprint, named sprint, no active sprint

### `zh sprint add <issue>...`
- [ ] Default to active sprint, `--sprint=<id>` to target specific
- [ ] `--dry-run` support
- [ ] `--continue-on-error` for batch operations
- [ ] Tests: add to active, add to specific, no active sprint, continue-on-error with partial failure

### `zh sprint remove <issue>...`
- [ ] `--dry-run` support
- [ ] `--continue-on-error` for batch operations
- [ ] Tests: remove, issue not in sprint, continue-on-error with partial failure

### `zh sprint velocity`
- [ ] Show velocity trends for recent sprints (points completed per sprint)
- [ ] Tests: with history, no sprints

### `zh sprint scope [sprint]`
- [ ] Show scope change history (issues added/removed during sprint)
- [ ] Tests: with changes, no changes

### `zh sprint review [sprint]`
- [ ] Show details of review associated with a sprint
- [ ] Tests: with data, no data

## Phase 13: Utility commands

### `zh label list`
- [ ] List all labels in workspace (aggregated across repos)
- [ ] Tests: with labels, no labels

### `zh priority list`
- [ ] List workspace priorities with colors
- [ ] Cache priorities
- [ ] Tests: with priorities, none configured

## Phase 14: --help and --dry-run audit

- [ ] Review all `--help` text for accuracy and completeness
- [ ] Verify `--dry-run` is implemented on every command listed in the spec
- [ ] Consistent dry-run output format across all commands
- [ ] Tests: help text doesn't error, dry-run on every applicable command

## Phase 15: Interactive mode & cold start wizard

### Cold start wizard
- [ ] Detect first run (no config file or missing API key)
- [ ] Prompt for ZenHub API key (with Bubble Tea text input)
- [ ] Validate API key by making a test API call
- [ ] Fetch workspace list, present selection prompt
- [ ] GitHub access selection: `gh` CLI / PAT / none
  - If `gh`: verify `gh auth status` works
  - If PAT: prompt for token, validate with a test call
  - If none: display list of features that won't work
- [ ] Write config file with selections
- [ ] Tests: mock API responses, verify config file output

### Interactive selection
- [ ] Bubble Tea list component for entity selection
- [ ] Wire `--interactive` to `pipeline show`, `issue show`, `epic show`, `sprint show`, `workspace show`
- [ ] Handle terminal detection (disable interactive in non-TTY)
- [ ] Tests: non-TTY falls back gracefully

## Phase 16: Shell completions & distribution

### Shell completions
- [ ] Cobra completion for bash, zsh, fish
- [ ] Dynamic completions for pipeline names, sprint names, epic names
- [ ] `zh completion <shell>` command with install instructions

### Distribution
- [ ] goreleaser config: binary builds for macOS (arm64, amd64), Linux (arm64, amd64)
- [ ] Homebrew formula generation
- [ ] `go install` compatibility
- [ ] Wire build vars (version, commit, date) into `zh version`
- [ ] GitHub Actions CI: test, lint, release on tag

## Phase 17: Legacy epic support (deferred)

- [ ] Detect epic type (ZenHub vs legacy) during resolution
- [ ] `zh epic edit` for legacy epics via GitHub API
- [ ] `zh epic set-state` for legacy epics via GitHub API
- [ ] `zh epic add` / `zh epic remove` for legacy epics
- [ ] Graceful error when GitHub access not configured
- [ ] Tests: each operation for both epic types
