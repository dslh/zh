# zh

A command-line interface for [ZenHub](https://www.zenhub.com/). Like GitHub's [`gh`](https://cli.github.com/), but for ZenHub.

Manage your board, issues, epics, sprints, and more from the terminal.

## Installation

### Homebrew (macOS / Linux)

```sh
brew install dslh/tap/zh
```

### Go install

```sh
go install github.com/dslh/zh@latest
```

### Binary releases

Prebuilt binaries for macOS and Linux (amd64, arm64) are attached to each [GitHub release](https://github.com/dslh/zh/releases).

## Getting started

Run any command and `zh` will walk you through setup interactively:

```
$ zh board
? Enter your ZenHub API key: ▊
```

The wizard will:
1. Ask for your [ZenHub API key](https://app.zenhub.com/settings/tokens)
2. Let you pick a default workspace
3. Optionally configure GitHub access (via `gh` CLI or personal access token)

Configuration is stored at `~/.config/zh/config.yml` (or `$XDG_CONFIG_HOME/zh/config.yml`). You can re-run the wizard at any time with `zh setup`.

For non-interactive environments (CI/CD, scripts), set environment variables instead:

```sh
export ZH_API_KEY=zh_xxx
export ZH_WORKSPACE=<workspace-id>
export ZH_GITHUB_TOKEN=ghp_xxx  # optional
```

## Commands

### Board

```sh
zh board                        # View the full board
zh board --pipeline="In Dev"    # Filter to one pipeline
```

### Issues

```sh
zh issue list                             # List issues in the workspace
zh issue list --pipeline=Backlog          # Filter by pipeline
zh issue list --sprint=current --all      # All issues in active sprint
zh issue show mpt#1234                    # View issue details
zh issue show -i                          # Interactive selection
zh issue move mpt#1234 "In Dev"           # Move to a pipeline
zh issue move mpt#1234 mpt#1235 Done      # Batch move
zh issue estimate mpt#1234 5              # Set estimate
zh issue estimate mpt#1234                # Clear estimate
zh issue close mpt#1234                   # Close an issue
zh issue reopen mpt#1234 --pipeline=Todo  # Reopen into a pipeline
zh issue connect mpt#1234 mpt#5678        # Connect a PR to an issue
zh issue disconnect mpt#1234 mpt#5678     # Disconnect a PR
zh issue block mpt#1234 mpt#1235          # Mark 1234 as blocking 1235
zh issue blockers mpt#1234                # List what's blocking an issue
zh issue blocking mpt#1234                # List what an issue blocks
zh issue priority mpt#1234 High           # Set priority
zh issue label add mpt#1234 -- bug        # Add a label
zh issue label remove mpt#1234 -- bug     # Remove a label
zh issue activity mpt#1234                # ZenHub activity feed
zh issue activity mpt#1234 --github       # Include GitHub timeline events
```

### Epics

```sh
zh epic list                              # List epics
zh epic show "Auth Redesign"              # View epic by title substring
zh epic show -i                           # Interactive selection
zh epic create "New Epic" --body="desc"   # Create a ZenHub epic
zh epic create "Legacy" --repo=mpt        # Create a legacy epic
zh epic edit "Auth" --title="Auth v2"     # Edit title/body
zh epic delete "Old Epic"                 # Delete an epic
zh epic set-state "Auth" in_progress      # Set epic state
zh epic set-dates "Auth" --start=2025-01-20 --end=2025-03-01
zh epic add "Auth" mpt#1234 mpt#1235      # Add issues to epic
zh epic remove "Auth" mpt#1234            # Remove issues
zh epic remove "Auth" --all               # Remove all child issues
zh epic progress "Auth"                   # View completion status
zh epic estimate "Auth" 21                # Set estimate
zh epic assignee add "Auth" @alice        # Add assignees
zh epic label add "Auth" backend          # Add labels
zh epic key-date list "Auth"              # List key dates
zh epic key-date add "Auth" "Beta" 2025-02-15
zh epic alias "Auth" auth                 # Set shorthand alias
```

### Sprints

```sh
zh sprint list                    # List sprints
zh sprint show                    # View active sprint
zh sprint show "Sprint 42"       # View specific sprint
zh sprint show -i                 # Interactive selection
zh sprint add mpt#1234            # Add issue to active sprint
zh sprint add mpt#1234 --sprint=next
zh sprint remove mpt#1234        # Remove issue from sprint
zh sprint velocity                # Velocity trends
zh sprint scope                   # Scope change history
zh sprint review                  # Sprint retrospective
```

### Pipelines

```sh
zh pipeline list                          # List pipelines
zh pipeline show "In Dev"                 # View pipeline details
zh pipeline show -i                       # Interactive selection
zh pipeline create "QA" --position=3      # Create pipeline
zh pipeline edit "QA" --name="Testing"    # Rename pipeline
zh pipeline delete "QA" --into="Done"     # Delete, moving issues
zh pipeline alias "In Dev" dev            # Set shorthand alias
zh pipeline automations "In Dev"          # View automations
```

### Workspaces

```sh
zh workspace list                 # List available workspaces
zh workspace show                 # Show current workspace
zh workspace switch "My Board"    # Switch default workspace
zh workspace repos                # List connected repos
zh workspace repos --github       # Include GitHub metadata
zh workspace stats                # Workspace metrics
```

### Utilities

```sh
zh label list             # List labels in the workspace
zh priority list          # List configured priorities
zh cache clear            # Clear all cached data
zh cache clear --workspace  # Clear current workspace cache only
zh version                # Show version info
```

## Identifying entities

### Issues and PRs

Issues can be referenced in several ways:

| Format | Example |
|--------|---------|
| `repo#number` | `mpt#1234` |
| `owner/repo#number` | `gohiring/mpt#1234` |
| ZenHub ID | `Z2lkOi8vcmFwdG9yL0lzc3VlLzU2Nzg5` |
| `--repo` + number | `--repo=mpt 1234 2345` |
| `--repo` + branch | `--repo=mpt feat/login` (requires GitHub access) |

### Pipelines

By exact name, unique substring, ZenHub ID, or alias (`zh pipeline alias`).

### Epics

By exact title, unique substring, ZenHub ID, `repo#number` (legacy epics), or alias (`zh epic alias`).

### Sprints

By name, unique substring, ZenHub ID, or relative reference: `current`, `next`, `previous`.

When a substring matches multiple entities, the command errors with a list of candidates so you can refine your query.

## Output

Human-readable output is the default. Use `--output=json` on any command for structured JSON, useful for scripting:

```sh
zh issue list --pipeline=Backlog --output=json | jq '.[].title'
```

Colors follow a semantic palette (green for success, red for errors, yellow for dry-run, cyan for IDs) and are suppressed automatically when piping or when `NO_COLOR` is set.

## Dry run

Most mutation commands support `--dry-run` to preview what would happen without making changes:

```
$ zh issue move mpt#1234 mpt#1235 "In Dev" --dry-run
Would move 2 issues to "In Development":

  mpt#1234 Fix login button alignment (currently in "Backlog")
  mpt#1235 Update error messages (currently in "New Issues")
```

## Batch operations

Commands that accept multiple items stop on first error by default. Use `--continue-on-error` to process all items and get a summary of successes and failures:

```
$ zh issue close mpt#1234 mpt#1235 mpt#1236 --continue-on-error
Closed 2 of 3 issues:

  mpt#1234 Fix login button alignment
  mpt#1235 Update error messages

Failed:

  mpt#1236 Permission denied
```

## GitHub integration

Some features require GitHub API access, configured during setup as either `gh` CLI (recommended) or a personal access token.

Without GitHub access configured, the following are affected:

- `zh epic edit`, `set-state`, `add`, `remove` will not work for legacy epics (those backed by a GitHub issue)
- `zh issue activity --github` will be ignored
- `zh issue show` for PRs will not include review/merge/CI status
- Branch name resolution via `--repo` will not work

## Shell completions

```sh
# Bash (Linux)
zh completion bash > /etc/bash_completion.d/zh

# Bash (macOS, requires bash-completion)
zh completion bash > $(brew --prefix)/etc/bash_completion.d/zh

# Zsh
zh completion zsh > "${fpath[1]}/_zh"

# Fish
zh completion fish > ~/.config/fish/completions/zh.fish
```

Completions include dynamic suggestions for pipeline names, sprint names, epic titles, workspace names, labels, and priorities.

## Configuration

Config file location: `~/.config/zh/config.yml` (respects `$XDG_CONFIG_HOME`)

```yaml
api_key: zh_xxx
workspace: Z2lkOi8vcmFwdG9yL1dvcmtzcGFjZS8xMjM0
github:
  method: gh    # "gh", "pat", or "none"
  token: ghp_xxx  # only when method: pat
aliases:
  pipelines:
    dev: "In Development"
    review: "Code Review"
  epics:
    auth: "Z2lkOi8vcmFwdG9yL1plbmh1YkVwaWMvMTIzNDU"
```

### Environment variables

| Variable | Description |
|----------|-------------|
| `ZH_API_KEY` | ZenHub API key (overrides config) |
| `ZH_WORKSPACE` | Default workspace ID (overrides config) |
| `ZH_GITHUB_TOKEN` | GitHub PAT (overrides config) |
| `NO_COLOR` | Disable color output |

### Exit codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | General error (API failure, network) |
| 2 | Usage error (invalid flags, missing args) |
| 3 | Authentication failure |
| 4 | Entity not found |

## License

MIT
