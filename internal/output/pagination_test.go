package output

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestAddPaginationFlags(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	var limit int
	var all bool
	AddPaginationFlags(cmd, &limit, &all)

	// Verify defaults
	if limit != DefaultLimit {
		t.Errorf("default limit: got %d, want %d", limit, DefaultLimit)
	}
	if all {
		t.Error("default all: got true, want false")
	}

	// Verify flags are registered
	if f := cmd.Flags().Lookup("limit"); f == nil {
		t.Error("--limit flag not registered")
	}
	if f := cmd.Flags().Lookup("all"); f == nil {
		t.Error("--all flag not registered")
	}
}

func TestAddPaginationFlagsMutuallyExclusive(t *testing.T) {
	cmd := &cobra.Command{
		Use:  "test",
		RunE: func(cmd *cobra.Command, args []string) error { return nil },
	}
	var limit int
	var all bool
	AddPaginationFlags(cmd, &limit, &all)

	cmd.SetArgs([]string{"--limit=50", "--all"})
	err := cmd.Execute()
	if err == nil {
		t.Error("expected error when both --limit and --all are set")
	}
}

func TestEffectiveLimit(t *testing.T) {
	tests := []struct {
		name  string
		limit int
		all   bool
		want  int
	}{
		{"default", 100, false, 100},
		{"custom limit", 50, false, 50},
		{"all overrides limit", 100, true, 0},
		{"all with custom limit", 50, true, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EffectiveLimit(tt.limit, tt.all)
			if got != tt.want {
				t.Errorf("got %d, want %d", got, tt.want)
			}
		})
	}
}

func TestTruncate(t *testing.T) {
	items := []string{"a", "b", "c", "d", "e"}

	tests := []struct {
		name      string
		limit     int
		wantLen   int
		wantTrunc bool
	}{
		{"under limit", 10, 5, false},
		{"at limit", 5, 5, false},
		{"over limit", 3, 3, true},
		{"limit one", 1, 1, true},
		{"zero limit (unlimited)", 0, 5, false},
		{"negative limit (unlimited)", -1, 5, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, truncated := Truncate(items, tt.limit)
			if len(got) != tt.wantLen {
				t.Errorf("length: got %d, want %d", len(got), tt.wantLen)
			}
			if truncated != tt.wantTrunc {
				t.Errorf("truncated: got %v, want %v", truncated, tt.wantTrunc)
			}
		})
	}
}

func TestTruncateEmpty(t *testing.T) {
	got, truncated := Truncate([]int{}, 10)
	if len(got) != 0 {
		t.Errorf("expected empty slice, got length %d", len(got))
	}
	if truncated {
		t.Error("expected no truncation for empty slice")
	}
}
