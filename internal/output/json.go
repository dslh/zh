package output

import (
	"encoding/json"
	"fmt"
	"io"
)

// JSON writes v as indented JSON to w.
// Returns an error if marshaling fails.
func JSON(w io.Writer, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("formatting JSON output: %w", err)
	}
	_, err = fmt.Fprintln(w, string(data))
	return err
}

// IsJSON reports whether the output format flag is set to "json".
func IsJSON(format string) bool {
	return format == "json"
}
