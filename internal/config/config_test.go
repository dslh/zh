package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFromFile(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "zh")
	if err := os.MkdirAll(configPath, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configPath, "config.yml"), []byte(`
api_key: test-key-123
workspace: ws-456
github:
  method: pat
  token: ghp_test
aliases:
  pipelines:
    ip: "In Progress"
  epics:
    auth: "epic-id-789"
`), 0o600); err != nil {
		t.Fatal(err)
	}

	t.Setenv("XDG_CONFIG_HOME", dir)
	// Clear env vars that would override
	t.Setenv("ZH_API_KEY", "")
	t.Setenv("ZH_WORKSPACE", "")
	t.Setenv("ZH_GITHUB_TOKEN", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.APIKey != "test-key-123" {
		t.Errorf("APIKey = %q, want %q", cfg.APIKey, "test-key-123")
	}
	if cfg.Workspace != "ws-456" {
		t.Errorf("Workspace = %q, want %q", cfg.Workspace, "ws-456")
	}
	if cfg.GitHub.Method != "pat" {
		t.Errorf("GitHub.Method = %q, want %q", cfg.GitHub.Method, "pat")
	}
	if cfg.GitHub.Token != "ghp_test" {
		t.Errorf("GitHub.Token = %q, want %q", cfg.GitHub.Token, "ghp_test")
	}
	if cfg.Aliases.Pipelines["ip"] != "In Progress" {
		t.Errorf("Aliases.Pipelines[ip] = %q, want %q", cfg.Aliases.Pipelines["ip"], "In Progress")
	}
	if cfg.Aliases.Epics["auth"] != "epic-id-789" {
		t.Errorf("Aliases.Epics[auth] = %q, want %q", cfg.Aliases.Epics["auth"], "epic-id-789")
	}
}

func TestEnvVarsOverrideConfigFile(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "zh")
	if err := os.MkdirAll(configPath, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configPath, "config.yml"), []byte(`
api_key: file-key
workspace: file-workspace
github:
  method: none
`), 0o600); err != nil {
		t.Fatal(err)
	}

	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("ZH_API_KEY", "env-key")
	t.Setenv("ZH_WORKSPACE", "env-workspace")
	t.Setenv("ZH_GITHUB_TOKEN", "env-token")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.APIKey != "env-key" {
		t.Errorf("APIKey = %q, want %q (env should override file)", cfg.APIKey, "env-key")
	}
	if cfg.Workspace != "env-workspace" {
		t.Errorf("Workspace = %q, want %q (env should override file)", cfg.Workspace, "env-workspace")
	}
	if cfg.GitHub.Token != "env-token" {
		t.Errorf("GitHub.Token = %q, want %q (env should override file)", cfg.GitHub.Token, "env-token")
	}
}

func TestMissingConfigReturnsZeroValues(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("ZH_API_KEY", "")
	t.Setenv("ZH_WORKSPACE", "")
	t.Setenv("ZH_GITHUB_TOKEN", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.APIKey != "" {
		t.Errorf("APIKey = %q, want empty", cfg.APIKey)
	}
	if cfg.Workspace != "" {
		t.Errorf("Workspace = %q, want empty", cfg.Workspace)
	}
	if cfg.GitHub.Method != "none" {
		t.Errorf("GitHub.Method = %q, want %q", cfg.GitHub.Method, "none")
	}
}

func TestWriteAndReadBack(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("ZH_API_KEY", "")
	t.Setenv("ZH_WORKSPACE", "")
	t.Setenv("ZH_GITHUB_TOKEN", "")

	original := &Config{
		APIKey:    "written-key",
		Workspace: "written-ws",
		GitHub: GitHubConfig{
			Method: "gh",
		},
		Aliases: AliasConfig{
			Pipelines: map[string]string{"dev": "In Development"},
			Epics:     map[string]string{},
		},
	}

	if err := Write(original); err != nil {
		t.Fatalf("Write() error: %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() after Write() error: %v", err)
	}

	if cfg.APIKey != original.APIKey {
		t.Errorf("APIKey = %q, want %q", cfg.APIKey, original.APIKey)
	}
	if cfg.Workspace != original.Workspace {
		t.Errorf("Workspace = %q, want %q", cfg.Workspace, original.Workspace)
	}
	if cfg.GitHub.Method != original.GitHub.Method {
		t.Errorf("GitHub.Method = %q, want %q", cfg.GitHub.Method, original.GitHub.Method)
	}
	if cfg.Aliases.Pipelines["dev"] != "In Development" {
		t.Errorf("Aliases.Pipelines[dev] = %q, want %q", cfg.Aliases.Pipelines["dev"], "In Development")
	}
}
