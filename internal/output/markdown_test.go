package output

import (
	"bytes"
	"strings"
	"testing"
)

func TestRenderMarkdown(t *testing.T) {
	var buf bytes.Buffer
	err := RenderMarkdown(&buf, "Hello **world**", 80)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "Hello") || !strings.Contains(out, "world") {
		t.Errorf("expected rendered markdown to contain 'Hello' and 'world', got: %s", out)
	}
}

func TestRenderMarkdownEmpty(t *testing.T) {
	var buf bytes.Buffer
	err := RenderMarkdown(&buf, "", 80)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if buf.Len() != 0 {
		t.Errorf("expected no output for empty content, got: %q", buf.String())
	}
}

func TestRenderMarkdownPreservesContent(t *testing.T) {
	input := `# Heading

This is a paragraph with **bold** and *italic* text.

- Item one
- Item two
- Item three

` + "```go\nfmt.Println(\"hello\")\n```"

	var buf bytes.Buffer
	err := RenderMarkdown(&buf, input, 80)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := buf.String()
	for _, expected := range []string{"Heading", "paragraph", "bold", "italic", "Item one", "Item two", "Item three", "hello"} {
		if !strings.Contains(out, expected) {
			t.Errorf("expected output to contain %q, got:\n%s", expected, out)
		}
	}
}

func TestRenderMarkdownZeroWidth(t *testing.T) {
	var buf bytes.Buffer
	err := RenderMarkdown(&buf, "Some text", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "Some text") {
		t.Errorf("expected output to contain 'Some text', got: %s", buf.String())
	}
}
