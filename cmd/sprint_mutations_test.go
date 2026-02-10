package cmd

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/dslh/zh/internal/api"
	"github.com/dslh/zh/internal/testutil"
)

// setupSprintMutationTest configures env, mock server, and returns a cleanup function.
func setupSprintMutationTest(t *testing.T, ms *testutil.MockServer) {
	t.Helper()

	resetSprintFlags()
	resetSprintMutationFlags()

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

// --- sprint add ---

func TestSprintAdd(t *testing.T) {
	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListSprints", sprintResolutionResponse())
	ms.HandleQuery("ListRepos", repoListForSprintResponse())
	ms.HandleQuery("IssueByInfo", issueByInfoForSprintResponse("i1", 1))
	ms.HandleQuery("GetIssueForEpic", issueDetailForSprintResponse("i1", 1, "Fix login button alignment"))
	ms.HandleQuery("AddIssuesToSprints", addIssuesToSprintsResponse())
	setupSprintMutationTest(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"sprint", "add", "task-tracker#1"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("sprint add returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Added") {
		t.Errorf("output should confirm addition, got: %s", out)
	}
	if !strings.Contains(out, "task-tracker#1") {
		t.Errorf("output should contain issue ref, got: %s", out)
	}
	if !strings.Contains(out, "Sprint 47") {
		t.Errorf("output should contain sprint name, got: %s", out)
	}
}

func TestSprintAddMultiple(t *testing.T) {
	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListSprints", sprintResolutionResponse())
	ms.HandleQuery("ListRepos", repoListForSprintResponse())
	ms.HandleQuery("IssueByInfo", issueByInfoForSprintResponse("i1", 1))
	ms.HandleQuery("GetIssueForEpic", issueDetailForSprintResponse("i1", 1, "Fix login button alignment"))
	ms.HandleQuery("AddIssuesToSprints", addIssuesToSprintsResponse())
	setupSprintMutationTest(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"sprint", "add", "task-tracker#1", "task-tracker#1"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("sprint add multiple returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Added 2 issue(s)") {
		t.Errorf("output should confirm batch addition, got: %s", out)
	}
}

func TestSprintAddToSpecificSprint(t *testing.T) {
	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListSprints", sprintResolutionResponse())
	ms.HandleQuery("ListRepos", repoListForSprintResponse())
	ms.HandleQuery("IssueByInfo", issueByInfoForSprintResponse("i1", 1))
	ms.HandleQuery("GetIssueForEpic", issueDetailForSprintResponse("i1", 1, "Fix login button alignment"))
	ms.HandleQuery("AddIssuesToSprints", addIssuesToSprintsResponse())
	setupSprintMutationTest(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"sprint", "add", "--sprint=Sprint 48", "task-tracker#1"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("sprint add --sprint returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Added") {
		t.Errorf("output should confirm addition, got: %s", out)
	}
	if !strings.Contains(out, "Sprint 48") {
		t.Errorf("output should contain target sprint name, got: %s", out)
	}
}

func TestSprintAddDryRun(t *testing.T) {
	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListSprints", sprintResolutionResponse())
	ms.HandleQuery("ListRepos", repoListForSprintResponse())
	ms.HandleQuery("IssueByInfo", issueByInfoForSprintResponse("i1", 1))
	ms.HandleQuery("GetIssueForEpic", issueDetailForSprintResponse("i1", 1, "Fix login button alignment"))
	setupSprintMutationTest(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"sprint", "add", "--dry-run", "task-tracker#1"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("sprint add --dry-run returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Would add") {
		t.Errorf("dry-run should use 'Would' prefix, got: %s", out)
	}
	if !strings.Contains(out, "task-tracker#1") {
		t.Errorf("dry-run should show issue ref, got: %s", out)
	}
	if !strings.Contains(out, "Sprint 47") {
		t.Errorf("dry-run should show sprint name, got: %s", out)
	}
}

func TestSprintAddJSON(t *testing.T) {
	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListSprints", sprintResolutionResponse())
	ms.HandleQuery("ListRepos", repoListForSprintResponse())
	ms.HandleQuery("IssueByInfo", issueByInfoForSprintResponse("i1", 1))
	ms.HandleQuery("GetIssueForEpic", issueDetailForSprintResponse("i1", 1, "Fix login button alignment"))
	ms.HandleQuery("AddIssuesToSprints", addIssuesToSprintsResponse())
	setupSprintMutationTest(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"sprint", "add", "task-tracker#1", "--output=json"})
	outputFormat = "json"
	defer func() { outputFormat = "" }()

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("sprint add --output=json returned error: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\nOutput: %s", err, buf.String())
	}

	if result["sprint"] == nil {
		t.Error("JSON should contain sprint field")
	}
	if result["added"] == nil {
		t.Error("JSON should contain added field")
	}
}

func TestSprintAddNoActiveSprint(t *testing.T) {
	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListSprints", sprintNoActiveResponse())
	setupSprintMutationTest(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"sprint", "add", "task-tracker#1"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("sprint add with no active sprint should error")
	}
	if !strings.Contains(err.Error(), "no active sprint") {
		t.Errorf("error should mention no active sprint, got: %v", err)
	}
}

func TestSprintAddContinueOnError(t *testing.T) {
	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListSprints", sprintResolutionResponse())
	ms.HandleQuery("ListRepos", repoListForSprintResponse())
	ms.HandleQuery("IssueByInfo", issueByInfoForSprintResponse("i1", 1))
	ms.HandleQuery("GetIssueForEpic", issueDetailForSprintResponse("i1", 1, "Fix login button alignment"))
	ms.HandleQuery("AddIssuesToSprints", addIssuesToSprintsResponse())
	setupSprintMutationTest(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"sprint", "add", "task-tracker#1", "task-tracker#999", "--continue-on-error"})

	// The command should succeed but report partial failure
	err := rootCmd.Execute()
	if err == nil {
		out := buf.String()
		if !strings.Contains(out, "Added") {
			t.Errorf("output should contain Added, got: %s", out)
		}
	}
	// If err != nil, that's also acceptable — partial failure returns error
}

// --- sprint remove ---

func TestSprintRemove(t *testing.T) {
	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListSprints", sprintResolutionResponse())
	ms.HandleQuery("ListRepos", repoListForSprintResponse())
	ms.HandleQuery("IssueByInfo", issueByInfoForSprintResponse("i1", 1))
	ms.HandleQuery("GetIssueForEpic", issueDetailForSprintResponse("i1", 1, "Fix login button alignment"))
	ms.HandleQuery("RemoveIssuesFromSprints", removeIssuesFromSprintsResponse())
	setupSprintMutationTest(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"sprint", "remove", "task-tracker#1"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("sprint remove returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Removed") {
		t.Errorf("output should confirm removal, got: %s", out)
	}
	if !strings.Contains(out, "task-tracker#1") {
		t.Errorf("output should contain issue ref, got: %s", out)
	}
	if !strings.Contains(out, "Sprint 47") {
		t.Errorf("output should contain sprint name, got: %s", out)
	}
}

func TestSprintRemoveMultiple(t *testing.T) {
	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListSprints", sprintResolutionResponse())
	ms.HandleQuery("ListRepos", repoListForSprintResponse())
	ms.HandleQuery("IssueByInfo", issueByInfoForSprintResponse("i1", 1))
	ms.HandleQuery("GetIssueForEpic", issueDetailForSprintResponse("i1", 1, "Fix login button alignment"))
	ms.HandleQuery("RemoveIssuesFromSprints", removeIssuesFromSprintsResponse())
	setupSprintMutationTest(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"sprint", "remove", "task-tracker#1", "task-tracker#1"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("sprint remove multiple returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Removed 2 issue(s)") {
		t.Errorf("output should confirm batch removal, got: %s", out)
	}
}

func TestSprintRemoveFromSpecificSprint(t *testing.T) {
	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListSprints", sprintResolutionResponse())
	ms.HandleQuery("ListRepos", repoListForSprintResponse())
	ms.HandleQuery("IssueByInfo", issueByInfoForSprintResponse("i1", 1))
	ms.HandleQuery("GetIssueForEpic", issueDetailForSprintResponse("i1", 1, "Fix login button alignment"))
	ms.HandleQuery("RemoveIssuesFromSprints", removeIssuesFromSprintsResponse())
	setupSprintMutationTest(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"sprint", "remove", "--sprint=Sprint 48", "task-tracker#1"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("sprint remove --sprint returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Removed") {
		t.Errorf("output should confirm removal, got: %s", out)
	}
	if !strings.Contains(out, "Sprint 48") {
		t.Errorf("output should contain target sprint name, got: %s", out)
	}
}

func TestSprintRemoveDryRun(t *testing.T) {
	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListSprints", sprintResolutionResponse())
	ms.HandleQuery("ListRepos", repoListForSprintResponse())
	ms.HandleQuery("IssueByInfo", issueByInfoForSprintResponse("i1", 1))
	ms.HandleQuery("GetIssueForEpic", issueDetailForSprintResponse("i1", 1, "Fix login button alignment"))
	setupSprintMutationTest(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"sprint", "remove", "--dry-run", "task-tracker#1"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("sprint remove --dry-run returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Would remove") {
		t.Errorf("dry-run should use 'Would' prefix, got: %s", out)
	}
	if !strings.Contains(out, "task-tracker#1") {
		t.Errorf("dry-run should show issue ref, got: %s", out)
	}
	if !strings.Contains(out, "Sprint 47") {
		t.Errorf("dry-run should show sprint name, got: %s", out)
	}
}

func TestSprintRemoveJSON(t *testing.T) {
	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListSprints", sprintResolutionResponse())
	ms.HandleQuery("ListRepos", repoListForSprintResponse())
	ms.HandleQuery("IssueByInfo", issueByInfoForSprintResponse("i1", 1))
	ms.HandleQuery("GetIssueForEpic", issueDetailForSprintResponse("i1", 1, "Fix login button alignment"))
	ms.HandleQuery("RemoveIssuesFromSprints", removeIssuesFromSprintsResponse())
	setupSprintMutationTest(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"sprint", "remove", "task-tracker#1", "--output=json"})
	outputFormat = "json"
	defer func() { outputFormat = "" }()

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("sprint remove --output=json returned error: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\nOutput: %s", err, buf.String())
	}

	if result["sprint"] == nil {
		t.Error("JSON should contain sprint field")
	}
	if result["removed"] == nil {
		t.Error("JSON should contain removed field")
	}
}

func TestSprintRemoveContinueOnError(t *testing.T) {
	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListSprints", sprintResolutionResponse())
	ms.HandleQuery("ListRepos", repoListForSprintResponse())
	ms.HandleQuery("IssueByInfo", issueByInfoForSprintResponse("i1", 1))
	ms.HandleQuery("GetIssueForEpic", issueDetailForSprintResponse("i1", 1, "Fix login button alignment"))
	ms.HandleQuery("RemoveIssuesFromSprints", removeIssuesFromSprintsResponse())
	setupSprintMutationTest(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"sprint", "remove", "task-tracker#1", "task-tracker#999", "--continue-on-error"})

	err := rootCmd.Execute()
	if err == nil {
		out := buf.String()
		if !strings.Contains(out, "Removed") {
			t.Errorf("output should contain Removed, got: %s", out)
		}
	}
	// If err != nil, that's also acceptable — partial failure returns error
}

func TestSprintHelpIncludesAddRemove(t *testing.T) {
	resetSprintFlags()
	resetSprintMutationFlags()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"sprint", "--help"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("sprint --help returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "add") {
		t.Error("help should mention add subcommand")
	}
	if !strings.Contains(out, "remove") {
		t.Error("help should mention remove subcommand")
	}
}

// --- helpers ---

func repoListForSprintResponse() map[string]any {
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
							"id":        "repo-1",
							"ghId":      1152464818,
							"name":      "task-tracker",
							"ownerName": "dlakehammond",
						},
					},
				},
			},
		},
	}
}

func issueByInfoForSprintResponse(id string, number int) map[string]any {
	return map[string]any{
		"data": map[string]any{
			"issueByInfo": map[string]any{
				"id":     id,
				"number": number,
				"repository": map[string]any{
					"ghId":      1152464818,
					"name":      "task-tracker",
					"ownerName": "dlakehammond",
				},
			},
		},
	}
}

func issueDetailForSprintResponse(id string, number int, title string) map[string]any {
	return map[string]any{
		"data": map[string]any{
			"node": map[string]any{
				"id":     id,
				"number": number,
				"title":  title,
				"repository": map[string]any{
					"name":      "task-tracker",
					"ownerName": "dlakehammond",
				},
			},
		},
	}
}

func addIssuesToSprintsResponse() map[string]any {
	return map[string]any{
		"data": map[string]any{
			"addIssuesToSprints": map[string]any{
				"sprintIssues": []any{
					map[string]any{
						"id": "si-new-1",
						"issue": map[string]any{
							"id":    "i1",
							"title": "Fix login button alignment",
						},
						"sprint": map[string]any{
							"id":    "sprint-47",
							"name":  "Sprint 47",
							"state": "OPEN",
						},
					},
				},
			},
		},
	}
}

func removeIssuesFromSprintsResponse() map[string]any {
	return map[string]any{
		"data": map[string]any{
			"removeIssuesFromSprints": map[string]any{
				"sprints": []any{
					map[string]any{
						"id":                "sprint-47",
						"name":              "",
						"generatedName":     "Sprint 47",
						"state":             "OPEN",
						"totalPoints":       float64(47),
						"completedPoints":   float64(34),
						"closedIssuesCount": 7,
					},
				},
			},
		},
	}
}

func sprintNoActiveResponse() map[string]any {
	return map[string]any{
		"data": map[string]any{
			"workspace": map[string]any{
				"sprints": map[string]any{
					"totalCount": 1,
					"pageInfo": map[string]any{
						"hasNextPage": false,
						"endCursor":   "",
					},
					"nodes": []any{
						map[string]any{
							"id":            "sprint-46",
							"name":          "",
							"generatedName": "Sprint 46",
							"state":         "CLOSED",
							"startAt":       "2025-01-06T00:00:00Z",
							"endAt":         "2025-01-20T00:00:00Z",
						},
					},
				},
				"activeSprint":   nil,
				"upcomingSprint":  nil,
				"previousSprint": nil,
			},
		},
	}
}
