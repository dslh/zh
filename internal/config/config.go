// Package config manages zh configuration using Viper and XDG base directories.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// GitHubConfig holds GitHub access configuration.
type GitHubConfig struct {
	Method string `mapstructure:"method"` // "gh", "pat", or "none"
	Token  string `mapstructure:"token"`  // only when method=pat
}

// AliasConfig holds entity aliases.
type AliasConfig struct {
	Pipelines map[string]string `mapstructure:"pipelines"`
	Epics     map[string]string `mapstructure:"epics"`
}

// Config holds the complete zh configuration.
type Config struct {
	APIKey     string       `mapstructure:"api_key"`
	RESTAPIKey string       `mapstructure:"rest_api_key"`
	Workspace  string       `mapstructure:"workspace"`
	GitHub     GitHubConfig `mapstructure:"github"`
	Aliases    AliasConfig  `mapstructure:"aliases"`
}

var v *viper.Viper

// Load reads the configuration from the config file and environment variables.
// Environment variables take precedence over config file values.
func Load() (*Config, error) {
	v = viper.New()

	configDir := configDir()

	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(configDir)

	// Bind environment variables (take precedence over config file)
	v.SetEnvPrefix("")
	_ = v.BindEnv("api_key", "ZH_API_KEY")
	_ = v.BindEnv("rest_api_key", "ZH_REST_API_KEY")
	_ = v.BindEnv("workspace", "ZH_WORKSPACE")
	_ = v.BindEnv("github.token", "ZH_GITHUB_TOKEN")

	// Set defaults
	v.SetDefault("github.method", "none")
	v.SetDefault("aliases.pipelines", map[string]string{})
	v.SetDefault("aliases.epics", map[string]string{})

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("reading config: %w", err)
		}
		// Config file not found is OK — values come from env or defaults.
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	// If ZH_GITHUB_TOKEN is set and method isn't explicitly configured, assume PAT.
	if cfg.GitHub.Token != "" && !v.IsSet("github.method") {
		cfg.GitHub.Method = "pat"
	}

	return &cfg, nil
}

// Write persists the given config to the config file.
func Write(cfg *Config) error {
	dir := configDir()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	v := viper.New()
	v.SetConfigType("yaml")

	v.Set("api_key", cfg.APIKey)
	if cfg.RESTAPIKey != "" {
		v.Set("rest_api_key", cfg.RESTAPIKey)
	}
	v.Set("workspace", cfg.Workspace)
	v.Set("github.method", cfg.GitHub.Method)
	if cfg.GitHub.Token != "" {
		v.Set("github.token", cfg.GitHub.Token)
	}
	v.Set("aliases.pipelines", cfg.Aliases.Pipelines)
	v.Set("aliases.epics", cfg.Aliases.Epics)

	path := filepath.Join(dir, "config.yml")
	return v.WriteConfigAs(path)
}

// configDir returns the XDG-compliant config directory for zh.
func configDir() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "zh")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "zh")
}

// Dir returns the config directory path (for use by other packages).
func Dir() string {
	return configDir()
}
