package cmd

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/dslh/zh/internal/api"
	"github.com/dslh/zh/internal/cache"
	"github.com/dslh/zh/internal/gh"
	"github.com/dslh/zh/internal/resolve"
	"github.com/dslh/zh/internal/testutil"
)

// --- issue list ---

func TestIssueList(t *testing.T) {
	resetIssueFlags()

	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListPipelines", pipelineResolutionResponse())
	ms.HandleQuery("ListIssuesByPipeline", issueListByPipelineResponse())

	setupIssueTestEnv(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"issue", "list"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("issue list returned error: %v", err)
	}

	out := buf.String()

	// Check headers
	if !strings.Contains(out, "ISSUE") {
		t.Error("output should contain ISSUE header")
	}
	if !strings.Contains(out, "TITLE") {
		t.Error("output should contain TITLE header")
	}
	if !strings.Contains(out, "PIPELINE") {
		t.Error("output should contain PIPELINE header")
	}

	// Check issue content
	if !strings.Contains(out, "task-tracker#1") {
		t.Errorf("output should contain issue reference, got: %s", out)
	}
	if !strings.Contains(out, "Fix login") {
		t.Error("output should contain issue title")
	}
	if !strings.Contains(out, "dlakehammond") {
		t.Error("output should contain assignee")
	}
	if !strings.Contains(out, "bug") {
		t.Error("output should contain label")
	}

	// Check footer
	if !strings.Contains(out, "issue(s)") {
		t.Errorf("output should contain issue count footer, got: %s", out)
	}
}

func TestIssueListJSON(t *testing.T) {
	resetIssueFlags()

	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListPipelines", pipelineResolutionResponse())
	ms.HandleQuery("ListIssuesByPipeline", issueListByPipelineResponse())

	setupIssueTestEnv(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"issue", "list", "--output=json"})
	outputFormat = "json"
	defer func() { outputFormat = "" }()

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("issue list --output=json returned error: %v", err)
	}

	var result []any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\nOutput: %s", err, buf.String())
	}

	// 3 pipelines * 2 issues each = 6 total
	if len(result) != 6 {
		t.Errorf("expected 6 issues in JSON output (2 per pipeline * 3 pipelines), got %d", len(result))
	}
}

func TestIssueListEmpty(t *testing.T) {
	resetIssueFlags()

	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListPipelines", pipelineResolutionResponse())
	ms.HandleQuery("ListIssuesByPipeline", issueListEmptyResponse())

	setupIssueTestEnv(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"issue", "list"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("issue list returned error: %v", err)
	}

	if !strings.Contains(buf.String(), "No issues found") {
		t.Errorf("expected empty message, got: %s", buf.String())
	}
}

func TestIssueListNoWorkspace(t *testing.T) {
	resetIssueFlags()

	cacheDir := t.TempDir()
	configDir := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", cacheDir)
	t.Setenv("XDG_CONFIG_HOME", configDir)
	t.Setenv("ZH_API_KEY", "test-key")
	t.Setenv("ZH_WORKSPACE", "")
	t.Setenv("ZH_GITHUB_TOKEN", "")

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"issue", "list"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("issue list should error when no workspace configured")
	}
	if !strings.Contains(err.Error(), "no workspace") {
		t.Errorf("error = %q, want mention of no workspace", err.Error())
	}
}

func TestIssueListPipelineFilter(t *testing.T) {
	resetIssueFlags()

	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListPipelines", pipelineResolutionResponse())
	ms.HandleQuery("ListIssuesByPipeline", issueListByPipelineResponse())

	setupIssueTestEnv(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"issue", "list", "--pipeline=In Development"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("issue list --pipeline returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "task-tracker#1") {
		t.Errorf("output should contain issues, got: %s", out)
	}
}

func TestIssueListClosedState(t *testing.T) {
	resetIssueFlags()

	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListClosedIssues", issueListClosedResponse())

	setupIssueTestEnv(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"issue", "list", "--state=closed"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("issue list --state=closed returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "task-tracker#10") {
		t.Errorf("output should contain closed issue, got: %s", out)
	}
}

// --- issue show ---

func TestIssueShow(t *testing.T) {
	resetIssueFlags()

	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListRepos", repoResolutionResponse())
	ms.HandleQuery("IssueByInfo", issueByInfoResolutionResponse())
	ms.HandleQuery("GetIssueDetails", issueShowResponse())

	setupIssueTestEnv(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"issue", "show", "task-tracker#1"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("issue show returned error: %v", err)
	}

	out := buf.String()

	// Check title
	if !strings.Contains(out, "ISSUE: task-tracker#1: Fix login button") {
		t.Errorf("output should contain issue title, got: %s", out)
	}

	// Check fields
	if !strings.Contains(out, "State") {
		t.Error("output should contain State field")
	}
	if !strings.Contains(out, "Pipeline") {
		t.Error("output should contain Pipeline field")
	}
	if !strings.Contains(out, "In Development") {
		t.Error("output should show pipeline name")
	}
	if !strings.Contains(out, "Estimate") {
		t.Error("output should contain Estimate field")
	}
	if !strings.Contains(out, "@dlakehammond") {
		t.Error("output should contain assignee with @ prefix")
	}
	if !strings.Contains(out, "bug") {
		t.Error("output should contain label")
	}

	// Check sections
	if !strings.Contains(out, "DESCRIPTION") {
		t.Error("output should contain DESCRIPTION section")
	}
	if !strings.Contains(out, "LINKS") {
		t.Error("output should contain LINKS section")
	}
	if !strings.Contains(out, "TIMELINE") {
		t.Error("output should contain TIMELINE section")
	}
}

func TestIssueShowJSON(t *testing.T) {
	resetIssueFlags()

	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListRepos", repoResolutionResponse())
	ms.HandleQuery("IssueByInfo", issueByInfoResolutionResponse())
	ms.HandleQuery("GetIssueDetails", issueShowResponse())

	setupIssueTestEnv(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"issue", "show", "task-tracker#1", "--output=json"})
	outputFormat = "json"
	defer func() { outputFormat = "" }()

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("issue show --output=json returned error: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\nOutput: %s", err, buf.String())
	}

	if result["title"] != "Fix login button alignment" {
		t.Errorf("JSON should contain title, got: %v", result["title"])
	}
	if result["number"] != float64(1) {
		t.Errorf("JSON should contain number, got: %v", result["number"])
	}
}

func TestIssueShowWithBlockers(t *testing.T) {
	resetIssueFlags()

	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListRepos", repoResolutionResponse())
	ms.HandleQuery("IssueByInfo", issueByInfoResolutionResponse())
	ms.HandleQuery("GetIssueDetails", issueShowWithBlockersResponse())

	setupIssueTestEnv(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"issue", "show", "task-tracker#1"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("issue show returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "BLOCKED BY") {
		t.Error("output should contain BLOCKED BY section")
	}
	if !strings.Contains(out, "task-tracker#5") {
		t.Error("output should show blocking issue reference")
	}
}

func TestIssueShowWithConnectedPRs(t *testing.T) {
	resetIssueFlags()

	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListRepos", repoResolutionResponse())
	ms.HandleQuery("IssueByInfo", issueByInfoResolutionResponse())
	ms.HandleQuery("GetIssueDetails", issueShowWithPRsResponse())

	setupIssueTestEnv(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"issue", "show", "task-tracker#1"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("issue show returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "CONNECTED PRS") {
		t.Error("output should contain CONNECTED PRS section")
	}
	if !strings.Contains(out, "task-tracker#10") {
		t.Error("output should show connected PR reference")
	}
}

func TestIssueShowNotFound(t *testing.T) {
	resetIssueFlags()

	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListRepos", repoResolutionResponse())
	ms.HandleQuery("IssueByInfo", map[string]any{
		"data": map[string]any{
			"issueByInfo": nil,
		},
	})

	setupIssueTestEnv(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"issue", "show", "task-tracker#999"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("issue show should error for nonexistent issue")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error = %q, want mention of not found", err.Error())
	}
}

func TestIssueShowWithGitHub(t *testing.T) {
	resetIssueFlags()

	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListRepos", repoResolutionResponse())
	ms.HandleQuery("IssueByInfo", issueByInfoResolutionResponse())
	ms.HandleQuery("GetIssueDetails", issueShowResponse())

	ghMs := testutil.NewMockServer(t)
	ghMs.HandleQuery("GetIssueGitHub", issueShowGitHubResponse())

	setupIssueTestEnvWithGitHub(t, ms, ghMs)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"issue", "show", "task-tracker#1"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("issue show with GitHub returned error: %v", err)
	}

	out := buf.String()

	// Check GitHub-enriched fields
	if !strings.Contains(out, "Author") {
		t.Error("output should contain Author field")
	}
	if !strings.Contains(out, "@testuser") {
		t.Error("output should contain author login")
	}
	if !strings.Contains(out, "REACTIONS") {
		t.Error("output should contain REACTIONS section")
	}
	if !strings.Contains(out, "+1") {
		t.Error("output should contain thumbs up reaction")
	}
}

func TestIssueShowPRWithGitHub(t *testing.T) {
	resetIssueFlags()

	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListRepos", repoResolutionResponse())
	ms.HandleQuery("IssueByInfo", issueByInfoResolutionResponse())
	ms.HandleQuery("GetIssueDetails", issueShowPRResponse())

	ghMs := testutil.NewMockServer(t)
	ghMs.HandleQuery("GetIssueGitHub", issueShowGitHubPRResponse())

	setupIssueTestEnvWithGitHub(t, ms, ghMs)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"issue", "show", "task-tracker#1"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("issue show PR with GitHub returned error: %v", err)
	}

	out := buf.String()

	// Check PR-specific fields
	if !strings.Contains(out, "PR:") {
		t.Errorf("output should show PR type, got: %s", out)
	}
	if !strings.Contains(out, "REVIEWS") {
		t.Error("output should contain REVIEWS section")
	}
	if !strings.Contains(out, "@reviewer1") {
		t.Error("output should contain reviewer login")
	}
	if !strings.Contains(out, "Approved") {
		t.Error("output should contain review state")
	}
	if !strings.Contains(out, "CI") {
		t.Error("output should contain CI field")
	}
	if !strings.Contains(out, "Passing") {
		t.Error("output should show CI status as Passing")
	}
}

func TestIssueShowJSONWithGitHub(t *testing.T) {
	resetIssueFlags()

	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListRepos", repoResolutionResponse())
	ms.HandleQuery("IssueByInfo", issueByInfoResolutionResponse())
	ms.HandleQuery("GetIssueDetails", issueShowResponse())

	ghMs := testutil.NewMockServer(t)
	ghMs.HandleQuery("GetIssueGitHub", issueShowGitHubResponse())

	setupIssueTestEnvWithGitHub(t, ms, ghMs)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"issue", "show", "task-tracker#1", "--output=json"})
	outputFormat = "json"
	defer func() { outputFormat = "" }()

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("issue show JSON with GitHub returned error: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\nOutput: %s", err, buf.String())
	}

	if result["author"] != "testuser" {
		t.Errorf("JSON should contain author, got: %v", result["author"])
	}
	if result["reactions"] == nil {
		t.Error("JSON should contain reactions")
	}
}

func TestIssueHelpText(t *testing.T) {
	resetIssueFlags()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"issue", "--help"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("issue --help returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "list") {
		t.Error("help should mention list subcommand")
	}
	if !strings.Contains(out, "show") {
		t.Error("help should mention show subcommand")
	}
}

// --- test helpers ---

func setupIssueTestEnv(t *testing.T, ms *testutil.MockServer) {
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

	// Pre-populate pipeline cache for list tests
	_ = cache.Set(resolve.PipelineCacheKey("ws-123"), []resolve.CachedPipeline{
		{ID: "p1", Name: "New Issues"},
		{ID: "p2", Name: "In Development"},
		{ID: "p3", Name: "Done"},
	})
}

func repoResolutionResponse() map[string]any {
	return map[string]any{
		"data": map[string]any{
			"workspace": map[string]any{
				"repositoriesConnection": map[string]any{
					"totalCount": 1,
					"pageInfo": map[string]any{
						"hasNextPage": false,
						"endCursor":   "",
					},
					"nodes": []any{
						map[string]any{
							"id":        "r1",
							"ghId":      12345,
							"name":      "task-tracker",
							"ownerName": "dlakehammond",
						},
					},
				},
			},
		},
	}
}

func issueByInfoResolutionResponse() map[string]any {
	return map[string]any{
		"data": map[string]any{
			"issueByInfo": map[string]any{
				"id":     "i1",
				"number": 1,
				"repository": map[string]any{
					"ghId":      12345,
					"name":      "task-tracker",
					"ownerName": "dlakehammond",
				},
			},
		},
	}
}

func issueListByPipelineResponse() map[string]any {
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
						"htmlUrl":     "https://github.com/dlakehammond/task-tracker/issues/1",
						"pullRequest": false,
						"estimate":    map[string]any{"value": 3},
						"repository": map[string]any{
							"id":        "r1",
							"ghId":      12345,
							"name":      "task-tracker",
							"ownerName": "dlakehammond",
						},
						"assignees": map[string]any{
							"nodes": []any{
								map[string]any{"login": "dlakehammond"},
							},
						},
						"labels": map[string]any{
							"nodes": []any{
								map[string]any{"name": "bug", "color": "d73a4a"},
							},
						},
						"sprints": map[string]any{
							"nodes": []any{},
						},
						"pipelineIssue": map[string]any{
							"pipeline": map[string]any{"id": "p2", "name": "In Development"},
							"priority": map[string]any{"name": "High priority", "color": "#f00"},
						},
					},
					map[string]any{
						"id":          "i2",
						"number":      2,
						"title":       "Add error handling to API client",
						"state":       "OPEN",
						"htmlUrl":     "https://github.com/dlakehammond/task-tracker/issues/2",
						"pullRequest": false,
						"estimate":    nil,
						"repository": map[string]any{
							"id":        "r1",
							"ghId":      12345,
							"name":      "task-tracker",
							"ownerName": "dlakehammond",
						},
						"assignees":     map[string]any{"nodes": []any{}},
						"labels":        map[string]any{"nodes": []any{}},
						"sprints":       map[string]any{"nodes": []any{}},
						"pipelineIssue": nil,
					},
				},
			},
		},
	}
}

func issueListEmptyResponse() map[string]any {
	return map[string]any{
		"data": map[string]any{
			"searchIssuesByPipeline": map[string]any{
				"totalCount": 0,
				"pageInfo": map[string]any{
					"hasNextPage": false,
					"endCursor":   "",
				},
				"nodes": []any{},
			},
		},
	}
}

func issueListClosedResponse() map[string]any {
	return map[string]any{
		"data": map[string]any{
			"searchClosedIssues": map[string]any{
				"totalCount": 1,
				"pageInfo": map[string]any{
					"hasNextPage": false,
					"endCursor":   "",
				},
				"nodes": []any{
					map[string]any{
						"id":          "i10",
						"number":      10,
						"title":       "Old closed issue",
						"state":       "CLOSED",
						"htmlUrl":     "https://github.com/dlakehammond/task-tracker/issues/10",
						"pullRequest": false,
						"estimate":    nil,
						"repository": map[string]any{
							"id":        "r1",
							"ghId":      12345,
							"name":      "task-tracker",
							"ownerName": "dlakehammond",
						},
						"assignees":     map[string]any{"nodes": []any{}},
						"labels":        map[string]any{"nodes": []any{}},
						"sprints":       map[string]any{"nodes": []any{}},
						"pipelineIssue": nil,
					},
				},
			},
		},
	}
}

func issueShowResponse() map[string]any {
	return map[string]any{
		"data": map[string]any{
			"issueByInfo": map[string]any{
				"id":          "i1",
				"number":      1,
				"title":       "Fix login button alignment",
				"body":        "The login button is misaligned on mobile.",
				"state":       "OPEN",
				"pullRequest": false,
				"htmlUrl":     "https://github.com/dlakehammond/task-tracker/issues/1",
				"zenhubUrl":   "https://app.zenhub.com/workspaces/ws-123/issues/gh/dlakehammond/task-tracker/1",
				"createdAt":   "2026-01-15T10:00:00Z",
				"closedAt":    nil,
				"estimate":    map[string]any{"value": 3},
				"pipelineIssue": map[string]any{
					"pipeline":           map[string]any{"id": "p2", "name": "In Development"},
					"priority":           map[string]any{"name": "High priority", "color": "#f00"},
					"latestTransferTime": "2026-01-20T14:00:00Z",
				},
				"assignees": map[string]any{
					"nodes": []any{
						map[string]any{"login": "dlakehammond", "name": "Doug"},
					},
				},
				"labels": map[string]any{
					"nodes": []any{
						map[string]any{"id": "l1", "name": "bug", "color": "d73a4a"},
					},
				},
				"connectedPrs":      map[string]any{"nodes": []any{}},
				"blockingIssues":    map[string]any{"nodes": []any{}},
				"blockedIssues":     map[string]any{"nodes": []any{}},
				"parentZenhubEpics": map[string]any{"nodes": []any{}},
				"sprints":           map[string]any{"nodes": []any{}},
				"repository": map[string]any{
					"id":   "r1",
					"ghId": 12345,
					"name": "task-tracker",
					"owner": map[string]any{
						"login": "dlakehammond",
					},
				},
				"milestone": nil,
			},
		},
	}
}

func issueShowWithBlockersResponse() map[string]any {
	resp := issueShowResponse()
	data := resp["data"].(map[string]any)
	issue := data["issueByInfo"].(map[string]any)
	issue["blockingIssues"] = map[string]any{
		"nodes": []any{
			map[string]any{
				"id":     "i5",
				"number": 5,
				"title":  "Prerequisite database migration",
				"state":  "OPEN",
				"repository": map[string]any{
					"name":  "task-tracker",
					"owner": map[string]any{"login": "dlakehammond"},
				},
			},
		},
	}
	return resp
}

func issueShowWithPRsResponse() map[string]any {
	resp := issueShowResponse()
	data := resp["data"].(map[string]any)
	issue := data["issueByInfo"].(map[string]any)
	issue["connectedPrs"] = map[string]any{
		"nodes": []any{
			map[string]any{
				"id":          "pr10",
				"number":      10,
				"title":       "Fix button alignment CSS",
				"state":       "OPEN",
				"htmlUrl":     "https://github.com/dlakehammond/task-tracker/pull/10",
				"pullRequest": true,
				"repository": map[string]any{
					"name":  "task-tracker",
					"owner": map[string]any{"login": "dlakehammond"},
				},
			},
		},
	}
	return resp
}

func issueShowPRResponse() map[string]any {
	resp := issueShowResponse()
	data := resp["data"].(map[string]any)
	issue := data["issueByInfo"].(map[string]any)
	issue["pullRequest"] = true
	issue["title"] = "Fix button alignment CSS"
	return resp
}

func issueShowGitHubResponse() map[string]any {
	return map[string]any{
		"data": map[string]any{
			"repository": map[string]any{
				"issueOrPullRequest": map[string]any{
					"author": map[string]any{"login": "testuser"},
					"reactionGroups": []any{
						map[string]any{
							"content":  "THUMBS_UP",
							"reactors": map[string]any{"totalCount": 5},
						},
						map[string]any{
							"content":  "HEART",
							"reactors": map[string]any{"totalCount": 2},
						},
						map[string]any{
							"content":  "CONFUSED",
							"reactors": map[string]any{"totalCount": 0},
						},
					},
				},
			},
		},
	}
}

func issueShowGitHubPRResponse() map[string]any {
	return map[string]any{
		"data": map[string]any{
			"repository": map[string]any{
				"issueOrPullRequest": map[string]any{
					"author":  map[string]any{"login": "testuser"},
					"isDraft": false,
					"merged":  false,
					"reactionGroups": []any{
						map[string]any{
							"content":  "THUMBS_UP",
							"reactors": map[string]any{"totalCount": 3},
						},
					},
					"reviews": map[string]any{
						"nodes": []any{
							map[string]any{
								"author": map[string]any{"login": "reviewer1"},
								"state":  "APPROVED",
							},
						},
					},
					"commits": map[string]any{
						"nodes": []any{
							map[string]any{
								"commit": map[string]any{
									"statusCheckRollup": map[string]any{
										"state": "SUCCESS",
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func setupIssueTestEnvWithGitHub(t *testing.T, ms *testutil.MockServer, ghMs *testutil.MockServer) {
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

	origGh := ghNewFunc
	ghNewFunc = func(method, token string, opts ...gh.Option) *gh.Client {
		return gh.New("pat", "test-token", append(opts, gh.WithEndpoint(ghMs.URL()))...)
	}
	t.Cleanup(func() { ghNewFunc = origGh })

	// Pre-populate pipeline cache for list tests
	_ = cache.Set(resolve.PipelineCacheKey("ws-123"), []resolve.CachedPipeline{
		{ID: "p1", Name: "New Issues"},
		{ID: "p2", Name: "In Development"},
		{ID: "p3", Name: "Done"},
	})
}
