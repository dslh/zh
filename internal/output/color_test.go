package output

import (
	"testing"
)

func TestColorFunctionsNoColor(t *testing.T) {
	// Tests run in a non-TTY environment, so color should be disabled.
	// Verify that color functions return the input unchanged.
	tests := []struct {
		name string
		fn   func(string) string
	}{
		{"Green", Green},
		{"Red", Red},
		{"Yellow", Yellow},
		{"Cyan", Cyan},
		{"Dim", Dim},
		{"Bold", Bold},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.fn("hello")
			if got != "hello" {
				t.Errorf("expected %q, got %q", "hello", got)
			}
		})
	}
}

func TestColorFunctionsNoColorEnv(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	got := Green("hello")
	if got != "hello" {
		t.Errorf("expected %q with NO_COLOR set, got %q", "hello", got)
	}
}

func TestFormattedColorFunctions(t *testing.T) {
	// In non-TTY, format functions should return plain formatted text.
	tests := []struct {
		name   string
		fn     func(string, ...any) string
		format string
		args   []any
		want   string
	}{
		{"Greenf", Greenf, "count: %d", []any{5}, "count: 5"},
		{"Redf", Redf, "%s failed", []any{"task"}, "task failed"},
		{"Yellowf", Yellowf, "warning: %s", []any{"dry-run"}, "warning: dry-run"},
		{"Cyanf", Cyanf, "id=%s", []any{"abc"}, "id=abc"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.fn(tt.format, tt.args...)
			if got != tt.want {
				t.Errorf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

func TestColorEnabledRespectsNoColor(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	if colorEnabled() {
		t.Error("expected colorEnabled()=false when NO_COLOR is set (even empty)")
	}
}
