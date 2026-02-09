package output

import (
	"bytes"
	"testing"

	"github.com/dslh/zh/internal/testutil"
)

func TestListWriter(t *testing.T) {
	var buf bytes.Buffer
	lw := NewListWriter(&buf, "NAME", "STATE", "ISSUES")
	lw.Row("Sprint 42", "active", "12")
	lw.Row("Sprint 41", "closed", "8")
	lw.Row("Sprint 40", "closed", "15")
	lw.FlushWithFooter("Total: 3 sprints")

	testutil.AssertSnapshot(t, "list-view.txt", buf.String())
}

func TestListWriterNoFooter(t *testing.T) {
	var buf bytes.Buffer
	lw := NewListWriter(&buf, "PIPELINE", "ISSUES")
	lw.Row("Backlog", "25")
	lw.Row("In Development", "8")
	lw.Row("Review", "3")
	lw.Flush()

	testutil.AssertSnapshot(t, "list-view-no-footer.txt", buf.String())
}

func TestListWriterEmpty(t *testing.T) {
	var buf bytes.Buffer
	lw := NewListWriter(&buf, "NAME", "STATE")
	lw.FlushWithFooter("Total: 0 items")

	testutil.AssertSnapshot(t, "list-view-empty.txt", buf.String())
}

func TestListWriterSingleColumn(t *testing.T) {
	var buf bytes.Buffer
	lw := NewListWriter(&buf, "LABEL")
	lw.Row("bug")
	lw.Row("enhancement")
	lw.Row("documentation")
	lw.Flush()

	testutil.AssertSnapshot(t, "list-view-single-column.txt", buf.String())
}

func TestListWriterWideColumns(t *testing.T) {
	var buf bytes.Buffer
	lw := NewListWriter(&buf, "REPO", "DESCRIPTION", "LANGUAGE")
	lw.Row("dlakehammond/task-tracker", "A task tracking application", "Go")
	lw.Row("dlakehammond/recipe-book", "Recipe collection manager", "Python")
	lw.Flush()

	testutil.AssertSnapshot(t, "list-view-wide.txt", buf.String())
}
