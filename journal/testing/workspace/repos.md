# Manual Testing: `zh workspace repos`

## Summary

All tests passed. No bugs found. The command correctly lists repositories connected to the current workspace, supports GitHub enrichment via `--github`, and produces proper JSON output.

## Test Environment

- Workspace: Dev Test (`69866ab95c14bf002977146b`)
- Repos: `dlakehammond/task-tracker`, `dlakehammond/recipe-book`

## Tests Performed

### Basic listing (`zh workspace repos`)

**Result: PASS**

Displays a table with REPO, GITHUB ID, PRIVATE, ARCHIVED columns. Both connected repos appear with correct metadata:

```
REPO                         GITHUB ID     PRIVATE    ARCHIVED
────────────────────────────────────────────────────────────────────────────────
dlakehammond/task-tracker    1152464818    no         no
dlakehammond/recipe-book     1152470189    no         no

Total: 2 repo(s)
```

Verified data matches ZenHub GraphQL API response.

### GitHub enrichment (`zh workspace repos --github`)

**Result: PASS**

Shows enriched columns (DESCRIPTION, LANGUAGE, STARS, PRIVATE) using GitHub API data:

```
REPO                         DESCRIPTION                    LANGUAGE    STARS    PRIVATE
────────────────────────────────────────────────────────────────────────────────────────
dlakehammond/task-tracker    A simple CLI task tracker      Python      0        no
dlakehammond/recipe-book     A digital recipe collection    Python      0        no

Total: 2 repo(s)
```

Verified descriptions, languages, and star counts match GitHub GraphQL API responses.

### JSON output (`zh workspace repos --output=json`)

**Result: PASS**

Outputs a well-formed JSON array with id, name, ownerName, ghId, isPrivate, isArchived fields for each repo.

### JSON + GitHub (`zh workspace repos --github --output=json`)

**Result: PASS**

JSON output includes a nested `github` object with description, language, and stars fields for each repo.

### Verbose mode (`zh workspace repos --verbose`)

**Result: PASS**

Logs the ZenHub GraphQL request/response to stderr. With `--github`, also logs individual GitHub API calls for each repo.

### GitHub not configured (`--github` with `method: none`)

**Result: PASS**

Shows warning on stderr: "Warning: GitHub access not configured -- ignoring --github flag". Falls back to non-enriched table output. Exit code 0.

### No workspace configured

**Result: PASS**

Errors with: "Error: no API key configured -- set ZH_API_KEY or run 'zh setup' in a terminal". Exit code 3.

### Piped output (no TTY)

**Result: PASS**

Output is clean text without color codes when piped through `cat`.

### Caching

**Result: PASS**

After running the command, the repo cache file (`repos-{workspaceId}.json`) is populated with repo ID, ghId, name, and ownerName for issue reference resolution.

### Help (`zh workspace repos --help`)

**Result: PASS**

Shows usage, description, and available flags (--github, --output, --verbose).

## Bugs Found

None.
