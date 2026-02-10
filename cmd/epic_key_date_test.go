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

func setupEpicKeyDateTest(t *testing.T, ms *testutil.MockServer) {
	t.Helper()

	resetEpicFlags()
	resetEpicMutationFlags()
	resetEpicAssigneeLabelFlags()
	resetEpicKeyDateFlags()

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

// --- epic key-date list ---

func TestEpicKeyDateList(t *testing.T) {
	ms := testutil.NewMockServer(t)
	handleEpicResolutionForMutations(ms)
	ms.HandleQuery("GetEpicKeyDates", epicKeyDatesResponse())
	setupEpicKeyDateTest(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"epic", "key-date", "list", "Q1 Platform"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("epic key-date list returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Beta Release") {
		t.Errorf("output should contain key date name, got: %s", out)
	}
	if !strings.Contains(out, "2026-04-01") {
		t.Errorf("output should contain key date date, got: %s", out)
	}
	if !strings.Contains(out, "Code Freeze") {
		t.Errorf("output should contain second key date name, got: %s", out)
	}
}

func TestEpicKeyDateListNone(t *testing.T) {
	ms := testutil.NewMockServer(t)
	handleEpicResolutionForMutations(ms)
	ms.HandleQuery("GetEpicKeyDates", epicKeyDatesEmptyResponse())
	setupEpicKeyDateTest(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"epic", "key-date", "list", "Q1 Platform"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("epic key-date list returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "No key dates") {
		t.Errorf("output should indicate no key dates, got: %s", out)
	}
}

func TestEpicKeyDateListJSON(t *testing.T) {
	ms := testutil.NewMockServer(t)
	handleEpicResolutionForMutations(ms)
	ms.HandleQuery("GetEpicKeyDates", epicKeyDatesResponse())
	setupEpicKeyDateTest(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"epic", "key-date", "list", "Q1 Platform", "--output=json"})
	outputFormat = "json"
	defer func() { outputFormat = "" }()

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("epic key-date list --output=json returned error: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\nOutput: %s", err, buf.String())
	}

	if result["epic"] == nil {
		t.Error("JSON should contain epic field")
	}
	if result["keyDates"] == nil {
		t.Error("JSON should contain keyDates field")
	}
}

func TestEpicKeyDateListLegacyError(t *testing.T) {
	ms := testutil.NewMockServer(t)
	setupEpicKeyDateTest(t, ms)

	_ = cache.Set(resolve.EpicCacheKey("ws-123"), []resolve.CachedEpic{
		{ID: "legacy-epic-1", Title: "Bug Tracker Improvements", Type: "legacy", IssueNumber: 1, RepoName: "task-tracker", RepoOwner: "dlakehammond"},
	})

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"epic", "key-date", "list", "Bug Tracker"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("epic key-date list on legacy epic should return error")
	}
	if !strings.Contains(err.Error(), "legacy epic") {
		t.Errorf("error should mention legacy epic, got: %v", err)
	}
}

// --- epic key-date add ---

func TestEpicKeyDateAdd(t *testing.T) {
	ms := testutil.NewMockServer(t)
	handleEpicResolutionForMutations(ms)
	ms.HandleQuery("CreateZenhubEpicKeyDate", createKeyDateResponse())
	setupEpicKeyDateTest(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"epic", "key-date", "add", "Q1 Platform", "Beta Release", "2026-04-01"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("epic key-date add returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Added key date") {
		t.Errorf("output should confirm addition, got: %s", out)
	}
	if !strings.Contains(out, "Beta Release") {
		t.Errorf("output should contain key date name, got: %s", out)
	}
	if !strings.Contains(out, "2026-04-01") {
		t.Errorf("output should contain date, got: %s", out)
	}
}

func TestEpicKeyDateAddDryRun(t *testing.T) {
	ms := testutil.NewMockServer(t)
	handleEpicResolutionForMutations(ms)
	setupEpicKeyDateTest(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"epic", "key-date", "add", "Q1 Platform", "Beta Release", "2026-04-01", "--dry-run"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("epic key-date add --dry-run returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Would add") {
		t.Errorf("dry-run should use 'Would' prefix, got: %s", out)
	}
	if !strings.Contains(out, "Beta Release") {
		t.Errorf("dry-run should show key date name, got: %s", out)
	}
}

func TestEpicKeyDateAddJSON(t *testing.T) {
	ms := testutil.NewMockServer(t)
	handleEpicResolutionForMutations(ms)
	ms.HandleQuery("CreateZenhubEpicKeyDate", createKeyDateResponse())
	setupEpicKeyDateTest(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"epic", "key-date", "add", "Q1 Platform", "Beta Release", "2026-04-01", "--output=json"})
	outputFormat = "json"
	defer func() { outputFormat = "" }()

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("epic key-date add --output=json returned error: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\nOutput: %s", err, buf.String())
	}

	if result["epic"] == nil {
		t.Error("JSON should contain epic field")
	}
	if result["keyDate"] == nil {
		t.Error("JSON should contain keyDate field")
	}
}

func TestEpicKeyDateAddInvalidDate(t *testing.T) {
	ms := testutil.NewMockServer(t)
	handleEpicResolutionForMutations(ms)
	setupEpicKeyDateTest(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"epic", "key-date", "add", "Q1 Platform", "Beta Release", "not-a-date"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("epic key-date add with invalid date should return error")
	}
	if !strings.Contains(err.Error(), "invalid date") {
		t.Errorf("error should mention invalid date, got: %v", err)
	}
}

func TestEpicKeyDateAddLegacyError(t *testing.T) {
	ms := testutil.NewMockServer(t)
	setupEpicKeyDateTest(t, ms)

	_ = cache.Set(resolve.EpicCacheKey("ws-123"), []resolve.CachedEpic{
		{ID: "legacy-epic-1", Title: "Bug Tracker Improvements", Type: "legacy", IssueNumber: 1, RepoName: "task-tracker", RepoOwner: "dlakehammond"},
	})

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"epic", "key-date", "add", "Bug Tracker", "Beta Release", "2026-04-01"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("epic key-date add on legacy epic should return error")
	}
	if !strings.Contains(err.Error(), "legacy epic") {
		t.Errorf("error should mention legacy epic, got: %v", err)
	}
}

// --- epic key-date remove ---

func TestEpicKeyDateRemove(t *testing.T) {
	ms := testutil.NewMockServer(t)
	handleEpicResolutionForMutations(ms)
	ms.HandleQuery("GetEpicKeyDates", epicKeyDatesResponse())
	ms.HandleQuery("DeleteZenhubEpicKeyDate", deleteKeyDateResponse())
	setupEpicKeyDateTest(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"epic", "key-date", "remove", "Q1 Platform", "Beta Release"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("epic key-date remove returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Removed key date") {
		t.Errorf("output should confirm removal, got: %s", out)
	}
	if !strings.Contains(out, "Beta Release") {
		t.Errorf("output should contain key date name, got: %s", out)
	}
}

func TestEpicKeyDateRemoveDryRun(t *testing.T) {
	ms := testutil.NewMockServer(t)
	handleEpicResolutionForMutations(ms)
	ms.HandleQuery("GetEpicKeyDates", epicKeyDatesResponse())
	setupEpicKeyDateTest(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"epic", "key-date", "remove", "Q1 Platform", "Beta Release", "--dry-run"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("epic key-date remove --dry-run returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Would remove") {
		t.Errorf("dry-run should use 'Would remove' prefix, got: %s", out)
	}
	if !strings.Contains(out, "Beta Release") {
		t.Errorf("dry-run should show key date name, got: %s", out)
	}
}

func TestEpicKeyDateRemoveJSON(t *testing.T) {
	ms := testutil.NewMockServer(t)
	handleEpicResolutionForMutations(ms)
	ms.HandleQuery("GetEpicKeyDates", epicKeyDatesResponse())
	ms.HandleQuery("DeleteZenhubEpicKeyDate", deleteKeyDateResponse())
	setupEpicKeyDateTest(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"epic", "key-date", "remove", "Q1 Platform", "Beta Release", "--output=json"})
	outputFormat = "json"
	defer func() { outputFormat = "" }()

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("epic key-date remove --output=json returned error: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\nOutput: %s", err, buf.String())
	}

	if result["operation"] != "remove" {
		t.Errorf("JSON operation should be 'remove', got: %v", result["operation"])
	}
}

func TestEpicKeyDateRemoveNotFound(t *testing.T) {
	ms := testutil.NewMockServer(t)
	handleEpicResolutionForMutations(ms)
	ms.HandleQuery("GetEpicKeyDates", epicKeyDatesResponse())
	setupEpicKeyDateTest(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"epic", "key-date", "remove", "Q1 Platform", "Nonexistent Date"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("epic key-date remove with nonexistent name should return error")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should mention not found, got: %v", err)
	}
}

func TestEpicKeyDateRemoveLegacyError(t *testing.T) {
	ms := testutil.NewMockServer(t)
	setupEpicKeyDateTest(t, ms)

	_ = cache.Set(resolve.EpicCacheKey("ws-123"), []resolve.CachedEpic{
		{ID: "legacy-epic-1", Title: "Bug Tracker Improvements", Type: "legacy", IssueNumber: 1, RepoName: "task-tracker", RepoOwner: "dlakehammond"},
	})

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"epic", "key-date", "remove", "Bug Tracker", "Beta Release"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("epic key-date remove on legacy epic should return error")
	}
	if !strings.Contains(err.Error(), "legacy epic") {
		t.Errorf("error should mention legacy epic, got: %v", err)
	}
}

// --- helpers ---

func epicKeyDatesResponse() map[string]any {
	return map[string]any{
		"data": map[string]any{
			"node": map[string]any{
				"id":    "epic-zen-1",
				"title": "Q1 Platform Improvements",
				"keyDates": map[string]any{
					"totalCount": 2,
					"nodes": []any{
						map[string]any{
							"id":          "kd-1",
							"date":        "2026-04-01",
							"description": "Beta Release",
							"color":       nil,
						},
						map[string]any{
							"id":          "kd-2",
							"date":        "2026-03-15",
							"description": "Code Freeze",
							"color":       nil,
						},
					},
				},
			},
		},
	}
}

func epicKeyDatesEmptyResponse() map[string]any {
	return map[string]any{
		"data": map[string]any{
			"node": map[string]any{
				"id":    "epic-zen-1",
				"title": "Q1 Platform Improvements",
				"keyDates": map[string]any{
					"totalCount": 0,
					"nodes":      []any{},
				},
			},
		},
	}
}

func createKeyDateResponse() map[string]any {
	return map[string]any{
		"data": map[string]any{
			"createZenhubEpicKeyDate": map[string]any{
				"keyDate": map[string]any{
					"id":          "kd-new",
					"date":        "2026-04-01",
					"description": "Beta Release",
					"color":       nil,
				},
				"zenhubEpic": map[string]any{
					"id":    "epic-zen-1",
					"title": "Q1 Platform Improvements",
				},
			},
		},
	}
}

func deleteKeyDateResponse() map[string]any {
	return map[string]any{
		"data": map[string]any{
			"deleteZenhubEpicKeyDate": map[string]any{
				"keyDate": map[string]any{
					"id":          "kd-1",
					"date":        "2026-04-01",
					"description": "Beta Release",
				},
				"zenhubEpic": map[string]any{
					"id":    "epic-zen-1",
					"title": "Q1 Platform Improvements",
				},
			},
		},
	}
}
