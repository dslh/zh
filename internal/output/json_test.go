package output

import (
	"bytes"
	"strings"
	"testing"
)

func TestJSON(t *testing.T) {
	var buf bytes.Buffer
	data := map[string]any{
		"name":  "Sprint 42",
		"state": "active",
		"count": 12,
	}
	err := JSON(&buf, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := buf.String()
	if !strings.Contains(got, `"name": "Sprint 42"`) {
		t.Errorf("expected JSON to contain name field, got:\n%s", got)
	}
	if !strings.Contains(got, `"state": "active"`) {
		t.Errorf("expected JSON to contain state field, got:\n%s", got)
	}
}

func TestJSONSlice(t *testing.T) {
	var buf bytes.Buffer
	data := []map[string]string{
		{"name": "a"},
		{"name": "b"},
	}
	err := JSON(&buf, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := buf.String()
	if !strings.HasPrefix(got, "[") {
		t.Errorf("expected JSON array, got:\n%s", got)
	}
}

func TestIsJSON(t *testing.T) {
	if !IsJSON("json") {
		t.Error("expected IsJSON(\"json\") = true")
	}
	if IsJSON("") {
		t.Error("expected IsJSON(\"\") = false")
	}
	if IsJSON("text") {
		t.Error("expected IsJSON(\"text\") = false")
	}
}
