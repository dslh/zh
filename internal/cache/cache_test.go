package cache

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestDir(t *testing.T) {
	t.Run("uses XDG_CACHE_HOME when set", func(t *testing.T) {
		t.Setenv("XDG_CACHE_HOME", "/tmp/test-cache")
		got := Dir()
		want := "/tmp/test-cache/zh"
		if got != want {
			t.Errorf("Dir() = %q, want %q", got, want)
		}
	})

	t.Run("falls back to ~/.cache/zh", func(t *testing.T) {
		t.Setenv("XDG_CACHE_HOME", "")
		got := Dir()
		home, _ := os.UserHomeDir()
		want := filepath.Join(home, ".cache", "zh")
		if got != want {
			t.Errorf("Dir() = %q, want %q", got, want)
		}
	})
}

func TestKeyFilename(t *testing.T) {
	tests := []struct {
		key  Key
		want string
	}{
		{NewKey("workspaces"), "workspaces.json"},
		{NewScopedKey("pipelines", "ws123"), "pipelines-ws123.json"},
		{NewScopedKey("repos", "69866ab95c14bf002977146b"), "repos-69866ab95c14bf002977146b.json"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.key.Filename(); got != tt.want {
				t.Errorf("Filename() = %q, want %q", got, tt.want)
			}
		})
	}
}

type testItem struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func setupCacheDir(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", dir)
}

func TestGetSetRoundTrip(t *testing.T) {
	setupCacheDir(t)
	key := NewScopedKey("pipelines", "ws123")

	items := []testItem{
		{ID: "1", Name: "Backlog"},
		{ID: "2", Name: "In Progress"},
	}

	if err := Set(key, items); err != nil {
		t.Fatalf("Set() error: %v", err)
	}

	got, ok := Get[[]testItem](key)
	if !ok {
		t.Fatal("Get() returned false, want true")
	}
	if len(got) != 2 {
		t.Fatalf("Get() returned %d items, want 2", len(got))
	}
	if got[0].Name != "Backlog" {
		t.Errorf("got[0].Name = %q, want %q", got[0].Name, "Backlog")
	}
	if got[1].Name != "In Progress" {
		t.Errorf("got[1].Name = %q, want %q", got[1].Name, "In Progress")
	}
}

func TestGetMissReturnsNotOK(t *testing.T) {
	setupCacheDir(t)
	key := NewScopedKey("pipelines", "nonexistent")

	_, ok := Get[[]testItem](key)
	if ok {
		t.Error("Get() returned true for nonexistent cache, want false")
	}
}

func TestGetCorruptedCacheReturnsNotOK(t *testing.T) {
	setupCacheDir(t)
	key := NewKey("corrupted")

	// Write invalid JSON
	dir := Dir()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(key.path(), []byte("not json"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, ok := Get[[]testItem](key)
	if ok {
		t.Error("Get() returned true for corrupted cache, want false")
	}
}

func TestClearRemovesFile(t *testing.T) {
	setupCacheDir(t)
	key := NewScopedKey("pipelines", "ws123")

	if err := Set(key, []testItem{{ID: "1", Name: "test"}}); err != nil {
		t.Fatal(err)
	}

	// Verify file exists
	if _, ok := Get[[]testItem](key); !ok {
		t.Fatal("cache should exist before clear")
	}

	if err := Clear(key); err != nil {
		t.Fatalf("Clear() error: %v", err)
	}

	if _, ok := Get[[]testItem](key); ok {
		t.Error("Get() returned true after Clear(), want false")
	}
}

func TestClearNonexistentIsNoOp(t *testing.T) {
	setupCacheDir(t)
	key := NewKey("nonexistent")

	if err := Clear(key); err != nil {
		t.Fatalf("Clear() on nonexistent cache should not error, got: %v", err)
	}
}

func TestClearAll(t *testing.T) {
	setupCacheDir(t)

	// Create multiple cache files
	keys := []Key{
		NewKey("workspaces"),
		NewScopedKey("pipelines", "ws1"),
		NewScopedKey("repos", "ws2"),
	}
	for _, k := range keys {
		if err := Set(k, testItem{ID: "1"}); err != nil {
			t.Fatal(err)
		}
	}

	if err := ClearAll(); err != nil {
		t.Fatalf("ClearAll() error: %v", err)
	}

	for _, k := range keys {
		if _, ok := Get[testItem](k); ok {
			t.Errorf("cache %q should be cleared", k.Filename())
		}
	}
}

func TestClearAllWhenNoCacheDir(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", filepath.Join(t.TempDir(), "nonexistent"))
	if err := ClearAll(); err != nil {
		t.Fatalf("ClearAll() with no cache dir should not error, got: %v", err)
	}
}

func TestClearWorkspace(t *testing.T) {
	setupCacheDir(t)

	// Create workspace-scoped and unscoped cache files
	wsKey1 := NewScopedKey("pipelines", "target-ws")
	wsKey2 := NewScopedKey("repos", "target-ws")
	otherKey := NewScopedKey("pipelines", "other-ws")
	globalKey := NewKey("workspaces")

	for _, k := range []Key{wsKey1, wsKey2, otherKey, globalKey} {
		if err := Set(k, testItem{ID: "1"}); err != nil {
			t.Fatal(err)
		}
	}

	if err := ClearWorkspace("target-ws"); err != nil {
		t.Fatalf("ClearWorkspace() error: %v", err)
	}

	// Scoped files for target workspace should be gone
	if _, ok := Get[testItem](wsKey1); ok {
		t.Error("pipelines for target-ws should be cleared")
	}
	if _, ok := Get[testItem](wsKey2); ok {
		t.Error("repos for target-ws should be cleared")
	}

	// Other workspace and global files should remain
	if _, ok := Get[testItem](otherKey); !ok {
		t.Error("pipelines for other-ws should NOT be cleared")
	}
	if _, ok := Get[testItem](globalKey); !ok {
		t.Error("workspaces (global) should NOT be cleared")
	}
}

func TestGetOrRefreshCacheHit(t *testing.T) {
	setupCacheDir(t)
	key := NewScopedKey("pipelines", "ws1")

	items := []testItem{
		{ID: "1", Name: "Backlog"},
		{ID: "2", Name: "In Progress"},
	}
	if err := Set(key, items); err != nil {
		t.Fatal(err)
	}

	refreshCalled := false
	result, err := GetOrRefresh(
		key,
		func() ([]testItem, error) {
			refreshCalled = true
			return nil, fmt.Errorf("should not be called")
		},
		func(cached []testItem) (testItem, bool) {
			for _, item := range cached {
				if item.Name == "Backlog" {
					return item, true
				}
			}
			return testItem{}, false
		},
	)

	if err != nil {
		t.Fatalf("GetOrRefresh() error: %v", err)
	}
	if refreshCalled {
		t.Error("refresh should not be called on cache hit")
	}
	if result.ID != "1" {
		t.Errorf("result.ID = %q, want %q", result.ID, "1")
	}
}

func TestGetOrRefreshCacheMissThenRefresh(t *testing.T) {
	setupCacheDir(t)
	key := NewScopedKey("pipelines", "ws2")

	refreshCalled := false
	result, err := GetOrRefresh(
		key,
		func() ([]testItem, error) {
			refreshCalled = true
			return []testItem{
				{ID: "10", Name: "New Pipeline"},
			}, nil
		},
		func(cached []testItem) (testItem, bool) {
			for _, item := range cached {
				if item.Name == "New Pipeline" {
					return item, true
				}
			}
			return testItem{}, false
		},
	)

	if err != nil {
		t.Fatalf("GetOrRefresh() error: %v", err)
	}
	if !refreshCalled {
		t.Error("refresh should be called on cache miss")
	}
	if result.ID != "10" {
		t.Errorf("result.ID = %q, want %q", result.ID, "10")
	}

	// Verify cache was populated
	got, ok := Get[[]testItem](key)
	if !ok {
		t.Fatal("cache should be populated after refresh")
	}
	if len(got) != 1 || got[0].Name != "New Pipeline" {
		t.Errorf("cached data = %+v, want [{ID:10 Name:New Pipeline}]", got)
	}
}

func TestGetOrRefreshLookupMissThenRefresh(t *testing.T) {
	setupCacheDir(t)
	key := NewScopedKey("pipelines", "ws3")

	// Pre-populate with stale data (missing the item we want)
	stale := []testItem{{ID: "1", Name: "Old"}}
	if err := Set(key, stale); err != nil {
		t.Fatal(err)
	}

	refreshCalled := false
	result, err := GetOrRefresh(
		key,
		func() ([]testItem, error) {
			refreshCalled = true
			return []testItem{
				{ID: "1", Name: "Old"},
				{ID: "2", Name: "Renamed"},
			}, nil
		},
		func(cached []testItem) (testItem, bool) {
			for _, item := range cached {
				if item.Name == "Renamed" {
					return item, true
				}
			}
			return testItem{}, false
		},
	)

	if err != nil {
		t.Fatalf("GetOrRefresh() error: %v", err)
	}
	if !refreshCalled {
		t.Error("refresh should be called when lookup misses in stale cache")
	}
	if result.ID != "2" {
		t.Errorf("result.ID = %q, want %q", result.ID, "2")
	}
}

func TestGetOrRefreshRefreshError(t *testing.T) {
	setupCacheDir(t)
	key := NewScopedKey("pipelines", "ws4")

	_, err := GetOrRefresh(
		key,
		func() ([]testItem, error) {
			return nil, fmt.Errorf("API is down")
		},
		func(cached []testItem) (testItem, bool) {
			return testItem{}, false
		},
	)

	if err == nil {
		t.Fatal("GetOrRefresh() should return error when refresh fails")
	}
	if err.Error() != "API is down" {
		t.Errorf("error = %q, want %q", err.Error(), "API is down")
	}
}

func TestGetOrRefreshNotFoundAfterRefresh(t *testing.T) {
	setupCacheDir(t)
	key := NewScopedKey("pipelines", "ws5")

	result, err := GetOrRefresh(
		key,
		func() ([]testItem, error) {
			return []testItem{{ID: "1", Name: "Only"}}, nil
		},
		func(cached []testItem) (testItem, bool) {
			for _, item := range cached {
				if item.Name == "Nonexistent" {
					return item, true
				}
			}
			return testItem{}, false
		},
	)

	if err != nil {
		t.Fatalf("GetOrRefresh() error: %v", err)
	}
	if result.ID != "" {
		t.Errorf("expected zero value when not found after refresh, got %+v", result)
	}
}
