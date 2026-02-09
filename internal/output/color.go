// Package output provides structured formatting for CLI output.
//
// It implements the rendering conventions described in the zh spec:
// detail views, list views, mutation confirmations, progress bars,
// date formatting, and JSON output mode. All output respects the
// NO_COLOR environment variable and suppresses color when stdout
// is not a TTY.
package output

import (
	"fmt"
	"os"
)

// Color codes for ANSI escape sequences.
const (
	reset = "\033[0m"

	green  = "\033[32m"
	red    = "\033[31m"
	yellow = "\033[33m"
	cyan   = "\033[36m"
	dim    = "\033[2m"
	bold   = "\033[1m"
)

// colorEnabled reports whether color output is permitted.
// It returns false when NO_COLOR is set or stdout is not a TTY.
func colorEnabled() bool {
	if _, ok := os.LookupEnv("NO_COLOR"); ok {
		return false
	}
	stat, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return stat.Mode()&os.ModeCharDevice != 0
}

// wrap returns s wrapped in the given ANSI code, or s unchanged if color is disabled.
func wrap(code, s string) string {
	if !colorEnabled() {
		return s
	}
	return code + s + reset
}

// Green formats s in green (success, completed states).
func Green(s string) string { return wrap(green, s) }

// Red formats s in red (errors, failures).
func Red(s string) string { return wrap(red, s) }

// Yellow formats s in yellow (warnings, dry-run).
func Yellow(s string) string { return wrap(yellow, s) }

// Cyan formats s in cyan (entity IDs, links).
func Cyan(s string) string { return wrap(cyan, s) }

// Dim formats s in dim/gray (secondary information).
func Dim(s string) string { return wrap(dim, s) }

// Bold formats s in bold (titles, section headers).
func Bold(s string) string { return wrap(bold, s) }

// Greenf formats and colors in green.
func Greenf(format string, args ...any) string { return Green(fmt.Sprintf(format, args...)) }

// Redf formats and colors in red.
func Redf(format string, args ...any) string { return Red(fmt.Sprintf(format, args...)) }

// Yellowf formats and colors in yellow.
func Yellowf(format string, args ...any) string { return Yellow(fmt.Sprintf(format, args...)) }

// Cyanf formats and colors in cyan.
func Cyanf(format string, args ...any) string { return Cyan(fmt.Sprintf(format, args...)) }
