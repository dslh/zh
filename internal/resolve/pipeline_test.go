package resolve

import (
	"testing"

	"github.com/dslh/zh/internal/api"
	"github.com/dslh/zh/internal/cache"
	"github.com/dslh/zh/internal/exitcode"
	"github.com/dslh/zh/internal/testutil"
)

func setupPipelineCache(t *testing.T, workspaceID string, pipelines []CachedPipeline) {
	t.Helper()
	cacheDir := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", cacheDir)
	_ = cache.Set(PipelineCacheKey(workspaceID), pipelines)
}

func testPipelines() []CachedPipeline {
	return []CachedPipeline{
		{ID: "p1", Name: "New Issues"},
		{ID: "p2", Name: "In Development"},
		{ID: "p3", Name: "Code Review"},
		{ID: "p4", Name: "Done"},
	}
}

func TestPipelineResolveByID(t *testing.T) {
	setupPipelineCache(t, "ws1", testPipelines())

	result, err := Pipeline(nil, "ws1", "p2", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "p2" || result.Name != "In Development" {
		t.Errorf("got %+v, want ID=p2 Name=In Development", result)
	}
}

func TestPipelineResolveByExactName(t *testing.T) {
	setupPipelineCache(t, "ws1", testPipelines())

	result, err := Pipeline(nil, "ws1", "Code Review", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "p3" {
		t.Errorf("got ID=%s, want p3", result.ID)
	}
}

func TestPipelineResolveByExactNameCaseInsensitive(t *testing.T) {
	setupPipelineCache(t, "ws1", testPipelines())

	result, err := Pipeline(nil, "ws1", "code review", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "p3" {
		t.Errorf("got ID=%s, want p3", result.ID)
	}
}

func TestPipelineResolveByUniqueSubstring(t *testing.T) {
	setupPipelineCache(t, "ws1", testPipelines())

	result, err := Pipeline(nil, "ws1", "Review", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "p3" || result.Name != "Code Review" {
		t.Errorf("got %+v, want ID=p3 Name=Code Review", result)
	}
}

func TestPipelineResolveByAlias(t *testing.T) {
	setupPipelineCache(t, "ws1", testPipelines())

	aliases := map[string]string{
		"ip": "In Development",
		"cr": "Code Review",
	}

	result, err := Pipeline(nil, "ws1", "ip", aliases)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "p2" {
		t.Errorf("got ID=%s, want p2", result.ID)
	}
}

func TestPipelineResolveAmbiguous(t *testing.T) {
	pipelines := []CachedPipeline{
		{ID: "p1", Name: "In Development"},
		{ID: "p2", Name: "In Review"},
	}
	setupPipelineCache(t, "ws1", pipelines)

	_, err := Pipeline(nil, "ws1", "In", nil)
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
	if !containsStr(errMsg, "In Development") || !containsStr(errMsg, "In Review") {
		t.Errorf("error should list candidates, got: %s", errMsg)
	}
}

func TestPipelineResolveNotFoundRefreshesCache(t *testing.T) {
	cacheDir := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", cacheDir)

	// Pre-populate cache without the target pipeline
	old := []CachedPipeline{
		{ID: "p1", Name: "Old Pipeline"},
	}
	_ = cache.Set(PipelineCacheKey("ws1"), old)

	// Mock server returns updated pipeline list
	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListPipelines", map[string]any{
		"data": map[string]any{
			"workspace": map[string]any{
				"pipelinesConnection": map[string]any{
					"nodes": []any{
						map[string]any{"id": "p1", "name": "Old Pipeline"},
						map[string]any{"id": "p2", "name": "New Pipeline"},
					},
				},
			},
		},
	})

	client := api.New("test-key", api.WithEndpoint(ms.URL()))

	result, err := Pipeline(client, "ws1", "New Pipeline", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "p2" {
		t.Errorf("got ID=%s, want p2", result.ID)
	}

	// Verify cache was updated
	cached, ok := cache.Get[[]CachedPipeline](PipelineCacheKey("ws1"))
	if !ok {
		t.Fatal("cache should contain updated pipelines")
	}
	if len(cached) != 2 {
		t.Errorf("cache should have 2 entries, got %d", len(cached))
	}
}

func TestPipelineResolveNotFoundAfterRefresh(t *testing.T) {
	cacheDir := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", cacheDir)

	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListPipelines", map[string]any{
		"data": map[string]any{
			"workspace": map[string]any{
				"pipelinesConnection": map[string]any{
					"nodes": []any{
						map[string]any{"id": "p1", "name": "Todo"},
					},
				},
			},
		},
	})

	client := api.New("test-key", api.WithEndpoint(ms.URL()))

	_, err := Pipeline(client, "ws1", "nonexistent", nil)
	if err == nil {
		t.Fatal("expected not found error, got nil")
	}

	ec := exitcode.ExitCode(err)
	if ec != exitcode.NotFound {
		t.Errorf("exit code = %d, want %d (NotFound)", ec, exitcode.NotFound)
	}
}

func TestFetchPipelines(t *testing.T) {
	cacheDir := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", cacheDir)

	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListPipelines", map[string]any{
		"data": map[string]any{
			"workspace": map[string]any{
				"pipelinesConnection": map[string]any{
					"nodes": []any{
						map[string]any{"id": "p1", "name": "Backlog"},
						map[string]any{"id": "p2", "name": "In Progress"},
						map[string]any{"id": "p3", "name": "Done"},
					},
				},
			},
		},
	})

	client := api.New("test-key", api.WithEndpoint(ms.URL()))

	pipelines, err := FetchPipelines(client, "ws1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pipelines) != 3 {
		t.Errorf("expected 3 pipelines, got %d", len(pipelines))
	}

	// Verify cache was populated
	cached, ok := cache.Get[[]CachedPipeline](PipelineCacheKey("ws1"))
	if !ok {
		t.Fatal("cache should be populated after fetch")
	}
	if len(cached) != 3 {
		t.Errorf("cache should have 3 entries, got %d", len(cached))
	}
}

func containsStr(s, sub string) bool {
	return len(s) >= len(sub) && searchStr(s, sub)
}

func searchStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
