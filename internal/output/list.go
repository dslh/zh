package output

import (
	"fmt"
	"io"
	"strings"
)

// ListWriter builds a column-aligned tabular list view.
//
// Usage:
//
//	lw := output.NewListWriter(w, "NAME", "STATE", "COUNT")
//	lw.Row("Sprint 1", "active", "12")
//	lw.Row("Sprint 2", "closed", "8")
//	lw.Footer("Total: 2 sprints")
type ListWriter struct {
	w       io.Writer
	headers []string
	rows    [][]string
}

// NewListWriter creates a ListWriter with the given column headers.
// Headers should be in ALL CAPS.
func NewListWriter(w io.Writer, headers ...string) *ListWriter {
	return &ListWriter{
		w:       w,
		headers: headers,
	}
}

// Row adds a row of values. The number of values should match the number of headers.
func (lw *ListWriter) Row(values ...string) {
	lw.rows = append(lw.rows, values)
}

// Flush renders the table to the writer: headers, separator, rows, and optional footer.
func (lw *ListWriter) Flush() {
	lw.FlushWithFooter("")
}

// FlushWithFooter renders the table and appends a footer line (e.g. "Total: 5 items").
// Pass an empty string to omit the footer.
func (lw *ListWriter) FlushWithFooter(footer string) {
	colCount := len(lw.headers)
	if colCount == 0 {
		return
	}

	// Compute column widths: max of header and all row values.
	widths := make([]int, colCount)
	for i, h := range lw.headers {
		widths[i] = len(h)
	}
	for _, row := range lw.rows {
		for i := 0; i < colCount && i < len(row); i++ {
			if len(row[i]) > widths[i] {
				widths[i] = len(row[i])
			}
		}
	}

	// Print headers.
	lw.printRow(lw.headers, widths, true)

	// Separator spanning the full table width.
	totalWidth := 0
	for i, w := range widths {
		totalWidth += w
		if i < colCount-1 {
			totalWidth += 4 // column gap
		}
	}
	if totalWidth < separatorWidth {
		totalWidth = separatorWidth
	}
	fmt.Fprintln(lw.w, strings.Repeat("─", totalWidth))

	// Rows.
	for _, row := range lw.rows {
		lw.printRow(row, widths, false)
	}

	// Footer.
	if footer != "" {
		fmt.Fprintln(lw.w)
		fmt.Fprintln(lw.w, footer)
	}
}

func (lw *ListWriter) printRow(values []string, widths []int, isHeader bool) {
	colCount := len(widths)
	var b strings.Builder
	for i := 0; i < colCount; i++ {
		val := ""
		if i < len(values) {
			val = values[i]
		}
		if isHeader {
			val = Bold(val)
		}

		if i < colCount-1 {
			// Pad to column width + gap. Use the unformatted length for padding.
			raw := ""
			if i < len(values) {
				raw = values[i]
			}
			b.WriteString(val)
			padding := widths[i] - len(raw) + 4
			if padding > 0 {
				b.WriteString(strings.Repeat(" ", padding))
			}
		} else {
			b.WriteString(val)
		}
	}
	fmt.Fprintln(lw.w, b.String())
}
