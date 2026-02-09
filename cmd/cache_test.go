package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dslh/zh/internal/cache"
)

func resetCacheFlags() {
	cacheClearWorkspace = false
}

func TestCacheClearAll(t *testing.T) {
	resetCacheFlags()
	cacheDir := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", cacheDir)

	// Create a cache file
	key := cache.NewScopedKey("pipelines", "ws1")
	if err := cache.Set(key, map[string]string{"test": "data"}); err != nil {
		t.Fatal(err)
	}

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"cache", "clear"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("cache clear returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Cleared all cached data") {
		t.Errorf("output = %q, want confirmation message", out)
	}

	// Verify cache file is gone
	if _, ok := cache.Get[map[string]string](key); ok {
		t.Error("cache file should be removed after clear")
	}
}

func TestCacheClearWorkspace(t *testing.T) {
	resetCacheFlags()
	cacheDir := t.TempDir()
	configDir := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", cacheDir)
	t.Setenv("XDG_CONFIG_HOME", configDir)
	t.Setenv("ZH_API_KEY", "test")
	t.Setenv("ZH_WORKSPACE", "target-ws")
	t.Setenv("ZH_GITHUB_TOKEN", "")

	// Create scoped and unscoped cache files
	targetKey := cache.NewScopedKey("pipelines", "target-ws")
	otherKey := cache.NewScopedKey("pipelines", "other-ws")
	if err := cache.Set(targetKey, "data1"); err != nil {
		t.Fatal(err)
	}
	if err := cache.Set(otherKey, "data2"); err != nil {
		t.Fatal(err)
	}

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"cache", "clear", "--workspace"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("cache clear --workspace returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Cleared cache for current workspace") {
		t.Errorf("output = %q, want workspace-specific confirmation", out)
	}

	// Target workspace cache should be gone
	if _, ok := cache.Get[string](targetKey); ok {
		t.Error("target workspace cache should be removed")
	}

	// Other workspace cache should remain
	if _, ok := cache.Get[string](otherKey); !ok {
		t.Error("other workspace cache should remain")
	}
}

func TestCacheClearWorkspaceNoConfig(t *testing.T) {
	resetCacheFlags()
	cacheDir := t.TempDir()
	configDir := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", cacheDir)
	t.Setenv("XDG_CONFIG_HOME", configDir)
	t.Setenv("ZH_API_KEY", "")
	t.Setenv("ZH_WORKSPACE", "")
	t.Setenv("ZH_GITHUB_TOKEN", "")

	rootCmd.SetOut(new(bytes.Buffer))
	rootCmd.SetArgs([]string{"cache", "clear", "--workspace"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("cache clear --workspace should error when no workspace configured")
	}
	if !strings.Contains(err.Error(), "no workspace configured") {
		t.Errorf("error = %q, want mention of no workspace", err.Error())
	}
}

func TestCacheHelpText(t *testing.T) {
	resetCacheFlags()
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"cache", "--help"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("cache --help returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "cache") {
		t.Error("help should mention cache")
	}
	if !strings.Contains(out, "clear") {
		t.Error("help should mention clear subcommand")
	}
}

func TestCacheClearEmptyDir(t *testing.T) {
	resetCacheFlags()
	// Point at a nonexistent cache dir — clear should succeed
	t.Setenv("XDG_CACHE_HOME", filepath.Join(t.TempDir(), "nonexistent"))

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"cache", "clear"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("cache clear with no cache dir should not error: %v", err)
	}
}

func TestCacheClearPreservesNonJsonFiles(t *testing.T) {
	resetCacheFlags()
	cacheDir := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", cacheDir)

	zhDir := filepath.Join(cacheDir, "zh")
	if err := os.MkdirAll(zhDir, 0o700); err != nil {
		t.Fatal(err)
	}

	// Create a non-JSON file that should be preserved
	nonJSON := filepath.Join(zhDir, "notes.txt")
	if err := os.WriteFile(nonJSON, []byte("keep me"), 0o600); err != nil {
		t.Fatal(err)
	}

	// Create a JSON cache file that should be removed
	if err := cache.Set(cache.NewKey("workspaces"), "data"); err != nil {
		t.Fatal(err)
	}

	rootCmd.SetOut(new(bytes.Buffer))
	rootCmd.SetArgs([]string{"cache", "clear"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("cache clear returned error: %v", err)
	}

	// Non-JSON file should still exist
	if _, err := os.Stat(nonJSON); os.IsNotExist(err) {
		t.Error("non-JSON file should be preserved")
	}

	// JSON cache file should be gone
	if _, ok := cache.Get[string](cache.NewKey("workspaces")); ok {
		t.Error("JSON cache file should be removed")
	}
}
