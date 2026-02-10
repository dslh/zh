package output

import (
	"bytes"
	"testing"

	"github.com/dslh/zh/internal/testutil"
)

func TestMutationSingle(t *testing.T) {
	var buf bytes.Buffer
	MutationSingle(&buf, "Set estimate on mpt#1234 to 5")
	got := buf.String()
	want := "Set estimate on mpt#1234 to 5\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestMutationBatch(t *testing.T) {
	var buf bytes.Buffer
	MutationBatch(&buf, `Moved 3 issues to "In Development":`, []MutationItem{
		{Ref: "mpt#1234", Title: "Fix login button alignment"},
		{Ref: "mpt#1235", Title: "Update error messages"},
		{Ref: "api#567", Title: "Add rate limiting headers"},
	})
	testutil.AssertSnapshot(t, "mutation-batch.txt", buf.String())
}

func TestMutationPartialFailure(t *testing.T) {
	var buf bytes.Buffer
	MutationPartialFailure(&buf, "Closed 2 of 3 issues:",
		[]MutationItem{
			{Ref: "mpt#1234", Title: "Fix login button alignment"},
			{Ref: "mpt#1235", Title: "Update error messages"},
		},
		[]FailedItem{
			{Ref: "api#568", Reason: "Permission denied"},
		},
	)
	testutil.AssertSnapshot(t, "mutation-partial-failure.txt", buf.String())
}

func TestMutationDryRun(t *testing.T) {
	var buf bytes.Buffer
	MutationDryRun(&buf, `Would move 2 issues to "In Development":`, []MutationItem{
		{Ref: "mpt#2451", Title: "Lock browser version...", Context: `(currently in "Backlog")`},
		{Ref: "api#1662", Title: "Synchronize posting ids...", Context: `(currently in "New Issues")`},
	})
	testutil.AssertSnapshot(t, "mutation-dry-run.txt", buf.String())
}

func TestMutationDryRunDetail(t *testing.T) {
	var buf bytes.Buffer
	MutationDryRunDetail(&buf, `Would create pipeline "QA Review" at position 3.`, []DetailLine{
		{Key: "Description", Value: "QA verification"},
	})
	testutil.AssertSnapshot(t, "mutation-dry-run-detail.txt", buf.String())
}

func TestMutationDryRunDetailMultiple(t *testing.T) {
	var buf bytes.Buffer
	MutationDryRunDetail(&buf, `Would update pipeline "In Development":`, []DetailLine{
		{Key: "Name", Value: "In Development -> Active Work"},
		{Key: "Position", Value: "-> 3"},
		{Key: "Description", Value: "-> QA verification"},
	})
	testutil.AssertSnapshot(t, "mutation-dry-run-detail-multi.txt", buf.String())
}

func TestMutationDryRunDetailNoDetails(t *testing.T) {
	var buf bytes.Buffer
	MutationDryRunDetail(&buf, `Would set state of epic "Q1 Roadmap" to closed.`, nil)
	got := buf.String()
	want := "Would set state of epic \"Q1 Roadmap\" to closed.\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
