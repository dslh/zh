package testutil

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestMockServerHandleQuery(t *testing.T) {
	ms := NewMockServer(t)

	type response struct {
		Data struct {
			Viewer struct {
				Login string `json:"login"`
			} `json:"viewer"`
		} `json:"data"`
	}

	ms.HandleQuery("viewer", map[string]any{
		"data": map[string]any{
			"viewer": map[string]any{
				"login": "testuser",
			},
		},
	})

	body := `{"query":"{ viewer { login } }"}`
	resp, err := http.Post(ms.URL(), "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	data, _ := io.ReadAll(resp.Body)
	var r response
	if err := json.Unmarshal(data, &r); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if r.Data.Viewer.Login != "testuser" {
		t.Errorf("expected login=testuser, got %q", r.Data.Viewer.Login)
	}
}

func TestMockServerNoMatch(t *testing.T) {
	ms := NewMockServer(t)

	body := `{"query":"{ something }"}`
	resp, err := http.Post(ms.URL(), "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 for unmatched request, got %d", resp.StatusCode)
	}
}
