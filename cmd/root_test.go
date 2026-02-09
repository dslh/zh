package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestRootHelp(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"--help"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("root --help returned error: %v", err)
	}

	out := buf.String()
	if out == "" {
		t.Fatal("root --help produced no output")
	}
	if !strings.Contains(out, "zh") {
		t.Errorf("help output should mention zh, got: %s", out)
	}
	if !strings.Contains(out, "--verbose") {
		t.Error("help output should mention --verbose flag")
	}
	if !strings.Contains(out, "--output") {
		t.Error("help output should mention --output flag")
	}
}

func TestVersionSubcommand(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"version"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("version returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "zh version") {
		t.Errorf("version output should contain 'zh version', got: %s", out)
	}
	if !strings.Contains(out, "commit:") {
		t.Errorf("version output should contain 'commit:', got: %s", out)
	}
}

func TestUnknownCommand(t *testing.T) {
	rootCmd.SetOut(new(bytes.Buffer))
	rootCmd.SetArgs([]string{"nonexistent"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for unknown command")
	}
}
