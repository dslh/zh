package resolve

import (
	"testing"

	"github.com/dslh/zh/internal/api"
	"github.com/dslh/zh/internal/cache"
	"github.com/dslh/zh/internal/exitcode"
	"github.com/dslh/zh/internal/testutil"
)

func setupSprintCache(t *testing.T, workspaceID string, sprints []CachedSprint) {
	t.Helper()
	cacheDir := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", cacheDir)
	_ = cache.Set(SprintCacheKey(workspaceID), sprints)
}

func testSprints() []CachedSprint {
	return []CachedSprint{
		{ID: "s1", Name: "", GeneratedName: "Sprint 45", State: "CLOSED", StartAt: "2025-01-06T00:00:00Z", EndAt: "2025-01-19T00:00:00Z"},
		{ID: "s2", Name: "", GeneratedName: "Sprint 46", State: "CLOSED", StartAt: "2025-01-20T00:00:00Z", EndAt: "2025-02-02T00:00:00Z"},
		{ID: "s3", Name: "Q1 Final", GeneratedName: "Sprint 47", State: "OPEN", StartAt: "2025-02-03T00:00:00Z", EndAt: "2025-02-16T00:00:00Z"},
		{ID: "s4", Name: "", GeneratedName: "Sprint 48", State: "OPEN", StartAt: "2025-02-17T00:00:00Z", EndAt: "2025-03-02T00:00:00Z"},
	}
}

func TestSprintDisplayName(t *testing.T) {
	s1 := CachedSprint{Name: "", GeneratedName: "Sprint 47"}
	if s1.DisplayName() != "Sprint 47" {
		t.Errorf("got %s, want Sprint 47", s1.DisplayName())
	}

	s2 := CachedSprint{Name: "Q1 Final", GeneratedName: "Sprint 47"}
	if s2.DisplayName() != "Q1 Final" {
		t.Errorf("got %s, want Q1 Final", s2.DisplayName())
	}
}

func TestSprintResolveByID(t *testing.T) {
	setupSprintCache(t, "ws1", testSprints())

	result, err := Sprint(nil, "ws1", "s3")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "s3" || result.Name != "Q1 Final" {
		t.Errorf("got %+v, want ID=s3 Name=Q1 Final", result)
	}
}

func TestSprintResolveByExactName(t *testing.T) {
	setupSprintCache(t, "ws1", testSprints())

	result, err := Sprint(nil, "ws1", "Sprint 46")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "s2" {
		t.Errorf("got ID=%s, want s2", result.ID)
	}
}

func TestSprintResolveByCustomName(t *testing.T) {
	setupSprintCache(t, "ws1", testSprints())

	result, err := Sprint(nil, "ws1", "Q1 Final")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "s3" {
		t.Errorf("got ID=%s, want s3", result.ID)
	}
}

func TestSprintResolveByExactNameCaseInsensitive(t *testing.T) {
	setupSprintCache(t, "ws1", testSprints())

	result, err := Sprint(nil, "ws1", "sprint 46")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "s2" {
		t.Errorf("got ID=%s, want s2", result.ID)
	}
}

func TestSprintResolveByGeneratedNameWhenCustomSet(t *testing.T) {
	setupSprintCache(t, "ws1", testSprints())

	// Sprint 47 has custom name "Q1 Final", but should still match by generated name
	result, err := Sprint(nil, "ws1", "Sprint 47")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "s3" {
		t.Errorf("got ID=%s, want s3", result.ID)
	}
}

func TestSprintResolveByUniqueSubstring(t *testing.T) {
	setupSprintCache(t, "ws1", testSprints())

	result, err := Sprint(nil, "ws1", "Final")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "s3" {
		t.Errorf("got ID=%s, want s3", result.ID)
	}
}

func TestSprintResolveAmbiguous(t *testing.T) {
	sprints := []CachedSprint{
		{ID: "s1", GeneratedName: "Sprint 45", State: "CLOSED"},
		{ID: "s2", GeneratedName: "Sprint 46", State: "CLOSED"},
	}
	setupSprintCache(t, "ws1", sprints)

	_, err := Sprint(nil, "ws1", "Sprint")
	if err == nil {
		t.Fatal("expected ambiguous error, got nil")
	}

	ec := exitcode.ExitCode(err)
	if ec != exitcode.UsageError {
		t.Errorf("exit code = %d, want %d (UsageError)", ec, exitcode.UsageError)
	}

	errMsg := err.Error()
	if !containsStr(errMsg, "ambiguous") {
		t.Errorf("error should mention 'ambiguous', got: %s", errMsg)
	}
	if !containsStr(errMsg, "Sprint 45") || !containsStr(errMsg, "Sprint 46") {
		t.Errorf("error should list candidates, got: %s", errMsg)
	}
}

func TestSprintResolveRelativeCurrent(t *testing.T) {
	cacheDir := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", cacheDir)

	sprints := testSprints()
	accessors := sprintAccessors{
		ActiveID:   "s3",
		UpcomingID: "s4",
		PreviousID: "s2",
	}
	_ = cache.Set(SprintCacheKey("ws1"), sprints)
	_ = cache.Set(SprintAccessorsCacheKey("ws1"), accessors)

	result, err := Sprint(nil, "ws1", "current")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "s3" {
		t.Errorf("got ID=%s, want s3", result.ID)
	}
}

func TestSprintResolveRelativeNext(t *testing.T) {
	cacheDir := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", cacheDir)

	sprints := testSprints()
	accessors := sprintAccessors{
		ActiveID:   "s3",
		UpcomingID: "s4",
		PreviousID: "s2",
	}
	_ = cache.Set(SprintCacheKey("ws1"), sprints)
	_ = cache.Set(SprintAccessorsCacheKey("ws1"), accessors)

	result, err := Sprint(nil, "ws1", "next")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "s4" {
		t.Errorf("got ID=%s, want s4", result.ID)
	}
}

func TestSprintResolveRelativePrevious(t *testing.T) {
	cacheDir := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", cacheDir)

	sprints := testSprints()
	accessors := sprintAccessors{
		ActiveID:   "s3",
		UpcomingID: "s4",
		PreviousID: "s2",
	}
	_ = cache.Set(SprintCacheKey("ws1"), sprints)
	_ = cache.Set(SprintAccessorsCacheKey("ws1"), accessors)

	result, err := Sprint(nil, "ws1", "previous")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "s2" {
		t.Errorf("got ID=%s, want s2", result.ID)
	}
}

func TestSprintResolveNoActiveSprint(t *testing.T) {
	cacheDir := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", cacheDir)

	sprints := []CachedSprint{
		{ID: "s1", GeneratedName: "Sprint 1", State: "CLOSED", StartAt: "2024-01-01T00:00:00Z", EndAt: "2024-01-14T00:00:00Z"},
	}
	accessors := sprintAccessors{}
	_ = cache.Set(SprintCacheKey("ws1"), sprints)
	_ = cache.Set(SprintAccessorsCacheKey("ws1"), accessors)

	_, err := Sprint(nil, "ws1", "current")
	if err == nil {
		t.Fatal("expected error for no active sprint, got nil")
	}

	ec := exitcode.ExitCode(err)
	if ec != exitcode.NotFound {
		t.Errorf("exit code = %d, want %d (NotFound)", ec, exitcode.NotFound)
	}
}

func TestSprintResolveNotFoundRefreshesCache(t *testing.T) {
	cacheDir := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", cacheDir)

	// Pre-populate cache without the target sprint
	old := []CachedSprint{
		{ID: "s1", GeneratedName: "Sprint 1", State: "CLOSED"},
	}
	_ = cache.Set(SprintCacheKey("ws1"), old)

	// Mock server returns updated sprint list
	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListSprints", map[string]any{
		"data": map[string]any{
			"workspace": map[string]any{
				"sprints": map[string]any{
					"totalCount": 2,
					"pageInfo": map[string]any{
						"hasNextPage": false,
						"endCursor":   "",
					},
					"nodes": []any{
						map[string]any{
							"id":            "s1",
							"name":          "",
							"generatedName": "Sprint 1",
							"state":         "CLOSED",
							"startAt":       "2025-01-06T00:00:00Z",
							"endAt":         "2025-01-19T00:00:00Z",
						},
						map[string]any{
							"id":            "s2",
							"name":          "",
							"generatedName": "Sprint 2",
							"state":         "OPEN",
							"startAt":       "2025-01-20T00:00:00Z",
							"endAt":         "2025-02-02T00:00:00Z",
						},
					},
				},
				"activeSprint":   map[string]any{"id": "s2"},
				"upcomingSprint": nil,
				"previousSprint": map[string]any{"id": "s1"},
			},
		},
	})

	client := api.New("test-key", api.WithEndpoint(ms.URL()))

	result, err := Sprint(client, "ws1", "Sprint 2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "s2" {
		t.Errorf("got ID=%s, want s2", result.ID)
	}

	// Verify cache was updated
	cached, ok := cache.Get[[]CachedSprint](SprintCacheKey("ws1"))
	if !ok {
		t.Fatal("cache should contain updated sprints")
	}
	if len(cached) != 2 {
		t.Errorf("cache should have 2 entries, got %d", len(cached))
	}
}

func TestSprintResolveNotFoundAfterRefresh(t *testing.T) {
	cacheDir := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", cacheDir)

	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListSprints", map[string]any{
		"data": map[string]any{
			"workspace": map[string]any{
				"sprints": map[string]any{
					"totalCount": 1,
					"pageInfo": map[string]any{
						"hasNextPage": false,
						"endCursor":   "",
					},
					"nodes": []any{
						map[string]any{
							"id":            "s1",
							"name":          "",
							"generatedName": "Sprint 1",
							"state":         "CLOSED",
							"startAt":       "2025-01-06T00:00:00Z",
							"endAt":         "2025-01-19T00:00:00Z",
						},
					},
				},
				"activeSprint":   nil,
				"upcomingSprint": nil,
				"previousSprint": nil,
			},
		},
	})

	client := api.New("test-key", api.WithEndpoint(ms.URL()))

	_, err := Sprint(client, "ws1", "nonexistent")
	if err == nil {
		t.Fatal("expected not found error, got nil")
	}

	ec := exitcode.ExitCode(err)
	if ec != exitcode.NotFound {
		t.Errorf("exit code = %d, want %d (NotFound)", ec, exitcode.NotFound)
	}
}

func TestFetchSprints(t *testing.T) {
	cacheDir := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", cacheDir)

	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListSprints", map[string]any{
		"data": map[string]any{
			"workspace": map[string]any{
				"sprints": map[string]any{
					"totalCount": 3,
					"pageInfo": map[string]any{
						"hasNextPage": false,
						"endCursor":   "",
					},
					"nodes": []any{
						map[string]any{
							"id":            "s1",
							"name":          "",
							"generatedName": "Sprint 1",
							"state":         "CLOSED",
							"startAt":       "2025-01-06T00:00:00Z",
							"endAt":         "2025-01-19T00:00:00Z",
						},
						map[string]any{
							"id":            "s2",
							"name":          "Mid Q1",
							"generatedName": "Sprint 2",
							"state":         "OPEN",
							"startAt":       "2025-01-20T00:00:00Z",
							"endAt":         "2025-02-02T00:00:00Z",
						},
						map[string]any{
							"id":            "s3",
							"name":          "",
							"generatedName": "Sprint 3",
							"state":         "OPEN",
							"startAt":       "2025-02-03T00:00:00Z",
							"endAt":         "2025-02-16T00:00:00Z",
						},
					},
				},
				"activeSprint":   map[string]any{"id": "s2"},
				"upcomingSprint": map[string]any{"id": "s3"},
				"previousSprint": map[string]any{"id": "s1"},
			},
		},
	})

	client := api.New("test-key", api.WithEndpoint(ms.URL()))

	sprints, err := FetchSprints(client, "ws1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sprints) != 3 {
		t.Errorf("expected 3 sprints, got %d", len(sprints))
	}

	// Verify cache was populated
	cached, ok := cache.Get[[]CachedSprint](SprintCacheKey("ws1"))
	if !ok {
		t.Fatal("cache should be populated after fetch")
	}
	if len(cached) != 3 {
		t.Errorf("cache should have 3 entries, got %d", len(cached))
	}

	// Verify sprint accessors were cached
	accessors, ok := cache.Get[sprintAccessors](SprintAccessorsCacheKey("ws1"))
	if !ok {
		t.Fatal("sprint accessors cache should be populated after fetch")
	}
	if accessors.ActiveID != "s2" {
		t.Errorf("active sprint ID = %s, want s2", accessors.ActiveID)
	}
	if accessors.UpcomingID != "s3" {
		t.Errorf("upcoming sprint ID = %s, want s3", accessors.UpcomingID)
	}
	if accessors.PreviousID != "s1" {
		t.Errorf("previous sprint ID = %s, want s1", accessors.PreviousID)
	}
}

func TestSprintRelativeRefreshesWhenNoCacheExists(t *testing.T) {
	cacheDir := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", cacheDir)

	// No cache pre-populated — should fetch from API
	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListSprints", map[string]any{
		"data": map[string]any{
			"workspace": map[string]any{
				"sprints": map[string]any{
					"totalCount": 1,
					"pageInfo": map[string]any{
						"hasNextPage": false,
						"endCursor":   "",
					},
					"nodes": []any{
						map[string]any{
							"id":            "s1",
							"name":          "",
							"generatedName": "Sprint 1",
							"state":         "OPEN",
							"startAt":       "2025-01-20T00:00:00Z",
							"endAt":         "2025-02-02T00:00:00Z",
						},
					},
				},
				"activeSprint":   map[string]any{"id": "s1"},
				"upcomingSprint": nil,
				"previousSprint": nil,
			},
		},
	})

	client := api.New("test-key", api.WithEndpoint(ms.URL()))

	result, err := Sprint(client, "ws1", "current")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "s1" {
		t.Errorf("got ID=%s, want s1", result.ID)
	}
}
