# Manual Testing: `zh sprint review`

## Summary

All tests passed. No bugs were found. The command handles all sprint identifier
types, optional flags, JSON output, and edge cases (no review, ambiguous names,
nonexistent sprints) correctly.

## Test Environment

- Workspace: Dev Test (`69866ab95c14bf002977146b`)
- Active sprint: Sprint: Feb 8 - Feb 22, 2026
- A sprint review was generated via the ZenHub `generateSprintReview` mutation
  to test the full output path (issues were temporarily added to the sprint and
  closed, then reopened and removed after testing).

## Tests Performed

### Default behavior (no arguments)

```
zh sprint review
```
Defaults to the active sprint. Shows "No review has been generated" when no
review exists. After generating a review, displays the full review with
rendered markdown, progress bar, initiator, and hint flags.

### Sprint identifiers

| Identifier | Command | Result |
|---|---|---|
| Default (current) | `zh sprint review` | Resolves to active sprint |
| `current` | `zh sprint review current` | Same as default |
| `previous` | `zh sprint review previous` | Resolves to last closed sprint |
| `next` | `zh sprint review next` | Resolves to next upcoming sprint |
| Name substring | `zh sprint review "Feb 8"` | Matches unique sprint |
| Full name substring | `zh sprint review "Jan 22 - Feb 5"` | Matches correct sprint |
| ZenHub ID | `zh sprint review Z2lkOi8v...` | Direct lookup by ID |
| Ambiguous name | `zh sprint review "Sprint:"` | Error listing 20 candidates (exit 2) |
| Nonexistent | `zh sprint review "nonexistent sprint"` | Error with suggestion (exit 4) |

### Flags

| Flag | Behavior |
|---|---|
| `--features` | Shows FEATURES section with AI-grouped issues per feature |
| `--schedules` | Shows SCHEDULES section with review meeting dates and status |
| `--late-closes` | Shows ISSUES CLOSED AFTER REVIEW section (empty in test — no late closes) |
| `--raw` | Outputs review body as raw text (HTML tags visible, no Glamour rendering) |
| `--output=json` | Full JSON output with all sprint and review data |
| `--verbose` | Logs API request/response details to stderr |
| `--help` | Displays help text with usage, flags, and examples |
| All flags combined | `--features --schedules --late-closes` renders all sections |

### Edge cases

- **No review**: Graceful message "No review has been generated for this sprint."
- **Too many arguments**: `zh sprint review arg1 arg2` returns usage error (exit 2)
- **Piped output**: No ANSI color codes when stdout is not a TTY
- **Hints**: When optional sections have data but flags aren't set, hints are
  shown (e.g. "Use --features to see 2 feature group(s).")

### Review content rendering

- Markdown body is rendered via Glamour with proper formatting
- `<b>` HTML tags in the review body are rendered as bold in Glamour mode
- `--raw` flag correctly bypasses Glamour and shows raw body text
- Progress section shows points bar and closed issues count
- Feature breakdown shows issues grouped by feature with repo, number, title, estimate, and state

## Bugs Found

None.
