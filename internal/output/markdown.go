package output

import (
	"io"

	"github.com/charmbracelet/glamour"
)

// RenderMarkdown renders user-authored markdown content (issue descriptions,
// epic bodies, sprint reviews) to the terminal using Glamour. It auto-detects
// the terminal background color for appropriate styling, and wraps text to the
// given width. Pass 0 for width to use Glamour's default (80).
//
// When color is disabled (NO_COLOR set or non-TTY), Glamour still produces
// readable plain-text output.
func RenderMarkdown(w io.Writer, content string, width int) error {
	if content == "" {
		return nil
	}

	opts := []glamour.TermRendererOption{
		glamour.WithAutoStyle(),
		glamour.WithEmoji(),
	}
	if width > 0 {
		opts = append(opts, glamour.WithWordWrap(width))
	}

	r, err := glamour.NewTermRenderer(opts...)
	if err != nil {
		return err
	}

	rendered, err := r.Render(content)
	if err != nil {
		return err
	}

	_, err = io.WriteString(w, rendered)
	return err
}
