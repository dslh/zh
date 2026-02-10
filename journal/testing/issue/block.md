# Manual testing: zh issue block / blockers / blocking

## Commands tested

- `zh issue block <blocker> <blocked>`
- `zh issue blockers <issue>`
- `zh issue blocking <issue>`

## Test environment

Workspace: Dev Test (`69866ab95c14bf002977146b`)
Repos: `dlakehammond/task-tracker`, `dlakehammond/recipe-book`

## zh issue block

### Basic issue-to-issue block (repo#number)

```
$ zh issue block task-tracker#2 task-tracker#1
Marked task-tracker#2 as blocking task-tracker#1.

Note: Blocks cannot be removed via the API. Use ZenHub's web UI to remove blocking relationships.
```

Verified via `zh issue blockers task-tracker#1` and `zh issue blocking task-tracker#2`.

### --dry-run

```
$ zh issue block task-tracker#3 task-tracker#4 --dry-run
Would mark task-tracker#3 as blocking task-tracker#4

  task-tracker#3 (ISSUE, blocking)
  task-tracker#4 (ISSUE, blocked)
```

### Cross-repo block (dry-run)

```
$ zh issue block recipe-book#2 task-tracker#3 --dry-run
Would mark recipe-book#2 as blocking task-tracker#3

  recipe-book#2 (ISSUE, blocking)
  task-tracker#3 (ISSUE, blocked)
```

### Cross-repo block (actual)

```
$ zh issue block recipe-book#2 task-tracker#4
Marked recipe-book#2 as blocking task-tracker#4.
```

Verified via `zh issue blockers task-tracker#4`, which showed both `task-tracker#3` and `recipe-book#2` as blockers.

### --repo flag with bare numbers

```
$ zh issue block --repo=task-tracker 3 4 --dry-run
Would mark task-tracker#3 as blocking task-tracker#4

  task-tracker#3 (ISSUE, blocking)
  task-tracker#4 (ISSUE, blocked)
```

### owner/repo#number format

```
$ zh issue block dlakehammond/task-tracker#3 dlakehammond/task-tracker#4 --dry-run
Would mark task-tracker#3 as blocking task-tracker#4

  task-tracker#3 (ISSUE, blocking)
  task-tracker#4 (ISSUE, blocked)
```

### JSON output (--dry-run)

```
$ zh issue block task-tracker#3 task-tracker#4 --dry-run --output=json
{
  "blocked": { "id": "...", "ref": "task-tracker#4", "type": "ISSUE" },
  "blocking": { "id": "...", "ref": "task-tracker#3", "type": "ISSUE" },
  "dryRun": true
}
```

### JSON output (actual execution)

```
$ zh issue block task-tracker#3 task-tracker#4 --output=json
{
  "blocked": { "id": "...", "ref": "task-tracker#4", "type": "ISSUE" },
  "blocking": { "id": "...", "ref": "task-tracker#3", "type": "ISSUE" }
}
```

### Issue blocking epic (--blocked-type=epic)

```
$ zh issue block task-tracker#3 "Q1 Platform Improvements" --blocked-type=epic
Marked task-tracker#3 as blocking Q1 Platform Improvements.
```

### Epic blocking issue (--blocker-type=epic, dry-run)

```
$ zh issue block "Bug Bash Sprint" task-tracker#4 --blocker-type=epic --dry-run
Would mark Bug Bash Sprint as blocking task-tracker#4

  Bug Bash Sprint (ZENHUB_EPIC, blocking)
  task-tracker#4 (ISSUE, blocked)
```

### Invalid --blocker-type

```
$ zh issue block task-tracker#3 task-tracker#4 --blocker-type=invalid
Error: invalid --blocker-type "invalid" — must be 'issue' or 'epic'
```

Exit code: 2 (usage error).

### Invalid --blocked-type

```
$ zh issue block task-tracker#3 task-tracker#4 --blocked-type=bogus
Error: invalid --blocked-type "bogus" — must be 'issue' or 'epic'
```

Exit code: 2 (usage error).

### Missing argument

```
$ zh issue block task-tracker#3
Error: accepts 2 arg(s), received 1
```

Exit code: 2.

### Duplicate block

```
$ zh issue block task-tracker#2 task-tracker#1
Error: creating blockage: Not unique
```

Exit code: 1. API correctly rejects duplicate blocking relationships.

### Self-blocking

```
$ zh issue block task-tracker#1 task-tracker#1
Error: creating blockage: Validation failed: Target is reserved
```

Exit code: 1. API correctly rejects self-blocking.

## zh issue blockers

### Issue with blockers

```
$ zh issue blockers task-tracker#1
task-tracker#1 is blocked by:

  task-tracker#2  Task list crashes when no tasks exist  (open)
```

### Issue with no blockers

```
$ zh issue blockers recipe-book#1
recipe-book#1 has no blockers.
```

### --repo flag

```
$ zh issue blockers --repo=task-tracker 1
task-tracker#1 is blocked by:

  task-tracker#2  Task list crashes when no tasks exist  (open)
```

### owner/repo#number format

```
$ zh issue blockers dlakehammond/task-tracker#1
task-tracker#1 is blocked by:

  task-tracker#2  Task list crashes when no tasks exist  (open)
```

### ZenHub ID

```
$ zh issue blockers Z2lkOi8vcmFwdG9yL0lzc3VlLzM4NjI2MTgzMA
task-tracker#1 is blocked by:

  task-tracker#2  Task list crashes when no tasks exist  (open)
```

### JSON output (with blockers)

```
$ zh issue blockers task-tracker#1 --output=json
{
  "blockers": [{ "id": "...", "number": 2, "ref": "task-tracker#2", ... }],
  "issue": { "id": "...", "number": 1, "ref": "task-tracker#1", "title": "Add due dates to tasks" }
}
```

### JSON output (no blockers)

```
$ zh issue blockers recipe-book#1 --output=json
{
  "blockers": [],
  "issue": { "id": "...", "number": 1, "ref": "recipe-book#1", "title": "Support ingredient quantities" }
}
```

### Missing argument

```
$ zh issue blockers
Error: accepts 1 arg(s), received 0
```

Exit code: 2.

## zh issue blocking

### Issue blocking other issues

```
$ zh issue blocking task-tracker#2
task-tracker#2 is blocking:

  task-tracker#1  Add due dates to tasks  (open)
```

### Issue not blocking anything

```
$ zh issue blocking recipe-book#1
recipe-book#1 is not blocking anything.
```

### Issue blocking both epic and issue

```
$ zh issue blocking task-tracker#3
task-tracker#3 is blocking:

  [epic] Q1 Platform Improvements  (open)
  task-tracker#4  Should we support subtasks?  (open)
```

### --repo flag

```
$ zh issue blocking --repo=task-tracker 2
task-tracker#2 is blocking:

  task-tracker#1  Add due dates to tasks  (open)
```

### ZenHub ID

```
$ zh issue blocking Z2lkOi8vcmFwdG9yL0lzc3VlLzM4NjI2MTgzMQ
task-tracker#2 is blocking:

  task-tracker#1  Add due dates to tasks  (open)
```

### JSON output (with items)

```
$ zh issue blocking task-tracker#3 --output=json
{
  "blocking": [
    { "id": "...", "state": "OPEN", "title": "Q1 Platform Improvements", "type": "ZenhubEpic" },
    { "id": "...", "number": 4, "ref": "task-tracker#4", ... }
  ],
  "issue": { "id": "...", "number": 3, "ref": "task-tracker#3", "title": "Add color output for task list" }
}
```

### JSON output (empty)

```
$ zh issue blocking recipe-book#1 --output=json
{
  "blocking": [],
  "issue": { "id": "...", "number": 1, "ref": "recipe-book#1", "title": "Support ingredient quantities" }
}
```

## Cross-verification

Blocking relationships created during testing were verified from both sides:
- `zh issue blockers` confirmed blockers appeared on the blocked issue
- `zh issue blocking` confirmed the blocker was listed as blocking the expected issues
- `zh issue show task-tracker#1` confirmed the "BLOCKED BY" section appeared with the correct blocker

## Bugs found and fixed

### Duplicate epic title in dry-run output

When using `--blocker-type=epic` or `--blocked-type=epic`, the dry-run output displayed the epic title twice. For example:

```
Bug Bash Sprint Bug Bash Sprint (ZENHUB_EPIC, blocking)
```

**Root cause:** For epics, `blockItem.Ref` and `blockItem.Title` are both set to the epic title. The `MutationDryRun` function renders `Ref + " " + Title`, causing duplication.

**Fix:** Added `blockItemDryRunTitle()` helper in `cmd/issue_block.go` that returns an empty string when `Ref == Title` (the epic case), preventing the duplicate. After fix:

```
Bug Bash Sprint (ZENHUB_EPIC, blocking)
```

## Help output

All three commands (`block`, `blockers`, `blocking`) display correct help text with appropriate flags, examples, and descriptions.
