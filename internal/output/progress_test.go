package output

import "testing"

func TestFormatProgress(t *testing.T) {
	tests := []struct {
		name      string
		completed int
		total     int
		want      string
	}{
		{
			"partial",
			34, 52,
			"34/52 completed (65%)  █████████████░░░░░░░",
		},
		{
			"complete",
			10, 10,
			"10/10 completed (100%)  ████████████████████",
		},
		{
			"zero progress",
			0, 20,
			"0/20 completed (0%)  ░░░░░░░░░░░░░░░░░░░░",
		},
		{
			"zero total",
			0, 0,
			"0/0 completed (0%)  ░░░░░░░░░░░░░░░░░░░░",
		},
		{
			"half",
			5, 10,
			"5/10 completed (50%)  ██████████░░░░░░░░░░",
		},
		{
			"nearly complete",
			19, 20,
			"19/20 completed (95%)  ███████████████████░",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatProgress(tt.completed, tt.total)
			if got != tt.want {
				t.Errorf("FormatProgress(%d, %d) =\n  %q\nwant\n  %q", tt.completed, tt.total, got, tt.want)
			}
		})
	}
}
