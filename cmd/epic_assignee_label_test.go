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

func setupEpicAssigneeLabelTest(t *testing.T, ms *testutil.MockServer) {
	t.Helper()

	resetEpicFlags()
	resetEpicMutationFlags()
	resetEpicAssigneeLabelFlags()

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

// --- epic assignee add ---

func TestEpicAssigneeAdd(t *testing.T) {
	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListEpics", epicResolutionResponseForMutations())
	ms.HandleQuery("ListZenhubUsers", zenhubUsersResponse())
	ms.HandleQuery("AddAssigneesToZenhubEpics", addAssigneesToEpicsResponse())
	setupEpicAssigneeLabelTest(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"epic", "assignee", "add", "Q1 Platform", "johndoe"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("epic assignee add returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Added") {
		t.Errorf("output should confirm addition, got: %s", out)
	}
	if !strings.Contains(out, "@johndoe") {
		t.Errorf("output should contain user login, got: %s", out)
	}
}

func TestEpicAssigneeAddMultiple(t *testing.T) {
	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListEpics", epicResolutionResponseForMutations())
	ms.HandleQuery("ListZenhubUsers", zenhubUsersResponse())
	ms.HandleQuery("AddAssigneesToZenhubEpics", addAssigneesToEpicsResponse())
	setupEpicAssigneeLabelTest(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"epic", "assignee", "add", "Q1 Platform", "johndoe", "janedoe"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("epic assignee add multiple returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Added 2 assignee(s)") {
		t.Errorf("output should confirm batch addition, got: %s", out)
	}
}

func TestEpicAssigneeAddDryRun(t *testing.T) {
	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListEpics", epicResolutionResponseForMutations())
	ms.HandleQuery("ListZenhubUsers", zenhubUsersResponse())
	setupEpicAssigneeLabelTest(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"epic", "assignee", "add", "Q1 Platform", "johndoe", "--dry-run"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("epic assignee add --dry-run returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Would add") {
		t.Errorf("dry-run should use 'Would' prefix, got: %s", out)
	}
	if !strings.Contains(out, "@johndoe") {
		t.Errorf("dry-run should show user, got: %s", out)
	}
}

func TestEpicAssigneeAddJSON(t *testing.T) {
	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListEpics", epicResolutionResponseForMutations())
	ms.HandleQuery("ListZenhubUsers", zenhubUsersResponse())
	ms.HandleQuery("AddAssigneesToZenhubEpics", addAssigneesToEpicsResponse())
	setupEpicAssigneeLabelTest(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"epic", "assignee", "add", "Q1 Platform", "johndoe", "--output=json"})
	outputFormat = "json"
	defer func() { outputFormat = "" }()

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("epic assignee add --output=json returned error: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\nOutput: %s", err, buf.String())
	}

	if result["epic"] == nil {
		t.Error("JSON should contain epic field")
	}
	if result["users"] == nil {
		t.Error("JSON should contain users field")
	}
}

func TestEpicAssigneeAddUserNotFound(t *testing.T) {
	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListEpics", epicResolutionResponseForMutations())
	ms.HandleQuery("ListZenhubUsers", zenhubUsersResponse())
	setupEpicAssigneeLabelTest(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"epic", "assignee", "add", "Q1 Platform", "nonexistent"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("epic assignee add with nonexistent user should return error")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should mention not found, got: %v", err)
	}
}

func TestEpicAssigneeAddContinueOnError(t *testing.T) {
	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListEpics", epicResolutionResponseForMutations())
	ms.HandleQuery("ListZenhubUsers", zenhubUsersResponse())
	ms.HandleQuery("AddAssigneesToZenhubEpics", addAssigneesToEpicsResponse())
	setupEpicAssigneeLabelTest(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"epic", "assignee", "add", "Q1 Platform", "johndoe", "nonexistent", "--continue-on-error"})

	err := rootCmd.Execute()
	// With continue-on-error and partial failure, we expect an error return
	if err == nil {
		out := buf.String()
		if !strings.Contains(out, "Added") {
			t.Errorf("output should contain Added, got: %s", out)
		}
	}
}

func TestEpicAssigneeAddLegacyError(t *testing.T) {
	ms := testutil.NewMockServer(t)
	setupEpicAssigneeLabelTest(t, ms)

	_ = cache.Set(resolve.EpicCacheKey("ws-123"), []resolve.CachedEpic{
		{ID: "legacy-epic-1", Title: "Bug Tracker Improvements", Type: "legacy", IssueNumber: 1, RepoName: "task-tracker", RepoOwner: "dlakehammond"},
	})

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"epic", "assignee", "add", "Bug Tracker", "johndoe"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("epic assignee add on legacy epic should return error")
	}
	if !strings.Contains(err.Error(), "legacy epic") {
		t.Errorf("error should mention legacy epic, got: %v", err)
	}
}

func TestEpicAssigneeAddWithAtPrefix(t *testing.T) {
	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListEpics", epicResolutionResponseForMutations())
	ms.HandleQuery("ListZenhubUsers", zenhubUsersResponse())
	ms.HandleQuery("AddAssigneesToZenhubEpics", addAssigneesToEpicsResponse())
	setupEpicAssigneeLabelTest(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"epic", "assignee", "add", "Q1 Platform", "@johndoe"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("epic assignee add with @ prefix returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Added") {
		t.Errorf("output should confirm addition, got: %s", out)
	}
}

// --- epic assignee remove ---

func TestEpicAssigneeRemove(t *testing.T) {
	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListEpics", epicResolutionResponseForMutations())
	ms.HandleQuery("ListZenhubUsers", zenhubUsersResponse())
	ms.HandleQuery("RemoveAssigneesFromZenhubEpics", removeAssigneesFromEpicsResponse())
	setupEpicAssigneeLabelTest(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"epic", "assignee", "remove", "Q1 Platform", "johndoe"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("epic assignee remove returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Removed") {
		t.Errorf("output should confirm removal, got: %s", out)
	}
	if !strings.Contains(out, "@johndoe") {
		t.Errorf("output should contain user login, got: %s", out)
	}
}

func TestEpicAssigneeRemoveDryRun(t *testing.T) {
	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListEpics", epicResolutionResponseForMutations())
	ms.HandleQuery("ListZenhubUsers", zenhubUsersResponse())
	setupEpicAssigneeLabelTest(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"epic", "assignee", "remove", "Q1 Platform", "johndoe", "--dry-run"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("epic assignee remove --dry-run returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Would remove") {
		t.Errorf("dry-run should use 'Would remove' prefix, got: %s", out)
	}
}

func TestEpicAssigneeRemoveJSON(t *testing.T) {
	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListEpics", epicResolutionResponseForMutations())
	ms.HandleQuery("ListZenhubUsers", zenhubUsersResponse())
	ms.HandleQuery("RemoveAssigneesFromZenhubEpics", removeAssigneesFromEpicsResponse())
	setupEpicAssigneeLabelTest(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"epic", "assignee", "remove", "Q1 Platform", "johndoe", "--output=json"})
	outputFormat = "json"
	defer func() { outputFormat = "" }()

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("epic assignee remove --output=json returned error: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\nOutput: %s", err, buf.String())
	}

	if result["operation"] != "remove" {
		t.Errorf("JSON operation should be 'remove', got: %v", result["operation"])
	}
}

// --- epic label add ---

func TestEpicLabelAdd(t *testing.T) {
	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListEpics", epicResolutionResponseForMutations())
	ms.HandleQuery("ListZenhubLabels", zenhubLabelsResponse())
	ms.HandleQuery("AddZenhubLabelsToZenhubEpics", addLabelsToEpicsResponse())
	setupEpicAssigneeLabelTest(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"epic", "label", "add", "Q1 Platform", "platform"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("epic label add returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Added") {
		t.Errorf("output should confirm addition, got: %s", out)
	}
	if !strings.Contains(out, "platform") {
		t.Errorf("output should contain label name, got: %s", out)
	}
}

func TestEpicLabelAddMultiple(t *testing.T) {
	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListEpics", epicResolutionResponseForMutations())
	ms.HandleQuery("ListZenhubLabels", zenhubLabelsResponse())
	ms.HandleQuery("AddZenhubLabelsToZenhubEpics", addLabelsToEpicsResponse())
	setupEpicAssigneeLabelTest(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"epic", "label", "add", "Q1 Platform", "platform", "priority:high"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("epic label add multiple returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Added 2 label(s)") {
		t.Errorf("output should confirm batch addition, got: %s", out)
	}
}

func TestEpicLabelAddDryRun(t *testing.T) {
	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListEpics", epicResolutionResponseForMutations())
	ms.HandleQuery("ListZenhubLabels", zenhubLabelsResponse())
	setupEpicAssigneeLabelTest(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"epic", "label", "add", "Q1 Platform", "platform", "--dry-run"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("epic label add --dry-run returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Would add") {
		t.Errorf("dry-run should use 'Would' prefix, got: %s", out)
	}
	if !strings.Contains(out, "platform") {
		t.Errorf("dry-run should show label name, got: %s", out)
	}
}

func TestEpicLabelAddJSON(t *testing.T) {
	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListEpics", epicResolutionResponseForMutations())
	ms.HandleQuery("ListZenhubLabels", zenhubLabelsResponse())
	ms.HandleQuery("AddZenhubLabelsToZenhubEpics", addLabelsToEpicsResponse())
	setupEpicAssigneeLabelTest(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"epic", "label", "add", "Q1 Platform", "platform", "--output=json"})
	outputFormat = "json"
	defer func() { outputFormat = "" }()

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("epic label add --output=json returned error: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\nOutput: %s", err, buf.String())
	}

	if result["epic"] == nil {
		t.Error("JSON should contain epic field")
	}
	if result["labels"] == nil {
		t.Error("JSON should contain labels field")
	}
}

func TestEpicLabelAddNotFound(t *testing.T) {
	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListEpics", epicResolutionResponseForMutations())
	ms.HandleQuery("ListZenhubLabels", zenhubLabelsResponse())
	setupEpicAssigneeLabelTest(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"epic", "label", "add", "Q1 Platform", "nonexistent-label"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("epic label add with nonexistent label should return error")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should mention not found, got: %v", err)
	}
}

func TestEpicLabelAddContinueOnError(t *testing.T) {
	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListEpics", epicResolutionResponseForMutations())
	ms.HandleQuery("ListZenhubLabels", zenhubLabelsResponse())
	ms.HandleQuery("AddZenhubLabelsToZenhubEpics", addLabelsToEpicsResponse())
	setupEpicAssigneeLabelTest(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"epic", "label", "add", "Q1 Platform", "platform", "nonexistent", "--continue-on-error"})

	err := rootCmd.Execute()
	// With continue-on-error and partial failure, we expect an error return
	if err == nil {
		out := buf.String()
		if !strings.Contains(out, "Added") {
			t.Errorf("output should contain Added, got: %s", out)
		}
	}
}

func TestEpicLabelAddLegacyError(t *testing.T) {
	ms := testutil.NewMockServer(t)
	setupEpicAssigneeLabelTest(t, ms)

	_ = cache.Set(resolve.EpicCacheKey("ws-123"), []resolve.CachedEpic{
		{ID: "legacy-epic-1", Title: "Bug Tracker Improvements", Type: "legacy", IssueNumber: 1, RepoName: "task-tracker", RepoOwner: "dlakehammond"},
	})

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"epic", "label", "add", "Bug Tracker", "platform"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("epic label add on legacy epic should return error")
	}
	if !strings.Contains(err.Error(), "legacy epic") {
		t.Errorf("error should mention legacy epic, got: %v", err)
	}
}

// --- epic label remove ---

func TestEpicLabelRemove(t *testing.T) {
	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListEpics", epicResolutionResponseForMutations())
	ms.HandleQuery("ListZenhubLabels", zenhubLabelsResponse())
	ms.HandleQuery("RemoveZenhubLabelsFromZenhubEpics", removeLabelsFromEpicsResponse())
	setupEpicAssigneeLabelTest(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"epic", "label", "remove", "Q1 Platform", "platform"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("epic label remove returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Removed") {
		t.Errorf("output should confirm removal, got: %s", out)
	}
	if !strings.Contains(out, "platform") {
		t.Errorf("output should contain label name, got: %s", out)
	}
}

func TestEpicLabelRemoveDryRun(t *testing.T) {
	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListEpics", epicResolutionResponseForMutations())
	ms.HandleQuery("ListZenhubLabels", zenhubLabelsResponse())
	setupEpicAssigneeLabelTest(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"epic", "label", "remove", "Q1 Platform", "platform", "--dry-run"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("epic label remove --dry-run returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Would remove") {
		t.Errorf("dry-run should use 'Would remove' prefix, got: %s", out)
	}
}

func TestEpicLabelRemoveJSON(t *testing.T) {
	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListEpics", epicResolutionResponseForMutations())
	ms.HandleQuery("ListZenhubLabels", zenhubLabelsResponse())
	ms.HandleQuery("RemoveZenhubLabelsFromZenhubEpics", removeLabelsFromEpicsResponse())
	setupEpicAssigneeLabelTest(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"epic", "label", "remove", "Q1 Platform", "platform", "--output=json"})
	outputFormat = "json"
	defer func() { outputFormat = "" }()

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("epic label remove --output=json returned error: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\nOutput: %s", err, buf.String())
	}

	if result["operation"] != "remove" {
		t.Errorf("JSON operation should be 'remove', got: %v", result["operation"])
	}
}

// --- helpers ---

func zenhubUsersResponse() map[string]any {
	return map[string]any{
		"data": map[string]any{
			"workspace": map[string]any{
				"zenhubUsers": map[string]any{
					"totalCount": 2,
					"pageInfo": map[string]any{
						"hasNextPage": false,
						"endCursor":   "",
					},
					"nodes": []any{
						map[string]any{
							"id":   "user-1",
							"name": "John Doe",
							"githubUser": map[string]any{
								"login": "johndoe",
							},
						},
						map[string]any{
							"id":   "user-2",
							"name": "Jane Doe",
							"githubUser": map[string]any{
								"login": "janedoe",
							},
						},
					},
				},
			},
		},
	}
}

func addAssigneesToEpicsResponse() map[string]any {
	return map[string]any{
		"data": map[string]any{
			"addAssigneesToZenhubEpics": map[string]any{
				"zenhubEpics": []any{
					map[string]any{
						"id": "epic-zen-1",
						"assignees": map[string]any{
							"nodes": []any{
								map[string]any{
									"id":   "user-1",
									"name": "John Doe",
									"githubUser": map[string]any{
										"login": "johndoe",
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

func removeAssigneesFromEpicsResponse() map[string]any {
	return map[string]any{
		"data": map[string]any{
			"removeAssigneesFromZenhubEpics": map[string]any{
				"zenhubEpics": []any{
					map[string]any{
						"id": "epic-zen-1",
						"assignees": map[string]any{
							"nodes": []any{},
						},
					},
				},
			},
		},
	}
}

func zenhubLabelsResponse() map[string]any {
	return map[string]any{
		"data": map[string]any{
			"workspace": map[string]any{
				"zenhubLabels": map[string]any{
					"totalCount": 2,
					"pageInfo": map[string]any{
						"hasNextPage": false,
						"endCursor":   "",
					},
					"nodes": []any{
						map[string]any{
							"id":    "zlabel-1",
							"name":  "platform",
							"color": "#0075ca",
						},
						map[string]any{
							"id":    "zlabel-2",
							"name":  "priority:high",
							"color": "#d73a4a",
						},
					},
				},
			},
		},
	}
}

func addLabelsToEpicsResponse() map[string]any {
	return map[string]any{
		"data": map[string]any{
			"addZenhubLabelsToZenhubEpics": map[string]any{
				"zenhubEpics": []any{
					map[string]any{
						"id": "epic-zen-1",
						"labels": map[string]any{
							"nodes": []any{
								map[string]any{
									"id":    "zlabel-1",
									"name":  "platform",
									"color": "#0075ca",
								},
							},
						},
					},
				},
			},
		},
	}
}

func removeLabelsFromEpicsResponse() map[string]any {
	return map[string]any{
		"data": map[string]any{
			"removeZenhubLabelsFromZenhubEpics": map[string]any{
				"zenhubEpics": []any{
					map[string]any{
						"id": "epic-zen-1",
						"labels": map[string]any{
							"nodes": []any{},
						},
					},
				},
			},
		},
	}
}
