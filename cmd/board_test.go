package cmd

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/dslh/zh/internal/api"
	"github.com/dslh/zh/internal/testutil"
)

// --- board ---

func TestBoard(t *testing.T) {
	resetBoardFlags()
	resetPipelineFlags()

	ms := testutil.NewMockServer(t)
	ms.HandleQuery("GetBoard", boardResponse())

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
	rootCmd.SetArgs([]string{"board"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("board returned error: %v", err)
	}

	out := buf.String()

	// Check pipeline headers
	if !strings.Contains(out, "New Issues") {
		t.Error("output should contain 'New Issues' pipeline")
	}
	if !strings.Contains(out, "In Development") {
		t.Error("output should contain 'In Development' pipeline")
	}
	if !strings.Contains(out, "Done") {
		t.Error("output should contain 'Done' pipeline")
	}
	if !strings.Contains(out, "Closed") {
		t.Error("output should contain 'Closed' pipeline")
	}

	// Check issue references
	if !strings.Contains(out, "task-tracker#1") {
		t.Error("output should contain issue reference task-tracker#1")
	}
	if !strings.Contains(out, "recipe-book#2") {
		t.Error("output should contain issue reference recipe-book#2")
	}

	// Check closed issue appears in Closed pipeline
	if !strings.Contains(out, "Initial setup") {
		t.Error("output should contain closed issue title 'Initial setup'")
	}

	// Check issue titles
	if !strings.Contains(out, "Fix login button") {
		t.Error("output should contain issue title")
	}

	// Check footer (3 real pipelines + 1 Closed = 4)
	if !strings.Contains(out, "4 pipeline(s)") {
		t.Errorf("output should show 4 pipelines, got: %s", out)
	}
	if !strings.Contains(out, "4 issue(s)") {
		t.Errorf("output should show 4 issues, got: %s", out)
	}
}

func TestBoardEmptyPipelines(t *testing.T) {
	resetBoardFlags()
	resetPipelineFlags()

	ms := testutil.NewMockServer(t)
	ms.HandleQuery("GetBoard", boardEmptyResponse())

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
	rootCmd.SetArgs([]string{"board"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("board returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "No issues") {
		t.Errorf("empty pipeline should show 'No issues', got: %s", out)
	}
}

func TestBoardNoPipelines(t *testing.T) {
	resetBoardFlags()
	resetPipelineFlags()

	ms := testutil.NewMockServer(t)
	ms.HandleQuery("GetBoard", map[string]any{
		"data": map[string]any{
			"workspace": map[string]any{
				"id":          "ws-123",
				"displayName": "Test Workspace",
				"pipelinesConnection": map[string]any{
					"nodes": []any{},
				},
			},
			"searchClosedIssues": map[string]any{
				"totalCount": 0,
				"nodes":      []any{},
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
	rootCmd.SetArgs([]string{"board"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("board returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "No pipelines found") {
		t.Errorf("expected no pipelines message, got: %s", out)
	}
}

func TestBoardJSON(t *testing.T) {
	resetBoardFlags()
	resetPipelineFlags()

	ms := testutil.NewMockServer(t)
	ms.HandleQuery("GetBoard", boardResponse())

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
	rootCmd.SetArgs([]string{"board", "--output=json"})
	outputFormat = "json"
	defer func() { outputFormat = "" }()

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("board --output=json returned error: %v", err)
	}

	var result []any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\nOutput: %s", err, buf.String())
	}

	if len(result) != 4 {
		t.Errorf("expected 4 pipelines in JSON output (3 real + Closed), got %d", len(result))
	}
}

func TestBoardFilteredPipeline(t *testing.T) {
	resetBoardFlags()
	resetPipelineFlags()

	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListPipelines", pipelineResolutionResponse())
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
	rootCmd.SetArgs([]string{"board", "--pipeline=In Development"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("board --pipeline returned error: %v", err)
	}

	out := buf.String()

	if !strings.Contains(out, "In Development") {
		t.Error("output should contain filtered pipeline name")
	}
	if !strings.Contains(out, "task-tracker#1") {
		t.Error("output should contain issue reference")
	}
	if !strings.Contains(out, "1 pipeline") {
		t.Errorf("output should show '1 pipeline', got: %s", out)
	}
}

func TestBoardFilteredPipelineJSON(t *testing.T) {
	resetBoardFlags()
	resetPipelineFlags()

	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListPipelines", pipelineResolutionResponse())
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
	rootCmd.SetArgs([]string{"board", "--pipeline=In Development", "--output=json"})
	outputFormat = "json"
	defer func() { outputFormat = "" }()

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("board --pipeline --output=json returned error: %v", err)
	}

	var result []any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\nOutput: %s", err, buf.String())
	}

	if len(result) != 1 {
		t.Errorf("expected 1 pipeline in filtered JSON output, got %d", len(result))
	}
}

func TestBoardClosedPipeline(t *testing.T) {
	resetBoardFlags()
	resetPipelineFlags()

	ms := testutil.NewMockServer(t)
	ms.HandleQuery("SearchClosedIssues", map[string]any{
		"data": map[string]any{
			"searchClosedIssues": map[string]any{
				"totalCount": 1,
				"nodes": []any{
					boardIssueData("i4", 1, "Initial setup", "CLOSED", false, 1,
						"task-tracker", "dlakehammond", "alice"),
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
	rootCmd.SetArgs([]string{"board", "--pipeline=Closed"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("board --pipeline=Closed returned error: %v", err)
	}

	out := buf.String()

	if !strings.Contains(out, "Closed") {
		t.Error("output should contain 'Closed' pipeline header")
	}
	if !strings.Contains(out, "Initial setup") {
		t.Error("output should contain closed issue title")
	}
	if !strings.Contains(out, "task-tracker#1") {
		t.Error("output should contain closed issue reference")
	}
	if !strings.Contains(out, "1 pipeline") {
		t.Errorf("output should show '1 pipeline', got: %s", out)
	}
}

func TestBoardClosedPipelineJSON(t *testing.T) {
	resetBoardFlags()
	resetPipelineFlags()

	ms := testutil.NewMockServer(t)
	ms.HandleQuery("SearchClosedIssues", map[string]any{
		"data": map[string]any{
			"searchClosedIssues": map[string]any{
				"totalCount": 1,
				"nodes": []any{
					boardIssueData("i4", 1, "Initial setup", "CLOSED", false, 1,
						"task-tracker", "dlakehammond", "alice"),
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
	rootCmd.SetArgs([]string{"board", "--pipeline=Closed", "--output=json"})
	outputFormat = "json"
	defer func() { outputFormat = "" }()

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("board --pipeline=Closed --output=json returned error: %v", err)
	}

	var result []any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\nOutput: %s", err, buf.String())
	}

	if len(result) != 1 {
		t.Errorf("expected 1 pipeline in closed JSON output, got %d", len(result))
	}
}

func TestBoardFilteredPipelineFiltersClosedIssues(t *testing.T) {
	resetBoardFlags()
	resetPipelineFlags()

	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListPipelines", pipelineResolutionResponse())
	ms.HandleQuery("GetPipelineIssues", map[string]any{
		"data": map[string]any{
			"searchIssuesByPipeline": map[string]any{
				"totalCount": 3,
				"pageInfo": map[string]any{
					"hasNextPage": false,
					"endCursor":   "",
				},
				"nodes": []any{
					map[string]any{
						"id":          "i1",
						"number":      1,
						"title":       "Open issue",
						"state":       "OPEN",
						"pullRequest": false,
						"estimate":    nil,
						"assignees":   map[string]any{"nodes": []any{}},
						"labels":      map[string]any{"nodes": []any{}},
						"repository": map[string]any{
							"name":      "task-tracker",
							"ownerName": "dlakehammond",
						},
						"blockingIssues": map[string]any{"totalCount": 0},
						"pipelineIssue":  nil,
					},
					map[string]any{
						"id":          "i2",
						"number":      2,
						"title":       "Closed issue in pipeline",
						"state":       "CLOSED",
						"pullRequest": false,
						"estimate":    nil,
						"assignees":   map[string]any{"nodes": []any{}},
						"labels":      map[string]any{"nodes": []any{}},
						"repository": map[string]any{
							"name":      "task-tracker",
							"ownerName": "dlakehammond",
						},
						"blockingIssues": map[string]any{"totalCount": 0},
						"pipelineIssue":  nil,
					},
					map[string]any{
						"id":          "i3",
						"number":      3,
						"title":       "Another open issue",
						"state":       "OPEN",
						"pullRequest": false,
						"estimate":    nil,
						"assignees":   map[string]any{"nodes": []any{}},
						"labels":      map[string]any{"nodes": []any{}},
						"repository": map[string]any{
							"name":      "task-tracker",
							"ownerName": "dlakehammond",
						},
						"blockingIssues": map[string]any{"totalCount": 0},
						"pipelineIssue":  nil,
					},
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
	rootCmd.SetArgs([]string{"board", "--pipeline=In Development"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("board --pipeline returned error: %v", err)
	}

	out := buf.String()

	// Open issues should appear
	if !strings.Contains(out, "Open issue") {
		t.Error("output should contain open issue")
	}
	if !strings.Contains(out, "Another open issue") {
		t.Error("output should contain second open issue")
	}

	// Closed issue should be filtered out
	if strings.Contains(out, "Closed issue in pipeline") {
		t.Error("output should not contain closed issue")
	}

	// Count should reflect filtering (3 total - 1 closed = 2)
	if !strings.Contains(out, "2 issue(s)") {
		t.Errorf("output should show 2 issues after filtering, got: %s", out)
	}
}

// --- helpers ---

func boardResponse() map[string]any {
	return map[string]any{
		"data": map[string]any{
			"workspace": map[string]any{
				"id":          "ws-123",
				"displayName": "Test Workspace",
				"pipelinesConnection": map[string]any{
					"nodes": []any{
						map[string]any{
							"id":   "p1",
							"name": "New Issues",
							"issues": map[string]any{
								"totalCount": 1,
								"nodes": []any{
									boardIssueData("i1", 1, "Fix login button", "OPEN", false, 3,
										"task-tracker", "dlakehammond", "alice"),
								},
							},
						},
						map[string]any{
							"id":   "p2",
							"name": "In Development",
							"issues": map[string]any{
								"totalCount": 2,
								"nodes": []any{
									boardIssueData("i2", 2, "Add search feature", "OPEN", false, 5,
										"task-tracker", "dlakehammond", "bob"),
									boardIssueData("i3", 2, "Fix recipe validation", "OPEN", false, 0,
										"recipe-book", "dlakehammond", ""),
								},
							},
						},
						map[string]any{
							"id":   "p3",
							"name": "Done",
							"issues": map[string]any{
								"totalCount": 0,
								"nodes":      []any{},
							},
						},
					},
				},
			},
			"searchClosedIssues": map[string]any{
				"totalCount": 1,
				"nodes": []any{
					boardIssueData("i4", 1, "Initial setup", "CLOSED", false, 1,
						"task-tracker", "dlakehammond", "alice"),
				},
			},
		},
	}
}

func boardEmptyResponse() map[string]any {
	return map[string]any{
		"data": map[string]any{
			"workspace": map[string]any{
				"id":          "ws-123",
				"displayName": "Test Workspace",
				"pipelinesConnection": map[string]any{
					"nodes": []any{
						map[string]any{
							"id":   "p1",
							"name": "New Issues",
							"issues": map[string]any{
								"totalCount": 0,
								"nodes":      []any{},
							},
						},
						map[string]any{
							"id":   "p2",
							"name": "In Development",
							"issues": map[string]any{
								"totalCount": 0,
								"nodes":      []any{},
							},
						},
					},
				},
			},
			"searchClosedIssues": map[string]any{
				"totalCount": 0,
				"nodes":      []any{},
			},
		},
	}
}

func boardIssueData(id string, number int, title, state string, pullRequest bool, estimate float64, repo, owner, assignee string) map[string]any {
	issue := map[string]any{
		"id":          id,
		"number":      number,
		"title":       title,
		"state":       state,
		"pullRequest": pullRequest,
		"repository": map[string]any{
			"name":      repo,
			"ownerName": owner,
		},
		"labels":        map[string]any{"nodes": []any{}},
		"pipelineIssue": nil,
	}

	if estimate > 0 {
		issue["estimate"] = map[string]any{"value": estimate}
	} else {
		issue["estimate"] = nil
	}

	if assignee != "" {
		issue["assignees"] = map[string]any{
			"nodes": []any{map[string]any{"login": assignee}},
		}
	} else {
		issue["assignees"] = map[string]any{"nodes": []any{}}
	}

	return issue
}
