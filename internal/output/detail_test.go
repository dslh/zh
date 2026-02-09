package output

import (
	"bytes"
	"testing"

	"github.com/dslh/zh/internal/testutil"
)

func TestDetailWriter(t *testing.T) {
	var buf bytes.Buffer
	d := NewDetailWriter(&buf, "ISSUE", "Fix login button alignment")
	d.Fields([]KeyValue{
		KV("State", "open"),
		KV("Estimate", "5"),
		KV("Pipeline", "In Development"),
		KV("Assignees", "alice, bob"),
	})
	d.Section("DESCRIPTION")
	buf.WriteString("The login button is misaligned on mobile devices.\n")
	d.Section("CONNECTED PRS")
	buf.WriteString("  mpt#5 Fix button CSS\n")

	testutil.AssertSnapshot(t, "detail-view.txt", buf.String())
}

func TestDetailWriterMinimal(t *testing.T) {
	var buf bytes.Buffer
	d := NewDetailWriter(&buf, "SPRINT", "Sprint 42")
	d.Fields([]KeyValue{
		KV("State", "active"),
		KV("Dates", "Jan 20 → Feb 2, 2025"),
	})

	testutil.AssertSnapshot(t, "detail-view-minimal.txt", buf.String())
}

func TestDetailWriterSingleField(t *testing.T) {
	var buf bytes.Buffer
	d := NewDetailWriter(&buf, "PIPELINE", "Backlog")
	d.Field("Issues", "12")

	testutil.AssertSnapshot(t, "detail-view-single-field.txt", buf.String())
}
