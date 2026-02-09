# 004 — Output Framework

Phase 3 partial: core output rendering infrastructure in `internal/output/`.

## What was done

- **Color support** (`color.go`):
  - ANSI color functions: Green, Red, Yellow, Cyan, Dim, Bold (plus `f` variants for formatting)
  - Respects `NO_COLOR` env var (any value, including empty)
  - Suppresses color when stdout is not a TTY (piped/redirected)
  - All output legible without color — functions return input unchanged when disabled

- **Detail view renderer** (`detail.go`):
  - `NewDetailWriter(w, entityType, title)` — writes ALL CAPS title with `══` double-line separator
  - `Fields([]KeyValue)` — right-aligned keys for visual alignment, auto-computed width
  - `Field(key, value)` — single key-value pair without alignment
  - `Section(name)` — ALL CAPS section header with `──` single-line separator
  - Both separator types span 80 characters

- **List view renderer** (`list.go`):
  - `NewListWriter(w, headers...)` — column-aligned tabular output
  - Auto-computed column widths from headers and data
  - `Flush()` / `FlushWithFooter(footer)` for rendering
  - Separator is minimum 80 characters, expands for wider tables
  - 4-space column gap

- **Mutation confirmation renderer** (`mutation.go`):
  - `MutationSingle` — one-line confirmation
  - `MutationBatch` — header + ref-aligned indented list
  - `MutationPartialFailure` — success block then red "Failed:" block
  - `MutationDryRun` — yellow "Would" prefix with dim context annotations

- **Progress bar** (`progress.go`):
  - `FormatProgress(completed, total)` → `"34/52 completed (65%)  █████████████░░░░░░░"`
  - Fixed 20-char bar using `█` filled and `░` remaining
  - Handles edge cases: zero total, zero progress, complete

- **JSON output** (`json.go`):
  - `JSON(w, v)` — writes indented JSON to any writer
  - `IsJSON(format)` — helper for checking `--output=json` flag

- **Date formatting** (`date.go`):
  - `FormatDate` → `"Jan 20, 2025"`
  - `FormatDateRange` → same-month `"Jan 20 → 31, 2025"`, cross-month `"Jan 20 → Feb 2, 2025"`, cross-year with both years
  - `FormatDateISO` → `"2025-01-20"` for JSON output

- **Missing values** (`values.go`):
  - `TableMissing = "-"` for table cells
  - `DetailMissing = "None"` for detail view metadata

## Deferred

- Glamour markdown renderer (needs `glamour` dependency; will add when first `show` command is implemented)
- Issue reference formatting (needs workspace context for disambiguation; will add in Phase 5 with identifier resolution)
- `--limit` and `--all` flag support (will add in Phase 4 alongside first list command)

## Tests

- 31 new tests across 7 test files
- 10 snapshot golden files for visual output verification
- Table-driven tests for date formatting, progress bars, color functions
- All 68 project tests pass, lint clean
