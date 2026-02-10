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

func resetPipelineFlags() {
	pipelineShowLimit = 100
	pipelineShowAll = false
}

// --- pipeline list ---

func TestPipelineList(t *testing.T) {
	resetPipelineFlags()

	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListPipelinesFull", pipelineListResponse())

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
	defer func() { apiNewFunc = origNew }()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"pipeline", "list"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("pipeline list returned error: %v", err)
	}

	out := buf.String()

	// Check headers
	if !strings.Contains(out, "PIPELINE") {
		t.Error("output should contain PIPELINE header")
	}
	if !strings.Contains(out, "ISSUES") {
		t.Error("output should contain ISSUES header")
	}
	if !strings.Contains(out, "STAGE") {
		t.Error("output should contain STAGE header")
	}

	// Check pipeline names in order
	if !strings.Contains(out, "New Issues") {
		t.Error("output should contain 'New Issues'")
	}
	if !strings.Contains(out, "In Development") {
		t.Error("output should contain 'In Development'")
	}
	if !strings.Contains(out, "Done") {
		t.Error("output should contain 'Done'")
	}

	// Check position numbers
	if !strings.Contains(out, "1") {
		t.Error("output should show position number 1")
	}

	// Check stage formatting
	if !strings.Contains(out, "Development") {
		t.Error("output should show formatted stage")
	}

	// Check footer
	if !strings.Contains(out, "Total: 3 pipeline(s)") {
		t.Errorf("output should show total count, got: %s", out)
	}

	// Verify cache was populated
	cached, ok := cache.Get[[]resolve.CachedPipeline](resolve.PipelineCacheKey("ws-123"))
	if !ok {
		t.Error("pipelines should be cached after listing")
	}
	if len(cached) != 3 {
		t.Errorf("expected 3 cached pipelines, got %d", len(cached))
	}
}

func TestPipelineListJSON(t *testing.T) {
	resetPipelineFlags()

	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListPipelinesFull", pipelineListResponse())

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
	defer func() { apiNewFunc = origNew }()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"pipeline", "list", "--output=json"})
	outputFormat = "json"
	defer func() { outputFormat = "" }()

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("pipeline list --output=json returned error: %v", err)
	}

	var result []any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\nOutput: %s", err, buf.String())
	}

	if len(result) != 3 {
		t.Errorf("expected 3 pipelines in JSON output, got %d", len(result))
	}
}

func TestPipelineListEmpty(t *testing.T) {
	resetPipelineFlags()

	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListPipelinesFull", map[string]any{
		"data": map[string]any{
			"workspace": map[string]any{
				"pipelinesConnection": map[string]any{
					"totalCount": 0,
					"nodes":      []any{},
				},
			},
		},
	})

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
	defer func() { apiNewFunc = origNew }()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"pipeline", "list"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("pipeline list returned error: %v", err)
	}

	if !strings.Contains(buf.String(), "No pipelines found") {
		t.Errorf("expected empty message, got: %s", buf.String())
	}
}

func TestPipelineListNoWorkspace(t *testing.T) {
	resetPipelineFlags()

	cacheDir := t.TempDir()
	configDir := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", cacheDir)
	t.Setenv("XDG_CONFIG_HOME", configDir)
	t.Setenv("ZH_API_KEY", "test-key")
	t.Setenv("ZH_WORKSPACE", "")
	t.Setenv("ZH_GITHUB_TOKEN", "")

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"pipeline", "list"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("pipeline list should error when no workspace configured")
	}
	if !strings.Contains(err.Error(), "no workspace") {
		t.Errorf("error = %q, want mention of no workspace", err.Error())
	}
}

// --- pipeline show ---

func TestPipelineShow(t *testing.T) {
	resetPipelineFlags()

	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListPipelines", pipelineResolutionResponse())
	ms.HandleQuery("GetPipelineDetails", pipelineDetailResponseData())
	ms.HandleQuery("GetPipelineIssues", pipelineIssuesResponseData())

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
	defer func() { apiNewFunc = origNew }()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"pipeline", "show", "In Development"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("pipeline show returned error: %v", err)
	}

	out := buf.String()

	if !strings.Contains(out, "PIPELINE: In Development") {
		t.Error("output should contain pipeline title")
	}
	if !strings.Contains(out, "Development") {
		t.Error("output should show stage")
	}
	if !strings.Contains(out, "ISSUES") {
		t.Error("output should contain ISSUES section")
	}
	if !strings.Contains(out, "Fix login") {
		t.Error("output should contain issue title")
	}
	if !strings.Contains(out, "task-tracker#1") {
		t.Error("output should contain issue reference")
	}
}

func TestPipelineShowBySubstring(t *testing.T) {
	resetPipelineFlags()

	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListPipelines", pipelineResolutionResponse())
	ms.HandleQuery("GetPipelineDetails", pipelineDetailResponseData())
	ms.HandleQuery("GetPipelineIssues", pipelineIssuesResponseData())

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
	defer func() { apiNewFunc = origNew }()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"pipeline", "show", "Dev"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("pipeline show by substring returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "PIPELINE: In Development") {
		t.Errorf("should resolve by substring, got: %s", out)
	}
}

func TestPipelineShowNotFound(t *testing.T) {
	resetPipelineFlags()

	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListPipelines", pipelineResolutionResponse())

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
	defer func() { apiNewFunc = origNew }()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"pipeline", "show", "nonexistent"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("pipeline show should error for nonexistent pipeline")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error = %q, want mention of not found", err.Error())
	}
}

func TestPipelineShowJSON(t *testing.T) {
	resetPipelineFlags()

	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListPipelines", pipelineResolutionResponse())
	ms.HandleQuery("GetPipelineDetails", pipelineDetailResponseData())
	ms.HandleQuery("GetPipelineIssues", pipelineIssuesResponseData())

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
	defer func() { apiNewFunc = origNew }()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"pipeline", "show", "In Development", "--output=json"})
	outputFormat = "json"
	defer func() { outputFormat = "" }()

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("pipeline show --output=json returned error: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\nOutput: %s", err, buf.String())
	}

	if _, ok := result["pipeline"]; !ok {
		t.Error("JSON output should contain 'pipeline' key")
	}
	if _, ok := result["issues"]; !ok {
		t.Error("JSON output should contain 'issues' key")
	}
}

func TestPipelineHelpText(t *testing.T) {
	resetPipelineFlags()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"pipeline", "--help"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("pipeline --help returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "list") {
		t.Error("help should mention list subcommand")
	}
	if !strings.Contains(out, "show") {
		t.Error("help should mention show subcommand")
	}
}

// --- pipeline automations ---

func TestPipelineAutomations(t *testing.T) {
	resetPipelineFlags()

	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListPipelines", pipelineResolutionResponse())
	ms.HandleQuery("PipelineAutomations", pipelineAutomationsResponse())

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
	defer func() { apiNewFunc = origNew }()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"pipeline", "automations", "In Development"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("pipeline automations returned error: %v", err)
	}

	out := buf.String()

	if !strings.Contains(out, "AUTOMATIONS: In Development") {
		t.Error("output should contain automations title")
	}
	if !strings.Contains(out, "PIPELINE-TO-PIPELINE AUTOMATIONS") {
		t.Error("output should contain P2P automations section")
	}
	if !strings.Contains(out, "Moves to") {
		t.Error("output should show 'Moves to' direction")
	}
	if !strings.Contains(out, "Done") {
		t.Error("output should show destination pipeline name")
	}
	if !strings.Contains(out, "Moves from") {
		t.Error("output should show 'Moves from' direction")
	}
	if !strings.Contains(out, "New Issues") {
		t.Error("output should show source pipeline name")
	}
}

func TestPipelineAutomationsNoAutomations(t *testing.T) {
	resetPipelineFlags()

	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListPipelines", pipelineResolutionResponse())
	ms.HandleQuery("PipelineAutomations", pipelineAutomationsEmptyResponse())

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
	defer func() { apiNewFunc = origNew }()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"pipeline", "automations", "In Development"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("pipeline automations returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "No automations configured") {
		t.Errorf("expected no automations message, got: %s", out)
	}
}

func TestPipelineAutomationsJSON(t *testing.T) {
	resetPipelineFlags()

	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListPipelines", pipelineResolutionResponse())
	ms.HandleQuery("PipelineAutomations", pipelineAutomationsResponse())

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
	defer func() { apiNewFunc = origNew }()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"pipeline", "automations", "In Development", "--output=json"})
	outputFormat = "json"
	defer func() { outputFormat = "" }()

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("pipeline automations --output=json returned error: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\nOutput: %s", err, buf.String())
	}

	if result["pipeline"] != "In Development" {
		t.Errorf("JSON should contain pipeline name, got: %v", result)
	}
	if _, ok := result["eventAutomations"]; !ok {
		t.Error("JSON output should contain 'eventAutomations' key")
	}
	if _, ok := result["p2pSources"]; !ok {
		t.Error("JSON output should contain 'p2pSources' key")
	}
	if _, ok := result["p2pDestinations"]; !ok {
		t.Error("JSON output should contain 'p2pDestinations' key")
	}
}

func TestPipelineAutomationsWithEventAutomations(t *testing.T) {
	resetPipelineFlags()

	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListPipelines", pipelineResolutionResponse())
	ms.HandleQuery("PipelineAutomations", pipelineAutomationsWithEventsResponse())

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
	defer func() { apiNewFunc = origNew }()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"pipeline", "automations", "In Development"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("pipeline automations returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "EVENT AUTOMATIONS") {
		t.Error("output should contain EVENT AUTOMATIONS section")
	}
	if !strings.Contains(out, "auto-close") {
		t.Error("output should contain automation element details")
	}
}

// --- helpers ---

func pipelineAutomationsResponse() map[string]any {
	return map[string]any{
		"data": map[string]any{
			"workspace": map[string]any{
				"pipelinesConnection": map[string]any{
					"nodes": []any{
						buildAutomationPipelineNode("p1", "New Issues", 0, nil, 0, nil, 0, nil),
						buildAutomationPipelineNode("p2", "In Development", 0, nil,
							1, []any{
								map[string]any{
									"id":                  "auto-1",
									"destinationPipeline": map[string]any{"id": "p3", "name": "Done"},
									"createdAt":           "2026-01-15T10:00:00Z",
								},
							},
							1, []any{
								map[string]any{
									"id":             "auto-2",
									"sourcePipeline": map[string]any{"id": "p1", "name": "New Issues"},
									"createdAt":      "2026-01-15T10:00:00Z",
								},
							},
						),
						buildAutomationPipelineNode("p3", "Done", 0, nil, 0, nil, 0, nil),
					},
				},
			},
		},
	}
}

func pipelineAutomationsEmptyResponse() map[string]any {
	return map[string]any{
		"data": map[string]any{
			"workspace": map[string]any{
				"pipelinesConnection": map[string]any{
					"nodes": []any{
						buildAutomationPipelineNode("p1", "New Issues", 0, nil, 0, nil, 0, nil),
						buildAutomationPipelineNode("p2", "In Development", 0, nil, 0, nil, 0, nil),
						buildAutomationPipelineNode("p3", "Done", 0, nil, 0, nil, 0, nil),
					},
				},
			},
		},
	}
}

func pipelineAutomationsWithEventsResponse() map[string]any {
	return map[string]any{
		"data": map[string]any{
			"workspace": map[string]any{
				"pipelinesConnection": map[string]any{
					"nodes": []any{
						buildAutomationPipelineNode("p1", "New Issues", 0, nil, 0, nil, 0, nil),
						buildAutomationPipelineNode("p2", "In Development",
							1, []any{
								map[string]any{
									"id":             "event-auto-1",
									"elementDetails": map[string]any{"type": "auto-close", "trigger": "pr_merged"},
									"createdAt":      "2026-01-10T08:00:00Z",
									"updatedAt":      "2026-01-10T08:00:00Z",
								},
							},
							0, nil, 0, nil,
						),
						buildAutomationPipelineNode("p3", "Done", 0, nil, 0, nil, 0, nil),
					},
				},
			},
		},
	}
}

// buildAutomationPipelineNode builds a pipeline node for the automations response.
func buildAutomationPipelineNode(id, name string, eventCount int, eventNodes []any, srcCount int, srcNodes []any, destCount int, destNodes []any) map[string]any {
	if eventNodes == nil {
		eventNodes = []any{}
	}
	if srcNodes == nil {
		srcNodes = []any{}
	}
	if destNodes == nil {
		destNodes = []any{}
	}
	return map[string]any{
		"id":   id,
		"name": name,
		"pipelineConfiguration": map[string]any{
			"pipelineAutomations": map[string]any{
				"totalCount": eventCount,
				"nodes":      eventNodes,
			},
		},
		"pipelineToPipelineAutomationSources": map[string]any{
			"totalCount": srcCount,
			"nodes":      srcNodes,
		},
		"pipelineToPipelineAutomationDestinations": map[string]any{
			"totalCount": destCount,
			"nodes":      destNodes,
		},
	}
}

func pipelineListResponse() map[string]any {
	return map[string]any{
		"data": map[string]any{
			"workspace": map[string]any{
				"pipelinesConnection": map[string]any{
					"totalCount": 3,
					"nodes": []any{
						map[string]any{
							"id":                  "p1",
							"name":                "New Issues",
							"description":         nil,
							"stage":               nil,
							"isDefaultPRPipeline": false,
							"issues":              map[string]any{"totalCount": 12},
						},
						map[string]any{
							"id":                  "p2",
							"name":                "In Development",
							"description":         "Active work",
							"stage":               "DEVELOPMENT",
							"isDefaultPRPipeline": true,
							"issues":              map[string]any{"totalCount": 5},
						},
						map[string]any{
							"id":                  "p3",
							"name":                "Done",
							"description":         nil,
							"stage":               "COMPLETED",
							"isDefaultPRPipeline": false,
							"issues":              map[string]any{"totalCount": 42},
						},
					},
				},
			},
		},
	}
}

func pipelineResolutionResponse() map[string]any {
	return map[string]any{
		"data": map[string]any{
			"workspace": map[string]any{
				"pipelinesConnection": map[string]any{
					"nodes": []any{
						map[string]any{"id": "p1", "name": "New Issues"},
						map[string]any{"id": "p2", "name": "In Development"},
						map[string]any{"id": "p3", "name": "Done"},
					},
				},
			},
		},
	}
}

func pipelineDetailResponseData() map[string]any {
	return map[string]any{
		"data": map[string]any{
			"node": map[string]any{
				"id":                  "p2",
				"name":                "In Development",
				"description":         "Active development work",
				"stage":               "DEVELOPMENT",
				"isDefaultPRPipeline": true,
				"createdAt":           "2026-02-06T22:27:05Z",
				"updatedAt":           "2026-02-08T10:00:00Z",
				"pipelineConfiguration": map[string]any{
					"showAgeInPipeline": true,
					"staleIssues":       true,
					"staleInterval":     14,
				},
				"issues": map[string]any{"totalCount": 2},
			},
		},
	}
}

func pipelineIssuesResponseData() map[string]any {
	return map[string]any{
		"data": map[string]any{
			"searchIssuesByPipeline": map[string]any{
				"totalCount": 2,
				"pageInfo": map[string]any{
					"hasNextPage": false,
					"endCursor":   "",
				},
				"nodes": []any{
					map[string]any{
						"id":          "i1",
						"number":      1,
						"title":       "Fix login button alignment",
						"state":       "OPEN",
						"pullRequest": false,
						"estimate":    map[string]any{"value": 3},
						"assignees": map[string]any{
							"nodes": []any{
								map[string]any{"login": "dlakehammond"},
							},
						},
						"labels": map[string]any{
							"nodes": []any{
								map[string]any{"name": "bug"},
							},
						},
						"repository": map[string]any{
							"name":      "task-tracker",
							"ownerName": "dlakehammond",
						},
						"blockingIssues": map[string]any{"totalCount": 0},
						"pipelineIssue": map[string]any{
							"priority": map[string]any{"name": "High priority"},
						},
					},
					map[string]any{
						"id":          "i2",
						"number":      2,
						"title":       "Add error handling",
						"state":       "OPEN",
						"pullRequest": false,
						"estimate":    nil,
						"assignees":   map[string]any{"nodes": []any{}},
						"labels":      map[string]any{"nodes": []any{}},
						"repository": map[string]any{
							"name":      "task-tracker",
							"ownerName": "dlakehammond",
						},
						"blockingIssues": map[string]any{"totalCount": 1},
						"pipelineIssue":  nil,
					},
				},
			},
		},
	}
}
