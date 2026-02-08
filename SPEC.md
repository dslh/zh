# zh - ZenHub CLI

zh is a command line tool. Like GitHub's `gh`, but for ZenHub.

## Supported commands

### `zh board`

View the workspace board — pipelines and their issues.

| Subcommand | Description |
|---|---|
| `zh board` | Display all pipelines with their issues (default view) |
| `zh board --pipeline=<name>` | Filter to a single pipeline |

### `zh pipeline`

Manage pipelines (board columns).

| Subcommand | Description |
|---|---|
| `zh pipeline list` | List all pipelines in the workspace |
| `zh pipeline show <name>` | View details about a pipeline and the issues in it |
| `zh pipeline create <name>` | Create a new pipeline. `--position=<n>`, `--description=<text>` |
| `zh pipeline edit <name>` | Update a pipeline's name, position, or description |
| `zh pipeline delete <name> --into=<name>` | Delete a pipeline, moving its issues into the target pipeline |
| `zh pipeline alias <name> <alias>` | Set a shorthand name that can be used to reference the pipeline in future calls to `zh` |
| `zh pipeline automations <name>` | List configured automations for a pipeline |

### `zh issue`

View and manage issues.

| Subcommand | Description |
|---|---|
| `zh issue list` | List issues in the workspace. `--pipeline=<name>`, `--sprint=<id>`, `--epic=<epic>`, `--assignee=<user>`, `--label=<label>`, `--estimate=<value>`, `--no-estimate` |
| `zh issue show <issue>` | View issue details: title, state, estimate, pipeline, assignees, labels, connected PRs, blockers |
| `zh issue move <issue>... <pipeline>` | Move one or more issues to a pipeline. `--position=<top\|bottom\|n>` |
| `zh issue estimate <issue> <value>` | Set the estimate on an issue. Omit value to clear |
| `zh issue close <issue>...` | Close one or more issues |
| `zh issue reopen <issue>... --pipeline=<name>` | Reopen issues into a pipeline. `--position=<top\|bottom>` |
| `zh issue connect <issue> <pr>` | Connect a PR to an issue |
| `zh issue disconnect <issue> <pr>` | Disconnect a PR from an issue |
| `zh issue block <blocker> <blocked>` | Mark `<blocker>` as blocking `<blocked>`. Supports `--type=issue\|epic` for either side |
| `zh issue priority <issue>... <priority>` | Set priority on issues. Omit priority to clear |
| `zh issue label add <issue>... <label>...` | Add labels to issues |
| `zh issue label remove <issue>... <label>...` | Remove labels from issues |
| `zh issue activity <issue>` | Show ZenHub activity feed (pipeline moves, estimate changes, etc.). `--github` to include GitHub timeline events |
| `zh issue blockers <issue>` | List issues and epics blocking this issue |
| `zh issue blocking <issue>` | List issues and epics that this issue is blocking |

**Note:** When using `zh issue block`, blocks can be created but cannot be removed via the API. Use ZenHub's web UI to remove blocking relationships.

### `zh epic`

Manage ZenHub Epics.

| Subcommand | Description |
|---|---|
| `zh epic list` | List epics in the workspace |
| `zh epic show <epic>` | View epic details: title, state, dates, child issues, assignees |
| `zh epic create <title>` | Create an epic. `--body=<text>`, `--repo=<repo>` |
| `zh epic edit <epic>` | Update title/body. `--title=<text>`, `--body=<text>` |
| `zh epic delete <epic>` | Delete an epic |
| `zh epic set-state <epic> <state>` | Set state: `open`, `todo`, `in_progress`, `closed`. `--apply-to-issues` |
| `zh epic set-dates <epic>` | Set start/end dates. `--start=<date>`, `--end=<date>` |
| `zh epic add <epic> <issue>...` | Add issues to an epic |
| `zh epic remove <epic> <issue>...` | Remove issues from an epic |
| `zh epic alias <epic> <alias>` | Set a shorthand name that can be used to reference the epic in future calls to `zh` |
| `zh epic progress <epic>` | Show completion status (issue count and estimate progress) |
| `zh epic estimate <epic> <value>` | Set estimate on an epic. Omit value to clear |
| `zh epic assignee add <epic> <user>...` | Add assignees to an epic |
| `zh epic assignee remove <epic> <user>...` | Remove assignees from an epic |
| `zh epic label add <epic> <label>...` | Add labels to an epic |
| `zh epic label remove <epic> <label>...` | Remove labels from an epic |
| `zh epic key-date list <epic>` | List key dates (milestones) within an epic |
| `zh epic key-date add <epic> <name> <date>` | Add a key date to an epic |
| `zh epic key-date remove <epic> <name>` | Remove a key date from an epic |

**Legacy epics:** ZenHub has two types of epics—standalone ZenHub epics and legacy epics backed by a GitHub issue. For legacy epics, `edit`, `set-state`, `add`, and `remove` require GitHub API access (via `gh` CLI or PAT). These commands will fail with an error if GitHub access is not configured.

### `zh sprint`

View and manage sprints.

| Subcommand | Description |
|---|---|
| `zh sprint list` | List sprints (active, upcoming, recent) |
| `zh sprint show [sprint]` | View sprint details and issues. Defaults to active sprint |
| `zh sprint add <issue>...` | Add issues to the active sprint. `--sprint=<id>` to target a specific sprint |
| `zh sprint remove <issue>...` | Remove issues from a sprint |
| `zh sprint velocity` | Show velocity trends for recent sprints |
| `zh sprint scope [sprint]` | Show scope change history for a sprint. Defaults to active sprint |
| `zh sprint review [sprint]` | View sprint retrospective. Defaults to active sprint |

### `zh workspace`

Workspace information and configuration.

| Subcommand | Description |
|---|---|
| `zh workspace list` | List available workspaces. `--favorites`, `--recent` |
| `zh workspace show [name]` | Show workspace details: name, repos, pipelines, sprint config. Defaults to current workspace |
| `zh workspace switch <name>` | Switch the default workspace |
| `zh workspace repos` | List repos connected to the workspace |
| `zh workspace stats` | Show workspace metrics (velocity, automations) |

### `zh cache`

Manage the local cache.

| Subcommand | Description |
|---|---|
| `zh cache clear` | Clear all cached data. `--workspace` to clear only current workspace cache |

### `zh version`

Display version information.

| Subcommand | Description |
|---|---|
| `zh version` | Show version, build date, and commit hash |

### `zh label`

View labels available in the workspace.

| Subcommand | Description |
|---|---|
| `zh label list` | List all labels in the workspace |

### `zh priority`

View priorities configured for the workspace.

| Subcommand | Description |
|---|---|
| `zh priority list` | List workspace priorities with their colors |

## General features

### Output format

All commands output in markdown format. All commands support `--output=json`, which does what you would expect.

### Issue identifiers

Commands that operate on issues and PRs accept any of the following identifiers:
 - ZenHub ID, e.g. Z2lkOi8vcmFwdG9yL0lzc3VlLzU2Nzg5
 - GitHub owner/repo#id format, e.g. gohiring/mpt#1234
 - GitHub repo#id format, e.g. mpt#1234, provided the workspace is not linked to multiple repos with the same name but different owners
 - `--repo` param with GitHub number, as in `--repo=mpt 1234` or `--repo=gohiring/mpt 1234`, useful when referencing multiple issues/PRs in the same repo, e.g. `--repo=mpt 1234 2345`
 - When `--repo` flag is used to provide a GitHub repository, PRs can then be specified by the branch name. Requires GitHub API or CLI access, e.g. `gh pr list --repo {owner}/{repo} --head {branch}`

### Pipeline identifiers

Pipelines can be specified:
 - By their identifier
 - By their exact name
 - By any substring of the name that is unique within the workspace (if multiple pipelines match, the command errors with a list of candidates)
 - By an alias set with `zh pipeline alias`

### Epic identifiers

ZenHub has two types of epics: legacy epics (backed by a GitHub issue) and standalone ZenHub epics. Commands that operate on epics accept any of the following identifiers:
 - ZenHub ID, e.g. Z2lkOi8vcmFwdG9yL1plbmh1YkVwaWMvMTIzNDU
 - Exact title match
 - Any substring of the title that is unique within the workspace (if multiple epics match, the command errors with a list of candidates)
 - GitHub owner/repo#id or repo#id format, for legacy epics that are backed by a GitHub issue
 - An alias set with `zh epic alias`

### Sprint identifiers

Sprints can be specified:
 - By their ZenHub ID
 - By their name, e.g. "Sprint 42"
 - By any unique substring of the name (if multiple sprints match, the command errors with a list of candidates)
 - By relative reference: `current` (or omit for commands that default to active sprint), `next`, `previous`

### Caching

ZenHub's API offers limited search capabilities. For example, to find the `repositoryGhId` for a repo based on its human-readable name (e.g. `gohiring/mpt`) requires listing all repos in a workspace and searching over all results. So `zh` will cache information about workspaces, pipelines, and GitHub repositories for faster lookup.

Cache invalidation strategy: **invalidate on miss**. When a lookup fails to find an entity in the cache, the entire cache for that resource type is refreshed from the API. This handles renamed entities gracefully (old name misses, triggering a refresh that pulls in the new name) and avoids stale data without requiring TTLs or background refresh.

### Cold start

When run for the first time, `zh` enters an interactive mode. First it asks for an API key. Then it fetches a list of available workspaces from the API, and asks the user to select a default workspace from the list. Then, the tool asks if it should access GitHub via the `gh` CLI tool, using a PAT (personal access token), or not at all. If PAT is specified, the tool asks for one. If "not at all" is selected, the user should be informed of the features that will not work.

### GitHub access

Some features require GitHub API access (via `gh` CLI or PAT). Without GitHub access configured:

**Will not work:**
- `zh epic edit`, `zh epic set-state`, `zh epic add`, `zh epic remove` for legacy epics (those backed by a GitHub issue)
- `zh issue activity --github` (the flag will be ignored)
- Branch name resolution when specifying PRs via `--repo`

**Will have limited output:**
- `zh issue show` for PRs will not include review status, merge status, or CI status
- `zh issue show` will not include issue author, reactions, or participants
- `zh workspace repos` will not include repo language, description, or stars

### --help

A --help flag is available for all subcommands, command groups, and for zh itself, providing complete documentation for the tool.

### --dry-run

A --dry-run flag is available for commands that modify state. When specified, the command displays what actions would be taken without executing them. This is useful for:
 - Previewing destructive operations like `zh pipeline delete` or `zh epic delete`
 - Verifying bulk operations like `zh issue move` or `zh issue close` before committing
 - Confirming which entity was matched when using substring identifiers for pipelines or epics

Commands that support --dry-run:
 - `zh pipeline create`, `zh pipeline edit`, `zh pipeline delete`
 - `zh issue move`, `zh issue estimate`, `zh issue close`, `zh issue reopen`, `zh issue connect`, `zh issue disconnect`, `zh issue block`, `zh issue priority`, `zh issue label add`, `zh issue label remove`
 - `zh epic create`, `zh epic edit`, `zh epic delete`, `zh epic set-state`, `zh epic set-dates`, `zh epic add`, `zh epic remove`, `zh epic estimate`, `zh epic assignee add`, `zh epic assignee remove`, `zh epic label add`, `zh epic label remove`, `zh epic key-date add`, `zh epic key-date remove`
 - `zh sprint add`, `zh sprint remove`

### show --interactive/-i

All `show` subcommands support an --interactive flag which can be passed in place of an entity identifier. When passed, the user is presented with the entities returned by the corresponding `list` subcommand and can use the arrow and enter keys to select one to be shown.

### Autocomplete

The command should come with subcommand autocompletion support for major shells

## Technical details

### Language

Go. The stated goal is to be "like GitHub's `gh`", which is written in Go. Beyond that, Go is well-suited for CLI tools:
- Single binary distribution with no runtime dependencies
- Fast startup time
- Excellent CLI ecosystem
- Trivial cross-platform compilation

### ZenHub API

ZenHub uses a GraphQL API:
- **Endpoint:** `https://api.zenhub.com/public/graphql`
- **Authentication:** Bearer token in `Authorization` header
- **Rate limits:** Standard GraphQL rate limiting applies

### Libraries

| Purpose | Library | Notes |
|---------|---------|-------|
| CLI framework | Cobra | Industry standard, powers `gh`, `kubectl`, `docker`. Handles subcommands, flags, help generation, and shell completions |
| Config management | Viper | Pairs with Cobra, handles config files, env vars, and flag binding |
| Terminal markdown | Glamour | What `gh` uses for rendering markdown in the terminal |
| Interactive selection | Bubble Tea + Lip Gloss | For `--interactive` prompts and the cold start wizard |
| GitHub API | go-github | For direct GitHub access beyond what `gh` provides |
| HTTP client | Standard library | `net/http` is sufficient for ZenHub's GraphQL API |

### Configuration

Follow the XDG Base Directory spec. Config lives at `~/.config/zh/config.yml` (or `$XDG_CONFIG_HOME/zh/config.yml`).

```yaml
api_key: zh_xxx
workspace: Z2lkOi8vcmFwdG9yL1dvcmtzcGFjZS8xMjM0
github:
  method: gh  # or "pat" or "none"
  token: ghp_xxx  # only if method=pat
aliases:
  pipelines:
    ip: "In Progress"
    review: "Code Review"
  epics:
    auth: "Z2lkOi8vcmFwdG9yL1plbmh1YkVwaWMvMTIzNDU"
```

### Cache

Cache lives at `~/.cache/zh/` (or `$XDG_CACHE_HOME/zh/`). Simple JSON files, one per resource type:
- `workspaces.json` — workspace metadata
- `pipelines-{workspace_id}.json` — pipelines per workspace
- `repos-{workspace_id}.json` — repo name to GitHub ID mappings
- `sprints-{workspace_id}.json` — sprint ID, name, state, dates (short TTL; sprints change frequently)
- `epics-{workspace_id}.json` — epic ID, title, type (ZenHub vs legacy)
- `users-{workspace_id}.json` — GitHub login to ZenHub user ID mapping (required for assignee resolution)
- `labels-{repo_id}.json` — label name to ID mapping (repo-scoped)
- `priorities-{workspace_id}.json` — priority name, ID, color (workspace-scoped)
- `estimates-{repo_id}.json` — valid estimate values (for validation and autocompletion)

A `zh cache clear` command should be available for manual cache invalidation.

### Environment variables

For CI/CD and scripting, credentials and settings can be provided via environment variables:
- `ZH_API_KEY` — ZenHub API key
- `ZH_WORKSPACE` — Default workspace ID
- `ZH_GITHUB_TOKEN` — GitHub PAT (when not using `gh` CLI)

Environment variables take precedence over config file values.

### Exit codes

Consistent exit codes for scripting:
- `0` — Success
- `1` — General error (API failure, network issues)
- `2` — Usage error (invalid flags, missing arguments)
- `3` — Authentication failure
- `4` — Entity not found (issue, pipeline, epic doesn't exist or couldn't be resolved)

### Batch operations

Commands that accept multiple items (e.g., `zh issue move <issue>...`, `zh issue close <issue>...`) stop on first error. This ensures predictable state after failure and avoids wasting API calls when the root cause is systemic (auth, permissions, network). Users can re-run the command with remaining items after fixing the issue.

### Debug mode

A `--verbose` flag is available on all commands. When set, logs API requests and responses to stderr. Useful for troubleshooting and bug reports.

### Distribution

- **Homebrew** — Primary distribution channel for macOS/Linux (`brew install zh`)
- **go install** — For Go developers (`go install github.com/.../zh@latest`)
- **Binary releases** — Prebuilt binaries for major platforms, attached to GitHub releases

goreleaser handles all of the above from a single configuration.

### Pagination

List commands transparently fetch all pages from the API by default. A `--limit=<n>` flag is available to cap results when only a subset is needed.

### Testing

Unit tests with mocked API responses. The ZenHub GraphQL API is available via MCP for schema introspection and read-only validation during development.

#### Test accounts and resources

A dedicated GitHub account (`dlakehammond`) and ZenHub account are available for integration testing. The `gh` CLI is authenticated as `dlakehammond`.

**ZenHub workspace:** "Dev Test" (`69866ab95c14bf002977146b`)
- Organization: `hambend@gmail.com` (`Z2lkOi8vcmFwdG9yL1plbmh1Yk9yZ2FuaXphdGlvbi8xNjQ5NDc`)
- Pipelines: Todo, Doing

**Repositories:**

| Repo | GitHub ID | Issues | PRs |
|------|-----------|--------|-----|
| `dlakehammond/task-tracker` | 1152464818 | #1 enhancement, #2 bug, #3 enhancement, #4 question | #5 fixes #2, #6 closes #1 |
| `dlakehammond/recipe-book` | 1152470189 | #1 enhancement, #2 bug, #3 enhancement | #4 fixes #2 |

Both repos are connected to the Dev Test workspace. MCP servers for both ZenHub and GitHub GraphQL APIs are configured and authenticated. The repositories are each cloned locally in the repos/ directory.

The accounts, and everything in them, exist only for the purpose of building the `zh` CLI tool. It is permitted to execute any operations, even write operations such as creating or closing GitHub issues or moving them between ZenHub pipelines. This can be helpful during development to verify query design and during testing to verify implementation.

## API Research

Research has been undertaken to understand how each subcommand might be implemented in terms of ZenHub API calls that would need to be made. See the research/ directory, which follows the naming convention `research/<topic>/<command>.md`, e.g. research/issue/list.md for `zh issue list`.
