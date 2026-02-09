package output

import "time"

// FormatDate formats a time as a standalone date: "Jan 20, 2025".
func FormatDate(t time.Time) string {
	return t.Format("Jan 2, 2006")
}

// FormatDateRange formats a date range with an arrow separator.
// Same-month ranges abbreviate: "Jan 20 → 31, 2025".
// Cross-month ranges: "Jan 20 → Feb 2, 2025".
// The year appears only on the end date.
func FormatDateRange(start, end time.Time) string {
	if start.Year() == end.Year() && start.Month() == end.Month() {
		return start.Format("Jan 2") + " → " + end.Format("2, 2006")
	}
	if start.Year() == end.Year() {
		return start.Format("Jan 2") + " → " + end.Format("Jan 2, 2006")
	}
	return start.Format("Jan 2, 2006") + " → " + end.Format("Jan 2, 2006")
}

// FormatDateISO formats a time as ISO 8601 date: "2025-01-20".
func FormatDateISO(t time.Time) string {
	return t.Format("2006-01-02")
}
