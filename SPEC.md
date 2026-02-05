# zh - ZenHub CLI

zh is a command line tool. Like GitHub's `gh`, but for ZenHub.

## Supported commands

### `zh board`

View the workspace board — pipelines and their issues.

| Subcommand | Description |
|---|---|
| `zh board` | Display all pipelines with their issues (default view) |
| `zh board --pipeline=<name>` | Filter to a single pipeline |
| `zh board --view=<name>` | Apply a saved view (filter preset) |

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
| `zh pipeline create <name>` | Create a new pipeline. `--position=<n>`, `--description=<text>` |
| `zh pipeline edit <name>` | Update a pipeline's name, position, or description |
| `zh pipeline delete <name> --into=<name>` | Delete a pipeline, moving its issues into the target pipeline |
| `zh pipeline alias <name> <alias>` | Set a shorthand name that can be used to reference the pipeline in future calls to `zh` |

### `zh issue`

View and manage issues.

| Subcommand | Description |
|---|---|
| `zh issue view <issue>` | View issue details: title, state, estimate, pipeline, assignees, labels, connected PRs, blockers |
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
| `zh epic view <epic>` | View epic details: title, state, dates, child issues, assignees |
| `zh epic create <title>` | Create an epic. `--body=<text>`, `--repo=<repo>` |
| `zh epic edit <epic>` | Update title/body. `--title=<text>`, `--body=<text>` |
| `zh epic delete <epic>` | Delete an epic |
| `zh epic set-state <epic> <state>` | Set state: `open`, `todo`, `in_progress`, `closed`. `--apply-to-issues` |
| `zh epic set-dates <epic>` | Set start/end dates. `--start=<date>`, `--end=<date>` |
| `zh epic add <epic> <issue>...` | Add issues to an epic |
| `zh epic remove <epic> <issue>...` | Remove issues from an epic |

### `zh sprint`

View and manage sprints.

| Subcommand | Description |
|---|---|
| `zh sprint list` | List sprints (active, upcoming, recent) |
| `zh sprint view [sprint]` | View sprint details and issues. Defaults to active sprint |
| `zh sprint add <issue>...` | Add issues to the active sprint. `--sprint=<id>` to target a specific sprint |
| `zh sprint remove <issue>...` | Remove issues from a sprint |

### `zh workspace`

Workspace information and configuration.

| Subcommand | Description |
|---|---|
| `zh workspace list` | List available workspaces |
| `zh workspace view` | Show current workspace details: name, repos, pipelines, sprint config |
| `zh workspace select` | Switch the default workspace |
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
 - When `--repo` flag is used to provide a GitHub repository, PRs can then be specified by the branch name. Requires GitHub API access.

### Pipeline identifiers

Pipelines can be specified:
 - By their identifier
 - By their exact name
 - By any substring of the name that is unique within the workspace
 - By an alias set with `zh pipeline alias`

### Caching

ZenHub's API offers limited search capabilities. For example, to find the `repositoryGhId` for a repo based on its human-readable name (e.g. `gohiring/mpt`) requires listing all repos in a workspace and searching over all results. So `zh` will cache information about workspaces, pipelines, and GitHub repositories for faster lookup. Caching should be indefinite, fetching from the API should only take place when something can't be found in the cache.

### Cold start

When run for the first time, `zh` enters an interactive mode. First it asks for an API key. Then it fetches a list of available workspaces from the API, and asks the user to select a default workspace from the list. Then, the tool asks if it should access GitHub via the `gh` CLI tool, using a PAT (personal access token), or not at all. If PAT is specified, the tool asks for one. If "not at all" is selected, the user should be informed of the features that will not work.

### Autocomplete

The command should come with subcommand autocompletion support for major shells
