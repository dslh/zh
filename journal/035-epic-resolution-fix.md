# 035 — Epic list/resolution: include non-roadmap epics

## Problem

Epic listing and resolution relied solely on the `workspace.roadmap.items` query, which only returns epics that have been added to the ZenHub roadmap. Epics created via `zh epic create` (which calls `createZenhubEpic`) are not automatically added to the roadmap, so they were invisible to `zh epic list`, `zh epic show`, and all other commands that depend on epic resolution.

## Solution

Combined two API queries for comprehensive epic discovery:

1. **`workspace.zenhubEpics`** — Returns all standalone ZenHub epics in the workspace, regardless of roadmap membership. This catches newly created epics.
2. **`workspace.roadmap.items`** — Returns all roadmap items, which is the only way to discover legacy (issue-backed) epics.

Results are deduplicated by ID so epics appearing in both sources aren't listed twice.

## Changes

- **`internal/resolve/epic.go`**: Split `FetchEpics` into `fetchZenhubEpics` + `fetchRoadmapEpics`, combining results with deduplication. Renamed queries to `ListZenhubEpics` and `ListRoadmapEpics`.
- **`cmd/epic.go`**: Split `fetchEpicList` into `fetchZenhubEpicList` + `fetchRoadmapEpicList` with the same pattern. Added `parseZenhubEpicListItem` for the non-roadmap query format. Renamed queries to `ListZenhubEpicsFull` and `ListRoadmapEpicsFull`.
- **`cmd/epic_test.go`**: Replaced single-query mock helpers (`epicListResponse`, `epicResolutionResponse`) with dual-query helpers (`handleEpicListQueries`, `handleEpicResolutionQueries`).
- **`cmd/epic_mutations_test.go`**: Replaced `epicResolutionResponseForMutations` with `handleEpicResolutionForMutations`.
- **`cmd/epic_assignee_label_test.go`**, **`cmd/epic_key_date_test.go`**: Updated to use new mock helpers.
- **`internal/resolve/epic_test.go`**: Updated mock responses for the new dual-query pattern. Added `TestFetchEpicsDeduplicates` to verify dedup behavior.

## Verification

- All existing tests pass
- Lint clean
- Manual test: created an epic via `zh epic create`, confirmed it appears in `zh epic list` and resolves via `zh epic show` by substring — both previously failed with the roadmap-only approach
