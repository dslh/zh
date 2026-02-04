# zh - ZenHub CLI

zh is a command line tool. Like GitHub's `gh`, but for ZenHub.

## Supported commands

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

### Caching

ZenHub's API offers limited search capabilities. For example, to find the `repositoryGhId` for a repo based on its human-readable name (e.g. `gohiring/mpt`) requires listing all repos in a workspace and searching over all results. So `zh` will cache information about workspaces, pipelines, and GitHub repositories for faster lookup. Caching should be indefinite, fetching from the API should only take place when something can't be found in the cache.

### Cold start

When run for the first time, `zh` enters an interactive mode. First it asks for an API key. Then it fetches a list of available workspaces from the API, and asks the user to select a default workspace from the list. Then, the tool asks if it should access GitHub via the `gh` CLI tool, using a PAT (personal access token), or not at all. If PAT is specified, the tool asks for one. If "not at all" is selected, the user should be informed of the features that will not work.

