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

func setupLabelTest(t *testing.T, ms *testutil.MockServer) {
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

// --- label list ---

func TestLabelList(t *testing.T) {
	ms := testutil.NewMockServer(t)
	ms.HandleQuery("GetWorkspaceLabels", labelListResponse())
	setupLabelTest(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"label", "list"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("label list returned error: %v", err)
	}

	out := buf.String()

	// Check headers
	if !strings.Contains(out, "LABEL") {
		t.Error("output should contain LABEL header")
	}
	if !strings.Contains(out, "COLOR") {
		t.Error("output should contain COLOR header")
	}

	// Check label names (should be sorted alphabetically)
	if !strings.Contains(out, "bug") {
		t.Error("output should contain 'bug'")
	}
	if !strings.Contains(out, "enhancement") {
		t.Error("output should contain 'enhancement'")
	}
	if !strings.Contains(out, "help wanted") {
		t.Error("output should contain 'help wanted'")
	}

	// Check colors
	if !strings.Contains(out, "#d73a4a") {
		t.Error("output should contain bug color #d73a4a")
	}

	// Check sorted order: bug < enhancement < help wanted
	bugIdx := strings.Index(out, "bug")
	enhIdx := strings.Index(out, "enhancement")
	helpIdx := strings.Index(out, "help wanted")
	if bugIdx > enhIdx || enhIdx > helpIdx {
		t.Errorf("labels should be sorted alphabetically, got bug@%d enhancement@%d help wanted@%d", bugIdx, enhIdx, helpIdx)
	}

	// Check footer
	if !strings.Contains(out, "Total: 3 label(s)") {
		t.Errorf("output should show total count, got: %s", out)
	}

	// Verify cache was populated
	cached, ok := cache.Get[[]resolve.CachedLabel](resolve.LabelCacheKey("ws-123"))
	if !ok {
		t.Error("labels should be cached after listing")
	}
	if len(cached) != 3 {
		t.Errorf("expected 3 cached labels, got %d", len(cached))
	}
}

func TestLabelListJSON(t *testing.T) {
	ms := testutil.NewMockServer(t)
	ms.HandleQuery("GetWorkspaceLabels", labelListResponse())
	setupLabelTest(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"label", "list", "--output=json"})
	outputFormat = "json"
	defer func() { outputFormat = "" }()

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("label list --output=json returned error: %v", err)
	}

	var result []any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\nOutput: %s", err, buf.String())
	}

	if len(result) != 3 {
		t.Errorf("expected 3 labels in JSON output, got %d", len(result))
	}
}

func TestLabelListEmpty(t *testing.T) {
	ms := testutil.NewMockServer(t)
	ms.HandleQuery("GetWorkspaceLabels", labelListEmptyResponse())
	setupLabelTest(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"label", "list"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("label list returned error: %v", err)
	}

	if !strings.Contains(buf.String(), "No labels found") {
		t.Errorf("expected empty message, got: %s", buf.String())
	}
}

func TestLabelListDeduplicates(t *testing.T) {
	ms := testutil.NewMockServer(t)
	ms.HandleQuery("GetWorkspaceLabels", labelListDuplicateResponse())
	setupLabelTest(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"label", "list"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("label list returned error: %v", err)
	}

	out := buf.String()

	// "bug" appears in both repos but should be deduplicated
	if !strings.Contains(out, "Total: 2 label(s)") {
		t.Errorf("expected 2 labels after dedup, got: %s", out)
	}
}

func TestLabelListNoWorkspace(t *testing.T) {
	cacheDir := t.TempDir()
	configDir := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", cacheDir)
	t.Setenv("XDG_CONFIG_HOME", configDir)
	t.Setenv("ZH_API_KEY", "test-key")
	t.Setenv("ZH_WORKSPACE", "")
	t.Setenv("ZH_GITHUB_TOKEN", "")

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"label", "list"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("label list should error when no workspace configured")
	}
	if !strings.Contains(err.Error(), "no workspace") {
		t.Errorf("error = %q, want mention of no workspace", err.Error())
	}
}

// --- test data ---

func labelListResponse() map[string]any {
	return map[string]any{
		"data": map[string]any{
			"workspace": map[string]any{
				"repositoriesConnection": map[string]any{
					"nodes": []any{
						map[string]any{
							"labels": map[string]any{
								"nodes": []any{
									map[string]any{"id": "l1", "name": "bug", "color": "d73a4a"},
									map[string]any{"id": "l2", "name": "enhancement", "color": "a2eeef"},
								},
							},
						},
						map[string]any{
							"labels": map[string]any{
								"nodes": []any{
									map[string]any{"id": "l3", "name": "help wanted", "color": "008672"},
								},
							},
						},
					},
				},
			},
		},
	}
}

func labelListEmptyResponse() map[string]any {
	return map[string]any{
		"data": map[string]any{
			"workspace": map[string]any{
				"repositoriesConnection": map[string]any{
					"nodes": []any{},
				},
			},
		},
	}
}

func labelListDuplicateResponse() map[string]any {
	return map[string]any{
		"data": map[string]any{
			"workspace": map[string]any{
				"repositoriesConnection": map[string]any{
					"nodes": []any{
						map[string]any{
							"labels": map[string]any{
								"nodes": []any{
									map[string]any{"id": "l1", "name": "bug", "color": "d73a4a"},
									map[string]any{"id": "l2", "name": "enhancement", "color": "a2eeef"},
								},
							},
						},
						map[string]any{
							"labels": map[string]any{
								"nodes": []any{
									map[string]any{"id": "l1b", "name": "bug", "color": "d73a4a"},
								},
							},
						},
					},
				},
			},
		},
	}
}
