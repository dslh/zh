# Manual Testing: `zh epic progress`

## Summary

`zh epic progress <epic>` shows completion status for an epic: issue count (closed/total) and estimate progress (completed/total), with progress bars.

## Tests Performed

### Identifier types

| Identifier type | Input | Result |
|---|---|---|
| Exact title | `'Q1 Platform Improvements'` | OK |
| Title substring | `'Q1 Platform'` | OK |
| ZenHub ID | `Z2lkOi8vcmFwdG9yL1plbmh1YkVwaWMvMjMyNDIy` | OK |
| Legacy epic repo#number | `task-tracker#8` | OK |
| Legacy epic owner/repo#number | `dlakehammond/task-tracker#8` | OK |
| Alias | `q1` (set via `zh epic alias`) | OK |

### Output formats

| Scenario | Result |
|---|---|
| Default (human) output with child issues | Correct: shows issues and estimates progress bars |
| Default output with no child issues | Correct: shows "No child issues." |
| Default output with 100% progress | Correct: fully filled progress bar |
| Default output with no estimates | Correct: only shows Issues line, omits Estimates |
| JSON output (`-o json`) with child issues | Correct: structured JSON with issues/estimates counts |
| JSON output with no child issues | Correct: all counts are 0 |
| Verbose output (`-v`) | Correct: shows GraphQL query and response |

### Data accuracy

Verified progress data against ZenHub GraphQL API directly:
- Issues: 2/5 completed (40%) -- matched API response
- Estimates: 6/14 completed (42%) -- matched API response (1+3+5+2+3=14 total, 3+3=6 closed)

### Error handling

| Scenario | Result | Exit code |
|---|---|---|
| No arguments | `Error: accepts 1 arg(s), received 0` | 2 (was 1, fixed) |
| Non-existent epic | `Error: epic "nonexistent" not found` | 4 |
| Ambiguous substring | Lists matching candidates | 2 |
| Unknown flag | `Error: unknown flag: --badflg` | 2 (was 1, fixed) |

### Edge cases

| Scenario | Result |
|---|---|
| Epic with all issues closed (100%) | Full progress bar, correct percentage |
| Epic with no estimates on any issue | Estimates line omitted entirely |
| ZenHub epic (standalone) | Displays without issue reference prefix |
| Legacy epic (GitHub-backed) | Displays with `repo#number:` prefix in title |

## Bugs Found and Fixed

### Exit code for Cobra argument validation errors (codebase-wide)

**Problem:** Cobra's built-in argument validators (`ExactArgs`, `MinimumNArgs`, etc.) and flag parsing errors return plain `error` values. The `exitcode.ExitCode()` function only recognizes `*exitcode.Error` types, so these errors fell through to exit code 1 (GeneralError) instead of exit code 2 (UsageError).

This affected all commands using Cobra's argument validators, not just `epic progress`.

**Fix:** Updated `cmd/root.go` `Execute()` to detect Cobra usage errors (argument validation, unknown commands/flags) and wrap them as `exitcode.Usage()` errors before returning. Detection is based on known Cobra error message patterns (`arg(s)`, `unknown command`, `unknown flag`, `unknown shorthand flag`).

**Verification:** Missing args now returns exit 2; unknown flags return exit 2; not-found errors still return exit 4; success still returns exit 0.
