# 005 — Output Framework Completion

Phase 3 complete: the three remaining output framework items.

## What was done

- **Glamour markdown renderer** (`markdown.go`):
  - `RenderMarkdown(w, content, width)` — renders user-authored markdown (issue descriptions, epic bodies) to the terminal
  - Uses `WithAutoStyle()` to detect terminal background for appropriate theming
  - Emoji support enabled via `WithEmoji()`
  - Configurable word-wrap width; pass 0 for Glamour's default
  - No-op on empty content
  - Added `glamour` dependency (v0.10.0)

- **Issue reference formatter** (`issueref.go`):
  - `NewIssueRefFormatter(repoFullNames)` — takes all `owner/repo` names in the workspace
  - Detects ambiguous repo names (same name under different owners)
  - `FormatRef(owner, repo, number)` → `repo#number` (short) or `owner/repo#number` (long, when ambiguous)
  - Per spec: "handled by a shared formatting function that checks the workspace context"

- **Pagination helpers** (`pagination.go`):
  - `AddPaginationFlags(cmd, &limit, &all)` — registers `--limit` (default 100) and `--all` on a Cobra command, marked mutually exclusive
  - `EffectiveLimit(limit, all)` — resolves the actual limit (0 = unlimited when `--all`)
  - `Truncate[T](items, limit)` — generic slice truncation with a `truncated` flag for footer messaging
  - `DefaultLimit = 100` constant

## Tests

- 4 markdown renderer tests: basic rendering, empty input, complex content (headings, lists, code blocks), zero-width fallback
- 5 issue reference tests: short form, long form, mixed ambiguity, empty repos, single repo
- 6 pagination tests: flag registration, mutual exclusivity, effective limit resolution, truncation edge cases (under/at/over limit, zero, negative, empty)
- All 83 project tests pass, lint clean
