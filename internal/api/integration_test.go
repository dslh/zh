package api

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/dslh/zh/internal/config"
)

func TestIntegrationViewerQuery(t *testing.T) {
	if os.Getenv("ZH_INTEGRATION") == "" {
		t.Skip("set ZH_INTEGRATION=1 to run integration tests")
	}

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("config.Load() error: %v", err)
	}
	if cfg.APIKey == "" {
		t.Fatal("no API key configured")
	}

	client := New(cfg.APIKey)
	data, err := client.Execute("{ viewer { id } }", nil)
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	var result struct {
		Viewer struct {
			ID string `json:"id"`
		} `json:"viewer"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if result.Viewer.ID == "" {
		t.Error("viewer ID should not be empty")
	}
	t.Logf("viewer ID: %s", result.Viewer.ID)
}
