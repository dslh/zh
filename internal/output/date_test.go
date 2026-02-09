package output

import (
	"testing"
	"time"
)

func TestFormatDate(t *testing.T) {
	tests := []struct {
		name string
		date time.Time
		want string
	}{
		{"standard", time.Date(2025, 1, 20, 0, 0, 0, 0, time.UTC), "Jan 20, 2025"},
		{"single digit day", time.Date(2025, 3, 5, 0, 0, 0, 0, time.UTC), "Mar 5, 2025"},
		{"december", time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC), "Dec 31, 2024"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatDate(tt.date)
			if got != tt.want {
				t.Errorf("FormatDate() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatDateRange(t *testing.T) {
	tests := []struct {
		name  string
		start time.Time
		end   time.Time
		want  string
	}{
		{
			"same month",
			time.Date(2025, 1, 20, 0, 0, 0, 0, time.UTC),
			time.Date(2025, 1, 31, 0, 0, 0, 0, time.UTC),
			"Jan 20 → 31, 2025",
		},
		{
			"cross month same year",
			time.Date(2025, 1, 20, 0, 0, 0, 0, time.UTC),
			time.Date(2025, 2, 2, 0, 0, 0, 0, time.UTC),
			"Jan 20 → Feb 2, 2025",
		},
		{
			"cross year",
			time.Date(2024, 12, 15, 0, 0, 0, 0, time.UTC),
			time.Date(2025, 1, 5, 0, 0, 0, 0, time.UTC),
			"Dec 15, 2024 → Jan 5, 2025",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatDateRange(tt.start, tt.end)
			if got != tt.want {
				t.Errorf("FormatDateRange() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatDateISO(t *testing.T) {
	date := time.Date(2025, 1, 20, 0, 0, 0, 0, time.UTC)
	got := FormatDateISO(date)
	want := "2025-01-20"
	if got != want {
		t.Errorf("FormatDateISO() = %q, want %q", got, want)
	}
}
