# zh - ZenHub CLI

zh is a command line tool. Like GitHub's `gh`, but for ZenHub.

## Supported commands

### `zh board`

View the workspace board — pipelines and their issues.

| Subcommand | Description |
|---|---|
| `zh board` | Display all pipelines with their issues (default view) |
| `zh board --pipeline=<name>` | Filter to a single pipeline |
| `zh board --view=<name>` | Apply a saved view (filter preset). Can be combined with --pipeline |

### `zh view`

Manage saved views (board filter presets).

| Subcommand | Description |
|---|---|
| `zh view list` | List your saved views |
| `zh view show <name>` | Show the filters in a saved view |
| `zh view create <name>` | Create a saved view from filter flags. `--assignee=<user>`, `--label=<label>`, `--repo=<repo>`, etc. |
| `zh view delete <name>` | Delete a saved view |

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

### `zh issue`

View and manage issues.

| Subcommand | Description |
|---|---|
| `zh issue list` | List issues in the workspace. `--pipeline=<name>`, `--sprint=<id>`, `--epic=<epic>`, `--assignee=<user>`, `--label=<label>`, `--estimate=<value>`, `--blocked`, `--no-estimate`, `--view=<name>` |
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

### `zh sprint`

View and manage sprints.

| Subcommand | Description |
|---|---|
| `zh sprint list` | List sprints (active, upcoming, recent) |
| `zh sprint show [sprint]` | View sprint details and issues. Defaults to active sprint |
| `zh sprint add <issue>...` | Add issues to the active sprint. `--sprint=<id>` to target a specific sprint |
| `zh sprint remove <issue>...` | Remove issues from a sprint |

### `zh workspace`

Workspace information and configuration.

| Subcommand | Description |
|---|---|
| `zh workspace list` | List available workspaces |
| `zh workspace show [name]` | Show workspace details: name, repos, pipelines, sprint config. Defaults to current workspace |
| `zh workspace switch <name>` | Switch the default workspace |
| `zh workspace repos` | List repos connected to the workspace |

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
 - By any substring of the name that is unique within the workspace
 - By an alias set with `zh pipeline alias`

### Epic identifiers

ZenHub has two types of epics: legacy epics (backed by a GitHub issue) and standalone ZenHub epics. Commands that operate on epics accept any of the following identifiers:
 - ZenHub ID, e.g. Z2lkOi8vcmFwdG9yL1plbmh1YkVwaWMvMTIzNDU
 - Exact title match
 - Any substring of the title that is unique within the workspace
 - GitHub owner/repo#id or repo#id format, for legacy epics that are backed by a GitHub issue
 - An alias set with `zh epic alias`

### Sprint identifiers

Sprints can be specified:
 - By their ZenHub ID
 - By their name, e.g. "Sprint 42"
 - By any unique substring of the name
 - By relative reference: `current` (or omit for commands that default to active sprint), `next`, `previous`

### Caching

ZenHub's API offers limited search capabilities. For example, to find the `repositoryGhId` for a repo based on its human-readable name (e.g. `gohiring/mpt`) requires listing all repos in a workspace and searching over all results. So `zh` will cache information about workspaces, pipelines, and GitHub repositories for faster lookup. Caching should be indefinite, fetching from the API should only take place when something can't be found in the cache.

### Cold start

When run for the first time, `zh` enters an interactive mode. First it asks for an API key. Then it fetches a list of available workspaces from the API, and asks the user to select a default workspace from the list. Then, the tool asks if it should access GitHub via the `gh` CLI tool, using a PAT (personal access token), or not at all. If PAT is specified, the tool asks for one. If "not at all" is selected, the user should be informed of the features that will not work.

### --help

A --help flag is available for all subcommands, command groups, and for zh itself, providing complete documentation for the tool.

### --dry-run

A --dry-run flag is available for commands that modify state. When specified, the command displays what actions would be taken without executing them. This is useful for:
 - Previewing destructive operations like `zh pipeline delete` or `zh epic delete`
 - Verifying bulk operations like `zh issue move` or `zh issue close` before committing
 - Confirming which entity was matched when using substring identifiers for pipelines or epics

Commands that support --dry-run:
 - `zh view create`, `zh view delete`
 - `zh pipeline create`, `zh pipeline edit`, `zh pipeline delete`
 - `zh issue move`, `zh issue estimate`, `zh issue close`, `zh issue reopen`, `zh issue connect`, `zh issue disconnect`, `zh issue block`, `zh issue priority`, `zh issue label add`, `zh issue label remove`
 - `zh epic create`, `zh epic edit`, `zh epic delete`, `zh epic set-state`, `zh epic set-dates`, `zh epic add`, `zh epic remove`
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

### Debug mode

A `--verbose` flag is available on all commands. When set, logs API requests and responses to stderr. Useful for troubleshooting and bug reports.

### Distribution

- **Homebrew** — Primary distribution channel for macOS/Linux (`brew install zh`)
- **go install** — For Go developers (`go install github.com/.../zh@latest`)
- **Binary releases** — Prebuilt binaries for major platforms, attached to GitHub releases

goreleaser handles all of the above from a single configuration.

### Pagination

List commands transparently fetch all pages from the API by default. A `--limit=<n>` flag is available to cap results when only a subset is needed.
