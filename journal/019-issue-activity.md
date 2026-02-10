# 019: Issue Activity Command

## Scope

Implemented `zh issue activity <issue>` — the remaining unchecked item in Phase 10 (Issue commands: metadata).

## Work done

- **API research**: Explored ZenHub's `timelineItems` field on the `Issue` type. Each item has `key` (event type string), `data` (JSON blob), and `createdAt`. Also explored the `activityFeed` union (Comment | TimelineItem) but chose `timelineItems` directly since it's more focused. Documented findings in `research/issue/activity.md`.

- **`zh issue activity <issue>`**: New command in `cmd/issue_activity.go` that fetches and displays the ZenHub activity timeline for an issue. Supports:
  - Resolution via all standard issue identifier formats (repo#number, owner/repo#number, ZenHub ID, bare number with --repo)
  - Both `issueByInfo` and `node` query paths (matching existing patterns)
  - Pagination for large activity feeds
  - Human-readable event descriptions for: estimate set/clear, priority set/clear, PR connect/disconnect, pipeline transfer, sprint add/remove, epic add/remove
  - Unknown event types fall back to a formatted version of the key
  - `--output=json` with full raw event data

- **`--github` flag**: When specified, also fetches GitHub timeline events and merges them chronologically with ZenHub events. Handles: LabeledEvent, UnlabeledEvent, AssignedEvent, UnassignedEvent, ClosedEvent, ReopenedEvent, CrossReferencedEvent, IssueComment, RenamedTitleEvent, MilestonedEvent, DemilestonedEvent, MergedEvent, HeadRefDeletedEvent. Source tags ([ZenHub]/[GitHub]) are shown only when --github is active. Gracefully handles missing GitHub access with a warning.

- **Tests**: 17 new tests covering:
  - Activity with events / no events
  - JSON output
  - GitHub merged timeline
  - GitHub flag without access configured (warning)
  - Help text
  - Unit tests for all ZenHub event description parsers
  - Unit tests for GitHub event description parsers

## Files changed

- `cmd/issue_activity.go` — new command implementation
- `cmd/issue_activity_test.go` — tests
- `research/issue/activity.md` — API research documentation
- `ROADMAP.md` — checked off Phase 10 activity items
