// Package cache provides file-backed JSON caching with XDG-compliant paths.
//
// Cache files live at $XDG_CACHE_HOME/zh/ (default ~/.cache/zh/).
// Each resource type is stored in a separate JSON file, optionally scoped
// by workspace ID (e.g. "pipelines-{workspace_id}.json").
//
// Invalidation follows the invalidate-on-miss pattern: when a lookup fails,
// the caller refreshes the cache from the API and retries.
package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Dir returns the XDG-compliant cache directory for zh.
func Dir() string {
	if xdg := os.Getenv("XDG_CACHE_HOME"); xdg != "" {
		return filepath.Join(xdg, "zh")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".cache", "zh")
}

// Key identifies a cache file. Use NewKey for unscoped resources or
// NewScopedKey for workspace-scoped resources.
type Key struct {
	Resource    string // e.g. "workspaces", "pipelines"
	WorkspaceID string // e.g. "69866ab95c14bf002977146b" (empty for unscoped)
}

// NewKey returns a cache key for an unscoped resource (e.g. "workspaces").
func NewKey(resource string) Key {
	return Key{Resource: resource}
}

// NewScopedKey returns a cache key scoped to a workspace (e.g. "pipelines", "ws123").
func NewScopedKey(resource, workspaceID string) Key {
	return Key{Resource: resource, WorkspaceID: workspaceID}
}

// Filename returns the cache filename for this key.
// Unscoped: "{resource}.json", scoped: "{resource}-{workspace_id}.json".
func (k Key) Filename() string {
	if k.WorkspaceID != "" {
		return fmt.Sprintf("%s-%s.json", k.Resource, k.WorkspaceID)
	}
	return k.Resource + ".json"
}

// path returns the full filesystem path for this cache key.
func (k Key) path() string {
	return filepath.Join(Dir(), k.Filename())
}

// Get reads a cached value from disk. Returns the value and true if found,
// or the zero value and false if the cache file doesn't exist.
func Get[T any](key Key) (T, bool) {
	var zero T

	data, err := os.ReadFile(key.path())
	if err != nil {
		return zero, false
	}

	var value T
	if err := json.Unmarshal(data, &value); err != nil {
		return zero, false
	}

	return value, true
}

// Set writes a value to the cache as JSON.
func Set[T any](key Key, value T) error {
	dir := Dir()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("creating cache directory: %w", err)
	}

	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling cache data: %w", err)
	}

	return os.WriteFile(key.path(), data, 0o600)
}

// Clear removes the cache file for the given key.
// Returns nil if the file doesn't exist.
func Clear(key Key) error {
	err := os.Remove(key.path())
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// ClearAll removes all cache files.
func ClearAll() error {
	dir := Dir()
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("reading cache directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") {
			if err := os.Remove(filepath.Join(dir, entry.Name())); err != nil {
				return fmt.Errorf("removing %s: %w", entry.Name(), err)
			}
		}
	}
	return nil
}

// ClearWorkspace removes all cache files scoped to the given workspace ID.
func ClearWorkspace(workspaceID string) error {
	dir := Dir()
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("reading cache directory: %w", err)
	}

	suffix := "-" + workspaceID + ".json"
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), suffix) {
			if err := os.Remove(filepath.Join(dir, entry.Name())); err != nil {
				return fmt.Errorf("removing %s: %w", entry.Name(), err)
			}
		}
	}
	return nil
}

// GetOrRefresh implements the invalidate-on-miss pattern. It first checks the
// cache. If the value is present, lookup is called. If lookup returns true,
// the result is returned. If lookup returns false (miss), refresh is called to
// repopulate the cache from the API, and lookup is retried.
//
// This handles renamed entities gracefully: the old name misses, triggering a
// refresh that pulls in the new name.
func GetOrRefresh[T any, R any](
	key Key,
	refresh func() (T, error),
	lookup func(T) (R, bool),
) (R, error) {
	var zero R

	// Try cache first
	if cached, ok := Get[T](key); ok {
		if result, found := lookup(cached); found {
			return result, nil
		}
	}

	// Cache miss or lookup miss — refresh from API
	fresh, err := refresh()
	if err != nil {
		return zero, err
	}

	// Persist refreshed data
	if err := Set(key, fresh); err != nil {
		return zero, fmt.Errorf("updating cache: %w", err)
	}

	// Retry lookup
	if result, found := lookup(fresh); found {
		return result, nil
	}

	return zero, nil
}
