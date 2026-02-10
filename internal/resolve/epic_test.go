package resolve

import (
	"testing"

	"github.com/dslh/zh/internal/api"
	"github.com/dslh/zh/internal/cache"
	"github.com/dslh/zh/internal/exitcode"
	"github.com/dslh/zh/internal/testutil"
)

func setupEpicCache(t *testing.T, workspaceID string, epics []CachedEpic) {
	t.Helper()
	cacheDir := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", cacheDir)
	_ = cache.Set(EpicCacheKey(workspaceID), epics)
}

func testEpics() []CachedEpic {
	return []CachedEpic{
		{ID: "e1", Title: "Q1 Platform Improvements", Type: "zenhub"},
		{ID: "e2", Title: "Authentication Overhaul", Type: "zenhub"},
		{ID: "e3", Title: "Mobile App Phase 2", Type: "legacy", IssueNumber: 42, RepoName: "mpt", RepoOwner: "gohiring"},
		{ID: "e4", Title: "API Rate Limiting", Type: "legacy", IssueNumber: 15, RepoName: "api", RepoOwner: "gohiring"},
	}
}

func TestEpicResolveByID(t *testing.T) {
	setupEpicCache(t, "ws1", testEpics())

	result, err := Epic(nil, "ws1", "e2", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "e2" || result.Title != "Authentication Overhaul" {
		t.Errorf("got %+v, want ID=e2 Title=Authentication Overhaul", result)
	}
	if result.Type != "zenhub" {
		t.Errorf("got Type=%s, want zenhub", result.Type)
	}
}

func TestEpicResolveByExactTitle(t *testing.T) {
	setupEpicCache(t, "ws1", testEpics())

	result, err := Epic(nil, "ws1", "Mobile App Phase 2", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "e3" {
		t.Errorf("got ID=%s, want e3", result.ID)
	}
	if result.Type != "legacy" {
		t.Errorf("got Type=%s, want legacy", result.Type)
	}
}

func TestEpicResolveByExactTitleCaseInsensitive(t *testing.T) {
	setupEpicCache(t, "ws1", testEpics())

	result, err := Epic(nil, "ws1", "mobile app phase 2", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "e3" {
		t.Errorf("got ID=%s, want e3", result.ID)
	}
}

func TestEpicResolveByUniqueSubstring(t *testing.T) {
	setupEpicCache(t, "ws1", testEpics())

	result, err := Epic(nil, "ws1", "Rate Limiting", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "e4" || result.Title != "API Rate Limiting" {
		t.Errorf("got %+v, want ID=e4 Title=API Rate Limiting", result)
	}
}

func TestEpicResolveByAlias(t *testing.T) {
	setupEpicCache(t, "ws1", testEpics())

	aliases := map[string]string{
		"auth": "Authentication Overhaul",
		"q1":   "e1",
	}

	// Alias to title
	result, err := Epic(nil, "ws1", "auth", aliases)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "e2" {
		t.Errorf("got ID=%s, want e2", result.ID)
	}

	// Alias to ID
	result, err = Epic(nil, "ws1", "q1", aliases)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "e1" {
		t.Errorf("got ID=%s, want e1", result.ID)
	}
}

func TestEpicResolveByRepoNumber(t *testing.T) {
	setupEpicCache(t, "ws1", testEpics())

	// Short form: repo#number
	result, err := Epic(nil, "ws1", "mpt#42", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "e3" {
		t.Errorf("got ID=%s, want e3", result.ID)
	}

	// Long form: owner/repo#number
	result, err = Epic(nil, "ws1", "gohiring/api#15", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "e4" {
		t.Errorf("got ID=%s, want e4", result.ID)
	}
}

func TestEpicResolveAmbiguous(t *testing.T) {
	epics := []CachedEpic{
		{ID: "e1", Title: "Platform Improvements Q1", Type: "zenhub"},
		{ID: "e2", Title: "Platform Improvements Q2", Type: "zenhub"},
	}
	setupEpicCache(t, "ws1", epics)

	_, err := Epic(nil, "ws1", "Platform", nil)
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
	if !containsStr(errMsg, "Platform Improvements Q1") || !containsStr(errMsg, "Platform Improvements Q2") {
		t.Errorf("error should list candidates, got: %s", errMsg)
	}
}

func TestEpicResolveNotFoundRefreshesCache(t *testing.T) {
	cacheDir := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", cacheDir)

	// Pre-populate cache without the target epic
	old := []CachedEpic{
		{ID: "e1", Title: "Old Epic", Type: "zenhub"},
	}
	_ = cache.Set(EpicCacheKey("ws1"), old)

	// Mock server returns updated epic list from both queries
	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListZenhubEpics", map[string]any{
		"data": map[string]any{
			"workspace": map[string]any{
				"zenhubEpics": map[string]any{
					"pageInfo": map[string]any{
						"hasNextPage": false,
						"endCursor":   "",
					},
					"nodes": []any{
						map[string]any{
							"id":    "e1",
							"title": "Old Epic",
						},
						map[string]any{
							"id":    "e2",
							"title": "New Epic",
						},
					},
				},
			},
		},
	})
	ms.HandleQuery("ListRoadmapEpics", map[string]any{
		"data": map[string]any{
			"workspace": map[string]any{
				"roadmap": map[string]any{
					"items": map[string]any{
						"pageInfo": map[string]any{
							"hasNextPage": false,
							"endCursor":   "",
						},
						"nodes": []any{},
					},
				},
			},
		},
	})

	client := api.New("test-key", api.WithEndpoint(ms.URL()))

	result, err := Epic(client, "ws1", "New Epic", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "e2" {
		t.Errorf("got ID=%s, want e2", result.ID)
	}

	// Verify cache was updated
	cached, ok := cache.Get[[]CachedEpic](EpicCacheKey("ws1"))
	if !ok {
		t.Fatal("cache should contain updated epics")
	}
	if len(cached) != 2 {
		t.Errorf("cache should have 2 entries, got %d", len(cached))
	}
}

func TestEpicResolveNotFoundAfterRefresh(t *testing.T) {
	cacheDir := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", cacheDir)

	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListZenhubEpics", map[string]any{
		"data": map[string]any{
			"workspace": map[string]any{
				"zenhubEpics": map[string]any{
					"pageInfo": map[string]any{
						"hasNextPage": false,
						"endCursor":   "",
					},
					"nodes": []any{
						map[string]any{
							"id":    "e1",
							"title": "Only Epic",
						},
					},
				},
			},
		},
	})
	ms.HandleQuery("ListRoadmapEpics", map[string]any{
		"data": map[string]any{
			"workspace": map[string]any{
				"roadmap": map[string]any{
					"items": map[string]any{
						"pageInfo": map[string]any{
							"hasNextPage": false,
							"endCursor":   "",
						},
						"nodes": []any{},
					},
				},
			},
		},
	})

	client := api.New("test-key", api.WithEndpoint(ms.URL()))

	_, err := Epic(client, "ws1", "nonexistent", nil)
	if err == nil {
		t.Fatal("expected not found error, got nil")
	}

	ec := exitcode.ExitCode(err)
	if ec != exitcode.NotFound {
		t.Errorf("exit code = %d, want %d (NotFound)", ec, exitcode.NotFound)
	}
}

func TestFetchEpics(t *testing.T) {
	cacheDir := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", cacheDir)

	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListZenhubEpics", map[string]any{
		"data": map[string]any{
			"workspace": map[string]any{
				"zenhubEpics": map[string]any{
					"pageInfo": map[string]any{
						"hasNextPage": false,
						"endCursor":   "",
					},
					"nodes": []any{
						map[string]any{
							"id":    "e1",
							"title": "ZenHub Epic",
						},
					},
				},
			},
		},
	})
	ms.HandleQuery("ListRoadmapEpics", map[string]any{
		"data": map[string]any{
			"workspace": map[string]any{
				"roadmap": map[string]any{
					"items": map[string]any{
						"pageInfo": map[string]any{
							"hasNextPage": false,
							"endCursor":   "",
						},
						"nodes": []any{
							map[string]any{
								"__typename": "ZenhubEpic",
								"id":         "e1",
								"title":      "ZenHub Epic",
							},
							map[string]any{
								"__typename": "Epic",
								"id":         "e2",
								"issue": map[string]any{
									"title":  "Legacy Epic",
									"number": 99,
									"repository": map[string]any{
										"name":      "mpt",
										"ownerName": "gohiring",
									},
								},
							},
							map[string]any{
								"__typename": "Project",
								"id":         "p1",
								"name":       "Some Project",
							},
						},
					},
				},
			},
		},
	})

	client := api.New("test-key", api.WithEndpoint(ms.URL()))

	epics, err := FetchEpics(client, "ws1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Projects should be filtered out, ZenHub epic deduplicated, only unique epics remain
	if len(epics) != 2 {
		t.Errorf("expected 2 epics, got %d", len(epics))
	}

	// Verify ZenHub epic (from zenhubEpics query)
	if epics[0].Type != "zenhub" || epics[0].Title != "ZenHub Epic" {
		t.Errorf("expected zenhub epic, got %+v", epics[0])
	}

	// Verify legacy epic (from roadmap query)
	if epics[1].Type != "legacy" || epics[1].Title != "Legacy Epic" || epics[1].IssueNumber != 99 {
		t.Errorf("expected legacy epic, got %+v", epics[1])
	}
	if epics[1].RepoName != "mpt" || epics[1].RepoOwner != "gohiring" {
		t.Errorf("expected repo info, got repoName=%s repoOwner=%s", epics[1].RepoName, epics[1].RepoOwner)
	}

	// Verify cache was populated
	cached, ok := cache.Get[[]CachedEpic](EpicCacheKey("ws1"))
	if !ok {
		t.Fatal("cache should be populated after fetch")
	}
	if len(cached) != 2 {
		t.Errorf("cache should have 2 entries, got %d", len(cached))
	}
}

func TestFetchEpicsDeduplicates(t *testing.T) {
	cacheDir := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", cacheDir)

	ms := testutil.NewMockServer(t)
	// Same ZenHub epic appears in both queries
	ms.HandleQuery("ListZenhubEpics", map[string]any{
		"data": map[string]any{
			"workspace": map[string]any{
				"zenhubEpics": map[string]any{
					"pageInfo": map[string]any{
						"hasNextPage": false,
						"endCursor":   "",
					},
					"nodes": []any{
						map[string]any{
							"id":    "e1",
							"title": "Shared Epic",
						},
						map[string]any{
							"id":    "e2",
							"title": "ZenHub Only Epic",
						},
					},
				},
			},
		},
	})
	ms.HandleQuery("ListRoadmapEpics", map[string]any{
		"data": map[string]any{
			"workspace": map[string]any{
				"roadmap": map[string]any{
					"items": map[string]any{
						"pageInfo": map[string]any{
							"hasNextPage": false,
							"endCursor":   "",
						},
						"nodes": []any{
							map[string]any{
								"__typename": "ZenhubEpic",
								"id":         "e1",
								"title":      "Shared Epic",
							},
						},
					},
				},
			},
		},
	})

	client := api.New("test-key", api.WithEndpoint(ms.URL()))

	epics, err := FetchEpics(client, "ws1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(epics) != 2 {
		t.Errorf("expected 2 epics (deduplicated), got %d", len(epics))
	}
}
