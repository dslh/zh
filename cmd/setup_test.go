package cmd

import (
	"bytes"
	"net/http"
	"strings"
	"testing"

	"github.com/dslh/zh/internal/api"
	"github.com/dslh/zh/internal/config"
	"github.com/dslh/zh/internal/testutil"
)

// --- needsSetup ---

func TestNeedsSetupNoConfig(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configDir)
	t.Setenv("ZH_API_KEY", "")
	t.Setenv("ZH_WORKSPACE", "")
	t.Setenv("ZH_GITHUB_TOKEN", "")

	if !needsSetup() {
		t.Error("needsSetup should return true when no config exists")
	}
}

func TestNeedsSetupWithAPIKey(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configDir)
	t.Setenv("ZH_API_KEY", "test-key")
	t.Setenv("ZH_WORKSPACE", "")
	t.Setenv("ZH_GITHUB_TOKEN", "")

	if needsSetup() {
		t.Error("needsSetup should return false when API key is set via env")
	}
}

// --- validateAPIKeyNonInteractive ---

func TestValidateAPIKeySuccess(t *testing.T) {
	ms := testutil.NewMockServer(t)
	ms.HandleQuery("zenhubOrganizations", workspacesForSetupResponse())

	origNew := apiNewFunc
	apiNewFunc = func(apiKey string, opts ...api.Option) *api.Client {
		return api.New(apiKey, append(opts, api.WithEndpoint(ms.URL()))...)
	}
	defer func() { apiNewFunc = origNew }()

	choices, err := validateAPIKeyNonInteractive("test-key")
	if err != nil {
		t.Fatalf("validateAPIKeyNonInteractive returned error: %v", err)
	}

	if len(choices) != 2 {
		t.Fatalf("expected 2 workspaces, got %d", len(choices))
	}

	if choices[0].name != "Development" {
		t.Errorf("first workspace name = %q, want Development", choices[0].name)
	}
	if choices[0].orgName != "TestOrg" {
		t.Errorf("first workspace org = %q, want TestOrg", choices[0].orgName)
	}
	if choices[1].name != "DevOps" {
		t.Errorf("second workspace name = %q, want DevOps", choices[1].name)
	}
}

func TestValidateAPIKeyAuthFailure(t *testing.T) {
	ms := testutil.NewMockServer(t)
	ms.Handle(
		func(req testutil.GraphQLRequest) bool { return true },
		func(w http.ResponseWriter, _ testutil.GraphQLRequest) {
			w.WriteHeader(401)
			_, _ = w.Write([]byte(`{"error": "unauthorized"}`))
		},
	)

	origNew := apiNewFunc
	apiNewFunc = func(apiKey string, opts ...api.Option) *api.Client {
		return api.New(apiKey, append(opts, api.WithEndpoint(ms.URL()))...)
	}
	defer func() { apiNewFunc = origNew }()

	_, err := validateAPIKeyNonInteractive("bad-key")
	if err == nil {
		t.Fatal("validateAPIKeyNonInteractive should return error for bad key")
	}
}

// --- setupModel logic ---

func TestSetupModelInit(t *testing.T) {
	m := newSetupModel()
	if m.step != stepAPIKey {
		t.Errorf("initial step = %d, want stepAPIKey (%d)", m.step, stepAPIKey)
	}
	if m.cancelled {
		t.Error("model should not be cancelled initially")
	}
}

func TestSetupModelGitHubChoices(t *testing.T) {
	m := newSetupModel()
	if len(m.githubChoices) != 3 {
		t.Fatalf("expected 3 github choices, got %d", len(m.githubChoices))
	}
	if m.githubChoices[0] != "gh" {
		t.Errorf("first github choice = %q, want gh", m.githubChoices[0])
	}
	if m.githubChoices[1] != "pat" {
		t.Errorf("second github choice = %q, want pat", m.githubChoices[1])
	}
	if m.githubChoices[2] != "none" {
		t.Errorf("third github choice = %q, want none", m.githubChoices[2])
	}
}

func TestGithubMethodLabels(t *testing.T) {
	tests := []struct {
		method string
		want   string
	}{
		{"gh", "gh CLI (recommended)"},
		{"pat", "Personal access token"},
		{"none", "No GitHub access"},
	}
	for _, tt := range tests {
		label := githubMethodLabel(tt.method)
		if label != tt.want {
			t.Errorf("githubMethodLabel(%q) = %q, want %q", tt.method, label, tt.want)
		}
	}
}

func TestGithubMethodDescriptions(t *testing.T) {
	for _, method := range []string{"gh", "pat", "none"} {
		desc := githubMethodDescription(method)
		if desc == "" {
			t.Errorf("githubMethodDescription(%q) should not be empty", method)
		}
	}

	noneDesc := githubMethodDescription("none")
	if !strings.Contains(noneDesc, "Legacy epic") {
		t.Error("none description should mention legacy epic limitations")
	}
	if !strings.Contains(noneDesc, "Branch name") {
		t.Error("none description should mention branch name resolution")
	}
}

// --- PersistentPreRunE skip behavior ---

func TestSetupSkipsVersion(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configDir)
	t.Setenv("ZH_API_KEY", "")
	t.Setenv("ZH_WORKSPACE", "")
	t.Setenv("ZH_GITHUB_TOKEN", "")

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"version"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("version should not error even without config: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "zh version") {
		t.Errorf("version should still work without config, got: %s", out)
	}
}

func TestSetupSkipsHelp(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configDir)
	t.Setenv("ZH_API_KEY", "")
	t.Setenv("ZH_WORKSPACE", "")
	t.Setenv("ZH_GITHUB_TOKEN", "")

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"--help"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("--help should not error even without config: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "zh") {
		t.Errorf("help should still work without config, got: %s", out)
	}
}

func TestSetupSkipsCacheClear(t *testing.T) {
	configDir := t.TempDir()
	cacheDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configDir)
	t.Setenv("XDG_CACHE_HOME", cacheDir)
	t.Setenv("ZH_API_KEY", "")
	t.Setenv("ZH_WORKSPACE", "")
	t.Setenv("ZH_GITHUB_TOKEN", "")

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"cache", "clear"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("cache clear should not error even without config: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Cleared") {
		t.Errorf("cache clear should still work without config, got: %s", out)
	}
}

// --- Config write ---

func TestSetupWritesConfig(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configDir)
	t.Setenv("ZH_API_KEY", "")
	t.Setenv("ZH_WORKSPACE", "")
	t.Setenv("ZH_GITHUB_TOKEN", "")

	cfg := &config.Config{
		APIKey:    "test-api-key",
		Workspace: "ws-123",
		GitHub: config.GitHubConfig{
			Method: "none",
		},
	}

	if err := config.Write(cfg); err != nil {
		t.Fatalf("config.Write failed: %v", err)
	}

	// Reload and verify
	loaded, err := config.Load()
	if err != nil {
		t.Fatalf("config.Load failed: %v", err)
	}

	if loaded.APIKey != "test-api-key" {
		t.Errorf("APIKey = %q, want test-api-key", loaded.APIKey)
	}
	if loaded.Workspace != "ws-123" {
		t.Errorf("Workspace = %q, want ws-123", loaded.Workspace)
	}
	if loaded.GitHub.Method != "none" {
		t.Errorf("GitHub.Method = %q, want none", loaded.GitHub.Method)
	}
}

// --- gh CLI check ---

func TestGhAuthCheckSuccess(t *testing.T) {
	origCheck := ghAuthCheckFunc
	ghAuthCheckFunc = func() error { return nil }
	defer func() { ghAuthCheckFunc = origCheck }()

	m := newSetupModel()
	msg := m.validateGhCLI()
	result, ok := msg.(githubValidatedMsg)
	if !ok {
		t.Fatalf("expected githubValidatedMsg, got %T", msg)
	}
	if result.err != nil {
		t.Errorf("expected no error, got: %v", result.err)
	}
}

func TestGhAuthCheckFailure(t *testing.T) {
	origCheck := ghAuthCheckFunc
	ghAuthCheckFunc = func() error { return &execError{} }
	defer func() { ghAuthCheckFunc = origCheck }()

	m := newSetupModel()
	msg := m.validateGhCLI()
	result, ok := msg.(githubValidatedMsg)
	if !ok {
		t.Fatalf("expected githubValidatedMsg, got %T", msg)
	}
	if result.err == nil {
		t.Error("expected error for failed gh auth check")
	}
	if !strings.Contains(result.err.Error(), "gh auth status") {
		t.Errorf("error should mention gh auth status, got: %v", result.err)
	}
}

// --- workspace choice ---

func TestWorkspaceChoiceFilterValue(t *testing.T) {
	ws := workspaceChoice{
		id:      "ws1",
		name:    "Development",
		orgName: "TestOrg",
	}

	filter := ws.FilterValue()
	if !strings.Contains(filter, "Development") {
		t.Error("filter value should contain workspace name")
	}
	if !strings.Contains(filter, "TestOrg") {
		t.Error("filter value should contain org name")
	}
}

// --- setup help text ---

func TestSetupHelpText(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"setup", "--help"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("setup --help returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "setup wizard") {
		t.Error("setup help should mention setup wizard")
	}
}

// --- Non-interactive detection ---

func TestSetupNonInteractiveReturnsError(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configDir)
	t.Setenv("ZH_API_KEY", "")
	t.Setenv("ZH_WORKSPACE", "")
	t.Setenv("ZH_GITHUB_TOKEN", "")

	// Override isInteractive to simulate non-TTY
	origInteractive := isInteractive
	isInteractive = func() bool { return false }
	defer func() { isInteractive = origInteractive }()

	// Call runSetup directly to avoid Cobra state issues between tests.
	err := runSetup(setupCmd, nil)
	if err == nil {
		t.Fatal("setup should error in non-interactive environment")
	}
	if !strings.Contains(err.Error(), "interactive terminal") {
		t.Errorf("error should mention interactive terminal, got: %v", err)
	}
}

func TestSetupPreRunNonInteractive(t *testing.T) {
	configDir := t.TempDir()
	cacheDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configDir)
	t.Setenv("XDG_CACHE_HOME", cacheDir)
	t.Setenv("ZH_API_KEY", "")
	t.Setenv("ZH_WORKSPACE", "")
	t.Setenv("ZH_GITHUB_TOKEN", "")

	// Override isInteractive to simulate non-TTY
	origInteractive := isInteractive
	isInteractive = func() bool { return false }
	defer func() { isInteractive = origInteractive }()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"workspace", "list"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("workspace list should error without config in non-interactive mode")
	}
	if !strings.Contains(err.Error(), "no API key") {
		t.Errorf("error should mention no API key, got: %v", err)
	}
}

// --- helpers ---

type execError struct{}

func (e *execError) Error() string { return "exit status 1" }

func workspacesForSetupResponse() map[string]any {
	return map[string]any{
		"data": map[string]any{
			"viewer": map[string]any{
				"zenhubOrganizations": map[string]any{
					"nodes": []any{
						map[string]any{
							"id":   "org1",
							"name": "TestOrg",
							"workspaces": map[string]any{
								"nodes": []any{
									map[string]any{
										"id":               "ws1",
										"name":             "Development",
										"displayName":      "Development",
										"viewerPermission": "ADMIN",
										"repositoriesConnection": map[string]any{"totalCount": 5},
										"pipelinesConnection":    map[string]any{"totalCount": 3},
									},
									map[string]any{
										"id":               "ws2",
										"name":             "DevOps",
										"displayName":      "DevOps",
										"viewerPermission": "WRITE",
										"repositoriesConnection": map[string]any{"totalCount": 2},
										"pipelinesConnection":    map[string]any{"totalCount": 4},
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
