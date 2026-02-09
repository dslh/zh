package output

import "testing"

func TestMissingValues(t *testing.T) {
	if TableMissing != "-" {
		t.Errorf("TableMissing = %q, want %q", TableMissing, "-")
	}
	if DetailMissing != "None" {
		t.Errorf("DetailMissing = %q, want %q", DetailMissing, "None")
	}
}
