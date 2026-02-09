package output

import (
	"fmt"
	"io"
	"strings"
)

const separatorWidth = 80

// doubleSeparator is a line of ══ characters spanning separatorWidth.
var doubleSeparator = strings.Repeat("═", separatorWidth)

// singleSeparator is a line of ── characters spanning separatorWidth.
var singleSeparator = strings.Repeat("─", separatorWidth)

// DetailWriter builds a detail view for a single entity.
//
// Usage:
//
//	d := output.NewDetailWriter(w, "ISSUE", "Fix login button")
//	d.Field("State", "open")
//	d.Field("Estimate", "5")
//	d.Section("DESCRIPTION")
//	fmt.Fprintln(w, "The login button is broken...")
type DetailWriter struct {
	w io.Writer
}

// NewDetailWriter writes the entity title line and double separator.
// entityType should be ALL CAPS (e.g. "ISSUE", "EPIC", "SPRINT").
func NewDetailWriter(w io.Writer, entityType, title string) *DetailWriter {
	fmt.Fprintf(w, "%s: %s\n", Bold(entityType), Bold(title))
	fmt.Fprintln(w, doubleSeparator)
	fmt.Fprintln(w)
	return &DetailWriter{w: w}
}

// Field writes a key-value metadata line with the key right-aligned to a
// consistent column width. The keyWidth is calculated from the longest key
// across all fields — callers should use Fields for automatic alignment.
func (d *DetailWriter) Field(key, value string) {
	d.fieldWithWidth(key, value, 0)
}

func (d *DetailWriter) fieldWithWidth(key, value string, width int) {
	if width > 0 {
		fmt.Fprintf(d.w, "%*s:  %s\n", width, key, value)
	} else {
		fmt.Fprintf(d.w, "%s:  %s\n", key, value)
	}
}

// KeyValue is a key-value pair for use with Fields.
type KeyValue struct {
	Key   string
	Value string
}

// KV is a convenience constructor for KeyValue.
func KV(key, value string) KeyValue {
	return KeyValue{Key: key, Value: value}
}

// Fields writes a block of key-value pairs with right-aligned keys.
// The alignment width is computed automatically from the longest key.
func (d *DetailWriter) Fields(fields []KeyValue) {
	width := 0
	for _, f := range fields {
		if len(f.Key) > width {
			width = len(f.Key)
		}
	}
	for _, f := range fields {
		d.fieldWithWidth(f.Key, f.Value, width)
	}
}

// Section writes a section header with a single-line separator.
// The header name should be in ALL CAPS.
func (d *DetailWriter) Section(name string) {
	fmt.Fprintln(d.w)
	fmt.Fprintln(d.w, Bold(name))
	fmt.Fprintln(d.w, singleSeparator)
}
