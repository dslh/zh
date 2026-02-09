package output

import "testing"

func TestIssueRefFormatterShortForm(t *testing.T) {
	f := NewIssueRefFormatter([]string{
		"acme/frontend",
		"acme/backend",
	})

	got := f.FormatRef("acme", "frontend", 42)
	want := "frontend#42"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestIssueRefFormatterLongForm(t *testing.T) {
	f := NewIssueRefFormatter([]string{
		"acme/app",
		"bigcorp/app",
		"acme/backend",
	})

	got := f.FormatRef("acme", "app", 123)
	want := "acme/app#123"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	got = f.FormatRef("bigcorp", "app", 456)
	want = "bigcorp/app#456"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestIssueRefFormatterMixed(t *testing.T) {
	f := NewIssueRefFormatter([]string{
		"acme/app",
		"bigcorp/app",
		"acme/backend",
	})

	// "app" is ambiguous → long form
	got := f.FormatRef("acme", "app", 1)
	if got != "acme/app#1" {
		t.Errorf("ambiguous repo: got %q, want %q", got, "acme/app#1")
	}

	// "backend" is unique → short form
	got = f.FormatRef("acme", "backend", 2)
	if got != "backend#2" {
		t.Errorf("unique repo: got %q, want %q", got, "backend#2")
	}
}

func TestIssueRefFormatterEmpty(t *testing.T) {
	f := NewIssueRefFormatter(nil)

	got := f.FormatRef("acme", "frontend", 1)
	want := "frontend#1"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestIssueRefFormatterSingleRepo(t *testing.T) {
	f := NewIssueRefFormatter([]string{"dlakehammond/task-tracker"})

	got := f.FormatRef("dlakehammond", "task-tracker", 5)
	want := "task-tracker#5"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
