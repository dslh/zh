package resolve

import (
	"testing"

	"github.com/dslh/zh/internal/api"
	"github.com/dslh/zh/internal/cache"
	"github.com/dslh/zh/internal/exitcode"
	"github.com/dslh/zh/internal/testutil"
)

func setupZenhubLabelCache(t *testing.T, workspaceID string, labels []CachedZenhubLabel) {
	t.Helper()
	cacheDir := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", cacheDir)
	_ = cache.Set(ZenhubLabelCacheKey(workspaceID), labels)
}

func testZenhubLabels() []CachedZenhubLabel {
	return []CachedZenhubLabel{
		{ID: "zl1", Name: "platform", Color: "#0075ca"},
		{ID: "zl2", Name: "priority:high", Color: "#d73a4a"},
		{ID: "zl3", Name: "Backend", Color: "#008672"},
	}
}

func TestZenhubLabelResolveByID(t *testing.T) {
	setupZenhubLabelCache(t, "ws1", testZenhubLabels())

	result, err := ZenhubLabel(nil, "ws1", "zl2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "zl2" || result.Name != "priority:high" {
		t.Errorf("got %+v, want ID=zl2 Name=priority:high", result)
	}
}

func TestZenhubLabelResolveByName(t *testing.T) {
	setupZenhubLabelCache(t, "ws1", testZenhubLabels())

	result, err := ZenhubLabel(nil, "ws1", "platform")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "zl1" {
		t.Errorf("got ID=%s, want zl1", result.ID)
	}
}

func TestZenhubLabelResolveByNameCaseInsensitive(t *testing.T) {
	setupZenhubLabelCache(t, "ws1", testZenhubLabels())

	result, err := ZenhubLabel(nil, "ws1", "BACKEND")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "zl3" {
		t.Errorf("got ID=%s, want zl3", result.ID)
	}
}

func zenhubLabelsAPIResponse() map[string]any {
	return map[string]any{
		"data": map[string]any{
			"workspace": map[string]any{
				"zenhubLabels": map[string]any{
					"totalCount": 3,
					"pageInfo": map[string]any{
						"hasNextPage": false,
						"endCursor":   "",
					},
					"nodes": []any{
						map[string]any{"id": "zl1", "name": "platform", "color": "#0075ca"},
						map[string]any{"id": "zl2", "name": "priority:high", "color": "#d73a4a"},
						map[string]any{"id": "zl3", "name": "Backend", "color": "#008672"},
					},
				},
			},
		},
	}
}

func TestZenhubLabelNotFound(t *testing.T) {
	cacheDir := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", cacheDir)
	_ = cache.Set(ZenhubLabelCacheKey("ws1"), testZenhubLabels())

	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListZenhubLabels", zenhubLabelsAPIResponse())
	client := api.New("test-key", api.WithEndpoint(ms.URL()))

	_, err := ZenhubLabel(client, "ws1", "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent label")
	}
	if ec := exitcode.ExitCode(err); ec != exitcode.NotFound {
		t.Errorf("exit code = %d, want %d (NotFound)", ec, exitcode.NotFound)
	}
}

func TestZenhubLabelsResolveMultiple(t *testing.T) {
	setupZenhubLabelCache(t, "ws1", testZenhubLabels())

	results, err := ZenhubLabels(nil, "ws1", []string{"platform", "priority:high"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("got %d results, want 2", len(results))
	}
}

func TestZenhubLabelsResolveNotFound(t *testing.T) {
	cacheDir := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", cacheDir)
	_ = cache.Set(ZenhubLabelCacheKey("ws1"), testZenhubLabels())

	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListZenhubLabels", zenhubLabelsAPIResponse())
	client := api.New("test-key", api.WithEndpoint(ms.URL()))

	_, err := ZenhubLabels(client, "ws1", []string{"platform", "nonexistent"})
	if err == nil {
		t.Fatal("expected error for nonexistent label")
	}
	if ec := exitcode.ExitCode(err); ec != exitcode.NotFound {
		t.Errorf("exit code = %d, want %d (NotFound)", ec, exitcode.NotFound)
	}
}

func TestZenhubLabelResolveWithAPIRefresh(t *testing.T) {
	cacheDir := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", cacheDir)

	// Start with empty cache
	_ = cache.Set(ZenhubLabelCacheKey("ws1"), []CachedZenhubLabel{})

	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListZenhubLabels", map[string]any{
		"data": map[string]any{
			"workspace": map[string]any{
				"zenhubLabels": map[string]any{
					"totalCount": 1,
					"pageInfo": map[string]any{
						"hasNextPage": false,
						"endCursor":   "",
					},
					"nodes": []any{
						map[string]any{
							"id":    "zl-new",
							"name":  "new-label",
							"color": "#000000",
						},
					},
				},
			},
		},
	})

	client := api.New("test-key", api.WithEndpoint(ms.URL()))

	result, err := ZenhubLabel(client, "ws1", "new-label")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "zl-new" {
		t.Errorf("got ID=%s, want zl-new", result.ID)
	}
}
