package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dslh/zh/internal/api"
	"github.com/dslh/zh/internal/cache"
	"github.com/dslh/zh/internal/config"
	"github.com/dslh/zh/internal/resolve"
	"github.com/dslh/zh/internal/testutil"
)

// setupMutationTest configures env, mock server, and returns a cleanup function.
func setupMutationTest(t *testing.T, ms *testutil.MockServer) {
	t.Helper()

	resetPipelineFlags()
	resetPipelineMutationFlags()

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

// --- pipeline create ---

func TestPipelineCreate(t *testing.T) {
	ms := testutil.NewMockServer(t)
	ms.HandleQuery("CreatePipeline", createPipelineResponse())
	setupMutationTest(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"pipeline", "create", "QA Review"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("pipeline create returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Created pipeline") {
		t.Errorf("output should confirm creation, got: %s", out)
	}
	if !strings.Contains(out, "QA Review") {
		t.Errorf("output should contain pipeline name, got: %s", out)
	}

	// Verify cache was invalidated
	_, ok := cache.Get[[]resolve.CachedPipeline](resolve.PipelineCacheKey("ws-123"))
	if ok {
		t.Error("pipeline cache should be cleared after create")
	}
}

func TestPipelineCreateWithFlags(t *testing.T) {
	ms := testutil.NewMockServer(t)
	ms.HandleQuery("CreatePipeline", createPipelineResponse())
	setupMutationTest(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"pipeline", "create", "QA Review", "--position=3", "--description=QA verification"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("pipeline create with flags returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "at position 3") {
		t.Errorf("output should mention position, got: %s", out)
	}
}

func TestPipelineCreateDryRun(t *testing.T) {
	ms := testutil.NewMockServer(t)
	setupMutationTest(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"pipeline", "create", "QA Review", "--position=3", "--description=QA verification", "--dry-run"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("pipeline create --dry-run returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Would create") {
		t.Errorf("dry-run should use 'Would' prefix, got: %s", out)
	}
	if !strings.Contains(out, "QA Review") {
		t.Errorf("dry-run should contain pipeline name, got: %s", out)
	}
	if !strings.Contains(out, "position 3") {
		t.Errorf("dry-run should mention position, got: %s", out)
	}
	if !strings.Contains(out, "QA verification") {
		t.Errorf("dry-run should mention description, got: %s", out)
	}
}

func TestPipelineCreateJSON(t *testing.T) {
	ms := testutil.NewMockServer(t)
	ms.HandleQuery("CreatePipeline", createPipelineResponse())
	setupMutationTest(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"pipeline", "create", "QA Review", "--output=json"})
	outputFormat = "json"
	defer func() { outputFormat = "" }()

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("pipeline create --output=json returned error: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\nOutput: %s", err, buf.String())
	}

	if result["name"] != "QA Review" {
		t.Errorf("JSON should contain name, got: %v", result)
	}
}

// --- pipeline edit ---

func TestPipelineEdit(t *testing.T) {
	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListPipelines", pipelineResolutionResponse())
	ms.HandleQuery("UpdatePipeline", updatePipelineResponse())
	setupMutationTest(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"pipeline", "edit", "In Development", "--name=Active Work"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("pipeline edit returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Updated pipeline") {
		t.Errorf("output should confirm update, got: %s", out)
	}
}

func TestPipelineEditPosition(t *testing.T) {
	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListPipelines", pipelineResolutionResponse())
	ms.HandleQuery("UpdatePipeline", updatePipelineResponse())
	setupMutationTest(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"pipeline", "edit", "In Development", "--position=0"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("pipeline edit --position returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Updated pipeline") {
		t.Errorf("output should confirm update, got: %s", out)
	}
}

func TestPipelineEditNoFlags(t *testing.T) {
	ms := testutil.NewMockServer(t)
	setupMutationTest(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"pipeline", "edit", "In Development"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("pipeline edit with no flags should error")
	}
	if !strings.Contains(err.Error(), "no changes specified") {
		t.Errorf("error = %q, want mention of no changes", err.Error())
	}
}

func TestPipelineEditDryRun(t *testing.T) {
	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListPipelines", pipelineResolutionResponse())
	setupMutationTest(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"pipeline", "edit", "In Development", "--name=Active Work", "--dry-run"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("pipeline edit --dry-run returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Would update") {
		t.Errorf("dry-run should use 'Would' prefix, got: %s", out)
	}
	if !strings.Contains(out, "In Development") {
		t.Errorf("dry-run should show current name, got: %s", out)
	}
	if !strings.Contains(out, "Active Work") {
		t.Errorf("dry-run should show new name, got: %s", out)
	}
}

func TestPipelineEditJSON(t *testing.T) {
	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListPipelines", pipelineResolutionResponse())
	ms.HandleQuery("UpdatePipeline", updatePipelineResponse())
	setupMutationTest(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"pipeline", "edit", "In Development", "--name=Active Work", "--output=json"})
	outputFormat = "json"
	defer func() { outputFormat = "" }()

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("pipeline edit --output=json returned error: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\nOutput: %s", err, buf.String())
	}
}

// --- pipeline delete ---

func TestPipelineDelete(t *testing.T) {
	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListPipelines", pipelineResolutionResponse())
	ms.HandleQuery("GetPipelineDetails", pipelineDetailForDelete())
	ms.HandleQuery("DeletePipeline", deletePipelineResponse())
	setupMutationTest(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"pipeline", "delete", "New Issues", "--into=Done"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("pipeline delete returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Deleted pipeline") {
		t.Errorf("output should confirm deletion, got: %s", out)
	}
	if !strings.Contains(out, "New Issues") {
		t.Errorf("output should contain deleted pipeline name, got: %s", out)
	}

	// Verify cache was invalidated
	_, ok := cache.Get[[]resolve.CachedPipeline](resolve.PipelineCacheKey("ws-123"))
	if ok {
		t.Error("pipeline cache should be cleared after delete")
	}
}

func TestPipelineDeleteDryRun(t *testing.T) {
	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListPipelines", pipelineResolutionResponse())
	ms.HandleQuery("GetPipelineDetails", pipelineDetailForDelete())
	setupMutationTest(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"pipeline", "delete", "New Issues", "--into=Done", "--dry-run"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("pipeline delete --dry-run returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Would delete") {
		t.Errorf("dry-run should use 'Would' prefix, got: %s", out)
	}
	if !strings.Contains(out, "Issues to move: 12") {
		t.Errorf("dry-run should show issue count, got: %s", out)
	}
	if !strings.Contains(out, "Done") {
		t.Errorf("dry-run should show destination, got: %s", out)
	}
}

func TestPipelineDeleteSameTarget(t *testing.T) {
	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListPipelines", pipelineResolutionResponse())
	setupMutationTest(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"pipeline", "delete", "Done", "--into=Done"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("pipeline delete into itself should error")
	}
	if !strings.Contains(err.Error(), "cannot delete pipeline into itself") {
		t.Errorf("error = %q, want mention of cannot delete into itself", err.Error())
	}
}

func TestPipelineDeleteJSON(t *testing.T) {
	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListPipelines", pipelineResolutionResponse())
	ms.HandleQuery("GetPipelineDetails", pipelineDetailForDelete())
	ms.HandleQuery("DeletePipeline", deletePipelineResponse())
	setupMutationTest(t, ms)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"pipeline", "delete", "New Issues", "--into=Done", "--output=json"})
	outputFormat = "json"
	defer func() { outputFormat = "" }()

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("pipeline delete --output=json returned error: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\nOutput: %s", err, buf.String())
	}

	if result["deleted"] != "New Issues" {
		t.Errorf("JSON should contain deleted pipeline name, got: %v", result)
	}
}

// --- pipeline alias ---

func TestPipelineAlias(t *testing.T) {
	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListPipelines", pipelineResolutionResponse())
	setupMutationTest(t, ms)

	// Write a minimal config file
	configDir := os.Getenv("XDG_CONFIG_HOME")
	writeTestConfig(t, configDir)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"pipeline", "alias", "In Development", "dev"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("pipeline alias returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "dev") {
		t.Errorf("output should contain alias name, got: %s", out)
	}
	if !strings.Contains(out, "In Development") {
		t.Errorf("output should contain pipeline name, got: %s", out)
	}

	// Verify alias was written to config
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("loading config: %v", err)
	}
	if cfg.Aliases.Pipelines["dev"] != "In Development" {
		t.Errorf("alias should be saved in config, got: %v", cfg.Aliases.Pipelines)
	}
}

func TestPipelineAliasList(t *testing.T) {
	ms := testutil.NewMockServer(t)
	setupMutationTest(t, ms)

	configDir := os.Getenv("XDG_CONFIG_HOME")
	writeTestConfigWithAliases(t, configDir, map[string]string{
		"dev":  "In Development",
		"done": "Done",
	})

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"pipeline", "alias", "--list"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("pipeline alias --list returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "ALIAS") {
		t.Errorf("output should contain ALIAS header, got: %s", out)
	}
	if !strings.Contains(out, "dev") {
		t.Errorf("output should contain alias 'dev', got: %s", out)
	}
	if !strings.Contains(out, "done") {
		t.Errorf("output should contain alias 'done', got: %s", out)
	}
}

func TestPipelineAliasDelete(t *testing.T) {
	ms := testutil.NewMockServer(t)
	setupMutationTest(t, ms)

	configDir := os.Getenv("XDG_CONFIG_HOME")
	writeTestConfigWithAliases(t, configDir, map[string]string{
		"dev": "In Development",
	})

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"pipeline", "alias", "--delete", "dev"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("pipeline alias --delete returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Removed alias") {
		t.Errorf("output should confirm removal, got: %s", out)
	}

	// Verify alias was removed from config
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("loading config: %v", err)
	}
	if _, ok := cfg.Aliases.Pipelines["dev"]; ok {
		t.Error("alias should be removed from config")
	}
}

func TestPipelineAliasDeleteNotFound(t *testing.T) {
	ms := testutil.NewMockServer(t)
	setupMutationTest(t, ms)

	configDir := os.Getenv("XDG_CONFIG_HOME")
	writeTestConfig(t, configDir)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"pipeline", "alias", "--delete", "nonexistent"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("pipeline alias --delete should error for nonexistent alias")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error = %q, want mention of not found", err.Error())
	}
}

func TestPipelineAliasAlreadyExists(t *testing.T) {
	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListPipelines", pipelineResolutionResponse())
	setupMutationTest(t, ms)

	configDir := os.Getenv("XDG_CONFIG_HOME")
	writeTestConfigWithAliases(t, configDir, map[string]string{
		"dev": "In Development",
	})

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"pipeline", "alias", "Done", "dev"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("pipeline alias should error when alias already exists")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("error = %q, want mention of already exists", err.Error())
	}
}

func TestPipelineAliasUsedInResolution(t *testing.T) {
	ms := testutil.NewMockServer(t)
	ms.HandleQuery("ListPipelines", pipelineResolutionResponse())
	ms.HandleQuery("GetPipelineDetails", pipelineDetailResponseData())
	ms.HandleQuery("GetPipelineIssues", pipelineIssuesResponseData())
	setupMutationTest(t, ms)

	configDir := os.Getenv("XDG_CONFIG_HOME")
	writeTestConfigWithAliases(t, configDir, map[string]string{
		"dev": "In Development",
	})

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"pipeline", "show", "dev"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("pipeline show with alias returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "PIPELINE: In Development") {
		t.Errorf("alias should resolve to pipeline, got: %s", out)
	}
}

// --- helpers ---

func writeTestConfig(t *testing.T, configDir string) {
	t.Helper()
	zhDir := filepath.Join(configDir, "zh")
	if err := os.MkdirAll(zhDir, 0o700); err != nil {
		t.Fatalf("creating config dir: %v", err)
	}
	content := "api_key: test-key\nworkspace: ws-123\ngithub:\n  method: none\n"
	if err := os.WriteFile(filepath.Join(zhDir, "config.yml"), []byte(content), 0o600); err != nil {
		t.Fatalf("writing config: %v", err)
	}
}

func writeTestConfigWithAliases(t *testing.T, configDir string, aliases map[string]string) {
	t.Helper()
	zhDir := filepath.Join(configDir, "zh")
	if err := os.MkdirAll(zhDir, 0o700); err != nil {
		t.Fatalf("creating config dir: %v", err)
	}
	content := "api_key: test-key\nworkspace: ws-123\ngithub:\n  method: none\naliases:\n  pipelines:\n"
	for k, v := range aliases {
		content += "    " + k + ": " + v + "\n"
	}
	content += "  epics: {}\n"
	if err := os.WriteFile(filepath.Join(zhDir, "config.yml"), []byte(content), 0o600); err != nil {
		t.Fatalf("writing config: %v", err)
	}
}

func createPipelineResponse() map[string]any {
	return map[string]any{
		"data": map[string]any{
			"createPipeline": map[string]any{
				"pipeline": map[string]any{
					"id":          "p-new",
					"name":        "QA Review",
					"description": "QA verification",
					"stage":       nil,
					"createdAt":   "2026-02-10T12:00:00Z",
				},
			},
		},
	}
}

func updatePipelineResponse() map[string]any {
	return map[string]any{
		"data": map[string]any{
			"updatePipeline": map[string]any{
				"pipeline": map[string]any{
					"id":                  "p2",
					"name":                "Active Work",
					"description":         "Active development work",
					"stage":               "DEVELOPMENT",
					"isDefaultPRPipeline": true,
					"updatedAt":           "2026-02-10T12:00:00Z",
				},
			},
		},
	}
}

func pipelineDetailForDelete() map[string]any {
	return map[string]any{
		"data": map[string]any{
			"node": map[string]any{
				"id":     "p1",
				"name":   "New Issues",
				"issues": map[string]any{"totalCount": 12},
			},
		},
	}
}

func deletePipelineResponse() map[string]any {
	return map[string]any{
		"data": map[string]any{
			"deletePipeline": map[string]any{
				"clientMutationId": nil,
				"destinationPipeline": map[string]any{
					"id":     "p3",
					"name":   "Done",
					"issues": map[string]any{"totalCount": 54},
				},
			},
		},
	}
}
