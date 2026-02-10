# Manual Testing: `zh label list`

## Summary

All tests passed. No bugs found.

## Test Environment

- Workspace: Dev Test (`69866ab95c14bf002977146b`)
- Repositories: `dlakehammond/task-tracker`, `dlakehammond/recipe-book`
- Both repos have the same 10 default GitHub labels (bug, documentation, duplicate, enhancement, Epic, good first issue, help wanted, invalid, question, wontfix)

## Tests Performed

### Basic output

```
$ zh label list
LABEL               COLOR
────────────────────────────────────────────────────────────────────────────────
bug                 #d73a4a
documentation       #0075ca
duplicate           #cfd3d7
enhancement         #a2eeef
Epic                #4660F9
good first issue    #7057ff
help wanted         #008672
invalid             #e4e669
question            #d876e3
wontfix             #ffffff

Total: 10 label(s)
```

- Labels sorted alphabetically (case-insensitive: "Epic" sorts between "enhancement" and "good first issue")
- Deduplication working: 20 labels across 2 repos reduced to 10 unique labels
- Colors displayed with `#` prefix
- Footer shows correct count

### JSON output (`--output=json`)

Returns a JSON array of label objects with `id`, `name`, and `color` fields. All 10 labels present, sorted alphabetically.

### Verbose mode (`--verbose`)

Outputs the GraphQL request/response to stderr, including the query, variables, status code, and response body. Standard output still shows the tabular format.

### Help text (`--help`)

- `zh label list --help` shows command description and available flags
- `zh label --help` shows the label command group with its subcommands
- Both display correctly

### Bare `zh label` (no subcommand)

Displays the help text for the label command group, listing `list` as an available subcommand.

### Invalid flag (`--badarg`)

Returns exit code 2 (usage error) with an appropriate error message.

### Invalid output format (`--output=invalid`)

Falls through to default tabular format (non-"json" values are treated as default). This is consistent with other commands.

## Bugs Found

None.
