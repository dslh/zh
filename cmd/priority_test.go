package cmd

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/dslh/zh/internal/api"
	"github.com/dslh/zh/internal/cache"
	"github.com/dslh/zh/internal/resolve"
	"github.com/dslh/zh/internal/testutil"
)

func setupPriorityTest(t *testing.T, ms *testutil.MockServer) {
	t.Helper()

	cacheDir := t.TempDir()
	configDir := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", cacheDir)
	t.Setenv("XDG_CONFIG_HOME", configDir)
	t.Setenv("ZH_API_KEY", "test-key")
	t.Setenv("ZH_WORKSPACE", "ws-123")
	t.Setenv("ZH_GITHUB_TOKEN", "")

	origNew := apiNewFunc
	apiNewFunc = func(apiKey string, opts ...api.Option) *api.Client {
		return api.New(apiKey, append(opts, api.WithEndpoint(ms.URL()))...)
	}
	t.Cleanup(func() { apiNewFunc = origNew })
}

// --- priority list ---

func TestPriorityList(t *testing.T) {
	ms := testutil.NewMockServer(t)
	ms.HandleQuery("GetWorkspacePriorities", priorityListResponse())
	setupPriorityTest(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"priority", "list"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("priority list returned error: %v", err)
	}

	out := buf.String()

	// Check headers
	if !strings.Contains(out, "PRIORITY") {
		t.Error("output should contain PRIORITY header")
	}
	if !strings.Contains(out, "COLOR") {
		t.Error("output should contain COLOR header")
	}

	// Check priority names (in API order, not sorted)
	if !strings.Contains(out, "Urgent") {
		t.Error("output should contain 'Urgent'")
	}
	if !strings.Contains(out, "High") {
		t.Error("output should contain 'High'")
	}
	if !strings.Contains(out, "Medium") {
		t.Error("output should contain 'Medium'")
	}
	if !strings.Contains(out, "Low") {
		t.Error("output should contain 'Low'")
	}

	// Check colors
	if !strings.Contains(out, "#ff5630") {
		t.Error("output should contain Urgent color #ff5630")
	}

	// Check footer
	if !strings.Contains(out, "Total: 4 priority(s)") {
		t.Errorf("output should show total count, got: %s", out)
	}

	// Verify cache was populated
	cached, ok := cache.Get[[]resolve.CachedPriority](resolve.PriorityCacheKey("ws-123"))
	if !ok {
		t.Error("priorities should be cached after listing")
	}
	if len(cached) != 4 {
		t.Errorf("expected 4 cached priorities, got %d", len(cached))
	}
}

func TestPriorityListJSON(t *testing.T) {
	ms := testutil.NewMockServer(t)
	ms.HandleQuery("GetWorkspacePriorities", priorityListResponse())
	setupPriorityTest(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"priority", "list", "--output=json"})
	outputFormat = "json"
	defer func() { outputFormat = "" }()

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("priority list --output=json returned error: %v", err)
	}

	var result []any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\nOutput: %s", err, buf.String())
	}

	if len(result) != 4 {
		t.Errorf("expected 4 priorities in JSON output, got %d", len(result))
	}
}

func TestPriorityListEmpty(t *testing.T) {
	ms := testutil.NewMockServer(t)
	ms.HandleQuery("GetWorkspacePriorities", priorityListEmptyResponse())
	setupPriorityTest(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"priority", "list"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("priority list returned error: %v", err)
	}

	if !strings.Contains(buf.String(), "No priorities configured") {
		t.Errorf("expected empty message, got: %s", buf.String())
	}
}

func TestPriorityListNoWorkspace(t *testing.T) {
	cacheDir := t.TempDir()
	configDir := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", cacheDir)
	t.Setenv("XDG_CONFIG_HOME", configDir)
	t.Setenv("ZH_API_KEY", "test-key")
	t.Setenv("ZH_WORKSPACE", "")
	t.Setenv("ZH_GITHUB_TOKEN", "")

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"priority", "list"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("priority list should error when no workspace configured")
	}
	if !strings.Contains(err.Error(), "no workspace") {
		t.Errorf("error = %q, want mention of no workspace", err.Error())
	}
}

// --- test data ---

func priorityListResponse() map[string]any {
	return map[string]any{
		"data": map[string]any{
			"workspace": map[string]any{
				"prioritiesConnection": map[string]any{
					"nodes": []any{
						map[string]any{"id": "pr1", "name": "Urgent", "color": "ff5630", "description": "Needs immediate attention"},
						map[string]any{"id": "pr2", "name": "High", "color": "ff7452", "description": "Important"},
						map[string]any{"id": "pr3", "name": "Medium", "color": "ffab00", "description": "Normal priority"},
						map[string]any{"id": "pr4", "name": "Low", "color": "36b37e", "description": "Can wait"},
					},
				},
			},
		},
	}
}

func priorityListEmptyResponse() map[string]any {
	return map[string]any{
		"data": map[string]any{
			"workspace": map[string]any{
				"prioritiesConnection": map[string]any{
					"nodes": []any{},
				},
			},
		},
	}
}
